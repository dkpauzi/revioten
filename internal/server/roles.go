package server

// Handler isi peran (M8.1, D-66). Aturannya ada di internal/auth/roles.go.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
)

// RoleService adalah operasi katalog hak dan isi peran.
type RoleService interface {
	ListPermissions(ctx context.Context, actor access.Actor) ([]auth.PermissionInfo, error)
	SetRolePermissions(ctx context.Context, actor access.Actor, roleID string, perms []access.Permission) (access.Role, error)
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
	return SetRolePermissions200JSONResponse(Role{Id: role.ID, Level: RoleLevel(role.Level), Permissions: permissionCodes(role.Permissions)}), nil
}
