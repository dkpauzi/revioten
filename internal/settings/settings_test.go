package settings

import (
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
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

// TestSignDates: tanggal tanda tangan per slot bisa dimatikan dan saat ini
// mati (D-52). Sejak D-209 saklar ini pengaturan dokumen yang diubah PPAP
// Engineer (E-11), bukan admin; semua akun bisa membacanya.
func TestSignDates(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	engineer := newActor(t, pool, "ppap.uji", access.AccountWork, access.RolePPAPEngineer)
	worker := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)

	got, err := s.Get(ctx, worker)
	if err != nil || got.SignDatesEnabled {
		t.Fatalf("isi awal: %+v, %v; ingin mati", got, err)
	}
	on := true
	for _, actor := range []access.Actor{worker, admin} {
		if _, err := s.Apply(ctx, actor, Update{SignDatesEnabled: &on}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s: %v, ingin ErrForbidden", actor.Login, err)
		}
	}
	// Tanpa saklar pun hak tetap diperiksa.
	if _, err := s.Apply(ctx, worker, Update{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja tanpa saklar: %v, ingin ErrForbidden", err)
	}
	got, err = s.Apply(ctx, engineer, Update{SignDatesEnabled: &on})
	if err != nil || !got.SignDatesEnabled {
		t.Fatalf("Apply: %+v, %v", got, err)
	}
	rows, err := db.New(pool).ListAuditChain(ctx)
	if err != nil || len(rows) != 1 || rows[0].Action != audit.ActionSystemSettingsUpdated ||
		rows[0].Details != `{"sign_dates_enabled":"false → true"}` {
		t.Errorf("audit log: %+v, %v", rows, err)
	}
	// Simpan tanpa perubahan, atau tanpa saklar, tidak dicatat.
	if _, err := s.Apply(ctx, engineer, Update{SignDatesEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	if kept, err := s.Apply(ctx, engineer, Update{}); err != nil || !kept.SignDatesEnabled {
		t.Errorf("tanpa saklar: %+v, %v; ingin tetap menyala", kept, err)
	}
	if rows, _ := db.New(pool).ListAuditChain(ctx); len(rows) != 1 {
		t.Errorf("simpan tanpa perubahan menambah entri: %d", len(rows))
	}
}

// TestSettingsSplit: pengaturan terbagi dua (D-209). Pengaturan dokumen (tanggal
// di slot, QR, nomor form submission) milik PPAP Engineer (E-11); pengaman
// (mode migrasi, aturan penyetujuan form) tetap milik admin (A-3). Permintaan
// yang mencampur keduanya butuh kedua hak, jadi ditolak untuk siapa pun.
func TestSettingsSplit(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	engineer := newActor(t, pool, "ppap.uji", access.AccountWork, access.RolePPAPEngineer)
	on, number, anyMode := true, "QR-ENG-24-K015 (Rev.01)", ApprovalAny

	for name, in := range map[string]Update{
		"QR":         {QREnabled: &on},
		"nomor form": {SubmissionFormNumber: &number},
		"tanggal":    {SignDatesEnabled: &on},
	} {
		if _, err := s.Apply(ctx, admin, in); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("admin mengubah %s: %v, ingin ErrForbidden", name, err)
		}
	}
	for name, in := range map[string]Update{
		"mode migrasi":       {MigrationMode: &on},
		"aturan penyetujuan": {FormApprovalMode: &anyMode},
		"campuran":           {MigrationMode: &on, QREnabled: &on},
	} {
		if _, err := s.Apply(ctx, engineer, in); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("PPAP Engineer mengubah %s: %v, ingin ErrForbidden", name, err)
		}
	}
	if _, err := s.Apply(ctx, admin, Update{MigrationMode: &on, QREnabled: &on}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("admin mengubah campuran: %v, ingin ErrForbidden", err)
	}

	got, err := s.Apply(ctx, engineer, Update{QREnabled: &on, SubmissionFormNumber: &number})
	if err != nil || !got.QREnabled || got.SubmissionFormNumber != number {
		t.Fatalf("PPAP Engineer: %+v, %v", got, err)
	}
	got, err = s.Apply(ctx, admin, Update{MigrationMode: &on})
	if err != nil || !got.MigrationMode || !got.QREnabled {
		t.Fatalf("admin: %+v, %v", got, err)
	}
}

// TestFormApprovalMode: aturan persetujuan form set dan tabel AP (D-139)
// diatur admin; bawaannya semua penyetuju harus setuju.
func TestFormApprovalMode(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	worker := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)

	got, err := s.Get(ctx, worker)
	if err != nil || got.FormApprovalMode != ApprovalAll {
		t.Fatalf("isi awal: %+v, %v; ingin all", got, err)
	}
	anyMode, bad := ApprovalAny, "most"
	if _, err := s.Apply(ctx, worker, Update{FormApprovalMode: &anyMode}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja: %v, ingin ErrForbidden", err)
	}
	var invalid *problem.ValidationError
	if _, err := s.Apply(ctx, admin, Update{FormApprovalMode: &bad}); !errors.As(err, &invalid) || invalid.Problems[0].Field != "form_approval_mode" {
		t.Errorf("aturan tidak dikenal: %v", err)
	}
	got, err = s.Apply(ctx, admin, Update{FormApprovalMode: &anyMode})
	if err != nil || got.FormApprovalMode != ApprovalAny || got.SignDatesEnabled {
		t.Fatalf("Apply: %+v, %v", got, err)
	}
	rows, err := db.New(pool).ListAuditChain(ctx)
	if err != nil || len(rows) != 1 || rows[0].Details != `{"form_approval_mode":"all → any"}` {
		t.Errorf("audit log: %+v, %v", rows, err)
	}
}
