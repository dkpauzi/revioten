package auth

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// newAdminActor membuat Service dengan admin pertama, lalu login sebagai admin.
func newAdminActor(t *testing.T) (*Service, access.Actor) {
	t.Helper()
	s, _ := newServiceWithAdmin(t)
	actor, _, err := s.Login(t.Context(), adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	return s, actor
}

// conflictCodes mengambil "field:kode" dari *problem.ConflictError.
func conflictCodes(t *testing.T, err error) []string {
	t.Helper()
	var conflict *problem.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("ingin *problem.ConflictError, dapat %v", err)
	}
	var codes []string
	for _, p := range conflict.Problems {
		codes = append(codes, p.Field+":"+p.Code)
	}
	return codes
}

// TestCreateAccount: admin membuat akun kerja dengan password sementara.
// Pemiliknya wajib mengganti password saat login pertama (E-08).
func TestCreateAccount(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()

	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{
		PersonName: " Penyusun Uji ", EmployeeNumber: "K-001", Kind: KindWork, Login: "Penyusun.Uji",
		Roles: []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}},
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if info.Login != "penyusun.uji" || info.PersonName != "Penyusun Uji" || info.EmployeeNumber != "K-001" ||
		!info.MustChangePassword || info.Disabled || info.Kind != KindWork {
		t.Errorf("info %+v", info)
	}
	if !slices.Equal(info.Roles, []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}}) {
		t.Errorf("peran %+v", info.Roles)
	}
	if len(temporary) < minPasswordLength {
		t.Errorf("password sementara %d karakter, ingin minimal %d", len(temporary), minPasswordLength)
	}

	actor, token, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatalf("login dengan password sementara: %v", err)
	}
	if !actor.MustChangePassword {
		t.Error("akun baru belum wajib ganti password")
	}
	if _, err := s.ChangePassword(ctx, actor, temporary, "burung hantu di pohon mangga"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("sesi dengan password sementara masih berlaku: %v", err)
	}

	// Akun admin untuk orang yang sama (seperti PPAP Staff, D-72).
	adminInfo, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonID: info.PersonID, Kind: KindAdmin, Login: "admin.penyusun"})
	if err != nil {
		t.Fatalf("CreateAccount admin: %v", err)
	}
	if adminInfo.PersonID != info.PersonID ||
		!slices.Equal(adminInfo.Roles, []access.RoleGrant{{Role: access.RoleAdmin, Scope: access.ScopeAll}}) {
		t.Errorf("akun admin %+v", adminInfo)
	}
}

