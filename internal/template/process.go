package template

// Process template atau pustaka proses (Tahap 3, M3.2, UC-21, R-TPL-01,
// R-TPL-04, D-64). Isinya proses, sub-proses, dan poin, masing-masing dengan
// ID tetap. Nama sub-proses dan teks poin hanya tampilan, jadi bisa diganti
// tanpa memutus rujukan; yang tidak dipakai lagi disembunyikan, tidak dihapus.
//
// Contoh dari brief bagian 3: proses IQC punya sub-proses IQC General, IQC
// MSD, IQC PCB, IQC Wireharness, dan IQC Connector. Setiap sub-proses punya
// poin untuk PFD, PFMEA, dan CP.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Batas isian process template. ASUMSI E-73: teks poin satu baris paling
// banyak 500 karakter, paling banyak 50 sub-proses per versi, dan paling
// banyak 100 poin per sub-proses untuk satu jenis dokumen. Nama sub-proses
// paling banyak 200 karakter, sama dengan nama proses (D-110).
const (
	maxSubprocessName = 200
	maxPointText      = 500
	maxSubprocesses   = 50
	maxPointsPerList  = 100
)

// ProcessRef adalah proses master yang dirujuk process template.
type ProcessRef struct {
	ID   string
	Name string // nama tampilan yang berlaku sekarang (D-06)
}

// DocumentTypeRef adalah jenis dokumen satu poin: PFD, PFMEA, atau CP.
type DocumentTypeRef struct {
	ID   string
	Key  string // pfd, pfmea, atau cp
	Name string
}

// ProcessTemplate adalah process template satu proses beserta ringkasan
// versinya. Satu template per proses (D-129, keputusan E-72); cakupan ada di
// setiap poin (M3.3).
type ProcessTemplate struct {
	ID        string
	Process   ProcessRef
	Versions  []ProcessVersionSummary // menurut nomor versi, tidak pernah nil
	CreatedAt time.Time               // selalu UTC
}

// ProcessVersionSummary adalah ringkasan satu versi untuk daftar template.
type ProcessVersionSummary struct {
	ID          string
	Version     int
	Status      string
	PublishedAt *time.Time // nil selama draft; selalu UTC (M3.4)
}

// ProcessVersion adalah satu versi process template lengkap dengan isinya.
type ProcessVersion struct {
	ID           string
	TemplateID   string
	Process      ProcessRef
	Version      int
	Status       string
	Subprocesses []Subprocess // menurut urutan, tidak pernah nil
	CreatedAt    time.Time    // selalu UTC
	PublishedAt  *time.Time   // nil selama draft; selalu UTC (M3.4)
}

// Subprocess adalah satu sub-proses dalam satu versi beserta poinnya.
type Subprocess struct {
	ID       string // tetap lintas versi (ASUMSI E-33)
	Name     string
	Position int // urutan dalam versi, mulai 1
	Hidden   bool
	Points   []Point // menurut jenis dokumen (PFD, PFMEA, CP), lalu urutan; tidak pernah nil
}

// Point adalah satu poin: satu butir setingkat baris untuk satu jenis
// dokumen (R-TPL-04), beserta cakupan dan variannya (M3.3, D-129).
type Point struct {
	ID           string // tetap lintas versi; baris dokumen nanti menunjuk ID ini
	DocumentType DocumentTypeRef
	Text         string
	Position     int // urutan dalam sub-proses untuk jenis dokumen yang sama, mulai 1
	Hidden       bool
	Scope        Scope // biasanya dipakai untuk: umum, customer, group model, atau model
	// Content adalah isi poin per kolom format, dengan key tetap kolom sebagai
	// kuncinya (D-146); tidak pernah nil. Text tetap menjadi judul poin.
	Content  map[string]any
	Variants []Variant // dari cakupan terluas ke tersempit, tidak pernah nil
}

// SubprocessUpdate adalah isian perubahan sub-proses. Nama wajib dikirim;
// isian nil berarti tetap (D-111).
type SubprocessUpdate struct {
	Name     string
	Position *int
	Hidden   *bool
}

// NewPoint adalah isian poin baru. Sub-proses, jenis dokumen, dan cakupannya
// tetap setelah poin dibuat. Cakupan kosong berarti umum.
type NewPoint struct {
	SubprocessID   string
	DocumentTypeID string
	Text           string
	Scope          ScopeInput
	Content        map[string]any // boleh nil (D-146)
}

