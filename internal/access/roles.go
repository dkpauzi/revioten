package access

import (
	"context"
	"fmt"
	"slices"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// RoleGrant adalah satu peran yang dipegang akun, beserta cakupannya.
type RoleGrant struct {
	Role  string
	Scope string // ScopeAll atau ScopeOwnProjectGroup
}

// Role adalah satu peran di katalog beserta hak-haknya. Sejak D-199 peran
// punya nama sendiri dalam dua bahasa, dan admin boleh menambah peran.
type Role struct {
	ID          string
	Level       string // view, edit, verify, atau admin
	NameEN      string
	NameID      string
	Active      bool // peran nonaktif tidak bisa diberikan lagi
	Builtin     bool // peran bawaan sistem, bukan tambahan admin
	GroupScope  bool // boleh dibatasi ke Project Group sendiri (D-71)
	Holders     int  // jumlah akun yang memegangnya
	Permissions []Permission
}

// ListRoles mengembalikan katalog peran beserta haknya (A-1).
func (c *Checker) ListRoles(ctx context.Context, actor Actor) ([]Role, error) {
	if err := c.Authorize(ctx, actor, ManageUsers, Resource{}); err != nil {
		return nil, err
	}
	return LoadRoles(ctx, c.q)
}

// LoadRoles membaca katalog peran tanpa pemeriksaan izin. Hanya dipanggil
// operasi yang sudah lolos Authorize.
func LoadRoles(ctx context.Context, q *db.Queries) ([]Role, error) {
	roles, err := q.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog peran: %w", err)
	}
	perms, err := q.ListRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog hak: %w", err)
	}
	list := make([]Role, 0, len(roles))
	for _, r := range roles {
		role := Role{
			ID: r.ID, Level: r.Level, NameEN: r.NameEn, NameID: r.NameID,
			Active: r.Active, Builtin: r.Builtin, GroupScope: r.GroupScope, Holders: int(r.Holders),
		}
		for _, p := range perms {
			if p.RoleID == r.ID {
				role.Permissions = append(role.Permissions, Permission(p.PermissionCode))
			}
		}
		list = append(list, role)
	}
	return list, nil
}

// ValidateRoles memeriksa peran yang akan diberikan ke akun berjenis kind,
// terhadap katalog peran. Aturan ini tidak bisa dimatikan siapa pun
// (R-SEC-11 butir 8, D-72): akun admin hanya memegang peran Admin, yang
// dipasang otomatis, dan akun kerja tidak pernah memegang peran Admin.
// Peran nonaktif tidak bisa diberikan (D-199).
func ValidateRoles(kind string, grants []RoleGrant, catalog []Role) []problem.Problem {
	notAllowed := func(message string) []problem.Problem {
		return []problem.Problem{{Field: "roles", Code: problem.NotAllowed, Message: message}}
	}
	if kind == AccountAdmin {
		if len(grants) > 0 {
			return notAllowed("An admin account always holds only the admin role.")
		}
		return nil
	}

	var seen []string
	for _, g := range grants {
		i := slices.IndexFunc(catalog, func(r Role) bool { return r.ID == g.Role })
		switch {
		case g.Role == RoleAdmin:
			return notAllowed("A work account cannot hold the admin role. Use a separate admin account.")
		case g.Role == RoleViewer:
			return notAllowed("Every account already has the viewer rights.")
		case i < 0:
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "Unknown role: " + g.Role + "."}}
		case !catalog[i].Active:
			return notAllowed("The role " + catalog[i].NameEN + " is inactive and cannot be given.")
		case slices.Contains(seen, g.Role):
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "A role is listed twice: " + g.Role + "."}}
		case g.Scope != ScopeAll && g.Scope != ScopeOwnProjectGroup:
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "Unknown scope for role " + g.Role + "."}}
		case g.Scope == ScopeOwnProjectGroup && !catalog[i].GroupScope:
			return notAllowed("The role " + catalog[i].NameEN + " cannot be limited to the own project group.")
		}
		seen = append(seen, g.Role)
	}
	return nil
}

// ReplaceRoles mengganti seluruh peran akun dengan grants. Dipanggil di dalam
// transaksi pemanggil, setelah ValidateRoles.
func ReplaceRoles(ctx context.Context, q *db.Queries, accountID string, grants []RoleGrant) error {
	if err := q.DeleteAccountRoles(ctx, accountID); err != nil {
		return err
	}
	for _, g := range grants {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: accountID, RoleID: g.Role, Scope: g.Scope}); err != nil {
			return err
		}
	}
	return nil
}
