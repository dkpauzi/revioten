package numbering

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/customer"
)

// lineResults menulis hasil per baris impor sebagai "baris:teks:hasil".
func lineResults(r ImportReport) []string {
	var out []string
	for _, l := range r.Lines {
		out = append(out, fmt.Sprintf("%d:%s:%s", l.Line, l.Text, l.Result))
	}
	return out
}

// groupTexts menulis laporan per kode customer dan jenis dokumen sebagai
// "kode jenis jumlah tertinggi lubang".
func groupTexts(r ImportReport) []string {
	var out []string
	for _, g := range r.Groups {
		out = append(out, fmt.Sprintf("%s %s %d %s %s", g.CustomerCode, g.DocumentType.Key, g.Count, g.Highest, rangeTexts(g.Gaps)))
	}
	return out
}

// misalignedTexts menulis nomor tidak sejajar sebagai "kode rentang jenis".
func misalignedTexts(r ImportReport) []string {
	var out []string
	for _, m := range r.Misaligned {
		var keys []string
		for _, d := range m.Present {
			keys = append(keys, d.Key)
		}
		out = append(out, fmt.Sprintf("%s %s %s", m.CustomerCode, rangeTexts([]SequenceRange{m.Range}), strings.Join(keys, "+")))
	}
	return out
}

func rangeTexts(list []SequenceRange) string {
	var out []string
	for _, r := range list {
		if r.First == r.Last {
			out = append(out, r.First)
		} else {
			out = append(out, r.First+".."+r.Last)
		}
	}
	return "[" + strings.Join(out, ",") + "]"
}

