package auth

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// M8.1 · Admin mengubah isi peran (D-66, R-SEC-10), dengan aturan yang tidak
// bisa dimatikan (R-SEC-11).
func TestSetRolePermissions(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	checker := access.NewChecker(s.pool)
	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{
		PersonName: "DCC Uji", Kind: KindWork, Login: "dcc.uji",
		Roles: []access.RoleGrant{{Role: access.RoleManager, Scope: access.ScopeAll}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dcc, _, err := s.Login(ctx, "dcc.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	_ = info

	catalog, err := s.ListPermissions(ctx, admin)
	if err != nil || len(catalog) != 24 || catalog[0].Code != access.ViewReleased {
		t.Fatalf("katalog: %d hak, %v", len(catalog), err)
	}
	if _, err := s.ListPermissions(ctx, dcc); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja membaca katalog: %v", err)
	}

	// Contoh D-66: Manager diberi hak edit draft tanpa mengubah kode.
	if err := checker.Authorize(ctx, dcc, access.EditDrafts, access.Resource{}); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("sebelum diubah: %v", err)
	}
	role, err := s.SetRolePermissions(ctx, admin, access.RoleManager,
		[]access.Permission{access.EditDrafts, access.ViewDrafts, access.ViewPerformance, access.GenbaNotes})
	if err != nil {
		t.Fatal(err)
	}
	// Hak dasar L-1 sampai L-3 tetap ada, dan urutannya seperti katalog.
	want := []access.Permission{access.ViewReleased, access.ViewRegister, access.ExportReleased, access.ViewDrafts, access.ViewPerformance, access.EditDrafts, access.GenbaNotes}
	if !slices.Equal(role.Permissions, want) {
		t.Errorf("isi peran: %v", role.Permissions)
	}
	if err := checker.Authorize(ctx, dcc, access.EditDrafts, access.Resource{}); err != nil {
		t.Errorf("hak baru belum berlaku: %v", err)
	}
	if err := checker.Authorize(ctx, dcc, access.SignDigitally, access.Resource{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("hak yang dicabut masih berlaku: %v", err)
	}
	var details string
	if err := s.pool.QueryRow(ctx, `SELECT details::text FROM audit_log WHERE action = 'role.permissions_changed'`).Scan(&details); err != nil ||
		details != `{"added":"E-2","removed":"S-1","role":"manager"}` {
		t.Errorf("audit: %s, %v", details, err)
	}

	code := func(err error) string {
		var invalid *problem.ValidationError
		if !errors.As(err, &invalid) {
			return ""
		}
		return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
	}
	if _, err := s.SetRolePermissions(ctx, dcc, access.RoleAuthor, nil); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja mengubah peran: %v", err)
	}
	for _, id := range []string{access.RoleAdmin, access.RoleViewer} {
		if _, err := s.SetRolePermissions(ctx, admin, id, nil); code(err) != "role:not_allowed" {
			t.Errorf("peran %s: %v", id, err)
		}
	}
	if _, err := s.SetRolePermissions(ctx, admin, access.RoleAuthor, []access.Permission{access.ViewAuditLog}); code(err) != "permissions:not_allowed" {
		t.Errorf("hak admin di peran kerja: %v", err)
	}
	if _, err := s.SetRolePermissions(ctx, admin, access.RoleAuthor, []access.Permission{"X-9"}); code(err) != "permissions:invalid_format" {
		t.Errorf("hak tidak dikenal: %v", err)
	}
	if _, err := s.SetRolePermissions(ctx, admin, "approver", nil); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("peran tidak ada: %v", err)
	}

	// Aturan yang sama dijaga database, juga bila kode aplikasi dilewati.
	for _, sql := range []string{
		`INSERT INTO role_permissions (role_id, permission_code) VALUES ('author', 'A-1')`,
		`DELETE FROM role_permissions WHERE role_id = 'author' AND permission_code = 'L-1'`,
		`DELETE FROM role_permissions WHERE role_id = 'admin' AND permission_code = 'A-4'`,
		`INSERT INTO role_permissions (role_id, permission_code) VALUES ('viewer', 'E-2')`,
	} {
		if _, err := s.pool.Exec(ctx, sql); err == nil {
			t.Errorf("database menerima: %s", sql)
		}
	}
}
