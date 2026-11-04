package template

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// pfmea adalah versi format kecil untuk test tanpa database: satu field
// header, dan kolom baris yang mewakili setiap jenis.
func pfmea() FormatVersion {
	return FormatVersion{
		ID: "v1", RiskMethod: RiskRPN, HighlightRPNAbove: 100, HighlightSeverityAtLeast: 8,
		Fields: []Field{
			{ID: "h-date", Area: AreaHeader, Kind: KindDate, Label: "Key Date", Position: 1},
			{ID: "f-mode", Area: AreaRow, Kind: KindText, Label: "Potential Failure Mode", Position: 1, Required: true},
			{ID: "f-s", Area: AreaRow, Kind: KindRating, RatingScale: ScaleSeverity, RatingSet: 1, Label: "Severity", Position: 2},
			{ID: "f-class", Area: AreaRow, Kind: KindSymbol, Label: "Classification", Position: 3},
			{ID: "f-o", Area: AreaRow, Kind: KindRating, RatingScale: ScaleOccurrence, RatingSet: 1, Label: "Occurrence", Position: 4},
			{ID: "f-d", Area: AreaRow, Kind: KindRating, RatingScale: ScaleDetection, RatingSet: 1, Label: "Detection", Position: 5},
			{ID: "f-rpn", Area: AreaRow, Kind: KindRPN, RatingSet: 1, Label: "RPN", Position: 6},
			{ID: "f-s2", Area: AreaRow, Kind: KindRating, RatingScale: ScaleSeverity, RatingSet: 2, Label: "Result Severity", Position: 7},
			{ID: "f-o2", Area: AreaRow, Kind: KindRating, RatingScale: ScaleOccurrence, RatingSet: 2, Label: "Result Occurrence", Position: 8},
			{ID: "f-d2", Area: AreaRow, Kind: KindRating, RatingScale: ScaleDetection, RatingSet: 2, Label: "Result Detection", Position: 9},
			{ID: "f-rpn2", Area: AreaRow, Kind: KindRPN, RatingSet: 2, Label: "Result RPN", Position: 10},
			{ID: "f-flow", Area: AreaRow, Kind: KindSelect, Label: "Flow Symbol", Position: 11,
				Options: []Option{{Key: "process", Label: "Process"}, {Key: "decision", Label: "Decision"}}},
			{ID: "f-old", Area: AreaRow, Kind: KindText, Label: "Old column", Position: 12, Hidden: true},
			// Kolom terisi otomatis (D-122, D-123): tidak diketik, jadi wajib pun
			// tidak pernah dianggap kosong.
			{ID: "f-no", Area: AreaRow, Kind: KindOperationNumber, Label: "Process Item No.", Position: 13},
			{ID: "f-proc", Area: AreaRow, Kind: KindProcess, Label: "Process Step / Function", Position: 14, Required: true},
			{ID: "f-pflow", Area: AreaRow, Kind: KindProcessFlow, Label: "Process Flow", Position: 15},
		},
		Symbols: []Symbol{
			{Key: "critical_process", Name: "Critical Process"},
			{Key: "revision", Name: "Revision", Automatic: true},
			{Key: "regulation", Name: "Regulation"},
		},
	}
}

// decode meniru isi baris yang datang sebagai JSON dari layar.
func decode(t *testing.T, text string) map[string]any {
	t.Helper()
	var content map[string]any
	if err := json.Unmarshal([]byte(text), &content); err != nil {
		t.Fatal(err)
	}
	return content
}

func codes(c ContentCheck) []string {
	var out []string
	for _, p := range c.Problems {
		out = append(out, p.Field+":"+p.Code)
	}
	return out
}

