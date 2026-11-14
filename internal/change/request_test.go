package change

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.1 · Entitas perubahan (R-CHG-01, D-113): satu perubahan punya
// mekanisme, pemicu, jenis, alasan, rujukan sumber, kategori 4M opsional, dan
// daftar dokumen terdampak. Belum ada verifikasi (M5.3).

// masters membaca ID master perubahan menurut namanya.
type masters struct {
	trigger, yokoten, content, layout, man string
}

func readMasters(t *testing.T, e doctest.Env) masters {
	t.Helper()
	ctx := t.Context()
	var m masters
	for query, dst := range map[string]*string{
		"SELECT id FROM change_sources WHERE name = 'Customer claim'": &m.trigger,
		"SELECT id FROM change_sources WHERE mechanism = 'yokoten'":   &m.yokoten,
		"SELECT id FROM change_types WHERE key = 'content'":           &m.content,
		"SELECT id FROM change_types WHERE key = 'layout'":            &m.layout,
		"SELECT id FROM four_m_categories WHERE key = 'man'":          &m.man,
	} {
		if err := e.Pool.QueryRow(ctx, query).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func docNumbers(c Change) []string {
	var out []string
	for _, d := range c.Documents {
		out = append(out, d.Number)
	}
	return out
}

func TestCreateChange(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	other := e.NewSet(t, "cd", e.CustomerCD, e.ModelB1)

	in := NewChange{
		CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, FourMCategoryID: m.man,
		Reason: " Add AOI check for false NG ", SourceReference: " Claim 2026-031 ",
		DocumentIDs: []string{set["pfmea"].ID, set["cp"].ID},
	}
	if _, err := s.Create(ctx, e.Viewer, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mengajukan: %v", err)
	}
	c, err := s.Create(ctx, e.Author, in)
	if err != nil {
		t.Fatal(err)
	}
	if c.Ref != "CHG-0001" || c.Mechanism != MechanismManual || c.Status != StatusSubmitted || c.Reason != "Add AOI check for false NG" ||
		c.SourceReference != "Claim 2026-031" || c.Trigger.Name != "Customer claim" || c.ChangeType.Key != KeyContent ||
		c.FourM == nil || c.FourM.Key != "man" || c.CustomerCode != "AB" || c.RequestedBy != "Orang penyusun.uji" {
		t.Errorf("perubahan baru: %+v", c)
	}
	// Satu perubahan menunjuk beberapa dokumen (kriteria selesai M5.1).
	if len(c.Documents) != 2 || c.Documents[0].DocumentType.Key != "pfmea" || c.Documents[1].DocumentType.Key != "cp" {
		t.Errorf("dokumen terdampak: %+v", c.Documents)
	}

	// Penolakan.
	bad := func(name string, change func(*NewChange), want string) {
		t.Helper()
		x := in
		x.DocumentIDs = slices.Clone(in.DocumentIDs)
		change(&x)
		_, err := s.Create(ctx, e.Author, x)
		if got := problemCodes(t, err); !slices.Contains(got, want) {
			t.Errorf("%s: %v", name, got)
		}
	}
	bad("tanpa alasan", func(x *NewChange) { x.Reason = " " }, "reason:required")
	bad("tanpa dokumen", func(x *NewChange) { x.DocumentIDs = nil }, "document_ids:required")
	bad("dokumen customer lain", func(x *NewChange) { x.DocumentIDs = []string{other["cp"].ID} }, "document_ids:invalid_format")
	bad("pemicu mekanisme", func(x *NewChange) { x.TriggerID = m.yokoten }, "trigger_id:not_allowed")
	bad("tanpa pemicu", func(x *NewChange) { x.TriggerID = "" }, "trigger_id:required")
	bad("jenis asing", func(x *NewChange) { x.ChangeTypeID = "01900000-0000-7000-8000-000000000000" }, "change_type_id:invalid_format")

	// Dokumen terdampak ditambah dan dilepas; dokumen yang sama tidak dobel.
	if c, err = s.AddDocument(ctx, e.Author, c.ID, set["pfd"].ID); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(docNumbers(c), []string{set["pfd"].Number, set["pfmea"].Number, set["cp"].Number}) {
		t.Errorf("sesudah tambah: %v", docNumbers(c))
	}
	if _, err := s.AddDocument(ctx, e.Author, c.ID, set["pfd"].ID); !slices.Contains(problemCodes(t, err), "document_id:taken") {
		t.Errorf("dokumen dobel: %v", err)
	}
	if c, err = s.RemoveDocument(ctx, e.Author, c.ID, set["pfd"].ID); err != nil || len(c.Documents) != 2 {
		t.Errorf("lepas dokumen: %+v, %v", c.Documents, err)
	}
	// Dokumen terakhir tidak bisa dilepas: perubahan selalu menunjuk dokumen.
	if c, err = s.RemoveDocument(ctx, e.Author, c.ID, set["pfmea"].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoveDocument(ctx, e.Author, c.ID, set["cp"].ID); !slices.Contains(problemCodes(t, err), "document_id:not_allowed") {
		t.Errorf("lepas dokumen terakhir: %v", err)
	}

	// Daftar per customer; akun tanpa hak melihat ditolak.
	second, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.layout, Reason: "Layout", DocumentIDs: []string{set["pfd"].ID}})
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, e.Author, e.CustomerAB)
	if err != nil || len(list) != 2 || list[0].Ref != second.Ref || list[1].Ref != "CHG-0001" {
		t.Errorf("daftar: %+v, %v", list, err)
	}
	if _, err := s.List(ctx, e.Viewer, e.CustomerAB); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer melihat: %v", err)
	}
	// Daftar perubahan satu dokumen.
	byDoc, err := s.ForDocument(ctx, e.Author, set["pfd"].ID)
	if err != nil || len(byDoc) != 1 || byDoc[0].ID != second.ID {
		t.Errorf("perubahan dokumen PFD: %+v, %v", byDoc, err)
	}

	var actions []string
	rows, err := e.Pool.Query(ctx, "SELECT action FROM audit_log WHERE target_type = 'change' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	rows.Close()
	want := []string{"change.created", "change.document_added", "change.document_removed", "change.document_removed", "change.created"}
	if !slices.Equal(actions, want) {
		t.Errorf("audit: %v", actions)
	}
}
