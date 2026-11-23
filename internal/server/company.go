package server

// Nama perusahaan dan logo cetakan (bagian C, E-144). Admin mengubahnya; semua
// akun yang login bisa membaca dan melihat logonya.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/settings"
)

func toAPICompany(c settings.Company) Company {
	out := Company{Name: c.Name, CustomLogo: c.CustomLogo, LogoUpdatedAt: c.LogoUpdatedAt}
	if c.LogoSHA256 != "" {
		hash := c.LogoSHA256
		out.LogoSha256 = &hash
	}
	return out
}

// GetCompany menjawab GET /api/v1/company.
func (a *apiServer) GetCompany(ctx context.Context, _ GetCompanyRequestObject) (GetCompanyResponseObject, error) {
	c, err := a.settings.Company(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return GetCompany200JSONResponse(toAPICompany(c)), nil
}

// SetCompanyName menjawab PUT /api/v1/company.
func (a *apiServer) SetCompanyName(ctx context.Context, req SetCompanyNameRequestObject) (SetCompanyNameResponseObject, error) {
	c, err := a.settings.SetCompanyName(ctx, actorFrom(ctx), req.Body.Name)
	if err != nil {
		return nil, err
	}
	return SetCompanyName200JSONResponse(toAPICompany(c)), nil
}

// GetCompanyLogo menjawab GET /api/v1/company/logo. Hak membaca diperiksa
// lewat Company, karena Logo sendiri dipakai saat mencetak tanpa aktor.
func (a *apiServer) GetCompanyLogo(ctx context.Context, _ GetCompanyLogoRequestObject) (GetCompanyLogoResponseObject, error) {
	if _, err := a.settings.Company(ctx, actorFrom(ctx)); err != nil {
		return nil, err
	}
	logo, err := a.settings.Logo(ctx)
	if err != nil {
		return nil, err
	}
	return GetCompanyLogo200ImagepngResponse{Body: bytes.NewReader(logo), ContentLength: int64(len(logo))}, nil
}

// SetCompanyLogo menjawab PUT /api/v1/company/logo (isi permintaan: PNG).
func (a *apiServer) SetCompanyLogo(ctx context.Context, req SetCompanyLogoRequestObject) (SetCompanyLogoResponseObject, error) {
	data, err := io.ReadAll(io.LimitReader(req.Body, settings.MaxLogoBytes+1))
	if err != nil {
		return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "logo", Code: problem.InvalidFormat, Message: "The logo could not be read."}}}
	}
	c, err := a.settings.SetLogo(ctx, actorFrom(ctx), data, req.Params.Revise)
	if err != nil {
		return nil, err
	}
	return SetCompanyLogo200JSONResponse(toAPICompany(c)), nil
}

// ResetCompanyLogo menjawab DELETE /api/v1/company/logo.
func (a *apiServer) ResetCompanyLogo(ctx context.Context, req ResetCompanyLogoRequestObject) (ResetCompanyLogoResponseObject, error) {
	c, err := a.settings.ResetLogo(ctx, actorFrom(ctx), req.Params.Revise)
	if err != nil {
		return nil, err
	}
	return ResetCompanyLogo200JSONResponse(toAPICompany(c)), nil
}

// GetPendingLogoChange menjawab GET /api/v1/customers/{customerId}/pending-logo-change (D-223).
func (a *apiServer) GetPendingLogoChange(ctx context.Context, req GetPendingLogoChangeRequestObject) (GetPendingLogoChangeResponseObject, error) {
	p, err := a.changeRequests.PendingLogoChange(ctx, actorFrom(ctx), req.CustomerId)
	if err != nil {
		return nil, err
	}
	out := PendingLogoChange{Pending: p != nil}
	if p != nil {
		id, at, def := p.ID, p.ChangedAt, p.LogoSHA256 == ""
		out.LogoChangeId, out.ChangedAt, out.DefaultLogo = &id, &at, &def
	}
	return GetPendingLogoChange200JSONResponse(out), nil
}

// SubmitLogoChange menjawab POST /api/v1/logo-changes (D-223).
func (a *apiServer) SubmitLogoChange(ctx context.Context, req SubmitLogoChangeRequestObject) (SubmitLogoChangeResponseObject, error) {
	b := req.Body
	in := change.LogoChange{CustomerID: b.CustomerId, LogoChangeID: b.LogoChangeId, ChangeTypeID: b.ChangeTypeId, Reason: b.Reason}
	if b.FourMCategoryId != nil {
		in.FourMCategoryID = *b.FourMCategoryId
	}
	c, err := a.changeRequests.SubmitLogoChange(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return SubmitLogoChange201JSONResponse(toAPIChange(c)), nil
}