// TestImport adalah kriteria selesai M2.5: impor satu customer menghasilkan
// laporan yang benar. Nomor lama masuk dengan teks aslinya, penghitung naik ke
// nomor tertinggi, dan lubang, nomor ganda, serta nomor tidak sejajar hanya
// dilaporkan, tidak pernah diperbaiki (R-NUM-06, R-POR-01).
func TestImport(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)
	f.newCustomer(t, "NB")
	// Customer ini dulu berkode KT, sekarang KU (E-11).
	ku, err := f.customers.ChangeCode(ctx, f.master, f.kt.ID, "KU")
	if err != nil {
		t.Fatal(err)
	}
	// Satu nomor sudah didaftarkan lewat sistem sebelum impor.
	if got := f.register(t, f.author, ku, "cp-baru"); got != "SC-KU-001" {
		t.Fatalf("nomor sebelum impor %q", got)
	}

	text := strings.Join([]string{
		"SH-KU-001",
		"SH-KU-002",
		"SH-KU-005\tModel A\t-03", // kolom lain dari Excel diabaikan
		"",
		"SF-KU-001",
		"SF-KU-002",
		"SC-KU-001", // sudah ada di register
		"SC-KU-002",
		"SC-KU-002", // ganda
		"SC-KU-007",
		"SC-KT-004",    // kode lama customer ini
		"SC-NB-001",    // kode customer lain
		"SC-ZZ-001",    // kode yang tidak dikenal
		"SC-KU-01",     // bukan bentuk nomor berjalan
		"SW-KU-001",    // jenis dokumen dari WI, tidak dinomori sistem
		"  sh-ku-003 ", // huruf kecil
	}, "\r\n")
	report, err := f.s.Import(ctx, control, ImportInput{CustomerID: ku.ID, Status: StatusReleased, Text: text})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	wantLines := []string{
		"1:SH-KU-001:imported", "2:SH-KU-002:imported", "3:SH-KU-005:imported",
		"5:SF-KU-001:imported", "6:SF-KU-002:imported", "7:SC-KU-001:existing",
		"8:SC-KU-002:imported", "9:SC-KU-002:duplicate", "10:SC-KU-007:imported",
		"11:SC-KT-004:imported", "12:SC-NB-001:customer_mismatch", "13:SC-ZZ-001:customer_mismatch",
		"14:SC-KU-01:invalid", "15:SW-KU-001:invalid", "16:sh-ku-003:invalid",
	}
	if got := lineResults(report); !slices.Equal(got, wantLines) {
		t.Errorf("hasil per baris:\n%v\ningin:\n%v", got, wantLines)
	}
	if report.Imported != 8 || report.DryRun || report.Lines[7].FirstLine != 8 {
		t.Errorf("jumlah diimpor %d, dry run %v, baris pertama nomor ganda %d", report.Imported, report.DryRun, report.Lines[7].FirstLine)
	}

	// Laporan per kode (kode lama dulu) dan jenis dokumen: jumlah, nomor
	// tertinggi, dan lubang. Lubang tidak diisi.
	wantGroups := []string{
		"KT pfd 0  []", "KT pfmea 0  []", "KT cp 1 004 [001..003]",
		"KU pfd 3 005 [003..004]", "KU pfmea 2 002 []", "KU cp 3 007 [003..006]",
	}
	if got := groupTexts(report); !slices.Equal(got, wantGroups) {
		t.Errorf("laporan per kode dan jenis:\n%v\ningin:\n%v", got, wantGroups)
	}
	// Nomor tidak sejajar: nomor berjalan yang tidak dimiliki ketiga jenis.
	wantMisaligned := []string{"KT [004] cp", "KU [005] pfd", "KU [007] cp"}
	if got := misalignedTexts(report); !slices.Equal(got, wantMisaligned) {
		t.Errorf("nomor tidak sejajar:\n%v\ningin:\n%v", got, wantMisaligned)
	}

	// Penghitung naik ke nomor tertinggi; nomor baru melanjutkan, lubang tetap lubang.
	for key, want := range map[string]string{KeyPFD: "SH-KU-006", KeyPFMEA: "SF-KU-003", KeyCP: "SC-KU-008"} {
		if next, err := f.s.NextNumber(ctx, f.author, ku.ID, f.types[key].ID); err != nil || next != want {
			t.Errorf("nomor %s berikutnya %q, %v; ingin %s", key, next, err, want)
		}
	}

	// Nomor lama tersimpan dengan teks asli, status pilihan, dan asal impor.
	list, _, err := f.s.ListNumbers(ctx, f.author, ListFilter{Query: "SC-KT-004"})
	if err != nil || len(list) != 1 {
		t.Fatalf("mencari SC-KT-004: %+v, %v", list, err)
	}
	if n := list[0]; n.Status != StatusReleased || n.Origin != OriginImport || n.CustomerCode != "KT" ||
		n.CustomerCurrentCode != "KU" || n.Sequence != 4 || n.RegisteredByID != control.PersonID {
		t.Errorf("nomor impor %+v", n)
	}

	// Setiap nomor yang diimpor tercatat di audit log.
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM audit_log WHERE action = $1", audit.ActionDocumentNumberImported).Scan(&count); err != nil || count != 8 {
		t.Errorf("entri audit impor %d, %v; ingin 8", count, err)
	}
	var details string
	err = f.pool.QueryRow(ctx, "SELECT details FROM audit_log WHERE action = $1 ORDER BY seq LIMIT 1", audit.ActionDocumentNumberImported).Scan(&details)
	if err != nil || details != `{"number":"SH-KU-001","status":"released"}` {
		t.Errorf("entri audit impor pertama %s, %v", details, err)
	}

	// Impor yang sama diulang: semua nomor sudah ada, tidak ada batch baru.
	again, err := f.s.Import(ctx, control, ImportInput{CustomerID: ku.ID, Status: StatusReleased, Text: text})
	if err != nil || again.Imported != 0 || again.Lines[0].Result != LineExisting || again.Lines[2].Result != LineExisting {
		t.Errorf("impor diulang: %v, %v", lineResults(again), err)
	}
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM number_batches WHERE kind = 'import'").Scan(&count); err != nil || count != 1 {
		t.Errorf("batch impor %d, %v; ingin 1", count, err)
	}
}

