package printout

// Kolom Process Flow dan halaman di Excel (C3, D-122, G11). Excel memakai
// rencana halaman yang sama dengan PDF dan pratinjau: tinggi baris tetap,
// pemisah halaman manual di awal setiap halaman, sel nomor langkah dan proses
// digabung per halaman, dan alur sebagai shape (simbol, garis, konektor).
// Shape ditulis ulang setiap kali berkas dibuat, jadi tidak ada yang perlu
// digeser tangan.

import (
	"math"

	"github.com/xuri/excelize/v2"
)

// excelColPx: lebar kolom Excel (jumlah karakter) dalam piksel.
func excelColPx(chars float64) float64 { return math.Floor(chars*7 + 5) }

// mmPt: milimeter ke point.
const mmPt = 72 / 25.4

// xlsxPage menyimpan baris Excel setiap baris isi.
type xlsxLayout struct {
	plan     Plan
	rowPt    []float64 // tinggi baris Excel (pt) setiap baris Sheet.Rows
	excelRow []int     // nomor baris Excel setiap baris Sheet.Rows
	offset   int       // 1 bila kolom tanda revisi dicetak
	flowPx   float64   // lebar kolom alur (piksel)
}

// planRows menghitung tinggi baris Excel dari rencana halaman. Excel
// mengecilkan halaman supaya muat selebar kertas (FitToWidth), jadi tinggi
// baris dibagi skala itu, supaya di kertas tingginya sama dengan rencana.
// Bila kepala form Excel lebih tinggi dari rencananya, baris sedikit
// dirapatkan, supaya pemisah halaman Excel tidak bergeser.
func planRows(sh Sheet, plan Plan, widths []float64, headPt float64) []float64 {
	sheetPt := 0.0
	for _, w := range widths {
		sheetPt += excelColPx(w) * 0.75
	}
	scale := 1.0
	if sheetPt > 0 {
		scale = math.Min(1, plan.ContentW*mmPt/sheetPt)
	}
	// Ruang baris per halaman di Excel, dalam point lembar kerja: tinggi kertas
	// dikurangi margin (pageSetup menambah 6 mm atas dan bawah), lalu kepala.
	printable := (plan.PaperH - sh.Layout.MarginsMM.Top - sh.Layout.MarginsMM.Bottom - 12) * mmPt / scale
	available := printable - headPt
	planned := plan.AvailableMM * mmPt / scale
	squeeze := 1.0
	if planned > 0 && available < planned {
		squeeze = available / planned
	}
	out := make([]float64, len(plan.RowsMM))
	for i, h := range plan.RowsMM {
		out[i] = math.Min(409, h*mmPt/scale*squeeze)
	}
	return out
}

// writeFlowXLSX memasang pemisah halaman, sel gabungan, dan shape alur.
func writeFlowXLSX(f *excelize.File, name string, sh Sheet, x xlsxLayout) error {
	pages := pagesOf(sh, x.plan)
	for p, pg := range x.plan.Pages {
		if len(pg.Rows) == 0 {
			continue
		}
		if p > 0 {
			if err := f.InsertPageBreak(name, cell(1, x.excelRow[pg.Rows[0]])); err != nil {
				return err
			}
		}
		// Sel nomor langkah dan proses digabung per langkah di halaman ini.
		for i, hr := range pages[p].Rows {
			for c, hc := range hr.Cells {
				if hc.Rowspan > 1 && !hc.Skip {
					top := x.excelRow[pg.Rows[i]]
					col := c + 1 + x.offset
					if err := f.MergeCell(name, cell(col, top), cell(col, top+hc.Rowspan-1)); err != nil {
						return err
					}
				}
			}
		}
		if sh.FlowColumn < 0 {
			continue
		}
		if err := flowShapes(f, name, sh, x, pg); err != nil {
			return err
		}
	}
	return nil
}

// px: point ke piksel layar Excel (96 dpi).
func px(pt float64) float64 { return pt * 96 / 72 }

