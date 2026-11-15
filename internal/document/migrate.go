package document

// Migrasi dokumen lama (Tahap 7; R-MIG-03, R-MIG-06, R-REV-01, R-REV-05).
//
// Dokumen lama masuk sebagai "baseline migrasi": draft di revisi yang
// tertulis di hardcopy (misalnya -04), bersama tanggal berlaku, penyusun, dan
// approver sebagai teks. Isinya diketik atau dirakit di layar yang sama
// dengan dokumen baru (R-CMP-06). Riwayat revisi sebelum baseline disimpan
// sebagai teks. Sesudah orang lain mencocokkannya dengan hardcopy bertanda
// tangan, revisi itu terbit sebagai Released (migrated) dan dibekukan seperti
// revisi terbit lain. Semua ini hanya jalan selama mode migrasi menyala.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Asal dokumen (brief bagian 4).
const (
	OriginNew      = "new"
	OriginMigrated = "migrated"
)

const (
	maxLegacyRevision = 999
	maxLegacyName     = 200
	maxLegacyApprover = 500
	maxHistoryEntries = 200
	maxHistoryLabel   = 20
	maxHistoryText    = 1000
	maxEffortMinutes  = 100000
)

// LegacyDocument adalah isian form dokumen lama (M7.1).
type LegacyDocument struct {
	NewDocument
	Revision      int    // revisi saat ini di hardcopy, misalnya 4 untuk -04
	EffectiveDate string // YYYY-MM-DD
	PreparedBy    string // nama penyusun di hardcopy
	ApprovedBy    string // nama approver di hardcopy
}

// Baseline adalah data hardcopy satu dokumen migrasi.
type Baseline struct {
	Revision      int
	EffectiveDate string
	PreparedBy    string
	ApprovedBy    string
	CreatedBy     string
	CreatedAt     time.Time
	VerifiedBy    string     // "" sebelum diverifikasi
	VerifiedAt    *time.Time // nil sebelum diverifikasi
	EffortMinutes *int       // menit kerja migrasi (M7.6); nil bila tidak diisi
	History       []HistoryEntry
}

// HistoryEntry adalah satu baris riwayat revisi dari tabel Excel lama (R-REV-05).
type HistoryEntry struct {
	Revision    string // teks seperti di tabel lama, misalnya -03
	Date        string // YYYY-MM-DD, boleh kosong
	Description string
}

// MigratableNumber adalah satu nomor register yang belum punya dokumen.
type MigratableNumber struct {
	ID           string
	Number       string
	Status       string
	DocumentType string
}

// requireMigrationMode menolak bila saklar mode migrasi mati (R-MIG-06).
func requireMigrationMode(ctx context.Context, q *db.Queries) error {
	on, err := q.GetMigrationMode(ctx)
	if err != nil {
		return fmt.Errorf("membaca mode migrasi: %w", err)
	}
	if !on {
		return problems(invalid("migration_mode", problem.NotAllowed, "Migration mode is off. Ask the admin to switch it on in Settings."))
	}
	return nil
}

// MigratableNumbers memberi nomor satu customer yang bisa dipakai dokumen
// lama (E-10): belum punya dokumen, dan bukan Void, Obsolete, atau Skipped.
func (s *Service) MigratableNumbers(ctx context.Context, actor access.Actor, customerID string) ([]MigratableNumber, error) {
	if !db.IsUUID(customerID) {
		return nil, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListMigratableNumbers(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("membaca nomor: %w", err)
	}
	out := make([]MigratableNumber, 0, len(rows))
	for _, r := range rows {
		out = append(out, MigratableNumber{ID: r.ID, Number: r.Number, Status: r.Status, DocumentType: r.DocumentTypeName})
	}
	return out, nil
}

// parseDate membaca tanggal YYYY-MM-DD.
func parseDate(field, value string, required bool) (pgtype.Date, *problem.Problem) {
	if value == "" {
		if required {
			return pgtype.Date{}, invalid(field, problem.Required, "Enter a date.")
		}
		return pgtype.Date{}, nil
	}
	day, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return pgtype.Date{}, invalid(field, problem.InvalidFormat, "Enter the date as YYYY-MM-DD.")
	}
	return pgtype.Date{Time: day, Valid: true}, nil
}

