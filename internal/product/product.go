// Package product berisi aturan bisnis master produk (UC-03, M1.4): group
// model, model, dan part number milik satu customer. Semua master ini hanya
// diubah Admin (E-10) dan tidak pernah dihapus.
package product

import "time"

// ModelGroup adalah kumpulan model dalam satu customer yang berbagi isi
// template (D-39). Satu group hanya milik satu customer (D-42).
type ModelGroup struct {
	ID         string
	CustomerID string
	Name       string
	CreatedAt  time.Time // selalu UTC
}

// Model adalah satu model milik satu customer. Model paling banyak masuk satu
// group model (D-42); ModelGroupID kosong berarti tanpa group.
type Model struct {
	ID             string
	CustomerID     string
	ModelGroupID   string
	ModelGroupName string
	Name           string
	CreatedAt      time.Time // selalu UTC
	// EOLAt diisi bila model sudah end of life (D-200); nil bila masih aktif.
	EOLAt *time.Time
}

// PartNumber adalah satu nomor part milik satu model (ASUMSI E-30).
type PartNumber struct {
	ID         string
	CustomerID string
	ModelID    string
	ModelName  string
	Number     string
	Name       string // nama part, boleh kosong
	CreatedAt  time.Time
}

// NewModelGroup adalah input untuk menambah group model.
type NewModelGroup struct {
	CustomerID string
	Name       string
}

// NewModel adalah input untuk menambah model. ModelGroupID boleh kosong.
type NewModel struct {
	CustomerID   string
	Name         string
	ModelGroupID string
}

// ModelUpdate adalah isian model yang bisa diubah. Customer model tetap.
type ModelUpdate struct {
	Name         string
	ModelGroupID string
}

// NewPartNumber adalah input untuk menambah part number.
type NewPartNumber struct {
	ModelID string
	Number  string
	Name    string
}

// PartNumberUpdate adalah isian part number yang bisa diubah. Modelnya tetap.
type PartNumberUpdate struct {
	Number string
	Name   string
}

// Batas panjang isian, dalam karakter.
// ASUMSI E-30: brief belum menyebut batas ini.
const (
	maxNameLength   = 200
	maxNumberLength = 100
)
