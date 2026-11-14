package document

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/template"
)

// M7.3 · Pembanding Excel (R-MIG-04): Excel lama hanya dibandingkan, dan
// selisih dua arah tampil.
func TestCompareExcel(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	cp := f.newSet(t)["cp"]
	format, err := template.LoadFormatVersion(ctx, f.q, cp.FormatVersionID)
	if err != nil {
		t.Fatal(err)
	}
	var characteristic string
	for _, fld := range format.Fields {
		if fld.Area == template.AreaRow && fld.Key == "product_characteristic" {
			characteristic = fld.ID
		}
	}
	for _, text := range []string{"Part number", "MSL label", "Only in the system"} {
		if _, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{ProcessID: f.process, Content: map[string]any{characteristic: text}}); err != nil {
			t.Fatal(err)
		}
	}

	book := excelize.NewFile()
	defer book.Close()
	if _, err := book.NewSheet("CP"); err != nil {
		t.Fatal(err)
	}
	for cell, value := range map[string]any{
		"A1": "CONTROL PLAN lama", "B3": "part  NUMBER", "B4": "MSL label", "B5": "Only in Excel", "C4": 7,
	} {
		if err := book.SetCellValue("CP", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	var file bytes.Buffer
	if err := book.Write(&file); err != nil {
		t.Fatal(err)
	}

	if _, err := f.s.CompareExcel(ctx, f.viewer, cp.ID, "", bytes.NewReader(file.Bytes())); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa hak: %v", err)
	}
	if _, err := f.s.CompareExcel(ctx, f.author, cp.ID, "", strings.NewReader("bukan excel")); code(t, err) != "file:invalid_format" {
		t.Errorf("bukan Excel: %v", err)
	}
	if _, err := f.s.CompareExcel(ctx, f.author, cp.ID, "Tidak ada", bytes.NewReader(file.Bytes())); code(t, err) != "sheet:invalid_format" {
		t.Errorf("sheet tidak ada: %v", err)
	}
	got, err := f.s.CompareExcel(ctx, f.author, cp.ID, "", bytes.NewReader(file.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	// Sheet CP dipilih sendiri karena paling banyak cocok.
	if got.Sheet != "CP" || len(got.Sheets) != 2 || got.Matched != 2 {
		t.Fatalf("sheet %q, %d sheet, %d cocok", got.Sheet, len(got.Sheets), got.Matched)
	}
	missing := map[string]bool{}
	for _, c := range got.MissingInExcel {
		missing[c.Text] = true
	}
	// Nama proses baris juga isi dokumen, dan Excel ini tidak memuatnya.
	if !missing["Only in the system"] || missing["Part number"] {
		t.Errorf("tidak ada di Excel: %+v", got.MissingInExcel)
	}
	extra := map[string]string{}
	for _, c := range got.MissingInDocument {
		extra[c.Text] = c.Cell
	}
	if extra["Only in Excel"] != "B5" || extra["CONTROL PLAN lama"] != "A1" || len(extra) != 2 {
		t.Errorf("tidak ada di dokumen: %+v", got.MissingInDocument)
	}
	// Dokumen tidak berubah: pembanding tidak mengimpor apa pun.
	if d := f.get(t, cp.ID); len(d.Rows) != 3 {
		t.Errorf("baris dokumen: %d", len(d.Rows))
	}
}