// TestCheckContent: isi baris diperiksa per ID field, sesuai jenisnya (R-DOC-01).
func TestCheckContent(t *testing.T) {
	v := pfmea()
	good := decode(t, `{"f-mode": "Solder bridge", "f-s": 9, "f-class": "critical_process", "f-o": 3, "f-d": 4,
		"f-flow": "decision", "f-old": "data lama tetap sah"}`)
	if c := CheckContent(v, AreaRow, good); !c.Valid() || len(c.Missing) != 0 {
		t.Errorf("isi yang benar: %+v", c)
	}

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"field tidak dikenal", `{"f-mode": "x", "f-lain": "y"}`, []string{"f-lain:not_allowed"}},
		{"field header di baris", `{"f-mode": "x", "h-date": "2026-10-31"}`, []string{"h-date:not_allowed"}},
		{"rating bukan angka bulat", `{"f-mode": "x", "f-s": 9.5}`, []string{"f-s:invalid_format"}},
		{"rating di luar 1 sampai 10", `{"f-mode": "x", "f-o": 11, "f-d": 0}`, []string{"f-o:invalid_format", "f-d:invalid_format"}},
		{"rating sebagai teks", `{"f-mode": "x", "f-s": "9"}`, []string{"f-s:invalid_format"}},
		{"RPN dihitung, tidak diisi", `{"f-mode": "x", "f-rpn": 108}`, []string{"f-rpn:not_allowed"}},
		{"nomor operasi dari baris", `{"f-mode": "x", "f-no": "5.2"}`, []string{"f-no:not_allowed"}},
		{"proses dari master proses", `{"f-mode": "x", "f-proc": "Solder Paste Printing"}`, []string{"f-proc:not_allowed"}},
		{"process flow dari PFD", `{"f-mode": "x", "f-pflow": "decision"}`, []string{"f-pflow:not_allowed"}},
		{"pilihan tidak ada", `{"f-mode": "x", "f-flow": "merge"}`, []string{"f-flow:invalid_format"}},
		{"simbol tidak ada", `{"f-mode": "x", "f-class": "important_character"}`, []string{"f-class:invalid_format"}},
		// R-REV-09: tanda revisi dipasang sistem dari diff, tidak dipilih tangan.
		{"simbol otomatis", `{"f-mode": "x", "f-class": "revision"}`, []string{"f-class:not_allowed"}},
		{"teks bukan string", `{"f-mode": 5}`, []string{"f-mode:invalid_format"}},
		{"teks terlalu panjang", `{"f-mode": "` + strings.Repeat("a", MaxTextLength+1) + `"}`, []string{"f-mode:too_long"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codes(CheckContent(v, AreaRow, decode(t, tt.content))); !slices.Equal(got, tt.want) {
				t.Errorf("CheckContent = %v, ingin %v", got, tt.want)
			}
		})
	}

	// Field wajib yang kosong tidak membuat isi tidak sah; isinya dilaporkan
	// belum lengkap (R-TPL-02: kolom wajib baru menandai data yang belum lengkap).
	c := CheckContent(v, AreaRow, decode(t, `{"f-s": 5, "f-mode": "  "}`))
	if !c.Valid() || !slices.Equal(c.Missing, []string{"f-mode"}) {
		t.Errorf("field wajib kosong: %+v", c)
	}

	// Field header: tanggal berbentuk YYYY-MM-DD.
	if got := codes(CheckContent(v, AreaHeader, decode(t, `{"h-date": "31/10/2026"}`))); !slices.Equal(got, []string{"h-date:invalid_format"}) {
		t.Errorf("tanggal salah: %v", got)
	}
	if c := CheckContent(v, AreaHeader, decode(t, `{"h-date": "2026-10-31"}`)); !c.Valid() {
		t.Errorf("tanggal benar ditolak: %+v", c)
	}
}

// TestRPNAndHighlights: RPN dihitung dari severity, occurrence, dan detection
// dengan set yang sama; aturan sorot hanya menandai, tidak menolak (R-TPL-11).
func TestRPNAndHighlights(t *testing.T) {
	v := pfmea()
	content := decode(t, `{"f-mode": "x", "f-s": 9, "f-o": 3, "f-d": 4, "f-s2": 5, "f-o2": 2}`)
	rpn := RPN(v, content)
	if len(rpn) != 1 || rpn["f-rpn"] != 108 {
		t.Errorf("RPN = %v, ingin hanya f-rpn 108 (set 2 belum lengkap)", rpn)
	}
	if got := Highlights(v, content); !slices.Equal(got, []string{"f-s", "f-rpn"}) {
		t.Errorf("sorot = %v, ingin RPN di atas 100 dan severity 9", got)
	}
	if got := Highlights(v, decode(t, `{"f-s": 7, "f-o": 2, "f-d": 7}`)); len(got) != 0 {
		t.Errorf("RPN 98 dan severity 7 tidak disorot: %v", got)
	}
	// Severity hasil tindakan juga disorot; ambang 0 berarti aturan mati.
	if got := Highlights(v, decode(t, `{"f-s2": 8}`)); !slices.Equal(got, []string{"f-s2"}) {
		t.Errorf("severity hasil 8: %v", got)
	}
	v.HighlightRPNAbove, v.HighlightSeverityAtLeast = 0, 0
	if got := Highlights(v, content); len(got) != 0 {
		t.Errorf("aturan mati tetap menyorot: %v", got)
	}
}

// TestKeyFrom: key pilihan dan simbol dibuat dari label, sekali saja.
func TestKeyFrom(t *testing.T) {
	tests := []struct {
		label string
		taken []string
		want  string
	}{
		{"Critical Process", nil, "critical_process"},
		{"  NG / Scrap (A) ", nil, "ng_scrap_a"},
		{"Process", []string{"process"}, "process_2"},
		{"Process", []string{"process", "process_2"}, "process_3"},
		{"5S", nil, "k_5s"},
		{"—", nil, "k"},
	}
	for _, tt := range tests {
		if got := keyFrom(tt.label, tt.taken); got != tt.want {
			t.Errorf("keyFrom(%q, %v) = %q, ingin %q", tt.label, tt.taken, got, tt.want)
		}
	}
}
