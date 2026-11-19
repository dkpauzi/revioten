package review_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/review"
)

func TestMain(m *testing.M) {
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}

func allOK() []review.Answer {
	var out []review.Answer
	for _, item := range review.ChecklistItems {
		out = append(out, review.Answer{Item: item, Answer: review.AnswerOK})
	}
	return out
}

func fieldCode(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if errors.As(err, &invalid) && len(invalid.Problems) > 0 {
		return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
	}
	var conflict *problem.ConflictError
	if errors.As(err, &conflict) && len(conflict.Problems) > 0 {
		return conflict.Problems[0].Field + ":" + conflict.Problems[0].Code
	}
	t.Fatalf("ingin penolakan input, dapat %v", err)
	return ""
}

// D-138, D-196 · Set dokumen yang 3 tahun tidak direvisi muncul di daftar
// review: kuning 90 hari sebelumnya, merah setelah lewat. Review tanpa
// perubahan dicatat satu orang, dikonfirmasi orang kedua, lalu memulai lagi
// jam 3 tahun.
func TestPeriodicReview(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	for _, key := range []string{"pfd", "pfmea", "cp"} {
		doctest.AttachScan(t, e.Pool, set[key].ID)
		if _, err := e.Docs.Release(ctx, dcc, set[key].ID); err != nil {
			t.Fatal(err)
		}
	}
	// Set customer CD belum pernah terbit: tidak pernah ikut.
	e.NewSet(t, "cd", e.CustomerCD, e.ModelB1)

	base := review.NewService(e.Pool, e.Checker)
	products := product.NewService(e.Pool, e.Checker)
	at := func(months, days int) *review.Service {
		return base.WithNow(func() time.Time { return time.Now().AddDate(0, months, days) })
	}
	sets := func(s *review.Service, all bool) review.Overview {
		t.Helper()
		o, err := s.Sets(ctx, e.Viewer, review.Filter{All: all})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}

	// Baru terbit: tidak ada peringatan, tetapi ada di daftar lengkap.
	if o := sets(base, false); len(o.Sets) != 0 || o.PeriodMonths != 36 || o.WarningDays != 90 {
		t.Fatalf("baru terbit: %+v", o)
	}
	o := sets(base, true)
	if len(o.Sets) != 1 || o.Sets[0].Status != review.StatusOK || len(o.Sets[0].Documents) != 3 || o.Sets[0].Models[0] != "K-125" {
		t.Fatalf("daftar lengkap: %+v", o)
	}
	due := o.Sets[0].DueOn
	if want := clock.Date(time.Now()).AddDate(3, 0, 0); !due.Equal(want) {
		t.Errorf("jatuh tempo %v, ingin %v", due, want)
	}

	// 60 hari sebelum jatuh tempo: kuning. Sehari sesudahnya: merah.
	if o := sets(at(36, -60), false); len(o.Sets) != 1 || o.Sets[0].Status != review.StatusDueSoon || o.DueSoon != 1 || o.Sets[0].DaysLeft != 60 {
		t.Errorf("akan jatuh tempo: %+v", o)
	}
	late := at(36, 1)
	if o := sets(late, false); len(o.Sets) != 1 || o.Sets[0].Status != review.StatusOverdue || o.Overdue != 1 || o.Sets[0].DaysLeft != -1 {
		t.Errorf("lewat jatuh tempo: %+v", o)
	}
	// 200 hari sebelum jatuh tempo: rencana satu tahun, hanya di daftar lengkap.
	if o := sets(at(36, -200), true); len(o.Sets) != 1 || o.Sets[0].Status != review.StatusDueYear || o.DueYear != 1 {
		t.Errorf("dalam satu tahun: %+v", o)
	}
	// Jam per dokumen (D-202): setiap dokumen punya jatuh temponya sendiri.
	if o := sets(late, false); !o.Sets[0].Documents[2].Overdue || o.Sets[0].Engineer != "Orang penyusun.uji" {
		t.Errorf("per dokumen: %+v", o.Sets[0])
	}
	// Model EOL (D-200): set keluar dari daftar, kecuali diminta.
	if _, err := products.SetEOL(ctx, e.Viewer, e.ModelK125, true); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer menandai EOL: %v", err)
	}
	if _, err := products.SetEOL(ctx, e.Author, e.ModelK125, true); err != nil {
		t.Fatal(err)
	}
	if o := sets(late, false); len(o.Sets) != 0 || o.Overdue != 0 {
		t.Errorf("EOL masih diperingatkan: %+v", o)
	}
	if o, err := late.Sets(ctx, e.Viewer, review.Filter{EOL: true}); err != nil || len(o.Sets) != 1 || o.Sets[0].Status != review.StatusEOL {
		t.Errorf("tampilkan EOL: %+v, %v", o, err)
	}
	if _, err := products.SetEOL(ctx, e.Author, e.ModelK125, false); err != nil {
		t.Fatal(err)
	}
	if body, name, err := late.Export(ctx, e.Viewer, review.Filter{All: true}); err != nil || len(body) < 100 || name == "" {
		t.Errorf("Excel: %d %v", len(body), err)
	}

	// Hanya set satu dokumen.
	if o, err := late.Sets(ctx, e.Viewer, review.Filter{DocumentID: set["cp"].ID}); err != nil || len(o.Sets) != 1 {
		t.Errorf("per dokumen: %+v, %v", o, err)
	}

	verifier := doctest.NewActor(t, e.Pool, "verifier.uji", "work", doctest.All(access.RoleVerifier))
	today := clock.Date(time.Now().AddDate(0, 36, 1)).Format(time.DateOnly)
	in := review.NewReview{DocumentID: set["cp"].ID, ReviewedOn: today, Participants: "Ahmad (PPAP), Budi (QA)", Conclusion: "No change needed.", Checklist: allOK()}

	// Viewer tidak mencatat; isian harus lengkap.
	if _, err := late.Record(ctx, e.Viewer, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mencatat: %v", err)
	}
	bad := in
	bad.Checklist = allOK()[:3]
	if _, err := late.Record(ctx, e.Author, bad); fieldCode(t, err) != "checklist:required" {
		t.Errorf("daftar periksa kurang: %v", err)
	}
	bad = in
	bad.Checklist = append(allOK()[1:], review.Answer{Item: review.ChecklistItems[0], Answer: "needs_change"})
	if _, err := late.Record(ctx, e.Author, bad); fieldCode(t, err) != "checklist:invalid_format" {
		t.Errorf("butir perlu perubahan: %v", err)
	}
	bad = in
	bad.ReviewedOn = clock.Date(time.Now().AddDate(0, 36, 3)).Format(time.DateOnly)
	if _, err := late.Record(ctx, e.Author, bad); fieldCode(t, err) != "reviewed_on:not_allowed" {
		t.Errorf("tanggal masa depan: %v", err)
	}

	r, err := late.Record(ctx, e.Author, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "submitted" || r.Ref != "PRV-0001" || len(r.Revisions) != 3 || len(r.Revisions[2].SHA256) != 64 || len(r.Checklist) != len(review.ChecklistItems) {
		t.Fatalf("catatan: %+v", r)
	}
	// Satu catatan terbuka per set.
	if _, err := late.Record(ctx, e.Author, in); fieldCode(t, err) != "document_id:taken" {
		t.Errorf("catatan kedua: %v", err)
	}
	// Belum dikonfirmasi: jam belum mulai lagi, tetapi set menandai catatannya.
	if o := sets(late, false); len(o.Sets) != 1 || o.Sets[0].OpenReviewID != r.ID {
		t.Errorf("menunggu konfirmasi: %+v", o)
	}

	// Pencatat tidak mengonfirmasi sendiri; tanpa V-1 ditolak.
	if _, err := late.Confirm(ctx, e.Author, r.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa V-1: %v", err)
	}
	both := doctest.NewActor(t, e.Pool, "ppap.verifier", "work", doctest.All(access.RoleAuthor), doctest.All(access.RoleVerifier))
	own := in
	own.DocumentID = set["pfd"].ID
	if r2, err := late.Reviews(ctx, e.Viewer, "", true); err != nil || len(r2) != 1 {
		t.Fatalf("daftar menunggu: %+v, %v", r2, err)
	}
	if r, err = late.Confirm(ctx, verifier, r.ID); err != nil {
		t.Fatal(err)
	}
	if r.Status != "confirmed" || r.ConfirmedBy != "Orang verifier.uji" || r.ConfirmedAt == nil {
		t.Errorf("terkonfirmasi: %+v", r)
	}
	if _, err := late.Confirm(ctx, verifier, r.ID); fieldCode(t, err) != "status:not_allowed" {
		t.Errorf("konfirmasi dua kali: %v", err)
	}
	// Jam mulai lagi dari tanggal review.
	if o := sets(late, false); len(o.Sets) != 0 {
		t.Errorf("sesudah review: %+v", o)
	}
	if o := sets(late, true); o.Sets[0].LastReview == nil || o.Sets[0].Status != review.StatusOK {
		t.Errorf("review terakhir: %+v", o.Sets[0])
	}

	// Catatan yang dikonfirmasi tidak bisa diubah, juga lewat SQL.
	if _, err := e.Pool.Exec(ctx, "UPDATE periodic_reviews SET status = status WHERE id = $1", r.ID); err == nil {
		t.Error("database menerima perubahan catatan terkonfirmasi")
	}

	// Orang yang mencatat juga verifikator tetap tidak mengonfirmasi sendiri.
	later := at(72, 2)
	own.ReviewedOn = clock.Date(time.Now().AddDate(0, 72, 2)).Format(time.DateOnly)
	r2, err := later.Record(ctx, both, own)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := later.Confirm(ctx, both, r2.ID); fieldCode(t, err) != "confirmed_by:not_allowed" {
		t.Errorf("konfirmasi sendiri: %v", err)
	}
}
