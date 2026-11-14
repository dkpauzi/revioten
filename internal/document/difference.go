package document

// Perbedaan lokal dan putus tautan (M4.8, R-CMP-04).
//
// Baris hasil perakitan tetap tertaut ke poin template-nya. Bila penyusun
// mengubah isinya, baris itu "berbeda dari template". Perbedaan tidak
// disimpan: dihitung saat dilihat, dengan membandingkan isi baris dan isi
// yang akan ditulis poin itu di versi template yang dipakai merakit.
// Penyusun yang memang ingin baris berdiri sendiri memutus tautannya, dengan
// alasan; baris yang putus tidak lagi dibandingkan.

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Keadaan tautan satu baris terhadap poin template-nya.
const (
	LinkSame      = "same"      // isinya sama dengan template
	LinkDifferent = "different" // ada perubahan lokal
	LinkDetached  = "detached"  // tautan diputus dengan alasan
)

const maxDetachReason = 500

// RowDifference adalah perbandingan satu baris tertaut dengan poin template-nya.
type RowDifference struct {
	RowID     string
	Position  int
	PointID   string
	PointText string // teks poin atau varian yang berlaku untuk model
	Status    string
	Reason    string            // alasan putus tautan
	Fields    []FieldDifference // kolom yang berbeda; tidak pernah nil
}

// FieldDifference adalah satu kolom yang isinya berbeda. Nilai kosong ditulis "".
type FieldDifference struct {
	FieldID  string // ID kolom format; "" untuk proses
	Label    string
	Template any
	Row      any
}

// Differences membandingkan setiap baris tertaut dokumen ini dengan poin
// template-nya (L-4). Baris yang diketik atau ditempel sendiri tidak ikut.
func (s *Service) Differences(ctx context.Context, actor access.Actor, id string) ([]RowDifference, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	format, err := template.LoadFormatVersion(ctx, s.q, d.FormatVersionID)
	if err != nil {
		return nil, result(err, "membaca format")
	}
	model, err := s.q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return nil, result(err, "membaca model")
	}
	// Poin yang berlaku per versi template; satu dokumen bisa dirakit dari
	// beberapa process template.
	type version struct {
		process template.ProcessRef
		points  map[string]template.ResolvedPoint
	}
	versions := map[string]version{}
	out := []RowDifference{}
	for _, r := range d.Rows {
		if r.TemplatePointID == "" {
			continue
		}
		item := RowDifference{RowID: r.ID, Position: r.Position, PointID: r.TemplatePointID, Status: LinkSame, Fields: []FieldDifference{}}
		v, ok := versions[r.TemplateVersionID]
		if !ok {
			loaded, err := template.LoadProcessVersion(ctx, s.q, r.TemplateVersionID)
			if err != nil {
				return nil, result(err, "membaca versi process template")
			}
			v = version{process: loaded.Process, points: map[string]template.ResolvedPoint{}}
			for _, sub := range template.ResolveFor(loaded, model.ID, model.CustomerID, deref(model.ModelGroupID)) {
				for _, p := range sub.Points {
					v.points[p.OriginID] = p
				}
			}
			versions[r.TemplateVersionID] = v
		}
		point, known := v.points[r.TemplatePointID]
		item.PointText = point.Text
		if r.Detached {
			item.Status, item.Reason = LinkDetached, r.DetachReason
			out = append(out, item)
			continue
		}
		if known {
			item.Fields = compare(format, v.process, point, r)
		}
		if len(item.Fields) > 0 {
			item.Status = LinkDifferent
		}
		out = append(out, item)
	}
	return out, nil
}

// compare memberi kolom yang isinya berbeda antara baris dan poin. Hanya
// kolom ber-key yang dibandingkan, karena hanya kolom itu yang bisa diisi
// template (D-146). Proses baris dibandingkan dengan proses template.
func compare(format template.FormatVersion, process template.ProcessRef, point template.ResolvedPoint, r Row) []FieldDifference {
	out := []FieldDifference{}
	if r.ProcessID != process.ID {
		label := "Process"
		if i := slices.IndexFunc(format.Fields, func(f template.Field) bool { return f.Area == template.AreaRow && f.Kind == template.KindProcess }); i >= 0 {
			label = format.Fields[i].Label
		}
		out = append(out, FieldDifference{Label: label, Template: process.Name, Row: r.ProcessName})
	}
	expected := rowContent(format, point.Content)
	for _, f := range format.Fields {
		// Relevant Document diisi dari master dokumen terkait, bukan dari template (D-130).
		if f.Area != template.AreaRow || f.Key == "" || f.Hidden || f.Key == template.KeyRelevantDocument {
			continue
		}
		want, got := shown(expected[f.ID]), shown(r.Content[f.ID])
		if !sameValue(want, got) {
			out = append(out, FieldDifference{FieldID: f.ID, Label: f.Label, Template: want, Row: got})
		}
	}
	return out
}

// shown mengganti nilai yang tidak ada dengan "", supaya layar selalu punya isi.
func shown(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// sameValue membandingkan dua nilai lewat bentuk JSON-nya, supaya angka 7
// dan 7.0 dianggap sama.
func sameValue(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// Detach memutus tautan satu baris ke poin template-nya (E-2). Alasan wajib
// (R-CMP-04). Baris tetap ada dan tetap mengingat poin asalnya.
func (s *Service) Detach(ctx context.Context, actor access.Actor, id, rowID, reason string) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, _ template.FormatVersion) (string, map[string]string, error) {
		i := slices.IndexFunc(d.Rows, func(r Row) bool { return r.ID == rowID })
		if i < 0 {
			return "", nil, problem.ErrNotFound
		}
		reason, p := problem.Text("reason", reason, true, maxDetachReason)
		if err := problems(p); err != nil {
			return "", nil, err
		}
		notLinked := problems(invalid("row", problem.NotAllowed, "Only a row that is linked to a process template point can be detached."))
		if d.Rows[i].TemplatePointID == "" || d.Rows[i].Detached {
			return "", nil, notLinked
		}
		changed, err := q.DetachDocumentRow(ctx, db.DetachDocumentRowParams{ID: rowID, DocumentID: d.ID, DetachReason: &reason})
		if err != nil {
			return "", nil, err
		}
		if changed == 0 {
			return "", nil, notLinked
		}
		return audit.ActionDocumentRowDetached, map[string]string{"position": strconv.Itoa(d.Rows[i].Position), "reason": reason}, nil
	})
}
