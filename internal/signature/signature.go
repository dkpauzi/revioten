// Package signature menjalankan tanda tangan digital (fase 3; D-198, D-203,
// D-204).
//
// Alurnya:
//
//  1. Pengirim (hak E-5) mengirim draft untuk tanda tangan. Isi draft
//     dibekukan sebagai snapshot dengan SHA-256, rute dibekukan, dan
//     penanda tangan setiap slot dipilih. Tanggal rilis = tanggal dikirim
//     (D-204). Bila pengirim sendiri yang Prepared, tanda tangannya dicatat
//     saat itu juga.
//  2. Penanda tangan (hak S-1) menandatangani berurutan menurut rute (D-49),
//     dengan mengetik ulang password. Tanda tangan tidak bisa diubah.
//  3. Penanda tangan berikutnya boleh menolak dengan alasan; pengirim boleh
//     menarik kembali. Keduanya membuka kunci draft.
//  4. Sesudah semua slot bertanda tangan, rilis dicatat tanpa scan
//     (document.Release).
//
// Aturan slot (D-203): Prepared ("Process. Eng") adalah pengirim, atau
// Project Leader tingkat Leader (jabatan Project Engineer) Project Group
// customer. Checked ("Engineer") hanya pemegang jabatan Project Engineer atau
// Project Leader Project Group itu; bila Project Engineer menjadi Prepared,
// Checked otomatis Project Leader. Slot Approved diisi pemegang jabatannya.
// Satu orang hanya satu slot (R-APR-12).
package signature

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Cara tanda tangan per customer (D-198).
const (
	ModeDigital = "digital"
	ModeWetOnly = "wet_only"
)

// Status permintaan.
const (
	StatusWaiting   = "waiting"
	StatusCompleted = "completed"
	StatusRejected  = "rejected"
	StatusWithdrawn = "withdrawn"
)

// Tahap slot rute.
const (
	StagePrepared = "prepared"
	StageChecked  = "checked"
	StageApproved = "approved"
)

// Kunci jabatan yang dipakai aturan D-203.
const (
	positionProjectEngineer = "project_engineer" // Project Leader tingkat Leader
	positionProjectLeader   = "project_leader"   // Project Leader tingkat Supervisor
)

// PasswordConfirmer memeriksa ulang password aktor (auth.Service).
type PasswordConfirmer interface {
	ConfirmPassword(ctx context.Context, actor access.Actor, password string) error
}

// Service menjalankan tanda tangan digital.
type Service struct {
	pool      *pgxpool.Pool
	q         *db.Queries
	authz     access.Authorizer
	passwords PasswordConfirmer
	now       func() time.Time
}

// NewService membuat Service.
func NewService(pool *pgxpool.Pool, authz access.Authorizer, passwords PasswordConfirmer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, passwords: passwords, now: time.Now}
}

// Candidate adalah satu orang yang boleh mengisi satu slot.
type Candidate struct {
	PersonID    string
	Name        string
	PositionKey string
	Position    string
}

// SlotPlan adalah satu slot rute beserta calon penanda tangannya.
type SlotPlan struct {
	Order      int
	Stage      string
	PrintTitle string
	Candidates []Candidate // tidak pernah nil
}

// Plan adalah bahan formulir "Send for signature".
type Plan struct {
	DocumentID string
	Number     string
	Revision   int
	Mode       string // digital atau wet_only
	RouteID    string // "" bila dokumen belum punya rute
	RouteName  string
	Slots      []SlotPlan // tidak pernah nil
}

// SendInput adalah isian "Send for signature".
type SendInput struct {
	// Signers: urutan slot ke ID orang. Slot dengan satu calon boleh kosong.
	Signers  map[int]string
	Note     string
	Password string
}

// Slot adalah satu slot permintaan beserta tanda tangannya.
type Slot struct {
	Order      int
	Stage      string
	PrintTitle string
	Position   string
	PersonID   string
	PersonName string
	SignedAt   *time.Time
}

