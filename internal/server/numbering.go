package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/numbering"
)

// GetNumberSettings menjawab GET /api/v1/number-settings.
func (a *apiServer) GetNumberSettings(ctx context.Context, _ GetNumberSettingsRequestObject) (GetNumberSettingsResponseObject, error) {
	settings, err := a.numbering.GetSettings(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return GetNumberSettings200JSONResponse{FactoryCode: settings.FactoryCode}, nil
}

// UpdateNumberSettings menjawab PUT /api/v1/number-settings.
func (a *apiServer) UpdateNumberSettings(ctx context.Context, req UpdateNumberSettingsRequestObject) (UpdateNumberSettingsResponseObject, error) {
	settings, err := a.numbering.SetFactoryCode(ctx, actorFrom(ctx), req.Body.FactoryCode)
	if err != nil {
		return nil, err
	}
	return UpdateNumberSettings200JSONResponse{FactoryCode: settings.FactoryCode}, nil
}

// ListDocumentTypes menjawab GET /api/v1/document-types.
func (a *apiServer) ListDocumentTypes(ctx context.Context, _ ListDocumentTypesRequestObject) (ListDocumentTypesResponseObject, error) {
	list, err := a.numbering.ListDocumentTypes(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]DocumentType, 0, len(list))
	for _, d := range list {
		items = append(items, toAPIDocumentType(d))
	}
	return ListDocumentTypes200JSONResponse{Items: items}, nil
}

// CreateDocumentType menjawab POST /api/v1/document-types.
func (a *apiServer) CreateDocumentType(ctx context.Context, req CreateDocumentTypeRequestObject) (CreateDocumentTypeResponseObject, error) {
	in := numbering.DocumentTypeInput{Name: req.Body.Name, Letter: req.Body.Letter}
	created, err := a.numbering.CreateDocumentType(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateDocumentType201JSONResponse(toAPIDocumentType(created)), nil
}

// UpdateDocumentType menjawab PUT /api/v1/document-types/{documentTypeId}.
func (a *apiServer) UpdateDocumentType(ctx context.Context, req UpdateDocumentTypeRequestObject) (UpdateDocumentTypeResponseObject, error) {
	in := numbering.DocumentTypeInput{Name: req.Body.Name, Letter: req.Body.Letter}
	updated, err := a.numbering.UpdateDocumentType(ctx, actorFrom(ctx), req.DocumentTypeId, in)
	if err != nil {
		return nil, err
	}
	return UpdateDocumentType200JSONResponse(toAPIDocumentType(updated)), nil
}

// ListSpecialCodes menjawab GET /api/v1/special-codes.
func (a *apiServer) ListSpecialCodes(ctx context.Context, _ ListSpecialCodesRequestObject) (ListSpecialCodesResponseObject, error) {
	list, err := a.numbering.ListSpecialCodes(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]SpecialCode, 0, len(list))
	for _, c := range list {
		items = append(items, toAPISpecialCode(c))
	}
	return ListSpecialCodes200JSONResponse{Items: items}, nil
}

// CreateSpecialCode menjawab POST /api/v1/special-codes.
func (a *apiServer) CreateSpecialCode(ctx context.Context, req CreateSpecialCodeRequestObject) (CreateSpecialCodeResponseObject, error) {
	in := numbering.NewSpecialCode{Code: req.Body.Code, Name: req.Body.Name}
	created, err := a.numbering.CreateSpecialCode(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateSpecialCode201JSONResponse(toAPISpecialCode(created)), nil
}

// UpdateSpecialCode menjawab PUT /api/v1/special-codes/{specialCodeId}.
func (a *apiServer) UpdateSpecialCode(ctx context.Context, req UpdateSpecialCodeRequestObject) (UpdateSpecialCodeResponseObject, error) {
	renamed, err := a.numbering.RenameSpecialCode(ctx, actorFrom(ctx), req.SpecialCodeId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return UpdateSpecialCode200JSONResponse(toAPISpecialCode(renamed)), nil
}

// toAPIDocumentType mengubah DocumentType domain menjadi DocumentType kontrak
// API. Jenis dokumen tanpa key dikirim tanpa field key.
func toAPIDocumentType(d numbering.DocumentType) DocumentType {
	out := DocumentType{Id: d.ID, Name: d.Name, Letter: d.Letter, Prefix: d.Prefix}
	if d.Key != "" {
		key := DocumentTypeKey(d.Key)
		out.Key = &key
	}
	return out
}

// toAPISpecialCode mengubah SpecialCode domain menjadi SpecialCode kontrak API.
func toAPISpecialCode(c numbering.SpecialCode) SpecialCode {
	return SpecialCode{Id: c.ID, Code: c.Code, Name: c.Name, CreatedAt: c.CreatedAt}
}
