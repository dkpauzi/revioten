package printout

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// Bagian B (E-144, D-161): cetakan PFD berupa halaman diagram seperti
// SH-TR-020-02, bukan tabel. Setiap langkah berupa kotak "nomor. nama";
// inspeksi berupa belah ketupat yang memuat kotak itu; NG ke kotak SCRAP,
// NCP, atau REWORK di kiri; alur turun dalam kolom, lalu berlanjut ke kolom
// berikutnya dan halaman berikutnya lewat konektor A, B, C.

// tokairikaSteps adalah 19 langkah PFD SH-TR-020-02.
func tokairikaSteps() []DiagramStep {
	var out []DiagramStep
	for _, f := range tokairikaFlow {
		out = append(out, DiagramStep{Op: f[0], Name: f[1], Symbol: tokairikaSymbol(f[0], f[2]), NG: f[2]})
	}
	return out
}

// TestDiagramLayout: setiap langkah tepat sekali, urut, dan konektor di
// setiap pindah kolom dengan huruf yang sama di ujung dan pangkal.
func TestDiagramLayout(t *testing.T) {
	steps := tokairikaSteps()
	// A4 tegak, margin 10 mm, kepala dan legenda 70 mm.
	pages := DiagramLayout(steps, 190, 200)
	var ops []string
	var columns []DiagramColumn
	for _, p := range pages {
		if len(p.Columns) == 0 || len(p.Columns) > 2 {
			t.Fatalf("kolom per halaman: %d", len(p.Columns))
		}
		columns = append(columns, p.Columns...)
	}
	for _, c := range columns {
		for _, n := range c.Nodes {
			ops = append(ops, n.Step.Op)
		}
	}
	var want []string
	for _, s := range steps {
		want = append(want, s.Op)
	}
	if strings.Join(ops, ",") != strings.Join(want, ",") {
		t.Fatalf("urutan langkah %v", ops)
	}
	if len(columns) < 2 {
		t.Fatalf("19 langkah dalam %d kolom, ingin lebih dari satu", len(columns))
	}
	for i := 0; i < len(columns)-1; i++ {
		if columns[i].Out == "" || columns[i].Out != columns[i+1].In {
			t.Errorf("kolom %d: keluar %q, kolom berikut masuk %q", i, columns[i].Out, columns[i+1].In)
		}
	}
	if columns[0].In != "" || columns[len(columns)-1].Out != "" {
		t.Errorf("kolom pertama masuk %q, kolom terakhir keluar %q", columns[0].In, columns[len(columns)-1].Out)
	}
	if columns[0].Out != "A" {
		t.Errorf("konektor pertama %q, ingin A", columns[0].Out)
	}
	// Simpul tidak keluar dari tinggi area dan tidak saling tumpang.
	for _, c := range columns {
		for i, n := range c.Nodes {
			if n.Y-n.H/2 < 0 || n.Y+n.H/2 > 200 {
				t.Errorf("langkah %s di luar area: y %.1f, tinggi %.1f", n.Step.Op, n.Y, n.H)
			}
			if i > 0 && c.Nodes[i-1].Y+c.Nodes[i-1].H/2 >= n.Y-n.H/2 {
				t.Errorf("langkah %s menumpuk langkah sebelumnya", n.Step.Op)
			}
		}
	}
	if got := DiagramLayout(nil, 190, 200); len(got) != 1 || len(got[0].Columns) != 0 {
		t.Errorf("PFD kosong: %+v", got)
	}
}

