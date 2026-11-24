package numbering

// Impor nomor lama dari berkas Excel (I4). PPAP Engineer (E-8) mengunduh
// template, mengisi satu set per baris (kode customer, nomor PFD, PFMEA, CP,
// dan status), lalu mengunggahnya. Aturannya sama dengan impor tempel (M2.5,
// D-216): nomor masuk dengan teks aslinya, satu baris satu batch, nomor yang
// sudah ada dilewati, dan lubang serta set yang tidak sejajar dilaporkan.
// Bedanya: satu berkas boleh memuat beberapa customer, dan status ditulis per
// baris. Nomor boleh memuat revisi, misalnya SC-KT-001-05 (opsi b).

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// SheetNumbers adalah nama sheet di template.
const SheetNumbers = "Numbers"

// MaxImportFileBytes adalah ukuran berkas paling besar.
const MaxImportFileBytes = 5 << 20 // 5 MB

// numberImportHeaders adalah judul kolom template, urut seperti di berkas.
// Kolom PFD, PFMEA, dan CP hanya menunjuk letak nomor; jenisnya tetap dibaca
// dari awalan nomor, seperti impor tempel.
var numberImportHeaders = []string{"Customer code", "PFD", "PFMEA", "CP", "Status"}

// fileStatuses adalah isian kolom Status yang diterima; kosong berarti
// Released (jawaban pemilik untuk pertanyaan I4).
var fileStatuses = map[string]string{"": StatusReleased, "released": StatusReleased, "reserved": StatusReserved}

// FileImportInput adalah isian impor berkas.
type FileImportInput struct {
	Data   []byte
	OneSet bool // nomor yang tidak ada di baris mengikuti nomor pertama baris itu
	DryRun bool // pratinjau: laporan saja
}

// ImportTemplate membuat berkas Excel template (E-8): sheet Numbers dengan
// judul kolom dan sheet Help. Template tidak memuat baris contoh.
func (s *Service) ImportTemplate(ctx context.Context, actor access.Actor) ([]byte, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{}); err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", SheetNumbers); err != nil {
		return nil, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	header := make([]any, 0, len(numberImportHeaders))
	for _, h := range numberImportHeaders {
		header = append(header, h)
	}
	_ = f.SetSheetRow(SheetNumbers, "A1", &header)
	_ = f.SetCellStyle(SheetNumbers, "A1", "E1", bold)
	_ = f.SetColWidth(SheetNumbers, "A", "A", 16)
	_ = f.SetColWidth(SheetNumbers, "B", "D", 22)
	_ = f.SetColWidth(SheetNumbers, "E", "E", 14)

	if _, err := f.NewSheet("Help"); err != nil {
		return nil, err
	}
	help := [][]any{
		{"Column", "What to fill in"},
		{"Customer code", "Required. The current or an old code of the customer, for example TR."},
		{"PFD, PFMEA, CP", "The numbers of one document set, for example SH-TR-020-02. The revision at the end is optional and is used when the document is migrated."},
		{"Status", "Released or Reserved. Empty means Released."},
		{"One set", "When One set is ticked on the screen, one number per row is enough; the other numbers follow its running number."},
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
		return nil, fmt.Errorf("menulis template nomor: %w", err)
	}
	return buf.Bytes(), nil
}

