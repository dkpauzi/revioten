package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/dkpauzi/revioten/sql/migrations"
)

// Migrate menjalankan semua migrasi goose yang belum dijalankan, menurut urutan
// nomor file di sql/migrations. Migrasi yang sudah tercatat di tabel
// goose_db_version dilewati, jadi fungsi ini aman dipanggil setiap kali
// program mulai.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// goose bekerja dengan database/sql, sedangkan program ini memakai pgxpool.
	// OpenDBFromPool membuat *sql.DB yang meminjam koneksi dari pool yang sama.
	sqlDB := stdlib.OpenDBFromPool(pool)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("menyiapkan migrasi: %w", err)
	}
	// Close hanya menutup sqlDB. Pool tetap terbuka untuk program.
	defer provider.Close()

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("menjalankan migrasi: %w", err)
	}
	for _, r := range results {
		slog.Info("migrasi dijalankan", "file", r.Source.Path, "durasi", r.Duration)
	}
	return nil
}
