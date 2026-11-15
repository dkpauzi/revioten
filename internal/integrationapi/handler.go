package integrationapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/integration"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service adalah operasi integrasi yang dipakai adapter ini.
// *integration.Service memenuhinya; contract test memakai versi tiruan.
type Service interface {
	Authenticate(ctx context.Context, token, ip string) (integration.Client, error)
	ListDocuments(ctx context.Context, c integration.Client, f integration.ListFilter) (integration.Page, error)
	GetRevision(ctx context.Context, c integration.Client, documentID string, revision *int) (integration.Revision, error)
	Propose(ctx context.Context, c integration.Client, key string, in integration.Proposal) (integration.ProposedChange, bool, error)
	ChangeStatus(ctx context.Context, c integration.Client, id string) (integration.ProposedChange, error)
	CreateExport(ctx context.Context, c integration.Client, documentID string, revision *int) (integration.Job, error)
	GetExport(ctx context.Context, c integration.Client, id string) (integration.Job, error)
	OpenExport(ctx context.Context, c integration.Client, id string) (integration.Job, *os.File, error)
}

// maxBodyBytes adalah ukuran body permintaan paling besar (1 MiB, R-API-13).
const maxBodyBytes = 1 << 20

// DefaultRatePerMinute adalah batas laju bawaan per token (R-API-14).
const DefaultRatePerMinute = 120

type api struct {
	svc   Service
	limit *limiter
	log   *slog.Logger
}

type clientKey struct{}

func clientFrom(ctx context.Context) integration.Client {
	c, _ := ctx.Value(clientKey{}).(integration.Client)
	return c
}

// NewHandler menyusun route /integration/v1. ratePerMinute adalah jumlah
// permintaan per token per menit; nol berarti DefaultRatePerMinute.
func NewHandler(svc Service, ratePerMinute int, log *slog.Logger) http.Handler {
	if ratePerMinute <= 0 {
		ratePerMinute = DefaultRatePerMinute
	}
	if log == nil {
		log = slog.Default()
	}
	a := &api{svc: svc, limit: newLimiter(ratePerMinute, time.Minute, time.Now), log: log}
	mux := http.NewServeMux()
	strict := NewStrictHandlerWithOptions(a, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  a.badRequest,
		ResponseErrorHandlerFunc: a.fail,
	})
	HandlerWithOptions(strict, StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: a.badRequest,
		Middlewares:      []MiddlewareFunc{a.authenticate},
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Alamat di luar kontrak tetap dijawab dengan format error yang sama.
		if _, pattern := mux.Handler(r); pattern == "" {
			a.writeError(w, r, http.StatusNotFound, ErrorCodeNotFound, "This address does not exist in the integration API.", nil)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		mux.ServeHTTP(w, r)
	})
}

// authenticate mengenali aplikasi dari token bearer, lalu menerapkan batas
// laju per token. Token tidak pernah ditulis ke log (R-SEC-14).
func (a *api) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			token = ""
		}
		client, err := a.svc.Authenticate(r.Context(), strings.TrimSpace(token), audit.RequestFrom(r.Context()).IP)
		if errors.Is(err, integration.ErrUnauthenticated) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			a.writeError(w, r, http.StatusUnauthorized, ErrorCodeUnauthenticated, "Send a valid service account token as a bearer token.", nil)
			return
		}
		if err != nil {
			a.fail(w, r, err)
			return
		}
		if wait, limited := a.limit.take(client.TokenID); limited {
			w.Header().Set("Retry-After", strconv.Itoa(wait))
			a.writeError(w, r, http.StatusTooManyRequests, ErrorCodeRateLimited, "Too many requests for this token. Wait and try again.", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientKey{}, client)))
	})
}

