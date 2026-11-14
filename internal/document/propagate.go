package document

// Propagasi template (M5.8, R-TPL-08, R-TPL-09). Saat process template
// terbit di versi baru, dokumen yang mengikuti template itu bisa menerima isi
// barunya. Isi tidak pernah ditulis langsung ke dokumen: propagasi membuat
// perubahan (R-TPL-06), dan isi baru ditulis ke draft saat perubahan itu
// diterima (lihat internal/change). File ini berisi rencananya (pratinjau)
// dan penerapannya ke satu draft, serta mode tautan per dokumen.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Mode tautan dokumen ke satu process template (R-TPL-08).
const (
	ModeFollow   = "follow"   // ikut setiap versi baru (bawaan)
	ModePinned   = "pinned"   // tetap di versinya, kecuali propagasi bercakupan all
	ModeDetached = "detached" // tidak pernah ikut propagasi
)

// Keadaan satu baris di rencana propagasi.
const (
	PlanUpdate   = "update"   // isi template berubah dan baris belum diubah lokal
	PlanConflict = "conflict" // isi template berubah, tetapi baris sudah diubah lokal
	PlanRemoved  = "removed"  // poinnya tidak ada lagi untuk model ini di versi baru
	PlanSame     = "same"     // isi template tidak berubah
)

// TemplatePlan adalah rencana propagasi satu versi template ke satu dokumen.
type TemplatePlan struct {
	DocumentID string
	Number     string
	CustomerID string
	Status     string // status dokumen: draft atau released
	Mode       string
	Updates    int
	Conflicts  int
	Removed    int
	Rows       []PlanRow // baris yang berubah, berkonflik, atau poinnya hilang; tidak pernah nil
}

// PlanRow adalah satu baris di rencana propagasi.
type PlanRow struct {
	RowID     string
	Position  int
	Status    string
	PointText string
	Fields    []FieldDifference // isi baris sekarang dan isi template baru
}

// TemplateLink adalah satu process template yang dipakai baris dokumen.
type TemplateLink struct {
	TemplateID  string
	ProcessName string
	Version     int // versi terbaru yang dipakai baris dokumen ini
	Mode        string
	Reason      string
}

// planned menyimpan rencana satu baris beserta isi barunya.
type planned struct {
	row     Row
	status  string
	content map[string]any
}

// PlanTemplate menghitung rencana propagasi versi versionID ke satu dokumen,
// di dalam queries q (boleh transaksi). Mode tautan ikut dilaporkan; pemanggil
// yang memutuskan dokumen mana yang ikut.
func PlanTemplate(ctx context.Context, q *db.Queries, docID, versionID string) (TemplatePlan, error) {
	plan, _, err := planTemplate(ctx, q, docID, versionID)
	return plan, err
}

func planTemplate(ctx context.Context, q *db.Queries, docID, versionID string) (TemplatePlan, []planned, error) {
	d, err := getDocument(ctx, q, docID)
	if err != nil {
		return TemplatePlan{}, nil, err
	}
	next, err := template.LoadProcessVersion(ctx, q, versionID)
	if err != nil {
		return TemplatePlan{}, nil, fmt.Errorf("membaca versi template: %w", err)
	}
	format, err := template.LoadFormatVersion(ctx, q, d.FormatVersionID)
	if err != nil {
		return TemplatePlan{}, nil, err
	}
	model, err := q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return TemplatePlan{}, nil, fmt.Errorf("membaca model: %w", err)
	}
	mode, err := q.GetDocumentTemplateMode(ctx, db.GetDocumentTemplateModeParams{DocumentID: docID, ProcessTemplateID: next.TemplateID})
	if errors.Is(err, pgx.ErrNoRows) {
		mode, err = ModeFollow, nil
	}
	if err != nil {
		return TemplatePlan{}, nil, err
	}
	points := func(v template.ProcessVersion) map[string]template.ResolvedPoint {
		out := map[string]template.ResolvedPoint{}
		for _, sub := range template.ResolveFor(v, model.ID, model.CustomerID, deref(model.ModelGroupID)) {
			for _, p := range sub.Points {
				if p.DocumentType.Key == d.DocumentType.Key {
					out[p.OriginID] = p
				}
			}
		}
		return out
	}
	newPoints := points(next)
	type old struct {
		v      template.ProcessVersion
		points map[string]template.ResolvedPoint
	}
	olds := map[string]old{}
	plan := TemplatePlan{DocumentID: d.ID, Number: d.Number, CustomerID: d.CustomerID, Status: d.Status, Mode: mode, Rows: []PlanRow{}}
	var rows []planned
	for _, r := range d.Rows {
		if r.TemplatePointID == "" || r.Detached || r.TemplateVersionID == next.ID {
			continue
		}
		o, ok := olds[r.TemplateVersionID]
		if !ok {
			v, err := template.LoadProcessVersion(ctx, q, r.TemplateVersionID)
			if err != nil {
				return TemplatePlan{}, nil, fmt.Errorf("membaca versi template baris: %w", err)
			}
			o = old{v: v, points: points(v)}
			olds[r.TemplateVersionID] = o
		}
		if o.v.TemplateID != next.TemplateID {
			continue // baris dari process template lain
		}
		before, after := o.points[r.TemplatePointID], newPoints[r.TemplatePointID]
		item := planned{row: r}
		switch {
		case after.OriginID == "":
			item.status = PlanRemoved
		case sameValue(rowContent(format, before.Content), rowContent(format, after.Content)):
			item.status = PlanSame
		case len(compare(format, o.v.Process, before, r)) > 0:
			item.status = PlanConflict
		default:
			item.status = PlanUpdate
		}
		if item.status != PlanRemoved {
			item.content = withTemplate(format, r.Content, rowContent(format, after.Content))
		}
		rows = append(rows, item)
		if item.status == PlanSame {
			continue
		}
		line := PlanRow{RowID: r.ID, Position: r.Position, Status: item.status, PointText: before.Text, Fields: []FieldDifference{}}
		switch item.status {
		case PlanUpdate:
			plan.Updates++
		case PlanConflict:
			plan.Conflicts++
		case PlanRemoved:
			plan.Removed++
		}
		if item.status != PlanRemoved {
			line.PointText = after.Text
			line.Fields = compare(format, next.Process, after, r)
		}
		plan.Rows = append(plan.Rows, line)
	}
	return plan, rows, nil
}

