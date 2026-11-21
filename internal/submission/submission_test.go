package submission_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
	"github.com/dkpauzi/revioten/internal/submission"
)

// D-207 · Form Control Submission per dokumen.

func TestMain(m *testing.M) {
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

func day(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, clock.Zone)
	return &t
}

func TestSubmissionControl(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := submission.NewService(e.Pool, e.Checker, store, "")
	engineer := doctest.NewActor(t, e.Pool, "ppap.engineer", "work", doctest.All(access.RolePPAPEngineer))
	recorder := doctest.NewActor(t, e.Pool, "ppap.dua", "work", doctest.All(access.RolePPAPEngineer))
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	cp := set["cp"]

	// Revisi 0 terbit dengan scan bertanda tangan basah.
	doctest.AttachScan(t, e.Pool, cp.ID)
	if _, err := e.Docs.Release(ctx, recorder, cp.ID); err != nil {
		t.Fatal(err)
	}

	// Akun tanpa L-4 tidak melihat form (draft disaring RLS atau ditolak izin).
	if _, err := svc.Get(ctx, e.Viewer, cp.ID); !errors.Is(err, access.ErrForbidden) && !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("viewer: %v", err)
	}
	c, err := svc.Get(ctx, engineer, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	today := clock.Date(time.Now())
	if c.Number != cp.Number || c.DocumentType != "cp" || c.CustomerCode != "AB" || c.Models != "K-125" || !c.SubmissionRequired || len(c.Rows) != 1 {
		t.Fatalf("form: %+v", c)
	}
	r := c.Rows[0]
	if r.Reason != "First release" || r.Status != "released" || !r.StartedOn.Equal(today) || r.FinishedOn == nil || !r.FinishedOn.Equal(today) ||
		r.TopSignOn == nil || r.TopSignBy != "scan" || !r.Required || r.Days != 0 {
		t.Errorf("baris revisi 0: %+v", r)
	}

	// Kolom yang diketik: hak E-7.
	entry := submission.Entry{SubmittedOn: day(2026, 11, 20), Package: "Customer cover + CP", Result: submission.ResultApproved, ResultOn: day(2026, 11, 25), Remark: "MP approval"}
	if _, err := svc.Save(ctx, e.Author, cp.ID, 0, entry); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa E-7: %v", err)
	}
	for name, x := range map[string]struct {
		in   submission.Entry
		want string
	}{
		"hasil tanpa tanggal kirim": {submission.Entry{Result: submission.ResultApproved}, "submitted_on:required"},
		"hasil sebelum dikirim":     {submission.Entry{SubmittedOn: day(2026, 11, 20), Result: submission.ResultApproved, ResultOn: day(2026, 11, 19)}, "result_on:not_allowed"},
		"tanggal hasil tanpa hasil": {submission.Entry{SubmittedOn: day(2026, 11, 20), Result: submission.ResultWaiting, ResultOn: day(2026, 11, 21)}, "result_on:not_allowed"},
		"hasil tidak dikenal":       {submission.Entry{SubmittedOn: day(2026, 11, 20), Result: "ok"}, "result:invalid_format"},
		"tidak perlu tanpa alasan":  {submission.Entry{Required: new(false)}, "override_reason:required"},
	} {
		if _, err := svc.Save(ctx, engineer, cp.ID, 0, x.in); codeOf(t, err) != x.want {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := svc.Save(ctx, engineer, cp.ID, 9, entry); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("revisi yang tidak ada: %v", err)
	}
	c, err = svc.Save(ctx, engineer, cp.ID, 0, entry)
	if err != nil {
		t.Fatal(err)
	}
	r = c.Rows[0]
	if r.SubmittedOn == nil || !r.SubmittedOn.Equal(*day(2026, 11, 20)) || r.Result != submission.ResultApproved || r.Package != "Customer cover + CP" ||
		r.Remark != "MP approval" || r.UpdatedBy != "Orang ppap.engineer" {
		t.Errorf("sesudah disimpan: %+v", r)
	}

	// Revisi yang tidak perlu dikirim: alasan wajib, tanggal kirim harus kosong.
	if _, err := svc.Save(ctx, engineer, cp.ID, 0, submission.Entry{Required: new(false), OverrideReason: "Internal change only", SubmittedOn: day(2026, 11, 20)}); codeOf(t, err) != "submitted_on:not_allowed" {
		t.Errorf("tidak perlu tapi dikirim: %v", err)
	}
	c, err = svc.Save(ctx, engineer, cp.ID, 0, submission.Entry{Required: new(false), OverrideReason: "Internal change only", Remark: "Customer informed by e-mail"})
	if err != nil || c.Rows[0].Required || c.Rows[0].OverrideReason != "Internal change only" {
		t.Errorf("tidak perlu dikirim: %+v, %v", c.Rows, err)
	}

	// Bawaan customer: hanya admin (A-2); mengikuti bawaan menghapus pengecualian.
	if err := svc.SetCustomerDefault(ctx, engineer, e.CustomerAB, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja mengubah bawaan: %v", err)
	}
	if err := svc.SetCustomerDefault(ctx, e.Admin, e.CustomerAB, false); err != nil {
		t.Fatal(err)
	}
	c, _ = svc.Save(ctx, engineer, cp.ID, 0, submission.Entry{Required: new(false)})
	if c.SubmissionRequired || c.Rows[0].Required || c.Rows[0].OverrideReason != "" {
		t.Errorf("mengikuti bawaan customer: %+v", c)
	}
	_ = svc.SetCustomerDefault(ctx, e.Admin, e.CustomerAB, true)
	if _, err := svc.Save(ctx, engineer, cp.ID, 0, entry); err != nil {
		t.Fatal(err)
	}

	// Scan bukti customer (opsional).
	if _, err := svc.UploadScan(ctx, engineer, cp.ID, 0, "approval.exe", strings.NewReader("x")); codeOf(t, err) != "file:not_allowed" {
		t.Errorf("jenis berkas: %v", err)
	}
	c, err = svc.UploadScan(ctx, engineer, cp.ID, 0, "approval.pdf", strings.NewReader("%PDF-1.4 bukti"))
	if err != nil || c.Rows[0].ScanName != "approval.pdf" {
		t.Fatalf("scan: %+v, %v", c.Rows, err)
	}
	sc, f, err := svc.OpenScan(ctx, engineer, cp.ID, 0)
	if err != nil || sc.FileName != "approval.pdf" {
		t.Fatalf("buka scan: %+v, %v", sc, err)
	}
	f.Close()

	// Cetak Excel: judul, nomor dokumen, isi baris, dan 30 baris per halaman.
	file, err := svc.Export(ctx, engineer, cp.ID, "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Body))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := book.GetRows(book.GetSheetList()[0])
	all := ""
	for _, row := range rows {
		all += strings.Join(row, "|") + "\n"
	}
	for _, want := range []string{"Form Control Submission Document", ": " + cp.Number, "Submission Document", "First release", "20-Nov-26", "Approved 25-Nov-26", "MP approval", "\n30\n"} {
		if !strings.Contains(all, want) {
			t.Errorf("Excel tanpa %q:\n%s", want, all)
		}
	}
	if _, err := svc.Export(ctx, engineer, cp.ID, "docx"); codeOf(t, err) != "format:invalid_format" {
		t.Errorf("format: %v", err)
	}
	var n int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action LIKE '%submission%'`).Scan(&n); err != nil || n < 5 {
		t.Errorf("audit: %d, %v", n, err)
	}
}
