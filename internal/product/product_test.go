package product

import (
	"errors"
	"os"
	"slices"
	"testing"

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

// fixture: database baru, akun admin, dan dua customer (AB dan CD) di GR1.
type fixture struct {
	s      *Service
	pool   *pgxpool.Pool
	master access.Actor
	ab, cd customer.Customer
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.NewPool(t)
	checker := access.NewChecker(pool)
	// Master data diurus PPAP Engineer (E-11, D-209).
	master := newActor(t, pool, "ppap.master", access.AccountWork, access.RolePPAPEngineer)
	customers := customer.NewService(pool, checker)
	group, err := customers.CreateProjectGroup(ctx, master, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	ab, err := customers.Create(ctx, master, customer.NewCustomer{Code: "AB", Name: "Customer A", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	cd, err := customers.Create(ctx, master, customer.NewCustomer{Code: "CD", Name: "Customer C", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{s: NewService(pool, checker), pool: pool, master: master, ab: ab, cd: cd}
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

// lastAudit mengambil entri audit terakhir.
func lastAudit(t *testing.T, pool *pgxpool.Pool) db.AuditLog {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil || len(rows) == 0 {
		t.Fatalf("audit log: %d entri, %v", len(rows), err)
	}
	return rows[len(rows)-1]
}

// TestModelToCustomer: kriteria selesai M1.4. Model terhubung ke customer,
// paling banyak ke satu group model milik customer yang sama (D-42), dan part
// number terhubung ke model.
func TestModelToCustomer(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	group, err := f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: " Board Family "})
	if err != nil {
		t.Fatalf("CreateModelGroup: %v", err)
	}
	if group.Name != "Board Family" || group.CustomerID != f.ab.ID {
		t.Errorf("group model: %+v", group)
	}
	inGroup, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100", ModelGroupID: group.ID})
	if err != nil {
		t.Fatalf("CreateModel dengan group: %v", err)
	}
	alone, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X200"})
	if err != nil {
		t.Fatalf("CreateModel tanpa group: %v", err)
	}
	if inGroup.ModelGroupID != group.ID || inGroup.ModelGroupName != "Board Family" || alone.ModelGroupID != "" {
		t.Errorf("model: %+v dan %+v", inGroup, alone)
	}
	part, err := f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: inGroup.ID, Number: " PN-0001 ", Name: "Main Board"})
	if err != nil {
		t.Fatalf("CreatePartNumber: %v", err)
	}
	if part.Number != "PN-0001" || part.CustomerID != f.ab.ID || part.ModelName != "X100" {
		t.Errorf("part number: %+v", part)
	}

	models, err := f.s.ListModels(ctx, f.master, f.ab.ID)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range models {
		names = append(names, m.Name+"/"+m.ModelGroupName)
	}
	if !slices.Equal(names, []string{"X100/Board Family", "X200/"}) {
		t.Errorf("ListModels AB: %v", names)
	}
	// Customer lain tidak melihat model AB.
	if other, err := f.s.ListModels(ctx, f.master, f.cd.ID); err != nil || len(other) != 0 {
		t.Errorf("ListModels CD: %+v, %v", other, err)
	}
	parts, err := f.s.ListPartNumbers(ctx, f.master, f.ab.ID)
	if err != nil || len(parts) != 1 || parts[0].ID != part.ID {
		t.Errorf("ListPartNumbers AB: %+v, %v", parts, err)
	}
	groups, err := f.s.ListModelGroups(ctx, f.master, f.ab.ID)
	if err != nil || len(groups) != 1 || groups[0].ID != group.ID {
		t.Errorf("ListModelGroups AB: %+v, %v", groups, err)
	}
	if _, err := f.s.ListModels(ctx, f.master, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("customer yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
}

// TestModelGroupBelongsToCustomer: group model hanya milik satu customer, dan
// model hanya boleh masuk group milik customernya sendiri (D-42).
func TestModelGroupBelongsToCustomer(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	groupAB, err := f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"})
	if err != nil {
		t.Fatal(err)
	}
	// Nama group yang sama boleh dipakai customer lain.
	groupCD, err := f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.cd.ID, Name: "Family"})
	if err != nil {
		t.Fatalf("nama group yang sama di customer lain: %v", err)
	}

	_, err = f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100", ModelGroupID: groupCD.ID})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"model_group_id:not_allowed"}) {
		t.Errorf("group milik customer lain: %v", got)
	}
	model, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.UpdateModel(ctx, f.master, model.ID, ModelUpdate{Name: "X100", ModelGroupID: groupCD.ID})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"model_group_id:not_allowed"}) {
		t.Errorf("pindah ke group milik customer lain: %v", got)
	}

	// Masuk group, lalu keluar lagi. Satu model paling banyak satu group.
	moved, err := f.s.UpdateModel(ctx, f.master, model.ID, ModelUpdate{Name: "X100A", ModelGroupID: groupAB.ID})
	if err != nil || moved.ModelGroupID != groupAB.ID || moved.Name != "X100A" {
		t.Fatalf("UpdateModel: %+v, %v", moved, err)
	}
	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionModelUpdated || r.Details != `{"model_group":"- → Family","name":"X100 → X100A"}` {
		t.Errorf("entri audit: %s %s", r.Action, r.Details)
	}
	out, err := f.s.UpdateModel(ctx, f.master, model.ID, ModelUpdate{Name: "X100A"})
	if err != nil || out.ModelGroupID != "" {
		t.Errorf("keluar dari group: %+v, %v", out, err)
	}
}

