package server

// Handler master dokumen terkait (D-130, D-156). Aturannya ada di
// internal/template/refdoc.go; handler hanya menerjemahkan bentuk data.

import (
	"context"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dkpauzi/revioten/internal/template"
)

// ListReferencedDocuments menjawab GET /api/v1/referenced-documents.
func (a *apiServer) ListReferencedDocuments(ctx context.Context, _ ListReferencedDocumentsRequestObject) (ListReferencedDocumentsResponseObject, error) {
	list, err := a.refDocs.List(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := ReferencedDocumentList{Items: make([]ReferencedDocument, 0, len(list))}
	for _, d := range list {
		out.Items = append(out.Items, toAPIRefDocument(d))
	}
	return ListReferencedDocuments200JSONResponse(out), nil
}

// CreateReferencedDocument menjawab POST /api/v1/referenced-documents.
func (a *apiServer) CreateReferencedDocument(ctx context.Context, req CreateReferencedDocumentRequestObject) (CreateReferencedDocumentResponseObject, error) {
	b := req.Body
	d, err := a.refDocs.Create(ctx, actorFrom(ctx), template.NewRefDocument{
		Number: b.Number, Title: b.Title, Level: deref(b.Level), Revision: deref(b.Revision), RevisionDate: deref(b.RevisionDate),
	})
	if err != nil {
		return nil, err
	}
	return CreateReferencedDocument201JSONResponse(toAPIRefDocument(d)), nil
}

// UpdateReferencedDocument menjawab PUT /api/v1/referenced-documents/{refDocId}.
func (a *apiServer) UpdateReferencedDocument(ctx context.Context, req UpdateReferencedDocumentRequestObject) (UpdateReferencedDocumentResponseObject, error) {
	b := req.Body
	d, err := a.refDocs.Update(ctx, actorFrom(ctx), req.RefDocId, template.RefDocumentUpdate{
		Title: b.Title, Level: deref(b.Level), Revision: deref(b.Revision), RevisionDate: deref(b.RevisionDate), Active: b.Active,
	})
	if err != nil {
		return nil, err
	}
	return UpdateReferencedDocument200JSONResponse(toAPIRefDocument(d)), nil
}

// AddReferencedDocumentLink menjawab POST /api/v1/referenced-documents/{refDocId}/links.
func (a *apiServer) AddReferencedDocumentLink(ctx context.Context, req AddReferencedDocumentLinkRequestObject) (AddReferencedDocumentLinkResponseObject, error) {
	b := req.Body
	d, err := a.refDocs.AddLink(ctx, actorFrom(ctx), req.RefDocId, template.NewRefLink{
		ProcessID: b.ProcessId, Scope: template.ScopeInput{Level: deref(b.ScopeLevel), TargetID: deref(b.ScopeTargetId)},
	})
	if err != nil {
		return nil, err
	}
	return AddReferencedDocumentLink201JSONResponse(toAPIRefDocument(d)), nil
}

// RemoveReferencedDocumentLink menjawab DELETE /api/v1/referenced-documents/{refDocId}/links/{linkId}.
func (a *apiServer) RemoveReferencedDocumentLink(ctx context.Context, req RemoveReferencedDocumentLinkRequestObject) (RemoveReferencedDocumentLinkResponseObject, error) {
	d, err := a.refDocs.RemoveLink(ctx, actorFrom(ctx), req.RefDocId, req.LinkId)
	if err != nil {
		return nil, err
	}
	return RemoveReferencedDocumentLink200JSONResponse(toAPIRefDocument(d)), nil
}

// FillDocumentReferences menjawab POST /api/v1/documents/{documentId}/references.
func (a *apiServer) FillDocumentReferences(ctx context.Context, req FillDocumentReferencesRequestObject) (FillDocumentReferencesResponseObject, error) {
	d, err := a.documents.FillReferences(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return FillDocumentReferences200JSONResponse(toAPIDocument(d)), nil
}

// toAPIRefDocument mengubah dokumen terkait domain menjadi bentuk kontrak API.
func toAPIRefDocument(d template.RefDocument) ReferencedDocument {
	out := ReferencedDocument{
		Id: d.ID, Number: d.Number, Title: d.Title, Level: optional(d.Level), Revision: optional(d.Revision),
		Active: d.Active, CreatedAt: d.CreatedAt, Links: make([]ReferencedDocumentLink, 0, len(d.Links)),
	}
	if t, err := time.Parse(time.DateOnly, d.RevisionDate); err == nil {
		out.RevisionDate = &openapi_types.Date{Time: t}
	}
	for _, l := range d.Links {
		out.Links = append(out.Links, ReferencedDocumentLink{
			Id: l.ID, Process: ProcessRef{Id: l.Process.ID, Name: l.Process.Name}, Scope: toAPIScope(l.Scope),
		})
	}
	return out
}
