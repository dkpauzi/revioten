package printout

// Excel A3 dan A4 (M6.2, R-PRT-01). Satu sheet cover (bila layout memintanya)
// dan satu sheet isi. Sheet isi punya judul kolom yang berulang di setiap
// halaman, area cetak, muat satu halaman lebar, dan header berisi watermark
// (M6.4) serta footer berisi nomor, revisi, status, halaman, dan hash (M6.9).
// Excel tidak menghitung tinggi baris dengan teks terbungkus untuk berkas
// buatan program, jadi tingginya diperkirakan dari panjang teks.

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// Ukuran kertas Excel (ECMA-376): 8 = A3, 9 = A4.
var paperSize = map[string]int{"A3": 8, "A4": 9}

// XLSX menulis model cetak sebagai berkas Excel.
func XLSX(sh Sheet) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	st, err := newStyles(f, sh.Layout.FontSize)
	if err != nil {
		return nil, err
	}
	data := sheetName(sh)
	if err := f.SetSheetName("Sheet1", data); err != nil {
		return nil, err
	}
	if sh.Layout.Cover {
		if _, err := f.NewSheet("Cover"); err != nil {
			return nil, err
		}
		// Cover di depan. Dipindah sebelum ditulis, karena nama terdefinisi
		// (area cetak) menyimpan nomor urut sheet.
		if err := f.MoveSheet("Cover", data); err != nil {
			return nil, err
		}
		f.SetActiveSheet(0)
		if err := writeCover(f, "Cover", sh, st); err != nil {
			return nil, err
		}
	}
	if err := writeData(f, data, sh, st); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sheetName(sh Sheet) string {
	switch sh.Snapshot.DocumentType {
	case "pfd":
		return "PFD"
	case "pfmea":
		return "PFMEA"
	case "cp":
		return "CP"
	}
	return "Document"
}

type styles struct{ title, label, value, head, cell, mark, section, small int }

func newStyles(f *excelize.File, size float64) (styles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1},
	}
	wrap := &excelize.Alignment{WrapText: true, Vertical: "top"}
	mk := func(s *excelize.Style) int {
		id, err := f.NewStyle(s)
		if err != nil {
			panic(err) // gaya tetap; gagal berarti bug
		}
		return id
	}
	var out styles
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("membuat gaya Excel: %v", r)
			}
		}()
		out.title = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size + 6, Bold: true}})
		out.label = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size, Bold: true}, Border: border, Alignment: wrap})
		out.value = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size}, Border: border, Alignment: wrap})
		out.head = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size, Bold: true}, Border: border,
			Alignment: &excelize.Alignment{WrapText: true, Vertical: "center", Horizontal: "center"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"D9D9D9"}, Pattern: 1}})
		out.cell = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size}, Border: border, Alignment: wrap})
		out.mark = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size, Bold: true}, Border: border, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "top"}})
		out.section = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size + 1, Bold: true}, Border: border, Fill: excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1}})
		out.small = mk(&excelize.Style{Font: &excelize.Font{Family: "Arial", Size: size - 1, Italic: true}, Border: border, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}})
	}()
	return out, err
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// widthsChars mengubah lebar relatif menjadi lebar Excel (jumlah karakter),
// kira-kira selebar halaman yang bisa dipakai.
func widthsChars(sh Sheet, extra float64) []float64 {
	pageW, _ := sh.Layout.PageMM()
	usable := pageW - sh.Layout.MarginsMM.Left - sh.Layout.MarginsMM.Right
	total := extra
	for _, c := range sh.Columns {
		total += c.Width
	}
	if total == 0 {
		total = 1
	}
	// Satu karakter huruf 8 pt kira-kira 1,6 mm.
	perUnit := usable / 1.6 * (8 / sh.Layout.FontSize) / total
	out := make([]float64, 0, len(sh.Columns)+1)
	if extra > 0 {
		out = append(out, math.Max(3, extra*perUnit))
	}
	for _, c := range sh.Columns {
		out = append(out, math.Max(3, c.Width*perUnit))
	}
	return out
}

