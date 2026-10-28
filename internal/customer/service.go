package customer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi customer dan Project Group terhadap database.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	authz   access.Authorizer
}

// NewService membuat Service yang memakai pool database. authz memeriksa izin
// setiap operasi (M1.2).
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, queries: db.New(pool), authz: authz}
}

// List mengembalikan semua customer, urut menurut kode, beserta riwayat kodenya.
// ASUMSI E-24: matriks U3 belum punya hak "lihat master data". Semua akun
// butuh daftar customer (misalnya untuk menyaring dokumen), jadi cukup hak
// dasar L-2 yang dimiliki semua akun.
func (s *Service) List(ctx context.Context, actor access.Actor) ([]Customer, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar customer: %w", err)
	}
	codeRows, err := s.queries.ListCustomerCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil riwayat kode customer: %w", err)
	}
	codes := map[string][]CodePeriod{}
	for _, r := range codeRows {
		codes[r.CustomerID] = append(codes[r.CustomerID], codePeriod(r.Code, r.ValidFrom, r.ValidUntil))
	}
	list := make([]Customer, 0, len(rows))
	for _, r := range rows {
		c := Customer{
			ID: r.ID, Code: r.Code, Name: r.Name, ProjectGroupID: deref(r.ProjectGroupID),
			ProjectGroupCode: deref(r.ProjectGroupCode), Active: r.Active, CreatedAt: r.CreatedAt.UTC(),
			Codes: codes[r.ID],
		}
		list = append(list, c)
	}
	return list, nil
}

// Get mengembalikan satu customer beserta riwayat kodenya (L-2, ASUMSI E-24).
func (s *Service) Get(ctx context.Context, actor access.Actor, id string) (Customer, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: id}); err != nil {
		return Customer{}, err
	}
	return get(ctx, s.queries, id)
}

// Create memeriksa input lalu menyimpan customer baru. Hasilnya
// *problem.ValidationError bila input salah, atau *problem.ConflictError bila
// kodenya sudah dipakai.
//
// Mengelola master data adalah hak A-2, yang hanya dimiliki akun admin (E-10).
func (s *Service) Create(ctx context.Context, actor access.Actor, in NewCustomer) (Customer, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Customer{}, err
	}
	in, problems := validate(in)
	codeOK := !slices.ContainsFunc(problems, func(p problem.Problem) bool { return p.Field == "code" })
	if codeOK {
		// Kode khusus diperiksa di sini juga, supaya semua masalah tampil
		// sekaligus. Pemeriksaan yang menentukan ada di dalam transaksi.
		reserved, err := specialCode(ctx, s.queries, in.Code)
		if err != nil {
			return Customer{}, err
		}
		if reserved != nil {
			problems = append([]problem.Problem{*reserved}, problems...)
		}
	}
	group, groupProblem, err := s.findProjectGroup(ctx, in.ProjectGroupID)
	if err != nil {
		return Customer{}, err
	}
	if groupProblem != nil {
		problems = append(problems, *groupProblem)
	}
	if len(problems) > 0 {
		return Customer{}, &problem.ValidationError{Problems: problems}
	}

	// Customer, kode pertamanya, dan catatan auditnya disimpan dalam satu transaksi (M1.3).
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		if err := lockCodeAndCheck(ctx, q, in.Code); err != nil {
			return err
		}
		row, err := q.CreateCustomer(ctx, db.CreateCustomerParams{Code: in.Code, Name: in.Name, ProjectGroupID: &group.ID})
		if err != nil {
			return err
		}
		id = row.ID
		if err := q.AddCustomerCode(ctx, db.AddCustomerCodeParams{Code: row.Code, CustomerID: row.ID, ValidFrom: row.CreatedAt}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionCustomerCreated, TargetType: audit.TargetCustomer, TargetID: row.ID,
			Details: map[string]string{"code": row.Code, "name": row.Name, "project_group": group.Code},
		})
	})
	var invalid *problem.ValidationError
	switch conflict := codeConflict(err); {
	case conflict != nil:
		return Customer{}, conflict
	case errors.As(err, &invalid):
		return Customer{}, err
	case err != nil:
		return Customer{}, fmt.Errorf("menyimpan customer: %w", err)
	}
	return get(ctx, s.queries, id)
}

