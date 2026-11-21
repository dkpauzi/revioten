package printout

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/template"
)

// Judul cetakan per jenis dokumen, seperti di form asli.
var titles = map[string]string{"pfd": "PROCESS FLOW DIAGRAM", "pfmea": "PROCESS FMEA", "cp": "CONTROL PLAN"}

// Input adalah semua bahan satu cetakan.
type Input struct {
	Snapshot      document.Snapshot  // isi yang dicetak: snapshot revisi terbit, atau isi draft
	Previous      *document.Snapshot // revisi terbit sebelumnya, untuk tanda revisi; nil untuk revisi pertama
	Format        template.FormatVersion
	Layout        Layout
	CustomerName  string
	RevisionLabel string // misalnya -01
	Status        string // DRAFT, EFFECTIVE, atau SUPERSEDED (M6.9)
	Watermark     string // UNCONTROLLED COPY, atau CONTROLLED COPY No. n (M6.4)
	ContentHash   string // SHA-256 isi yang dicetak (M6.5, M6.9)
	ReleaseNote   string // catatan rilis pengganti tanda tangan di PDF sistem (M6.9); "" untuk draft
	ReleaseDate   string // tanggal rilis (D-59), YYYY-MM-DD; "" bila belum
	Slots         []Slot
	History       []HistoryRow
	QRPNG         []byte // gambar QR (M6.5); nil bila QR mati
	VerifyURL     string
	// Steps: langkah PFD satu set menurut nomor langkah, untuk kolom Process
	// Flow CP (D-122, C3). Untuk PFD diisi Build dari barisnya sendiri.
	Steps map[string]Step
}

// Slot adalah satu kotak tanda tangan dari rute approval (R-APR-08, R-APR-17).
type Slot struct {
	Title string   // judul cetak, misalnya "QA / QC Mgr."
	Stage string   // prepared, checked, atau approved
	Names []string // pemegang jabatan; satu nama tercetak, lebih dari satu dibiarkan kosong untuk ditulis tangan
	// Signed: "Signed digitally <tanggal>" untuk slot yang sudah ditandatangani
	// digital (D-198); kosong untuk tanda tangan basah.
	Signed string
}

// Name memberi nama yang dicetak di slot: satu pemegang, atau kosong.
func (s Slot) Name() string {
	if len(s.Names) == 1 {
		return s.Names[0]
	}
	return ""
}

// HistoryRow adalah satu baris tabel riwayat revisi (R-REV-04, M6.8).
type HistoryRow struct {
	No          int
	Date        string
	Revision    string
	Description string
}

// Pair adalah satu isian header: label dan nilai.
type Pair struct {
	Label string
	Value string
}

// Column adalah satu kolom tabel cetak.
type Column struct {
	Label string
	Group string  // judul grup di atas kolom; "" tanpa grup
	Width float64 // lebar relatif
}

// Row adalah satu baris tabel cetak. Section berisi judul bagian per PCB
// (D-155) untuk baris pemisah; baris pemisah tidak punya sel.
type Row struct {
	Section string
	Mark    string // tanda revisi, misalnya ▲1 (R-REV-09)
	Cells   []string
}

// Sheet adalah model cetak netral.
type Sheet struct {
	Input
	Title   string
	Number  string
	Header  []Pair
	Columns []Column
	Rows    []Row
	// Ops: nomor langkah setiap baris Rows ("" untuk baris judul bagian).
	Ops []string
	// Processes: nama proses setiap baris Rows, untuk menggabung sel proses.
	Processes []string
	// FlowColumn: kolom tempat alur digambar (Process Flow di CP, Flow Symbol
	// di PFD); -1 bila format tidak punya. OpColumn dan ProcessColumn: kolom
	// nomor langkah dan proses yang digabung per langkah; -1 bila tidak ada.
	FlowColumn, OpColumn, ProcessColumn int
}

