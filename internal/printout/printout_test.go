package printout

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/template"
)

// sample membentuk bahan cetak CP kecil: dua baris di revisi 1, satu berubah
// sejak revisi 0, dan rute dengan tiga slot.
func sample(t *testing.T, cover bool) Sheet {
	t.Helper()
	format := template.FormatVersion{Fields: []template.Field{
		{ID: "h1", Area: template.AreaHeader, Kind: template.KindText, Label: "Part Name"},
		{ID: "op", Area: template.AreaRow, Kind: template.KindOperationNumber, Label: "Part / Process Number", Group: "PROCESS"},
		{ID: "pr", Area: template.AreaRow, Kind: template.KindProcess, Label: "Process Name / Operation Description", Group: "PROCESS"},
		{ID: "pc", Area: template.AreaRow, Kind: template.KindText, Key: "product_characteristic", Label: "Product Characteristics", Group: "CHARACTERISTICS"},
		{ID: "me", Area: template.AreaRow, Kind: template.KindSelect, Key: "control_method", Label: "Method", Options: []template.Option{{Key: "spi", Label: "SPI 100%"}}},
		{ID: "hd", Area: template.AreaRow, Kind: template.KindText, Label: "Hidden", Hidden: true},
	}}
	prev := document.Snapshot{Number: "SC-AB-001", Revision: 0, Rows: []document.SnapshotRow{
		{ID: "r1", Position: 1, ProcessName: "SMT Printing", OperationNumber: "10", Content: map[string]any{"pc": "Solder paste height"}},
		{ID: "r2", Position: 2, ProcessName: "Reflow", OperationNumber: "20", Content: map[string]any{"pc": "Profile"}},
	}}
	cur := document.Snapshot{Number: "SC-AB-001", DocumentType: "cp", CustomerCode: "AB", Revision: 1, Models: []string{"K-125"},
		Header: map[string]any{"h1": "Main board"},
		Rows: []document.SnapshotRow{
			{ID: "r1", Position: 1, ProcessName: "SMT Printing", OperationNumber: "10", Content: map[string]any{"pc": "Solder paste height", "me": "spi"}},
			{ID: "r2", Position: 2, ProcessName: "Reflow", OperationNumber: "20", Content: map[string]any{"pc": "Profile"}},
		}}
	layout, err := ParseLayout([]byte(`{"paper":"A3","orientation":"landscape","font_size":8,"margins_mm":{"top":10,"right":8,"bottom":12,"left":8},"revision_marks":true}`))
	if err != nil {
		t.Fatal(err)
	}
	layout.Cover = cover
	return Build(Input{
		Snapshot: cur, Previous: &prev, Format: format, Layout: layout, CustomerName: "Customer AB", RevisionLabel: "-01",
		Status: "DRAFT", Watermark: "UNCONTROLLED COPY", ContentHash: strings.Repeat("ab", 32),
		Slots:   []Slot{{Title: "Process. Eng", Names: []string{"Ani"}}, {Title: "Engineer", Names: []string{"Budi", "Cici"}}, {Title: "QA / QC Mgr.", Names: []string{"Dedi"}}},
		History: []HistoryRow{{No: 1, Date: "2026-11-01", Revision: "-00", Description: "Initial release"}, {No: 2, Date: "", Revision: "-01", Description: "Use SPI"}},
	})
}