// Update mengubah nama, Project Group, dan status aktif customer (A-2).
// Customer tidak pernah dihapus; menonaktifkan adalah penggantinya (E-09).
// Hanya field yang berubah yang dicatat di audit log.
func (s *Service) Update(ctx context.Context, actor access.Actor, id string, in CustomerUpdate) (Customer, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: id}); err != nil {
		return Customer{}, err
	}
	in, problems := validateUpdate(in)
	group, groupProblem, err := s.findProjectGroup(ctx, in.ProjectGroupID)
	if err != nil {
		return Customer{}, err
	}
	if groupProblem != nil {
		problems = append(problems, *groupProblem)
	}
	if len(problems) > 0 {
		return Customer{}, &problem.ValidationError{Problems: problems}
	}

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		before, err := lockAndGet(ctx, q, id)
		if err != nil {
			return err
		}
		id = before.ID
		err = q.UpdateCustomer(ctx, db.UpdateCustomerParams{ID: id, Name: in.Name, ProjectGroupID: &group.ID, Active: in.Active})
		if err != nil {
			return err
		}
		details := map[string]string{}
		if before.Name != in.Name {
			details["name"] = audit.Change(before.Name, in.Name)
		}
		if before.ProjectGroupID != group.ID {
			details["project_group"] = audit.Change(before.ProjectGroupCode, group.Code)
		}
		if before.Active != in.Active {
			details["active"] = audit.Change(strconv.FormatBool(before.Active), strconv.FormatBool(in.Active))
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionCustomerUpdated, TargetType: audit.TargetCustomer, TargetID: id, Details: details,
		})
	})
	if errors.Is(err, problem.ErrNotFound) {
		return Customer{}, err
	}
	if err != nil {
		return Customer{}, fmt.Errorf("mengubah customer: %w", err)
	}
	return get(ctx, s.queries, id)
}

// ChangeCode mengganti kode customer (A-2, E-11). Kode lama tetap tercatat
// beserta masa berlakunya, sehingga nomor dokumen lama yang memakai kode lama
// tetap bisa dicocokkan dengan customer ini (R-NUM-02).
// ASUMSI E-31: satu kode hanya pernah dipakai satu customer, termasuk kode
// lama, supaya nomor lama tidak pernah menunjuk dua customer.
func (s *Service) ChangeCode(ctx context.Context, actor access.Actor, id, code string) (Customer, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: id}); err != nil {
		return Customer{}, err
	}
	code = normalizeCode(code)
	if p, ok := checkCode(code); !ok {
		return Customer{}, &problem.ValidationError{Problems: []problem.Problem{p}}
	}

	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		before, err := lockAndGet(ctx, q, id)
		if err != nil {
			return err
		}
		id = before.ID
		if before.Code == code {
			return &problem.ValidationError{Problems: []problem.Problem{{
				Field: "code", Code: problem.NotAllowed, Message: "This is already the current code.",
			}}}
		}
		if err := lockCodeAndCheck(ctx, q, code); err != nil {
			return err
		}
		// Kode lama berakhir tepat saat kode baru mulai berlaku.
		now := time.Now().UTC()
		if err := q.CloseCustomerCode(ctx, db.CloseCustomerCodeParams{CustomerID: id, ValidUntil: &now}); err != nil {
			return err
		}
		if err := q.AddCustomerCode(ctx, db.AddCustomerCodeParams{Code: code, CustomerID: id, ValidFrom: now}); err != nil {
			return err
		}
		if err := q.SetCustomerCode(ctx, db.SetCustomerCodeParams{ID: id, Code: code}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionCustomerCodeChanged, TargetType: audit.TargetCustomer, TargetID: id,
			Details: map[string]string{"before": before.Code, "after": code},
		})
	})
	var invalid *problem.ValidationError
	switch conflict := codeConflict(err); {
	case conflict != nil:
		return Customer{}, conflict
	case errors.As(err, &invalid), errors.Is(err, problem.ErrNotFound):
		return Customer{}, err
	case err != nil:
		return Customer{}, fmt.Errorf("mengganti kode customer: %w", err)
	}
	return get(ctx, s.queries, id)
}

