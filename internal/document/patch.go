package document

// Operasi patch dari usulan perubahan luar (M9.4, R-API-07, R-API-11).
//
// Aplikasi lain tidak pernah mengubah dokumen secara langsung. Ia mengirim
// daftar operasi (tambah, ubah, hapus baris menurut ID baris yang tetap)
// terhadap satu revisi terbit. Operasi itu disimpan bersama perubahannya, dan
// baru ditulis ke draft sesudah seorang verifikator menerimanya.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Jenis operasi patch.
const (
	PatchAdd    = "add"
	PatchModify = "modify"
	PatchRemove = "remove"
)

// MaxPatchOps membatasi jumlah operasi dalam satu usulan.
const MaxPatchOps = 200

// PatchOp adalah satu operasi terhadap satu baris.
type PatchOp struct {
	Op string `json:"op"`
	// RowID wajib untuk modify dan remove: ID baris yang tetap (R-DOC-02).
	RowID string `json:"row_id,omitempty"`
	// ProcessID opsional untuk add.
	ProcessID string `json:"process_id,omitempty"`
	// SectionID wajib untuk add bila dokumen punya bagian per PCB (D-155).
	SectionID string `json:"section_id,omitempty"`
	// Content berisi isi per ID kolom format. Untuk modify, hanya kolom yang
	// disebut yang berubah; nilai null mengosongkan kolom itu.
	Content map[string]any `json:"content,omitempty"`
}

// merged menggabungkan isi lama dengan perubahan satu operasi modify.
func merged(old, change map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range old {
		out[k] = v
	}
	for k, v := range change {
		if v == nil {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

// checkPatch memeriksa daftar operasi terhadap baris yang ada. rows memetakan
// ID baris ke isinya; sections berisi ID bagian dokumen.
func checkPatch(ctx context.Context, q *db.Queries, format template.FormatVersion, rows map[string]map[string]any, sections []string, ops []PatchOp) error {
	if len(ops) == 0 {
		return problems(invalid("operations", problem.Required, "Send at least one operation."))
	}
	if len(ops) > MaxPatchOps {
		return problems(invalid("operations", problem.TooLong, fmt.Sprintf("Send at most %d operations.", MaxPatchOps)))
	}
	touched := map[string]bool{}
	for i, op := range ops {
		field := "operations." + strconv.Itoa(i)
		switch op.Op {
		case PatchAdd:
			if op.RowID != "" {
				return problems(invalid(field+".row_id", problem.NotAllowed, "A new row gets its ID from the system."))
			}
			if op.ProcessID != "" {
				exists, err := q.ProcessExists(ctx, op.ProcessID)
				if !db.IsUUID(op.ProcessID) || err != nil || !exists {
					return problems(invalid(field+".process_id", problem.InvalidFormat, "Choose an existing process."))
				}
			}
			if len(sections) > 0 && !slices.Contains(sections, op.SectionID) {
				return problems(invalid(field+".section_id", problem.Required, "Choose a section of the document."))
			}
			if len(sections) == 0 && op.SectionID != "" {
				return problems(invalid(field+".section_id", problem.NotAllowed, "The document has no sections."))
			}
			if err := checkContent(format, template.AreaRow, merged(nil, op.Content)); err != nil {
				return atRow(i, err)
			}
		case PatchModify, PatchRemove:
			old, ok := rows[op.RowID]
			if !ok {
				return problems(invalid(field+".row_id", problem.InvalidFormat, "Choose a row of the base revision."))
			}
			if touched[op.RowID] {
				return problems(invalid(field+".row_id", problem.NotAllowed, "A row is changed at most once in one proposal."))
			}
			touched[op.RowID] = true
			if op.Op == PatchModify {
				if len(op.Content) == 0 {
					return problems(invalid(field+".content", problem.Required, "Send the columns to change."))
				}
				if err := checkContent(format, template.AreaRow, merged(old, op.Content)); err != nil {
					return atRow(i, err)
				}
			}
		default:
			return problems(invalid(field+".op", problem.InvalidFormat, "Use add, modify, or remove."))
		}
	}
	return nil
}

// CheckPatchAgainstSnapshot memeriksa operasi terhadap satu revisi terbit,
// saat usulan diterima dari luar.
func CheckPatchAgainstSnapshot(ctx context.Context, q *db.Queries, snap Snapshot, ops []PatchOp) error {
	format, err := template.LoadFormatVersion(ctx, q, snap.FormatVersionID)
	if err != nil {
		return err
	}
	rows := map[string]map[string]any{}
	for _, r := range snap.Rows {
		rows[r.ID] = r.Content
	}
	var sections []string
	for _, s := range snap.Sections {
		sections = append(sections, s.ID)
	}
	return checkPatch(ctx, q, format, rows, sections, ops)
}

// ApplyPatch menulis operasi ke draft terbuka satu dokumen, di dalam queries
// q (transaksi pemanggil). Dipanggil saat usulan diterima verifikator. Baris
// yang sudah tidak ada di draft membuat seluruh penerapan ditolak: sistem
// tidak pernah menggabung perubahan luar secara diam-diam (R-API-11).
func ApplyPatch(ctx context.Context, q *db.Queries, docID string, ops []PatchOp) (added, modified, removed int, err error) {
	d, err := getDocument(ctx, q, docID)
	if err != nil {
		return 0, 0, 0, err
	}
	if err := requireDraft(d); err != nil {
		return 0, 0, 0, err
	}
	format, err := template.LoadFormatVersion(ctx, q, d.FormatVersionID)
	if err != nil {
		return 0, 0, 0, err
	}
	rows := map[string]map[string]any{}
	for _, r := range d.Rows {
		rows[r.ID] = r.Content
	}
	var sections []string
	for _, s := range d.Sections {
		sections = append(sections, s.ID)
	}
	if err := checkPatch(ctx, q, format, rows, sections, ops); err != nil {
		return 0, 0, 0, err
	}
	for _, op := range ops {
		switch op.Op {
		case PatchAdd:
			content, err := json.Marshal(merged(nil, op.Content))
			if err != nil {
				return 0, 0, 0, err
			}
			if _, err := q.AddDocumentRow(ctx, db.AddDocumentRowParams{
				DocumentID: docID, ProcessID: optional(op.ProcessID), Content: content, SectionID: optional(op.SectionID),
			}); err != nil {
				return 0, 0, 0, err
			}
			added++
		case PatchModify:
			content, err := json.Marshal(merged(rows[op.RowID], op.Content))
			if err != nil {
				return 0, 0, 0, err
			}
			if _, err := q.PatchRowContent(ctx, db.PatchRowContentParams{ID: op.RowID, DocumentID: docID, Content: content}); err != nil {
				return 0, 0, 0, err
			}
			modified++
		case PatchRemove:
			gone, err := q.DeleteDocumentRow(ctx, db.DeleteDocumentRowParams{ID: op.RowID, DocumentID: docID})
			if err != nil {
				return 0, 0, 0, err
			}
			if err := q.ShiftRowsUp(ctx, db.ShiftRowsUpParams{DocumentID: docID, Position: gone.Position}); err != nil {
				return 0, 0, 0, err
			}
			if gone.TemplatePointID != nil {
				if err := q.ExcludePoint(ctx, db.ExcludePointParams{DocumentID: docID, PointID: *gone.TemplatePointID}); err != nil {
					return 0, 0, 0, err
				}
			}
			removed++
		}
	}
	return added, modified, removed, q.TouchDocument(ctx, docID)
}
