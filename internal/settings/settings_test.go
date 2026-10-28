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
// mati (D-52). Saklarnya satu pengaturan global yang diubah Admin lewat A-3
// (R-APR-14, UC-07); semua akun bisa membacanya.
func TestSignDates(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	worker := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)

	got, err := s.Get(ctx, worker)
	if err != nil || got.SignDatesEnabled {
		t.Fatalf("isi awal: %+v, %v; ingin mati", got, err)
	}
	on := true
	if _, err := s.Apply(ctx, worker, Update{SignDatesEnabled: &on}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja: %v, ingin ErrForbidden", err)
	}
	// Tanpa saklar pun hak tetap diperiksa.
	if _, err := s.Apply(ctx, worker, Update{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja tanpa saklar: %v, ingin ErrForbidden", err)
	}
	got, err = s.Apply(ctx, admin, Update{SignDatesEnabled: &on})
	if err != nil || !got.SignDatesEnabled {
		t.Fatalf("Apply: %+v, %v", got, err)
	}
	rows, err := db.New(pool).ListAuditChain(ctx)
	if err != nil || len(rows) != 1 || rows[0].Action != audit.ActionSystemSettingsUpdated ||
		rows[0].Details != `{"sign_dates_enabled":"false → true"}` {
		t.Errorf("audit log: %+v, %v", rows, err)
	}
	// Simpan tanpa perubahan, atau tanpa saklar, tidak dicatat.
	if _, err := s.Apply(ctx, admin, Update{SignDatesEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	if kept, err := s.Apply(ctx, admin, Update{}); err != nil || !kept.SignDatesEnabled {
		t.Errorf("tanpa saklar: %+v, %v; ingin tetap menyala", kept, err)
	}
	if rows, _ := db.New(pool).ListAuditChain(ctx); len(rows) != 1 {
		t.Errorf("simpan tanpa perubahan menambah entri: %d", len(rows))
	}
}
