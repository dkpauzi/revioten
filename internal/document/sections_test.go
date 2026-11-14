package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/template"
)

// D-155 pola 2 · Bagian per PCB (referensi D-159, SC-KT-040-00): satu set
// dokumen dengan satu nomor, berisi beberapa bagian dengan alur sendiri.

// pcbSet membuat set PFD, PFMEA, dan CP untuk model K-125 dengan tiga part
// number: PN-1 (PCB A), PN-2 dan PN-3 (Heatsink R dan L).
func (f fixture) pcbSet(t *testing.T) (map[string]Document, []string) {
	t.Helper()
	ctx := t.Context()
	parts := []string{f.pn1}
	for _, n := range []string{"PN-2", "PN-3"} {
		id, err := f.q.CreatePartNumber(ctx, db.CreatePartNumberParams{CustomerID: f.custAB, ModelID: f.k125, Number: n})
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, id)
	}
	reg, err := f.numbers.RegisterSet(ctx, f.author, numbering.NewSet{CustomerID: f.custAB, IdempotencyKey: "pcb-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Document{}
	for _, n := range reg.Numbers {
		if n.Status != "reserved" {
			continue
		}
		d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: n.ID, ModelIDs: []string{f.k125}, PartNumberIDs: parts})
		if err != nil {
			t.Fatal(err)
		}
		out[d.DocumentType.Key] = d
	}
	return out, parts
}

func sectionNames(d Document) []string {
	var out []string
	for _, s := range d.Sections {
		out = append(out, s.Name)
	}
	return out
}

