package document

// Tanda "siap cetak" (D-202). Penyusun menekan Ready to print saat isi draft
// selesai. Tandanya hanya dicatat untuk laporan kinerja: memisahkan tahap
// "edit sampai siap cetak" dari "menunggu cetak". Mengubah isi sesudahnya
// tetap boleh; menandai lagi memperbarui waktunya.

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// MarkReady memberi tanda siap cetak pada draft terbuka dokumen id (E-2).
func (s *Service) MarkReady(ctx context.Context, actor access.Actor, id string) ([]Revision, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return nil, err
	}
	if d.Status != StatusDraft {
		return nil, problems(invalid("status", problem.NotAllowed, "Only a draft can be marked ready to print."))
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.MarkRevisionReady(ctx, db.MarkRevisionReadyParams{DocumentID: id, ReadyByPersonID: &actor.PersonID})
		if err != nil {
			return err
		}
		if n == 0 {
			return problems(invalid("status", problem.NotAllowed, "Only a draft can be marked ready to print."))
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentMarkedReady, TargetType: "document", TargetID: id,
			Details: map[string]string{"number": d.Number, "revision": strconv.Itoa(d.Revision)}})
	})
	if err != nil {
		return nil, result(err, fmt.Sprintf("menandai %s siap cetak", d.Number))
	}
	return s.Revisions(ctx, actor, id)
}
