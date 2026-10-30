package numbering

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Register nomor (M2.3, UC-12, UC-13, R-NUM-04): daftar dengan saringan, dan
// perubahan status oleh dokumen kontrol. Nomor tidak pernah dihapus atau
// dipakai ulang: role aplikasi tidak punya DELETE di document_numbers, hanya
// kolom status yang boleh diubah, dan penghitung tidak pernah turun.

// Batas daftar register.
const (
	defaultListLimit = 100
	maxListLimit     = 500
	maxQueryLength   = 50
	maxReasonLength  = 500
)

// ListFilter adalah saringan register. Field kosong berarti tidak disaring.
type ListFilter struct {
	CustomerID     string
	DocumentTypeID string
	Status         string
	Query          string // potongan teks nomor, tanpa membedakan huruf besar kecil
	Before         string // ID nomor terakhir di halaman sebelumnya
	Limit          int    // 0 berarti 100
}

// statuses adalah semua status nomor yang sah.
var statuses = []string{StatusReserved, StatusInUse, StatusReleased, StatusObsolete, StatusVoid, StatusSkipped}

// ListNumbers mengembalikan register menurut saringan, terbaru dulu (L-2,
// UC-12). Hasil kedua adalah ID untuk halaman berikutnya, atau "" bila sudah
// habis. Register terbuka untuk semua akun, karena hanya berisi nomor dan
// pendaftarnya (D-103).
func (s *Service) ListNumbers(ctx context.Context, actor access.Actor, f ListFilter) ([]DocumentNumber, string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, "", err
	}
	params, err := listParams(f)
	if err != nil {
		return nil, "", err
	}
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	rows, err := s.q.ListDocumentNumbers(ctx, params)
	if err != nil {
		return nil, "", fmt.Errorf("membaca register: %w", err)
	}
	list := make([]DocumentNumber, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromNumber(db.GetDocumentNumberRow(r), int(settings.RevisionDigits)))
	}
	next := ""
	if len(list) == int(params.MaxItems) {
		next = list[len(list)-1].ID
	}
	return list, next, nil
}

// listParams memeriksa saringan dan menyusunnya untuk query.
func listParams(f ListFilter) (db.ListDocumentNumbersParams, error) {
	var list []*problem.Problem
	optional := func(field, value string, ok bool) *string {
		if value == "" {
			return nil
		}
		if !ok {
			list = append(list, &problem.Problem{Field: field, Code: problem.InvalidFormat, Message: "This filter value is not valid."})
		}
		return &value
	}
	p := db.ListDocumentNumbersParams{
		CustomerID:     optional("customer_id", f.CustomerID, db.IsUUID(f.CustomerID)),
		DocumentTypeID: optional("document_type_id", f.DocumentTypeID, db.IsUUID(f.DocumentTypeID)),
		Status:         optional("status", f.Status, slices.Contains(statuses, f.Status)),
		Query:          optional("q", f.Query, utf8.RuneCountInString(f.Query) <= maxQueryLength),
		Before:         optional("before", f.Before, db.IsUUID(f.Before)),
		MaxItems:       defaultListLimit,
	}
	switch {
	case f.Limit < 0 || f.Limit > maxListLimit:
		list = append(list, &problem.Problem{Field: "limit", Code: problem.InvalidFormat, Message: "Use 1 to 500."})
	case f.Limit > 0:
		p.MaxItems = int32(f.Limit)
	}
	return p, problems(list...)
}

// Void membatalkan nomor yang belum rilis (Reserved atau In use), dengan
// alasan wajib (E-8, UC-13, R-NUM-04). Nomornya tetap di register dan tidak
// pernah dipakai lagi.
func (s *Service) Void(ctx context.Context, actor access.Actor, id, reason string) (DocumentNumber, error) {
	reason, reasonProblem := problem.Text("reason", reason, true, maxReasonLength)
	return s.changeStatus(ctx, actor, id, statusChange{
		to: StatusVoid, from: []string{StatusReserved, StatusInUse}, reason: reason, reasonProblem: reasonProblem,
		action:  audit.ActionDocumentNumberVoided,
		refusal: "Only Reserved or In use numbers can be voided.",
	})
}

