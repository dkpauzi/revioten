package metrics_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/metrics"
)

// D-197 · Minggu laporan berjalan Jumat sampai Kamis; bulan dan tahun kalender.
func TestPeriodOf(t *testing.T) {
	day := func(s string) time.Time {
		d, err := time.ParseInLocation(time.DateOnly, s, clock.Zone)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, tt := range []struct{ kind, date, from, to string }{
		{metrics.PeriodWeek, "2026-11-13", "2026-11-13", "2026-11-20"}, // Jumat: awal minggu
		{metrics.PeriodWeek, "2026-11-15", "2026-11-13", "2026-11-20"}, // Minggu
		{metrics.PeriodWeek, "2026-11-19", "2026-11-13", "2026-11-20"}, // Kamis: akhir minggu
		{metrics.PeriodWeek, "2026-11-12", "2026-11-06", "2026-11-13"}, // Kamis sebelumnya
		{metrics.PeriodMonth, "2026-02-28", "2026-02-01", "2026-03-01"},
		{metrics.PeriodYear, "2026-11-15", "2026-01-01", "2027-01-01"},
	} {
		p, err := metrics.PeriodOf(tt.kind, day(tt.date))
		if err != nil {
			t.Fatal(err)
		}
		if !p.From.Equal(day(tt.from)) || !p.To.Equal(day(tt.to)) {
			t.Errorf("%s %s: %s sampai %s", tt.kind, tt.date, p.From.Format(time.DateOnly), p.To.Format(time.DateOnly))
		}
	}
	// Pukul 23.30 UTC hari Kamis sudah Jumat di WIB: minggu baru.
	thursdayLate := time.Date(2026, 11, 19, 23, 30, 0, 0, time.UTC)
	if p, _ := metrics.PeriodOf(metrics.PeriodWeek, thursdayLate); !p.From.Equal(day("2026-11-20")) {
		t.Errorf("zona waktu: %v", p.From)
	}
	if _, err := metrics.PeriodOf("quarter", day("2026-11-15")); err == nil {
		t.Error("periode tidak dikenal diterima")
	}
}

// D-197 · Laporan kinerja dihitung dari permintaan perubahan, catatan cetak,
// unggahan scan, dan rilis. Hanya pemegang L-5 yang boleh membacanya.
func TestPerformance(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := metrics.NewService(e.Pool, e.Checker)
	changes := change.NewRequestService(e.Pool, e.Checker)
	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]

	var claim, content string
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM change_sources WHERE name = 'Customer claim'").Scan(&claim); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM change_types WHERE key = 'content'").Scan(&content); err != nil {
		t.Fatal(err)
	}
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	if _, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id)
		SELECT id, per_project_group, $1 FROM positions WHERE key = 'apqp_ppap_controller'`, controller.PersonID); err != nil {
		t.Fatal(err)
	}
	c, err := changes.Create(ctx, e.Author, change.NewChange{CustomerID: e.CustomerAB, TriggerID: claim, ChangeTypeID: content, Reason: "Claim 12", DocumentIDs: []string{cp.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changes.Verify(ctx, controller, c.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.MarkReady(ctx, e.Author, cp.ID); err != nil {
		t.Fatal(err)
	}
	// Penyusun mencetak untuk tanda tangan; scan bertanda tangan diunggah dua
	// jam kemudian (kolom scan boleh diisi role aplikasi).
	if _, err := e.Pool.Exec(ctx, `INSERT INTO print_logs (document_id, revision_id, revision, purpose, format, content_sha256, printed_by_person_id, printed_by_account_id)
		SELECT document_id, id, revision, 'signature', 'pdf', repeat('b', 64), $2, $3
		FROM document_revisions WHERE document_id = $1 AND status = 'draft'`, cp.ID, e.Author.PersonID, e.Author.AccountID); err != nil {
		t.Fatal(err)
	}
	doctest.AttachScan(t, e.Pool, cp.ID)
	if _, err := e.Pool.Exec(ctx, `UPDATE document_revisions SET scan_uploaded_at = now() + interval '2 hours' WHERE document_id = $1 AND status = 'draft'`, cp.ID); err != nil {
		t.Fatal(err)
	}
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}

	// Viewer tidak punya L-5; Penyusun (PPAP Engineer) punya.
	if _, err := s.Performance(ctx, e.Viewer, metrics.PeriodWeek, time.Now(), "", ""); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
	p, err := s.Performance(ctx, e.Author, metrics.PeriodWeek, time.Now(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Submitted != 1 || p.Accepted != 1 || p.Rejected != 0 || p.Releases != 1 {
		t.Fatalf("jumlah: %+v", p)
	}
	if len(p.ByTrigger) != 1 || p.ByTrigger[0].Name != "Customer claim" || p.ByTrigger[0].Count != 1 {
		t.Errorf("pemicu: %+v", p.ByTrigger)
	}
	if d := p.Stages.PrintToSigned; d.Count != 1 || d.AverageHours < 1.9 || d.AverageHours > 2.1 {
		t.Errorf("cetak sampai tanda tangan: %+v", d)
	}
	// Siap cetak ditandai sebelum dicetak: tahap menunggu cetak ikut dihitung.
	if st := p.Stages; st.EditToReady.Count != 1 || st.WaitingToPrint.Count != 1 || st.AcceptedToRelease.Count != 1 || st.SignedToReleased.Count != 0 {
		t.Errorf("tahap lain: %+v", st)
	}
	if p.ByCategory[metrics.CategoryCustomerClaim] != 1 || p.ReleaseItems[0].Category != metrics.CategoryCustomerClaim {
		t.Errorf("kelompok pemicu: %+v", p.ByCategory)
	}
	if len(p.Trend) != 8 || p.Trend[7].Releases != 1 || p.Previous.Submitted != 0 || p.OpenAtEnd != 0 {
		t.Errorf("tren dan pembanding: %+v %+v %d", p.Trend, p.Previous, p.OpenAtEnd)
	}
	if len(p.ReleaseItems) != 1 || p.ReleaseItems[0].PrintedBy != "Orang penyusun.uji" {
		t.Errorf("rincian rilis: %+v", p.ReleaseItems)
	}
	var author *metrics.PersonPerformance
	for i := range p.People {
		if p.People[i].PersonID == e.Author.PersonID {
			author = &p.People[i]
		}
	}
	if author == nil || author.Submitted != 1 || author.Accepted != 1 || author.Releases != 1 || author.Stages.PrintToSigned.Count != 1 || author.CustomerClaim != 1 {
		t.Errorf("per orang: %+v", p.People)
	}

	// Saringan satu orang: orang lain kosong.
	if p, err = s.Performance(ctx, e.Author, metrics.PeriodWeek, time.Now(), "", controller.PersonID); err != nil || p.Submitted != 0 || p.Releases != 0 {
		t.Errorf("orang lain: %+v, %v", p, err)
	}
	if _, err := s.Performance(ctx, e.Author, metrics.PeriodWeek, time.Now(), "", "bukan-uuid"); err == nil {
		t.Error("orang salah diterima")
	}
	// Tahun: 12 titik bulan, dan unduhan Excel bisa dibuat.
	if p, err = s.Performance(ctx, e.Author, metrics.PeriodYear, time.Now(), "", ""); err != nil || len(p.Trend) != 12 {
		t.Errorf("tren tahun: %d, %v", len(p.Trend), err)
	}
	if body, name, err := s.Export(ctx, e.Author, metrics.PeriodWeek, time.Now(), "", ""); err != nil || len(body) < 100 || !strings.HasSuffix(name, ".xlsx") {
		t.Errorf("Excel: %d %s %v", len(body), name, err)
	}

	// Customer lain dan minggu lalu kosong.
	if p, err = s.Performance(ctx, e.Author, metrics.PeriodWeek, time.Now(), e.CustomerCD, ""); err != nil || p.Submitted != 0 || p.Releases != 0 {
		t.Errorf("customer CD: %+v, %v", p, err)
	}
	if p, err = s.Performance(ctx, e.Author, metrics.PeriodWeek, time.Now().AddDate(0, 0, -7), "", ""); err != nil || p.Submitted != 0 {
		t.Errorf("minggu lalu: %+v, %v", p, err)
	}
	// Tahun ini memuat minggu ini.
	if p, err = s.Performance(ctx, e.Author, metrics.PeriodYear, time.Now(), "", ""); err != nil || p.Submitted != 1 || p.Releases != 1 {
		t.Errorf("tahun: %+v, %v", p, err)
	}
}
