package problem

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Text merapikan dan memeriksa isian teks satu baris, misalnya nama di master
// data: spasi di tepi dibuang, wajib diisi bila required, paling banyak max
// karakter, dan tanpa karakter kontrol seperti baris baru (ASUMSI E-21: teks
// master dicetak di dokumen, jadi satu baris). Hasilnya teks yang sudah
// dirapikan, dan masalahnya (nil bila tidak ada).
func Text(field, value string, required bool, max int) (string, *Problem) {
	value = strings.TrimSpace(value)
	switch {
	case value == "" && required:
		return "", &Problem{Field: field, Code: Required, Message: "Enter a value."}
	case utf8.RuneCountInString(value) > max:
		return "", &Problem{Field: field, Code: TooLong, Message: fmt.Sprintf("Use at most %d characters.", max)}
	case strings.ContainsFunc(value, unicode.IsControl):
		return "", &Problem{Field: field, Code: InvalidFormat, Message: "Use one line without control characters."}
	}
	return value, nil
}
