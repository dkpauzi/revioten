package customer

import (
	"errors"
	"os"
	"reflect"
	"slices"
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

// fixture adalah Service dengan database baru, satu akun admin, dan satu
// Project Group GR1 untuk customer yang dibuat test.
type fixture struct {
	s     *Service
	pool  *pgxpool.Pool
	admin access.Actor
	group ProjectGroup
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.NewPool(t)
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	group, err := s.CreateProjectGroup(t.Context(), admin, ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatalf("CreateProjectGroup: %v", err)
	}
	return fixture{s: s, pool: pool, admin: admin, group: group}
}

// problemCodes mengambil "field:kode" dari *problem.ValidationError atau *problem.ConflictError.
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

// lastAudit mengambil entri audit terakhir.
func lastAudit(t *testing.T, pool *pgxpool.Pool) db.AuditLog {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("audit log kosong")
	}
	return rows[len(rows)-1]
}

// TestPermissions: mengubah master butuh hak A-2 (hanya akun admin, E-10),
// sedangkan melihat daftar cukup hak dasar semua akun (E-24, D-103).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleDocumentControl)
	viewer := newActor(t, f.pool, "viewer.uji", access.AccountWork)
	in := NewCustomer{Code: "AB", Name: "Customer", ProjectGroupID: f.group.ID}

	for _, actor := range []access.Actor{author, viewer} {
		if _, err := f.s.Create(ctx, actor, in); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("Create oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if _, err := f.s.CreateProjectGroup(ctx, actor, ProjectGroupInput{Code: "GR9", Name: "X"}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("CreateProjectGroup oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if _, err := f.s.List(ctx, actor); err != nil {
			t.Errorf("List oleh %s: %v", actor.Login, err)
		}
		if _, err := f.s.ListProjectGroups(ctx, actor); err != nil {
			t.Errorf("ListProjectGroups oleh %s: %v", actor.Login, err)
		}
	}
	c, err := f.s.Create(ctx, f.admin, in)
	if err != nil {
		t.Fatalf("Create oleh admin: %v", err)
	}
	for _, actor := range []access.Actor{author, viewer} {
		if _, err := f.s.Update(ctx, actor, c.ID, CustomerUpdate{Name: "X", ProjectGroupID: f.group.ID, Active: new(true)}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("Update oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if _, err := f.s.ChangeCode(ctx, actor, c.ID, "CD"); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("ChangeCode oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
	}
}

func TestListEmpty(t *testing.T) {
	f := newFixture(t)

	list, err := f.s.List(t.Context(), f.admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Daftar kosong harus berupa slice kosong, bukan nil, supaya JSON-nya [].
	if list == nil || len(list) != 0 {
		t.Errorf("List = %#v, ingin slice kosong", list)
	}
}

func TestCreateAndList(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	before := time.Now()

	created, err := f.s.Create(ctx, f.admin, NewCustomer{Code: " bb ", Name: " Customer B ", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatalf("Create BB: %v", err)
	}
	if _, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AA", Name: "Customer A", ProjectGroupID: f.group.ID}); err != nil {
		t.Fatalf("Create AA: %v", err)
	}

	// Hasil Create sudah dirapikan dan diberi id serta waktu oleh database.
	if created.ID == "" {
		t.Error("ID kosong")
	}
	if created.Code != "BB" || created.Name != "Customer B" || !created.Active {
		t.Errorf("Create = %+v, ingin kode BB, nama Customer B, dan aktif", created)
	}
	if created.ProjectGroupID != f.group.ID || created.ProjectGroupCode != "GR1" {
		t.Errorf("Project Group = %q %q, ingin GR1", created.ProjectGroupID, created.ProjectGroupCode)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt dalam zona %v, ingin UTC", created.CreatedAt.Location())
	}
	if created.CreatedAt.Before(before.Add(-time.Minute)) || created.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("CreatedAt = %v, tidak dekat dengan waktu test", created.CreatedAt)
	}
	// Riwayat kode dimulai dengan kode pertama, yang sedang berlaku (E-11).
	if len(created.Codes) != 1 || created.Codes[0].Code != "BB" || created.Codes[0].ValidUntil != nil {
		t.Errorf("riwayat kode = %+v", created.Codes)
	}

	list, err := f.s.List(ctx, f.admin)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var codes []string
	for _, c := range list {
		codes = append(codes, c.Code)
	}
	if !slices.Equal(codes, []string{"AA", "BB"}) {
		t.Errorf("urutan kode = %v, ingin [AA BB]", codes)
	}
	if !reflect.DeepEqual(list[1], created) {
		t.Errorf("baris BB dari List = %+v, ingin sama dengan hasil Create %+v", list[1], created)
	}
}

func TestCreateDuplicateCode(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	if _, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer Pertama", ProjectGroupID: f.group.ID}); err != nil {
		t.Fatalf("Create pertama: %v", err)
	}

	// Kode yang sama setelah dirapikan (" ab " menjadi AB) harus ditolak.
	_, err := f.s.Create(ctx, f.admin, NewCustomer{Code: " ab ", Name: "Customer Kedua", ProjectGroupID: f.group.ID})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"code:taken"}) {
		t.Errorf("masalah = %v, ingin [code:taken]", got)
	}
}