// rowHeight memperkirakan tinggi baris (pt) dari panjang teks terpanjang.
func rowHeight(values []string, widths []float64, size float64) float64 {
	lines := 1
	for i, v := range values {
		if i >= len(widths) {
			break
		}
		perLine := int(math.Max(1, widths[i]*1.15))
		n := 0
		for _, part := range strings.Split(v, "\n") {
			n += int(math.Max(1, math.Ceil(float64(utf8.RuneCountInString(part))/float64(perLine))))
		}
		lines = max(lines, n)
	}
	return math.Min(409, float64(lines)*size*1.35+3)
}

func pageSetup(f *excelize.File, name string, sh Sheet, titleRows string, lastCol, lastRow int) error {
	size := paperSize[sh.Layout.Paper]
	orient := sh.Layout.Orientation
	one, zero := 1, 0
	if err := f.SetPageLayout(name, &excelize.PageLayoutOptions{Size: &size, Orientation: &orient, FitToWidth: &one, FitToHeight: &zero}); err != nil {
		return err
	}
	fit := true
	if err := f.SetSheetProps(name, &excelize.SheetPropsOptions{FitToPage: &fit}); err != nil {
		return err
	}
	inch := func(mm float64) *float64 { v := mm / 25.4; return &v }
	m := sh.Layout.MarginsMM
	if err := f.SetPageMargins(name, &excelize.PageLayoutMarginsOptions{Top: inch(m.Top + 6), Right: inch(m.Right), Bottom: inch(m.Bottom + 6), Left: inch(m.Left), Header: inch(4), Footer: inch(4)}); err != nil {
		return err
	}
	area := fmt.Sprintf("'%s'!$A$1:$%s$%d", name, colName(lastCol), lastRow)
	if err := f.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Area", RefersTo: area, Scope: name}); err != nil {
		return err
	}
	if titleRows != "" {
		if err := f.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Titles", RefersTo: fmt.Sprintf("'%s'!%s", name, titleRows), Scope: name}); err != nil {
			return err
		}
	}
	hash := sh.ContentHash
	if len(hash) > 12 {
		hash = hash[:12]
	}
	footer := fmt.Sprintf("&L&\"Arial\"&8%s %s · %s&C&\"Arial\"&8Page &P of &N&R&\"Arial\"&8SHA-256 %s", sh.Number, sh.RevisionLabel, sh.Status, hash)
	header := ""
	if sh.Watermark != "" {
		header = "&C&\"Arial,Bold\"&16&KC00000" + sh.Watermark
	}
	return f.SetHeaderFooter(name, &excelize.HeaderFooterOptions{OddHeader: header, OddFooter: footer})
}

func colName(c int) string {
	name, _ := excelize.ColumnNumberToName(c)
	return name
}

