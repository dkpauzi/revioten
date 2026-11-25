package template

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// I3: PPAP Engineer mengimpor isi process template dari berkas Excel.

// contentBook membuat berkas Excel; sheets memetakan nama sheet ke barisnya,
// baris pertama judul kolom.
func contentBook(t *testing.T, sheets map[string][][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	first := true
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for k := range sheets {
			if !yield(k) {
				return
			}
		}
	}) {
		if first {
			_ = f.SetSheetName("Sheet1", name)
			first = false
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		for y, row := range sheets[name] {
			for x, text := range row {
				cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
				_ = f.SetCellStr(name, cell, text)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func contentStatuses(r ContentImportReport) []string {
	var out []string
	for _, row := range r.Rows {
		out = append(out, row.Sheet+":"+row.Status)
	}
	return out
}

func TestImportContent(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()

	// Laser Marking: versi 1 terbit dengan satu poin PFMEA umum.
	laser := f.newTemplate(t, "Laser Marking", "Laser")
	laser = f.addPoint(t, laser, "Laser", "pfmea", "Laser type")
	if _, err := f.s.PublishProcessVersion(ctx, f.maintainer, laser.ID); err != nil {
		t.Fatal(err)
	}
	// SMT: proses tanpa template, dengan alias.
	smt := f.process(t, "SMT")
	if err := f.q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: smt, Kind: "alias", Name: "Surface mount"}); err != nil {
		t.Fatal(err)
	}

	data := contentBook(t, map[string][][]string{
		SheetContentPFMEA: {
			{"process", "Sub-process", "Point text", "Use point for", "Customer code", "Model group", "Model", "failure_mode", "severity"},
			{"laser marking", "laser", "Laser type", "", "", "", "", "", ""},                             // sudah ada di versi terbit
			{"Laser Marking", "Laser", "Laser power", "Model", "ab", "", "K-125", "Mark too light", "7"}, // baru, khusus model
			{"Surface mount", "Printing", "Paste volume", "Customer", "CD", "", "", "Too little paste", "8"},
			{"SMT", "Printing", "Bad rating", "", "", "", "", "", "11"}, // rating di luar 1-10
			{"Unknown", "A", "B"}, // proses tidak terdaftar
			{"SMT", "Printing", "paste volume", "customer", "CD"}, // sama dengan baris sebelumnya
			{"SMT", "Printing", "Z", "Model group", "AB", "Nope"}, // group model tidak ada
		},
		SheetContentCP: {
			{"Process", "Sub-process", "Point text", "reaction_plan"},
			{"SMT", "Printing", "Paste height", "Stop the line"},
		},
	})

	if _, err := f.s.ImportContent(ctx, f.author, data, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-6: %v", err)
	}
	bad := contentBook(t, map[string][][]string{SheetContentPFMEA: {{"Process", "Sub-process", "Point text", "Unknown column"}, {"SMT", "A", "B", "x"}}})
	if _, err := f.s.ImportContent(ctx, f.maintainer, bad, false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("kolom tidak dikenal: %v", err)
	}
	other := contentBook(t, map[string][][]string{"Sheet": {{"Process"}}})
	if _, err := f.s.ImportContent(ctx, f.maintainer, other, false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("tanpa sheet PFMEA atau Control Plan: %v", err)
	}

	want := []string{"PFMEA:existing", "PFMEA:new", "PFMEA:new", "PFMEA:invalid", "PFMEA:invalid", "PFMEA:existing", "PFMEA:invalid", "Control Plan:new"}
	preview, err := f.s.ImportContent(ctx, f.maintainer, data, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := contentStatuses(preview); !slices.Equal(got, want) || preview.Committed {
		t.Fatalf("preview: %v", got)
	}
	if p := preview.Rows[3].Problems; len(p) != 1 || p[0].Field != "severity" {
		t.Errorf("rating salah: %+v", p)
	}
	if p := preview.Rows[4].Problems; len(p) != 1 || p[0].Field != "process" {
		t.Errorf("proses tidak terdaftar: %+v", p)
	}
	if p := preview.Rows[6].Problems; len(p) != 1 || p[0].Field != "model_group" {
		t.Errorf("group model tidak ada: %+v", p)
	}
	if r := preview.Rows[1]; r.Process != "Laser Marking" || r.ScopeLevel != ScopeModel || r.ScopeTarget != "K-125 (AB)" {
		t.Errorf("baris model: %+v", r)
	}
	wantVersions := []ContentImportVersion{
		{Process: "Laser Marking", Version: 2, Change: VersionNewVersion, Points: 1},
		{Process: "SMT", Version: 1, Change: VersionNewTemplate, Points: 2},
	}
	if !slices.Equal(preview.Versions, wantVersions) {
		t.Errorf("versi preview: %+v", preview.Versions)
	}
	templates, _ := f.s.ListProcessTemplates(ctx, f.maintainer)
	if len(templates) != 1 || len(templates[0].Versions) != 1 {
		t.Fatalf("Preview menyimpan sesuatu: %+v", templates)
	}

	done, err := f.s.ImportContent(ctx, f.maintainer, data, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := contentStatuses(done); !slices.Equal(got, want) || !done.Committed || !slices.Equal(done.Versions, wantVersions) {
		t.Fatalf("import: %v %+v", got, done.Versions)
	}
	templates, _ = f.s.ListProcessTemplates(ctx, f.maintainer)
	versionOf := map[string]ProcessVersionSummary{}
	for _, pt := range templates {
		versionOf[pt.Process.Name] = pt.Versions[len(pt.Versions)-1]
	}
	laser2, err := f.s.GetProcessVersion(ctx, f.maintainer, versionOf["Laser Marking"].ID)
	if err != nil || laser2.Version != 2 || laser2.Status != StatusDraft {
		t.Fatalf("Laser Marking versi 2: %+v %v", laser2, err)
	}
	sub := subprocessNamed(t, laser2, "Laser")
	if got := pointTexts(sub); !slices.Equal(got, []string{"pfmea:Laser type", "pfmea:Laser power"}) {
		t.Errorf("poin Laser: %v", got)
	}
	power := sub.Points[1]
	if power.Scope.Level != ScopeModel || power.Scope.TargetID != f.modelX || power.Content["failure_mode"] != "Mark too light" || power.Content["severity"] != float64(7) {
		t.Errorf("poin Laser power: %+v", power)
	}
	smtV, err := f.s.GetProcessVersion(ctx, f.maintainer, versionOf["SMT"].ID)
	if err != nil || smtV.Version != 1 {
		t.Fatalf("SMT versi 1: %v", err)
	}
	printing := subprocessNamed(t, smtV, "Printing")
	if len(printing.Points) != 2 || printing.Points[0].Scope.TargetID != f.custB ||
		printing.Points[1].DocumentType.Key != "cp" || printing.Points[1].Content["reaction_plan"] != "Stop the line" {
		t.Errorf("poin Printing: %+v", printing.Points)
	}

	// Berkas yang sama diunggah lagi: semua poin yang benar sudah ada.
	again, err := f.s.ImportContent(ctx, f.maintainer, data, true)
	if err != nil || again.New != 0 || again.Existing != 5 || again.Invalid != 3 || len(again.Versions) != 0 {
		t.Errorf("import ulang: %+v, %v", again, err)
	}
}

func TestContentImportTemplate(t *testing.T) {
	f := newProcessFixture(t)
	if _, err := f.s.ContentImportTemplate(t.Context(), f.author); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-6: %v", err)
	}
	data, err := f.s.ContentImportTemplate(t.Context(), f.maintainer)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	for _, sheet := range []string{SheetContentPFMEA, SheetContentCP} {
		rows, err := book.GetRows(sheet)
		if err != nil || len(rows) != 1 || !slices.Equal(rows[0][:7], contentFixedHeaders) || len(rows[0]) <= 7 {
			t.Errorf("sheet %s: %v, %v", sheet, rows, err)
			continue
		}
		// Kolom yang terisi sendiri dan Relevant Document tidak ditawarkan.
		for _, h := range rows[0] {
			if h == "Relevant Document" || h == "RPN" || h == "Process Flow" {
				t.Errorf("sheet %s memuat kolom %s", sheet, h)
			}
		}
	}
	// Template yang diunduh bisa langsung diunggah lagi (tanpa baris: ditolak dengan alasan jelas).
	if _, err := f.s.ImportContent(t.Context(), f.maintainer, data, false); !slices.Equal(problemCodes(t, err), []string{"file:required"}) {
		t.Errorf("template kosong: %v", err)
	}
}
