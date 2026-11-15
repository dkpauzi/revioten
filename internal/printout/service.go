package printout

// Layanan cetak (Tahap 6): memilih revisi dan tujuan cetak, memeriksa hak,
// membentuk model cetak, menulis Excel atau PDF, mencatat setiap cetakan
// (M6.6), menyimpan scan bertanda tangan (M6.7), mengatur layout (M6.1), dan
// halaman verifikasi (M6.5).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
	"github.com/dkpauzi/revioten/internal/template"
)

// Tujuan cetak (R-PRT-04).
const (
	PurposeDraft      = "draft"      // draft untuk dibaca; UNCONTROLLED COPY
	PurposeSignature  = "signature"  // draft untuk tanda tangan basah; mengisi tanggal rilis (D-59)
	PurposeReference  = "reference"  // revisi terbit untuk referensi; UNCONTROLLED COPY
	PurposeControlled = "controlled" // salinan terkendali bernomor; CONTROLLED COPY No. n
)

// MaxScanBytes adalah ukuran scan bertanda tangan paling besar (ASUMSI E-109).
const MaxScanBytes = 30 << 20

// Request adalah permintaan satu cetakan.
type Request struct {
	Revision *int   // nil berarti revisi yang dikerjakan atau yang berlaku
	Purpose  string // draft, signature, reference, atau controlled
	Format   string // xlsx atau pdf
	BaseURL  string // alamat aplikasi untuk QR, misalnya http://127.0.0.1:8080
}

// File adalah hasil cetak.
type File struct {
	Name        string
	ContentType string
	Body        []byte
}

// Log adalah satu catatan cetak.
type Log struct {
	ID         string
	Revision   int
	Purpose    string
	Format     string
	CopyNumber *int
	SHA256     string
	PrintedBy  string
	PrintedAt  time.Time
}

// Service menjalankan operasi cetak.
type Service struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	authz   access.Authorizer
	docs    *document.Service
	store   *storage.Store
	browser string // kosong: dicari saat dibutuhkan
	baseURL string // alamat aplikasi untuk QR (M6.5)
}

// NewService membuat layanan cetak. browser boleh kosong (dicari otomatis).
func NewService(pool *pgxpool.Pool, authz access.Authorizer, docs *document.Service, store *storage.Store, browser string) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, docs: docs, store: store, browser: browser}
}

// WithBaseURL mengisi alamat aplikasi yang dipakai QR, misalnya http://192.168.1.10:8080.
func (s *Service) WithBaseURL(url string) *Service {
	s.baseURL = url
	return s
}