func (a *api) writeError(w http.ResponseWriter, r *http.Request, status int, code ErrorCode, message string, details []problem.Problem) {
	body := Error{Code: code, Message: message, RequestId: audit.RequestFrom(r.Context()).ID}
	if len(details) > 0 {
		list := make([]ErrorDetail, 0, len(details))
		for _, d := range details {
			list = append(list, ErrorDetail{Field: d.Field, Code: d.Code, Message: d.Message})
		}
		body.Details = &list
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// badRequest menjawab permintaan yang tidak bisa dibaca (JSON rusak, parameter salah).
func (a *api) badRequest(w http.ResponseWriter, r *http.Request, _ error) {
	a.writeError(w, r, http.StatusBadRequest, ErrorCodeInvalidRequest, "The request could not be read. Check the JSON body and the parameters.", nil)
}

// fail menerjemahkan error layanan ke jawaban API. Rincian error yang tidak
// dikenal hanya masuk log server bersama request ID (R-API-13).
func (a *api) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		a.writeError(w, r, http.StatusUnprocessableEntity, ErrorCodeValidationFailed, "The request was refused. See details.", invalid.Problems)
	case errors.As(err, &conflict):
		a.writeError(w, r, http.StatusConflict, ErrorCodeConflict, "The request conflicts with the current state. See details.", conflict.Problems)
	case errors.Is(err, problem.ErrNotFound):
		a.writeError(w, r, http.StatusNotFound, ErrorCodeNotFound, "The requested data does not exist.", nil)
	case errors.Is(err, access.ErrForbidden):
		a.writeError(w, r, http.StatusForbidden, ErrorCodeForbidden, "This token does not have the scope for this operation.", nil)
	default:
		a.log.Error("permintaan API integrasi gagal", "request_id", audit.RequestFrom(r.Context()).ID, "method", r.Method, "path", r.URL.Path, "error", err.Error())
		a.writeError(w, r, http.StatusInternalServerError, ErrorCodeInternalError, "Something went wrong on the server.", nil)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListReleasedDocuments menjawab GET /integration/v1/documents.
func (a *api) ListReleasedDocuments(ctx context.Context, req ListReleasedDocumentsRequestObject) (ListReleasedDocumentsResponseObject, error) {
	p := req.Params
	f := integration.ListFilter{CustomerID: deref(p.CustomerId), UpdatedSince: p.UpdatedSince, Cursor: deref(p.Cursor)}
	if p.Limit != nil {
		f.Limit = *p.Limit
		if f.Limit == 0 {
			f.Limit = -1 // nol yang dikirim sengaja tetap ditolak
		}
	}
	page, err := a.svc.ListDocuments(ctx, clientFrom(ctx), f)
	if err != nil {
		return nil, err
	}
	out := DocumentPage{Items: make([]ReleasedDocument, 0, len(page.Items)), NextCursor: optional(page.NextCursor)}
	for _, d := range page.Items {
		out.Items = append(out.Items, ReleasedDocument{Id: d.ID, Number: d.Number, CustomerId: d.CustomerID, CustomerCode: d.CustomerCode,
			DocumentType: d.DocumentType, Revision: d.Revision, ReleasedAt: d.ReleasedAt, Sha256: d.SHA256, Migrated: d.Migrated})
	}
	return ListReleasedDocuments200JSONResponse(out), nil
}

// GetReleasedRevision menjawab GET /integration/v1/documents/{documentId}/revisions/{revision}.
func (a *api) GetReleasedRevision(ctx context.Context, req GetReleasedRevisionRequestObject) (GetReleasedRevisionResponseObject, error) {
	var revision *int
	if req.Revision != "current" {
		n, err := strconv.Atoi(req.Revision)
		if err != nil {
			return nil, problem.ErrNotFound
		}
		revision = &n
	}
	r, err := a.svc.GetRevision(ctx, clientFrom(ctx), req.DocumentId, revision)
	if err != nil {
		return nil, err
	}
	s := r.Snapshot
	out := Revision{DocumentId: r.DocumentID, Number: r.Number, CustomerId: r.CustomerID, DocumentType: s.DocumentType, Revision: r.Revision, Current: r.Current,
		ReleasedAt: r.ReleasedAt, ReleaseDate: optional(r.ReleaseDate), Sha256: r.SHA256, Migrated: r.Migrated,
		Models: orEmpty(s.Models), PartNumbers: orEmpty(s.PartNumbers), Header: s.Header,
		Fields: make([]Field, 0, len(r.Fields)), Sections: make([]Section, 0, len(s.Sections)), Rows: make([]Row, 0, len(s.Rows))}
	if out.Header == nil {
		out.Header = map[string]any{}
	}
	for _, f := range r.Fields {
		out.Fields = append(out.Fields, Field{Id: f.ID, Key: optional(f.Key), Label: f.Label, Area: FieldArea(f.Area)})
	}
	for _, sec := range s.Sections {
		out.Sections = append(out.Sections, Section{Id: sec.ID, Name: sec.Name, PartNumbers: orEmpty(sec.PartNumbers)})
	}
	for _, row := range s.Rows {
		content := row.Content
		if content == nil {
			content = map[string]any{}
		}
		out.Rows = append(out.Rows, Row{Id: row.ID, Position: row.Position, SectionId: optional(row.SectionID), ProcessId: optional(row.ProcessID),
			ProcessName: optional(row.ProcessName), OperationNumber: optional(row.OperationNumber), Content: content})
	}
	return GetReleasedRevision200JSONResponse(out), nil
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func toProposed(c integration.ProposedChange) ProposedChange {
	return ProposedChange{Id: c.ID, Ref: c.Ref, Status: ProposedChangeStatus(c.Status), CreatedAt: c.CreatedAt, VerifiedAt: c.VerifiedAt, VerificationNote: optional(c.VerificationNote)}
}

// ProposeChange menjawab POST /integration/v1/changes.
func (a *api) ProposeChange(ctx context.Context, req ProposeChangeRequestObject) (ProposeChangeResponseObject, error) {
	b := req.Body
	in := integration.Proposal{DocumentID: b.DocumentId, BaseRevision: b.BaseRevision, SourceApplication: b.SourceApplication,
		OnBehalfOf: deref(b.OnBehalfOf), Reason: b.Reason, Reference: deref(b.Reference)}
	if b.ChangeType != nil {
		in.ChangeType = string(*b.ChangeType)
	}
	for _, op := range b.Operations {
		item := document.PatchOp{Op: string(op.Op), RowID: deref(op.RowId), ProcessID: deref(op.ProcessId), SectionID: deref(op.SectionId)}
		if op.Content != nil {
			item.Content = *op.Content
		}
		in.Operations = append(in.Operations, item)
	}
	change, replayed, err := a.svc.Propose(ctx, clientFrom(ctx), req.Params.IdempotencyKey, in)
	if err != nil {
		return nil, err
	}
	if replayed {
		return ProposeChange200JSONResponse(toProposed(change)), nil
	}
	return ProposeChange201JSONResponse(toProposed(change)), nil
}

// GetProposedChange menjawab GET /integration/v1/changes/{changeId}.
func (a *api) GetProposedChange(ctx context.Context, req GetProposedChangeRequestObject) (GetProposedChangeResponseObject, error) {
	change, err := a.svc.ChangeStatus(ctx, clientFrom(ctx), req.ChangeId)
	if err != nil {
		return nil, err
	}
	return GetProposedChange200JSONResponse(toProposed(change)), nil
}

func toJob(j integration.Job) ExportJob {
	out := ExportJob{Id: j.ID, DocumentId: j.DocumentID, Revision: j.Revision, Status: ExportJobStatus(j.Status), CreatedAt: j.CreatedAt,
		FinishedAt: j.FinishedAt, FileName: optional(j.FileName), Error: optional(j.Error)}
	if j.SizeBytes > 0 {
		out.SizeBytes = &j.SizeBytes
	}
	return out
}

// CreateExport menjawab POST /integration/v1/exports.
func (a *api) CreateExport(ctx context.Context, req CreateExportRequestObject) (CreateExportResponseObject, error) {
	job, err := a.svc.CreateExport(ctx, clientFrom(ctx), req.Body.DocumentId, req.Body.Revision)
	if err != nil {
		return nil, err
	}
	return CreateExport202JSONResponse(toJob(job)), nil
}

// GetExport menjawab GET /integration/v1/exports/{exportId}.
func (a *api) GetExport(ctx context.Context, req GetExportRequestObject) (GetExportResponseObject, error) {
	job, err := a.svc.GetExport(ctx, clientFrom(ctx), req.ExportId)
	if err != nil {
		return nil, err
	}
	return GetExport200JSONResponse(toJob(job)), nil
}

// DownloadExport menjawab GET /integration/v1/exports/{exportId}/file.
func (a *api) DownloadExport(ctx context.Context, req DownloadExportRequestObject) (DownloadExportResponseObject, error) {
	job, file, err := a.svc.OpenExport(ctx, clientFrom(ctx), req.ExportId)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(job.FileName)
	return DownloadExport200ApplicationoctetStreamResponse{Body: bytes.NewReader(body), ContentLength: int64(len(body)),
		Headers: DownloadExport200ResponseHeaders{ContentDisposition: &disposition}}, nil
}

// limiter adalah batas laju jendela tetap per token: paling banyak max
// permintaan dalam satu jendela waktu. Cukup untuk satu proses server; versi
// dengan beberapa server butuh penghitung bersama.
type limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	seen   map[string]*usage
}

type usage struct {
	start time.Time
	count int
}

func newLimiter(max int, window time.Duration, now func() time.Time) *limiter {
	return &limiter{max: max, window: window, now: now, seen: map[string]*usage{}}
}

// take mencatat satu permintaan. Bila batas terlampaui, hasilnya detik yang
// harus ditunggu (untuk header Retry-After) dan true.
func (l *limiter) take(key string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	u, ok := l.seen[key]
	if !ok || now.Sub(u.start) >= l.window {
		l.seen[key] = &usage{start: now, count: 1}
		return 0, false
	}
	if u.count >= l.max {
		wait := int(u.start.Add(l.window).Sub(now).Seconds()) + 1
		return wait, true
	}
	u.count++
	return 0, false
}
