package server

// Handler dokumen terbit untuk pembaca (D-195), review berkala (D-196), dan
// laporan kinerja (D-197). Aturannya ada di internal/document, internal/review,
// dan internal/metrics; handler hanya menerjemahkan bentuk data.

import (
	"context"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/metrics"
	"github.com/dkpauzi/revioten/internal/review"
)

// ReviewService adalah review berkala yang dipakai server.
type ReviewService interface {
	Sets(ctx context.Context, actor access.Actor, f review.Filter) (review.Overview, error)
	Record(ctx context.Context, actor access.Actor, in review.NewReview) (review.Review, error)
	Confirm(ctx context.Context, actor access.Actor, id string) (review.Review, error)
	Reviews(ctx context.Context, actor access.Actor, customerID string, openOnly bool) ([]review.Review, error)
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
	f := review.Filter{CustomerID: deref(req.Params.CustomerId), DocumentID: deref(req.Params.DocumentId), All: req.Params.All != nil && *req.Params.All}
	o, err := a.reviews.Sets(ctx, actorFrom(ctx), f)
	if err != nil {
		return nil, err
	}
	out := ReviewSetList{PeriodMonths: o.PeriodMonths, WarningDays: o.WarningDays, DueSoon: o.DueSoon, Overdue: o.Overdue, Items: make([]ReviewSet, 0, len(o.Sets))}
	for _, s := range o.Sets {
		item := ReviewSet{
			CustomerId: s.CustomerID, CustomerCode: s.CustomerCode, Sequence: s.Sequence, Models: orEmpty(s.Models),
			LastRelease: date(s.LastRelease), DueOn: date(s.DueOn), Status: ReviewSetStatus(s.Status), DaysLeft: s.DaysLeft,
			OpenReviewId: optional(s.OpenReviewID), DraftOpen: s.DraftOpen, Documents: make([]ReviewSetDocument, 0, len(s.Documents)),
		}
		if s.LastReview != nil {
			d := date(*s.LastReview)
			item.LastReview = &d
		}
		for _, d := range s.Documents {
			item.Documents = append(item.Documents, ReviewSetDocument{Id: d.ID, Number: d.Number, DocumentType: d.TypeKey, Revision: d.Revision})
		}
		out.Items = append(out.Items, item)
	}
	return ListReviewSets200JSONResponse(out), nil
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

// GetPerformance menjawab GET /api/v1/performance.
func (a *apiServer) GetPerformance(ctx context.Context, req GetPerformanceRequestObject) (GetPerformanceResponseObject, error) {
	day := time.Now()
	if req.Params.Date != nil {
		t := req.Params.Date.Time
		day = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, clock.Zone)
	}
	p, err := a.metrics.Performance(ctx, actorFrom(ctx), string(req.Params.Period), day, deref(req.Params.CustomerId))
	if err != nil {
		return nil, err
	}
	out := Performance{
		Period: PerformancePeriod(p.Period.Kind), From: date(p.Period.From), To: date(p.Period.To.AddDate(0, 0, -1)),
		Submitted: p.Submitted, Accepted: p.Accepted, Rejected: p.Rejected, ByTrigger: make([]TriggerCount, 0, len(p.ByTrigger)),
		Yokotens: p.Yokotens, YokotenTargets: p.YokotenTargets, YokotenChanges: p.YokotenChanges, Releases: p.Releases,
		AcceptedToRelease: duration(p.AcceptedToRelease), EditToPrint: duration(p.EditToPrint), PrintToSigned: duration(p.PrintToSigned),
		People: make([]PerformancePerson, 0, len(p.People)), ReleaseItems: make([]PerformanceRelease, 0, len(p.ReleaseItems)),
	}
	for _, t := range p.ByTrigger {
		out.ByTrigger = append(out.ByTrigger, TriggerCount{Name: t.Name, Mechanism: t.Mechanism, Count: t.Count})
	}
	for _, x := range p.People {
		out.People = append(out.People, PerformancePerson{
			PersonId: optional(x.PersonID), Name: x.Name, Submitted: x.Submitted, Accepted: x.Accepted, Rejected: x.Rejected,
			Yokotens: x.Yokotens, Releases: x.Releases, AcceptedToRelease: duration(x.AcceptedToRelease),
			EditToPrint: duration(x.EditToPrint), PrintToSigned: duration(x.PrintToSigned),
		})
	}
	for _, r := range p.ReleaseItems {
		out.ReleaseItems = append(out.ReleaseItems, PerformanceRelease{
			RevisionId: r.RevisionID, Number: r.Number, DocumentType: r.TypeKey, Revision: r.Revision, PrintedBy: optional(r.PrintedBy),
			AcceptedAt: r.AcceptedAt, PrintedAt: r.PrintedAt, ScanAt: r.ScanAt, ReleasedAt: r.ReleasedAt,
			AcceptedToReleaseHours: hours32(r.AcceptedToReleaseHours), EditToPrintHours: hours32(r.EditToPrintHours),
			PrintToSignedHours: hours32(r.PrintToSignedHours),
		})
	}
	return GetPerformance200JSONResponse(out), nil
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
