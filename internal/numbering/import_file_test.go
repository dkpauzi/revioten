package numbering

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
)

// I4: nomor lama dengan revisi (opsi b) dan impor dari berkas Excel.

// numberBook membuat berkas Excel satu sheet; baris pertama judul kolom.
func numberBook(t *testing.T, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetSheetName("Sheet1", SheetNumbers)
	for y, row := range rows {
		for x, text := range row {
			cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
			_ = f.SetCellStr(SheetNumbers, cell, text)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// revisionOf membaca revisi awal dan status satu nomor di register.
func (f numberFixture) revisionOf(t *testing.T, number string) (int, string) {
	t.Helper()
	var revision int
	var status string
	err := f.pool.QueryRow(t.Context(), "SELECT initial_revision, status FROM document_numbers WHERE number = $1", number).Scan(&revision, &status)
	if err != nil {
		t.Fatalf("%s: %v", number, err)
	}
	return revision, status
}

// lineRevisions menulis revisi setiap baris laporan; "-" bila tanpa revisi.
func lineRevisions(r ImportReport) []string {
	var out []string
	for _, l := range r.Lines {
		if l.Revision == nil {
			out = append(out, l.Text+":-")
			continue
		}
		out = append(out, fmt.Sprintf("%s:%02d", l.Text, *l.Revision))
	}
	return out
}

// TestImportRevision: nomor yang ditempel boleh memuat revisi, misalnya
// SC-KT-001-05. Nomor tersimpan tanpa revisi, dan revisinya menjadi revisi
// awal nomor itu. Nomor yang sudah ada tidak diubah.
func TestImportRevision(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	report, err := f.s.Import(ctx, f.master, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased,
		Text: "SH-KT-001-02\tSF-KT-001-3\tSC-KT-001\nSH-KT-002-04"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1:SH-KT-001:imported", "1:SF-KT-001:imported", "1:SC-KT-001:imported", "2:SH-KT-002:imported"}
	if got := lineResults(report); !slices.Equal(got, want) {
		t.Errorf("hasil: %v", got)
	}
	if got := lineRevisions(report); !slices.Equal(got, []string{"SH-KT-001:02", "SF-KT-001:03", "SC-KT-001:-", "SH-KT-002:04"}) {
		t.Errorf("revisi di laporan: %v", got)
	}
	for number, want := range map[string]int{"SH-KT-001": 2, "SF-KT-001": 3, "SC-KT-001": 0, "SH-KT-002": 4} {
		if rev, _ := f.revisionOf(t, number); rev != want {
			t.Errorf("%s: revisi %d, ingin %d", number, rev, want)
		}
	}

	// Satu set: nomor yang mengikuti tidak punya revisi.
	report, err = f.s.Import(ctx, f.master, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, OneSet: true, Text: "SH-KT-003-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got := lineRevisions(report); !slices.Equal(got, []string{"SH-KT-003:01", "SF-KT-003:-", "SC-KT-003:-"}) {
		t.Errorf("satu set: %v", got)
	}

	// Nomor yang sudah ada: revisinya tidak diubah.
	report, err = f.s.Import(ctx, f.master, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: "SH-KT-001-09"})
	if err != nil {
		t.Fatal(err)
	}
	if got := lineResults(report); !slices.Equal(got, []string{"1:SH-KT-001:existing"}) {
		t.Errorf("impor ulang: %v", got)
	}
	if rev, _ := f.revisionOf(t, "SH-KT-001"); rev != 2 {
		t.Errorf("revisi nomor yang sudah ada berubah menjadi %d", rev)
	}

	// Revisi lebih panjang dari digit revisi: bukan nomor.
	report, err = f.s.Import(ctx, f.master, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: "SH-KT-004-123"})
	if err != nil {
		t.Fatal(err)
	}
	if got := lineResults(report); !slices.Equal(got, []string{"1:SH-KT-004-123:invalid"}) {
		t.Errorf("revisi tiga digit: %v", got)
	}
}

// TestImportFile: berkas Excel berisi nomor lama beberapa customer, satu set
// per baris, dengan status Released (bawaan) atau Reserved. Preview tidak
// menyimpan, dan berkas yang sama bisa diunggah lagi.
func TestImportFile(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	tr := f.newCustomer(t, "TR")
	viewer := newActor(t, f.pool, "lihat.uji", access.AccountWork)

	data := numberBook(t, [][]string{
		{"customer code", "PFD", "PFMEA", "CP", "Status"}, // huruf kecil juga dikenali
		{"KT", "SH-KT-010-02", "SF-KT-010-02", "SC-KT-010-03", ""},
		{"tr", "SH-TR-005", "", "", "Reserved"},
		{"ZZ", "SH-ZZ-001", "", "", ""},
		{"KT", "SH-KT-011", "", "", "Obsolete"},
		{"", "", "", "", ""},
		{"KT", "SH-TR-006", "", "", ""},
		{"KT", "", "", "", ""},
	})

	if _, err := f.s.ImportFile(ctx, viewer, FileImportInput{Data: data, DryRun: true}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-8: %v", err)
	}
	if _, err := f.s.ImportFile(ctx, f.master, FileImportInput{Data: []byte("bukan excel"), DryRun: true}); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("bukan Excel: %v", err)
	}
	wrong := numberBook(t, [][]string{{"Customer", "PFD"}})
	if _, err := f.s.ImportFile(ctx, f.master, FileImportInput{Data: wrong, DryRun: true}); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("judul kolom salah: %v", err)
	}

	preview, err := f.s.ImportFile(ctx, f.master, FileImportInput{Data: data, OneSet: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"2:SH-KT-010:imported", "2:SF-KT-010:imported", "2:SC-KT-010:imported",
		"3:SH-TR-005:imported", "3:SF-TR-005:imported", "3:SC-TR-005:imported",
		"4:SH-ZZ-001:unknown_customer",
		"5:SH-KT-011:invalid_status",
		"7:SH-TR-006:customer_mismatch", "7:SF-TR-006:customer_mismatch", "7:SC-TR-006:customer_mismatch",
		"8::invalid",
	}
	if got := lineResults(preview); !slices.Equal(got, want) {
		t.Fatalf("preview:\n%v\ningin:\n%v", got, want)
	}
	if !preview.DryRun || preview.Imported != 6 || preview.Lines[3].CustomerCode != "TR" || preview.Lines[4].Status != StatusReserved || preview.Lines[0].Status != StatusReleased {
		t.Errorf("preview: %+v", preview)
	}
	if list, _, _ := f.s.ListNumbers(ctx, f.master, ListFilter{}); len(list) != 0 {
		t.Fatalf("Preview menyimpan %d nomor", len(list))
	}

	done, err := f.s.ImportFile(ctx, f.master, FileImportInput{Data: data, OneSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if done.DryRun || done.Imported != 6 {
		t.Fatalf("import: %+v", done)
	}
	for number, want := range map[string]struct {
		rev    int
		status string
	}{
		"SH-KT-010": {2, StatusReleased}, "SC-KT-010": {3, StatusReleased},
		"SH-TR-005": {0, StatusReserved}, "SC-TR-005": {0, StatusReserved},
	} {
		if rev, status := f.revisionOf(t, number); rev != want.rev || status != want.status {
			t.Errorf("%s: revisi %d, status %s", number, rev, status)
		}
	}
	var groupCodes []string
	for _, g := range done.Groups {
		if !slices.Contains(groupCodes, g.CustomerCode) {
			groupCodes = append(groupCodes, g.CustomerCode)
		}
	}
	if !slices.Equal(groupCodes, []string{"KT", "TR"}) {
		t.Errorf("laporan register per customer: %v", groupCodes)
	}
	_ = tr

	// Berkas yang sama diunggah lagi: semua sudah ada.
	again, err := f.s.ImportFile(ctx, f.master, FileImportInput{Data: data, OneSet: true})
	if err != nil || again.Imported != 0 || again.Lines[0].Result != LineExisting || again.Lines[5].Result != LineExisting {
		t.Errorf("impor ulang: %+v, %v", again, err)
	}
}

func TestNumberImportTemplate(t *testing.T) {
	f := newNumberFixture(t)
	data, err := f.s.ImportTemplate(t.Context(), f.master)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	rows, err := book.GetRows(SheetNumbers)
	if err != nil || len(rows) != 1 || !slices.Equal(rows[0], numberImportHeaders) {
		t.Errorf("sheet %s: %v, %v", SheetNumbers, rows, err)
	}
}
