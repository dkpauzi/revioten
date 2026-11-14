package change

import (
	"slices"
	"strings"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// TestRevisionLifecycle menjalankan satu alur revisi dan yokoten dari awal
// sampai akhir: revisi 0 terbit, perubahan membuka revisi 1, isi diubah,
// revisi 1 terbit oleh orang lain, diff 0 ke 1 benar, revisi 0 tetap utuh,
// lalu perbaikannya disebarkan ke model lain sampai terbit.
func TestRevisionLifecycle(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	product := doctest.FieldID(t, e.CPFormat, "Product Characteristics")
	method := doctest.FieldID(t, e.CPFormat, "Method")

	cp := e.NewSet(t, "k125", e.CustomerAB, e.ModelK125)["cp"]
	other := e.NewSet(t, "k200", e.CustomerAB, e.ModelK200)
	d, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste", method: "Visual"}})
	if err != nil {
		t.Fatal(err)
	}
	row := d.Rows[0]
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}

	// Perubahan untuk dokumen terbit: diajukan, diterima, membuka revisi 1.
	c, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "Use SPI", DocumentIDs: []string{cp.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}}); err == nil {
		t.Fatal("dokumen terbit bisa diubah sebelum perubahan diterima")
	}
	if c, err = s.Verify(ctx, controller, c.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}
	if d := c.Documents[0]; d.DraftRevision == nil || *d.DraftRevision != 1 || d.Merged {
		t.Fatalf("draft revisi 1: %+v", d)
	}
	d, err = e.Docs.Get(ctx, e.Author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Baris di draft punya ID yang sama dengan baris revisi 0 (R-DOC-02).
	if d.Rows[0].ID != row.ID {
		t.Fatalf("ID baris berubah: %s", d.Rows[0].ID)
	}
	if _, err := e.Docs.UpdateRow(ctx, e.Author, cp.ID, row.ID, document.RowUpdate{ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste", method: "SPI"}, Version: d.Rows[0].Version}); err != nil {
		t.Fatal(err)
	}
	// Penyusun draft tidak mencatat rilisnya; dokumen kontrol lain boleh.
	both := doctest.NewActor(t, e.Pool, "ppap.dcc", "work", doctest.All(access.RoleAuthor), doctest.All(access.RoleDocumentControl))
	if _, err := e.Docs.UpdateRow(ctx, both, cp.ID, row.ID, document.RowUpdate{ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste height", method: "SPI"}, Version: d.Rows[0].Version + 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.Release(ctx, both, cp.ID); err == nil {
		t.Fatal("penyusun mencatat rilisnya sendiri")
	}
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}
	one := 1
	diff, err := e.Docs.Diff(ctx, e.Author, cp.ID, 0, &one, "")
	if err != nil || len(diff.Rows) != 1 || diff.Rows[0].Status != document.RowChanged || len(diff.Rows[0].Cells) != 2 {
		t.Fatalf("diff 0 ke 1: %+v, %v", diff.Rows, err)
	}
	revs, _ := e.Docs.Revisions(ctx, e.Author, cp.ID)
	if len(revs) != 2 || revs[0].Status != "released" || revs[1].Status != "released" || !slices.Contains(revs[0].Changes, c.Ref) {
		t.Fatalf("riwayat: %+v", revs)
	}
	var old string
	if err := e.Pool.QueryRow(ctx, "SELECT snapshot FROM document_revisions WHERE document_id = $1 AND revision = 0", cp.ID).Scan(&old); err != nil || !containsAll(old, "Visual", "Solder paste") {
		t.Errorf("snapshot revisi 0: %v", err)
	}

	// Yokoten ke K-200: apply, diterima, semua dokumen K-200 terbit.
	y, err := s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: c.ID, Reason: "SPI on K-200 too", Targets: TargetFilter{ModelIDs: []string{e.ModelK200}}})
	if err != nil {
		t.Fatal(err)
	}
	if y, err = s.Decide(ctx, e.Author, y.ID, y.Targets[0].ID, DecisionApply, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(ctx, controller, y.Targets[0].ChangeID, true, "Same line"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pfd", "pfmea", "cp"} {
		if _, err := e.Docs.Release(ctx, dcc, other[key].ID); err != nil {
			t.Fatal(err)
		}
	}
	if y, _ = s.GetYokoten(ctx, e.Author, y.ID); y.Targets[0].Progress != ProgressReleased {
		t.Errorf("kemajuan yokoten: %s", y.Targets[0].Progress)
	}
	// Perubahan yokoten menunjuk sumbernya.
	target, _ := s.Get(ctx, e.Author, y.Targets[0].ChangeID)
	if target.Source == nil || target.Source.ID != c.ID {
		t.Errorf("sumber perubahan yokoten: %+v", target.Source)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
