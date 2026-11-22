package printout

// Diagram PFD di Excel (bagian B). Kepala form ditulis sebagai sel dan
// berulang di setiap halaman; legenda dan alur setiap halaman digambar
// sebagai shape di satu blok baris, dengan pemisah halaman di awal blok.
// Lembar dibagi menjadi kisi kolom dan baris yang sama besar, supaya posisi
// milimeter dari DiagramLayout mudah diubah menjadi sel dan jarak.

import (
	"fmt"
	"math"

	"github.com/xuri/excelize/v2"
)

const (
	gridCols   = 40   // jumlah kolom kisi
	gridColW   = 3.0  // lebar kolom (karakter)
	gridRowPt  = 15.0 // tinggi baris kisi (point)
	gridRowPx  = gridRowPt * 96 / 72
	diagFontPt = 7.0
)

// writeDiagram menulis sheet PFD sebagai diagram.
func writeDiagram(f *excelize.File, name string, sh Sheet, st styles) error {
	for i := 1; i <= gridCols; i++ {
		if err := f.SetColWidth(name, colName(i), colName(i), gridColW); err != nil {
			return err
		}
	}
	if err := f.SetSheetProps(name, &excelize.SheetPropsOptions{CustomHeight: boolPtr(true), DefaultRowHeight: floatPtr(gridRowPt)}); err != nil {
		return err
	}
	grid := make([]float64, gridCols)
	for i := range grid {
		grid[i] = gridColW
	}
	head, ok := writeFormHead(f, name, sh, grid)
	if !ok {
		head = writeHead(f, name, sh, st, gridCols, 3)
	}
	titleRows := fmt.Sprintf("$1:$%d", head-1)
	plan := BuildPlan(sh)

	// Skala: Excel mengecilkan lembar supaya muat selebar kertas.
	sheetPt := excelColPx(gridColW) * gridCols * 0.75
	scale := math.Min(1, plan.ContentW*mmPt/sheetPt)
	pxPerMM := mmPt / (0.75 * scale)

	row := head
	for p, page := range plan.Diagram {
		if p > 0 {
			if err := f.InsertPageBreak(name, cell(1, row)); err != nil {
				return err
			}
		}
		top := row
		tops, err := rowTops(f, name, top)
		if err != nil {
			return err
		}
		topY := tops[top]
		d := &drawer{f: f, name: name, abs: func(x, y float64) (float64, float64) { return x, topY + y }}
		mm := func(v float64) float64 { return v * pxPerMM }
		if err := diagramLegendXLSX(d, mm, plan.DiagramW); err != nil {
			return err
		}
		off := mm(plan.LegendMM)
		if err := diagramPageXLSX(d, mm, off, page, plan.DiagramW, plan.DiagramH); err != nil {
			return err
		}
		row = top + int(math.Ceil((off+mm(plan.DiagramH))/gridRowPx))
	}
	return pageSetup(f, name, sh, titleRows, gridCols, row)
}

// diagramLegendXLSX menggambar legenda di atas setiap halaman.
func diagramLegendXLSX(d *drawer, mm func(float64) float64, width float64) error {
	if err := d.text(mm(1), mm(0.5), mm(20), mm(4), "Legend", 7); err != nil {
		return err
	}
	per := 5
	slot := (width - 4) / float64(per)
	for i, it := range legendItems {
		row, col := i/per, i%per
		cx := 2 + slot*float64(col) + 6.5
		cy := 8.5 + float64(row)*9
		var err error
		if it.ng != "" {
			err = d.ngBox(mm(cx-6), mm(cy-2.5), mm(12), mm(5), it.ng, 5)
		} else {
			w, h := 12.0, 5.0
			if it.symbol == "decision" {
				h = 6
			}
			if it.symbol == "connector" {
				w, h = 5, 5
			}
			err = d.shape(excelShape(it.symbol), mm(cx-w/2), mm(cy-h/2), mm(w), mm(h), nil, 5)
		}
		if err != nil {
			return err
		}
		if err := d.text(mm(cx+6.5), mm(cy-2), mm(slot-13), mm(4), it.label, 5.5); err != nil {
			return err
		}
	}
	return nil
}

