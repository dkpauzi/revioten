package printout

// Nomor karakteristik Control Plan (D-222), seperti kolom F dan I di
// SC-TR-020-04. Aturan pemilik (23 November 2026): No produk diisi pada baris
// yang Product Characteristics-nya diisi, No proses pada baris yang Process
// Characteristics-nya diisi, nomornya berurutan. Nomor mulai lagi dari 1 di
// setiap proses, seperti di referensi: baris berurutan dengan bagian, nomor
// langkah, dan nama proses yang sama. Karakteristik berisi "--" mendapat
// "--", dan yang kosong dibiarkan kosong.

import (
	"strconv"
	"strings"

	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/template"
)

// charNumbers memberi nomor produk dan proses setiap baris menurut ID baris.
func charNumbers(format template.FormatVersion, rows []document.SnapshotRow) map[string][2]string {
	product, process := "", ""
	for _, f := range format.Fields {
		switch f.Key {
		case "product_characteristic":
			product = f.ID
		case "process_characteristic":
			process = f.ID
		}
	}
	out := map[string][2]string{}
	counts := [2]int{}
	group := ""
	for _, r := range rows {
		key := r.SectionID + "\x00" + r.OperationNumber + "\x00" + r.ProcessName
		if key != group {
			group, counts = key, [2]int{}
		}
		var nums [2]string
		for i, id := range []string{product, process} {
			value := strings.TrimSpace(text(r.Content[id]))
			switch {
			case id == "" || value == "":
			case value == "--":
				nums[i] = "--"
			default:
				counts[i]++
				nums[i] = strconv.Itoa(counts[i])
			}
		}
		out[r.ID] = nums
	}
	return out
}