// specialCode memberi masalah code:reserved bila kode itu kode khusus, misalnya
// ZZ untuk dokumen umum (E-12, M1.6). Daftar kode khusus adalah master data.
func specialCode(ctx context.Context, q *db.Queries, code string) (*problem.Problem, error) {
	reserved, err := q.SpecialCodeExists(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("memeriksa kode khusus: %w", err)
	}
	if !reserved {
		return nil, nil
	}
	return &problem.Problem{
		Field: "code", Code: problem.Reserved, Message: "This is a special code for documents without a customer, such as ZZ.",
	}, nil
}

// lockCodeAndCheck mengambil kunci kode di posisi customer (kunci advisory
// 11003, dipakai juga saat kode khusus ditambah), lalu memeriksa kode khusus
// sekali lagi. Dengan kunci itu, kode khusus baru tidak bisa terdaftar di
// antara pemeriksaan dan penyimpanan kode customer (ASUMSI E-37).
func lockCodeAndCheck(ctx context.Context, q *db.Queries, code string) error {
	if err := q.LockNumberCodes(ctx); err != nil {
		return err
	}
	reserved, err := specialCode(ctx, q, code)
	if err != nil {
		return err
	}
	if reserved != nil {
		return &problem.ValidationError{Problems: []problem.Problem{*reserved}}
	}
	return nil
}

// codeConflict menerjemahkan pelanggaran batasan UNIQUE pada kode customer.
// customer_codes_pkey berarti kode itu sedang atau pernah dipakai customer lain.
func codeConflict(err error) error {
	if db.IsUniqueViolation(err, "customers_code_key") || db.IsUniqueViolation(err, "customer_codes_pkey") {
		return &problem.ConflictError{Problems: []problem.Problem{{
			Field: "code", Code: problem.Taken, Message: "This code is used, or was used before, by another customer.",
		}}}
	}
	return nil
}

// findProjectGroup mencari Project Group untuk dialokasikan ke customer.
// Hasilnya masalah input bila ID diisi tapi Project Group-nya tidak ada.
// ID kosong sudah dilaporkan validate.
func (s *Service) findProjectGroup(ctx context.Context, id string) (ProjectGroup, *problem.Problem, error) {
	if id == "" {
		return ProjectGroup{}, nil, nil
	}
	group, err := getProjectGroup(ctx, s.queries, id)
	if errors.Is(err, problem.ErrNotFound) {
		return ProjectGroup{}, &problem.Problem{
			Field: "project_group_id", Code: problem.InvalidFormat, Message: "This project group does not exist.",
		}, nil
	}
	return group, nil, err
}

// lockAndGet mengunci baris customer di transaksi q, lalu membacanya.
func lockAndGet(ctx context.Context, q *db.Queries, id string) (Customer, error) {
	if !db.IsUUID(id) {
		return Customer{}, problem.ErrNotFound
	}
	if err := q.LockCustomer(ctx, id); err != nil {
		return Customer{}, fmt.Errorf("mengunci customer: %w", err)
	}
	return get(ctx, q, id)
}

// get membaca satu customer beserta riwayat kodenya. ID yang tidak ada (atau
// bukan UUID) memberi problem.ErrNotFound.
func get(ctx context.Context, q *db.Queries, id string) (Customer, error) {
	if !db.IsUUID(id) {
		return Customer{}, problem.ErrNotFound
	}
	r, err := q.GetCustomer(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, problem.ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("membaca customer: %w", err)
	}
	codeRows, err := q.ListCodesOfCustomer(ctx, r.ID)
	if err != nil {
		return Customer{}, fmt.Errorf("membaca riwayat kode customer: %w", err)
	}
	codes := make([]CodePeriod, 0, len(codeRows))
	for _, c := range codeRows {
		codes = append(codes, codePeriod(c.Code, c.ValidFrom, c.ValidUntil))
	}
	return Customer{
		ID: r.ID, Code: r.Code, Name: r.Name, ProjectGroupID: deref(r.ProjectGroupID),
		ProjectGroupCode: deref(r.ProjectGroupCode), Active: r.Active, CreatedAt: r.CreatedAt.UTC(),
		Codes: codes,
	}, nil
}

// codePeriod menyusun satu masa kode, dengan waktu dalam UTC.
func codePeriod(code string, from time.Time, until *time.Time) CodePeriod {
	p := CodePeriod{Code: code, ValidFrom: from.UTC()}
	if until != nil {
		u := until.UTC()
		p.ValidUntil = &u
	}
	return p
}

// deref mengembalikan isi pointer teks, atau "" bila nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
