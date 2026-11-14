package document

// Penautan baris ke poin template (M7.4, R-MIG-05, RK-12).
//
// Baris yang diketik atau ditempel sendiri tidak punya tautan ke poin
// template, jadi tidak pernah menerima propagasi. Di sini sistem mengusulkan
// pasangan baris dan poin lewat kemiripan teks, lalu orang mengonfirmasi.
// Sistem tidak pernah menautkan sendiri. Sesudah tertaut, isi baris yang
// berbeda dari template tampil sebagai perbedaan lokal (M4.8), dan versi
// template berikutnya bisa dipropagasi ke baris itu (M5.8).

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// minLinkScore adalah kemiripan terendah (0 sampai 100) supaya satu pasangan
// diusulkan. ASUMSI E-114: 60 dipilih sebagai titik awal dan belum diukur
// dengan data asli.
const minLinkScore = 60

// LinkSuggestion adalah satu usulan pasangan baris dan poin template.
type LinkSuggestion struct {
	RowID     string
	Position  int
	RowText   string // isi baris yang paling mirip
	PointID   string // ID poin asal
	VersionID string // versi terbit terakhir process template proses baris itu
	PointText string
	Matched   string // teks poin yang mirip dengan RowText
	Score     int    // 0 sampai 100
}

// RowLink adalah satu pasangan yang dikonfirmasi orang.
type RowLink struct {
	RowID     string
	PointID   string
	VersionID string
}

// normalize menyamakan huruf besar-kecil dan spasi.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// bigrams memberi pasangan dua huruf berurutan beserta jumlahnya.
func bigrams(s string) map[string]int {
	r := []rune(s)
	out := map[string]int{}
	for i := 0; i+1 < len(r); i++ {
		out[string(r[i:i+2])]++
	}
	return out
}

// similarity memberi kemiripan dua teks, 0 sampai 100, dengan koefisien Dice
// atas pasangan dua huruf: dua kali jumlah pasangan yang sama, dibagi jumlah
// pasangan kedua teks. "Parts Receiving" dan "Part Receiving" tetap mirip,
// walau tidak sama persis.
func similarity(a, b string) int {
	a, b = normalize(a), normalize(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 100
	}
	x, y := bigrams(a), bigrams(b)
	total, same := 0, 0
	for k, n := range x {
		total += n
		same += min(n, y[k])
	}
	for _, n := range y {
		total += n
	}
	if total == 0 {
		return 0
	}
	return 200 * same / total
}

