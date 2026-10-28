// Program revioten menjalankan server web sistem PFD, PFMEA, dan CP.
//
// Cara menjalankan dari root repo (PostgreSQL harus sudah menyala lewat
// docker compose up -d, dan file .env sudah diisi):
//
//	go run ./cmd/revioten
//	go run ./cmd/revioten -addr 127.0.0.1:8081
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/server"
	"github.com/dkpauzi/revioten/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server berhenti", "error", err)
		os.Exit(1)
	}
}

// run berisi seluruh langkah program. Semua error dikembalikan ke main, supaya
// perintah defer (misalnya menutup pool) tetap dijalankan sebelum program keluar.
func run() error {
	// Alamat bawaan hanya localhost, jadi server tidak terbuka ke jaringan (D-36).
	addr := flag.String("addr", "127.0.0.1:8080", "alamat dan port server")
	envFile := flag.String("env", ".env", "file berisi pengaturan database")
	flag.Parse()

	cfg, err := loadConfig(os.Getenv, *envFile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Skema database diperbarui setiap kali program mulai (lihat sql/migrations).
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	srv := &http.Server{
		Addr: *addr,
		// web.Files berisi hasil build SvelteKit yang di-embed ke binary ini.
		Handler: server.NewHandler(web.Files(), server.Deps{
			DB:        pool,
			Customers: customer.NewService(pool),
		}),
		// Batas waktu membaca header permintaan. Tanpa batas ini, klien yang
		// sangat lambat bisa menahan koneksi selamanya (pengaman dasar, G9).
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Pesan ini ditulis sebelum port dibuka. Bila port sudah dipakai,
	// baris berikutnya di log adalah error dari ListenAndServe.
	slog.Info("menjalankan server", "alamat", "http://"+*addr)

	// ListenAndServe baru kembali bila ada error, misalnya port sudah dipakai.
	return srv.ListenAndServe()
}
