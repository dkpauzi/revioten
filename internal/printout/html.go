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
  {{if .Slots}}<table class="slots"><tr>{{range .Slots}}<th>{{.Title}}</th>{{end}}</tr><tr>{{range .Slots}}<td class="sign"></td>{{end}}</tr><tr>{{range .Slots}}<td>{{.Name}}</td>{{end}}</tr></table>{{end}}
  <p></p>
  <table><tr><th>CUSTOMER APPROVAL</th></tr><tr><td class="approval"></td></tr></table>
  <div class="customer">{{.CustomerName}}</div>
</section>
{{end}}
<div class="top"><h1>{{.Title}}</h1><b>{{.Number}} {{.RevisionLabel}}</b></div>
<table class="head"><tr>{{range $i, $p := .Header}}{{if and $i (eq (mod4 $i) 0)}}</tr><tr>{{end}}<td class="l">{{$p.Label}}</td><td>{{$p.Value}}</td>{{end}}</tr></table>
{{if .Slots}}<table class="slots" style="margin-top:2mm"><tr>{{range .Slots}}<th>{{.Title}}</th>{{end}}</tr><tr>{{range .Slots}}<td class="sign">{{if $.Layout.Cover}}Refer approval sheet{{end}}</td>{{end}}</tr><tr>{{range .Slots}}<td>{{.Name}}</td>{{end}}</tr></table>{{end}}
<table style="margin-top:2mm">
  <colgroup>{{if .Layout.RevisionMarks}}<col style="width:{{pct 0.35 .TotalWidth}}%">{{end}}{{range .Columns}}<col style="width:{{pct .Width $.TotalWidth}}%">{{end}}</colgroup>
  <thead>
    {{if .Groups}}<tr>{{if .Layout.RevisionMarks}}<th></th>{{end}}{{range .Groups}}<th colspan="{{.Span}}">{{.Title}}</th>{{end}}</tr>{{end}}
    <tr>{{if .Layout.RevisionMarks}}<th>Rev</th>{{end}}{{range .Columns}}<th>{{.Label}}</th>{{end}}</tr>
  </thead>
  <tbody>
  {{range .Rows}}{{if .Cells}}<tr>{{if $.Layout.RevisionMarks}}<td class="mark">{{.Mark}}</td>{{end}}{{range .Cells}}<td>{{.}}</td>{{end}}</tr>{{else}}<tr class="section"><td colspan="{{$.ColumnCount}}">{{.Section}}</td></tr>{{end}}{{end}}
  </tbody>
</table>
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
	var buf bytes.Buffer
	if err := page.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
