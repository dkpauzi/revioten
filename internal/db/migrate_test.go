package db

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestMigrate membuktikan migrasi membuat tabel customers, dan aman dijalankan
// dua kali: migrasi yang sudah tercatat di goose_db_version dilewati.
func TestMigrate(t *testing.T) {
	pool := connectForTest(t)
	ctx := t.Context()

	for i := 1; i <= 2; i++ {
		if err := Migrate(ctx, pool); err != nil {
			t.Fatalf("Migrate ke-%d: %v", i, err)
		}
	}

	// to_regclass mengembalikan NULL bila tabel tidak ada.
	var table *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.customers')::text").Scan(&table); err != nil {
		t.Fatalf("memeriksa tabel customers: %v", err)
	}
	if table == nil {
		t.Fatal("tabel customers tidak ada setelah migrasi")
	}
}

// TestCustomerConstraints membuktikan batasan tabel customers menolak data
// yang salah walau ditulis langsung dengan SQL, tanpa lewat aturan di Go.
func TestCustomerConstraints(t *testing.T) {
	pool := connectForTest(t)
	ctx := t.Context()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	q := New(pool)

	// Satu baris yang benar, sebagai pembanding untuk kasus kode ganda.
	if _, err := q.CreateCustomer(ctx, CreateCustomerParams{Code: "CK", Name: "Contoh Customer"}); err != nil {
		t.Fatalf("baris yang benar ditolak: %v", err)
	}

	tests := []struct {
		name       string
		params     CreateCustomerParams
		constraint string
		unique     bool
	}{
		{"kode huruf kecil", CreateCustomerParams{Code: "cl", Name: "Contoh"}, "customers_code_format", false},
		{"kode tiga huruf", CreateCustomerParams{Code: "CLM", Name: "Contoh"}, "customers_code_format", false},
		{"kode berisi angka", CreateCustomerParams{Code: "C1", Name: "Contoh"}, "customers_code_format", false},
		{"kode ZZ", CreateCustomerParams{Code: "ZZ", Name: "Contoh"}, "customers_code_not_zz", false},
		{"kode sudah dipakai", CreateCustomerParams{Code: "CK", Name: "Contoh Lain"}, "customers_code_key", true},
		{"nama kosong", CreateCustomerParams{Code: "CL", Name: ""}, "customers_name_check", false},
		{"nama dengan spasi di tepi", CreateCustomerParams{Code: "CL", Name: " Contoh"}, "customers_name_check", false},
		{"nama 201 karakter", CreateCustomerParams{Code: "CL", Name: strings.Repeat("a", 201)}, "customers_name_check", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := q.CreateCustomer(ctx, tt.params)

			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("ingin ditolak batasan %s, dapat error %v", tt.constraint, err)
			}
			if pgErr.ConstraintName != tt.constraint {
				t.Errorf("batasan = %q, ingin %q", pgErr.ConstraintName, tt.constraint)
			}
			if got := IsUniqueViolation(err, "customers_code_key"); got != tt.unique {
				t.Errorf("IsUniqueViolation = %v, ingin %v", got, tt.unique)
			}
		})
	}

	// Nama 200 karakter (huruf beraksen dihitung satu karakter) masih boleh.
	if _, err := q.CreateCustomer(ctx, CreateCustomerParams{Code: "CM", Name: strings.Repeat("é", 200)}); err != nil {
		t.Errorf("nama 200 karakter ditolak: %v", err)
	}
}

// TestIsUniqueViolationOtherErrors berjalan tanpa database.
func TestIsUniqueViolationOtherErrors(t *testing.T) {
	if IsUniqueViolation(nil, "customers_code_key") {
		t.Error("nil dianggap unique violation")
	}
	if IsUniqueViolation(errors.New("error biasa"), "customers_code_key") {
		t.Error("error biasa dianggap unique violation")
	}
	other := &pgconn.PgError{Code: "23505", ConstraintName: "batasan_lain"}
	if IsUniqueViolation(other, "customers_code_key") {
		t.Error("unique violation dari batasan lain dianggap sama")
	}
}
