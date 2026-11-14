package change

// Yokoten (M5.9, R-CHG-04, D-40, D-118). Satu perbaikan yang sudah diterima
// disebarkan ke model lain. Target dipilih lewat customer, group model, atau
// model, lalu disaring proses dan teks di baris dokumen. Setiap target adalah
// satu model beserta dokumennya. Pemilik target memutuskan apply, adapt, atau
// not applicable (dengan alasan). Apply dan adapt membuat perubahan
// bermekanisme yokoten untuk dokumen model itu, yang diverifikasi seperti
// perubahan lain (M5.3) dan membuka draft saat diterima (M5.4). Papan status
// menunjukkan kemajuan per model.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Keputusan pemilik target.
const (
	DecisionPending       = "pending"
	DecisionApply         = "apply"
	DecisionAdapt         = "adapt"
	DecisionNotApplicable = "not_applicable"
)

// Kemajuan satu target di papan status.
const (
	ProgressUndecided     = "undecided"
	ProgressNotApplicable = "not_applicable"
	ProgressVerification  = "waiting_verification"
	ProgressRejected      = "rejected"
	ProgressInDraft       = "in_draft"
	ProgressReleased      = "released"
)

const maxYokotenText = 200

// TargetFilter adalah pilihan target yokoten. Satu dari CustomerID,
// ModelGroupID, atau ModelIDs wajib; ProcessID dan Text menyaring dokumen.
type TargetFilter struct {
	CustomerID   string
	ModelGroupID string
	ModelIDs     []string
	ProcessID    string
	Text         string
}

// NewYokoten adalah isian yokoten baru.
type NewYokoten struct {
	SourceChangeID string
	Reason         string
	Targets        TargetFilter
	DryRun         bool // hanya cari target, jangan simpan
}

// Yokoten adalah satu penyebaran beserta papan statusnya.
type Yokoten struct {
	ID            string
	Ref           string // YKT-0001 (ASUMSI E-106)
	SourceChange  string // nomor rujukan perubahan sumber
	SourceID      string
	Reason        string
	ProcessFilter string // nama proses saringan; "" tanpa saringan
	TextFilter    string
	CreatedBy     string
	CreatedAt     time.Time
	Targets       []YokotenTarget // menurut customer lalu model; tidak pernah nil
}

// YokotenTarget adalah satu model target.
type YokotenTarget struct {
	ID           string
	ModelID      string
	ModelName    string
	CustomerID   string
	CustomerCode string
	Decision     string
	Reason       string
	DecidedBy    string
	DecidedAt    *time.Time
	ChangeID     string
	ChangeRef    string
	Progress     string
	Documents    []TargetDocument
}

// TargetDocument adalah satu dokumen target.
type TargetDocument struct {
	ID               string
	Number           string
	DocumentType     string
	Revision         int
	Status           string
	ReleasedRevision *int
}

// candidate adalah satu model target hasil pencarian beserta dokumennya.
type candidate struct {
	target YokotenTarget
	docs   []string
}

