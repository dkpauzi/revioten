package customer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxNameLength adalah panjang nama paling banyak, dalam karakter.
// ASUMSI E-21: brief belum menyebut batas ini.
const maxNameLength = 200

// reservedCode dipakai untuk dokumen General/Common, bukan untuk customer.
// ASUMSI E-12: kode ZZ di WI SW-ZZ-000 berarti dokumen umum.
const reservedCode = "ZZ"

// validate merapikan input lalu memeriksa aturannya. Bila ada masalah, semua
// masalah dikumpulkan sekaligus supaya layar bisa menandai setiap field.
// Aturan yang sama juga dijaga batasan tabel customers sebagai pengaman terakhir.
func validate(in NewCustomer) (NewCustomer, error) {
	out := NewCustomer{
		Code: strings.ToUpper(strings.TrimSpace(in.Code)),
		Name: strings.TrimSpace(in.Name),
	}

	var problems []Problem
	if p, ok := checkCode(out.Code); !ok {
		problems = append(problems, p)
	}
	if p, ok := checkName(out.Name); !ok {
		problems = append(problems, p)
	}
	if len(problems) > 0 {
		return NewCustomer{}, &ValidationError{Problems: problems}
	}
	return out, nil
}

// checkCode memeriksa kode yang sudah dirapikan: tepat dua huruf A sampai Z (E-12).
func checkCode(code string) (Problem, bool) {
	switch {
	case code == "":
		return Problem{Field: "code", Code: ProblemRequired, Message: "Enter the customer code."}, false
	case !isTwoLetters(code):
		return Problem{Field: "code", Code: ProblemInvalidFormat, Message: "Use two letters A to Z."}, false
	case code == reservedCode:
		return Problem{Field: "code", Code: ProblemReserved, Message: "ZZ is reserved for general documents."}, false
	}
	return Problem{}, true
}

// isTwoLetters melaporkan apakah s terdiri dari tepat dua huruf A sampai Z.
// Huruf lain seperti Ä ditolak, karena kode dipakai di nomor dokumen.
func isTwoLetters(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// checkName memeriksa nama yang sudah dirapikan.
func checkName(name string) (Problem, bool) {
	switch {
	case name == "":
		return Problem{Field: "name", Code: ProblemRequired, Message: "Enter the customer name."}, false
	case utf8.RuneCountInString(name) > maxNameLength:
		return Problem{Field: "name", Code: ProblemTooLong, Message: "Use at most 200 characters."}, false
	case strings.ContainsFunc(name, unicode.IsControl):
		// ASUMSI E-21: nama satu baris, karena dicetak di dokumen.
		return Problem{Field: "name", Code: ProblemInvalidFormat, Message: "Use one line without control characters."}, false
	}
	return Problem{}, true
}
