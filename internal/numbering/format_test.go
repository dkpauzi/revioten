package numbering

import (
	"slices"
	"testing"
)

// TestFormatSequence: nomor berjalan tiga digit sampai 999, lalu huruf dan
// tiga digit: A001 sampai A999, B001, dan seterusnya sampai Z999 (E-12, D-89).
func TestFormatSequence(t *testing.T) {
	tests := map[int]string{
		0: "000", 1: "001", 42: "042", 999: "999",
		1000: "A001", 1998: "A999", 1999: "B001", 26973: "Z999",
	}
	for n, want := range tests {
		got, err := FormatSequence(n)
		if err != nil || got != want {
			t.Errorf("FormatSequence(%d) = %q, %v; ingin %q", n, got, err, want)
		}
		back, ok := ParseSequence(want)
		if !ok || back != n {
			t.Errorf("ParseSequence(%q) = %d, %v; ingin %d", want, back, ok, n)
		}
	}
	for _, n := range []int{-1, MaxSequence + 1} {
		if _, err := FormatSequence(n); err == nil {
			t.Errorf("FormatSequence(%d) harus ditolak", n)
		}
	}
	for _, text := range []string{"", "1", "01", "1000", "A000", "a001", "AA01", "A01", "Ä001"} {
		if n, ok := ParseSequence(text); ok {
			t.Errorf("ParseSequence(%q) = %d, harus ditolak", text, n)
		}
	}
}

// TestFormatRevision: revisi ditulis dengan tanda hubung dan jumlah digit dari
// master data, misalnya -04 (R-REV-01, M2.4).
func TestFormatRevision(t *testing.T) {
	if got := FormatRevision(4, 2); got != "-04" {
		t.Errorf("FormatRevision(4, 2) = %q", got)
	}
	if got := FormatRevision(0, 2); got != "-00" {
		t.Errorf("FormatRevision(0, 2) = %q", got)
	}
	if got := FormatRevision(7, 3); got != "-007" {
		t.Errorf("FormatRevision(7, 3) = %q", got)
	}
}

// TestPattern: pola bawaan menghasilkan nomor seperti SC-KT-001 (kriteria
// selesai M2.1), dan teks nomor bisa dibaca kembali menjadi bagiannya.
func TestPattern(t *testing.T) {
	p, err := ParsePattern(DefaultPattern)
	if err != nil {
		t.Fatalf("ParsePattern: %v", err)
	}
	if got := p.Format("SC", "KT", "001"); got != "SC-KT-001" {
		t.Errorf("Format = %q, ingin SC-KT-001", got)
	}
	parts, ok := p.Parse("SC-HSE-A123")
	if !ok || parts != (Parts{Prefix: "SC", Customer: "HSE", Sequence: "A123"}) {
		t.Errorf("Parse(SC-HSE-A123) = %+v, %v", parts, ok)
	}
	for _, text := range []string{"SC-KT-01", "SCKT001", "SC-KT-001-04", "sc-kt-001", "SW-ZZ-000 rev 1", ""} {
		if parts, ok := p.Parse(text); ok {
			t.Errorf("Parse(%q) = %+v, harus ditolak", text, parts)
		}
	}

	// Pola lain tetap bisa dibaca kembali, asal setiap bagian dipisah tanda baca.
	other, err := ParsePattern("{customer}/{prefix}.{seq}")
	if err != nil {
		t.Fatalf("ParsePattern pola lain: %v", err)
	}
	number := other.Format("SF", "AB", "B002")
	if number != "AB/SF.B002" {
		t.Errorf("Format pola lain = %q", number)
	}
	if parts, ok := other.Parse(number); !ok || parts.Sequence != "B002" || parts.Customer != "AB" {
		t.Errorf("Parse pola lain = %+v, %v", parts, ok)
	}
}

// TestParsePatternRules: setiap token tepat sekali, dan token yang bersebelahan
// harus dipisah tanda baca, supaya nomor lama bisa dibaca kembali tanpa ragu
// (M2.5). Huruf dan angka tidak boleh ada di luar token.
func TestParsePatternRules(t *testing.T) {
	bad := []string{
		"", "{prefix}{customer}{seq}", "{prefix}-{customer}", "{prefix}-{customer}-{seq}-{seq}",
		"{prefix}-{customer}-{seq}-X", "{prefix}-{cust}-{seq}", "{prefix}-{customer}-{seq}}",
		"{prefix}-{customer}-{seq}------------------------------------------",
	}
	for _, text := range bad {
		if _, err := ParsePattern(text); err == nil {
			t.Errorf("ParsePattern(%q) harus ditolak", text)
		}
	}
	good := []string{DefaultPattern, "{prefix}/{customer}/{seq}", "{customer}-{prefix}-{seq}", "[{prefix}-{customer}-{seq}]"}
	for _, text := range good {
		if _, err := ParsePattern(text); err != nil {
			t.Errorf("ParsePattern(%q): %v", text, err)
		}
	}
}

// TestSequenceOrder: urutan nomor berjalan sama dengan urutan angkanya, juga
// setelah 999.
func TestSequenceOrder(t *testing.T) {
	var texts []string
	for _, n := range []int{1, 999, 1000, 1998, 1999, MaxSequence} {
		s, err := FormatSequence(n)
		if err != nil {
			t.Fatal(err)
		}
		texts = append(texts, s)
	}
	if !slices.Equal(texts, []string{"001", "999", "A001", "A999", "B001", "Z999"}) {
		t.Errorf("urutan %v", texts)
	}
}

// TestSplitRevision: I4 opsi b. Akhiran revisi nomor lama dipisahkan menurut
// jumlah digit revisi di master data; nomor tanpa revisi tetap diterima.
func TestSplitRevision(t *testing.T) {
	p, err := ParsePattern(DefaultPattern)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		text, number string
		digits       int
		revision     int
		ok           bool
	}{
		{"SC-KT-001-05", "SC-KT-001", 2, 5, true},
		{"SC-KT-001-5", "SC-KT-001", 2, 5, true},
		{"SC-KT-001-00", "SC-KT-001", 2, 0, true},
		{"SC-KT-001", "SC-KT-001", 2, 0, false},         // tanpa revisi
		{"SC-KT-001-123", "SC-KT-001-123", 2, 0, false}, // lebih dari dua digit
		{"SC-KT-001-123", "SC-KT-001", 3, 123, true},
		{"SC-KT-001-A1", "SC-KT-001-A1", 2, 0, false}, // bukan angka
		{"SC-KT-01-05", "SC-KT-01-05", 2, 0, false},   // sisanya bukan nomor
		{"SC-KT-001-", "SC-KT-001-", 2, 0, false},
	}
	for _, c := range cases {
		number, revision, ok := p.SplitRevision(c.text, c.digits)
		if number != c.number || revision != c.revision || ok != c.ok {
			t.Errorf("SplitRevision(%q, %d) = %q, %d, %v; ingin %q, %d, %v", c.text, c.digits, number, revision, ok, c.number, c.revision, c.ok)
		}
	}
}
