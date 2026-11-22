package printout

import (
	"strings"
	"testing"
)

// Bagian B (E-144, D-161): kolom Process Flow CP mengikuti SC-TR-020-04.
// Tujuan NG digambar sebagai kotak di kiri alur: SCRAP dan NCP berbentuk
// predefined process (persegi dengan garis ganda di kiri dan kanan), REWORK
// kotak proses biasa. Label OK hanya untuk belah ketupat; langkah proses yang
// punya cabang NG (Router 6.1) tidak diberi OK.

func flowOf(nodes ...FlowNode) string {
	tops := make([]float64, len(nodes))
	for i := range nodes {
		nodes[i].Row = i
		tops[i] = float64(i) * 12
	}
	return FlowSVG(PlanPage{Rows: make([]int, len(nodes)), Nodes: nodes}, tops, float64(len(nodes))*12, 45)
}

func TestFlowSVGNGBoxes(t *testing.T) {
	svg := flowOf(
		FlowNode{Op: "2", Symbol: "decision", NG: "ncp"},
		FlowNode{Op: "6.1", Symbol: "process", NG: "scrap"},
		FlowNode{Op: "7", Symbol: "decision", NG: "rework"},
	)
	for _, want := range []string{
		`data-ng-box="ncp"`, `data-ng-box="scrap"`, `data-ng-box="rework"`,
		`>NCP</text>`, `>SCRAP</text>`, `>REWORK</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG tidak memuat %s", want)
		}
	}
	// Predefined process: dua garis tegak di dalam kotak SCRAP dan NCP, bukan di REWORK.
	if got := strings.Count(svg, `class="pp"`); got != 4 {
		t.Errorf("garis predefined process: %d, ingin 4", got)
	}
	// OK hanya di dua belah ketupat.
	if got := strings.Count(svg, `>OK</text>`); got != 2 {
		t.Errorf("label OK: %d, ingin 2", got)
	}
	if got := strings.Count(svg, `>NG</text>`); got != 3 {
		t.Errorf("label NG: %d, ingin 3", got)
	}
	// Kotak NG di kiri garis alur: x kotak lebih kecil dari pusat alur.
	if !strings.Contains(svg, `marker-end="url(#arrow)"`) {
		t.Error("garis alur tanpa kepala panah")
	}
}

// TestFlowSVGAlternateProcess: alternate process berbentuk segi enam
// (flowChartPreparation), seperti legenda PFD referensi.
func TestFlowSVGAlternateProcess(t *testing.T) {
	svg := flowOf(FlowNode{Op: "3", Symbol: "alternate_process"})
	if !strings.Contains(svg, `data-symbol="alternate_process"`) || !strings.Contains(svg, "<polygon") {
		t.Errorf("alternate process bukan segi enam:\n%s", svg)
	}
}
