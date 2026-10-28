// Package process berisi aturan bisnis master proses (UC-04, M1.5). Setiap
// proses punya ID tetap, satu nama tampilan, dan boleh punya alias (D-06,
// R-DOC-04). Data lain menunjuk proses lewat ID-nya, jadi nama bisa diganti
// tanpa memutus rujukan. Master ini dikelola PPAP Engineer lewat hak E-6
// (D-64), dan proses tidak pernah dihapus.
//
// ASUMSI E-33: master ini hanya berisi proses, misalnya IQC. Sub-proses
// (IQC MSD, IQC PCB) menyusul bersama process template di M3.2 (R-TPL-01).
package process

import (
	"fmt"
	"strings"
	"time"

	"github.com/dkpauzi/revioten/internal/problem"
)

// Process adalah satu langkah produksi, misalnya IQC. ID-nya tetap; nama
// hanya tampilan dan bisa diganti.
type Process struct {
	ID   string
	Name string
	// Aliases adalah nama lain proses yang sama, misalnya nama di dokumen
	// lama (F-05). Urut menurut abjad, dan tidak pernah nil.
	Aliases   []string
	CreatedAt time.Time // selalu UTC
}

// Input adalah isian proses. Saat proses diubah, Aliases menggantikan
// seluruh alias lama.
type Input struct {
	Name    string
	Aliases []string
}

// Jenis nama di tabel process_names.
const (
	kindDisplay = "display"
	kindAlias   = "alias"
)

// Batas isian.
// ASUMSI E-32: brief belum menyebut batas ini.
const (
	maxNameLength = 200
	maxAliases    = 20
)

// cleanName merapikan dan memeriksa satu nama: aturan teks master biasa
// (wajib, paling banyak 200 karakter, satu baris), lalu spasi berturut-turut
// di tengah dijadikan satu. Dengan begitu "Part  Receiving" dan
// "Part Receiving" tidak menjadi dua nama (ASUMSI E-32).
func cleanName(field, value string) (string, *problem.Problem) {
	name, p := problem.Text(field, value, true, maxNameLength)
	if p != nil {
		return "", p
	}
	return strings.Join(strings.Fields(name), " "), nil
}

// clean merapikan dan memeriksa seluruh isian. Untuk alias, hanya masalah
// pertama yang dilaporkan, supaya layar cukup menampilkan satu pesan di
// bawah kotak alias. Nama atau alias yang sudah dipakai diperiksa database,
// bukan di sini.
func clean(in Input) (name string, aliases []string, err error) {
	name, nameProblem := cleanName("name", in.Name)
	aliases, aliasProblem := cleanAliases(in.Aliases)
	var problems []problem.Problem
	for _, p := range []*problem.Problem{nameProblem, aliasProblem} {
		if p != nil {
			problems = append(problems, *p)
		}
	}
	if len(problems) > 0 {
		return "", nil, &problem.ValidationError{Problems: problems}
	}
	return name, aliases, nil
}

// cleanAliases merapikan semua alias dengan aturan yang sama seperti nama.
func cleanAliases(values []string) ([]string, *problem.Problem) {
	if len(values) > maxAliases {
		return nil, &problem.Problem{Field: "aliases", Code: problem.TooLong,
			Message: fmt.Sprintf("Use at most %d aliases.", maxAliases)}
	}
	aliases := make([]string, 0, len(values))
	for _, value := range values {
		alias, p := cleanName("aliases", value)
		if p != nil {
			return nil, p
		}
		aliases = append(aliases, alias)
	}
	return aliases, nil
}