// Migrate memasukkan satu dokumen lama sebagai baseline migrasi (E-10, M7.1).
func (s *Service) Migrate(ctx context.Context, actor access.Actor, in LegacyDocument) (Document, error) {
	if in.NumberID == "" {
		return Document{}, problems(invalid("document_number_id", problem.Required, "Choose the document number."))
	}
	if !db.IsUUID(in.NumberID) {
		return Document{}, problems(invalid("document_number_id", problem.InvalidFormat, "Choose an existing document number."))
	}
	number, err := s.q.GetNumberForDocument(ctx, in.NumberID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, problems(invalid("document_number_id", problem.InvalidFormat, "Choose an existing document number."))
	}
	if err != nil {
		return Document{}, fmt.Errorf("membaca nomor: %w", err)
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: number.CustomerID}); err != nil {
		return Document{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := requireMigrationMode(ctx, q); err != nil {
			return err
		}
		number, err := q.GetNumberForDocument(ctx, in.NumberID)
		if err != nil {
			return err
		}
		used, err := q.DocumentExistsForNumber(ctx, number.ID)
		if err != nil {
			return err
		}
		if used || (number.Status != "reserved" && number.Status != "in_use" && number.Status != "released") {
			return problems(invalid("document_number_id", problem.NotAllowed, "Choose a number that no document uses yet and that is not Void, Obsolete, or Skipped."))
		}
		customer, err := q.GetCustomerForDocument(ctx, number.CustomerID)
		if err != nil {
			return err
		}
		list := s.checkScope(ctx, q, number.CustomerID, in.NewDocument)
		route, reason, routeProblem := checkRoute(ctx, q, customer.DefaultRouteID, in.NewDocument)
		list = append(list, routeProblem)
		set, setProblem := formSetOf(ctx, q, customer.DefaultFormSetID)
		list = append(list, setProblem)
		if in.Revision < 0 || in.Revision > maxLegacyRevision {
			list = append(list, invalid("revision", problem.InvalidFormat, fmt.Sprintf("Enter the current revision as a number from 0 to %d.", maxLegacyRevision)))
		}
		date, dateProblem := parseDate("effective_date", in.EffectiveDate, true)
		list = append(list, dateProblem)
		prepared, p := problem.Text("prepared_by", in.PreparedBy, true, maxLegacyName)
		list = append(list, p)
		approved, p := problem.Text("approved_by", in.ApprovedBy, true, maxLegacyApprover)
		list = append(list, p)
		if err := problems(list...); err != nil {
			return err
		}
		formatID := map[string]string{"pfd": set.PfdFormatVersionID, "pfmea": set.PfmeaFormatVersionID, "cp": set.CpFormatVersionID}[deref(number.DocumentTypeKey)]
		if formatID == "" {
			return problems(invalid("document_number_id", problem.NotAllowed, "Only PFD, PFMEA, and CP numbers make documents."))
		}

		revision := int16(in.Revision)
		id, err = q.CreateDocument(ctx, db.CreateDocumentParams{
			DocumentNumberID: number.ID, CustomerID: number.CustomerID, DocumentTypeID: number.DocumentTypeID,
			Revision: revision, FormSetVersionID: set.ID, FormatVersionID: formatID,
			ApprovalRouteID: route, RouteReason: optional(reason), CreatedByPersonID: actor.PersonID, Origin: OriginMigrated,
		})
		if err != nil {
			return err
		}
		// Dokumen lama mulai di revisinya yang sekarang (R-REV-01).
		if _, err := q.CreateDocumentRevision(ctx, db.CreateDocumentRevisionParams{DocumentID: id, Revision: revision, CreatedByPersonID: actor.PersonID}); err != nil {
			return err
		}
		if err := q.CreateDocumentBaseline(ctx, db.CreateDocumentBaselineParams{
			DocumentID: id, Revision: revision, EffectiveDate: date, PreparedBy: prepared, ApprovedBy: approved, CreatedByPersonID: actor.PersonID,
		}); err != nil {
			return err
		}
		for _, m := range in.ModelIDs {
			if err := q.AddDocumentModel(ctx, db.AddDocumentModelParams{DocumentID: id, ModelID: m}); err != nil {
				return err
			}
		}
		for _, p := range in.PartNumberIDs {
			if err := q.AddDocumentPartNumber(ctx, db.AddDocumentPartNumberParams{DocumentID: id, PartNumberID: p}); err != nil {
				return err
			}
		}
		// Nomor Reserved menjadi In use; status nomor lama hasil impor tetap.
		if number.Status == "reserved" {
			now := time.Now()
			if err := q.SetDocumentNumberStatus(ctx, db.SetDocumentNumberStatusParams{ID: number.ID, Status: "in_use", StatusChangedAt: &now}); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentMigrated, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"document_number": number.Number, "revision": strconv.Itoa(in.Revision), "effective_date": in.EffectiveDate}})
	})
	if err != nil {
		return Document{}, result(err, "memasukkan dokumen lama")
	}
	return getDocument(ctx, s.q, id)
}

