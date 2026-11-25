package server

// Import nama proses dari berkas Excel (I2). Handler tipis: membaca berkas
// dari multipart, lalu meneruskannya ke process.Service.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/process"
)

// GetProcessImportTemplate menjawab GET /api/v1/process-imports/template.
func (a *apiServer) GetProcessImportTemplate(ctx context.Context, _ GetProcessImportTemplateRequestObject) (GetProcessImportTemplateResponseObject, error) {
	data, err := a.processes.ImportTemplate(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	disposition := `attachment; filename="processes-import-template.xlsx"`
	return GetProcessImportTemplate200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: GetProcessImportTemplate200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ImportProcesses menjawab POST /api/v1/process-imports?commit=. Berkas yang
// tidak terbaca diteruskan sebagai berkas kosong, supaya pemeriksaan hak tetap
// lebih dulu.
func (a *apiServer) ImportProcesses(ctx context.Context, req ImportProcessesRequestObject) (ImportProcessesResponseObject, error) {
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
		if data, err = io.ReadAll(io.LimitReader(part, process.MaxImportBytes+1)); err != nil {
			return nil, err
		}
		break
	}
	report, err := a.processes.Import(ctx, actorFrom(ctx), data, req.Params.Commit)
	if err != nil {
		return nil, err
	}
	out := ProcessImport{Rows: make([]ProcessImportRow, 0, len(report.Rows)), New: report.New, Existing: report.Existing,
		Invalid: report.Invalid, Committed: report.Committed}
	for _, r := range report.Rows {
		problems := make([]ErrorDetail, 0, len(r.Problems))
		for _, p := range r.Problems {
			problems = append(problems, ErrorDetail{Field: p.Field, Code: ErrorDetailCode(p.Code), Message: p.Message})
		}
		out.Rows = append(out.Rows, ProcessImportRow{
			Row: r.Row, Name: r.Name, AliasesText: r.AliasesText, Aliases: r.Aliases,
			Status: ProcessImportRowStatus(r.Status), NotAdded: r.NotAdded, Problems: problems,
		})
	}
	return ImportProcesses200JSONResponse(out), nil
}
