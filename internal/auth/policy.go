package auth

import (
	_ "embed"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Aturan panjang password (E-08 (D-85), dicek 28 Oktober 2026):
//   - minimal 15 karakter: NIST SP 800-63B-4 mewajibkannya untuk password
//     yang dipakai sendirian (tanpa faktor kedua), dan ASVS 5.0 6.2.1
//     sangat menyarankannya;
//   - maksimal 128 karakter: NIST dan ASVS 6.2.9 meminta minimal 64 boleh
//     dipakai; batas atas menjaga biaya argon2id tetap wajar.
//
// Panjang dihitung per karakter Unicode (NIST), bukan per byte. Tidak ada aturan
// komposisi seperti wajib angka atau huruf besar (NIST, ASVS 6.2.5), dan tidak
// ada kewajiban ganti berkala (ASVS 6.2.10).
const (
	minPasswordLength = 15
	maxPasswordLength = 128
)

// serviceWords adalah kata yang terkait dengan sistem ini. Password yang
// memuatnya mudah ditebak (ASVS 6.2.11, NIST: context-specific words).
var serviceWords = []string{"revioten"}

//go:embed common_passwords.txt
var commonPasswordsFile string

// commonPasswords membaca daftar password umum sekali saja, saat pertama dipakai.
var commonPasswords = sync.OnceValue(func() map[string]bool {
	list := map[string]bool{}
	for line := range strings.Lines(commonPasswordsFile) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			list[line] = true
		}
	}
	return list
})

// checkNewPassword memeriksa password baru. contextWords berisi kata yang
// terkait dengan pemilik akun, misalnya nama login dan nama orangnya.
// Pesan masalah tidak pernah memuat password.
func checkNewPassword(field, password string, contextWords ...string) []problem.Problem {
	length := utf8.RuneCountInString(password)
	switch {
	case length < minPasswordLength:
		return []problem.Problem{{Field: field, Code: problem.TooShort, Message: "Use at least 15 characters."}}
	case length > maxPasswordLength:
		return []problem.Problem{{Field: field, Code: problem.TooLong, Message: "Use at most 128 characters."}}
	case isGuessable(password, contextWords):
		return []problem.Problem{{Field: field, Code: problem.TooCommon,
			Message: "This password is too easy to guess. Do not use common passwords, your login, your name, or the system name."}}
	}
	return nil
}

// isGuessable melaporkan apakah password ada di daftar password umum, atau
// memuat salah satu kata konteks. Huruf besar dan kecil tidak dibedakan.
func isGuessable(password string, contextWords []string) bool {
	lower := strings.ToLower(password)
	if commonPasswords()[lower] {
		return true
	}
	for _, word := range slices.Concat(contextWords, serviceWords) {
		// Nama orang dibandingkan tanpa spasi, jadi "Contoh Admin" juga
		// menangkap "contohadmin".
		word = strings.ToLower(strings.ReplaceAll(word, " ", ""))
		if len(word) >= 3 && strings.Contains(strings.ReplaceAll(lower, " ", ""), word) {
			return true
		}
	}
	return false
}
