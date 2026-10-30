package numbering

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Nomor berjalan (bagian DDD di WI SW-ZZ-000, E-12, D-89): tiga digit sampai
// 999, lalu satu huruf dan tiga digit: A001 sampai A999, B001, dan seterusnya
// sampai Z999. Aturan di atas 999 berlaku "untuk saat ini" (pernyataan
// pemilik), jadi bentuknya disimpan sebagai master data (number_settings).
const (
	// FirstSequence adalah nomor berjalan pertama untuk kode customer dan
	// jenis dokumen yang belum punya nomor.
	// ASUMSI E-44: 001, sama dengan contoh SC-KT-001 di M2.1 dan dengan A001
	// di atas 999. Nomor 000 di register lama tetap bisa diimpor (M2.5).
	FirstSequence = 1
	// blockSize adalah jumlah nomor dalam satu blok huruf (001 sampai 999).
	blockSize = 999
	// MaxSequence adalah nomor berjalan terakhir: Z999.
	MaxSequence = blockSize + 26*blockSize
)

// SequenceFormatLetterBlock adalah satu-satunya bentuk nomor berjalan saat
// ini, sesuai aturan pemilik (E-12). Nilainya disimpan di number_settings.
const SequenceFormatLetterBlock = "letter_block"

// ErrSequenceFull berarti nomor berjalan sudah mencapai Z999.
var ErrSequenceFull = errors.New("nomor berjalan sudah habis (Z999)")

// FormatSequence menulis nomor berjalan n, misalnya 1 menjadi 001 dan 1000
// menjadi A001. Nilai 0 (000) hanya mungkin dari nomor lama yang diimpor.
func FormatSequence(n int) (string, error) {
	switch {
	case n < 0:
		return "", fmt.Errorf("nomor berjalan %d tidak sah", n)
	case n > MaxSequence:
		return "", ErrSequenceFull
	case n <= blockSize:
		return fmt.Sprintf("%03d", n), nil
	}
	k := n - blockSize - 1
	letter := rune('A' + k/blockSize)
	return fmt.Sprintf("%c%03d", letter, k%blockSize+1), nil
}

// ParseSequence membaca teks nomor berjalan menjadi angka. ok false bila
// bentuknya tidak sah, misalnya 01, A000, atau a001.
func ParseSequence(s string) (n int, ok bool) {
	digits := func(t string) (int, bool) {
		v := 0
		for _, c := range t {
			if c < '0' || c > '9' {
				return 0, false
			}
			v = v*10 + int(c-'0')
		}
		return v, true
	}
	switch len(s) {
	case 3:
		return digits(s)
	case 4:
		letter := s[0]
		within, ok := digits(s[1:])
		if letter < 'A' || letter > 'Z' || !ok || within < 1 {
			return 0, false
		}
		return blockSize + int(letter-'A')*blockSize + within, true
	}
	return 0, false
}

// FormatRevision menulis nomor revisi dengan jumlah digit dari master data,
// misalnya 4 menjadi -04 (R-REV-01). Revisi disimpan sebagai angka, dan tidak
// menjadi bagian nomor yang terdaftar (E-12, D-89).
func FormatRevision(revision, digits int) string {
	return fmt.Sprintf("-%0*d", digits, revision)
}

// DefaultPattern adalah pola nomor bawaan: awalan, kode customer, dan nomor
// berjalan, dipisah tanda hubung, misalnya SC-KT-001 (R-NUM-01, D-89).
const DefaultPattern = "{prefix}-{customer}-{seq}"

// Token di pola nomor.
const (
	tokenPrefix   = "{prefix}"   // kode pabrik + huruf jenis dokumen, misalnya SC
	tokenCustomer = "{customer}" // kode customer, atau kode khusus
	tokenSequence = "{seq}"      // nomor berjalan, misalnya 001 atau A001
)

// maxPatternLength adalah panjang pola paling banyak, dalam karakter.
const maxPatternLength = 50

