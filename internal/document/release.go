package document

// Rilis dengan snapshot (M5.5, R-REV-02, R-APR-02). Saat revisi dicatat terbit,
// isi dokumen dibekukan sebagai teks JSON beserta SHA-256-nya. Revisi terbit
// tidak pernah berubah lagi (trigger di migrasi 00034). Pencatat rilis bukan
// orang yang menyusun draft itu (R-SEC-11).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Revision adalah satu revisi dokumen.
type Revision struct {
	ID         string
	Revision   int
	Status     string // draft atau released
	CreatedBy  string
	CreatedAt  time.Time
	ReleasedBy string     // "" untuk draft
	ReleasedAt *time.Time // nil untuk draft
	SHA256     string     // SHA-256 snapshot; "" untuk draft
}

// Snapshot adalah isi satu revisi terbit. Bentuknya tetap, supaya hash bisa
// dihitung ulang dan dua revisi bisa dibandingkan (M5.6).
type Snapshot struct {
	DocumentID       string            `json:"document_id"`
	Number           string            `json:"number"`
	DocumentType     string            `json:"document_type"`
	CustomerCode     string            `json:"customer_code"`
	Revision         int               `json:"revision"`
	FormSetVersionID string            `json:"form_set_version_id"`
	FormatVersionID  string            `json:"format_version_id"`
	RouteID          string            `json:"route_id,omitempty"`
	RouteName        string            `json:"route_name,omitempty"`
	Header           map[string]any    `json:"header"`
	Models           []string          `json:"models"`
	PartNumbers      []string          `json:"part_numbers"`
	Sections         []SnapshotSection `json:"sections"`
	Rows             []SnapshotRow     `json:"rows"`
}

// SnapshotSection adalah satu bagian per PCB di snapshot.
type SnapshotSection struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	PartNumbers []string `json:"part_numbers"`
}

// SnapshotRow adalah satu baris di snapshot. Nama proses ikut dibekukan,
// karena nama proses bisa diganti sesudahnya (D-06).
type SnapshotRow struct {
	ID              string         `json:"id"`
	Position        int            `json:"position"`
	SectionID       string         `json:"section_id,omitempty"`
	ProcessID       string         `json:"process_id,omitempty"`
	ProcessName     string         `json:"process_name,omitempty"`
	OperationNumber string         `json:"operation_number,omitempty"`
	TemplatePointID string         `json:"template_point_id,omitempty"`
	Content         map[string]any `json:"content"`
}

// makeSnapshot membekukan isi dokumen. json.Marshal menulis kunci map menurut
// abjad, jadi isi yang sama selalu menghasilkan teks dan hash yang sama.
func makeSnapshot(d Document) ([]byte, string, error) {
	s := Snapshot{
		DocumentID: d.ID, Number: d.Number, DocumentType: d.DocumentType.Key, CustomerCode: d.CustomerCode, Revision: d.Revision,
		FormSetVersionID: d.FormSetVersionID, FormatVersionID: d.FormatVersionID, Header: d.Header,
		Models: []string{}, PartNumbers: []string{}, Sections: []SnapshotSection{}, Rows: []SnapshotRow{},
	}
	if s.Header == nil {
		s.Header = map[string]any{}
	}
	if d.Route != nil {
		s.RouteID, s.RouteName = d.Route.ID, d.Route.Name
	}
	for _, m := range d.Models {
		s.Models = append(s.Models, m.Name)
	}
	for _, p := range d.PartNumbers {
		s.PartNumbers = append(s.PartNumbers, p.Number)
	}
	for _, sec := range d.Sections {
		item := SnapshotSection{ID: sec.ID, Name: sec.Name, PartNumbers: []string{}}
		for _, p := range sec.PartNumbers {
			item.PartNumbers = append(item.PartNumbers, p.Number)
		}
		s.Sections = append(s.Sections, item)
	}
	for _, r := range d.Rows {
		content := r.Content
		if content == nil {
			content = map[string]any{}
		}
		s.Rows = append(s.Rows, SnapshotRow{
			ID: r.ID, Position: r.Position, SectionID: r.SectionID, ProcessID: r.ProcessID, ProcessName: r.ProcessName,
			OperationNumber: r.OperationNumber, TemplatePointID: r.TemplatePointID, Content: content,
		})
	}
	text, err := json.Marshal(s)
	if err != nil {
		return nil, "", fmt.Errorf("menulis snapshot: %w", err)
	}
	sum := sha256.Sum256(text)
	return text, hex.EncodeToString(sum[:]), nil
}

// Release mencatat draft terbuka satu dokumen sebagai revisi terbit (E-7).
func (s *Service) Release(ctx context.Context, actor access.Actor, id string) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.RecordRelease, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if err := requireDraft(d); err != nil {
			return err
		}
		// Aturan yang tidak bisa dimatikan: pencatat rilis bukan penyusun (R-SEC-11).
		prepared, err := q.PreparedDraft(ctx, db.PreparedDraftParams{DocumentID: &id, PersonID: &actor.PersonID})
		if err != nil {
			return err
		}
		if prepared {
			return problems(invalid("recorder", problem.NotAllowed, "The person who prepared the draft cannot record its release."))
		}
		draft, err := q.GetOpenDraftRevision(ctx, id)
		if err != nil {
			return fmt.Errorf("membaca draft: %w", err)
		}
		text, sum, err := makeSnapshot(d)
		if err != nil {
			return err
		}
		snapshot := string(text)
		n, err := q.ReleaseDocumentRevision(ctx, db.ReleaseDocumentRevisionParams{
			ID: draft.ID, Snapshot: &snapshot, Sha256: &sum, PersonID: &actor.PersonID, AccountID: &actor.AccountID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return problems(invalid("status", problem.NotAllowed, "The draft is already released."))
		}
		if err := q.MarkDocumentReleased(ctx, db.MarkDocumentReleasedParams{ID: id, ReleasedRevision: &draft.Revision}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentReleased, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"number": d.Number, "revision": strconv.Itoa(int(draft.Revision)), "sha256": sum},
		})
	})
	if err != nil {
		return Document{}, result(err, "mencatat rilis")
	}
	return s.load(ctx, id)
}

// Revisions memberi semua revisi satu dokumen, yang terbaru dulu (L-4).
func (s *Service) Revisions(ctx context.Context, actor access.Actor, id string) ([]Revision, error) {
	if _, err := s.Get(ctx, actor, id); err != nil {
		return nil, err
	}
	rows, err := s.q.ListDocumentRevisions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("membaca revisi: %w", err)
	}
	out := make([]Revision, 0, len(rows))
	for _, r := range rows {
		item := Revision{
			ID: r.ID, Revision: int(r.Revision), Status: r.Status, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt.UTC(),
			ReleasedBy: deref(r.ReleasedBy), SHA256: deref(r.SnapshotSha256),
		}
		if r.ReleasedAt != nil {
			at := r.ReleasedAt.UTC()
			item.ReleasedAt = &at
		}
		out = append(out, item)
	}
	return out, nil
}
