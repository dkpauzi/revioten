package change

// Entitas perubahan (M5.1, R-CHG-01, D-113, UC-40). Satu perubahan milik satu
// customer dan punya dua isian sumber (D-113): mekanisme (manual, yokoten,
// propagasi template, SQAM) dan pemicu (sumber perubahan biasa, misalnya
// customer claim). Lalu jenis perubahan, kategori 4M opsional, alasan,
// rujukan sumber, dan daftar dokumen terdampak. Verifikasi menyusul di M5.3,
// lampiran di M5.2.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
)

// Mekanisme manual: perubahan yang diajukan orang lewat layar.
const MechanismManual = "manual"

// Status perubahan: diajukan, lalu diterima atau ditolak verifikator (M5.3).
const (
	StatusSubmitted = "submitted"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
)

// Batas isian perubahan, sama dengan CHECK di migrasi 00030.
const (
	maxReason          = 2000
	maxSourceReference = 200
)

// Ref adalah rujukan bernama ke master, dengan key bila ada.
type Ref struct {
	ID   string
	Key  string
	Name string
}

// Change adalah satu perubahan beserta dokumen terdampaknya.
type Change struct {
	ID              string
	Ref             string // nomor rujukan, misalnya CHG-0001 (ASUMSI E-98)
	CustomerID      string
	CustomerCode    string
	Mechanism       string
	Trigger         *Ref // pemicu; boleh kosong untuk mekanisme sistem
	ChangeType      Ref
	FourM           *Ref
	Reason          string
	SourceReference string
	Status          string
	RequestedBy     string             // nama orang pemohon
	RequestedByID   string             // ID orang pemohon
	Documents       []AffectedDocument // menurut jenis dokumen lalu nomor; tidak pernah nil
	Attachments     []Attachment       // lampiran aktif menurut waktu unggah; tidak pernah nil (M5.2)
	Verification    *Verification      // terisi sesudah diterima atau ditolak (M5.3)
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AffectedDocument adalah satu dokumen terdampak.
type AffectedDocument struct {
	ID           string
	Number       string
	DocumentType Ref
	Revision     int
	Status       string
	// M5.4: revisi yang berlaku, draft yang dihasilkan perubahan ini, dan
	// apakah perubahan masuk ke draft yang sudah terbuka (R-CHG-03).
	ReleasedRevision *int
	DraftRevision    *int
	Merged           bool
}

// NewChange adalah isian perubahan manual baru.
type NewChange struct {
	CustomerID      string
	TriggerID       string
	ChangeTypeID    string
	FourMCategoryID string // boleh kosong
	Reason          string
	SourceReference string // boleh kosong
	DocumentIDs     []string
}

// RequestService menjalankan operasi perubahan.
type RequestService struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
	store *storage.Store // lampiran (M5.2); nil berarti unggahan ditolak
}

// NewRequestService membuat layanan perubahan.
func NewRequestService(pool *pgxpool.Pool, authz access.Authorizer) *RequestService {
	return &RequestService{pool: pool, q: db.New(pool), authz: authz}
}

// List memberi perubahan satu customer, yang terbaru dulu (L-4).
func (s *RequestService) List(ctx context.Context, actor access.Actor, customerID string) ([]Change, error) {
	if !db.IsUUID(customerID) {
		return []Change{}, nil
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	ids, err := s.q.ListChangeIDs(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("membaca perubahan: %w", err)
	}
	return s.many(ctx, ids)
}

// ForDocument memberi perubahan yang menunjuk satu dokumen (L-4).
func (s *RequestService) ForDocument(ctx context.Context, actor access.Actor, documentID string) ([]Change, error) {
	if !db.IsUUID(documentID) {
		return nil, problem.ErrNotFound
	}
	doc, err := s.q.GetDocumentForChange(ctx, documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, problem.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: doc.CustomerID}); err != nil {
		return nil, err
	}
	ids, err := s.q.ListChangeIDsForDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	return s.many(ctx, ids)
}

// Get membaca satu perubahan (L-4 untuk customer-nya).
func (s *RequestService) Get(ctx context.Context, actor access.Actor, id string) (Change, error) {
	c, err := s.load(ctx, id)
	if err != nil {
		return Change{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: c.CustomerID}); err != nil {
		return Change{}, err
	}
	return c, nil
}

// Create mengajukan perubahan manual (E-3 untuk customer itu).
func (s *RequestService) Create(ctx context.Context, actor access.Actor, in NewChange) (Change, error) {
	if !db.IsUUID(in.CustomerID) {
		return Change{}, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Change{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		fields, err := checkFields(ctx, q, in)
		if err != nil {
			return err
		}
		docs, err := checkDocuments(ctx, q, in.CustomerID, in.DocumentIDs)
		if err != nil {
			return err
		}
		id, err = q.CreateChange(ctx, db.CreateChangeParams{
			CustomerID: in.CustomerID, Mechanism: MechanismManual, TriggerSourceID: &in.TriggerID, ChangeTypeID: in.ChangeTypeID,
			FourMCategoryID: optional(in.FourMCategoryID), Reason: fields.reason, SourceReference: optional(fields.reference),
			RequestedByPersonID: actor.PersonID, RequestedByAccountID: actor.AccountID,
		})
		if err != nil {
			return err
		}
		for _, d := range docs {
			if err := q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: id, DocumentID: d.ID}); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionChangeCreated, TargetType: audit.TargetChange, TargetID: id,
			Details: map[string]string{"reason": fields.reason, "documents": strconv.Itoa(len(docs))},
		})
	})
	if err != nil {
		return Change{}, result(err, "mengajukan perubahan")
	}
	return s.load(ctx, id)
}

