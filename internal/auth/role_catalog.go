package auth

// Katalog peran yang bisa ditambah admin (D-199, gaya Spatie Laravel
// Permission): admin menambah peran, mengganti namanya, dan menonaktifkan
// peran yang tidak dipegang akun mana pun. Peran tidak pernah dihapus,
// supaya audit log tetap terbaca. Isi hak peran tetap diatur lewat
// SetRolePermissions di roles.go. Database menjaga aturan yang sama
// (trigger roles_guard, migrasi 00049).

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// RoleInput adalah isian peran baru atau perubahan peran.
type RoleInput struct {
	NameEN     string
	NameID     string
	GroupScope bool // boleh dibatasi ke Project Group sendiri (D-71)
	// CopyFrom (opsional, hanya untuk peran baru): hak peran ini disalin.
	CopyFrom string
}

// role membaca satu peran beserta haknya dari katalog.
func (s *Service) role(ctx context.Context, roleID string) (access.Role, error) {
	roles, err := access.LoadRoles(ctx, s.q)
	if err != nil {
		return access.Role{}, err
	}
	for _, r := range roles {
		if r.ID == roleID {
			return r, nil
		}
	}
	return access.Role{}, problem.ErrNotFound
}

// checkRoleNames memeriksa dua nama peran: wajib, paling panjang 60 huruf,
// dan belum dipakai peran lain.
func checkRoleNames(ctx context.Context, q *db.Queries, roleID string, in RoleInput) error {
	var problems []problem.Problem
	for _, f := range []struct{ field, value string }{{"name_en", in.NameEN}, {"name_id", in.NameID}} {
		switch {
		case f.value == "":
			problems = append(problems, problem.Problem{Field: f.field, Code: problem.Required, Message: "Fill in the name of the role."})
		case len([]rune(f.value)) > 60:
			problems = append(problems, problem.Problem{Field: f.field, Code: problem.TooLong, Message: "The name can be at most 60 characters."})
		}
	}
	if len(problems) > 0 {
		return &problem.ValidationError{Problems: problems}
	}
	taken, err := q.RoleNameTaken(ctx, db.RoleNameTakenParams{ID: roleID, NameEn: in.NameEN, NameID: in.NameID})
	if err != nil {
		return err
	}
	if taken {
		return roleProblem("name_en", problem.Taken, "Another role already has this name.")
	}
	return nil
}

// roleSlug membuat ID peran dari nama Inggris: huruf kecil, angka, dan garis
// bawah, misalnya "QA Auditor" menjadi "qa_auditor". ID tetap selamanya,
// walau namanya kemudian diganti.
func roleSlug(name string) string {
	var b strings.Builder
	under := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			under = false
		case !under && b.Len() > 0:
			b.WriteByte('_')
			under = true
		}
	}
	slug := strings.TrimRight(b.String(), "_")
	if len(slug) > 40 {
		slug = strings.TrimRight(slug[:40], "_")
	}
	if slug == "" {
		slug = "role"
	}
	return slug
}

// keepError meneruskan galat validasi dan "tidak ada" apa adanya, dan
// membungkus galat lain dengan keterangan.
func keepError(err error, what string) error {
	var v *problem.ValidationError
	if errors.As(err, &v) || errors.Is(err, problem.ErrNotFound) {
		return err
	}
	return fmt.Errorf("%s: %w", what, err)
}

// CreateRole menambah peran kerja baru (A-1). Peran baru memegang hak dasar
// L-1 sampai L-3, ditambah hak peran CopyFrom bila diisi (tanpa hak admin).
func (s *Service) CreateRole(ctx context.Context, actor access.Actor, in RoleInput) (access.Role, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return access.Role{}, err
	}
	in.NameEN, in.NameID, in.CopyFrom = strings.TrimSpace(in.NameEN), strings.TrimSpace(in.NameID), strings.TrimSpace(in.CopyFrom)
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockRoleCatalog(ctx); err != nil {
			return err
		}
		if err := checkRoleNames(ctx, q, "", in); err != nil {
			return err
		}
		perms := slices.Clone(basicPermissions)
		if in.CopyFrom != "" {
			if in.CopyFrom == access.RoleAdmin {
				return roleProblem("copy_from", problem.NotAllowed, "The rights of the admin role cannot be copied to a work role.")
			}
			exists, err := q.RoleIDTaken(ctx, in.CopyFrom)
			if err != nil {
				return err
			}
			if !exists {
				return roleProblem("copy_from", problem.InvalidFormat, "Unknown role: "+in.CopyFrom+".")
			}
			from, err := q.ListPermissionsOfRole(ctx, in.CopyFrom)
			if err != nil {
				return err
			}
			for _, p := range from {
				if !slices.Contains(perms, access.Permission(p)) && !strings.HasPrefix(p, "A-") {
					perms = append(perms, access.Permission(p))
				}
			}
		}
		// ID dari nama; bila sudah dipakai, diberi angka di belakang.
		base := roleSlug(in.NameEN)
		id = base
		for n := 2; ; n++ {
			taken, err := q.RoleIDTaken(ctx, id)
			if err != nil {
				return err
			}
			if !taken {
				break
			}
			id = fmt.Sprintf("%s_%d", base, n)
		}
		if err := q.CreateRole(ctx, db.CreateRoleParams{ID: id, NameEn: in.NameEN, NameID: in.NameID, GroupScope: in.GroupScope}); err != nil {
			return err
		}
		codes := make([]string, 0, len(perms))
		for _, p := range perms {
			if err := q.AddRolePermission(ctx, db.AddRolePermissionParams{RoleID: id, PermissionCode: string(p)}); err != nil {
				return err
			}
			codes = append(codes, string(p))
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action:  audit.ActionRoleCreated,
			Details: map[string]string{"role": id, "name": in.NameEN, "copied_from": in.CopyFrom, "permissions": strings.Join(codes, ", ")},
		})
	})
	if err != nil {
		return access.Role{}, keepError(err, "menambah peran")
	}
	return s.role(ctx, id)
}

