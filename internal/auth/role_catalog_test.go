package auth

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

func fieldCode(err error) string {
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		return ""
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

// D-199 · Isi awal matriks mengikuti docs/rencana/matriks-peran-dan-hak.md
// bagian 4, dan peran lama nonaktif.
func TestRoleMatrixSeed(t *testing.T) {
	s, admin := newAdminActor(t)
	roles, err := access.LoadRoles(t.Context(), s.q)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]access.Role{}
	var active []string
	for _, r := range roles {
		byID[r.ID] = r
		if r.Active {
			active = append(active, r.ID)
		}
	}
	wantActive := []string{"ppap_engineer", "ppap_engineer_supervisor", "project_leader", "manager", "viewer", "admin"}
	if !slices.Equal(active, wantActive) {
		t.Errorf("peran aktif: %v", active)
	}
	codes := func(id string) string {
		var out string
		for _, p := range byID[id].Permissions {
			out += string(p) + " "
		}
		return out
	}
	want := map[string]string{
		"ppap_engineer":            "L-1 L-2 L-3 L-4 L-5 E-1 E-2 E-3 E-4 E-5 E-6 E-7 E-8 E-9 E-10 E-11 V-2 S-1 G-1 ",
		"ppap_engineer_supervisor": "L-1 L-2 L-3 L-4 L-5 E-1 E-2 E-3 E-4 E-5 E-6 E-7 E-8 E-9 E-10 E-11 V-1 V-2 V-3 S-1 G-1 ",
		"project_leader":           "L-1 L-2 L-3 L-4 E-1 E-2 E-3 E-4 E-5 V-1 S-1 G-1 ",
		"manager":                  "L-1 L-2 L-3 L-4 L-5 S-1 G-1 ",
		"viewer":                   "L-1 L-2 L-3 G-1 ",
		"admin":                    "L-1 L-2 L-3 A-1 A-3 A-4 ",
	}
	for id, w := range want {
		if got := codes(id); got != w {
			t.Errorf("%s: %q, ingin %q", id, got, w)
		}
	}
	if !byID["project_leader"].GroupScope || byID["ppap_engineer"].GroupScope {
		t.Errorf("cakupan Project Group: %+v", byID["project_leader"])
	}
	if byID["manager"].NameID != "Manager" || !byID["manager"].Builtin {
		t.Errorf("manager: %+v", byID["manager"])
	}

	// Akun admin tidak ikut mendapat hak Viewer di luar hak dasar (G-1).
	checker := access.NewChecker(s.pool)
	if err := checker.Authorize(t.Context(), admin, access.GenbaNotes, access.Resource{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("admin memegang G-1: %v", err)
	}
	// Peran lama tidak bisa diberikan lagi.
	_, _, err = s.CreateAccount(t.Context(), admin, NewAccount{
		PersonName: "Lama", Kind: KindWork, Login: "lama.uji",
		Roles: []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeAll}},
	})
	if fieldCode(err) != "roles:not_allowed" {
		t.Errorf("peran lama diberikan: %v", err)
	}
}

