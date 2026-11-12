package server

// Handler dokumen (M4.1). Aturannya ada di internal/document; handler hanya
// menerjemahkan bentuk data.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/document"
)

// ListDocuments menjawab GET /api/v1/documents?customer_id=.
func (a *apiServer) ListDocuments(ctx context.Context, req ListDocumentsRequestObject) (ListDocumentsResponseObject, error) {
	list, err := a.documents.List(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	out := DocumentList{Items: make([]DocumentSummary, 0, len(list))}
	for _, d := range list {
		out.Items = append(out.Items, DocumentSummary{
			Id: d.ID, Number: d.Number, DocumentType: DocumentTypeRef{Id: d.DocumentType.ID, Key: DocumentTypeRefKey(d.DocumentType.Key), Name: d.DocumentType.Name},
			Revision: d.Revision, Status: DocumentSummaryStatus(d.Status), FormSetName: d.FormSetName, RowCount: d.RowCount, UpdatedAt: d.UpdatedAt,
		})
	}
	return ListDocuments200JSONResponse(out), nil
}

// CreateDocument menjawab POST /api/v1/documents.
func (a *apiServer) CreateDocument(ctx context.Context, req CreateDocumentRequestObject) (CreateDocumentResponseObject, error) {
	b := req.Body
	var parts []string
	if b.PartNumberIds != nil {
		parts = *b.PartNumberIds
	}
	d, err := a.documents.Create(ctx, actorFrom(ctx), document.NewDocument{
		NumberID: b.DocumentNumberId, ModelIDs: b.ModelIds, PartNumberIDs: parts,
		RouteID: deref(b.RouteId), RouteReason: deref(b.RouteReason),
	})
	if err != nil {
		return nil, err
	}
	return CreateDocument201JSONResponse(toAPIDocument(d)), nil
}

// GetDocument menjawab GET /api/v1/documents/{documentId}.
func (a *apiServer) GetDocument(ctx context.Context, req GetDocumentRequestObject) (GetDocumentResponseObject, error) {
	d, err := a.documents.Get(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return GetDocument200JSONResponse(toAPIDocument(d)), nil
}

// SetDocumentHeader menjawab PUT /api/v1/documents/{documentId}/header.
func (a *apiServer) SetDocumentHeader(ctx context.Context, req SetDocumentHeaderRequestObject) (SetDocumentHeaderResponseObject, error) {
	d, err := a.documents.SetHeader(ctx, actorFrom(ctx), req.DocumentId, req.Body.Header, req.Body.HeaderVersion)
	if err != nil {
		return nil, err
	}
	return SetDocumentHeader200JSONResponse(toAPIDocument(d)), nil
}

// AddDocumentRow menjawab POST /api/v1/documents/{documentId}/rows.
func (a *apiServer) AddDocumentRow(ctx context.Context, req AddDocumentRowRequestObject) (AddDocumentRowResponseObject, error) {
	b := req.Body
	d, err := a.documents.AddRow(ctx, actorFrom(ctx), req.DocumentId, document.NewRow{
		ProcessID: deref(b.ProcessId), OperationNumber: deref(b.OperationNumber), Content: b.Content,
	})
	if err != nil {
		return nil, err
	}
	return AddDocumentRow201JSONResponse(toAPIDocument(d)), nil
}

// AddDocumentRows menjawab POST /api/v1/documents/{documentId}/rows/batch (M4.4).
func (a *apiServer) AddDocumentRows(ctx context.Context, req AddDocumentRowsRequestObject) (AddDocumentRowsResponseObject, error) {
	rows := make([]document.NewRow, 0, len(req.Body.Rows))
	for _, b := range req.Body.Rows {
		rows = append(rows, document.NewRow{ProcessID: deref(b.ProcessId), OperationNumber: deref(b.OperationNumber), Content: b.Content})
	}
	d, err := a.documents.AddRows(ctx, actorFrom(ctx), req.DocumentId, rows)
	if err != nil {
		return nil, err
	}
	return AddDocumentRows201JSONResponse(toAPIDocument(d)), nil
}

// UpdateDocumentRow menjawab PUT /api/v1/documents/{documentId}/rows/{rowId}.
func (a *apiServer) UpdateDocumentRow(ctx context.Context, req UpdateDocumentRowRequestObject) (UpdateDocumentRowResponseObject, error) {
	b := req.Body
	// Versi yang tidak dikirim menjadi 0, dan layanan menolaknya (version:required).
	version := 0
	if b.Version != nil {
		version = *b.Version
	}
	d, err := a.documents.UpdateRow(ctx, actorFrom(ctx), req.DocumentId, req.RowId, document.RowUpdate{
		ProcessID: deref(b.ProcessId), OperationNumber: deref(b.OperationNumber), Content: b.Content, Version: version,
	})
	if err != nil {
		return nil, err
	}
	return UpdateDocumentRow200JSONResponse(toAPIDocument(d)), nil
}

// DeleteDocumentRow menjawab DELETE /api/v1/documents/{documentId}/rows/{rowId} (M4.2).
func (a *apiServer) DeleteDocumentRow(ctx context.Context, req DeleteDocumentRowRequestObject) (DeleteDocumentRowResponseObject, error) {
	d, err := a.documents.DeleteRow(ctx, actorFrom(ctx), req.DocumentId, req.RowId)
	if err != nil {
		return nil, err
	}
	return DeleteDocumentRow200JSONResponse(toAPIDocument(d)), nil
}

// toAPIDocument mengubah Document domain menjadi Document kontrak API.
func toAPIDocument(d document.Document) Document {
	out := Document{
		Id: d.ID, DocumentNumberId: d.NumberID, Number: d.Number, CustomerId: d.CustomerID, CustomerCode: d.CustomerCode,
		DocumentType: DocumentTypeRef{Id: d.DocumentType.ID, Key: DocumentTypeRefKey(d.DocumentType.Key), Name: d.DocumentType.Name},
		Revision:     d.Revision, Status: DocumentStatus(d.Status), FormSetVersionId: d.FormSetVersionID, FormSetName: d.FormSetName,
		FormatVersionId: d.FormatVersionID, Header: d.Header, HeaderVersion: d.HeaderVersion, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		Models: make([]NamedRef, 0, len(d.Models)), PartNumbers: make([]DocumentPartNumber, 0, len(d.PartNumbers)),
		Rows: make([]DocumentRow, 0, len(d.Rows)), RouteReason: optional(d.RouteReason),
		Set: make([]DocumentSetMember, 0, len(d.Set)),
	}
	for _, m := range d.Set {
		out.Set = append(out.Set, DocumentSetMember{Id: m.ID, Number: m.Number, DocumentType: DocumentTypeRef{Id: m.DocumentType.ID, Key: DocumentTypeRefKey(m.DocumentType.Key), Name: m.DocumentType.Name}})
	}
	if out.Header == nil {
		out.Header = map[string]any{}
	}
	if d.Route != nil {
		out.Route = &NamedRef{Id: d.Route.ID, Name: d.Route.Name}
	}
	for _, m := range d.Models {
		out.Models = append(out.Models, NamedRef{Id: m.ID, Name: m.Name})
	}
	for _, p := range d.PartNumbers {
		out.PartNumbers = append(out.PartNumbers, DocumentPartNumber{Id: p.ID, Number: p.Number, Name: optional(p.Name)})
	}
	for _, r := range d.Rows {
		content := r.Content
		if content == nil {
			content = map[string]any{}
		}
		out.Rows = append(out.Rows, DocumentRow{
			Id: r.ID, Position: r.Position, Version: r.Version, ProcessId: optional(r.ProcessID), ProcessName: optional(r.ProcessName),
			OperationNumber: optional(r.OperationNumber), Content: content, TemplatePointId: optional(r.TemplatePointID),
		})
	}
	return out
}

// GetDocumentComposition menjawab GET /api/v1/documents/{documentId}/composition (M4.5).
func (a *apiServer) GetDocumentComposition(ctx context.Context, req GetDocumentCompositionRequestObject) (GetDocumentCompositionResponseObject, error) {
	c, err := a.documents.Composition(ctx, actorFrom(ctx), req.DocumentId, req.Params.ProcessTemplateVersionId)
	if err != nil {
		return nil, err
	}
	out := Composition{
		ProcessTemplateVersionId: c.VersionID, Version: c.Version, Process: NamedRef{Id: c.Process.ID, Name: c.Process.Name},
		Subprocesses: make([]ComposedSubprocess, 0, len(c.Subprocesses)),
	}
	for _, s := range c.Subprocesses {
		sub := ComposedSubprocess{Id: s.ID, Name: s.Name, Points: make([]ComposedPoint, 0, len(s.Points))}
		for _, p := range s.Points {
			sub.Points = append(sub.Points, ComposedPoint{
				PointId: p.PointID, Text: p.Text, DocumentId: p.DocumentID, State: ComposedPointState(p.State), Reason: optional(p.Reason),
				DocumentType: DocumentTypeRef{Id: p.DocumentType.ID, Key: DocumentTypeRefKey(p.DocumentType.Key), Name: p.DocumentType.Name},
			})
		}
		out.Subprocesses = append(out.Subprocesses, sub)
	}
	return GetDocumentComposition200JSONResponse(out), nil
}

// ComposeDocument menjawab PUT /api/v1/documents/{documentId}/composition (M4.5).
func (a *apiServer) ComposeDocument(ctx context.Context, req ComposeDocumentRequestObject) (ComposeDocumentResponseObject, error) {
	b := req.Body
	in := document.Compose{VersionID: b.ProcessTemplateVersionId}
	if b.Include != nil {
		in.Include = *b.Include
	}
	if b.Exclude != nil {
		for _, e := range *b.Exclude {
			in.Exclude = append(in.Exclude, document.Exclusion{PointID: e.PointId, Reason: deref(e.Reason)})
		}
	}
	d, err := a.documents.Compose(ctx, actorFrom(ctx), req.DocumentId, in)
	if err != nil {
		return nil, err
	}
	return ComposeDocument200JSONResponse(toAPIDocument(d)), nil
}
