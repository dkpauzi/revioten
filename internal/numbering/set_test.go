package numbering

import (
	"fmt"
	"slices"
	"sync"
	"testing"
)

// setTexts menulis hasil pendaftaran sebagai "nomor:status".
func setTexts(reg Registration) []string {
	var texts []string
	for _, n := range reg.Numbers {
		texts = append(texts, n.Number+":"+n.Status)
	}
	return texts
}

// TestRegisterSet: kriteria M2.4 "tombol daftarkan sebagai set yang
// menyejajarkan nomor". Nomor set adalah nomor bebas berikutnya di ketiga
// jenis; nomor yang dilewati penghitung yang tertinggal dicatat Skipped,
// supaya register tidak berlubang (ASUMSI E-64).
func TestRegisterSet(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	// Penghitung tidak sejajar, seperti register Excel lama: CP di 003, PFMEA di 001, PFD kosong.
	for _, key := range []string{"cp1", "cp2", "cp3"} {
		f.register(t, f.author, f.kt, key)
	}
	if _, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyPFMEA].ID, IdempotencyKey: "fm1"}); err != nil {
		t.Fatal(err)
	}

	reg, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: "set-1"})
	if err != nil {
		t.Fatalf("RegisterSet: %v", err)
	}
	want := []string{
		"SH-KT-001:skipped", "SH-KT-002:skipped", "SH-KT-003:skipped", "SH-KT-004:reserved",
		"SF-KT-002:skipped", "SF-KT-003:skipped", "SF-KT-004:reserved",
		"SC-KT-004:reserved",
	}
	if got := setTexts(reg); !slices.Equal(got, want) {
		t.Errorf("hasil set:\n%v\ningin:\n%v", got, want)
	}
	for _, n := range reg.Numbers {
		if n.Origin != OriginSet || n.BatchID != reg.Numbers[0].BatchID {
			t.Errorf("nomor %s: asal %s, batch %s", n.Number, n.Origin, n.BatchID)
		}
	}

	// Setelah set, penghitung sejajar; nomor tunggal berikutnya melanjutkan.
	if got := f.register(t, f.author, f.kt, "cp5"); got != "SC-KT-005" {
		t.Errorf("CP setelah set: %q", got)
	}
	reg, err = f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: "set-2"})
	want = []string{"SH-KT-005:skipped", "SH-KT-006:reserved", "SF-KT-005:skipped", "SF-KT-006:reserved", "SC-KT-006:reserved"}
	if got := setTexts(reg); err != nil || !slices.Equal(got, want) {
		t.Errorf("set kedua: %v, %v", got, err)
	}

	// Kunci yang sama: set yang sama, tanpa nomor baru.
	again, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: "set-2"})
	if err != nil || !again.Replayed || !slices.Equal(setTexts(again), want) {
		t.Errorf("set diulang: %+v, %v", again, err)
	}
	if _, err := f.s.RegisterSet(ctx, f.author, NewSet{}); !slices.Equal(problemCodes(t, err), []string{"customer_id:required", "idempotency_key:required"}) {
		t.Errorf("set tanpa isian: %v", err)
	}
}

// TestInitialRevision adalah kriteria selesai M2.4: revisi awal bisa -04,
// dengan format revisi dari master data (R-REV-01, E-12).
func TestInitialRevision(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	reg, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, InitialRevision: 4, IdempotencyKey: "lama-04"})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range reg.Numbers {
		if n.InitialRevision != 4 || n.RevisionLabel != "-04" {
			t.Errorf("%s: revisi awal %d, label %q; ingin 4 dan -04", n.Number, n.InitialRevision, n.RevisionLabel)
		}
	}

	digits := 3
	if _, err := f.s.UpdateSettings(ctx, f.master, SettingsInput{FactoryCode: "S", RevisionDigits: &digits}); err != nil {
		t.Fatal(err)
	}
	list, _, err := f.s.ListNumbers(ctx, f.author, ListFilter{DocumentTypeID: f.types[KeyCP].ID})
	if err != nil || len(list) != 1 || list[0].RevisionLabel != "-004" {
		t.Errorf("label setelah digit revisi jadi 3: %+v, %v", list, err)
	}
	if _, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, InitialRevision: 1000, IdempotencyKey: "k"}); !slices.Equal(problemCodes(t, err), []string{"initial_revision:invalid_format"}) {
		t.Errorf("revisi 1000 dengan 3 digit: %v", err)
	}
}

// TestRegisterSetConcurrent: set dan nomor tunggal yang didaftarkan bersamaan
// tidak saling mengunci (deadlock), dan setiap jenis dokumen tetap punya nomor
// berjalan 1 sampai tertinggi tanpa lubang dan tanpa ganda (R-NUM-03).
func TestRegisterSetConcurrent(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := range 10 {
		wg.Go(func() {
			_, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: fmt.Sprintf("set-%d", i)})
			errs <- err
		})
		wg.Go(func() {
			_, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: fmt.Sprintf("cp-%d", i)})
			errs <- err
		})
		wg.Go(func() {
			_, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyPFD].ID, IdempotencyKey: fmt.Sprintf("pfd-%d", i)})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("pendaftaran bersamaan: %v", err)
		}
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
	}
}
