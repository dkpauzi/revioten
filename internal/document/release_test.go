package document_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// M5.5 · Snapshot rilis (R-REV-02, R-APR-02): isi dibekukan dengan SHA-256,
// revisi terbit tidak bisa diubah, dan pencatat rilis bukan penyusun draft.

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

func TestRelease(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	// Orang yang menyusun dan juga dokumen kontrol (D-68).
	both := doctest.NewActor(t, e.Pool, "ppap.dcc", "work", doctest.All(access.RoleAuthor), doctest.All(access.RoleDocumentControl))
	if _, err := e.Docs.AddRow(ctx, both, cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Fatal(err)
	}

	if _, err := e.Docs.Release(ctx, e.Author, cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa E-7: %v", err)
	}
	if _, err := e.Docs.Release(ctx, both, cp.ID); codeOf(t, err) != "recorder:not_allowed" {
		t.Errorf("penyusun mencatat rilisnya sendiri: %v", err)
	}
	d, err := e.Docs.Release(ctx, recorder, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != document.StatusReleased || d.ReleasedRevision == nil || *d.ReleasedRevision != 0 {
		t.Errorf("terbit: %s %v", d.Status, d.ReleasedRevision)
	}
	revs, err := e.Docs.Revisions(ctx, e.Author, cp.ID)
	if err != nil || len(revs) != 1 || revs[0].Status != "released" || revs[0].ReleasedBy != "Orang dcc.uji" || len(revs[0].SHA256) != 64 {
		t.Fatalf("revisi: %+v, %v", revs, err)
	}
	// Hash di database cocok dengan isi snapshot yang tersimpan.
	var snapshot string
	if err := e.Pool.QueryRow(ctx, "SELECT snapshot FROM document_revisions WHERE document_id = $1 AND revision = 0", cp.ID).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256([]byte(snapshot)); hex.EncodeToString(sum[:]) != revs[0].SHA256 {
		t.Error("hash tidak cocok dengan snapshot")
	}

	// Revisi terbit tidak bisa diubah atau dihapus, juga lewat SQL langsung.
	for _, sql := range []string{
		"UPDATE document_revisions SET snapshot = snapshot WHERE document_id = $1 AND revision = 0",
		"DELETE FROM document_revisions WHERE document_id = $1 AND revision = 0",
	} {
		if _, err := e.Pool.Exec(ctx, sql, cp.ID); err == nil {
			t.Errorf("database menerima: %s", sql)
		}
	}
	// Tanpa draft terbuka: tidak bisa diubah, dan tidak bisa dirilis lagi.
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}}); codeOf(t, err) != "status:not_allowed" {
		t.Errorf("ubah dokumen terbit: %v", err)
	}
	if _, err := e.Docs.Release(ctx, recorder, cp.ID); codeOf(t, err) != "status:not_allowed" {
		t.Errorf("rilis tanpa draft: %v", err)
	}

	// Draft revisi 1 (biasanya dibuka perubahan yang diterima, M5.4). Orang
	// yang menyusun revisi 0 tetapi tidak menyusun revisi 1 boleh mencatatnya.
	if _, err := e.Pool.Exec(ctx, `INSERT INTO document_revisions (document_id, revision, created_by_person_id) VALUES ($1, 1, $2)`, cp.ID, recorder.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE documents SET status = 'draft', revision = 1 WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if d, err = e.Docs.Release(ctx, both, cp.ID); err != nil || *d.ReleasedRevision != 1 {
		t.Fatalf("rilis revisi 1: %v", err)
	}
	if revs, _ = e.Docs.Revisions(ctx, e.Author, cp.ID); len(revs) != 2 || revs[0].Revision != 1 || revs[0].SHA256 == revs[1].SHA256 {
		t.Errorf("dua revisi: %+v", revs)
	}
}
