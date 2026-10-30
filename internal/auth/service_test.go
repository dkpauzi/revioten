package auth

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// TestMain menyalakan PostgreSQL untuk test di paket ini (lihat dbtest).
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// testClock adalah jam palsu yang bisa dimajukan, supaya batas waktu sesi dan
// kunci akun bisa dites tanpa menunggu.
type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// Data akun admin yang dipakai banyak test. Nama fiktif (CLAUDE.md bagian 7).
const (
	adminName     = "Admin Uji"
	adminLogin    = "admin.uji"
	adminPassword = "kuda terbang di atas sawah"
)

// newTestService membuat Service dengan database baru dan jam palsu.
func newTestService(t *testing.T) (*Service, *testClock) {
	t.Helper()
	clock := &testClock{now: time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC)}
	pool := dbtest.NewPool(t)
	s := NewService(pool, access.NewChecker(pool))
	s.now = clock.Now
	return s, clock
}

// newServiceWithAdmin membuat Service dan satu akun admin siap dipakai login.
func newServiceWithAdmin(t *testing.T) (*Service, *testClock) {
	t.Helper()
	s, clock := newTestService(t)
	if _, err := s.CreateFirstAdmin(t.Context(), adminName, adminLogin, adminPassword); err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	return s, clock
}

// problemCodes mengambil "field:kode" dari *problem.ValidationError.
func problemCodes(t *testing.T, err error) []string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("ingin *problem.ValidationError, dapat %v", err)
	}
	var codes []string
	for _, p := range invalid.Problems {
		codes = append(codes, p.Field+":"+p.Code)
	}
	return codes
}

func TestCreateFirstAdmin(t *testing.T) {
	s, _ := newTestService(t)
	ctx := t.Context()

	// Input yang salah ditolak sekaligus untuk semua field.
	_, err := s.CreateFirstAdmin(ctx, "  ", "A B", "pendek")
	want := []string{"name:required", "login:invalid_format", "password:too_short"}
	if got := problemCodes(t, err); !slices.Equal(got, want) {
		t.Errorf("masalah %v, ingin %v", got, want)
	}

	actor, err := s.CreateFirstAdmin(ctx, " "+adminName+" ", " Admin.Uji ", adminPassword)
	if err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	if actor.AccountKind != KindAdmin || actor.Login != adminLogin || actor.PersonName != adminName || actor.Type != access.ActorHuman {
		t.Errorf("actor %+v, ingin akun admin %s milik %s", actor, adminLogin, adminName)
	}
	// Akun admin langsung memegang peran Admin (D-72, M1.2).
	if err := access.NewChecker(s.pool).Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		t.Errorf("admin pertama tidak punya hak A-1: %v", err)
	}

	// Perintah ini hanya bisa dipakai sekali (E-07).
	if _, err := s.CreateFirstAdmin(ctx, "Admin Lain", "admin.lain", adminPassword); !errors.Is(err, ErrAdminExists) {
		t.Errorf("CreateFirstAdmin kedua: %v, ingin ErrAdminExists", err)
	}
}

// TestCreateFirstAdminConcurrent: sepuluh perintah bersamaan tetap hanya
// menghasilkan satu akun admin, karena kunci advisory di database.
func TestCreateFirstAdminConcurrent(t *testing.T) {
	s, _ := newTestService(t)
	ctx := t.Context()

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range errs {
		wg.Go(func() {
			login := "admin.paralel" + string(rune('a'+i))
			_, errs[i] = s.CreateFirstAdmin(ctx, "Admin Paralel", login, adminPassword)
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, ErrAdminExists):
			t.Errorf("error tak terduga: %v", err)
		}
	}
	if succeeded != 1 {
		t.Errorf("%d perintah berhasil, ingin tepat 1", succeeded)
	}
	var admins int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE kind = 'admin'").Scan(&admins); err != nil {
		t.Fatal(err)
	}
	if admins != 1 {
		t.Errorf("ada %d akun admin, ingin 1", admins)
	}
}

func TestLogin(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()

	// Nama login tidak membedakan huruf besar dan kecil, dan spasi di tepi dibuang.
	actor, token, err := s.Login(ctx, " ADMIN.UJI ", adminPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" || actor.Login != adminLogin || actor.AccountKind != KindAdmin {
		t.Errorf("Login = %+v, token %q", actor, token)
	}

	// Database hanya menyimpan hash token, bukan tokennya.
	var stored []byte
	if err := s.pool.QueryRow(ctx, "SELECT token_hash FROM sessions").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, hashToken(token)) || bytes.Contains(stored, []byte(token)) {
		t.Error("tabel sessions tidak berisi hash token")
	}

	got, err := s.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got != actor {
		t.Errorf("Authenticate = %+v, ingin %+v", got, actor)
	}
}

