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
	return GetNumberSettings200JSONResponse(toAPINumberSettings(settings)), nil
}

// UpdateNumberSettings menjawab PUT /api/v1/number-settings. pattern dan
// revision_digits yang tidak dikirim berarti tetap (D-111).
func (a *apiServer) UpdateNumberSettings(ctx context.Context, req UpdateNumberSettingsRequestObject) (UpdateNumberSettingsResponseObject, error) {
	in := numbering.SettingsInput{FactoryCode: req.Body.FactoryCode, Pattern: req.Body.Pattern, RevisionDigits: req.Body.RevisionDigits}
	settings, err := a.numbering.UpdateSettings(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return UpdateNumberSettings200JSONResponse(toAPINumberSettings(settings)), nil
}

// RegisterDocumentNumber menjawab POST /api/v1/document-numbers (M2.1, M2.2).
// Permintaan yang diulang dengan Idempotency-Key yang sama dijawab 200 dengan
// nomor lamanya, bukan 201 dengan nomor baru.
func (a *apiServer) RegisterDocumentNumber(ctx context.Context, req RegisterDocumentNumberRequestObject) (RegisterDocumentNumberResponseObject, error) {
	in := numbering.NewNumber{CustomerID: req.Body.CustomerId, DocumentTypeID: req.Body.DocumentTypeId}
	if req.Body.InitialRevision != nil {
		in.InitialRevision = *req.Body.InitialRevision
	}
	if req.Params.IdempotencyKey != nil {
		in.IdempotencyKey = *req.Params.IdempotencyKey
	}
	reg, err := a.numbering.Register(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	if reg.Replayed {
		return RegisterDocumentNumber200JSONResponse(toAPIRegistration(reg)), nil
	}
	return RegisterDocumentNumber201JSONResponse(toAPIRegistration(reg)), nil
}

// GetNextDocumentNumber menjawab GET /api/v1/document-numbers/next.
func (a *apiServer) GetNextDocumentNumber(ctx context.Context, req GetNextDocumentNumberRequestObject) (GetNextDocumentNumberResponseObject, error) {
	number, err := a.numbering.NextNumber(ctx, actorFrom(ctx), req.Params.CustomerId, req.Params.DocumentTypeId)
	if err != nil {
		return nil, err
	}
	return GetNextDocumentNumber200JSONResponse{Number: number}, nil
}

// toAPINumberSettings mengubah Settings domain menjadi NumberSettings kontrak API.
func toAPINumberSettings(s numbering.Settings) NumberSettings {
	return NumberSettings{
		FactoryCode: s.FactoryCode, Pattern: s.Pattern,
		SequenceFormat: NumberSettingsSequenceFormat(s.SequenceFormat), RevisionDigits: s.RevisionDigits,
	}
}

// toAPIRegistration mengubah hasil pendaftaran menjadi NumberRegistration kontrak API.
func toAPIRegistration(r numbering.Registration) NumberRegistration {
	items := make([]DocumentNumber, 0, len(r.Numbers))
	for _, n := range r.Numbers {
		items = append(items, toAPIDocumentNumber(n))
	}
	return NumberRegistration{Items: items, Replayed: r.Replayed}
}

// toAPIDocumentNumber mengubah DocumentNumber domain menjadi DocumentNumber kontrak API.
func toAPIDocumentNumber(n numbering.DocumentNumber) DocumentNumber {
	out := DocumentNumber{
		Id: n.ID, Number: n.Number,
		Customer:     CustomerRef{Id: n.CustomerID, Code: n.CustomerCurrentCode, Name: n.CustomerName},
		CustomerCode: n.CustomerCode,
		DocumentType: DocumentTypeRef{Id: n.DocumentTypeID, Key: DocumentTypeRefKey(n.DocumentTypeKey), Name: n.DocumentTypeName},
		Sequence:     n.Sequence, Status: DocumentNumberStatus(n.Status), InitialRevision: n.InitialRevision,
		RevisionLabel: n.RevisionLabel, Origin: DocumentNumberOrigin(n.Origin), BatchId: n.BatchID,
		RegisteredBy: Person{Id: n.RegisteredByID, Name: n.RegisteredByName}, CreatedAt: n.CreatedAt,
		StatusChangedAt: n.StatusChangedAt,
	}
	if n.StatusReason != "" {
		out.StatusReason = &n.StatusReason
	}
	return out
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