// ImportFile membaca berkas Excel nomor lama (E-8). Baris dikelompokkan per
// customer, dan setiap customer diimpor dalam transaksinya sendiri, jadi
// customer yang sudah tersimpan tetap tersimpan bila customer lain gagal.
// Laporannya urut menurut baris Excel.
func (s *Service) ImportFile(ctx context.Context, actor access.Actor, in FileImportInput) (ImportReport, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{}); err != nil {
		return ImportReport{}, err
	}
	if len(in.Data) > MaxImportFileBytes {
		return ImportReport{}, fileProblem(problem.TooLong, "The file is larger than 5 MB.")
	}
	lines, err := readNumberFile(in.Data)
	if err != nil {
		return ImportReport{}, err
	}
	customers, err := s.customersByCode(ctx)
	if err != nil {
		return ImportReport{}, err
	}

	// Baris yang customer atau statusnya salah langsung dilaporkan; sisanya
	// dikelompokkan per customer, urut menurut kemunculan pertama.
	var order []string
	byCustomer := map[string][]importRow{}
	var rejected []ImportLine
	for _, l := range lines {
		id, known := customers[l.code]
		status, statusOK := fileStatuses[strings.ToLower(l.status)]
		result := ""
		switch {
		case !known:
			result = LineUnknownCustomer
		case !statusOK:
			result = LineInvalidStatus
		case len(l.numbers) == 0:
			result = LineInvalid
		}
		if result == "" {
			list, err := s.checkCustomer(ctx, id)
			if err != nil {
				return ImportReport{}, err
			}
			if len(list) > 0 {
				result = LineInactiveCustomer
			}
		}
		if result != "" {
			texts := l.numbers
			if len(texts) == 0 {
				texts = []string{""}
			}
			for _, text := range texts {
				rejected = append(rejected, ImportLine{Line: l.row, Text: text, Result: result, CustomerCode: l.code, Status: status})
			}
			continue
		}
		if _, seen := byCustomer[id]; !seen {
			order = append(order, id)
		}
		for _, text := range l.numbers {
			byCustomer[id] = append(byCustomer[id], importRow{ImportLine: ImportLine{Line: l.row, Text: text, CustomerCode: l.code, Status: status}})
		}
	}

	env, err := s.importEnv(ctx)
	if err != nil {
		return ImportReport{}, err
	}
	report := ImportReport{DryRun: in.DryRun, Lines: rejected}
	for _, id := range order {
		if err := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{CustomerID: id}); err != nil {
			return ImportReport{}, err
		}
		part, err := s.importRows(ctx, actor, env, id, byCustomer[id], in.OneSet, in.DryRun)
		if err != nil {
			return ImportReport{}, err
		}
		report.Imported += part.Imported
		report.Lines = append(report.Lines, part.Lines...)
		report.Groups = append(report.Groups, part.Groups...)
		report.Misaligned = append(report.Misaligned, part.Misaligned...)
	}
	slices.SortStableFunc(report.Lines, func(a, b ImportLine) int { return a.Line - b.Line })
	return report, nil
}

// fileLine adalah satu baris berkas yang tidak kosong.
type fileLine struct {
	row     int // nomor baris di Excel, mulai dari 2
	code    string
	status  string
	numbers []string // isi kolom PFD, PFMEA, dan CP yang tidak kosong
}

// readNumberFile membaca sheet Numbers (atau sheet pertama). Judul kolom
// dicocokkan tanpa membedakan huruf besar kecil, jadi urutannya bebas.
func readNumberFile(data []byte) ([]fileLine, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fileProblem(problem.InvalidFormat, "Choose an Excel file (.xlsx) made from the template.")
	}
	defer f.Close()
	sheet := SheetNumbers
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		sheet = f.GetSheetName(0)
	}
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, fileProblem(problem.InvalidFormat, "The Numbers sheet is empty. Use the template.")
	}
	column := map[string]int{}
	for i, h := range rows[0] {
		column[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, h := range numberImportHeaders[:4] {
		if _, ok := column[strings.ToLower(h)]; !ok {
			return nil, fileProblem(problem.InvalidFormat, fmt.Sprintf("The column %q is missing. Use the template.", h))
		}
	}
	cell := func(row []string, header string) string {
		i, ok := column[strings.ToLower(header)]
		if !ok || i >= len(row) {
			return ""
		}
		text := strings.TrimSpace(row[i])
		if runes := []rune(text); len(runes) > maxImportCell {
			text = string(runes[:maxImportCell])
		}
		return text
	}
	var out []fileLine
	count := 0
	for y, row := range rows[1:] {
		l := fileLine{row: y + 2, code: strings.ToUpper(cell(row, "Customer code")), status: cell(row, "Status")}
		for _, h := range numberImportHeaders[1:4] {
			if text := cell(row, h); text != "" {
				l.numbers = append(l.numbers, text)
			}
		}
		if l.code == "" && l.status == "" && len(l.numbers) == 0 {
			continue
		}
		count += max(len(l.numbers), 1)
		out = append(out, l)
	}
	switch {
	case len(out) == 0:
		return nil, fileProblem(problem.Required, "The file has no numbers.")
	case count > MaxImportLines:
		return nil, fileProblem(problem.TooLong, fmt.Sprintf("Import at most %d numbers at a time.", MaxImportLines))
	}
	return out, nil
}

// customersByCode memetakan setiap kode customer, juga kode lama (E-11), ke
// ID customer-nya.
func (s *Service) customersByCode(ctx context.Context) (map[string]string, error) {
	list, err := s.q.ListCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca customer: %w", err)
	}
	out := map[string]string{}
	for _, c := range list {
		out[c.Code] = c.ID
	}
	history, err := s.q.ListCustomerCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca kode customer: %w", err)
	}
	for _, h := range history {
		if _, taken := out[h.Code]; !taken {
			out[h.Code] = h.CustomerID
		}
	}
	return out, nil
}

// fileProblem membuat penolakan untuk seluruh berkas.
func fileProblem(code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: code, Message: message}}}
}
