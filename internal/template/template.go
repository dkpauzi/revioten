// Package template berisi format template (Tahap 3, M3.1, UC-20, R-TPL-01,
// R-TPL-02, R-TPL-11). Konsep: struktur dokumen sebagai data yang berversi.
// Setiap field (kolom baris dan field header) punya ID tetap, dan labelnya
// hanya tampilan. Isi baris dokumen (M4.1) memakai ID itu, jadi label bisa
// diganti tanpa merusak data, dan field yang tidak dipakai lagi disembunyikan,
// tidak dihapus.
package template

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Area field: header dokumen, atau kolom baris.
const (
	AreaHeader = "header"
	AreaRow    = "row"
)

// Jenis field. Jenis tidak bisa diubah setelah field dibuat, karena
// mengubahnya merusak isi yang sudah ada.
const (
	KindText   = "text"   // teks satu sel
	KindDate   = "date"   // tanggal YYYY-MM-DD
	KindSelect = "select" // satu pilihan dari daftar milik field
	KindSymbol = "symbol" // satu simbol karakteristik milik versi (R-TPL-11)
	KindRating = "rating" // nilai 1 sampai 10 pada satu skala (severity, occurrence, detection)
	KindRPN    = "rpn"    // dihitung: severity x occurrence x detection dengan set yang sama
	KindAP     = "ap"     // dihitung: Action Priority dari tabel AP versi itu, dengan set yang sama (D-134)

	// Kolom terisi otomatis (D-122, D-123, migrasi 00014). Isinya tidak
	// diketik, tetapi diambil dari baris dokumen, jadi kolomnya selalu tampil
	// dan tercetak seperti di form Excel.
	KindOperationNumber = "operation_number" // nomor operasi baris, misalnya 5.2 (R-CMP-01)
	KindProcess         = "process"          // nama proses baris dari master proses (D-06, R-DOC-04)
	KindProcessFlow     = "process_flow"     // simbol alur langkah itu, dibentuk dari PFD (R-DOC-09)
	// Field header yang diisi dari register nomor, tidak diketik (form set langkah B dan C).
	KindDocumentNumber = "document_number"
)

// Skala rating (R-TPL-01).
const (
	ScaleSeverity   = "severity"
	ScaleOccurrence = "occurrence"
	ScaleDetection  = "detection"
)

// Metode risiko (R-TPL-02): RPN (AIAG 4th), AP (AIAG-VDA, hanya kompatibel
// form), atau tanpa metode risiko (PFD, CP).
const (
	RiskRPN  = "rpn"
	RiskAP   = "ap"
	RiskNone = "none"
)

// Metodologi PFMEA (R-TPL-02). Kosong berarti form yang dipakai sekarang.
const (
	MethodAIAG4th = "aiag_4th"
	MethodAIAGVDA = "aiag_vda"
	MethodSQAM    = "sqam"
	// Form Control Plan manual AIAG 2024 (D-135). Nilai sendiri, supaya CP
	// lama dan CP 2024 bisa dibedakan tanpa melihat nama template.
	MethodAIAGCP2024 = "aiag_cp_2024"
)

// Status versi. Published dipakai mulai M3.4 (R-TPL-05).
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Batas isian. ASUMSI E-70: label 100 karakter, teks satu sel 2000 karakter,
// kriteria skala 500 karakter, dan paling banyak 50 pilihan per field.
const (
	maxNameLength     = 200
	maxLabelLength    = 100
	MaxTextLength     = 2000
	maxCriteriaLength = 500
	maxOptions        = 50
	maxSymbols        = 20
	maxRating         = 10
)

