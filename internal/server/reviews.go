package server

// Handler dokumen terbit untuk pembaca (D-195), review berkala (D-196), dan
// laporan kinerja (D-197). Aturannya ada di internal/document, internal/review,
// dan internal/metrics; handler hanya menerjemahkan bentuk data.

import (
	"bytes"
	"context"
	"net/url"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/inbox"
	"github.com/dkpauzi/revioten/internal/metrics"
	"github.com/dkpauzi/revioten/internal/review"
)

// ReviewService adalah review berkala yang dipakai server.
type ReviewService interface {
	Sets(ctx context.Context, actor access.Actor, f review.Filter) (review.Overview, error)
	Record(ctx context.Context, actor access.Actor, in review.NewReview) (review.Review, error)
	Confirm(ctx context.Context, actor access.Actor, id string) (review.Review, error)
	Reviews(ctx context.Context, actor access.Actor, customerID string, openOnly bool) ([]review.Review, error)
	Export(ctx context.Context, actor access.Actor, f review.Filter) ([]byte, string, error)
}

// InboxService adalah daftar "Waiting for you" di Home (D-202).
type InboxService interface {
	Tasks(ctx context.Context, actor access.Actor) ([]inbox.Task, int, error)
}

// ListReleasedDocuments menjawab GET /api/v1/released-documents.
func (a *apiServer) ListReleasedDocuments(ctx context.Context, req ListReleasedDocumentsRequestObject) (ListReleasedDocumentsResponseObject, error) {
	list, err := a.documents.ListReleased(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	out := DocumentList{Items: make([]DocumentSummary, 0, len(list))}
	for _, d := range list {
		out.Items = append(out.Items, DocumentSummary{
			Id: d.ID, Number: d.Number, DocumentType: DocumentTypeRef{Id: d.DocumentType.ID, Key: DocumentTypeRefKey(d.DocumentType.Key), Name: d.DocumentType.Name},
			Revision: d.Revision, Status: DocumentSummaryStatus(d.Status), FormSetName: d.FormSetName, RowCount: d.RowCount, UpdatedAt: d.UpdatedAt, Models: orEmpty(d.Models),
			Origin: DocumentSummaryOrigin(orNew(d.Origin)), ReleasedMigrated: d.ReleasedMigrated,
		})
	}
	return ListReleasedDocuments200JSONResponse(out), nil
}

// GetReleasedDocument menjawab GET /api/v1/released-documents/{documentId}.
func (a *apiServer) GetReleasedDocument(ctx context.Context, req GetReleasedDocumentRequestObject) (GetReleasedDocumentResponseObject, error) {
	d, err := a.documents.Released(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return GetReleasedDocument200JSONResponse(toAPIDocument(d)), nil
}

// GetReleasedPfmeaCpComparison menjawab GET /api/v1/released-documents/{documentId}/pfmea-cp.
func (a *apiServer) GetReleasedPfmeaCpComparison(ctx context.Context, req GetReleasedPfmeaCpComparisonRequestObject) (GetReleasedPfmeaCpComparisonResponseObject, error) {
	c, err := a.documents.ReleasedComparison(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return GetReleasedPfmeaCpComparison200JSONResponse(toAPIComparison(c)), nil
}

// ListReviewSets menjawab GET /api/v1/periodic-reviews/sets.
func (a *apiServer) ListReviewSets(ctx context.Context, req ListReviewSetsRequestObject) (ListReviewSetsResponseObject, error) {
	p := req.Params
	o, err := a.reviews.Sets(ctx, actorFrom(ctx), review.Filter{CustomerID: deref(p.CustomerId), DocumentID: deref(p.DocumentId), All: isTrue(p.All), EOL: isTrue(p.Eol)})
	if err != nil {
		return nil, err
	}
	return ListReviewSets200JSONResponse(toAPIReviewSets(o)), nil
}

// ExportReviewSets menjawab GET /api/v1/periodic-reviews/sets/export (D-202).
func (a *apiServer) ExportReviewSets(ctx context.Context, req ExportReviewSetsRequestObject) (ExportReviewSetsResponseObject, error) {
	p := req.Params
	body, name, err := a.reviews.Export(ctx, actorFrom(ctx), review.Filter{CustomerID: deref(p.CustomerId), All: isTrue(p.All), EOL: isTrue(p.Eol)})
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(name)
	return ExportReviewSets200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(body), ContentLength: int64(len(body)),
		Headers: ExportReviewSets200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

func isTrue(b *bool) bool { return b != nil && *b }

func toAPIReviewSets(o review.Overview) ReviewSetList {
	out := ReviewSetList{PeriodMonths: o.PeriodMonths, WarningDays: o.WarningDays, DueSoon: o.DueSoon, Overdue: o.Overdue, DueYear: o.DueYear,
		ReviewedNoChange: o.ReviewedNoChange, ReviewedRevised: o.ReviewedRevised, Items: make([]ReviewSet, 0, len(o.Sets))}
	for _, s := range o.Sets {
		item := ReviewSet{
			CustomerId: s.CustomerID, CustomerCode: s.CustomerCode, Sequence: s.Sequence, Models: orEmpty(s.Models),
			LastRelease: date(s.LastRelease), DueOn: date(s.DueOn), Status: ReviewSetStatus(s.Status), DaysLeft: s.DaysLeft,
			OpenReviewId: optional(s.OpenReviewID), DraftOpen: s.DraftOpen, Eol: s.EOL, Engineer: s.Engineer,
			Documents: make([]ReviewSetDocument, 0, len(s.Documents)),
		}
		if s.LastReview != nil {
			d := date(*s.LastReview)
			item.LastReview = &d
		}
		for _, d := range s.Documents {
			item.Documents = append(item.Documents, ReviewSetDocument{Id: d.ID, Number: d.Number, DocumentType: d.TypeKey, Revision: d.Revision,
				LastRelease: date(d.LastRelease), DueOn: date(d.DueOn), Overdue: d.Overdue})
		}
		out.Items = append(out.Items, item)
	}
	return out
}

// ListPeriodicReviews menjawab GET /api/v1/periodic-reviews.
func (a *apiServer) ListPeriodicReviews(ctx context.Context, req ListPeriodicReviewsRequestObject) (ListPeriodicReviewsResponseObject, error) {
	list, err := a.reviews.Reviews(ctx, actorFrom(ctx), deref(req.Params.CustomerId), req.Params.OpenOnly != nil && *req.Params.OpenOnly)
	if err != nil {
		return nil, err
	}
	out := PeriodicReviewList{Items: make([]PeriodicReview, 0, len(list))}
	for _, r := range list {
		out.Items = append(out.Items, toAPIReview(r))
	}
	return ListPeriodicReviews200JSONResponse(out), nil
}

// RecordPeriodicReview menjawab POST /api/v1/periodic-reviews.
func (a *apiServer) RecordPeriodicReview(ctx context.Context, req RecordPeriodicReviewRequestObject) (RecordPeriodicReviewResponseObject, error) {
	b := req.Body
	in := review.NewReview{DocumentID: b.DocumentId, ReviewedOn: b.ReviewedOn, Participants: b.Participants, Conclusion: b.Conclusion}
	for _, c := range b.Checklist {
		in.Checklist = append(in.Checklist, review.Answer{Item: string(c.Item), Answer: string(c.Answer), Note: deref(c.Note)})
	}
	r, err := a.reviews.Record(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return RecordPeriodicReview201JSONResponse(toAPIReview(r)), nil
}

// ConfirmPeriodicReview menjawab POST /api/v1/periodic-reviews/{reviewId}/confirm.
func (a *apiServer) ConfirmPeriodicReview(ctx context.Context, req ConfirmPeriodicReviewRequestObject) (ConfirmPeriodicReviewResponseObject, error) {
	r, err := a.reviews.Confirm(ctx, actorFrom(ctx), req.ReviewId)
	if err != nil {
		return nil, err
	}
	return ConfirmPeriodicReview200JSONResponse(toAPIReview(r)), nil
}

func toAPIReview(r review.Review) PeriodicReview {
	out := PeriodicReview{
		Id: r.ID, Ref: r.Ref, CustomerId: r.CustomerID, CustomerCode: r.CustomerCode, Sequence: r.Sequence,
		ReviewedOn: date(r.ReviewedOn), Participants: r.Participants, Conclusion: r.Conclusion, Status: PeriodicReviewStatus(r.Status),
		RecordedBy: r.RecordedBy, RecordedAt: r.RecordedAt, ConfirmedBy: optional(r.ConfirmedBy), ConfirmedAt: r.ConfirmedAt,
		Checklist: make([]ReviewAnswer, 0, len(r.Checklist)), Revisions: make([]ReviewedRevision, 0, len(r.Revisions)),
	}
	for _, c := range r.Checklist {
		out.Checklist = append(out.Checklist, ReviewAnswer{Item: ReviewAnswerItem(c.Item), Answer: ReviewAnswerAnswer(c.Answer), Note: optional(c.Note)})
	}
	for _, v := range r.Revisions {
		out.Revisions = append(out.Revisions, ReviewedRevision{DocumentId: v.DocumentID, Number: v.Number, DocumentType: v.TypeKey, Revision: v.Revision, Sha256: v.SHA256})
	}
	return out
}

// performanceDay memberi tanggal laporan: tanggal di permintaan, atau hari ini.
func performanceDay(d *openapi_types.Date) time.Time {
	if d == nil {
		return time.Now()
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, clock.Zone)
}

// GetPerformance menjawab GET /api/v1/performance.
func (a *apiServer) GetPerformance(ctx context.Context, req GetPerformanceRequestObject) (GetPerformanceResponseObject, error) {
	p := req.Params
	r, err := a.metrics.Performance(ctx, actorFrom(ctx), string(p.Period), performanceDay(p.Date), deref(p.CustomerId), deref(p.PersonId))
	if err != nil {
		return nil, err
	}
	return GetPerformance200JSONResponse(toAPIPerformance(r)), nil
}

// ExportPerformance menjawab GET /api/v1/performance/export (D-202).
func (a *apiServer) ExportPerformance(ctx context.Context, req ExportPerformanceRequestObject) (ExportPerformanceResponseObject, error) {
	p := req.Params
	body, name, err := a.metrics.Export(ctx, actorFrom(ctx), string(p.Period), performanceDay(p.Date), deref(p.CustomerId), deref(p.PersonId))
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(name)
	return ExportPerformance200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(body), ContentLength: int64(len(body)),
		Headers: ExportPerformance200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

func stages(s metrics.Stages) PerformanceStages {
	return PerformanceStages{AcceptedToRelease: duration(s.AcceptedToRelease), EditToReady: duration(s.EditToReady),
		WaitingToPrint: duration(s.WaitingToPrint), PrintToSigned: duration(s.PrintToSigned), SignedToReleased: duration(s.SignedToReleased)}
}

func toAPIPerformance(p metrics.Performance) Performance {
	out := Performance{
		Period: PerformancePeriod(p.Period.Kind), From: date(p.Period.From), To: date(p.Period.To.AddDate(0, 0, -1)), Week: p.Period.Week(),
		Submitted: p.Submitted, Accepted: p.Accepted, Rejected: p.Rejected, Waiting: p.Waiting, ByTrigger: make([]TriggerCount, 0, len(p.ByTrigger)),
		ByCategory: CategoryCounts(p.ByCategory), Yokotens: p.Yokotens, YokotenTargets: p.YokotenTargets, YokotenChanges: p.YokotenChanges,
		YokotenItems: make([]PerformanceYokoten, 0, len(p.YokotenItems)), Releases: p.Releases, OpenAtEnd: p.OpenAtEnd, Stages: stages(p.Stages),
		People: make([]PerformancePerson, 0, len(p.People)), ReleaseItems: make([]PerformanceRelease, 0, len(p.ReleaseItems)),
		Trend: make([]PerformanceTrendPoint, 0, len(p.Trend)),
	}
	out.Previous.Submitted, out.Previous.Releases, out.Previous.AcceptedToReleaseMedian = p.Previous.Submitted, p.Previous.Releases, float32(p.Previous.AcceptedToReleaseMedian)
	for _, t := range p.ByTrigger {
		out.ByTrigger = append(out.ByTrigger, TriggerCount{Name: t.Name, Mechanism: t.Mechanism, Count: t.Count})
	}
	for _, y := range p.YokotenItems {
		out.YokotenItems = append(out.YokotenItems, PerformanceYokoten{Ref: y.Ref, Reason: y.Reason, Started: y.Started, Targets: y.Targets,
			Applied: y.Applied, NotApplicable: y.NotApplicable, Pending: y.Pending, DaysToClose: y.DaysToClose})
	}
	for _, x := range p.People {
		out.People = append(out.People, PerformancePerson{
			PersonId: optional(x.PersonID), Name: x.Name, Submitted: x.Submitted, Accepted: x.Accepted, Rejected: x.Rejected, Open: x.Open,
			Yokotens: x.Yokotens, Releases: x.Releases, CustomerClaim: x.CustomerClaim, AuditFinding: x.AuditFinding, Yokoten: x.Yokoten,
			Stages: stages(x.Stages),
		})
	}
	for _, r := range p.ReleaseItems {
		out.ReleaseItems = append(out.ReleaseItems, PerformanceRelease{
			RevisionId: r.RevisionID, Number: r.Number, DocumentType: r.TypeKey, Revision: r.Revision, Category: r.Category, PrintedBy: optional(r.PrintedBy),
			AcceptedAt: r.AcceptedAt, PrintedAt: r.PrintedAt, ScanAt: r.ScanAt, ReleasedAt: r.ReleasedAt,
			AcceptedToReleaseHours: hours32(r.AcceptedToReleaseHours), EditToReadyHours: hours32(r.EditToReadyHours),
			WaitingToPrintHours: hours32(r.WaitingToPrintHours), PrintToSignedHours: hours32(r.PrintToSignedHours),
			SignedToReleasedHours: hours32(r.SignedToReleasedHours),
		})
	}
	for _, t := range p.Trend {
		out.Trend = append(out.Trend, PerformanceTrendPoint{Label: t.Label, From: date(t.From), To: date(t.To.AddDate(0, 0, -1)), Releases: t.Releases,
			AcceptedToReleaseMedian: float32(t.AcceptedToReleaseMedian), ByCategory: CategoryCounts(t.ByCategory)})
	}
	return out
}

// GetInbox menjawab GET /api/v1/inbox (D-202).
func (a *apiServer) GetInbox(ctx context.Context, _ GetInboxRequestObject) (GetInboxResponseObject, error) {
	tasks, total, err := a.inbox.Tasks(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := Inbox{Total: total, Items: make([]InboxTask, 0, len(tasks))}
	for _, t := range tasks {
		item := InboxTask{Kind: InboxTaskKind(t.Kind), Title: t.Title, Detail: optional(t.Detail), Link: t.Link, Days: t.Days}
		if !t.At.IsZero() {
			at := t.At.UTC()
			item.At = &at
		}
		out.Items = append(out.Items, item)
	}
	return GetInbox200JSONResponse(out), nil
}

// SetModelEol menjawab PUT /api/v1/models/{modelId}/eol (D-200).
func (a *apiServer) SetModelEol(ctx context.Context, req SetModelEolRequestObject) (SetModelEolResponseObject, error) {
	m, err := a.products.SetEOL(ctx, actorFrom(ctx), req.ModelId, req.Body.Eol)
	if err != nil {
		return nil, err
	}
	return SetModelEol200JSONResponse(toAPIModel(m)), nil
}

// MarkDocumentReady menjawab POST /api/v1/documents/{documentId}/ready (D-202).
func (a *apiServer) MarkDocumentReady(ctx context.Context, req MarkDocumentReadyRequestObject) (MarkDocumentReadyResponseObject, error) {
	list, err := a.documents.MarkReady(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return MarkDocumentReady200JSONResponse(toAPIRevisions(list)), nil
}

func duration(d metrics.Duration) DurationSummary {
	return DurationSummary{Count: d.Count, AverageHours: float32(d.AverageHours), MedianHours: float32(d.MedianHours)}
}

func hours32(h *float64) *float32 {
	if h == nil {
		return nil
	}
	v := float32(*h)
	return &v
}

// date memberi tanggal kalender t sebagai tanggal kontrak API.
func date(t time.Time) openapi_types.Date {
	return openapi_types.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}
