package submission

// Cetak Form Control Submission (D-207): satu dokumen per form, A4 landscape,
// 30 baris per halaman. Excel ditulis dengan excelize; PDF dari HTML lewat
// Chrome atau Edge yang sama dengan cetak dokumen (R-PRT-03).

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
)

// rowsPerPage adalah jumlah baris satu halaman form.
const rowsPerPage = 30

// formTitle dan formRemark dicetak di setiap form.
const (
	formTitle  = "Form Control Submission Document"
	formRemark = "Each document before migration to MP or from a change request that needs submission and signature by top management and customer is controlled by this form. One form per document: PFD, PFMEA, and Control Plan each have their own form."
)

// File adalah hasil cetak.
type File struct {
	Name        string
	ContentType string
	Body        []byte
}

// printedRow adalah satu baris form dalam bentuk teks.
type printedRow struct {
	No, Revision, Reason, Start, Finish, TopSign, Submit, Result, Remark string
}

// shortDate menulis tanggal seperti di form kertas: 03-Mar-25.
func shortDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02-Jan-06")
}

// printed menyusun baris form, ditambah baris kosong sampai kelipatan 30.
func printed(c Control) []printedRow {
	out := []printedRow{}
	for i, r := range c.Rows {
		start := r.StartedOn
		p := printedRow{
			No: strconv.Itoa(i + 1), Revision: fmt.Sprintf("%02d", r.Revision), Reason: r.Reason,
			Start: shortDate(&start), Finish: shortDate(r.FinishedOn), TopSign: shortDate(r.TopSignOn), Remark: r.Remark,
		}
		switch {
		case !r.Required:
			p.Submit = "Not required"
			p.Result = "-"
		case r.SubmittedOn != nil:
			p.Submit = shortDate(r.SubmittedOn)
		}
		switch r.Result {
		case ResultApproved:
			p.Result = "Approved " + shortDate(r.ResultOn)
		case ResultRejected:
			p.Result = "Rejected " + shortDate(r.ResultOn)
		case ResultWaiting:
			p.Result = "Waiting"
		}
		out = append(out, p)
	}
	for len(out) == 0 || len(out)%rowsPerPage != 0 {
		out = append(out, printedRow{No: strconv.Itoa(len(out) + 1)})
	}
	return out
}

