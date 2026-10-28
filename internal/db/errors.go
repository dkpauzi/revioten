package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// uniqueViolation adalah kode error PostgreSQL untuk pelanggaran batasan UNIQUE.
const uniqueViolation = "23505"

// IsUniqueViolation melaporkan apakah err berasal dari batasan UNIQUE bernama
// constraint. Dengan cara ini aturan "kode tidak boleh ganda" dijaga oleh
// database, termasuk saat dua permintaan datang bersamaan, dan kode Go hanya
// menerjemahkan penolakannya.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == uniqueViolation &&
		pgErr.ConstraintName == constraint
}
