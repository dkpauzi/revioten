package printout

// Gambar kolom Process Flow untuk PDF (C3, D-122, G11): satu SVG per halaman,
// dalam milimeter, dari rencana halaman. Ukurannya sama dengan pratinjau
// (web/src/lib/documents/PreviewSheet.svelte) dan shape Excel (xlsx.go).
//
// Bentuknya mengikuti CP SC-TR-020-04 (bagian B, E-144): tujuan NG berupa
// kotak di kiri alur (SCRAP dan NCP berbentuk predefined process, REWORK
// kotak proses biasa) yang disambung garis siku dengan kepala panah; label OK
// hanya di bawah belah ketupat; garis alur berpanah dari simbol ke simbol.

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
	ngBoxWMM   = 16.0
	ngBoxHMM   = 5.5
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

// predefined: SCRAP dan NCP digambar sebagai predefined process; REWORK
// sebagai kotak proses biasa (legenda PFD SH-TR-020-02).
func predefined(ng string) bool { return ng != "rework" }

// flowX memberi pusat garis alur di dalam kolom selebar width. Alur sedikit
// ke kanan, supaya kotak NG muat di kirinya.
func flowX(width float64) float64 {
	return math.Max(width*0.6, shapeWMM/2+1)
}

// arrowDefs adalah penanda kepala panah untuk garis alur.
const arrowDefs = `<defs><marker id="arrow" viewBox="0 0 6 6" refX="6" refY="3" markerUnits="userSpaceOnUse" markerWidth="1.8" markerHeight="1.8" orient="auto"><path d="M0,0 L6,3 L0,6 z" fill="#000"/></marker></defs>`

const strokeStyle = `fill="#fff" stroke="#000" stroke-width="0.3"`

