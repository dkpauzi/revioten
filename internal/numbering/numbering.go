// Package numbering berisi bagian nomor dokumen menurut WI SW-ZZ-000 (M1.6,
// UC-05, E-12): kode pabrik, jenis dokumen dengan hurufnya, dan kode khusus di
// posisi kode customer. Awalan nomor adalah kode pabrik ditambah huruf jenis,
// misalnya SH untuk PFD (R-NUM-01). Semua bagian ini master data yang diubah
// Admin (A-2); perubahan hanya berlaku untuk nomor baru, karena nomor yang
// sudah terdaftar disimpan sebagai teks (R-NUM-02). Pola nomor dan
// penghitungnya menyusul di M2.1.
package numbering

import (
	"regexp"
	"strings"
	"time"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Key jenis dokumen yang dibuat sistem ini (R-NUM-01). Jenis lain dari WI
// tidak punya key.
const (
	KeyPFD   = "pfd"
	KeyPFMEA = "pfmea"
	KeyCP    = "cp"
)

// Settings adalah pengaturan nomor. Untuk sekarang isinya kode pabrik (E-12).
type Settings struct {
	ID          string
	FactoryCode string
}

// DocumentType adalah satu jenis dokumen (huruf B di WI SW-ZZ-000).
type DocumentType struct {
	ID     string
	Key    string // KeyPFD, KeyPFMEA, KeyCP, atau kosong untuk jenis lain
	Name   string
	Letter string
	Prefix string // kode pabrik + Letter, misalnya SH
}

// DocumentTypeInput adalah isian jenis dokumen.
type DocumentTypeInput struct {
	Name   string
	Letter string
}

// SpecialCode adalah kode khusus di posisi kode customer, untuk dokumen yang
// bukan milik satu customer, misalnya ZZ untuk General/Common (E-12).
type SpecialCode struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time // selalu UTC
}

// NewSpecialCode adalah isian kode khusus baru. Kodenya tidak bisa diubah
// setelah dibuat, hanya namanya.
type NewSpecialCode struct {
	Code string
	Name string
}

// maxNameLength adalah panjang nama paling banyak, dalam karakter.
// ASUMSI E-30: sama dengan master lain.
const maxNameLength = 200

// Bentuk kode menurut WI SW-ZZ-000 (E-12): kode pabrik dan huruf jenis
// dokumen masing-masing satu huruf. Kode khusus dua atau tiga huruf, seperti
// ZZ dan HSE (E-37, D-114).
var (
	letterPattern      = regexp.MustCompile(`^[A-Z]$`)
	specialCodePattern = regexp.MustCompile(`^[A-Z]{2,3}$`)
)

// upper merapikan kode: spasi di tepi dibuang, dan huruf menjadi huruf besar.
func upper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// checkCode memeriksa kode yang sudah dirapikan terhadap polanya.
func checkCode(field, value string, pattern *regexp.Regexp, message string) *problem.Problem {
	switch {
	case value == "":
		return &problem.Problem{Field: field, Code: problem.Required, Message: "Enter a value."}
	case !pattern.MatchString(value):
		return &problem.Problem{Field: field, Code: problem.InvalidFormat, Message: message}
	}
	return nil
}

// cleanDocumentType merapikan dan memeriksa isian jenis dokumen.
func cleanDocumentType(in DocumentTypeInput) (DocumentTypeInput, error) {
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	letter := upper(in.Letter)
	letterProblem := checkCode("letter", letter, letterPattern, "Use one letter A to Z.")
	return DocumentTypeInput{Name: name, Letter: letter}, problems(nameProblem, letterProblem)
}

// problems mengumpulkan masalah input yang ada menjadi satu ValidationError.
// Hasilnya nil bila semua masalah nil.
func problems(ps ...*problem.Problem) error {
	var list []problem.Problem
	for _, p := range ps {
		if p != nil {
			list = append(list, *p)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return &problem.ValidationError{Problems: list}
}
