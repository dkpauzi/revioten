package document

// Kolom Relevant Document dari master dokumen terkait (D-130, D-156). Baris
// pertama setiap proses mendapat nomor dokumen yang tertaut ke proses itu
// dan berlaku untuk model dokumen, satu nomor per baris. Sel yang sudah
// berisi tidak pernah ditimpa, jadi perubahan penyusun tetap.

import (
	"context"
	"strconv"
	"strings"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// relevantField memberi ID kolom Relevant Document di format, atau "" bila
// format itu tidak punya kolomnya (misalnya PFMEA).
func relevantField(format template.FormatVersion) string {
	for _, f := range format.Fields {
		if f.Area == template.AreaRow && f.Key == template.KeyRelevantDocument && !f.Hidden {
			return f.ID
		}
	}
	return ""
}

// FillReferences mengisi kolom Relevant Document yang masih kosong dari
// master dokumen terkait (E-2). Untuk baris yang diketik sendiri, atau
// sesudah master diubah.
func (s *Service) FillReferences(ctx context.Context, actor access.Actor, id string) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error) {
		if relevantField(format) == "" {
			return "", nil, problems(invalid("format", problem.NotAllowed, "This form has no Relevant Document column."))
		}
		n, err := fillReferences(ctx, q, d, format)
		return audit.ActionDocumentRefsFilled, map[string]string{"rows": strconv.Itoa(n)}, err
	})
}

// fillReferences mengisi sel kosong baris pertama setiap proses dan memberi
// banyaknya baris yang terisi. Model yang dipakai model pertama dokumen
// (ASUMSI E-93, D-155 pola 1). Dipanggil di dalam transaksi pemanggil, yang
// sudah mengunci dokumen.
func fillReferences(ctx context.Context, q *db.Queries, d Document, format template.FormatVersion) (int, error) {
	field := relevantField(format)
	if field == "" || len(d.Models) == 0 {
		return 0, nil
	}
	model, err := q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return 0, err
	}
	// Baris pertama setiap proses dihitung per bagian (D-155), karena tiap
	// bagian dicetak dengan daftar prosesnya sendiri.
	seen := map[string]bool{}
	filled := 0
	for _, r := range d.Rows { // sudah urut menurut posisi
		key := r.SectionID + "/" + r.ProcessID
		if r.ProcessID == "" || seen[key] {
			continue
		}
		seen[key] = true
		if text, _ := r.Content[field].(string); strings.TrimSpace(text) != "" {
			continue
		}
		numbers, err := template.ReferencedNumbers(ctx, q, r.ProcessID, model.ID, model.CustomerID, deref(model.ModelGroupID))
		if err != nil {
			return 0, err
		}
		if len(numbers) == 0 {
			continue
		}
		content := map[string]any{}
		for k, v := range r.Content {
			content[k] = v
		}
		content[field] = strings.Join(numbers, "\n")
		_, data, err := checkRow(ctx, q, format, r.ProcessID, r.OperationNumber, content)
		if err != nil {
			return 0, err
		}
		changed, err := q.UpdateDocumentRow(ctx, db.UpdateDocumentRowParams{
			ID: r.ID, DocumentID: d.ID, ProcessID: optional(r.ProcessID), OperationNumber: optional(r.OperationNumber),
			Content: data, ExpectedVersion: int32(r.Version), OperationManual: r.OperationManual,
		})
		if err != nil {
			return 0, err
		}
		if changed == 0 {
			return 0, conflict("version", "Someone else changed a row. Load the document again.")
		}
		filled++
	}
	return filled, nil
}
