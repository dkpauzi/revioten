package numbering

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Status nomor di register (R-NUM-04). Nomor tidak pernah dihapus atau
// dipakai ulang; yang berubah hanya statusnya.
const (
	StatusReserved = "reserved" // didaftarkan, dokumennya belum dibuat
	StatusInUse    = "in_use"   // dokumennya sudah dibuat (Tahap 4)
	StatusReleased = "released" // rilis sudah dicatat (Tahap 6)
	StatusObsolete = "obsolete" // ditarik atau digantikan
	StatusVoid     = "void"     // dibatalkan dengan alasan
	// StatusSkipped: dilewati saat daftar sebagai set, supaya register tidak
	// berlubang (ASUMSI E-64, M2.4).
	StatusSkipped = "skipped"
)

// Asal nomor: satu permintaan pendaftaran, satu set (M2.4), atau impor (M2.5).
const (
	OriginSingle = "single"
	OriginSet    = "set"
	OriginImport = "import"
)

// DocumentNumber adalah satu nomor di register.
type DocumentNumber struct {
	ID     string
	Number string // teks nomor, tidak pernah dihitung ulang (R-NUM-02)
	// Customer pemilik nomor, beserta kode dan namanya yang berlaku sekarang.
	CustomerID          string
	CustomerCurrentCode string
	CustomerName        string
	// CustomerCode adalah kode customer di dalam nomor. Bisa kode lama bila
	// kode customer pernah diganti (E-11).
	CustomerCode     string
	DocumentTypeID   string
	DocumentTypeKey  string // KeyPFD, KeyPFMEA, atau KeyCP
	DocumentTypeName string
	Sequence         int // nilai nomor berjalan, misalnya 1000 untuk A001
	Status           string
	InitialRevision  int
	RevisionLabel    string // revisi awal seperti tercetak, misalnya -04 (M2.4)
	BatchID          string
	Origin           string // OriginSingle, OriginSet, atau OriginImport
	RegisteredByID   string // orang yang mendaftarkan atau mengimpor
	RegisteredByName string
	StatusReason     string
	StatusChangedAt  *time.Time
	CreatedAt        time.Time // selalu UTC
}

// NewNumber adalah isian pendaftaran satu nomor (UC-10).
type NewNumber struct {
	CustomerID     string
	DocumentTypeID string
	// InitialRevision adalah revisi awal dokumen, misalnya 4 untuk dokumen
	// lama yang masuk di revisi -04 (M2.4). Biasanya 0.
	InitialRevision int
	// IdempotencyKey dibuat klien untuk setiap permintaan baru. Permintaan
	// yang diulang dengan kunci yang sama mendapat nomor yang sama (R-NUM-03).
	IdempotencyKey string
}

// Registration adalah hasil satu permintaan pendaftaran.
type Registration struct {
	Numbers []DocumentNumber
	// Replayed true bila kunci idempoten sudah pernah dipakai untuk permintaan
	// yang sama: tidak ada nomor baru, dan hasil lamanya dikembalikan.
	Replayed bool
}

// idempotencyKeyPattern: 1 sampai 100 karakter huruf, angka, titik, garis
// bawah, titik dua, atau tanda hubung, misalnya UUID dari browser.
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)

// errReplay menandai kunci idempoten yang sudah pernah dipakai. Hanya dipakai
// di dalam paket ini untuk membatalkan transaksi.
var errReplay = errors.New("kunci idempoten sudah dipakai")

// Register mendaftarkan satu nomor PFD, PFMEA, atau CP untuk satu customer
// (E-1, UC-10). ASUMSI E-37 butir 4: nomor hanya untuk customer (R-NUM-01);
// kode khusus seperti ZZ belum dipakai untuk menomori PFD, PFMEA, dan CP. Nomor dialokasikan di server: penghitung kode customer dan
// jenis dokumen dinaikkan di dalam transaksi yang sama dengan penyimpanan
// nomor dan entri audit-nya, jadi pendaftaran bersamaan tidak pernah bentrok
// dan tidak meninggalkan lubang (R-NUM-03, M2.2).
func (s *Service) Register(ctx context.Context, actor access.Actor, in NewNumber) (Registration, error) {
	if err := s.authz.Authorize(ctx, actor, access.RegisterNumbers, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Registration{}, err
	}
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return Registration{}, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	docType, list, err := s.checkRegistration(ctx, in.CustomerID, in.DocumentTypeID, in.InitialRevision, in.IdempotencyKey, int(settings.RevisionDigits))
	if err != nil {
		return Registration{}, err
	}
	if err := problems(list...); err != nil {
		return Registration{}, err
	}
	pattern, err := ParsePattern(settings.Pattern)
	if err != nil {
		return Registration{}, fmt.Errorf("pola nomor di database tidak sah: %w", err)
	}
	hash := requestHash(OriginSingle, in.CustomerID, in.DocumentTypeID, strconv.Itoa(in.InitialRevision))

	var batchID string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		code, err := customerCodeForNumber(ctx, q, in.CustomerID)
		if err != nil {
			return err
		}
		batchID, err = startBatch(ctx, q, actor, OriginSingle, in.CustomerID, in.IdempotencyKey, hash)
		if err != nil {
			return err
		}
		n, err := q.NextCounterValue(ctx, db.NextCounterValueParams{
			CustomerCode: code, DocumentTypeID: docType.ID, FirstValue: FirstSequence,
		})
		if err != nil {
			return err
		}
		return insertNumber(ctx, q, actor, pattern, newRow{
			customerID: in.CustomerID, code: code, docType: docType, sequence: int(n),
			status: StatusReserved, initialRevision: in.InitialRevision, batchID: batchID,
		})
	})
	if errors.Is(err, errReplay) {
		return s.replay(ctx, actor, in.IdempotencyKey, hash)
	}
	if err := registerResult(err); err != nil {
		return Registration{}, err
	}
	numbers, err := s.batchNumbers(ctx, batchID)
	return Registration{Numbers: numbers}, err
}

