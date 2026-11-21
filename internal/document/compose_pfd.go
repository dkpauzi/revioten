package document

// Perakitan PFMEA dan CP per langkah PFD (C4, D-211).
//
// Alurnya mengikuti pemilik: PFD diisi lebih dulu, per dokumen, tanpa
// template. Setiap langkah PFD lalu dicocokkan dengan process template
// prosesnya (versi terbit terakhir, sesudah varian customer dan model
// diterapkan). Penyusun memilih langkah, dan baris PFMEA dan CP dibuat dengan
// nomor langkah dan proses dari PFD, berurutan mengikuti PFD. Baris yang
// sudah ada tidak diubah isinya.
//
// Langkah yang prosesnya tidak punya template, atau templatenya tidak punya
// poin untuk jenis dokumen itu, mendapat satu baris berisi "--" di setiap
// kolom teks, seperti CP Tokairika (E-136): setiap langkah PFD muncul di CP.

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

// Keadaan satu langkah PFD di satu dokumen PFMEA atau CP.
const (
	StepPresent    = "present"     // dokumen sudah punya baris untuk langkah ini
	StepToAdd      = "to_add"      // akan mendapat baris dari poin template
	StepNoTemplate = "no_template" // akan mendapat satu baris "--"
	StepNotDraft   = "not_draft"   // dokumen tidak punya draft terbuka, jadi tidak diubah
)

// noControl adalah isi kolom teks baris tanpa poin template (E-136).
const noControl = "--"

// PFDComposition adalah tampilan perakitan dari PFD untuk satu set.
type PFDComposition struct {
	PFDID     string           // "" bila set tanpa PFD
	Documents []TypeRef        // PFMEA dan CP di set, dengan ID dokumennya di TypeRef.ID
	Steps     []PFDStepPlan    // menurut urutan PFD; tidak pernah nil
	Extra     []StepOutsidePFD // baris PFMEA atau CP yang langkahnya tidak ada di PFD; tidak pernah nil
}

// PFDStepPlan adalah satu langkah PFD beserta template dan keadaannya.
type PFDStepPlan struct {
	FlowStep
	TemplateVersionID string // "" bila proses tanpa template terbit
	TemplateVersion   int
	Targets           []StepTarget // satu per dokumen PFMEA atau CP; tidak pernah nil
}

// StepTarget adalah keadaan satu langkah di satu dokumen.
type StepTarget struct {
	DocumentID   string
	DocumentType string // pfmea atau cp
	Points       int    // jumlah poin template yang berlaku untuk dokumen ini
	State        string
}

// StepOutsidePFD adalah nomor langkah di PFMEA atau CP yang tidak ada di PFD.
type StepOutsidePFD struct {
	DocumentID   string
	DocumentType string
	SectionKey   string
	Operation    string
	ProcessName  string
}

// StepChoice memilih satu langkah PFD untuk dirakit.
type StepChoice struct {
	SectionKey string
	Operation  string
	ProcessID  string
}

func (c StepChoice) key() string { return c.SectionKey + "\x00" + c.Operation + "\x00" + c.ProcessID }

// pfdPlan adalah bahan perakitan dari PFD yang dibaca di dalam satu transaksi.
type pfdPlan struct {
	steps    []FlowStep
	targets  []Document                          // PFMEA dan CP satu set
	versions map[string]template.ProcessVersion  // ID proses menjadi versi terbit terakhir
	resolved map[string][]template.ResolvedPoint // ID proses menjadi poin yang berlaku untuk model
}

