package printout

// Rencana halaman (C3, D-122, G11): satu perhitungan untuk pratinjau, PDF, dan
// Excel, supaya ketiganya punya pemisah halaman, simbol Process Flow, dan
// konektor yang sama.
//
// Tinggi baris tidak bisa diukur di server, jadi diperkirakan dari panjang
// teks dan lebar kolom, lalu dipakai sebagai tinggi tetap di ketiga keluaran.
// Perkiraannya sengaja sedikit longgar, supaya teks tidak terpotong.

import (
	"math"
	"strings"
	"unicode/utf8"
)

// Step adalah satu langkah PFD: simbol alur dan, untuk inspeksi, arah NG (R-DOC-09).
type Step struct {
	Symbol string
	NG     string
}

// FlowNode adalah satu simbol di kolom Process Flow: baris ke-Row di halamannya.
type FlowNode struct {
	Row    int
	Op     string
	Symbol string
	NG     string
	// TopMM: pusat simbol dari atas barisnya. Biasanya nodeTopMM; di baris
	// paling atas halaman yang punya konektor masuk, simbolnya digeser ke
	// bawah konektor (D-226). 0 dianggap nodeTopMM.
	TopMM float64
}

// centerMM memberi pusat simbol dari atas barisnya.
func (n FlowNode) centerMM() float64 {
	if n.TopMM == 0 {
		return nodeTopMM
	}
	return n.TopMM
}

// PlanPage adalah satu halaman: indeks baris Sheet.Rows, simbol alur, dan
// konektor di atas (In) dan di bawah (Out) halaman bila alur berlanjut.
type PlanPage struct {
	Rows  []int
	Nodes []FlowNode
	In    string
	Out   string
	// Filled: halaman ini berakhir lebih awal karena baris awal langkah
	// dipindah ke halaman berikutnya, jadi baris terakhirnya dipertinggi
	// sampai halaman penuh (D-226).
	Filled bool
}

// Plan adalah rencana halaman satu cetakan. Semua ukuran dalam milimeter.
type Plan struct {
	PaperW, PaperH float64
	ContentW       float64   // lebar area isi (kertas dikurangi margin kiri dan kanan)
	HeadMM         float64   // kepala form: judul, isian header, blok tanda tangan
	TheadMM        float64   // judul kolom (grup dan label)
	GroupRowMM     float64   // tinggi baris judul grup; 0 bila tidak ada
	LabelRowMM     float64   // tinggi baris label kolom
	MarkMM         float64   // lebar kolom tanda revisi; 0 bila tidak dicetak
	AvailableMM    float64   // ruang baris per halaman
	ColumnsMM      []float64 // lebar setiap kolom isi
	RowsMM         []float64 // tinggi setiap baris Sheet.Rows
	Pages          []PlanPage
	// PFD dicetak sebagai diagram (bagian B): satu DiagramPage per halaman,
	// legenda setinggi LegendMM, dan area diagram DiagramW x DiagramH.
	Diagram            []DiagramPage
	LegendMM           float64
	DiagramW, DiagramH float64
}

// Ukuran huruf ke milimeter: 1 pt = 0,3528 mm. Lebar rata-rata huruf Arial
// kira-kira setengah ukurannya; tinggi baris 1,3 kali ukurannya.
const ptMM = 0.3528

func charMM(font float64) float64 { return font * ptMM * 0.6 }
func lineMM(font float64) float64 { return font * ptMM * 1.3 }

// cellPadMM: jarak teks ke garis sel, atas ditambah bawah.
const cellPadMM = 2.0

// flowRowMM: tinggi paling kecil baris yang memulai satu langkah, supaya
// simbol alur muat.
const flowRowMM = 9.0

// connGapMM: panjang panah dari konektor masuk ke simbol pertama halaman.
const connGapMM = 1.5

// shiftedTopMM memberi pusat simbol di baris paling atas halaman yang punya
// konektor masuk: di bawah konektor, dengan panah pendek di antaranya (D-226).
func shiftedTopMM(symbol string) float64 {
	_, h := flowSize(symbol)
	return connEdgeMM + connRMM + connGapMM + h/2
}

// topRowMM memberi tinggi paling kecil baris awal langkah yang menjadi baris
// paling atas halaman 2 dan seterusnya: simbol yang digeser beserta jarak
// bawah yang sama dengan baris biasa.
func topRowMM(symbol string) float64 {
	_, h := flowSize(symbol)
	return shiftedTopMM(symbol) + h/2 + (flowRowMM - nodeTopMM - shapeHMM/2)
}

// linesOf menghitung banyak baris teks satu sel selebar widthMM, dengan
// membungkus per kata seperti browser dan Excel: kata yang tidak muat pindah
// ke baris berikutnya, dan kata yang lebih panjang dari satu baris dipotong.
func linesOf(text string, widthMM, font float64) int {
	perLine := int(math.Max(1, math.Floor((widthMM-cellPadMM)/charMM(font))))
	n := 0
	for _, part := range strings.Split(text, "\n") {
		lines, used := 1, 0
		for _, word := range strings.Fields(part) {
			w := utf8.RuneCountInString(word)
			switch {
			case used == 0:
				used = w
			case used+1+w <= perLine:
				used += 1 + w
				continue
			default:
				lines++
				used = w
			}
			for used > perLine {
				lines++
				used -= perLine
			}
		}
		n += lines
	}
	return n
}

