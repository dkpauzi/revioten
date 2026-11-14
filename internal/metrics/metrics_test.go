package metrics_test

import (
	"errors"
	"os"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/metrics"
)

// TestMain menyalakan PostgreSQL untuk test dasbor.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// M7.7 · Dasbor ukuran: angka dihitung dari data, per customer atau semua.
func TestMetrics(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := metrics.NewService(e.Pool, e.Checker)
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))

	// Database kosong: semua nol, daftar kosong, dan tidak ada pembagian dengan nol.
	m, err := s.Get(ctx, e.Viewer, "")
	if err != nil {
		t.Fatal(err)
	}
	if m.Rows.Total != 0 || m.Rows.LinkedPercent != 0 || len(m.Documents) != 0 || len(m.Yokoten) != 0 || m.Revision.Count != 0 || m.Migration.AverageMinutes != 0 {
		t.Fatalf("kosong: %+v", m)
	}

	// Customer AB: satu set, CP berisi dua baris yang diketik sendiri lalu terbit.
	ab := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	for range 2 {
		if _, err := e.Docs.AddRow(ctx, e.Author, ab["cp"].ID, document.NewRow{Content: map[string]any{}}); err != nil {
			t.Fatal(err)
		}
	}
	doctest.AttachScan(t, e.Pool, ab["cp"].ID)
	if _, err := e.Docs.Release(ctx, dcc, ab["cp"].ID); err != nil {
		t.Fatal(err)
	}
	// Customer CD: satu set tanpa baris.
	e.NewSet(t, "cd", e.CustomerCD, e.ModelB1)

	count := func(m metrics.Metrics, key, status string) int {
		for _, d := range m.Documents {
			if d.TypeKey == key && d.Status == status {
				return d.Count
			}
		}
		return 0
	}
	if m, err = s.Get(ctx, e.Viewer, ""); err != nil {
		t.Fatal(err)
	}
	if m.Rows.Total != 2 || m.Rows.Free != 2 || m.Rows.Linked != 0 || count(m, "cp", "released") != 1 || count(m, "cp", "draft") != 1 || count(m, "pfd", "draft") != 2 {
		t.Errorf("semua customer: %+v", m)
	}
	// Revisi awal tidak dihitung sebagai waktu revisi.
	if m.Revision.Count != 0 {
		t.Errorf("waktu revisi: %+v", m.Revision)
	}
	if m, err = s.Get(ctx, e.Viewer, e.CustomerCD); err != nil {
		t.Fatal(err)
	}
	if m.Rows.Total != 0 || count(m, "cp", "draft") != 1 || count(m, "cp", "released") != 0 {
		t.Errorf("customer CD: %+v", m)
	}

	if _, err := s.Get(ctx, e.Viewer, "bukan-uuid"); err == nil {
		t.Error("customer salah diterima")
	}
	if _, err := s.Get(ctx, access.Actor{}, ""); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa akun: %v", err)
	}
}
