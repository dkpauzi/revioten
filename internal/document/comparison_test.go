package document

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/template"
)

// M4.9 · PFMEA dan CP berdampingan (R-DOC-06, D-65): tampilan per proses dan
// tanda "dicek oleh, tanggal" per proses per revisi.

// groupNames memberi nama proses setiap kelompok, menurut urutan.
func groupNames(c Comparison) []string {
	var out []string
	for _, g := range c.Processes {
		out = append(out, g.ProcessName)
	}
	return out
}

func TestComparison(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	set := f.newSet(t)
	id := func(text string) string { return tpl.points[text] }
	checker := newActor(t, f.pool, "ppap.engineer.cek", access.AccountWork, all(access.RolePFMEACPChecker))

	// IQC dirakit ke PFMEA dan CP; satu baris CP SMT Printing diketik sendiri.
	cp, err := f.s.Compose(ctx, f.author, set["cp"].ID, Compose{
		VersionID: tpl.version.ID,
		Include:   []string{id("Wrong part received"), id("Warped PCB"), id("Check part number against PO"), id("Check warpage")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cp, err = f.s.AddRow(ctx, f.author, cp.ID, NewRow{ProcessID: f.process}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.AddRow(ctx, f.author, cp.ID, NewRow{}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.s.Comparison(ctx, f.viewer, cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa hak melihat: %v", err)
	}
	// Dibuka dari dokumen mana pun di set; hasilnya sama.
	c, err := f.s.Comparison(ctx, f.author, set["pfd"].ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.PFMEA.ID != set["pfmea"].ID || c.CP.ID != set["cp"].ID {
		t.Fatalf("pasangan dokumen: %+v %+v", c.PFMEA.ID, c.CP.ID)
	}
	// Urutan proses mengikuti kemunculan pertama: PFMEA dulu, lalu CP.
	// Baris tanpa proses dikumpulkan di akhir.
	if !slices.Equal(groupNames(c), []string{"IQC", "SMT Printing", ""}) {
		t.Fatalf("kelompok proses: %v", groupNames(c))
	}
	iqcGroup := c.Processes[0]
	if len(iqcGroup.PFMEARows) != 2 || len(iqcGroup.CPRows) != 2 || iqcGroup.Check != nil {
		t.Errorf("kelompok IQC: %+v", iqcGroup)
	}
	if smt := c.Processes[1]; len(smt.PFMEARows) != 0 || len(smt.CPRows) != 1 {
		t.Errorf("kelompok SMT: %+v", smt)
	}

	// Tanda dicek hanya oleh pemegang V-2 (D-65).
	if _, err := f.s.MarkChecked(ctx, f.author, cp.ID, "", tpl.process, true); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun memberi tanda: %v", err)
	}
	before := time.Now().Add(-time.Minute)
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, true); err != nil {
		t.Fatal(err)
	}
	mark := c.Processes[0].Check
	if mark == nil || mark.By != "Orang ppap.engineer.cek" || mark.At.Before(before) || mark.Stale || mark.PFMEARevision != 0 || mark.CPRevision != 0 {
		t.Fatalf("tanda: %+v", mark)
	}
	if c.Processes[1].Check != nil {
		t.Errorf("proses lain ikut bertanda: %+v", c.Processes[1].Check)
	}
	// Memberi tanda lagi tidak menggandakan; tanda tetap milik pemberi pertama.
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, true); err != nil || c.Processes[0].Check == nil {
		t.Errorf("tanda kedua: %+v, %v", c.Processes[0].Check, err)
	}

	// Baris proses itu berubah sesudah dicek: tandanya perlu dicek ulang.
	row := cp.Rows[0]
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")
	if _, err := f.s.UpdateRow(ctx, f.author, cp.ID, row.ID, RowUpdate{ProcessID: row.ProcessID, Content: map[string]any{product: "Part number and revision"}, Version: row.Version}); err != nil {
		t.Fatal(err)
	}
	c, _ = f.s.Comparison(ctx, f.author, cp.ID)
	if mark := c.Processes[0].Check; mark == nil || !mark.Stale {
		t.Errorf("tanda sesudah baris berubah: %+v", mark)
	}
	// Dicek ulang: tanda diperbarui dan tidak lagi perlu dicek ulang.
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, true); err != nil || c.Processes[0].Check == nil || c.Processes[0].Check.Stale {
		t.Errorf("cek ulang: %+v, %v", c.Processes[0].Check, err)
	}

	// Baris yang dihapus juga perubahan: tanda baru, lalu satu baris PFMEA IQC dihapus.
	if _, err := f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, false); err != nil {
		t.Fatal(err)
	}
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, true); err != nil || c.Processes[0].Check == nil || c.Processes[0].Check.Stale {
		t.Fatalf("tanda baru: %+v, %v", c.Processes[0].Check, err)
	}
	if _, err := f.s.DeleteRow(ctx, f.author, c.PFMEA.ID, c.PFMEA.Rows[1].ID); err != nil {
		t.Fatal(err)
	}
	c, _ = f.s.Comparison(ctx, f.author, cp.ID)
	if mark := c.Processes[0].Check; mark == nil || !mark.Stale {
		t.Errorf("tanda sesudah baris dihapus: %+v", mark)
	}
	// Proses lain tidak ikut berubah: SMT Printing diberi tanda dan tetap segar.
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", f.process, true); err != nil || c.Processes[1].Check == nil || c.Processes[1].Check.Stale {
		t.Errorf("tanda SMT: %+v, %v", c.Processes[1].Check, err)
	}

	// Tanda dicabut.
	if c, err = f.s.MarkChecked(ctx, checker, cp.ID, "", tpl.process, false); err != nil || c.Processes[0].Check != nil {
		t.Errorf("cabut tanda: %+v, %v", c.Processes[0].Check, err)
	}
	// Proses yang tidak ada di kedua dokumen ditolak.
	if _, err := f.s.MarkChecked(ctx, checker, cp.ID, "", "01900000-0000-7000-8000-000000000000", true); !slices.Contains(codes(t, err), "process_id:invalid_format") {
		t.Errorf("proses asing: %v", err)
	}
	var actions []string
	rows, err := f.pool.Query(ctx, "SELECT action FROM audit_log WHERE action LIKE 'document.pfmea_cp_%' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	rows.Close()
	want := []string{"document.pfmea_cp_checked", "document.pfmea_cp_checked", "document.pfmea_cp_unchecked", "document.pfmea_cp_checked", "document.pfmea_cp_checked", "document.pfmea_cp_unchecked"}
	if !slices.Equal(actions, want) {
		t.Errorf("audit: %v", actions)
	}

	// Dokumen tanpa pasangan PFMEA dan CP di set-nya tidak punya perbandingan.
	single, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Comparison(ctx, f.author, single.ID); !slices.Contains(codes(t, err), "set:not_allowed") {
		t.Errorf("dokumen tunggal: %v", err)
	}
}