// heightOf memperkirakan tinggi satu baris tabel dari isinya.
func heightOf(cells []string, widthsMM []float64, font float64) float64 {
	lines := 1
	for i, v := range cells {
		if i < len(widthsMM) {
			lines = max(lines, linesOf(v, widthsMM[i], font))
		}
	}
	return float64(lines)*lineMM(font) + cellPadMM
}

// paperMM memberi lebar dan tinggi kertas menurut layout.
func paperMM(l Layout) (float64, float64) {
	w, h := 297.0, 420.0
	if l.Paper == "A4" {
		w, h = 210, 297
	}
	if l.Orientation == "landscape" {
		w, h = h, w
	}
	return w, h
}

// Paginate membagi baris ke halaman menurut tingginya. Setiap halaman berisi
// paling sedikit satu baris, juga bila baris itu lebih tinggi dari halaman.
// keep[i] true berarti baris i tidak boleh menjadi baris terakhir halaman
// selama masih ada baris sesudahnya: baris pertama satu langkah selalu
// bersama baris berikutnya, supaya simbolnya tidak menumpuk dengan konektor
// di bawah halaman. topMin[i] adalah tinggi paling kecil baris i bila baris
// itu menjadi baris paling atas halaman 2 dan seterusnya (ruang untuk
// konektor masuk, D-226); 0 berarti tidak ada. kept[p] true berarti halaman p
// berakhir karena aturan keep: baris berikutnya sebenarnya masih muat.
func Paginate(heights []float64, available float64, keep []bool, topMin []float64) (pages [][]int, kept []bool) {
	pages = [][]int{{}}
	kept = []bool{false}
	used := 0.0
	for i, h := range heights {
		need := h
		if i < len(keep) && keep[i] && i+1 < len(heights) {
			need += heights[i+1]
		}
		last := len(pages) - 1
		if len(pages[last]) > 0 && used+need > available {
			kept[last] = used+h <= available
			if i < len(topMin) {
				h = math.Max(h, topMin[i])
			}
			pages = append(pages, []int{i})
			kept = append(kept, false)
			used = h
			continue
		}
		pages[last] = append(pages[last], i)
		used += h
	}
	return pages, kept
}

// opGroup adalah baris berurutan dengan nomor langkah yang sama.
type opGroup struct {
	start, end int
	op         string
}

// opGroups mengelompokkan baris berurutan menurut nomor langkah saja. Satu
// langkah boleh punya beberapa proses (di CP Tokairika langkah 2 berisi IQC
// untuk beberapa material), tetapi simbolnya tetap satu.
func opGroups(ops []string) []opGroup {
	var groups []opGroup
	for i, op := range ops {
		if n := len(groups); n > 0 && op != "" && groups[n-1].op == op && groups[n-1].end == i-1 {
			groups[n-1].end = i
			continue
		}
		groups = append(groups, opGroup{start: i, end: i, op: op})
	}
	return groups
}

// ConnectorLetter memberi huruf konektor ke-n: A sampai Z, lalu AA, AB, ...
func ConnectorLetter(n int) string {
	s := ""
	for i := n; i >= 0; i = i/26 - 1 {
		s = string(rune('A'+i%26)) + s
	}
	return s
}

// flowPages menempatkan simbol dan konektor. Simbol ada di baris pertama
// setiap langkah; simbolnya dari PFD, atau "process" bila langkah itu tidak
// ada di PFD. Konektor dipasang di setiap batas halaman yang dilewati alur,
// dengan huruf berurutan. Karena dihitung dari baris, menambah atau menghapus
// baris menggeser simbol dan konektor sendiri (G11).
func flowPages(ops []string, pages [][]int, steps map[string]Step) []PlanPage {
	var flowing []opGroup
	for _, g := range opGroups(ops) {
		if g.op != "" {
			flowing = append(flowing, g)
		}
	}
	first := func(p int) int { return pages[p][0] }
	last := func(p int) int { return pages[p][len(pages[p])-1] }
	crosses := func(p int) bool {
		if len(pages[p]) == 0 || len(pages[p+1]) == 0 {
			return false
		}
		started, continues := false, false
		for _, g := range flowing {
			started = started || g.start <= last(p)
			continues = continues || g.end >= first(p+1)
		}
		return started && continues
	}
	out := make([]PlanPage, len(pages))
	letters := 0
	for p, rows := range pages {
		pos := map[int]int{}
		for i, r := range rows {
			pos[r] = i
		}
		page := PlanPage{Rows: rows}
		for _, g := range flowing {
			at, ok := pos[g.start]
			if !ok {
				continue
			}
			step, known := steps[g.op]
			if !known || step.Symbol == "" {
				step.Symbol = "process"
			}
			page.Nodes = append(page.Nodes, FlowNode{Row: at, Op: g.op, Symbol: step.Symbol, NG: step.NG, TopMM: nodeTopMM})
		}
		if p > 0 && crosses(p-1) {
			page.In = out[p-1].Out
			// Simbol di baris paling atas digeser ke bawah konektor masuk (D-226).
			for i, n := range page.Nodes {
				if n.Row == 0 {
					page.Nodes[i].TopMM = shiftedTopMM(n.Symbol)
				}
			}
		}
		if p < len(pages)-1 && crosses(p) {
			page.Out = ConnectorLetter(letters)
			letters++
		}
		out[p] = page
	}
	return out
}