func TestSections(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	set, parts := f.pcbSet(t)
	cp := set["cp"]
	// Satu baris sudah ada sebelum bagian dibuat.
	cp, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{OperationNumber: "10"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.s.AddSection(ctx, f.viewer, cp.ID, NewSection{Name: "PCB A"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer menambah bagian: %v", err)
	}
	cp, err = f.s.AddSection(ctx, f.author, cp.ID, NewSection{Name: " PCB A ", PartNumberIDs: []string{parts[0]}})
	if err != nil {
		t.Fatal(err)
	}
	pcbA := cp.Sections[0]
	if pcbA.Name != "PCB A" || pcbA.Position != 1 || len(pcbA.PartNumbers) != 1 || pcbA.PartNumbers[0].Number != "PN-1" {
		t.Errorf("bagian pertama: %+v", pcbA)
	}
	// Baris yang sudah ada masuk ke bagian pertama.
	if cp.Rows[0].SectionID != pcbA.ID {
		t.Errorf("baris lama: %+v", cp.Rows[0])
	}
	cp, err = f.s.AddSection(ctx, f.author, cp.ID, NewSection{Name: "Heatsink Assy (R/L)", PartNumberIDs: []string{parts[2], parts[1]}})
	if err != nil {
		t.Fatal(err)
	}
	heat := cp.Sections[1]
	if len(heat.PartNumbers) != 2 || heat.PartNumbers[0].Number != "PN-2" || heat.Key == pcbA.Key {
		t.Errorf("bagian kedua: %+v", heat)
	}
	// Bagian berlaku untuk seluruh set, dengan kunci yang sama (ASUMSI E-97).
	pfmea := f.get(t, set["pfmea"].ID)
	if !slices.Equal(sectionNames(pfmea), []string{"PCB A", "Heatsink Assy (R/L)"}) || pfmea.Sections[1].Key != heat.Key {
		t.Errorf("bagian PFMEA: %+v", pfmea.Sections)
	}

	// Penolakan.
	for name, c := range map[string]struct {
		in   NewSection
		want string
	}{
		"nama kembar":  {NewSection{Name: "PCB A"}, "name:taken"},
		"tanpa nama":   {NewSection{Name: " "}, "name:required"},
		"part lain":    {NewSection{Name: "X", PartNumberIDs: []string{"01900000-0000-7000-8000-000000000000"}}, "part_number_ids:invalid_format"},
		"part dipakai": {NewSection{Name: "Y", PartNumberIDs: []string{parts[0]}}, "part_number_ids:taken"},
	} {
		if _, err := f.s.AddSection(ctx, f.author, cp.ID, c.in); !slices.Contains(codes(t, err), c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}

	// Sesudah ada bagian, setiap baris baru wajib menyebut bagiannya.
	if _, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{}); !slices.Contains(codes(t, err), "section_id:required") {
		t.Errorf("baris tanpa bagian: %v", err)
	}
	if _, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{SectionID: pfmea.Sections[0].ID}); !slices.Contains(codes(t, err), "section_id:invalid_format") {
		t.Errorf("bagian dokumen lain: %v", err)
	}
	if cp, err = f.s.AddRows(ctx, f.author, cp.ID, []NewRow{{SectionID: heat.ID, ProcessID: f.process}, {SectionID: heat.ID}}); err != nil {
		t.Fatal(err)
	}

	// Urutan dan nomor operasi per bagian: Heatsink mulai lagi dari 10.
	var heatRows []string
	for _, r := range cp.Rows {
		if r.SectionID == heat.ID {
			heatRows = append(heatRows, r.ID)
		}
	}
	if _, err := f.s.Reorder(ctx, f.author, cp.ID, Order{SectionID: heat.ID, RowIDs: append([]string{cp.Rows[0].ID}, heatRows...)}); !slices.Contains(codes(t, err), "row_ids:invalid_format") {
		t.Errorf("urutan dengan baris bagian lain: %v", err)
	}
	cp, err = f.s.Reorder(ctx, f.author, cp.ID, Order{SectionID: heat.ID, RowIDs: []string{heatRows[1], heatRows[0]}, Renumber: true})
	if err != nil {
		t.Fatal(err)
	}
	var heatOps []string
	for _, r := range cp.Rows {
		if r.SectionID == heat.ID {
			heatOps = append(heatOps, r.OperationNumber)
		}
	}
	if !slices.Equal(heatOps, []string{"10", "20"}) || cp.Rows[0].OperationNumber != "10" || cp.Rows[0].SectionID != pcbA.ID {
		t.Errorf("nomor per bagian: %v, baris PCB A %+v", heatOps, cp.Rows[0])
	}

	// Ganti nama dan urutan: berlaku di semua dokumen set.
	name, position := "Heatsink Assy", 1
	if cp, err = f.s.UpdateSection(ctx, f.author, cp.ID, heat.ID, SectionUpdate{Name: name, Position: &position, PartNumberIDs: &[]string{parts[1]}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sectionNames(cp), []string{"Heatsink Assy", "PCB A"}) || len(cp.Sections[0].PartNumbers) != 1 {
		t.Errorf("sesudah ubah: %+v", cp.Sections)
	}
	if pfd := f.get(t, set["pfd"].ID); !slices.Equal(sectionNames(pfd), []string{"Heatsink Assy", "PCB A"}) {
		t.Errorf("PFD sesudah ubah: %v", sectionNames(pfd))
	}

	// Bagian berisi baris tidak bisa dihapus; bagian kosong bisa.
	if _, err := f.s.RemoveSection(ctx, f.author, cp.ID, pcbA.ID); !slices.Contains(codes(t, err), "section:not_allowed") {
		t.Errorf("hapus bagian berisi: %v", err)
	}
	if cp, err = f.s.AddSection(ctx, f.author, cp.ID, NewSection{Name: "Base Assy (R/L)"}); err != nil {
		t.Fatal(err)
	}
	if cp, err = f.s.RemoveSection(ctx, f.author, cp.ID, cp.Sections[2].ID); err != nil || len(cp.Sections) != 2 {
		t.Errorf("hapus bagian kosong: %+v, %v", cp.Sections, err)
	}
	var actions []string
	rows, err := f.pool.Query(ctx, "SELECT action FROM audit_log WHERE action LIKE 'document.section_%' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	rows.Close()
	if !slices.Equal(actions, []string{"document.section_added", "document.section_added", "document.section_changed", "document.section_added", "document.section_removed"}) {
		t.Errorf("audit: %v", actions)
	}
}

// TestComposeBySection: perakitan, pengecualian, dokumen terkait, dan cek
// PFMEA-CP berjalan per bagian. Poin yang sama boleh ada di dua bagian.
func TestComposeBySection(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set, _ := f.pcbSet(t)
	cp, err := f.s.AddSection(ctx, f.author, set["cp"].ID, NewSection{Name: "PCB A"})
	if err != nil {
		t.Fatal(err)
	}
	if cp, err = f.s.AddSection(ctx, f.author, cp.ID, NewSection{Name: "Heatsink Assy"}); err != nil {
		t.Fatal(err)
	}
	pcbA, heat := cp.Sections[0], cp.Sections[1]
	id := func(text string) string { return tpl.points[text] }

	// Tanpa bagian ditolak, karena set ini sudah punya bagian.
	if _, err := f.s.Compose(ctx, f.author, cp.ID, Compose{VersionID: tpl.version.ID, Include: []string{id("Check MSL label")}}); !slices.Contains(codes(t, err), "section_key:required") {
		t.Errorf("rakit tanpa bagian: %v", err)
	}
	if cp, err = f.s.Compose(ctx, f.author, cp.ID, Compose{
		VersionID: tpl.version.ID, SectionKey: pcbA.Key,
		Include: []string{id("Wrong part received"), id("Check part number against PO"), id("Check MSL label")},
		Exclude: []Exclusion{{PointID: id("Check HIC"), Reason: "No HIC on PCB A"}},
	}); err != nil {
		t.Fatal(err)
	}
	if cp, err = f.s.Compose(ctx, f.author, cp.ID, Compose{
		VersionID: tpl.version.ID, SectionKey: heat.Key,
		Include: []string{id("Check part number against PO"), id("Check HIC")},
	}); err != nil {
		t.Fatal(err)
	}
	count := func(d Document, sectionID string) int {
		n := 0
		for _, r := range d.Rows {
			if r.SectionID == sectionID {
				n++
			}
		}
		return n
	}
	if count(cp, pcbA.ID) != 2 || count(cp, heat.ID) != 2 {
		t.Errorf("baris CP per bagian: %d, %d", count(cp, pcbA.ID), count(cp, heat.ID))
	}
	pfmea := f.get(t, set["pfmea"].ID)
	if count(pfmea, pfmea.Sections[0].ID) != 1 || count(pfmea, pfmea.Sections[1].ID) != 0 {
		t.Errorf("baris PFMEA per bagian: %+v", pfmea.Rows)
	}

	// Keadaan poin per bagian: HIC dikecualikan di PCB A, dipakai di Heatsink.
	view, err := f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, pcbA.Key)
	if err != nil {
		t.Fatal(err)
	}
	if st := states(view); st["Check HIC"] != StateExcluded || st["Check MSL label"] != StateIncluded {
		t.Errorf("keadaan PCB A: %v", st)
	}
	if view, err = f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, heat.Key); err != nil {
		t.Fatal(err)
	}
	if st := states(view); st["Check HIC"] != StateIncluded || st["Check MSL label"] != StateUndecided {
		t.Errorf("keadaan Heatsink: %v", st)
	}

	// Dokumen terkait: baris pertama proses di setiap bagian.
	refs := template.NewRefDocService(f.pool, access.NewChecker(f.pool))
	doc, err := refs.Create(ctx, tpl.actor, template.NewRefDocument{Number: "SW-ZZ-103", Title: "Material Receiving"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := refs.AddLink(ctx, tpl.actor, doc.ID, template.NewRefLink{ProcessID: tpl.process}); err != nil {
		t.Fatal(err)
	}
	if cp, err = f.s.FillReferences(ctx, f.author, cp.ID); err != nil {
		t.Fatal(err)
	}
	relevant := fieldID(t, f.cpFormat, template.AreaRow, "Relevant Document")
	filled := 0
	for _, r := range cp.Rows {
		if r.Content[relevant] == "SW-ZZ-103" {
			filled++
		}
	}
	if filled != 2 {
		t.Errorf("baris berisi dokumen terkait: %d, ingin 2 (satu per bagian)", filled)
	}

	// Cek PFMEA-CP per bagian: tanda di PCB A tidak berlaku untuk Heatsink.
	checker := newActor(t, f.pool, "ppap.cek.bagian", access.AccountWork, all(access.RolePFMEACPChecker))
	c, err := f.s.MarkChecked(ctx, checker, cp.ID, pcbA.Key, tpl.process, true)
	if err != nil {
		t.Fatal(err)
	}
	var marked, unmarked int
	for _, g := range c.Processes {
		if g.ProcessID != tpl.process {
			continue
		}
		switch {
		case g.SectionKey == pcbA.Key && g.Check != nil:
			marked++
		case g.SectionKey == heat.Key && g.Check == nil:
			unmarked++
		}
	}
	if marked != 1 || unmarked != 1 {
		t.Errorf("tanda per bagian: %+v", c.Processes)
	}
	if _, err := f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, true); !slices.Contains(codes(t, err), "section_key:required") {
		t.Errorf("tanda tanpa bagian: %v", err)
	}
}

// TestCopyWithSections: menyalin dari set yang punya bagian membuat bagian
// yang sama (nama dan urutan, tanpa part number) di set tujuan.
func TestCopyWithSections(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	source, _ := f.pcbSet(t)
	src, err := f.s.AddSection(ctx, f.author, source["cp"].ID, NewSection{Name: "PCB A"})
	if err != nil {
		t.Fatal(err)
	}
	if src, err = f.s.AddSection(ctx, f.author, src.ID, NewSection{Name: "Base Assy"}); err != nil {
		t.Fatal(err)
	}
	if src, err = f.s.AddRows(ctx, f.author, src.ID, []NewRow{{SectionID: src.Sections[1].ID, OperationNumber: "10"}, {SectionID: src.Sections[0].ID, OperationNumber: "20"}}); err != nil {
		t.Fatal(err)
	}
	target := f.newSetFor(t, "salinan", f.k200)
	got, err := f.s.CopyFrom(ctx, f.author, target["cp"].ID, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sectionNames(got), []string{"PCB A", "Base Assy"}) || len(got.Rows) != 2 {
		t.Fatalf("salinan: %+v %+v", got.Sections, got.Rows)
	}
	if got.Rows[0].SectionID != got.Sections[1].ID || got.Rows[1].SectionID != got.Sections[0].ID {
		t.Errorf("baris salinan tidak masuk bagian yang sama namanya: %+v", got.Rows)
	}
	if pfmea := f.get(t, target["pfmea"].ID); pfmea.Sections[0].Key != got.Sections[0].Key {
		t.Errorf("kunci bagian set tujuan berbeda: %+v", pfmea.Sections)
	}
}