// texts memberi semua isi teks satu baris atau satu poin.
func texts(content map[string]any) []string {
	var out []string
	for _, v := range content {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// candidates memberi poin yang bisa ditautkan untuk baris proses processID:
// poin jenis dokumen ini di versi terbit terakhir process template proses
// itu, yang berlaku untuk model dokumen. ok false berarti proses itu belum
// punya process template yang terbit.
func candidates(ctx context.Context, q *db.Queries, d Document, processID string) (string, []template.ResolvedPoint, bool, error) {
	versionID, err := q.LatestPublishedProcessVersion(ctx, processID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	v, err := template.LoadProcessVersion(ctx, q, versionID)
	if err != nil {
		return "", nil, false, err
	}
	model, err := q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return "", nil, false, err
	}
	var points []template.ResolvedPoint
	for _, sub := range template.ResolveFor(v, model.ID, model.CustomerID, deref(model.ModelGroupID)) {
		for _, p := range sub.Points {
			if p.DocumentType.Key == d.DocumentType.Key {
				points = append(points, p)
			}
		}
	}
	return versionID, points, true, nil
}

// LinkSuggestions mengusulkan poin template untuk baris yang belum tertaut
// (L-4). Satu poin diusulkan paling banyak untuk satu baris per bagian,
// dimulai dari pasangan yang paling mirip.
func (s *Service) LinkSuggestions(ctx context.Context, actor access.Actor, id string) ([]LinkSuggestion, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	type version struct {
		id     string
		points []template.ResolvedPoint
		ok     bool
	}
	versions := map[string]version{}
	var all []LinkSuggestion
	for _, r := range d.Rows {
		if r.TemplatePointID != "" || r.ProcessID == "" {
			continue
		}
		v, seen := versions[r.ProcessID]
		if !seen {
			vid, points, ok, err := candidates(ctx, s.q, d, r.ProcessID)
			if err != nil {
				return nil, result(err, "membaca process template")
			}
			v = version{id: vid, points: points, ok: ok}
			versions[r.ProcessID] = v
		}
		if !v.ok {
			continue
		}
		rowTexts := texts(r.Content)
		for _, p := range v.points {
			best := LinkSuggestion{RowID: r.ID, Position: r.Position, PointID: p.OriginID, VersionID: v.id, PointText: p.Text}
			for _, rt := range rowTexts {
				for _, pt := range append([]string{p.Text}, texts(p.Content)...) {
					if score := similarity(rt, pt); score > best.Score {
						best.Score, best.RowText, best.Matched = score, rt, pt
					}
				}
			}
			if best.Score >= minLinkScore {
				all = append(all, best)
			}
		}
	}
	// Pasangan paling mirip dipilih dulu; baris dan poin dipakai sekali.
	sort.SliceStable(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	section := map[string]string{}
	usedPoint := map[string]bool{}
	for _, r := range d.Rows {
		section[r.ID] = r.SectionID
		if r.TemplatePointID != "" {
			usedPoint[r.SectionID+"/"+r.TemplatePointID] = true
		}
	}
	usedRow := map[string]bool{}
	out := []LinkSuggestion{}
	for _, c := range all {
		key := section[c.RowID] + "/" + c.PointID
		if usedRow[c.RowID] || usedPoint[key] {
			continue
		}
		usedRow[c.RowID], usedPoint[key] = true, true
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

// LinkRows menautkan baris ke poin template sesudah dikonfirmasi orang (E-2).
// Isi baris tidak berubah.
func (s *Service) LinkRows(ctx context.Context, actor access.Actor, id string, links []RowLink) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	if len(links) == 0 {
		return Document{}, problems(invalid("links", problem.Required, "Choose at least one row to link."))
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if err := requireDraft(d); err != nil {
			return err
		}
		usedPoint := map[string]bool{}
		for _, r := range d.Rows {
			if r.TemplatePointID != "" {
				usedPoint[r.SectionID+"/"+r.TemplatePointID] = true
			}
		}
		type version struct {
			id     string
			points []template.ResolvedPoint
		}
		versions := map[string]version{}
		for i, l := range links {
			field := "links." + strconv.Itoa(i)
			at := slices.IndexFunc(d.Rows, func(r Row) bool { return r.ID == l.RowID })
			if at < 0 {
				return problems(invalid(field+".row_id", problem.InvalidFormat, "Choose a row of this document."))
			}
			r := d.Rows[at]
			if r.TemplatePointID != "" || r.ProcessID == "" {
				return problems(invalid(field+".row_id", problem.NotAllowed, "Choose a row that has a process and is not linked yet."))
			}
			v, seen := versions[r.ProcessID]
			if !seen {
				vid, points, _, err := candidates(ctx, q, d, r.ProcessID)
				if err != nil {
					return err
				}
				v = version{id: vid, points: points}
				versions[r.ProcessID] = v
			}
			known := slices.ContainsFunc(v.points, func(p template.ResolvedPoint) bool { return p.OriginID == l.PointID })
			if !known || l.VersionID != v.id {
				return problems(invalid(field+".point_id", problem.InvalidFormat, "Choose a point of the latest published process template of the row's process."))
			}
			key := r.SectionID + "/" + l.PointID
			if usedPoint[key] {
				return problems(invalid(field+".point_id", problem.NotAllowed, "Another row is already linked to this point."))
			}
			usedPoint[key] = true
			n, err := q.LinkDocumentRow(ctx, db.LinkDocumentRowParams{ID: r.ID, DocumentID: id, TemplatePointID: &l.PointID, TemplateVersionID: &l.VersionID})
			if err != nil {
				return err
			}
			if n == 0 {
				return problems(invalid(field+".row_id", problem.NotAllowed, "The row is already linked."))
			}
		}
		if err := q.TouchDocument(ctx, id); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentRowsLinked, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"number": d.Number, "rows": strconv.Itoa(len(links))}})
	})
	if err != nil {
		return Document{}, result(err, "menautkan baris")
	}
	return s.load(ctx, id)
}