// NextNumber mengembalikan nomor yang akan didapat pendaftaran berikutnya
// untuk satu customer dan jenis dokumen, tanpa mendaftarkannya (L-2). Nomor
// sebenarnya baru ditentukan saat pendaftaran, karena orang lain bisa
// mendaftar lebih dulu.
func (s *Service) NextNumber(ctx context.Context, actor access.Actor, customerID, documentTypeID string) (string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return "", err
	}
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return "", fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	docType, list, err := s.checkRegistration(ctx, customerID, documentTypeID, 0, "preview", int(settings.RevisionDigits))
	if err != nil {
		return "", err
	}
	if err := problems(list...); err != nil {
		return "", err
	}
	pattern, err := ParsePattern(settings.Pattern)
	if err != nil {
		return "", fmt.Errorf("pola nomor di database tidak sah: %w", err)
	}
	c, err := s.q.GetCustomer(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("membaca customer: %w", err)
	}
	last, err := s.q.GetCounterValue(ctx, db.GetCounterValueParams{CustomerCode: c.Code, DocumentTypeID: docType.ID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		last = FirstSequence - 1
	case err != nil:
		return "", fmt.Errorf("membaca penghitung: %w", err)
	}
	seq, err := FormatSequence(int(last) + 1)
	if errors.Is(err, ErrSequenceFull) {
		return "", sequenceFull()
	}
	if err != nil {
		return "", err
	}
	return pattern.Format(docType.Prefix, c.Code, seq), nil
}

// checkRegistration memeriksa isian pendaftaran satu nomor: customer ada dan
// aktif, jenis dokumen PFD, PFMEA, atau CP, revisi awal sesuai jumlah digit
// revisi, dan kunci idempoten. Masalah input dikembalikan sebagai daftar;
// error hanya untuk kegagalan database.
func (s *Service) checkRegistration(ctx context.Context, customerID, documentTypeID string, revision int, key string, revisionDigits int) (DocumentType, []*problem.Problem, error) {
	list, err := s.checkCustomer(ctx, customerID)
	if err != nil {
		return DocumentType{}, nil, err
	}
	docType, typeProblem, err := s.checkDocumentType(ctx, documentTypeID)
	if err != nil {
		return DocumentType{}, nil, err
	}
	list = append(list, typeProblem)
	return docType, append(list, checkRevisionAndKey(revision, key, revisionDigits)...), nil
}

// checkCustomer memeriksa customer tujuan nomor: ada dan aktif.
func (s *Service) checkCustomer(ctx context.Context, customerID string) ([]*problem.Problem, error) {
	switch {
	case customerID == "":
		return []*problem.Problem{{Field: "customer_id", Code: problem.Required, Message: "Choose the customer."}}, nil
	case !db.IsUUID(customerID):
		return []*problem.Problem{{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}}, nil
	}
	c, err := s.q.GetCustomer(ctx, customerID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return []*problem.Problem{{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}}, nil
	case err != nil:
		return nil, fmt.Errorf("membaca customer: %w", err)
	case !c.Active:
		// ASUMSI E-67: customer nonaktif tidak mendapat nomor baru, sama
		// seperti tidak mendapat model baru (E-09).
		return []*problem.Problem{{Field: "customer_id", Code: problem.NotAllowed, Message: "This customer is inactive."}}, nil
	}
	return nil, nil
}