// Request adalah satu permintaan tanda tangan.
type Request struct {
	ID            string
	DocumentID    string
	Number        string
	DocumentType  string
	CustomerID    string
	CustomerCode  string
	Revision      int
	RouteName     string
	ContentSHA256 string
	ReleaseDate   string // YYYY-MM-DD, tanggal dikirim (D-204)
	Note          string
	Status        string
	SentBy        string
	SentByID      string
	SentAt        time.Time
	ClosedAt      *time.Time
	ClosedBy      string
	RejectReason  string
	Slots         []Slot // tidak pernah nil
	// NextSlot: urutan slot yang menunggu; 0 bila tidak ada.
	NextSlot int
	// Changes: nomor perubahan di revisi ini, misalnya CHG-0001.
	Changes []string
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

// loadDocument membaca dokumen lewat query q (RLS ikut menyaring).
func loadDocument(ctx context.Context, q *db.Queries, id string) (document.Document, error) {
	d, err := document.Load(ctx, q, id)
	if err != nil {
		return document.Document{}, err
	}
	return d, nil
}

// SetMode mengatur cara tanda tangan satu customer (A-2).
func (s *Service) SetMode(ctx context.Context, actor access.Actor, customerID, mode string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: customerID}); err != nil {
		return err
	}
	if mode != ModeDigital && mode != ModeWetOnly {
		return invalid("mode", problem.InvalidFormat, "Choose digital or wet only.")
	}
	if !db.IsUUID(customerID) {
		return problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		before, err := q.CustomerSignatureMode(ctx, customerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if before == mode {
			return nil
		}
		if err := q.SetCustomerSignatureMode(ctx, db.SetCustomerSignatureModeParams{ID: customerID, SignatureMode: mode}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSignatureModeSet, TargetType: audit.TargetCustomer, TargetID: customerID,
			Details: map[string]string{"before": before, "after": mode},
		})
	})
	return keep(err, "mengatur cara tanda tangan")
}