func TestCreateAccountRules(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	first, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Orang Pertama", EmployeeNumber: "K-1", Kind: KindWork, Login: "orang.pertama"})
	if err != nil {
		t.Fatal(err)
	}

	invalid := []struct {
		name string
		in   NewAccount
		want []string
	}{
		{"tanpa orang", NewAccount{Kind: KindWork, Login: "tanpa.orang"}, []string{"person_name:required"}},
		{"orang lama dan baru sekaligus", NewAccount{PersonID: first.PersonID, PersonName: "X", Kind: KindWork, Login: "dua.orang"}, []string{"person_id:not_allowed"}},
		{"orang yang tidak ada", NewAccount{PersonID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", Kind: KindWork, Login: "hantu.uji"}, []string{"person_id:invalid_format"}},
		{"jenis akun salah", NewAccount{PersonName: "X", Kind: "tamu", Login: "tamu.uji"}, []string{"kind:invalid_format"}},
		{"login salah", NewAccount{PersonName: "X", Kind: KindWork, Login: "a"}, []string{"login:invalid_format"}},
		{"akun admin dengan peran kerja", NewAccount{PersonName: "X", Kind: KindAdmin, Login: "admin.kerja", Roles: []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeAll}}}, []string{"roles:not_allowed"}},
		{"akun kerja dengan peran Admin", NewAccount{PersonName: "X", Kind: KindWork, Login: "kerja.admin", Roles: []access.RoleGrant{{Role: access.RoleAdmin, Scope: access.ScopeAll}}}, []string{"roles:not_allowed"}},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.CreateAccount(ctx, admin, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	conflicts := []struct {
		name string
		in   NewAccount
		want []string
	}{
		{"login sudah dipakai", NewAccount{PersonName: "X", Kind: KindWork, Login: "orang.pertama"}, []string{"login:taken"}},
		{"orang sudah punya akun kerja", NewAccount{PersonID: first.PersonID, Kind: KindWork, Login: "orang.pertama2"}, []string{"kind:taken"}},
		{"nomor karyawan sudah dipakai", NewAccount{PersonName: "X", EmployeeNumber: "K-1", Kind: KindWork, Login: "orang.kedua"}, []string{"employee_number:taken"}},
	}
	for _, tt := range conflicts {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.CreateAccount(ctx, admin, tt.in)
			if got := conflictCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
}

// TestAccountManagementRequiresAdmin: semua operasi pengelolaan akun butuh hak
// A-1, yang hanya dimiliki akun admin (U3).
func TestAccountManagementRequiresAdmin(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	allWork := []access.RoleGrant{
		{Role: access.RoleAuthor, Scope: access.ScopeAll}, {Role: access.RoleTemplateMaintainer, Scope: access.ScopeAll},
		{Role: access.RolePFMEACPChecker, Scope: access.ScopeAll}, {Role: access.RoleVerifier, Scope: access.ScopeAll},
		{Role: access.RoleDocumentControl, Scope: access.ScopeAll}, {Role: access.RoleMigration, Scope: access.ScopeAll},
	}
	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Pekerja Uji", Kind: KindWork, Login: "pekerja.uji", Roles: allWork})
	if err != nil {
		t.Fatal(err)
	}
	worker, _, err := s.Login(ctx, "pekerja.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}

	checks := map[string]func() error{
		"ListAccounts": func() error { _, err := s.ListAccounts(ctx, worker); return err },
		"CreateAccount": func() error {
			_, _, err := s.CreateAccount(ctx, worker, NewAccount{PersonName: "X", Kind: KindWork, Login: "x.uji"})
			return err
		},
		"ResetPassword":  func() error { _, err := s.ResetPassword(ctx, worker, admin.AccountID); return err },
		"DisableAccount": func() error { _, err := s.DisableAccount(ctx, worker, admin.AccountID); return err },
		"EnableAccount":  func() error { _, err := s.EnableAccount(ctx, worker, info.ID); return err },
	}
	for name, check := range checks {
		if err := check(); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s oleh akun kerja: %v, ingin ErrForbidden", name, err)
		}
	}
}

func TestListAccounts(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	if _, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji",
		Roles: []access.RoleGrant{{Role: access.RoleVerifier, Scope: access.ScopeAll}}}); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListAccounts(ctx, admin)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	var logins []string
	for _, a := range list {
		logins = append(logins, a.Login)
	}
	if !slices.Equal(logins, []string{adminLogin, "penyusun.uji"}) {
		t.Errorf("login %v", logins)
	}
	if !slices.Equal(list[1].Roles, []access.RoleGrant{{Role: access.RoleVerifier, Scope: access.ScopeAll}}) {
		t.Errorf("peran %+v", list[1].Roles)
	}
}

// TestResetPassword: reset oleh admin memasang password sementara, mengakhiri
// semua sesi akun itu, dan membuka kuncinya.
func TestResetPassword(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"})
	if err != nil {
		t.Fatal(err)
	}
	worker, token, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangePassword(ctx, worker, temporary, "burung hantu di pohon mangga"); err != nil {
		t.Fatal(err)
	}
	for range lockThreshold {
		_, _, _ = s.Login(ctx, "penyusun.uji", "bukan password yang benar")
	}

	newTemporary, err := s.ResetPassword(ctx, admin, info.ID)
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("sesi lama masih berlaku: %v", err)
	}
	actor, _, err := s.Login(ctx, "penyusun.uji", newTemporary)
	if err != nil {
		t.Fatalf("login setelah reset (kunci harus terbuka): %v", err)
	}
	if !actor.MustChangePassword {
		t.Error("setelah reset belum wajib ganti password")
	}

	// Admin mengganti password sendiri lewat halaman ganti password, bukan reset.
	// UUID dalam huruf besar menunjuk akun yang sama, jadi juga ditolak.
	for _, id := range []string{admin.AccountID, strings.ToUpper(admin.AccountID)} {
		_, err = s.ResetPassword(ctx, admin, id)
		if got := problemCodes(t, err); !slices.Equal(got, []string{"account_id:not_allowed"}) {
			t.Errorf("reset akun sendiri (%s): %v", id, got)
		}
	}
	if _, err := s.ResetPassword(ctx, admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("akun tidak ada: %v, ingin problem.ErrNotFound", err)
	}
}

