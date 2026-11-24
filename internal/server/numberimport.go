package server

// Impor nomor lama dari berkas Excel (I4). Handler tipis: membaca berkas dari
// multipart, lalu meneruskannya ke numbering.Service.

import (
	"bytes"
	"context"
	"io"

	"github.com/dkpauzi/revioten/internal/numbering"
)

// GetNumberImportTemplate menjawab GET /api/v1/document-number-imports/template.
func (a *apiServer) GetNumberImportTemplate(ctx context.Context, _ GetNumberImportTemplateRequestObject) (GetNumberImportTemplateResponseObject, error) {
	data, err := a.numbering.ImportTemplate(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	disposition := `attachment; filename="legacy-numbers-import-template.xlsx"`
	return GetNumberImportTemplate200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: GetNumberImportTemplate200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ImportDocumentNumbersFile menjawab POST
// /api/v1/document-number-imports/file?commit=&one_set=. Berkas yang tidak
// terbaca diteruskan sebagai berkas kosong, supaya pemeriksaan hak tetap lebih
// dulu daripada pesan "pilih berkas".
func (a *apiServer) ImportDocumentNumbersFile(ctx context.Context, req ImportDocumentNumbersFileRequestObject) (ImportDocumentNumbersFileResponseObject, error) {
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
		if data, err = io.ReadAll(io.LimitReader(part, numbering.MaxImportFileBytes+1)); err != nil {
			return nil, err
		}
		break
	}
	report, err := a.numbering.ImportFile(ctx, actorFrom(ctx), numbering.FileImportInput{
		Data: data, OneSet: req.Params.OneSet, DryRun: !req.Params.Commit,
	})
	if err != nil {
		return nil, err
	}
	return ImportDocumentNumbersFile200JSONResponse(toAPIImportReport(report)), nil
}
