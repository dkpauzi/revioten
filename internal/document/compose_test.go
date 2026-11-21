package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// M4.5 · Perakitan dari process template (R-CMP-01 sampai R-CMP-04, D-146).

// iqc adalah process template IQC untuk test perakitan.
type iqc struct {
	version template.ProcessVersion // versi 1, sudah terbit
	process string
	points  map[string]string // teks poin menjadi ID poin
	svc     *template.ProcessService
	actor   access.Actor
}

// newIQC membuat process template IQC dengan sub-proses General, MSD, PCB,
// dan Wireharness. Poin "Check warpage" punya varian untuk model K-125.
func (f fixture) newIQC(t *testing.T) iqc {
	t.Helper()
	ctx := t.Context()
	maintainer := newActor(t, f.pool, "ppap.engineer", access.AccountWork, all(access.RoleTemplateMaintainer))
	svc := template.NewProcessService(f.pool, access.NewChecker(f.pool))
	process, err := f.q.CreateProcess(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: process, Kind: "display", Name: "IQC"}); err != nil {
		t.Fatal(err)
	}
	tpl, err := svc.CreateProcessTemplate(ctx, maintainer, process)
	if err != nil {
		t.Fatal(err)
	}
	versionID := tpl.Versions[0].ID
	types := map[string]string{}
	for _, key := range []string{"pfd", "pfmea", "cp"} {
		var id string
		if err := f.pool.QueryRow(ctx, "SELECT id FROM document_types WHERE key = $1", key).Scan(&id); err != nil {
			t.Fatal(err)
		}
		types[key] = id
	}
	var v template.ProcessVersion
	type point struct {
		kind, text string
		content    map[string]any
	}
	add := func(subprocess string, points ...point) {
		if v, err = svc.AddSubprocess(ctx, maintainer, versionID, subprocess); err != nil {
			t.Fatal(err)
		}
		sub := v.Subprocesses[len(v.Subprocesses)-1]
		for _, p := range points {
			v, err = svc.AddPoint(ctx, maintainer, versionID, template.NewPoint{SubprocessID: sub.ID, DocumentTypeID: types[p.kind], Text: p.text, Content: p.content})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	add("IQC General",
		point{"pfmea", "Wrong part received", map[string]any{"failure_mode": "Wrong part received", "failure_effect": "Line stop", "severity": 7, "work_element": "man"}},
		point{"cp", "Check part number against PO", map[string]any{"product_characteristic": "Part number", "control_method": "Visual"}},
	)
	add("IQC MSD",
		point{"pfmea", "Floor life exceeded", map[string]any{"failure_mode": "Floor life exceeded"}},
		point{"cp", "Check MSL label", map[string]any{"product_characteristic": "MSL label"}},
		point{"cp", "Check HIC", map[string]any{"product_characteristic": "Humidity indicator card"}},
	)
	add("IQC PCB",
		point{"pfmea", "Warped PCB", map[string]any{"failure_mode": "Warped PCB"}},
		point{"cp", "Check warpage", map[string]any{"product_characteristic": "Warpage"}},
	)
	add("IQC Wireharness", point{"cp", "Check crimp height", nil})

	out := iqc{process: process, points: map[string]string{}, svc: svc, actor: maintainer}
	for _, s := range v.Subprocesses {
		for _, p := range s.Points {
			out.points[p.Text] = p.ID
		}
	}
	if _, err := svc.AddVariant(ctx, maintainer, versionID, out.points["Check warpage"], template.NewVariant{
		Scope: template.ScopeInput{Level: template.ScopeModel, TargetID: f.k125}, Text: "Check warpage (K-125)",
		Content: map[string]any{"product_characteristic": "Warpage max 0.5%"},
	}); err != nil {
		t.Fatal(err)
	}
	if out.version, err = svc.PublishProcessVersion(ctx, maintainer, versionID); err != nil {
		t.Fatal(err)
	}
	return out
}

// newSet membuat dokumen PFD, PFMEA, dan CP satu set untuk model K-125.
func (f fixture) newSet(t *testing.T) map[string]Document {
	t.Helper()
	ctx := t.Context()
	reg, err := f.numbers.RegisterSet(ctx, f.author, numbering.NewSet{CustomerID: f.custAB, IdempotencyKey: "set-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Document{}
	for _, n := range reg.Numbers {
		if n.Status != "reserved" {
			continue
		}
		d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: n.ID, ModelIDs: []string{f.k125}})
		if err != nil {
			t.Fatal(err)
		}
		out[d.DocumentType.Key] = d
	}
	return out
}

func (f fixture) get(t *testing.T, id string) Document {
	t.Helper()
	d, err := f.s.Get(t.Context(), f.author, id)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// states memberi keadaan setiap poin di tampilan perakitan, menurut teksnya.
func states(c Composition) map[string]string {
	out := map[string]string{}
	for _, s := range c.Subprocesses {
		for _, p := range s.Points {
			out[p.Text] = p.State
		}
	}
	return out
}

// TestCompose: kriteria selesai M4.5. IQC General, MSD, dan PCB dengan
// sebagian poin dicoret menghasilkan baris PFD, PFMEA, dan CP yang benar.
func TestCompose(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set := f.newSet(t)
	cp := set["cp"]

	// Sebelum dirakit semua poin belum diputuskan. Varian K-125 yang tampil.
	view, err := f.s.Composition(ctx, f.viewer, cp.ID, tpl.version.ID, "")
	if !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa hak melihat perakitan: %v", err)
	}
	if view, err = f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, ""); err != nil {
		t.Fatal(err)
	}
	st := states(view)
	if len(st) != 8 || st["Check warpage (K-125)"] != StateUndecided || st["Wrong part received"] != StateUndecided {
		t.Fatalf("keadaan awal: %v", st)
	}

	id := func(text string) string { return tpl.points[text] }
	in := Compose{
		VersionID: tpl.version.ID,
		Include: []string{
			id("Check warpage"), id("Warped PCB"), // sengaja tidak urut: urutan baris mengikuti template
			id("Wrong part received"), id("Check part number against PO"),
			id("Floor life exceeded"), id("Check MSL label"),
		},
		Exclude: []Exclusion{{PointID: id("Check HIC"), Reason: " No HIC for this part "}},
	}
	if _, err := f.s.Compose(ctx, f.viewer, cp.ID, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer merakit: %v", err)
	}
	got, err := f.s.Compose(ctx, f.author, cp.ID, in)
	if err != nil {
		t.Fatal(err)
	}

	// CP: tiga baris menurut urutan template, tertaut ke poin asal, proses IQC.
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")
	method := fieldID(t, f.cpFormat, template.AreaRow, "Method")
	var texts []string
	for _, r := range got.Rows {
		texts = append(texts, r.Content[product].(string))
		if r.ProcessID != tpl.process || r.TemplatePointID == "" || r.OperationNumber != "" {
			t.Errorf("baris CP: %+v", r)
		}
	}
	if !slices.Equal(texts, []string{"Part number", "MSL label", "Warpage max 0.5%"}) {
		t.Errorf("isi CP: %v", texts)
	}
	if got.Rows[0].Content[method] != "Visual" || got.Rows[2].TemplatePointID != id("Check warpage") {
		t.Errorf("baris CP pertama dan ketiga: %+v, %+v", got.Rows[0], got.Rows[2])
	}

	// PFMEA: failure mode masuk ke kolomnya, severity sebagai angka. Kunci
	// work_element hanya ada di form AIAG-VDA, jadi dilewati di form AIAG 4th.
	pfmea := f.get(t, set["pfmea"].ID)
	pfmeaFormat, err := template.LoadFormatVersion(ctx, f.q, pfmea.FormatVersionID)
	if err != nil {
		t.Fatal(err)
	}
	mode := fieldID(t, pfmeaFormat, template.AreaRow, "Potential Failure Mode")
	severity := fieldID(t, pfmeaFormat, template.AreaRow, "Severity")
	if len(pfmea.Rows) != 3 || pfmea.Rows[0].Content[mode] != "Wrong part received" || pfmea.Rows[0].Content[severity] != float64(7) ||
		len(pfmea.Rows[0].Content) != 3 || pfmea.Rows[2].Content[mode] != "Warped PCB" {
		t.Errorf("baris PFMEA: %+v", pfmea.Rows)
	}
	// PFD tidak dirakit dari template (D-214): diisi per dokumen.
	if pfd := f.get(t, set["pfd"].ID); len(pfd.Rows) != 0 {
		t.Errorf("baris PFD: %+v", pfd.Rows)
	}

	// Tampilan perakitan: dimasukkan, dikecualikan dengan alasan, belum diputuskan.
	view, err = f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	st = states(view)
	if st["Check MSL label"] != StateIncluded || st["Check HIC"] != StateExcluded || st["Check crimp height"] != StateUndecided || st["Warped PCB"] != StateIncluded {
		t.Errorf("keadaan sesudah rakit: %v", st)
	}
	for _, s := range view.Subprocesses {
		for _, p := range s.Points {
			if p.Text == "Check HIC" && p.Reason != "No HIC for this part" {
				t.Errorf("alasan: %q", p.Reason)
			}
		}
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "document.composed" || details != `{"added":"6","excluded":"1","process":"IQC","removed":"0","version":"1"}` {
		t.Errorf("audit: %s %s", action, details)
	}

	// Perubahan lokal bertahan saat dirakit ulang; poin yang dicoret
	// kehilangan barisnya, dan poin yang dicentang lagi mendapat baris baru.
	row := got.Rows[0]
	if got, err = f.s.UpdateRow(ctx, f.author, cp.ID, row.ID, RowUpdate{ProcessID: row.ProcessID, Content: map[string]any{product: "Part number and revision"}, Version: row.Version}); err != nil {
		t.Fatal(err)
	}
	got, err = f.s.Compose(ctx, f.author, cp.ID, Compose{
		VersionID: tpl.version.ID,
		Include:   []string{id("Check part number against PO"), id("Check HIC")},
		Exclude:   []Exclusion{{PointID: id("Check MSL label")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	texts = nil
	for _, r := range got.Rows {
		texts = append(texts, r.Content[product].(string))
	}
	if !slices.Equal(texts, []string{"Part number and revision", "Warpage max 0.5%", "Humidity indicator card"}) || got.Rows[1].Position != 2 {
		t.Errorf("isi CP sesudah rakit ulang: %v", texts)
	}

	// Baris tertaut yang dihapus dari tabel menjadi poin yang dikecualikan.
	if _, err := f.s.DeleteRow(ctx, f.author, cp.ID, got.Rows[1].ID); err != nil {
		t.Fatal(err)
	}
	view, _ = f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, "")
	if st = states(view); st["Check warpage (K-125)"] != StateExcluded || st["Check HIC"] != StateIncluded || st["Check MSL label"] != StateExcluded {
		t.Errorf("keadaan sesudah hapus baris: %v", st)
	}
}

// TestComposeRules: penolakan perakitan.
func TestComposeRules(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	// Dokumen CP tunggal: hanya poin CP yang ikut.
	cp, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k200}})
	if err != nil {
		t.Fatal(err)
	}
	view, err := f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	// Model K-200 tidak punya varian, jadi poin asalnya yang tampil.
	if st := states(view); len(st) != 5 || st["Check warpage"] != StateUndecided {
		t.Errorf("poin untuk dokumen CP tunggal: %v", st)
	}

	for name, c := range map[string]struct {
		in   Compose
		want string
	}{
		"poin jenis lain":  {Compose{VersionID: tpl.version.ID, Include: []string{tpl.points["Warped PCB"]}}, "include:invalid_format"},
		"poin tak dikenal": {Compose{VersionID: tpl.version.ID, Exclude: []Exclusion{{PointID: "bukan-id"}}}, "exclude:invalid_format"},
		"dua keadaan": {Compose{VersionID: tpl.version.ID, Include: []string{tpl.points["Check HIC"]},
			Exclude: []Exclusion{{PointID: tpl.points["Check HIC"]}}}, "exclude:not_allowed"},
		"alasan panjang": {Compose{VersionID: tpl.version.ID, Exclude: []Exclusion{{PointID: tpl.points["Check HIC"], Reason: string(make([]byte, 501))}}}, "exclude:too_long"},
		"tanpa versi":    {Compose{VersionID: "", Include: []string{tpl.points["Check HIC"]}}, "process_template_version_id:invalid_format"},
	} {
		_, err := f.s.Compose(ctx, f.author, cp.ID, c.in)
		if got := codes(t, err); !slices.Contains(got, c.want) {
			t.Errorf("%s: %v", name, got)
		}
	}

	// Versi draft tidak dipakai untuk merakit.
	v2, err := tpl.svc.NewProcessVersion(ctx, tpl.actor, tpl.version.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Compose(ctx, f.author, cp.ID, Compose{VersionID: v2.ID, Include: []string{tpl.points["Check HIC"]}})
	if got := codes(t, err); !slices.Contains(got, "process_template_version_id:not_allowed") {
		t.Errorf("versi draft: %v", got)
	}
	if _, err := f.s.Composition(ctx, f.author, cp.ID, "01900000-0000-7000-8000-000000000000", ""); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("versi tidak ada: %v", err)
	}
	if after := f.get(t, cp.ID); len(after.Rows) != 0 {
		t.Errorf("baris sesudah penolakan: %+v", after.Rows)
	}
}
