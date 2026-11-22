package numbering

import (
	"slices"
	"strings"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// TestImportSets: D-216 dan E-140. Satu baris adalah satu set (sampai tiga
// kolom), setiap baris mendapat batch sendiri, dan bawaannya nomor yang tidak
// ada di baris mengikuti nomor berjalan nomor pertama.
func TestImportSets(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)

	// Bukan satu set: nomor diketik apa adanya, juga yang tidak sejajar.
	text := strings.Join([]string{
		"SH-KT-001\tSF-KT-001\tSC-KT-001",
		"SC-KT-005\tSF-KT-005\tSH-KT-004", // urutan kolom bebas, nomor tidak sejajar
		"SH-KT-007\tSH-KT-008",            // dua PFD di satu baris
	}, "\n")
	report, err := f.s.Import(ctx, control, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"1:SH-KT-001:imported", "1:SF-KT-001:imported", "1:SC-KT-001:imported",
		"2:SC-KT-005:imported", "2:SF-KT-005:imported", "2:SH-KT-004:imported",
		"3:SH-KT-007:imported", "3:SH-KT-008:type_repeated",
	}
	if got := lineResults(report); !slices.Equal(got, want) {
		t.Errorf("hasil:\n%v\ningin:\n%v", got, want)
	}
	// Satu batch per baris: nomor satu baris berbagi batch, baris lain tidak.
	batch := func(number string) string {
		t.Helper()
		var id string
		if err := f.pool.QueryRow(ctx, "SELECT batch_id FROM document_numbers WHERE number = $1", number).Scan(&id); err != nil {
			t.Fatalf("%s: %v", number, err)
		}
		return id
	}
	if batch("SC-KT-005") != batch("SH-KT-004") || batch("SF-KT-005") != batch("SH-KT-004") {
		t.Error("nomor satu baris harus satu batch")
	}
	if batch("SH-KT-001") == batch("SH-KT-004") || batch("SH-KT-001") == batch("SH-KT-007") {
		t.Error("baris berbeda harus batch berbeda")
	}

	// Satu set: cukup satu kolom, dua lainnya mengikuti.
	report, err = f.s.Import(ctx, control, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, OneSet: true,
		Text: "SH-KT-020\nSC-KT-021\tSF-KT-021\nSF-KT-001"})
	if err != nil {
		t.Fatal(err)
	}
	want = []string{
		"1:SH-KT-020:imported", "1:SF-KT-020:imported", "1:SC-KT-020:imported",
		"2:SC-KT-021:imported", "2:SF-KT-021:imported", "2:SH-KT-021:imported",
		"3:SF-KT-001:existing", "3:SH-KT-001:existing", "3:SC-KT-001:existing",
	}
	if got := lineResults(report); !slices.Equal(got, want) {
		t.Errorf("satu set:\n%v\ningin:\n%v", got, want)
	}
	if !report.Lines[1].Derived || report.Lines[0].Derived || !report.Lines[5].Derived {
		t.Errorf("tanda nomor yang mengikuti: %+v", report.Lines)
	}
	if batch("SH-KT-020") != batch("SC-KT-020") || batch("SH-KT-020") == batch("SH-KT-021") {
		t.Error("satu set per baris")
	}
}
