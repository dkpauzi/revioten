// Package submission menyusun Form Control Submission per dokumen (D-207).
//
// Setiap dokumen (PFD, PFMEA, Control Plan) punya daftarnya sendiri; satu
// baris adalah satu revisi. Kolom yang dihitung sistem:
//   - Reason: perubahan yang masuk ke revisi itu beserta pemicunya;
//   - Date of start: perubahan pertama diterima, atau draft dibuat untuk
//     revisi awal;
//   - Date of finish: tanggal rilis;
//   - Top management sign: tanda tangan digital terakhir (D-198), atau scan
//     bertanda tangan diunggah untuk tanda tangan basah.
//
// Kolom yang diketik pemegang E-7: perlu dikirim ke customer (bawaannya dari
// pengaturan customer, pengecualian dengan alasan), tanggal kirim dan isi
// paket, hasil customer dan tanggalnya, scan bukti (opsional), dan catatan.
// Catatan submission terpisah dari revisi terbit, jadi tetap bisa diisi
// sesudah rilis; setiap perubahan dicatat di audit log.
package submission

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
)

// Hasil customer.
const (
	ResultWaiting  = "waiting"
	ResultApproved = "approved"
	ResultRejected = "rejected"
)

// MaxScanBytes adalah ukuran scan bukti customer paling besar (sama dengan
// scan bertanda tangan, M6.7).
const MaxScanBytes = 30 << 20

// Service menyusun dan mengubah Form Control Submission.
type Service struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	authz   access.Authorizer
	store   *storage.Store
	browser string
	now     func() time.Time
}

// NewService membuat Service. store menyimpan scan bukti customer (boleh nil
// di test yang tidak mengunggah); browser adalah Chrome atau Edge untuk PDF.
func NewService(pool *pgxpool.Pool, authz access.Authorizer, store *storage.Store, browser string) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, store: store, browser: browser, now: time.Now}
}

// Row adalah satu revisi di form.
type Row struct {
	RevisionID string
	Revision   int
	Status     string // draft atau released
	Signing    string // "", waiting, atau completed (tanda tangan digital)
	Migrated   bool
	Reason     string
	StartedOn  time.Time  // tanggal WIB
	FinishedOn *time.Time // tanggal rilis; nil bila belum
	Days       int        // hari kalender dari mulai sampai selesai, atau sampai hari ini
	TopSignOn  *time.Time
	TopSignBy  string // digital atau scan; "" bila belum
	// Required: perlu dikirim ke customer. RequiredDefault: bawaan customer.
	Required        bool
	RequiredDefault bool
	OverrideReason  string
	SubmittedOn     *time.Time
	Package         string
	Result          string // "", waiting, approved, rejected
	ResultOn        *time.Time
	ScanName        string
	Remark          string
	UpdatedBy       string
	UpdatedAt       *time.Time
}

// Control adalah satu form: header dan baris per revisi.
type Control struct {
	DocumentID         string
	Number             string
	DocumentType       string // pfd, pfmea, cp
	DocumentTypeName   string
	CustomerCode       string
	CustomerName       string
	Models             string
	SubmissionRequired bool // bawaan customer
	FormNumber         string
	Rows               []Row // lama ke baru, tidak pernah nil
}

