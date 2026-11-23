package server

// Import akun dari berkas Excel (I1, D-224). Handler tipis: membaca berkas
// dari multipart, lalu meneruskannya ke auth.Service.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/auth"
)

// GetAccountImportTemplate menjawab GET /api/v1/accounts/import-template.
func (a *apiServer) GetAccountImportTemplate(ctx context.Context, _ GetAccountImportTemplateRequestObject) (GetAccountImportTemplateResponseObject, error) {
	data, err := a.auth.AccountImportTemplate(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	disposition := `attachment; filename="accounts-import-template.xlsx"`
	return GetAccountImportTemplate200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: GetAccountImportTemplate200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ImportAccounts menjawab POST /api/v1/accounts/import?commit=. Berkas yang
// tidak terbaca diteruskan sebagai berkas kosong, supaya pemeriksaan hak tetap
// lebih dulu daripada pesan "pilih berkas".
func (a *apiServer) ImportAccounts(ctx context.Context, req ImportAccountsRequestObject) (ImportAccountsResponseObject, error) {
	var data []byte
	for {
		part, err := req.Body.NextPart()
		if err != nil {
			break // io.EOF, atau bukan multipart
		}
		if part.FormName() != "file" {
			continue
		}
		// Satu byte lebih dari batas, supaya berkas yang terlalu besar dikenali layanan.
		if data, err = io.ReadAll(io.LimitReader(part, auth.MaxAccountImportBytes+1)); err != nil {
			return nil, err
		}
		break
	}
	report, err := a.auth.ImportAccounts(ctx, actorFrom(ctx), data, req.Params.Commit)
	if err != nil {
		return nil, err
	}
	return ImportAccounts200JSONResponse(toAPIAccountImport(report)), nil
}

func toAPIAccountImport(r auth.AccountImport) AccountImport {
	out := AccountImport{Rows: make([]AccountImportRow, 0, len(r.Rows)), New: r.New, Existing: r.Existing, Invalid: r.Invalid, Committed: r.Committed}
	if !r.ExpiresAt.IsZero() {
		out.ExpiresAt = &r.ExpiresAt
	}
	for _, row := range r.Rows {
		roles := make([]RoleGrant, 0, len(row.Roles))
		for _, g := range row.Roles {
			roles = append(roles, RoleGrant{Role: g.Role, Scope: RoleGrantScope(g.Scope)})
		}
		problems := make([]ErrorDetail, 0, len(row.Problems))
		for _, p := range row.Problems {
			problems = append(problems, ErrorDetail{Field: p.Field, Code: ErrorDetailCode(p.Code), Message: p.Message})
		}
		out.Rows = append(out.Rows, AccountImportRow{
			Row: row.Row, Name: row.Name, EmployeeNumber: row.EmployeeNumber, Login: row.Login, Email: row.Email,
			RolesText: row.RolesText, ScopeText: row.ScopeText, Roles: roles, Status: AccountImportRowStatus(row.Status), Problems: problems,
		})
	}
	return out
}
