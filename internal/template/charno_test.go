package template

import (
	"slices"
	"testing"
)

// D-222 (migrasi 00057): Control Plan v1 punya kolom No di depan Product
// Characteristics dan di depan Process Characteristics, dalam grup
// CHARACTERISTICS ( Key Parameter ). Keduanya terisi otomatis, jadi isian
// yang diketik ditolak.
func TestCharacteristicNumberColumns(t *testing.T) {
	f := newFixture(t)
	v := f.version(t, "Control Plan v1")
	got := labels(v, AreaRow)
	want := []string{"Process Flow", "No", "Process Name / Operation Description", "Machine, Equipment, Jigs, Fixture, Tools for Mfg",
		"Product No", "Product Characteristics", "Product Specification / Tolerance", "Process No", "Process Characteristics", "Process Specification / Tolerance"}
	if !slices.Equal(got[:len(want)], want) {
		t.Fatalf("kolom CP: %v", got)
	}
	var product, process Field
	for _, fd := range v.Fields {
		switch fd.Kind {
		case KindProductCharNo:
			product = fd
		case KindProcessCharNo:
			process = fd
		}
	}
	if product.Group != "CHARACTERISTICS ( Key Parameter )" || process.Group != product.Group {
		t.Errorf("grup: %q, %q", product.Group, process.Group)
	}
	if p := checkValue(product, "3", nil); p == nil || p.Code != "not_allowed" {
		t.Errorf("nomor diketik: %v", p)
	}
	// Satu kolom per jenis.
	if _, err := f.s.AddField(t.Context(), f.master, v.ID, NewField{Area: AreaRow, Kind: KindProductCharNo, Label: "No lagi"}); !slices.Equal(problemCodes(t, err), []string{"kind:taken"}) {
		t.Errorf("kolom kedua: %v", err)
	}
}
