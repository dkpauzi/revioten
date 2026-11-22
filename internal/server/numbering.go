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

// ListDocumentNumbers menjawab GET /api/v1/document-numbers (M2.3).
func (a *apiServer) ListDocumentNumbers(ctx context.Context, req ListDocumentNumbersRequestObject) (ListDocumentNumbersResponseObject, error) {
	p := req.Params
	f := numbering.ListFilter{
		CustomerID: deref(p.CustomerId), DocumentTypeID: deref(p.DocumentTypeId), Query: deref(p.Q), Before: deref(p.Before),
	}
	if p.Status != nil {
		f.Status = string(*p.Status)
	}
	if p.Limit != nil {
		f.Limit = *p.Limit
	}
	list, next, err := a.numbering.ListNumbers(ctx, actorFrom(ctx), f)
	if err != nil {
		return nil, err
	}
	items := make([]DocumentNumber, 0, len(list))
	for _, n := range list {
		items = append(items, toAPIDocumentNumber(n))
	}
	out := ListDocumentNumbers200JSONResponse{Items: items}
	if next != "" {
		out.NextBefore = &next
	}
	return out, nil
}

// VoidDocumentNumber menjawab POST /api/v1/document-numbers/{documentNumberId}/void (M2.3).
func (a *apiServer) VoidDocumentNumber(ctx context.Context, req VoidDocumentNumberRequestObject) (VoidDocumentNumberResponseObject, error) {
	n, err := a.numbering.Void(ctx, actorFrom(ctx), req.DocumentNumberId, deref(req.Body.Reason))
	if err != nil {
		return nil, err
	}
	return VoidDocumentNumber200JSONResponse(toAPIDocumentNumber(n)), nil
}

// MakeDocumentNumberObsolete menjawab POST /api/v1/document-numbers/{documentNumberId}/obsolete (M2.3).
func (a *apiServer) MakeDocumentNumberObsolete(ctx context.Context, req MakeDocumentNumberObsoleteRequestObject) (MakeDocumentNumberObsoleteResponseObject, error) {
	n, err := a.numbering.MakeObsolete(ctx, actorFrom(ctx), req.DocumentNumberId, deref(req.Body.Reason))
	if err != nil {
		return nil, err
	}
	return MakeDocumentNumberObsolete200JSONResponse(toAPIDocumentNumber(n)), nil
}

// RegisterDocumentNumberSet menjawab POST /api/v1/document-numbers/sets (M2.4).
func (a *apiServer) RegisterDocumentNumberSet(ctx context.Context, req RegisterDocumentNumberSetRequestObject) (RegisterDocumentNumberSetResponseObject, error) {
	in := numbering.NewSet{CustomerID: req.Body.CustomerId, IdempotencyKey: deref(req.Params.IdempotencyKey)}
	if req.Body.InitialRevision != nil {
		in.InitialRevision = *req.Body.InitialRevision
	}
	reg, err := a.numbering.RegisterSet(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	if reg.Replayed {
		return RegisterDocumentNumberSet200JSONResponse(toAPIRegistration(reg)), nil
	}
	return RegisterDocumentNumberSet201JSONResponse(toAPIRegistration(reg)), nil
}

// ImportDocumentNumbers menjawab POST /api/v1/document-number-imports (M2.5).
// Pratinjau (dry_run) dan impor sungguhan sama-sama dijawab 200 dengan laporan.
func (a *apiServer) ImportDocumentNumbers(ctx context.Context, req ImportDocumentNumbersRequestObject) (ImportDocumentNumbersResponseObject, error) {
	in := numbering.ImportInput{
		CustomerID: req.Body.CustomerId, Status: string(req.Body.Status), Text: req.Body.Text,
		DryRun: req.Body.DryRun != nil && *req.Body.DryRun,
		OneSet: req.Body.OneSet != nil && *req.Body.OneSet,
	}
	report, err := a.numbering.Import(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return ImportDocumentNumbers200JSONResponse(toAPIImportReport(report)), nil
}

// toAPIImportReport mengubah ImportReport domain menjadi NumberImportReport
// kontrak API. Daftar kosong dikirim sebagai [] dan bukan null.
func toAPIImportReport(r numbering.ImportReport) NumberImportReport {
	out := NumberImportReport{
		DryRun: r.DryRun, Status: NumberImportReportStatus(r.Status), Imported: r.Imported,
		Lines:      make([]NumberImportLine, 0, len(r.Lines)),
		Groups:     make([]NumberImportGroup, 0, len(r.Groups)),
		Misaligned: make([]NumberMisalignment, 0, len(r.Misaligned)),
	}
	for _, l := range r.Lines {
		line := NumberImportLine{Line: l.Line, Text: l.Text, Result: NumberImportLineResult(l.Result)}
		if l.FirstLine != 0 {
			line.FirstLine = &l.FirstLine
		}
		if l.Derived {
			line.Derived = &l.Derived
		}
		out.Lines = append(out.Lines, line)
	}
	for _, g := range r.Groups {
		group := NumberImportGroup{
			CustomerCode: g.CustomerCode, DocumentType: toAPIDocumentTypeRef(g.DocumentType), Count: g.Count,
			Gaps: make([]SequenceRange, 0, len(g.Gaps)),
		}
		for _, gap := range g.Gaps {
			group.Gaps = append(group.Gaps, SequenceRange{First: gap.First, Last: gap.Last})
		}
		if g.Highest != "" {
			group.Highest = &g.Highest
		}
		out.Groups = append(out.Groups, group)
	}
	for _, m := range r.Misaligned {
		present := make([]DocumentTypeRef, 0, len(m.Present))
		for _, d := range m.Present {
			present = append(present, toAPIDocumentTypeRef(d))
		}
		out.Misaligned = append(out.Misaligned, NumberMisalignment{
			CustomerCode: m.CustomerCode, Range: SequenceRange{First: m.Range.First, Last: m.Range.Last}, Present: present,
		})
	}
	return out
}

// toAPIDocumentTypeRef mengubah jenis dokumen PFD, PFMEA, atau CP menjadi
// DocumentTypeRef kontrak API.
func toAPIDocumentTypeRef(d numbering.DocumentType) DocumentTypeRef {
	return DocumentTypeRef{Id: d.ID, Key: DocumentTypeRefKey(d.Key), Name: d.Name}
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

// deref mengembalikan isi pointer teks, atau "" bila nil (parameter yang tidak dikirim).
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
