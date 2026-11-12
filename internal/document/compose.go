package document

// Perakitan dokumen dari process template (M4.5, R-CMP-01 sampai R-CMP-04).
//
// Penyusun memilih satu versi process template yang sudah terbit, lalu
// mencentang poin yang dipakai. Setiap poin yang dicentang menjadi satu baris
// yang tertaut ke poin asalnya (R-CMP-04); poin yang tidak dicentang dicatat
// sebagai dikecualikan, dengan alasan opsional (R-CMP-03). Poin yang belum
// pernah diputuskan, misalnya poin baru di versi template berikutnya, tampil
// sebagai "belum diputuskan".
//
// Satu perakitan mengisi PFD, PFMEA, dan CP sekaligus (R-CMP-02): poin PFD
// masuk ke dokumen PFD di set yang sama, dan seterusnya. Dokumen dari nomor
// tunggal hanya menerima poin jenisnya sendiri.

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Keadaan satu poin untuk satu dokumen (R-CMP-03).
const (
	StateIncluded  = "included"  // ada baris yang tertaut ke poin ini
	StateExcluded  = "excluded"  // dicatat tidak dipakai
	StateUndecided = "undecided" // belum pernah diputuskan
)

const maxExclusionReason = 500

// Composition adalah tampilan perakitan satu versi process template untuk
// satu dokumen beserta dokumen lain di set-nya.
type Composition struct {
	VersionID    string
	Version      int
	Process      Ref
	Subprocesses []ComposedSubprocess // tidak pernah nil
}

// ComposedSubprocess adalah sub-proses beserta poin yang berlaku untuk model dokumen.
type ComposedSubprocess struct {
	ID     string
	Name   string
	Points []ComposedPoint // tidak pernah nil
}

// ComposedPoint adalah satu poin dan keadaannya di dokumen tujuannya.
type ComposedPoint struct {
	PointID      string // ID poin asal; baris tertaut ke sini
	DocumentType TypeRef
	Text         string // teks poin atau varian yang berlaku untuk model
	DocumentID   string // dokumen tujuan di set (jenis dokumen yang sama)
	State        string
	Reason       string // alasan dikecualikan, boleh kosong
}

// Compose adalah keputusan penyusun atas poin satu versi process template.
// Poin yang tidak disebut tetap seperti semula.
type Compose struct {
	VersionID string
	Include   []string // ID poin asal yang dipakai
	Exclude   []Exclusion
}

// Exclusion adalah satu poin yang dikecualikan, dengan alasan opsional.
type Exclusion struct {
	PointID string
	Reason  string
}

// target adalah satu poin yang berlaku beserta dokumen tujuannya.
type target struct {
	point template.ResolvedPoint
	doc   Document
}

// composeContext memuat versi template, dokumen satu set, dan poin yang
// berlaku untuk model pertama dokumen (ASUMSI E-93).
func composeContext(ctx context.Context, q *db.Queries, d Document, versionID string) (template.ProcessVersion, []template.ResolvedSubprocess, map[string]Document, error) {
	v, err := template.LoadProcessVersion(ctx, q, versionID)
	if err != nil {
		return v, nil, nil, err
	}
	docs := map[string]Document{d.DocumentType.Key: d}
	for _, m := range d.Set {
		if m.ID == d.ID {
			continue
		}
		other, err := getDocument(ctx, q, m.ID)
		if err != nil {
			return v, nil, nil, err
		}
		docs[other.DocumentType.Key] = other
	}
	model, err := q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return v, nil, nil, err
	}
	return v, template.ResolveFor(v, model.ID, model.CustomerID, deref(model.ModelGroupID)), docs, nil
}

