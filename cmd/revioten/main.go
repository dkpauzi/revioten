// Program revioten menjalankan server web sistem PFD, PFMEA, dan CP.
//
// Cara menjalankan dari root repo (PostgreSQL harus sudah menyala lewat
// docker compose up -d, dan file .env sudah diisi):
//
//	go run ./cmd/revioten                        menjalankan server
//	go run ./cmd/revioten -addr 127.0.0.1:8081   server di port lain
//	go run ./cmd/revioten create-admin -name "Nama Pemegang" -login admin.nama
//	                                             membuat akun admin pertama (E-07)
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/server"
	"github.com/dkpauzi/revioten/web"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("program berhenti", "error", err)
		os.Exit(1)
	}
}

// run memilih perintah dari argumen. Semua error dikembalikan ke main, supaya
// perintah defer (misalnya menutup pool) tetap dijalankan sebelum program keluar.
func run(args []string) error {
	if len(args) > 0 && args[0] == "create-admin" {
		return runCreateAdmin(args[1:], os.Stdin, os.Stdout)
	}
	return runServer(args)
}

// runServer menjalankan server web.
func runServer(args []string) error {
	fs := flag.NewFlagSet("revioten", flag.ExitOnError)
	// Alamat bawaan hanya localhost, jadi server tidak terbuka ke jaringan (D-36).
	addr := fs.String("addr", "127.0.0.1:8080", "alamat dan port server")
	envFile := fs.String("env", ".env", "file berisi pengaturan database")
	_ = fs.Parse(args) // ExitOnError: flag yang salah langsung menghentikan program

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

	// Satu titik pemeriksaan izin dipakai semua service (M1.2).
	checker := access.NewChecker(pool)
	srv := &http.Server{
		Addr: *addr,
		// web.Files berisi hasil build SvelteKit yang di-embed ke binary ini.
		Handler: server.NewHandler(web.Files(), server.Deps{
			DB:        pool,
			Customers: customer.NewService(pool, checker),
			Auth:      auth.NewService(pool, checker),
			Access:    checker,
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