// Plan memberi bahan formulir "Send for signature" untuk satu dokumen (E-5).
func (s *Service) Plan(ctx context.Context, actor access.Actor, documentID string) (Plan, error) {
	if !db.IsUUID(documentID) {
		return Plan{}, problem.ErrNotFound
	}
	d, err := loadDocument(ctx, s.q, documentID)
	if err != nil {
		return Plan{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.PrintForSignature, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Plan{}, err
	}
	return s.plan(ctx, s.q, actor, d)
}

// plan menyusun slot dan calon penanda tangan menurut aturan D-203.
func (s *Service) plan(ctx context.Context, q *db.Queries, actor access.Actor, d document.Document) (Plan, error) {
	mode, err := q.CustomerSignatureMode(ctx, d.CustomerID)
	if err != nil {
		return Plan{}, fmt.Errorf("membaca cara tanda tangan: %w", err)
	}
	p := Plan{DocumentID: d.ID, Number: d.Number, Revision: d.Revision, Mode: mode, Slots: []SlotPlan{}}
	if d.Route == nil {
		return p, nil
	}
	p.RouteID, p.RouteName = d.Route.ID, d.Route.Name
	signers, err := q.PeopleWithPermission(ctx, db.PeopleWithPermissionParams{Permission: string(access.SignDigitally), CustomerID: &d.CustomerID})
	if err != nil {
		return Plan{}, fmt.Errorf("membaca pemegang hak tanda tangan: %w", err)
	}
	// Orang dengan hak S-1 yang tidak dibatasi (peran bercakupan semua
	// customer) boleh menandatangani slot per Project Group untuk Project
	// Group lain juga, misalnya Project Leader GR5 yang membantu GR1 (D-206).
	unrestricted, err := q.PeopleWithPermission(ctx, db.PeopleWithPermissionParams{Permission: string(access.SignDigitally)})
	if err != nil {
		return Plan{}, fmt.Errorf("membaca pemegang hak tanda tangan: %w", err)
	}
	rows, err := q.RouteSlotCandidates(ctx, db.RouteSlotCandidatesParams{RouteID: d.Route.ID, CustomerID: d.CustomerID})
	if err != nil {
		return Plan{}, fmt.Errorf("membaca slot rute: %w", err)
	}
	for _, r := range rows {
		if len(p.Slots) == 0 || p.Slots[len(p.Slots)-1].Order != int(r.SlotOrder) {
			p.Slots = append(p.Slots, SlotPlan{Order: int(r.SlotOrder), Stage: r.Stage, PrintTitle: r.PrintTitle, Candidates: []Candidate{}})
		}
		if r.PersonID == nil || !slices.Contains(signers, *r.PersonID) {
			continue
		}
		if !r.OwnGroup && !slices.Contains(unrestricted, *r.PersonID) {
			continue // pemegang jabatan Project Group lain yang dibatasi ke Project Group-nya
		}
		c := Candidate{PersonID: *r.PersonID, Name: deref(r.PersonName), PositionKey: deref(r.PositionKey), Position: deref(r.PositionName)}
		if r.GroupCode != nil {
			c.Position += " " + *r.GroupCode
		}
		slot := &p.Slots[len(p.Slots)-1]
		switch r.Stage {
		case StagePrepared:
			// Prepared: pengirim, atau Project Leader tingkat Leader (D-203).
			if c.PersonID != actor.PersonID && c.PositionKey != positionProjectEngineer {
				continue
			}
		case StageChecked:
			// Checked: hanya Project Leader tingkat Leader atau Supervisor (D-203).
			if c.PositionKey != positionProjectEngineer && c.PositionKey != positionProjectLeader {
				continue
			}
		}
		if slices.ContainsFunc(slot.Candidates, func(x Candidate) bool { return x.PersonID == c.PersonID }) {
			continue // satu orang bisa memegang dua jabatan yang diizinkan slot itu
		}
		slot.Candidates = append(slot.Candidates, c)
	}
	return p, nil
}

// chooseSigners memeriksa pilihan penanda tangan terhadap rencana.
func chooseSigners(p Plan, chosen map[int]string) ([]Candidate, error) {
	if len(p.Slots) == 0 {
		return nil, invalid("route", problem.Required, "Choose the approval route of the document before sending it for signature.")
	}
	out := make([]Candidate, 0, len(p.Slots))
	var preparedKey string
	for _, slot := range p.Slots {
		person := chosen[slot.Order]
		if person == "" && len(slot.Candidates) == 1 {
			person = slot.Candidates[0].PersonID
		}
		field := "signers." + strconv.Itoa(slot.Order)
		if len(slot.Candidates) == 0 {
			return nil, invalid(field, problem.Required,
				fmt.Sprintf("Nobody can sign the slot %q: record the holder of its position in Master data → Positions, and give the person a role with the right S-1.", slot.PrintTitle))
		}
		if person == "" {
			return nil, invalid(field, problem.Required, fmt.Sprintf("Choose who signs the slot %q.", slot.PrintTitle))
		}
		i := slices.IndexFunc(slot.Candidates, func(c Candidate) bool { return c.PersonID == person })
		if i < 0 {
			return nil, invalid(field, problem.NotAllowed, fmt.Sprintf("This person cannot sign the slot %q.", slot.PrintTitle))
		}
		c := slot.Candidates[i]
		if slices.ContainsFunc(out, func(x Candidate) bool { return x.PersonID == c.PersonID }) {
			return nil, invalid(field, problem.NotAllowed, "One person signs only one slot of a route.")
		}
		switch slot.Stage {
		case StagePrepared:
			preparedKey = c.PositionKey
		case StageChecked:
			// Bila Project Leader tingkat Leader menjadi Prepared, Checked
			// adalah Project Leader tingkat Supervisor (D-203).
			if preparedKey == positionProjectEngineer && c.PositionKey != positionProjectLeader {
				return nil, invalid(field, problem.NotAllowed, "When the Project Leader (Leader) prepares, the Project Leader (Supervisor) checks.")
			}
		}
		out = append(out, c)
	}
	return out, nil
}

// Send mengirim draft terbuka satu dokumen untuk tanda tangan digital (E-5).
func (s *Service) Send(ctx context.Context, actor access.Actor, documentID string, in SendInput) (Request, error) {
	if !db.IsUUID(documentID) {
		return Request{}, problem.ErrNotFound
	}
	d, err := loadDocument(ctx, s.q, documentID)
	if err != nil {
		return Request{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.PrintForSignature, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Request{}, err
	}
	note := strings.TrimSpace(in.Note)
	if len([]rune(note)) > 1000 {
		return Request{}, invalid("note", problem.TooLong, "The note can be at most 1000 characters.")
	}
	if err := s.passwords.ConfirmPassword(ctx, actor, in.Password); err != nil {
		return Request{}, err
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocument(ctx, documentID); err != nil {
			return err
		}
		d, err := loadDocument(ctx, q, documentID)
		if err != nil {
			return err
		}
		switch {
		case d.Status != document.StatusDraft:
			return invalid("status", problem.NotAllowed, fmt.Sprintf("%s has no open draft.", d.Number))
		case d.Signing != "":
			return invalid("status", problem.NotAllowed, fmt.Sprintf("%s is already sent for signature.", d.Number))
		case d.Baseline != nil:
			return invalid("status", problem.NotAllowed, "A migration baseline is released by verifying it against the hard copy.")
		}
		p, err := s.plan(ctx, q, actor, d)
		if err != nil {
			return err
		}
		if p.Mode != ModeDigital {
			return invalid("mode", problem.NotAllowed, fmt.Sprintf("Customer %s requires wet signatures. Print the document for signature instead.", d.CustomerCode))
		}
		signers, err := chooseSigners(p, in.Signers)
		if err != nil {
			return err
		}
		draft, err := q.GetOpenDraftRevision(ctx, documentID)
		if err != nil {
			return fmt.Errorf("membaca draft: %w", err)
		}
		_, text, err := document.CurrentSnapshot(d)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(text)
		hash := hex.EncodeToString(sum[:])
		// Tanggal rilis = tanggal dikirim, kalender WIB (D-204).
		today := clock.Date(s.now())
		id, err = q.CreateSignatureRequest(ctx, db.CreateSignatureRequestParams{
			DocumentID: documentID, RevisionID: draft.ID, Revision: draft.Revision, CustomerID: d.CustomerID,
			RouteID: p.RouteID, RouteName: p.RouteName, Snapshot: string(text), ContentSha256: hash,
			ReleaseDate: pgtype.Date{Time: today, Valid: true}, Note: optional(note),
			SentByPersonID: actor.PersonID, SentByAccountID: actor.AccountID,
		})
		if err != nil {
			return err
		}
		for i, slot := range p.Slots {
			c := signers[i]
			if err := q.AddSignatureSlot(ctx, db.AddSignatureSlotParams{
				RequestID: id, SlotOrder: int16(slot.Order), Stage: slot.Stage, PrintTitle: slot.PrintTitle,
				PositionName: c.Position, PersonID: c.PersonID, PersonName: c.Name,
			}); err != nil {
				return err
			}
		}
		if err := q.SetRevisionReleaseDate(ctx, db.SetRevisionReleaseDateParams{ID: draft.ID, ReleaseDate: pgtype.Date{Time: today, Valid: true}}); err != nil {
			return err
		}
		if err := audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSignatureSent, TargetType: audit.TargetDocument, TargetID: documentID,
			Details: map[string]string{"number": d.Number, "revision": strconv.Itoa(int(draft.Revision)), "request": id, "route": p.RouteName, "sha256": hash},
		}); err != nil {
			return err
		}
		// Pengirim yang memilih dirinya sebagai Prepared langsung menandatangani.
		if signers[0].PersonID == actor.PersonID {
			if err := sign(ctx, q, actor, id, documentID, p.Slots[0], signers[0], hash); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Request{}, keep(err, "mengirim untuk tanda tangan")
	}
	return s.get(ctx, s.q, id)
}

// sign mencatat tanda tangan satu slot, lalu menutup permintaan bila semua
// slot sudah bertanda tangan. Dipanggil di dalam transaksi.
func sign(ctx context.Context, q *db.Queries, actor access.Actor, requestID, documentID string, slot SlotPlan, c Candidate, hash string) error {
	ip := audit.RequestFrom(ctx).IP
	if err := q.AddSignature(ctx, db.AddSignatureParams{
		RequestID: requestID, SlotOrder: int16(slot.Order), PersonID: actor.PersonID, AccountID: actor.AccountID,
		PersonName: c.Name, PrintTitle: slot.PrintTitle, Stage: slot.Stage, ContentSha256: hash, SourceIp: optional(ip),
	}); err != nil {
		return err
	}
	if err := audit.Record(ctx, q, actor, audit.Event{
		Action: audit.ActionSignatureSigned, TargetType: audit.TargetDocument, TargetID: documentID,
		Details: map[string]string{"request": requestID, "slot": slot.PrintTitle, "stage": slot.Stage, "sha256": hash},
	}); err != nil {
		return err
	}
	_, err := q.NextSignerOf(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := q.CloseSignatureRequest(ctx, db.CloseSignatureRequestParams{ID: requestID, Status: StatusCompleted, ClosedBy: &actor.PersonID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSignatureCompleted, TargetType: audit.TargetDocument, TargetID: documentID,
			Details: map[string]string{"request": requestID},
		})
	}
	return err
}

// Sign mencatat tanda tangan aktor pada slot yang sedang menunggu (S-1).
func (s *Service) Sign(ctx context.Context, actor access.Actor, requestID, password string) (Request, error) {
	r, err := s.Get(ctx, actor, requestID)
	if err != nil {
		return Request{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.SignDigitally, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Request{}, err
	}
	if err := s.passwords.ConfirmPassword(ctx, actor, password); err != nil {
		return Request{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		locked, err := q.LockSignatureRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if locked.Status != StatusWaiting {
			return invalid("status", problem.NotAllowed, "This request is no longer waiting for signatures.")
		}
		next, err := q.NextSignerOf(ctx, requestID)
		if err != nil {
			return err
		}
		if next.PersonID != actor.PersonID {
			return invalid("signer", problem.NotAllowed, fmt.Sprintf("It is not your turn: %s signs %q first.", next.PersonName, next.PrintTitle))
		}
		slot := slotOf(r, int(next.SlotOrder))
		return sign(ctx, q, actor, requestID, locked.DocumentID,
			SlotPlan{Order: slot.Order, Stage: slot.Stage, PrintTitle: slot.PrintTitle},
			Candidate{PersonID: actor.PersonID, Name: slot.PersonName}, locked.ContentSha256)
	})
	if err != nil {
		return Request{}, keep(err, "menandatangani")
	}
	return s.get(ctx, s.q, requestID)
}

// Reject menolak permintaan (penanda tangan yang sedang ditunggu, S-1).
// Alasan wajib. Draft terbuka lagi, dan tanggal rilisnya dikosongkan.
func (s *Service) Reject(ctx context.Context, actor access.Actor, requestID, reason, password string) (Request, error) {
	r, err := s.Get(ctx, actor, requestID)
	if err != nil {
		return Request{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.SignDigitally, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Request{}, err
	}
	reason = strings.TrimSpace(reason)
	switch {
	case reason == "":
		return Request{}, invalid("reason", problem.Required, "Write why you reject it.")
	case len([]rune(reason)) > 1000:
		return Request{}, invalid("reason", problem.TooLong, "The reason can be at most 1000 characters.")
	}
	if err := s.passwords.ConfirmPassword(ctx, actor, password); err != nil {
		return Request{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		locked, err := q.LockSignatureRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if locked.Status != StatusWaiting {
			return invalid("status", problem.NotAllowed, "This request is no longer waiting for signatures.")
		}
		next, err := q.NextSignerOf(ctx, requestID)
		if err != nil {
			return err
		}
		if next.PersonID != actor.PersonID {
			return invalid("signer", problem.NotAllowed, "Only the person whose signature is awaited can reject.")
		}
		return closeRequest(ctx, q, actor, locked, StatusRejected, &reason, audit.ActionSignatureRejected)
	})
	if err != nil {
		return Request{}, keep(err, "menolak permintaan tanda tangan")
	}
	return s.get(ctx, s.q, requestID)
}

// Withdraw menarik kembali permintaan yang masih menunggu (pengirimnya, E-5).
func (s *Service) Withdraw(ctx context.Context, actor access.Actor, requestID string) (Request, error) {
	r, err := s.Get(ctx, actor, requestID)
	if err != nil {
		return Request{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.PrintForSignature, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Request{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		locked, err := q.LockSignatureRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if locked.Status != StatusWaiting {
			return invalid("status", problem.NotAllowed, "This request is no longer waiting for signatures.")
		}
		if locked.SentByPersonID != actor.PersonID {
			return invalid("sender", problem.NotAllowed, "Only the person who sent it can withdraw it.")
		}
		return closeRequest(ctx, q, actor, locked, StatusWithdrawn, nil, audit.ActionSignatureWithdrawn)
	})
	if err != nil {
		return Request{}, keep(err, "menarik kembali permintaan tanda tangan")
	}
	return s.get(ctx, s.q, requestID)
}

// closeRequest menutup permintaan sebagai ditolak atau ditarik kembali, dan
// mengosongkan tanggal rilis draft supaya diisi lagi saat dikirim ulang.
func closeRequest(ctx context.Context, q *db.Queries, actor access.Actor, r db.LockSignatureRequestRow, status string, reason *string, action string) error {
	if err := q.CloseSignatureRequest(ctx, db.CloseSignatureRequestParams{ID: r.ID, Status: status, ClosedBy: &actor.PersonID, Reason: reason}); err != nil {
		return err
	}
	if err := q.SetRevisionReleaseDate(ctx, db.SetRevisionReleaseDateParams{ID: r.RevisionID}); err != nil {
		return err
	}
	details := map[string]string{"request": r.ID, "revision": strconv.Itoa(int(r.Revision))}
	if reason != nil {
		details["reason"] = *reason
	}
	return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetDocument, TargetID: r.DocumentID, Details: details})
}

func slotOf(r Request, order int) Slot {
	for _, s := range r.Slots {
		if s.Order == order {
			return s
		}
	}
	return Slot{Order: order}
}

// Get membaca satu permintaan (L-4 untuk customer dokumennya).
func (s *Service) Get(ctx context.Context, actor access.Actor, requestID string) (Request, error) {
	if !db.IsUUID(requestID) {
		return Request{}, problem.ErrNotFound
	}
	r, err := s.get(ctx, s.q, requestID)
	if err != nil {
		return Request{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (s *Service) get(ctx context.Context, q *db.Queries, id string) (Request, error) {
	row, err := q.GetSignatureRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, problem.ErrNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("membaca permintaan tanda tangan: %w", err)
	}
	r := Request{
		ID: row.ID, DocumentID: row.DocumentID, Number: row.Number, DocumentType: deref(row.DocumentTypeKey),
		CustomerID: row.CustomerID, CustomerCode: row.CustomerCode, Revision: int(row.Revision), RouteName: row.RouteName,
		ContentSHA256: row.ContentSha256, Note: deref(row.Note), Status: row.Status, SentBy: row.SentBy, SentByID: row.SentByPersonID,
		SentAt: row.SentAt.UTC(), ClosedBy: deref(row.ClosedBy), RejectReason: deref(row.RejectReason), Slots: []Slot{}, Changes: []string{},
	}
	if row.ReleaseDate.Valid {
		r.ReleaseDate = row.ReleaseDate.Time.Format(time.DateOnly)
	}
	if row.ClosedAt != nil {
		at := row.ClosedAt.UTC()
		r.ClosedAt = &at
	}
	slots, err := q.ListSignatureSlots(ctx, id)
	if err != nil {
		return Request{}, fmt.Errorf("membaca slot tanda tangan: %w", err)
	}
	for _, sl := range slots {
		item := Slot{Order: int(sl.SlotOrder), Stage: sl.Stage, PrintTitle: sl.PrintTitle, Position: sl.PositionName, PersonID: sl.PersonID, PersonName: sl.PersonName}
		if sl.SignedAt != nil {
			at := sl.SignedAt.UTC()
			item.SignedAt = &at
		} else if r.NextSlot == 0 && r.Status == StatusWaiting {
			r.NextSlot = item.Order
		}
		r.Slots = append(r.Slots, item)
	}
	changes, err := q.ListRevisionChanges(ctx, &row.RevisionID)
	if err != nil {
		return Request{}, fmt.Errorf("membaca perubahan revisi: %w", err)
	}
	for _, c := range changes {
		if c.Seq != nil {
			r.Changes = append(r.Changes, fmt.Sprintf("CHG-%04d", *c.Seq))
		}
	}
	return r, nil
}

// list membaca beberapa permintaan menurut ID, yang tidak terbaca dilewati.
func (s *Service) list(ctx context.Context, ids []string) ([]Request, error) {
	out := make([]Request, 0, len(ids))
	for _, id := range ids {
		r, err := s.get(ctx, s.q, id)
		if errors.Is(err, problem.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// ForDocument memberi permintaan tanda tangan satu dokumen, terbaru dulu (L-4).
func (s *Service) ForDocument(ctx context.Context, actor access.Actor, documentID string) ([]Request, error) {
	if !db.IsUUID(documentID) {
		return nil, problem.ErrNotFound
	}
	d, err := loadDocument(ctx, s.q, documentID)
	if err != nil {
		return nil, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return nil, err
	}
	ids, err := s.q.ListDocumentSignatureRequests(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("membaca permintaan tanda tangan: %w", err)
	}
	return s.list(ctx, ids)
}

// Overview adalah isi halaman Documents → Signatures.
type Overview struct {
	Waiting   []Request // menunggu tanda tangan aktor sekarang
	Sent      []Request // dikirim aktor, terbaru dulu
	ToRelease []Request // lengkap, menunggu rilis dicatat (hanya pemegang E-7)
}

// List memberi permintaan yang menyangkut aktor. Setiap akun yang login boleh
// membacanya untuk dirinya sendiri; isi dokumen tetap disaring RLS.
func (s *Service) List(ctx context.Context, actor access.Actor) (Overview, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: actor.AccountID}); err != nil {
		return Overview{}, err
	}
	var o Overview
	waiting, err := s.q.SignatureRequestsWaitingFor(ctx, actor.PersonID)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca permintaan: %w", err)
	}
	if o.Waiting, err = s.list(ctx, waiting); err != nil {
		return Overview{}, err
	}
	sent, err := s.q.SignatureRequestsSentBy(ctx, actor.PersonID)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca permintaan: %w", err)
	}
	if o.Sent, err = s.list(ctx, sent); err != nil {
		return Overview{}, err
	}
	done, err := s.q.SignatureRequestsCompleted(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca permintaan: %w", err)
	}
	all, err := s.list(ctx, done)
	if err != nil {
		return Overview{}, err
	}
	o.ToRelease = []Request{}
	for _, r := range all {
		if s.authz.Authorize(ctx, actor, access.RecordRelease, access.Resource{CustomerID: r.CustomerID}) == nil {
			o.ToRelease = append(o.ToRelease, r)
		}
	}
	return o, nil
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
