package server

// Handler perubahan (M5.1 dan seterusnya). Aturannya ada di
// internal/change/request.go; handler hanya menerjemahkan bentuk data.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/problem"
)

// ListChanges menjawab GET /api/v1/changes?customer_id=.
func (a *apiServer) ListChanges(ctx context.Context, req ListChangesRequestObject) (ListChangesResponseObject, error) {
	list, err := a.changeRequests.List(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	return ListChanges200JSONResponse(toAPIChangeList(list)), nil
}

// CreateChange menjawab POST /api/v1/changes.
func (a *apiServer) CreateChange(ctx context.Context, req CreateChangeRequestObject) (CreateChangeResponseObject, error) {
	b := req.Body
	c, err := a.changeRequests.Create(ctx, actorFrom(ctx), change.NewChange{
		CustomerID: b.CustomerId, TriggerID: b.TriggerId, ChangeTypeID: b.ChangeTypeId, FourMCategoryID: deref(b.FourMCategoryId),
		Reason: b.Reason, SourceReference: deref(b.SourceReference), DocumentIDs: b.DocumentIds,
	})
	if err != nil {
		return nil, err
	}
	return CreateChange201JSONResponse(toAPIChange(c)), nil
}

// GetChange menjawab GET /api/v1/changes/{changeId}.
func (a *apiServer) GetChange(ctx context.Context, req GetChangeRequestObject) (GetChangeResponseObject, error) {
	c, err := a.changeRequests.Get(ctx, actorFrom(ctx), req.ChangeId)
	if err != nil {
		return nil, err
	}
	return GetChange200JSONResponse(toAPIChange(c)), nil
}

// AddChangeDocument menjawab POST /api/v1/changes/{changeId}/documents.
func (a *apiServer) AddChangeDocument(ctx context.Context, req AddChangeDocumentRequestObject) (AddChangeDocumentResponseObject, error) {
	c, err := a.changeRequests.AddDocument(ctx, actorFrom(ctx), req.ChangeId, req.Body.DocumentId)
	if err != nil {
		return nil, err
	}
	return AddChangeDocument201JSONResponse(toAPIChange(c)), nil
}

// RemoveChangeDocument menjawab DELETE /api/v1/changes/{changeId}/documents/{documentId}.
func (a *apiServer) RemoveChangeDocument(ctx context.Context, req RemoveChangeDocumentRequestObject) (RemoveChangeDocumentResponseObject, error) {
	c, err := a.changeRequests.RemoveDocument(ctx, actorFrom(ctx), req.ChangeId, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return RemoveChangeDocument200JSONResponse(toAPIChange(c)), nil
}

// ListDocumentChanges menjawab GET /api/v1/documents/{documentId}/changes.
func (a *apiServer) ListDocumentChanges(ctx context.Context, req ListDocumentChangesRequestObject) (ListDocumentChangesResponseObject, error) {
	list, err := a.changeRequests.ForDocument(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return ListDocumentChanges200JSONResponse(toAPIChangeList(list)), nil
}

func toAPIChangeList(list []change.Change) ChangeList {
	out := ChangeList{Items: make([]Change, 0, len(list))}
	for _, c := range list {
		out.Items = append(out.Items, toAPIChange(c))
	}
	return out
}

// toAPIChange mengubah perubahan domain menjadi bentuk kontrak API.
func toAPIChange(c change.Change) Change {
	ref := func(r change.Ref) ChangeRef { return ChangeRef{Id: r.ID, Key: optional(r.Key), Name: r.Name} }
	out := Change{
		Id: c.ID, Ref: c.Ref, CustomerId: c.CustomerID, CustomerCode: c.CustomerCode, Mechanism: ChangeMechanism(c.Mechanism),
		ChangeType: ref(c.ChangeType), Reason: c.Reason, SourceReference: optional(c.SourceReference), Status: ChangeStatus(c.Status),
		RequestedBy: c.RequestedBy, Documents: make([]AffectedDocument, 0, len(c.Documents)), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		Attachments: make([]ChangeAttachment, 0, len(c.Attachments)),
		Impacts:     make([]ChangeImpact, 0, len(c.Impacts)),
	}
	for _, x := range c.Impacts {
		out.Impacts = append(out.Impacts, ChangeImpact{
			Id: x.ID, Kind: ChangeImpactKind(x.Kind), DocumentReference: optional(x.DocumentReference), ReferencedDocumentId: optional(x.ReferencedDocumentID),
			Decision: ChangeImpactDecision(x.Decision),
			Reason:   optional(x.Reason), UpdatedBy: x.UpdatedBy, UpdatedAt: x.UpdatedAt,
		})
	}
	for _, x := range c.Attachments {
		out.Attachments = append(out.Attachments, ChangeAttachment{
			Id: x.ID, FileName: x.FileName, ContentType: x.ContentType, Size: x.Size, Sha256: x.SHA256, UploadedBy: x.UploadedBy, UploadedAt: x.UploadedAt,
		})
	}
	if c.Source != nil {
		out.SourceChange = &ChangeRef{Id: c.Source.ID, Name: c.Source.Name}
	}
	if c.EffectiveDate != nil {
		out.EffectiveDate = &openapi_types.Date{Time: *c.EffectiveDate}
	}
	if tpl := c.Template; tpl != nil {
		out.Template = &ChangeTemplate{VersionId: tpl.VersionID, Scope: ChangeTemplateScope(tpl.Scope), ConflictPolicy: ChangeTemplateConflictPolicy(tpl.ConflictPolicy)}
		if tpl.EffectiveDate != nil {
			out.Template.EffectiveDate = &openapi_types.Date{Time: *tpl.EffectiveDate}
		}
	}
	if v := c.Verification; v != nil {
		out.Verification = &ChangeVerification{VerifiedBy: v.VerifiedBy, VerifiedAt: v.VerifiedAt, Note: v.Note}
	}
	if c.Trigger != nil {
		t := ref(*c.Trigger)
		out.Trigger = &t
	}
	if c.FourM != nil {
		f := ref(*c.FourM)
		out.FourM = &f
	}
	for _, d := range c.Documents {
		item := AffectedDocument{Id: d.ID, Number: d.Number, DocumentType: ref(d.DocumentType), Revision: d.Revision, Status: d.Status,
			ReleasedRevision: d.ReleasedRevision, DraftRevision: d.DraftRevision,
			RowsUpdated: d.RowsUpdated, RowsConflict: d.RowsConflict, RowsRemoved: d.RowsRemoved}
		if d.DraftRevision != nil {
			item.Merged = &d.Merged
		}
		out.Documents = append(out.Documents, item)
	}
	return out
}

// AddChangeAttachment menjawab POST /api/v1/changes/{changeId}/attachments (M5.2).
// Bagian multipart bernama "file" dibaca langsung ke penyimpanan.
func (a *apiServer) AddChangeAttachment(ctx context.Context, req AddChangeAttachmentRequestObject) (AddChangeAttachmentResponseObject, error) {
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.InvalidFormat, Message: "Send the file as multipart form data."}}}
		}
		if part.FormName() != "file" {
			continue
		}
		c, err := a.changeRequests.AddAttachment(ctx, actorFrom(ctx), req.ChangeId, part.FileName(), part)
		if err != nil {
			return nil, err
		}
		return AddChangeAttachment201JSONResponse(toAPIChange(c)), nil
	}
	return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.Required, Message: "Choose a file."}}}
}