// Pattern adalah pola nomor yang sudah diperiksa (R-NUM-02). Pola hanya
// berlaku untuk nomor baru; nomor yang sudah terdaftar disimpan sebagai teks
// dan tidak pernah dihitung ulang.
type Pattern struct {
	text   string
	parser *regexp.Regexp
}

// Parts adalah bagian satu nomor dokumen.
type Parts struct {
	Prefix   string
	Customer string
	Sequence string
}

// ParsePattern memeriksa pola nomor. Aturannya:
//   - token {prefix}, {customer}, dan {seq} masing-masing tepat sekali;
//   - di antara dua token ada pemisah, dan teks di luar token hanya tanda
//     baca, bukan huruf atau angka. Dengan begitu setiap nomor lama bisa
//     dibaca kembali menjadi bagiannya tanpa ragu (impor, M2.5);
//   - paling banyak 50 karakter.
func ParsePattern(text string) (Pattern, error) {
	if text == "" || utf8.RuneCountInString(text) > maxPatternLength {
		return Pattern{}, errors.New("pola nomor wajib diisi, paling banyak 50 karakter")
	}
	var expr strings.Builder
	expr.WriteString("^")
	seen := map[string]bool{}
	literalBefore := true // awal pola boleh langsung token
	rest := text
	for rest != "" {
		token := ""
		for _, t := range []string{tokenPrefix, tokenCustomer, tokenSequence} {
			if strings.HasPrefix(rest, t) {
				token = t
			}
		}
		if token != "" {
			if seen[token] {
				return Pattern{}, fmt.Errorf("token %s muncul lebih dari sekali", token)
			}
			if !literalBefore {
				return Pattern{}, errors.New("dua token harus dipisah tanda baca, misalnya tanda hubung")
			}
			seen[token] = true
			literalBefore = false
			expr.WriteString(tokenExpr[token])
			rest = rest[len(token):]
			continue
		}
		r, size := utf8.DecodeRuneInString(rest)
		if r == '{' || r == '}' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsControl(r) || unicode.IsSpace(r) {
			return Pattern{}, errors.New("di luar token, pola hanya boleh berisi tanda baca seperti - / . _")
		}
		expr.WriteString(regexp.QuoteMeta(string(r)))
		literalBefore = true
		rest = rest[size:]
	}
	for _, t := range []string{tokenPrefix, tokenCustomer, tokenSequence} {
		if !seen[t] {
			return Pattern{}, fmt.Errorf("token %s wajib ada", t)
		}
	}
	expr.WriteString("$")
	return Pattern{text: text, parser: regexp.MustCompile(expr.String())}, nil
}

// tokenExpr adalah pola regexp untuk setiap token. Nama grup dipakai Parse.
var tokenExpr = map[string]string{
	tokenPrefix:   `(?P<prefix>[A-Z]{2})`,
	tokenCustomer: `(?P<customer>[A-Z]{2,3})`,
	tokenSequence: `(?P<seq>[0-9]{3}|[A-Z][0-9]{3})`,
}

// Text mengembalikan pola dalam bentuk teks.
func (p Pattern) Text() string {
	return p.text
}

// Format menyusun nomor dokumen dari bagiannya.
func (p Pattern) Format(prefix, customer, sequence string) string {
	r := strings.NewReplacer(tokenPrefix, prefix, tokenCustomer, customer, tokenSequence, sequence)
	return r.Replace(p.text)
}

// Parse membaca teks nomor menjadi bagiannya. ok false bila teks tidak
// mengikuti pola, atau nomor berjalannya tidak sah.
func (p Pattern) Parse(number string) (Parts, bool) {
	m := p.parser.FindStringSubmatch(number)
	if m == nil {
		return Parts{}, false
	}
	parts := Parts{
		Prefix:   m[p.parser.SubexpIndex("prefix")],
		Customer: m[p.parser.SubexpIndex("customer")],
		Sequence: m[p.parser.SubexpIndex("seq")],
	}
	if _, ok := ParseSequence(parts.Sequence); !ok {
		return Parts{}, false
	}
	return parts, true
}
