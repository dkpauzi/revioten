package server

// Handler Form Control Submission (D-207). Aturannya ada di internal/submission.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/submission"
)

// SubmissionService adalah operasi Form Control Submission.
type SubmissionService interface {
	Get(ctx context.Context, actor access.Actor, documentID string) (submission.Control, error)
	Save(ctx context.Context, actor access.Actor, documentID string, revision int, in submission.Entry) (submission.Control, error)
	SetCustomerDefault(ctx context.Context, actor access.Actor, customerID string, required bool) error
	UploadScan(ctx context.Context, actor access.Actor, documentID string, revision int, fileName string, r io.Reader) (submission.Control, error)
	OpenScan(ctx context.Context, actor access.Actor, documentID string, revision int) (submission.Scan, *os.File, error)
	Export(ctx context.Context, actor access.Actor, documentID, format string) (submission.File, error)
}

func apiDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

// dayOf mengubah tanggal kontrak menjadi tanggal di zona pabrik.
func dayOf(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, clock.Zone)
	return &t
}

func toAPISubmission(c submission.Control) SubmissionControl {
	out := SubmissionControl{
		DocumentId: c.DocumentID, Number: c.Number, DocumentType: c.DocumentType, DocumentTypeName: c.DocumentTypeName,
		CustomerCode: c.CustomerCode, CustomerName: c.CustomerName, Models: c.Models, SubmissionRequired: c.SubmissionRequired,
		FormNumber: c.FormNumber, Rows: make([]SubmissionRow, 0, len(c.Rows)),
	}
	for _, r := range c.Rows {
		row := SubmissionRow{
			Revision: r.Revision, Status: SubmissionRowStatus(r.Status), Migrated: r.Migrated, Reason: r.Reason,
			StartedOn: openapi_types.Date{Time: r.StartedOn}, FinishedOn: apiDate(r.FinishedOn), Days: r.Days, TopSignOn: apiDate(r.TopSignOn),
			Required: r.Required, RequiredDefault: r.RequiredDefault, OverrideReason: optional(r.OverrideReason),
			SubmittedOn: apiDate(r.SubmittedOn), Package: optional(r.Package), ResultOn: apiDate(r.ResultOn),
			ScanName: optional(r.ScanName), Remark: optional(r.Remark), UpdatedBy: optional(r.UpdatedBy), UpdatedAt: r.UpdatedAt,
		}
		if r.Signing != "" {
			s := SubmissionRowSigning(r.Signing)
			row.Signing = &s
		}
		if r.TopSignBy != "" {
			b := SubmissionRowTopSignBy(r.TopSignBy)
			row.TopSignBy = &b
		}
		if r.Result != "" {
			res := SubmissionRowResult(r.Result)
			row.Result = &res
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// GetSubmissionControl menjawab GET /api/v1/documents/{documentId}/submission-control.
func (a *apiServer) GetSubmissionControl(ctx context.Context, req GetSubmissionControlRequestObject) (GetSubmissionControlResponseObject, error) {
	c, err := a.submissions.Get(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return GetSubmissionControl200JSONResponse(toAPISubmission(c)), nil
}

// SaveSubmissionEntry menjawab PUT /api/v1/documents/{documentId}/submission-control/{revision}.
func (a *apiServer) SaveSubmissionEntry(ctx context.Context, req SaveSubmissionEntryRequestObject) (SaveSubmissionEntryResponseObject, error) {
	b := req.Body
	in := submission.Entry{
		Required: b.Required, OverrideReason: value(b.OverrideReason), SubmittedOn: dayOf(b.SubmittedOn), Package: value(b.Package),
		ResultOn: dayOf(b.ResultOn), Remark: value(b.Remark),
	}
	if b.Result != nil {
		in.Result = string(*b.Result)
	}
	c, err := a.submissions.Save(ctx, actorFrom(ctx), req.DocumentId, req.Revision, in)
	if err != nil {
		return nil, err
	}
	return SaveSubmissionEntry200JSONResponse(toAPISubmission(c)), nil
}

// UploadSubmissionScan menjawab POST /api/v1/documents/{documentId}/submission-control/{revision}/scan.
func (a *apiServer) UploadSubmissionScan(ctx context.Context, req UploadSubmissionScanRequestObject) (UploadSubmissionScanResponseObject, error) {
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
		c, err := a.submissions.UploadScan(ctx, actorFrom(ctx), req.DocumentId, req.Revision, part.FileName(), part)
		if err != nil {
			return nil, err
		}
		return UploadSubmissionScan200JSONResponse(toAPISubmission(c)), nil
	}
	return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.Required, Message: "Choose the customer approval file."}}}
}

// GetSubmissionScan menjawab GET /api/v1/documents/{documentId}/submission-control/{revision}/scan.
func (a *apiServer) GetSubmissionScan(ctx context.Context, req GetSubmissionScanRequestObject) (GetSubmissionScanResponseObject, error) {
	sc, f, err := a.submissions.OpenScan(ctx, actorFrom(ctx), req.DocumentId, req.Revision)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(sc.FileName)
	return GetSubmissionScan200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(content), ContentLength: int64(len(content)),
		Headers: GetSubmissionScan200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ExportSubmissionControl menjawab GET /api/v1/documents/{documentId}/submission-control/export.
func (a *apiServer) ExportSubmissionControl(ctx context.Context, req ExportSubmissionControlRequestObject) (ExportSubmissionControlResponseObject, error) {
	f, err := a.submissions.Export(ctx, actorFrom(ctx), req.DocumentId, string(req.Params.Format))
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(f.Name)
	return ExportSubmissionControl200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(f.Body), ContentLength: int64(len(f.Body)),
		Headers: ExportSubmissionControl200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// SetCustomerSubmissionRequired menjawab PUT /api/v1/customers/{customerId}/submission-required.
func (a *apiServer) SetCustomerSubmissionRequired(ctx context.Context, req SetCustomerSubmissionRequiredRequestObject) (SetCustomerSubmissionRequiredResponseObject, error) {
	actor := actorFrom(ctx)
	if err := a.submissions.SetCustomerDefault(ctx, actor, req.CustomerId, req.Body.Required); err != nil {
		return nil, err
	}
	list, err := a.customers.List(ctx, actor)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.ID == req.CustomerId {
			return SetCustomerSubmissionRequired200JSONResponse(toAPICustomer(c)), nil
		}
	}
	return nil, problem.ErrNotFound
}
