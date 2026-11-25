package printout

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/settings"
	"github.com/dkpauzi/revioten/internal/template"
)

func TestPaginate(t *testing.T) {
	got, kept := Paginate([]float64{40, 40, 40, 40, 40}, 100, nil, nil)
	if !reflect.DeepEqual(got, [][]int{{0, 1}, {2, 3}, {4}}) || !reflect.DeepEqual(kept, []bool{false, false, false}) {
		t.Errorf("isi halaman %v, %v", got, kept)
	}
	// Baris yang lebih tinggi dari halaman tetap mendapat halaman sendiri.
	if got, _ := Paginate([]float64{30, 200, 30}, 100, nil, nil); !reflect.DeepEqual(got, [][]int{{0}, {1}, {2}}) {
		t.Errorf("baris tinggi %v", got)
	}
	if got, _ := Paginate(nil, 100, nil, nil); !reflect.DeepEqual(got, [][]int{{}}) {
		t.Errorf("tanpa baris %v", got)
	}
	// Baris 1 memulai langkah: tidak boleh menjadi baris terakhir halaman, jadi
	// pindah ke halaman berikutnya bersama baris 2. Halaman 1 berakhir karena
	// aturan ini (kept), jadi nanti diisi penuh (D-226).
	got, kept = Paginate([]float64{40, 40, 40}, 100, []bool{true, true, false}, nil)
	if !reflect.DeepEqual(got, [][]int{{0}, {1, 2}}) || !reflect.DeepEqual(kept, []bool{true, false}) {
		t.Errorf("baris awal langkah %v, %v", got, kept)
	}
	// Baris di atas halaman 2 dan seterusnya mendapat tinggi paling kecil
	// topMin (ruang konektor masuk), dan itu ikut dihitung (D-226).
	if got, _ := Paginate([]float64{40, 40, 40, 40}, 100, nil, []float64{0, 0, 60, 0}); !reflect.DeepEqual(got, [][]int{{0, 1}, {2, 3}}) {
		t.Errorf("topMin muat %v", got)
	}
	if got, _ := Paginate([]float64{40, 40, 40, 40}, 100, nil, []float64{0, 0, 70, 0}); !reflect.DeepEqual(got, [][]int{{0, 1}, {2}, {3}}) {
		t.Errorf("topMin tidak muat %v", got)
	}
}

func TestConnectorLetter(t *testing.T) {
	var got []string
	for _, n := range []int{0, 1, 25, 26, 27} {
		got = append(got, ConnectorLetter(n))
	}
	if strings.Join(got, ",") != "A,B,Z,AA,AB" {
		t.Errorf("huruf %v", got)
	}
}

