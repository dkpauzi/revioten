package printout

// Kepala lembar dan cover seperti dokumen Tokairika TR-020 (bagian C, E-144):
//
//   - CP (SC-TR-020-04): logo dan nama perusahaan, judul CONTROL PLAN, tabel
//     persetujuan Department / Name / Signature / Date di kanan; Control Plan
//     No., Product, Core Team di kiri; nomor dan nama part besar di tengah;
//     Remark, Customer Name, dan Rev. Description di bawah.
//   - PFMEA (SF-TR-020-03): nama customer besar, judul tiga baris, lalu Part
//     Name, Part No., Model, Process, Process Responsibility, Key Date,
//     Revision, FMEA NO., Prepared by, Core Team, dan Remark.
//   - PFD (SH-TR-020-02): DOC. NO, judul "Process Flow Chart for <model>
//     Unit", lalu CUSTOMER, MODEL, PART NAME, PART NUMBER, PROCESS LINE.
//   - Cover ketiganya: logo, "<jenis> No :", judul, isian customer sampai
//     Process or Line, Document Revision History, INHOUSE APPROVAL (Prepared
//     by, Checked by, Approved by dengan nama dan jabatan), CUSTOMER APPROVAL,
//     dan nama customer.
//
// HTML kepala dipakai PDF dan pratinjau (head_html), jadi keduanya sama.
// Excel menulis susunan yang sama dengan sel gabungan (xlsxhead.go).

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"math"
	"strings"
)

// FormHead adalah isian kepala lembar dan cover.
type FormHead struct {
	Kind           string // pfd, pfmea, cp
	Company        string
	Customer       string
	Models         string
	PartNames      string
	PartNumbers    string
	ProcessLine    string
	CoreTeam       string
	Remark         string
	Responsibility string
	KeyDate        string
	Revision       string // tanpa tanda hubung, misalnya 04
	PreparedBy     string // nama di slot Prepared
}

// formHead menyusun isian kepala dari bahan cetakan. Field header dibaca
// lewat key tetapnya (migrasi 00056), bukan labelnya.
func formHead(sh Sheet) FormHead {
	h := FormHead{
		Kind: sh.Snapshot.DocumentType, Company: sh.CompanyName, Customer: sh.CustomerName,
		Models: strings.Join(sh.Snapshot.Models, ", "), PartNumbers: strings.Join(sh.Snapshot.PartNumbers, ", "),
		Revision: strings.TrimPrefix(sh.RevisionLabel, "-"),
	}
	var names []string
	for _, n := range sh.Snapshot.PartNames {
		if n != "" {
			names = append(names, n)
		}
	}
	h.PartNames = strings.Join(names, ", ")
	for _, f := range sh.Format.Fields {
		if f.Area != "header" || f.Key == "" {
			continue
		}
		v := text(sh.Snapshot.Header[f.ID])
		switch f.Key {
		case "process_line":
			h.ProcessLine = v
		case "core_team":
			h.CoreTeam = v
		case "remark":
			h.Remark = v
		case "process_responsibility":
			h.Responsibility = v
		case "key_date":
			h.KeyDate = v
		}
	}
	for _, s := range sh.Slots {
		if s.Stage == "prepared" {
			h.PreparedBy = s.Name()
		}
	}
	return h
}

// coverTitles: judul cover dan label nomor per jenis dokumen.
var coverTitles = map[string][2]string{
	"cp":    {"Process Control Plan (PCP)", "PCP No :"},
	"pfmea": {"Process Failure Mode and Effects Analysis (P-FMEA)", "PFMEA No :"},
	"pfd":   {"Process Flow Diagram (PFD)", "PFD No :"},
}

// stageLabel: judul kolom slot di INHOUSE APPROVAL.
func stageLabel(stage string) string {
	switch stage {
	case "prepared":
		return "Prepared by"
	case "checked":
		return "Checked by"
	}
	return "Approved by"
}

// department: isi kolom Department di tabel persetujuan kepala CP.
func department(s Slot) string {
	switch s.Stage {
	case "prepared":
		return "Prepared by"
	case "checked":
		return "Checked by"
	}
	return s.Title
}

// cpSlotRows: jumlah baris tabel persetujuan kepala CP.
func cpSlotRows(sh Sheet) int { return max(5, len(sh.Slots)) }