// D-199 · Admin menambah peran, mengganti namanya, dan menonaktifkannya.
func TestCreateRenameDeactivateRole(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()

	role, err := s.CreateRole(ctx, admin, RoleInput{NameEN: "QA Auditor", NameID: "Auditor QA", CopyFrom: access.RoleManager})
	if err != nil {
		t.Fatal(err)
	}
	if role.ID != "qa_auditor" || role.Builtin || !role.Active || role.Holders != 0 {
		t.Errorf("peran baru: %+v", role)
	}
	// Hak disalin dari Manager.
	if !slices.Equal(role.Permissions, []access.Permission{"L-1", "L-2", "L-3", "L-4", "L-5", "S-1", "G-1"}) {
		t.Errorf("hak peran baru: %v", role.Permissions)
	}
	// Tanpa salinan: hanya hak dasar. Nama yang sama di huruf lain diberi ID berangka.
	plain, err := s.CreateRole(ctx, admin, RoleInput{NameEN: "QA-auditor 2", NameID: "Auditor QA kedua"})
	if err != nil {
		t.Fatal(err)
	}
	if plain.ID != "qa_auditor_2" || len(plain.Permissions) != 3 {
		t.Errorf("peran kedua: %+v", plain)
	}
	if _, err := s.CreateRole(ctx, admin, RoleInput{NameEN: "qa auditor", NameID: "Lain"}); fieldCode(err) != "name_en:taken" {
		t.Errorf("nama ganda: %v", err)
	}
	if _, err := s.CreateRole(ctx, admin, RoleInput{NameEN: " ", NameID: "x"}); fieldCode(err) != "name_en:required" {
		t.Errorf("nama kosong: %v", err)
	}
	if _, err := s.CreateRole(ctx, admin, RoleInput{NameEN: "Copy admin", NameID: "Salin admin", CopyFrom: access.RoleAdmin}); fieldCode(err) != "copy_from:not_allowed" {
		t.Errorf("salin admin: %v", err)
	}

	// Peran baru bisa diberikan ke akun kerja, dan haknya berlaku.
	_, temporary, err := s.CreateAccount(ctx, admin, NewAccount{
		PersonName: "Auditor", Kind: KindWork, Login: "auditor.uji",
		Roles: []access.RoleGrant{{Role: role.ID, Scope: access.ScopeAll}},
	})
	if err != nil {
		t.Fatal(err)
	}
	auditor, _, err := s.Login(ctx, "auditor.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	if err := access.NewChecker(s.pool).Authorize(ctx, auditor, access.ViewPerformance, access.Resource{}); err != nil {
		t.Errorf("hak peran baru: %v", err)
	}
	// Cakupan Project Group hanya untuk peran yang mengizinkannya.
	if err := s.SetRoles(ctx, admin, auditor.AccountID, []access.RoleGrant{{Role: role.ID, Scope: access.ScopeOwnProjectGroup}}); fieldCode(err) != "roles:not_allowed" {
		t.Errorf("cakupan Project Group: %v", err)
	}

	// Ganti nama; ID tetap.
	renamed, err := s.UpdateRole(ctx, admin, role.ID, RoleInput{NameEN: "Internal auditor", NameID: "Auditor internal", GroupScope: true})
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != role.ID || renamed.NameEN != "Internal auditor" || !renamed.GroupScope || renamed.Holders != 1 {
		t.Errorf("ganti nama: %+v", renamed)
	}
	if err := s.SetRoles(ctx, admin, auditor.AccountID, []access.RoleGrant{{Role: role.ID, Scope: access.ScopeOwnProjectGroup}}); err != nil {
		t.Errorf("cakupan sesudah diizinkan: %v", err)
	}
	if _, err := s.UpdateRole(ctx, admin, role.ID, RoleInput{NameEN: "Internal auditor", NameID: "Auditor internal"}); fieldCode(err) != "group_scope:not_allowed" {
		t.Errorf("mematikan cakupan yang dipakai: %v", err)
	}
	for _, id := range []string{access.RoleAdmin, access.RoleViewer} {
		if _, err := s.UpdateRole(ctx, admin, id, RoleInput{NameEN: "X", NameID: "X"}); fieldCode(err) != "role:not_allowed" {
			t.Errorf("ubah %s: %v", id, err)
		}
		if _, err := s.SetRoleActive(ctx, admin, id, false); fieldCode(err) != "role:not_allowed" {
			t.Errorf("nonaktifkan %s: %v", id, err)
		}
	}

	// Nonaktif hanya bila tidak ada pemegang.
	if _, err := s.SetRoleActive(ctx, admin, role.ID, false); fieldCode(err) != "active:not_allowed" {
		t.Errorf("nonaktifkan peran yang dipegang: %v", err)
	}
	if err := s.SetRoles(ctx, admin, auditor.AccountID, nil); err != nil {
		t.Fatal(err)
	}
	off, err := s.SetRoleActive(ctx, admin, role.ID, false)
	if err != nil || off.Active {
		t.Fatalf("nonaktifkan: %+v, %v", off, err)
	}
	if err := s.SetRoles(ctx, admin, auditor.AccountID, []access.RoleGrant{{Role: role.ID, Scope: access.ScopeAll}}); fieldCode(err) != "roles:not_allowed" {
		t.Errorf("peran nonaktif diberikan: %v", err)
	}
	if on, err := s.SetRoleActive(ctx, admin, role.ID, true); err != nil || !on.Active {
		t.Errorf("aktifkan lagi: %+v, %v", on, err)
	}

	// Akun kerja tidak bisa mengelola peran.
	if _, err := s.CreateRole(ctx, auditor, RoleInput{NameEN: "A", NameID: "A"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja menambah peran: %v", err)
	}

	// Setiap perubahan tercatat di audit log.
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action IN ('role.created', 'role.updated', 'role.deactivated', 'role.activated')`).Scan(&n); err != nil || n != 5 {
		t.Errorf("audit: %d, %v", n, err)
	}
	// Database menolak penghapusan peran dan penonaktifan peran yang dipegang.
	if _, err := s.pool.Exec(ctx, `DELETE FROM roles WHERE id = 'qa_auditor_2'`); err == nil {
		t.Error("peran terhapus")
	}
}

func TestRoleSlug(t *testing.T) {
	for in, want := range map[string]string{
		"QA Auditor":      "qa_auditor",
		"  Leader / QC  ": "leader_qc",
		"Ünïcode only":    "n_code_only",
		"!!!":             "role",
	} {
		if got := roleSlug(in); got != want {
			t.Errorf("roleSlug(%q) = %q, ingin %q", in, got, want)
		}
	}
}
