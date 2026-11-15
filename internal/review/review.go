// Package review mengurus review berkala PFD, PFMEA, dan Control Plan
// (D-138, D-196). Satu set dokumen jatuh tempo bila sejak rilis terakhir atau
// review terakhir yang terkonfirmasi sudah lewat lama satu siklus (bawaan 36
// bulan). Review punya dua hasil: dengan perubahan (permintaan perubahan biasa
// dengan pemicu "Periodic review") atau tanpa perubahan (catatan review di
// sini, dikonfirmasi orang kedua). Rancangan: docs/rencana/review-berkala-3-tahun.md.
package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Status jatuh tempo satu set.
const (
	StatusOK      = "ok"       // belum jatuh tempo
	StatusDueSoon = "due_soon" // jatuh tempo dalam masa peringatan (kuning)
	StatusOverdue = "overdue"  // lewat jatuh tempo (merah)
)

// Jawaban satu butir daftar periksa. "Perlu perubahan" tidak ada di sini:
// butir yang perlu diubah berarti review dengan perubahan.
const (
	AnswerOK            = "ok"
	AnswerNotApplicable = "not_applicable"
)

// ChecklistItems adalah butir daftar periksa review, menurut rancangan
// bagian 4.2. ASUMSI E-86: daftarnya tetap di kode, belum diatur admin.
var ChecklistItems = []string{
	"process_flow",            // proses di PFD sama dengan proses di line
	"claims_findings",         // komplain, temuan audit, atau NG internal sudah masuk PFMEA
	"occurrence_detection",    // nilai Occurrence dan Detection sesuai data terbaru
	"special_characteristics", // karakteristik khusus sesuai drawing dan spesifikasi terbaru
	"control_methods",         // metode kontrol, frekuensi, dan sampel CP masih dijalankan
	"references",              // alat ukur dan dokumen rujukan masih berlaku
	"reaction_plan",           // reaction plan masih sesuai prosedur
	"pfmea_cp_sync",           // PFMEA dan CP masih sinkron
}

// SetDocument adalah satu dokumen di set beserta revisi yang berlaku.
type SetDocument struct {
	ID       string
	Number   string
	TypeKey  string
	Revision int
}

// Set adalah satu set dokumen dan status review-nya.
type Set struct {
	CustomerID   string
	CustomerCode string // kode customer di nomor
	Sequence     int
	Documents    []SetDocument
	Models       []string
	LastRelease  time.Time  // tanggal rilis terakhir di set
	LastReview   *time.Time // review tanpa perubahan terakhir yang terkonfirmasi
	DueOn        time.Time
	Status       string
	DaysLeft     int    // negatif bila lewat jatuh tempo
	OpenReviewID string // catatan yang menunggu konfirmasi; "" bila tidak ada
	DraftOpen    bool   // salah satu dokumen sedang punya draft revisi
}

// Overview adalah daftar set beserta pengaturannya.
type Overview struct {
	PeriodMonths int
	WarningDays  int
	DueSoon      int
	Overdue      int
	Sets         []Set // tidak pernah nil
}

// Filter memilih set yang dibaca.
type Filter struct {
	CustomerID string // "" untuk semua customer
	DocumentID string // "" untuk semua dokumen; selain itu hanya set dokumen ini
	// All juga memberi set yang belum jatuh tempo.
	All bool
}

// Answer adalah jawaban satu butir daftar periksa.
type Answer struct {
	Item   string `json:"item"`
	Answer string `json:"answer"`
	Note   string `json:"note,omitempty"`
}

// ReviewedRevision adalah satu revisi yang direview, dibekukan di catatan.
type ReviewedRevision struct {
	DocumentID string `json:"document_id"`
	Number     string `json:"number"`
	TypeKey    string `json:"type_key"`
	Revision   int    `json:"revision"`
	SHA256     string `json:"sha256"`
}

// Review adalah satu catatan review tanpa perubahan.
type Review struct {
	ID           string
	Ref          string // PRV-0001
	CustomerID   string
	CustomerCode string
	Sequence     int
	ReviewedOn   time.Time
	Participants string
	Conclusion   string
	Checklist    []Answer
	Revisions    []ReviewedRevision
	Status       string // submitted atau confirmed
	RecordedBy   string
	RecordedAt   time.Time
	ConfirmedBy  string
	ConfirmedAt  *time.Time
}