var (
	scales = []string{ScaleSeverity, ScaleOccurrence, ScaleDetection}
	kinds  = []string{KindText, KindDate, KindSelect, KindSymbol, KindRating, KindRPN, KindAP, KindOperationNumber, KindProcess, KindProcessFlow, KindDocumentNumber}
	// filledKinds adalah kolom terisi otomatis: tidak diketik, dan paling
	// banyak satu kolom per jenis dalam satu versi.
	filledKinds = []string{KindOperationNumber, KindProcess, KindProcessFlow}
	// notTyped adalah semua jenis yang isinya tidak pernah diketik: dihitung
	// (RPN, AP) atau diisi sistem (kolom terisi otomatis, nomor dokumen).
	notTyped  = []string{KindRPN, KindAP, KindOperationNumber, KindProcess, KindProcessFlow, KindDocumentNumber}
	methods   = []string{MethodAIAG4th, MethodAIAGVDA, MethodSQAM, MethodAIAGCP2024}
	risks     = []string{RiskRPN, RiskAP, RiskNone}
	areaOrder = map[string]int{AreaHeader: 0, AreaRow: 1}
)

// FormatTemplate adalah satu format template beserta ringkasan versinya.
type FormatTemplate struct {
	ID               string
	Name             string
	DocumentTypeID   string
	DocumentTypeKey  string // pfd, pfmea, atau cp
	DocumentTypeName string
	Methodology      string // "" untuk form yang dipakai sekarang
	CustomerID       string // "" untuk template umum
	CustomerCode     string
	Versions         []VersionSummary // menurut nomor versi, tidak pernah nil
	CreatedAt        time.Time        // selalu UTC
}

// VersionSummary adalah ringkasan satu versi untuk daftar template.
type VersionSummary struct {
	ID          string
	Version     int
	Status      string
	RiskMethod  string
	PublishedAt *time.Time // nil selama draft; selalu UTC (M3.4)
}

// FormatVersion adalah satu versi format template lengkap dengan isinya.
type FormatVersion struct {
	ID         string
	TemplateID string
	Version    int
	Status     string
	RiskMethod string
	// Aturan sorot (R-TPL-11): 0 berarti mati.
	HighlightRPNAbove        int
	HighlightSeverityAtLeast int
	Fields                   []Field  // header dulu, lalu kolom baris, menurut urutan
	Ratings                  []Rating // severity, occurrence, lalu detection, nilai 1 sampai 10
	Symbols                  []Symbol
	APRules                  []APRule // tabel AP; kosong kecuali metode risiko AP (D-134); tidak pernah nil
	// ChangedByPersonID adalah orang yang terakhir mengubah isi versi ini;
	// "" bila belum pernah diubah. Dipakai aturan D-139, tidak dikirim ke API.
	ChangedByPersonID string
	// Persetujuan versi bermetode AP (D-139); kosong untuk metode lain.
	Approvals        []Approver
	PendingApprovers []Approver
	CreatedAt        time.Time  // selalu UTC
	PublishedAt      *time.Time // nil selama draft; selalu UTC (M3.4)
}

// Field adalah satu field header atau kolom baris dalam satu versi.
type Field struct {
	ID          string // tetap lintas versi; isi baris memakai ID ini
	Area        string
	Kind        string
	RatingScale string // untuk KindRating
	RatingSet   int    // untuk KindRating, KindRPN, dan KindAP: 1 nilai sekarang, 2 hasil tindakan
	Key         string // key tetap, misalnya failure_mode; "" untuk field tanpa key (migrasi 00018)
	Group       string // judul grup di atas kolom, misalnya "FAILURE ANALYSIS (STEP 4)"; "" tanpa grup
	Label       string
	Position    int // urutan dalam area, mulai 1
	Required    bool
	Hidden      bool
	Options     []Option // untuk KindSelect
}

// Option adalah satu pilihan field select. Key tetap dan dipakai isi baris;
// label bisa diganti.
type Option struct {
	Key   string
	Label string
}

// Rating adalah kriteria satu nilai pada satu skala.
type Rating struct {
	Scale    string
	Value    int
	Criteria string
}

// Symbol adalah simbol karakteristik, misalnya Critical Process. Key tetap
// dan dipakai isi baris; nama bisa diganti. Automatic menandai simbol yang
// dipasang sistem, misalnya Revision pada baris yang berubah (R-REV-09):
// simbol ini tercetak di legenda, tetapi tidak dipilih tangan di kolom
// simbol. Tanda ini hanya diatur migrasi, tidak lewat layar.
type Symbol struct {
	Key       string
	Name      string
	Automatic bool
}

