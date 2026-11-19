// Package customer berisi aturan bisnis master customer dan Project Group
// (R-API-01, UC-03): validasi, penyimpanan, riwayat kode, dan daftar. Paket ini
// tidak tahu apa-apa tentang HTTP; adapter di internal/server yang
// menerjemahkan hasilnya ke format API.
package customer

import "time"

// Customer adalah satu customer yang sudah tersimpan.
type Customer struct {
	ID   string
	Code string // kode yang sedang berlaku
	Name string
	// Project Group tempat customer dialokasikan (D-54). Kosong hanya untuk
	// customer yang dibuat sebelum M1.4 dan belum dialokasikan.
	ProjectGroupID   string
	ProjectGroupCode string
	// Rute approval default customer (D-47, M1.8). Kosong bila belum diatur
	// Admin (E-39, D-116).
	DefaultRouteID   string
	DefaultRouteName string
	// SignatureMode: digital atau wet_only (D-198).
	SignatureMode string
	// Active false berarti customer dinonaktifkan. Customer tidak pernah dihapus (E-09).
	Active    bool
	CreatedAt time.Time // selalu UTC
	// Codes adalah riwayat kode, lama ke baru. Kode terakhir sedang berlaku (E-11).
	Codes []CodePeriod
}

// CodePeriod adalah satu kode customer beserta masa berlakunya.
type CodePeriod struct {
	Code       string
	ValidFrom  time.Time
	ValidUntil *time.Time // nil untuk kode yang sedang berlaku
}

// NewCustomer adalah input untuk menambah customer.
type NewCustomer struct {
	Code           string
	Name           string
	ProjectGroupID string
}

// CustomerUpdate adalah isian customer yang bisa diubah, selain kode. Kode
// diganti lewat ChangeCode, supaya riwayatnya tercatat (E-11).
type CustomerUpdate struct {
	Name           string
	ProjectGroupID string
	// Active nil berarti statusnya tetap. Pointer membedakan "tidak dikirim"
	// dari false, supaya klien API yang lupa mengirim field ini tidak
	// menonaktifkan customer tanpa sengaja (E-34, D-111).
	Active *bool
}

// ProjectGroup adalah satu Project Group (GR): kelompok customer yang dipegang
// satu Project Leader dan satu Project Engineer (D-54). Pemegang jabatannya
// dicatat di master jabatan (M1.8, E-05).
type ProjectGroup struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time // selalu UTC
}

// ProjectGroupInput adalah isian untuk menambah atau mengubah Project Group.
type ProjectGroupInput struct {
	Code string
	Name string
}