// NewReview adalah isian catatan review tanpa perubahan.
type NewReview struct {
	DocumentID   string // salah satu dokumen di set
	ReviewedOn   string // YYYY-MM-DD
	Participants string
	Conclusion   string
	Checklist    []Answer
}

// Service mengurus review berkala.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
	// now bisa diganti di test.
	now func() time.Time
}

// NewService membuat Service.
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, now: time.Now}
}

// WithNow memberi salinan Service dengan jam lain. Hanya untuk test.
func (s *Service) WithNow(now func() time.Time) *Service {
	c := *s
	c.now = now
	return &c
}

// Sets memberi set dokumen dan status review-nya (L-2). Tanpa Filter.All,
// hanya set yang akan atau sudah lewat jatuh tempo.
func (s *Service) Sets(ctx context.Context, actor access.Actor, f Filter) (Overview, error) {
	if f.CustomerID != "" && !db.IsUUID(f.CustomerID) {
		return Overview{}, invalid("customer_id", problem.InvalidFormat, "Choose an existing customer.")
	}
	if f.DocumentID != "" && !db.IsUUID(f.DocumentID) {
		return Overview{}, problem.ErrNotFound
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: f.CustomerID}); err != nil {
		return Overview{}, err
	}
	return Compute(ctx, s.q, f, s.now())
}

// Compute menghitung status review semua set yang terlihat di ctx. Dipakai
// juga pekerja email (internal/notify) tanpa pemeriksaan izin.
func Compute(ctx context.Context, q *db.Queries, f Filter, now time.Time) (Overview, error) {
	settings, err := q.GetReviewSettings(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca pengaturan review: %w", err)
	}
	out := Overview{PeriodMonths: int(settings.ReviewPeriodMonths), WarningDays: int(settings.ReviewWarningDays), Sets: []Set{}}
	rows, err := q.ReviewSets(ctx, db.ReviewSetsParams{CustomerID: optional(f.CustomerID), DocumentID: optional(f.DocumentID)})
	if err != nil {
		return Overview{}, fmt.Errorf("membaca set dokumen: %w", err)
	}
	today := clock.Date(now)
	for _, r := range rows {
		if !r.LastRelease.Valid {
			continue
		}
		set := Set{
			CustomerID: r.CustomerID, CustomerCode: r.CustomerCode, Sequence: int(r.Sequence), Models: r.Models,
			LastRelease: dateOf(r.LastRelease.Time), OpenReviewID: r.OpenReviewID, DraftOpen: r.DraftOpen,
		}
		for i := range r.DocumentIds {
			set.Documents = append(set.Documents, SetDocument{ID: r.DocumentIds[i], Number: r.Numbers[i], TypeKey: r.TypeKeys[i], Revision: int(r.Revisions[i])})
		}
		anchor := set.LastRelease
		if r.LastReview.Valid {
			d := dateOf(r.LastReview.Time)
			set.LastReview = &d
			if d.After(anchor) {
				anchor = d
			}
		}
		set.DueOn = anchor.AddDate(0, out.PeriodMonths, 0)
		set.DaysLeft = clock.Days(today, set.DueOn)
		switch {
		case set.DaysLeft < 0:
			set.Status = StatusOverdue
			out.Overdue++
		case set.DaysLeft <= out.WarningDays:
			set.Status = StatusDueSoon
			out.DueSoon++
		default:
			set.Status = StatusOK
		}
		if set.Status != StatusOK || f.All {
			out.Sets = append(out.Sets, set)
		}
	}
	// Yang paling mendesak di atas.
	slices.SortStableFunc(out.Sets, func(a, b Set) int { return a.DaysLeft - b.DaysLeft })
	return out, nil
}