// NewTemplate adalah isian format template baru. Bila CopyFromVersionID diisi,
// versi 1 menyalin field, metode risiko, aturan sorot, tabel skala, dan
// simbol dari versi itu, dengan ID field baru.
type NewTemplate struct {
	Name              string
	DocumentTypeID    string
	Methodology       string
	CustomerID        string
	RiskMethod        string // diabaikan bila menyalin
	CopyFromVersionID string
}

// NewField adalah isian field baru. Key pilihan dibuat dari labelnya.
type NewField struct {
	Area        string
	Kind        string
	Label       string
	Required    bool
	RatingScale string
	RatingSet   int
	Options     []Option
}

// FieldUpdate adalah isian perubahan field. Area dan jenis tidak bisa diubah.
// Isian nil berarti tetap (D-111), supaya klien yang tidak mengirim hidden
// tidak memunculkan kembali kolom tersembunyi. Options untuk field select:
// semua key lama wajib ada (pilihan tidak bisa dihapus), dan pilihan baru
// tanpa key mendapat key dari labelnya.
type FieldUpdate struct {
	Label    string
	Required *bool
	Hidden   *bool
	Position *int
	Options  []Option
}

// HighlightSettings adalah aturan sorot satu versi; 0 berarti mati, nil
// berarti tetap (D-111).
type HighlightSettings struct {
	RPNAbove        *int
	SeverityAtLeast *int
}

// ContentCheck adalah hasil pemeriksaan isi satu baris atau header.
type ContentCheck struct {
	// Problems berisi nilai yang tidak sah. Field-nya adalah ID field.
	Problems []problem.Problem
	// Missing berisi ID field wajib yang masih kosong. Isi seperti ini tetap
	// sah, tapi belum lengkap (R-TPL-02).
	Missing []string
}

// Valid melaporkan apakah semua nilai sah.
func (c ContentCheck) Valid() bool {
	return len(c.Problems) == 0
}

// CheckContent memeriksa isi satu baris (area row) atau header (area header)
// terhadap versi format (R-DOC-01). content memakai ID field sebagai key,
// seperti hasil JSON dari layar. Field tersembunyi tetap boleh berisi, karena
// datanya disimpan (R-TPL-02).
func CheckContent(v FormatVersion, area string, content map[string]any) ContentCheck {
	var c ContentCheck
	known := map[string]bool{}
	for _, f := range v.Fields {
		if f.Area != area {
			continue
		}
		known[f.ID] = true
		value := content[f.ID]
		if blank(value) {
			// RPN dihitung dan kolom terisi otomatis diambil dari baris, jadi
			// keduanya tidak pernah dianggap kosong.
			if f.Required && !f.Hidden && !slices.Contains(notTyped, f.Kind) {
				c.Missing = append(c.Missing, f.ID)
			}
			continue
		}
		if p := checkValue(f, value, v.Symbols); p != nil {
			c.Problems = append(c.Problems, *p)
		}
	}
	var unknown []string
	for id := range content {
		if !known[id] {
			unknown = append(unknown, id)
		}
	}
	sort.Strings(unknown)
	for _, id := range unknown {
		c.Problems = append(c.Problems, problem.Problem{Field: id, Code: problem.NotAllowed, Message: "This field is not in the format template."})
	}
	return c
}

// blank melaporkan nilai kosong: tidak ada, null, atau teks berisi spasi saja.
func blank(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	}
	return false
}