// Print membuat satu cetakan dan mencatatnya.
func (s *Service) Print(ctx context.Context, actor access.Actor, id string, req Request) (File, error) {
	d, err := s.docs.Get(ctx, actor, id)
	if err != nil {
		return File{}, err
	}
	if req.Format != "xlsx" && req.Format != "pdf" {
		return File{}, invalid("format", problem.InvalidFormat, "Choose Excel or PDF.")
	}
	// Revisi yang dicetak: bawaannya draft yang terbuka, atau revisi yang berlaku.
	revision := d.Revision
	if req.Revision != nil {
		revision = *req.Revision
	}
	info, err := s.q.GetRevisionPrintInfo(ctx, db.GetRevisionPrintInfoParams{DocumentID: id, Revision: int16(revision)})
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, invalid("revision", problem.NotAllowed, fmt.Sprintf("Revision %d does not exist.", revision))
	}
	if err != nil {
		return File{}, err
	}
	released := info.Status == "released"
	need := map[string]access.Permission{PurposeDraft: access.ViewDrafts, PurposeSignature: access.PrintForSignature, PurposeReference: access.ExportReleased, PurposeControlled: access.ControlledCopies}[req.Purpose]
	switch {
	case need == "":
		return File{}, invalid("purpose", problem.InvalidFormat, "Choose what the print is for.")
	case released && (req.Purpose == PurposeDraft || req.Purpose == PurposeSignature):
		return File{}, invalid("purpose", problem.NotAllowed, "A released revision prints as a reference or a controlled copy.")
	case !released && (req.Purpose == PurposeReference || req.Purpose == PurposeControlled):
		return File{}, invalid("purpose", problem.NotAllowed, "A draft prints as a draft or for signature.")
	}
	if err := s.authz.Authorize(ctx, actor, need, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return File{}, err
	}

	// Isi: snapshot revisi terbit, atau isi draft sekarang.
	var snap document.Snapshot
	var hash string
	if released {
		if snap, err = document.ParseSnapshot(deref(info.Snapshot)); err != nil {
			return File{}, fmt.Errorf("membaca snapshot: %w", err)
		}
		hash = deref(info.SnapshotSha256)
	} else {
		var text []byte
		if snap, text, err = document.CurrentSnapshot(d); err != nil {
			return File{}, err
		}
		sum := sha256.Sum256(text)
		hash = hex.EncodeToString(sum[:])
	}
	if req.Purpose == PurposeSignature && snap.RouteID == "" {
		// Tanpa rute, cetak untuk tanda tangan ditolak (R-APR-09, D-116).
		return File{}, invalid("route", problem.Required, "Choose the approval route of the document before printing for signature.")
	}
	in, err := s.input(ctx, d, snap, info, revision, hash, req)
	if err != nil {
		return File{}, err
	}

	var copyNumber *int32
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if req.Purpose == PurposeControlled {
			n, err := q.NextCopyNumber(ctx, info.ID)
			if err != nil {
				return err
			}
			copyNumber = &n
			in.Watermark = fmt.Sprintf("CONTROLLED COPY No. %d", n)
		}
		if req.Purpose == PurposeSignature {
			// Tanggal rilis diisi saat dicetak untuk tanda tangan (D-59).
			today := time.Now()
			if err := q.SetRevisionReleaseDate(ctx, db.SetRevisionReleaseDateParams{ID: info.ID, ReleaseDate: pgtype.Date{Time: today, Valid: true}}); err != nil {
				return err
			}
			in.ReleaseDate = today.Format(time.DateOnly)
		}
		if _, err := q.AddPrintLog(ctx, db.AddPrintLogParams{
			DocumentID: id, RevisionID: info.ID, Revision: int16(revision), Purpose: req.Purpose, Format: req.Format,
			CopyNumber: copyNumber, ContentSha256: hash, PrintedByPersonID: actor.PersonID, PrintedByAccountID: actor.AccountID,
		}); err != nil {
			return err
		}
		details := map[string]string{"number": d.Number, "revision": strconv.Itoa(revision), "purpose": req.Purpose, "format": req.Format}
		if copyNumber != nil {
			details["copy"] = strconv.Itoa(int(*copyNumber))
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentPrinted, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return File{}, fmt.Errorf("mencatat cetakan: %w", err)
	}

	sheet := Build(in)
	name := fmt.Sprintf("%s%s-%s", d.Number, in.RevisionLabel, req.Purpose)
	if copyNumber != nil {
		name += fmt.Sprintf("-%d", *copyNumber)
	}
	if req.Format == "xlsx" {
		body, err := XLSX(sheet)
		if err != nil {
			return File{}, fmt.Errorf("menulis Excel: %w", err)
		}
		return File{Name: name + ".xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Body: body}, nil
	}
	browser := s.browser
	if browser == "" {
		if browser, err = FindBrowser(""); err != nil {
			return File{}, invalid("format", problem.NotAllowed, "PDF is not available on this computer: no Chrome or Edge was found. Download Excel instead.")
		}
	}
	html, err := HTML(sheet)
	if err != nil {
		return File{}, err
	}
	body, err := PDF(ctx, browser, html)
	if err != nil {
		return File{}, err
	}
	return File{Name: name + ".pdf", ContentType: "application/pdf", Body: body}, nil
}