// flowShapes menggambar alur satu halaman. Semua shape dijangkarkan di sel
// kolom alur baris pertama halaman, dengan jarak dalam piksel.
func flowShapes(f *excelize.File, name string, sh Sheet, x xlsxLayout, pg PlanPage) error {
	anchor := cell(sh.FlowColumn+1+x.offset, x.excelRow[pg.Rows[0]])
	scale := x.flowPx / (x.plan.ColumnsMM[sh.FlowColumn] * 96 / 25.4) // piksel lembar per piksel kertas
	mm := func(v float64) float64 { return v * 96 / 25.4 * scale }
	tops := make([]float64, len(pg.Rows))
	y := 0.0
	for i, r := range pg.Rows {
		tops[i] = y
		y += px(x.rowPt[r])
	}
	height := y
	cx := math.Min(x.flowPx/2, mm(shapeWMM/2+2.5))
	black := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"000000"}}
	white := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFFFF"}}
	outline := excelize.LineOptions{Width: 0.75, Fill: black}
	add := func(typ string, left, top, w, h float64, text string, fill excelize.Fill) error {
		s := &excelize.Shape{Cell: anchor, Type: typ, Width: uint(math.Max(1, w)), Height: uint(math.Max(1, h)), Fill: fill, Line: outline,
			Format: excelize.GraphicOptions{OffsetX: int(math.Round(left)), OffsetY: int(math.Round(top))}}
		if text != "" {
			s.Paragraph = []excelize.RichTextRun{{Text: text, Font: &excelize.Font{Bold: true, Size: 7, Color: "000000", Family: "Arial"}}}
		}
		return f.AddShape(name, s)
	}
	line := func(x1, y1, w, h float64) error {
		s := &excelize.Shape{Cell: anchor, Type: "line", Width: uint(math.Max(1, w)), Height: uint(math.Max(1, h)), Line: outline,
			Format: excelize.GraphicOptions{OffsetX: int(math.Round(x1)), OffsetY: int(math.Round(y1))}}
		return f.AddShape(name, s)
	}
	ys := make([]float64, len(pg.Nodes))
	for i, n := range pg.Nodes {
		ys[i] = tops[n.Row] + mm(nodeTopMM)
	}
	top, bottom := height, 0.0
	if len(ys) > 0 {
		top, bottom = ys[0], ys[len(ys)-1]
	}
	if pg.In != "" {
		top = mm(connEdgeMM)
	}
	if pg.Out != "" {
		bottom = height - mm(connEdgeMM)
	}
	if bottom > top {
		if err := line(cx, top, 1, bottom-top); err != nil {
			return err
		}
	}
	r := mm(connRMM)
	if pg.In != "" {
		if err := add("ellipse", cx-r, mm(connEdgeMM)-r, 2*r, 2*r, pg.In, white); err != nil {
			return err
		}
	}
	w, h := mm(shapeWMM), mm(shapeHMM)
	for i, n := range pg.Nodes {
		yy := ys[i]
		if n.NG != "" {
			if err := line(cx+w/2, yy, x.flowPx-cx-w/2-2, 1); err != nil {
				return err
			}
			// Tulisan NG dan arahnya di sel baris itu, rata kanan.
			row := x.excelRow[pg.Rows[n.Row]]
			if err := f.SetCellValue(name, cell(sh.FlowColumn+1+x.offset, row), "NG → "+ngLabel(n.NG)); err != nil {
				return err
			}
		}
		typ := map[string]string{"decision": "flowChartDecision", "merge": "flowChartMerge", "connector": "flowChartConnector",
			"alternate_process": "flowChartAlternateProcess", "document": "flowChartDocument"}[n.Symbol]
		if typ == "" {
			typ = "flowChartProcess"
		}
		if err := add(typ, cx-w/2, yy-h/2, w, h, n.Op, white); err != nil {
			return err
		}
	}
	if pg.Out != "" {
		if err := add("ellipse", cx-r, height-mm(connEdgeMM)-r, 2*r, 2*r, pg.Out, white); err != nil {
			return err
		}
	}
	return nil
}