// headMM memberi tinggi kepala lembar menurut jenis dokumen.
func headMM(sh Sheet) float64 {
	switch sh.Snapshot.DocumentType {
	case "cp":
		return 12 + 5*float64(cpSlotRows(sh)) + 10 + 2
	case "pfmea":
		return 16 + 6*5 + 2
	case "pfd":
		return 12 + 5*5 + 2
	}
	h := 9 + math.Ceil(float64(len(sh.Header))/3)*6
	if len(sh.Slots) > 0 {
		h += 20
	}
	return h
}

// headSlot adalah satu baris tabel persetujuan kepala CP.
type headSlot struct {
	Department, Name, Signature, Date string
}

type headView struct {
	FormHead
	Number    string
	Title     string
	CoverNo   string
	Logo      template.URL
	Slots     []headSlot // CP: dari atas (penyetuju terakhir) ke bawah (Prepared)
	CoreSpan  int
	Cover     []Slot
	History   []HistoryRow
	QR        template.URL
	VerifyURL string
	Release   string
}

func pngURL(b []byte) template.URL {
	if len(b) == 0 {
		return ""
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(b))
}

func newHeadView(sh Sheet) headView {
	v := headView{FormHead: formHead(sh), Number: sh.Number, Logo: pngURL(sh.Logo), Cover: sh.Slots, History: sh.History,
		QR: pngURL(sh.QRPNG), VerifyURL: sh.VerifyURL, Release: sh.ReleaseNote}
	t := coverTitles[sh.Snapshot.DocumentType]
	v.Title, v.CoverNo = t[0], t[1]
	if v.Title == "" {
		v.Title = sh.Title
	}
	n := cpSlotRows(sh)
	v.CoreSpan = n - 2
	v.Slots = make([]headSlot, n)
	for i := range sh.Slots {
		s := sh.Slots[len(sh.Slots)-1-i]
		row := headSlot{Department: department(s), Name: s.Name(), Signature: s.Signed}
		if sh.Layout.Cover {
			row.Signature, row.Date = "Refer approval sheet", "Refer approval sheet"
		}
		v.Slots[i] = row
	}
	return v
}