// input mengumpulkan bahan cetak dari database.
func (s *Service) input(ctx context.Context, d document.Document, snap document.Snapshot, info db.GetRevisionPrintInfoRow, revision int, hash string, req Request) (Input, error) {
	format, err := template.LoadFormatVersion(ctx, s.q, snap.FormatVersionID)
	if err != nil {
		return Input{}, err
	}
	row, err := s.q.GetDocumentLayout(ctx, db.GetDocumentLayoutParams{CustomerID: d.CustomerID, DocumentTypeID: d.DocumentType.ID})
	if err != nil {
		return Input{}, fmt.Errorf("membaca layout: %w", err)
	}
	layout, err := ParseLayout(row.Layout)
	if err != nil {
		return Input{}, err
	}
	customer, err := s.q.GetCustomer(ctx, d.CustomerID)
	if err != nil {
		return Input{}, err
	}
	digits, err := s.q.GetRevisionDigits(ctx)
	if err != nil {
		digits = 2
	}
	label := func(n int) string { return fmt.Sprintf("-%0*d", digits, n) }

	in := Input{
		Snapshot: snap, Format: format, Layout: layout, CustomerName: customer.Name, RevisionLabel: label(revision),
		ContentHash: hash, Watermark: "UNCONTROLLED COPY", Status: "DRAFT",
	}
	if info.ReleaseDate.Valid {
		in.ReleaseDate = info.ReleaseDate.Time.Format(time.DateOnly)
	}
	if info.Status == "released" {
		in.Status = "SUPERSEDED"
		if d.ReleasedRevision != nil && *d.ReleasedRevision == revision {
			in.Status = "EFFECTIVE"
		}
		// Catatan rilis pengganti tanda tangan di cetakan sistem (M6.9, R-PUB-06).
		in.ReleaseNote = fmt.Sprintf("Released %s, recorded by %s, record %s, signed scan on file", in.ReleaseDate, deref(info.ReleasedBy), info.ID[:8])
		if info.Migrated {
			in.ReleaseNote = fmt.Sprintf("Released (migrated) %s, verified against the signed hard copy by %s, record %s, scan on file", in.ReleaseDate, deref(info.ReleasedBy), info.ID[:8])
		}
	}
	// Tanda revisi dibanding revisi terbit sebelumnya (R-REV-09).
	for prev := revision - 1; prev >= 0; prev-- {
		p, err := s.q.GetRevisionPrintInfo(ctx, db.GetRevisionPrintInfoParams{DocumentID: d.ID, Revision: int16(prev)})
		if err != nil || p.Snapshot == nil {
			continue
		}
		if ps, err := document.ParseSnapshot(*p.Snapshot); err == nil {
			in.Previous = &ps
		}
		break
	}
	if snap.RouteID != "" {
		slots, err := s.q.ListRouteSlotHolders(ctx, db.ListRouteSlotHoldersParams{RouteID: snap.RouteID, CustomerID: d.CustomerID})
		if err != nil {
			return Input{}, fmt.Errorf("membaca rute: %w", err)
		}
		for _, sl := range slots {
			in.Slots = append(in.Slots, Slot{Title: sl.PrintTitle, Stage: sl.Stage, Names: sl.Holders})
		}
	}
	// Riwayat revisi sampai revisi yang dicetak (M6.8).
	history, err := s.q.ListRevisionHistory(ctx, d.ID)
	if err != nil {
		return Input{}, err
	}
	// Riwayat sebelum baseline migrasi berasal dari teks tabel lama (M7.2,
	// R-REV-05). Bila teks itu ada, baris revisi baseline sendiri tidak
	// dibuat sistem, karena tabel lama sudah memuatnya.
	baselineRevision := -1
	if d.Baseline != nil {
		for _, e := range d.Baseline.History {
			in.History = append(in.History, HistoryRow{No: len(in.History) + 1, Date: e.Date, Revision: e.Revision, Description: e.Description})
		}
		if len(d.Baseline.History) > 0 {
			baselineRevision = d.Baseline.Revision
		}
	}
	for _, h := range history {
		if int(h.Revision) > revision || int(h.Revision) <= baselineRevision {
			continue
		}
		date := ""
		switch {
		case h.ReleaseDate.Valid:
			date = h.ReleaseDate.Time.Format(time.DateOnly)
		case h.ReleasedAt != nil:
			date = h.ReleasedAt.Format(time.DateOnly)
		}
		desc := h.Description
		if desc == "" && d.Baseline != nil && int(h.Revision) == d.Baseline.Revision {
			desc = "Migrated baseline"
		}
		if desc == "" && len(in.History) == 0 {
			desc = "Initial release"
		}
		in.History = append(in.History, HistoryRow{No: len(in.History) + 1, Date: date, Revision: label(int(h.Revision)), Description: desc})
	}
	// QR menuju halaman verifikasi bila saklarnya menyala (M6.5, D-62).
	if req.BaseURL == "" {
		req.BaseURL = s.baseURL
	}
	if on, err := s.q.GetQREnabled(ctx); err == nil && on && req.BaseURL != "" {
		in.VerifyURL = strings.TrimRight(req.BaseURL, "/") + "/verify?id=" + info.ID
		if in.QRPNG, err = QRPNG(in.VerifyURL); err != nil {
			return Input{}, err
		}
	}
	return in, nil
}

