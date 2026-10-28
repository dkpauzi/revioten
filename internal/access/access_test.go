package access

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// TestMain menyalakan PostgreSQL untuk test di paket ini (lihat dbtest).
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// allPermissions adalah katalog hak U3, urut seperti di matriks.
var allPermissions = []Permission{
	ViewReleased, ViewRegister, ExportReleased, ViewDrafts,
	RegisterNumbers, EditDrafts, ProposeChanges, WorkRevisions, PrintForSignature,
	ManageTemplates, RecordRelease, ManageNumbers, ControlledCopies, MigrateDocuments,
	VerifyChanges, CheckPFMEACP,
	ManageUsers, ManageMasterData, ManageSettings, ViewAuditLog,
}

// u3Matrix adalah matriks U3-peta-pengguna bagian 3, ditulis ulang di test
// supaya isi awal di migrasi 00003 bisa dibandingkan dengannya.
var u3Matrix = map[string][]Permission{
	RoleAuthor:             {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, RegisterNumbers, EditDrafts, ProposeChanges, WorkRevisions, PrintForSignature},
	RoleTemplateMaintainer: {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, ManageTemplates},
	RolePFMEACPChecker:     {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, CheckPFMEACP},
	RoleVerifier:           {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, VerifyChanges},
	RoleDocumentControl:    {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, RecordRelease, ManageNumbers, ControlledCopies},
	RoleMigration:          {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, MigrateDocuments},
	RoleViewer:             {ViewReleased, ViewRegister, ExportReleased},
	RoleAdmin:              {ViewReleased, ViewRegister, ExportReleased, ManageUsers, ManageMasterData, ManageSettings, ViewAuditLog},
}

// newAccount membuat orang dan akun langsung lewat query, lalu memberi peran.
func newAccount(t *testing.T, pool *pgxpool.Pool, login, kind string, grants ...RoleGrant) Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
		PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "tidak-dipakai",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range grants {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: g.Role, Scope: g.Scope}); err != nil {
			t.Fatal(err)
		}
	}
	return Actor{Type: ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login, PersonID: person.ID, PersonName: person.Name}
}

// TestRoleMatrix membuktikan setiap peran hanya punya hak sesuai matriks U3,
// ditambah hak dasar Viewer yang dimiliki semua akun.
func TestRoleMatrix(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)

	for role, want := range u3Matrix {
		t.Run(role, func(t *testing.T) {
			kind := AccountWork
			var grants []RoleGrant
			switch role {
			case RoleAdmin:
				kind = AccountAdmin
				grants = []RoleGrant{{Role: RoleAdmin, Scope: ScopeAll}}
			case RoleViewer:
				// Viewer tidak disimpan: akun tanpa peran sudah punya hak dasar.
			default:
				grants = []RoleGrant{{Role: role, Scope: ScopeAll}}
			}
			actor := newAccount(t, pool, "uji."+role, kind, grants...)

			for _, perm := range allPermissions {
				err := c.Authorize(t.Context(), actor, perm, Resource{})
				allowed := slices.Contains(want, perm)
				switch {
				case allowed && err != nil:
					t.Errorf("%s ditolak, padahal ada di matriks: %v", perm, err)
				case !allowed && !errors.Is(err, ErrForbidden):
					t.Errorf("%s: %v, ingin ErrForbidden", perm, err)
				}
			}
		})
	}
}

