// Package server adalah adapter HTTP aplikasi: daftar route dan handler.
//
// Adapter ini sengaja tipis. Aturan bisnis tidak ditulis di sini, tapi di
// package domain di bawah internal/, mulai M0.5 (R-API-01).
package server

import "net/http"

// NewHandler menyusun semua route aplikasi menjadi satu http.Handler.
func NewHandler() http.Handler {
	mux := http.NewServeMux()

	// Pola "GET /health" hanya cocok untuk method GET (dan HEAD).
	// Method lain otomatis dijawab 405, dan path yang tidak terdaftar 404.
	// ASUMSI E-17: /health ada di root, di luar /api/v1, dan belum masuk
	// kontrak OpenAPI. Kedudukannya diputuskan sebelum M0.5.
	mux.HandleFunc("GET /health", handleHealth)

	return mux
}