// Record mencatat review tanpa perubahan untuk set dokumen in.DocumentID
// (E-3). Revisi yang berlaku dibekukan bersama SHA-256-nya. Catatan baru
// memulai lagi jam review setelah dikonfirmasi orang kedua.
func (s *Service) Record(ctx context.Context, actor access.Actor, in NewReview) (Review, error) {
	if !db.IsUUID(in.DocumentID) {
		return Review{}, problem.ErrNotFound
	}
	set, err := s.q.SetOfDocument(ctx, in.DocumentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, problem.ErrNotFound
	}
	if err != nil {
		return Review{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: set.CustomerID}); err != nil {
		return Review{}, err
	}

	var problems []problem.Problem
	reviewedOn, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(in.ReviewedOn), clock.Zone)
	switch {
	case strings.TrimSpace(in.ReviewedOn) == "":
		problems = append(problems, problem.Problem{Field: "reviewed_on", Code: problem.Required, Message: "Enter the review date."})
	case err != nil:
		problems = append(problems, problem.Problem{Field: "reviewed_on", Code: problem.InvalidFormat, Message: "Enter the date as YYYY-MM-DD."})
	case reviewedOn.After(clock.Date(s.now())):
		problems = append(problems, problem.Problem{Field: "reviewed_on", Code: problem.NotAllowed, Message: "The review date cannot be in the future."})
	}
	participants, p := problem.Text("participants", in.Participants, true, 500)
	if p != nil {
		problems = append(problems, *p)
	}
	conclusion, p := problem.Text("conclusion", in.Conclusion, true, 2000)
	if p != nil {
		problems = append(problems, *p)
	}
	answers, p := checkChecklist(in.Checklist)
	if p != nil {
		problems = append(problems, *p)
	}
	if len(problems) > 0 {
		return Review{}, &problem.ValidationError{Problems: problems}
	}

	revisions, err := s.q.SetReleasedRevisions(ctx, db.SetReleasedRevisionsParams{CustomerCode: set.CustomerCode, Sequence: set.Sequence})
	if err != nil {
		return Review{}, err
	}
	if len(revisions) == 0 {
		return Review{}, invalid("document_id", problem.NotAllowed, "This document set has no released revision to review.")
	}
	frozen := make([]ReviewedRevision, 0, len(revisions))
	for _, r := range revisions {
		frozen = append(frozen, ReviewedRevision{DocumentID: r.DocumentID, Number: r.Number, TypeKey: deref(r.TypeKey), Revision: int(r.Revision), SHA256: deref(r.SnapshotSha256)})
	}
	checklistJSON, err := json.Marshal(answers)
	if err != nil {
		return Review{}, err
	}
	revisionsJSON, err := json.Marshal(frozen)
	if err != nil {
		return Review{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreatePeriodicReview(ctx, db.CreatePeriodicReviewParams{
			CustomerID: set.CustomerID, CustomerCode: set.CustomerCode, Sequence: set.Sequence,
			ReviewedOn: pgtype.Date{Time: reviewedOn, Valid: true}, Participants: participants, Conclusion: conclusion,
			Checklist: checklistJSON, Revisions: revisionsJSON, RecordedByPersonID: actor.PersonID, RecordedByAccountID: actor.AccountID,
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionPeriodicReviewRecorded, TargetType: "periodic_review", TargetID: id,
			Details: map[string]string{"set": setName(set.CustomerCode, int(set.Sequence)), "reviewed_on": reviewedOn.Format(time.DateOnly)}})
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "periodic_reviews_one_open" {
		return Review{}, &problem.ConflictError{Problems: []problem.Problem{{Field: "document_id", Code: problem.Taken,
			Message: "A review of this document set is already waiting for confirmation."}}}
	}
	if err != nil {
		return Review{}, fmt.Errorf("mencatat review: %w", err)
	}
	return s.get(ctx, id)
}

// Confirm mengonfirmasi catatan review (V-1). Pencatatnya tidak boleh
// mengonfirmasi sendiri (R-SEC-11), dan revisi yang berlaku harus masih sama
// dengan yang direview.
func (s *Service) Confirm(ctx context.Context, actor access.Actor, id string) (Review, error) {
	r, err := s.get(ctx, id)
	if err != nil {
		return Review{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.VerifyChanges, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Review{}, err
	}
	if r.Status != "submitted" {
		return Review{}, invalid("status", problem.NotAllowed, "This review is already confirmed.")
	}
	raw, err := s.q.GetPeriodicReview(ctx, id)
	if err != nil {
		return Review{}, err
	}
	if raw.RecordedByPersonID == actor.PersonID {
		return Review{}, invalid("confirmed_by", problem.NotAllowed, "The person who recorded the review cannot confirm it.")
	}
	current, err := s.q.SetReleasedRevisions(ctx, db.SetReleasedRevisionsParams{CustomerCode: r.CustomerCode, Sequence: int32(r.Sequence)})
	if err != nil {
		return Review{}, err
	}
	if !sameRevisions(r.Revisions, current) {
		return Review{}, invalid("revisions", problem.Changed, "A revision of this document set was released after the review. Record a new review.")
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.ConfirmPeriodicReview(ctx, db.ConfirmPeriodicReviewParams{ID: id, ConfirmedByPersonID: &actor.PersonID, ConfirmedByAccountID: &actor.AccountID})
		if err != nil {
			return err
		}
		if n == 0 {
			return invalid("status", problem.NotAllowed, "This review is already confirmed.")
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionPeriodicReviewConfirmed, TargetType: "periodic_review", TargetID: id,
			Details: map[string]string{"set": setName(r.CustomerCode, r.Sequence)}})
	})
	if err != nil {
		var invalidErr *problem.ValidationError
		if errors.As(err, &invalidErr) {
			return Review{}, err
		}
		return Review{}, fmt.Errorf("mengonfirmasi review: %w", err)
	}
	return s.get(ctx, id)
}