func TestDisableAndEnableAccount(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"})
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}

	disabled, err := s.DisableAccount(ctx, admin, info.ID)
	if err != nil {
		t.Fatalf("DisableAccount: %v", err)
	}
	if !disabled.Disabled {
		t.Error("akun belum nonaktif")
	}
	// Akun nonaktif: semua sesinya berakhir (ASVS 7.4.2) dan tidak bisa login.
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("sesi akun nonaktif masih berlaku: %v", err)
	}
	if _, _, err := s.Login(ctx, "penyusun.uji", temporary); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("akun nonaktif bisa login: %v", err)
	}

	enabled, err := s.EnableAccount(ctx, admin, info.ID)
	if err != nil {
		t.Fatalf("EnableAccount: %v", err)
	}
	if enabled.Disabled {
		t.Error("akun masih nonaktif")
	}
	if _, _, err := s.Login(ctx, "penyusun.uji", temporary); err != nil {
		t.Errorf("login setelah diaktifkan lagi: %v", err)
	}

	// Akun sendiri tidak bisa dinonaktifkan. Admin kedua dibuat dulu, supaya
	// penolakan ini pasti dari aturan akun sendiri, bukan aturan admin terakhir.
	// UUID dalam huruf besar menunjuk akun yang sama, jadi juga ditolak.
	second, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Admin Kedua", Kind: KindAdmin, Login: "admin.kedua"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{admin.AccountID, strings.ToUpper(admin.AccountID)} {
		_, err = s.DisableAccount(ctx, admin, id)
		if got := problemCodes(t, err); !slices.Equal(got, []string{"account_id:not_allowed"}) {
			t.Errorf("nonaktifkan akun sendiri (%s): %v", id, got)
		}
	}
	if _, err := s.DisableAccount(ctx, admin, second.ID); err != nil {
		t.Errorf("menonaktifkan admin kedua (masih ada admin lain): %v", err)
	}
	// Sekarang admin ini admin aktif terakhir (D-73); TestDisableLastAdminConcurrent
	// menguji penolakannya saat dua admin saling menonaktifkan.
}

