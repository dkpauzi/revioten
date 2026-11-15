package integration_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/integration"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func code(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if errors.As(err, &invalid) && len(invalid.Problems) > 0 {
		return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
	}
	var conflict *problem.ConflictError
	if errors.As(err, &conflict) && len(conflict.Problems) > 0 {
		return "conflict:" + conflict.Problems[0].Field + ":" + conflict.Problems[0].Code
	}
	t.Fatalf("ingin penolakan, dapat %v", err)
	return ""
}

type env struct {
	doctest.Env
	svc     *integration.Service
	dcc     access.Actor
	product string // ID kolom Product Characteristics di CP
}

func setup(t *testing.T) env {
	t.Helper()
	e := doctest.Setup(t)
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	printer := printout.NewService(e.Pool, e.Checker, e.Docs, store, "")
	return env{
		Env: e, svc: integration.NewService(e.Pool, e.Checker, store, printer),
		dcc:     doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl)),
		product: doctest.FieldID(t, e.CPFormat, "Product Characteristics"),
	}
}

// released membuat satu set dokumen, mengisi CP dengan satu baris, dan merilis
// semua dokumen yang disebut di kinds. Hasilnya dokumen per jenis.
func (e env) released(t *testing.T, key, customerID, modelID string, kinds ...string) map[string]document.Document {
	t.Helper()
	ctx := t.Context()
	set := e.NewSet(t, key, customerID, modelID)
	if _, err := e.Docs.AddRow(ctx, e.Author, set["cp"].ID, document.NewRow{ProcessID: e.ProcessSMT, Content: map[string]any{e.product: "Solder paste height"}}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range kinds {
		doctest.AttachScan(t, e.Pool, set[kind].ID)
		d, err := e.Docs.Release(ctx, e.dcc, set[kind].ID)
		if err != nil {
			t.Fatal(err)
		}
		set[kind] = d
	}
	return set
}

// client membuat service account untuk customer tertentu dan mengembalikan
// aplikasi yang sudah dikenali dari token barunya.
func (e env) client(t *testing.T, name string, scopes []string, customers ...string) (integration.Client, integration.Account, string) {
	t.Helper()
	ctx := t.Context()
	acc, err := e.svc.CreateAccount(ctx, e.Admin, integration.NewAccount{
		Name: name, OwnerTeam: "Tim MES", Contact: "mes@contoh.test", Purpose: "Uji integrasi", Scopes: scopes, CustomerIDs: customers,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := e.svc.IssueToken(ctx, e.Admin, acc.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := e.svc.Authenticate(ctx, token, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	return c, acc, token
}

// TestTokens: kriteria selesai M9.1. Token kedaluwarsa, dicabut, milik akun
// nonaktif, atau dari jaringan lain selalu ditolak; nilainya tidak disimpan.
func TestTokens(t *testing.T) {
	e := setup(t)
	ctx := t.Context()

	if _, err := e.svc.CreateAccount(ctx, e.Author, integration.NewAccount{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja membuat service account: %v", err)
	}
	if _, err := e.svc.CreateAccount(ctx, e.Admin, integration.NewAccount{Name: "MES", OwnerTeam: "T", Contact: "c", Purpose: "p",
		Scopes: []string{"documents:delete"}, CustomerIDs: []string{e.CustomerAB}}); code(t, err) != "scopes:invalid_format" {
		t.Errorf("scope tidak dikenal: %v", err)
	}
	if _, err := e.svc.CreateAccount(ctx, e.Admin, integration.NewAccount{Name: "MES", OwnerTeam: "T", Contact: "c", Purpose: "p",
		Scopes: []string{integration.ScopeDocumentsRead}, CustomerIDs: []string{e.CustomerAB}, Networks: []string{"kantor"}}); code(t, err) != "networks:invalid_format" {
		t.Errorf("jaringan salah: %v", err)
	}

	c, acc, token := e.client(t, "MES", integration.Scopes, e.CustomerAB)
	if c.Name != "MES" || len(c.CustomerIDs) != 1 || !strings.HasPrefix(token, "rvt_"+c.TokenPrefix+"_") {
		t.Fatalf("aplikasi: %+v", c)
	}
	if _, err := e.svc.CreateAccount(ctx, e.Admin, integration.NewAccount{Name: "mes", OwnerTeam: "T", Contact: "c", Purpose: "p",
		Scopes: []string{integration.ScopeDocumentsRead}, CustomerIDs: []string{e.CustomerAB}}); code(t, err) != "conflict:name:taken" {
		t.Errorf("nama ganda: %v", err)
	}
	// Yang disimpan hanya hash; nilai token tidak ada di tabel token maupun audit log.
	var leaked int
	if err := e.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM service_tokens WHERE token_hash = $1 OR prefix = $1)
		+ (SELECT count(*) FROM audit_log WHERE details LIKE '%' || $1 || '%')`, token).Scan(&leaked); err != nil || leaked != 0 {
		t.Errorf("nilai token tersimpan: %d, %v", leaked, err)
	}

	for _, bad := range []string{"", "bukan token", token + "x", "rvt_" + c.TokenPrefix + "_salah"} {
		if _, err := e.svc.Authenticate(ctx, bad, "127.0.0.1"); !errors.Is(err, integration.ErrUnauthenticated) {
			t.Errorf("token %q diterima: %v", bad, err)
		}
	}

	// Rotasi: token kedua terbit, keduanya berlaku, lalu token pertama dicabut.
	second, _, err := e.svc.IssueToken(ctx, e.Admin, acc.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Authenticate(ctx, token, "127.0.0.1"); err != nil {
		t.Errorf("token lama selama rotasi: %v", err)
	}
	if err := e.svc.RevokeToken(ctx, e.Admin, acc.ID, c.TokenID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Authenticate(ctx, token, "127.0.0.1"); !errors.Is(err, integration.ErrUnauthenticated) {
		t.Errorf("token dicabut diterima: %v", err)
	}
	if _, err := e.svc.Authenticate(ctx, second, "127.0.0.1"); err != nil {
		t.Errorf("token baru: %v", err)
	}
	// Kedaluwarsa: token 30 hari diperiksa dengan jam 31 hari ke depan.
	future := e.svc.WithClock(func() time.Time { return time.Now().Add(31 * 24 * time.Hour) })
	if _, err := future.Authenticate(ctx, second, "127.0.0.1"); !errors.Is(err, integration.ErrUnauthenticated) {
		t.Errorf("token kedaluwarsa diterima: %v", err)
	}
	if _, _, err := e.svc.IssueToken(ctx, e.Admin, acc.ID, 500); code(t, err) != "days:invalid_format" {
		t.Errorf("masa berlaku terlalu panjang: %v", err)
	}
	// Penolakan token yang dikenal tercatat, dengan alasannya.
	var refused int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'service_account.token_refused' AND actor_type = 'service'`).Scan(&refused); err != nil || refused != 2 {
		t.Errorf("penolakan tercatat: %d, %v", refused, err)
	}

	// Jaringan yang diizinkan.
	lan, err := e.svc.CreateAccount(ctx, e.Admin, integration.NewAccount{Name: "ERP", OwnerTeam: "T", Contact: "c", Purpose: "p",
		Scopes: []string{integration.ScopeDocumentsRead}, CustomerIDs: []string{e.CustomerAB}, Networks: []string{"10.0.0.0/8"}})
	if err != nil {
		t.Fatal(err)
	}
	lanToken, _, err := e.svc.IssueToken(ctx, e.Admin, lan.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Authenticate(ctx, lanToken, "192.168.1.5"); !errors.Is(err, integration.ErrUnauthenticated) {
		t.Errorf("jaringan lain diterima: %v", err)
	}
	if _, err := e.svc.Authenticate(ctx, lanToken, "10.1.2.3"); err != nil {
		t.Errorf("jaringan yang diizinkan: %v", err)
	}
	// Akun nonaktif: semua tokennya ditolak, dan token baru tidak bisa terbit.
	if err := e.svc.DisableAccount(ctx, e.Admin, lan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Authenticate(ctx, lanToken, "10.1.2.3"); !errors.Is(err, integration.ErrUnauthenticated) {
		t.Errorf("akun nonaktif diterima: %v", err)
	}
	if _, _, err := e.svc.IssueToken(ctx, e.Admin, lan.ID, 0); code(t, err) != "service_account:not_allowed" {
		t.Errorf("token untuk akun nonaktif: %v", err)
	}
	list, err := e.svc.ListAccounts(ctx, e.Admin)
	if err != nil || len(list) != 2 || list[0].Name != "ERP" || !list[0].Disabled || len(list[1].Tokens) != 2 {
		t.Errorf("daftar: %+v, %v", list, err)
	}
}

// TestRead: kriteria selesai M9.3. Sinkronisasi bertahap berjalan, hanya data
// rilis yang keluar, dan token satu customer tidak bisa membaca customer lain.
func TestRead(t *testing.T) {
	e := setup(t)
	ctx := t.Context()
	ab := e.released(t, "ab", e.CustomerAB, e.ModelK125, "pfd", "pfmea", "cp")
	draftOnly := e.NewSet(t, "ab2", e.CustomerAB, e.ModelK200)["cp"]
	cd := e.released(t, "cd", e.CustomerCD, e.ModelB1, "cp")
	c, _, _ := e.client(t, "MES", integration.Scopes, e.CustomerAB)

	// Halaman pertama dua dokumen, halaman kedua satu; draft dan customer CD tidak ikut.
	first, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{Limit: 2})
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("halaman pertama: %+v, %v", first, err)
	}
	second, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{Limit: 2, Cursor: first.NextCursor})
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("halaman kedua: %+v, %v", second, err)
	}
	seen := map[string]bool{}
	for _, d := range append(first.Items, second.Items...) {
		seen[d.ID] = true
		if d.CustomerID != e.CustomerAB || d.SHA256 == "" || d.ReleasedAt.IsZero() {
			t.Errorf("dokumen: %+v", d)
		}
	}
	if len(seen) != 3 || !seen[ab["cp"].ID] || seen[draftOnly.ID] || seen[cd["cp"].ID] {
		t.Errorf("isi daftar: %v", seen)
	}
	// Sinkronisasi bertahap: sesudah waktu rilis terakhir, tidak ada yang baru.
	later := second.Items[0].ReleasedAt.Add(time.Second)
	if page, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{UpdatedSince: &later}); err != nil || len(page.Items) != 0 {
		t.Errorf("updated_since: %+v, %v", page, err)
	}
	if page, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{CustomerID: e.CustomerCD}); err != nil || len(page.Items) != 0 {
		t.Errorf("customer lain di daftar: %+v, %v", page, err)
	}
	if _, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{Limit: 500}); code(t, err) != "limit:invalid_format" {
		t.Errorf("halaman terlalu besar: %v", err)
	}
	if _, err := e.svc.ListDocuments(ctx, c, integration.ListFilter{Cursor: "rusak"}); code(t, err) != "cursor:invalid_format" {
		t.Errorf("cursor rusak: %v", err)
	}

	rev, err := e.svc.GetRevision(ctx, c, ab["cp"].ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rev.Revision != 0 || !rev.Current || len(rev.Snapshot.Rows) != 1 || rev.Snapshot.Rows[0].Content[e.product] != "Solder paste height" || len(rev.Fields) == 0 {
		t.Errorf("revisi: %+v", rev)
	}
	// Customer lain, dokumen yang belum terbit, dan revisi yang tidak ada: semuanya "tidak ada".
	one := 1
	for name, try := range map[string]func() error{
		"customer lain":        func() error { _, err := e.svc.GetRevision(ctx, c, cd["cp"].ID, nil); return err },
		"dokumen belum terbit": func() error { _, err := e.svc.GetRevision(ctx, c, draftOnly.ID, nil); return err },
		"revisi tidak ada":     func() error { _, err := e.svc.GetRevision(ctx, c, ab["cp"].ID, &one); return err },
		"ekspor customer lain": func() error { _, err := e.svc.CreateExport(ctx, c, cd["cp"].ID, nil); return err },
	} {
		if err := try(); !errors.Is(err, problem.ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Tanpa scope yang sesuai ditolak.
	narrow, _, _ := e.client(t, "Pembaca daftar", []string{integration.ScopeDocumentsRead}, e.CustomerAB)
	if _, err := e.svc.GetRevision(ctx, narrow, ab["cp"].ID, nil); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa revisions:read: %v", err)
	}
	if _, _, err := e.svc.Propose(ctx, narrow, "k", integration.Proposal{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa changes:propose: %v", err)
	}
}

// TestPropose: kriteria selesai M9.4. Usulan dari luar menunggu verifikasi
// orang, isinya masuk ke draft baru saat diterima, dan kunci idempoten serta
// revisi dasar dijaga.
func TestPropose(t *testing.T) {
	e := setup(t)
	ctx := t.Context()
	cp := e.released(t, "ab", e.CustomerAB, e.ModelK125, "cp")["cp"]
	row := cp.Rows[0]
	c, _, _ := e.client(t, "MES", integration.Scopes, e.CustomerAB)
	changes := change.NewRequestService(e.Pool, e.Checker)
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	if _, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id)
		SELECT id, per_project_group, $1 FROM positions WHERE key = 'apqp_ppap_controller'`, controller.PersonID); err != nil {
		t.Fatal(err)
	}

	in := integration.Proposal{
		DocumentID: cp.ID, BaseRevision: 0, SourceApplication: "MES", OnBehalfOf: "Budi Santoso", Reason: "SPI limit changed at the line",
		Operations: []document.PatchOp{
			{Op: document.PatchModify, RowID: row.ID, Content: map[string]any{e.product: "Solder paste height 120 um"}},
			{Op: document.PatchAdd, ProcessID: e.ProcessSMT, Content: map[string]any{e.product: "Solder paste volume"}},
		},
	}
	if _, _, err := e.svc.Propose(ctx, c, " ", in); code(t, err) != "Idempotency-Key:required" {
		t.Errorf("tanpa kunci idempoten: %v", err)
	}
	wrongBase := in
	wrongBase.BaseRevision = 3
	if _, _, err := e.svc.Propose(ctx, c, "k-base", wrongBase); code(t, err) != "conflict:base_revision:changed" {
		t.Errorf("revisi dasar salah: %v", err)
	}
	unknownRow := in
	unknownRow.Operations = []document.PatchOp{{Op: document.PatchRemove, RowID: "01900000-0000-7000-8000-000000000000"}}
	if _, _, err := e.svc.Propose(ctx, c, "k-row", unknownRow); code(t, err) != "operations.0.row_id:invalid_format" {
		t.Errorf("baris tidak dikenal: %v", err)
	}

	got, replayed, err := e.svc.Propose(ctx, c, "k-1", in)
	if err != nil || replayed || got.Status != "submitted" || !strings.HasPrefix(got.Ref, "CHG-") {
		t.Fatalf("usulan: %+v, %v, %v", got, replayed, err)
	}
	// Diulang dengan kunci dan isi yang sama: hasil yang sama, tanpa perubahan kedua.
	again, replayed, err := e.svc.Propose(ctx, c, "k-1", in)
	if err != nil || !replayed || again.ID != got.ID {
		t.Errorf("ulangan: %+v, %v, %v", again, replayed, err)
	}
	other := in
	other.Reason = "Alasan lain"
	if _, _, err := e.svc.Propose(ctx, c, "k-1", other); code(t, err) != "conflict:Idempotency-Key:taken" {
		t.Errorf("kunci sama isi berbeda: %v", err)
	}
	var count int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM changes`).Scan(&count); err != nil || count != 1 {
		t.Errorf("jumlah perubahan: %d, %v", count, err)
	}

	// Belum ada yang berubah di dokumen: usulan menunggu orang.
	if d, _ := e.Docs.Get(ctx, e.Author, cp.ID); d.Status != document.StatusReleased || d.Rows[0].Content[e.product] != "Solder paste height" {
		t.Errorf("dokumen berubah sebelum verifikasi: %+v", d)
	}
	seen, err := changes.Get(ctx, e.Author, got.ID)
	if err != nil || seen.Mechanism != change.MechanismIntegration || seen.RequestedBy != "MES" || seen.RequestedByID != "" {
		t.Fatalf("perubahan di sisi orang: %+v, %v", seen, err)
	}
	if st, err := e.svc.ChangeStatus(ctx, c, got.ID); err != nil || st.Status != "submitted" {
		t.Errorf("status: %+v, %v", st, err)
	}
	stranger, _, _ := e.client(t, "ERP", integration.Scopes, e.CustomerAB)
	if _, err := e.svc.ChangeStatus(ctx, stranger, got.ID); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("usulan aplikasi lain terbaca: %v", err)
	}

	// Verifikator menerima: revisi 1 terbuka dan operasinya masuk ke draft.
	if _, err := changes.Verify(ctx, controller, got.ID, true, "Sesuai laporan lini"); err != nil {
		t.Fatal(err)
	}
	d, err := e.Docs.Get(ctx, e.Author, cp.ID)
	if err != nil || d.Status != document.StatusDraft || d.Revision != 1 || len(d.Rows) != 2 ||
		d.Rows[0].Content[e.product] != "Solder paste height 120 um" || d.Rows[1].Content[e.product] != "Solder paste volume" {
		t.Fatalf("draft sesudah diterima: %+v, %v", d, err)
	}
	if st, _ := e.svc.ChangeStatus(ctx, c, got.ID); st.Status != "accepted" || st.VerificationNote != "Sesuai laporan lini" {
		t.Errorf("status sesudah diterima: %+v", st)
	}

	// Orang yang disebut pemanggil sebagai pemohon tidak memverifikasinya (R-API-09).
	behalf := in
	behalf.OnBehalfOf = controller.Login
	behalf.Operations = in.Operations[:1]
	mine, _, err := e.svc.Propose(ctx, c, "k-2", behalf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changes.Verify(ctx, controller, mine.ID, true, "ok"); code(t, err) != "verifier:not_allowed" {
		t.Errorf("pemohon yang disebut memverifikasi: %v", err)
	}
}

