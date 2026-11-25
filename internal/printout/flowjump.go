package printout

// Lompatan di Process Flow (D-227, E-147). Langkah PFD bisa menuju langkah
// lain selain langkah berikutnya: maju (OK dari AOI langsung ke ICT, melewati
// VMI) atau mundur (No di Finish 2nd side kembali ke Loading PCB). Field PFD-nya
// Jump to (nomor langkah) dan Jump label (ok, ng, yes, no).
//
// Kiri garis alur untuk kotak NG, kanan untuk lompatan. Garis lompatan
// bergabung ke garis alur sedikit di atas simbol tujuan, supaya sudut kanan
// simbol tujuan tetap bebas untuk lompatannya sendiri. Bentuk gambarnya
// dihitung sekali sebagai daftar bagian (jumpPart), lalu digambar ke SVG
// (pratinjau, PDF) dan ke shape Excel dengan ukuran yang sama.

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// Jarak lompatan dari pusat garis alur Control Plan, dalam milimeter.
const (
	jumpLaneMM = 9.5  // garis tegak lompatan, di kanan ujung belah ketupat (7 mm)
	jumpConnMM = 11.6 // pusat konektor lompatan
	jumpJoinMM = 1.5  // garis lompatan bergabung sejauh ini di atas simbol tujuan
	jumpFontMM = 2.2
)

// jumpText memberi tulisan label lompatan dari kunci pilihannya.
func jumpText(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "ok":
		return "OK"
	case "ng":
		return "NG"
	case "yes":
		return "Yes"
	case "no":
		return "No"
	}
	return ""
}

// oppositeJump memberi kunci label garis lurus ke bawah: lawan label lompatan.
func oppositeJump(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "ok":
		return "ng"
	case "ng":
		return "ok"
	case "yes":
		return "no"
	case "no":
		return "yes"
	}
	return ""
}

// Jenis peringatan rencana halaman.
const (
	WarnJumpUnknown     = "jump_unknown"      // nomor tujuan tidak ada di PFD (di bagian yang sama)
	WarnJumpSelf        = "jump_self"         // langkah menuju dirinya sendiri
	WarnJumpNotDecision = "jump_not_decision" // lompatan di simbol selain Decision
	WarnFlowNarrow      = "flow_narrow"       // kolom Process Flow lebih sempit dari minFlowWidthMM
)

// PlanWarning adalah satu hal yang perlu diperiksa penyusun sebelum mencetak.
type PlanWarning struct {
	Kind      string
	Operation string  // langkah yang punya lompatan
	Target    string  // nomor tujuan lompatan
	WidthMM   float64 // flow_narrow: lebar kolom sekarang
}

// sectionsOf memberi nomor bagian per PCB setiap baris (D-155): naik satu di
// setiap baris judul bagian.
func sectionsOf(sh Sheet) []int {
	sec := make([]int, len(sh.Rows))
	n := 0
	for i, r := range sh.Rows {
		if r.Section != "" {
			n++
		}
		sec[i] = n
	}
	return sec
}

// jumpWarnings memeriksa lompatan setiap langkah: tujuan harus ada di bagian
// yang sama, bukan langkah itu sendiri, dan simbolnya Decision.
func jumpWarnings(ops []string, sec []int, steps map[string]Step) []PlanWarning {
	var flowing []opGroup
	for _, g := range opGroups(ops) {
		if g.op != "" {
			flowing = append(flowing, g)
		}
	}
	section := func(row int) int {
		if row < len(sec) {
			return sec[row]
		}
		return 0
	}
	var out []PlanWarning
	seen := map[string]bool{}
	for k, g := range flowing {
		step := steps[g.op]
		target := strings.TrimSpace(step.Jump)
		if target == "" || seen[g.op] {
			continue
		}
		seen[g.op] = true
		w := PlanWarning{Operation: g.op, Target: target}
		switch {
		case target == g.op:
			w.Kind = WarnJumpSelf
		case jumpTarget(flowing, k, target, section) < 0:
			w.Kind = WarnJumpUnknown
		case step.Symbol != "decision":
			w.Kind = WarnJumpNotDecision
		default:
			continue
		}
		out = append(out, w)
	}
	return out
}

// jumpPart adalah satu bagian gambar lompatan. Kind: "line" (garis), "arrow"
// (garis berkepala panah di ujung kedua), "conn" (konektor berpusat di X1,
// Y1), "label" (tulisan berpusat di X1, Y1).
type jumpPart struct {
	Kind           string
	X1, Y1, X2, Y2 float64
	Text           string
}