// writeData menulis sheet isi: judul, header, blok tanda tangan, lalu tabel.
func writeData(f *excelize.File, name string, sh Sheet, st styles) error {
	markCol := 0.0
	if sh.Layout.RevisionMarks {
		markCol = 0.35
	}
	widths := widthsChars(sh, markCol)
	cols := len(widths)
	for i, w := range widths {
		if err := f.SetColWidth(name, colName(i+1), colName(i+1), w); err != nil {
			return err
		}
	}
	row := 1
	_ = f.SetCellValue(name, cell(1, row), sh.Title)
	_ = f.SetCellStyle(name, cell(1, row), cell(1, row), st.title)
	_ = f.SetCellValue(name, cell(cols, row), sh.Number+" "+sh.RevisionLabel)
	_ = f.SetCellStyle(name, cell(cols, row), cell(cols, row), st.title)
	row += 2
	// Header: pasangan label dan nilai, beberapa pasangan per baris.
	perRow := max(1, cols/4)
	span := max(1, cols/(perRow*2))
	for i := 0; i < len(sh.Header); i += perRow {
		col := 1
		for _, p := range sh.Header[i:min(i+perRow, len(sh.Header))] {
			labelEnd, valueEnd := col+span-1, col+2*span-1
			_ = f.MergeCell(name, cell(col, row), cell(labelEnd, row))
			_ = f.MergeCell(name, cell(labelEnd+1, row), cell(valueEnd, row))
			_ = f.SetCellValue(name, cell(col, row), p.Label)
			_ = f.SetCellStyle(name, cell(col, row), cell(labelEnd, row), st.label)
			_ = f.SetCellValue(name, cell(labelEnd+1, row), p.Value)
			_ = f.SetCellStyle(name, cell(labelEnd+1, row), cell(valueEnd, row), st.value)
			col = valueEnd + 1
		}
		row++
	}
	row++
	// Blok tanda tangan di setiap sheet: dari rute (R-APR-08). Dengan cover,
	// sheet isi menulis "Refer approval sheet" (R-PRT-07).
	if len(sh.Slots) > 0 {
		per := max(1, cols/len(sh.Slots))
		for i, s := range sh.Slots {
			start := 1 + i*per
			end := start + per - 1
			if i == len(sh.Slots)-1 {
				end = cols
			}
			_ = f.MergeCell(name, cell(start, row), cell(end, row))
			_ = f.SetCellValue(name, cell(start, row), s.Title)
			_ = f.SetCellStyle(name, cell(start, row), cell(end, row), st.head)
			_ = f.MergeCell(name, cell(start, row+1), cell(end, row+1))
			sign := ""
			if sh.Layout.Cover {
				sign = "Refer approval sheet"
			}
			_ = f.SetCellValue(name, cell(start, row+1), sign)
			_ = f.SetCellStyle(name, cell(start, row+1), cell(end, row+1), st.small)
			_ = f.MergeCell(name, cell(start, row+2), cell(end, row+2))
			_ = f.SetCellValue(name, cell(start, row+2), s.Name())
			_ = f.SetCellStyle(name, cell(start, row+2), cell(end, row+2), st.value)
		}
		_ = f.SetRowHeight(name, row+1, 28)
		row += 4
	}
	// Judul tabel: baris grup (bila ada) dan baris label, berulang di setiap halaman.
	first := row
	offset := 0
	if sh.Layout.RevisionMarks {
		offset = 1
	}
	hasGroups := false
	for _, c := range sh.Columns {
		hasGroups = hasGroups || c.Group != ""
	}
	if hasGroups {
		for i := 0; i < len(sh.Columns); {
			j := i
			for j+1 < len(sh.Columns) && sh.Columns[j+1].Group == sh.Columns[i].Group {
				j++
			}
			_ = f.MergeCell(name, cell(i+1+offset, row), cell(j+1+offset, row))
			_ = f.SetCellValue(name, cell(i+1+offset, row), sh.Columns[i].Group)
			_ = f.SetCellStyle(name, cell(i+1+offset, row), cell(j+1+offset, row), st.head)
			i = j + 1
		}
		if offset == 1 {
			_ = f.SetCellStyle(name, cell(1, row), cell(1, row), st.head)
		}
		row++
	}
	labels := []string{}
	if offset == 1 {
		labels = append(labels, "Rev")
	}
	for _, c := range sh.Columns {
		labels = append(labels, c.Label)
	}
	for i, l := range labels {
		_ = f.SetCellValue(name, cell(i+1, row), l)
	}
	_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.head)
	_ = f.SetRowHeight(name, row, rowHeight(labels, widths, sh.Layout.FontSize))
	titleRows := fmt.Sprintf("$%d:$%d", first, row)
	row++
	for _, r := range sh.Rows {
		if r.Section != "" || r.Cells == nil {
			_ = f.MergeCell(name, cell(1, row), cell(cols, row))
			_ = f.SetCellValue(name, cell(1, row), r.Section)
			_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.section)
			row++
			continue
		}
		values := []string{}
		if offset == 1 {
			values = append(values, r.Mark)
		}
		values = append(values, r.Cells...)
		for i, v := range values {
			_ = f.SetCellValue(name, cell(i+1, row), v)
		}
		_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.cell)
		if offset == 1 {
			_ = f.SetCellStyle(name, cell(1, row), cell(1, row), st.mark)
		}
		_ = f.SetRowHeight(name, row, rowHeight(values, widths, sh.Layout.FontSize))
		row++
	}
	return pageSetup(f, name, sh, titleRows, cols, row-1)
}