// TestLoginRejected: semua kegagalan memberi error yang sama, supaya penyerang
// tidak bisa menebak nama login mana yang ada.
func TestLoginRejected(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()

	tests := map[string][2]string{
		"password salah":     {adminLogin, "bukan password yang benar"},
		"nama login tak ada": {"tidak.ada", adminPassword},
		"semua kosong":       {"", ""},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := s.Login(ctx, input[0], input[1]); !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login: %v, ingin ErrInvalidCredentials", err)
			}
		})
	}
}

// TestLoginLockout: setelah 10 kali gagal berturut-turut, akun dikunci 15 menit.
// Selama terkunci, password yang benar pun ditolak (E-08, D-85).
func TestLoginLockout(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()

	for range lockThreshold {
		if _, _, err := s.Login(ctx, adminLogin, "bukan password yang benar"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login salah: %v", err)
		}
	}
	if _, _, err := s.Login(ctx, adminLogin, adminPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login benar saat terkunci: %v, ingin ErrInvalidCredentials", err)
	}

	clock.Advance(lockDuration + time.Second)
	if _, _, err := s.Login(ctx, adminLogin, adminPassword); err != nil {
		t.Fatalf("Login setelah kunci habis: %v", err)
	}

	// Login berhasil mengembalikan hitungan gagal ke nol: 9 kali gagal tidak mengunci.
	for range lockThreshold - 1 {
		_, _, _ = s.Login(ctx, adminLogin, "bukan password yang benar")
	}
	if _, _, err := s.Login(ctx, adminLogin, adminPassword); err != nil {
		t.Errorf("Login setelah 9 kali gagal: %v, ingin berhasil", err)
	}
}

// TestLoginDisabledAfterLimit: kegagalan ke-100 berturut-turut menonaktifkan
// akun kerja (NIST SP 800-63B-4: paling banyak 100); hanya admin yang bisa
// membukanya lagi. Akun admin tidak dinonaktifkan, hanya dikunci 15 menit,
// supaya orang yang tahu nama login admin tidak bisa mengunci admin terakhir
// selamanya (E-22 opsi c, D-101).
func TestLoginDisabledAfterLimit(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()
	admin, _, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	_, temporary, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"})
	if err != nil {
		t.Fatal(err)
	}
	// Kedua akun tinggal satu kegagalan sebelum batas penonaktifan.
	if _, err := s.pool.Exec(ctx, "UPDATE accounts SET failed_logins = $1", disableThreshold-1); err != nil {
		t.Fatal(err)
	}

	_, _, _ = s.Login(ctx, "penyusun.uji", "bukan password yang benar")
	_, _, _ = s.Login(ctx, adminLogin, "bukan password yang benar")
	clock.Advance(24 * time.Hour)

	if _, _, err := s.Login(ctx, "penyusun.uji", temporary); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login akun kerja nonaktif: %v, ingin ErrInvalidCredentials", err)
	}
	if _, _, err := s.Login(ctx, adminLogin, adminPassword); err != nil {
		t.Errorf("Login admin setelah kunci habis: %v, ingin berhasil", err)
	}
	disabled := map[string]bool{}
	rows, err := s.pool.Query(ctx, "SELECT login, disabled_at IS NOT NULL FROM accounts")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var login string
		var off bool
		if err := rows.Scan(&login, &off); err != nil {
			t.Fatal(err)
		}
		disabled[login] = off
	}
	if !disabled["penyusun.uji"] || disabled[adminLogin] {
		t.Errorf("nonaktif: %v, ingin hanya penyusun.uji", disabled)
	}
}

// TestResetAdminPassword: perintah reset-admin-password di komputer server
// membuka akun admin yang lupa password, terkunci, atau nonaktif, memasang
// password sementara yang wajib diganti, dan mencatatnya dengan aktor sistem
// (E-22 opsi a, D-101). Akun kerja tetap direset admin lewat aplikasi.
func TestResetAdminPassword(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()
	admin, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"}); err != nil {
		t.Fatal(err)
	}
	// Admin terkunci dan nonaktif, seperti setelah banyak percobaan gagal.
	if _, err := s.pool.Exec(ctx, "UPDATE accounts SET failed_logins = 120, locked_until = now() + interval '1 hour', disabled_at = now() WHERE login = $1", adminLogin); err != nil {
		t.Fatal(err)
	}

	temporary, err := s.ResetAdminPassword(ctx, " Admin.Uji ")
	if err != nil {
		t.Fatalf("ResetAdminPassword: %v", err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("sesi lama masih berlaku: %v", err)
	}
	actor, _, err := s.Login(ctx, adminLogin, temporary)
	if err != nil {
		t.Fatalf("Login dengan password sementara: %v", err)
	}
	if !actor.MustChangePassword {
		t.Error("password sementara harus wajib diganti")
	}
	var actorType, details string
	err = s.pool.QueryRow(ctx, "SELECT actor_type, details FROM audit_log WHERE action = $1 ORDER BY seq DESC LIMIT 1",
		audit.ActionPasswordReset).Scan(&actorType, &details)
	if err != nil {
		t.Fatal(err)
	}
	if actorType != string(access.ActorSystem) || details != `{"enabled":"true","via":"reset-admin-password"}` {
		t.Errorf("entri audit: %s %s", actorType, details)
	}
	if strings.Contains(details, temporary) {
		t.Error("password sementara masuk audit log")
	}

	if _, err := s.ResetAdminPassword(ctx, "penyusun.uji"); !errors.Is(err, ErrNotAdminAccount) {
		t.Errorf("akun kerja: %v, ingin ErrNotAdminAccount", err)
	}
	if _, err := s.ResetAdminPassword(ctx, "tidak.ada"); !errors.Is(err, ErrNotAdminAccount) {
		t.Errorf("login tidak ada: %v, ingin ErrNotAdminAccount", err)
	}
}

