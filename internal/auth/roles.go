package auth

// Isi peran diatur admin (M8.1, D-66, R-SEC-10). Katalog hak tetap; yang
// diubah adalah hak mana yang dipegang satu peran kerja. Aturan yang tidak
// bisa dimatikan (R-SEC-11) dijaga di sini dan sekali lagi oleh trigger di
// database (migrasi 00042):
//   - isi peran admin dan viewer tetap;
//   - peran kerja tidak pernah memegang hak admin (A-1 sampai A-4);
//   - hak dasar L-1 sampai L-3 selalu dipegang setiap peran.
// Setiap perubahan dicatat di audit log (RK-63).

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// PermissionInfo adalah satu hak di katalog.
type PermissionInfo struct {
	Code  access.Permission
	Level string // view, edit, verify, atau admin
}

// basicPermissions adalah hak dasar yang selalu dipegang setiap peran.
var basicPermissions = []access.Permission{access.ViewReleased, access.ViewRegister, access.ExportReleased}

func roleProblem(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

// ListPermissions mengembalikan katalog hak menurut urutan U3 (A-1).
func (s *Service) ListPermissions(ctx context.Context, actor access.Actor) ([]PermissionInfo, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog hak: %w", err)
	}
	out := make([]PermissionInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, PermissionInfo{Code: access.Permission(r.Code), Level: r.Level})
	}
	return out, nil
}

// SetRolePermissions mengganti seluruh hak satu peran kerja (A-1).
func (s *Service) SetRolePermissions(ctx context.Context, actor access.Actor, roleID string, perms []access.Permission) (access.Role, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return access.Role{}, err
	}
	if roleID == access.RoleAdmin || roleID == access.RoleViewer {
		return access.Role{}, roleProblem("role", problem.NotAllowed, "The admin and viewer roles cannot be changed.")
	}
	var role access.Role
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Baris peran dikunci, supaya dua admin tidak mengubah peran yang sama bersamaan.
		_, err := q.LockRole(ctx, roleID)
		if err == pgx.ErrNoRows {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		catalog, err := q.ListPermissions(ctx)
		if err != nil {
			return err
		}
		// Hasil akhirnya disusun menurut urutan katalog, tanpa ganda.
		var want []access.Permission
		for _, p := range perms {
			known := slices.ContainsFunc(catalog, func(c db.ListPermissionsRow) bool { return c.Code == string(p) })
			switch {
			case !known:
				return roleProblem("permissions", problem.InvalidFormat, "Unknown permission: "+string(p)+".")
			case strings.HasPrefix(string(p), "A-"):
				return roleProblem("permissions", problem.NotAllowed, "A work role cannot hold admin rights. Use the separate admin account.")
			}
		}
		for _, c := range catalog {
			code := access.Permission(c.Code)
			if slices.Contains(perms, code) || slices.Contains(basicPermissions, code) {
				want = append(want, code)
			}
		}
		before, err := q.ListPermissionsOfRole(ctx, roleID)
		if err != nil {
			return err
		}
		var added, removed []string
		for _, p := range want {
			if !slices.Contains(before, string(p)) {
				added = append(added, string(p))
				if err := q.AddRolePermission(ctx, db.AddRolePermissionParams{RoleID: roleID, PermissionCode: string(p)}); err != nil {
					return err
				}
			}
		}
		for _, p := range before {
			if !slices.Contains(want, access.Permission(p)) {
				removed = append(removed, p)
				if err := q.RemoveRolePermission(ctx, db.RemoveRolePermissionParams{RoleID: roleID, PermissionCode: p}); err != nil {
					return err
				}
			}
		}
		role = access.Role{ID: roleID, Permissions: want}
		if len(added) == 0 && len(removed) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionRolePermissionsChanged,
			// ID peran berupa teks, bukan UUID, jadi dicatat di rincian dan entri ini tanpa target.
			Details: map[string]string{"role": roleID, "added": strings.Join(added, ", "), "removed": strings.Join(removed, ", ")},
		})
	})
	if err != nil {
		if _, ok := err.(*problem.ValidationError); ok || err == problem.ErrNotFound {
			return access.Role{}, err
		}
		return access.Role{}, fmt.Errorf("menyimpan isi peran: %w", err)
	}
	return s.role(ctx, role.ID)
}