// TestFlowPages: simbol satu per langkah, konektor di setiap batas halaman.
// Langkah 2 punya dua proses (seperti IQC di CP Tokairika), tetapi simbolnya satu.
func TestFlowPages(t *testing.T) {
	steps := map[string]Step{"1": {Symbol: "process"}, "2": {Symbol: "decision", NG: "ncp"}, "3": {Symbol: "process"}}
	ops := []string{"1", "2", "2", "2", "3"}
	pages := flowPages(ops, [][]int{{0, 1}, {2, 3}, {4}}, steps)
	type summary struct {
		In, Out string
		Ops     []string
	}
	var got []summary
	for _, p := range pages {
		s := summary{In: p.In, Out: p.Out}
		for _, n := range p.Nodes {
			s.Ops = append(s.Ops, n.Op+":"+n.Symbol+":"+n.NG)
		}
		got = append(got, s)
	}
	want := []summary{
		{Out: "A", Ops: []string{"1:process:", "2:decision:ncp"}},
		{In: "A", Out: "B"},
		{In: "B", Ops: []string{"3:process:"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("halaman\n got %+v\nwant %+v", got, want)
	}
	// Langkah yang tidak ada di PFD memakai simbol proses.
	if p := flowPages([]string{"9"}, [][]int{{0}}, steps); p[0].Nodes[0].Symbol != "process" {
		t.Errorf("langkah tanpa PFD: %+v", p[0].Nodes)
	}
}

// TestBuildPlan: tinggi baris dari panjang teks, baris awal langkah cukup
// tinggi untuk simbol, dan baris baru di awal menggeser konektor sendiri.
func TestBuildPlan(t *testing.T) {
	sheet := func(rows int) Sheet {
		sh := Sheet{FlowColumn: 0, OpColumn: 1, ProcessColumn: -1, Columns: []Column{{Label: "Process Flow", Width: 1}, {Label: "No", Width: 1}, {Label: "Text", Width: 4}}}
		sh.Layout = Layout{Paper: "A4", Orientation: "portrait", FontSize: 8, MarginsMM: Margins{Top: 10, Right: 10, Bottom: 10, Left: 10}}
		sh.Steps = map[string]Step{}
		for i := 0; i < rows; i++ {
			op := []string{"1", "2", "3", "4"}[i*4/rows]
			sh.Ops = append(sh.Ops, op)
			sh.Rows = append(sh.Rows, Row{Cells: []string{"", op, strings.Repeat("kata ", 3+i%5)}})
		}
		return sh
	}
	p := BuildPlan(sheet(60))
	if p.PaperW != 210 || p.PaperH != 297 || p.ContentW != 190 {
		t.Errorf("kertas %v x %v, isi %v", p.PaperW, p.PaperH, p.ContentW)
	}
	if p.RowsMM[0] < flowRowMM {
		t.Errorf("baris awal langkah %v mm, ingin paling sedikit %v", p.RowsMM[0], flowRowMM)
	}
	if len(p.Pages) < 2 || p.Pages[0].Out != "A" || p.Pages[1].In != "A" {
		t.Fatalf("halaman %+v", p.Pages)
	}
	total := 0
	for _, page := range p.Pages {
		used := 0.0
		for _, r := range page.Rows {
			used += p.RowsMM[r]
		}
		if used > p.AvailableMM && len(page.Rows) > 1 {
			t.Errorf("halaman berisi %v mm, ruangnya %v", used, p.AvailableMM)
		}
		total += len(page.Nodes)
	}
	if total != 4 {
		t.Errorf("%d simbol, ingin 4 (satu per langkah)", total)
	}
	// Lebih banyak baris: halamannya bertambah, konektornya ikut dihitung.
	more := BuildPlan(sheet(120))
	if len(more.Pages) <= len(p.Pages) || more.Pages[len(more.Pages)-1].In == "" {
		t.Errorf("sesudah baris bertambah: %d halaman", len(more.Pages))
	}
}

// TestPageEdges (D-226, opsi 2): baris awal proses tidak pernah menjadi
// baris terakhir halaman; halaman yang berakhir karena aturan itu diisi penuh
// oleh baris terakhirnya; dan simbol di baris paling atas halaman 2 dan
// seterusnya digeser ke bawah konektor masuk, dengan baris yang cukup tinggi.
func TestPageEdges(t *testing.T) {
	sh := Sheet{FlowColumn: 0, OpColumn: 1, ProcessColumn: -1, Columns: []Column{{Label: "Process Flow", Width: 1}, {Label: "No", Width: 1}, {Label: "Text", Width: 4}}}
	sh.Layout = Layout{Paper: "A4", Orientation: "portrait", FontSize: 8, MarginsMM: Margins{Top: 10, Right: 10, Bottom: 10, Left: 10}}
	sh.Steps = map[string]Step{}
	for i := 0; i < 240; i++ {
		op := fmt.Sprint(i/2 + 1) // setiap langkah dua baris
		if i%6 == 0 {
			sh.Steps[op] = Step{Symbol: "decision", NG: "ncp"}
		}
		sh.Ops = append(sh.Ops, op)
		sh.Rows = append(sh.Rows, Row{Cells: []string{"", op, strings.Repeat("kata ", 3+i%9)}})
	}
	p := BuildPlan(sh)
	starts := map[int]bool{}
	for i := range sh.Ops {
		starts[i] = i == 0 || sh.Ops[i] != sh.Ops[i-1]
	}
	filled := 0
	for pi, page := range p.Pages {
		used := 0.0
		for _, r := range page.Rows {
			used += p.RowsMM[r]
		}
		last := page.Rows[len(page.Rows)-1]
		if pi < len(p.Pages)-1 && starts[last] {
			t.Errorf("halaman %d berakhir dengan baris awal langkah %d", pi+1, last)
		}
		if page.Filled {
			filled++
			if math.Abs(used-p.AvailableMM) > 0.15 {
				t.Errorf("halaman %d diisi %v mm, ruangnya %v", pi+1, used, p.AvailableMM)
			}
		}
		for _, n := range page.Nodes {
			_, h := flowSize(n.Symbol)
			if n.Row == 0 && page.In != "" {
				if top := n.TopMM - h/2; top < connEdgeMM+connRMM+1 {
					t.Errorf("halaman %d: simbol %s mulai %v mm, menumpuk dengan konektor", pi+1, n.Op, top)
				}
				if bottom := n.TopMM + h/2; bottom > p.RowsMM[page.Rows[0]] {
					t.Errorf("halaman %d: simbol %s keluar dari barisnya (%v > %v)", pi+1, n.Op, bottom, p.RowsMM[page.Rows[0]])
				}
			} else if n.TopMM != nodeTopMM {
				t.Errorf("halaman %d: simbol %s di %v mm, ingin %v", pi+1, n.Op, n.TopMM, nodeTopMM)
			}
		}
	}
	if filled == 0 {
		t.Error("contoh tidak memicu aturan baris awal langkah")
	}
}

// TestSampleTokairika menulis contoh CP berpola SC-TR-020-04 sebagai HTML dan
// PDF untuk diperiksa mata. Hanya berjalan bila REVIOTEN_SAMPLE_DIR diisi.
func TestSampleTokairika(t *testing.T) {
	dir := os.Getenv("REVIOTEN_SAMPLE_DIR")
	if dir == "" {
		t.Skip("REVIOTEN_SAMPLE_DIR kosong")
	}
	sh := tokairikaSheet()
	page, err := HTML(sh)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample-cp.html"), page, 0o600); err != nil {
		t.Fatal(err)
	}
	if browser, err := FindBrowser(""); err == nil {
		pdf, err := PDF(t.Context(), browser, page)
		if err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(dir, "sample-cp.pdf"), pdf, 0o600)
	}
	body, err := XLSX(sh)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "sample-cp.xlsx"), body, 0o600)
	// PFMEA berpola SF-TR-020-03 (bagian C).
	pfmea := tokairikaPFMEA()
	if pg, err := HTML(pfmea); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "sample-pfmea.html"), pg, 0o600)
	}
	if body, err := XLSX(pfmea); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "sample-pfmea.xlsx"), body, 0o600)
	}
	// PFD berpola SH-TR-020-02 sebagai diagram (bagian B).
	pfd := tokairikaPFD()
	pfdPage, err := HTML(pfd)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "sample-pfd.html"), pfdPage, 0o600)
	if browser, err := FindBrowser(""); err == nil {
		if pdf, err := PDF(t.Context(), browser, pfdPage); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "sample-pfd.pdf"), pdf, 0o600)
		}
	}
	if body, err := XLSX(pfd); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "sample-pfd.xlsx"), body, 0o600)
	}
}

