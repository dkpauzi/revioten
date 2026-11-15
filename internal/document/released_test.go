package document_test

import (
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// D-195 (E-128 opsi a) · Akun Viewer membaca revisi yang berlaku dari
// snapshot, termasuk PFMEA dan CP berdampingan, dan tidak pernah melihat draft.
func TestReleasedView(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	pfmea, cp := set["pfmea"], set["cp"]
	viewer := doctest.NewActor(t, e.Pool, "audit.internal", "work", doctest.All(access.RoleViewer))
	checker := doctest.NewActor(t, e.Pool, "checker.uji", "work", doctest.All(access.RolePFMEACPChecker))
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	field := doctest.FieldID(t, e.CPFormat, "Product Characteristics")
	for _, d := range []document.Document{pfmea, cp} {
		if _, err := e.Docs.AddRow(ctx, e.Author, d.ID, document.NewRow{ProcessID: e.ProcessSMT, Content: map[string]any{}}); err != nil {
			t.Fatal(err)
		}
	}
	d, err := e.Docs.Get(ctx, e.Author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.UpdateRow(ctx, e.Author, cp.ID, d.Rows[0].ID, document.RowUpdate{ProcessID: e.ProcessSMT, Content: map[string]any{field: "PCB rev 0"}, Version: d.Rows[0].Version}); err != nil {
		t.Fatal(err)
	}

	// Draft: tidak ada bagi Viewer, dan halaman dokumen biasa tetap tertutup.
	if list, err := e.Docs.ListReleased(ctx, viewer, e.CustomerAB); err != nil || len(list) != 0 {
		t.Errorf("daftar sebelum terbit: %+v, %v", list, err)
	}
	if _, err := e.Docs.Released(ctx, viewer, cp.ID); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("draft terbaca: %v", err)
	}
	if _, err := e.Docs.Get(ctx, viewer, cp.ID); !errors.Is(err, access.ErrForbidden) && !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("halaman draft: %v", err)
	}

	// Tanda dicek diberikan pada draft, lalu kedua dokumen terbit.
	if _, err := e.Docs.MarkChecked(ctx, checker, cp.ID, "", e.ProcessSMT, true); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []document.Document{pfmea, cp} {
		doctest.AttachScan(t, e.Pool, doc.ID)
		if _, err := e.Docs.Release(ctx, dcc, doc.ID); err != nil {
			t.Fatal(err)
		}
	}
	list, err := e.Docs.ListReleased(ctx, viewer, e.CustomerAB)
	if err != nil || len(list) != 2 || list[0].Status != document.StatusReleased {
		t.Fatalf("daftar terbit: %+v, %v", list, err)
	}

	// Revisi 1 dibuka sebagai draft dan isinya diubah: Viewer tetap membaca revisi 0.
	if _, err := e.Pool.Exec(ctx, `INSERT INTO document_revisions (document_id, revision, created_by_person_id) VALUES ($1, 1, $2)`, cp.ID, e.Author.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE documents SET status = 'draft', revision = 1 WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if d, err = e.Docs.Get(ctx, e.Author, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.UpdateRow(ctx, e.Author, cp.ID, d.Rows[0].ID, document.RowUpdate{ProcessID: e.ProcessSMT, Content: map[string]any{field: "PCB rev 1"}, Version: d.Rows[0].Version}); err != nil {
		t.Fatal(err)
	}
	r, err := e.Docs.Released(ctx, viewer, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Revision != 0 || r.Status != document.StatusReleased || len(r.Rows) != 1 || r.Rows[0].Content[field] != "PCB rev 0" || r.Rows[0].ProcessName != "SMT Printing" {
		t.Errorf("isi terbit: rev %d %s %+v", r.Revision, r.Status, r.Rows)
	}
	if len(r.Set) != 2 {
		t.Errorf("set terbit (PFD belum terbit): %+v", r.Set)
	}

	// PFMEA dan CP berdampingan dari revisi yang berlaku, dengan tanda dicek.
	c, err := e.Docs.ReleasedComparison(ctx, viewer, pfmea.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Processes) != 1 || c.Processes[0].Check == nil || c.Processes[0].Check.By != "Orang checker.uji" || c.CP.Rows[0].Content[field] != "PCB rev 0" {
		t.Errorf("perbandingan terbit: %+v", c.Processes)
	}
	// Tanda tidak bisa diubah Viewer.
	if _, err := e.Docs.MarkChecked(ctx, viewer, cp.ID, "", e.ProcessSMT, false); err == nil {
		t.Error("viewer mengubah tanda dicek")
	}
	// Set tanpa PFMEA dan CP terbit.
	other := e.NewSet(t, "cd", e.CustomerCD, e.ModelB1)
	if _, err := e.Docs.ReleasedComparison(ctx, viewer, other["cp"].ID); err == nil {
		t.Error("perbandingan tanpa dokumen terbit diterima")
	}
}
