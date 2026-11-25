package process

// Import nama proses dari berkas Excel (I2). PPAP Engineer (E-6, D-64)
// mengunduh template, mengisi satu proses per baris (nama dan alias dipisah
// titik koma), lalu mengunggahnya. Preview melaporkan setiap baris tanpa
// menyimpan; Import menyimpan baris yang benar lewat Create, jadi aturan, hak,
// dan audit log sama dengan layar Processes. Proses yang sudah ada dilewati
// dan tidak diubah, sehingga berkas yang sama aman diunggah lagi sesudah
// database direset. Alias baru untuk proses yang sudah ada tidak ditambahkan
// diam-diam: barisnya menyebut alias itu (rencana I2).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// SheetProcesses adalah nama sheet di template.
const SheetProcesses = "Processes"

// Batas berkas import.
const (
	MaxImportBytes = 5 << 20 // 5 MB
	MaxImportRows  = 2000
)

// Hasil satu baris.
const (
	RowNew      = "new"      // proses baru (dibuat bila Import)
	RowExisting = "existing" // nama tampilan sudah ada, tidak diubah
	RowInvalid  = "invalid"  // ada isian yang salah; lihat Problems
)

// importHeaders adalah judul kolom template. Urutan kolom di berkas bebas.
var importHeaders = []string{"Process name", "Aliases"}

// ImportRow adalah satu baris berkas beserta hasilnya.
type ImportRow struct {
	Row         int    // nomor baris di Excel, mulai dari 2
	Name        string // isi kolom Process name, sudah dirapikan bila sah
	AliasesText string // isi kolom Aliases apa adanya
	Aliases     []string
	Status      string
	// NotAdded adalah alias baris "sudah ada" yang belum dimiliki proses itu.
	// Alias ini tidak ditambahkan; tambahkan lewat layar Processes.
	NotAdded []string
	Problems []problem.Problem
}

// ImportReport adalah laporan satu Preview atau Import.
type ImportReport struct {
	Rows      []ImportRow
	New       int
	Existing  int
	Invalid   int
	Committed bool
}

// nameOwner adalah pemilik satu nama: nama tampilan prosesnya, dan apakah
// nama itu nama tampilan atau alias.
type nameOwner struct {
	process string
	display bool
}

// ImportTemplate membuat berkas Excel template (E-6): sheet Processes dengan
// judul kolom dan sheet Help, tanpa baris contoh.
func (s *Service) ImportTemplate(ctx context.Context, actor access.Actor) ([]byte, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", SheetProcesses); err != nil {
		return nil, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	_ = f.SetSheetRow(SheetProcesses, "A1", &[]any{importHeaders[0], importHeaders[1]})
	_ = f.SetCellStyle(SheetProcesses, "A1", "B1", bold)
	_ = f.SetColWidth(SheetProcesses, "A", "A", 32)
	_ = f.SetColWidth(SheetProcesses, "B", "B", 60)

	if _, err := f.NewSheet("Help"); err != nil {
		return nil, err
	}
	help := [][]any{
		{"Column", "What to fill in"},
		{"Process name", "Required, for example IQC. At most 200 characters. A process that already exists is skipped."},
		{"Aliases", fmt.Sprintf("Optional. Other names of the same process, separated by semicolons, at most %d. A name can belong to one process only, as its name or as an alias.", maxAliases)},
	}
	for i, line := range help {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Help", cell, &line)
	}
	_ = f.SetCellStyle("Help", "A1", "B1", bold)
	_ = f.SetColWidth("Help", "A", "A", 18)
	_ = f.SetColWidth("Help", "B", "B", 100)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("menulis template proses: %w", err)
	}
	return buf.Bytes(), nil
}

// Import membaca berkas (E-6). commit false memberi laporan saja (Preview);
// commit true juga menyimpan proses baru, satu baris satu transaksi.
func (s *Service) Import(ctx context.Context, actor access.Actor, data []byte, commit bool) (ImportReport, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ImportReport{}, err
	}
	rows, err := readProcessBook(data)
	if err != nil {
		return ImportReport{}, err
	}
	existing, err := s.List(ctx, actor)
	if err != nil {
		return ImportReport{}, err
	}
	// names memetakan setiap nama (huruf kecil, seperti name_key di database)
	// ke pemiliknya: proses yang sudah ada, lalu yang dibuat baris sebelumnya.
	names := map[string]nameOwner{}
	for _, p := range existing {
		names[strings.ToLower(p.Name)] = nameOwner{process: p.Name, display: true}
		for _, a := range p.Aliases {
			names[strings.ToLower(a)] = nameOwner{process: p.Name}
		}
	}

	report := ImportReport{Rows: rows, Committed: commit}
	for i := range report.Rows {
		if err := s.importRow(ctx, actor, names, &report.Rows[i], commit); err != nil {
			return ImportReport{}, err
		}
		switch report.Rows[i].Status {
		case RowNew:
			report.New++
		case RowExisting:
			report.Existing++
		default:
			report.Invalid++
		}
	}
	return report, nil
}