// Entry adalah isian yang diketik untuk satu revisi.
type Entry struct {
	// Required nil berarti mengikuti bawaan customer.
	Required       *bool
	OverrideReason string
	SubmittedOn    *time.Time
	Package        string
	Result         string
	ResultOn       *time.Time
	Remark         string
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

// keep meneruskan galat validasi, "tidak ada", dan "tidak berhak" apa adanya.
func keep(err error, doing string) error {
	if err == nil {
		return nil
	}
	var v *problem.ValidationError
	if errors.As(err, &v) || errors.Is(err, problem.ErrNotFound) || errors.Is(err, access.ErrForbidden) {
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// customerOf membaca customer satu dokumen; dokumen yang tidak terlihat
// (RLS) atau tidak ada memberi ErrNotFound.
func (s *Service) customerOf(ctx context.Context, documentID string) (string, error) {
	if !db.IsUUID(documentID) {
		return "", problem.ErrNotFound
	}
	d, err := s.q.GetDocument(ctx, documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("membaca dokumen: %w", err)
	}
	return d.CustomerID, nil
}

// Get menyusun form satu dokumen (L-4).
func (s *Service) Get(ctx context.Context, actor access.Actor, documentID string) (Control, error) {
	customerID, err := s.customerOf(ctx, documentID)
	if err != nil {
		return Control{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: customerID}); err != nil {
		return Control{}, err
	}
	return s.load(ctx, documentID)
}

func (s *Service) load(ctx context.Context, documentID string) (Control, error) {
	h, err := s.q.SubmissionHeader(ctx, documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Control{}, problem.ErrNotFound
	}
	if err != nil {
		return Control{}, fmt.Errorf("membaca header form: %w", err)
	}
	c := Control{
		DocumentID: h.ID, Number: h.Number, DocumentType: deref(h.TypeKey), DocumentTypeName: h.TypeName,
		CustomerCode: h.CustomerCode, CustomerName: h.CustomerName, Models: h.Models,
		SubmissionRequired: h.SubmissionRequired, FormNumber: h.FormNumber, Rows: []Row{},
	}
	rows, err := s.q.SubmissionRows(ctx, documentID)
	if err != nil {
		return Control{}, fmt.Errorf("membaca revisi: %w", err)
	}
	today := clock.Date(s.now())
	for _, r := range rows {
		row := Row{
			RevisionID: r.ID, Revision: int(r.Revision), Status: r.Status, Signing: deref(r.Signing), Migrated: r.Migrated,
			StartedOn: clock.Date(r.StartedAt), RequiredDefault: h.SubmissionRequired, Required: h.SubmissionRequired,
			OverrideReason: deref(r.OverrideReason), Package: deref(r.Package), Result: deref(r.Result),
			ScanName: deref(r.ResultScanName), Remark: deref(r.Remark), UpdatedBy: deref(r.UpdatedBy),
		}
		if r.RequiredOverride != nil {
			row.Required = *r.RequiredOverride
		}
		if r.Status == "released" {
			finished := today
			switch {
			case r.ReleaseDate.Valid:
				finished = clock.Date(r.ReleaseDate.Time)
			case r.ReleasedAt != nil:
				finished = clock.Date(*r.ReleasedAt)
			}
			row.FinishedOn = &finished
			row.Days = clock.Days(row.StartedOn, finished)
		} else {
			row.Days = clock.Days(row.StartedOn, today)
		}
		switch {
		case r.SignedAt != nil && deref(r.Signing) == "completed":
			at := clock.Date(*r.SignedAt)
			row.TopSignOn, row.TopSignBy = &at, "digital"
		case r.ScanUploadedAt != nil:
			at := clock.Date(*r.ScanUploadedAt)
			row.TopSignOn, row.TopSignBy = &at, "scan"
		}
		row.SubmittedOn = dateOf(r.SubmittedOn)
		row.ResultOn = dateOf(r.ResultOn)
		if r.UpdatedAt != nil && r.UpdatedBy != nil {
			at := r.UpdatedAt.UTC()
			row.UpdatedAt = &at
		}
		if row.Reason, err = s.reason(ctx, r.ID, row); err != nil {
			return Control{}, err
		}
		c.Rows = append(c.Rows, row)
	}
	return c, nil
}

// reason menyusun kolom Reason dari perubahan yang masuk ke revisi itu.
func (s *Service) reason(ctx context.Context, revisionID string, row Row) (string, error) {
	changes, err := s.q.SubmissionReasons(ctx, &revisionID)
	if err != nil {
		return "", fmt.Errorf("membaca alasan revisi: %w", err)
	}
	var parts []string
	for _, c := range changes {
		ref := ""
		if c.Seq != nil {
			ref = fmt.Sprintf("CHG-%04d", *c.Seq)
		}
		label := c.TriggerName
		if c.Mechanism == "yokoten" {
			label = "Yokoten"
		}
		parts = append(parts, strings.TrimSpace(strings.Trim(ref+" · "+label, " ·")))
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, "; "), nil
	case row.Migrated:
		return "Legacy document", nil
	case row.Revision == 0:
		return "First release", nil
	}
	return "", nil
}

// Save menyimpan isian yang diketik untuk satu revisi (E-7).
func (s *Service) Save(ctx context.Context, actor access.Actor, documentID string, revision int, in Entry) (Control, error) {
	customerID, err := s.customerOf(ctx, documentID)
	if err != nil {
		return Control{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.RecordRelease, access.Resource{CustomerID: customerID}); err != nil {
		return Control{}, err
	}
	h, err := s.q.SubmissionHeader(ctx, documentID)
	if err != nil {
		return Control{}, fmt.Errorf("membaca header form: %w", err)
	}
	params, err := clean(in, h.SubmissionRequired)
	if err != nil {
		return Control{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		rev, err := q.RevisionOfDocument(ctx, db.RevisionOfDocumentParams{DocumentID: documentID, Revision: int16(revision)})
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := q.LockSubmission(ctx, db.LockSubmissionParams{RevisionID: rev.ID, DocumentID: documentID, UpdatedByPersonID: actor.PersonID}); err != nil {
			return err
		}
		params.RevisionID, params.PersonID = rev.ID, actor.PersonID
		if err := q.SaveSubmission(ctx, params); err != nil {
			return err
		}
		details := map[string]string{"number": h.Number, "revision": strconv.Itoa(revision), "result": deref(params.Result)}
		if params.SubmittedOn.Valid {
			details["submitted_on"] = params.SubmittedOn.Time.Format(time.DateOnly)
		}
		if params.RequiredOverride != nil {
			details["required"] = strconv.FormatBool(*params.RequiredOverride)
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSubmissionChanged, TargetType: audit.TargetDocument, TargetID: documentID, Details: details,
		})
	})
	if err != nil {
		return Control{}, keep(err, "menyimpan submission")
	}
	return s.load(ctx, documentID)
}

// clean memeriksa isian dan menyusunnya menjadi parameter query.
func clean(in Entry, customerDefault bool) (db.SaveSubmissionParams, error) {
	var p db.SaveSubmissionParams
	reason := strings.TrimSpace(in.OverrideReason)
	required := customerDefault
	// Pengecualian hanya dicatat bila berbeda dari bawaan customer, selalu dengan alasan.
	if in.Required != nil && *in.Required != customerDefault {
		if reason == "" {
			return p, invalid("override_reason", problem.Required, "Write why this revision differs from the customer setting.")
		}
		if utf8.RuneCountInString(reason) > 500 {
			return p, invalid("override_reason", problem.TooLong, "The reason can be at most 500 characters.")
		}
		required = *in.Required
		p.RequiredOverride, p.OverrideReason = &required, &reason
	}
	pkg, remark := strings.TrimSpace(in.Package), strings.TrimSpace(in.Remark)
	switch {
	case utf8.RuneCountInString(pkg) > 500:
		return p, invalid("package", problem.TooLong, "The package can be at most 500 characters.")
	case utf8.RuneCountInString(remark) > 1000:
		return p, invalid("remark", problem.TooLong, "The remark can be at most 1000 characters.")
	case in.Result != "" && in.Result != ResultWaiting && in.Result != ResultApproved && in.Result != ResultRejected:
		return p, invalid("result", problem.InvalidFormat, "Choose waiting, approved, or rejected.")
	case !required && (in.SubmittedOn != nil || in.Result != ""):
		return p, invalid("submitted_on", problem.NotAllowed, "This revision is not submitted to the customer. Clear the date and the result, or mark it as required.")
	case in.Result != "" && in.SubmittedOn == nil:
		return p, invalid("submitted_on", problem.Required, "Fill in the date sent to the customer first.")
	case in.ResultOn != nil && in.Result != ResultApproved && in.Result != ResultRejected:
		return p, invalid("result_on", problem.NotAllowed, "A result date needs the result approved or rejected.")
	case in.ResultOn != nil && in.SubmittedOn != nil && in.ResultOn.Before(*in.SubmittedOn):
		return p, invalid("result_on", problem.NotAllowed, "The result date cannot be before the date sent.")
	}
	p.SubmittedOn, p.ResultOn = pgDate(in.SubmittedOn), pgDate(in.ResultOn)
	p.Package, p.Remark, p.Result = optional(pkg), optional(remark), optional(in.Result)
	return p, nil
}

// SetCustomerDefault mengatur bawaan "perlu dikirim ke customer" (E-11).
func (s *Service) SetCustomerDefault(ctx context.Context, actor access.Actor, customerID string, required bool) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: customerID}); err != nil {
		return err
	}
	if !db.IsUUID(customerID) {
		return problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.GetCustomer(ctx, customerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if c.SubmissionRequired == required {
			return nil
		}
		if err := q.SetCustomerSubmissionRequired(ctx, db.SetCustomerSubmissionRequiredParams{ID: customerID, SubmissionRequired: required}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSubmissionDefaultChanged, TargetType: audit.TargetCustomer, TargetID: customerID,
			Details: map[string]string{"submission_required": audit.Change(strconv.FormatBool(c.SubmissionRequired), strconv.FormatBool(required))},
		})
	})
	return keep(err, "mengatur bawaan submission customer")
}