// PointUpdate adalah isian perubahan poin. Teks wajib dikirim; isian nil
// berarti tetap (D-111).
type PointUpdate struct {
	Text     string
	Position *int
	Hidden   *bool
	Content  *map[string]any // nil berarti isi tetap (D-146)
}

// itemChange adalah hasil gabungan perubahan sub-proses atau poin dengan
// nilai lamanya: nama atau teks, urutan, dan tersembunyi.
type itemChange struct {
	Text     string
	Position int
	Hidden   bool
}

// cleanItemUpdate merapikan dan memeriksa perubahan satu sub-proses atau
// poin, lalu menggabungkannya dengan nilai lamanya. field adalah nama isian
// teksnya ("name" atau "text"), dan count adalah banyaknya saudara dalam
// daftar yang sama, untuk batas urutan.
func cleanItemUpdate(field, text string, maxLength int, position *int, hidden *bool, old itemChange, count int) (itemChange, []*problem.Problem) {
	clean, textProblem := problem.Text(field, text, true, maxLength)
	list := []*problem.Problem{textProblem}
	out := itemChange{Text: clean, Position: old.Position, Hidden: old.Hidden}
	if position != nil {
		out.Position = *position
	}
	if hidden != nil {
		out.Hidden = *hidden
	}
	if out.Position < 1 || out.Position > count {
		list = append(list, &problem.Problem{Field: "position", Code: problem.InvalidFormat,
			Message: fmt.Sprintf("Use a position from 1 to %d.", count)})
	}
	return out, list
}

// findSubprocess mencari sub-proses dalam versi menurut ID-nya.
func findSubprocess(v ProcessVersion, id string) (Subprocess, bool) {
	i := slices.IndexFunc(v.Subprocesses, func(s Subprocess) bool { return s.ID == id })
	if i < 0 {
		return Subprocess{}, false
	}
	return v.Subprocesses[i], true
}

// findPoint mencari poin dalam versi menurut ID-nya, beserta sub-prosesnya.
func findPoint(v ProcessVersion, id string) (Subprocess, Point, bool) {
	for _, s := range v.Subprocesses {
		for _, p := range s.Points {
			if p.ID == id {
				return s, p, true
			}
		}
	}
	return Subprocess{}, Point{}, false
}

// findVariant mencari varian dalam versi menurut ID-nya, beserta sub-proses
// dan poin asalnya.
func findVariant(v ProcessVersion, id string) (Subprocess, Point, Variant, bool) {
	for _, s := range v.Subprocesses {
		for _, p := range s.Points {
			for _, variant := range p.Variants {
				if variant.ID == id {
					return s, p, variant, true
				}
			}
		}
	}
	return Subprocess{}, Point{}, Variant{}, false
}

// countPoints menghitung poin satu sub-proses untuk satu jenis dokumen,
// termasuk yang tersembunyi.
func countPoints(s Subprocess, documentTypeID string) int {
	n := 0
	for _, p := range s.Points {
		if p.DocumentType.ID == documentTypeID {
			n++
		}
	}
	return n
}

// Batas isi poin per kolom (D-146).
const maxPointColumns = 60

var pointKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

// cleanPointContent merapikan isi poin per kolom (D-146). Kuncinya key tetap
// kolom format. Nilainya teks, atau bilangan bulat 1 sampai 10 untuk kolom
// rating. Teks kosong dibuang. Isi tidak diperiksa terhadap satu format
// tertentu, karena satu poin melayani beberapa form; nilai yang tidak cocok
// dengan kolom form dokumen dilewati saat perakitan (M4.5).
func cleanPointContent(content map[string]any) (map[string]any, *problem.Problem) {
	bad := &problem.Problem{Field: "content", Code: problem.InvalidFormat,
		Message: "Content is keyed by the column key, with text or a whole number from 1 to 10 as value."}
	out := map[string]any{}
	if len(content) > maxPointColumns {
		return nil, bad
	}
	for key, value := range content {
		if !pointKeyPattern.MatchString(key) {
			return nil, bad
		}
		switch v := value.(type) {
		case nil:
			continue
		case string:
			v = strings.TrimSpace(v)
			if utf8.RuneCountInString(v) > MaxTextLength {
				return nil, bad
			}
			if v != "" {
				out[key] = v
			}
		default:
			n, ok := whole(value)
			if !ok || n < 1 || n > maxRating {
				return nil, bad
			}
			out[key] = n
		}
	}
	return out, nil
}

// pointContent membaca isi poin dari database; isi kosong menjadi map kosong.
func pointContent(data []byte) map[string]any {
	out := map[string]any{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &out)
	}
	return out
}
