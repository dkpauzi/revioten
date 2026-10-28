// Package migrations menyimpan file migrasi goose dan memasukkannya ke dalam
// binary (go:embed), sehingga program bisa memperbarui skema database sendiri
// saat dijalankan, tanpa membawa folder sql/ ke komputer lain.
//
// Aturan penamaan: 00001_nama.sql, 00002_nama.sql, dan seterusnya. Angka diberi
// nol di depan supaya urutan abjad sama dengan urutan angka, karena sqlc
// membaca file migrasi menurut urutan abjad.
package migrations

import "embed"

// FS berisi semua file .sql di folder ini.
//
//go:embed *.sql
var FS embed.FS