// TestExport: M9.5. Ekspor diantrekan, dikerjakan di latar, lalu diunduh.
func TestExport(t *testing.T) {
	e := setup(t)
	ctx := t.Context()
	cp := e.released(t, "ab", e.CustomerAB, e.ModelK125, "cp")["cp"]
	c, _, _ := e.client(t, "MES", integration.Scopes, e.CustomerAB)

	job, err := e.svc.CreateExport(ctx, c, cp.ID, nil)
	if err != nil || job.Status != "queued" {
		t.Fatalf("job: %+v, %v", job, err)
	}
	if _, _, err := e.svc.OpenExport(ctx, c, job.ID); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("unduh sebelum selesai: %v", err)
	}
	if worked, err := e.svc.ProcessNextExport(ctx); err != nil || !worked {
		t.Fatalf("mengerjakan job: %v, %v", worked, err)
	}
	if worked, err := e.svc.ProcessNextExport(ctx); err != nil || worked {
		t.Errorf("antrean seharusnya kosong: %v, %v", worked, err)
	}
	done, err := e.svc.GetExport(ctx, c, job.ID)
	if err != nil || done.Status != "done" || !strings.HasSuffix(done.FileName, "-00-export.xlsx") || done.SizeBytes == 0 {
		t.Fatalf("job selesai: %+v, %v", done, err)
	}
	_, f, err := e.svc.OpenExport(ctx, c, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	head := make([]byte, 2)
	_, _ = io.ReadFull(f, head)
	f.Close()
	if string(head) != "PK" { // berkas xlsx adalah arsip zip
		t.Errorf("isi berkas: %q", head)
	}
	other, _, _ := e.client(t, "ERP", integration.Scopes, e.CustomerAB)
	if _, err := e.svc.GetExport(ctx, other, job.ID); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("job aplikasi lain terbaca: %v", err)
	}
}