// AddDocument menambah satu dokumen terdampak (E-3).
func (s *RequestService) AddDocument(ctx context.Context, actor access.Actor, id, documentID string) (Change, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, c Change) (string, map[string]string, error) {
		docs, err := checkDocuments(ctx, q, c.CustomerID, []string{documentID})
		if err != nil {
			return "", nil, err
		}
		for _, d := range c.Documents {
			if d.ID == documentID {
				return "", nil, &problem.ConflictError{Problems: []problem.Problem{{Field: "document_id", Code: problem.Taken, Message: "This document is already affected by the change."}}}
			}
		}
		err = q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: c.ID, DocumentID: documentID})
		return audit.ActionChangeDocumentAdded, map[string]string{"document": docs[0].Number}, err
	})
}

// RemoveDocument melepas satu dokumen terdampak (E-3). Dokumen terakhir
// tidak bisa dilepas, karena perubahan selalu menunjuk dokumen.
func (s *RequestService) RemoveDocument(ctx context.Context, actor access.Actor, id, documentID string) (Change, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, c Change) (string, map[string]string, error) {
		number := ""
		for _, d := range c.Documents {
			if d.ID == documentID {
				number = d.Number
			}
		}
		if number == "" {
			return "", nil, problem.ErrNotFound
		}
		if len(c.Documents) == 1 {
			return "", nil, problems(invalid("document_id", problem.NotAllowed, "A change always affects at least one document."))
		}
		if _, err := q.RemoveChangeDocument(ctx, db.RemoveChangeDocumentParams{ChangeID: c.ID, DocumentID: documentID}); err != nil {
			return "", nil, err
		}
		return audit.ActionChangeDocumentRemoved, map[string]string{"document": number}, nil
	})
}

// edit menjalankan satu perubahan pada perubahan yang masih diajukan, di
// dalam transaksi: hak E-3 untuk customer-nya, dikunci, lalu entri audit.
func (s *RequestService) edit(ctx context.Context, actor access.Actor, id string,
	change func(q *db.Queries, c Change) (string, map[string]string, error)) (Change, error) {
	c, err := s.load(ctx, id)
	if err != nil {
		return Change{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: c.CustomerID}); err != nil {
		return Change{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockChange(ctx, id); err != nil {
			return err
		}
		c, err := getChange(ctx, q, id)
		if err != nil {
			return err
		}
		if c.Status != StatusSubmitted {
			return problems(invalid("status", problem.NotAllowed, "Only a submitted change can be changed."))
		}
		action, details, err := change(q, c)
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetChange, TargetID: id, Details: details})
	})
	if err != nil {
		return Change{}, result(err, "mengubah perubahan")
	}
	return s.load(ctx, id)
}

func (s *RequestService) load(ctx context.Context, id string) (Change, error) {
	if !db.IsUUID(id) {
		return Change{}, problem.ErrNotFound
	}
	return getChange(ctx, s.q, id)
}