// CreateYokoten mencari target dan, bila bukan dry run, menyimpannya (E-3).
// Target hanya dari customer yang boleh diajukan perubahannya oleh pembuat
// (D-118), dan model yang dokumennya sudah di perubahan sumber tidak ikut.
func (s *RequestService) CreateYokoten(ctx context.Context, actor access.Actor, in NewYokoten) (Yokoten, error) {
	source, err := s.Get(ctx, actor, in.SourceChangeID)
	if err != nil {
		return Yokoten{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: source.CustomerID}); err != nil {
		return Yokoten{}, err
	}
	if source.Status != StatusAccepted {
		return Yokoten{}, problems(invalid("source_change_id", problem.NotAllowed, "Spread only a change that is accepted."))
	}
	reason, p := problem.Text("reason", in.Reason, true, maxReason)
	if err := problems(p); err != nil {
		return Yokoten{}, err
	}
	f := in.Targets
	f.Text = strings.TrimSpace(f.Text)
	if utf8.RuneCountInString(f.Text) > maxYokotenText {
		return Yokoten{}, problems(invalid("text", problem.TooLong, "Search text can be at most 200 characters."))
	}
	found, err := s.findTargets(ctx, actor, source, f)
	if err != nil {
		return Yokoten{}, err
	}
	out := Yokoten{SourceChange: source.Ref, SourceID: source.ID, Reason: reason, TextFilter: f.Text, Targets: []YokotenTarget{}}
	for _, c := range found {
		out.Targets = append(out.Targets, c.target)
	}
	if in.DryRun {
		return out, nil
	}
	if len(found) == 0 {
		return Yokoten{}, problems(invalid("targets", problem.Required, "No model matches the targets."))
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if id, err = q.CreateYokoten(ctx, db.CreateYokotenParams{
			SourceChangeID: source.ID, Reason: reason, FilterProcessID: optional(f.ProcessID), FilterText: optional(f.Text),
			CreatedByPersonID: actor.PersonID, CreatedByAccountID: actor.AccountID,
		}); err != nil {
			return err
		}
		models := make([]string, 0, len(found))
		for _, c := range found {
			target, err := q.AddYokotenTarget(ctx, db.AddYokotenTargetParams{YokotenID: id, ModelID: c.target.ModelID, CustomerID: c.target.CustomerID})
			if err != nil {
				return err
			}
			for _, doc := range c.docs {
				if err := q.AddYokotenTargetDocument(ctx, db.AddYokotenTargetDocumentParams{TargetID: target, DocumentID: doc}); err != nil {
					return err
				}
			}
			models = append(models, c.target.CustomerCode+" "+c.target.ModelName)
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionYokotenCreated, TargetType: audit.TargetChange, TargetID: source.ID,
			Details: map[string]string{"yokoten": id, "models": strings.Join(models, ", ")}})
	})
	if err != nil {
		return Yokoten{}, result(err, "membuat yokoten")
	}
	return s.GetYokoten(ctx, actor, id)
}

// findTargets mengubah pilihan target menjadi model beserta dokumennya.
func (s *RequestService) findTargets(ctx context.Context, actor access.Actor, source Change, f TargetFilter) ([]candidate, error) {
	var models []string
	var err error
	switch {
	case len(f.ModelIDs) > 0:
		for _, m := range f.ModelIDs {
			if !db.IsUUID(m) {
				return nil, problems(invalid("model_ids", problem.InvalidFormat, "Choose models from the list."))
			}
		}
		models = f.ModelIDs
	case f.ModelGroupID != "":
		if !db.IsUUID(f.ModelGroupID) {
			return nil, problems(invalid("model_group_id", problem.InvalidFormat, "Choose a model group."))
		}
		models, err = s.q.ListModelsOfGroup(ctx, &f.ModelGroupID)
	case f.CustomerID != "":
		if !db.IsUUID(f.CustomerID) {
			return nil, problems(invalid("customer_id", problem.InvalidFormat, "Choose a customer."))
		}
		models, err = s.q.ListModelsOfCustomer(ctx, f.CustomerID)
	default:
		return nil, problems(invalid("targets", problem.Required, "Choose a customer, a model group, or models."))
	}
	if err != nil {
		return nil, fmt.Errorf("membaca model target: %w", err)
	}
	if f.ProcessID != "" && !db.IsUUID(f.ProcessID) {
		return nil, problems(invalid("process_id", problem.InvalidFormat, "Choose a process."))
	}
	rows, err := s.q.ListYokotenCandidates(ctx, db.ListYokotenCandidatesParams{ModelIds: models, ProcessID: optional(f.ProcessID), Text: optional(escapeLike(f.Text))})
	if err != nil {
		return nil, fmt.Errorf("mencari dokumen target: %w", err)
	}
	// Dokumen perubahan sumber sudah berubah; model dokumen itu tidak ikut.
	inSource := map[string]bool{}
	for _, d := range source.Documents {
		inSource[d.ID] = true
	}
	skipModel := map[string]bool{}
	for _, r := range rows {
		if inSource[r.DocumentID] {
			skipModel[r.ModelID] = true
		}
	}
	allowed := map[string]bool{}
	var out []candidate
	index := map[string]int{}
	for _, r := range rows {
		if skipModel[r.ModelID] {
			continue
		}
		ok, seen := allowed[r.CustomerID]
		if !seen {
			// Hanya customer yang boleh diajukan perubahannya oleh pembuat (D-118).
			err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: r.CustomerID})
			switch {
			case errors.Is(err, access.ErrForbidden):
				ok = false
			case err != nil:
				return nil, err
			default:
				ok = true
			}
			allowed[r.CustomerID] = ok
		}
		if !ok {
			continue
		}
		i, exists := index[r.ModelID]
		if !exists {
			i = len(out)
			index[r.ModelID] = i
			out = append(out, candidate{target: YokotenTarget{
				ModelID: r.ModelID, ModelName: r.ModelName, CustomerID: r.CustomerID, CustomerCode: r.CustomerCode,
				Decision: DecisionPending, Progress: ProgressUndecided, Documents: []TargetDocument{},
			}})
		}
		out[i].docs = append(out[i].docs, r.DocumentID)
		out[i].target.Documents = append(out[i].target.Documents, TargetDocument{ID: r.DocumentID, Number: r.Number, DocumentType: deref(r.DocumentTypeKey)})
	}
	return out, nil
}