// tokairikaFlow: 19 langkah PFD SH-TR-020-02 (nomor, nama, arah NG). Router
// (6.1) adalah proses dengan cabang NG ke SCRAP tanpa label OK; langkah lain
// yang punya NG adalah inspeksi.
var tokairikaFlow = [][3]string{{"1", "Part Receiving", ""}, {"2", "IQC", "ncp"}, {"3", "Material Storage", ""}, {"4", "Kitting", ""},
	{"5.1", "Loading PCB", ""}, {"5.2", "Cleaning", ""}, {"5.3", "Solder Paste Printing", ""}, {"5.4", "SPI", "scrap"},
	{"5.5", "SMT", ""}, {"5.6", "Reflow", ""}, {"5.7.1", "AOI", "scrap"}, {"5.7.2", "VMI", "scrap"}, {"6.1", "Router", "scrap"},
	{"6.2", "Dust removal", ""}, {"7", "ICT", "scrap"}, {"8", "LED Lighting Test", "scrap"}, {"9", "Final Visual Inspection", "scrap"},
	{"10", "OQC", "ncp"}, {"11", "Delivery", ""}}

// tokairikaSymbol memberi simbol langkah TR-020: inspeksi bila punya NG, kecuali Router.
func tokairikaSymbol(op, ng string) string {
	if ng != "" && op != "6.1" {
		return "decision"
	}
	return "process"
}