// TestDiagramSVG: kotak berisi nomor dan nama, belah ketupat memuat kotak,
// kotak NG, label OK hanya untuk inspeksi, dan legenda.
func TestDiagramSVG(t *testing.T) {
	steps := append(tokairikaSteps(), DiagramStep{Op: "12", Name: "Touch up", Symbol: "decision", NG: "rework"})
	pages := DiagramLayout(steps, 190, 200)
	var all strings.Builder
	for _, p := range pages {
		all.WriteString(DiagramSVG(p, 190, 200))
	}
	svg := all.String()
	for _, want := range []string{
		`data-step="1"`, `>1. Part Receiving</tspan>`, `data-step="5.7.1"`, `data-ng-box="scrap"`, `data-ng-box="ncp"`, `data-ng-box="rework"`,
		`data-connector="A"`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG tidak memuat %s", want)
		}
	}
	// Inspeksi: belah ketupat dan kotak di dalamnya.
	if got := strings.Count(svg, `data-symbol="decision"`); got != 9 {
		t.Errorf("inspeksi %d, ingin 9 (8 dari TR-020 dan Touch up)", got)
	}
	// OK hanya untuk inspeksi (Router punya NG tanpa OK).
	if got := strings.Count(svg, `>OK</text>`); got != 9 {
		t.Errorf("label OK %d, ingin 9", got)
	}
	if got := strings.Count(svg, `>NG</text>`); got != 10 {
		t.Errorf("label NG %d, ingin 10", got)
	}
	legend := DiagramLegendSVG(190)
	for _, label := range []string{"PROCESS", "ALTERNATE PROCESS", "DECISION", "CONNECTOR", "MERGE", "DOCUMENT", "REWORK", "SCRAP", "NCP"} {
		if !strings.Contains(legend, ">"+label+"<") {
			t.Errorf("legenda tanpa %s", label)
		}
	}
}

// TestDiagramWrap: nama panjang dibungkus beberapa baris di dalam kotak.
func TestDiagramWrap(t *testing.T) {
	lines := wrapText("4. Material Kitting, Stock-Out and Issue Part", 20)
	if len(lines) < 2 || strings.Join(lines, " ") != "4. Material Kitting, Stock-Out and Issue Part" {
		t.Errorf("baris %q", lines)
	}
	for _, l := range lines {
		if len([]rune(l)) > 20 {
			t.Errorf("baris terlalu panjang %q", l)
		}
	}
}

// TestDiagramXLSX: sheet PFD berisi shape diagram, dan garis alur diberi
// kepala panah (Excel tidak bisa membuatnya lewat excelize, jadi ditambahkan
// ke XML drawing).
func TestDiagramXLSX(t *testing.T) {
	sh := tokairikaPFD()
	body, err := XLSX(sh)
	if err != nil {
		t.Fatal(err)
	}
	drawing := drawingXML(t, body)
	for _, want := range []string{"flowChartPredefinedProcess", "flowChartDecision", "flowChartProcess", "1. Part Receiving", "SCRAP", "NCP", `<a:tailEnd type="triangle"`, "ALTERNATE PROCESS"} {
		if !strings.Contains(drawing, want) {
			t.Errorf("drawing PFD tidak memuat %s", want)
		}
	}
}

// drawingXML mengambil semua XML drawing dari berkas Excel.
func drawingXML(t *testing.T, body []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/drawings/drawing") {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			out.Write(b)
		}
	}
	return out.String()
}

// TestDiagramHTML: PDF PFD berisi legenda dan diagram, tanpa tabel baris.
func TestDiagramHTML(t *testing.T) {
	sh := tokairikaPFD()
	plan := BuildPlan(sh)
	if len(plan.Diagram) == 0 || len(plan.Pages) != len(plan.Diagram) {
		t.Fatalf("rencana PFD: %d halaman diagram, %d halaman", len(plan.Diagram), len(plan.Pages))
	}
	page, err := HTML(sh)
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	if got := strings.Count(html, `<g data-step=`); got != 19 {
		t.Errorf("%d langkah di HTML, ingin 19", got)
	}
	if got := strings.Count(html, `class="legend"`); got != len(plan.Diagram) {
		t.Errorf("legenda %d kali, ingin sekali per halaman (%d)", got, len(plan.Diagram))
	}
	if strings.Contains(html, "<tbody>") {
		t.Error("PFD masih dicetak sebagai tabel")
	}
}
