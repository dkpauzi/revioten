package customer

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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

// newTestService membuat Service dengan database baru dan satu akun admin.
func newTestService(t *testing.T) (*Service, access.Actor) {
	t.Helper()
	pool := dbtest.NewPool(t)
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	return NewService(pool, access.NewChecker(pool)), admin
}

// TestPermissions: menambah customer butuh hak A-2 (hanya akun admin, E-10),
// sedangkan melihat daftar cukup hak dasar semua akun (ASUMSI E-24).
func TestPermissions(t *testing.T) {
	pool := dbtest.NewPool(t)
	s := NewService(pool, access.NewChecker(pool))
	ctx := t.Context()
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	author := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleDocumentControl)
	viewer := newActor(t, pool, "viewer.uji", access.AccountWork)

	for _, actor := range []access.Actor{author, viewer} {
		if _, err := s.Create(ctx, actor, NewCustomer{Code: "AB", Name: "Customer"}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("Create oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if _, err := s.List(ctx, actor); err != nil {
			t.Errorf("List oleh %s: %v", actor.Login, err)
		}
	}
	if _, err := s.Create(ctx, admin, NewCustomer{Code: "AB", Name: "Customer"}); err != nil {
		t.Errorf("Create oleh admin: %v", err)
	}
}

func TestListEmpty(t *testing.T) {
	s, admin := newTestService(t)

	list, err := s.List(t.Context(), admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Daftar kosong harus berupa slice kosong, bukan nil, supaya JSON-nya [].
	if list == nil || len(list) != 0 {
		t.Errorf("List = %#v, ingin slice kosong", list)
	}
}

func TestCreateAndList(t *testing.T) {
	s, admin := newTestService(t)
	ctx := t.Context()
	before := time.Now()

	created, err := s.Create(ctx, admin, NewCustomer{Code: " bb ", Name: " Customer B "})
	if err != nil {
		t.Fatalf("Create BB: %v", err)
	}
	if _, err := s.Create(ctx, admin, NewCustomer{Code: "AA", Name: "Customer A"}); err != nil {
		t.Fatalf("Create AA: %v", err)
	}

	// Hasil Create sudah dirapikan dan diberi id serta waktu oleh database.
	if created.ID == "" {
		t.Error("ID kosong")
	}
	if created.Code != "BB" || created.Name != "Customer B" {
		t.Errorf("Create = %+v, ingin kode BB dan nama Customer B", created)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt dalam zona %v, ingin UTC", created.CreatedAt.Location())
	}
	if created.CreatedAt.Before(before.Add(-time.Minute)) || created.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("CreatedAt = %v, tidak dekat dengan waktu test", created.CreatedAt)
	}

	list, err := s.List(ctx, admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var codes []string
	for _, c := range list {
		codes = append(codes, c.Code)
	}
	if len(codes) != 2 || codes[0] != "AA" || codes[1] != "BB" {
		t.Errorf("urutan kode = %v, ingin [AA BB]", codes)
	}
	if list[1] != created {
		t.Errorf("baris BB dari List = %+v, ingin sama dengan hasil Create %+v", list[1], created)
	}
}

func TestCreateDuplicateCode(t *testing.T) {
	s, admin := newTestService(t)
	ctx := t.Context()
	if _, err := s.Create(ctx, admin, NewCustomer{Code: "AB", Name: "Customer Pertama"}); err != nil {
		t.Fatalf("Create pertama: %v", err)
	}

	// Kode yang sama setelah dirapikan (" ab " menjadi AB) harus ditolak.
	_, err := s.Create(ctx, admin, NewCustomer{Code: " ab ", Name: "Customer Kedua"})

	var conflict *problem.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("ingin *problem.ConflictError, dapat %v", err)
	}
	if len(conflict.Problems) != 1 || conflict.Problems[0].Field != "code" || conflict.Problems[0].Code != problem.Taken {
		t.Errorf("masalah = %+v, ingin satu masalah code:taken", conflict.Problems)
	}
}

func TestCreateInvalidIsNotStored(t *testing.T) {
	s, admin := newTestService(t)
	ctx := t.Context()

	_, err := s.Create(ctx, admin, NewCustomer{Code: "ZZ", Name: "General"})
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("ingin *problem.ValidationError, dapat %v", err)
	}

	list, err := s.List(ctx, admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ada %d customer tersimpan, ingin 0", len(list))
	}
}

// TestCreateWritesAudit: customer baru tercatat di audit log bersama
// perubahannya (M1.3), dan percobaan yang ditolak tidak dicatat.
func TestCreateWritesAudit(t *testing.T) {
	pool := dbtest.NewPool(t)
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	ctx := audit.WithRequest(t.Context(), audit.Request{ID: "UJI-CUST", IP: "192.0.2.40"})

	c, err := s.Create(ctx, admin, NewCustomer{Code: " ab ", Name: "Example Customer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, admin, NewCustomer{Code: "AB", Name: "Customer Ganda"}); err == nil {
		t.Fatal("kode ganda diterima")
	}

	rows, err := db.New(pool).ListAuditChain(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("%d entri audit, ingin 1", len(rows))
	}
	r := rows[0]
	if r.Action != audit.ActionCustomerCreated || r.ActorType != "human" || *r.ActorAccountID != admin.AccountID ||
		*r.TargetType != audit.TargetCustomer || *r.TargetID != c.ID || *r.RequestID != "UJI-CUST" {
		t.Errorf("entri audit: %+v", r)
	}
	if r.Details != `{"code":"AB","name":"Example Customer"}` {
		t.Errorf("rincian: %s", r.Details)
	}
}