// tokairikaHead mengisi bahan kepala dan cover seperti set TR-020: customer,
// model, part, logo bawaan, slot rute lima tingkat, riwayat revisi, dan field
// header ber-key (migrasi 00056).
func tokairikaHead(sh *Sheet, kind, revision string, remark bool) {
	sh.Snapshot.DocumentType = kind
	sh.Snapshot.Models = []string{"D660A"}
	sh.Snapshot.PartNumbers = []string{"75J081-7022"}
	sh.Snapshot.PartNames = []string{"PC BOARD CP LH"}
	sh.CustomerName, sh.CompanyName, sh.Logo = "PT. TOKAIRIKA INDONESIA", "PT. SIIX EMS INDONESIA", settings.DefaultLogo
	sh.RevisionLabel = "-" + revision
	sh.Layout.Cover = true
	sh.Slots = []Slot{
		{Title: "Process. Eng", Stage: "prepared", Names: []string{"Dimas K."}},
		{Title: "Engineer", Stage: "checked", Names: []string{"A. Fikry"}},
		{Title: "NPI Manager", Stage: "approved", Names: []string{"Oie S"}},
		{Title: "QA / QC Mgr.", Stage: "approved", Names: []string{"Carter S."}},
		{Title: "General Mgr.", Stage: "approved", Names: []string{"Takagi T"}},
	}
	sh.History = []HistoryRow{{1, "2014-02-04", "00", "Initial Release"}, {2, "2026-10-23", revision, "Change False NG (Good Ratio) AOI process refer to SW-ZZ-116"}}
	sh.Snapshot.Header = map[string]any{"h-line": "PCBA Line Process", "h-team": "Dimas K, Khury, Fikry(NPI), Vera(WH), Nugraha (IQC), Jefri(QA/QC)", "h-resp": "Process Engineering", "h-key": "2026-10-23"}
	sh.Format.Fields = []template.Field{{ID: "h-line", Area: "header", Key: "process_line"}, {ID: "h-team", Area: "header", Key: "core_team"},
		{ID: "h-resp", Area: "header", Key: "process_responsibility"}, {ID: "h-key", Area: "header", Key: "key_date"}}
	if remark {
		sh.Snapshot.Header["h-remark"] = "SEV - Severity (1~10); OCC - Occurrence (1~10); DET - Detection (1~10)"
		sh.Format.Fields = append(sh.Format.Fields, template.Field{ID: "h-remark", Area: "header", Key: "remark"})
	}
}

