package document

// Pembanding Excel (M7.3, R-MIG-04). Excel lama diunggah hanya untuk
// dibandingkan dengan isi dokumen; tidak ada yang diimpor dan berkasnya
// tidak disimpan (D-151).
//
// Form Excel lama tidak seragam (sel gabung, judul bertingkat, kolom
// bergeser), jadi pembanding ini tidak mencoba memetakan kolom. Ia
// membandingkan isi sel sebagai kumpulan teks: setiap isi sel dokumen dicari
// di sheet Excel, dan setiap teks di sheet Excel dicari di dokumen. Huruf
// besar-kecil dan spasi diabaikan. Hasilnya dua daftar selisih yang
// diperiksa orang.

import (
	"context"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// MaxCompareBytes adalah ukuran terbesar Excel pembanding.
const MaxCompareBytes = 20 << 20

// maxCompareItems membatasi panjang setiap daftar selisih.
const maxCompareItems = 500

// ExcelComparison adalah hasil membandingkan dokumen dengan satu sheet Excel.
type ExcelComparison struct {
	Sheets            []string       // semua sheet di berkas
	Sheet             string         // sheet yang dibandingkan
	Matched           int            // isi sel dokumen yang ada di Excel
	MissingInExcel    []DocumentCell // ada di dokumen, tidak ada di Excel
	MissingInDocument []ExcelCell    // ada di Excel, tidak ada di dokumen
	Truncated         bool           // daftar dipotong karena terlalu panjang
}

// DocumentCell adalah satu isi dokumen. Position 0 berarti header.
type DocumentCell struct {
	Position int
	Label    string
	Text     string
}

// ExcelCell adalah satu sel Excel, misalnya C12.
type ExcelCell struct {
	Cell string
	Text string
}

// cellString mengubah nilai isi menjadi teks; ok false untuk nilai yang
// bukan teks atau angka.
func cellString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, strings.TrimSpace(x) != ""
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case int:
		return strconv.Itoa(x), true
	}
	return "", false
}

// hasLetter melaporkan apakah teks memuat huruf.
func hasLetter(s string) bool {
	return strings.ContainsFunc(s, unicode.IsLetter)
}

// documentCells memberi semua isi dokumen yang dibandingkan, dan teks form
// (label kolom, judul grup) yang diabaikan di sisi Excel.
func documentCells(d Document, format template.FormatVersion) ([]DocumentCell, map[string]bool) {
	var cells []DocumentCell
	furniture := map[string]bool{}
	for _, f := range format.Fields {
		furniture[normalize(f.Label)] = true
		if f.Group != "" {
			furniture[normalize(f.Group)] = true
		}
		if f.Area == template.AreaHeader && !f.Hidden {
			if text, ok := cellString(d.Header[f.ID]); ok {
				cells = append(cells, DocumentCell{Label: f.Label, Text: text})
			}
		}
	}
	for _, r := range d.Rows {
		if r.ProcessName != "" {
			cells = append(cells, DocumentCell{Position: r.Position, Label: "Process", Text: r.ProcessName})
		}
		if r.OperationNumber != "" {
			cells = append(cells, DocumentCell{Position: r.Position, Label: "Operation No.", Text: r.OperationNumber})
		}
		for _, f := range format.Fields {
			if f.Area != template.AreaRow || f.Hidden {
				continue
			}
			if text, ok := cellString(r.Content[f.ID]); ok {
				cells = append(cells, DocumentCell{Position: r.Position, Label: f.Label, Text: text})
			}
		}
	}
	return cells, furniture
}

// CompareExcel membandingkan isi dokumen dengan satu sheet Excel lama (L-4).
// sheet kosong berarti sheet yang paling banyak cocok.
func (s *Service) CompareExcel(ctx context.Context, actor access.Actor, id, sheet string, r io.Reader) (ExcelComparison, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return ExcelComparison{}, err
	}
	format, err := template.LoadFormatVersion(ctx, s.q, d.FormatVersionID)
	if err != nil {
		return ExcelComparison{}, result(err, "membaca format")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxCompareBytes+1))
	if err != nil {
		return ExcelComparison{}, result(err, "membaca berkas")
	}
	if len(data) == 0 {
		return ExcelComparison{}, problems(invalid("file", problem.Required, "Choose the legacy Excel file."))
	}
	if len(data) > MaxCompareBytes {
		return ExcelComparison{}, problems(invalid("file", problem.TooLong, "The Excel file can be at most 20 MB."))
	}
	book, err := excelize.OpenReader(strings.NewReader(string(data)))
	if err != nil {
		return ExcelComparison{}, problems(invalid("file", problem.InvalidFormat, "Upload an Excel workbook (.xlsx). Save an old .xls file as .xlsx first."))
	}
	defer book.Close()

	cells, furniture := documentCells(d, format)
	inDocument := map[string]bool{}
	for _, c := range cells {
		inDocument[normalize(c.Text)] = true
	}
	// compare menghitung selisih untuk satu sheet.
	compare := func(name string) (ExcelComparison, error) {
		rows, err := book.GetRows(name)
		if err != nil {
			return ExcelComparison{}, err
		}
		out := ExcelComparison{Sheet: name, MissingInExcel: []DocumentCell{}, MissingInDocument: []ExcelCell{}}
		inExcel := map[string]bool{}
		reported := map[string]bool{}
		for y, row := range rows {
			for x, text := range row {
				key := normalize(text)
				if key == "" {
					continue
				}
				inExcel[key] = true
				// Angka lepas (nomor urut, rating) hanya diperiksa dari sisi dokumen.
				if inDocument[key] || furniture[key] || reported[key] || !hasLetter(key) {
					continue
				}
				reported[key] = true
				cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
				out.MissingInDocument = append(out.MissingInDocument, ExcelCell{Cell: cell, Text: strings.TrimSpace(text)})
			}
		}
		for _, c := range cells {
			if inExcel[normalize(c.Text)] {
				out.Matched++
			} else {
				out.MissingInExcel = append(out.MissingInExcel, c)
			}
		}
		return out, nil
	}

	sheets := book.GetSheetList()
	var best ExcelComparison
	if sheet != "" {
		found := false
		for _, name := range sheets {
			found = found || name == sheet
		}
		if !found {
			return ExcelComparison{}, problems(invalid("sheet", problem.InvalidFormat, "Choose a sheet of the uploaded file."))
		}
		if best, err = compare(sheet); err != nil {
			return ExcelComparison{}, result(err, "membaca sheet")
		}
	} else {
		for i, name := range sheets {
			c, err := compare(name)
			if err != nil {
				return ExcelComparison{}, result(err, "membaca sheet")
			}
			if i == 0 || c.Matched > best.Matched {
				best = c
			}
		}
	}
	best.Sheets = sheets
	if len(best.MissingInExcel) > maxCompareItems {
		best.MissingInExcel, best.Truncated = best.MissingInExcel[:maxCompareItems], true
	}
	if len(best.MissingInDocument) > maxCompareItems {
		best.MissingInDocument, best.Truncated = best.MissingInDocument[:maxCompareItems], true
	}
	return best, nil
}