// TestSessionIdleTimeout: sesi berakhir setelah 30 menit tanpa aktivitas, dan
// setiap aktivitas memperpanjangnya.
func TestSessionIdleTimeout(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()
	_, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		clock.Advance(idleTimeout - time.Minute)
		if _, err := s.Authenticate(ctx, token); err != nil {
			t.Fatalf("Authenticate sebelum batas tidak aktif: %v", err)
		}
	}
	clock.Advance(idleTimeout)
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate setelah tidak aktif: %v, ingin ErrUnauthenticated", err)
	}
}

// TestSessionAbsoluteTimeout: walau terus dipakai, sesi berakhir 8 jam setelah login.
func TestSessionAbsoluteTimeout(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()
	_, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}

	elapsed := time.Duration(0)
	for elapsed+20*time.Minute < absoluteTimeout {
		clock.Advance(20 * time.Minute)
		elapsed += 20 * time.Minute
		if _, err := s.Authenticate(ctx, token); err != nil {
			t.Fatalf("Authenticate setelah %v: %v", elapsed, err)
		}
	}
	clock.Advance(absoluteTimeout - elapsed)
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate setelah 8 jam: %v, ingin ErrUnauthenticated", err)
	}
}

func TestLogout(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()
	_, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}

	actor, err := s.Authenticate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Logout(ctx, actor, token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate setelah logout: %v, ingin ErrUnauthenticated", err)
	}
	// Logout dengan token yang tidak dikenal tidak dianggap error.
	if err := s.Logout(ctx, actor, "token-tidak-dikenal"); err != nil {
		t.Errorf("Logout token tak dikenal: %v", err)
	}
}

// TestAuthenticateDisabledAccount: sesi dari akun yang dinonaktifkan langsung
// tidak berlaku (ASVS 7.4.2).
func TestAuthenticateDisabledAccount(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()
	_, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	queries := db.New(s.pool)
	now := clock.Now()
	if err := queries.DisableAccount(ctx, db.DisableAccountParams{ID: mustAccountID(t, s), DisabledAt: &now}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate akun nonaktif: %v, ingin ErrUnauthenticated", err)
	}
}

func TestChangePassword(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()
	actor, token, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	// Sesi kedua, misalnya di browser lain.
	_, otherToken, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, "UPDATE accounts SET must_change_password = true"); err != nil {
		t.Fatal(err)
	}

	const newPassword = "burung hantu di pohon mangga"
	_, err = s.ChangePassword(ctx, actor, "bukan password lama", "pendek")
	want := []string{"current_password:incorrect", "new_password:too_short"}
	if got := problemCodes(t, err); !slices.Equal(got, want) {
		t.Errorf("masalah %v, ingin %v", got, want)
	}

	newToken, err := s.ChangePassword(ctx, actor, adminPassword, newPassword)
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	// Semua sesi lama berakhir; sesi baru menggantikan sesi yang dipakai (ASVS 7.4.3).
	for _, old := range []string{token, otherToken} {
		if _, err := s.Authenticate(ctx, old); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("sesi lama masih berlaku: %v", err)
		}
	}
	got, err := s.Authenticate(ctx, newToken)
	if err != nil {
		t.Fatalf("Authenticate token baru: %v", err)
	}
	if got.MustChangePassword {
		t.Error("tanda wajib ganti password belum dihapus")
	}
	if _, _, err := s.Login(ctx, adminLogin, adminPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("password lama masih bisa dipakai: %v", err)
	}
	if _, _, err := s.Login(ctx, adminLogin, newPassword); err != nil {
		t.Errorf("password baru ditolak: %v", err)
	}
}

// mustAccountID mengambil ID satu-satunya akun di database test.
func mustAccountID(t *testing.T, s *Service) string {
	t.Helper()
	var id string
	if err := s.pool.QueryRow(t.Context(), "SELECT id FROM accounts").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
