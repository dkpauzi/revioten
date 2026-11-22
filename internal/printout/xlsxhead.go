package printout

// Kepala lembar dan cover di Excel (bagian C, E-144), dengan susunan yang
// sama dengan head.go. Kolom kepala adalah kolom "logis" (misalnya tujuh
// kolom kepala CP) yang dipetakan ke kolom lembar menurut lebar relatifnya,
// lalu digabung. Logo ditempel dengan jangkar satu sel, supaya ukurannya
// tidak ikut melebar seperti shape (lihat xlsxdraw.go).

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// headWriter menulis kepala di satu sheet.
type headWriter struct {
	f      *excelize.File
	name   string
	cols   [][2]int // kolom lembar (mulai 1) untuk setiap kolom logis
	scale  float64  // skala muat-lebar Excel
	styles map[string]int
	size   float64
}

// logicalCols membagi kolom lembar selebar widths (piksel) menjadi kolom
// logis menurut persentase pcts. Setiap kolom logis mendapat paling sedikit
// satu kolom lembar.
func logicalCols(widths []float64, pcts []float64) [][2]int {
	total := 0.0
	for _, w := range widths {
		total += w
	}
	out := make([][2]int, len(pcts))
	col, acc, want := 0, 0.0, 0.0
	for i, p := range pcts {
		want += p / 100 * total
		start := col
		for col < len(widths) && (col == start || acc+widths[col]/2 <= want) && len(widths)-col > len(pcts)-1-i {
			acc += widths[col]
			col++
		}
		out[i] = [2]int{start + 1, max(start+1, col)}
	}
	out[len(pcts)-1][1] = len(widths)
	return out
}

// style membuat atau mengambil gaya: ukuran relatif, tebal, bergaris, rata.
func (h *headWriter) style(rel float64, bold, border bool, align, valign string, wrap bool) int {
	key := fmt.Sprint(rel, bold, border, align, valign, wrap)
	if id, ok := h.styles[key]; ok {
		return id
	}
	s := &excelize.Style{Font: &excelize.Font{Family: "Arial", Size: math.Max(5, h.size*rel), Bold: bold},
		Alignment: &excelize.Alignment{Horizontal: align, Vertical: valign, WrapText: wrap}}
	if border {
		s.Border = []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}}
	}
	id, _ := h.f.NewStyle(s)
	h.styles[key] = id
	return id
}

// put menulis teks di kolom logis a sampai b, baris r1 sampai r2 (digabung).
func (h *headWriter) put(a, b, r1, r2 int, v string, style int) {
	c1, c2 := cell(h.cols[a][0], r1), cell(h.cols[b][1], r2)
	if c1 != c2 {
		_ = h.f.MergeCell(h.name, c1, c2)
	}
	_ = h.f.SetCellValue(h.name, c1, v)
	_ = h.f.SetCellStyle(h.name, c1, c2, style)
}

// height memberi tinggi baris (point lembar) untuk mm di kertas.
func (h *headWriter) height(r int, mm float64) {
	_ = h.f.SetRowHeight(h.name, r, math.Min(409, mm*mmPt/h.scale))
}

// logo menempel logo setinggi mm di sel kolom logis a, baris r.
func (h *headWriter) logo(data []byte, a, r int, mm float64) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Height == 0 {
		return
	}
	px := mm * 96 / 25.4 / h.scale
	k := px / float64(cfg.Height)
	_ = h.f.AddPictureFromBytes(h.name, cell(h.cols[a][0], r), &excelize.Picture{Extension: ".png", File: data,
		Format: &excelize.GraphicOptions{ScaleX: k, ScaleY: k, OffsetX: 2, OffsetY: 2, Positioning: "oneCell"}})
}

// newHeadWriter menyiapkan penulis kepala untuk kolom lembar selebar widths
// (karakter) dan kolom logis pcts.
func newHeadWriter(f *excelize.File, name string, sh Sheet, widths []float64, pcts []float64) *headWriter {
	px := make([]float64, len(widths))
	sheetPt := 0.0
	for i, w := range widths {
		px[i] = excelColPx(w)
		sheetPt += px[i] * 0.75
	}
	content := (paperWidthMM(sh.Layout) - sh.Layout.MarginsMM.Left - sh.Layout.MarginsMM.Right) * mmPt
	scale := 1.0
	if sheetPt > 0 {
		scale = math.Min(1, content/sheetPt)
	}
	size := sh.Layout.FontSize
	if size <= 0 {
		size = 8
	}
	return &headWriter{f: f, name: name, cols: logicalCols(px, pcts), scale: scale, styles: map[string]int{}, size: size}
}

// paperWidthMM memberi lebar kertas layout dalam milimeter.
func paperWidthMM(l Layout) float64 {
	w, _ := paperMM(l)
	return w
}