// TestCreateNeedsProjectGroup: setiap customer dialokasikan ke satu Project Group (D-54).
func TestCreateNeedsProjectGroup(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer"})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"project_group_id:required"}) {
		t.Errorf("tanpa Project Group: %v", got)
	}
	_, err = f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer", ProjectGroupID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"project_group_id:invalid_format"}) {
		t.Errorf("Project Group yang tidak ada: %v", got)
	}
}

func TestCreateInvalidIsNotStored(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "ZZ", Name: "General", ProjectGroupID: f.group.ID})
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("ingin *problem.ValidationError, dapat %v", err)
	}

	list, err := f.s.List(ctx, f.admin)
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
	f := newFixture(t)
	ctx := audit.WithRequest(t.Context(), audit.Request{ID: "UJI-CUST", IP: "192.0.2.40"})

	c, err := f.s.Create(ctx, f.admin, NewCustomer{Code: " ab ", Name: "Example Customer", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer Ganda", ProjectGroupID: f.group.ID}); err == nil {
		t.Fatal("kode ganda diterima")
	}

	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionCustomerCreated || r.ActorType != "human" || *r.ActorAccountID != f.admin.AccountID ||
		*r.TargetType != audit.TargetCustomer || *r.TargetID != c.ID || *r.RequestID != "UJI-CUST" {
		t.Errorf("entri audit: %+v", r)
	}
	if r.Details != `{"code":"AB","name":"Example Customer","project_group":"GR1"}` {
		t.Errorf("rincian: %s", r.Details)
	}
}

