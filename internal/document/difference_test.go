package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/template"
)

// M4.8 · Perbedaan lokal dan putus tautan (R-CMP-04): baris hasil perakitan
// yang diubah ditandai berbeda dari template, dan memutus tautan butuh alasan.

// diffOf mencari perbandingan satu baris.
func diffOf(t *testing.T, list []RowDifference, rowID string) RowDifference {
	t.Helper()
	i := slices.IndexFunc(list, func(d RowDifference) bool { return d.RowID == rowID })
	if i < 0 {
		t.Fatalf("baris %s tidak ada di perbandingan", rowID)
	}
	return list[i]
}

func TestLocalDifferences(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set := f.newSet(t)
	id := func(text string) string { return tpl.points[text] }
	cp, err := f.s.Compose(ctx, f.author, set["cp"].ID, Compose{
		VersionID: tpl.version.ID,
		Include:   []string{id("Check part number against PO"), id("Check MSL label"), id("Check warpage")},
	})
	if err != nil {
		t.Fatal(err)
	}
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")
	method := fieldID(t, f.cpFormat, template.AreaRow, "Method")
	frequency := fieldID(t, f.cpFormat, template.AreaRow, "Frequency")
	if cp, err = f.s.AddRow(ctx, f.author, cp.ID, NewRow{Content: map[string]any{product: "Typed"}}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.s.Differences(ctx, f.viewer, cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa hak melihat perbandingan: %v", err)
	}
	// Baru dirakit: tiga baris tertaut sama dengan template. Baris yang
	// diketik sendiri tidak ikut dibandingkan. Varian K-125 yang dibandingkan.
	diffs, err := f.s.Differences(ctx, f.author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 3 {
		t.Fatalf("jumlah perbandingan: %d", len(diffs))
	}
	for _, d := range diffs {
		if d.Status != LinkSame || len(d.Fields) != 0 {
			t.Errorf("baris baru dirakit: %+v", d)
		}
	}
	if diffs[2].PointText != "Check warpage (K-125)" {
		t.Errorf("teks poin: %q", diffs[2].PointText)
	}

	// Perubahan lokal: satu kolom diganti, satu kolom template dikosongkan,
	// dan satu kolom yang kosong di template diisi.
	first := cp.Rows[0]
	if cp, err = f.s.UpdateRow(ctx, f.author, cp.ID, first.ID, RowUpdate{
		ProcessID: first.ProcessID, Version: first.Version,
		Content: map[string]any{product: "Part number and revision", frequency: "Each lot"},
	}); err != nil {
		t.Fatal(err)
	}
	if diffs, err = f.s.Differences(ctx, f.author, cp.ID); err != nil {
		t.Fatal(err)
	}
	d := diffOf(t, diffs, first.ID)
	if d.Status != LinkDifferent || len(d.Fields) != 3 {
		t.Fatalf("baris yang diubah: %+v", d)
	}
	byField := map[string]FieldDifference{}
	for _, fd := range d.Fields {
		byField[fd.FieldID] = fd
	}
	if fd := byField[product]; fd.Template != "Part number" || fd.Row != "Part number and revision" || fd.Label != "Product Characteristics" {
		t.Errorf("kolom diganti: %+v", fd)
	}
	if fd := byField[method]; fd.Template != "Visual" || fd.Row != "" {
		t.Errorf("kolom dikosongkan: %+v", fd)
	}
	if fd := byField[frequency]; fd.Template != "" || fd.Row != "Each lot" {
		t.Errorf("kolom diisi: %+v", fd)
	}
	if other := diffOf(t, diffs, cp.Rows[1].ID); other.Status != LinkSame {
		t.Errorf("baris lain: %+v", other)
	}

	// Proses yang diganti juga perbedaan.
	second := cp.Rows[1]
	if cp, err = f.s.UpdateRow(ctx, f.author, cp.ID, second.ID, RowUpdate{ProcessID: f.process, Content: second.Content, Version: second.Version}); err != nil {
		t.Fatal(err)
	}
	diffs, _ = f.s.Differences(ctx, f.author, cp.ID)
	if d := diffOf(t, diffs, second.ID); d.Status != LinkDifferent || len(d.Fields) != 1 || d.Fields[0].Template != "IQC" || d.Fields[0].Row != "SMT Printing" {
		t.Errorf("proses diganti: %+v", d)
	}

	// Putus tautan butuh alasan, dan hanya untuk baris yang tertaut.
	if _, err := f.s.Detach(ctx, f.viewer, cp.ID, first.ID, "x"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer memutus tautan: %v", err)
	}
	if _, err := f.s.Detach(ctx, f.author, cp.ID, first.ID, "  "); !slices.Contains(codes(t, err), "reason:required") {
		t.Errorf("tanpa alasan: %v", err)
	}
	typed := cp.Rows[3]
	if _, err := f.s.Detach(ctx, f.author, cp.ID, typed.ID, "x"); !slices.Contains(codes(t, err), "row:not_allowed") {
		t.Errorf("baris yang tidak tertaut: %v", err)
	}
	cp, err = f.s.Detach(ctx, f.author, cp.ID, first.ID, " Customer drawing asks for the revision ")
	if err != nil {
		t.Fatal(err)
	}
	if r := cp.Rows[0]; !r.Detached || r.DetachReason != "Customer drawing asks for the revision" || r.TemplatePointID == "" {
		t.Errorf("baris sesudah putus tautan: %+v", r)
	}
	if _, err := f.s.Detach(ctx, f.author, cp.ID, first.ID, "again"); !slices.Contains(codes(t, err), "row:not_allowed") {
		t.Errorf("putus dua kali: %v", err)
	}
	diffs, _ = f.s.Differences(ctx, f.author, cp.ID)
	if d := diffOf(t, diffs, first.ID); d.Status != LinkDetached || d.Reason != "Customer drawing asks for the revision" || len(d.Fields) != 0 {
		t.Errorf("perbandingan baris yang putus: %+v", d)
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "document.row_detached" || details != `{"position":"1","reason":"Customer drawing asks for the revision"}` {
		t.Errorf("audit: %s %s", action, details)
	}
}