// tokairikaPFD adalah PFD berpola SH-TR-020-02: satu baris per langkah.
func tokairikaPFD() Sheet {
	sh := Sheet{Title: "PROCESS FLOW DIAGRAM", Number: "SH-TR-020", FlowColumn: 2, OpColumn: 0, ProcessColumn: 1,
		Columns: []Column{{Label: "No", Width: 0.5}, {Label: "Process", Width: 1.6}, {Label: "Flow Symbol", Width: 1.3}, {Label: "NG Disposition", Width: 0.8}},
		Header:  []Pair{{"Customer", "PT. TOKAIRIKA INDONESIA (TR)"}, {"Model Name", "D660A"}, {"Document No.", "SH-TR-020"}, {"Revision", "-02"}}}
	sh.Layout = Layout{Paper: "A4", Orientation: "portrait", FontSize: 8, MarginsMM: Margins{Top: 10, Right: 10, Bottom: 12, Left: 10}}
	tokairikaHead(&sh, "pfd", "02", false)
	sh.Steps = map[string]Step{}
	for _, f := range tokairikaFlow {
		sh.Steps[f[0]] = Step{Symbol: tokairikaSymbol(f[0], f[2]), NG: f[2]}
		sh.Ops = append(sh.Ops, f[0])
		sh.Processes = append(sh.Processes, f[1])
		sh.Rows = append(sh.Rows, Row{Cells: []string{f[0], f[1], "", f[2]}})
	}
	return sh
}

// tokairikaPFMEA adalah PFMEA berpola SF-TR-020-03 dengan beberapa baris.
func tokairikaPFMEA() Sheet {
	cols := []Column{{Label: "Process Item No.", Width: 0.5}, {Label: "Process Step / Function", Width: 1.2}, {Label: "Requirement", Width: 1.2},
		{Label: "Potential Failure Mode", Width: 1.4}, {Label: "Potential Effect(s) of Failure", Width: 1.4}, {Label: "SEV", Width: 0.4}, {Label: "Class", Width: 0.4},
		{Label: "Potential Cause(s) / Mechanism(s) of Failure", Width: 1.6}, {Label: "Controls Prevention", Group: "Current Process", Width: 1.6},
		{Label: "OCC", Group: "Current Process", Width: 0.4}, {Label: "Controls Detection", Group: "Current Process", Width: 1.8}, {Label: "DET", Group: "Current Process", Width: 0.4},
		{Label: "RPN", Width: 0.5}, {Label: "Recommended Action", Width: 0.8}, {Label: "Responsibility & Target Completion Date", Width: 0.8},
		{Label: "Actions Taken Completion Date", Group: "Action Results", Width: 0.8}, {Label: "SEV", Group: "Action Results", Width: 0.4},
		{Label: "OCC", Group: "Action Results", Width: 0.4}, {Label: "DET", Group: "Action Results", Width: 0.4}, {Label: "RPN", Group: "Action Results", Width: 0.5}}
	sh := Sheet{Title: "PROCESS FMEA", Number: "SF-TR-020", FlowColumn: -1, OpColumn: 0, ProcessColumn: 1, Columns: cols}
	sh.Layout = Layout{Paper: "A3", Orientation: "portrait", FontSize: 7, MarginsMM: Margins{Top: 10, Right: 8, Bottom: 12, Left: 8}}
	tokairikaHead(&sh, "pfmea", "03", true)
	for _, f := range tokairikaFlow[:4] {
		sh.Ops = append(sh.Ops, f[0])
		sh.Processes = append(sh.Processes, f[1])
		sh.Rows = append(sh.Rows, Row{Cells: []string{f[0], f[1], "Correct Parts Number", "Wrong part number accepted", "Wrong part stored and issued", "7", "--",
			"Supplier issuing error", "Compare supplier label with DO / PO", "2", "100% check of supplier label", "6", "84", "--", "", "", "", "", "", ""}})
	}
	return sh
}

