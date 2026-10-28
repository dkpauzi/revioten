// Package customer berisi aturan bisnis master customer (R-API-01): validasi,
// penyimpanan, dan daftar. Paket ini tidak tahu apa-apa tentang HTTP; adapter
// di internal/server yang menerjemahkan hasilnya ke format API.
package customer

import (
	"strings"
	"time"
)

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

// Kode masalah. Nilainya sama dengan ErrorDetail.code di kontrak API.
const (
	ProblemRequired      = "required"
	ProblemInvalidFormat = "invalid_format"
	ProblemTooLong       = "too_long"
	ProblemReserved      = "reserved"
	ProblemTaken         = "taken"
)

// Problem adalah satu masalah pada satu field input.
type Problem struct {
	Field   string // nama field di kontrak API, misalnya "code"
	Code    string // salah satu konstanta Problem...
	Message string // penjelasan singkat dalam bahasa Inggris (Q-14)
}

// ValidationError berarti input tidak memenuhi aturan. API menjawabnya 422.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	return "input customer tidak valid: " + describe(e.Problems)
}

// ConflictError berarti input bentrok dengan data yang sudah ada, misalnya
// kode yang sudah dipakai. API menjawabnya 409.
type ConflictError struct {
	Problems []Problem
}

func (e *ConflictError) Error() string {
	return "customer bentrok dengan data yang ada: " + describe(e.Problems)
}

// describe menyusun daftar masalah menjadi teks seperti "code:required, name:too_long".
func describe(problems []Problem) string {
	parts := make([]string, 0, len(problems))
	for _, p := range problems {
		parts = append(parts, p.Field+":"+p.Code)
	}
	return strings.Join(parts, ", ")
}