// Export mencetak form satu dokumen sebagai xlsx atau pdf (L-4).
func (s *Service) Export(ctx context.Context, actor access.Actor, documentID, format string) (File, error) {
	if format != "xlsx" && format != "pdf" {
		return File{}, invalid("format", problem.InvalidFormat, "Choose Excel or PDF.")
	}
	c, err := s.Get(ctx, actor, documentID)
	if err != nil {
		return File{}, err
	}
	name := "submission-control-" + c.Number
	if format == "xlsx" {
		body, err := xlsx(c)
		if err != nil {
			return File{}, err
		}
		return File{Name: name + ".xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Body: body}, nil
	}
	browser := s.browser
	if browser == "" {
		if browser, err = printout.FindBrowser(""); err != nil {
			return File{}, invalid("format", problem.NotAllowed, "PDF is not available on this computer: no Chrome or Edge was found. Download Excel instead.")
		}
	}
	page, err := htmlPage(c)
	if err != nil {
		return File{}, err
	}
	body, err := printout.PDF(ctx, browser, page)
	if err != nil {
		return File{}, err
	}
	return File{Name: name + ".pdf", ContentType: "application/pdf", Body: body}, nil
}

// xlsx menulis form ke Excel: header, tabel dengan grup "Submission Document",
// catatan, dan nomor form. Judul tabel diulang di setiap halaman cetak.
func xlsx(c Control) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Submission control"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	border := []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}}
	head, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 9}, Fill: excelize.Fill{Type: "pattern", Color: []string{"D9D9D9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: border})
	cell, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 9}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: border})
	left, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 9}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: border})
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}, Alignment: &excelize.Alignment{Horizontal: "center"}})
	small, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 8}, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
	label, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10}})

	widths := []float64{5, 9, 26, 11, 11, 13, 13, 16, 34}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, w)
	}
	set := func(ref string, v any, style int) {
		_ = f.SetCellValue(sheet, ref, v)
		_ = f.SetCellStyle(sheet, ref, ref, style)
	}
	_ = f.MergeCell(sheet, "A1", "I1")
	set("A1", formTitle, title)
	_ = f.SetRowHeight(sheet, 1, 24)
	for _, h := range []struct{ ref, text string }{
		{"A3", "Customer"}, {"C3", ": " + c.CustomerName + " (" + c.CustomerCode + ")"}, {"F3", "Document type"}, {"G3", ": " + c.DocumentTypeName},
		{"A4", "Model"}, {"C4", ": " + c.Models}, {"F4", "Document number"}, {"G4", ": " + c.Number},
	} {
		set(h.ref, h.text, label)
	}
	// Judul tabel dua baris: grup "Submission Document" di atas empat kolom.
	for _, m := range [][2]string{{"A6", "A7"}, {"B6", "B7"}, {"C6", "C7"}, {"H6", "H7"}, {"I6", "I7"}, {"D6", "G6"}} {
		_ = f.MergeCell(sheet, m[0], m[1])
	}
	for _, h := range []struct{ ref, text string }{
		{"A6", "No"}, {"B6", "Revision"}, {"C6", "Reason"}, {"D6", "Submission Document"}, {"H6", "Customer result"}, {"I6", "Remark"},
		{"D7", "Date of Start"}, {"E7", "Date of Finish"}, {"F7", "Top management sign"}, {"G7", "Submit to customer"},
	} {
		set(h.ref, h.text, head)
	}
	_ = f.SetCellStyle(sheet, "A6", "I7", head)
	_ = f.SetRowHeight(sheet, 7, 24)
	row := 8
	for _, r := range printed(c) {
		values := []string{r.No, r.Revision, r.Reason, r.Start, r.Finish, r.TopSign, r.Submit, r.Result, r.Remark}
		for i, v := range values {
			col, _ := excelize.ColumnNumberToName(i + 1)
			style := cell
			if i == 2 || i == 8 {
				style = left
			}
			set(fmt.Sprintf("%s%d", col, row), v, style)
		}
		row++
	}
	row++
	_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("I%d", row))
	set(fmt.Sprintf("A%d", row), "Remark: "+formRemark, small)
	_ = f.SetRowHeight(sheet, row, 26)
	if c.FormNumber != "" {
		set(fmt.Sprintf("A%d", row+1), c.FormNumber, small)
	}
	// A4 landscape, lebar satu halaman, judul tabel diulang setiap halaman.
	size, orient, one, zero := 9, "landscape", 1, 0
	if err := f.SetPageLayout(sheet, &excelize.PageLayoutOptions{Size: &size, Orientation: &orient, FitToWidth: &one, FitToHeight: &zero}); err != nil {
		return nil, err
	}
	fit := true
	_ = f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fit})
	_ = f.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Titles", RefersTo: "'" + sheet + "'!$6:$7", Scope: sheet})
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var pageTemplate = template.Must(template.New("form").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>{{.Title}}</title>
<style>
@page { size: A4 landscape; margin: 10mm; }
body { font-family: "Times New Roman", serif; font-size: 9pt; color: #000; margin: 0; }
h1 { text-align: center; font-size: 15pt; margin: 0 0 4mm; }
.head { display: grid; grid-template-columns: 30mm 1fr 35mm 1fr; gap: 1mm 2mm; margin-bottom: 3mm; }
table { border-collapse: collapse; width: 100%; table-layout: fixed; }
th, td { border: 0.6pt solid #000; padding: 1mm; text-align: center; vertical-align: middle; height: 5mm; overflow-wrap: anywhere; }
th { background: #d9d9d9; font-weight: normal; }
td.left { text-align: left; }
thead { display: table-header-group; }
tr { page-break-inside: avoid; }
.remark { font-size: 8pt; margin-top: 3mm; }
.form { font-size: 8pt; margin-top: 2mm; }
</style></head><body>
<h1>{{.Title}}</h1>
<div class="head">
<span>Customer</span><span>: {{.C.CustomerName}} ({{.C.CustomerCode}})</span><span>Document type</span><span>: {{.C.DocumentTypeName}}</span>
<span>Model</span><span>: {{.C.Models}}</span><span>Document number</span><span>: {{.C.Number}}</span>
</div>
<table>
<colgroup><col style="width:5%"><col style="width:7%"><col style="width:18%"><col style="width:9%"><col style="width:9%"><col style="width:10%"><col style="width:10%"><col style="width:11%"><col style="width:21%"></colgroup>
<thead>
<tr><th rowspan="2">No</th><th rowspan="2">Revision</th><th rowspan="2">Reason</th><th colspan="4">Submission Document</th><th rowspan="2">Customer result</th><th rowspan="2">Remark</th></tr>
<tr><th>Date of Start</th><th>Date of Finish</th><th>Top management sign</th><th>Submit to customer</th></tr>
</thead>
<tbody>
{{range .Rows}}<tr><td>{{.No}}</td><td>{{.Revision}}</td><td class="left">{{.Reason}}</td><td>{{.Start}}</td><td>{{.Finish}}</td><td>{{.TopSign}}</td><td>{{.Submit}}</td><td>{{.Result}}</td><td class="left">{{.Remark}}</td></tr>
{{end}}</tbody>
</table>
<p class="remark"><strong>Remark:</strong><br>{{.Remark}}</p>
{{if .C.FormNumber}}<p class="form">{{.C.FormNumber}}</p>{{end}}
</body></html>`))

// htmlPage menyusun halaman HTML form untuk dicetak ke PDF.
func htmlPage(c Control) ([]byte, error) {
	var buf bytes.Buffer
	err := pageTemplate.Execute(&buf, struct {
		Title, Remark string
		C             Control
		Rows          []printedRow
	}{formTitle, formRemark, c, printed(c)})
	if err != nil {
		return nil, fmt.Errorf("menyusun halaman form: %w", err)
	}
	return buf.Bytes(), nil
}
