package printout

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/dkpauzi/revioten/internal/template"
)

// D-227 (E-147): kolom Process Flow Control Plan v1 cukup lebar untuk kotak
// NG di kiri dan lompatan di kanan, dan kotak NG lurus ke samping kiri.

// cpV1Fields: kolom baris Control Plan v1 yang tampil, menurut jenisnya.
func cpV1Fields() []template.Field {
	fields := []template.Field{{Kind: template.KindProcessFlow}, {Kind: template.KindOperationNumber}, {Kind: template.KindProcess}}
	for _, key := range []string{"machine_tool", "product_characteristic", "product_specification", "process_characteristic", "process_specification"} {
		fields = append(fields, template.Field{Kind: "text", Key: key})
	}
	fields = append(fields, template.Field{Kind: template.KindProductCharNo}, template.Field{Kind: template.KindProcessCharNo}, template.Field{Kind: template.KindSymbol})
	for _, key := range []string{"evaluation_technique", "control_method", "sample_frequency", "owner", "relevant_document", "pokayoke", "reaction_plan"} {
		fields = append(fields, template.Field{Kind: "text", Key: key})
	}
	return fields
}

// TestFlowColumnWidth: dengan layout bawaan (A3 landscape, margin 8 mm, tanda
// revisi), kolom Process Flow Control Plan v1 paling sedikit 37 mm.
func TestFlowColumnWidth(t *testing.T) {
	sh := Sheet{FlowColumn: 0, OpColumn: 1, ProcessColumn: 2}
	for _, f := range cpV1Fields() {
		sh.Columns = append(sh.Columns, Column{Label: f.Key, Width: defaultWidth(f)})
	}
	sh.Layout = Layout{Paper: "A3", Orientation: "landscape", FontSize: 7, RevisionMarks: true, MarginsMM: Margins{Top: 10, Right: 8, Bottom: 12, Left: 8}}
	plan := BuildPlan(sh)
	if w := plan.ColumnsMM[0]; w < minFlowWidthMM {
		t.Errorf("kolom Process Flow %.1f mm, ingin paling sedikit %.0f mm", w, minFlowWidthMM)
	}
}

var ngRect = regexp.MustCompile(`<rect x="([0-9.]+)" y="([0-9.]+)" width="([0-9.]+)" height="([0-9.]+)" [^>]*data-ng-box`)

// TestNGBoxSideways: kotak NG sejajar dengan pusat simbol (lurus ke kiri),
// di kiri simbol, dan cukup lebar untuk tulisan REWORK.
func TestNGBoxSideways(t *testing.T) {
	const width = 37.6
	for _, symbol := range []string{"decision", "process"} {
		svg := FlowSVG(PlanPage{Rows: []int{0}, Nodes: []FlowNode{{Op: "5", Symbol: symbol, NG: "rework"}}}, []float64{0}, 12, width)
		m := ngRect.FindStringSubmatch(svg)
		if m == nil {
			t.Fatalf("%s: kotak NG tidak ada:\n%s", symbol, svg)
		}
		x, y, w, h := num(m[1]), num(m[2]), num(m[3]), num(m[4])
		w0, _ := flowSize(symbol)
		if mid := y + h/2; mid < nodeTopMM-0.01 || mid > nodeTopMM+0.01 {
			t.Errorf("%s: tengah kotak NG di %.2f, pusat simbol %.2f", symbol, mid, nodeTopMM)
		}
		if w < 12 {
			t.Errorf("%s: kotak NG %.2f mm, ingin paling sedikit 12 mm", symbol, w)
		}
		if x < 0 || x+w > flowX(width)-w0/2 {
			t.Errorf("%s: kotak NG %.2f..%.2f tidak di kiri simbol (%.2f)", symbol, x, x+w, flowX(width)-w0/2)
		}
	}
}

