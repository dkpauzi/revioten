package numbering

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// newActor membuat orang dan akun langsung lewat query, dengan peran tertentu.
func newActor(t *testing.T, pool *pgxpool.Pool, login, kind string, roles ...string) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: role, Scope: access.ScopeAll}); err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{Type: access.ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login, PersonID: person.ID}
}

// fixture: database baru dan akun admin (A-2).
type fixture struct {
	s     *Service
	pool  *pgxpool.Pool
	admin access.Actor
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.NewPool(t)
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	return fixture{s: NewService(pool, access.NewChecker(pool)), pool: pool, admin: admin}
}

// problemCodes mengambil "field:kode" dari penolakan input.
func problemCodes(t *testing.T, err error) []string {
	t.Helper()
	var problems []problem.Problem
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		problems = invalid.Problems
	case errors.As(err, &conflict):
		problems = conflict.Problems
	default:
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	var codes []string
	for _, p := range problems {
		codes = append(codes, p.Field+":"+p.Code)
	}
	return codes
}

// second mengambil error dari hasil dua nilai, supaya tabel test tetap ringkas.
func second[T any](_ T, err error) error { return err }

// prefixes menulis jenis dokumen sebagai "nama=awalan", urut seperti daftar.
func prefixes(t *testing.T, f fixture) []string {
	t.Helper()
	types, err := f.s.ListDocumentTypes(t.Context(), f.admin)
	if err != nil {
		t.Fatal(err)
	}
	var list []string
	for _, d := range types {
		list = append(list, d.Name+"="+d.Prefix)
	}
	return list
}

// TestDefaults: isi awal sesuai WI SW-ZZ-000 (E-12) dan R-NUM-01: kode pabrik
// S, jenis dokumen PFD, PFMEA, dan CP dengan awalan SH, SF, dan SC, serta kode
// khusus ZZ dan HSE.
func TestDefaults(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	settings, err := f.s.GetSettings(ctx, f.admin)
	if err != nil || settings.FactoryCode != "S" {
		t.Errorf("pengaturan nomor: %+v, %v", settings, err)
	}
	if got := prefixes(t, f); !slices.Equal(got, []string{"PFD=SH", "PFMEA=SF", "CP=SC"}) {
		t.Errorf("jenis dokumen: %v", got)
	}
	types, err := f.s.ListDocumentTypes(ctx, f.admin)
	if err != nil || types[0].Key != KeyPFD || types[1].Key != KeyPFMEA || types[2].Key != KeyCP {
		t.Errorf("key jenis dokumen: %+v, %v", types, err)
	}
	codes, err := f.s.ListSpecialCodes(ctx, f.admin)
	if err != nil || len(codes) != 2 || codes[0].Code != "HSE" || codes[1].Code != "ZZ" || codes[1].Name != "General/Common" {
		t.Errorf("kode khusus: %+v, %v", codes, err)
	}
}

// TestEditNumberParts: kriteria selesai M1.6 untuk bagian nomor. Admin
// mengubah kode pabrik, mengganti nama dan huruf jenis dokumen, dan menambah
// jenis dokumen lain dari WI. Awalan selalu dihitung dari keduanya (R-NUM-02).
func TestEditNumberParts(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	settings, err := f.s.SetFactoryCode(ctx, f.admin, " t ")
	if err != nil || settings.FactoryCode != "T" {
		t.Fatalf("SetFactoryCode: %+v, %v", settings, err)
	}
	if got := prefixes(t, f); !slices.Equal(got, []string{"PFD=TH", "PFMEA=TF", "CP=TC"}) {
		t.Errorf("awalan setelah kode pabrik diganti: %v", got)
	}

	types, err := f.s.ListDocumentTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	pfd := types[0]
	updated, err := f.s.UpdateDocumentType(ctx, f.admin, pfd.ID, DocumentTypeInput{Name: "Process Flow Diagram", Letter: "h"})
	if err != nil || updated.Name != "Process Flow Diagram" || updated.Prefix != "TH" || updated.Key != KeyPFD {
		t.Errorf("UpdateDocumentType: %+v, %v", updated, err)
	}
	wi, err := f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{Name: "Work Instruction", Letter: "w"})
	if err != nil || wi.Prefix != "TW" || wi.Key != "" {
		t.Errorf("CreateDocumentType: %+v, %v", wi, err)
	}
	if got := prefixes(t, f); !slices.Equal(got, []string{"Process Flow Diagram=TH", "PFMEA=TF", "CP=TC", "Work Instruction=TW"}) {
		t.Errorf("daftar setelah diubah: %v", got)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"kode pabrik dua huruf", second(f.s.SetFactoryCode(ctx, f.admin, "ST")), []string{"factory_code:invalid_format"}},
		{"kode pabrik kosong", second(f.s.SetFactoryCode(ctx, f.admin, " ")), []string{"factory_code:required"}},
		{"huruf dipakai jenis lain", second(f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{Name: "Inspection Standard", Letter: "F"})), []string{"letter:taken"}},
		{"nama dipakai jenis lain", second(f.s.UpdateDocumentType(ctx, f.admin, wi.ID, DocumentTypeInput{Name: "pfmea", Letter: "W"})), []string{"name:taken"}},
		{"huruf bukan satu huruf", second(f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{Name: "Inspection Standard", Letter: "IS"})), []string{"letter:invalid_format"}},
		{"isian kosong", second(f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{})), []string{"name:required", "letter:required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	if _, err := f.s.UpdateDocumentType(ctx, f.admin, "bukan-uuid", DocumentTypeInput{Name: "X", Letter: "X"}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("jenis dokumen yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
}

