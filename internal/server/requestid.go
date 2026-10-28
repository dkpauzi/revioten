package server

import (
	"context"
	"crypto/rand"
	"net/http"
)

// requestIDHeader adalah header untuk request ID (R-API-13).
const requestIDHeader = "X-Request-Id"

// requestIDKey adalah kunci request ID di context. Tipe kosong sendiri dipakai
// supaya kunci ini tidak bisa bentrok dengan kunci paket lain.
type requestIDKey struct{}

// withRequestID memberi setiap permintaan sebuah request ID. Request ID dari
// klien dipakai bila wajar; bila tidak ada atau tidak wajar, server membuat
// yang baru. Nilainya dikirim balik di header X-Request-Id, ditulis di setiap
// error API, dan dicatat di log, sehingga satu keluhan pengguna bisa dicocokkan
// dengan baris log server.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			// rand.Text menghasilkan 26 karakter acak (A-Z dan 2-7).
			id = rand.Text()
		}
		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validRequestID menerima 1 sampai 64 karakter: huruf, angka, titik, garis
// bawah, dan tanda hubung. Batas ini mencegah baris log palsu (misalnya lewat
// karakter baris baru) dan header yang sangat panjang.
// ASUMSI E-20: brief tidak menyebut bentuk request ID.
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
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
