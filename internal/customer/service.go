package customer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi customer terhadap database.
type Service struct {
	queries *db.Queries
}

// NewService membuat Service yang memakai pool database.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{queries: db.New(pool)}
}

// List mengembalikan semua customer, urut menurut kode.
func (s *Service) List(ctx context.Context) ([]Customer, error) {
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
func (s *Service) Create(ctx context.Context, in NewCustomer) (Customer, error) {
	in, err := validate(in)
	if err != nil {
		return Customer{}, err
	}

	row, err := s.queries.CreateCustomer(ctx, db.CreateCustomerParams{Code: in.Code, Name: in.Name})
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