// BuildPlan menghitung rencana halaman satu model cetak.
func BuildPlan(sh Sheet) Plan {
	l := sh.Layout
	font := l.FontSize
	if font <= 0 {
		font = 8
	}
	w, h := paperMM(l)
	p := Plan{PaperW: w, PaperH: h, ContentW: w - l.MarginsMM.Left - l.MarginsMM.Right}

	total := 0.0
	for _, c := range sh.Columns {
		total += c.Width
	}
	if l.RevisionMarks {
		total += 0.35
	}
	if total == 0 {
		total = 1
	}
	if l.RevisionMarks {
		p.MarkMM = 0.35 / total * p.ContentW
	}
	for _, c := range sh.Columns {
		p.ColumnsMM = append(p.ColumnsMM, c.Width/total*p.ContentW)
	}

	// Kepala form menurut jenis dokumen (head.go).
	p.HeadMM = headMM(sh)
	const footMM = 6
	if isDiagram(sh) {
		p.LegendMM = diagLegendMM
		p.DiagramW = p.ContentW
		p.DiagramH = h - l.MarginsMM.Top - l.MarginsMM.Bottom - p.HeadMM - p.LegendMM - footMM
		p.Diagram = DiagramLayout(diagramSteps(sh), p.DiagramW, p.DiagramH)
		p.Pages = make([]PlanPage, len(p.Diagram))
		return p
	}
	// Judul kolom: baris grup (bila ada) dan baris label.
	labels := make([]string, len(sh.Columns))
	hasGroups := false
	for i, c := range sh.Columns {
		labels[i] = c.Label
		hasGroups = hasGroups || c.Group != ""
	}
	p.LabelRowMM = math.Round(heightOf(labels, p.ColumnsMM, font*1.1)*10) / 10
	if hasGroups {
		p.GroupRowMM = math.Round((lineMM(font)+cellPadMM)*10) / 10
	}
	p.TheadMM = p.LabelRowMM + p.GroupRowMM
	p.AvailableMM = h - l.MarginsMM.Top - l.MarginsMM.Bottom - p.HeadMM - p.TheadMM - footMM

	ops := make([]string, len(sh.Rows))
	starts := map[int]bool{}
	for _, g := range opGroups(sh.Ops) {
		if g.op != "" {
			starts[g.start] = true
		}
	}
	for i, r := range sh.Rows {
		if i < len(sh.Ops) {
			ops[i] = sh.Ops[i]
		}
		height := heightOf(r.Cells, p.ColumnsMM, font)
		if r.Cells == nil {
			height = lineMM(font) + cellPadMM // baris judul bagian per PCB
		}
		if sh.FlowColumn >= 0 && starts[i] {
			height = math.Max(height, flowRowMM)
		}
		p.RowsMM = append(p.RowsMM, math.Round(height*10)/10)
	}
	keep := make([]bool, len(sh.Rows))
	topMin := make([]float64, len(sh.Rows))
	if sh.FlowColumn >= 0 {
		first := true
		for _, g := range opGroups(sh.Ops) {
			if g.op == "" {
				continue
			}
			keep[g.start] = true
			// Langkah pertama tidak pernah punya konektor masuk di atasnya.
			if !first {
				step, known := sh.Steps[g.op]
				if !known || step.Symbol == "" {
					step.Symbol = "process"
				}
				topMin[g.start] = math.Ceil(topRowMM(step.Symbol)*10) / 10
			}
			first = false
		}
	}
	pages, kept := Paginate(p.RowsMM, p.AvailableMM, keep, topMin)
	for pi, rows := range pages {
		if len(rows) == 0 {
			continue
		}
		if pi > 0 {
			r := rows[0]
			p.RowsMM[r] = math.Max(p.RowsMM[r], topMin[r])
		}
	}
	p.Pages = flowPages(ops, pages, sh.Steps)
	// Halaman yang berakhir karena baris awal langkah dipindah diisi penuh:
	// baris terakhirnya dipertinggi sampai ruang halaman habis (D-226).
	for pi, rows := range pages {
		if !kept[pi] || len(rows) == 0 {
			continue
		}
		used := 0.0
		for _, r := range rows {
			used += p.RowsMM[r]
		}
		if slack := math.Floor((p.AvailableMM-used)*10) / 10; slack > 0 {
			last := rows[len(rows)-1]
			p.RowsMM[last] = math.Round((p.RowsMM[last]+slack)*10) / 10
			p.Pages[pi].Filled = true
		}
	}
	return p
}
