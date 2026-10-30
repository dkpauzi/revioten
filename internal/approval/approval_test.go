package approval

import (
	"errors"
	"os"
	"slices"
	"strings"
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

// newPerson membuat satu orang tanpa akun, misalnya calon pemegang jabatan.
func newPerson(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	person, err := db.New(pool).CreatePerson(t.Context(), db.CreatePersonParams{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return person.ID
}

// fixture: database baru, akun admin, dan Project Group GR1 serta GR3.
type fixture struct {
	s        *Service
	pool     *pgxpool.Pool
	admin    access.Actor
	gr1, gr3 customer.ProjectGroup
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.NewPool(t)
	checker := access.NewChecker(pool)
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	customers := customer.NewService(pool, checker)
	gr1, err := customers.CreateProjectGroup(ctx, admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	gr3, err := customers.CreateProjectGroup(ctx, admin, customer.ProjectGroupInput{Code: "GR3", Name: "Project Group 3"})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{s: NewService(pool, checker), pool: pool, admin: admin, gr1: gr1, gr3: gr3}
}

// position mencari jabatan menurut key-nya.
func (f fixture) position(t *testing.T, key string) Position {
	t.Helper()
	list, err := f.s.ListPositions(t.Context(), f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range list {
		if p.Key == key {
			return p
		}
	}
	t.Fatalf("jabatan %s tidak ada", key)
	return Position{}
}

// route mencari rute menurut namanya.
func (f fixture) route(t *testing.T, name string) Route {
	t.Helper()
	list, err := f.s.ListRoutes(t.Context(), f.admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("rute %s tidak ada", name)
	return Route{}
}

// slotText menulis slot rute sebagai "tahap/judul cetak/jabatan,jabatan".
func slotText(r Route) []string {
	var list []string
	for _, s := range r.Slots {
		var names []string
		for _, p := range s.Positions {
			names = append(names, p.Name)
		}
		list = append(list, s.Stage+"/"+s.PrintTitle+"/"+strings.Join(names, ","))
	}
	return list
}

// inputOf mengubah rute menjadi isian, supaya test bisa mengubah sebagian saja.
func inputOf(r Route) RouteInput {
	in := RouteInput{Name: r.Name}
	for _, s := range r.Slots {
		slot := SlotInput{Stage: s.Stage, PrintTitle: s.PrintTitle}
		for _, p := range s.Positions {
			slot.PositionIDs = append(slot.PositionIDs, p.ID)
		}
		in.Slots = append(in.Slots, slot)
	}
	return in
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

// lastAudit mengambil entri audit terakhir.
func lastAudit(t *testing.T, pool *pgxpool.Pool) db.AuditLog {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil || len(rows) == 0 {
		t.Fatalf("audit log: %d entri, %v", len(rows), err)
	}
	return rows[len(rows)-1]
}

// TestDefaults: isi awal jabatan sesuai bagan organisasi NPI (D-54) dan dua
// rute approval (D-46): sampai QA QC Manager, dan sampai slot GM yang dipegang
// Advisor (D-56), dengan judul cetak dari cover TR-020 (D-76).
func TestDefaults(t *testing.T) {
	f := newFixture(t)
	positions, err := f.s.ListPositions(t.Context(), f.admin)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range positions {
		names = append(names, p.Name)
		perGroup := p.Key == KeyProjectLeader || p.Key == KeyProjectEngineer
		if p.PerProjectGroup != perGroup || p.Holders == nil {
			t.Errorf("jabatan %s: per Project Group %v, pemegang %#v", p.Key, p.PerProjectGroup, p.Holders)
		}
		// Satu pemegang untuk jabatan approver dan APQP/PPAP Controller (E-38, D-115).
		single := slices.Contains([]string{KeyAPQPController, KeyNPIManager, KeyQAQCManager, KeyAdvisor, KeyFactoryManager}, p.Key)
		if p.SingleHolder != single {
			t.Errorf("jabatan %s: satu pemegang %v, ingin %v", p.Key, p.SingleHolder, single)
		}
	}
	want := []string{
		"PPAP Engineer", "PPAP Staff", "Project Engineer", "Project Leader", "APQP/PPAP Controller",
		"NPI Manager", "QA QC Manager", "Advisor", "Factory Manager",
	}
	if !slices.Equal(names, want) {
		t.Errorf("jabatan:\n%v\ningin:\n%v", names, want)
	}

	toGM := []string{
		"prepared/Process. Eng/PPAP Engineer,PPAP Staff,Project Engineer,Project Leader",
		"checked/Engineer/Project Engineer,Project Leader",
		"approved/NPI Manager/NPI Manager",
		"approved/QA / QC Mgr./QA QC Manager",
		"approved/General Mgr./Advisor",
	}
	if got := slotText(f.route(t, "Up to GM")); !slices.Equal(got, toGM) {
		t.Errorf("rute sampai GM:\n%v\ningin:\n%v", got, toGM)
	}
	if got := slotText(f.route(t, "Up to QA QC Manager")); !slices.Equal(got, toGM[:4]) {
		t.Errorf("rute sampai QA QC Manager: %v", got)
	}
}

// TestHolders: Project Leader dan Project Engineer tepat satu per Project
// Group (D-54). Jabatan approver dan APQP/PPAP Controller dipegang satu orang
// (E-38, D-115); jabatan lain boleh punya beberapa pemegang. Satu orang boleh
// merangkap (D-58), misalnya Project Leader GR3 yang juga APQP/PPAP Controller.
func TestHolders(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	leader := f.position(t, KeyProjectLeader)
	controller := f.position(t, KeyAPQPController)
	npi := f.position(t, KeyNPIManager)
	personA, personB := newPerson(t, f.pool, "Pemegang Contoh A"), newPerson(t, f.pool, "Pemegang Contoh B")

	h, err := f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personA, ProjectGroupID: f.gr3.ID})
	if err != nil || h.PersonName != "Pemegang Contoh A" || h.ProjectGroupCode != "GR3" {
		t.Fatalf("AddHolder Project Leader GR3: %+v, %v", h, err)
	}
	if _, err := f.s.AddHolder(ctx, f.admin, controller.ID, NewHolder{PersonID: personA}); err != nil {
		t.Errorf("rangkap APQP/PPAP Controller: %v", err)
	}
	if _, err := f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personB, ProjectGroupID: f.gr1.ID}); err != nil {
		t.Errorf("Project Leader GR1: %v", err)
	}
	// PPAP Engineer boleh lebih dari satu orang.
	engineer := f.position(t, KeyPPAPEngineer)
	for _, person := range []string{personA, personB} {
		if _, err := f.s.AddHolder(ctx, f.admin, engineer.ID, NewHolder{PersonID: person}); err != nil {
			t.Errorf("PPAP Engineer: %v", err)
		}
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"group sudah punya Project Leader", second(f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personB, ProjectGroupID: f.gr3.ID})), []string{"project_group_id:taken"}},
		{"Project Leader tanpa group", second(f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personB})), []string{"project_group_id:required"}},
		{"NPI Manager dengan group", second(f.s.AddHolder(ctx, f.admin, npi.ID, NewHolder{PersonID: personB, ProjectGroupID: f.gr1.ID})), []string{"project_group_id:not_allowed"}},
		{"orang yang sama dua kali", second(f.s.AddHolder(ctx, f.admin, engineer.ID, NewHolder{PersonID: personA})), []string{"person_id:taken"}},
		{"APQP/PPAP Controller kedua", second(f.s.AddHolder(ctx, f.admin, controller.ID, NewHolder{PersonID: personB})), []string{"position_id:taken"}},
		{"orang tidak ada", second(f.s.AddHolder(ctx, f.admin, npi.ID, NewHolder{PersonID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"})), []string{"person_id:invalid_format"}},
		{"orang kosong", second(f.s.AddHolder(ctx, f.admin, npi.ID, NewHolder{})), []string{"person_id:required"}},
		{"group tidak ada", second(f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personB, ProjectGroupID: "bukan-uuid"})), []string{"project_group_id:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	// Pemegang dilepas, lalu group itu bisa mendapat Project Leader baru.
	if err := f.s.RemoveHolder(ctx, f.admin, leader.ID, h.ID); err != nil {
		t.Fatalf("RemoveHolder: %v", err)
	}
	if _, err := f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: personB, ProjectGroupID: f.gr3.ID}); err != nil {
		t.Errorf("Project Leader baru untuk GR3: %v", err)
	}
	var got []string
	for _, holder := range f.position(t, KeyProjectLeader).Holders {
		got = append(got, holder.ProjectGroupCode+"="+holder.PersonName)
	}
	if !slices.Equal(got, []string{"GR1=Pemegang Contoh B", "GR3=Pemegang Contoh B"}) {
		t.Errorf("pemegang Project Leader: %v", got)
	}
	if err := f.s.RemoveHolder(ctx, f.admin, npi.ID, h.ID); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("lepas pemegang yang sudah dilepas: %v, ingin problem.ErrNotFound", err)
	}
}