// TestSpecialCodes: kode khusus bisa ditambah dan diganti namanya. Kodenya
// tidak boleh sama dengan kode customer, sekarang atau dulu (E-11, ASUMSI E-37).
func TestSpecialCodes(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	customers := customer.NewService(f.pool, access.NewChecker(f.pool))
	group, err := customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	ab, err := customers.Create(ctx, f.admin, customer.NewCustomer{Code: "AB", Name: "Customer A", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := customers.ChangeCode(ctx, f.admin, ab.ID, "AC"); err != nil {
		t.Fatal(err)
	}

	qa, err := f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: " qa ", Name: "Quality general"})
	if err != nil || qa.Code != "QA" || qa.Name != "Quality general" {
		t.Fatalf("CreateSpecialCode: %+v, %v", qa, err)
	}
	renamed, err := f.s.RenameSpecialCode(ctx, f.admin, qa.ID, "Quality Assurance")
	if err != nil || renamed.Code != "QA" || renamed.Name != "Quality Assurance" {
		t.Errorf("RenameSpecialCode: %+v, %v", renamed, err)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"kode ganda", second(f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "zz", Name: "Lain"})), []string{"code:taken"}},
		{"kode customer sekarang", second(f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "AC", Name: "Lain"})), []string{"code:taken"}},
		{"kode customer dulu", second(f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "AB", Name: "Lain"})), []string{"code:taken"}},
		{"bentuk salah", second(f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "Q1", Name: "Lain"})), []string{"code:invalid_format"}},
		{"empat huruf", second(f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "ABCD", Name: "Lain"})), []string{"code:invalid_format"}},
		{"nama kosong", second(f.s.RenameSpecialCode(ctx, f.admin, qa.ID, "")), []string{"name:required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	// Sebaliknya: customer tidak bisa memakai kode khusus, juga yang baru ditambah.
	_, err = customers.Create(ctx, f.admin, customer.NewCustomer{Code: "qa", Name: "Customer Q", ProjectGroupID: group.ID})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"code:reserved"}) {
		t.Errorf("customer dengan kode khusus: %v", got)
	}
	_, err = customers.ChangeCode(ctx, f.admin, ab.ID, "QA")
	if got := problemCodes(t, err); !slices.Equal(got, []string{"code:reserved"}) {
		t.Errorf("ganti kode customer ke kode khusus: %v", got)
	}
}

// TestCodePositionConcurrent: kode khusus dan kode customer didaftarkan
// bersamaan dengan kode yang sama. Karena keduanya mengambil kunci 11003,
// tepat satu yang berhasil untuk setiap kode (ASUMSI E-37).
func TestCodePositionConcurrent(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	customers := customer.NewService(f.pool, access.NewChecker(f.pool))
	group, err := customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}

	const codes = 16
	specialErrs := make([]error, codes)
	customerErrs := make([]error, codes)
	var wg sync.WaitGroup
	for i := range codes {
		code := fmt.Sprintf("Q%c", 'A'+i)
		wg.Go(func() {
			_, specialErrs[i] = f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: code, Name: "Khusus " + code})
		})
		wg.Go(func() {
			_, customerErrs[i] = customers.Create(ctx, f.admin, customer.NewCustomer{Code: code, Name: "Customer " + code, ProjectGroupID: group.ID})
		})
	}
	wg.Wait()

	for i := range codes {
		special, cust := specialErrs[i] == nil, customerErrs[i] == nil
		if special == cust {
			t.Errorf("kode Q%c: kode khusus berhasil %v, customer berhasil %v; ingin tepat satu (%v, %v)",
				'A'+i, special, cust, specialErrs[i], customerErrs[i])
		}
	}
}