// Logs memberi catatan cetak satu dokumen, yang terbaru dulu (L-4).
func (s *Service) Logs(ctx context.Context, actor access.Actor, id string) ([]Log, error) {
	if _, err := s.docs.Get(ctx, actor, id); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPrintLogs(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Log, 0, len(rows))
	for _, r := range rows {
		item := Log{ID: r.ID, Revision: int(r.Revision), Purpose: r.Purpose, Format: r.Format, SHA256: r.ContentSha256, PrintedBy: r.PrintedBy, PrintedAt: r.PrintedAt.UTC()}
		if r.CopyNumber != nil {
			n := int(*r.CopyNumber)
			item.CopyNumber = &n
		}
		out = append(out, item)
	}
	return out, nil
}

// scanTypes adalah jenis berkas scan yang diterima, menurut akhiran.
var scanTypes = map[string]string{".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg"}

// UploadScan menyimpan scan bertanda tangan untuk draft terbuka (E-7, M6.7).
// Scan baru menggantikan scan sebelumnya selama belum dirilis.
func (s *Service) UploadScan(ctx context.Context, actor access.Actor, id, fileName string, r io.Reader) (Scan, error) {
	d, err := s.docs.Get(ctx, actor, id)
	if err != nil {
		return Scan{}, err
	}
	// Scan hardcopy baseline migrasi diunggah petugas migrasi (E-10, M7.5).
	permission := access.RecordRelease
	if d.Origin == document.OriginMigrated && d.Baseline != nil && d.Baseline.VerifiedAt == nil {
		permission = access.MigrateDocuments
	}
	if err := s.authz.Authorize(ctx, actor, permission, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Scan{}, err
	}
	if d.Status != document.StatusDraft {
		return Scan{}, invalid("status", problem.NotAllowed, "Only a draft that waits for its release takes a signed scan.")
	}
	name := cleanName(fileName)
	contentType, ok := scanTypes[strings.ToLower(filepath.Ext(name))]
	if name == "" || !ok {
		return Scan{}, invalid("file", problem.NotAllowed, "Upload the signed scan as PDF, PNG, or JPEG.")
	}
	if s.store == nil {
		return Scan{}, errors.New("penyimpanan scan belum dipasang")
	}
	sum, size, err := s.store.PutScanned(ctx, r, MaxScanBytes)
	switch {
	case errors.Is(err, storage.ErrEmpty):
		return Scan{}, invalid("file", problem.Required, "The file is empty.")
	case errors.Is(err, storage.ErrTooLarge):
		return Scan{}, invalid("file", problem.TooLong, "A scan can be at most 30 MB.")
	case errors.Is(err, storage.ErrInfected):
		// Nama temuan hanya masuk log server, bukan ke layar (R-API-13).
		return Scan{}, invalid("file", problem.NotAllowed, "The antivirus scanner found malware in this file. It was not stored.")
	case errors.Is(err, storage.ErrScanUnavailable):
		return Scan{}, invalid("file", problem.NotAllowed, "The antivirus scanner cannot be reached, so the file was not stored. Try again or tell the admin.")
	case err != nil:
		return Scan{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		draft, err := q.GetOpenDraftRevision(ctx, id)
		if err != nil {
			return invalid("status", problem.NotAllowed, "The document has no open draft.")
		}
		n, err := q.SetRevisionScan(ctx, db.SetRevisionScanParams{ID: draft.ID, ScanFileName: &name, ScanContentType: &contentType, ScanSizeBytes: &size, ScanSha256: &sum, ScanUploadedByPersonID: &actor.PersonID})
		if err != nil {
			return err
		}
		if n == 0 {
			return invalid("status", problem.NotAllowed, "The document has no open draft.")
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentScanUploaded, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"number": d.Number, "revision": strconv.Itoa(int(draft.Revision)), "file": name, "sha256": sum}})
	})
	if err != nil {
		return Scan{}, fmt.Errorf("menyimpan scan: %w", err)
	}
	return s.ScanOf(ctx, actor, id, d.Revision)
}

