package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// C4 · PFMEA dan CP dirakit per langkah PFD (D-211, E-136, D-212).

// fillPFD mengisi PFD dengan langkah: nomor, proses, simbol, dan arah NG.
func (f fixture) fillPFD(t *testing.T, pfd Document, steps ...[4]string) {
	t.Helper()
	format, err := template.LoadFormatVersion(t.Context(), f.q, pfd.FormatVersionID)
	if err != nil {
		t.Fatal(err)
	}
	field := map[string]string{}
	for _, fd := range format.Fields {
		field[fd.Key] = fd.ID
	}
	var rows []NewRow
	for _, s := range steps {
		content := map[string]any{field["flow_symbol"]: s[2]}
		if s[3] != "" {
			content[field["ng_disposition"]] = s[3]
		}
		rows = append(rows, NewRow{OperationNumber: s[0], ProcessID: s[1], Content: content})
	}
	if _, err := f.s.AddRows(t.Context(), f.author, pfd.ID, rows); err != nil {
		t.Fatal(err)
	}
}

// stepState memberi keadaan setiap langkah di satu jenis dokumen.
func stepState(c PFDComposition, typeKey string) []string {
	var out []string
	for _, s := range c.Steps {
		for _, t := range s.Targets {
			if t.DocumentType == typeKey {
				out = append(out, s.Operation+":"+t.State)
			}
		}
	}
	return out
}

