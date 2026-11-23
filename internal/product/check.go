package product

// Pemeriksaan tanpa menyimpan, untuk Preview import master data (I5). Aturan
// yang dipakai sama dengan CreateModelGroup, CreateModel, dan CreatePartNumber.

import "github.com/dkpauzi/revioten/internal/problem"

// CheckName merapikan dan memeriksa nama group model atau model.
func CheckName(field, name string) (string, *problem.Problem) {
	return problem.Text(field, name, true, maxNameLength)
}

// CheckPartNumber merapikan dan memeriksa nomor dan nama part number.
func CheckPartNumber(number, name string) (string, string, []problem.Problem) {
	n, p1 := problem.Text("part_number", number, true, maxNumberLength)
	m, p2 := problem.Text("part_name", name, false, maxNameLength)
	var list []problem.Problem
	for _, p := range []*problem.Problem{p1, p2} {
		if p != nil {
			list = append(list, *p)
		}
	}
	return n, m, list
}
