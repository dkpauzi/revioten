package printout

// Cetakan PFD sebagai halaman diagram (bagian B, E-144), mengikuti
// SH-TR-020-02: kepala form, legenda simbol, lalu alur yang turun dalam
// kolom. Setiap langkah berupa kotak "nomor. nama proses"; inspeksi berupa
// belah ketupat yang memuat kotak itu; NG menuju kotak SCRAP, NCP, atau
// REWORK di kiri. Bila satu kolom penuh, alur berlanjut ke kolom berikutnya,
// lalu ke halaman berikutnya, lewat konektor A, B, C dengan huruf yang sama
// di ujung dan pangkalnya. Tabel PFD tetap dipakai untuk mengisi di layar.
//
// Semua ukuran dalam milimeter, dari kiri atas area diagram. Pratinjau, PDF
// (SVG), dan Excel (shape) memakai tata letak yang sama (DiagramLayout).

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// DiagramStep adalah satu langkah PFD di diagram.
type DiagramStep struct {
	Op     string
	Name   string
	Symbol string
	NG     string
}

// Label menulis isi kotak: "nomor. nama", seperti PFD referensi.
func (s DiagramStep) Label() string {
	switch {
	case s.Op == "":
		return s.Name
	case s.Name == "":
		return s.Op
	}
	return s.Op + ". " + s.Name
}

// DiagramNode adalah satu simbol: pusat (X, Y) dan ukuran luarnya.
type DiagramNode struct {
	Step DiagramStep
	X, Y float64
	W, H float64
}

// DiagramColumn adalah satu kolom alur: pusat garis alur, simbolnya, dan
// konektor masuk (In) di atas dan keluar (Out) di bawah.
type DiagramColumn struct {
	X       float64
	Nodes   []DiagramNode
	In, Out string
}

// DiagramPage adalah satu halaman diagram.
type DiagramPage struct {
	Columns []DiagramColumn
}

// Ukuran diagram.
const (
	diagColumns   = 2
	diagBoxW      = 34.0
	diagBoxH      = 11.0
	diagDiamondW  = 46.0
	diagDiamondH  = 19.0
	diagInnerW    = 28.0 // kotak di dalam belah ketupat
	diagInnerH    = 8.5
	diagGap       = 7.0 // jarak antar simbol, berisi panah
	diagConnR     = 3.0
	diagConnSpace = 10.0 // ruang konektor di atas atau bawah kolom
	diagNGW       = 19.0
	diagNGH       = 8.0
	diagFontMM    = 2.5 // tinggi huruf isi kotak (sekitar 7 pt)
	diagLegendMM  = 26.0
)

// diagSize memberi ukuran luar satu simbol.
func diagSize(s DiagramStep) (float64, float64) {
	switch s.Symbol {
	case "decision":
		return diagDiamondW, diagDiamondH
	case "connector":
		return 12, 12
	case "merge":
		return 16, 10
	}
	return diagBoxW, diagBoxH
}

// DiagramLayout menempatkan langkah ke kolom dan halaman. width dan height
// adalah ukuran area diagram satu halaman.
func DiagramLayout(steps []DiagramStep, width, height float64) []DiagramPage {
	colW := width / diagColumns
	var pages []DiagramPage
	page := DiagramPage{}
	index := 0
	col := DiagramColumn{X: colW * 0.6}
	y := 2.0
	letters := 0
	for _, s := range steps {
		w, h := diagSize(s)
		if len(col.Nodes) > 0 && y+h+diagConnSpace > height {
			letter := ConnectorLetter(letters)
			letters++
			col.Out = letter
			page.Columns = append(page.Columns, col)
			index++
			if index == diagColumns {
				pages = append(pages, page)
				page = DiagramPage{}
				index = 0
			}
			col = DiagramColumn{X: colW*float64(index) + colW*0.6, In: letter}
			y = diagConnSpace
		}
		col.Nodes = append(col.Nodes, DiagramNode{Step: s, X: col.X, Y: y + h/2, W: w, H: h})
		y += h + diagGap
	}
	if len(col.Nodes) > 0 || col.In != "" {
		page.Columns = append(page.Columns, col)
	}
	return append(pages, page)
}

