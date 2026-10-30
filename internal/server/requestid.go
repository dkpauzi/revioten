package server

import (
	"context"
	"crypto/rand"
	"net"
	"net/http"

	"github.com/dkpauzi/revioten/internal/audit"
)

// requestIDHeader adalah header untuk request ID (R-API-13).
const requestIDHeader = "X-Request-Id"

// withRequestID memberi setiap permintaan sebuah request ID. Request ID dari
// klien dipakai bila wajar; bila tidak ada atau tidak wajar, server membuat
// yang baru. Nilainya dikirim balik di header X-Request-Id, ditulis di setiap
// error API, dicatat di log, dan sejak M1.3 ikut di setiap entri audit log
// bersama alamat IP pengirim (R-API-03). Dengan begitu satu keluhan pengguna
// bisa dicocokkan dengan baris log server dan entri audit.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			// rand.Text menghasilkan 26 karakter acak (A-Z dan 2-7).
			id = rand.Text()
		}
		w.Header().Set(requestIDHeader, id)
		ctx := audit.WithRequest(r.Context(), audit.Request{ID: id, IP: remoteIP(r)})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// remoteIP mengambil alamat IP pengirim dari koneksi. Server berjalan tanpa
// proxy di depannya (D-36), jadi header seperti X-Forwarded-For tidak dipakai:
// header itu bisa diisi siapa saja. Bila nanti ada Caddy di depan (M8.3),
// aturan ini perlu ditinjau.
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// validRequestID menerima 1 sampai 64 karakter: huruf, angka, titik, garis
// bawah, dan tanda hubung. Batas ini mencegah baris log palsu (misalnya lewat
// karakter baris baru) dan header yang sangat panjang.
// E-20 (D-99): bentuk request ID ini bagian dari baseline konvensi API M0.5.
func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

// requestIDFrom mengambil request ID dari context permintaan.
func requestIDFrom(ctx context.Context) string {
	return audit.RequestFrom(ctx).ID
}
