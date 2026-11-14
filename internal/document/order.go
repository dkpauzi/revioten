package document

// Urutan baris dan nomor operasi (M4.6, R-CMP-01).
//
// Penyusun mengirim seluruh ID baris menurut urutan baru. Bila diminta,
// nomor operasi diisi otomatis per langkah: baris berurutan dengan proses
// yang sama adalah satu langkah dan mendapat satu nomor, kelipatan 10.
// Nomor yang diketik atau ditempel penyusun (operation_manual) tidak ditimpa.

import (
	"context"
	"strconv"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// operationStep adalah jarak nomor operasi otomatis: 10, 20, 30, dan
// seterusnya, supaya langkah baru bisa disisipkan (ASUMSI E-94).
const operationStep = 10

// Order adalah urutan baru baris satu dokumen.
type Order struct {
	RowIDs   []string // semua baris dokumen, tepat satu kali
	Renumber bool     // isi nomor operasi otomatis sesudah diurutkan
}

// Reorder menyimpan urutan baru baris (E-2), dan mengisi nomor operasi
// otomatis bila diminta.
func (s *Service) Reorder(ctx context.Context, actor access.Actor, id string, in Order) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, _ template.FormatVersion) (string, map[string]string, error) {
		bad := problems(invalid("row_ids", problem.InvalidFormat, "Send every row of the document exactly once."))
		if len(in.RowIDs) == 0 || len(in.RowIDs) != len(d.Rows) {
			return "", nil, bad
		}
		byID := map[string]Row{}
		for _, r := range d.Rows {
			byID[r.ID] = r
		}
		ordered := make([]Row, 0, len(in.RowIDs))
		seen := map[string]bool{}
		for _, rowID := range in.RowIDs {
			r, ok := byID[rowID]
			if !ok || seen[rowID] {
				return "", nil, bad
			}
			seen[rowID] = true
			ordered = append(ordered, r)
		}
		for i, r := range ordered {
			if r.Position == i+1 {
				continue
			}
			if err := q.SetRowPosition(ctx, db.SetRowPositionParams{ID: r.ID, DocumentID: d.ID, Position: int32(i + 1)}); err != nil {
				return "", nil, err
			}
		}
		if in.Renumber {
			for i, number := range operationNumbers(ordered) {
				r := ordered[i]
				if r.OperationManual || r.OperationNumber == number {
					continue
				}
				if err := q.SetRowOperationNumber(ctx, db.SetRowOperationNumberParams{ID: r.ID, DocumentID: d.ID, OperationNumber: &number}); err != nil {
					return "", nil, err
				}
			}
		}
		return audit.ActionDocumentRowsReordered, map[string]string{
			"count": strconv.Itoa(len(ordered)), "renumbered": strconv.FormatBool(in.Renumber),
		}, nil
	})
}

// operationNumbers memberi nomor operasi otomatis setiap baris menurut
// urutannya. Baris berurutan dengan proses yang sama adalah satu langkah;
// baris tanpa proses menjadi langkah sendiri.
func operationNumbers(rows []Row) []string {
	out := make([]string, len(rows))
	step := 0
	for i, r := range rows {
		if i == 0 || r.ProcessID == "" || r.ProcessID != rows[i-1].ProcessID {
			step++
		}
		out[i] = strconv.Itoa(step * operationStep)
	}
	return out
}

// isManual memutuskan tanda manual sesudah baris disimpan: nomor yang diganti
// penyusun menjadi manual, nomor yang dikosongkan kembali otomatis, dan nomor
// yang tidak disentuh membawa tanda lamanya.
func isManual(old Row, operation string) bool {
	if operation == "" {
		return false
	}
	if operation != old.OperationNumber {
		return true
	}
	return old.OperationManual
}
