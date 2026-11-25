package process

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
)

// I2: PPAP Engineer mengimpor nama proses dan aliasnya dari berkas Excel.

// processBook membuat berkas Excel satu sheet; baris pertama judul kolom.
func processBook(t *testing.T, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetSheetName("Sheet1", SheetProcesses)
	for y, row := range rows {
		for x, text := range row {
			cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
			_ = f.SetCellStr(SheetProcesses, cell, text)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func importStatuses(r ImportReport) []string {
	var out []string
	for _, row := range r.Rows {
		out = append(out, row.Status)
	}
	return out
}

func TestImportProcesses(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	viewer := newActor(t, f.pool, "lihat.uji", "work")
	if _, err := f.s.Create(ctx, f.engineer, Input{Name: "IQC", Aliases: []string{"Incoming inspection"}}); err != nil {
		t.Fatal(err)
	}

	data := processBook(t, [][]string{
		{"process name", "Aliases"},                          // huruf kecil juga dikenali
		{"iqc", "Incoming inspection; Receiving inspection"}, // sudah ada; satu alias baru tidak ditambahkan
		{"SMT", "Surface mount;  SMT line "},                 // baru
		{"Reflow", ""},                                       // baru, tanpa alias
		{"", ""},                                             // kosong, diabaikan
		{"Incoming inspection", ""},                          // alias proses IQC
		{"AOI", "Surface mount"},                             // alias milik SMT dari baris sebelumnya
		{"Coating", "Spray; spray"},                          // alias ganda di satu baris
		{"", "Tanpa nama"},                                   // nama wajib
		{"smt", ""},                                          // proses dari baris sebelumnya: sudah ada
	})

	if _, err := f.s.Import(ctx, viewer, data, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-6: %v", err)
	}
	if _, err := f.s.Import(ctx, f.engineer, []byte("bukan excel"), false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("bukan Excel: %v", err)
	}
	if _, err := f.s.Import(ctx, f.engineer, processBook(t, [][]string{{"Nama"}}), false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("judul kolom salah: %v", err)
	}

	preview, err := f.s.Import(ctx, f.engineer, data, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{RowExisting, RowNew, RowNew, RowInvalid, RowInvalid, RowInvalid, RowInvalid, RowExisting}
	if got := importStatuses(preview); !slices.Equal(got, want) || preview.Committed {
		t.Fatalf("preview: %v", got)
	}
	if !slices.Equal(preview.Rows[0].NotAdded, []string{"Receiving inspection"}) {
		t.Errorf("alias yang tidak ditambahkan: %v", preview.Rows[0].NotAdded)
	}
	if !slices.Equal(preview.Rows[1].Aliases, []string{"Surface mount", "SMT line"}) {
		t.Errorf("alias dirapikan: %q", preview.Rows[1].Aliases)
	}
	if preview.Rows[3].Row != 6 || preview.Rows[3].Problems[0].Field != "process_name" || !strings.Contains(preview.Rows[3].Problems[0].Message, "IQC") {
		t.Errorf("nama yang sudah alias: %+v", preview.Rows[3])
	}
	if preview.Rows[4].Problems[0].Field != "aliases" || preview.Rows[5].Problems[0].Field != "aliases" || preview.Rows[6].Problems[0].Field != "process_name" {
		t.Errorf("penolakan: %+v / %+v / %+v", preview.Rows[4].Problems, preview.Rows[5].Problems, preview.Rows[6].Problems)
	}
	if list, _ := f.s.List(ctx, f.engineer); len(list) != 1 {
		t.Fatalf("Preview menyimpan proses: %d", len(list))
	}

	done, err := f.s.Import(ctx, f.engineer, data, true)
	if err != nil {
		t.Fatal(err)
	}
	if !done.Committed || done.New != 2 || done.Existing != 2 || done.Invalid != 4 {
		t.Fatalf("import: new %d, existing %d, invalid %d", done.New, done.Existing, done.Invalid)
	}
	list, _ := f.s.List(ctx, f.engineer)
	names := map[string][]string{}
	for _, p := range list {
		names[p.Name] = p.Aliases
	}
	if len(list) != 3 || !slices.Equal(names["SMT"], []string{"SMT line", "Surface mount"}) || len(names["Reflow"]) != 0 ||
		!slices.Equal(names["IQC"], []string{"Incoming inspection"}) {
		t.Errorf("proses sesudah import: %v", names)
	}

	// Berkas yang sama diunggah lagi: yang benar sudah ada semua.
	again, err := f.s.Import(ctx, f.engineer, data, true)
	if err != nil || again.New != 0 || again.Existing != 4 {
		t.Errorf("import ulang: new %d, existing %d, %v", again.New, again.Existing, err)
	}
}

func TestProcessImportTemplate(t *testing.T) {
	f := newFixture(t)
	data, err := f.s.ImportTemplate(t.Context(), f.engineer)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	rows, err := book.GetRows(SheetProcesses)
	if err != nil || len(rows) != 1 || !slices.Equal(rows[0], importHeaders) {
		t.Errorf("sheet %s: %v, %v", SheetProcesses, rows, err)
	}
}