// Composition menampilkan poin satu versi process template untuk dokumen ini
// dan dokumen lain di set-nya, beserta keadaan setiap poin (L-4).
func (s *Service) Composition(ctx context.Context, actor access.Actor, id, versionID string) (Composition, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return Composition{}, err
	}
	v, resolved, docs, err := composeContext(ctx, s.q, d, versionID)
	if err != nil {
		return Composition{}, result(err, "membaca perakitan")
	}
	excluded := map[string]map[string]string{} // ID dokumen, lalu ID poin menjadi alasan
	for _, doc := range docs {
		rows, err := s.q.ListPointExclusions(ctx, doc.ID)
		if err != nil {
			return Composition{}, result(err, "membaca poin yang dikecualikan")
		}
		excluded[doc.ID] = map[string]string{}
		for _, r := range rows {
			excluded[doc.ID][r.PointID] = deref(r.Reason)
		}
	}
	out := Composition{VersionID: v.ID, Version: v.Version, Process: Ref{ID: v.Process.ID, Name: v.Process.Name}, Subprocesses: []ComposedSubprocess{}}
	for _, sub := range resolved {
		item := ComposedSubprocess{ID: sub.ID, Name: sub.Name, Points: []ComposedPoint{}}
		for _, p := range sub.Points {
			doc, ok := docs[p.DocumentType.Key]
			if !ok {
				continue // set ini tidak punya dokumen jenis itu
			}
			point := ComposedPoint{
				PointID: p.OriginID, Text: p.Text, DocumentID: doc.ID, State: StateUndecided,
				DocumentType: TypeRef{ID: p.DocumentType.ID, Key: p.DocumentType.Key, Name: p.DocumentType.Name},
			}
			if slices.ContainsFunc(doc.Rows, func(r Row) bool { return r.TemplatePointID == p.OriginID }) {
				point.State = StateIncluded
			} else if reason, ok := excluded[doc.ID][p.OriginID]; ok {
				point.State, point.Reason = StateExcluded, reason
			}
			item.Points = append(item.Points, point)
		}
		if len(item.Points) > 0 {
			out.Subprocesses = append(out.Subprocesses, item)
		}
	}
	return out, nil
}

