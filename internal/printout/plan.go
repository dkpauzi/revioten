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
	"sort"
	"strings"
	"unicode/utf8"
)

// Step adalah satu langkah PFD: simbol alur, untuk inspeksi arah NG
// (R-DOC-09), dan lompatan ke langkah lain (D-227): Jump nomor langkah
// tujuan, JumpLabel ok, ng, yes, atau no.
type Step struct {
	Symbol    string
	NG        string
	Jump      string
	JumpLabel string
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
	// Lompatan (D-227). Jump: label garis lompatan (OK, NG, Yes, No); Down:
	// label garis lurus ke bawah (lawan Jump, atau OK di belah ketupat yang
	// punya kotak NG). JumpOut: huruf konektor lompatan yang keluar dari
	// simbol ini; JumpIn: huruf konektor lompatan yang menuju simbol ini.
	Jump    string
	Down    string
	JumpOut string
	JumpIn  []string
}

// FlowLine adalah lompatan maju di satu halaman, digambar sebagai garis di
// kanan alur: dari simbol ke-From ke simbol ke-To di Nodes halaman itu.
type FlowLine struct {
	From, To int
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
	Lines []FlowLine
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
	// Warnings: hal yang perlu diperiksa penyusun sebelum mencetak (D-227),
	// ditampilkan di atas pratinjau.
	Warnings []PlanWarning
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
// ada di PFD. Konektor dipasang di setiap batas halaman yang dilewati alur.
// Karena dihitung dari baris, menambah atau menghapus baris menggeser simbol
// dan konektor sendiri (G11).
//
// Lompatan (D-227): lompatan maju di halaman yang sama menjadi garis di kanan
// alur; lompatan mundur, yang menyeberang halaman, atau yang garisnya akan
// menumpuk dengan garis lain menjadi sepasang konektor. Konektor halaman dan
// konektor lompatan memakai satu urutan huruf menurut urutan muncul. sec
// memberi nomor bagian per PCB setiap baris (nil berarti satu bagian), karena
// tujuan lompatan dicari di bagian yang sama.
func flowPages(ops []string, sec []int, pages [][]int, steps map[string]Step) []PlanPage {
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
	// at: letak simbol setiap langkah (indeks di flowing): halaman dan indeks
	// di Nodes halaman itu.
	at := make([][2]int, len(flowing))
	for p, rows := range pages {
		pos := map[int]int{}
		for i, r := range rows {
			pos[r] = i
		}
		page := PlanPage{Rows: rows}
		for k, g := range flowing {
			row, ok := pos[g.start]
			if !ok {
				continue
			}
			step, known := steps[g.op]
			if !known || step.Symbol == "" {
				step.Symbol = "process"
			}
			n := FlowNode{Row: row, Op: g.op, Symbol: step.Symbol, NG: step.NG, TopMM: nodeTopMM}
			if step.NG != "" && step.Symbol == "decision" {
				n.Down = "OK"
			}
			at[k] = [2]int{p, len(page.Nodes)}
			page.Nodes = append(page.Nodes, n)
		}
		out[p] = page
	}

	// Huruf konektor dibagikan menurut urutan muncul: konektor halaman di
	// bawah halamannya, konektor lompatan di langkah yang lebih atas.
	type event struct {
		pos  float64
		page int // konektor halaman: halaman yang berakhir; -1 untuk lompatan
		from int // lompatan: indeks langkah asal dan tujuan di flowing
		to   int
	}
	var events []event
	for p := 0; p+1 < len(pages); p++ {
		if crosses(p) {
			events = append(events, event{pos: float64(last(p)) + 0.5, page: p})
		}
	}
	section := func(row int) int {
		if row < len(sec) {
			return sec[row]
		}
		return 0
	}
	for k, g := range flowing {
		step := steps[g.op]
		to := jumpTarget(flowing, k, step.Jump, section)
		if to < 0 {
			continue
		}
		p, i := at[k][0], at[k][1]
		if label := jumpText(step.JumpLabel); label != "" {
			out[p].Nodes[i].Jump = label
			out[p].Nodes[i].Down = jumpText(oppositeJump(step.JumpLabel))
		}
		if to > k && at[to][0] == p && !linesOverlap(out[p].Lines, i, at[to][1]) {
			out[p].Lines = append(out[p].Lines, FlowLine{From: i, To: at[to][1]})
			continue
		}
		events = append(events, event{pos: float64(min(g.start, flowing[to].start)), page: -1, from: k, to: to})
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].pos < events[j].pos })
	for n, e := range events {
		letter := ConnectorLetter(n)
		if e.page >= 0 {
			out[e.page].Out = letter
			out[e.page+1].In = letter
			continue
		}
		from, to := at[e.from], at[e.to]
		out[from[0]].Nodes[from[1]].JumpOut = letter
		out[to[0]].Nodes[to[1]].JumpIn = append(out[to[0]].Nodes[to[1]].JumpIn, letter)
	}
	// Simbol di baris paling atas digeser ke bawah konektor masuk (D-226).
	for p := range out {
		if out[p].In == "" {
			continue
		}
		for i, n := range out[p].Nodes {
			if n.Row == 0 {
				out[p].Nodes[i].TopMM = shiftedTopMM(n.Symbol)
			}
		}
	}
	return out
}

// jumpTarget memberi indeks langkah tujuan lompatan langkah ke-k di flowing:
// langkah dengan nomor op di bagian yang sama, bukan langkah itu sendiri.
// -1 bila tidak ada lompatan atau tujuannya tidak ditemukan.
func jumpTarget(flowing []opGroup, k int, op string, section func(int) int) int {
	op = strings.TrimSpace(op)
	if op == "" {
		return -1
	}
	for j, g := range flowing {
		if j != k && g.op == op && section(g.start) == section(flowing[k].start) {
			return j
		}
	}
	return -1
}

// linesOverlap melaporkan apakah garis lompatan dari simbol from ke to akan
// menumpuk dengan garis lain di halaman yang sama (satu jalur di kanan alur).
// Garis yang hanya bersambung di satu simbol tidak dianggap menumpuk.
func linesOverlap(lines []FlowLine, from, to int) bool {
	for _, l := range lines {
		if from < l.To && l.From < to {
			return true
		}
	}
	return false
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
		p.Warnings = jumpWarnings(sh.Ops, sectionsOf(sh), sh.Steps)
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
	sec := sectionsOf(sh)
	p.Pages = flowPages(ops, sec, pages, sh.Steps)
	if sh.FlowColumn >= 0 {
		p.Warnings = jumpWarnings(sh.Ops, sec, sh.Steps)
		if w := p.ColumnsMM[sh.FlowColumn]; w < minFlowWidthMM {
			p.Warnings = append(p.Warnings, PlanWarning{Kind: WarnFlowNarrow, WidthMM: math.Round(w*10) / 10})
		}
	}
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