// openBaseline memeriksa bahwa dokumen adalah baseline migrasi yang belum
// diverifikasi dan masih draft.
func openBaseline(d Document) error {
	if d.Origin != OriginMigrated || d.Baseline == nil {
		return problems(invalid("status", problem.NotAllowed, fmt.Sprintf("%s is not a migrated document.", d.Number)))
	}
	if d.Baseline.VerifiedAt != nil || d.Status != StatusDraft {
		return problems(invalid("status", problem.NotAllowed, fmt.Sprintf("The baseline of %s is already verified against the hard copy.", d.Number)))
	}
	return nil
}

// SetHistory mengganti seluruh riwayat revisi teks satu dokumen migrasi
// (E-10, M7.2). Urutan daftar adalah urutan cetak.
func (s *Service) SetHistory(ctx context.Context, actor access.Actor, id string, entries []HistoryEntry) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := requireMigrationMode(ctx, q); err != nil {
			return err
		}
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if err := openBaseline(d); err != nil {
			return err
		}
		if len(entries) > maxHistoryEntries {
			return problems(invalid("entries", problem.TooLong, fmt.Sprintf("Enter at most %d history entries.", maxHistoryEntries)))
		}
		type clean struct {
			label, text string
			date        pgtype.Date
		}
		var list []*problem.Problem
		cleaned := make([]clean, 0, len(entries))
		for i, e := range entries {
			field := "entries." + strconv.Itoa(i)
			label, p1 := problem.Text(field+".revision", e.Revision, true, maxHistoryLabel)
			date, p2 := parseDate(field+".date", e.Date, false)
			text, p3 := problem.Text(field+".description", e.Description, true, maxHistoryText)
			list = append(list, p1, p2, p3)
			cleaned = append(cleaned, clean{label: label, text: text, date: date})
		}
		if err := problems(list...); err != nil {
			return err
		}
		if err := q.DeleteHistoryEntries(ctx, id); err != nil {
			return err
		}
		for i, c := range cleaned {
			if err := q.AddHistoryEntry(ctx, db.AddHistoryEntryParams{
				DocumentID: id, Position: int32(i + 1), RevisionLabel: c.label, EntryDate: c.date, Description: c.text, CreatedByPersonID: actor.PersonID,
			}); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentHistoryChanged, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"number": d.Number, "entries": strconv.Itoa(len(cleaned))}})
	})
	if err != nil {
		return Document{}, result(err, "menyimpan riwayat revisi")
	}
	return s.load(ctx, id)
}

