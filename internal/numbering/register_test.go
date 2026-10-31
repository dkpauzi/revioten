package numbering

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/customer"
)

// numberFixture: fixture dengan Project Group GR1, customer KT, akun Penyusun
// (PPAP Engineer, semua customer), dan ID jenis dokumen sistem.
type numberFixture struct {
	fixture
	customers *customer.Service
	group     customer.ProjectGroup
	kt        customer.Customer
	author    access.Actor
	types     map[string]DocumentType // per key: pfd, pfmea, cp
	other     DocumentType            // jenis dokumen dari WI, tanpa key
}

func newNumberFixture(t *testing.T) numberFixture {
	t.Helper()
	f := numberFixture{fixture: newFixture(t), types: map[string]DocumentType{}}
	ctx := t.Context()
	f.customers = customer.NewService(f.pool, access.NewChecker(f.pool))
	var err error
	f.group, err = f.customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	f.kt = f.newCustomer(t, "KT")
	f.author = newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)
	types, err := f.s.ListDocumentTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range types {
		f.types[d.Key] = d
	}
	f.other, err = f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{Name: "Work Instruction", Letter: "W"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// newCustomer menambah customer dengan kode itu di GR1.
func (f numberFixture) newCustomer(t *testing.T, code string) customer.Customer {
	t.Helper()
	c, err := f.customers.Create(t.Context(), f.admin, customer.NewCustomer{Code: code, Name: "Customer " + code, ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// register mendaftarkan satu nomor dengan kunci idempoten baru, lalu
// mengembalikan teks nomornya.
func (f numberFixture) register(t *testing.T, actor access.Actor, c customer.Customer, key string) string {
	t.Helper()
	reg, err := f.s.Register(t.Context(), actor, NewNumber{CustomerID: c.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return reg.Numbers[0].Number
}

// TestRegister: kriteria selesai M2.1. Pola menghasilkan nomor seperti
// SC-KT-001, penghitung berjalan per kode customer dan jenis dokumen, dan
// kode customer di nomor adalah kode customer di master (R-POR-01).
func TestRegister(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()

	next, err := f.s.NextNumber(ctx, f.author, f.kt.ID, f.types[KeyCP].ID)
	if err != nil || next != "SC-KT-001" {
		t.Errorf("NextNumber pertama %q, %v; ingin SC-KT-001", next, err)
	}
	reg, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "kunci-1"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	n := reg.Numbers[0]
	if len(reg.Numbers) != 1 || reg.Replayed || n.Number != "SC-KT-001" || n.Sequence != 1 || n.Status != StatusReserved ||
		n.CustomerCode != "KT" || n.CustomerID != f.kt.ID || n.DocumentTypeKey != KeyCP || n.Origin != OriginSingle ||
		n.RegisteredByID != f.author.PersonID || n.InitialRevision != 0 {
		t.Errorf("hasil Register %+v", reg)
	}

	if got := f.register(t, f.author, f.kt, "kunci-2"); got != "SC-KT-002" {
		t.Errorf("nomor kedua %q, ingin SC-KT-002", got)
	}
	// Jenis dokumen lain punya penghitung sendiri.
	pfd, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyPFD].ID, IdempotencyKey: "kunci-3"})
	if err != nil || pfd.Numbers[0].Number != "SH-KT-001" {
		t.Errorf("nomor PFD pertama %+v, %v", pfd, err)
	}
	if next, _ := f.s.NextNumber(ctx, f.author, f.kt.ID, f.types[KeyCP].ID); next != "SC-KT-003" {
		t.Errorf("NextNumber setelah dua CP: %q", next)
	}

	// Setiap nomor tercatat di audit log dengan aktornya.
	var action, details string
	err = f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details)
	if err != nil || action != audit.ActionDocumentNumberRegistered || details != `{"number":"SH-KT-001","status":"reserved"}` {
		t.Errorf("entri audit terakhir %s %s, %v", action, details, err)
	}
}

// TestRegisterRules: isian yang salah ditolak per field, dan tidak ada nomor
// yang terpakai.
func TestRegisterRules(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	inactive := f.newCustomer(t, "IN")
	if _, err := f.customers.Update(ctx, f.admin, inactive.ID, customer.CustomerUpdate{Name: "Customer IN", ProjectGroupID: f.group.ID, Active: new(false)}); err != nil {
		t.Fatal(err)
	}
	cp := f.types[KeyCP].ID
	tests := []struct {
		name string
		in   NewNumber
		want []string
	}{
		{"semua kosong", NewNumber{}, []string{"customer_id:required", "document_type_id:required", "idempotency_key:required"}},
		{"customer tidak ada", NewNumber{CustomerID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", DocumentTypeID: cp, IdempotencyKey: "k"}, []string{"customer_id:invalid_format"}},
		{"customer nonaktif", NewNumber{CustomerID: inactive.ID, DocumentTypeID: cp, IdempotencyKey: "k"}, []string{"customer_id:not_allowed"}},
		{"jenis dokumen dari WI", NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.other.ID, IdempotencyKey: "k"}, []string{"document_type_id:not_allowed"}},
		{"revisi tiga digit", NewNumber{CustomerID: f.kt.ID, DocumentTypeID: cp, InitialRevision: 100, IdempotencyKey: "k"}, []string{"initial_revision:invalid_format"}},
		{"kunci dengan spasi", NewNumber{CustomerID: f.kt.ID, DocumentTypeID: cp, IdempotencyKey: "kunci salah"}, []string{"idempotency_key:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.Register(ctx, f.author, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	if got := f.register(t, f.author, f.kt, "kunci-sah"); got != "SC-KT-001" {
		t.Errorf("setelah penolakan, nomor pertama %q, ingin SC-KT-001", got)
	}
}

// TestRegisterIdempotent: permintaan yang diulang dengan kunci yang sama
// (misalnya tombol ditekan dua kali, atau jaringan putus lalu dikirim ulang)
// mendapat nomor yang sama, bukan nomor baru (R-NUM-03). Kunci yang sama untuk
// isi lain ditolak.
func TestRegisterIdempotent(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	in := NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "0f8c2a9e-kunci"}
	first, err := f.s.Register(ctx, f.author, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.s.Register(ctx, f.author, in)
	if err != nil || !again.Replayed || again.Numbers[0].ID != first.Numbers[0].ID {
		t.Errorf("permintaan ulang %+v, %v; ingin nomor yang sama", again, err)
	}
	if next, _ := f.s.NextNumber(ctx, f.author, f.kt.ID, f.types[KeyCP].ID); next != "SC-KT-002" {
		t.Errorf("permintaan ulang menaikkan penghitung: berikutnya %q", next)
	}

	in.DocumentTypeID = f.types[KeyPFD].ID
	if _, err := f.s.Register(ctx, f.author, in); !slices.Equal(problemCodes(t, err), []string{"idempotency_key:taken"}) {
		t.Errorf("kunci sama untuk isi lain: %v", err)
	}
	// Kunci berlaku per akun: akun lain boleh memakai teks kunci yang sama.
	other := newActor(t, f.pool, "penyusun.lain", access.AccountWork, access.RoleAuthor)
	if got := f.register(t, other, f.kt, "0f8c2a9e-kunci"); got != "SC-KT-002" {
		t.Errorf("akun lain dengan kunci yang sama: %q", got)
	}
}

// TestRegisterConcurrent adalah kriteria selesai M2.2: 50 pendaftaran
// paralel menghasilkan 50 nomor berbeda tanpa lubang (R-NUM-03), dan rantai
// audit tetap utuh (E-06).
func TestRegisterConcurrent(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	const total = 50
	numbers := make([]string, total)
	errs := make([]error, total)
	var wg sync.WaitGroup
	for i := range total {
		wg.Go(func() {
			reg, err := f.s.Register(ctx, f.author, NewNumber{
				CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: fmt.Sprintf("paralel-%02d", i),
			})
			errs[i] = err
			if err == nil {
				numbers[i] = reg.Numbers[0].Number
			}
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("pendaftaran %d: %v", i, err)
		}
	}
	slices.Sort(numbers)
	for i, got := range numbers {
		if want := fmt.Sprintf("SC-KT-%03d", i+1); got != want {
			t.Fatalf("nomor ke-%d %q, ingin %q (ada nomor ganda atau lubang): %v", i+1, got, want, numbers)
		}
	}
	v, err := audit.NewService(f.pool, access.NewChecker(f.pool)).Verify(ctx, f.admin)
	if err != nil || !v.Valid {
		t.Errorf("rantai audit setelah pendaftaran paralel: %+v, %v", v, err)
	}
}

// TestRegisterDuringCodeChange: pendaftaran satu nomor dan pendaftaran set yang
// berjalan bersamaan dengan penggantian kode customer tidak saling mengunci
// (deadlock), dan setiap kode tetap punya nomor berjalan tanpa lubang
// (R-POR-01, R-NUM-03). Ditemukan saat menyiapkan Gerbang 2: pendaftaran dulu
// menyimpan batch (foreign key ke customer) sebelum mengunci customer.
func TestRegisterDuringCodeChange(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	// Bentrok hanya terjadi bila penggantian kode mendapat kunci tepat di antara
	// dua perintah satu pendaftaran, jadi dicoba sepuluh putaran. Setiap putaran
	// menjalankan empat nomor CP, satu set, dan satu penggantian kode bersamaan.
	codes := []string{"KT"}
	for round := range 10 {
		code := fmt.Sprintf("L%c", 'A'+round)
		var wg sync.WaitGroup
		errs := make(chan error, 6)
		for i := range 4 {
			wg.Go(func() {
				_, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: fmt.Sprintf("cp-%d-%d", round, i)})
				errs <- err
			})
		}
		wg.Go(func() {
			_, err := f.s.RegisterSet(ctx, f.author, NewSet{CustomerID: f.kt.ID, IdempotencyKey: fmt.Sprintf("set-%d", round)})
			errs <- err
		})
		wg.Go(func() {
			_, err := f.customers.ChangeCode(ctx, f.admin, f.kt.ID, code)
			errs <- err
		})
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("putaran %d: %v", round, err)
			}
		}
		codes = append(codes, code)
	}

	// Per kode dan jenis dokumen, nomor berjalan urut dari 001 tanpa lubang.
	for _, code := range codes {
		for _, key := range []string{KeyPFD, KeyPFMEA, KeyCP} {
			rows, err := f.pool.Query(ctx, "SELECT sequence FROM document_numbers WHERE customer_code = $1 AND document_type_id = $2 ORDER BY sequence", code, f.types[key].ID)
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
					t.Fatalf("%s %s: nomor berjalan %v berlubang atau ganda", code, key, sequences)
				}
			}
		}
	}
	// 40 CP tunggal dan 10 set, semuanya dengan kode customer itu.
	var reserved, other int
	err := f.pool.QueryRow(ctx, "SELECT count(*) FILTER (WHERE document_type_id = $1 AND status = 'reserved'), count(*) FILTER (WHERE customer_code <> ALL($2)) FROM document_numbers",
		f.types[KeyCP].ID, codes).Scan(&reserved, &other)
	if err != nil || reserved != 50 || other != 0 {
		t.Errorf("CP Reserved %d (ingin 50), kode lain %d, %v", reserved, other, err)
	}
}