// buildPFDPlan membaca PFD, dokumen tujuan, dan template setiap proses.
func buildPFDPlan(ctx context.Context, q *db.Queries, d Document) (pfdPlan, error) {
	p := pfdPlan{versions: map[string]template.ProcessVersion{}, resolved: map[string][]template.ResolvedPoint{}}
	var err error
	if p.steps, err = loadFlow(ctx, q, d); err != nil {
		return p, err
	}
	for _, m := range d.Set {
		if m.DocumentType.Key == "pfd" {
			continue
		}
		doc := d
		if m.ID != d.ID {
			if doc, err = getDocument(ctx, q, m.ID); err != nil {
				return p, err
			}
		}
		p.targets = append(p.targets, doc)
	}
	latest, err := q.LatestPublishedProcessVersions(ctx)
	if err != nil {
		return p, err
	}
	byProcess := map[string]string{}
	for _, v := range latest {
		byProcess[v.ProcessID] = v.ID
	}
	model, err := q.GetModel(ctx, d.Models[0].ID)
	if err != nil {
		return p, err
	}
	for _, s := range p.steps {
		versionID, ok := byProcess[s.ProcessID]
		if s.ProcessID == "" || !ok {
			continue
		}
		if _, done := p.versions[s.ProcessID]; done {
			continue
		}
		v, err := template.LoadProcessVersion(ctx, q, versionID)
		if err != nil {
			return p, err
		}
		p.versions[s.ProcessID] = v
		// Poin yang berlaku untuk model pertama dokumen (ASUMSI E-93, sama dengan M4.5).
		var points []template.ResolvedPoint
		for _, sub := range template.ResolveFor(v, model.ID, model.CustomerID, deref(model.ModelGroupID)) {
			points = append(points, sub.Points...)
		}
		p.resolved[s.ProcessID] = points
	}
	return p, nil
}

// pointsFor memberi poin satu proses untuk satu jenis dokumen.
func (p pfdPlan) pointsFor(processID, typeKey string) []template.ResolvedPoint {
	var out []template.ResolvedPoint
	for _, pt := range p.resolved[processID] {
		if pt.DocumentType.Key == typeKey {
			out = append(out, pt)
		}
	}
	return out
}

// sectionKeys memberi kunci bagian setiap ID bagian dokumen.
func sectionKeys(d Document) map[string]string {
	out := map[string]string{}
	for _, s := range d.Sections {
		out[s.ID] = s.Key
	}
	return out
}

// hasStep melaporkan apakah dokumen sudah punya baris untuk langkah itu:
// nomor langkah dan proses yang sama, di bagian yang sama.
func hasStep(d Document, s FlowStep) bool {
	keys := sectionKeys(d)
	return slices.ContainsFunc(d.Rows, func(r Row) bool {
		return keys[r.SectionID] == s.SectionKey && r.OperationNumber == s.Operation && r.ProcessID == s.ProcessID
	})
}

// view menyusun tampilan perakitan dari bahan yang sudah dibaca.
func (p pfdPlan) view(d Document) PFDComposition {
	out := PFDComposition{PFDID: pfdOf(d), Documents: []TypeRef{}, Steps: []PFDStepPlan{}, Extra: []StepOutsidePFD{}}
	for _, t := range p.targets {
		out.Documents = append(out.Documents, TypeRef{ID: t.ID, Key: t.DocumentType.Key, Name: t.DocumentType.Name})
	}
	known := map[string]bool{}
	for _, s := range p.steps {
		known[s.SectionKey+"\x00"+s.Operation] = true
		item := PFDStepPlan{FlowStep: s, Targets: []StepTarget{}}
		if v, ok := p.versions[s.ProcessID]; ok {
			item.TemplateVersionID, item.TemplateVersion = v.ID, v.Version
		}
		for _, t := range p.targets {
			target := StepTarget{DocumentID: t.ID, DocumentType: t.DocumentType.Key, Points: len(p.pointsFor(s.ProcessID, t.DocumentType.Key))}
			switch {
			case hasStep(t, s):
				target.State = StepPresent
			case t.Status != StatusDraft || t.Signing != "":
				target.State = StepNotDraft
			case target.Points > 0:
				target.State = StepToAdd
			default:
				target.State = StepNoTemplate
			}
			item.Targets = append(item.Targets, target)
		}
		out.Steps = append(out.Steps, item)
	}
	// Pemeriksaan (C4): nomor langkah PFMEA atau CP yang tidak ada di PFD.
	for _, t := range p.targets {
		keys := sectionKeys(t)
		seen := map[string]bool{}
		for _, r := range t.Rows {
			k := keys[r.SectionID] + "\x00" + r.OperationNumber
			if r.OperationNumber == "" || known[k] || seen[k] {
				continue
			}
			seen[k] = true
			out.Extra = append(out.Extra, StepOutsidePFD{DocumentID: t.ID, DocumentType: t.DocumentType.Key, SectionKey: keys[r.SectionID], Operation: r.OperationNumber, ProcessName: r.ProcessName})
		}
	}
	return out
}