func num(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// jumpSteps: skenario pemilik (D-227). 20 AOI: OK langsung ke 40 (melewati
// VMI 30); 50 Finish 2nd side: No kembali ke 10 Loading PCB.
func jumpSteps() map[string]Step {
	return map[string]Step{
		"10": {Symbol: "process"},
		"20": {Symbol: "decision", Jump: "40", JumpLabel: "ok"},
		"30": {Symbol: "process", NG: "ncp"},
		"40": {Symbol: "process"},
		"50": {Symbol: "decision", Jump: "10", JumpLabel: "no"},
		"60": {Symbol: "process"},
	}
}

var jumpOps = []string{"10", "20", "30", "40", "50", "60"}

// TestFlowJumpsOnePage: lompatan maju menjadi garis, lompatan mundur menjadi
// sepasang konektor huruf; garis ke bawah mendapat label lawannya.
func TestFlowJumpsOnePage(t *testing.T) {
	pages := flowPages(jumpOps, nil, [][]int{{0, 1, 2, 3, 4, 5}}, jumpSteps())
	p := pages[0]
	if !reflect.DeepEqual(p.Lines, []FlowLine{{From: 1, To: 3}}) {
		t.Errorf("garis lompatan %+v, ingin 20 ke 40", p.Lines)
	}
	if n := p.Nodes[1]; n.Jump != "OK" || n.Down != "NG" || n.JumpOut != "" {
		t.Errorf("langkah 20: %+v", n)
	}
	if n := p.Nodes[4]; n.Jump != "No" || n.Down != "Yes" || n.JumpOut != "A" {
		t.Errorf("langkah 50: %+v", n)
	}
	if n := p.Nodes[0]; !reflect.DeepEqual(n.JumpIn, []string{"A"}) {
		t.Errorf("langkah 10: konektor masuk %v, ingin A", n.JumpIn)
	}
}

// TestFlowJumpLetters: konektor halaman dan konektor lompatan memakai satu
// urutan huruf menurut urutan muncul, tanpa huruf ganda.
func TestFlowJumpLetters(t *testing.T) {
	pages := flowPages(jumpOps, nil, [][]int{{0, 1, 2}, {3, 4, 5}}, jumpSteps())
	// Urutan muncul: 10 (tujuan No dari 50) = A, 20 (OK ke 40, menyeberang
	// halaman) = B, bawah halaman 1 = C.
	got := []string{pages[0].Nodes[0].JumpIn[0], pages[0].Nodes[1].JumpOut, pages[0].Out, pages[1].In, pages[1].Nodes[0].JumpIn[0], pages[1].Nodes[1].JumpOut}
	want := []string{"A", "B", "C", "C", "B", "A"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("huruf %v, ingin %v", got, want)
	}
	if len(pages[0].Lines)+len(pages[1].Lines) != 0 {
		t.Errorf("lompatan menyeberang halaman tidak boleh menjadi garis: %+v %+v", pages[0].Lines, pages[1].Lines)
	}
}

// TestFlowJumpOverlap: garis kedua yang menumpuk di jalur yang sama menjadi
// konektor.
func TestFlowJumpOverlap(t *testing.T) {
	steps := map[string]Step{"1": {Symbol: "decision", Jump: "4", JumpLabel: "ok"}, "2": {Symbol: "decision", Jump: "3", JumpLabel: "ok"}}
	pages := flowPages([]string{"1", "2", "3", "4"}, nil, [][]int{{0, 1, 2, 3}}, steps)
	if len(pages[0].Lines) != 1 || pages[0].Nodes[1].JumpOut != "A" {
		t.Errorf("garis %+v, langkah 2 %+v", pages[0].Lines, pages[0].Nodes[1])
	}
}

// TestFlowJumpSections: tujuan dicari di bagian per PCB yang sama.
func TestFlowJumpSections(t *testing.T) {
	steps := map[string]Step{"1": {Symbol: "process"}, "2": {Symbol: "decision", Jump: "1", JumpLabel: "no"}}
	ops := []string{"", "1", "2", "", "1", "2"}
	sec := []int{1, 1, 1, 2, 2, 2}
	pages := flowPages(ops, sec, [][]int{{0, 1, 2, 3, 4, 5}}, steps)
	p := pages[0]
	if p.Nodes[1].JumpOut != "A" || p.Nodes[0].JumpIn[0] != "A" || p.Nodes[3].JumpOut != "B" || p.Nodes[2].JumpIn[0] != "B" {
		t.Errorf("lompatan per bagian: %+v", p.Nodes)
	}
}

// TestJumpWarnings: tujuan yang tidak ada, ke diri sendiri, dan lompatan di
// simbol selain Decision.
func TestJumpWarnings(t *testing.T) {
	steps := map[string]Step{
		"1": {Symbol: "process", Jump: "3", JumpLabel: "ok"},
		"2": {Symbol: "decision", Jump: "2"},
		"3": {Symbol: "decision", Jump: "9"},
		"4": {Symbol: "decision", Jump: "1", JumpLabel: "no"},
	}
	got := jumpWarnings([]string{"1", "2", "3", "4"}, nil, steps)
	want := []PlanWarning{
		{Kind: WarnJumpNotDecision, Operation: "1", Target: "3"},
		{Kind: WarnJumpSelf, Operation: "2", Target: "2"},
		{Kind: WarnJumpUnknown, Operation: "3", Target: "9"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("peringatan\n got %+v\nwant %+v", got, want)
	}
}

// diagramJumpSteps: skenario pemilik sebagai langkah diagram PFD.
func diagramJumpSteps() []DiagramStep {
	steps := jumpSteps()
	var out []DiagramStep
	for _, op := range jumpOps {
		s := steps[op]
		out = append(out, DiagramStep{Op: op, Name: "Step " + op, Symbol: s.Symbol, NG: s.NG, Jump: s.Jump, JumpLabel: s.JumpLabel})
	}
	return out
}

// TestDiagramJumps: di kolom yang sama, lompatan maju dan mundur menjadi garis
// di jalur yang berbeda; di kolom yang berbeda menjadi konektor huruf yang
// berurutan dengan konektor kolom.
func TestDiagramJumps(t *testing.T) {
	pages := DiagramLayout(diagramJumpSteps(), 194, 250)
	c := pages[0].Columns[0]
	if !reflect.DeepEqual(c.Lines, []DiagramLine{{From: 1, To: 3, Lane: 0}, {From: 4, To: 0, Lane: 1}}) {
		t.Errorf("garis satu kolom: %+v", c.Lines)
	}
	if c.Nodes[1].Jump != "OK" || c.Nodes[1].Down != "NG" || c.Nodes[4].Down != "Yes" {
		t.Errorf("label: %+v", c.Nodes)
	}
	svg := DiagramSVG(pages[0], 194, 250)
	for _, want := range []string{`data-jump="line"`, `data-jump-label="OK"`, `data-jump-label="No"`, `data-down="Yes"`} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG tidak memuat %s", want)
		}
	}

	// Ruang untuk tiga langkah per kolom: 10, 20, 30 | 40, 50, 60.
	pages = DiagramLayout(diagramJumpSteps(), 194, 75)
	cols := pages[0].Columns
	if len(cols) != 2 {
		t.Fatalf("%d kolom, ingin 2", len(cols))
	}
	got := []string{cols[0].Nodes[0].JumpIn[0], cols[0].Nodes[1].JumpOut, cols[0].Out, cols[1].In, cols[1].Nodes[0].JumpIn[0], cols[1].Nodes[1].JumpOut}
	if want := []string{"A", "B", "C", "C", "B", "A"}; !reflect.DeepEqual(got, want) {
		t.Errorf("huruf %v, ingin %v", got, want)
	}
	for _, p := range diagramJumpParts(cols[0]) {
		if p.Kind == "conn" && p.X1+diagConnR > 194/2 {
			t.Errorf("konektor lompatan keluar dari kolom: %.2f", p.X1+diagConnR)
		}
	}
}