// checkValue memeriksa satu nilai yang tidak kosong menurut jenis field.
func checkValue(f Field, value any, symbols []Symbol) *problem.Problem {
	bad := func(code, message string) *problem.Problem {
		return &problem.Problem{Field: f.ID, Code: code, Message: message}
	}
	switch f.Kind {
	case KindRPN:
		return bad(problem.NotAllowed, "RPN is calculated from severity, occurrence, and detection.")
	case KindAP:
		return bad(problem.NotAllowed, "The action priority is read from the AP table, not typed.")
	case KindDocumentNumber:
		return bad(problem.NotAllowed, "The document number comes from the register, not typed.")
	case KindOperationNumber:
		return bad(problem.NotAllowed, "The operation number comes from the row, not typed.")
	case KindProcess:
		return bad(problem.NotAllowed, "The process comes from the process master through the row, not typed.")
	case KindProcessFlow:
		return bad(problem.NotAllowed, "The process flow comes from the PFD steps, not typed.")
	case KindRating:
		if n, ok := whole(value); !ok || n < 1 || n > maxRating {
			return bad(problem.InvalidFormat, "Use a whole number from 1 to 10.")
		}
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return bad(problem.InvalidFormat, "Use text.")
	}
	switch f.Kind {
	case KindText:
		if utf8.RuneCountInString(text) > MaxTextLength {
			return bad(problem.TooLong, fmt.Sprintf("Use at most %d characters.", MaxTextLength))
		}
	case KindDate:
		if _, err := time.Parse(time.DateOnly, text); err != nil {
			return bad(problem.InvalidFormat, "Use a date as YYYY-MM-DD.")
		}
	case KindSelect:
		if !slices.ContainsFunc(f.Options, func(o Option) bool { return o.Key == text }) {
			return bad(problem.InvalidFormat, "Choose one of the options of this field.")
		}
	case KindSymbol:
		// ASUMSI E-70: satu sel berisi satu simbol.
		i := slices.IndexFunc(symbols, func(s Symbol) bool { return s.Key == text })
		switch {
		case i < 0:
			return bad(problem.InvalidFormat, "Choose one of the symbols of this format template.")
		case symbols[i].Automatic:
			return bad(problem.NotAllowed, "This symbol is placed by the system on changed rows, not chosen by hand.")
		}
	}
	return nil
}

// whole mengubah angka dari JSON (float64 atau json.Number) atau dari Go
// menjadi bilangan bulat. ok false bila bukan angka bulat.
func whole(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		if v != math.Trunc(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return int(v), true
	case json.Number:
		n, err := strconv.Atoi(v.String())
		return n, err == nil
	}
	return 0, false
}

// RPN menghitung setiap kolom RPN dari severity, occurrence, dan detection
// dengan set yang sama. Kolom RPN yang salah satu nilainya belum sah tidak
// ikut di hasil.
func RPN(v FormatVersion, content map[string]any) map[string]int {
	ratings := map[string]int{} // "skala/set" menjadi nilai
	for _, f := range v.Fields {
		if f.Kind != KindRating {
			continue
		}
		if n, ok := whole(content[f.ID]); ok && n >= 1 && n <= maxRating {
			ratings[fmt.Sprintf("%s/%d", f.RatingScale, f.RatingSet)] = n
		}
	}
	out := map[string]int{}
	for _, f := range v.Fields {
		if f.Kind != KindRPN {
			continue
		}
		s, sOK := ratings[fmt.Sprintf("%s/%d", ScaleSeverity, f.RatingSet)]
		o, oOK := ratings[fmt.Sprintf("%s/%d", ScaleOccurrence, f.RatingSet)]
		d, dOK := ratings[fmt.Sprintf("%s/%d", ScaleDetection, f.RatingSet)]
		if sOK && oOK && dOK {
			out[f.ID] = s * o * d
		}
	}
	return out
}

// AP mencari Action Priority setiap kolom AP dari severity, occurrence, dan
// detection dengan set yang sama, memakai tabel AP versi itu (D-134). Kolom
// AP yang salah satu nilainya belum sah tidak ikut di hasil.
func AP(v FormatVersion, content map[string]any) map[string]string {
	ratings := map[string]int{} // "skala/set" menjadi nilai
	for _, f := range v.Fields {
		if f.Kind != KindRating {
			continue
		}
		if n, ok := whole(content[f.ID]); ok && n >= 1 && n <= maxRating {
			ratings[fmt.Sprintf("%s/%d", f.RatingScale, f.RatingSet)] = n
		}
	}
	out := map[string]string{}
	for _, f := range v.Fields {
		if f.Kind != KindAP {
			continue
		}
		s, sOK := ratings[fmt.Sprintf("%s/%d", ScaleSeverity, f.RatingSet)]
		o, oOK := ratings[fmt.Sprintf("%s/%d", ScaleOccurrence, f.RatingSet)]
		d, dOK := ratings[fmt.Sprintf("%s/%d", ScaleDetection, f.RatingSet)]
		if !sOK || !oOK || !dOK {
			continue
		}
		if priority, ok := ActionPriority(v.APRules, s, o, d); ok {
			out[f.ID] = priority
		}
	}
	return out
}

