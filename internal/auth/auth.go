// Package auth berisi login lokal (M1.1, UC-01): akun, password argon2id, sesi,
// dan pembuatan akun admin pertama. Paket ini tidak tahu apa-apa tentang HTTP
// atau cookie; adapter di internal/server yang menghubungkannya ke API.
package auth

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Jenis akun (E-02, D-72). Akun admin hanya untuk tugas admin dan tidak pernah
// memegang peran kerja (R-SEC-13).
const (
	KindWork  = "work"
	KindAdmin = "admin"
)

// Aturan sesi dan gagal login (ASUMSI E-08, dicek 28 Oktober 2026):
//   - idleTimeout 30 menit: OWASP Session Management Cheat Sheet menyebut
//     15 sampai 30 menit untuk aplikasi berisiko rendah; NIST SP 800-63B-4
//     untuk AAL2 meminta paling lama 1 jam.
//   - absoluteTimeout 8 jam: OWASP menyebut 4 sampai 8 jam untuk pekerja
//     kantor yang memakai aplikasi seharian; NIST untuk AAL1 meminta paling
//     lama 30 hari dan AAL2 24 jam.
//   - setiap 10 kali gagal berturut-turut, akun dikunci 15 menit (OWASP
//     Authentication Cheat Sheet: ambang dan lama kunci dipilih aplikasi).
//   - kegagalan ke-100 berturut-turut menonaktifkan akun (NIST SP 800-63B-4
//     bagian rate limiting: paling banyak 100).
const (
	idleTimeout      = 30 * time.Minute
	absoluteTimeout  = 8 * time.Hour
	lockThreshold    = 10
	lockDuration     = 15 * time.Minute
	disableThreshold = 100
)

var (
	// ErrInvalidCredentials berarti login gagal. Penyebabnya sengaja tidak
	// dibedakan (nama login tidak ada, password salah, akun terkunci, atau
	// nonaktif), supaya penyerang tidak bisa menebak akun mana yang ada.
	ErrInvalidCredentials = errors.New("nama login atau password salah")
	// ErrUnauthenticated berarti tidak ada sesi yang berlaku.
	ErrUnauthenticated = errors.New("belum login atau sesi sudah berakhir")
	// ErrAdminExists berarti akun admin pertama sudah pernah dibuat (E-07).
	ErrAdminExists = errors.New("akun admin sudah ada")
)

// Actor adalah siapa yang memanggil sebuah operasi (R-API-03). Saat ini
// hanya manusia lewat sesi login; service account dan customer user menyusul.
type Actor struct {
	AccountID   string
	AccountKind string // KindWork atau KindAdmin
	Login       string
	PersonID    string
	PersonName  string
	// MustChangePassword true bila password direset admin. Selama itu, akun
	// hanya boleh mengganti password, melihat sesinya, dan logout (E-08).
	MustChangePassword bool
}

// normalizeLogin merapikan nama login: spasi di tepi dibuang dan huruf dibuat kecil.
func normalizeLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

// loginPattern: 3 sampai 64 karakter huruf kecil a-z, angka, titik, garis
// bawah, atau tanda hubung. ASUMSI E-08: brief belum menyebut bentuk nama login.
var loginPattern = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

// checkLogin memeriksa nama login yang sudah dirapikan.
func checkLogin(login string) (problem.Problem, bool) {
	switch {
	case login == "":
		return problem.Problem{Field: "login", Code: problem.Required, Message: "Enter a login name."}, false
	case !loginPattern.MatchString(login):
		return problem.Problem{Field: "login", Code: problem.InvalidFormat,
			Message: "Use 3 to 64 characters: letters a to z, digits, dot, underscore, or hyphen."}, false
	}
	return problem.Problem{}, true
}

// checkPersonName memeriksa nama orang yang sudah dirapikan: wajib, paling
// banyak 200 karakter, dan satu baris (sama dengan nama customer, ASUMSI E-21).
func checkPersonName(name string) (problem.Problem, bool) {
	switch {
	case name == "":
		return problem.Problem{Field: "name", Code: problem.Required, Message: "Enter the name of the person."}, false
	case utf8.RuneCountInString(name) > 200:
		return problem.Problem{Field: "name", Code: problem.TooLong, Message: "Use at most 200 characters."}, false
	case strings.ContainsFunc(name, unicode.IsControl):
		return problem.Problem{Field: "name", Code: problem.InvalidFormat, Message: "Use one line without control characters."}, false
	}
	return problem.Problem{}, true
}