// escapeLike menjaga % dan _ di teks pencarian supaya dicari apa adanya.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Decide mencatat keputusan pemilik satu target (E-3 untuk customer target).
// Apply dan adapt membuat perubahan bermekanisme yokoten untuk dokumen target;
// not applicable wajib alasan. Keputusan tidak bisa diubah (ASUMSI E-106).
func (s *RequestService) Decide(ctx context.Context, actor access.Actor, yokotenID, targetID, decision, reason string) (Yokoten, error) {
	y, err := s.GetYokoten(ctx, actor, yokotenID)
	if err != nil {
		return Yokoten{}, err
	}
	var target *YokotenTarget
	for i := range y.Targets {
		if y.Targets[i].ID == targetID {
			target = &y.Targets[i]
		}
	}
	if target == nil {
		return Yokoten{}, problem.ErrNotFound
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: target.CustomerID}); err != nil {
		return Yokoten{}, err
	}
	reason = strings.TrimSpace(reason)
	switch {
	case decision != DecisionApply && decision != DecisionAdapt && decision != DecisionNotApplicable:
		return Yokoten{}, problems(invalid("decision", problem.InvalidFormat, "Choose apply, adapt, or not applicable."))
	case decision == DecisionNotApplicable && reason == "":
		return Yokoten{}, problems(invalid("reason", problem.Required, "Write why the improvement does not apply to this model."))
	case utf8.RuneCountInString(reason) > maxImpactReason:
		return Yokoten{}, problems(invalid("reason", problem.TooLong, "A reason can be at most 500 characters."))
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.GetYokotenTarget(ctx, targetID)
		if err != nil {
			return err
		}
		if row.Decision != DecisionPending {
			return problems(invalid("decision", problem.NotAllowed, "This target is already decided."))
		}
		var changeID *string
		if decision != DecisionNotApplicable {
			text := y.Reason
			if reason != "" {
				text = y.Reason + "\n" + reason
			}
			ref := y.Ref + " · " + y.SourceChange
			id, err := q.CreateYokotenChange(ctx, db.CreateYokotenChangeParams{
				CustomerID: target.CustomerID, Reason: truncate(text, maxReason), SourceReference: &ref,
				PersonID: actor.PersonID, AccountID: actor.AccountID, SourceChangeID: y.SourceID,
			})
			if err != nil {
				return err
			}
			for _, d := range target.Documents {
				if err := q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: id, DocumentID: d.ID}); err != nil {
					return err
				}
			}
			if err := audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionChangeCreated, TargetType: audit.TargetChange, TargetID: id,
				Details: map[string]string{"mechanism": MechanismYokoten, "yokoten": y.Ref, "source": y.SourceChange, "model": target.ModelName}}); err != nil {
				return err
			}
			changeID = &id
		}
		n, err := q.DecideYokotenTarget(ctx, db.DecideYokotenTargetParams{Decision: decision, Reason: optional(reason), PersonID: &actor.PersonID, ChangeID: changeID, ID: targetID})
		if err != nil {
			return err
		}
		if n == 0 {
			return problems(invalid("decision", problem.NotAllowed, "This target is already decided."))
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionYokotenDecided, TargetType: audit.TargetChange, TargetID: y.SourceID,
			Details: map[string]string{"yokoten": y.Ref, "model": target.ModelName, "decision": decision}})
	})
	if err != nil {
		return Yokoten{}, result(err, "memutuskan target yokoten")
	}
	return s.GetYokoten(ctx, actor, yokotenID)
}