// UpdateRole mengganti nama peran dan apakah peran boleh dibatasi ke Project
// Group sendiri (A-1). ID peran tidak berubah. Peran Admin dan Viewer tetap
// (R-SEC-11). Cakupan Project Group tidak bisa dimatikan selama masih ada
// akun yang memegang peran itu dengan cakupan tersebut.
func (s *Service) UpdateRole(ctx context.Context, actor access.Actor, roleID string, in RoleInput) (access.Role, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return access.Role{}, err
	}
	if roleID == access.RoleAdmin || roleID == access.RoleViewer {
		return access.Role{}, roleProblem("role", problem.NotAllowed, "The admin and viewer roles cannot be changed.")
	}
	in.NameEN, in.NameID = strings.TrimSpace(in.NameEN), strings.TrimSpace(in.NameID)
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockRoleCatalog(ctx); err != nil {
			return err
		}
		before, err := q.GetRole(ctx, roleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := checkRoleNames(ctx, q, roleID, in); err != nil {
			return err
		}
		if before.GroupScope && !in.GroupScope {
			scoped, err := q.CountScopedRoleHolders(ctx, roleID)
			if err != nil {
				return err
			}
			if scoped > 0 {
				return roleProblem("group_scope", problem.NotAllowed,
					"Some accounts hold this role for their own project group only. Change those accounts first.")
			}
		}
		if before.NameEn == in.NameEN && before.NameID == in.NameID && before.GroupScope == in.GroupScope {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.UpdateRole(ctx, db.UpdateRoleParams{ID: roleID, NameEn: in.NameEN, NameID: in.NameID, GroupScope: in.GroupScope}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionRoleUpdated,
			Details: map[string]string{
				"role": roleID, "name_before": before.NameEn, "name_after": in.NameEN,
				"group_scope": fmt.Sprint(in.GroupScope),
			},
		})
	})
	if err != nil {
		return access.Role{}, keepError(err, "mengubah peran")
	}
	return s.role(ctx, roleID)
}

// SetRoleActive menonaktifkan atau mengaktifkan lagi satu peran kerja (A-1).
// Peran hanya bisa dinonaktifkan bila tidak ada akun yang memegangnya.
func (s *Service) SetRoleActive(ctx context.Context, actor access.Actor, roleID string, active bool) (access.Role, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return access.Role{}, err
	}
	if roleID == access.RoleAdmin || roleID == access.RoleViewer {
		return access.Role{}, roleProblem("role", problem.NotAllowed, "The admin and viewer roles cannot be changed.")
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		locked, err := q.LockRole(ctx, roleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if locked.Active == active {
			return nil
		}
		if !active {
			holders, err := q.CountRoleHolders(ctx, roleID)
			if err != nil {
				return err
			}
			if holders > 0 {
				return roleProblem("active", problem.NotAllowed,
					fmt.Sprintf("%d accounts still hold this role. Take the role away from them first.", holders))
			}
		}
		if err := q.SetRoleActive(ctx, db.SetRoleActiveParams{ID: roleID, Active: active}); err != nil {
			return err
		}
		action := audit.ActionRoleActivated
		if !active {
			action = audit.ActionRoleDeactivated
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, Details: map[string]string{"role": roleID}})
	})
	if err != nil {
		return access.Role{}, keepError(err, "mengubah status peran")
	}
	return s.role(ctx, roleID)
}