// Compose menjalankan keputusan penyusun (E-2): poin yang dipakai mendapat
// baris bila belum punya, dan poin yang dikecualikan kehilangan barisnya dan
// dicatat. Semuanya dalam satu transaksi untuk seluruh dokumen di set.
func (s *Service) Compose(ctx context.Context, actor access.Actor, id string, in Compose) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Dokumen satu set dikunci menurut urutan ID, supaya dua perakitan
		// bersamaan tidak saling menunggu (deadlock).
		ids := []string{d.ID}
		for _, m := range d.Set {
			if m.ID != d.ID {
				ids = append(ids, m.ID)
			}
		}
		slices.Sort(ids)
		for _, docID := range ids {
			if err := q.LockDocument(ctx, docID); err != nil {
				return err
			}
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if !db.IsUUID(in.VersionID) {
			return problems(invalid("process_template_version_id", problem.InvalidFormat, "Choose a process template version."))
		}
		v, resolved, docs, err := composeContext(ctx, q, d, in.VersionID)
		if err != nil {
			return err
		}
		if v.Status != template.StatusPublished {
			return problems(invalid("process_template_version_id", problem.NotAllowed, "Only a published process template version can be used. Ask a template maintainer to publish it."))
		}

		// Poin yang berlaku, menurut urutan template, beserta dokumen tujuannya.
		var order []string
		targets := map[string]target{}
		for _, sub := range resolved {
			for _, p := range sub.Points {
				if doc, ok := docs[p.DocumentType.Key]; ok {
					order = append(order, p.OriginID)
					targets[p.OriginID] = target{point: p, doc: doc}
				}
			}
		}
		for _, pointID := range in.Include {
			if _, ok := targets[pointID]; !ok {
				return problems(invalid("include", problem.InvalidFormat, "Choose points of this process template version that apply to this document."))
			}
		}
		for _, e := range in.Exclude {
			if _, ok := targets[e.PointID]; !ok {
				return problems(invalid("exclude", problem.InvalidFormat, "Choose points of this process template version that apply to this document."))
			}
			if slices.Contains(in.Include, e.PointID) {
				return problems(invalid("exclude", problem.NotAllowed, "A point is either used or left out, not both."))
			}
		}

		formats := map[string]template.FormatVersion{}
		rowCount := map[string]int{}
		for _, doc := range docs {
			if formats[doc.ID], err = template.LoadFormatVersion(ctx, q, doc.FormatVersionID); err != nil {
				return err
			}
			rowCount[doc.ID] = len(doc.Rows)
		}
		touched := map[string]bool{}
		added, removed := 0, 0

		// Dikecualikan dulu, supaya batas jumlah baris dihitung sesudah baris dicoret.
		for _, e := range in.Exclude {
			t := targets[e.PointID]
			reason, p := problem.Text("exclude", e.Reason, false, maxExclusionReason)
			if err := problems(p); err != nil {
				return err
			}
			if i := slices.IndexFunc(t.doc.Rows, func(r Row) bool { return r.TemplatePointID == e.PointID }); i >= 0 {
				if err := removeRow(ctx, q, t.doc.ID, t.doc.Rows[i].ID); err != nil {
					return err
				}
				removed++
				rowCount[t.doc.ID]--
			}
			if err := q.ExcludePoint(ctx, db.ExcludePointParams{DocumentID: t.doc.ID, PointID: e.PointID, Reason: optional(reason)}); err != nil {
				return err
			}
			touched[t.doc.ID] = true
		}

		// Dipakai: menurut urutan template, bukan urutan kiriman.
		for _, pointID := range order {
			if !slices.Contains(in.Include, pointID) {
				continue
			}
			t := targets[pointID]
			if err := q.ClearPointExclusion(ctx, db.ClearPointExclusionParams{DocumentID: t.doc.ID, PointID: pointID}); err != nil {
				return err
			}
			touched[t.doc.ID] = true
			if slices.ContainsFunc(t.doc.Rows, func(r Row) bool { return r.TemplatePointID == pointID }) {
				continue // sudah punya baris; perubahan lokalnya dibiarkan
			}
			if rowCount[t.doc.ID] >= maxRows {
				return problems(invalid("include", problem.NotAllowed, "A document has at most 1000 rows."))
			}
			content, err := json.Marshal(rowContent(formats[t.doc.ID], t.point.Content))
			if err != nil {
				return err
			}
			if _, err := q.AddComposedRow(ctx, db.AddComposedRowParams{
				DocumentID: t.doc.ID, ProcessID: optional(v.Process.ID), Content: content,
				TemplatePointID: &pointID, TemplateVersionID: &v.ID,
			}); err != nil {
				return err
			}
			added++
			rowCount[t.doc.ID]++
		}

		for docID := range touched {
			if err := q.TouchDocument(ctx, docID); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentComposed, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{
				"process": v.Process.Name, "version": strconv.Itoa(v.Version),
				"added": strconv.Itoa(added), "removed": strconv.Itoa(removed), "excluded": strconv.Itoa(len(in.Exclude)),
			},
		})
	})
	if err != nil {
		return Document{}, result(err, "merakit dokumen")
	}
	return getDocument(ctx, s.q, id)
}

// removeRow menghapus satu baris dan menggeser baris sesudahnya naik.
func removeRow(ctx context.Context, q *db.Queries, documentID, rowID string) error {
	gone, err := q.DeleteDocumentRow(ctx, db.DeleteDocumentRowParams{ID: rowID, DocumentID: documentID})
	if err != nil {
		return err
	}
	return q.ShiftRowsUp(ctx, db.ShiftRowsUpParams{DocumentID: documentID, Position: gone.Position})
}

// rowContent menyusun isi baris dari isi poin (D-146): setiap kolom format
// yang punya key mengambil nilai poin dengan key yang sama. Nilai yang tidak
// cocok dengan jenis kolomnya dilewati, supaya baris tetap sah (R-DOC-01).
func rowContent(format template.FormatVersion, point map[string]any) map[string]any {
	content := map[string]any{}
	for _, f := range format.Fields {
		if f.Area != template.AreaRow || f.Key == "" {
			continue
		}
		if value, ok := point[f.Key]; ok {
			content[f.ID] = value
		}
	}
	for _, p := range template.CheckContent(format, template.AreaRow, content).Problems {
		delete(content, p.Field)
	}
	return content
}