// Scan adalah scan bertanda tangan satu revisi.
type Scan struct {
	Revision    int
	FileName    string
	ContentType string
	Size        int64
	SHA256      string
	UploadedBy  string
	UploadedAt  time.Time
}

// ScanOf membaca keterangan scan satu revisi (L-4). Tanpa scan: ErrNotFound.
func (s *Service) ScanOf(ctx context.Context, actor access.Actor, id string, revision int) (Scan, error) {
	if _, err := s.docs.Get(ctx, actor, id); err != nil {
		return Scan{}, err
	}
	info, err := s.q.GetRevisionPrintInfo(ctx, db.GetRevisionPrintInfoParams{DocumentID: id, Revision: int16(revision)})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && info.ScanSha256 == nil) {
		return Scan{}, problem.ErrNotFound
	}
	if err != nil {
		return Scan{}, err
	}
	out := Scan{Revision: revision, FileName: deref(info.ScanFileName), ContentType: deref(info.ScanContentType), SHA256: *info.ScanSha256, UploadedBy: deref(info.ScanUploadedBy)}
	if info.ScanSizeBytes != nil {
		out.Size = *info.ScanSizeBytes
	}
	if info.ScanUploadedAt != nil {
		out.UploadedAt = info.ScanUploadedAt.UTC()
	}
	return out, nil
}

// OpenScan membuka berkas scan satu revisi untuk diunduh (L-4).
func (s *Service) OpenScan(ctx context.Context, actor access.Actor, id string, revision int) (Scan, *os.File, error) {
	sc, err := s.ScanOf(ctx, actor, id, revision)
	if err != nil {
		return Scan{}, nil, err
	}
	if s.store == nil {
		return Scan{}, nil, errors.New("penyimpanan scan belum dipasang")
	}
	f, err := s.store.Open(sc.SHA256)
	if err != nil {
		return Scan{}, nil, err
	}
	return sc, f, nil
}

// Verification adalah isi halaman verifikasi satu revisi (M6.5).
type Verification struct {
	RevisionID   string
	DocumentID   string
	Number       string
	DocumentType string
	CustomerCode string
	Revision     int
	Status       string // effective, superseded, draft, void, atau obsolete
	ReleasedAt   *time.Time
	SHA256       string
}

// Verify membaca status satu revisi untuk halaman verifikasi (L-1).
func (s *Service) Verify(ctx context.Context, actor access.Actor, revisionID string) (Verification, error) {
	// Halaman verifikasi menjawab status revisi mana pun, juga draft customer
	// lain (L-1, D-32), jadi row-level security dilepas untuk bacaan ini.
	ctx = db.AsSystem(ctx)
	if !db.IsUUID(revisionID) {
		return Verification{}, problem.ErrNotFound
	}
	r, err := s.q.GetRevisionForVerify(ctx, revisionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Verification{}, problem.ErrNotFound
	}
	if err != nil {
		return Verification{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewReleased, access.Resource{CustomerID: r.CustomerID}); err != nil {
		return Verification{}, err
	}
	v := Verification{RevisionID: r.ID, DocumentID: r.DocumentID, Number: r.Number, DocumentType: r.DocumentTypeName, CustomerCode: r.CustomerCode,
		Revision: int(r.Revision), SHA256: deref(r.SnapshotSha256)}
	if r.ReleasedAt != nil {
		at := r.ReleasedAt.UTC()
		v.ReleasedAt = &at
	}
	switch {
	case r.NumberStatus == "void" || r.NumberStatus == "obsolete":
		v.Status = r.NumberStatus
	case r.Status != "released":
		v.Status = "draft"
	case r.ReleasedRevision != nil && int(*r.ReleasedRevision) == int(r.Revision):
		v.Status = "effective"
	default:
		v.Status = "superseded"
	}
	return v, nil
}

func cleanName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = strings.TrimSpace(filepath.Base("/" + name))
	if name == "/" || name == "." {
		return ""
	}
	for utf8.RuneCountInString(name) > 200 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