// PFDComposition menampilkan langkah PFD satu set beserta template dan
// keadaannya di PFMEA dan CP (L-4).
func (s *Service) PFDComposition(ctx context.Context, actor access.Actor, id string) (PFDComposition, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return PFDComposition{}, err
	}
	p, err := buildPFDPlan(ctx, s.q, d)
	if err != nil {
		return PFDComposition{}, result(err, "membaca langkah PFD")
	}
	return p.view(d), nil
}

// ComposeFromPFD membuat baris PFMEA dan CP untuk langkah PFD yang dipilih
// (E-2), dalam satu transaksi untuk seluruh set. Dokumen yang sudah punya
// baris untuk satu langkah tidak diubah untuk langkah itu.
func (s *Service) ComposeFromPFD(ctx context.Context, actor access.Actor, id string, choices []StepChoice) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		for _, docID := range setIDs(d) {
			if err := q.LockDocument(ctx, docID); err != nil {
				return err
			}
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if d.DocumentType.Key == "pfd" {
			return problems(invalid("document", problem.NotAllowed, "The PFD is filled in directly. Compose from the PFD on the PFMEA or the Control Plan."))
		}
		if err := requireDraft(d); err != nil {
			return err
		}
		if len(choices) == 0 {
			return problems(invalid("steps", problem.Required, "Choose at least one PFD step."))
		}
		p, err := buildPFDPlan(ctx, q, d)
		if err != nil {
			return err
		}
		if len(p.steps) == 0 {
			return problems(invalid("steps", problem.NotAllowed, "The PFD of this set has no steps yet. Fill in the PFD first."))
		}
		chosen := map[string]bool{}
		for _, c := range choices {
			chosen[c.key()] = true
		}
		matched := 0
		for _, st := range p.steps {
			if chosen[st.key()] {
				matched++
			}
		}
		if matched != len(chosen) {
			return problems(invalid("steps", problem.InvalidFormat, "Choose steps of the PFD of this set."))
		}

		added, empty := 0, 0
		for _, t := range p.targets {
			if t.Status != StatusDraft || t.Signing != "" {
				continue // dokumen tanpa draft terbuka tidak diubah
			}
			format, err := template.LoadFormatVersion(ctx, q, t.FormatVersionID)
			if err != nil {
				return err
			}
			sectionOf := map[string]string{} // kunci bagian menjadi ID bagian dokumen ini
			for _, sec := range t.Sections {
				sectionOf[sec.Key] = sec.ID
			}
			// Poin yang sudah tertaut di setiap bagian: satu poin hanya boleh
			// tertaut ke satu baris per bagian (indeks unik migrasi 00029).
			linked := map[string]bool{}
			for _, r := range t.Rows {
				if r.TemplatePointID != "" {
					linked[r.SectionID+"\x00"+r.TemplatePointID] = true
				}
			}
			count := len(t.Rows)
			touched := false
			for _, st := range p.steps {
				if !chosen[st.key()] || hasStep(t, st) {
					continue
				}
				section, ok := sectionOf[st.SectionKey]
				if st.SectionKey != "" && !ok {
					continue // bagian ini tidak ada di dokumen tujuan
				}
				op := st.Operation
				add := func(content map[string]any, pointID, versionID string) error {
					if count >= maxRows {
						return problems(invalid("steps", problem.NotAllowed, "A document has at most 1000 rows."))
					}
					data, err := json.Marshal(content)
					if err != nil {
						return err
					}
					if _, err := q.AddStepRow(ctx, db.AddStepRowParams{
						DocumentID: t.ID, OperationNumber: &op, Content: data, ProcessID: optional(st.ProcessID),
						TemplatePointID: optional(pointID), TemplateVersionID: optional(versionID), SectionID: optional(section),
					}); err != nil {
						return err
					}
					count++
					touched = true
					return nil
				}
				points := p.pointsFor(st.ProcessID, t.DocumentType.Key)
				if len(points) == 0 {
					if err := add(placeholderContent(format), "", ""); err != nil {
						return err
					}
					empty++
					continue
				}
				v := p.versions[st.ProcessID]
				for _, pt := range points {
					pointID, versionID := pt.OriginID, v.ID
					// Proses yang sama di dua langkah: baris kedua tidak tertaut
					// ke poin, karena poin sudah tertaut di langkah pertama
					// (ASUMSI E-138).
					if linked[section+"\x00"+pt.OriginID] {
						pointID, versionID = "", ""
					} else {
						linked[section+"\x00"+pt.OriginID] = true
					}
					if err := add(rowContent(format, pt.Content), pointID, versionID); err != nil {
						return err
					}
					added++
				}
			}
			if !touched {
				continue
			}
			if err := orderByPFD(ctx, q, t.ID, p.steps); err != nil {
				return err
			}
			current, err := getDocument(ctx, q, t.ID)
			if err != nil {
				return err
			}
			// Kolom Relevant Document baris pertama setiap proses (D-130, D-156).
			if _, err := fillReferences(ctx, q, current, format); err != nil {
				return err
			}
			if err := q.TouchDocument(ctx, t.ID); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentComposed, TargetType: audit.TargetDocument, TargetID: id,
			Details: map[string]string{"source": "pfd", "steps": strconv.Itoa(len(chosen)), "added": strconv.Itoa(added), "without_template": strconv.Itoa(empty)},
		})
	})
	if err != nil {
		return Document{}, result(err, "merakit dari PFD")
	}
	return getDocument(ctx, s.q, id)
}

