package server

// Handler pengelolaan service account API integrasi (M9.1). Aturannya ada di
// internal/integration.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/integration"
)

// ServiceAccountService adalah operasi admin untuk service account.
type ServiceAccountService interface {
	ListAccounts(ctx context.Context, actor access.Actor) ([]integration.Account, error)
	CreateAccount(ctx context.Context, actor access.Actor, in integration.NewAccount) (integration.Account, error)
	IssueToken(ctx context.Context, actor access.Actor, accountID string, days int) (string, integration.Token, error)
	RevokeToken(ctx context.Context, actor access.Actor, accountID, tokenID string) error
	DisableAccount(ctx context.Context, actor access.Actor, accountID string) error
}

func toAPIServiceToken(t integration.Token) ServiceToken {
	return ServiceToken{Id: t.ID, Prefix: t.Prefix, ExpiresAt: t.ExpiresAt, RevokedAt: t.RevokedAt, LastUsedAt: t.LastUsedAt, CreatedAt: t.CreatedAt}
}

func toAPIServiceAccount(a integration.Account) ServiceAccount {
	out := ServiceAccount{Id: a.ID, Name: a.Name, OwnerTeam: a.OwnerTeam, Contact: a.Contact, Purpose: a.Purpose, Scopes: orEmpty(a.Scopes),
		CustomerIds: orEmpty(a.CustomerIDs), CustomerCodes: orEmpty(a.CustomerCodes), Networks: orEmpty(a.Networks), Disabled: a.Disabled,
		CreatedAt: a.CreatedAt, Tokens: make([]ServiceToken, 0, len(a.Tokens))}
	for _, t := range a.Tokens {
		out.Tokens = append(out.Tokens, toAPIServiceToken(t))
	}
	return out
}

// ListServiceAccounts menjawab GET /api/v1/service-accounts.
func (a *apiServer) ListServiceAccounts(ctx context.Context, _ ListServiceAccountsRequestObject) (ListServiceAccountsResponseObject, error) {
	list, err := a.serviceAccounts.ListAccounts(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := ServiceAccountList{Items: make([]ServiceAccount, 0, len(list)), Scopes: integration.Scopes}
	for _, s := range list {
		out.Items = append(out.Items, toAPIServiceAccount(s))
	}
	return ListServiceAccounts200JSONResponse(out), nil
}

// CreateServiceAccount menjawab POST /api/v1/service-accounts.
func (a *apiServer) CreateServiceAccount(ctx context.Context, req CreateServiceAccountRequestObject) (CreateServiceAccountResponseObject, error) {
	b := req.Body
	in := integration.NewAccount{Name: b.Name, OwnerTeam: b.OwnerTeam, Contact: b.Contact, Purpose: b.Purpose, Scopes: b.Scopes, CustomerIDs: b.CustomerIds}
	if b.Networks != nil {
		in.Networks = *b.Networks
	}
	created, err := a.serviceAccounts.CreateAccount(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateServiceAccount201JSONResponse(toAPIServiceAccount(created)), nil
}

// IssueServiceToken menjawab POST /api/v1/service-accounts/{id}/tokens.
func (a *apiServer) IssueServiceToken(ctx context.Context, req IssueServiceTokenRequestObject) (IssueServiceTokenResponseObject, error) {
	days := 0
	if req.Body.Days != nil {
		days = *req.Body.Days
	}
	token, info, err := a.serviceAccounts.IssueToken(ctx, actorFrom(ctx), req.ServiceAccountId, days)
	if err != nil {
		return nil, err
	}
	return IssueServiceToken201JSONResponse{Token: token, Info: toAPIServiceToken(info)}, nil
}

// RevokeServiceToken menjawab DELETE /api/v1/service-accounts/{id}/tokens/{tokenId}.
func (a *apiServer) RevokeServiceToken(ctx context.Context, req RevokeServiceTokenRequestObject) (RevokeServiceTokenResponseObject, error) {
	if err := a.serviceAccounts.RevokeToken(ctx, actorFrom(ctx), req.ServiceAccountId, req.TokenId); err != nil {
		return nil, err
	}
	return RevokeServiceToken204Response{}, nil
}

// DisableServiceAccount menjawab POST /api/v1/service-accounts/{id}/disable.
func (a *apiServer) DisableServiceAccount(ctx context.Context, req DisableServiceAccountRequestObject) (DisableServiceAccountResponseObject, error) {
	if err := a.serviceAccounts.DisableAccount(ctx, actorFrom(ctx), req.ServiceAccountId); err != nil {
		return nil, err
	}
	return DisableServiceAccount204Response{}, nil
}
