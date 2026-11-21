package printout

// Gambar kolom Process Flow untuk PDF (C3, D-122, G11): satu SVG per halaman,
// dalam milimeter, dari rencana halaman. Ukurannya sama dengan pratinjau
// (web/src/lib/documents/PreviewSheet.svelte) dan shape Excel (xlsx.go).

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// Ukuran simbol alur dalam milimeter.
const (
	shapeWMM   = 12.0
	shapeHMM   = 6.5
	nodeTopMM  = 4.5 // pusat simbol dari atas baris
	connRMM    = 2.6 // jari-jari lingkaran konektor
	connEdgeMM = 3.2 // pusat konektor dari tepi atas atau bawah area baris
)

// ngLabel memberi tulisan arah NG, seperti di CP Tokairika.
func ngLabel(ng string) string {
	switch ng {
	case "scrap":
		return "SCRAP"
	case "ncp":
		return "NCP"
	case "rework":
		return "REWORK"
	}
	return strings.ToUpper(ng)
}

// flowX memberi pusat garis alur di dalam kolom selebar width.
func flowX(width float64) float64 {
	return math.Min(width/2, shapeWMM/2+2.5)
}

// FlowSVG menggambar alur satu halaman. tops: posisi atas setiap baris halaman
// itu dari atas area baris; height: tinggi area baris; width: lebar kolom.
func FlowSVG(page PlanPage, tops []float64, height, width float64) string {
	var b strings.Builder
	cx := flowX(width)
	fmt.Fprintf(&b, `<svg class="flow" xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, width, height, width, height)
	ys := make([]float64, len(page.Nodes))
	for i, n := range page.Nodes {
		ys[i] = tops[n.Row] + nodeTopMM
	}
	top, bottom := height, 0.0
	if len(ys) > 0 {
		top, bottom = ys[0], ys[len(ys)-1]
	}
	if page.In != "" {
		top = connEdgeMM
	}
	if page.Out != "" {
		bottom = height - connEdgeMM
	}
	if bottom > top {
		fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3"/>`, cx, top, cx, bottom)
	}
	connector := func(y float64, letter string) {
		fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#fff" stroke="#000" stroke-width="0.3" data-connector="%s"/>`, cx, y, connRMM, html.EscapeString(letter))
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.8" font-weight="bold" text-anchor="middle" dominant-baseline="central">%s</text>`, cx, y, html.EscapeString(letter))
	}
	if page.In != "" {
		connector(connEdgeMM, page.In)
	}
	for i, n := range page.Nodes {
		y := ys[i]
		if n.NG != "" {
			fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3"/>`, cx+shapeWMM/2+1, y, width-0.8, y)
			fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.3">NG</text>`, cx+shapeWMM/2+1.6, y-0.8)
			fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.3" font-weight="bold" text-anchor="end">%s</text>`, width-0.8, y+2.8, html.EscapeString(ngLabel(n.NG)))
			fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.3">OK</text>`, cx+0.8, y+shapeHMM/2+2.6)
		}
		l, r, t, bt := cx-shapeWMM/2, cx+shapeWMM/2, y-shapeHMM/2, y+shapeHMM/2
		style := `fill="#fff" stroke="#000" stroke-width="0.3"`
		switch n.Symbol {
		case "decision":
			fmt.Fprintf(&b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f L %.2f %.2f Z" %s/>`, cx, t-0.8, r+1, y, cx, bt+0.8, l-1, y, style)
		case "merge":
			fmt.Fprintf(&b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f Z" %s/>`, l, t, r, t, cx, bt, style)
		case "connector":
			fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" %s/>`, cx, y, shapeHMM/2, style)
		case "alternate_process":
			fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="2" %s/>`, l, t, shapeWMM, shapeHMM, style)
		case "document":
			fmt.Fprintf(&b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f Q %.2f %.2f %.2f %.2f T %.2f %.2f Z" %s/>`, l, t, r, t, r, bt-1, cx+shapeWMM/4, bt-2.6, cx, bt-1, l, bt-1, style)
		default:
			fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" %s/>`, l, t, shapeWMM, shapeHMM, style)
		}
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.6" font-weight="bold" text-anchor="middle" dominant-baseline="central" data-step="%s">%s</text>`, cx, y, html.EscapeString(n.Op), html.EscapeString(n.Op))
	}
	if page.Out != "" {
		connector(height-connEdgeMM, page.Out)
	}
	b.WriteString(`</svg>`)
	return b.String()
}
