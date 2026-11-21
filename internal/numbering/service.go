package numbering

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi bagian nomor terhadap database.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database. authz memeriksa izin
// setiap operasi (M1.2).
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz}
}

// Melihat memakai hak L-2 seperti master lain (E-24, D-103), dan mengubah
// memakai E-11 (UC-05).

// GetSettings mengembalikan pengaturan nomor.
func (s *Service) GetSettings(ctx context.Context, actor access.Actor) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Settings{}, err
	}
	row, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	return fromSettings(row), nil
}

// UpdateSettings mengganti kode pabrik, pola nomor, dan jumlah digit revisi
// (E-11). Pattern dan RevisionDigits nil berarti tetap (D-111). Perubahan hanya
// berlaku untuk nomor baru; nomor yang sudah terdaftar tetap (R-NUM-02).
func (s *Service) UpdateSettings(ctx context.Context, actor access.Actor, in SettingsInput) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Settings{}, err
	}
	code := upper(in.FactoryCode)
	var list []*problem.Problem
	list = append(list, checkCode("factory_code", code, letterPattern, "Use one letter A to Z."))
	if in.Pattern != nil {
		if _, err := ParsePattern(*in.Pattern); err != nil {
			list = append(list, &problem.Problem{Field: "pattern", Code: problem.InvalidFormat,
				Message: "Use {prefix}, {customer}, and {seq} once each, separated by punctuation such as -, at most 50 characters."})
		}
	}
	if in.RevisionDigits != nil && (*in.RevisionDigits < 1 || *in.RevisionDigits > 3) {
		list = append(list, &problem.Problem{Field: "revision_digits", Code: problem.InvalidFormat, Message: "Use 1 to 3 digits."})
	}
	if err := problems(list...); err != nil {
		return Settings{}, err
	}

	var settings Settings
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.LockNumberSettings(ctx)
		if err != nil {
			return err
		}
		before := fromSettings(row)
		settings = before
		settings.FactoryCode = code
		if in.Pattern != nil {
			settings.Pattern = *in.Pattern
		}
		if in.RevisionDigits != nil {
			settings.RevisionDigits = *in.RevisionDigits
		}
		details := map[string]string{}
		if before.FactoryCode != settings.FactoryCode {
			details["factory_code"] = audit.Change(before.FactoryCode, settings.FactoryCode)
		}
		if before.Pattern != settings.Pattern {
			details["pattern"] = audit.Change(before.Pattern, settings.Pattern)
		}
		if before.RevisionDigits != settings.RevisionDigits {
			details["revision_digits"] = audit.Change(strconv.Itoa(before.RevisionDigits), strconv.Itoa(settings.RevisionDigits))
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		err = q.UpdateNumberSettings(ctx, db.UpdateNumberSettingsParams{
			FactoryCode: settings.FactoryCode, Pattern: settings.Pattern, RevisionDigits: int16(settings.RevisionDigits),
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionNumberSettingsUpdated, TargetType: audit.TargetNumberSettings, TargetID: before.ID, Details: details,
		})
	})
	return settings, result(err, "mengubah pengaturan nomor")
}

// fromSettings mengubah baris pengaturan nomor menjadi Settings.
func fromSettings(r db.NumberSetting) Settings {
	return Settings{
		ID: r.ID, FactoryCode: r.FactoryCode, Pattern: r.Pattern,
		SequenceFormat: r.SequenceFormat, RevisionDigits: int(r.RevisionDigits),
	}
}

// ListDocumentTypes mengembalikan semua jenis dokumen beserta awalannya.
// PFD, PFMEA, dan CP selalu di depan, lalu jenis lain menurut urutan dibuat.
func (s *Service) ListDocumentTypes(ctx context.Context, actor access.Actor) ([]DocumentType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListDocumentTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil jenis dokumen: %w", err)
	}
	list := make([]DocumentType, 0, len(rows))
	for _, r := range rows {
		list = append(list, DocumentType{ID: r.ID, Key: deref(r.Key), Name: r.Name, Letter: r.Letter, Prefix: r.Prefix})
	}
	return list, nil
}

// CreateDocumentType menambah jenis dokumen lain dari WI, misalnya Work
// Instruction (W). Jenis baru tidak punya key: sistem belum membuat dokumennya.
func (s *Service) CreateDocumentType(ctx context.Context, actor access.Actor, in DocumentTypeInput) (DocumentType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return DocumentType{}, err
	}
	in, err := cleanDocumentType(in)
	if err != nil {
		return DocumentType{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreateDocumentType(ctx, db.CreateDocumentTypeParams{Name: in.Name, Letter: in.Letter})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentTypeCreated, TargetType: audit.TargetDocumentType, TargetID: id,
			Details: map[string]string{"name": in.Name, "letter": in.Letter},
		})
	})
	if err := result(err, "menyimpan jenis dokumen"); err != nil {
		return DocumentType{}, err
	}
	return getDocumentType(ctx, s.q, id)
}

// UpdateDocumentType mengganti nama dan huruf satu jenis dokumen. Key-nya
// tetap, jadi PFD tetap PFD walau nama dan hurufnya diganti.
func (s *Service) UpdateDocumentType(ctx context.Context, actor access.Actor, id string, in DocumentTypeInput) (DocumentType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return DocumentType{}, err
	}
	in, err := cleanDocumentType(in)
	if err != nil {
		return DocumentType{}, err
	}
	if !db.IsUUID(id) {
		return DocumentType{}, problem.ErrNotFound
	}

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocumentType(ctx, id); err != nil {
			return err
		}
		before, err := getDocumentType(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.UpdateDocumentType(ctx, db.UpdateDocumentTypeParams{ID: id, Name: in.Name, Letter: in.Letter}); err != nil {
			return err
		}
		details := map[string]string{}
		if before.Name != in.Name {
			details["name"] = audit.Change(before.Name, in.Name)
		}
		if before.Letter != in.Letter {
			details["letter"] = audit.Change(before.Letter, in.Letter)
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentTypeUpdated, TargetType: audit.TargetDocumentType, TargetID: id, Details: details,
		})
	})
	if err := result(err, "mengubah jenis dokumen"); err != nil {
		return DocumentType{}, err
	}
	return getDocumentType(ctx, s.q, id)
}