// Key kolom tindakan di form PFMEA AIAG-VDA. Aturan sorot AP melihat kedua
// kolom ini lewat key-nya, karena labelnya boleh diganti.
const (
	keyPreventionAction = "prevention_action"
	keyDetectionAction  = "detection_action"
)

// apHighWithoutAction melaporkan apakah baris dengan AP H belum punya
// tindakan: Prevention Action dan Detection Action sama-sama kosong. Form
// tanpa kedua kolom itu tidak pernah disorot.
func apHighWithoutAction(v FormatVersion, content map[string]any) bool {
	found := false
	for _, f := range v.Fields {
		if f.Key != keyPreventionAction && f.Key != keyDetectionAction {
			continue
		}
		found = true
		if !blank(content[f.ID]) {
			return false
		}
	}
	return found
}

// Highlights mengembalikan ID field yang disorot menurut aturan sorot versi:
// RPN di atas ambang, dan severity paling sedikit ambang (R-TPL-11). Sorotan
// hanya tanda; isi tetap sah dan tetap bisa disimpan (R-DOC-06).
// ASUMSI E-71: "severity 8" di form berarti 8 ke atas, dan aturan berlaku
// untuk nilai sekarang maupun hasil tindakan.
func Highlights(v FormatVersion, content map[string]any) []string {
	rpn := RPN(v, content)
	ap := AP(v, content)
	noAction := apHighWithoutAction(v, content)
	var out []string
	for _, f := range v.Fields {
		switch {
		// AP H (nilai sekarang) tanpa tindakan: hanya menandai, tidak memblokir (prompt form set, R-TPL-11).
		case f.Kind == KindAP && f.RatingSet == 1:
			if ap[f.ID] == PriorityHigh && noAction {
				out = append(out, f.ID)
			}
		case f.Kind == KindRPN && v.HighlightRPNAbove > 0:
			if n, ok := rpn[f.ID]; ok && n > v.HighlightRPNAbove {
				out = append(out, f.ID)
			}
		case f.Kind == KindRating && f.RatingScale == ScaleSeverity && v.HighlightSeverityAtLeast > 0:
			if n, ok := whole(content[f.ID]); ok && n >= v.HighlightSeverityAtLeast && n <= maxRating {
				out = append(out, f.ID)
			}
		}
	}
	return out
}

// keyFrom membuat key dari label: huruf kecil, huruf dan angka saja dengan
// garis bawah di antaranya, diawali huruf, paling banyak 50 karakter. Key
// yang sudah dipakai mendapat akhiran _2, _3, dan seterusnya. Key dibuat
// sekali; label boleh berubah sesudahnya.
func keyFrom(label string, taken []string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(label) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if underscore && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			underscore = false
		default:
			underscore = true
		}
	}
	key := b.String()
	if key == "" || key[0] < 'a' || key[0] > 'z' {
		key = strings.TrimSuffix("k_"+key, "_")
	}
	if len(key) > 44 {
		key = strings.TrimRight(key[:44], "_")
	}
	candidate := key
	for n := 2; slices.Contains(taken, candidate); n++ {
		candidate = fmt.Sprintf("%s_%d", key, n)
	}
	return candidate
}

