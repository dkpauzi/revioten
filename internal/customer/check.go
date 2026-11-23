package customer

// Pemeriksaan tanpa menyimpan, untuk Preview import master data (I5). Aturan
// yang dipakai sama dengan Create dan CreateProjectGroup, jadi yang lolos di
// Preview juga lolos saat disimpan (kecuali bila data berubah di antaranya).

import (
	"context"

	"github.com/dkpauzi/revioten/internal/problem"
)

// CheckNewProjectGroup merapikan dan memeriksa isian Project Group baru.
func CheckNewProjectGroup(in ProjectGroupInput) (ProjectGroupInput, []problem.Problem) {
	return validateProjectGroup(in)
}

// CheckNewCustomer merapikan dan memeriksa kode dan nama customer baru,
// termasuk kode khusus seperti ZZ (M1.6). Project Group diperiksa pemanggil.
func (s *Service) CheckNewCustomer(ctx context.Context, code, name string) (string, string, []problem.Problem, error) {
	in, list := validate(NewCustomer{Code: code, Name: name, ProjectGroupID: "-"})
	if _, ok := checkCode(in.Code); ok {
		reserved, err := specialCode(ctx, s.queries, in.Code)
		if err != nil {
			return "", "", nil, err
		}
		if reserved != nil {
			list = append([]problem.Problem{*reserved}, list...)
		}
	}
	return in.Code, in.Name, list, nil
}

// NormalizeCode merapikan kode customer seperti saat disimpan (huruf besar).
func NormalizeCode(code string) string { return normalizeCode(code) }