// ListSpecialCodes mengembalikan semua kode khusus, urut menurut kode.
func (s *Service) ListSpecialCodes(ctx context.Context, actor access.Actor) ([]SpecialCode, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListSpecialCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil kode khusus: %w", err)
	}
	list := make([]SpecialCode, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromSpecialCode(r))
	}
	return list, nil
}

// CreateSpecialCode menambah kode khusus. Kodenya tidak boleh sama dengan kode
// customer, sekarang atau dulu (E-11), karena keduanya menempati posisi yang
// sama di nomor dokumen (E-37, D-114).
func (s *Service) CreateSpecialCode(ctx context.Context, actor access.Actor, in NewSpecialCode) (SpecialCode, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return SpecialCode{}, err
	}
	code := upper(in.Code)
	codeProblem := checkCode("code", code, specialCodePattern, "Use two or three letters A to Z.")
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	if err := problems(codeProblem, nameProblem); err != nil {
		return SpecialCode{}, err
	}

	var created SpecialCode
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Kunci bersama dengan pendaftaran kode customer: pemeriksaan di bawah
		// tidak bisa didahului customer baru dengan kode yang sama.
		if err := q.LockNumberCodes(ctx); err != nil {
			return err
		}
		used, err := q.CustomerCodeExists(ctx, code)
		if err != nil {
			return err
		}
		if used {
			return &problem.ConflictError{Problems: []problem.Problem{{
				Field: "code", Code: problem.Taken, Message: "This code is used by a customer, now or before.",
			}}}
		}
		row, err := q.CreateSpecialCode(ctx, db.CreateSpecialCodeParams{Code: code, Name: name})
		if err != nil {
			return err
		}
		created = fromSpecialCode(row)
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSpecialCodeCreated, TargetType: audit.TargetSpecialCode, TargetID: row.ID,
			Details: map[string]string{"code": code, "name": name},
		})
	})
	return created, result(err, "menyimpan kode khusus")
}

// RenameSpecialCode mengganti nama kode khusus. Kodenya tetap.
func (s *Service) RenameSpecialCode(ctx context.Context, actor access.Actor, id, name string) (SpecialCode, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return SpecialCode{}, err
	}
	name, nameProblem := problem.Text("name", name, true, maxNameLength)
	if err := problems(nameProblem); err != nil {
		return SpecialCode{}, err
	}
	if !db.IsUUID(id) {
		return SpecialCode{}, problem.ErrNotFound
	}

	var renamed SpecialCode
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockSpecialCode(ctx, id); err != nil {
			return err
		}
		before, err := getSpecialCode(ctx, q, id)
		if err != nil {
			return err
		}
		renamed = before
		renamed.Name = name
		if before.Name == name {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.RenameSpecialCode(ctx, db.RenameSpecialCodeParams{ID: id, Name: name}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSpecialCodeUpdated, TargetType: audit.TargetSpecialCode, TargetID: id,
			Details: map[string]string{"name": audit.Change(before.Name, name)},
		})
	})
	return renamed, result(err, "mengganti nama kode khusus")
}

// getDocumentType membaca satu jenis dokumen beserta awalannya.
func getDocumentType(ctx context.Context, q *db.Queries, id string) (DocumentType, error) {
	if !db.IsUUID(id) {
		return DocumentType{}, problem.ErrNotFound
	}
	r, err := q.GetDocumentType(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentType{}, problem.ErrNotFound
	}
	if err != nil {
		return DocumentType{}, fmt.Errorf("membaca jenis dokumen: %w", err)
	}
	return DocumentType{ID: r.ID, Key: deref(r.Key), Name: r.Name, Letter: r.Letter, Prefix: r.Prefix}, nil
}

// getSpecialCode membaca satu kode khusus.
func getSpecialCode(ctx context.Context, q *db.Queries, id string) (SpecialCode, error) {
	r, err := q.GetSpecialCode(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return SpecialCode{}, problem.ErrNotFound
	}
	if err != nil {
		return SpecialCode{}, fmt.Errorf("membaca kode khusus: %w", err)
	}
	return fromSpecialCode(r), nil
}

// fromSpecialCode mengubah baris hasil sqlc menjadi SpecialCode.
func fromSpecialCode(r db.SpecialCode) SpecialCode {
	return SpecialCode{ID: r.ID, Code: r.Code, Name: r.Name, CreatedAt: r.CreatedAt.UTC()}
}

// result menerjemahkan error dari transaksi: nama, huruf, atau kode yang sudah
// dipakai menjadi ConflictError, penolakan input dan "tidak ada" diteruskan
// apa adanya, dan error lain dibungkus dengan keterangan doing.
func result(err error, doing string) error {
	taken := func(field, message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
	}
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "document_types_letter_key"):
		return taken("letter", "This letter is already used by another document type.")
	case db.IsUniqueViolation(err, "document_types_name_key"):
		return taken("name", "This name is already used by another document type.")
	case db.IsUniqueViolation(err, "special_codes_code_key"):
		return taken("code", "This code is already a special code.")
	case errors.As(err, &invalid), errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// deref mengembalikan isi pointer teks, atau "" bila nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
