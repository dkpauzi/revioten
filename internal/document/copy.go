package document

// Salin komposisi dari dokumen model lain (M4.7, R-CMP-05). Dokumen model
// baru dimulai dari baris dokumen model serupa, lalu disesuaikan. Salinan
// mendapat ID baris baru, jadi mengubahnya tidak mengubah sumbernya.

import (
	"context"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// CopyFrom menyalin baris dan poin yang dikecualikan dari dokumen sumber ke
// dokumen ini (E-2), beserta dokumen lain di set yang sama: PFD dari PFD,
// PFMEA dari PFMEA, CP dari CP. Syaratnya: dokumen tujuan masih kosong, dan
// sumbernya dokumen lain dengan customer, jenis, dan versi format yang sama.
func (s *Service) CopyFrom(ctx context.Context, actor access.Actor, id, sourceID string) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		missing := problems(invalid("source_document_id", problem.InvalidFormat, "Choose an existing document."))
		if !db.IsUUID(sourceID) {
			return missing
		}
		source, err := getDocument(ctx, q, sourceID)
		if err != nil {
			return missing
		}
		wrong := func(message string) error {
			return problems(invalid("source_document_id", problem.NotAllowed, message))
		}
		switch {
		case source.ID == d.ID || slices.ContainsFunc(d.Set, func(m SetMember) bool { return m.ID == source.ID }):
			return wrong("Choose a document of another model.")
		case source.CustomerID != d.CustomerID:
			return wrong("Choose a document of the same customer.")
		case source.DocumentType.ID != d.DocumentType.ID:
			return wrong("Choose a document of the same document type.")
		}
		// Dokumen tujuan satu set dikunci menurut urutan ID (seperti perakitan).
		ids := []string{d.ID}
		for _, m := range d.Set {
			if m.ID != d.ID {
				ids = append(ids, m.ID)
			}
		}
		slices.Sort(ids)
		for _, docID := range ids {
			if err := q.LockDocument(ctx, docID); err != nil {
				return err
			}
		}
		// Pasangan tujuan dan sumber per jenis dokumen.
		sources := map[string]string{source.DocumentType.Key: source.ID}
		for _, m := range source.Set {
			sources[m.DocumentType.Key] = m.ID
		}
		type pair struct{ target, source Document }
		var pairs []pair
		for _, docID := range ids {
			target, err := getDocument(ctx, q, docID)
			if err != nil {
				return err
			}
			from, ok := sources[target.DocumentType.Key]
			if !ok {
				continue // sumber tidak punya dokumen jenis itu
			}
			src, err := getDocument(ctx, q, from)
			if err != nil {
				return err
			}
			if len(target.Rows) > 0 {
				return problems(invalid("rows", problem.NotAllowed, "Copy only into a document set that has no rows yet."))
			}
			if src.FormatVersionID != target.FormatVersionID {
				return wrong("The two documents use different form versions.")
			}
			pairs = append(pairs, pair{target: target, source: src})
		}
		total := 0
		for _, p := range pairs {
			n, err := q.CopyDocumentRows(ctx, db.CopyDocumentRowsParams{TargetID: p.target.ID, SourceID: p.source.ID})
			if err != nil {
				return err
			}
			total += int(n)
			if err := q.CopyPointExclusions(ctx, db.CopyPointExclusionsParams{TargetID: p.target.ID, SourceID: p.source.ID}); err != nil {
				return err
			}
			if err := q.TouchDocument(ctx, p.target.ID); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentCopied, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"source": source.Number, "rows": strconv.Itoa(total)},
		})
	})
	if err != nil {
		return Document{}, result(err, "menyalin komposisi")
	}
	return getDocument(ctx, s.q, id)
}
