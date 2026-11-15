package access

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestMain menyalakan PostgreSQL untuk test di paket ini (lihat dbtest).
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// allPermissions adalah katalog hak U3, urut seperti di matriks.
var allPermissions = []Permission{
	ViewReleased, ViewRegister, ExportReleased, ViewDrafts, ViewPerformance,
	RegisterNumbers, EditDrafts, ProposeChanges, WorkRevisions, PrintForSignature,
	ManageTemplates, RecordRelease, ManageNumbers, ControlledCopies, MigrateDocuments,
	VerifyChanges, CheckPFMEACP, ApproveForms,
	ManageUsers, ManageMasterData, ManageSettings, ViewAuditLog,
}

// u3Matrix adalah matriks U3-peta-pengguna bagian 3, ditulis ulang di test
// supaya isi awal di migrasi 00003 bisa dibandingkan dengannya.
var u3Matrix = map[string][]Permission{
	RoleAuthor:             {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, ViewPerformance, RegisterNumbers, EditDrafts, ProposeChanges, WorkRevisions, PrintForSignature},
	RoleTemplateMaintainer: {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, ManageTemplates},
	RolePFMEACPChecker:     {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, CheckPFMEACP},
	RoleVerifier:           {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, VerifyChanges},
	// Penyetuju form (D-139, migrasi 00019): menerbitkan form set dan tabel AP.
	// L-5 untuk Penyusun dan Penyetuju form: laporan kinerja (D-197, migrasi 00047).
	RoleFormApprover:    {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, ViewPerformance, ApproveForms},
	RoleDocumentControl: {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, RecordRelease, ManageNumbers, ControlledCopies},
	RoleMigration:       {ViewReleased, ViewRegister, ExportReleased, ViewDrafts, MigrateDocuments},
	RoleViewer:          {ViewReleased, ViewRegister, ExportReleased},
	RoleAdmin:           {ViewReleased, ViewRegister, ExportReleased, ManageUsers, ManageMasterData, ManageSettings, ViewAuditLog},
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
// untuk customer di group tempat orangnya menjadi Project Leader atau Project
// Engineer (D-71). Sumbernya master Project Group dan pemegang jabatan (E-05,
// M1.8), jadi hak ikut berubah begitu pemegang jabatan berubah.
func TestAuthorizeScope(t *testing.T) {
	pool := dbtest.NewPool(t)
	c := NewChecker(pool)
	ctx := t.Context()
	q := db.New(pool)
	grouped := newAccount(t, pool, "uji.pl", AccountWork, RoleGrant{Role: RoleAuthor, Scope: ScopeOwnProjectGroup})
	everyone := newAccount(t, pool, "uji.ppap", AccountWork, RoleGrant{Role: RoleAuthor, Scope: ScopeAll})

	// GR1 dengan customer AB, dan GR2 dengan customer CD.
	newCustomer := func(group, code string) Resource {
		t.Helper()
		g, err := q.CreateProjectGroup(ctx, db.CreateProjectGroupParams{Code: group, Name: "Group " + group})
		if err != nil {
			t.Fatal(err)
		}
		cust, err := q.CreateCustomer(ctx, db.CreateCustomerParams{Code: code, Name: "Customer " + code, ProjectGroupID: &g.ID})
		if err != nil {
			t.Fatal(err)
		}
		return Resource{CustomerID: cust.ID}
	}
	ab, cd := newCustomer("GR1", "AB"), newCustomer("GR2", "CD")
	abGroup, err := q.GetCustomer(ctx, ab.CustomerID)
	if err != nil {
		t.Fatal(err)
	}
	positions, err := q.ListPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	positionID := func(key string) string {
		for _, p := range positions {
			if p.Key == key {
				return p.ID
			}
		}
		t.Fatalf("jabatan %s tidak ada", key)
		return ""
	}
	hold := func(key string) string {
		t.Helper()
		id, err := q.AddPositionHolder(ctx, db.AddPositionHolderParams{
			PositionID: positionID(key), PerProjectGroup: true, PersonID: grouped.PersonID, ProjectGroupID: abGroup.ProjectGroupID,
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	check := func(what string, actor Actor, res Resource, allowed bool) {
		t.Helper()
		err := c.Authorize(ctx, actor, EditDrafts, res)
		if allowed && err != nil {
			t.Errorf("%s: %v, ingin boleh", what, err)
		}
		if !allowed && !errors.Is(err, ErrForbidden) {
			t.Errorf("%s: %v, ingin ErrForbidden", what, err)
		}
	}

	// Belum memegang jabatan: tidak ada customer yang termasuk group-nya.
	check("sebelum jadi Project Leader", grouped, ab, false)

	leader := hold("project_leader")
	check("customer di group sendiri", grouped, ab, true)
	check("customer di group lain", grouped, cd, false)
	// E-41 (D-118): hak bercakupan tidak berlaku untuk operasi tanpa customer.
	check("tanpa customer", grouped, Resource{}, false)
	check("cakupan semua customer", everyone, cd, true)
	check("cakupan semua customer, tanpa customer", everyone, Resource{}, true)
	// Hak dasar Viewer berlaku untuk semua customer.
	if err := c.Authorize(ctx, grouped, ViewReleased, cd); err != nil {
		t.Errorf("L-1 untuk customer lain: %v", err)
	}

	// Jabatan dilepas: hak untuk AB langsung hilang. Project Engineer juga berlaku.
	if err := q.DeletePositionHolder(ctx, leader); err != nil {
		t.Fatal(err)
	}
	check("setelah jabatan dilepas", grouped, ab, false)
	hold("project_engineer")
	check("sebagai Project Engineer", grouped, ab, true)
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