// Reviews memberi catatan review terbaru (L-2). openOnly hanya memberi yang
// menunggu konfirmasi.
func (s *Service) Reviews(ctx context.Context, actor access.Actor, customerID string, openOnly bool) ([]Review, error) {
	if customerID != "" && !db.IsUUID(customerID) {
		return nil, invalid("customer_id", problem.InvalidFormat, "Choose an existing customer.")
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPeriodicReviews(ctx, db.ListPeriodicReviewsParams{CustomerID: optional(customerID), OpenOnly: openOnly})
	if err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(rows))
	for _, r := range rows {
		item, err := toReview(db.GetPeriodicReviewRow(r))
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) get(ctx context.Context, id string) (Review, error) {
	if !db.IsUUID(id) {
		return Review{}, problem.ErrNotFound
	}
	r, err := s.q.GetPeriodicReview(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, problem.ErrNotFound
	}
	if err != nil {
		return Review{}, err
	}
	return toReview(r)
}

func toReview(r db.GetPeriodicReviewRow) (Review, error) {
	out := Review{
		ID: r.ID, CustomerID: r.CustomerID, CustomerCode: r.CustomerCode, Sequence: int(r.Sequence),
		ReviewedOn: dateOf(r.ReviewedOn.Time), Participants: r.Participants, Conclusion: r.Conclusion, Status: r.Status,
		RecordedBy: r.RecordedBy, RecordedAt: r.RecordedAt.UTC(), ConfirmedBy: deref(r.ConfirmedBy),
		Checklist: []Answer{}, Revisions: []ReviewedRevision{},
	}
	if r.Seq != nil {
		out.Ref = fmt.Sprintf("PRV-%04d", *r.Seq)
	}
	if r.ConfirmedAt != nil {
		at := r.ConfirmedAt.UTC()
		out.ConfirmedAt = &at
	}
	if err := json.Unmarshal(r.Checklist, &out.Checklist); err != nil {
		return Review{}, fmt.Errorf("membaca daftar periksa: %w", err)
	}
	if err := json.Unmarshal(r.Revisions, &out.Revisions); err != nil {
		return Review{}, fmt.Errorf("membaca revisi yang direview: %w", err)
	}
	return out, nil
}

// checkChecklist memastikan setiap butir dijawab tepat sekali dengan OK atau
// Not applicable, dan mengembalikannya menurut urutan butir.
func checkChecklist(in []Answer) ([]Answer, *problem.Problem) {
	byItem := map[string]Answer{}
	for _, a := range in {
		if !slices.Contains(ChecklistItems, a.Item) || (a.Answer != AnswerOK && a.Answer != AnswerNotApplicable) {
			return nil, &problem.Problem{Field: "checklist", Code: problem.InvalidFormat,
				Message: "Answer each checklist item with OK or Not applicable. An item that needs a change means a review with changes."}
		}
		note, p := problem.Text("checklist", a.Note, false, 500)
		if p != nil {
			return nil, p
		}
		a.Note = note
		byItem[a.Item] = a
	}
	out := make([]Answer, 0, len(ChecklistItems))
	for _, item := range ChecklistItems {
		a, ok := byItem[item]
		if !ok {
			return nil, &problem.Problem{Field: "checklist", Code: problem.Required, Message: "Answer every checklist item."}
		}
		out = append(out, a)
	}
	return out, nil
}

func sameRevisions(reviewed []ReviewedRevision, current []db.SetReleasedRevisionsRow) bool {
	if len(reviewed) != len(current) {
		return false
	}
	for i, c := range current {
		if reviewed[i].DocumentID != c.DocumentID || reviewed[i].Revision != int(c.Revision) {
			return false
		}
	}
	return true
}

// SetName memberi nama tampilan satu set, misalnya "AB · 0012".
func setName(code string, sequence int) string {
	return fmt.Sprintf("%s · %04d", strings.ToUpper(code), sequence)
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, clock.Zone)
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}