// TestImportDryRun: pratinjau memberi laporan yang sama tanpa menyimpan
// apa pun, termasuk penghitung dan audit log.
func TestImportDryRun(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)
	in := ImportInput{CustomerID: f.kt.ID, Status: StatusInUse, Text: "SC-KT-010\nSC-KT-011\n", DryRun: true}

	preview, err := f.s.Import(ctx, control, in)
	if err != nil {
		t.Fatalf("pratinjau: %v", err)
	}
	if !preview.DryRun || preview.Imported != 2 || !slices.Equal(groupTexts(preview), []string{"KT pfd 0  []", "KT pfmea 0  []", "KT cp 2 011 [001..009]"}) {
		t.Errorf("pratinjau %+v", preview)
	}
	for _, table := range []string{"document_numbers", "number_batches", "number_counters"} {
		var count int
		if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Errorf("%s berisi %d baris setelah pratinjau, %v", table, count, err)
		}
	}
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM audit_log WHERE action = $1", audit.ActionDocumentNumberImported).Scan(&count); err != nil || count != 0 {
		t.Errorf("entri audit setelah pratinjau %d, %v", count, err)
	}

	in.DryRun = false
	report, err := f.s.Import(ctx, control, in)
	if err != nil || report.DryRun || report.Imported != 2 {
		t.Fatalf("impor setelah pratinjau: %+v, %v", report, err)
	}
	if next, _ := f.s.NextNumber(ctx, f.author, f.kt.ID, f.types[KeyCP].ID); next != "SC-KT-012" {
		t.Errorf("nomor berikutnya %q, ingin SC-KT-012", next)
	}
}

// TestImportRules: isian yang salah ditolak per field, dan hanya pemegang
// E-8 (dokumen kontrol) yang boleh mengimpor (UC-14).
func TestImportRules(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)
	inactive := f.newCustomer(t, "IN")
	if _, err := f.customers.Update(ctx, f.master, inactive.ID, customer.CustomerUpdate{Name: "Customer IN", ProjectGroupID: f.group.ID, Active: new(false)}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.s.Import(ctx, f.author, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: "SC-KT-001"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun mengimpor: %v, ingin ErrForbidden", err)
	}
	tooMany := strings.Repeat("SC-KT-001\n", MaxImportLines+1)
	tests := []struct {
		name string
		in   ImportInput
		want []string
	}{
		{"semua kosong", ImportInput{Text: " \n\t\n"}, []string{"customer_id:required", "status:required", "text:required"}},
		{"status Skipped", ImportInput{CustomerID: f.kt.ID, Status: StatusSkipped, Text: "SC-KT-001"}, []string{"status:invalid_format"}},
		{"customer nonaktif", ImportInput{CustomerID: inactive.ID, Status: StatusVoid, Text: "SC-IN-001"}, []string{"customer_id:not_allowed"}},
		{"terlalu banyak baris", ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: tooMany}, []string{"text:too_long"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.Import(ctx, control, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("Import(%s) = %v, ingin %v", tt.name, got, tt.want)
			}
		})
	}
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM document_numbers").Scan(&count); err != nil || count != 0 {
		t.Errorf("nomor tersimpan setelah impor yang ditolak: %d, %v", count, err)
	}
}