// Build membentuk model cetak dari bahan cetakan.
func Build(in Input) Sheet {
	s := in.Snapshot
	sh := Sheet{Input: in, Title: titles[s.DocumentType], Number: s.Number}
	if sh.Title == "" {
		sh.Title = strings.ToUpper(s.DocumentType)
	}
	sh.Header = append(sh.Header,
		Pair{"Customer", strings.TrimSpace(in.CustomerName + " (" + s.CustomerCode + ")")},
		Pair{"Model Name", strings.Join(s.Models, ", ")},
	)
	if len(s.PartNumbers) > 0 {
		sh.Header = append(sh.Header, Pair{"Part Number", strings.Join(s.PartNumbers, ", ")})
	}
	sh.Header = append(sh.Header, Pair{"Document No.", s.Number}, Pair{"Revision", in.RevisionLabel})
	if in.ReleaseDate != "" {
		sh.Header = append(sh.Header, Pair{"Release Date", in.ReleaseDate})
	}
	for _, f := range in.Format.Fields {
		if f.Area != template.AreaHeader || f.Hidden {
			continue
		}
		sh.Header = append(sh.Header, Pair{f.Label, cellText(in.Format, f, s.Header[f.ID], s.Header, document.SnapshotRow{}, s)})
	}

	var fields []template.Field
	for _, f := range in.Format.Fields {
		if f.Area == template.AreaRow && !f.Hidden {
			fields = append(fields, f)
		}
	}
	sh.FlowColumn, sh.OpColumn, sh.ProcessColumn = -1, -1, -1
	symbolField, ngField := "", ""
	for i, f := range fields {
		sh.Columns = append(sh.Columns, Column{Label: f.Label, Group: f.Group, Width: in.Layout.Width(f.Key, f.ID, defaultWidth(f))})
		switch {
		case f.Kind == template.KindProcessFlow:
			sh.FlowColumn = i
		case s.DocumentType == "pfd" && f.Key == "flow_symbol":
			sh.FlowColumn = i
			symbolField = f.ID
		case f.Kind == template.KindOperationNumber:
			sh.OpColumn = i
		case f.Kind == template.KindProcess:
			sh.ProcessColumn = i
		}
		if s.DocumentType == "pfd" && f.Key == "ng_disposition" {
			ngField = f.ID
		}
	}
	// PFD menggambar alurnya sendiri dari barisnya (R-DOC-09).
	if s.DocumentType == "pfd" && symbolField != "" {
		sh.Steps = StepsOf(s.Rows, symbolField, ngField)
	}

	changed := marks(in)
	sections := map[string]string{}
	for _, sec := range s.Sections {
		sections[sec.ID] = sec.Name
	}
	current := ""
	for _, r := range s.Rows {
		if r.SectionID != current && len(s.Sections) > 0 {
			current = r.SectionID
			sh.Rows = append(sh.Rows, Row{Section: sections[r.SectionID]})
			sh.Ops = append(sh.Ops, "")
			sh.Processes = append(sh.Processes, "")
		}
		sh.Ops = append(sh.Ops, r.OperationNumber)
		sh.Processes = append(sh.Processes, r.ProcessName)
		row := Row{Mark: changed[r.ID], Cells: make([]string, 0, len(fields))}
		for _, f := range fields {
			row.Cells = append(row.Cells, cellText(in.Format, f, r.Content[f.ID], r.Content, r, s))
		}
		sh.Rows = append(sh.Rows, row)
	}
	return sh
}

// StepsOf membaca langkah PFD menurut nomor langkahnya (R-DOC-09): simbol dari
// field Flow Symbol dan arah NG dari field NG Disposition.
func StepsOf(rows []document.SnapshotRow, symbolField, ngField string) map[string]Step {
	steps := map[string]Step{}
	for _, r := range rows {
		if r.OperationNumber == "" {
			continue
		}
		steps[r.OperationNumber] = Step{Symbol: text(r.Content[symbolField]), NG: text(r.Content[ngField])}
	}
	return steps
}