// MakeObsolete menandai nomor yang sudah rilis sebagai Obsolete, misalnya
// karena ditarik (E-8, UC-13). Alasan boleh kosong.
// ASUMSI E-68: alasan wajib hanya untuk Void, sesuai diagram status nomor di
// brief bagian 4 ("cancelled with reason").
func (s *Service) MakeObsolete(ctx context.Context, actor access.Actor, id, reason string) (DocumentNumber, error) {
	reason, reasonProblem := problem.Text("reason", reason, false, maxReasonLength)
	return s.changeStatus(ctx, actor, id, statusChange{
		to: StatusObsolete, from: []string{StatusReleased}, reason: reason, reasonProblem: reasonProblem,
		action:  audit.ActionDocumentNumberObsoleted,
		refusal: "Only Released numbers can be made obsolete.",
	})
}

// statusChange adalah satu perubahan status yang diizinkan.
type statusChange struct {
	to            string
	from          []string // status asal yang boleh
	reason        string
	reasonProblem *problem.Problem
	action        string
	refusal       string
}

// changeStatus menjalankan perubahan status dalam satu transaksi: baris
// nomor dikunci, status asalnya diperiksa, lalu status, alasan, dan waktunya
// disimpan bersama entri audit.
func (s *Service) changeStatus(ctx context.Context, actor access.Actor, id string, c statusChange) (DocumentNumber, error) {
	before, err := s.getNumber(ctx, s.q, id)
	if errors.Is(err, problem.ErrNotFound) {
		// Hak diperiksa dulu, supaya akun tanpa hak tidak bisa menebak nomor mana yang ada.
		if authErr := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{}); authErr != nil {
			return DocumentNumber{}, authErr
		}
		return DocumentNumber{}, err
	}
	if err != nil {
		return DocumentNumber{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{CustomerID: before.CustomerID}); err != nil {
		return DocumentNumber{}, err
	}
	if err := problems(c.reasonProblem); err != nil {
		return DocumentNumber{}, err
	}

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocumentNumber(ctx, before.ID); err != nil {
			return err
		}
		current, err := s.getNumber(ctx, q, before.ID)
		if err != nil {
			return err
		}
		if !slices.Contains(c.from, current.Status) {
			return &problem.ConflictError{Problems: []problem.Problem{{Field: "status", Code: problem.NotAllowed, Message: c.refusal}}}
		}
		var reason *string
		if c.reason != "" {
			reason = &c.reason
		}
		now := time.Now().UTC()
		err = q.SetDocumentNumberStatus(ctx, db.SetDocumentNumberStatusParams{
			ID: current.ID, Status: c.to, StatusReason: reason, StatusChangedAt: &now,
		})
		if err != nil {
			return err
		}
		details := map[string]string{"number": current.Number, "status": audit.Change(current.Status, c.to)}
		if c.reason != "" {
			details["reason"] = c.reason
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: c.action, TargetType: audit.TargetDocumentNumber, TargetID: current.ID, Details: details,
		})
	})
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return DocumentNumber{}, err
	case err != nil:
		return DocumentNumber{}, fmt.Errorf("mengubah status nomor: %w", err)
	}
	return s.getNumber(ctx, s.q, before.ID)
}

// getNumber membaca satu nomor dengan q (pool atau transaksi).
func (s *Service) getNumber(ctx context.Context, q *db.Queries, id string) (DocumentNumber, error) {
	if !db.IsUUID(id) {
		return DocumentNumber{}, problem.ErrNotFound
	}
	settings, err := q.GetNumberSettings(ctx)
	if err != nil {
		return DocumentNumber{}, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	r, err := q.GetDocumentNumber(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentNumber{}, problem.ErrNotFound
	}
	if err != nil {
		return DocumentNumber{}, fmt.Errorf("membaca nomor: %w", err)
	}
	return fromNumber(r, int(settings.RevisionDigits)), nil
}
