package document

import (
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/template"
)

// D-130, D-156 · Kolom Relevant Document baris pertama setiap proses diisi
// dari master dokumen terkait, hanya bila selnya kosong.

func TestFillReferences(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set := f.newSet(t)
	refs := template.NewRefDocService(f.pool, access.NewChecker(f.pool))
	link := func(number, processID string, scope template.ScopeInput) {
		t.Helper()
		d, err := refs.Create(ctx, tpl.actor, template.NewRefDocument{Number: number, Title: "WI " + number})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := refs.AddLink(ctx, tpl.actor, d.ID, template.NewRefLink{ProcessID: processID, Scope: scope}); err != nil {
			t.Fatal(err)
		}
	}
	link("SW-ZZ-103", tpl.process, template.ScopeInput{})
	link("SW-MT-052", tpl.process, template.ScopeInput{Level: template.ScopeModel, TargetID: f.k125})
	link("SW-ZZ-999", f.process, template.ScopeInput{})
	link("SW-ZZ-500", tpl.process, template.ScopeInput{Level: template.ScopeModel, TargetID: f.k200}) // model lain: tidak ikut

	id := func(text string) string { return tpl.points[text] }
	cp, err := f.s.Compose(ctx, f.author, set["cp"].ID, Compose{
		VersionID: tpl.version.ID,
		Include:   []string{id("Wrong part received"), id("Check part number against PO"), id("Check MSL label"), id("Check warpage")},
	})
	if err != nil {
		t.Fatal(err)
	}
	relevant := fieldID(t, f.cpFormat, template.AreaRow, "Relevant Document")
	// Hanya baris pertama proses IQC yang terisi; urut nomor, satu per baris.
	if got := cp.Rows[0].Content[relevant]; got != "SW-MT-052\nSW-ZZ-103" {
		t.Errorf("baris pertama IQC: %q", got)
	}
	for _, r := range cp.Rows[1:] {
		if _, ok := r.Content[relevant]; ok {
			t.Errorf("baris lain ikut terisi: %+v", r)
		}
	}
	// Kolom itu bukan isi template, jadi baris tetap sama dengan template (M4.8).
	diffs, err := f.s.Differences(ctx, f.author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diffs {
		if d.Status != LinkSame {
			t.Errorf("baris tampil berbeda karena dokumen terkait: %+v", d)
		}
	}

	// Penyusun mengubah isinya; mengisi ulang tidak menimpanya.
	first := cp.Rows[0]
	content := map[string]any{}
	for k, v := range first.Content {
		content[k] = v
	}
	content[relevant] = "SW-ZZ-103 only"
	if cp, err = f.s.UpdateRow(ctx, f.author, cp.ID, first.ID, RowUpdate{ProcessID: first.ProcessID, Content: content, Version: first.Version}); err != nil {
		t.Fatal(err)
	}
	// Baris yang diketik sendiri untuk proses lain diisi lewat tombol isi.
	if cp, err = f.s.AddRow(ctx, f.author, cp.ID, NewRow{ProcessID: f.process}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.FillReferences(ctx, f.viewer, cp.ID); err == nil {
		t.Error("viewer mengisi dokumen terkait")
	}
	if cp, err = f.s.FillReferences(ctx, f.author, cp.ID); err != nil {
		t.Fatal(err)
	}
	if cp.Rows[0].Content[relevant] != "SW-ZZ-103 only" || cp.Rows[len(cp.Rows)-1].Content[relevant] != "SW-ZZ-999" {
		t.Errorf("sesudah isi ulang: %q, %q", cp.Rows[0].Content[relevant], cp.Rows[len(cp.Rows)-1].Content[relevant])
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "document.references_filled" || details != `{"rows":"1"}` {
		t.Errorf("audit: %s %s", action, details)
	}

	// Form tanpa kolom Relevant Document (PFMEA) menolak dengan jelas.
	if _, err := f.s.FillReferences(ctx, f.author, set["pfmea"].ID); !slices.Contains(codes(t, err), "format:not_allowed") {
		t.Errorf("PFMEA: %v", err)
	}
}
