package server

// Import master data dari berkas Excel (I5). Handler tipis: membaca berkas
// dari multipart, lalu meneruskannya ke masterimport.Service.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/masterimport"
)

// GetMasterDataImportTemplate menjawab GET /api/v1/master-data/import-template.
func (a *apiServer) GetMasterDataImportTemplate(ctx context.Context, _ GetMasterDataImportTemplateRequestObject) (GetMasterDataImportTemplateResponseObject, error) {
	data, err := a.masterImport.Template(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	disposition := `attachment; filename="master-data-import-template.xlsx"`
	return GetMasterDataImportTemplate200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: GetMasterDataImportTemplate200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ImportMasterData menjawab POST /api/v1/master-data/import?commit=. Berkas
// yang tidak terbaca diteruskan sebagai berkas kosong, supaya pemeriksaan hak
// tetap lebih dulu.
func (a *apiServer) ImportMasterData(ctx context.Context, req ImportMasterDataRequestObject) (ImportMasterDataResponseObject, error) {
	var data []byte
	for {
		part, err := req.Body.NextPart()
		if err != nil {
			break
		}
		if part.FormName() != "file" {
			continue
		}
		if data, err = io.ReadAll(io.LimitReader(part, masterimport.MaxBytes+1)); err != nil {
			return nil, err
		}
		break
	}
	report, err := a.masterImport.Import(ctx, actorFrom(ctx), data, req.Params.Commit)
	if err != nil {
		return nil, err
	}
	out := MasterDataImport{Customers: toAPIMasterRows(report.Customers), Models: toAPIMasterRows(report.Models),
		New: report.New, Existing: report.Existing, Invalid: report.Invalid, Committed: report.Committed}
	return ImportMasterData200JSONResponse(out), nil
}

func toAPIMasterRows(rows []masterimport.Row) []MasterDataImportRow {
	out := make([]MasterDataImportRow, 0, len(rows))
	for _, r := range rows {
		creates := make([]MasterDataImportRowCreates, 0, len(r.Creates))
		for _, c := range r.Creates {
			creates = append(creates, MasterDataImportRowCreates(c))
		}
		problems := make([]ErrorDetail, 0, len(r.Problems))
		for _, p := range r.Problems {
			problems = append(problems, ErrorDetail{Field: p.Field, Code: ErrorDetailCode(p.Code), Message: p.Message})
		}
		out = append(out, MasterDataImportRow{Row: r.Row, Cells: r.Cells, Status: MasterDataImportRowStatus(r.Status), Creates: creates, Problems: problems})
	}
	return out
}
