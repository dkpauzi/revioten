// Package customer berisi aturan bisnis master customer (R-API-01): validasi,
// penyimpanan, dan daftar. Paket ini tidak tahu apa-apa tentang HTTP; adapter
// di internal/server yang menerjemahkan hasilnya ke format API.
package customer

import "time"

// Customer adalah satu customer yang sudah tersimpan.
type Customer struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time // selalu UTC
}

// NewCustomer adalah input untuk menambah customer.
type NewCustomer struct {
	Code string
	Name string
}
