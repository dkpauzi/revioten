package change

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.7 · Daftar periksa dampak (R-CHG-05): dokumen terkait, keputusan, dan
// alasan tersimpan di perubahan.
func TestImpactChecklist(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	c, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "False NG on AOI", DocumentIDs: []string{set["pfmea"].ID}})
	if err != nil {
		t.Fatal(err)
	}
	items := []ImpactInput{
		{Kind: "pfmea", DocumentReference: set["pfmea"].Number, Decision: ImpactAffected},
		{Kind: "cp", DocumentReference: set["cp"].Number, Decision: ImpactNotAffected, Reason: "Control method stays the same"},
		{Kind: "wi", DocumentReference: "SW-ZZ-116", Decision: ImpactAffected, Reason: "AOI program note"},
	}
	if _, err := s.SaveImpacts(ctx, e.Viewer, c.ID, items); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
	// Tidak terdampak tanpa alasan, dan jenis yang tidak dikenal, ditolak.
	for name, x := range map[string]struct {
		in   ImpactInput
		want string
	}{
		"tanpa alasan":  {ImpactInput{Kind: "cp", Decision: ImpactNotAffected}, "items.0.reason:required"},
		"jenis asing":   {ImpactInput{Kind: "drawing", Decision: ImpactAffected}, "items.0.kind:invalid_format"},
		"tanpa putusan": {ImpactInput{Kind: "wi"}, "items.0.decision:required"},
	} {
		_, err := s.SaveImpacts(ctx, e.Author, c.ID, []ImpactInput{x.in})
		if got := problemCodes(t, err); !slices.Contains(got, x.want) {
			t.Errorf("%s: %v", name, got)
		}
	}
	c, err = s.SaveImpacts(ctx, e.Author, c.ID, items)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Impacts) != 3 || c.Impacts[1].Decision != ImpactNotAffected || c.Impacts[1].Reason != "Control method stays the same" || c.Impacts[2].DocumentReference != "SW-ZZ-116" || c.Impacts[0].UpdatedBy != "Orang penyusun.uji" {
		t.Errorf("daftar: %+v", c.Impacts)
	}
	// Disimpan ulang secara utuh: baris yang tidak dikirim hilang.
	if c, err = s.SaveImpacts(ctx, e.Author, c.ID, items[:1]); err != nil || len(c.Impacts) != 1 {
		t.Errorf("simpan ulang: %+v, %v", c.Impacts, err)
	}

	// Sesudah diterima daftar masih bisa diisi; sesudah ditolak tidak.
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	if _, err := s.Verify(ctx, controller, c.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveImpacts(ctx, e.Author, c.ID, items); err != nil {
		t.Errorf("sesudah diterima: %v", err)
	}
	r, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "Other", DocumentIDs: []string{set["cp"].ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(ctx, controller, r.ID, false, "No 8D"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveImpacts(ctx, e.Author, r.ID, items); !slices.Contains(problemCodes(t, err), "status:not_allowed") {
		t.Errorf("sesudah ditolak: %v", err)
	}
}

// TestImpactFromMaster: WI dipilih dari master dokumen terkait, dan nomor,
// revisi, serta judulnya disalin dari master (D-168, R-DOC-11).
func TestImpactFromMaster(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	c, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "AOI", DocumentIDs: []string{set["cp"].ID}})
	if err != nil {
		t.Fatal(err)
	}
	var wi string
	if err := e.Pool.QueryRow(ctx, `INSERT INTO referenced_documents (number, title, revision) VALUES ('SW-ZZ-116', 'AOI program setting', '02') RETURNING id`).Scan(&wi); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveImpacts(ctx, e.Author, c.ID, []ImpactInput{{Kind: "wi", ReferencedDocumentID: "01900000-0000-7000-8000-000000000000", Decision: ImpactAffected}}); !slices.Contains(problemCodes(t, err), "items.0.referenced_document_id:invalid_format") {
		t.Errorf("dokumen master tidak ada: %v", err)
	}
	c, err = s.SaveImpacts(ctx, e.Author, c.ID, []ImpactInput{
		{Kind: "wi", ReferencedDocumentID: wi, DocumentReference: "diketik", Decision: ImpactAffected},
		{Kind: "inspection_standard", DocumentReference: "IS-AB-01", Decision: ImpactNotAffected, Reason: "No new item"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if x := c.Impacts[0]; x.ReferencedDocumentID != wi || x.DocumentReference != "SW-ZZ-116 rev 02 · AOI program setting" {
		t.Errorf("dari master: %+v", x)
	}
	if x := c.Impacts[1]; x.ReferencedDocumentID != "" || x.DocumentReference != "IS-AB-01" {
		t.Errorf("ditulis bebas: %+v", x)
	}
}
