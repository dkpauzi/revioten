// Program revioten menjalankan server web sistem PFD, PFMEA, dan CP.
//
// Cara menjalankan dari root repo (PostgreSQL harus sudah menyala lewat
// docker compose up -d, dan file .env sudah diisi):
//
//	go run ./cmd/revioten                        menjalankan server
//	go run ./cmd/revioten -addr 127.0.0.1:8081   server di port lain
//	go run ./cmd/revioten create-admin -name "Nama Pemegang" -login admin.nama
//	                                             membuat akun admin pertama (E-07)
//	go run ./cmd/revioten reset-admin-password -login admin.nama
//	                                             membuka akun admin yang lupa
//	                                             password atau terkunci (E-22)
//	go run ./cmd/revioten backup                 backup terenkripsi ke BACKUP_DIR (M1.7)
//	go run ./cmd/revioten restore -file <file>   pulihkan backup ke database baru (M1.7)
package main

import (
	"cmp"
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/metrics"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/server"
	"github.com/dkpauzi/revioten/internal/settings"
	"github.com/dkpauzi/revioten/internal/storage"
	"github.com/dkpauzi/revioten/internal/template"
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
	if len(args) > 0 {
		switch args[0] {
		case "create-admin":
			return runCreateAdmin(args[1:], os.Stdin, os.Stdout)
		case "reset-admin-password":
			return runResetAdmin(args[1:], os.Stdout)
		case "backup":
			return runBackup(args[1:], os.Stdout)
		case "restore":
			return runRestore(args[1:], os.Stdout)
		}
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
	pool, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Satu titik pemeriksaan izin dipakai semua service (M1.2).
	checker := access.NewChecker(pool)
	// Lampiran perubahan disimpan di folder privat, bukan di database (M5.2).
	attachments, err := storage.New(cfg.AttachmentDir)
	if err != nil {
		return err
	}
	// Unggahan dipindai antivirus bila CLAMAV_ADDR diisi (M8.2, R-SEC-04).
	if cfg.ClamAVAddr != "" {
		attachments = attachments.WithScanner(storage.ClamAV{Addr: cfg.ClamAVAddr})
		slog.Info("unggahan dipindai antivirus", "clamd", cfg.ClamAVAddr)
	} else {
		slog.Warn("unggahan tidak dipindai antivirus: CLAMAV_ADDR kosong")
	}
	docs := document.NewService(pool, checker)
	// Cetak memakai folder lampiran yang sama untuk scan bertanda tangan (M6.7).
	printing := printout.NewService(pool, checker, docs, attachments, cfg.PDFBrowser).WithBaseURL(cmp.Or(cfg.PublicBaseURL, "http://"+*addr))
	authSvc := auth.NewService(pool, checker)
	srv := &http.Server{
		Addr: *addr,
		// web.Files berisi hasil build SvelteKit yang di-embed ke binary ini.
		Handler: server.NewHandler(web.Files(), server.Deps{
			DB:               pool,
			Customers:        customer.NewService(pool, checker),
			Auth:             authSvc,
			Roles:            authSvc,
			Access:           checker,
			Audit:            audit.NewService(pool, checker),
			Products:         product.NewService(pool, checker),
			Processes:        process.NewService(pool, checker),
			Numbering:        numbering.NewService(pool, checker),
			Changes:          change.NewService(pool, checker),
			Approval:         approval.NewService(pool, checker),
			Settings:         settings.NewService(pool, checker),
			Templates:        template.NewService(pool, checker),
			ProcessTemplates: template.NewProcessService(pool, checker),
			Documents:        docs,
			RefDocs:          template.NewRefDocService(pool, checker),
			ChangeRequests:   change.NewRequestService(pool, checker).WithStorage(attachments),
			Printing:         printing,
			Migration:        docs,
			Metrics:          metrics.NewService(pool, checker),
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

// openDatabase menyiapkan database, lalu membuka pool untuk program (E-06):
//  1. sebagai pemilik database: menjalankan migrasi yang belum dijalankan (lihat
//     sql/migrations) dan memasang password role aplikasi dari APP_DB_PASSWORD;
//  2. koneksi pemilik ditutup. Semua operasi berikutnya memakai role aplikasi,
//     yang hanya punya hak yang dibutuhkan, dan tidak bisa mengubah atau
//     menghapus audit log (R-SEC-03).
//
// E-29 (D-108): untuk demo di localhost, program yang sama memegang kredensial
// pemilik saat mulai. Versi IT sebaiknya menjalankan migrasi sebagai langkah
// terpisah, supaya server yang berjalan tidak pernah memegangnya.
func openDatabase(ctx context.Context, cfg config) (*pgxpool.Pool, error) {
	owner, err := db.Connect(ctx, cfg.OwnerURL)
	if err != nil {
		return nil, err
	}
	err = db.Migrate(ctx, owner)
	if err == nil {
		err = db.SetAppRolePassword(ctx, owner, cfg.AppPassword)
	}
	owner.Close()
	if err != nil {
		return nil, err
	}
	return db.Connect(ctx, cfg.AppURL)
}