// scanTypes adalah jenis scan bukti yang diterima.
var scanTypes = map[string]string{".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg"}

// UploadScan menyimpan scan bukti persetujuan customer untuk satu revisi (E-7).
func (s *Service) UploadScan(ctx context.Context, actor access.Actor, documentID string, revision int, fileName string, r io.Reader) (Control, error) {
	customerID, err := s.customerOf(ctx, documentID)
	if err != nil {
		return Control{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.RecordRelease, access.Resource{CustomerID: customerID}); err != nil {
		return Control{}, err
	}
	name := filepath.Base(strings.TrimSpace(strings.ReplaceAll(fileName, "\\", "/")))
	contentType, ok := scanTypes[strings.ToLower(filepath.Ext(name))]
	if name == "" || name == "." || !ok {
		return Control{}, invalid("file", problem.NotAllowed, "Upload the customer approval as PDF, PNG, or JPEG.")
	}
	if s.store == nil {
		return Control{}, errors.New("penyimpanan scan belum dipasang")
	}
	sum, size, err := s.store.PutScanned(ctx, r, MaxScanBytes)
	switch {
	case errors.Is(err, storage.ErrEmpty):
		return Control{}, invalid("file", problem.Required, "The file is empty.")
	case errors.Is(err, storage.ErrTooLarge):
		return Control{}, invalid("file", problem.TooLong, "A scan can be at most 30 MB.")
	case errors.Is(err, storage.ErrInfected):
		return Control{}, invalid("file", problem.NotAllowed, "The antivirus scanner found malware in this file. It was not stored.")
	case errors.Is(err, storage.ErrScanUnavailable):
		return Control{}, invalid("file", problem.NotAllowed, "The antivirus scanner cannot be reached, so the file was not stored. Try again or tell the admin.")
	case err != nil:
		return Control{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		rev, err := q.RevisionOfDocument(ctx, db.RevisionOfDocumentParams{DocumentID: documentID, Revision: int16(revision)})
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := q.LockSubmission(ctx, db.LockSubmissionParams{RevisionID: rev.ID, DocumentID: documentID, UpdatedByPersonID: actor.PersonID}); err != nil {
			return err
		}
		if err := q.SetSubmissionScan(ctx, db.SetSubmissionScanParams{
			RevisionID: rev.ID, ResultScanName: &name, ResultScanType: &contentType, ResultScanSize: &size, ResultScanSha256: &sum, UpdatedByPersonID: actor.PersonID,
		}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSubmissionScanUploaded, TargetType: audit.TargetDocument, TargetID: documentID,
			Details: map[string]string{"revision": strconv.Itoa(revision), "file": name, "sha256": sum},
		})
	})
	if err != nil {
		return Control{}, keep(err, "menyimpan scan customer")
	}
	return s.load(ctx, documentID)
}

// Scan adalah scan bukti customer satu revisi.
type Scan struct {
	FileName    string
	ContentType string
	Size        int64
}

// OpenScan membuka scan bukti customer satu revisi (L-4).
func (s *Service) OpenScan(ctx context.Context, actor access.Actor, documentID string, revision int) (Scan, *os.File, error) {
	if _, err := s.Get(ctx, actor, documentID); err != nil {
		return Scan{}, nil, err
	}
	row, err := s.q.GetSubmissionScan(ctx, db.GetSubmissionScanParams{DocumentID: documentID, Revision: int16(revision)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Scan{}, nil, problem.ErrNotFound
	}
	if err != nil {
		return Scan{}, nil, err
	}
	if s.store == nil {
		return Scan{}, nil, errors.New("penyimpanan scan belum dipasang")
	}
	f, err := s.store.Open(deref(row.ResultScanSha256))
	if err != nil {
		return Scan{}, nil, err
	}
	sc := Scan{FileName: deref(row.ResultScanName), ContentType: deref(row.ResultScanType)}
	if row.ResultScanSize != nil {
		sc.Size = *row.ResultScanSize
	}
	return sc, f, nil
}

func dateOf(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, clock.Zone)
	return &t
}

func pgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
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