// TestRegisterAfterCodeChange: setelah kode customer diganti, nomor baru
// memakai kode baru dan mulai lagi dari nomor pertama; nomor lama tetap
// memakai kode lama (E-11, D-88, D-93).
func TestRegisterAfterCodeChange(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	old := f.register(t, f.author, f.kt, "sebelum")
	if _, err := f.customers.ChangeCode(ctx, f.admin, f.kt.ID, "KX"); err != nil {
		t.Fatal(err)
	}
	if got := f.register(t, f.author, f.kt, "sesudah"); got != "SC-KX-001" {
		t.Errorf("nomor setelah kode diganti %q, ingin SC-KX-001", got)
	}
	var stored string
	if err := f.pool.QueryRow(ctx, "SELECT number FROM document_numbers WHERE number = $1", old).Scan(&stored); err != nil || stored != "SC-KT-001" {
		t.Errorf("nomor lama %q, %v", stored, err)
	}
}

// TestRegisterScope: Project Leader hanya mendaftarkan nomor untuk customer di
// Project Group sendiri (D-71, D-81), dan akun tanpa peran Penyusun ditolak.
func TestRegisterScope(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	gr2, err := f.customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR2", Name: "Project Group 2"})
	if err != nil {
		t.Fatal(err)
	}
	ko, err := f.customers.Create(ctx, f.admin, customer.NewCustomer{Code: "KO", Name: "Customer KO", ProjectGroupID: gr2.ID})
	if err != nil {
		t.Fatal(err)
	}
	leader := newActor(t, f.pool, "pl.uji", access.AccountWork)
	if err := addScopedAuthor(t, f, leader); err != nil {
		t.Fatal(err)
	}
	approvals := approval.NewService(f.pool, access.NewChecker(f.pool))
	positions, err := approvals.ListPositions(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range positions {
		if p.Key == approval.KeyProjectLeader {
			if _, err := approvals.AddHolder(ctx, f.admin, p.ID, approval.NewHolder{PersonID: leader.PersonID, ProjectGroupID: f.group.ID}); err != nil {
				t.Fatal(err)
			}
		}
	}

	if got := f.register(t, leader, f.kt, "milik-gr1"); got != "SC-KT-001" {
		t.Errorf("Project Leader GR1 untuk KT: %q", got)
	}
	_, err = f.s.Register(ctx, leader, NewNumber{CustomerID: ko.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "milik-gr2"})
	if !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Project Leader GR1 untuk KO (GR2): %v, ingin ErrForbidden", err)
	}
	viewer := newActor(t, f.pool, "viewer.uji", access.AccountWork)
	_, err = f.s.Register(ctx, viewer, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "viewer"})
	if !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Viewer: %v, ingin ErrForbidden", err)
	}
	if _, err := f.s.NextNumber(ctx, viewer, f.kt.ID, f.types[KeyCP].ID); err != nil {
		t.Errorf("Viewer melihat nomor berikutnya (L-2): %v", err)
	}
}