// VerifyMigration mencatat bahwa isi baseline sudah dicocokkan dengan
// hardcopy bertanda tangan (E-10, M7.5). Revisi itu terbit sebagai Released
// (migrated) dengan snapshot dan SHA-256. Pemeriksa bukan orang yang
// memasukkan atau mengisi dokumen itu (R-SEC-11, D-177). Scan hardcopy
// tidak wajib (D-186).
// effortMinutes adalah menit kerja migrasi dokumen ini (M7.6), boleh nil.
func (s *Service) VerifyMigration(ctx context.Context, actor access.Actor, id string, effortMinutes *int) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := requireMigrationMode(ctx, q); err != nil {
			return err
		}
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if err := openBaseline(d); err != nil {
			return err
		}
		prepared, err := q.PreparedDraft(ctx, db.PreparedDraftParams{DocumentID: &id, PersonID: &actor.PersonID})
		if err != nil {
			return err
		}
		if prepared {
			return problems(invalid("migration_verifier", problem.NotAllowed, "The person who entered the document cannot verify it against the hard copy. Ask another migration officer."))
		}
		var effort *int32
		if effortMinutes != nil {
			if *effortMinutes < 1 || *effortMinutes > maxEffortMinutes {
				return problems(invalid("effort_minutes", problem.InvalidFormat, fmt.Sprintf("Enter the minutes spent, from 1 to %d, or leave it empty.", maxEffortMinutes)))
			}
			v := int32(*effortMinutes)
			effort = &v
		}
		draft, err := q.GetOpenDraftRevision(ctx, id)
		if err != nil {
			return fmt.Errorf("membaca draft: %w", err)
		}
		// Scan hardcopy boleh diunggah sebagai bukti tambahan, tetapi tidak wajib:
		// yang menentukan adalah verifikasi orang kedua (D-186).
		info, err := q.GetRevisionPrintInfo(ctx, db.GetRevisionPrintInfoParams{DocumentID: id, Revision: draft.Revision})
		if err != nil {
			return fmt.Errorf("membaca draft: %w", err)
		}
		text, sum, err := makeSnapshot(d)
		if err != nil {
			return err
		}
		snapshot := string(text)
		date, _ := parseDate("effective_date", d.Baseline.EffectiveDate, true)
		n, err := q.ReleaseMigratedRevision(ctx, db.ReleaseMigratedRevisionParams{
			ID: draft.ID, ReleaseDate: date, Snapshot: &snapshot, Sha256: &sum, PersonID: &actor.PersonID, AccountID: &actor.AccountID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return problems(invalid("status", problem.NotAllowed, "The baseline is already verified."))
		}
		if err := q.MarkDocumentReleased(ctx, db.MarkDocumentReleasedParams{ID: id, ReleasedRevision: &draft.Revision}); err != nil {
			return err
		}
		if _, err := q.VerifyDocumentBaseline(ctx, db.VerifyDocumentBaselineParams{DocumentID: id, VerifiedByPersonID: &actor.PersonID, EffortMinutes: effort}); err != nil {
			return err
		}
		details := map[string]string{"number": d.Number, "revision": strconv.Itoa(int(draft.Revision)), "sha256": sum, "scan": strconv.FormatBool(info.ScanSha256 != nil)}
		if effort != nil {
			details["effort_minutes"] = strconv.Itoa(int(*effort))
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentMigrationVerified, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return Document{}, result(err, "memverifikasi baseline")
	}
	return s.load(ctx, id)
}

// loadBaseline membaca baseline dan riwayat teks satu dokumen migrasi.
func loadBaseline(ctx context.Context, q *db.Queries, id string) (*Baseline, error) {
	b, err := q.GetDocumentBaseline(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("membaca baseline: %w", err)
	}
	out := &Baseline{
		Revision: int(b.Revision), EffectiveDate: b.EffectiveDate.Time.Format(time.DateOnly), PreparedBy: b.PreparedBy, ApprovedBy: b.ApprovedBy,
		CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt.UTC(), VerifiedBy: deref(b.VerifiedBy), History: []HistoryEntry{},
	}
	if b.VerifiedAt != nil {
		at := b.VerifiedAt.UTC()
		out.VerifiedAt = &at
	}
	if b.EffortMinutes != nil {
		v := int(*b.EffortMinutes)
		out.EffortMinutes = &v
	}
	entries, err := q.ListHistoryEntries(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("membaca riwayat: %w", err)
	}
	for _, e := range entries {
		item := HistoryEntry{Revision: e.RevisionLabel, Description: e.Description}
		if e.EntryDate.Valid {
			item.Date = e.EntryDate.Time.Format(time.DateOnly)
		}
		out.History = append(out.History, item)
	}
	return out, nil
}
