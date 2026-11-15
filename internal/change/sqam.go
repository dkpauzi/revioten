package change

// Perubahan surut SQAM (M5.10, R-CHG-01, R-TPL-07). Revisi SQAM customer
// sering berlaku surut: semua dokumen aktif customer itu harus direvisi.
// Perubahan bermekanisme sqam menunjuk semua dokumen aktif sekaligus, dengan
// rujukan revisi SQAM, tanggal berlaku, dan jenis perubahan. Saat diterima,
// setiap dokumen mendapat draft baru (M5.4); revisi terbit tidak pernah
// ditulis ulang.

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// SQAMChange adalah isian perubahan surut SQAM.
type SQAMChange struct {
	CustomerID      string
	ChangeTypeID    string
	FourMCategoryID string // boleh kosong
	Reason          string
	SQAMReference   string     // misalnya "SQAM-KT rev 7, clause 4.3"
	EffectiveDate   *time.Time // tanggal berlaku revisi SQAM
	// DocumentTypeIDs memilih jenis dokumen yang terdampak (D-171); kosong
	// berarti semua jenis.
	DocumentTypeIDs []string
}

// SubmitSQAM mengajukan satu perubahan surut SQAM untuk semua dokumen aktif
// customer (E-3). Dokumen yang nomornya usang atau dibatalkan tidak ikut.
func (s *RequestService) SubmitSQAM(ctx context.Context, actor access.Actor, in SQAMChange) (Change, error) {
	if !db.IsUUID(in.CustomerID) {
		return Change{}, problems(invalid("customer_id", problem.Required, "Choose the customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Change{}, err
	}
	reason, p1 := problem.Text("reason", in.Reason, true, maxReason)
	reference, p2 := problem.Text("sqam_reference", in.SQAMReference, true, maxSourceReference)
	list := []*problem.Problem{p1, p2}
	if in.EffectiveDate == nil {
		list = append(list, invalid("effective_date", problem.Required, "Choose the date the SQAM revision takes effect."))
	}
	typeOK := false
	if db.IsUUID(in.ChangeTypeID) {
		var err error
		if typeOK, err = s.q.ChangeTypeExists(ctx, in.ChangeTypeID); err != nil {
			return Change{}, err
		}
	}
	if !typeOK {
		list = append(list, invalid("change_type_id", problem.Required, "Choose the change type."))
	}
	fourM := strings.TrimSpace(in.FourMCategoryID)
	if fourM != "" {
		ok := false
		if db.IsUUID(fourM) {
			var err error
			if ok, err = s.q.FourMCategoryExists(ctx, fourM); err != nil {
				return Change{}, err
			}
		}
		if !ok {
			list = append(list, invalid("four_m_category_id", problem.InvalidFormat, "Choose a 4M category from the list."))
		}
	}
	for _, typeID := range in.DocumentTypeIDs {
		if !db.IsUUID(typeID) {
			list = append(list, invalid("document_type_ids", problem.InvalidFormat, "Choose document types from the list."))
			break
		}
	}
	if err := problems(list...); err != nil {
		return Change{}, err
	}
	types := in.DocumentTypeIDs
	if types == nil {
		types = []string{}
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		docs, err := q.ListActiveCustomerDocuments(ctx, db.ListActiveCustomerDocumentsParams{CustomerID: in.CustomerID, TypeIds: types})
		if err != nil {
			return err
		}
		if len(docs) == 0 {
			return problems(invalid("customer_id", problem.NotAllowed, "The customer has no active document of the chosen types."))
		}
		if id, err = q.CreateSQAMChange(ctx, db.CreateSQAMChangeParams{
			CustomerID: in.CustomerID, ChangeTypeID: in.ChangeTypeID, FourMCategoryID: optional(fourM), Reason: reason,
			SourceReference: &reference, EffectiveDate: pgtype.Date{Time: *in.EffectiveDate, Valid: true},
			RequestedByPersonID: &actor.PersonID, RequestedByAccountID: &actor.AccountID,
		}); err != nil {
			return err
		}
		numbers := make([]string, 0, len(docs))
		for _, d := range docs {
			if err := q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: id, DocumentID: d.ID}); err != nil {
				return err
			}
			numbers = append(numbers, d.Number)
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionChangeCreated, TargetType: audit.TargetChange, TargetID: id,
			Details: map[string]string{"mechanism": MechanismSQAM, "sqam": reference, "effective_date": in.EffectiveDate.Format(time.DateOnly), "documents": strings.Join(numbers, ", ")}})
	})
	if err != nil {
		return Change{}, result(err, "mengajukan perubahan SQAM")
	}
	return s.load(ctx, id)
}
