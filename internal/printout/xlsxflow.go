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

// excelColPx: lebar kolom Excel (nilai width di berkas) dalam piksel layar,
// dengan rumus Excel untuk huruf bawaan Calibri 11 (lebar angka 7 piksel).
// excelize sendiri menghitung 8 piksel per huruf, jadi posisi shape tidak
// memakai hitungan excelize (lihat xlsxdraw.go).
func excelColPx(width float64) float64 { return math.Floor((math.Floor(256*width) + 18) / 256 * 7) }

// excelRowPx: tinggi baris Excel (point) dalam piksel layar.
func excelRowPx(pt float64) float64 { return math.Round(pt * 96 / 72) }

// mmPt: milimeter ke point.
const mmPt = 72 / 25.4

// xlsxPage menyimpan baris Excel setiap baris isi.
type xlsxLayout struct {
	plan     Plan
	rowPt    []float64 // tinggi baris Excel (pt) setiap baris Sheet.Rows
	excelRow []int     // nomor baris Excel setiap baris Sheet.Rows
	offset   int       // 1 bila kolom tanda revisi dicetak
	flowPx   float64   // lebar kolom alur (piksel)
	flowLeft float64   // jarak kolom alur dari kiri lembar (piksel)
	rowY     []float64 // jarak atas setiap baris Excel dari atas lembar (piksel); indeks = nomor baris
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

// flowShapes menggambar alur satu halaman. Excel memakai jarak vertikal
// hanya di dalam sel jangkarnya, jadi setiap shape dijangkarkan di baris
// tempat ia berada, dengan jarak di dalam baris itu. Bentuknya sama dengan
// FlowSVG: kotak NG di kiri, garis siku berpanah, OK hanya di belah ketupat.
func flowShapes(f *excelize.File, name string, sh Sheet, x xlsxLayout, pg PlanPage) error {
	scale := x.flowPx / (x.plan.ColumnsMM[sh.FlowColumn] * 96 / 25.4) // piksel lembar per piksel kertas
	mm := func(v float64) float64 { return v * 96 / 25.4 * scale }
	tops := make([]float64, len(pg.Rows))
	y := 0.0
	for i, r := range pg.Rows {
		tops[i] = y
		y += px(x.rowPt[r])
	}
	height := y
	pageTop := x.rowY[x.excelRow[pg.Rows[0]]]
	d := &drawer{f: f, name: name, abs: func(xx, yy float64) (float64, float64) { return x.flowLeft + xx, pageTop + yy }}
	widthMM := x.plan.ColumnsMM[sh.FlowColumn]
	cx := mm(flowX(widthMM))
	ys := make([]float64, len(pg.Nodes))
	for i, n := range pg.Nodes {
		ys[i] = tops[n.Row] + mm(nodeTopMM)
	}
	r := mm(connRMM)
	prev := math.NaN()
	if pg.In != "" {
		prev = mm(connEdgeMM) + r
	}
	for i, n := range pg.Nodes {
		_, h := flowSize(n.Symbol)
		if !math.IsNaN(prev) {
			if err := d.line(cx, prev, cx, ys[i]-mm(h)/2, true); err != nil {
				return err
			}
		}
		prev = ys[i] + mm(h)/2
	}
	if pg.Out != "" {
		if math.IsNaN(prev) {
			prev = 0
		}
		if err := d.line(cx, prev, cx, height-mm(connEdgeMM)-r, true); err != nil {
			return err
		}
	}
	if pg.In != "" {
		if err := d.shape("ellipse", cx-r, mm(connEdgeMM)-r, 2*r, 2*r, []string{pg.In}, 7); err != nil {
			return err
		}
	}
	for i, n := range pg.Nodes {
		yy := ys[i]
		w, h := flowSize(n.Symbol)
		if n.NG != "" {
			from := cx - mm(w)/2
			boxW := math.Max(mm(6), math.Min(mm(ngBoxWMM), from-mm(2.6)))
			boxX := math.Max(mm(0.6), from-mm(2)-boxW)
			boxH := mm(ngBoxHMM)
			boxY := math.Max(mm(0.4), yy-boxH-mm(0.8))
			if err := d.ngBox(boxX, boxY, boxW, boxH, n.NG, 6); err != nil {
				return err
			}
			mid := boxX + boxW/2
			if boxY+boxH < yy-mm(0.5) {
				if err := d.line(mid, yy, from, yy, false); err != nil {
					return err
				}
				if err := d.line(mid, yy, mid, boxY+boxH, true); err != nil {
					return err
				}
			} else if err := d.line(from, yy, boxX+boxW, yy, true); err != nil {
				return err
			}
			if err := d.text(mid, yy+mm(0.4), from-mid, mm(3), "NG", 6); err != nil {
				return err
			}
			if n.Symbol == "decision" {
				if err := d.text(cx+mm(0.5), yy+mm(h)/2, mm(5), mm(3), "OK", 6); err != nil {
					return err
				}
			}
		}
		if err := d.shape(excelShape(n.Symbol), cx-mm(w)/2, yy-mm(h)/2, mm(w), mm(h), []string{n.Op}, 7); err != nil {
			return err
		}
	}
	if pg.Out != "" {
		if err := d.shape("ellipse", cx-r, height-mm(connEdgeMM)-r, 2*r, 2*r, []string{pg.Out}, 7); err != nil {
			return err
		}
	}
	return nil
}
