package template

// Master dokumen terkait (D-130, D-156, R-DOC-11). WI, SOP, QEP, dan dokumen
// sejenis dicatat sekali dengan nomor, judul, level, revisi, dan tanggal
// revisinya. Setiap dokumen ditautkan ke proses beserta cakupannya: umum,
// customer, group model, atau model (aturan cakupan yang sama dengan poin
// template, D-129). Saat dokumen dirakit, kolom Relevant Document baris
// pertama setiap proses diisi dari daftar ini (internal/document).
//
// Yang mengubah: Pengelola template (E-6), sama dengan master proses (D-64),
// karena tautannya per proses. ASUMSI E-96.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// KeyRelevantDocument adalah key kolom Relevant Document (D-150). Kolom ini
// diisi dari master dokumen terkait dan tidak pernah disimpan di poin (D-130).
const KeyRelevantDocument = "relevant_document"

// Batas isian dokumen terkait, sama dengan CHECK di migrasi 00028.
const (
	maxRefTitle    = 200
	maxRefLevel    = 60
	maxRefRevision = 10
)

var refNumberPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,39}$`)

// RefDocument adalah satu dokumen terkait beserta tautannya ke proses.
type RefDocument struct {
	ID           string
	Number       string // huruf besar, misalnya SW-ZZ-103
	Title        string
	Level        string // misalnya "WI (Level 3)"; boleh kosong
	Revision     string // misalnya "19"; boleh kosong
	RevisionDate string // YYYY-MM-DD; boleh kosong
	Active       bool
	Links        []RefLink // menurut nama proses lalu cakupan; tidak pernah nil
	CreatedAt    time.Time
}

// RefLink adalah tautan satu dokumen ke satu proses dengan satu cakupan.
type RefLink struct {
	ID      string
	Process ProcessRef
	Scope   Scope
}

// NewRefDocument adalah isian dokumen terkait baru. Nomornya tetap sesudah dibuat.
type NewRefDocument struct {
	Number, Title, Level, Revision, RevisionDate string
}

// RefDocumentUpdate adalah isian perubahan. Judul wajib; Active nil berarti tetap.
type RefDocumentUpdate struct {
	Title, Level, Revision, RevisionDate string
	Active                               *bool
}

// NewRefLink adalah isian tautan baru. Cakupan kosong berarti umum.
type NewRefLink struct {
	ProcessID string
	Scope     ScopeInput
}

// RefDocService menjalankan operasi master dokumen terkait.
type RefDocService struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewRefDocService membuat layanan master dokumen terkait.
func NewRefDocService(pool *pgxpool.Pool, authz access.Authorizer) *RefDocService {
	return &RefDocService{pool: pool, q: db.New(pool), authz: authz}
}

// List memberi semua dokumen terkait menurut nomor (L-2).
func (s *RefDocService) List(ctx context.Context, actor access.Actor) ([]RefDocument, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListReferencedDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca dokumen terkait: %w", err)
	}
	links, err := refLinks(ctx, s.q)
	if err != nil {
		return nil, err
	}
	out := make([]RefDocument, 0, len(rows))
	for _, r := range rows {
		out = append(out, refDocument(db.GetReferencedDocumentRow(r), links[r.ID]))
	}
	return out, nil
}

// Create mencatat dokumen terkait baru (E-6).
func (s *RefDocService) Create(ctx context.Context, actor access.Actor, in NewRefDocument) (RefDocument, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return RefDocument{}, err
	}
	number := strings.ToUpper(strings.TrimSpace(in.Number))
	var numberProblem *problem.Problem
	switch {
	case number == "":
		numberProblem = &problem.Problem{Field: "number", Code: problem.Required, Message: "Enter the document number."}
	case !refNumberPattern.MatchString(number):
		numberProblem = &problem.Problem{Field: "number", Code: problem.InvalidFormat,
			Message: "Use letters, digits, and dashes only, for example SW-ZZ-103."}
	}
	fields, list := cleanRefFields(in.Title, in.Level, in.Revision, in.RevisionDate)
	if err := problems(append(list, numberProblem)...); err != nil {
		return RefDocument{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreateReferencedDocument(ctx, db.CreateReferencedDocumentParams{
			Number: number, Title: fields.title, Level: optional(fields.level), Revision: optional(fields.revision), RevisionDate: fields.date,
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionRefDocCreated, TargetType: audit.TargetReferencedDoc, TargetID: id,
			Details: map[string]string{"number": number, "title": fields.title},
		})
	})
	if db.IsUniqueViolation(err, "referenced_documents_number_key") {
		return RefDocument{}, &problem.ConflictError{Problems: []problem.Problem{{Field: "number", Code: problem.Taken,
			Message: "This document number is already in the list."}}}
	}
	if err != nil {
		return RefDocument{}, result(err, "mencatat dokumen terkait")
	}
	return s.get(ctx, id)
}

// Update mengganti judul, level, revisi, tanggal revisi, atau status aktif (E-6).
func (s *RefDocService) Update(ctx context.Context, actor access.Actor, id string, in RefDocumentUpdate) (RefDocument, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d RefDocument) (string, map[string]string, error) {
		fields, list := cleanRefFields(in.Title, in.Level, in.Revision, in.RevisionDate)
		if err := problems(list...); err != nil {
			return "", nil, err
		}
		active := d.Active
		if in.Active != nil {
			active = *in.Active
		}
		err := q.UpdateReferencedDocument(ctx, db.UpdateReferencedDocumentParams{
			ID: d.ID, Title: fields.title, Level: optional(fields.level), Revision: optional(fields.revision), RevisionDate: fields.date, Active: active,
		})
		details := map[string]string{}
		for key, change := range map[string][2]string{
			"title": {d.Title, fields.title}, "level": {d.Level, fields.level}, "revision": {d.Revision, fields.revision},
			"revision_date": {d.RevisionDate, fields.dateText},
		} {
			if change[0] != change[1] {
				details[key] = audit.Change(change[0], change[1])
			}
		}
		if active != d.Active {
			details["active"] = audit.Change(fmt.Sprint(d.Active), fmt.Sprint(active))
		}
		return audit.ActionRefDocUpdated, details, err
	})
}

// AddLink menautkan dokumen ke satu proses dengan satu cakupan (E-6).
func (s *RefDocService) AddLink(ctx context.Context, actor access.Actor, id string, in NewRefLink) (RefDocument, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d RefDocument) (string, map[string]string, error) {
		var processProblem *problem.Problem
		name := ""
		if !db.IsUUID(in.ProcessID) {
			processProblem = &problem.Problem{Field: "process_id", Code: problem.InvalidFormat, Message: "Choose an existing process."}
		} else {
			var err error
			name, err = q.GetProcessDisplayName(ctx, in.ProcessID)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				processProblem = &problem.Problem{Field: "process_id", Code: problem.InvalidFormat, Message: "Choose an existing process."}
			case err != nil:
				return "", nil, err
			}
		}
		target, scopeProblem, err := checkScope(ctx, q, in.Scope)
		if err != nil {
			return "", nil, err
		}
		if err := problems(processProblem, scopeProblem); err != nil {
			return "", nil, err
		}
		p := target.params("", "", nil)
		_, err = q.AddReferencedDocumentLink(ctx, db.AddReferencedDocumentLinkParams{
			DocumentID: d.ID, ProcessID: in.ProcessID, Scope: target.Scope.Level,
			CustomerID: p.CustomerID, ModelGroupID: p.ModelGroupID, ModelID: p.ModelID,
		})
		if db.IsUniqueViolation(err, "referenced_document_links_key") {
			return "", nil, &problem.ConflictError{Problems: []problem.Problem{{Field: "process_id", Code: problem.Taken,
				Message: "This document is already linked to this process for this scope."}}}
		}
		return audit.ActionRefDocLinked, map[string]string{"process": name, "scope": target.describe()}, err
	})
}

// RemoveLink melepas satu tautan (E-6). Dokumennya tetap.
func (s *RefDocService) RemoveLink(ctx context.Context, actor access.Actor, id, linkID string) (RefDocument, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d RefDocument) (string, map[string]string, error) {
		var link *RefLink
		for i := range d.Links {
			if d.Links[i].ID == linkID {
				link = &d.Links[i]
			}
		}
		if link == nil {
			return "", nil, problem.ErrNotFound
		}
		if _, err := q.DeleteReferencedDocumentLink(ctx, db.DeleteReferencedDocumentLinkParams{ID: linkID, DocumentID: d.ID}); err != nil {
			return "", nil, err
		}
		return audit.ActionRefDocUnlinked, map[string]string{
			"process": link.Process.Name, "scope": scopeTarget{Scope: link.Scope}.describe(),
		}, nil
	})
}

// ReferencedNumbers memberi nomor dokumen aktif yang tertaut ke satu proses
// dan berlaku untuk satu model, dijumlahkan dari semua cakupan dan diurutkan.
// groupID kosong untuk model tanpa group. Dipakai perakitan (internal/document).
func ReferencedNumbers(ctx context.Context, q *db.Queries, processID, modelID, customerID, groupID string) ([]string, error) {
	numbers, err := q.ReferencedNumbersForModel(ctx, db.ReferencedNumbersForModelParams{
		ProcessID: processID, CustomerID: &customerID, ModelGroupID: optional(groupID), ModelID: &modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("membaca dokumen terkait proses: %w", err)
	}
	return numbers, nil
}

// edit menjalankan satu perubahan dokumen terkait di dalam transaksi: hak
// E-6, dokumen dikunci, lalu entri audit bila ada yang berubah.
func (s *RefDocService) edit(ctx context.Context, actor access.Actor, id string,
	change func(q *db.Queries, d RefDocument) (string, map[string]string, error)) (RefDocument, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return RefDocument{}, err
	}
	if !db.IsUUID(id) {
		return RefDocument{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockReferencedDocument(ctx, id); err != nil {
			return err
		}
		d, err := getRefDocument(ctx, q, id)
		if err != nil {
			return err
		}
		action, details, err := change(q, d)
		if err != nil || len(details) == 0 {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetReferencedDoc, TargetID: id, Details: details})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return RefDocument{}, problem.ErrNotFound
	}
	if err != nil {
		var conflict *problem.ConflictError
		if errors.As(err, &conflict) {
			return RefDocument{}, err
		}
		return RefDocument{}, result(err, "mengubah dokumen terkait")
	}
	return s.get(ctx, id)
}

func (s *RefDocService) get(ctx context.Context, id string) (RefDocument, error) {
	d, err := getRefDocument(ctx, s.q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefDocument{}, problem.ErrNotFound
	}
	return d, err
}

func getRefDocument(ctx context.Context, q *db.Queries, id string) (RefDocument, error) {
	r, err := q.GetReferencedDocument(ctx, id)
	if err != nil {
		return RefDocument{}, err
	}
	links, err := refLinks(ctx, q)
	if err != nil {
		return RefDocument{}, err
	}
	return refDocument(r, links[r.ID]), nil
}

// refLinks membaca semua tautan, dikelompokkan per dokumen.
func refLinks(ctx context.Context, q *db.Queries) (map[string][]RefLink, error) {
	rows, err := q.ListReferencedDocumentLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca tautan dokumen terkait: %w", err)
	}
	out := map[string][]RefLink{}
	for _, r := range rows {
		out[r.DocumentID] = append(out[r.DocumentID], RefLink{
			ID: r.ID, Process: ProcessRef{ID: r.ProcessID, Name: r.ProcessName},
			Scope: scopeFromColumns(r.Scope, r.CustomerID, r.ModelGroupID, r.ModelID, r.CustomerCode, r.ModelGroupName, r.ModelName),
		})
	}
	return out, nil
}

func refDocument(r db.GetReferencedDocumentRow, links []RefLink) RefDocument {
	d := RefDocument{
		ID: r.ID, Number: r.Number, Title: r.Title, Level: derefText(r.Level), Revision: derefText(r.Revision),
		Active: r.Active, Links: orNone(links), CreatedAt: r.CreatedAt.UTC(),
	}
	if r.RevisionDate.Valid {
		d.RevisionDate = r.RevisionDate.Time.Format(time.DateOnly)
	}
	return d
}

// refFields adalah isian dokumen terkait yang sudah dirapikan.
type refFields struct {
	title, level, revision, dateText string
	date                             pgtype.Date
}

// cleanRefFields merapikan dan memeriksa judul, level, revisi, dan tanggal revisi.
func cleanRefFields(title, level, revision, date string) (refFields, []*problem.Problem) {
	var out refFields
	var list []*problem.Problem
	var p *problem.Problem
	out.title, p = problem.Text("title", title, true, maxRefTitle)
	list = append(list, p)
	out.level, p = problem.Text("level", level, false, maxRefLevel)
	list = append(list, p)
	out.revision, p = problem.Text("revision", revision, false, maxRefRevision)
	list = append(list, p)
	out.dateText = strings.TrimSpace(date)
	if out.dateText != "" {
		t, err := time.Parse(time.DateOnly, out.dateText)
		if err != nil {
			list = append(list, &problem.Problem{Field: "revision_date", Code: problem.InvalidFormat, Message: "Use a date as YYYY-MM-DD."})
		} else {
			out.date = pgtype.Date{Time: t, Valid: true}
		}
	}
	return out, list
}
