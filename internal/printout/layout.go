// Package printout membuat cetakan PFD, PFMEA, dan CP (Tahap 6). Satu model
// cetak netral (Sheet) dibentuk dari isi revisi, versi format, layout, rute
// approval, dan riwayat revisi; model yang sama ditulis ke Excel (excelize)
// dan ke HTML yang dicetak ke PDF lewat browser headless (hasil uji S5,
// ASUMSI E-108). Layout adalah data (R-PRT-02), bukan kode.
package printout

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Layout adalah isi satu layout cetak (M6.1, R-PRT-02). Bentuk JSON ini yang
// disimpan di print_layouts.layout dan diedit admin.
type Layout struct {
	Paper         string             `json:"paper"`       // A3 atau A4
	Orientation   string             `json:"orientation"` // landscape atau portrait
	FontSize      float64            `json:"font_size"`   // pt, 6 sampai 14
	MarginsMM     Margins            `json:"margins_mm"`
	Cover         bool               `json:"cover"`          // halaman cover TR-020 (R-PRT-07)
	RevisionMarks bool               `json:"revision_marks"` // tanda segitiga revisi (R-REV-09)
	Widths        map[string]float64 `json:"widths"`         // lebar relatif per key field atau ID field; bawaan 1
}

// Margins adalah margin halaman dalam milimeter.
type Margins struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// ParseLayout membaca dan memeriksa layout. Nilai yang tidak diisi memakai
// bawaan: A3 landscape, huruf 8 pt, margin 10 mm.
func ParseLayout(data []byte) (Layout, error) {
	l := Layout{Paper: "A3", Orientation: "landscape", FontSize: 8, MarginsMM: Margins{10, 10, 10, 10}, Widths: map[string]float64{}}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &l); err != nil {
			return Layout{}, invalidLayout("layout", "The layout is not valid JSON.")
		}
	}
	if l.Widths == nil {
		l.Widths = map[string]float64{}
	}
	if err := l.Validate(); err != nil {
		return Layout{}, err
	}
	return l, nil
}

// Validate memeriksa batas setiap isian layout.
func (l Layout) Validate() error {
	switch {
	case l.Paper != "A3" && l.Paper != "A4":
		return invalidLayout("paper", "Choose paper A3 or A4.")
	case l.Orientation != "landscape" && l.Orientation != "portrait":
		return invalidLayout("orientation", "Choose landscape or portrait.")
	case l.FontSize < 6 || l.FontSize > 14:
		return invalidLayout("font_size", "The font size must be between 6 and 14 pt.")
	}
	for _, m := range []float64{l.MarginsMM.Top, l.MarginsMM.Right, l.MarginsMM.Bottom, l.MarginsMM.Left} {
		if m < 3 || m > 30 || math.IsNaN(m) {
			return invalidLayout("margins_mm", "Margins must be between 3 and 30 mm.")
		}
	}
	for k, w := range l.Widths {
		if w <= 0 || w > 20 || math.IsNaN(w) {
			return invalidLayout("widths", fmt.Sprintf("The width of %s must be above 0 and at most 20.", k))
		}
	}
	return nil
}

// Width memberi lebar relatif satu kolom: menurut key field, lalu ID field,
// lalu bawaan per jenis kolom.
func (l Layout) Width(key, id string, fallback float64) float64 {
	if w, ok := l.Widths[key]; ok && key != "" {
		return w
	}
	if w, ok := l.Widths[id]; ok {
		return w
	}
	return fallback
}

// PageMM memberi ukuran halaman (lebar, tinggi) dalam milimeter.
func (l Layout) PageMM() (float64, float64) {
	w, h := 420.0, 297.0
	if l.Paper == "A4" {
		w, h = 297, 210
	}
	if l.Orientation == "portrait" {
		w, h = h, w
	}
	return w, h
}

func invalidLayout(field, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: problem.InvalidFormat, Message: message}}}
}