// GetChangeAttachment menjawab GET /api/v1/changes/{changeId}/attachments/{attachmentId}.
// Berkas selalu dikirim sebagai unduhan, tidak pernah dibuka di browser.
func (a *apiServer) GetChangeAttachment(ctx context.Context, req GetChangeAttachmentRequestObject) (GetChangeAttachmentResponseObject, error) {
	att, f, err := a.changeRequests.OpenAttachment(ctx, actorFrom(ctx), req.ChangeId, req.AttachmentId)
	if err != nil {
		return nil, err
	}
	// Isi dibaca dulu (paling besar 20 MB), supaya berkas bisa langsung ditutup.
	content, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(att.FileName)
	return GetChangeAttachment200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(content), ContentLength: int64(len(content)),
		Headers: GetChangeAttachment200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// RemoveChangeAttachment menjawab DELETE /api/v1/changes/{changeId}/attachments/{attachmentId}.
func (a *apiServer) RemoveChangeAttachment(ctx context.Context, req RemoveChangeAttachmentRequestObject) (RemoveChangeAttachmentResponseObject, error) {
	c, err := a.changeRequests.RemoveAttachment(ctx, actorFrom(ctx), req.ChangeId, req.AttachmentId)
	if err != nil {
		return nil, err
	}
	return RemoveChangeAttachment200JSONResponse(toAPIChange(c)), nil
}

// VerifyChange menjawab POST /api/v1/changes/{changeId}/verification (M5.3).
func (a *apiServer) VerifyChange(ctx context.Context, req VerifyChangeRequestObject) (VerifyChangeResponseObject, error) {
	if !req.Body.Decision.Valid() {
		return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "decision", Code: problem.InvalidFormat, Message: "Choose accept or reject."}}}
	}
	c, err := a.changeRequests.Verify(ctx, actorFrom(ctx), req.ChangeId, req.Body.Decision == ChangeVerificationInputDecisionAccept, req.Body.Note)
	if err != nil {
		return nil, err
	}
	return VerifyChange200JSONResponse(toAPIChange(c)), nil
}