// cleanNewField merapikan dan memeriksa isian field baru terhadap versi.
func cleanNewField(v FormatVersion, in NewField) (NewField, []*problem.Problem) {
	var list []*problem.Problem
	add := func(field, code, message string) {
		list = append(list, &problem.Problem{Field: field, Code: code, Message: message})
	}
	switch {
	case in.Area == "":
		add("area", problem.Required, "Choose header field or column.")
	case in.Area != AreaHeader && in.Area != AreaRow:
		add("area", problem.InvalidFormat, "Choose header field or column.")
	}
	kindProblems := len(list)
	switch {
	case in.Kind == "":
		add("kind", problem.Required, "Choose the kind of field.")
	case !slices.Contains(kinds, in.Kind):
		add("kind", problem.InvalidFormat, "Choose a kind from the list.")
	case (in.Kind == KindRating || in.Kind == KindRPN || in.Kind == KindAP || in.Kind == KindSymbol || slices.Contains(filledKinds, in.Kind)) && in.Area == AreaHeader:
		add("kind", problem.NotAllowed, "Ratings, RPN, symbols, and filled columns are columns, not header fields.")
	case in.Kind == KindRating && v.RiskMethod == RiskNone,
		in.Kind == KindRPN && v.RiskMethod != RiskRPN,
		in.Kind == KindAP && v.RiskMethod != RiskAP:
		add("kind", problem.NotAllowed, "The risk method of this format template has no such column.")
	case in.Kind == KindDocumentNumber && in.Area == AreaRow:
		add("kind", problem.NotAllowed, "The document number is a header field, not a column.")
	}
	kindAllowed := len(list) == kindProblems
	label, labelProblem := problem.Text("label", in.Label, true, maxLabelLength)
	list = append(list, labelProblem)
	out := NewField{Area: in.Area, Kind: in.Kind, Label: label, Required: in.Required}

	if in.Kind == KindRating || in.Kind == KindRPN || in.Kind == KindAP {
		if in.RatingSet != 1 && in.RatingSet != 2 {
			add("rating_set", problem.InvalidFormat, "Use set 1 for the current values or 2 for the action results.")
		}
		out.RatingSet = in.RatingSet
	}
	if in.Kind == KindRating {
		switch {
		case in.RatingScale == "":
			add("rating_scale", problem.Required, "Choose severity, occurrence, or detection.")
		case !slices.Contains(scales, in.RatingScale):
			add("rating_scale", problem.InvalidFormat, "Choose severity, occurrence, or detection.")
		}
		out.RatingScale = in.RatingScale
	}
	// Satu kolom per skala dan set, satu kolom RPN per set, supaya RPN dan
	// aturan sorot tahu nilai mana yang dipakai.
	if kindAllowed && (in.Kind == KindRating || in.Kind == KindRPN || in.Kind == KindAP) {
		for _, f := range v.Fields {
			if f.Kind == in.Kind && f.RatingSet == in.RatingSet && f.RatingScale == in.RatingScale {
				field := "rating_scale"
				if in.Kind == KindRPN || in.Kind == KindAP {
					field = "rating_set"
				}
				add(field, problem.Taken, "This format template already has this column.")
				break
			}
		}
	}

	// Kolom terisi otomatis cukup satu per jenis: isinya sama untuk satu baris.
	if kindAllowed && (slices.Contains(filledKinds, in.Kind) || in.Kind == KindDocumentNumber) &&
		slices.ContainsFunc(v.Fields, func(f Field) bool { return f.Kind == in.Kind }) {
		add("kind", problem.Taken, "This format template already has this column.")
	}

	options, optionProblem := cleanOptions(in.Kind, in.Options, nil)
	list = append(list, optionProblem)
	out.Options = options
	return out, list
}

// fieldChange adalah hasil akhir perubahan field: isian yang tidak dikirim
// sudah diisi nilai lama.
type fieldChange struct {
	Label    string
	Required bool
	Hidden   bool
	Position int
	Options  []Option
}

// cleanFieldUpdate merapikan dan memeriksa perubahan field terhadap versi,
// lalu menggabungkannya dengan nilai lama field itu.
func cleanFieldUpdate(v FormatVersion, f Field, in FieldUpdate) (fieldChange, []*problem.Problem) {
	label, labelProblem := problem.Text("label", in.Label, true, maxLabelLength)
	list := []*problem.Problem{labelProblem}
	out := fieldChange{Label: label, Required: f.Required, Hidden: f.Hidden, Position: f.Position, Options: f.Options}
	if in.Required != nil {
		out.Required = *in.Required
	}
	if in.Hidden != nil {
		out.Hidden = *in.Hidden
	}
	if in.Position != nil {
		out.Position = *in.Position
	}
	if out.Hidden && out.Required {
		list = append(list, &problem.Problem{Field: "required", Code: problem.NotAllowed, Message: "A hidden field cannot be required."})
	}
	count := 0
	for _, other := range v.Fields {
		if other.Area == f.Area {
			count++
		}
	}
	if out.Position < 1 || out.Position > count {
		list = append(list, &problem.Problem{Field: "position", Code: problem.InvalidFormat,
			Message: fmt.Sprintf("Use a position from 1 to %d.", count)})
	}
	switch {
	case in.Options == nil:
	case f.Kind != KindSelect:
		list = append(list, &problem.Problem{Field: "options", Code: problem.NotAllowed, Message: "Only select fields have options."})
	default:
		options, p := cleanOptions(f.Kind, in.Options, f.Options)
		list = append(list, p)
		out.Options = options
	}
	return out, list
}

