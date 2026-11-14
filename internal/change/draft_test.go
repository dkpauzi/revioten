package change

import (
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.4 · Draft revisi (R-CHG-03): perubahan yang diterima memberi setiap
// dokumen terdampak satu draft. Draft yang sudah terbuka menerima perubahan
// itu (merged); dokumen yang sudah terbit membuka revisi berikutnya tanpa
// mengubah revisi yang berlaku.
func TestDraftRevisions(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	cp := set["cp"]
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	accept := func(reason string) Change {
		t.Helper()
		c, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: reason, DocumentIDs: []string{cp.ID}})
		if err != nil {
			t.Fatal(err)
		}
		if c, err = s.Verify(ctx, controller, c.ID, true, "OK"); err != nil {
			t.Fatal(err)
		}
		return c
	}
	revisions := func() []int {
		var out []int
		rows, err := e.Pool.Query(ctx, "SELECT revision FROM document_revisions WHERE document_id = $1 ORDER BY revision", cp.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var r int
			_ = rows.Scan(&r)
			out = append(out, r)
		}
		return out
	}

	// Dokumen baru sudah punya draft revisi awal; perubahan masuk ke sana.
	c := accept("Add AOI check")
	if d := c.Documents[0]; d.DraftRevision == nil || *d.DraftRevision != 0 || !d.Merged || d.ReleasedRevision != nil {
		t.Errorf("masuk ke draft terbuka: %+v", d)
	}
	if got := revisions(); !slices.Equal(got, []int{0}) {
		t.Errorf("revisi: %v", got)
	}

	// Revisi 0 dicatat terbit oleh dokumen kontrol (M5.5).
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	if _, err := e.Docs.Release(ctx, recorder, cp.ID); err != nil {
		t.Fatal(err)
	}
	// Tanpa draft terbuka, isi dokumen tidak bisa diubah: lewat aplikasi...
	_, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}})
	if got := problemCodes(t, err); !slices.Contains(got, "status:not_allowed") {
		t.Errorf("ubah dokumen terbit: %v", got)
	}
	// ...maupun langsung di database.
	if _, err := e.Pool.Exec(ctx, `INSERT INTO document_rows (document_id, position) VALUES ($1, 99)`, cp.ID); err == nil {
		t.Error("database menerima baris baru di dokumen terbit")
	}

	// Perubahan berikutnya membuka revisi 1 sebagai draft; revisi 0 tetap berlaku.
	c = accept("Change reflow profile")
	if d := c.Documents[0]; d.DraftRevision == nil || *d.DraftRevision != 1 || d.Merged || d.ReleasedRevision == nil || *d.ReleasedRevision != 0 {
		t.Errorf("draft baru: %+v", d)
	}
	doc, err := e.Docs.Get(ctx, e.Author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status != document.StatusDraft || doc.Revision != 1 || doc.ReleasedRevision == nil || *doc.ReleasedRevision != 0 {
		t.Errorf("dokumen: status %s, revisi %d, berlaku %v", doc.Status, doc.Revision, doc.ReleasedRevision)
	}
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Errorf("ubah draft revisi 1: %v", err)
	}
	var status string
	if err := e.Pool.QueryRow(ctx, "SELECT status FROM document_revisions WHERE document_id = $1 AND revision = 0", cp.ID).Scan(&status); err != nil || status != "released" {
		t.Errorf("revisi 0: %s, %v", status, err)
	}
	// Perubahan ketiga masuk ke draft revisi 1 yang masih terbuka.
	c = accept("Add label check")
	if d := c.Documents[0]; d.DraftRevision == nil || *d.DraftRevision != 1 || !d.Merged {
		t.Errorf("masuk ke draft 1: %+v", d)
	}
	if got := revisions(); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("revisi: %v", got)
	}
}