// checkDocumentType memeriksa jenis dokumen: hanya PFD, PFMEA, dan CP yang
// dinomori sistem (R-NUM-01). Jenis lain dari WI dicatat sebagai dokumen
// terkait (E-12).
func (s *Service) checkDocumentType(ctx context.Context, documentTypeID string) (DocumentType, *problem.Problem, error) {
	if documentTypeID == "" {
		return DocumentType{}, &problem.Problem{Field: "document_type_id", Code: problem.Required, Message: "Choose the document type."}, nil
	}
	d, err := getDocumentType(ctx, s.q, documentTypeID)
	switch {
	case errors.Is(err, problem.ErrNotFound):
		return DocumentType{}, &problem.Problem{Field: "document_type_id", Code: problem.InvalidFormat, Message: "Choose an existing document type."}, nil
	case err != nil:
		return DocumentType{}, nil, err
	case d.Key == "":
		return DocumentType{}, &problem.Problem{Field: "document_type_id", Code: problem.NotAllowed, Message: "Only PFD, PFMEA, and CP numbers are registered here."}, nil
	}
	return d, nil, nil
}

// checkRevisionAndKey memeriksa revisi awal (sesuai jumlah digit revisi di
// master data, M2.4) dan kunci idempoten (M2.2).
func checkRevisionAndKey(revision int, key string, revisionDigits int) []*problem.Problem {
	var list []*problem.Problem
	if maxRevision := pow10(revisionDigits) - 1; revision < 0 || revision > maxRevision {
		list = append(list, &problem.Problem{Field: "initial_revision", Code: problem.InvalidFormat,
			Message: fmt.Sprintf("Use a revision from 0 to %d.", maxRevision)})
	}
	switch {
	case key == "":
		list = append(list, &problem.Problem{Field: "idempotency_key", Code: problem.Required, Message: "Send an Idempotency-Key header."})
	case !idempotencyKeyPattern.MatchString(key):
		list = append(list, &problem.Problem{Field: "idempotency_key", Code: problem.InvalidFormat,
			Message: "Use 1 to 100 letters, digits, dots, colons, underscores, or hyphens."})
	}
	return list
}

// pow10 mengembalikan 10 pangkat n.
func pow10(n int) int {
	v := 1
	for range n {
		v *= 10
	}
	return v
}