// arrow menggambar garis berkepala panah di ujungnya.
func arrow(b *strings.Builder, x1, y1, x2, y2 float64) {
	fmt.Fprintf(b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3" marker-end="url(#arrow)"/>`, x1, y1, x2, y2)
}

// ngBox menggambar kotak tujuan NG dengan tulisannya. font dalam milimeter.
func ngBox(b *strings.Builder, x, y, w, h float64, ng string, font float64) {
	fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" %s data-ng-box="%s"/>`, x, y, w, h, strokeStyle, html.EscapeString(ng))
	if predefined(ng) {
		in := math.Min(1.4, w/8)
		fmt.Fprintf(b, `<line class="pp" x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3"/>`, x+in, y, x+in, y+h)
		fmt.Fprintf(b, `<line class="pp" x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3"/>`, x+w-in, y, x+w-in, y+h)
	}
	fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%.2f" font-weight="bold" text-anchor="middle" dominant-baseline="central">%s</text>`, x+w/2, y+h/2, font, html.EscapeString(ngLabel(ng)))
}

// symbolShape menggambar bentuk satu simbol berpusat di (cx, y) dengan lebar
// w dan tinggi h. Belah ketupat memakai ukuran itu sebagai ukuran luarnya.
func symbolShape(b *strings.Builder, symbol string, cx, y, w, h float64) {
	l, r, t, bt := cx-w/2, cx+w/2, y-h/2, y+h/2
	attr := fmt.Sprintf(`%s data-symbol="%s"`, strokeStyle, html.EscapeString(symbol))
	switch symbol {
	case "decision":
		fmt.Fprintf(b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f L %.2f %.2f Z" %s/>`, cx, t, r, y, cx, bt, l, y, attr)
	case "merge":
		fmt.Fprintf(b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f Z" %s/>`, l, t, r, t, cx, bt, attr)
	case "connector":
		fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" %s/>`, cx, y, h/2, attr)
	case "alternate_process":
		// Segi enam (flowChartPreparation), seperti legenda PFD referensi.
		in := math.Min(2, w/6)
		fmt.Fprintf(b, `<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" %s/>`,
			l+in, t, r-in, t, r, y, r-in, bt, l+in, bt, l, y, attr)
	case "document":
		fmt.Fprintf(b, `<path d="M %.2f %.2f L %.2f %.2f L %.2f %.2f Q %.2f %.2f %.2f %.2f T %.2f %.2f Z" %s/>`, l, t, r, t, r, bt-1, cx+w/4, bt-2.6, cx, bt-1, l, bt-1, attr)
	default:
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" %s/>`, l, t, w, h, attr)
	}
}

// flowSize memberi ukuran simbol di kolom Process Flow CP.
func flowSize(symbol string) (float64, float64) {
	if symbol == "decision" {
		return shapeWMM + 2, shapeHMM + 1.6
	}
	return shapeWMM, shapeHMM
}

// ngBranch menggambar cabang NG: kotak tujuan di kiri dan sedikit di atas
// simbol, disambung garis siku (mendatar, lalu naik) berkepala panah, label
// NG di bawah garis, dan label OK di bawah belah ketupat.
func ngBranch(b *strings.Builder, n FlowNode, cx, y, left float64) {
	w, h := flowSize(n.Symbol)
	from := cx - w/2
	boxW := math.Min(ngBoxWMM, from-2-left)
	if boxW < 6 {
		boxW = 6
	}
	boxX := math.Max(left, from-2-boxW)
	boxY := math.Max(0.4, y-ngBoxHMM-0.8)
	ngBox(b, boxX, boxY, boxW, ngBoxHMM, n.NG, 2.2)
	mid := boxX + boxW/2
	if boxY+ngBoxHMM < y-0.5 {
		fmt.Fprintf(b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3"/>`, from, y, mid, y)
		arrow(b, mid, y, mid, boxY+ngBoxHMM)
	} else {
		arrow(b, from, y, boxX+boxW, y)
	}
	fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="2.2" text-anchor="middle">NG</text>`, (mid+from)/2+1, y+2.6)
	if n.Symbol == "decision" {
		fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="2.2">OK</text>`, cx+0.8, y+h/2+2.6)
	}
}

// FlowSVG menggambar alur satu halaman. tops: posisi atas setiap baris halaman
// itu dari atas area baris; height: tinggi area baris; width: lebar kolom.
func FlowSVG(page PlanPage, tops []float64, height, width float64) string {
	var b strings.Builder
	cx := flowX(width)
	fmt.Fprintf(&b, `<svg class="flow" xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, width, height, width, height)
	b.WriteString(arrowDefs)
	ys := make([]float64, len(page.Nodes))
	for i, n := range page.Nodes {
		ys[i] = tops[n.Row] + nodeTopMM
	}
	connector := func(y float64, letter string) {
		fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#fff" stroke="#000" stroke-width="0.3" data-connector="%s"/>`, cx, y, connRMM, html.EscapeString(letter))
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.8" font-weight="bold" text-anchor="middle" dominant-baseline="central">%s</text>`, cx, y, html.EscapeString(letter))
	}
	// Garis berpanah: dari konektor atas atau simbol sebelumnya ke simbol berikutnya.
	prev := math.NaN()
	if page.In != "" {
		prev = connEdgeMM + connRMM
	}
	for i, n := range page.Nodes {
		_, h := flowSize(n.Symbol)
		if !math.IsNaN(prev) {
			arrow(&b, cx, prev, cx, ys[i]-h/2)
		}
		prev = ys[i] + h/2
	}
	if page.Out != "" {
		if math.IsNaN(prev) {
			prev = 0
		}
		arrow(&b, cx, prev, cx, height-connEdgeMM-connRMM)
	}
	if page.In != "" {
		connector(connEdgeMM, page.In)
	}
	for i, n := range page.Nodes {
		y := ys[i]
		if n.NG != "" {
			ngBranch(&b, n, cx, y, 0.6)
		}
		w, h := flowSize(n.Symbol)
		symbolShape(&b, n.Symbol, cx, y, w, h)
		fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="2.6" font-weight="bold" text-anchor="middle" dominant-baseline="central" data-step="%s">%s</text>`, cx, y, html.EscapeString(n.Op), html.EscapeString(n.Op))
	}
	if page.Out != "" {
		connector(height-connEdgeMM, page.Out)
	}
	b.WriteString(`</svg>`)
	return b.String()
}