// tokairikaSheet adalah CP berpola SC-TR-020-04: 19 langkah PFD SH-TR-020-02,
// tiga baris per langkah, inspeksi dengan NG ke SCRAP atau NCP.
func tokairikaSheet() Sheet {
	flow := tokairikaFlow
	sh := Sheet{Title: "CONTROL PLAN", Number: "SC-TR-020", FlowColumn: 0, OpColumn: 1, ProcessColumn: 2,
		Columns: []Column{{Label: "Process Flow", Width: 1.3}, {Label: "No", Width: 0.5}, {Label: "Process Name / Operation Description", Width: 1.6},
			{Label: "Machine Equipment Jigs", Width: 1.4}, {Label: "Product Characteristic", Width: 1.4}, {Label: "Specification / Tolerance", Width: 1.4},
			{Label: "Process Characteristic", Width: 1.4}, {Label: "Evaluation / Measurement", Width: 1.4}, {Label: "Control Method", Width: 1.4}, {Label: "Reaction Plan", Width: 1.4}},
		Header: []Pair{{"Customer", "PT. TOKAIRIKA INDONESIA (TR)"}, {"Model Name", "D660A"}, {"Part Number", "75J081-7022"}, {"Document No.", "SC-TR-020"}, {"Revision", "-04"}}}
	sh.Layout = Layout{Paper: "A3", Orientation: "landscape", FontSize: 8, MarginsMM: Margins{Top: 12, Right: 10, Bottom: 14, Left: 10}}
	tokairikaHead(&sh, "cp", "04", true)
	sh.Steps = map[string]Step{}
	for _, f := range flow {
		sh.Steps[f[0]] = Step{Symbol: tokairikaSymbol(f[0], f[2]), NG: f[2]}
		for n := 1; n <= 3; n++ {
			sh.Ops = append(sh.Ops, f[0])
			sh.Processes = append(sh.Processes, f[1])
			sh.Rows = append(sh.Rows, Row{Cells: []string{"", f[0], f[1], "Barcode Scanner", "--", "--", fmt.Sprintf("%s item %d", f[1], n), "Visual check according to the work instruction", "Check sheet", "Hold lot, report to the superior"}})
		}
	}
	return sh
}

// TestTokairikaOutputs: PDF (HTML) dan Excel memakai rencana halaman yang sama:
// setiap langkah tepat satu simbol, konektor di setiap batas halaman, pemisah
// halaman Excel di awal halaman kedua dan seterusnya.
func TestTokairikaOutputs(t *testing.T) {
	sh := tokairikaSheet()
	plan := BuildPlan(sh)
	if len(plan.Pages) < 2 {
		t.Fatalf("%d halaman, ingin paling sedikit 2", len(plan.Pages))
	}
	page, err := HTML(sh)
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	if got := strings.Count(html, "data-step="); got != 19 {
		t.Errorf("%d simbol di HTML, ingin 19", got)
	}
	if got := strings.Count(html, `data-connector="A"`); got != 2 {
		t.Errorf("konektor A %d kali, ingin 2 (bawah halaman 1, atas halaman 2)", got)
	}
	if got := strings.Count(html, `class="sheet"`); got != len(plan.Pages) {
		t.Errorf("%d halaman HTML, rencana %d", got, len(plan.Pages))
	}
	body, err := XLSX(sh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := excelize.OpenReader(bytes.NewReader(body)); err != nil {
		t.Fatalf("Excel tidak bisa dibuka: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	var drawing, sheetXML string
	for _, zf := range zr.File {
		rc, _ := zf.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		if strings.HasPrefix(zf.Name, "xl/drawings/drawing") && strings.HasSuffix(zf.Name, ".xml") {
			drawing += string(data)
		}
		if strings.HasPrefix(zf.Name, "xl/worksheets/sheet") {
			sheetXML += string(data)
		}
	}
	if got := strings.Count(drawing, `prst="flowChartProcess"`) + strings.Count(drawing, `prst="flowChartDecision"`); got != 19 {
		t.Errorf("%d simbol di Excel, ingin 19", got)
	}
	if got := strings.Count(drawing, `prst="ellipse"`); got != 2*(len(plan.Pages)-1) {
		t.Errorf("%d konektor di Excel, ingin %d", got, 2*(len(plan.Pages)-1))
	}
	if got := strings.Count(sheetXML, "<brk "); got != len(plan.Pages)-1 {
		t.Errorf("%d pemisah halaman Excel, ingin %d", got, len(plan.Pages)-1)
	}
}