// writeCover menulis cover TR-020 (R-PRT-07): nomor, judul, header, tabel
// riwayat revisi (M6.8), slot approval dari rute, area CUSTOMER APPROVAL,
// dan nama customer di bawah.
func writeCover(f *excelize.File, name string, sh Sheet, st styles) error {
	const cols = 8
	for i := 1; i <= cols; i++ {
		_ = f.SetColWidth(name, colName(i), colName(i), 16)
	}
	_ = f.SetColWidth(name, "A", "A", 6)
	_ = f.SetColWidth(name, "D", "D", 44)
	row := 1
	_ = f.SetCellValue(name, cell(cols, row), sh.Number+" "+sh.RevisionLabel)
	_ = f.SetCellStyle(name, cell(cols, row), cell(cols, row), st.label)
	row += 2
	_ = f.MergeCell(name, cell(1, row), cell(cols, row))
	_ = f.SetCellValue(name, cell(1, row), sh.Title)
	_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.title)
	row += 2
	for _, p := range sh.Header {
		_ = f.MergeCell(name, cell(1, row), cell(2, row))
		_ = f.MergeCell(name, cell(3, row), cell(cols, row))
		_ = f.SetCellValue(name, cell(1, row), p.Label)
		_ = f.SetCellStyle(name, cell(1, row), cell(2, row), st.label)
		_ = f.SetCellValue(name, cell(3, row), p.Value)
		_ = f.SetCellStyle(name, cell(3, row), cell(cols, row), st.value)
		row++
	}
	row++
	// Riwayat revisi: No, Date, Revision, Description of Change.
	_ = f.MergeCell(name, cell(4, row), cell(cols, row))
	for i, h := range []string{"No", "Date", "Revision", "Description of Change"} {
		_ = f.SetCellValue(name, cell(i+1, row), h)
	}
	_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.head)
	row++
	for _, h := range sh.History {
		_ = f.MergeCell(name, cell(4, row), cell(cols, row))
		for i, v := range []any{h.No, h.Date, h.Revision, h.Description} {
			_ = f.SetCellValue(name, cell(i+1, row), v)
		}
		_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.cell)
		_ = f.SetRowHeight(name, row, rowHeight([]string{"", "", "", h.Description}, []float64{6, 16, 16, 100}, sh.Layout.FontSize))
		row++
	}
	row++
	// Slot approval dari rute: judul cetak, ruang tanda tangan, nama.
	if len(sh.Slots) > 0 {
		per := max(1, cols/len(sh.Slots))
		for i, s := range sh.Slots {
			start := 1 + i*per
			end := start + per - 1
			if i == len(sh.Slots)-1 {
				end = cols
			}
			for r, v := range []string{s.Title, "", s.Name()} {
				_ = f.MergeCell(name, cell(start, row+r), cell(end, row+r))
				_ = f.SetCellValue(name, cell(start, row+r), v)
				style := st.value
				if r == 0 {
					style = st.head
				}
				_ = f.SetCellStyle(name, cell(start, row+r), cell(end, row+r), style)
			}
		}
		_ = f.SetRowHeight(name, row+1, 46)
		row += 4
	}
	_ = f.MergeCell(name, cell(1, row), cell(cols, row))
	_ = f.SetCellValue(name, cell(1, row), "CUSTOMER APPROVAL")
	_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.head)
	_ = f.MergeCell(name, cell(1, row+1), cell(cols, row+1))
	_ = f.SetCellStyle(name, cell(1, row+1), cell(cols, row+1), st.value)
	_ = f.SetRowHeight(name, row+1, 60)
	row += 3
	if sh.ReleaseNote != "" {
		_ = f.MergeCell(name, cell(1, row), cell(cols, row))
		_ = f.SetCellValue(name, cell(1, row), sh.ReleaseNote)
		row++
	}
	_ = f.MergeCell(name, cell(1, row), cell(cols, row))
	_ = f.SetCellValue(name, cell(1, row), sh.CustomerName)
	_ = f.SetCellStyle(name, cell(1, row), cell(cols, row), st.label)
	if len(sh.QRPNG) > 0 {
		if err := f.AddPictureFromBytes(name, cell(cols-1, 1), &excelize.Picture{Extension: ".png", File: sh.QRPNG,
			Format: &excelize.GraphicOptions{ScaleX: 0.6, ScaleY: 0.6}}); err != nil {
			return err
		}
	}
	cover := sh
	cover.Layout.Orientation = "portrait"
	cover.Layout.Paper = "A4"
	return pageSetup(f, name, cover, "", cols, row)
}
