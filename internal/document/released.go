package document

// Tampilan baca dokumen terbit (D-195, keputusan E-128 opsi a). Akun dengan
// L-1 saja (Viewer: internal audit, QA, QC, produksi) membaca isi revisi yang
// berlaku dari snapshot-nya, termasuk PFMEA dan CP berdampingan, tanpa pernah
// melihat draft. Isi diambil dari snapshot, bukan dari baris dokumen, karena
// baris dokumen bisa sedang diedit di draft revisi berikutnya (R-REV-02).

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// ListReleased memberi dokumen satu customer yang punya revisi berlaku (L-1).
// Status dan revisi di daftar adalah milik revisi yang berlaku, bukan draft.
func (s *Service) ListReleased(ctx context.Context, actor access.Actor, customerID string) ([]Summary, error) {
	if !db.IsUUID(customerID) {
		return nil, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewReleased, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListCustomerReleasedDocuments(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("mengambil dokumen terbit: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, Summary{
			ID: r.ID, Number: r.Number, DocumentType: TypeRef{Key: deref(r.DocumentTypeKey), Name: r.DocumentTypeName},
			Revision: int(r.ReleasedRevision), Status: StatusReleased, FormSetName: r.FormSetName, UpdatedAt: r.ReleasedAt.UTC(),
			Models: r.ModelNames, Origin: r.Origin, ReleasedMigrated: r.ReleasedMigrated,
		})
	}
	return out, nil
}

// Released memberi isi revisi yang berlaku satu dokumen (L-1). Dokumen yang
// belum pernah terbit, atau nomornya Void, tidak ada bagi pembaca ini.
func (s *Service) Released(ctx context.Context, actor access.Actor, id string) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewReleased, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	return s.releasedDocument(ctx, d)
}

// releasedDocument menyusun Document dari snapshot revisi yang berlaku d.
func (s *Service) releasedDocument(ctx context.Context, d Document) (Document, error) {
	if d.ReleasedRevision == nil {
		return Document{}, problem.ErrNotFound
	}
	snap, err := s.releasedSnapshot(ctx, d.ID, *d.ReleasedRevision, "revision")
	if err != nil {
		var invalidErr *problem.ValidationError
		if errors.As(err, &invalidErr) {
			return Document{}, problem.ErrNotFound
		}
		return Document{}, err
	}
	out := d
	out.Revision = snap.Revision
	out.Status = StatusReleased
	out.FormatVersionID = snap.FormatVersionID
	out.FormSetVersionID = snap.FormSetVersionID
	out.Header = snap.Header
	if out.Header == nil {
		out.Header = map[string]any{}
	}
	out.HeaderVersion = 0
	out.Baseline = nil
	out.Route = nil
	if snap.RouteID != "" {
		out.Route = &Ref{ID: snap.RouteID, Name: snap.RouteName}
	}
	out.RouteReason = ""
	out.Models = make([]Ref, 0, len(snap.Models))
	for _, m := range snap.Models {
		out.Models = append(out.Models, Ref{Name: m})
	}
	out.PartNumbers = make([]PartRef, 0, len(snap.PartNumbers))
	for _, p := range snap.PartNumbers {
		out.PartNumbers = append(out.PartNumbers, PartRef{Number: p})
	}
	// Kunci bagian diambil dari bagian dokumen yang masih ada; bagian tidak
	// pernah dihapus selama masih berisi, jadi ID di snapshot selalu ditemukan.
	keys := map[string]string{}
	for _, sec := range d.Sections {
		keys[sec.ID] = sec.Key
	}
	out.Sections = make([]Section, 0, len(snap.Sections))
	for i, sec := range snap.Sections {
		key := keys[sec.ID]
		if key == "" {
			key = sec.ID
		}
		item := Section{ID: sec.ID, Key: key, Position: i + 1, Name: sec.Name, PartNumbers: make([]PartRef, 0, len(sec.PartNumbers))}
		for _, p := range sec.PartNumbers {
			item.PartNumbers = append(item.PartNumbers, PartRef{Number: p})
		}
		out.Sections = append(out.Sections, item)
	}
	out.Rows = make([]Row, 0, len(snap.Rows))
	for _, r := range snap.Rows {
		out.Rows = append(out.Rows, Row{
			ID: r.ID, Position: r.Position, SectionID: r.SectionID, ProcessID: r.ProcessID, ProcessName: r.ProcessName,
			OperationNumber: r.OperationNumber, TemplatePointID: r.TemplatePointID, Content: r.Content,
		})
	}
	// Set hanya berisi dokumen yang juga punya revisi berlaku.
	set := make([]SetMember, 0, len(d.Set))
	for _, m := range d.Set {
		if m.ID == d.ID {
			set = append(set, m)
			continue
		}
		released, err := s.q.DocumentReleasedRevision(ctx, m.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return Document{}, err
		}
		if released != nil {
			set = append(set, m)
		}
	}
	out.Set = set
	return out, nil
}

// ReleasedComparison memberi PFMEA dan CP revisi yang berlaku berdampingan
// per proses (L-1), dengan tanda dicek yang diberikan pada kedua revisi itu.
// Tanda tidak bisa diubah dari sini.
func (s *Service) ReleasedComparison(ctx context.Context, actor access.Actor, id string) (Comparison, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Comparison{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewReleased, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Comparison{}, err
	}
	live, err := pairLive(ctx, s.q, d)
	if err != nil {
		return Comparison{}, err
	}
	var released [2]Document
	for i, doc := range live {
		if released[i], err = s.releasedDocument(ctx, doc); errors.Is(err, problem.ErrNotFound) {
			return Comparison{}, problems(invalid("set", problem.NotAllowed,
				"The side-by-side check needs a released PFMEA and a released CP in the same document set."))
		} else if err != nil {
			return Comparison{}, err
		}
	}
	c, err := groupComparison(ctx, s.q, released[0], released[1])
	if err != nil {
		return Comparison{}, result(err, "membaca perbandingan PFMEA dan CP terbit")
	}
	// Snapshot tidak menyimpan versi baris, jadi tanda "perlu dicek ulang"
	// hanya bisa dihitung dari baris dokumen selama tidak ada draft terbuka:
	// saat itu baris dokumen sama dengan isi revisi yang berlaku. ASUMSI E-129:
	// selama draft terbuka, tanda ditampilkan seperti yang tercatat.
	draftOpen := live[0].Status != StatusReleased || live[1].Status != StatusReleased
	for i := range c.Processes {
		g := &c.Processes[i]
		if g.Check == nil {
			continue
		}
		g.Check.Stale = false
		if !draftOpen {
			g.Check.Stale = g.Check.fingerprint != fingerprint(live[0], live[1], g.SectionKey, g.ProcessID)
		}
	}
	return c, nil
}

// pairLive mencari dokumen PFMEA dan CP di set dokumen d, apa adanya.
func pairLive(ctx context.Context, q *db.Queries, d Document) ([2]Document, error) {
	pfmea, cp, err := pair(ctx, q, d)
	return [2]Document{pfmea, cp}, err
}