func TestComposeFromPFD(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set := f.newSet(t)
	cp, pfmea, pfd := set["cp"], set["pfmea"], set["pfd"]
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")

	// PFD kosong: tidak ada langkah, dan perakitan ditolak dengan pesan "isi PFD dulu".
	view, err := f.s.PFDComposition(ctx, f.author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.PFDID != pfd.ID || len(view.Steps) != 0 || len(view.Documents) != 2 {
		t.Fatalf("tampilan PFD kosong: %+v", view)
	}
	if _, err := f.s.ComposeFromPFD(ctx, f.author, cp.ID, []StepChoice{{Operation: "10", ProcessID: tpl.process}}); !hasProblem(err, "steps") {
		t.Errorf("rakit dari PFD kosong: %v", err)
	}

	// Langkah 10 IQC (ada template), 20 SMT Printing (tanpa template), 30 IQC lagi.
	f.fillPFD(t, pfd,
		[4]string{"10", tpl.process, "process", ""},
		[4]string{"20", f.process, "decision", "scrap"},
		[4]string{"30", tpl.process, "process", ""},
	)
	if _, err := f.s.PFDComposition(ctx, f.viewer, cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa hak melihat langkah PFD: %v", err)
	}
	if view, err = f.s.PFDComposition(ctx, f.author, cp.ID); err != nil {
		t.Fatal(err)
	}
	if got := stepState(view, "cp"); !slices.Equal(got, []string{"10:to_add", "20:no_template", "30:to_add"}) {
		t.Errorf("keadaan langkah CP: %v", got)
	}
	// D-213: IQC dipakai di langkah 10 dan 30, jadi diberi peringatan.
	if len(view.Repeated) != 1 || view.Repeated[0].ProcessName != "IQC" || !slices.Equal(view.Repeated[0].Operations, []string{"10", "30"}) {
		t.Errorf("proses berulang: %+v", view.Repeated)
	}
	if view.Steps[0].TemplateVersion != 1 || view.Steps[0].ProcessName != "IQC" || view.Steps[1].TemplateVersionID != "" {
		t.Errorf("template setiap langkah: %+v", view.Steps)
	}
	// IQC punya 5 poin CP untuk K-125 (General 1, MSD 2, PCB 1, Wireharness 1).
	if view.Steps[0].Targets[1].Points != 5 && view.Steps[0].Targets[0].Points != 5 {
		t.Errorf("jumlah poin: %+v", view.Steps[0].Targets)
	}

	// Baris yang diketik sendiri lebih dulu: langkah 99 tidak ada di PFD.
	if _, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{OperationNumber: "99", ProcessID: f.process, Content: map[string]any{product: "Own row"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.ComposeFromPFD(ctx, f.viewer, cp.ID, []StepChoice{{Operation: "10", ProcessID: tpl.process}}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer merakit dari PFD: %v", err)
	}
	if _, err := f.s.ComposeFromPFD(ctx, f.author, cp.ID, []StepChoice{{Operation: "40", ProcessID: tpl.process}}); !hasProblem(err, "steps") {
		t.Errorf("langkah yang tidak ada di PFD: %v", err)
	}
	if _, err := f.s.ComposeFromPFD(ctx, f.author, pfd.ID, []StepChoice{{Operation: "10", ProcessID: tpl.process}}); !hasProblem(err, "document") {
		t.Errorf("rakit PFD dari PFD: %v", err)
	}

	all := []StepChoice{{Operation: "10", ProcessID: tpl.process}, {Operation: "20", ProcessID: f.process}, {Operation: "30", ProcessID: tpl.process}}
	got, err := f.s.ComposeFromPFD(ctx, f.author, cp.ID, all)
	if err != nil {
		t.Fatal(err)
	}
	// CP: 5 baris IQC langkah 10, satu baris "--" langkah 20, 5 baris IQC
	// langkah 30, lalu baris sendiri langkah 99 yang tetap di akhir.
	var ops []string
	for _, r := range got.Rows {
		ops = append(ops, r.OperationNumber)
		if r.OperationNumber != "99" && !r.OperationManual {
			t.Errorf("nomor langkah dari PFD harus ditandai manual: %+v", r)
		}
	}
	want := []string{"10", "10", "10", "10", "10", "20", "30", "30", "30", "30", "30", "99"}
	if !slices.Equal(ops, want) {
		t.Fatalf("urutan CP: %v", ops)
	}
	if got.Rows[0].Content[product] != "Part number" || got.Rows[0].TemplatePointID == "" || got.Rows[0].ProcessID != tpl.process {
		t.Errorf("baris IQC pertama: %+v", got.Rows[0])
	}
	if got.Rows[5].Content[product] != noControl || got.Rows[5].TemplatePointID != "" || got.Rows[5].ProcessID != f.process {
		t.Errorf("baris tanpa template: %+v", got.Rows[5])
	}
	// IQC di langkah 30: isi sama, tetapi tidak tertaut, karena poinnya
	// sudah tertaut di langkah 10 (ASUMSI E-138).
	if got.Rows[6].Content[product] != "Part number" || got.Rows[6].TemplatePointID != "" {
		t.Errorf("IQC kedua: %+v", got.Rows[6])
	}

	// PFMEA di set yang sama ikut dirakit: IQC punya 3 poin PFMEA.
	pm := f.get(t, pfmea.ID)
	if len(pm.Rows) != 3+1+3 {
		t.Errorf("baris PFMEA: %d", len(pm.Rows))
	}

	// Dirakit lagi: tidak ada yang berubah, karena semua langkah sudah ada.
	again, err := f.s.ComposeFromPFD(ctx, f.author, pfmea.ID, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Rows) != len(pm.Rows) || len(f.get(t, cp.ID).Rows) != len(got.Rows) {
		t.Errorf("rakit ulang menambah baris")
	}

	// Pemeriksaan: semua langkah sudah ada, dan langkah 99 CP tidak ada di PFD.
	if view, err = f.s.PFDComposition(ctx, f.author, cp.ID); err != nil {
		t.Fatal(err)
	}
	if got := stepState(view, "cp"); !slices.Equal(got, []string{"10:present", "20:present", "30:present"}) {
		t.Errorf("keadaan sesudah dirakit: %v", got)
	}
	if len(view.Extra) != 1 || view.Extra[0].Operation != "99" || view.Extra[0].DocumentType != "cp" {
		t.Errorf("langkah di luar PFD: %+v", view.Extra)
	}

	// D-212: CP membawa langkah PFD di snapshot-nya; PFMEA tidak.
	snap, _, err := CurrentSnapshot(f.get(t, cp.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Flow) != 3 || snap.Flow[1].Symbol != "decision" || snap.Flow[1].NG != "scrap" || snap.Flow[1].ProcessName != "SMT Printing" {
		t.Errorf("langkah PFD di snapshot CP: %+v", snap.Flow)
	}
	if snap, _, _ := CurrentSnapshot(pm); len(snap.Flow) != 0 {
		t.Errorf("PFMEA tidak membawa langkah PFD: %+v", snap.Flow)
	}
}

// TestStepsOfPFD: baris berurutan dengan nomor dan proses yang sama adalah
// satu langkah; baris tanpa nomor dilewati (ASUMSI E-138).
func TestStepsOfPFD(t *testing.T) {
	format := template.FormatVersion{Fields: []template.Field{{ID: "s", Key: "flow_symbol"}, {ID: "n", Key: "ng_disposition"}}}
	pfd := Document{Rows: []Row{
		{OperationNumber: "1", ProcessID: "a", ProcessName: "Part Receiving", Content: map[string]any{"s": "process"}},
		{OperationNumber: "2", ProcessID: "b", ProcessName: "IQC", Content: map[string]any{"s": "decision", "n": "ncp"}},
		{OperationNumber: "2", ProcessID: "b", Content: map[string]any{}},
		{OperationNumber: "2", ProcessID: "c", ProcessName: "IQC PCB", Content: map[string]any{"s": "decision"}},
		{OperationNumber: "", ProcessID: "d", Content: map[string]any{}},
	}}
	got := StepsOfPFD(pfd, format)
	if len(got) != 3 || got[1].NG != "ncp" || got[2].ProcessName != "IQC PCB" || got[0].Symbol != "process" {
		t.Errorf("langkah: %+v", got)
	}
}

// hasProblem melaporkan apakah err penolakan input untuk field itu.
func hasProblem(err error, field string) bool {
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		return false
	}
	return slices.ContainsFunc(invalid.Problems, func(p problem.Problem) bool { return p.Field == field })
}