// SaveChangeImpacts menjawab PUT /api/v1/changes/{changeId}/impacts (M5.7).
func (a *apiServer) SaveChangeImpacts(ctx context.Context, req SaveChangeImpactsRequestObject) (SaveChangeImpactsResponseObject, error) {
	items := make([]change.ImpactInput, 0, len(req.Body.Items))
	for _, x := range req.Body.Items {
		items = append(items, change.ImpactInput{
			Kind: x.Kind, ReferencedDocumentID: deref(x.ReferencedDocumentId), DocumentReference: deref(x.DocumentReference), Decision: x.Decision, Reason: deref(x.Reason),
		})
	}
	c, err := a.changeRequests.SaveImpacts(ctx, actorFrom(ctx), req.ChangeId, items)
	if err != nil {
		return nil, err
	}
	return SaveChangeImpacts200JSONResponse(toAPIChange(c)), nil
}

// PreviewPropagation menjawab GET /api/v1/process-template-versions/{id}/propagation (M5.8).
func (a *apiServer) PreviewPropagation(ctx context.Context, req PreviewPropagationRequestObject) (PreviewPropagationResponseObject, error) {
	items, err := a.changeRequests.Preview(ctx, actorFrom(ctx), req.ProcessVersionId, req.Params.Scope)
	if err != nil {
		return nil, err
	}
	out := PropagationPreview{Items: make([]PropagationItem, 0, len(items))}
	for _, it := range items {
		item := PropagationItem{
			DocumentId: it.DocumentID, Number: it.Number, CustomerId: it.CustomerID, Status: it.Status, Mode: PropagationItemMode(it.Mode),
			Included: it.Included, Updates: it.Updates, Conflicts: it.Conflicts, Removed: it.Removed, Rows: make([]PropagationRow, 0, len(it.Rows)),
		}
		if it.Skipped != "" {
			skipped := PropagationItemSkipped(it.Skipped)
			item.Skipped = &skipped
		}
		for _, r := range it.Rows {
			row := PropagationRow{RowId: r.RowID, Position: r.Position, Status: PropagationRowStatus(r.Status), PointText: r.PointText, Fields: make([]FieldDifference, 0, len(r.Fields))}
			for _, f := range r.Fields {
				row.Fields = append(row.Fields, FieldDifference{FieldId: optional(f.FieldID), Label: f.Label, Template: f.Template, Row: f.Row})
			}
			item.Rows = append(item.Rows, row)
		}
		out.Items = append(out.Items, item)
	}
	return PreviewPropagation200JSONResponse(out), nil
}

// PropagateTemplate menjawab POST /api/v1/process-template-versions/{id}/propagation (M5.8).
func (a *apiServer) PropagateTemplate(ctx context.Context, req PropagateTemplateRequestObject) (PropagateTemplateResponseObject, error) {
	b := req.Body
	in := change.Propagation{
		VersionID: req.ProcessVersionId, Scope: b.Scope, ConflictPolicy: b.ConflictPolicy, ChangeTypeID: b.ChangeTypeId,
		Reason: b.Reason, SourceReference: deref(b.SourceReference),
	}
	if b.EffectiveDate != nil {
		at := b.EffectiveDate.Time
		in.EffectiveDate = &at
	}
	list, err := a.changeRequests.Propagate(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return PropagateTemplate201JSONResponse(toAPIChangeList(list)), nil
}

// SubmitSQAMChange menjawab POST /api/v1/sqam-changes (M5.10).
func (a *apiServer) SubmitSQAMChange(ctx context.Context, req SubmitSQAMChangeRequestObject) (SubmitSQAMChangeResponseObject, error) {
	b := req.Body
	in := change.SQAMChange{
		CustomerID: b.CustomerId, ChangeTypeID: b.ChangeTypeId, FourMCategoryID: deref(b.FourMCategoryId), Reason: b.Reason,
		SQAMReference: b.SqamReference,
	}
	if b.DocumentTypeIds != nil {
		in.DocumentTypeIDs = *b.DocumentTypeIds
	}
	if b.EffectiveDate != nil {
		date := b.EffectiveDate.Time
		in.EffectiveDate = &date
	}
	c, err := a.changeRequests.SubmitSQAM(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return SubmitSQAMChange201JSONResponse(toAPIChange(c)), nil
}