// importRow memeriksa, dan bila commit menyimpan, satu baris.
func (s *Service) importRow(ctx context.Context, actor access.Actor, names map[string]nameOwner, r *ImportRow, commit bool) error {
	r.Aliases, r.NotAdded = []string{}, []string{}
	name, aliases, err := clean(Input{Name: r.Name, Aliases: splitAliases(r.AliasesText)})
	if err != nil {
		var invalid *problem.ValidationError
		if !errors.As(err, &invalid) {
			return err
		}
		r.Status, r.Problems = RowInvalid, renamed(invalid.Problems)
		return nil
	}
	r.Name, r.Aliases = name, aliases

	if owner, used := names[strings.ToLower(name)]; used {
		if !owner.display {
			r.Status = RowInvalid
			r.Problems = append(r.Problems, problem.Problem{Field: "process_name", Code: problem.Taken,
				Message: fmt.Sprintf("%s is already an alias of the process %s.", name, owner.process)})
			return nil
		}
		// Proses yang sudah ada: tidak diubah. Alias yang belum dimilikinya dilaporkan.
		r.Status = RowExisting
		for _, a := range aliases {
			if names[strings.ToLower(a)].process != owner.process {
				r.NotAdded = append(r.NotAdded, a)
			}
		}
		return nil
	}

	seen := map[string]bool{strings.ToLower(name): true}
	for _, a := range aliases {
		key := strings.ToLower(a)
		if owner, used := names[key]; used {
			r.Problems = append(r.Problems, problem.Problem{Field: "aliases", Code: problem.Taken,
				Message: fmt.Sprintf("The alias %s already belongs to the process %s.", a, owner.process)})
			continue
		}
		if seen[key] {
			r.Problems = append(r.Problems, problem.Problem{Field: "aliases", Code: problem.Taken,
				Message: fmt.Sprintf("%s is written twice in this row.", a)})
			continue
		}
		seen[key] = true
	}
	if len(r.Problems) > 0 {
		r.Status = RowInvalid
		return nil
	}

	if commit {
		if _, err := s.Create(ctx, actor, Input{Name: name, Aliases: aliases}); err != nil {
			var invalid *problem.ValidationError
			var conflict *problem.ConflictError
			switch {
			case errors.As(err, &invalid):
				r.Problems = renamed(invalid.Problems)
			case errors.As(err, &conflict):
				r.Problems = renamed(conflict.Problems)
			default:
				return err // error database menghentikan import; baris sebelumnya tetap tersimpan
			}
			r.Status = RowInvalid
			return nil
		}
	}
	// Baris berikutnya melihat nama baru ini, juga di Preview.
	r.Status = RowNew
	names[strings.ToLower(name)] = nameOwner{process: name, display: true}
	for _, a := range aliases {
		names[strings.ToLower(a)] = nameOwner{process: name}
	}
	return nil
}

// splitAliases memotong kolom Aliases menurut titik koma. Bagian kosong
// dilewati.
func splitAliases(text string) []string {
	var out []string
	for _, part := range strings.Split(text, ";") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// renamed mengganti field "name" layanan menjadi nama kolom berkas.
func renamed(list []problem.Problem) []problem.Problem {
	out := make([]problem.Problem, 0, len(list))
	for _, p := range list {
		if p.Field == "name" {
			p.Field = "process_name"
		}
		out = append(out, p)
	}
	return out
}

// readProcessBook membaca sheet Processes (atau sheet pertama). Judul kolom
// dicocokkan tanpa membedakan huruf besar kecil.
func readProcessBook(data []byte) ([]ImportRow, error) {
	if len(data) == 0 {
		return nil, fileProblem(problem.Required, "Choose the Excel file.")
	}
	if len(data) > MaxImportBytes {
		return nil, fileProblem(problem.TooLong, "The Excel file can be at most 5 MB.")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fileProblem(problem.InvalidFormat, "Upload an Excel workbook (.xlsx). Save an old .xls file as .xlsx first.")
	}
	defer book.Close()
	sheet := SheetProcesses
	if index, err := book.GetSheetIndex(sheet); err != nil || index < 0 {
		sheet = book.GetSheetName(0)
	}
	lines, err := book.GetRows(sheet)
	if err != nil {
		return nil, fileProblem(problem.InvalidFormat, "The "+sheet+" sheet cannot be read.")
	}
	column := map[string]int{}
	if len(lines) > 0 {
		for i, h := range lines[0] {
			column[strings.ToLower(strings.TrimSpace(h))] = i
		}
	}
	nameCol, ok := column["process name"]
	if !ok {
		return nil, fileProblem(problem.InvalidFormat,
			"The first row must have the columns "+strings.Join(importHeaders, ", ")+". Download the template.")
	}
	aliasCol, hasAliases := column["aliases"]
	cell := func(line []string, i int, ok bool) string {
		if !ok || i >= len(line) {
			return ""
		}
		return line[i]
	}
	rows := []ImportRow{}
	for y, line := range lines[min(1, len(lines)):] {
		name, aliases := cell(line, nameCol, true), cell(line, aliasCol, hasAliases)
		if strings.TrimSpace(name) == "" && strings.TrimSpace(aliases) == "" {
			continue
		}
		rows = append(rows, ImportRow{Row: y + 2, Name: strings.TrimSpace(name), AliasesText: strings.TrimSpace(aliases)})
	}
	switch {
	case len(rows) == 0:
		return nil, fileProblem(problem.Required, "The Excel file has no rows to import.")
	case len(rows) > MaxImportRows:
		return nil, fileProblem(problem.TooLong, fmt.Sprintf("Import at most %d rows at a time.", MaxImportRows))
	}
	return rows, nil
}

func fileProblem(code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: code, Message: message}}}
}
