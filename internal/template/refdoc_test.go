package template

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// D-130, D-156 · Master dokumen terkait: WI, SOP, QEP, dan sejenisnya dicatat
// sekali, lalu ditautkan ke proses beserta cakupannya. Untuk satu model,
// semua tautan yang berlaku dijumlahkan.

func TestReferencedDocuments(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	s := NewRefDocService(f.pool, access.NewChecker(f.pool))
	receiving := f.process(t, "Receiving")
	iqc := f.process(t, "IQC")

	// Hanya Pengelola template (E-6) yang mengubah, seperti master proses (D-64).
	in := NewRefDocument{Number: " sw-zz-103 ", Title: " Material Receiving ", Level: "WI (Level 3)", Revision: "19", RevisionDate: "2026-06-26"}
	if _, err := s.Create(ctx, f.admin, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("admin membuat dokumen terkait: %v", err)
	}
	doc, err := s.Create(ctx, f.maintainer, in)
	if err != nil {
		t.Fatal(err)
	}
	// Nomor dirapikan menjadi huruf besar; teks lain dipangkas.
	if doc.Number != "SW-ZZ-103" || doc.Title != "Material Receiving" || doc.RevisionDate != "2026-06-26" || !doc.Active || len(doc.Links) != 0 {
		t.Errorf("dokumen baru: %+v", doc)
	}
	if _, err := s.Create(ctx, f.maintainer, NewRefDocument{Number: "SW-ZZ-103", Title: "Lagi"}); !slices.Equal(problemCodes(t, err), []string{"number:taken"}) {
		t.Errorf("nomor kembar: %v", err)
	}
	for name, bad := range map[string]NewRefDocument{
		"tanpa nomor":    {Title: "x"},
		"nomor aneh":     {Number: "SW ZZ/103", Title: "x"},
		"tanpa judul":    {Number: "SW-ZZ-900"},
		"tanggal salah":  {Number: "SW-ZZ-901", Title: "x", RevisionDate: "26-06-2026"},
		"revisi panjang": {Number: "SW-ZZ-902", Title: "x", Revision: "12345678901"},
	} {
		if _, err := s.Create(ctx, f.maintainer, bad); err == nil {
			t.Errorf("%s diterima", name)
		}
	}

	// Tautan ke proses: umum, customer, dan model.
	if doc, err = s.AddLink(ctx, f.maintainer, doc.ID, NewRefLink{ProcessID: receiving}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddLink(ctx, f.maintainer, doc.ID, NewRefLink{ProcessID: receiving}); !slices.Equal(problemCodes(t, err), []string{"process_id:taken"}) {
		t.Errorf("tautan kembar: %v", err)
	}
	if _, err := s.AddLink(ctx, f.maintainer, doc.ID, NewRefLink{ProcessID: "01900000-0000-7000-8000-000000000000"}); !slices.Equal(problemCodes(t, err), []string{"process_id:invalid_format"}) {
		t.Errorf("proses asing: %v", err)
	}
	other, err := s.Create(ctx, f.maintainer, NewRefDocument{Number: "SW-ZZ-318", Title: "Incoming SAP System Barcode"})
	if err != nil {
		t.Fatal(err)
	}
	if other, err = s.AddLink(ctx, f.maintainer, other.ID, NewRefLink{ProcessID: receiving, Scope: ScopeInput{Level: ScopeCustomer, TargetID: f.custA}}); err != nil {
		t.Fatal(err)
	}
	special, err := s.Create(ctx, f.maintainer, NewRefDocument{Number: "SW-MT-052", Title: "Model WI"})
	if err != nil {
		t.Fatal(err)
	}
	if special, err = s.AddLink(ctx, f.maintainer, special.ID, NewRefLink{ProcessID: receiving, Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}}); err != nil {
		t.Fatal(err)
	}
	if special, err = s.AddLink(ctx, f.maintainer, special.ID, NewRefLink{ProcessID: iqc}); err != nil {
		t.Fatal(err)
	}
	if len(special.Links) != 2 || special.Links[0].Process.Name == "" {
		t.Errorf("tautan dokumen model: %+v", special.Links)
	}

	// Untuk satu model, semua tautan yang berlaku dijumlahkan, urut nomor.
	numbers := func(processID, modelID string) []string {
		t.Helper()
		m, err := f.q.GetModel(ctx, modelID)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ReferencedNumbers(ctx, f.q, processID, m.ID, m.CustomerID, derefText(m.ModelGroupID))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := numbers(receiving, f.modelX); !slices.Equal(got, []string{"SW-MT-052", "SW-ZZ-103", "SW-ZZ-318"}) {
		t.Errorf("Receiving K-125: %v", got)
	}
	if got := numbers(receiving, f.modelLone); !slices.Equal(got, []string{"SW-ZZ-103", "SW-ZZ-318"}) {
		t.Errorf("Receiving K-200: %v", got)
	}
	if got := numbers(receiving, f.modelB); !slices.Equal(got, []string{"SW-ZZ-103"}) {
		t.Errorf("Receiving customer lain: %v", got)
	}

	// Dokumen nonaktif tidak dipakai lagi; tautan dihapus juga tidak.
	inactive := false
	if _, err := s.Update(ctx, f.maintainer, other.ID, RefDocumentUpdate{Title: other.Title, Revision: "05", Active: &inactive}); err != nil {
		t.Fatal(err)
	}
	// Tautan urut menurut nama proses: IQC lalu Receiving. Yang dilepas tautan Receiving.
	if special.Links[1].Process.ID != receiving {
		t.Fatalf("urutan tautan: %+v", special.Links)
	}
	if special, err = s.RemoveLink(ctx, f.maintainer, special.ID, special.Links[1].ID); err != nil || len(special.Links) != 1 {
		t.Fatalf("hapus tautan: %+v, %v", special.Links, err)
	}
	if got := numbers(receiving, f.modelX); !slices.Equal(got, []string{"SW-ZZ-103"}) {
		t.Errorf("sesudah nonaktif dan hapus tautan: %v", got)
	}

	// Semua akun bisa melihat (L-2).
	list, err := s.List(ctx, f.author)
	if err != nil || len(list) != 3 || list[0].Number != "SW-MT-052" || list[2].Active {
		t.Errorf("daftar: %+v, %v", list, err)
	}
	var actions []string
	rows, err := f.pool.Query(ctx, "SELECT action FROM audit_log WHERE action LIKE 'referenced_document.%' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	rows.Close()
	want := []string{"referenced_document.created", "referenced_document.linked", "referenced_document.created", "referenced_document.linked",
		"referenced_document.created", "referenced_document.linked", "referenced_document.linked", "referenced_document.updated", "referenced_document.unlinked"}
	if !slices.Equal(actions, want) {
		t.Errorf("audit: %v", actions)
	}
}

// TestPointContentRejectsRelevantDocument (D-130): kolom Relevant Document
// tidak pernah disimpan di poin template.
func TestPointContentRejectsRelevantDocument(t *testing.T) {
	if _, p := cleanPointContent(map[string]any{KeyRelevantDocument: "SW-ZZ-103"}); p == nil || p.Field != "content" {
		t.Errorf("relevant_document di poin diterima: %+v", p)
	}
}