func TestBuild(t *testing.T) {
	sh := sample(t, true)
	if sh.Title != "CONTROL PLAN" || len(sh.Columns) != 4 || len(sh.Rows) != 2 {
		t.Fatalf("model: %s, %d kolom, %d baris", sh.Title, len(sh.Columns), len(sh.Rows))
	}
	// Kolom otomatis dari atribut baris, select memakai labelnya, tersembunyi tidak ikut.
	if got := sh.Rows[0].Cells; got[0] != "10" || got[1] != "SMT Printing" || got[3] != "SPI 100%" {
		t.Errorf("isi baris: %v", got)
	}
	// Hanya baris yang berubah sejak revisi 0 diberi tanda ▲1 (R-REV-09).
	if sh.Rows[0].Mark != "▲1" || sh.Rows[1].Mark != "" {
		t.Errorf("tanda revisi: %q %q", sh.Rows[0].Mark, sh.Rows[1].Mark)
	}
	// Slot dengan lebih dari satu kandidat dibiarkan kosong untuk ditulis tangan.
	if sh.Slots[0].Name() != "Ani" || sh.Slots[1].Name() != "" {
		t.Errorf("nama slot: %q %q", sh.Slots[0].Name(), sh.Slots[1].Name())
	}
	if _, err := ParseLayout([]byte(`{"paper":"A5"}`)); err == nil {
		t.Error("kertas A5 diterima")
	}
}

func TestXLSX(t *testing.T) {
	data, err := XLSX(sample(t, true))
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 2 || got[0] != "Cover" || got[1] != "CP" {
		t.Fatalf("sheet: %v", got)
	}
	pl, err := f.GetPageLayout("CP")
	if err != nil || pl.Size == nil || *pl.Size != 8 || pl.Orientation == nil || *pl.Orientation != "landscape" {
		t.Errorf("tata letak halaman: %+v, %v", pl, err)
	}
	hf, _ := f.GetHeaderFooter("CP")
	if !strings.Contains(hf.OddHeader, "UNCONTROLLED COPY") || !strings.Contains(hf.OddFooter, "SC-AB-001 -01 · DRAFT") || !strings.Contains(hf.OddFooter, "Page &P of &N") {
		t.Errorf("header/footer: %+v", hf)
	}
	names := f.GetDefinedName()
	var titles string
	for _, n := range names {
		if n.Name == "_xlnm.Print_Titles" && n.Scope == "CP" {
			titles = n.RefersTo
		}
	}
	if titles == "" {
		t.Error("judul kolom tidak berulang di setiap halaman")
	}
	rows, _ := f.GetRows("CP")
	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, "|") + "\n"
	}
	for _, want := range []string{"CONTROL PLAN", "Refer approval sheet", "▲1", "SPI 100%", "Product Characteristics", "CHARACTERISTICS"} {
		if !strings.Contains(joined, want) {
			t.Errorf("sheet isi tanpa %q", want)
		}
	}
	cover, _ := f.GetRows("Cover")
	coverText := ""
	for _, r := range cover {
		coverText += strings.Join(r, "|") + "\n"
	}
	for _, want := range []string{"Description of Change", "Use SPI", "QA / QC Mgr.", "Dedi", "CUSTOMER APPROVAL", "Customer AB"} {
		if !strings.Contains(coverText, want) {
			t.Errorf("cover tanpa %q", want)
		}
	}
}

func TestHTMLAndPDF(t *testing.T) {
	html, err := HTML(sample(t, true))
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, want := range []string{"size: A3 landscape", "UNCONTROLLED COPY", "<thead>", "Refer approval sheet", "▲1", "CUSTOMER APPROVAL", "Description of Change"} {
		if !strings.Contains(s, want) {
			t.Errorf("HTML tanpa %q", want)
		}
	}
	browser, err := FindBrowser("")
	if err != nil {
		t.Skip("tidak ada browser untuk PDF:", err)
	}
	if testing.Short() {
		t.Skip("PDF dilewati di mode -short")
	}
	pdf, err := PDF(t.Context(), browser, html)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("bukan PDF")
	}
	// A3 landscape kira-kira 1191 x 842 pt.
	box := regexp.MustCompile(`/MediaBox\s*\[\s*0 0 (\d+)`).FindSubmatch(pdf)
	if box == nil || (string(box[1]) != "1190" && string(box[1]) != "1191") {
		t.Errorf("ukuran halaman: %q", box)
	}
	if _, err := QRPNG("http://127.0.0.1:8080/verify?id=x"); err != nil {
		t.Errorf("QR: %v", err)
	}
}