func truncate(s string, n int) string {
	for utf8.RuneCountInString(s) > n {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return strings.TrimSpace(s)
}

// GetYokoten membaca satu yokoten beserta papan statusnya (L-4 untuk
// customer perubahan sumber). Target customer lain yang tidak boleh dilihat
// pembaca tidak tampil (D-118).
func (s *RequestService) GetYokoten(ctx context.Context, actor access.Actor, id string) (Yokoten, error) {
	if !db.IsUUID(id) {
		return Yokoten{}, problem.ErrNotFound
	}
	r, err := s.q.GetYokoten(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Yokoten{}, problem.ErrNotFound
	}
	if err != nil {
		return Yokoten{}, fmt.Errorf("membaca yokoten: %w", err)
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: r.SourceCustomerID}); err != nil {
		return Yokoten{}, err
	}
	y := Yokoten{
		ID: r.ID, Ref: fmt.Sprintf("YKT-%04d", deref64(r.Seq)), SourceChange: changeRef(r.SourceSeq), SourceID: r.SourceChangeID, Reason: r.Reason,
		ProcessFilter: deref(r.FilterProcessName), TextFilter: deref(r.FilterText), CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt.UTC(), Targets: []YokotenTarget{},
	}
	targets, err := s.q.ListYokotenTargets(ctx, id)
	if err != nil {
		return Yokoten{}, fmt.Errorf("membaca target yokoten: %w", err)
	}
	for _, t := range targets {
		if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: t.CustomerID}); errors.Is(err, access.ErrForbidden) {
			continue
		} else if err != nil {
			return Yokoten{}, err
		}
		item := YokotenTarget{
			ID: t.ID, ModelID: t.ModelID, ModelName: t.ModelName, CustomerID: t.CustomerID, CustomerCode: t.CustomerCode,
			Decision: t.Decision, Reason: deref(t.Reason), DecidedBy: deref(t.DecidedBy), ChangeID: deref(t.ChangeID), Documents: []TargetDocument{},
		}
		if t.DecidedAt != nil {
			at := t.DecidedAt.UTC()
			item.DecidedAt = &at
		}
		if t.ChangeSeq != nil {
			item.ChangeRef = changeRef(t.ChangeSeq)
		}
		docs, err := s.q.ListYokotenTargetDocuments(ctx, t.ID)
		if err != nil {
			return Yokoten{}, err
		}
		for _, d := range docs {
			item.Documents = append(item.Documents, TargetDocument{
				ID: d.ID, Number: d.Number, DocumentType: deref(d.DocumentTypeKey), Revision: int(d.Revision), Status: d.Status, ReleasedRevision: intPtr(d.ReleasedRevision),
			})
		}
		item.Progress = progress(item, deref(t.ChangeStatus))
		y.Targets = append(y.Targets, item)
	}
	return y, nil
}

// progress menghitung kemajuan satu target untuk papan status.
func progress(t YokotenTarget, changeStatus string) string {
	switch {
	case t.Decision == DecisionPending:
		return ProgressUndecided
	case t.Decision == DecisionNotApplicable:
		return ProgressNotApplicable
	case changeStatus == StatusSubmitted:
		return ProgressVerification
	case changeStatus == StatusRejected:
		return ProgressRejected
	}
	for _, d := range t.Documents {
		if d.Status != "released" {
			return ProgressInDraft
		}
	}
	return ProgressReleased
}

// ListYokotens memberi yokoten yang perubahan sumbernya boleh dilihat, yang
// terbaru dulu (L-4).
func (s *RequestService) ListYokotens(ctx context.Context, actor access.Actor) ([]Yokoten, error) {
	ids, err := s.q.ListYokotenIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := []Yokoten{}
	for _, id := range ids {
		y, err := s.GetYokoten(ctx, actor, id)
		if errors.Is(err, access.ErrForbidden) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, y)
	}
	return out, nil
}

func deref64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
