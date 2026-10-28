package access

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// RoleGrant adalah satu peran yang dipegang akun, beserta cakupannya.
type RoleGrant struct {
	Role  string
	Scope string // ScopeAll atau ScopeOwnProjectGroup
}

// Role adalah satu peran di katalog beserta hak-haknya.
type Role struct {
	ID          string
	Level       string // view, edit, verify, atau admin
	Permissions []Permission
}

// groupScopedRoles adalah peran yang boleh bercakupan Project Group sendiri:
// Penyusun dan Verifikator milik Project Leader dan Project Engineer (U3 bagian 2).
var groupScopedRoles = []string{RoleAuthor, RoleVerifier}

// ListRoles mengembalikan katalog peran beserta haknya (A-1).
func (c *Checker) ListRoles(ctx context.Context, actor Actor) ([]Role, error) {
	if err := c.Authorize(ctx, actor, ManageUsers, Resource{}); err != nil {
		return nil, err
	}
	roles, err := c.q.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog peran: %w", err)
	}
	perms, err := c.q.ListRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog hak: %w", err)
	}
	list := make([]Role, 0, len(roles))
	for _, r := range roles {
		role := Role{ID: r.ID, Level: r.Level}
		for _, p := range perms {
			if p.RoleID == r.ID {
				role.Permissions = append(role.Permissions, Permission(p.PermissionCode))
			}
		}
		list = append(list, role)
	}
	return list, nil
}

// ValidateRoles memeriksa peran yang akan diberikan ke akun berjenis kind.
// Aturan ini tidak bisa dimatikan siapa pun (R-SEC-11 butir 8, D-72):
// akun admin hanya memegang peran Admin, yang dipasang otomatis, dan akun
// kerja tidak pernah memegang peran Admin.
func ValidateRoles(kind string, grants []RoleGrant) []problem.Problem {
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
		switch {
		case g.Role == RoleAdmin:
			return notAllowed("A work account cannot hold the admin role. Use a separate admin account.")
		case g.Role == RoleViewer:
			return notAllowed("Every account already has the viewer rights.")
		case !slices.Contains(workRoles, g.Role):
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "Unknown role: " + g.Role + "."}}
		case slices.Contains(seen, g.Role):
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "A role is listed twice: " + g.Role + "."}}
		case g.Scope != ScopeAll && g.Scope != ScopeOwnProjectGroup:
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "Unknown scope for role " + g.Role + "."}}
		case g.Scope == ScopeOwnProjectGroup && !slices.Contains(groupScopedRoles, g.Role):
			return notAllowed("Only the author and verifier roles can be limited to the own project group.")
		}
		seen = append(seen, g.Role)
	}
	return nil
}

// workRoles adalah peran yang boleh dipegang akun kerja.
var workRoles = []string{
	RoleAuthor, RoleTemplateMaintainer, RolePFMEACPChecker, RoleVerifier, RoleDocumentControl, RoleMigration,
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

// SetRoles mengganti peran sebuah akun kerja (A-1). Peran akun admin tidak bisa
// diubah, jadi akun admin terakhir juga tidak bisa kehilangan peran Admin (D-73).
func (c *Checker) SetRoles(ctx context.Context, actor Actor, accountID string, grants []RoleGrant) error {
	if err := c.Authorize(ctx, actor, ManageUsers, Resource{}); err != nil {
		return err
	}
	if !db.IsUUID(accountID) {
		return problem.ErrNotFound
	}
	// Bentuk baku UUID memakai huruf kecil, seperti yang ditulis database.
	accountID = strings.ToLower(accountID)
	kind, err := c.q.GetAccountKind(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return problem.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("membaca akun: %w", err)
	}
	if kind == AccountAdmin {
		return &problem.ValidationError{Problems: []problem.Problem{{
			Field: "roles", Code: problem.NotAllowed, Message: "The roles of an admin account cannot be changed.",
		}}}
	}
	if problems := ValidateRoles(kind, grants); len(problems) > 0 {
		return &problem.ValidationError{Problems: problems}
	}

	err = db.WithTx(ctx, c.pool, func(tx pgx.Tx) error {
		return ReplaceRoles(ctx, c.q.WithTx(tx), accountID, grants)
	})
	if err != nil {
		return fmt.Errorf("menyimpan peran: %w", err)
	}
	// M1.3: setiap perubahan hak juga dicatat di audit log (R-SEC-10).
	slog.Info("peran akun diubah", "oleh", actor.Login, "akun", accountID, "peran", describeGrants(grants))
	return nil
}

// describeGrants menulis peran seperti "author(own_project_group), verifier(all)".
func describeGrants(grants []RoleGrant) string {
	parts := make([]string, 0, len(grants))
	for _, g := range grants {
		parts = append(parts, g.Role+"("+g.Scope+")")
	}
	return strings.Join(parts, ", ")
}