// diagramPageXLSX menggambar satu halaman diagram mulai off piksel dari atas blok.
func diagramPageXLSX(d *drawer, mm func(float64) float64, off float64, page DiagramPage, width, height float64) error {
	y := func(v float64) float64 { return off + mm(v) }
	colW := width / diagColumns
	conn := func(cx, cy float64, letter string) error {
		return d.shape("ellipse", mm(cx-diagConnR), y(cy-diagConnR), mm(2*diagConnR), mm(2*diagConnR), []string{letter}, diagFontPt)
	}
	for ci, c := range page.Columns {
		left := colW * float64(ci)
		prev := math.NaN()
		if c.In != "" {
			if err := conn(c.X, diagConnR+1, c.In); err != nil {
				return err
			}
			prev = 2*diagConnR + 1
		}
		for _, n := range c.Nodes {
			if !math.IsNaN(prev) {
				if err := d.line(mm(c.X), y(prev), mm(c.X), y(n.Y-n.H/2), true); err != nil {
					return err
				}
			}
			if err := diagramNodeXLSX(d, mm, y, n, left); err != nil {
				return err
			}
			prev = n.Y + n.H/2
		}
		if c.Out != "" {
			if math.IsNaN(prev) {
				prev = 0
			}
			bottom := height - diagConnR - 1
			if err := d.line(mm(c.X), y(prev), mm(c.X), y(bottom-diagConnR), true); err != nil {
				return err
			}
			if err := conn(c.X, bottom, c.Out); err != nil {
				return err
			}
		}
	}
	return nil
}

// diagramNodeXLSX menggambar satu langkah: simbol, kotak di dalam belah
// ketupat, teks, dan cabang NG.
func diagramNodeXLSX(d *drawer, mm func(float64) float64, y func(float64) float64, n DiagramNode, colLeft float64) error {
	s := n.Step
	if s.NG != "" {
		from := n.X - n.W/2
		boxX := math.Max(colLeft+1, from-4-diagNGW)
		boxW := from - 4 - boxX
		if err := d.ngBox(mm(boxX), y(n.Y-diagNGH/2), mm(boxW), mm(diagNGH), s.NG, diagFontPt); err != nil {
			return err
		}
		if err := d.line(mm(from), y(n.Y), mm(boxX+boxW), y(n.Y), true); err != nil {
			return err
		}
		if err := d.text(mm(boxX+boxW+0.2), y(n.Y-3.8), mm(from-boxX-boxW-0.4), mm(3.5), "NG", 5); err != nil {
			return err
		}
		if s.Symbol == "decision" {
			if err := d.text(mm(n.X+0.8), y(n.Y+n.H/2+0.5), mm(5), mm(3.5), "OK", 6); err != nil {
				return err
			}
		}
	}
	label := s.Label()
	if s.Symbol == "merge" || s.Symbol == "connector" {
		label = s.Op
	}
	if s.Symbol == "decision" {
		if err := d.shape(excelShape(s.Symbol), mm(n.X-n.W/2), y(n.Y-n.H/2), mm(n.W), mm(n.H), nil, diagFontPt); err != nil {
			return err
		}
		return d.shape("flowChartProcess", mm(n.X-diagInnerW/2), y(n.Y-diagInnerH/2), mm(diagInnerW), mm(diagInnerH), wrapText(label, boxChars(diagInnerW)), diagFontPt)
	}
	return d.shape(excelShape(s.Symbol), mm(n.X-n.W/2), y(n.Y-n.H/2), mm(n.W), mm(n.H), wrapText(label, boxChars(n.W)), diagFontPt)
}

func boolPtr(b bool) *bool        { return &b }
func floatPtr(v float64) *float64 { return &v }
