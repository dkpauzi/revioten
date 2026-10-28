// Package problem berisi bentuk penolakan input yang dipakai semua paket
// domain: satu masalah per field, dan dua jenis error (input tidak memenuhi
// aturan, atau input bentrok dengan data yang sudah ada). Adapter HTTP di
// internal/server menerjemahkannya ke format error API (R-API-13).
package problem

import (
	"errors"
	"strings"
)

// Kode masalah. Nilainya sama dengan ErrorDetail.code di kontrak API, supaya
// layar bisa menerjemahkannya ke teks sendiri.
const (
	Required      = "required"
	InvalidFormat = "invalid_format"
	TooShort      = "too_short"
	TooLong       = "too_long"
	TooCommon     = "too_common"
	Reserved      = "reserved"
	Taken         = "taken"
	Incorrect     = "incorrect"
	NotAllowed    = "not_allowed"
)

// Codes berisi semua kode masalah. Dipakai test untuk memastikan setiap kode
// ada di kontrak API.
var Codes = []string{Required, InvalidFormat, TooShort, TooLong, TooCommon, Reserved, Taken, Incorrect, NotAllowed}

// ErrNotFound berarti data yang diminta tidak ada. API menjawabnya 404.
var ErrNotFound = errors.New("data tidak ditemukan")

// Problem adalah satu masalah pada satu field input.
type Problem struct {
	Field   string // nama field di kontrak API, misalnya "code"
	Code    string // salah satu konstanta di atas
	Message string // penjelasan singkat dalam bahasa Inggris (Q-14)
}

// ValidationError berarti input tidak memenuhi aturan. API menjawabnya 422.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	return "input tidak valid: " + describe(e.Problems)
}

// ConflictError berarti input bentrok dengan data yang sudah ada, misalnya
// kode yang sudah dipakai. API menjawabnya 409.
type ConflictError struct {
	Problems []Problem
}

func (e *ConflictError) Error() string {
	return "input bentrok dengan data yang ada: " + describe(e.Problems)
}

// describe menyusun daftar masalah menjadi teks seperti "code:required, name:too_long".
// Pesannya tidak ikut, jadi isi input (misalnya password) tidak pernah masuk log.
func describe(problems []Problem) string {
	parts := make([]string, 0, len(problems))
	for _, p := range problems {
		parts = append(parts, p.Field+":"+p.Code)
	}
	return strings.Join(parts, ", ")
}