// placeholderContent mengisi setiap kolom teks baris dengan "--" (E-136).
func placeholderContent(format template.FormatVersion) map[string]any {
	content := map[string]any{}
	for _, f := range format.Fields {
		if f.Area == template.AreaRow && !f.Hidden && f.Kind == template.KindText {
			content[f.ID] = noControl
		}
	}
	return content
}

// orderByPFD mengurutkan baris setiap bagian menurut urutan langkah PFD.
// Urutannya stabil: baris satu langkah tetap dalam urutan semula, baris baru
// di belakang baris lama langkah itu, dan baris yang langkahnya tidak ada di
// PFD tetap menempel pada baris sebelumnya (atau ke akhir bila tidak ada).
func orderByPFD(ctx context.Context, q *db.Queries, documentID string, steps []FlowStep) error {
	d, err := getDocument(ctx, q, documentID)
	if err != nil {
		return err
	}
	keys := sectionKeys(d)
	rank := map[string]int{}
	for i, s := range steps {
		k := s.SectionKey + "\x00" + s.Operation
		if _, ok := rank[k]; !ok {
			rank[k] = i
		}
		rank[s.key()] = i
	}
	groups := map[string][]Row{} // ID bagian menjadi barisnya
	for _, r := range d.Rows {
		groups[r.SectionID] = append(groups[r.SectionID], r)
	}
	for _, rows := range groups {
		ranks := make([]int, len(rows))
		// Baris di awal yang langkahnya tidak ada di PFD pindah ke akhir.
		last := len(steps)
		for i, r := range rows {
			step := FlowStep{SectionKey: keys[r.SectionID], Operation: r.OperationNumber, ProcessID: r.ProcessID}
			if n, ok := rank[step.key()]; ok {
				last = n
			} else if n, ok := rank[step.SectionKey+"\x00"+step.Operation]; ok && r.OperationNumber != "" {
				last = n
			}
			ranks[i] = last
		}
		positions := make([]int, len(rows))
		index := make([]int, len(rows))
		for i, r := range rows {
			positions[i] = r.Position
			index[i] = i
		}
		slices.SortStableFunc(index, func(a, b int) int { return ranks[a] - ranks[b] })
		for i, from := range index {
			r := rows[from]
			if r.Position == positions[i] {
				continue
			}
			if err := q.SetRowPosition(ctx, db.SetRowPositionParams{ID: r.ID, DocumentID: documentID, Position: int32(positions[i])}); err != nil {
				return err
			}
		}
	}
	return nil
}
