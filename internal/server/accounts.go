package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
)

// ListRoles menjawab GET /api/v1/roles.
func (a *apiServer) ListRoles(ctx context.Context, _ ListRolesRequestObject) (ListRolesResponseObject, error) {
	roles, err := a.access.ListRoles(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]Role, 0, len(roles))
	for _, r := range roles {
		items = append(items, Role{Id: r.ID, Level: RoleLevel(r.Level), Permissions: permissionCodes(r.Permissions)})
	}
	return ListRoles200JSONResponse{Items: items}, nil
}

// ListAccounts menjawab GET /api/v1/accounts.
func (a *apiServer) ListAccounts(ctx context.Context, _ ListAccountsRequestObject) (ListAccountsResponseObject, error) {
	list, err := a.auth.ListAccounts(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]AccountInfo, 0, len(list))
	for _, info := range list {
		items = append(items, toAPIAccount(info))
	}
	return ListAccounts200JSONResponse{Items: items}, nil
}

// CreateAccount menjawab POST /api/v1/accounts.
func (a *apiServer) CreateAccount(ctx context.Context, req CreateAccountRequestObject) (CreateAccountResponseObject, error) {
	body := req.Body
	in := auth.NewAccount{
		PersonID:       value(body.PersonId),
		PersonName:     value(body.PersonName),
		EmployeeNumber: value(body.EmployeeNumber),
		Kind:           string(body.Kind),
		Login:          body.Login,
		Roles:          toGrants(value(body.Roles)),
	}
	info, temporary, err := a.auth.CreateAccount(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateAccount201JSONResponse{Account: toAPIAccount(info), TemporaryPassword: temporary}, nil
}

// SetAccountRoles menjawab PUT /api/v1/accounts/{accountId}/roles.
func (a *apiServer) SetAccountRoles(ctx context.Context, req SetAccountRolesRequestObject) (SetAccountRolesResponseObject, error) {
	actor := actorFrom(ctx)
	if err := a.auth.SetRoles(ctx, actor, req.AccountId, toGrants(req.Body.Roles)); err != nil {
		return nil, err
	}
	info, err := a.auth.GetAccount(ctx, actor, req.AccountId)
	if err != nil {
		return nil, err
	}
	return SetAccountRoles200JSONResponse(toAPIAccount(info)), nil
}

// ResetAccountPassword menjawab POST /api/v1/accounts/{accountId}/password-reset.
func (a *apiServer) ResetAccountPassword(ctx context.Context, req ResetAccountPasswordRequestObject) (ResetAccountPasswordResponseObject, error) {
	temporary, err := a.auth.ResetPassword(ctx, actorFrom(ctx), req.AccountId)
	if err != nil {
		return nil, err
	}
	return ResetAccountPassword200JSONResponse{TemporaryPassword: temporary}, nil
}

// DisableAccount menjawab POST /api/v1/accounts/{accountId}/disable.
func (a *apiServer) DisableAccount(ctx context.Context, req DisableAccountRequestObject) (DisableAccountResponseObject, error) {
	info, err := a.auth.DisableAccount(ctx, actorFrom(ctx), req.AccountId)
	if err != nil {
		return nil, err
	}
	return DisableAccount200JSONResponse(toAPIAccount(info)), nil
}

// EnableAccount menjawab POST /api/v1/accounts/{accountId}/enable.
func (a *apiServer) EnableAccount(ctx context.Context, req EnableAccountRequestObject) (EnableAccountResponseObject, error) {
	info, err := a.auth.EnableAccount(ctx, actorFrom(ctx), req.AccountId)
	if err != nil {
		return nil, err
	}
	return EnableAccount200JSONResponse(toAPIAccount(info)), nil
}

// toAPIAccount mengubah AccountInfo dari paket auth menjadi AccountInfo kontrak API.
func toAPIAccount(info auth.AccountInfo) AccountInfo {
	roles := make([]RoleGrant, 0, len(info.Roles))
	for _, g := range info.Roles {
		roles = append(roles, RoleGrant{Role: g.Role, Scope: RoleGrantScope(g.Scope)})
	}
	var employeeNumber *string
	if info.EmployeeNumber != "" {
		employeeNumber = &info.EmployeeNumber
	}
	return AccountInfo{
		Id:                 info.ID,
		Kind:               AccountInfoKind(info.Kind),
		Login:              info.Login,
		Person:             PersonInfo{Id: info.PersonID, Name: info.PersonName, EmployeeNumber: employeeNumber},
		Roles:              roles,
		MustChangePassword: info.MustChangePassword,
		LockedUntil:        info.LockedUntil,
		Disabled:           info.Disabled,
		CreatedAt:          info.CreatedAt,
	}
}

// toGrants mengubah RoleGrant kontrak API menjadi RoleGrant paket access.
// Nilai peran dan cakupan diteruskan apa adanya; pemeriksaannya di paket access.
func toGrants(grants []RoleGrant) []access.RoleGrant {
	out := make([]access.RoleGrant, 0, len(grants))
	for _, g := range grants {
		out = append(out, access.RoleGrant{Role: g.Role, Scope: string(g.Scope)})
	}
	return out
}

// value mengembalikan isi pointer, atau nilai kosong bila nil. Field yang
// tidak wajib di kontrak menjadi pointer di kode hasil generate.
func value[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
