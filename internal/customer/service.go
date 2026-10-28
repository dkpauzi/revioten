package customer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi customer terhadap database.
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

// List mengembalikan semua customer, urut menurut kode.
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
	list := make([]Customer, 0, len(rows))
	for _, row := range rows {
		list = append(list, fromRow(row))
	}
	return list, nil
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
	in, err := validate(in)
	if err != nil {
		return Customer{}, err
	}

	// Customer dan catatan auditnya disimpan dalam satu transaksi (M1.3).
	var row db.Customer
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		var err error
		row, err = q.CreateCustomer(ctx, db.CreateCustomerParams{Code: in.Code, Name: in.Name})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionCustomerCreated, TargetType: audit.TargetCustomer, TargetID: row.ID,
			Details: map[string]string{"code": row.Code, "name": row.Name},
		})
	})
	// Kode ganda tidak diperiksa dengan SELECT lebih dulu: dua permintaan yang
	// datang bersamaan bisa sama-sama lolos. Batasan UNIQUE di database yang
	// menjaganya, dan penolakannya diterjemahkan di sini.
	if db.IsUniqueViolation(err, "customers_code_key") {
		return Customer{}, &problem.ConflictError{Problems: []problem.Problem{{
			Field:   "code",
			Code:    problem.Taken,
			Message: "This code is already used by another customer.",
		}}}
	}
	if err != nil {
		return Customer{}, fmt.Errorf("menyimpan customer: %w", err)
	}
	return fromRow(row), nil
}

// fromRow mengubah baris hasil sqlc menjadi Customer.
func fromRow(row db.Customer) Customer {
	return Customer{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.UTC(),
	}
}
