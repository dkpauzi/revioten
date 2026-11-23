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
	// ReleaseDate diisi saat dicetak untuk tanda tangan (D-59); "" bila belum.
	ReleaseDate string
	// ScanFileName adalah scan bertanda tangan revisi ini (M6.7); "" bila belum.
	ScanFileName string
	// Migrated: revisi ini terbit lewat verifikasi hardcopy (M7.5).
	Migrated bool
	// ReadyAt: draft ditandai siap cetak (D-202); nil bila belum.
	ReadyAt *time.Time
	// Changes adalah nomor rujukan perubahan yang masuk ke revisi ini (M5.6),
	// misalnya CHG-0001; tidak pernah nil.
	Changes []string
}

// Snapshot adalah isi satu revisi terbit. Bentuknya tetap, supaya hash bisa
// dihitung ulang dan dua revisi bisa dibandingkan (M5.6).
type Snapshot struct {
	DocumentID       string         `json:"document_id"`
	Number           string         `json:"number"`
	DocumentType     string         `json:"document_type"`
	CustomerCode     string         `json:"customer_code"`
	Revision         int            `json:"revision"`
	FormSetVersionID string         `json:"form_set_version_id"`
	FormatVersionID  string         `json:"format_version_id"`
	RouteID          string         `json:"route_id,omitempty"`
	RouteName        string         `json:"route_name,omitempty"`
	Header           map[string]any `json:"header"`
	Models           []string       `json:"models"`
	PartNumbers      []string       `json:"part_numbers"`
	// PartNames: nama part setiap PartNumbers, untuk cover dan kepala cetakan
	// (bagian C). Kosong di snapshot lama.
	PartNames []string `json:"part_names,omitempty"`
	// LogoSHA256: logo yang berlaku saat snapshot dibuat (D-223). Revisi terbit
	// selalu dicetak dengan logo ini. Kosong di snapshot sebelum D-223.
	LogoSHA256 string            `json:"logo_sha256,omitempty"`
	Sections   []SnapshotSection `json:"sections"`
	Rows       []SnapshotRow     `json:"rows"`
	// Flow: langkah PFD yang dipakai menggambar Process Flow CP (D-212).
	// Kosong di snapshot lama dan di dokumen selain CP.
	Flow []FlowStep `json:"flow,omitempty"`
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
	text, err := json.Marshal(buildSnapshot(d))
	if err != nil {
		return nil, "", fmt.Errorf("menulis snapshot: %w", err)
	}
	sum := sha256.Sum256(text)
	return text, hex.EncodeToString(sum[:]), nil
}

// buildSnapshot menyusun isi dokumen dalam bentuk snapshot.
func buildSnapshot(d Document) Snapshot {
	s := Snapshot{
		DocumentID: d.ID, Number: d.Number, DocumentType: d.DocumentType.Key, CustomerCode: d.CustomerCode, Revision: d.Revision,
		FormSetVersionID: d.FormSetVersionID, FormatVersionID: d.FormatVersionID, Header: d.Header,
		Models: []string{}, PartNumbers: []string{}, Sections: []SnapshotSection{}, Rows: []SnapshotRow{},
		Flow: d.Flow, LogoSHA256: d.LogoSHA256,
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
	named := false
	for _, p := range d.PartNumbers {
		s.PartNumbers = append(s.PartNumbers, p.Number)
		named = named || p.Name != ""
	}
	if named {
		for _, p := range d.PartNumbers {
			s.PartNames = append(s.PartNames, p.Name)
		}
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
	return s
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
		// Draft yang sudah ditandatangani digital boleh dirilis, jadi hanya
		// statusnya yang diperiksa di sini; tanda tangannya di bawah.
		if d.Status != StatusDraft {
			return problems(invalid("status", problem.NotAllowed, fmt.Sprintf("%s has no open draft.", d.Number)))
		}
		// Baseline migrasi terbit lewat verifikasi hardcopy, bukan rilis biasa (R-MIG-06).
		if d.Baseline != nil && d.Baseline.VerifiedAt == nil {
			return problems(invalid("status", problem.NotAllowed, "A migration baseline is released by verifying it against the hard copy."))
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
		// Tanda tangan digital (D-198): rilis butuh semua slot bertanda tangan,
		// dan isinya harus sama dengan isi yang ditandatangani. Tanpa scan.
		var signatureID *string
		switch d.Signing {
		case SigningWaiting:
			return problems(invalid("signatures", problem.NotAllowed, "The draft is still waiting for signatures."))
		case SigningCompleted:
			sig, err := q.ActiveSignatureRequestOfDraft(ctx, id)
			if err != nil {
				return fmt.Errorf("membaca tanda tangan: %w", err)
			}
			if sig.ContentSha256 != sum {
				return problems(invalid("signatures", problem.NotAllowed, "The content differs from the signed content. Send it for signature again."))
			}
			signatureID = &sig.ID
		default:
			// Tanda tangan basah: rilis dicatat sesudah scan bertanda tangan
			// diunggah (M6.7, R-APR-02).
			info, err := q.GetRevisionPrintInfo(ctx, db.GetRevisionPrintInfoParams{DocumentID: id, Revision: draft.Revision})
			if err != nil {
				return fmt.Errorf("membaca draft: %w", err)
			}
			if info.ScanSha256 == nil {
				return problems(invalid("scan", problem.Required, "Upload the signed scan before recording the release."))
			}
		}
		snapshot := string(text)
		n, err := q.ReleaseDocumentRevision(ctx, db.ReleaseDocumentRevisionParams{
			ID: draft.ID, Snapshot: &snapshot, Sha256: &sum, PersonID: &actor.PersonID, AccountID: &actor.AccountID,
			SignatureRequestID: signatureID,
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
			ReleasedBy: deref(r.ReleasedBy), SHA256: deref(r.SnapshotSha256), Changes: []string{}, ScanFileName: deref(r.ScanFileName), Migrated: r.Migrated,
		}
		if r.ReleaseDate.Valid {
			item.ReleaseDate = r.ReleaseDate.Time.Format(time.DateOnly)
		}
		changes, err := s.q.ListRevisionChanges(ctx, &r.ID)
		if err != nil {
			return nil, fmt.Errorf("membaca perubahan revisi: %w", err)
		}
		for _, c := range changes {
			// Bentuk nomor sama dengan internal/change (ASUMSI E-98).
			if c.Seq != nil {
				item.Changes = append(item.Changes, fmt.Sprintf("CHG-%04d", *c.Seq))
			}
		}
		if r.ReleasedAt != nil {
			at := r.ReleasedAt.UTC()
			item.ReleasedAt = &at
		}
		if r.ReadyAt != nil {
			at := r.ReadyAt.UTC()
			item.ReadyAt = &at
		}
		out = append(out, item)
	}
	return out, nil
}

// CurrentSnapshot memberi isi dokumen sekarang dalam bentuk snapshot, seperti
// yang akan dibekukan saat rilis. Dipakai cetak draft (Tahap 6).
func CurrentSnapshot(d Document) (Snapshot, []byte, error) {
	text, sum, err := makeSnapshot(d)
	if err != nil {
		return Snapshot{}, nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(text, &s); err != nil {
		return Snapshot{}, nil, err
	}
	_ = sum
	return s, text, nil
}

// ParseSnapshot membaca teks snapshot revisi terbit.
func ParseSnapshot(text string) (Snapshot, error) {
	var s Snapshot
	err := json.Unmarshal([]byte(text), &s)
	return s, err
}