// writeFormHead menulis kepala bentuk TR-020 untuk CP, PFMEA, dan PFD mulai
// baris 1. Hasilnya baris pertama sesudah kepala; ok false untuk jenis lain.
func writeFormHead(f *excelize.File, name string, sh Sheet, widths []float64) (int, bool) {
	hd := formHead(sh)
	switch sh.Snapshot.DocumentType {
	case "cp":
		h := newHeadWriter(f, name, sh, widths, []float64{11, 20, 33, 10, 14, 6, 6})
		th := h.style(1, true, true, "center", "center", true)
		l := h.style(1, true, true, "left", "center", false)
		v := h.style(1, false, true, "left", "center", true)
		small := h.style(0.75, false, true, "center", "center", true)
		h.put(0, 1, 1, 2, hd.Company, h.style(1.6, true, true, "right", "center", false))
		h.logo(sh.Logo, 0, 1, 9)
		h.put(2, 2, 1, 2, "CONTROL PLAN", h.style(2, true, true, "center", "center", false))
		h.put(3, 3, 1, 2, "Department", th)
		h.put(4, 4, 1, 2, "Name", th)
		h.put(5, 6, 1, 1, "Signature", th)
		h.put(5, 5, 2, 2, "Date", th)
		h.put(6, 6, 2, 2, "Signature", th)
		h.height(1, 6)
		h.height(2, 6)
		view := newHeadView(sh)
		n := len(view.Slots)
		for i, s := range view.Slots {
			r := 3 + i
			h.height(r, 5)
			switch i {
			case 0:
				h.put(0, 0, r, r, "Control Plan No.", l)
				h.put(1, 1, r, r, sh.Number, v)
				h.put(2, 2, r, r+n-1, hd.PartNumbers+" "+hd.PartNames, h.style(2.2, true, true, "center", "center", true))
			case 1:
				h.put(0, 0, r, r, "Product", l)
				h.put(1, 1, r, r, hd.PartNames, v)
			case 2:
				h.put(0, 0, r, r+n-3, "Core Team", l)
				h.put(1, 1, r, r+n-3, hd.CoreTeam, h.style(1, false, true, "left", "top", true))
			}
			h.put(3, 3, r, r, s.Department, l)
			h.put(4, 4, r, r, s.Name, v)
			h.put(5, 5, r, r, s.Date, small)
			h.put(6, 6, r, r, s.Signature, small)
		}
		r := 3 + n
		h.height(r, 5)
		h.height(r+1, 5)
		h.put(0, 0, r, r, "Customer Name", l)
		h.put(1, 1, r, r, hd.Customer, v)
		h.put(2, 2, r, r+1, "Remark : "+hd.Remark, h.style(1, false, true, "left", "top", true))
		h.put(0, 0, r+1, r+1, "Rev. Description", l)
		h.put(1, 1, r+1, r+1, hd.Revision, v)
		return r + 3, true
	case "pfmea":
		h := newHeadWriter(f, name, sh, widths, []float64{12, 22, 16, 20, 12, 18})
		l := h.style(1, false, false, "left", "center", false)
		h.put(0, 1, 1, 1, hd.Customer, h.style(2.2, true, false, "left", "top", true))
		h.put(2, 4, 1, 1, "POTENTIAL\nPROCESS FAILURE MODE AND EFFECTS ANALYSIS\n( PROCESS FMEA )", h.style(1.3, true, false, "center", "center", true))
		h.height(1, 16)
		rows := [][]string{
			{"Part Name", hd.PartNames, "Process Responsibility", hd.Responsibility, "FMEA NO.", sh.Number},
			{"Part No.", hd.PartNumbers, "Key Date", hd.KeyDate, "Prepared by :", hd.PreparedBy},
			{"Model", hd.Models, "Revision", hd.Revision, "", ""},
		}
		for i, row := range rows {
			r := 2 + i
			h.height(r, 5)
			for c, text := range row {
				h.put(c, c, r, r, text, l)
			}
		}
		for i, row := range [][2]string{{"Process", hd.ProcessLine}, {"Core Team", ": " + hd.CoreTeam}, {"Remark", ": " + hd.Remark}} {
			r := 5 + i
			h.height(r, 5)
			h.put(0, 0, r, r, row[0], l)
			h.put(1, 5, r, r, row[1], l)
		}
		return 9, true
	case "pfd":
		h := newHeadWriter(f, name, sh, widths, []float64{18, 57, 10, 15})
		h.put(0, 2, 1, 1, "Process Flow Chart for "+hd.Models+" Unit", h.style(2, true, false, "center", "center", false))
		h.put(3, 3, 1, 1, "DOC. NO "+sh.Number, h.style(1, false, false, "right", "top", false))
		h.height(1, 12)
		for i, row := range [][2]string{{"CUSTOMER", hd.Customer}, {"MODEL", hd.Models}, {"PART NAME", hd.PartNames}, {"PART NUMBER", hd.PartNumbers}, {"PROCESS LINE", hd.ProcessLine}} {
			r := 2 + i
			h.height(r, 5)
			h.put(0, 0, r, r, row[0], h.style(1, true, false, "left", "center", false))
			h.put(1, 3, r, r, row[1], h.style(1, false, false, "left", "center", false))
		}
		return 8, true
	}
	return 0, false
}

