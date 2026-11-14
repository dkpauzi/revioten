package server

// Handler perubahan (M5.1 dan seterusnya). Aturannya ada di
// internal/change/request.go; handler hanya menerjemahkan bentuk data.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/change"
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
		out.Documents = append(out.Documents, AffectedDocument{Id: d.ID, Number: d.Number, DocumentType: ref(d.DocumentType), Revision: d.Revision, Status: d.Status})
	}
	return out
}
