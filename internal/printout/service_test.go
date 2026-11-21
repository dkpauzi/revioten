package printout_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
	"github.com/dkpauzi/revioten/internal/template"
)

func code(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

func pageSize(t *testing.T, data []byte, sheet string) int {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pl, err := f.GetPageLayout(sheet)
	if err != nil || pl.Size == nil {
		t.Fatalf("tata letak: %v", err)
	}
	return *pl.Size
}

// TestPrintService: cetak draft, cetak untuk tanda tangan, scan wajib sebelum
// rilis, salinan terkendali bernomor, catatan cetak, verifikasi, dan layout
// per customer (M6.1 sampai M6.9).
func TestPrintService(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := printout.NewService(e.Pool, e.Checker, e.Docs, store, "")
	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	if _, err := e.Pool.Exec(ctx, `UPDATE documents SET approval_route_id = (SELECT id FROM approval_routes ORDER BY name LIMIT 1) WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	product := doctest.FieldID(t, e.CPFormat, "Product Characteristics")
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste height"}}); err != nil {
		t.Fatal(err)
	}
	signer := doctest.NewActor(t, e.Pool, "signer", "work", doctest.All(access.RoleAuthor), doctest.All(access.RoleDocumentControl))
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))

	// Draft: Excel A3 dengan watermark UNCONTROLLED COPY.
	f, err := s.Print(ctx, e.Author, cp.ID, printout.Request{Purpose: printout.PurposeDraft, Format: "xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(f.Name, "-00-draft.xlsx") || pageSize(t, f.Body, "CP") != 8 {
		t.Errorf("cetak draft: %s", f.Name)
	}
	if _, err := s.Print(ctx, e.Viewer, cp.ID, printout.Request{Purpose: printout.PurposeDraft, Format: "xlsx"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mencetak draft: %v", err)
	}
	if _, err := s.Print(ctx, e.Author, cp.ID, printout.Request{Purpose: printout.PurposeControlled, Format: "xlsx"}); code(t, err) != "purpose:not_allowed" {
		t.Errorf("salinan terkendali dari draft: %v", err)
	}
	// Cetak untuk tanda tangan mengisi tanggal rilis (D-59).
	if _, err := s.Print(ctx, signer, cp.ID, printout.Request{Purpose: printout.PurposeSignature, Format: "xlsx"}); err != nil {
		t.Fatal(err)
	}
	var date *string
	if err := e.Pool.QueryRow(ctx, "SELECT release_date::text FROM document_revisions WHERE document_id = $1 AND revision = 0", cp.ID).Scan(&date); err != nil || date == nil {
		t.Errorf("tanggal rilis: %v, %v", date, err)
	}

	// Rilis tanpa scan ditolak; dengan scan berhasil (M6.7).
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); code(t, err) != "scan:required" {
		t.Errorf("rilis tanpa scan: %v", err)
	}
	if _, err := s.UploadScan(ctx, e.Author, cp.ID, "signed.pdf", strings.NewReader("%PDF signed")); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun mengunggah scan: %v", err)
	}
	if _, err := s.UploadScan(ctx, dcc, cp.ID, "signed.exe", strings.NewReader("MZ")); code(t, err) != "file:not_allowed" {
		t.Errorf("scan .exe: %v", err)
	}
	scan, err := s.UploadScan(ctx, dcc, cp.ID, "SC-AB-001-00 signed.pdf", strings.NewReader("%PDF signed"))
	if err != nil || scan.FileName != "SC-AB-001-00 signed.pdf" || len(scan.SHA256) != 64 {
		t.Fatalf("scan: %+v, %v", scan, err)
	}
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}

	// Salinan terkendali bernomor 1, 2; referensi tanpa nomor.
	for want := 1; want <= 2; want++ {
		f, err := s.Print(ctx, dcc, cp.ID, printout.Request{Purpose: printout.PurposeControlled, Format: "xlsx"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(f.Name, "-00-controlled-"+string(rune('0'+want))+".xlsx") {
			t.Errorf("salinan %d: %s", want, f.Name)
		}
	}
	if _, err := s.Print(ctx, e.Author, cp.ID, printout.Request{Purpose: printout.PurposeControlled, Format: "xlsx"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun mencetak salinan terkendali: %v", err)
	}
	logs, err := s.Logs(ctx, e.Author, cp.ID)
	if err != nil || len(logs) != 4 || logs[0].Purpose != printout.PurposeControlled || logs[0].CopyNumber == nil || *logs[0].CopyNumber != 2 {
		t.Fatalf("catatan cetak: %+v, %v", logs, err)
	}

	// Verifikasi: revisi 0 berlaku.
	var rev0 string
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM document_revisions WHERE document_id = $1 AND revision = 0", cp.ID).Scan(&rev0); err != nil {
		t.Fatal(err)
	}
	v, err := s.Verify(ctx, e.Viewer, rev0)
	if err != nil || v.Status != "effective" || v.Number != cp.Number || len(v.SHA256) != 64 {
		t.Errorf("verifikasi: %+v, %v", v, err)
	}

	// Layout lain untuk customer AB: CP dicetak A4 (M6.1), tanpa mengubah kode.
	layouts, err := s.Layouts(ctx, e.Author)
	if err != nil || len(layouts) != 3 {
		t.Fatalf("layout bawaan: %d, %v", len(layouts), err)
	}
	var cpType string
	for _, l := range layouts {
		if l.DocumentTypeKey == "cp" {
			cpType = l.DocumentTypeID
		}
	}
	if _, err := s.CreateLayout(ctx, e.Author, printout.LayoutInput{Name: "AB CP A4", DocumentTypeID: cpType, Layout: json.RawMessage(`{"paper":"A4"}`)}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun membuat layout: %v", err)
	}
	if _, err := s.CreateLayout(ctx, e.Master, printout.LayoutInput{Name: "Bad", DocumentTypeID: cpType, Layout: json.RawMessage(`{"paper":"A5"}`)}); code(t, err) != "paper:invalid_format" {
		t.Errorf("layout A5: %v", err)
	}
	a4, err := s.CreateLayout(ctx, e.Master, printout.LayoutInput{Name: "AB CP A4", DocumentTypeID: cpType, Layout: json.RawMessage(`{"paper":"A4","orientation":"landscape","font_size":7,"margins_mm":{"top":8,"right":8,"bottom":8,"left":8}}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCustomerLayout(ctx, e.Master, e.CustomerAB, cpType, a4.ID); err != nil {
		t.Fatal(err)
	}
	f, err = s.Print(ctx, e.Author, cp.ID, printout.Request{Purpose: printout.PurposeReference, Format: "xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	if pageSize(t, f.Body, "CP") != 9 {
		t.Error("layout customer tidak dipakai")
	}
}

// TestPrintMigratedHistory: kriteria selesai M7.2. Riwayat revisi teks dokumen
// migrasi tampil lengkap di cetakan, dan scan hardcopy diunggah petugas
// migrasi (E-10) tanpa hak dokumen kontrol.
func TestPrintMigratedHistory(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := printout.NewService(e.Pool, e.Checker, e.Docs, store, "")
	if _, err := e.Pool.Exec(ctx, "UPDATE system_settings SET migration_mode = true"); err != nil {
		t.Fatal(err)
	}
	staff := doctest.NewActor(t, e.Pool, "ppap.staff", "work", doctest.All(access.RoleMigration), doctest.All(access.RoleAuthor))
	if _, err := e.Numbers.RegisterSet(ctx, e.Author, numbering.NewSet{CustomerID: e.CustomerAB, IdempotencyKey: "mig-" + t.Name()}); err != nil {
		t.Fatal(err)
	}
	numbers, err := e.Docs.MigratableNumbers(ctx, staff, e.CustomerAB)
	if err != nil || len(numbers) == 0 {
		t.Fatalf("nomor: %v", err)
	}
	d, err := e.Docs.Migrate(ctx, staff, document.LegacyDocument{
		NewDocument: document.NewDocument{NumberID: numbers[len(numbers)-1].ID, ModelIDs: []string{e.ModelK125}},
		Revision:    4, EffectiveDate: "2025-03-10", PreparedBy: "Budi Santoso", ApprovedBy: "Siti Aminah",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.SetHistory(ctx, staff, d.ID, []document.HistoryEntry{
		{Revision: "-00", Date: "2023-01-05", Description: "Initial release of the old form"},
		{Revision: "-04", Date: "2025-03-10", Description: "Add ICT step"},
	}); err != nil {
		t.Fatal(err)
	}
	f, err := s.Print(ctx, staff, d.ID, printout.Request{Purpose: printout.PurposeDraft, Format: "xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.Name, "-04-draft.xlsx") {
		t.Errorf("nama berkas: %s", f.Name)
	}
	book, err := excelize.OpenReader(bytes.NewReader(f.Body))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	var cover strings.Builder
	for _, sheet := range book.GetSheetList() {
		rows, _ := book.GetRows(sheet)
		for _, row := range rows {
			cover.WriteString(strings.Join(row, "|") + "\n")
		}
	}
	text := cover.String()
	if !strings.Contains(text, "Initial release of the old form") || !strings.Contains(text, "Add ICT step") || strings.Contains(text, "Migrated baseline") {
		t.Errorf("riwayat di cetakan:\n%s", text)
	}
	// Baris revisi baseline tidak digandakan sistem: -04 muncul sekali di tabel riwayat.
	if n := strings.Count(text, "Add ICT step"); n != 1 {
		t.Errorf("baris -04 muncul %d kali", n)
	}
	if _, err := s.UploadScan(ctx, staff, d.ID, "hardcopy.pdf", strings.NewReader("%PDF-1.4 hardcopy")); err != nil {
		t.Errorf("petugas migrasi mengunggah scan: %v", err)
	}
	if _, err := s.UploadScan(ctx, e.Author, d.ID, "hardcopy.pdf", strings.NewReader("%PDF-1.4 hardcopy")); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa E-10 mengunggah scan baseline: %v", err)
	}
}

// infected adalah pemindai tiruan yang selalu menemukan malware.
type infected struct{}

func (infected) Scan(context.Context, io.Reader) error {
	return fmt.Errorf("%w: Eicar-Test-Signature", storage.ErrInfected)
}

// TestUploadScanIsVirusScanned: M8.2. Scan bertanda tangan yang terdeteksi
// berbahaya ditolak, dan nama temuannya tidak sampai ke pesan untuk pengguna.
func TestUploadScanIsVirusScanned(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := printout.NewService(e.Pool, e.Checker, e.Docs, store.WithScanner(infected{}), "")
	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	_, err = s.UploadScan(ctx, dcc, cp.ID, "signed.pdf", strings.NewReader("%PDF-1.4 signed"))
	if code(t, err) != "file:not_allowed" || strings.Contains(err.Error(), "Eicar") {
		t.Errorf("scan berbahaya: %v", err)
	}
}

// TestFlowFrozenAtRelease (D-212): alur CP revisi terbit digambar dari
// langkah PFD yang dibekukan saat rilis, jadi PFD yang diubah sesudahnya
// tidak mengubah cetakan revisi itu.
func TestFlowFrozenAtRelease(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := printout.NewService(e.Pool, e.Checker, e.Docs, store, "")
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	cp, pfd := set["cp"], set["pfd"]
	format, err := template.LoadFormatVersion(ctx, db.New(e.Pool), pfd.FormatVersionID)
	if err != nil {
		t.Fatal(err)
	}
	field := map[string]string{}
	for _, f := range format.Fields {
		field[f.Key] = f.ID
	}
	pfdNow, err := e.Docs.AddRow(ctx, e.Author, pfd.ID, document.NewRow{OperationNumber: "10", ProcessID: e.ProcessSMT,
		Content: map[string]any{field["flow_symbol"]: "decision", field["ng_disposition"]: "scrap"}})
	if err != nil {
		t.Fatal(err)
	}
	product := doctest.FieldID(t, e.CPFormat, "Product Characteristics")
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{OperationNumber: "10", ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste height"}}); err != nil {
		t.Fatal(err)
	}
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	doctest.AttachScan(t, e.Pool, cp.ID)
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}

	// PFD diubah sesudah CP terbit: inspeksi menjadi proses biasa tanpa NG.
	row := pfdNow.Rows[0]
	if _, err := e.Docs.UpdateRow(ctx, e.Author, pfd.ID, row.ID, document.RowUpdate{OperationNumber: "10", ProcessID: e.ProcessSMT,
		Content: map[string]any{field["flow_symbol"]: "process"}, Version: row.Version}); err != nil {
		t.Fatal(err)
	}
	view, err := s.Preview(ctx, e.Author, cp.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow := string(view.Pages[0].Flow)
	if !strings.Contains(flow, "SCRAP") || !strings.Contains(flow, `data-step="10"`) {
		t.Errorf("alur revisi terbit harus memakai PFD saat rilis: %s", flow)
	}
}
