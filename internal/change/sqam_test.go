package change

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.10 · Perubahan surut SQAM (R-CHG-01, R-TPL-07): satu perubahan untuk
// semua dokumen aktif customer; saat diterima semua mendapat draft baru dan
// revisi terbit tidak ditulis ulang.
func TestSQAMChange(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	k125 := e.NewSet(t, "k125", e.CustomerAB, e.ModelK125)
	k200 := e.NewSet(t, "k200", e.CustomerAB, e.ModelK200)
	e.NewSet(t, "b1", e.CustomerCD, e.ModelB1) // customer lain tidak ikut
	// PFD K-200 sudah usang: tidak ikut.
	if _, err := e.Pool.Exec(ctx, `UPDATE document_numbers SET status = 'obsolete' WHERE number = $1`, k200["pfd"].Number); err != nil {
		t.Fatal(err)
	}
	// CP K-125 sudah terbit di revisi 0.
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	if _, err := e.Docs.Release(ctx, recorder, k125["cp"].ID); err != nil {
		t.Fatal(err)
	}
	var snapshot string
	if err := e.Pool.QueryRow(ctx, "SELECT snapshot_sha256 FROM document_revisions WHERE document_id = $1 AND revision = 0", k125["cp"].ID).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}

	date := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	in := SQAMChange{CustomerID: e.CustomerAB, ChangeTypeID: m.content, Reason: "SQAM rev 7 asks for the reaction plan owner", SQAMReference: "SQAM-AB rev 7", EffectiveDate: &date}
	if _, err := s.SubmitSQAM(ctx, e.Viewer, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
	bad := in
	bad.SQAMReference, bad.EffectiveDate = "", nil
	if got := problemCodes(t, func() error { _, err := s.SubmitSQAM(ctx, e.Author, bad); return err }()); !slices.Contains(got, "sqam_reference:required") || !slices.Contains(got, "effective_date:required") {
		t.Errorf("tanpa rujukan dan tanggal: %v", got)
	}
	c, err := s.SubmitSQAM(ctx, e.Author, in)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mechanism != MechanismSQAM || c.Trigger != nil || c.SourceReference != "SQAM-AB rev 7" || c.EffectiveDate == nil || !c.EffectiveDate.Equal(date) || len(c.Documents) != 5 {
		t.Fatalf("perubahan SQAM: %+v", c)
	}
	for _, d := range c.Documents {
		if d.ID == k200["pfd"].ID {
			t.Error("dokumen usang ikut")
		}
	}

	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	if c, err = s.Verify(ctx, controller, c.ID, true, "Applies from December"); err != nil {
		t.Fatal(err)
	}
	// Semua dokumen aktif mendapat draft: CP terbit membuka revisi 1, yang
	// lain menerima perubahan ke draft revisi 0 yang masih terbuka.
	for _, d := range c.Documents {
		if d.DraftRevision == nil {
			t.Errorf("%s tanpa draft", d.Number)
			continue
		}
		if d.ID == k125["cp"].ID {
			if *d.DraftRevision != 1 || d.Merged || d.ReleasedRevision == nil || *d.ReleasedRevision != 0 {
				t.Errorf("CP terbit: %+v", d)
			}
		} else if *d.DraftRevision != 0 || !d.Merged {
			t.Errorf("%s: %+v", d.Number, d)
		}
	}
	// Revisi terbit tidak ditulis ulang.
	var after string
	if err := e.Pool.QueryRow(ctx, "SELECT snapshot_sha256 FROM document_revisions WHERE document_id = $1 AND revision = 0", k125["cp"].ID).Scan(&after); err != nil || after != snapshot {
		t.Errorf("snapshot revisi 0 berubah: %v", err)
	}
	// Database menolak perubahan SQAM tanpa tanggal berlaku.
	if _, err := e.Pool.Exec(ctx, `UPDATE changes SET effective_date = NULL WHERE id = $1`, c.ID); err == nil {
		t.Error("database menerima perubahan SQAM tanpa tanggal berlaku")
	}
}