// TestFlowJumpsFit: di kolom Process Flow Control Plan v1 (bobot 2,1), garis,
// konektor, dan label lompatan muat di kanan, dan kotak NG 12 mm di kiri.
func TestFlowJumpsFit(t *testing.T) {
	sh := Sheet{FlowColumn: 0, OpColumn: 1, ProcessColumn: 2}
	for _, f := range cpV1Fields() {
		sh.Columns = append(sh.Columns, Column{Label: f.Key, Width: defaultWidth(f)})
	}
	sh.Layout = Layout{Paper: "A3", Orientation: "landscape", FontSize: 7, RevisionMarks: true, MarginsMM: Margins{Top: 10, Right: 8, Bottom: 12, Left: 8}}
	width := BuildPlan(sh).ColumnsMM[0]
	cx := flowX(width)
	if x, w := ngBoxSpan(cx, shapeWMM+2, 0.6); w < 12-0.01 || x < 0.6-0.01 {
		t.Errorf("kotak NG di kolom %.1f mm: x %.2f lebar %.2f", width, x, w)
	}
	pages := flowPages(jumpOps, nil, [][]int{{0, 1, 2, 3, 4, 5}}, jumpSteps())
	ys := make([]float64, len(pages[0].Nodes))
	for i := range ys {
		ys[i] = float64(i)*12 + nodeTopMM
	}
	same := func(v float64) float64 { return v }
	parts := cpJumpParts(pages[0], ys, cx, same)
	if len(parts) == 0 {
		t.Fatal("tidak ada gambar lompatan")
	}
	for _, p := range parts {
		right := math.Max(p.X1, p.X2)
		switch p.Kind {
		case "conn":
			right = p.X1 + connRMM
		case "label":
			right = p.X1 + 1.8
		}
		if right > width-0.3 {
			t.Errorf("%s %q sampai %.2f mm, kolom %.2f mm", p.Kind, p.Text, right, width)
		}
	}
	svg := FlowSVG(pages[0], []float64{0, 12, 24, 36, 48, 60}, 72, width)
	for _, want := range []string{`data-jump-label="OK"`, `data-jump-label="No"`, `data-down="NG"`, `data-down="Yes"`, `data-connector="A"`} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG tidak memuat %s", want)
		}
	}
}

