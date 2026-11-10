package auth

import (
	"slices"
	"strings"
	"testing"
)

func TestCheckNewPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		context  []string // nama login dan nama orang pemilik akun
		want     []string // kode masalah yang diharapkan; kosong berarti diterima
	}{
		{name: "8 karakter diterima", password: "satu dua"},
		{name: "7 karakter ditolak", password: "satu du", want: []string{"too_short"}},
		{name: "password umum pendek ditolak", password: "password1", want: []string{"too_common"}},
		{name: "deret angka pendek ditolak", password: "12345678", want: []string{"too_common"}},
		{name: "huruf beraksen dihitung satu karakter", password: strings.Repeat("é", 8)},
		{name: "128 karakter diterima", password: strings.Repeat("ab", 64)},
		{name: "129 karakter ditolak", password: strings.Repeat("ab", 64) + "c", want: []string{"too_long"}},
		{name: "tanpa aturan komposisi", password: "semua huruf kecil saja"},
		{name: "password umum ditolak", password: "1q2w3e4r5t6y7u8i9o0p", want: []string{"too_common"}},
		{name: "password umum dengan huruf besar ditolak", password: "1Q2W3E4R5T6Y7U8I9O0P", want: []string{"too_common"}},
		{name: "berisi nama login ditolak", password: "budi.s rahasia sekali", context: []string{"budi.s"}, want: []string{"too_common"}},
		{name: "berisi nama orang tanpa spasi ditolak", password: "contohadmin-rahasia", context: []string{"Contoh Admin"}, want: []string{"too_common"}},
		{name: "berisi nama layanan ditolak", password: "revioten rahasia 2026", want: []string{"too_common"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := checkNewPassword("password", tt.password, tt.context...)

			var got []string
			for _, p := range problems {
				got = append(got, p.Code)
				if p.Field != "password" || p.Message == "" {
					t.Errorf("masalah %+v harus memakai field password dan punya pesan", p)
				}
				if strings.Contains(p.Message, tt.password) {
					t.Errorf("pesan %q memuat password", p.Message)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}
}

// TestCommonPasswordsLoaded memastikan daftar password umum terbaca dari file.
func TestCommonPasswordsLoaded(t *testing.T) {
	if n := len(commonPasswords()); n < 300 {
		t.Errorf("daftar password umum berisi %d entri, ingin minimal 300", n)
	}
}