// wrapText membungkus teks per kata menjadi baris paling banyak max huruf.
// Kata yang lebih panjang dari max dibiarkan utuh di barisnya sendiri.
func wrapText(text string, max int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= max:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// boxChars: jumlah huruf per baris di dalam kotak selebar w.
func boxChars(w float64) int {
	return max(4, int((w-2)/(diagFontMM*0.55)))
}

// textLines menulis teks berbaris di tengah (cx, cy).
func textLines(b *strings.Builder, lines []string, cx, cy, font float64, attrs string) {
	lh := font * 1.15
	top := cy - lh*float64(len(lines)-1)/2
	fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%.2f" font-weight="bold" text-anchor="middle" dominant-baseline="central"%s>`, cx, top, font, attrs)
	for i, l := range lines {
		fmt.Fprintf(b, `<tspan x="%.2f" y="%.2f">%s</tspan>`, cx, top+lh*float64(i), html.EscapeString(l))
	}
	b.WriteString(`</text>`)
}

// connectorCircle menggambar konektor bertulisan huruf.
func connectorCircle(b *strings.Builder, cx, cy, r float64, letter string) {
	fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#fff" stroke="#000" stroke-width="0.3" data-connector="%s"/>`, cx, cy, r, html.EscapeString(letter))
	fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="3" font-weight="bold" text-anchor="middle" dominant-baseline="central">%s</text>`, cx, cy, html.EscapeString(letter))
}

// diagramNode menggambar satu langkah dengan cabang NG-nya.
func diagramNode(b *strings.Builder, n DiagramNode, colLeft float64) {
	s := n.Step
	if s.NG != "" {
		from := n.X - n.W/2
		boxX := math.Max(colLeft+1, from-4-diagNGW)
		boxW := from - 4 - boxX
		ngBox(b, boxX, n.Y-diagNGH/2, boxW, diagNGH, s.NG, 2.6)
		arrow(b, from, n.Y, boxX+boxW, n.Y)
		fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="2.4" text-anchor="middle">NG</text>`, (from+boxX+boxW)/2, n.Y-1)
		if s.Symbol == "decision" {
			fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="2.4">OK</text>`, n.X+1.2, n.Y+n.H/2+3)
		}
	}
	fmt.Fprintf(b, `<g data-step="%s">`, html.EscapeString(s.Op))
	symbolShape(b, s.Symbol, n.X, n.Y, n.W, n.H)
	textW := n.W
	if s.Symbol == "decision" {
		// Inspeksi: kotak proses di dalam belah ketupat, seperti SH-TR-020-02.
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" %s/>`, n.X-diagInnerW/2, n.Y-diagInnerH/2, diagInnerW, diagInnerH, strokeStyle)
		textW = diagInnerW
	}
	if s.Symbol == "merge" || s.Symbol == "connector" {
		textLines(b, []string{s.Op}, n.X, n.Y-1, diagFontMM, "")
	} else {
		lines := wrapText(s.Label(), boxChars(textW))
		textLines(b, lines, n.X, n.Y, diagFontMM, "")
	}
	b.WriteString(`</g>`)
}

// DiagramSVG menggambar satu halaman diagram selebar width dan setinggi height.
func DiagramSVG(page DiagramPage, width, height float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="diagram" xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, width, height, width, height)
	b.WriteString(arrowDefs)
	colW := width / diagColumns
	for ci, c := range page.Columns {
		left := colW * float64(ci)
		prev := math.NaN()
		if c.In != "" {
			connectorCircle(&b, c.X, diagConnR+1, diagConnR, c.In)
			prev = 2*diagConnR + 1
		}
		for _, n := range c.Nodes {
			if !math.IsNaN(prev) {
				arrow(&b, c.X, prev, c.X, n.Y-n.H/2)
			}
			diagramNode(&b, n, left)
			prev = n.Y + n.H/2
		}
		if c.Out != "" {
			if math.IsNaN(prev) {
				prev = 0
			}
			bottom := height - diagConnR - 1
			arrow(&b, c.X, prev, c.X, bottom-diagConnR)
			connectorCircle(&b, c.X, bottom, diagConnR, c.Out)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// legendItems: simbol dan nama di legenda, urutan SH-TR-020-02.
var legendItems = []struct{ symbol, ng, label string }{
	{"process", "", "PROCESS"}, {"alternate_process", "", "ALTERNATE PROCESS"}, {"decision", "", "DECISION"},
	{"connector", "", "CONNECTOR"}, {"merge", "", "MERGE"}, {"document", "", "DOCUMENT"},
	{"", "rework", "REWORK"}, {"", "scrap", "SCRAP"}, {"", "ncp", "NCP"},
}

// DiagramLegendSVG menggambar kotak legenda selebar width: dua baris simbol
// dengan namanya.
func DiagramLegendSVG(width float64) string {
	var b strings.Builder
	h := diagLegendMM - 2
	fmt.Fprintf(&b, `<svg class="legend" xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, width, h, width, h)
	fmt.Fprintf(&b, `<rect x="0.2" y="0.2" width="%.2f" height="%.2f" fill="none" stroke="#000" stroke-width="0.3"/>`, width-0.4, h-0.4)
	b.WriteString(`<text x="2" y="3.6" font-size="2.8" font-weight="bold">Legend</text>`)
	per := 5
	slot := (width - 4) / float64(per)
	for i, it := range legendItems {
		row, col := i/per, i%per
		cx := 2 + slot*float64(col) + 6.5
		cy := 8.5 + float64(row)*9
		if it.ng != "" {
			ngBox(&b, cx-6, cy-2.5, 12, 5, it.ng, 1.9)
			// Nama sudah tertulis di dalam kotak; label di samping tetap ditulis.
		} else {
			w, sh := 12.0, 5.0
			if it.symbol == "decision" {
				w, sh = 12, 6
			}
			if it.symbol == "connector" {
				w, sh = 5, 5
			}
			symbolShape(&b, it.symbol, cx, cy, w, sh)
		}
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="1.9" dominant-baseline="central">%s</text>`, cx+7, cy, html.EscapeString(it.label))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// diagramSteps menyusun langkah diagram dari model cetak PFD: baris pertama
// setiap nomor langkah, dengan simbol dan arah NG dari Steps.
func diagramSteps(sh Sheet) []DiagramStep {
	var out []DiagramStep
	for _, g := range opGroups(sh.Ops) {
		if g.op == "" {
			continue
		}
		step := sh.Steps[g.op]
		symbol := step.Symbol
		if symbol == "" {
			symbol = "process"
		}
		name := ""
		if g.start < len(sh.Processes) {
			name = sh.Processes[g.start]
		}
		out = append(out, DiagramStep{Op: g.op, Name: name, Symbol: symbol, NG: step.NG})
	}
	return out
}

// isDiagram: PFD dicetak sebagai diagram.
func isDiagram(sh Sheet) bool { return sh.Snapshot.DocumentType == "pfd" }