// TestAuthorizeCombinedRoles: hak akhir adalah gabungan semua peran (D-58).
func TestAuthorizeCombinedRoles(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	actor := newAccount(t, pool, "uji.rangkap", AccountWork,
		RoleGrant{Role: RoleAuthor, Scope: ScopeAll}, RoleGrant{Role: RoleVerifier, Scope: ScopeAll})

	for _, perm := range []Permission{EditDrafts, VerifyChanges} {
		if err := c.Authorize(t.Context(), actor, perm, Resource{}); err != nil {
			t.Errorf("%s: %v", perm, err)
		}
	}
	if err := c.Authorize(t.Context(), actor, RecordRelease, Resource{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("E-7: %v, ingin ErrForbidden", err)
	}
}

// TestAuthorizeScope: peran bercakupan Project Group sendiri hanya berlaku
// untuk customer di group itu (D-71). Master Project Group dan pemegang
// jabatannya baru ada di M1.4 dan M1.8, jadi untuk sekarang customer mana pun
// ditolak (ASUMSI E-04).
func TestAuthorizeScope(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	ctx := t.Context()
	grouped := newAccount(t, pool, "uji.pl", AccountWork, RoleGrant{Role: RoleAuthor, Scope: ScopeOwnProjectGroup})
	everyone := newAccount(t, pool, "uji.ppap", AccountWork, RoleGrant{Role: RoleAuthor, Scope: ScopeAll})
	customer := Resource{CustomerID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"}

	if err := c.Authorize(ctx, grouped, EditDrafts, Resource{}); err != nil {
		t.Errorf("tanpa customer: %v", err)
	}
	if err := c.Authorize(ctx, grouped, EditDrafts, customer); !errors.Is(err, ErrForbidden) {
		t.Errorf("customer di luar group: %v, ingin ErrForbidden", err)
	}
	if err := c.Authorize(ctx, everyone, EditDrafts, customer); err != nil {
		t.Errorf("cakupan semua customer: %v", err)
	}
	// Hak dasar Viewer berlaku untuk semua customer.
	if err := c.Authorize(ctx, grouped, ViewReleased, customer); err != nil {
		t.Errorf("L-1 untuk customer: %v", err)
	}
}

func TestAuthorizeOwnAccount(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	ctx := t.Context()
	actor := newAccount(t, pool, "uji.sendiri", AccountWork)
	other := newAccount(t, pool, "uji.lain", AccountWork)

	if err := c.Authorize(ctx, actor, ManageOwnAccount, Resource{AccountID: actor.AccountID}); err != nil {
		t.Errorf("akun sendiri: %v", err)
	}
	if err := c.Authorize(ctx, actor, ManageOwnAccount, Resource{AccountID: other.AccountID}); !errors.Is(err, ErrForbidden) {
		t.Errorf("akun orang lain: %v, ingin ErrForbidden", err)
	}
	// Tanpa aktor (misalnya kode yang lupa mengisi actor) selalu ditolak.
	if err := c.Authorize(ctx, Actor{}, ViewReleased, Resource{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("tanpa aktor: %v, ingin ErrForbidden", err)
	}
}

func TestPermissions(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	actor := newAccount(t, pool, "uji.dk", AccountWork, RoleGrant{Role: RoleDocumentControl, Scope: ScopeAll})

	got, err := c.Permissions(t.Context(), actor)
	if err != nil {
		t.Fatalf("Permissions: %v", err)
	}
	want := u3Matrix[RoleDocumentControl]
	if !slices.Equal(got, want) {
		t.Errorf("Permissions %v, ingin %v", got, want)
	}
}

func TestListRoles(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	admin := newAccount(t, pool, "uji.admin", AccountAdmin, RoleGrant{Role: RoleAdmin, Scope: ScopeAll})

	roles, err := c.ListRoles(t.Context(), admin)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != len(u3Matrix) {
		t.Fatalf("%d peran, ingin %d", len(roles), len(u3Matrix))
	}
	for _, r := range roles {
		if !slices.Equal(r.Permissions, u3Matrix[r.ID]) {
			t.Errorf("peran %s berisi %v, ingin %v", r.ID, r.Permissions, u3Matrix[r.ID])
		}
	}

	worker := newAccount(t, pool, "uji.kerja", AccountWork, RoleGrant{Role: RoleAuthor, Scope: ScopeAll})
	if _, err := c.ListRoles(t.Context(), worker); !errors.Is(err, ErrForbidden) {
		t.Errorf("ListRoles oleh akun kerja: %v, ingin ErrForbidden", err)
	}
}

func TestSetRoles(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	ctx := t.Context()
	admin := newAccount(t, pool, "uji.admin", AccountAdmin, RoleGrant{Role: RoleAdmin, Scope: ScopeAll})
	worker := newAccount(t, pool, "uji.kerja", AccountWork)

	grants := []RoleGrant{{Role: RoleAuthor, Scope: ScopeOwnProjectGroup}, {Role: RoleVerifier, Scope: ScopeAll}}
	if err := c.SetRoles(ctx, admin, worker.AccountID, grants); err != nil {
		t.Fatalf("SetRoles: %v", err)
	}
	if err := c.Authorize(ctx, worker, VerifyChanges, Resource{CustomerID: "x"}); err != nil {
		t.Errorf("peran baru belum berlaku: %v", err)
	}
	// Peran diganti seluruhnya, bukan ditambah.
	if err := c.SetRoles(ctx, admin, worker.AccountID, nil); err != nil {
		t.Fatalf("SetRoles kosong: %v", err)
	}
	if err := c.Authorize(ctx, worker, VerifyChanges, Resource{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("peran lama masih berlaku: %v", err)
	}

	// Aturan yang tidak bisa dimatikan (R-SEC-11 butir 8, D-72, D-73).
	tests := []struct {
		name    string
		account string
		grants  []RoleGrant
	}{
		{"akun admin tidak bisa diubah perannya", admin.AccountID, []RoleGrant{{Role: RoleAuthor, Scope: ScopeAll}}},
		{"akun kerja tidak boleh memegang Admin", worker.AccountID, []RoleGrant{{Role: RoleAdmin, Scope: ScopeAll}}},
		{"Viewer tidak diberikan karena sudah dimiliki semua", worker.AccountID, []RoleGrant{{Role: RoleViewer, Scope: ScopeAll}}},
		{"peran yang tidak ada", worker.AccountID, []RoleGrant{{Role: "approver", Scope: ScopeAll}}},
		{"peran ganda", worker.AccountID, []RoleGrant{{Role: RoleAuthor, Scope: ScopeAll}, {Role: RoleAuthor, Scope: ScopeOwnProjectGroup}}},
		{"cakupan group hanya untuk Penyusun dan Verifikator", worker.AccountID, []RoleGrant{{Role: RoleDocumentControl, Scope: ScopeOwnProjectGroup}}},
		{"cakupan yang tidak dikenal", worker.AccountID, []RoleGrant{{Role: RoleAuthor, Scope: "gr1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.SetRoles(ctx, admin, tt.account, tt.grants)
			var invalid *problem.ValidationError
			if !errors.As(err, &invalid) {
				t.Errorf("ingin *problem.ValidationError, dapat %v", err)
			}
		})
	}

	if err := c.SetRoles(ctx, worker, worker.AccountID, grants); !errors.Is(err, ErrForbidden) {
		t.Errorf("SetRoles oleh akun kerja: %v, ingin ErrForbidden", err)
	}
	if err := c.SetRoles(ctx, admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", nil); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("akun yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
	// ID yang bukan UUID juga berarti akun tidak ada, bukan error database (500).
	if err := c.SetRoles(ctx, admin, "bukan-uuid", nil); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("ID bukan UUID: %v, ingin problem.ErrNotFound", err)
	}
}