// cpJumpParts menghitung gambar lompatan satu halaman Control Plan. ys: pusat
// setiap simbol, cx: pusat garis alur, u: mengubah milimeter ke satuan ys
// (identitas untuk SVG, piksel untuk Excel).
func cpJumpParts(page PlanPage, ys []float64, cx float64, u func(float64) float64) []jumpPart {
	var out []jumpPart
	half := func(i int) (float64, float64) {
		w, h := flowSize(page.Nodes[i].Symbol)
		return u(w) / 2, u(h) / 2
	}
	// joinY: tempat garis lompatan bergabung ke garis alur di atas simbol i.
	joinY := func(i int) float64 {
		_, hh := half(i)
		top := ys[i] - hh
		above := 0.0
		switch {
		case i > 0:
			_, ph := half(i - 1)
			above = ys[i-1] + ph
		case page.In != "":
			above = u(connEdgeMM + connRMM)
		}
		return top - math.Min(u(jumpJoinMM), (top-above)/2)
	}
	r := u(connRMM)
	for _, l := range page.Lines {
		wh, _ := half(l.From)
		y, lane, jy := ys[l.From], cx+u(jumpLaneMM), joinY(l.To)
		out = append(out,
			jumpPart{Kind: "line", X1: cx + wh, Y1: y, X2: lane, Y2: y},
			jumpPart{Kind: "line", X1: lane, Y1: y, X2: lane, Y2: jy},
			jumpPart{Kind: "arrow", X1: lane, Y1: jy, X2: cx, Y2: jy})
		if n := page.Nodes[l.From]; n.Jump != "" {
			out = append(out, jumpPart{Kind: "label", X1: lane, Y1: y - u(1.6), Text: n.Jump})
		}
	}
	for i, n := range page.Nodes {
		wh, _ := half(i)
		x := cx + u(jumpConnMM)
		if n.JumpOut != "" {
			out = append(out, jumpPart{Kind: "arrow", X1: cx + wh, Y1: ys[i], X2: x - r, Y2: ys[i]},
				jumpPart{Kind: "conn", X1: x, Y1: ys[i], Text: n.JumpOut})
			if n.Jump != "" {
				out = append(out, jumpPart{Kind: "label", X1: x, Y1: ys[i] - r - u(1.4), Text: n.Jump})
			}
		}
		for k, letter := range n.JumpIn {
			// Beberapa konektor masuk ke satu simbol ditumpuk ke atas.
			jy := math.Max(r+u(0.3), joinY(i)) - float64(k)*(2*r+u(0.6))
			out = append(out, jumpPart{Kind: "conn", X1: x, Y1: jy, Text: letter},
				jumpPart{Kind: "arrow", X1: x - r, Y1: jy, X2: cx, Y2: jy})
		}
	}
	return out
}

// partsSVG menggambar bagian lompatan ke SVG. r: jari-jari konektor, font
// tinggi huruf label (mm).
func partsSVG(b *strings.Builder, parts []jumpPart, r, font float64) {
	for _, p := range parts {
		switch p.Kind {
		case "line":
			fmt.Fprintf(b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3" data-jump="line"/>`, p.X1, p.Y1, p.X2, p.Y2)
		case "arrow":
			fmt.Fprintf(b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#000" stroke-width="0.3" marker-end="url(#arrow)" data-jump="arrow"/>`, p.X1, p.Y1, p.X2, p.Y2)
		case "conn":
			fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="#fff" stroke="#000" stroke-width="0.3" data-connector="%s" data-jump="conn"/>`, p.X1, p.Y1, r, html.EscapeString(p.Text))
			fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%.2f" font-weight="bold" text-anchor="middle" dominant-baseline="central">%s</text>`, p.X1, p.Y1, r*1.08, html.EscapeString(p.Text))
		case "label":
			fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%.2f" text-anchor="middle" dominant-baseline="central" data-jump-label="%s">%s</text>`, p.X1, p.Y1, font, html.EscapeString(p.Text), html.EscapeString(p.Text))
		}
	}
}

// partsXLSX menggambar bagian lompatan sebagai shape Excel. fx dan fy
// mengubah koordinat bagian ke piksel lembar; r jari-jari konektor dalam
// satuan bagian; lw dan lh ukuran kotak label dalam satuan bagian.
func partsXLSX(d *drawer, parts []jumpPart, fx, fy func(float64) float64, r, lw, lh, size float64) error {
	for _, p := range parts {
		var err error
		switch p.Kind {
		case "line", "arrow":
			err = d.line(fx(p.X1), fy(p.Y1), fx(p.X2), fy(p.Y2), p.Kind == "arrow")
		case "conn":
			err = d.shape("ellipse", fx(p.X1-r), fy(p.Y1-r), fx(p.X1+r)-fx(p.X1-r), fy(p.Y1+r)-fy(p.Y1-r), []string{p.Text}, 7)
		case "label":
			err = d.text(fx(p.X1-lw/2), fy(p.Y1-lh/2), fx(p.X1+lw/2)-fx(p.X1-lw/2), fy(p.Y1+lh/2)-fy(p.Y1-lh/2), p.Text, size)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