// TestDisableLastAdminConcurrent: dua admin yang saling menonaktifkan pada saat
// bersamaan tidak boleh membuat sistem tanpa admin aktif (D-73).
func TestDisableLastAdminConcurrent(t *testing.T) {
	s, first := newAdminActor(t)
	ctx := t.Context()
	_, temporary, err := s.CreateAccount(ctx, first, NewAccount{PersonName: "Admin Kedua", Kind: KindAdmin, Login: "admin.kedua"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := s.Login(ctx, "admin.kedua", temporary)
	if err != nil {
		t.Fatal(err)
	}
	// Admin kedua harus ganti password dulu sebelum bisa mengelola akun.
	if _, err := s.ChangePassword(ctx, second, temporary, "burung hantu di pohon mangga"); err != nil {
		t.Fatal(err)
	}
	second.MustChangePassword = false

	// Tanpa kunci advisory, dua transaksi bisa sama-sama menghitung "2 admin
	// aktif" sebelum salah satunya selesai. Kait ini menahan setiap transaksi
	// sampai transaksi lain tiba di titik yang sama (paling lama 500 ms), jadi
	// keadaan berbahaya itu pasti terjadi bila kuncinya tidak ada. Dengan
	// kunci, transaksi kedua tertahan di kunci, dan yang pertama lanjut setelah
	// 500 ms.
	var reached atomic.Int32
	s.afterAdminCount = func() {
		reached.Add(1)
		deadline := time.Now().Add(500 * time.Millisecond)
		for reached.Load() < 2 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Go(func() { _, errs[0] = s.DisableAccount(ctx, first, second.AccountID) })
	wg.Go(func() { _, errs[1] = s.DisableAccount(ctx, second, first.AccountID) })
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Errorf("%d penonaktifan berhasil, ingin tepat 1: %v", succeeded, errs)
	}
	var enabled int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE kind = 'admin' AND disabled_at IS NULL").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Errorf("%d akun admin aktif, ingin 1", enabled)
	}
}

func TestGetAccount(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	created, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetAccount(ctx, admin, created.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Login != "penyusun.uji" || got.ID != created.ID {
		t.Errorf("akun %+v", got)
	}
	if _, err := s.GetAccount(ctx, admin, "bukan-uuid"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("ID yang salah: %v, ingin problem.ErrNotFound", err)
	}
	worker, _, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAccount(ctx, worker, created.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("GetAccount oleh akun kerja: %v, ingin ErrForbidden", err)
	}
}

// TestSetRoles: admin mengganti seluruh peran akun kerja (A-1), dan peran
// barunya langsung berlaku. Aturan yang tidak bisa dimatikan tetap berlaku
// (R-SEC-11 butir 8, D-72, D-73).
func TestSetRoles(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	checker := access.NewChecker(s.pool)
	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"})
	if err != nil {
		t.Fatal(err)
	}
	worker, _, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}

	grants := []access.RoleGrant{
		{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup},
		{Role: access.RoleVerifier, Scope: access.ScopeAll},
	}
	if err := s.SetRoles(ctx, admin, info.ID, grants); err != nil {
		t.Fatalf("SetRoles: %v", err)
	}
	if err := checker.Authorize(ctx, worker, access.VerifyChanges, access.Resource{CustomerID: "x"}); err != nil {
		t.Errorf("peran baru belum berlaku: %v", err)
	}
	// Peran diganti seluruhnya, bukan ditambah.
	if err := s.SetRoles(ctx, admin, info.ID, nil); err != nil {
		t.Fatalf("SetRoles kosong: %v", err)
	}
	if err := checker.Authorize(ctx, worker, access.VerifyChanges, access.Resource{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("peran lama masih berlaku: %v", err)
	}

	tests := []struct {
		name    string
		account string
		grants  []access.RoleGrant
	}{
		{"akun admin tidak bisa diubah perannya", admin.AccountID, []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeAll}}},
		{"akun kerja tidak boleh memegang Admin", info.ID, []access.RoleGrant{{Role: access.RoleAdmin, Scope: access.ScopeAll}}},
		{"Viewer tidak diberikan karena sudah dimiliki semua", info.ID, []access.RoleGrant{{Role: access.RoleViewer, Scope: access.ScopeAll}}},
		{"peran yang tidak ada", info.ID, []access.RoleGrant{{Role: "approver", Scope: access.ScopeAll}}},
		{"peran ganda", info.ID, []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeAll}, {Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}}},
		{"cakupan group hanya untuk Penyusun dan Verifikator", info.ID, []access.RoleGrant{{Role: access.RoleDocumentControl, Scope: access.ScopeOwnProjectGroup}}},
		{"cakupan yang tidak dikenal", info.ID, []access.RoleGrant{{Role: access.RoleAuthor, Scope: "gr1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.SetRoles(ctx, admin, tt.account, tt.grants)
			var invalid *problem.ValidationError
			if !errors.As(err, &invalid) {
				t.Errorf("ingin *problem.ValidationError, dapat %v", err)
			}
		})
	}

	if err := s.SetRoles(ctx, worker, info.ID, grants); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("SetRoles oleh akun kerja: %v, ingin ErrForbidden", err)
	}
	if err := s.SetRoles(ctx, admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", nil); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("akun yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
	// ID yang bukan UUID juga berarti akun tidak ada, bukan error database (500).
	if err := s.SetRoles(ctx, admin, "bukan-uuid", nil); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("ID bukan UUID: %v, ingin problem.ErrNotFound", err)
	}
}
