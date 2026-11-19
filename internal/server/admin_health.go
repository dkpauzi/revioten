package server

// Handler dasbor admin (fase 4, D-205). Aturannya ada di internal/health.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/health"
)

// HealthService adalah operasi dasbor admin dan pencatat error server.
type HealthService interface {
	Overview(ctx context.Context, actor access.Actor) (health.Overview, error)
	WaitingSignatures(ctx context.Context, actor access.Actor) ([]health.WaitingSignature, error)
	ErrorReports(ctx context.Context, actor access.Actor) ([]health.ErrorReport, error)
	ProblemReports(ctx context.Context, actor access.Actor) ([]health.ProblemReport, error)
	ReportProblem(ctx context.Context, actor access.Actor, page, description, lastRequestID string) (string, error)
	UpdateProblem(ctx context.Context, actor access.Actor, id, status, note string) error
	RecordError(ctx context.Context, requestID, method, path, accountID, message string) error
}

func toAPIWaiting(list []health.WaitingSignature) []WaitingSignature {
	out := make([]WaitingSignature, 0, len(list))
	for _, w := range list {
		out = append(out, WaitingSignature{
			RequestId: w.RequestID, DocumentId: w.DocumentID, Number: w.Number, Revision: w.Revision, CustomerCode: w.CustomerCode,
			Slot: w.Slot, Signer: w.Signer, SentBy: w.SentBy, SentAt: w.SentAt, WaitingSince: w.WaitingSince, Hours: float32(w.Hours),
		})
	}
	return out
}

func toAPIErrors(list []health.ErrorReport) []ErrorReport {
	out := make([]ErrorReport, 0, len(list))
	for _, e := range list {
		out = append(out, ErrorReport{Id: e.ID, OccurredAt: e.OccurredAt, RequestId: e.RequestID, Method: e.Method, Path: e.Path, Message: e.Message, Login: optional(e.Login)})
	}
	return out
}

func toAPIProblems(list []health.ProblemReport) []ProblemReport {
	out := make([]ProblemReport, 0, len(list))
	for _, p := range list {
		out = append(out, ProblemReport{
			Id: p.ID, CreatedAt: p.CreatedAt, ReportedBy: p.ReportedBy, Login: p.Login, Page: p.Page, Description: p.Description,
			LastRequestId: optional(p.LastRequestID), Status: ProblemReportStatus(p.Status), HandledBy: optional(p.HandledBy),
			HandledAt: p.HandledAt, AdminNote: optional(p.AdminNote),
		})
	}
	return out
}

func toAPIProcessed(p health.Processed) ProcessedPeriod {
	out := ProcessedPeriod{From: date(p.From), To: date(p.To.AddDate(0, 0, -1)), Total: p.Total, Counts: make([]ProcessedCount, 0, len(health.Categories))}
	for _, c := range health.Categories {
		out.Counts = append(out.Counts, ProcessedCount{Category: ProcessedCountCategory(c), Count: p.Counts[c]})
	}
	return out
}

// GetAdminHealth menjawab GET /api/v1/admin/health.
func (a *apiServer) GetAdminHealth(ctx context.Context, _ GetAdminHealthRequestObject) (GetAdminHealthResponseObject, error) {
	o, err := a.health.Overview(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := AdminHealth{
		SignaturesWaiting: o.SignaturesWaiting, TopSignatures: toAPIWaiting(o.TopSignatures), Errors24h: o.Errors24h, Errors7d: o.Errors7d,
		LatestErrors: toAPIErrors(o.LatestErrors), ProblemsOpen: o.ProblemsOpen, LatestProblems: toAPIProblems(o.LatestProblems),
		Week: toAPIProcessed(o.Week), Month: toAPIProcessed(o.Month), EmailFailed: o.EmailFailed, EmailPending: o.EmailPending,
		AccountsLocked: o.AccountsLocked, AccountsDisabled: o.AccountsDisabled, FailedLogins24h: o.FailedLogins24h,
		BackupConfigured: o.Backup.Configured, BackupFound: o.Backup.Found, AuditValid: o.AuditValid, AuditEntries: o.AuditEntries,
	}
	if o.Backup.Found {
		at := o.Backup.CreatedAt
		out.BackupAt = &at
		out.BackupFile = optional(o.Backup.File)
	}
	return GetAdminHealth200JSONResponse(out), nil
}

// ListWaitingSignatures menjawab GET /api/v1/admin/waiting-signatures.
func (a *apiServer) ListWaitingSignatures(ctx context.Context, _ ListWaitingSignaturesRequestObject) (ListWaitingSignaturesResponseObject, error) {
	list, err := a.health.WaitingSignatures(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return ListWaitingSignatures200JSONResponse(WaitingSignatureList{Items: toAPIWaiting(list)}), nil
}

// ListErrorReports menjawab GET /api/v1/admin/error-reports.
func (a *apiServer) ListErrorReports(ctx context.Context, _ ListErrorReportsRequestObject) (ListErrorReportsResponseObject, error) {
	list, err := a.health.ErrorReports(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return ListErrorReports200JSONResponse(ErrorReportList{Items: toAPIErrors(list)}), nil
}

// ListProblemReports menjawab GET /api/v1/admin/problem-reports.
func (a *apiServer) ListProblemReports(ctx context.Context, _ ListProblemReportsRequestObject) (ListProblemReportsResponseObject, error) {
	list, err := a.health.ProblemReports(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return ListProblemReports200JSONResponse(ProblemReportList{Items: toAPIProblems(list)}), nil
}

// UpdateProblemReport menjawab PATCH /api/v1/admin/problem-reports/{reportId}.
func (a *apiServer) UpdateProblemReport(ctx context.Context, req UpdateProblemReportRequestObject) (UpdateProblemReportResponseObject, error) {
	if err := a.health.UpdateProblem(ctx, actorFrom(ctx), req.ReportId, string(req.Body.Status), value(req.Body.Note)); err != nil {
		return nil, err
	}
	return UpdateProblemReport204Response{}, nil
}

// ReportProblem menjawab POST /api/v1/problem-reports.
func (a *apiServer) ReportProblem(ctx context.Context, req ReportProblemRequestObject) (ReportProblemResponseObject, error) {
	id, err := a.health.ReportProblem(ctx, actorFrom(ctx), req.Body.Page, req.Body.Description, value(req.Body.LastRequestId))
	if err != nil {
		return nil, err
	}
	return ReportProblem201JSONResponse(CreatedId{Id: id}), nil
}
