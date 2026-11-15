package server

// Handler migrasi dan dasbor (Tahap 7). Aturannya ada di internal/document
// (migrate.go, link.go, excelcompare.go) dan internal/metrics.

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/metrics"
	"github.com/dkpauzi/revioten/internal/problem"
)

// MigrationService adalah operasi migrasi dokumen lama (Tahap 7).
type MigrationService interface {
	MigratableNumbers(ctx context.Context, actor access.Actor, customerID string) ([]document.MigratableNumber, error)
	Migrate(ctx context.Context, actor access.Actor, in document.LegacyDocument) (document.Document, error)
	SetHistory(ctx context.Context, actor access.Actor, id string, entries []document.HistoryEntry) (document.Document, error)
	VerifyMigration(ctx context.Context, actor access.Actor, id string, effortMinutes *int) (document.Document, error)
	LinkSuggestions(ctx context.Context, actor access.Actor, id string) ([]document.LinkSuggestion, error)
	LinkRows(ctx context.Context, actor access.Actor, id string, links []document.RowLink) (document.Document, error)
	CompareExcel(ctx context.Context, actor access.Actor, id, sheet string, r io.Reader) (document.ExcelComparison, error)
}

// MetricsService menghitung ukuran dasbor (M7.7).
type MetricsService interface {
	Get(ctx context.Context, actor access.Actor, customerID string) (metrics.Metrics, error)
	Performance(ctx context.Context, actor access.Actor, kind string, date time.Time, customerID string) (metrics.Performance, error)
}

// ListMigratableNumbers menjawab GET /api/v1/migration/numbers.
func (a *apiServer) ListMigratableNumbers(ctx context.Context, req ListMigratableNumbersRequestObject) (ListMigratableNumbersResponseObject, error) {
	list, err := a.migration.MigratableNumbers(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	out := MigratableNumberList{Items: make([]MigratableNumber, 0, len(list))}
	for _, n := range list {
		out.Items = append(out.Items, MigratableNumber{Id: n.ID, Number: n.Number, Status: n.Status, DocumentType: n.DocumentType})
	}
	return ListMigratableNumbers200JSONResponse(out), nil
}

// MigrateDocument menjawab POST /api/v1/migration/documents.
func (a *apiServer) MigrateDocument(ctx context.Context, req MigrateDocumentRequestObject) (MigrateDocumentResponseObject, error) {
	b := req.Body
	in := document.LegacyDocument{
		NewDocument: document.NewDocument{NumberID: b.DocumentNumberId, ModelIDs: b.ModelIds, RouteID: deref(b.RouteId), RouteReason: deref(b.RouteReason)},
		Revision:    b.Revision, EffectiveDate: b.EffectiveDate, PreparedBy: b.PreparedBy, ApprovedBy: b.ApprovedBy,
	}
	if b.PartNumberIds != nil {
		in.PartNumberIDs = *b.PartNumberIds
	}
	d, err := a.migration.Migrate(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return MigrateDocument201JSONResponse(toAPIDocument(d)), nil
}

// SetDocumentHistory menjawab PUT /api/v1/documents/{documentId}/history.
func (a *apiServer) SetDocumentHistory(ctx context.Context, req SetDocumentHistoryRequestObject) (SetDocumentHistoryResponseObject, error) {
	entries := make([]document.HistoryEntry, 0, len(req.Body.Entries))
	for _, e := range req.Body.Entries {
		entries = append(entries, document.HistoryEntry{Revision: e.Revision, Date: deref(e.Date), Description: e.Description})
	}
	d, err := a.migration.SetHistory(ctx, actorFrom(ctx), req.DocumentId, entries)
	if err != nil {
		return nil, err
	}
	return SetDocumentHistory200JSONResponse(toAPIDocument(d)), nil
}

// VerifyMigration menjawab POST /api/v1/documents/{documentId}/migration/verify.
func (a *apiServer) VerifyMigration(ctx context.Context, req VerifyMigrationRequestObject) (VerifyMigrationResponseObject, error) {
	d, err := a.migration.VerifyMigration(ctx, actorFrom(ctx), req.DocumentId, req.Body.EffortMinutes)
	if err != nil {
		return nil, err
	}
	return VerifyMigration200JSONResponse(toAPIDocument(d)), nil
}

// ListLinkSuggestions menjawab GET /api/v1/documents/{documentId}/link-suggestions.
func (a *apiServer) ListLinkSuggestions(ctx context.Context, req ListLinkSuggestionsRequestObject) (ListLinkSuggestionsResponseObject, error) {
	list, err := a.migration.LinkSuggestions(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	out := LinkSuggestionList{Items: make([]LinkSuggestion, 0, len(list))}
	for _, s := range list {
		out.Items = append(out.Items, LinkSuggestion{
			RowId: s.RowID, Position: s.Position, RowText: s.RowText, PointId: s.PointID, VersionId: s.VersionID,
			PointText: s.PointText, Matched: s.Matched, Score: s.Score,
		})
	}
	return ListLinkSuggestions200JSONResponse(out), nil
}

// LinkDocumentRows menjawab POST /api/v1/documents/{documentId}/links.
func (a *apiServer) LinkDocumentRows(ctx context.Context, req LinkDocumentRowsRequestObject) (LinkDocumentRowsResponseObject, error) {
	links := make([]document.RowLink, 0, len(req.Body.Links))
	for _, l := range req.Body.Links {
		links = append(links, document.RowLink{RowID: l.RowId, PointID: l.PointId, VersionID: l.VersionId})
	}
	d, err := a.migration.LinkRows(ctx, actorFrom(ctx), req.DocumentId, links)
	if err != nil {
		return nil, err
	}
	return LinkDocumentRows200JSONResponse(toAPIDocument(d)), nil
}

// CompareDocumentWithExcel menjawab POST /api/v1/documents/{documentId}/excel-comparison.
// Bagian "sheet" boleh dikirim sebelum "file".
func (a *apiServer) CompareDocumentWithExcel(ctx context.Context, req CompareDocumentWithExcelRequestObject) (CompareDocumentWithExcelResponseObject, error) {
	sheet := ""
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.InvalidFormat, Message: "Send the file as multipart form data."}}}
		}
		switch part.FormName() {
		case "sheet":
			name, err := io.ReadAll(io.LimitReader(part, 200))
			if err != nil {
				return nil, err
			}
			sheet = string(name)
		case "file":
			c, err := a.migration.CompareExcel(ctx, actorFrom(ctx), req.DocumentId, sheet, part)
			if err != nil {
				return nil, err
			}
			out := ExcelComparison{Sheets: c.Sheets, Sheet: c.Sheet, Matched: c.Matched, Truncated: c.Truncated}
			out.MissingInExcel = make([]struct {
				Label string `json:"label"`

				// Position The row order; 0 for the header.
				Position int    `json:"position"`
				Text     string `json:"text"`
			}, len(c.MissingInExcel))
			for i, m := range c.MissingInExcel {
				out.MissingInExcel[i].Label, out.MissingInExcel[i].Position, out.MissingInExcel[i].Text = m.Label, m.Position, m.Text
			}
			out.MissingInDocument = make([]struct {
				Cell string `json:"cell"`
				Text string `json:"text"`
			}, len(c.MissingInDocument))
			for i, m := range c.MissingInDocument {
				out.MissingInDocument[i].Cell, out.MissingInDocument[i].Text = m.Cell, m.Text
			}
			return CompareDocumentWithExcel200JSONResponse(out), nil
		}
	}
	return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.Required, Message: "Choose the legacy Excel file."}}}
}