// TestRenamePositionKeepsRoutes: nama jabatan hanya tampilan (D-58). NPI
// Manager diganti PE Manager (D-55), dan slot rute ikut menampilkan nama baru.
func TestRenamePositionKeepsRoutes(t *testing.T) {
	f := newFixture(t)
	npi := f.position(t, KeyNPIManager)
	renamed, err := f.s.RenamePosition(t.Context(), f.admin, npi.ID, "PE Manager")
	if err != nil || renamed.Name != "PE Manager" || renamed.Key != KeyNPIManager {
		t.Fatalf("RenamePosition: %+v, %v", renamed, err)
	}
	slots := slotText(f.route(t, "Up to GM"))
	if slots[2] != "approved/NPI Manager/PE Manager" {
		t.Errorf("slot NPI: %s (judul cetak tetap, nama jabatan berubah)", slots[2])
	}
	if got := problemCodes(t, second(f.s.RenamePosition(t.Context(), f.admin, npi.ID, "advisor"))); !slices.Equal(got, []string{"name:taken"}) {
		t.Errorf("nama jabatan ganda: %v", got)
	}
}

// TestEditRoutes: rute adalah data (konsep M1.8). Slot GM dipindah dari
// Advisor ke Factory Manager cukup lewat master data (R-APR-15), dan rute baru
// bisa dibuat. Bentuk rute mengikuti D-45, D-48, dan D-49.
func TestEditRoutes(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	toGM := f.route(t, "Up to GM")
	factory := f.position(t, KeyFactoryManager)
	in := inputOf(toGM)
	in.Slots[4].PositionIDs = []string{factory.ID}
	updated, err := f.s.UpdateRoute(ctx, f.admin, toGM.ID, in)
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if updated.ID != toGM.ID || slotText(updated)[4] != "approved/General Mgr./Factory Manager" {
		t.Errorf("slot GM setelah dipindah: %v", slotText(updated))
	}

	// Rute baru tanpa slot GM, dengan judul cetak sendiri.
	short := inputOf(f.route(t, "Up to QA QC Manager"))
	short.Name = " Short route "
	short.Slots[1].PrintTitle = "Checker"
	created, err := f.s.CreateRoute(ctx, f.admin, short)
	if err != nil || created.Name != "Short route" || slotText(created)[1] != "checked/Checker/Project Engineer,Project Leader" {
		t.Fatalf("CreateRoute: %+v, %v", created, err)
	}

	base := inputOf(toGM)
	withSlots := func(change func(slots []SlotInput) []SlotInput) RouteInput {
		in := RouteInput{Name: "Route X"}
		for _, s := range base.Slots {
			in.Slots = append(in.Slots, SlotInput{Stage: s.Stage, PrintTitle: s.PrintTitle, PositionIDs: slices.Clone(s.PositionIDs)})
		}
		in.Slots = change(in.Slots)
		return in
	}
	tests := []struct {
		name string
		in   RouteInput
		want []string
	}{
		{"tanpa slot", RouteInput{Name: "Route X"}, []string{"slots:required"}},
		{"Checked sebelum Prepared", withSlots(func(s []SlotInput) []SlotInput { s[0], s[1] = s[1], s[0]; return s }), []string{"slots:invalid_format"}},
		{"dua Checked", withSlots(func(s []SlotInput) []SlotInput { s[2].Stage = StageChecked; return s }), []string{"slots:invalid_format"}},
		{"tanpa Approved", withSlots(func(s []SlotInput) []SlotInput { return s[:2] }), []string{"slots:invalid_format"}},
		{"tahap tidak dikenal", withSlots(func(s []SlotInput) []SlotInput { s[4].Stage = "signed"; return s }), []string{"slots:invalid_format"}},
		{"judul cetak kosong", withSlots(func(s []SlotInput) []SlotInput { s[3].PrintTitle = " "; return s }), []string{"print_title:required"}},
		{"slot tanpa jabatan", withSlots(func(s []SlotInput) []SlotInput { s[2].PositionIDs = nil; return s }), []string{"position_ids:required"}},
		{"jabatan tidak ada", withSlots(func(s []SlotInput) []SlotInput { s[2].PositionIDs = []string{"bukan-uuid"}; return s }), []string{"position_ids:invalid_format"}},
		{"nama kosong", RouteInput{Slots: base.Slots}, []string{"name:required"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, second(f.s.CreateRoute(ctx, f.admin, tt.in))); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
	dup := inputOf(toGM)
	dup.Name = "up to qa qc manager"
	if got := problemCodes(t, second(f.s.UpdateRoute(ctx, f.admin, toGM.ID, dup))); !slices.Equal(got, []string{"name:taken"}) {
		t.Errorf("nama rute ganda: %v", got)
	}
	if _, err := f.s.UpdateRoute(ctx, f.admin, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", base); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("rute yang tidak ada: %v, ingin problem.ErrNotFound", err)
	}
}

// TestPermissions: jabatan dan pemegangnya diubah lewat hak A-1 (UC-02), rute
// lewat A-2 (UC-06). Semua akun bisa melihat (E-24, D-103).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleDocumentControl)
	positions, err := f.s.ListPositions(ctx, author)
	if err != nil {
		t.Fatalf("ListPositions oleh akun kerja: %v", err)
	}
	routes, err := f.s.ListRoutes(ctx, author)
	if err != nil {
		t.Fatalf("ListRoutes oleh akun kerja: %v", err)
	}
	person := newPerson(t, f.pool, "Pemegang Contoh")

	for name, err := range map[string]error{
		"RenamePosition": second(f.s.RenamePosition(ctx, author, positions[0].ID, "X")),
		"AddHolder":      second(f.s.AddHolder(ctx, author, positions[0].ID, NewHolder{PersonID: person})),
		"RemoveHolder":   f.s.RemoveHolder(ctx, author, positions[0].ID, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"),
		"CreateRoute":    second(f.s.CreateRoute(ctx, author, inputOf(routes[0]))),
		"UpdateRoute":    second(f.s.UpdateRoute(ctx, author, routes[0].ID, inputOf(routes[0]))),
	} {
		if !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s oleh akun kerja: %v, ingin ErrForbidden", name, err)
		}
	}
}

// TestAudit: setiap perubahan jabatan, pemegang, dan rute tercatat.
func TestAudit(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	check := func(what, action, details string) {
		t.Helper()
		if r := lastAudit(t, f.pool); r.Action != action || r.Details != details {
			t.Errorf("%s: %s %s\ningin %s %s", what, r.Action, r.Details, action, details)
		}
	}

	advisor := f.position(t, KeyAdvisor)
	if _, err := f.s.RenamePosition(ctx, f.admin, advisor.ID, "Senior Advisor"); err != nil {
		t.Fatal(err)
	}
	check("ganti nama jabatan", audit.ActionPositionUpdated, `{"name":"Advisor → Senior Advisor"}`)
	leader := f.position(t, KeyProjectLeader)
	person := newPerson(t, f.pool, "Pemegang Contoh")
	h, err := f.s.AddHolder(ctx, f.admin, leader.ID, NewHolder{PersonID: person, ProjectGroupID: f.gr1.ID})
	if err != nil {
		t.Fatal(err)
	}
	check("pemegang ditambah", audit.ActionPositionHolderAdded, `{"person":"Pemegang Contoh","project_group":"GR1"}`)
	if err := f.s.RemoveHolder(ctx, f.admin, leader.ID, h.ID); err != nil {
		t.Fatal(err)
	}
	check("pemegang dilepas", audit.ActionPositionHolderRemoved, `{"person":"Pemegang Contoh","project_group":"GR1"}`)

	short := inputOf(f.route(t, "Up to QA QC Manager"))
	short.Name = "Short"
	created, err := f.s.CreateRoute(ctx, f.admin, short)
	if err != nil {
		t.Fatal(err)
	}
	r := lastAudit(t, f.pool)
	if r.Action != audit.ActionApprovalRouteCreated || !strings.Contains(r.Details, `"name":"Short"`) ||
		!strings.Contains(r.Details, `2 Checked: Engineer (Project Engineer, Project Leader)`) {
		t.Errorf("rute dibuat: %s %s", r.Action, r.Details)
	}
	renamed := inputOf(created)
	renamed.Name = "Short route"
	if _, err := f.s.UpdateRoute(ctx, f.admin, created.ID, renamed); err != nil {
		t.Fatal(err)
	}
	check("rute diganti nama", audit.ActionApprovalRouteUpdated, `{"name":"Short → Short route"}`)
}

// TestSingleHolderConcurrent: dua admin yang mencatat pemegang NPI Manager
// pada saat bersamaan tidak bisa sama-sama berhasil (E-38, D-115). Baris
// jabatan dikunci dulu, jadi pemeriksaan jumlah pemegang berjalan bergantian.
func TestSingleHolderConcurrent(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	npi := f.position(t, KeyNPIManager)
	people := []string{newPerson(t, f.pool, "Pemegang Contoh A"), newPerson(t, f.pool, "Pemegang Contoh B")}

	errs := make(chan error, len(people))
	for _, person := range people {
		go func() {
			_, err := f.s.AddHolder(ctx, f.admin, npi.ID, NewHolder{PersonID: person})
			errs <- err
		}()
	}
	var ok, taken int
	for range people {
		err := <-errs
		switch {
		case err == nil:
			ok++
		case slices.Equal(problemCodes(t, err), []string{"position_id:taken"}):
			taken++
		default:
			t.Errorf("error tak terduga: %v", err)
		}
	}
	if ok != 1 || taken != 1 {
		t.Errorf("berhasil %d, ditolak %d; ingin 1 dan 1", ok, taken)
	}
	if holders := f.position(t, KeyNPIManager).Holders; len(holders) != 1 {
		t.Errorf("pemegang NPI Manager: %d, ingin 1", len(holders))
	}
}
