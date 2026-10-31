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
	"fmt"
	"slices"
	"time"

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
// versinya. ASUMSI E-72: satu template per proses, dengan cakupan umum.
type ProcessTemplate struct {
	ID        string
	Process   ProcessRef
	Versions  []ProcessVersionSummary // menurut nomor versi, tidak pernah nil
	CreatedAt time.Time               // selalu UTC
}

// ProcessVersionSummary adalah ringkasan satu versi untuk daftar template.
type ProcessVersionSummary struct {
	ID      string
	Version int
	Status  string
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
// dokumen (R-TPL-04).
type Point struct {
	ID           string // tetap lintas versi; baris dokumen nanti menunjuk ID ini
	DocumentType DocumentTypeRef
	Text         string
	Position     int // urutan dalam sub-proses untuk jenis dokumen yang sama, mulai 1
	Hidden       bool
}

// SubprocessUpdate adalah isian perubahan sub-proses. Nama wajib dikirim;
// isian nil berarti tetap (D-111).
type SubprocessUpdate struct {
	Name     string
	Position *int
	Hidden   *bool
}

// NewPoint adalah isian poin baru. Sub-proses dan jenis dokumennya tetap
// setelah poin dibuat.
type NewPoint struct {
	SubprocessID   string
	DocumentTypeID string
	Text           string
}

// PointUpdate adalah isian perubahan poin. Teks wajib dikirim; isian nil
// berarti tetap (D-111).
type PointUpdate struct {
	Text     string
	Position *int
	Hidden   *bool
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