// GetMetrics menjawab GET /api/v1/metrics.
func (a *apiServer) GetMetrics(ctx context.Context, req GetMetricsRequestObject) (GetMetricsResponseObject, error) {
	m, err := a.metrics.Get(ctx, actorFrom(ctx), deref(req.Params.CustomerId))
	if err != nil {
		return nil, err
	}
	var out Metrics
	out.Rows.Total, out.Rows.Linked, out.Rows.Detached, out.Rows.Free = m.Rows.Total, m.Rows.Linked, m.Rows.Detached, m.Rows.Free
	out.Rows.LinkedPercent = float32(m.Rows.LinkedPercent)
	out.Documents = make([]struct {
		Count            int                    `json:"count"`
		DocumentType     string                 `json:"document_type"`
		DocumentTypeName string                 `json:"document_type_name"`
		Status           MetricsDocumentsStatus `json:"status"`
	}, len(m.Documents))
	for i, d := range m.Documents {
		out.Documents[i].Count, out.Documents[i].DocumentType, out.Documents[i].DocumentTypeName = d.Count, d.TypeKey, d.TypeName
		out.Documents[i].Status = MetricsDocumentsStatus(d.Status)
	}
	out.Yokoten = make([]struct {
		Count    int                    `json:"count"`
		Decision MetricsYokotenDecision `json:"decision"`
	}, len(m.Yokoten))
	for i, y := range m.Yokoten {
		out.Yokoten[i].Count, out.Yokoten[i].Decision = y.Count, MetricsYokotenDecision(y.Decision)
	}
	out.Revision.Count, out.Revision.AverageHours, out.Revision.MedianHours = m.Revision.Count, float32(m.Revision.AverageHours), float32(m.Revision.MedianHours)
	out.Migration.Documents, out.Migration.Verified, out.Migration.Measured = m.Migration.Documents, m.Migration.Verified, m.Migration.Measured
	out.Migration.TotalMinutes, out.Migration.AverageMinutes = m.Migration.TotalMinutes, float32(m.Migration.AverageMinutes)
	return GetMetrics200JSONResponse(out), nil
}

// toAPIBaseline mengubah baseline migrasi menjadi bentuk kontrak; nil untuk dokumen baru.
func toAPIBaseline(b *document.Baseline) *DocumentBaseline {
	if b == nil {
		return nil
	}
	out := &DocumentBaseline{
		Revision: b.Revision, EffectiveDate: b.EffectiveDate, PreparedBy: b.PreparedBy, ApprovedBy: b.ApprovedBy,
		CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt, VerifiedBy: optional(b.VerifiedBy), VerifiedAt: b.VerifiedAt,
		EffortMinutes: b.EffortMinutes, History: make([]HistoryEntry, 0, len(b.History)),
	}
	for _, h := range b.History {
		out.History = append(out.History, HistoryEntry{Revision: h.Revision, Date: optional(h.Date), Description: h.Description})
	}
	return out
}
