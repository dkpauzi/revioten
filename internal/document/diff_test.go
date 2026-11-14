package document_test

import (
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.6 · Diff dan riwayat (R-REV-03): dua revisi dibandingkan per baris dan
// per sel, baris dicocokkan lewat ID tetap, dan bisa disaring per proses.
func TestDiff(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	product := doctest.FieldID(t, e.CPFormat, "Product Characteristics")
	method := doctest.FieldID(t, e.CPFormat, "Method")
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	add := func(process, text string) document.Row {
		t.Helper()
		d, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{ProcessID: process, Content: map[string]any{product: text}})
		if err != nil {
			t.Fatal(err)
		}
		return d.Rows[len(d.Rows)-1]
	}
	a := add(e.ProcessSMT, "Solder paste height")
	b := add("", "Label position")
	c := add("", "Packing count")
	if _, err := e.Docs.Release(ctx, recorder, cp.ID); err != nil {
		t.Fatal(err)
	}
	// Draft revisi 1 (biasanya dibuka perubahan yang diterima, M5.4).
	if _, err := e.Pool.Exec(ctx, `INSERT INTO document_revisions (document_id, revision, created_by_person_id) VALUES ($1, 1, $2)`, cp.ID, recorder.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE documents SET status = 'draft', revision = 1 WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.UpdateRow(ctx, e.Author, cp.ID, a.ID, document.RowUpdate{ProcessID: e.ProcessSMT, Content: map[string]any{product: "Solder paste height", method: "SPI"}, Version: a.Version}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.DeleteRow(ctx, e.Author, cp.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	d := add("", "Carton label")

	diff, err := e.Docs.Diff(ctx, e.Author, cp.ID, 0, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]document.RowChange{}
	for _, r := range diff.Rows {
		status[r.RowID] = r
	}
	if r := status[a.ID]; r.Status != document.RowChanged || len(r.Cells) != 1 || r.Cells[0].Label != "Method" || r.Cells[0].Old != nil || r.Cells[0].New != "SPI" {
		t.Errorf("baris diubah: %+v", r)
	}
	if _, ok := status[b.ID]; ok {
		t.Errorf("baris yang tidak berubah ikut tampil: %+v", status[b.ID])
	}
	if status[c.ID].Status != document.RowRemoved || status[d.ID].Status != document.RowAdded || len(diff.Rows) != 3 || !diff.ToDraft {
		t.Errorf("diff: %+v", diff.Rows)
	}
	// Saring per proses: hanya baris SMT Printing.
	diff, err = e.Docs.Diff(ctx, e.Author, cp.ID, 0, nil, e.ProcessSMT)
	if err != nil || len(diff.Rows) != 1 || diff.Rows[0].RowID != a.ID {
		t.Errorf("per proses: %+v, %v", diff.Rows, err)
	}
	// Dua revisi terbit dibandingkan dari snapshot-nya.
	if _, err := e.Docs.Release(ctx, recorder, cp.ID); err != nil {
		t.Fatal(err)
	}
	one := 1
	diff, err = e.Docs.Diff(ctx, e.Author, cp.ID, 0, &one, "")
	if err != nil || len(diff.Rows) != 3 || diff.ToDraft {
		t.Errorf("revisi 0 ke 1: %+v, %v", diff.Rows, err)
	}
	// Revisi yang belum terbit ditolak.
	two := 2
	if _, err := e.Docs.Diff(ctx, e.Author, cp.ID, 0, &two, ""); codeOf(t, err) != "to:not_allowed" {
		t.Errorf("revisi 2: %v", err)
	}
}
