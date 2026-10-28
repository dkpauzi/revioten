package customer

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/problem"
)

// maxNameLength adalah panjang nama paling banyak, dalam karakter.
// ASUMSI E-21: brief belum menyebut batas ini.
const maxNameLength = 200

// validate merapikan input lalu memeriksa aturannya. Semua masalah dikumpulkan
// sekaligus supaya layar bisa menandai setiap field. Aturan yang sama juga
// dijaga batasan tabel customers sebagai pengaman terakhir. Keberadaan Project
// Group diperiksa Service, karena butuh database.
func validate(in NewCustomer) (NewCustomer, []problem.Problem) {
	out := NewCustomer{
		Code:           normalizeCode(in.Code),
		Name:           strings.TrimSpace(in.Name),
		ProjectGroupID: strings.TrimSpace(in.ProjectGroupID),
	}

	var problems []problem.Problem
	if p, ok := checkCode(out.Code); !ok {
		problems = append(problems, p)
	}
	if p, ok := checkName(out.Name); !ok {
		problems = append(problems, p)
	}
	if out.ProjectGroupID == "" {
		problems = append(problems, projectGroupRequired)
	}
	return out, problems
}

// validateUpdate seperti validate, untuk isian yang bisa diubah.
func validateUpdate(in CustomerUpdate) (CustomerUpdate, []problem.Problem) {
	out := CustomerUpdate{
		Name:           strings.TrimSpace(in.Name),
		ProjectGroupID: strings.TrimSpace(in.ProjectGroupID),
		Active:         in.Active,
	}
	var problems []problem.Problem
	if p, ok := checkName(out.Name); !ok {
		problems = append(problems, p)
	}
	if out.ProjectGroupID == "" {
		problems = append(problems, projectGroupRequired)
	}
	return out, problems
}

// projectGroupRequired: setiap customer dialokasikan ke satu Project Group (D-54).
var projectGroupRequired = problem.Problem{
	Field: "project_group_id", Code: problem.Required, Message: "Choose the project group of the customer.",
}

// normalizeCode membuang spasi di tepi dan membuat huruf menjadi huruf besar.
func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// checkCode memeriksa kode yang sudah dirapikan: tepat dua huruf A sampai Z
// (E-12). Kode khusus seperti ZZ diperiksa Service, karena daftarnya master
// data di database (M1.6).
func checkCode(code string) (problem.Problem, bool) {
	switch {
	case code == "":
		return problem.Problem{Field: "code", Code: problem.Required, Message: "Enter the customer code."}, false
	case !isTwoLetters(code):
		return problem.Problem{Field: "code", Code: problem.InvalidFormat, Message: "Use two letters A to Z."}, false
	}
	return problem.Problem{}, true
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
func checkName(name string) (problem.Problem, bool) {
	switch {
	case name == "":
		return problem.Problem{Field: "name", Code: problem.Required, Message: "Enter the customer name."}, false
	case utf8.RuneCountInString(name) > maxNameLength:
		return problem.Problem{Field: "name", Code: problem.TooLong, Message: "Use at most 200 characters."}, false
	case strings.ContainsFunc(name, unicode.IsControl):
		// ASUMSI E-21: nama satu baris, karena dicetak di dokumen.
		return problem.Problem{Field: "name", Code: problem.InvalidFormat, Message: "Use one line without control characters."}, false
	}
	return problem.Problem{}, true
}

// groupCodePattern: kode Project Group, misalnya GR1.
// ASUMSI E-30: huruf besar A sampai Z dan angka, 1 sampai 10 karakter.
var groupCodePattern = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// validateProjectGroup merapikan dan memeriksa isian Project Group.
func validateProjectGroup(in ProjectGroupInput) (ProjectGroupInput, []problem.Problem) {
	out := ProjectGroupInput{Code: normalizeCode(in.Code)}
	var problems []problem.Problem
	switch {
	case out.Code == "":
		problems = append(problems, problem.Problem{Field: "code", Code: problem.Required, Message: "Enter the project group code."})
	case !groupCodePattern.MatchString(out.Code):
		problems = append(problems, problem.Problem{Field: "code", Code: problem.InvalidFormat,
			Message: "Use 1 to 10 letters A to Z or digits, for example GR1."})
	}
	name, p := problem.Text("name", in.Name, true, maxNameLength)
	if p != nil {
		problems = append(problems, *p)
	}
	out.Name = name
	return out, problems
}