// cleanOptions memeriksa pilihan field select. old berisi pilihan sebelum
// diubah: semua key lama wajib tetap ada, karena isi baris bisa memakainya
// (ASUMSI E-70: pilihan dan simbol tidak bisa dihapus, hanya diganti nama).
// Pilihan tanpa key mendapat key dari labelnya.
func cleanOptions(kind string, options, old []Option) ([]Option, *problem.Problem) {
	bad := func(code, message string) ([]Option, *problem.Problem) {
		return nil, &problem.Problem{Field: "options", Code: code, Message: message}
	}
	if kind != KindSelect {
		if len(options) > 0 {
			return bad(problem.NotAllowed, "Only select fields have options.")
		}
		return nil, nil
	}
	switch {
	case len(options) == 0:
		return bad(problem.Required, "Add the options of this field.")
	case len(options) > maxOptions:
		return bad(problem.TooLong, fmt.Sprintf("Use at most %d options.", maxOptions))
	}
	var keys, labels []string
	out := make([]Option, 0, len(options))
	for _, o := range options {
		label, p := problem.Text("options", o.Label, true, maxLabelLength)
		if p != nil {
			return nil, p
		}
		if slices.Contains(labels, strings.ToLower(label)) {
			return bad(problem.Taken, "Two options have the same label.")
		}
		labels = append(labels, strings.ToLower(label))
		key := o.Key
		if key != "" && !slices.ContainsFunc(old, func(prev Option) bool { return prev.Key == key }) {
			return bad(problem.InvalidFormat, "An option has an unknown key.")
		}
		if slices.Contains(keys, key) && key != "" {
			return bad(problem.Taken, "Two options have the same key.")
		}
		keys = append(keys, key)
		out = append(out, Option{Key: key, Label: label})
	}
	for _, prev := range old {
		if !slices.Contains(keys, prev.Key) {
			return bad(problem.NotAllowed, "Options cannot be removed, because rows may use them. Rename them instead.")
		}
	}
	// Key baru dibuat sesudah semua key lama dikenal, supaya tidak bentrok.
	for i := range out {
		if out[i].Key == "" {
			out[i].Key = keyFrom(out[i].Label, keys)
			keys = append(keys, out[i].Key)
		}
	}
	return out, nil
}

// cleanSymbols memeriksa daftar simbol baru terhadap simbol lama: semua key
// lama wajib tetap ada (simbol tidak bisa dihapus), dan simbol tanpa key
// mendapat key dari namanya.
func cleanSymbols(symbols, old []Symbol) ([]Symbol, *problem.Problem) {
	bad := func(code, message string) ([]Symbol, *problem.Problem) {
		return nil, &problem.Problem{Field: "symbols", Code: code, Message: message}
	}
	if len(symbols) > maxSymbols {
		return bad(problem.TooLong, fmt.Sprintf("Use at most %d symbols.", maxSymbols))
	}
	var keys, names []string
	out := make([]Symbol, 0, len(symbols))
	for _, s := range symbols {
		name, p := problem.Text("symbols", s.Name, true, maxLabelLength)
		if p != nil {
			return nil, p
		}
		if slices.Contains(names, strings.ToLower(name)) {
			return bad(problem.Taken, "Two symbols have the same name.")
		}
		names = append(names, strings.ToLower(name))
		// Tanda otomatis diambil dari simbol lama, bukan dari isian; simbol
		// baru tidak pernah otomatis (R-REV-09).
		automatic := false
		if s.Key != "" {
			i := slices.IndexFunc(old, func(prev Symbol) bool { return prev.Key == s.Key })
			if i < 0 {
				return bad(problem.InvalidFormat, "A symbol has an unknown key.")
			}
			automatic = old[i].Automatic
		}
		keys = append(keys, s.Key)
		out = append(out, Symbol{Key: s.Key, Name: name, Automatic: automatic})
	}
	for _, prev := range old {
		if !slices.Contains(keys, prev.Key) {
			return bad(problem.NotAllowed, "Symbols cannot be removed, because rows may use them. Rename them instead.")
		}
	}
	for i := range out {
		if out[i].Key == "" {
			out[i].Key = keyFrom(out[i].Name, keys)
			keys = append(keys, out[i].Key)
		}
	}
	return out, nil
}