// requestHash menyusun hash isi permintaan, untuk mengenali kunci idempoten
// yang dipakai ulang dengan isi berbeda.
func requestHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintf(h, "%d:%s,", len(p), p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// startBatch mencatat permintaan dengan kunci idempotennya, tepat sesudah
// customer dikunci (customerCodeForNumber). Bila akun itu sudah memakai
// kuncinya, hasilnya errReplay dan transaksi dibatalkan tanpa mengubah apa pun.
func startBatch(ctx context.Context, q *db.Queries, actor access.Actor, kind, customerID, key, hash string) (string, error) {
	id, err := q.InsertNumberBatch(ctx, db.InsertNumberBatchParams{
		Kind: kind, CustomerID: customerID, AccountID: actor.AccountID, PersonID: actor.PersonID,
		IdempotencyKey: &key, RequestHash: &hash,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errReplay
	}
	return id, err
}

// replay mengembalikan hasil permintaan lama dengan kunci idempoten yang
// sama. Kunci yang dipakai ulang untuk isi yang berbeda ditolak.
// ASUMSI E-63: kunci berlaku per akun dan disimpan selamanya.
func (s *Service) replay(ctx context.Context, actor access.Actor, key, hash string) (Registration, error) {
	b, err := s.q.GetNumberBatchByKey(ctx, db.GetNumberBatchByKeyParams{AccountID: actor.AccountID, IdempotencyKey: &key})
	if err != nil {
		return Registration{}, fmt.Errorf("membaca permintaan lama: %w", err)
	}
	if b.RequestHash == nil || *b.RequestHash != hash {
		return Registration{}, &problem.ValidationError{Problems: []problem.Problem{{
			Field: "idempotency_key", Code: problem.Taken,
			Message: "This Idempotency-Key was already used for a different request.",
		}}}
	}
	numbers, err := s.batchNumbers(ctx, b.ID)
	return Registration{Numbers: numbers, Replayed: true}, err
}

// customerCodeForNumber membaca kode customer yang berlaku dan menguncinya
// sampai transaksi selesai (R-POR-01). Customer yang dinonaktifkan sejak
// pemeriksaan isian juga ditolak di sini.
//
// Urutan kunci: fungsi ini harus menjadi langkah pertama transaksi penomoran,
// sebelum menyimpan apa pun yang merujuk customer itu. Foreign key ke customer
// (misalnya number_batches.customer_id) memasang kunci KEY SHARE. Bila kunci
// FOR SHARE baru diminta sesudahnya, penggantian kode yang masuk di antaranya
// saling menunggu dengan pendaftaran ini (deadlock): ganti kode memegang FOR NO
// KEY UPDATE dan menunggu KEY SHARE dilepas untuk mengubah kolom code yang
// unik (TestRegisterDuringCodeChange).
func customerCodeForNumber(ctx context.Context, q *db.Queries, customerID string) (string, error) {
	c, err := q.LockCustomerForNumber(ctx, customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !c.Active {
		return "", &problem.ValidationError{Problems: []problem.Problem{{
			Field: "customer_id", Code: problem.NotAllowed, Message: "This customer is inactive.",
		}}}
	}
	return c.Code, nil
}

// newRow adalah isi satu nomor yang akan disimpan.
type newRow struct {
	customerID      string
	code            string
	docType         DocumentType
	sequence        int
	status          string
	initialRevision int
	batchID         string
}

// insertNumber menyusun teks nomor dari pola, menyimpannya, dan mencatat
// entri audit-nya.
func insertNumber(ctx context.Context, q *db.Queries, actor access.Actor, pattern Pattern, r newRow) error {
	seq, err := FormatSequence(r.sequence)
	if errors.Is(err, ErrSequenceFull) {
		return sequenceFull()
	}
	if err != nil {
		return err
	}
	number := pattern.Format(r.docType.Prefix, r.code, seq)
	id, err := q.InsertDocumentNumber(ctx, db.InsertDocumentNumberParams{
		Number: number, CustomerID: r.customerID, CustomerCode: r.code, DocumentTypeID: r.docType.ID,
		Sequence: int32(r.sequence), Status: r.status, InitialRevision: int16(r.initialRevision), BatchID: r.batchID,
	})
	if err != nil {
		return err
	}
	details := map[string]string{"number": number, "status": r.status}
	if r.initialRevision != 0 {
		details["initial_revision"] = strconv.Itoa(r.initialRevision)
	}
	return audit.Record(ctx, q, actor, audit.Event{
		Action: audit.ActionDocumentNumberRegistered, TargetType: audit.TargetDocumentNumber, TargetID: id, Details: details,
	})
}

// sequenceFull adalah penolakan saat nomor berjalan sudah mencapai Z999.
func sequenceFull() error {
	return &problem.ConflictError{Problems: []problem.Problem{{
		Field: "sequence", Code: problem.NotAllowed,
		Message: "No numbers are left for this customer code and document type (Z999 was the last).",
	}}}
}

// registerResult menerjemahkan error transaksi pendaftaran. Teks nomor yang
// sudah ada (misalnya nomor lama yang diimpor dengan pola lain) menjadi
// ConflictError; transaksinya batal, jadi penghitung tidak naik.
func registerResult(err error) error {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "document_numbers_number_key"), db.IsUniqueViolation(err, "document_numbers_sequence_key"):
		return &problem.ConflictError{Problems: []problem.Problem{{
			Field: "document_type_id", Code: problem.Taken,
			Message: "The next number is already in the register. Ask the document control to check the register.",
		}}}
	case errors.As(err, &invalid), errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("mendaftarkan nomor: %w", err)
}

// batchNumbers mengambil nomor hasil satu permintaan.
func (s *Service) batchNumbers(ctx context.Context, batchID string) ([]DocumentNumber, error) {
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	rows, err := s.q.ListBatchNumbers(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("membaca nomor: %w", err)
	}
	list := make([]DocumentNumber, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromNumber(db.GetDocumentNumberRow(r), int(settings.RevisionDigits)))
	}
	return list, nil
}

// fromNumber mengubah baris hasil sqlc menjadi DocumentNumber. Label revisi
// ditulis dengan jumlah digit revisi yang berlaku (revisionDigits).
func fromNumber(r db.GetDocumentNumberRow, revisionDigits int) DocumentNumber {
	n := DocumentNumber{
		ID: r.ID, Number: r.Number, CustomerID: r.CustomerID, CustomerCurrentCode: r.CustomerCurrentCode,
		CustomerName: r.CustomerName, CustomerCode: r.CustomerCode, DocumentTypeID: r.DocumentTypeID,
		DocumentTypeKey: deref(r.DocumentTypeKey), DocumentTypeName: r.DocumentTypeName, Sequence: int(r.Sequence),
		Status: r.Status, InitialRevision: int(r.InitialRevision), RevisionLabel: FormatRevision(int(r.InitialRevision), revisionDigits),
		BatchID: r.BatchID, Origin: r.Origin,
		RegisteredByID: r.RegisteredByID, RegisteredByName: r.RegisteredByName, StatusReason: deref(r.StatusReason),
		CreatedAt: r.CreatedAt.UTC(),
	}
	if r.StatusChangedAt != nil {
		at := r.StatusChangedAt.UTC()
		n.StatusChangedAt = &at
	}
	return n
}
