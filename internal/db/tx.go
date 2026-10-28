package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx menjalankan fn di dalam satu transaksi database.
//
// Bila fn berhasil, transaksi di-COMMIT, sehingga semua perubahannya tersimpan
// bersama. Bila fn mengembalikan error, transaksi di-ROLLBACK, sehingga tidak
// ada perubahan yang tersimpan, dan error dari fn dikembalikan apa adanya.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("memulai transaksi: %w", err)
	}

	// Rollback dijadwalkan untuk semua jalan keluar dari fungsi ini,
	// termasuk panic. Setelah Commit berhasil, Rollback tidak mengubah apa-apa.
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("menyimpan transaksi (COMMIT): %w", err)
	}
	return nil
}