// TestNamesUnique: nama group dan model unik dalam satu customer, dan nomor
// part unik dalam satu customer (ASUMSI E-30).
func TestNamesUnique(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	if _, err := f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"}); err != nil {
		t.Fatal(err)
	}
	m1, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X200"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: m1.ID, Number: "PN-1"}); err != nil {
		t.Fatal(err)
	}
	mCD, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.cd.ID, Name: "X100"})
	if err != nil {
		t.Fatalf("nama model yang sama di customer lain: %v", err)
	}
	if _, err := f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: mCD.ID, Number: "PN-1"}); err != nil {
		t.Fatalf("nomor part yang sama di customer lain: %v", err)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"group ganda", second(f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"})), []string{"name:taken"}},
		{"model ganda", second(f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100"})), []string{"name:taken"}},
		{"model diganti ke nama yang dipakai", second(f.s.UpdateModel(ctx, f.master, m2.ID, ModelUpdate{Name: "X100"})), []string{"name:taken"}},
		{"nomor part ganda di model lain", second(f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: m2.ID, Number: "PN-1"})), []string{"number:taken"}},
		{"isian kosong", second(f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID})), []string{"name:required"}},
		{"nomor part kosong", second(f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: m2.ID})), []string{"number:required"}},
		{"customer tidak ada", second(f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: "bukan-uuid", Name: "X"})), []string{"customer_id:invalid_format"}},
		{"model tidak ada", second(f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: "bukan-uuid", Number: "X"})), []string{"model_id:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
}

// second mengambil error dari hasil dua nilai, supaya tabel test tetap ringkas.
func second[T any](_ T, err error) error { return err }

// TestInactiveCustomer: customer yang dinonaktifkan tidak mendapat group,
// model, atau part number baru (E-09).
func TestInactiveCustomer(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	model, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100"})
	if err != nil {
		t.Fatal(err)
	}
	customers := customer.NewService(f.pool, access.NewChecker(f.pool))
	_, err = customers.Update(ctx, f.master, f.ab.ID, customer.CustomerUpdate{Name: f.ab.Name, ProjectGroupID: f.ab.ProjectGroupID, Active: new(false)})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"group", second(f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"})), []string{"customer_id:not_allowed"}},
		{"model", second(f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X200"})), []string{"customer_id:not_allowed"}},
		{"part number", second(f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: model.ID, Number: "PN-1"})), []string{"model_id:not_allowed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
}

// TestPermissions: hanya akun admin yang mengubah master (A-2, E-10); semua
// akun bisa melihat daftar (E-24, D-103).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleTemplateMaintainer)
	model, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100"})
	if err != nil {
		t.Fatal(err)
	}

	for name, err := range map[string]error{
		"CreateModelGroup": second(f.s.CreateModelGroup(ctx, author, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"})),
		"CreateModel":      second(f.s.CreateModel(ctx, author, NewModel{CustomerID: f.ab.ID, Name: "X200"})),
		"UpdateModel":      second(f.s.UpdateModel(ctx, author, model.ID, ModelUpdate{Name: "X300"})),
		"CreatePartNumber": second(f.s.CreatePartNumber(ctx, author, NewPartNumber{ModelID: model.ID, Number: "PN-1"})),
	} {
		if !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s oleh akun kerja: %v, ingin ErrForbidden", name, err)
		}
	}
	for name, err := range map[string]error{
		"ListModelGroups": second(f.s.ListModelGroups(ctx, author, f.ab.ID)),
		"ListModels":      second(f.s.ListModels(ctx, author, f.ab.ID)),
		"ListPartNumbers": second(f.s.ListPartNumbers(ctx, author, f.ab.ID)),
	} {
		if err != nil {
			t.Errorf("%s oleh akun kerja: %v", name, err)
		}
	}
}

// TestAudit: setiap master baru dan setiap perubahan tercatat di audit log.
func TestAudit(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	group, err := f.s.CreateModelGroup(ctx, f.master, NewModelGroup{CustomerID: f.ab.ID, Name: "Family"})
	if err != nil {
		t.Fatal(err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionModelGroupCreated || r.Details != `{"customer":"AB","name":"Family"}` {
		t.Errorf("group dibuat: %s %s", r.Action, r.Details)
	}
	if _, err := f.s.RenameModelGroup(ctx, f.master, group.ID, "Family B"); err != nil {
		t.Fatal(err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionModelGroupUpdated || r.Details != `{"name":"Family → Family B"}` {
		t.Errorf("group diganti nama: %s %s", r.Action, r.Details)
	}
	model, err := f.s.CreateModel(ctx, f.master, NewModel{CustomerID: f.ab.ID, Name: "X100", ModelGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionModelCreated || r.Details != `{"customer":"AB","model_group":"Family B","name":"X100"}` {
		t.Errorf("model dibuat: %s %s", r.Action, r.Details)
	}
	part, err := f.s.CreatePartNumber(ctx, f.master, NewPartNumber{ModelID: model.ID, Number: "PN-1"})
	if err != nil {
		t.Fatal(err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionPartNumberCreated || r.Details != `{"model":"X100","number":"PN-1"}` {
		t.Errorf("part number dibuat: %s %s", r.Action, r.Details)
	}
	updated, err := f.s.UpdatePartNumber(ctx, f.master, part.ID, PartNumberUpdate{Number: "PN-1A", Name: "Main Board"})
	if err != nil || updated.Number != "PN-1A" || updated.Name != "Main Board" {
		t.Fatalf("UpdatePartNumber: %+v, %v", updated, err)
	}
	if r := lastAudit(t, f.pool); r.Action != audit.ActionPartNumberUpdated || r.Details != `{"name":"- → Main Board","number":"PN-1 → PN-1A"}` {
		t.Errorf("part number diubah: %s %s", r.Action, r.Details)
	}
}
