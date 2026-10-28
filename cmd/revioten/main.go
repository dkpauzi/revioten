// Program revioten menjalankan server web sistem PFD, PFMEA, dan CP.
//
// Cara menjalankan dari root repo:
//
//	go run ./cmd/revioten
//	go run ./cmd/revioten -addr 127.0.0.1:8081
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dkpauzi/revioten/internal/server"
)

func main() {
	// Alamat bawaan hanya localhost, jadi server tidak terbuka ke jaringan (D-36).
	addr := flag.String("addr", "127.0.0.1:8080", "alamat dan port server")
	flag.Parse()

	srv := &http.Server{
		Addr:    *addr,
		Handler: server.NewHandler(),
		// Batas waktu membaca header permintaan. Tanpa batas ini, klien yang
		// sangat lambat bisa menahan koneksi selamanya (pengaman dasar, G9).
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Pesan ini ditulis sebelum port dibuka. Bila port sudah dipakai,
	// baris berikutnya di log adalah error dari ListenAndServe.
	slog.Info("menjalankan server", "alamat", "http://"+*addr)

	// ListenAndServe baru kembali bila ada error, misalnya port sudah dipakai.
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server berhenti", "error", err)
		os.Exit(1)
	}
}
