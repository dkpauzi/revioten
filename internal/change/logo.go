package change

// Perubahan logo (D-223, E-145 opsi a). Ganti logo diperlakukan seperti
// revisi dokumen. Bila admin menjawab "ya, revisi dokumen" saat mengganti
// logo, setiap customer yang punya dokumen aktif menunggu satu perubahan
// "Logo change". PPAP Engineer mengajukannya per customer; perubahan itu
// mencakup semua dokumen aktif customer seperti perubahan surut SQAM (M5.10),
// lalu diverifikasi, ditandatangani, dan dirilis seperti biasa. Alasannya
// masuk ke riwayat revisi. Yang menunggu hanya penggantian logo terakhir:
// logo yang sudah diganti lagi tidak perlu direvisi.

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// MechanismLogo: perubahan karena logo perusahaan diganti (D-223).
const MechanismLogo = "logo"

// PendingLogo adalah penggantian logo yang menunggu perubahan dari satu customer.
type PendingLogo struct {
	ID         string
	LogoSHA256 string // "" berarti kembali ke logo bawaan
	ChangedAt  time.Time
}

// LogoChange adalah isian perubahan logo satu customer.
type LogoChange struct {
	CustomerID      string
	LogoChangeID    string
	ChangeTypeID    string
	FourMCategoryID string // boleh kosong
	Reason          string // misalnya "Change company logo"
}

// PendingLogoChange memberi penggantian logo yang menunggu perubahan dari
// customer ini, atau nil: penggantian terakhir yang dijawab "ya, revisi",
// selama customer itu belum mengajukan perubahannya dan masih punya dokumen
// aktif.
func (s *RequestService) PendingLogoChange(ctx context.Context, actor access.Actor, customerID string) (*PendingLogo, error) {
	if !db.IsUUID(customerID) {
		return nil, problem.ErrNotFound
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	return pendingLogo(ctx, s.q, customerID)
}

func pendingLogo(ctx context.Context, q *db.Queries, customerID string) (*PendingLogo, error) {
	last, err := q.LatestLogoChange(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !last.Revise {
		return nil, nil
	}
	done, err := q.LogoChangeSubmitted(ctx, db.LogoChangeSubmittedParams{LogoChangeID: &last.ID, CustomerID: customerID})
	if err != nil || done {
		return nil, err
	}
	docs, err := q.ListActiveCustomerDocuments(ctx, db.ListActiveCustomerDocumentsParams{CustomerID: customerID, TypeIds: []string{}})
	if err != nil || len(docs) == 0 {
		return nil, err
	}
	p := &PendingLogo{ID: last.ID, ChangedAt: last.CreatedAt.UTC()}
	if last.LogoSha256 != nil {
		p.LogoSHA256 = *last.LogoSha256
	}
	return p, nil
}

// SubmitLogoChange mengajukan perubahan logo satu customer (E-3): semua
// dokumen aktif customer itu, sekali per penggantian logo.
func (s *RequestService) SubmitLogoChange(ctx context.Context, actor access.Actor, in LogoChange) (Change, error) {
	if !db.IsUUID(in.CustomerID) {
		return Change{}, problems(invalid("customer_id", problem.Required, "Choose the customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Change{}, err
	}
	reason, p1 := problem.Text("reason", in.Reason, true, maxReason)
	list := []*problem.Problem{p1}
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
	if err := problems(list...); err != nil {
		return Change{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		pending, err := pendingLogo(ctx, q, in.CustomerID)
		if err != nil {
			return err
		}
		if pending == nil || pending.ID != in.LogoChangeID {
			return problems(invalid("logo_change_id", problem.NotAllowed, "No logo change waits for this customer, or it was already submitted."))
		}
		docs, err := q.ListActiveCustomerDocuments(ctx, db.ListActiveCustomerDocumentsParams{CustomerID: in.CustomerID, TypeIds: []string{}})
		if err != nil {
			return err
		}
		if id, err = q.CreateLogoChange(ctx, db.CreateLogoChangeParams{
			CustomerID: in.CustomerID, ChangeTypeID: in.ChangeTypeID, FourMCategoryID: optional(fourM), Reason: reason,
			LogoChangeID: &pending.ID, RequestedByPersonID: &actor.PersonID, RequestedByAccountID: &actor.AccountID,
		}); err != nil {
			if db.IsUniqueViolation(err, "changes_logo_customer_key") {
				return problems(invalid("logo_change_id", problem.NotAllowed, "No logo change waits for this customer, or it was already submitted."))
			}
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
			Details: map[string]string{"mechanism": MechanismLogo, "logo_sha256": orDefaultLogo(pending.LogoSHA256), "documents": strings.Join(numbers, ", ")}})
	})
	if err != nil {
		return Change{}, result(err, "mengajukan perubahan logo")
	}
	return s.load(ctx, id)
}

func orDefaultLogo(sha string) string {
	if sha == "" {
		return "default"
	}
	return sha
}
