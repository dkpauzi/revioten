package change

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	// Pendengar notifikasi email terpasang, supaya setiap peristiwa di test
	// ini juga menguji penyusunan emailnya (D-194).
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
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

// sourceNames menulis sumber perubahan sebagai "nama/mekanisme/aktif".
func sourceNames(t *testing.T, f fixture) []string {
	t.Helper()
	sources, err := f.s.ListChangeSources(t.Context(), f.admin)
	if err != nil {
		t.Fatal(err)
	}
	var list []string
	for _, s := range sources {
		state := "aktif"
		if !s.Active {
			state = "nonaktif"
		}
		list = append(list, s.Name+"/"+s.Mechanism+"/"+state)
	}
	return list
}

// TestDefaults: isi awal jenis perubahan (R-TPL-10), sumber perubahan
// (R-CHG-06, dengan mekanisme sistem R-CHG-01), dan kategori 4M.
func TestDefaults(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	types, err := f.s.ListChangeTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, ct := range types {
		got = append(got, ct.Key+"="+ct.Name)
	}
	if !slices.Equal(got, []string{"content=Content", "layout=Layout only", "administrative=Administrative"}) {
		t.Errorf("jenis perubahan: %v", got)
	}

	want := []string{
		"Audit finding//aktif", "Customer claim//aktif", "ECR//aktif", "PP (pre-production)//aktif",
		"SQAM/sqam/aktif", "Template propagation/template/aktif", "Yokoten/yokoten/aktif",
	}
	if got := sourceNames(t, f); !slices.Equal(got, want) {
		t.Errorf("sumber perubahan:\n%v\ningin:\n%v", got, want)
	}

	categories, err := f.s.ListFourMCategories(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, c := range categories {
		got = append(got, c.Key+"="+c.Name)
	}
	if !slices.Equal(got, []string{"man=Man", "machine=Machine", "material=Material", "method=Method"}) {
		t.Errorf("kategori 4M: %v", got)
	}
}

// TestEditChangeLists: kriteria selesai M1.6 untuk daftar perubahan. Admin
// mengganti nama jenis perubahan dan kategori 4M, menambah sumber perubahan,
// lalu menonaktifkannya. Sumber milik mekanisme sistem selalu aktif (E-36, D-113).
func TestEditChangeLists(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	types, err := f.s.ListChangeTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := f.s.RenameChangeType(ctx, f.admin, types[1].ID, " Layout ")
	if err != nil || layout.Name != "Layout" || layout.Key != KeyLayout {
		t.Errorf("RenameChangeType: %+v, %v", layout, err)
	}
	categories, err := f.s.ListFourMCategories(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	man, err := f.s.RenameFourMCategory(ctx, f.admin, categories[0].ID, "Manpower")
	if err != nil || man.Name != "Manpower" || man.Key != "man" {
		t.Errorf("RenameFourMCategory: %+v, %v", man, err)
	}

	source, err := f.s.CreateChangeSource(ctx, f.admin, "Internal improvement")
	if err != nil || !source.Active || source.Mechanism != "" {
		t.Fatalf("CreateChangeSource: %+v, %v", source, err)
	}
	inactive := false
	off, err := f.s.UpdateChangeSource(ctx, f.admin, source.ID, ChangeSourceUpdate{Name: "Kaizen", Active: &inactive})
	if err != nil || off.Name != "Kaizen" || off.Active {
		t.Errorf("UpdateChangeSource: %+v, %v", off, err)
	}
	// Active yang tidak diisi berarti status tetap.
	kept, err := f.s.UpdateChangeSource(ctx, f.admin, source.ID, ChangeSourceUpdate{Name: "Kaizen idea"})
	if err != nil || kept.Active {
		t.Errorf("ubah nama tanpa status: %+v, %v", kept, err)
	}

	sources, err := f.s.ListChangeSources(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	var yokoten ChangeSource
	for _, s := range sources {
		if s.Mechanism == MechanismYokoten {
			yokoten = s
		}
	}
	renamed, err := f.s.UpdateChangeSource(ctx, f.admin, yokoten.ID, ChangeSourceUpdate{Name: "Yokoten (horizontal deployment)"})
	if err != nil || renamed.Name != "Yokoten (horizontal deployment)" || renamed.Mechanism != MechanismYokoten {
		t.Errorf("ganti nama sumber sistem: %+v, %v", renamed, err)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"sumber sistem dinonaktifkan", second(f.s.UpdateChangeSource(ctx, f.admin, yokoten.ID, ChangeSourceUpdate{Name: "Yokoten", Active: &inactive})), []string{"active:not_allowed"}},
		{"nama sumber ganda", second(f.s.CreateChangeSource(ctx, f.admin, "customer CLAIM")), []string{"name:taken"}},
		{"nama jenis perubahan ganda", second(f.s.RenameChangeType(ctx, f.admin, types[0].ID, "layout")), []string{"name:taken"}},
		{"nama kategori kosong", second(f.s.RenameFourMCategory(ctx, f.admin, categories[1].ID, " ")), []string{"name:required"}},
		{"nama sumber terlalu panjang", second(f.s.CreateChangeSource(ctx, f.admin, strings.Repeat("a", 201))), []string{"name:too_long"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	for name, err := range map[string]error{
		"RenameChangeType":    second(f.s.RenameChangeType(ctx, f.admin, "bukan-uuid", "X")),
		"UpdateChangeSource":  second(f.s.UpdateChangeSource(ctx, f.admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", ChangeSourceUpdate{Name: "X"})),
		"RenameFourMCategory": second(f.s.RenameFourMCategory(ctx, f.admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", "X")),
	} {
		if !errors.Is(err, problem.ErrNotFound) {
			t.Errorf("%s untuk ID yang tidak ada: %v, ingin problem.ErrNotFound", name, err)
		}
	}
}

// TestPermissions: daftar perubahan hanya diubah Admin (A-2, UC-05). Semua
// akun bisa melihatnya (E-24, D-103).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleTemplateMaintainer)
	types, err := f.s.ListChangeTypes(ctx, author)
	if err != nil {
		t.Fatalf("ListChangeTypes oleh akun kerja: %v", err)
	}
	sources, err := f.s.ListChangeSources(ctx, author)
	if err != nil {
		t.Fatalf("ListChangeSources oleh akun kerja: %v", err)
	}
	categories, err := f.s.ListFourMCategories(ctx, author)
	if err != nil {
		t.Fatalf("ListFourMCategories oleh akun kerja: %v", err)
	}

	for name, err := range map[string]error{
		"RenameChangeType":    second(f.s.RenameChangeType(ctx, author, types[0].ID, "X")),
		"CreateChangeSource":  second(f.s.CreateChangeSource(ctx, author, "X")),
		"UpdateChangeSource":  second(f.s.UpdateChangeSource(ctx, author, sources[0].ID, ChangeSourceUpdate{Name: "X"})),
		"RenameFourMCategory": second(f.s.RenameFourMCategory(ctx, author, categories[0].ID, "X")),
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

	types, err := f.s.ListChangeTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.RenameChangeType(ctx, f.admin, types[2].ID, "Administrative only"); err != nil {
		t.Fatal(err)
	}
	check("jenis perubahan", audit.ActionChangeTypeUpdated, `{"name":"Administrative → Administrative only"}`)
	source, err := f.s.CreateChangeSource(ctx, f.admin, "Kaizen")
	if err != nil {
		t.Fatal(err)
	}
	check("sumber ditambah", audit.ActionChangeSourceCreated, `{"name":"Kaizen"}`)
	inactive := false
	if _, err := f.s.UpdateChangeSource(ctx, f.admin, source.ID, ChangeSourceUpdate{Name: "Kaizen", Active: &inactive}); err != nil {
		t.Fatal(err)
	}
	check("sumber dinonaktifkan", audit.ActionChangeSourceUpdated, `{"active":"true → false"}`)
	categories, err := f.s.ListFourMCategories(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.RenameFourMCategory(ctx, f.admin, categories[3].ID, "Methods"); err != nil {
		t.Fatal(err)
	}
	check("kategori 4M", audit.ActionFourMCategoryUpdated, `{"name":"Method → Methods"}`)

	before := last().Seq
	if _, err := f.s.RenameFourMCategory(ctx, f.admin, categories[3].ID, "Methods"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.UpdateChangeSource(ctx, f.admin, source.ID, ChangeSourceUpdate{Name: "Kaizen"}); err != nil {
		t.Fatal(err)
	}
	if after := last().Seq; after != before {
		t.Errorf("simpan tanpa perubahan menambah entri: %d → %d", before, after)
	}
}

// TestSetApprovalRoute: kebijakan approval per jenis perubahan adalah rute
// yang dipakai (R-TPL-10, E-35, D-112). Kosong berarti rute default customer,
// sehingga semua jenis memakai alur yang sama sampai diatur lain (Q-15).
func TestSetApprovalRoute(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	routes, err := db.New(f.pool).ListApprovalRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	short := ""
	for _, r := range routes {
		if r.Name == "Up to QA QC Manager" {
			short = r.ID
		}
	}
	types, err := f.s.ListChangeTypes(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, ct := range types {
		if ct.ApprovalRouteID != "" {
			t.Errorf("jenis %s sudah punya rute: %+v", ct.Key, ct)
		}
	}

	layout, err := f.s.SetApprovalRoute(ctx, f.admin, types[1].ID, short)
	if err != nil || layout.ApprovalRouteID != short || layout.ApprovalRouteName != "Up to QA QC Manager" || layout.Key != KeyLayout {
		t.Fatalf("SetApprovalRoute: %+v, %v", layout, err)
	}
	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionChangeTypeRouteChanged || r.Details != `{"approval_route":"- → Up to QA QC Manager"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}
	back, err := f.s.SetApprovalRoute(ctx, f.admin, types[1].ID, "")
	if err != nil || back.ApprovalRouteID != "" {
		t.Errorf("kembali ke rute default customer: %+v, %v", back, err)
	}
	if r := lastAudit(t, f.pool); r.Details != `{"approval_route":"Up to QA QC Manager → -"}` {
		t.Errorf("entri audit kembali ke default: %s", r.Details)
	}

	if got := problemCodes(t, second(f.s.SetApprovalRoute(ctx, f.admin, types[1].ID, "bukan-uuid"))); !slices.Equal(got, []string{"route_id:invalid_format"}) {
		t.Errorf("rute tidak ada: %v", got)
	}
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)
	if _, err := f.s.SetApprovalRoute(ctx, author, types[1].ID, short); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja: %v, ingin ErrForbidden", err)
	}
}

// lastAudit mengambil entri audit terakhir.
func lastAudit(t *testing.T, pool *pgxpool.Pool) db.AuditLog {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil || len(rows) == 0 {
		t.Fatalf("audit log: %d entri, %v", len(rows), err)
	}
	return rows[len(rows)-1]
}
