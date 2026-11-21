package product

// Model end of life (D-200). PPAP Engineer dan Project Leader menandai model
// yang sudah tidak diproduksi, supaya set dokumennya keluar dari review
// berkala. Haknya E-2 (buat dan edit draft) untuk customer model itu, karena
// kedua peran itu yang memegangnya; master model lainnya tetap E-11.

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
)

// SetEOL menandai model end of life, atau mencabut tandanya (E-2).
func (s *Service) SetEOL(ctx context.Context, actor access.Actor, id string, eol bool) (Model, error) {
	m, err := getModel(ctx, s.q, id)
	if err != nil {
		return Model{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: m.CustomerID}); err != nil {
		return Model{}, err
	}
	if (m.EOLAt != nil) == eol {
		return m, nil
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if _, err := q.SetModelEOL(ctx, db.SetModelEOLParams{ID: id, Eol: eol, PersonID: actor.PersonID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionModelEOLChanged, TargetType: "model", TargetID: id,
			Details: map[string]string{"model": m.Name, "eol": strconv.FormatBool(eol)}})
	})
	if err != nil {
		return Model{}, fmt.Errorf("menandai model EOL: %w", err)
	}
	return getModel(ctx, s.q, id)
}