// withTemplate mengganti kolom ber-key dengan isi template baru. Kolom tanpa
// key dan Relevant Document (D-130) tetap isi baris.
func withTemplate(format template.FormatVersion, current, fromTemplate map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range current {
		out[k] = v
	}
	for _, f := range format.Fields {
		if f.Area != template.AreaRow || f.Key == "" || f.Key == template.KeyRelevantDocument {
			continue
		}
		if v, ok := fromTemplate[f.ID]; ok {
			out[f.ID] = v
		} else {
			delete(out, f.ID)
		}
	}
	return out
}

// ApplyTemplate menulis versi versionID ke draft dokumen docID di dalam
// transaksi q (M5.8). Baris yang berkonflik hanya ditimpa bila overwrite;
// bila tidak, baris itu dibiarkan dan dihitung sebagai konflik. Baris yang
// poinnya hilang dibiarkan dan dihitung (ASUMSI E-105).
func ApplyTemplate(ctx context.Context, q *db.Queries, docID, versionID string, overwrite bool) (updated, conflicts, removed int, err error) {
	_, rows, err := planTemplate(ctx, q, docID, versionID)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, p := range rows {
		write := p.status == PlanSame || p.status == PlanUpdate || (p.status == PlanConflict && overwrite)
		switch {
		case p.status == PlanRemoved:
			removed++
		case p.status == PlanConflict && !overwrite:
			conflicts++
		case p.status != PlanSame:
			updated++
		}
		if !write {
			continue
		}
		content, err := json.Marshal(p.content)
		if err != nil {
			return 0, 0, 0, err
		}
		if err := q.UpdateRowFromTemplate(ctx, db.UpdateRowFromTemplateParams{ID: p.row.ID, DocumentID: docID, Content: content, TemplateVersionID: &versionID}); err != nil {
			return 0, 0, 0, fmt.Errorf("menulis isi template: %w", err)
		}
	}
	return updated, conflicts, removed, nil
}

// Templates memberi process template yang dipakai dokumen beserta modenya (L-4).
func (s *Service) Templates(ctx context.Context, actor access.Actor, id string) ([]TemplateLink, error) {
	if _, err := s.Get(ctx, actor, id); err != nil {
		return nil, err
	}
	rows, err := s.q.ListDocumentTemplates(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("membaca process template dokumen: %w", err)
	}
	out := make([]TemplateLink, 0, len(rows))
	for _, r := range rows {
		out = append(out, TemplateLink{TemplateID: r.ID, ProcessName: r.ProcessName, Version: int(r.Version), Mode: r.Mode, Reason: deref(r.Reason)})
	}
	return out, nil
}

// SetTemplateMode mengubah mode tautan dokumen ke satu process template (E-2).
// Pinned dan detached wajib alasan. Mode boleh diubah juga pada dokumen yang
// sudah terbit, karena mode bukan isi dokumen.
func (s *Service) SetTemplateMode(ctx context.Context, actor access.Actor, id, templateID, mode, reason string) ([]TemplateLink, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return nil, err
	}
	if mode != ModeFollow && mode != ModePinned && mode != ModeDetached {
		return nil, problems(invalid("mode", problem.InvalidFormat, "Choose follow, pinned, or detached."))
	}
	reason, p := problem.Text("reason", reason, mode != ModeFollow, maxDetachReason)
	if err := problems(p); err != nil {
		return nil, err
	}
	links, err := s.q.ListDocumentTemplates(ctx, id)
	if err != nil {
		return nil, err
	}
	found := false
	for _, l := range links {
		found = found || l.ID == templateID
	}
	if !found {
		return nil, problem.ErrNotFound
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.SetDocumentTemplateMode(ctx, db.SetDocumentTemplateModeParams{
			DocumentID: id, ProcessTemplateID: templateID, Mode: mode, Reason: optional(reason), UpdatedByPersonID: actor.PersonID,
		}); err != nil {
			return err
		}
		details := map[string]string{"template": templateID, "mode": mode}
		if reason != "" {
			details["reason"] = reason
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentTemplateModeChanged, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return nil, result(err, "mengubah mode template")
	}
	return s.Templates(ctx, actor, id)
}