// defaultWidth memberi lebar bawaan menurut jenis kolom. Kolom alur cukup
// lebar untuk simbol dan cabang NG (C3).
func defaultWidth(f template.Field) float64 {
	if f.Kind == template.KindProcessFlow || f.Key == "flow_symbol" {
		return 1.3
	}
	switch f.Kind {
	case template.KindOperationNumber, template.KindRating, template.KindRPN, template.KindAP, template.KindSymbol:
		return 0.5
	case template.KindProcess:
		return 1.6
	case template.KindDate:
		return 0.8
	}
	return 1.4
}

// marks memberi tanda revisi untuk baris yang baru atau berubah sejak revisi
// terbit sebelumnya (R-REV-09, D-121). Hanya tanda revisi yang dicetak.
func marks(in Input) map[string]string {
	out := map[string]string{}
	if !in.Layout.RevisionMarks || in.Previous == nil {
		return out
	}
	before := map[string]document.SnapshotRow{}
	for _, r := range in.Previous.Rows {
		before[r.ID] = r
	}
	label := "▲" + strconv.Itoa(in.Snapshot.Revision)
	for _, r := range in.Snapshot.Rows {
		o, ok := before[r.ID]
		if !ok || !sameJSON(o.Content, r.Content) || o.ProcessID != r.ProcessID || o.OperationNumber != r.OperationNumber {
			out[r.ID] = label
		}
	}
	return out
}

func sameJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// cellText menulis isi satu sel untuk cetakan. Kolom terisi otomatis diambil
// dari atribut baris (D-122, D-123); RPN dan AP dihitung dari nilai baris.
func cellText(format template.FormatVersion, f template.Field, v any, content map[string]any, r document.SnapshotRow, s document.Snapshot) string {
	switch f.Kind {
	case template.KindProcess:
		return r.ProcessName
	case template.KindOperationNumber:
		return r.OperationNumber
	case template.KindDocumentNumber:
		return s.Number
	case template.KindRPN:
		if product, ok := ratingProduct(format, f.RatingSet, content); ok {
			return strconv.Itoa(product)
		}
		return ""
	case template.KindAP:
		sev, occ, det, ok := ratings(format, f.RatingSet, content)
		if !ok {
			return ""
		}
		p, _ := template.ActionPriority(format.APRules, sev, occ, det)
		return p
	case template.KindSelect:
		key := text(v)
		for _, o := range f.Options {
			if o.Key == key {
				return o.Label
			}
		}
		return key
	case template.KindSymbol:
		key := text(v)
		for _, sym := range format.Symbols {
			if sym.Key == key {
				return sym.Name
			}
		}
		return key
	}
	return text(v)
}

// ratings membaca severity, occurrence, dan detection satu set nilai.
func ratings(format template.FormatVersion, set int, content map[string]any) (int, int, int, bool) {
	got := map[string]int{}
	for _, f := range format.Fields {
		if f.Area != template.AreaRow || f.Kind != template.KindRating || f.RatingSet != set {
			continue
		}
		if n, ok := number(content[f.ID]); ok {
			got[f.RatingScale] = n
		}
	}
	s, ok1 := got["severity"]
	o, ok2 := got["occurrence"]
	d, ok3 := got["detection"]
	return s, o, d, ok1 && ok2 && ok3
}

func ratingProduct(format template.FormatVersion, set int, content map[string]any) (int, bool) {
	s, o, d, ok := ratings(format, set, content)
	return s * o * d, ok
}

func number(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		return n, err == nil
	}
	return 0, false
}

// text menulis satu nilai sebagai teks; angka bulat tanpa desimal.
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "Yes"
		}
		return "No"
	case []any:
		parts := make([]string, 0, len(x))
		for _, p := range x {
			parts = append(parts, text(p))
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}
