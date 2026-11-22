package printout

// HTML untuk PDF (M6.3). Model cetak yang sama dengan Excel ditulis sebagai
// HTML dengan @page sesuai layout, judul tabel yang berulang di setiap
// halaman (thead), watermark tetap di setiap halaman (M6.4), dan footer
// berisi nomor, revisi, status, hash, dan catatan rilis (M6.9).

import (
	"bytes"
	"encoding/base64"
	"html/template"

	"rsc.io/qr"
)

// QRPNG membuat gambar QR berisi alamat halaman verifikasi (M6.5).
func QRPNG(url string) ([]byte, error) {
	code, err := qr.Encode(url, qr.M)
	if err != nil {
		return nil, err
	}
	code.Scale = 4
	return code.PNG(), nil
}

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"pct":  func(w, total float64) float64 { return w / total * 100 },
	"mod4": func(i int) int { return i % 4 },
	"mod3": func(i int) int { return i % 3 },
	"b64":  func(b []byte) string { return base64.StdEncoding.EncodeToString(b) },
	"short": func(s string) string {
		if len(s) > 16 {
			return s[:16]
		}
		return s
	},
}).Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Number}} {{.RevisionLabel}}</title>
<style>
@page { size: {{.Layout.Paper}} {{.Layout.Orientation}}; margin: {{.Layout.MarginsMM.Top}}mm {{.Layout.MarginsMM.Right}}mm {{.Layout.MarginsMM.Bottom}}mm {{.Layout.MarginsMM.Left}}mm; }
* { box-sizing: border-box; }
body { margin: 0; font-family: Arial, Helvetica, sans-serif; font-size: {{.Layout.FontSize}}pt; color: #000; }
h1 { font-size: {{.Layout.FontSize}}pt; font-size: 1.8em; margin: 0 0 2mm; }
table { border-collapse: collapse; width: 100%; table-layout: fixed; }
th, td { border: 0.5pt solid #000; padding: 1pt 2pt; vertical-align: top; overflow-wrap: anywhere; white-space: pre-wrap; }
th { background: #d9d9d9; text-align: center; vertical-align: middle; }
thead { display: table-header-group; }
tr { break-inside: avoid; }
.head td.l { font-weight: bold; width: 14%; }
.slots td { text-align: center; }
.slots .sign { height: 12mm; font-style: italic; vertical-align: middle; }
.section td { font-weight: bold; background: #f2f2f2; }
.mark { text-align: center; font-weight: bold; }
.cover { break-after: page; }
.top { display: flex; justify-content: space-between; align-items: flex-start; }
.wm { position: fixed; top: 42%; left: 0; right: 0; text-align: center; font-size: 64pt; font-weight: bold; color: rgba(192, 0, 0, 0.14); transform: rotate(-18deg); pointer-events: none; }
.foot { position: fixed; bottom: -7mm; left: 0; right: 0; font-size: 7pt; display: flex; justify-content: space-between; }
.status { font-weight: bold; }
.customer { margin-top: 4mm; font-weight: bold; text-align: center; }
.approval { height: 22mm; }
.sheet { position: relative; overflow: hidden; break-after: page; }
.sheet:last-child { break-after: auto; }
.sheet .headbox { overflow: hidden; }
.sheet .tw { position: relative; }
.sheet td { padding: 0; }
.sheet td .c { overflow: hidden; padding: 1pt 2pt; }
.sheet th { padding: 0; }
.sheet th .h { overflow: hidden; padding: 1pt 2pt; display: flex; align-items: center; justify-content: center; }
.sheet td.flowcell { border-top: none; border-bottom: none; }
.sheet td.merged { font-weight: bold; }
.flowbox { position: absolute; }
.flowbox svg { display: block; }
.legendbox svg, .diagrambox svg { display: block; }
</style></head><body>
{{if .Watermark}}<div class="wm">{{.Watermark}}</div>{{end}}
<div class="foot"><span>{{.Number}} {{.RevisionLabel}} · <span class="status">{{.Status}}</span>{{if .ReleaseNote}} · {{.ReleaseNote}}{{end}}</span><span>SHA-256 {{short .ContentHash}}</span></div>
{{if .Layout.Cover}}
<section class="cover">
  <div class="top"><h1>{{.Title}}</h1><div><b>{{.Number}} {{.RevisionLabel}}</b>{{if .QRPNG}}<br><img alt="QR" src="data:image/png;base64,{{b64 .QRPNG}}" style="width:24mm"><br><small>{{.VerifyURL}}</small>{{end}}</div></div>
  <table class="head">{{range .Header}}<tr><td class="l">{{.Label}}</td><td>{{.Value}}</td></tr>{{end}}</table>
  <p></p>
  <table><colgroup><col style="width:6%"><col style="width:14%"><col style="width:12%"><col></colgroup>
    <thead><tr><th>No</th><th>Date</th><th>Revision</th><th>Description of Change</th></tr></thead>
    <tbody>{{range .History}}<tr><td>{{.No}}</td><td>{{.Date}}</td><td>{{.Revision}}</td><td>{{.Description}}</td></tr>{{end}}</tbody>
  </table>
  <p></p>
  {{if .Slots}}<table class="slots"><tr>{{range .Slots}}<th>{{.Title}}</th>{{end}}</tr><tr>{{range .Slots}}<td class="sign">{{.Signed}}</td>{{end}}</tr><tr>{{range .Slots}}<td>{{.Name}}</td>{{end}}</tr></table>{{end}}
  <p></p>
  <table><tr><th>CUSTOMER APPROVAL</th></tr><tr><td class="approval"></td></tr></table>
  <div class="customer">{{.CustomerName}}</div>
</section>
{{end}}
{{range $pi, $pg := .Pages}}<section class="sheet" style="height:{{$.Plan.ContentH}}mm">
<div class="headbox" style="height:{{$.Plan.HeadMM}}mm">
<div class="top"><h1>{{$.Title}}</h1><b>{{$.Number}} {{$.RevisionLabel}}</b></div>
<table class="head"><tr>{{range $i, $p := $.Header}}{{if and $i (eq (mod3 $i) 0)}}</tr><tr>{{end}}<td class="l">{{$p.Label}}</td><td>{{$p.Value}}</td>{{end}}</tr></table>
{{if $.Slots}}<table class="slots" style="margin-top:2mm"><tr>{{range $.Slots}}<th>{{.Title}}</th>{{end}}</tr><tr>{{range $.Slots}}<td class="sign">{{if $.Layout.Cover}}Refer approval sheet{{else}}{{.Signed}}{{end}}</td>{{end}}</tr><tr>{{range $.Slots}}<td>{{.Name}}</td>{{end}}</tr></table>{{end}}
</div>
{{if $pg.Diagram}}<div class="legendbox" style="height:{{$.Plan.LegendMM}}mm">{{$.Legend}}</div>
<div class="diagrambox">{{$pg.Diagram}}</div>
{{else}}<div class="tw">
<table>
  <colgroup>{{if $.Layout.RevisionMarks}}<col style="width:{{$.Plan.MarkMM}}mm">{{end}}{{range $.Plan.ColumnsMM}}<col style="width:{{.}}mm">{{end}}</colgroup>
  <thead>
    {{if $.Groups}}<tr>{{if $.Layout.RevisionMarks}}<th><div class="h" style="height:{{$.Plan.GroupRowMM}}mm"></div></th>{{end}}{{range $.Groups}}<th colspan="{{.Span}}"><div class="h" style="height:{{$.Plan.GroupRowMM}}mm">{{.Title}}</div></th>{{end}}</tr>{{end}}
    <tr>{{if $.Layout.RevisionMarks}}<th><div class="h" style="height:{{$.Plan.LabelRowMM}}mm">Rev</div></th>{{end}}{{range $.Columns}}<th><div class="h" style="height:{{$.Plan.LabelRowMM}}mm">{{.Label}}</div></th>{{end}}</tr>
  </thead>
  <tbody>
  {{range $pg.Rows}}{{if .Section}}<tr class="section" style="height:{{.HeightMM}}mm"><td colspan="{{$.ColumnCount}}"><div class="c">{{.Section}}</div></td></tr>{{else}}<tr style="height:{{.HeightMM}}mm">{{if $.Layout.RevisionMarks}}<td class="mark"><div class="c">{{.Mark}}</div></td>{{end}}{{range .Cells}}{{if not .Skip}}<td{{if gt .Rowspan 1}} rowspan="{{.Rowspan}}"{{end}}{{if .Flow}} class="flowcell"{{else if .Merged}} class="merged"{{end}}><div class="c" style="max-height:{{.HeightMM}}mm">{{.Text}}</div></td>{{end}}{{end}}</tr>{{end}}{{end}}
  </tbody>
</table>
{{if $pg.Flow}}<div class="flowbox" style="left:{{$.FlowLeftMM}}mm;top:{{$.Plan.TheadMM}}mm">{{$pg.Flow}}</div>{{end}}
</div>{{end}}
</section>
{{end}}
</body></html>`))

// group adalah satu judul grup kolom dengan lebarnya.
type group struct {
	Title string
	Span  int
}

type htmlView struct {
	Sheet
	TotalWidth  float64
	ColumnCount int
	Groups      []group
	Plan        planView
	Pages       []htmlPage
	FlowLeftMM  float64
	Legend      template.HTML // legenda diagram PFD
}

// planView adalah rencana halaman ditambah tinggi area isi.
type planView struct {
	Plan
	ContentH float64
}

// htmlCell adalah satu sel isi. Skip: tertutup sel gabungan di atasnya.
type htmlCell = PageCell

// PageCell adalah satu sel isi halaman. Skip: tertutup sel gabungan di atasnya.
type PageCell struct {
	Text     string
	Rowspan  int
	Skip     bool
	Merged   bool
	Flow     bool
	HeightMM float64
}

type htmlRow = PageRow

// PageRow adalah satu baris halaman.
type PageRow struct {
	Section  string
	Mark     string
	HeightMM float64
	Cells    []htmlCell
}

type htmlPage = PageContent

// PageContent adalah isi satu halaman dan SVG alurnya.
type PageContent struct {
	Rows []htmlRow
	Flow template.HTML
	// Diagram: SVG satu halaman diagram PFD (bagian B); kosong untuk tabel.
	Diagram template.HTML
}

// pagesOf menyusun isi setiap halaman: sel nomor langkah digabung per
// langkah, sel proses per langkah dan proses (seperti CP Tokairika), kolom
// alur dikosongkan karena digambar, dan SVG alur halaman itu.
func pagesOf(sh Sheet, plan Plan) []htmlPage {
	var pages []htmlPage
	if len(plan.Diagram) > 0 {
		for _, d := range plan.Diagram {
			pages = append(pages, htmlPage{Diagram: template.HTML(DiagramSVG(d, plan.DiagramW, plan.DiagramH))})
		}
		return pages
	}
	for _, pg := range plan.Pages {
		page := htmlPage{}
		tops := make([]float64, len(pg.Rows))
		y := 0.0
		for i, r := range pg.Rows {
			tops[i] = y
			y += plan.RowsMM[r]
		}
		same := func(i, j, col int) bool {
			a, b := pg.Rows[i], pg.Rows[j]
			if sh.Ops[a] == "" || sh.Ops[a] != sh.Ops[b] || sh.Rows[a].Cells == nil || sh.Rows[b].Cells == nil {
				return false
			}
			return col == sh.OpColumn || sh.Processes[a] == sh.Processes[b]
		}
		for i, r := range pg.Rows {
			row := sh.Rows[r]
			hr := htmlRow{Section: row.Section, Mark: row.Mark, HeightMM: plan.RowsMM[r]}
			if row.Cells == nil {
				hr.Section = row.Section + " "
			}
			for c, text := range row.Cells {
				cell := htmlCell{Text: text, Rowspan: 1, HeightMM: plan.RowsMM[r]}
				if c == sh.FlowColumn {
					cell.Flow, cell.Text = true, ""
				}
				if c == sh.OpColumn || c == sh.ProcessColumn {
					cell.Merged = true
					if i > 0 && same(i-1, i, c) {
						cell.Skip = true
					} else {
						for j := i + 1; j < len(pg.Rows) && same(i, j, c); j++ {
							cell.Rowspan++
							cell.HeightMM += plan.RowsMM[pg.Rows[j]]
						}
					}
				}
				hr.Cells = append(hr.Cells, cell)
			}
			page.Rows = append(page.Rows, hr)
		}
		if sh.FlowColumn >= 0 {
			page.Flow = template.HTML(FlowSVG(pg, tops, y, plan.ColumnsMM[sh.FlowColumn]))
		}
		pages = append(pages, page)
	}
	return pages
}

// HTML menulis model cetak sebagai halaman HTML siap cetak.
func HTML(sh Sheet) ([]byte, error) {
	v := htmlView{Sheet: sh, ColumnCount: len(sh.Columns)}
	if sh.Layout.RevisionMarks {
		v.TotalWidth = 0.35
		v.ColumnCount++
	}
	hasGroups := false
	for _, c := range sh.Columns {
		v.TotalWidth += c.Width
		hasGroups = hasGroups || c.Group != ""
	}
	if hasGroups {
		for i := 0; i < len(sh.Columns); {
			j := i
			for j+1 < len(sh.Columns) && sh.Columns[j+1].Group == sh.Columns[i].Group {
				j++
			}
			v.Groups = append(v.Groups, group{Title: sh.Columns[i].Group, Span: j - i + 1})
			i = j + 1
		}
	}
	if v.TotalWidth == 0 {
		v.TotalWidth = 1
	}
	plan := BuildPlan(sh)
	v.Plan = planView{Plan: plan, ContentH: plan.PaperH - sh.Layout.MarginsMM.Top - sh.Layout.MarginsMM.Bottom}
	v.Pages = pagesOf(sh, plan)
	if len(plan.Diagram) > 0 {
		v.Legend = template.HTML(DiagramLegendSVG(plan.DiagramW))
	}
	v.FlowLeftMM = plan.MarkMM
	for i := 0; i < sh.FlowColumn && i < len(plan.ColumnsMM); i++ {
		v.FlowLeftMM += plan.ColumnsMM[i]
	}
	var buf bytes.Buffer
	if err := page.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
