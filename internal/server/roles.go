package server

// Handler isi peran (M8.1, D-66) dan katalog peran (D-199). Aturannya ada di
// internal/auth/roles.go dan internal/auth/role_catalog.go.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
)

// RoleService adalah operasi katalog hak dan isi peran.
type RoleService interface {
	ListPermissions(ctx context.Context, actor access.Actor) ([]auth.PermissionInfo, error)
	SetRolePermissions(ctx context.Context, actor access.Actor, roleID string, perms []access.Permission) (access.Role, error)
	CreateRole(ctx context.Context, actor access.Actor, in auth.RoleInput) (access.Role, error)
	UpdateRole(ctx context.Context, actor access.Actor, roleID string, in auth.RoleInput) (access.Role, error)
	SetRoleActive(ctx context.Context, actor access.Actor, roleID string, active bool) (access.Role, error)
}

// toAPIRole menyalin satu peran ke bentuk API.
func toAPIRole(r access.Role) Role {
	return Role{
		Id: r.ID, Level: RoleLevel(r.Level), NameEn: r.NameEN, NameId: r.NameID,
		Active: r.Active, Builtin: r.Builtin, GroupScope: r.GroupScope, Holders: r.Holders,
		Permissions: permissionCodes(r.Permissions),
	}
}

// CreateRole menjawab POST /api/v1/roles.
func (a *apiServer) CreateRole(ctx context.Context, req CreateRoleRequestObject) (CreateRoleResponseObject, error) {
	role, err := a.roles.CreateRole(ctx, actorFrom(ctx), auth.RoleInput{
		NameEN: req.Body.NameEn, NameID: req.Body.NameId, GroupScope: value(req.Body.GroupScope), CopyFrom: value(req.Body.CopyFrom),
	})
	if err != nil {
		return nil, err
	}
	return CreateRole201JSONResponse(toAPIRole(role)), nil
}

// UpdateRole menjawab PATCH /api/v1/roles/{roleId}.
func (a *apiServer) UpdateRole(ctx context.Context, req UpdateRoleRequestObject) (UpdateRoleResponseObject, error) {
	role, err := a.roles.UpdateRole(ctx, actorFrom(ctx), req.RoleId, auth.RoleInput{
		NameEN: req.Body.NameEn, NameID: req.Body.NameId, GroupScope: req.Body.GroupScope,
	})
	if err != nil {
		return nil, err
	}
	return UpdateRole200JSONResponse(toAPIRole(role)), nil
}

// SetRoleActive menjawab PUT /api/v1/roles/{roleId}/active.
func (a *apiServer) SetRoleActive(ctx context.Context, req SetRoleActiveRequestObject) (SetRoleActiveResponseObject, error) {
	role, err := a.roles.SetRoleActive(ctx, actorFrom(ctx), req.RoleId, req.Body.Active)
	if err != nil {
		return nil, err
	}
	return SetRoleActive200JSONResponse(toAPIRole(role)), nil
}

// ListPermissions menjawab GET /api/v1/permissions.
func (a *apiServer) ListPermissions(ctx context.Context, _ ListPermissionsRequestObject) (ListPermissionsResponseObject, error) {
	list, err := a.roles.ListPermissions(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := PermissionList{Items: make([]PermissionInfo, 0, len(list))}
	for _, p := range list {
		out.Items = append(out.Items, PermissionInfo{Code: string(p.Code), Level: PermissionInfoLevel(p.Level)})
	}
	return ListPermissions200JSONResponse(out), nil
}

// SetRolePermissions menjawab PUT /api/v1/roles/{roleId}/permissions.
func (a *apiServer) SetRolePermissions(ctx context.Context, req SetRolePermissionsRequestObject) (SetRolePermissionsResponseObject, error) {
	perms := make([]access.Permission, 0, len(req.Body.Permissions))
	for _, p := range req.Body.Permissions {
		perms = append(perms, access.Permission(p))
	}
	role, err := a.roles.SetRolePermissions(ctx, actorFrom(ctx), req.RoleId, perms)
	if err != nil {
		return nil, err
	}
	return SetRolePermissions200JSONResponse(toAPIRole(role)), nil
}