// cleanRatings memeriksa perubahan teks kriteria skala.
func cleanRatings(v FormatVersion, ratings []Rating) ([]Rating, *problem.Problem) {
	if v.RiskMethod == RiskNone {
		return nil, &problem.Problem{Field: "ratings", Code: problem.NotAllowed, Message: "This format template has no risk method, so it has no rating tables."}
	}
	out := make([]Rating, 0, len(ratings))
	for _, r := range ratings {
		if !slices.Contains(scales, r.Scale) || r.Value < 1 || r.Value > maxRating {
			return nil, &problem.Problem{Field: "ratings", Code: problem.InvalidFormat,
				Message: "Use severity, occurrence, or detection with a value from 1 to 10."}
		}
		criteria, p := problem.Text("ratings", r.Criteria, false, maxCriteriaLength)
		if p != nil {
			return nil, p
		}
		out = append(out, Rating{Scale: r.Scale, Value: r.Value, Criteria: criteria})
	}
	return out, nil
}

// cleanHighlights menggabungkan aturan sorot baru dengan yang lama (nil
// berarti tetap), lalu memeriksanya terhadap metode risiko versi.
func cleanHighlights(v FormatVersion, in HighlightSettings) (rpnAbove, severityAtLeast int, p *problem.Problem) {
	rpnAbove, severityAtLeast = v.HighlightRPNAbove, v.HighlightSeverityAtLeast
	if in.RPNAbove != nil {
		rpnAbove = *in.RPNAbove
	}
	if in.SeverityAtLeast != nil {
		severityAtLeast = *in.SeverityAtLeast
	}
	switch {
	case rpnAbove != 0 && v.RiskMethod != RiskRPN:
		p = &problem.Problem{Field: "highlight_rpn_above", Code: problem.NotAllowed, Message: "Only RPN format templates have an RPN rule."}
	case rpnAbove < 0 || rpnAbove > 999:
		p = &problem.Problem{Field: "highlight_rpn_above", Code: problem.InvalidFormat, Message: "Use 0 (off) or 1 to 999."}
	case severityAtLeast != 0 && v.RiskMethod == RiskNone:
		p = &problem.Problem{Field: "highlight_severity_at_least", Code: problem.NotAllowed, Message: "This format template has no severity."}
	case severityAtLeast < 0 || severityAtLeast > maxRating:
		p = &problem.Problem{Field: "highlight_severity_at_least", Code: problem.InvalidFormat, Message: "Use 0 (off) or 1 to 10."}
	}
	return rpnAbove, severityAtLeast, p
}

// sortFields mengurutkan field: header dulu, lalu kolom baris, menurut urutan.
func sortFields(fields []Field) {
	slices.SortStableFunc(fields, func(a, b Field) int {
		if d := areaOrder[a.Area] - areaOrder[b.Area]; d != 0 {
			return d
		}
		return a.Position - b.Position
	})
}

// Typed melaporkan apakah isi kolom berjenis kind diketik orang. Kolom yang
// dihitung (RPN, AP) atau diisi sistem (nomor operasi, proses, Process Flow,
// nomor dokumen) tidak pernah diketik (D-122, D-123). Dipakai API integrasi
// supaya aplikasi lain tahu kolom mana yang boleh diisi.
func Typed(kind string) bool {
	return !slices.Contains(notTyped, kind)
}
