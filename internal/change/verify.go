package change

// Verifikasi atasan (M5.3, R-CHG-02, D-51, D-54, UC-41). Satu perubahan yang
// diajukan diterima atau ditolak oleh salah satu dari: Project Engineer atau
// Project Leader dari Project Group customer, atau APQP/PPAP Controller.
// Catatan wajib, dan pemohon tidak pernah memverifikasi permintaannya sendiri.

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

const maxVerificationNote = 2000

// Verification adalah hasil verifikasi satu perubahan.
type Verification struct {
	VerifiedBy   string // nama orang verifikator
	VerifiedByID string // ID orang verifikator
	VerifiedAt   time.Time
	Note         string
}

// Verify menerima (accept true) atau menolak satu perubahan yang diajukan (V-1).
func (s *RequestService) Verify(ctx context.Context, actor access.Actor, id string, accept bool, note string) (Change, error) {
	c, err := s.load(ctx, id)
	if err != nil {
		return Change{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.VerifyChanges, access.Resource{CustomerID: c.CustomerID}); err != nil {
		return Change{}, err
	}
	note = strings.TrimSpace(note)
	switch {
	case note == "":
		return Change{}, problems(invalid("note", problem.Required, "Write a note for the requester."))
	case utf8.RuneCountInString(note) > maxVerificationNote:
		return Change{}, problems(invalid("note", problem.TooLong, "A note can be at most 2000 characters."))
	}
	// Aturan yang tidak bisa dimatikan: pemohon bukan verifikator (R-SEC-11).
	if c.RequestedByID == actor.PersonID {
		return Change{}, problems(invalid("verifier", problem.NotAllowed, "The requester cannot verify their own change."))
	}
	// Hak V-1 saja belum cukup: orangnya harus memegang salah satu jabatan
	// verifikator untuk customer ini (D-51, D-54).
	ok, err := s.q.IsChangeVerifier(ctx, db.IsChangeVerifierParams{PersonID: actor.PersonID, CustomerID: c.CustomerID})
	if err != nil {
		return Change{}, err
	}
	if !ok {
		return Change{}, problems(invalid("verifier", problem.NotAllowed,
			"Only the Project Engineer or the Project Leader of the customer's project group, or the APQP/PPAP Controller, verifies a change."))
	}
	status, action := StatusRejected, audit.ActionChangeRejected
	if accept {
		status, action = StatusAccepted, audit.ActionChangeAccepted
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.VerifyChange(ctx, db.VerifyChangeParams{ID: id, Status: status, PersonID: &actor.PersonID, AccountID: &actor.AccountID, Note: &note})
		if err != nil {
			return err
		}
		if n == 0 {
			return problems(invalid("status", problem.NotAllowed, "Only a submitted change can be verified."))
		}
		if accept {
			if err := openDrafts(ctx, q, actor, c); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetChange, TargetID: id, Details: map[string]string{"note": note}})
	})
	if err != nil {
		return Change{}, result(err, "memverifikasi perubahan")
	}
	return s.load(ctx, id)
}

// openDrafts memberi setiap dokumen terdampak satu draft revisi (M5.4,
// R-CHG-03). Bila dokumen sudah punya draft terbuka, perubahan masuk ke draft
// itu (merged). Bila tidak, revisi berikutnya dibuka sebagai draft; revisi yang
// berlaku tidak berubah. Dokumen dikunci menurut urutan ID supaya dua
// verifikasi bersamaan tidak saling menunggu.
func openDrafts(ctx context.Context, q *db.Queries, actor access.Actor, c Change) error {
	ids := make([]string, 0, len(c.Documents))
	for _, d := range c.Documents {
		ids = append(ids, d.ID)
	}
	slices.Sort(ids)
	for _, docID := range ids {
		if err := q.LockDocument(ctx, docID); err != nil {
			return err
		}
		merged := true
		draft, err := q.GetOpenDraftRevision(ctx, docID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			merged = false
			last, err := q.MaxDocumentRevision(ctx, docID)
			if err != nil {
				return err
			}
			draft.Revision = last + 1
			if draft.ID, err = q.CreateDocumentRevision(ctx, db.CreateDocumentRevisionParams{
				DocumentID: docID, Revision: draft.Revision, CreatedByPersonID: actor.PersonID,
			}); err != nil {
				return err
			}
			if err := q.OpenDocumentDraft(ctx, db.OpenDocumentDraftParams{ID: docID, Revision: draft.Revision}); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		if err := q.LinkChangeDraft(ctx, db.LinkChangeDraftParams{ChangeID: c.ID, DocumentID: docID, DraftRevisionID: &draft.ID, Merged: merged}); err != nil {
			return err
		}
		details := map[string]string{"change": c.Ref, "revision": strconv.Itoa(int(draft.Revision)), "merged": strconv.FormatBool(merged)}
		if err := audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentDraftOpened, TargetType: audit.TargetDocument, TargetID: docID, Details: details}); err != nil {
			return err
		}
	}
	return nil
}