// TestCodeRegistrationWaitsForLock: penambahan kode khusus dan customer baru
// sama-sama menunggu kunci 11003. Selama transaksi lain memegang kunci itu,
// keduanya tertahan; setelah kunci dilepas, keduanya selesai.
func TestCodeRegistrationWaitsForLock(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	customers := customer.NewService(f.pool, access.NewChecker(f.pool))
	group, err := customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}

	holder, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if err := db.New(holder).LockNumberCodes(ctx); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 2)
	go func() {
		_, err := f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "QY", Name: "Khusus QY"})
		done <- err
	}()
	go func() {
		_, err := customers.Create(ctx, f.admin, customer.NewCustomer{Code: "QZ", Name: "Customer QZ", ProjectGroupID: group.ID})
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("pendaftaran selesai padahal kunci 11003 masih dipegang: %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Errorf("setelah kunci dilepas: %v", err)
		}
	}
}

// TestPermissions: bagian nomor hanya diubah Admin (A-2, UC-05). Semua akun
// bisa melihatnya (ASUMSI E-24).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleTemplateMaintainer)
	types, err := f.s.ListDocumentTypes(ctx, author)
	if err != nil {
		t.Fatalf("ListDocumentTypes oleh akun kerja: %v", err)
	}
	codes, err := f.s.ListSpecialCodes(ctx, author)
	if err != nil {
		t.Fatalf("ListSpecialCodes oleh akun kerja: %v", err)
	}
	if _, err := f.s.GetSettings(ctx, author); err != nil {
		t.Errorf("GetSettings oleh akun kerja: %v", err)
	}

	for name, err := range map[string]error{
		"SetFactoryCode":     second(f.s.SetFactoryCode(ctx, author, "T")),
		"CreateDocumentType": second(f.s.CreateDocumentType(ctx, author, DocumentTypeInput{Name: "Work Instruction", Letter: "W"})),
		"UpdateDocumentType": second(f.s.UpdateDocumentType(ctx, author, types[0].ID, DocumentTypeInput{Name: "X", Letter: "H"})),
		"CreateSpecialCode":  second(f.s.CreateSpecialCode(ctx, author, NewSpecialCode{Code: "QA", Name: "X"})),
		"RenameSpecialCode":  second(f.s.RenameSpecialCode(ctx, author, codes[0].ID, "X")),
	} {
		if !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s oleh akun kerja: %v, ingin ErrForbidden", name, err)
		}
	}
}

// TestAudit: setiap perubahan tercatat dengan nilai lama dan baru. Simpan
// tanpa perubahan tidak dicatat.
func TestAudit(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	last := func() db.AuditLog {
		t.Helper()
		rows, err := db.New(f.pool).ListAuditChain(ctx)
		if err != nil || len(rows) == 0 {
			t.Fatalf("audit log: %d entri, %v", len(rows), err)
		}
		return rows[len(rows)-1]
	}
	check := func(what, action, details string) {
		t.Helper()
		if r := last(); r.Action != action || r.Details != details {
			t.Errorf("%s: %s %s, ingin %s %s", what, r.Action, r.Details, action, details)
		}
	}

	if _, err := f.s.SetFactoryCode(ctx, f.admin, "T"); err != nil {
		t.Fatal(err)
	}
	check("kode pabrik", audit.ActionNumberSettingsUpdated, `{"factory_code":"S → T"}`)
	types, err := f.s.ListDocumentTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.UpdateDocumentType(ctx, f.admin, types[2].ID, DocumentTypeInput{Name: "Control Plan", Letter: "C"}); err != nil {
		t.Fatal(err)
	}
	check("jenis dokumen diubah", audit.ActionDocumentTypeUpdated, `{"name":"CP → Control Plan"}`)
	if _, err := f.s.CreateDocumentType(ctx, f.admin, DocumentTypeInput{Name: "Work Instruction", Letter: "W"}); err != nil {
		t.Fatal(err)
	}
	check("jenis dokumen ditambah", audit.ActionDocumentTypeCreated, `{"letter":"W","name":"Work Instruction"}`)
	qa, err := f.s.CreateSpecialCode(ctx, f.admin, NewSpecialCode{Code: "QA", Name: "Quality"})
	if err != nil {
		t.Fatal(err)
	}
	check("kode khusus ditambah", audit.ActionSpecialCodeCreated, `{"code":"QA","name":"Quality"}`)
	if _, err := f.s.RenameSpecialCode(ctx, f.admin, qa.ID, "Quality Assurance"); err != nil {
		t.Fatal(err)
	}
	check("kode khusus diganti nama", audit.ActionSpecialCodeUpdated, `{"name":"Quality → Quality Assurance"}`)

	// Tanpa perubahan: tidak ada entri baru.
	before := last().Seq
	if _, err := f.s.SetFactoryCode(ctx, f.admin, "T"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.RenameSpecialCode(ctx, f.admin, qa.ID, "Quality Assurance"); err != nil {
		t.Fatal(err)
	}
	if after := last().Seq; after != before {
		t.Errorf("simpan tanpa perubahan menambah entri: %d → %d", before, after)
	}
}