// TestUpdate: nama, Project Group, dan status aktif bisa diubah. Customer tidak
// dihapus, hanya dinonaktifkan (E-09). Yang berubah tercatat di audit log.
func TestUpdate(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	gr2, err := f.s.CreateProjectGroup(ctx, f.admin, ProjectGroupInput{Code: "GR2", Name: "Project Group 2"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer Lama", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}

	// new(false) (Go 1.26+) membuat pointer ke nilai false. Active nil berarti
	// "tidak dikirim", dan statusnya tetap (E-34, D-111).
	updated, err := f.s.Update(ctx, f.admin, c.ID, CustomerUpdate{Name: " Customer Baru ", ProjectGroupID: gr2.ID, Active: new(false)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Customer Baru" || updated.ProjectGroupCode != "GR2" || updated.Active || updated.Code != "AB" {
		t.Errorf("hasil Update: %+v", updated)
	}
	r := lastAudit(t, f.pool)
	want := `{"active":"true → false","name":"Customer Lama → Customer Baru","project_group":"GR1 → GR2"}`
	if r.Action != audit.ActionCustomerUpdated || r.Details != want {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}

	// Tanpa Active, customer yang nonaktif tetap nonaktif: klien yang lupa
	// mengirim field ini tidak mengubah status tanpa sengaja (E-34, D-111).
	kept, err := f.s.Update(ctx, f.admin, c.ID, CustomerUpdate{Name: "Customer Baru Lagi", ProjectGroupID: gr2.ID})
	if err != nil {
		t.Fatalf("Update tanpa Active: %v", err)
	}
	if kept.Active {
		t.Errorf("Update tanpa Active mengaktifkan customer: %+v", kept)
	}
	if r := lastAudit(t, f.pool); r.Details != `{"name":"Customer Baru → Customer Baru Lagi"}` {
		t.Errorf("entri audit tanpa Active: %s", r.Details)
	}

	tests := []struct {
		name string
		in   CustomerUpdate
		want []string
	}{
		{"nama kosong dan tanpa Project Group", CustomerUpdate{Name: " "}, []string{"name:required", "project_group_id:required"}},
		{"Project Group tidak ada", CustomerUpdate{Name: "X", ProjectGroupID: "bukan-uuid"}, []string{"project_group_id:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.Update(ctx, f.admin, c.ID, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	if _, err := f.s.Update(ctx, f.admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", CustomerUpdate{Name: "X", ProjectGroupID: gr2.ID}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("customer yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
}

// TestChangeCode: kode customer bisa diganti, dan kode lama tetap tercatat
// dengan masa berlakunya (E-11). Satu kode hanya pernah dipakai satu customer
// (E-31, D-109), supaya nomor lama tidak pernah menunjuk dua customer.
func TestChangeCode(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	ab, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer A", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}
	ef, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "EF", Name: "Customer E", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}

	changed, err := f.s.ChangeCode(ctx, f.admin, ab.ID, " cd ")
	if err != nil {
		t.Fatalf("ChangeCode: %v", err)
	}
	if changed.Code != "CD" || len(changed.Codes) != 2 {
		t.Fatalf("hasil ChangeCode: %+v", changed)
	}
	old, current := changed.Codes[0], changed.Codes[1]
	if old.Code != "AB" || old.ValidUntil == nil || current.Code != "CD" || current.ValidUntil != nil ||
		!current.ValidFrom.Equal(*old.ValidUntil) {
		t.Errorf("riwayat kode: %+v", changed.Codes)
	}
	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionCustomerCodeChanged || r.Details != `{"after":"CD","before":"AB"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}

	tests := []struct {
		name string
		code string
		want []string
	}{
		{"kode yang sedang dipakai customer lain", "EF", []string{"code:taken"}},
		{"kode yang pernah dipakai customer lain", "AB", []string{"code:taken"}},
		{"kode yang sama dengan kode sekarang", "cd", []string{"code:not_allowed"}},
		{"bentuk salah", "C1", []string{"code:invalid_format"}},
		{"kode khusus", "ZZ", []string{"code:reserved"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.ChangeCode(ctx, f.admin, ab.ID, tt.code)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	// Customer EF tidak ikut berubah.
	list, err := f.s.List(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.ID == ef.ID && (c.Code != "EF" || len(c.Codes) != 1) {
			t.Errorf("customer EF berubah: %+v", c)
		}
	}
}

// TestProjectGroups: Project Group punya kode dan nama yang bisa diubah (D-54).
func TestProjectGroups(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	if f.group.Code != "GR1" || f.group.Name != "Project Group 1" {
		t.Errorf("Project Group pertama: %+v", f.group)
	}
	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionProjectGroupCreated || r.Details != `{"code":"GR1","name":"Project Group 1"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}

	tests := []struct {
		name string
		in   ProjectGroupInput
		want []string
	}{
		{"kode ganda (setelah dirapikan)", ProjectGroupInput{Code: " gr1 ", Name: "Lain"}, []string{"code:taken"}},
		{"kode dengan spasi", ProjectGroupInput{Code: "GR 2", Name: "Lain"}, []string{"code:invalid_format"}},
		{"kosong", ProjectGroupInput{}, []string{"code:required", "name:required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.CreateProjectGroup(ctx, f.admin, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	updated, err := f.s.UpdateProjectGroup(ctx, f.admin, f.group.ID, ProjectGroupInput{Code: "gr01", Name: "Project Group Satu"})
	if err != nil {
		t.Fatalf("UpdateProjectGroup: %v", err)
	}
	if updated.Code != "GR01" || updated.Name != "Project Group Satu" {
		t.Errorf("hasil UpdateProjectGroup: %+v", updated)
	}
	r = lastAudit(t, f.pool)
	if r.Action != audit.ActionProjectGroupUpdated || r.Details != `{"code":"GR1 → GR01","name":"Project Group 1 → Project Group Satu"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}
	if _, err := f.s.UpdateProjectGroup(ctx, f.admin, "bukan-uuid", ProjectGroupInput{Code: "GR5", Name: "X"}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("Project Group yang tidak ada: %v", err)
	}

	groups, err := f.s.ListProjectGroups(ctx, f.admin)
	if err != nil || len(groups) != 1 || groups[0].Code != "GR01" {
		t.Errorf("ListProjectGroups: %+v, %v", groups, err)
	}
}

// routeIDs mengambil ID rute approval isi awal menurut namanya (M1.8).
func routeIDs(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	rows, err := db.New(pool).ListApprovalRoutes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]string, len(rows))
	for _, r := range rows {
		ids[r.Name] = r.ID
	}
	return ids
}

// TestSetDefaultRoute: kriteria selesai M1.8. Admin mengatur rute default
// customer (D-47) lewat data, dan rute itu ikut menampilkan nama barunya bila
// rute diganti nama, karena customer menunjuk ID rute.
func TestSetDefaultRoute(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	routes := routeIDs(t, f.pool)
	ab, err := f.s.Create(ctx, f.admin, NewCustomer{Code: "AB", Name: "Customer A", ProjectGroupID: f.group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if ab.DefaultRouteID != "" {
		t.Errorf("customer baru sudah punya rute default: %+v (E-39, D-116: belum diatur)", ab)
	}

	set, err := f.s.SetDefaultRoute(ctx, f.admin, ab.ID, routes["Up to GM"])
	if err != nil || set.DefaultRouteID != routes["Up to GM"] || set.DefaultRouteName != "Up to GM" {
		t.Fatalf("SetDefaultRoute: %+v, %v", set, err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionCustomerRouteChanged || r.Details != `{"default_route":"- → Up to GM"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}
	if _, err := f.s.SetDefaultRoute(ctx, f.admin, ab.ID, routes["Up to QA QC Manager"]); err != nil {
		t.Fatal(err)
	}
	if r := lastAudit(t, f.pool); r.Details != `{"default_route":"Up to GM → Up to QA QC Manager"}` {
		t.Errorf("entri audit ganti rute: %s", r.Details)
	}

	// Rute diganti nama langsung lewat query: customer menunjuk ID, jadi ikut.
	if err := db.New(f.pool).RenameApprovalRoute(ctx, db.RenameApprovalRouteParams{ID: routes["Up to QA QC Manager"], Name: "Short"}); err != nil {
		t.Fatal(err)
	}
	list, err := f.s.List(ctx, f.admin)
	if err != nil || len(list) != 1 || list[0].DefaultRouteName != "Short" {
		t.Errorf("daftar customer setelah rute diganti nama: %+v, %v", list, err)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"rute kosong", second(f.s.SetDefaultRoute(ctx, f.admin, ab.ID, "")), []string{"route_id:required"}},
		{"rute tidak ada", second(f.s.SetDefaultRoute(ctx, f.admin, ab.ID, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489")), []string{"route_id:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	if _, err := f.s.SetDefaultRoute(ctx, f.admin, "bukan-uuid", routes["Up to GM"]); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("customer yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
	worker := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)
	if _, err := f.s.SetDefaultRoute(ctx, worker, ab.ID, routes["Up to GM"]); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja: %v, ingin ErrForbidden", err)
	}
}

// second mengambil error dari hasil dua nilai, supaya tabel test tetap ringkas.
func second[T any](_ T, err error) error { return err }