func (s *RequestService) many(ctx context.Context, ids []string) ([]Change, error) {
	out := make([]Change, 0, len(ids))
	for _, id := range ids {
		c, err := getChange(ctx, s.q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// getChange membaca satu perubahan beserta dokumen terdampaknya.
func getChange(ctx context.Context, q *db.Queries, id string) (Change, error) {
	r, err := q.GetChange(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Change{}, problem.ErrNotFound
	}
	if err != nil {
		return Change{}, fmt.Errorf("membaca perubahan: %w", err)
	}
	c := Change{
		ID: r.ID, Ref: changeRef(r.Seq), CustomerID: r.CustomerID, CustomerCode: r.CustomerCode, Mechanism: r.Mechanism,
		ChangeType: Ref{ID: r.ChangeTypeID, Key: r.ChangeTypeKey, Name: r.ChangeTypeName},
		Reason:     r.Reason, SourceReference: deref(r.SourceReference), Status: r.Status,
		RequestedBy: r.RequestedBy, RequestedByID: r.RequestedByPersonID,
		Documents: []AffectedDocument{}, CreatedAt: r.CreatedAt.UTC(), UpdatedAt: r.UpdatedAt.UTC(),
	}
	if r.TriggerSourceID != nil {
		c.Trigger = &Ref{ID: *r.TriggerSourceID, Name: deref(r.TriggerName)}
	}
	if r.FourMCategoryID != nil {
		c.FourM = &Ref{ID: *r.FourMCategoryID, Key: deref(r.FourMKey), Name: deref(r.FourMName)}
	}
	if r.VerifiedByPersonID != nil && r.VerifiedAt != nil {
		c.Verification = &Verification{
			VerifiedBy: deref(r.VerifiedBy), VerifiedByID: *r.VerifiedByPersonID, VerifiedAt: r.VerifiedAt.UTC(), Note: deref(r.VerificationNote),
		}
	}
	if c.Attachments, err = attachments(ctx, q, id); err != nil {
		return Change{}, err
	}
	docs, err := q.ListChangeDocuments(ctx, id)
	if err != nil {
		return Change{}, err
	}
	for _, d := range docs {
		c.Documents = append(c.Documents, AffectedDocument{
			ID: d.ID, Number: d.Number, Revision: int(d.Revision), Status: d.Status, Merged: d.Merged,
			ReleasedRevision: intPtr(d.ReleasedRevision), DraftRevision: intPtr(d.DraftRevision),
			DocumentType: Ref{ID: d.DocumentTypeID, Key: deref(d.DocumentTypeKey), Name: d.DocumentTypeName},
		})
	}
	return c, nil
}

// changeRef menulis nomor rujukan perubahan, misalnya CHG-0001 (ASUMSI E-98).
func changeRef(seq *int64) string {
	if seq == nil {
		return ""
	}
	return fmt.Sprintf("CHG-%04d", *seq)
}

// changeFields adalah isian teks perubahan yang sudah dirapikan.
type changeFields struct {
	reason, reference string
}

// checkFields memeriksa pemicu, jenis, kategori 4M, alasan, dan rujukan.
func checkFields(ctx context.Context, q *db.Queries, in NewChange) (changeFields, error) {
	var out changeFields
	var list []*problem.Problem
	var p *problem.Problem
	out.reason, p = problem.Text("reason", in.Reason, true, maxReason)
	list = append(list, p)
	out.reference, p = problem.Text("source_reference", in.SourceReference, false, maxSourceReference)
	list = append(list, p)
	switch {
	case in.TriggerID == "":
		list = append(list, invalid("trigger_id", problem.Required, "Choose what triggered the change."))
	case !db.IsUUID(in.TriggerID):
		list = append(list, invalid("trigger_id", problem.InvalidFormat, "Choose an existing change source."))
	default:
		source, err := q.GetChangeSourceForChange(ctx, in.TriggerID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			list = append(list, invalid("trigger_id", problem.InvalidFormat, "Choose an existing change source."))
		case err != nil:
			return out, err
		case source.Mechanism != nil:
			list = append(list, invalid("trigger_id", problem.NotAllowed, "Yokoten, template propagation, and SQAM changes are started by the system, not chosen here."))
		case !source.Active:
			list = append(list, invalid("trigger_id", problem.NotAllowed, "This change source is no longer active."))
		}
	}
	exists := func(field, id string, check func(context.Context, string) (bool, error), message string) error {
		if id == "" {
			return nil
		}
		ok := db.IsUUID(id)
		if ok {
			var err error
			if ok, err = check(ctx, id); err != nil {
				return err
			}
		}
		if !ok {
			list = append(list, invalid(field, problem.InvalidFormat, message))
		}
		return nil
	}
	if in.ChangeTypeID == "" {
		list = append(list, invalid("change_type_id", problem.Required, "Choose the type of change."))
	} else if err := exists("change_type_id", in.ChangeTypeID, q.ChangeTypeExists, "Choose an existing type of change."); err != nil {
		return out, err
	}
	if err := exists("four_m_category_id", in.FourMCategoryID, q.FourMCategoryExists, "Choose an existing 4M category."); err != nil {
		return out, err
	}
	return out, problems(list...)
}

// affected adalah satu dokumen terdampak yang sudah diperiksa.
type affected struct {
	ID, Number string
}

// checkDocuments memeriksa dokumen terdampak: ada, milik customer perubahan,
// dan tidak ganda.
func checkDocuments(ctx context.Context, q *db.Queries, customerID string, ids []string) ([]affected, error) {
	if len(ids) == 0 {
		return nil, problems(invalid("document_ids", problem.Required, "Choose at least one affected document."))
	}
	var out []affected
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		bad := problems(invalid("document_ids", problem.InvalidFormat, "Choose documents of the customer of this change."))
		if !db.IsUUID(id) {
			return nil, bad
		}
		d, err := q.GetDocumentForChange(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, bad
		}
		if err != nil {
			return nil, err
		}
		if d.CustomerID != customerID {
			return nil, bad
		}
		out = append(out, affected{ID: d.ID, Number: d.Number})
	}
	return out, nil
}

// invalid membuat satu masalah input.
func invalid(field, code, message string) *problem.Problem {
	return &problem.Problem{Field: field, Code: code, Message: message}
}

// problems menggabungkan masalah input; nil bila tidak ada.
func problems(ps ...*problem.Problem) error {
	var list []problem.Problem
	for _, p := range ps {
		if p != nil {
			list = append(list, *p)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return &problem.ValidationError{Problems: list}
}

// optional mengubah teks kosong menjadi nil untuk kolom yang boleh kosong.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