var headTemplates = template.Must(template.New("heads").Funcs(template.FuncMap{"stage": stageLabel}).Parse(`
{{define "cp"}}<table class="fh fh-cp">
<colgroup><col style="width:11%"><col style="width:20%"><col style="width:33%"><col style="width:10%"><col style="width:14%"><col style="width:6%"><col style="width:6%"></colgroup>
<tr style="height:6mm"><td colspan="2" rowspan="2" class="company">{{if .Logo}}<img src="{{.Logo}}" alt="logo">{{end}}<span>{{.Company}}</span></td><td rowspan="2" class="ftitle">CONTROL PLAN</td><th rowspan="2">Department</th><th rowspan="2">Name</th><th colspan="2">Signature</th></tr>
<tr style="height:6mm"><th>Date</th><th>Signature</th></tr>
{{range $i, $s := .Slots}}<tr style="height:5mm">
{{if eq $i 0}}<td class="l">Control Plan No.</td><td>{{$.Number}}</td><td rowspan="{{len $.Slots}}" class="big">{{$.PartNumbers}} {{$.PartNames}}</td>
{{else if eq $i 1}}<td class="l">Product</td><td>{{$.PartNames}}</td>
{{else if eq $i 2}}<td class="l" rowspan="{{$.CoreSpan}}">Core Team</td><td rowspan="{{$.CoreSpan}}" class="core">{{$.CoreTeam}}</td>{{end}}
<td class="l">{{$s.Department}}</td><td>{{$s.Name}}</td><td class="small">{{$s.Date}}</td><td class="small">{{$s.Signature}}</td></tr>
{{end}}<tr style="height:5mm"><td class="l">Customer Name</td><td>{{.Customer}}</td><td rowspan="2" class="remark"><b>Remark :</b> {{.Remark}}</td><td colspan="4" class="nob"></td></tr>
<tr style="height:5mm"><td class="l">Rev. Description</td><td>{{.Revision}}</td><td colspan="4" class="nob"></td></tr>
</table>{{end}}
{{define "pfmea"}}<table class="fh fh-pfmea plain">
<colgroup><col style="width:12%"><col style="width:22%"><col style="width:16%"><col style="width:20%"><col style="width:12%"><col style="width:18%"></colgroup>
<tr style="height:16mm"><td colspan="2" class="cust">{{.Customer}}</td><td colspan="3" class="ftitle3">POTENTIAL<br>PROCESS FAILURE MODE AND EFFECTS ANALYSIS<br>( PROCESS FMEA )</td><td></td></tr>
<tr style="height:5mm"><td>Part Name</td><td>{{.PartNames}}</td><td>Process Responsibility</td><td>{{.Responsibility}}</td><td>FMEA NO.</td><td>{{.Number}}</td></tr>
<tr style="height:5mm"><td>Part No.</td><td>{{.PartNumbers}}</td><td>Key Date</td><td>{{.KeyDate}}</td><td>Prepared by :</td><td>{{.PreparedBy}}</td></tr>
<tr style="height:5mm"><td>Model</td><td>{{.Models}}</td><td>Revision</td><td>{{.Revision}}</td><td></td><td></td></tr>
<tr style="height:5mm"><td>Process</td><td colspan="5">{{.ProcessLine}}</td></tr>
<tr style="height:5mm"><td>Core Team</td><td colspan="5">: {{.CoreTeam}}</td></tr>
<tr style="height:5mm"><td>Remark</td><td colspan="5">: {{.Remark}}</td></tr>
</table>{{end}}
{{define "pfd"}}<table class="fh fh-pfd plain">
<colgroup><col style="width:18%"><col style="width:57%"><col style="width:10%"><col style="width:15%"></colgroup>
<tr style="height:12mm"><td colspan="3" class="ftitle">Process Flow Chart for {{.Models}} Unit</td><td class="docno">DOC. NO {{.Number}}</td></tr>
<tr style="height:5mm"><td class="l">CUSTOMER</td><td colspan="3">{{.Customer}}</td></tr>
<tr style="height:5mm"><td class="l">MODEL</td><td colspan="3">{{.Models}}</td></tr>
<tr style="height:5mm"><td class="l">PART NAME</td><td colspan="3">{{.PartNames}}</td></tr>
<tr style="height:5mm"><td class="l">PART NUMBER</td><td colspan="3">{{.PartNumbers}}</td></tr>
<tr style="height:5mm"><td class="l">PROCESS LINE</td><td colspan="3">{{.ProcessLine}}</td></tr>
</table>{{end}}
{{define "cover"}}<div class="cv">
<div class="cv-top">{{if .Logo}}<img class="cv-logo" src="{{.Logo}}" alt="logo">{{else}}<span></span>{{end}}<div class="cv-no"><b>{{.CoverNo}}</b> <b>{{.Number}}</b>{{if .QR}}<br><img alt="QR" src="{{.QR}}" style="width:22mm"><br><small>{{.VerifyURL}}</small>{{end}}</div></div>
<h1 class="cv-title">{{.Title}}</h1>
<table class="cv-fields">
<tr><td class="cv-l">Customer</td><td class="cv-c">:</td><td>{{.Customer}}</td></tr>
<tr><td class="cv-l">Model Name</td><td class="cv-c">:</td><td>{{.Models}}</td></tr>
<tr><td class="cv-l">Part Name</td><td class="cv-c">:</td><td>{{.PartNames}}</td></tr>
<tr><td class="cv-l">Part Number</td><td class="cv-c">:</td><td>{{.PartNumbers}}</td></tr>
<tr><td class="cv-l">Process or Line</td><td class="cv-c">:</td><td>{{.ProcessLine}}</td></tr>
</table>
<p class="cv-h">Document Revision History</p>
<table class="cv-grid"><colgroup><col style="width:6%"><col style="width:14%"><col style="width:10%"><col></colgroup>
<tr><th>No</th><th>Date</th><th>Revision</th><th>Description of Change</th></tr>
{{range .History}}<tr><td class="c">{{.No}}</td><td class="c">{{.Date}}</td><td class="c">{{.Revision}}</td><td class="desc">{{.Description}}</td></tr>{{end}}
</table>
{{if .Cover}}<p class="cv-h">INHOUSE APPROVAL</p>
<table class="cv-grid cv-sign">
<tr>{{range .Cover}}<th>{{.Stage | stage}}</th>{{end}}</tr>
<tr class="cv-space">{{range .Cover}}<td class="c">{{.Signed}}</td>{{end}}</tr>
<tr>{{range .Cover}}<td class="c">{{.Name}}</td>{{end}}</tr>
<tr>{{range .Cover}}<th>{{.Title}}</th>{{end}}</tr>
</table>{{end}}
<table class="cv-grid cv-cust"><tr><th>CUSTOMER APPROVAL</th></tr><tr><td class="cv-box"></td></tr></table>
{{if .Release}}<p class="cv-release">{{.Release}}</p>{{end}}
<p class="cv-customer">{{.Customer}}</p>
</div>{{end}}
`))