// addScopedAuthor memberi akun peran Penyusun dengan cakupan Project Group sendiri.
func addScopedAuthor(t *testing.T, f numberFixture, actor access.Actor) error {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), "INSERT INTO account_roles (account_id, role_id, scope) VALUES ($1, $2, $3)",
		actor.AccountID, access.RoleAuthor, access.ScopeOwnProjectGroup)
	return err
}

// TestPatternChange: pola dan jumlah digit revisi adalah master data yang
// diubah Admin (R-NUM-02, E-12). Nomor baru memakai pola baru; nomor yang
// sudah terdaftar tetap.
func TestPatternChange(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	f.register(t, f.author, f.kt, "pola-lama")

	pattern, digits := "{customer}/{prefix}/{seq}", 3
	settings, err := f.s.UpdateSettings(ctx, f.admin, SettingsInput{FactoryCode: "S", Pattern: &pattern, RevisionDigits: &digits})
	if err != nil || settings.Pattern != pattern || settings.RevisionDigits != 3 || settings.SequenceFormat != SequenceFormatLetterBlock {
		t.Fatalf("UpdateSettings %+v, %v", settings, err)
	}
	if got := f.register(t, f.author, f.kt, "pola-baru"); got != "KT/SC/002" {
		t.Errorf("nomor dengan pola baru %q, ingin KT/SC/002", got)
	}
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM document_numbers WHERE number = 'SC-KT-001'").Scan(&count); err != nil || count != 1 {
		t.Errorf("nomor lama berubah: %d, %v", count, err)
	}

	bad, tooMany := "{prefix}{customer}{seq}", 4
	_, err = f.s.UpdateSettings(ctx, f.admin, SettingsInput{FactoryCode: "S", Pattern: &bad, RevisionDigits: &tooMany})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"pattern:invalid_format", "revision_digits:invalid_format"}) {
		t.Errorf("masalah %v", got)
	}
	// Tanpa Pattern dan RevisionDigits, keduanya tetap (D-111).
	settings, err = f.s.UpdateSettings(ctx, f.admin, SettingsInput{FactoryCode: "T"})
	if err != nil || settings.Pattern != pattern || settings.RevisionDigits != 3 || settings.FactoryCode != "T" {
		t.Errorf("UpdateSettings tanpa pola: %+v, %v", settings, err)
	}
	if _, err := f.s.UpdateSettings(ctx, f.author, SettingsInput{FactoryCode: "S"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun mengubah pola: %v, ingin ErrForbidden", err)
	}
}