// TestImportConcurrent: impor yang rentangnya bertumpuk, pendaftaran satu
// nomor, dan pendaftaran set untuk customer yang sama berjalan bersamaan tanpa
// deadlock. Setiap nomor masuk tepat sekali, dan register tetap tanpa lubang
// karena rentang impor bersambung (R-NUM-03, R-NUM-06).
func TestImportConcurrent(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)

	// Impor ke-i berisi CP dan PFD 10i+1 sampai 10i+15. Urutan baris berbeda
	// antar impor, supaya urutan kunci penghitung diuji.
	texts := make([]string, 4)
	for i := range texts {
		var lines []string
		for v := 10*i + 1; v <= 10*i+15; v++ {
			lines = append(lines, fmt.Sprintf("SC-KT-%03d", v), fmt.Sprintf("SH-KT-%03d", v))
		}
		if i%2 == 1 {
			slices.Reverse(lines)
		}
		texts[i] = strings.Join(lines, "\n")
	}

	var wg sync.WaitGroup
	reports := make([]ImportReport, len(texts))
	errs := make(chan error, 40)
	for i, text := range texts {
		wg.Go(func() {
			var err error
			reports[i], err = f.s.Import(ctx, control, ImportInput{CustomerID: f.kt.ID, Status: StatusReleased, Text: text})
			errs <- err
		})
	}
	for i := range 10 {
		wg.Go(func() {
			_, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: fmt.Sprintf("cp-%d", i)})
			errs <- err
		})
		if i < 5 {
			wg.Go(func() {
				_, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: fmt.Sprintf("set-%d", i)})
				errs <- err
			})
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("berjalan bersamaan: %v", err)
		}
	}

	// Setiap baris impor masuk tepat sekali di seluruh impor.
	imported := 0
	for i, r := range reports {
		for _, l := range r.Lines {
			if l.Result != LineImported && l.Result != LineExisting {
				t.Errorf("impor %d baris %d: %s", i, l.Line, l.Result)
			}
		}
		imported += r.Imported
	}
	var stored int
	err := f.pool.QueryRow(ctx, "SELECT count(*) FROM document_numbers n JOIN number_batches b ON b.id = n.batch_id WHERE b.kind = 'import'").Scan(&stored)
	if err != nil || stored != imported {
		t.Errorf("nomor impor tersimpan %d, %v; laporan menyebut %d", stored, err, imported)
	}

	for _, key := range []string{KeyPFD, KeyPFMEA, KeyCP} {
		rows, err := f.pool.Query(ctx, "SELECT sequence FROM document_numbers WHERE document_type_id = $1 ORDER BY sequence", f.types[key].ID)
		if err != nil {
			t.Fatal(err)
		}
		var sequences []int
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err != nil {
				t.Fatal(err)
			}
			sequences = append(sequences, v)
		}
		for i, v := range sequences {
			if v != i+1 {
				t.Fatalf("%s: nomor berjalan %v berlubang atau ganda", key, sequences)
			}
		}
		if key != KeyPFMEA && len(sequences) < 45 {
			t.Errorf("%s: hanya %d nomor, ingin paling sedikit 45", key, len(sequences))
		}
	}
}

// TestBuildReport menguji penyusunan laporan tanpa database: lubang dihitung
// dari semua status (Skipped bukan lubang), sedangkan nomor tidak sejajar
// tidak menghitung Skipped, karena nomor itu tidak punya dokumen.
func TestBuildReport(t *testing.T) {
	types := []DocumentType{{ID: "1", Key: KeyPFD}, {ID: "2", Key: KeyPFMEA}, {ID: "3", Key: KeyCP}}
	row := func(typeID string, seq int, status string) registerRow {
		return registerRow{code: "KT", typeID: typeID, sequence: seq, status: status}
	}
	rows := []registerRow{
		row("1", 1, StatusSkipped), row("1", 2, StatusSkipped), row("1", 3, StatusReserved),
		row("2", 3, StatusReserved),
		row("3", 1, StatusReleased), row("3", 2, StatusReleased), row("3", 3, StatusReserved), row("3", 6, StatusVoid),
		row("3", 0, StatusReleased),
	}
	r := ImportReport{}
	r.Groups, r.Misaligned = buildReport([]string{"KT"}, types, rows)
	if got, want := groupTexts(r), []string{"KT pfd 3 003 []", "KT pfmea 1 003 [001..002]", "KT cp 5 006 [004..005]"}; !slices.Equal(got, want) {
		t.Errorf("laporan:\n%v\ningin:\n%v", got, want)
	}
	if got, want := misalignedTexts(r), []string{"KT [000..002] cp", "KT [006] cp"}; !slices.Equal(got, want) {
		t.Errorf("tidak sejajar:\n%v\ningin:\n%v", got, want)
	}
}