// HeadHTML menulis kepala lembar satu cetakan. Jenis lain memakai kepala
// umum di template halaman (string kosong).
func HeadHTML(sh Sheet) template.HTML {
	kind := sh.Snapshot.DocumentType
	if _, ok := coverTitles[kind]; !ok {
		return ""
	}
	var buf bytes.Buffer
	if err := headTemplates.ExecuteTemplate(&buf, kind, newHeadView(sh)); err != nil {
		return ""
	}
	return template.HTML(buf.String())
}

// CoverHTML menulis cover satu cetakan.
func CoverHTML(sh Sheet) template.HTML {
	var buf bytes.Buffer
	if err := headTemplates.ExecuteTemplate(&buf, "cover", newHeadView(sh)); err != nil {
		return ""
	}
	return template.HTML(buf.String())
}

// headCSS: gaya kepala dan cover, dipakai PDF dan pratinjau.
const headCSS = `
.fh { width: 100%; border-collapse: collapse; table-layout: fixed; }
.fh td, .fh th { border: 0.5pt solid #000; padding: 0 2pt; vertical-align: middle; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; background: none; font-weight: normal; text-align: left; }
.fh th { text-align: center; font-weight: bold; }
.fh .l { font-weight: bold; }
.fh .company { white-space: nowrap; font-weight: bold; font-size: 1.6em; text-align: right; }
.fh .company img { height: 9mm; vertical-align: middle; float: left; }
.fh .company span { line-height: 11mm; }
.fh .ftitle { text-align: center; font-weight: bold; font-size: 2em; }
.fh .big { text-align: center; font-weight: bold; font-size: 2.2em; white-space: normal; }
.fh .core, .fh .remark { white-space: normal; vertical-align: top; }
.fh .small { font-size: 0.8em; text-align: center; }
.fh .nob { border: none; }
.fh.plain td { border: none; }
.fh .cust { font-weight: bold; font-size: 2.2em; vertical-align: top; white-space: normal; }
.fh .ftitle3 { text-align: center; font-weight: bold; font-size: 1.3em; white-space: normal; }
.fh .docno { text-align: right; vertical-align: top; }
.cv { font-size: 10pt; }
.cv-top { display: flex; justify-content: space-between; align-items: flex-start; }
.cv-logo { height: 14mm; }
.cv-no { text-align: right; }
.cv-title { text-align: center; font-size: 20pt; margin: 10mm 0 8mm; }
.cv-fields { border-collapse: collapse; margin: 0 0 6mm 18mm; font-size: 13pt; }
.cv-fields td { border: none; padding: 1.5mm 2mm; }
.cv-l { font-weight: bold; width: 48mm; }
.cv-c { font-weight: bold; width: 6mm; text-align: center; }
.cv-h { text-align: center; font-weight: bold; font-size: 11pt; margin: 4mm 0 1mm; }
.cv-grid { width: 100%; border-collapse: collapse; table-layout: fixed; }
.cv-grid td, .cv-grid th { border: 0.5pt solid #000; padding: 1mm 2mm; background: none; }
.cv-grid th { text-align: center; font-weight: bold; }
.cv-grid .c { text-align: center; }
.cv-grid .desc { white-space: pre-wrap; }
.cv-sign .cv-space td { height: 18mm; font-style: italic; }
.cv-cust { margin-top: 6mm; }
.cv-box { height: 36mm; }
.cv-release { text-align: center; font-size: 8pt; margin: 2mm 0 0; }
.cv-customer { text-align: center; font-weight: bold; font-size: 11pt; margin-top: 4mm; }
`
