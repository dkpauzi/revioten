package server

// Import isi process template dari berkas Excel (I3). Handler tipis: membaca
// berkas dari multipart, lalu meneruskannya ke template.ProcessService.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/template"
)

// GetProcessContentImportTemplate menjawab GET /api/v1/process-template-imports/template.
func (a *apiServer) GetProcessContentImportTemplate(ctx context.Context, _ GetProcessContentImportTemplateRequestObject) (GetProcessContentImportTemplateResponseObject, error) {
	data, err := a.processTemplates.ContentImportTemplate(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	disposition := `attachment; filename="process-template-content-import-template.xlsx"`
	return GetProcessContentImportTemplate200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: GetProcessContentImportTemplate200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ImportProcessContent menjawab POST /api/v1/process-template-imports?commit=.
// Berkas yang tidak terbaca diteruskan sebagai berkas kosong, supaya
// pemeriksaan hak tetap lebih dulu.
func (a *apiServer) ImportProcessContent(ctx context.Context, req ImportProcessContentRequestObject) (ImportProcessContentResponseObject, error) {
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
		if data, err = io.ReadAll(io.LimitReader(part, template.MaxContentImportBytes+1)); err != nil {
			return nil, err
		}
		break
	}
	report, err := a.processTemplates.ImportContent(ctx, actorFrom(ctx), data, req.Params.Commit)
	if err != nil {
		return nil, err
	}
	out := ProcessContentImport{
		Rows:     make([]ProcessContentImportRow, 0, len(report.Rows)),
		Versions: make([]ProcessContentImportVersion, 0, len(report.Versions)),
		New:      report.New, Existing: report.Existing, Invalid: report.Invalid, Committed: report.Committed,
	}
	for _, r := range report.Rows {
		problems := make([]ErrorDetail, 0, len(r.Problems))
		for _, p := range r.Problems {
			problems = append(problems, ErrorDetail{Field: p.Field, Code: ErrorDetailCode(p.Code), Message: p.Message})
		}
		out.Rows = append(out.Rows, ProcessContentImportRow{
			Sheet: r.Sheet, Row: r.Row, Process: r.Process, Subprocess: r.Subprocess, Text: r.Text,
			ScopeLevel: r.ScopeLevel, ScopeTarget: r.ScopeTarget, Status: ProcessContentImportRowStatus(r.Status), Problems: problems,
		})
	}
	for _, v := range report.Versions {
		out.Versions = append(out.Versions, ProcessContentImportVersion{
			Process: v.Process, Version: v.Version, Change: ProcessContentImportVersionChange(v.Change), Points: v.Points,
		})
	}
	return ImportProcessContent200JSONResponse(out), nil
}
