// Package db mengurus hubungan ke PostgreSQL: connection pool dan transaksi.
// Mulai M0.5 package ini juga berisi kode hasil sqlc.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect membuat connection pool ke PostgreSQL, lalu memastikan database
// benar-benar bisa dihubungi. Pemanggil wajib menutup pool dengan Close.
//
// Contoh databaseURL: postgres://user:password@127.0.0.1:5432/revioten?sslmode=disable
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		// ASUMSI E-18: pesan error asli sengaja tidak diteruskan. Menurut
		// dokumentasi pgx, penyamaran password di pesan ini tidak dijamin,
		// jadi password bisa bocor ke log atau ke layar.
		return nil, errors.New("alamat database tidak bisa dibaca: periksa format postgres://user:password@host:port/database")
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("membuat pool database: %w", err)
	}

	// Pool belum tentu membuka koneksi saat dibuat. Ping memastikan database
	// bisa dihubungi sebelum pool dipakai.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("menghubungi database: %w", err)
	}

	return pool, nil
}