// writeCover menulis cover bentuk TR-020 (R-PRT-07) di A4 tegak: logo,
// "<jenis> No :", judul, isian customer sampai Process or Line, Document
// Revision History, INHOUSE APPROVAL dari rute, CUSTOMER APPROVAL, dan nama
// customer.
func writeCover(f *excelize.File, name string, sh Sheet, st styles) error {
	widths := []float64{4, 12, 12, 12, 12, 12, 12, 12, 12, 12}
	for i, w := range widths {
		_ = f.SetColWidth(name, colName(i+1), colName(i+1), w)
	}
	cover := sh
	cover.Layout.Orientation, cover.Layout.Paper = "portrait", "A4"
	cover.Layout.FontSize = 10
	h := newHeadWriter(f, name, cover, widths, []float64{4, 12, 12, 12, 12, 12, 12, 12, 12})
	view := newHeadView(sh)
	hd := view.FormHead
	last := len(h.cols) - 1
	h.height(1, 16)
	h.logo(sh.Logo, 0, 1, 14)
	h.put(6, 7, 1, 1, view.CoverNo, h.style(1, true, false, "right", "top", false))
	h.put(8, last, 1, 1, sh.Number, h.style(1, true, false, "left", "top", false))
	if len(sh.QRPNG) > 0 {
		_ = f.AddPictureFromBytes(name, cell(h.cols[last][0], 2), &excelize.Picture{Extension: ".png", File: sh.QRPNG,
			Format: &excelize.GraphicOptions{ScaleX: 0.5, ScaleY: 0.5, Positioning: "oneCell"}})
	}
	h.height(3, 14)
	h.put(0, last, 3, 3, view.Title, h.style(2, true, false, "center", "center", false))
	r := 5
	label := h.style(1.4, true, false, "left", "center", false)
	value := h.style(1.3, false, false, "left", "center", false)
	for _, row := range [][2]string{{"Customer", hd.Customer}, {"Model Name", hd.Models}, {"Part Name", hd.PartNames}, {"Part Number", hd.PartNumbers}, {"Process or Line", hd.ProcessLine}} {
		h.height(r, 7)
		h.put(1, 2, r, r, row[0], label)
		h.put(3, 3, r, r, ":", h.style(1.4, true, false, "center", "center", false))
		h.put(4, last, r, r, row[1], value)
		r++
	}
	r++
	title := h.style(1.1, true, false, "center", "center", false)
	th := h.style(1, true, true, "center", "center", true)
	cellC := h.style(1, false, true, "center", "center", true)
	cellL := h.style(1, false, true, "left", "center", true)
	h.put(0, last, r, r, "Document Revision History", title)
	r++
	h.put(0, 0, r, r, "No", th)
	h.put(1, 2, r, r, "Date", th)
	h.put(3, 3, r, r, "Revision", th)
	h.put(4, last, r, r, "Description of Change", th)
	r++
	for _, x := range sh.History {
		h.put(0, 0, r, r, strconv.Itoa(x.No), cellC)
		h.put(1, 2, r, r, x.Date, cellC)
		h.put(3, 3, r, r, x.Revision, cellC)
		h.put(4, last, r, r, x.Description, cellL)
		_ = f.SetRowHeight(name, r, rowHeight([]string{"", "", "", x.Description}, []float64{6, 16, 16, 100}, 10))
		r++
	}
	r++
	if len(sh.Slots) > 0 {
		h.put(0, last, r, r, "INHOUSE APPROVAL", title)
		r++
		// Slot dibagi rata ke kolom logis 0 sampai last.
		per := float64(last+1) / float64(len(sh.Slots))
		for i, s := range sh.Slots {
			a := int(math.Round(per * float64(i)))
			b := int(math.Round(per*float64(i+1))) - 1
			h.put(a, b, r, r, stageLabel(s.Stage), th)
			h.put(a, b, r+1, r+1, s.Signed, cellC)
			h.put(a, b, r+2, r+2, s.Name(), cellC)
			h.put(a, b, r+3, r+3, s.Title, th)
		}
		h.height(r+1, 18)
		r += 5
	}
	h.put(0, last, r, r, "CUSTOMER APPROVAL", th)
	h.put(0, last, r+1, r+1, "", cellC)
	h.height(r+1, 36)
	r += 3
	if sh.ReleaseNote != "" {
		h.put(0, last, r, r, sh.ReleaseNote, h.style(0.8, false, false, "center", "center", true))
		r++
	}
	h.put(0, last, r, r, hd.Customer, title)
	return pageSetup(f, name, cover, "", len(widths), r)
}