// jumpFlow mengubah langkah Tokairika menjadi skenario lompatan pemilik:
// AOI (5.7.1) NG ke VMI di bawahnya, OK langsung ke Router (6.1); Dust
// removal (6.2) diganti Finish 2nd side: Yes lanjut, No kembali ke Loading
// PCB (5.1).
func jumpFlow(steps map[string]Step, names []string, ops []string) []string {
	steps["5.7.1"] = Step{Symbol: "decision", Jump: "6.1", JumpLabel: "ok"}
	steps["6.2"] = Step{Symbol: "decision", Jump: "5.1", JumpLabel: "no"}
	for i, op := range ops {
		if op == "6.2" {
			names[i] = "Finish 2nd side"
		}
	}
	return names
}

// jumpCP: CP Tokairika dengan kolom Control Plan v1 dan lompatan pemilik.
func jumpCP() Sheet {
	cp := tokairikaSheet()
	cp.Columns = nil
	for _, f := range cpV1Fields() {
		label := f.Key
		switch f.Kind {
		case template.KindProcessFlow:
			label = "Process Flow"
		case template.KindOperationNumber:
			label = "No"
		case template.KindProcess:
			label = "Process Name / Operation Description"
		}
		cp.Columns = append(cp.Columns, Column{Label: label, Width: defaultWidth(f)})
	}
	for i := range cp.Rows {
		cells := []string{"", cp.Ops[i], "", "Barcode Scanner"}
		for len(cells) < len(cp.Columns) {
			cells = append(cells, "--")
		}
		cp.Rows[i].Cells = cells
	}
	cp.Layout = Layout{Paper: "A3", Orientation: "landscape", FontSize: 7, RevisionMarks: true, Cover: true, MarginsMM: Margins{Top: 10, Right: 8, Bottom: 12, Left: 8}}
	cp.Processes = jumpFlow(cp.Steps, cp.Processes, cp.Ops)
	for i := range cp.Rows {
		cp.Rows[i].Cells[2] = cp.Processes[i]
	}
	return cp
}

// writeJumpSamples menulis contoh CP dan PFD dengan lompatan untuk diperiksa
// mata: sample-cp-jump dan sample-pfd-jump (HTML, PDF, Excel).
func writeJumpSamples(t *testing.T, dir string) {
	pfd := tokairikaPFD()
	pfd.Processes = jumpFlow(pfd.Steps, pfd.Processes, pfd.Ops)
	for name, sh := range map[string]Sheet{"sample-cp-jump": jumpCP(), "sample-pfd-jump": pfd} {
		page, err := HTML(sh)
		if err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(dir, name+".html"), page, 0o600)
		if browser, err := FindBrowser(""); err == nil {
			if pdf, err := PDF(t.Context(), browser, page); err == nil {
				_ = os.WriteFile(filepath.Join(dir, name+".pdf"), pdf, 0o600)
			}
		}
		if body, err := XLSX(sh); err == nil {
			_ = os.WriteFile(filepath.Join(dir, name+".xlsx"), body, 0o600)
		}
	}
}

// TestJumpXLSX: Excel menggambar konektor lompatan sebagai shape juga: dua
// per batas halaman dan dua untuk No dari Finish 2nd side ke Loading PCB.
func TestJumpXLSX(t *testing.T) {
	sh := jumpCP()
	plan := BuildPlan(sh)
	body, err := XLSX(sh)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	drawing := ""
	for _, zf := range zr.File {
		if strings.HasPrefix(zf.Name, "xl/drawings/drawing") && strings.HasSuffix(zf.Name, ".xml") {
			rc, _ := zf.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			drawing += string(data)
		}
	}
	if got, want := strings.Count(drawing, `prst="ellipse"`), 2*(len(plan.Pages)-1)+2; got != want {
		t.Errorf("%d konektor di Excel, ingin %d", got, want)
	}
	for _, label := range []string{">OK<", ">No<", ">Yes<"} {
		if !strings.Contains(drawing, label) {
			t.Errorf("Excel tidak memuat label %s", label)
		}
	}
}
