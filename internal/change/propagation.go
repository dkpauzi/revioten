package change

// Propagasi template (M5.8, R-TPL-06 sampai R-TPL-09). Pengelola template
// (E-6) memilih satu versi process template yang sudah terbit, melihat
// pratinjau dampaknya, lalu menjalankannya. Hasilnya satu perubahan
// bermekanisme template per customer, yang menunjuk dokumen terdampak.
// Perubahan itu diverifikasi seperti perubahan lain (M5.3); saat diterima,
// setiap dokumen mendapat draft (M5.4) dan isi template baru ditulis ke
// draft itu. Revisi terbit tidak pernah ditulis ulang (R-TPL-07).

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Mekanisme dan cakupan propagasi (R-TPL-08). Cakupan "dokumen baru saja"
// tidak membuat perubahan apa pun, jadi cukup dengan tidak menjalankan propagasi.
const (
	ScopeFollowing    = "following" // dokumen yang mengikuti (bawaan)
	ScopeAll          = "all"       // termasuk yang dikunci versinya; wajib rujukan SQAM dan tanggal berlaku
	ConflictKeep      = "keep"      // baris yang diubah lokal dibiarkan dan ditandai
	ConflictOverwrite = "overwrite" // baris yang diubah lokal ditimpa isi template
)

// Propagation adalah isian satu propagasi.
type Propagation struct {
	VersionID       string
	Scope           string
	ConflictPolicy  string
	ChangeTypeID    string
	Reason          string
	SourceReference string     // rujukan revisi SQAM; wajib untuk cakupan all
	EffectiveDate   *time.Time // wajib untuk cakupan all
}

// PreviewItem adalah satu dokumen di pratinjau propagasi.
type PreviewItem struct {
	document.TemplatePlan
	Included bool   // ikut propagasi dengan cakupan ini
	Skipped  string // alasan tidak ikut: pinned, detached, no_change, atau already_proposed
}

// Preview menghitung dampak satu versi template tanpa mengubah apa pun (E-6).
func (s *RequestService) Preview(ctx context.Context, actor access.Actor, versionID, scope string) ([]PreviewItem, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return nil, err
	}
	if scope != ScopeFollowing && scope != ScopeAll {
		return nil, problems(invalid("scope", problem.InvalidFormat, "Choose following or all."))
	}
	return preview(ctx, s.q, versionID, scope)
}

func preview(ctx context.Context, q *db.Queries, versionID, scope string) ([]PreviewItem, error) {
	if !db.IsUUID(versionID) {
		return nil, problem.ErrNotFound
	}
	v, err := template.LoadProcessVersion(ctx, q, versionID)
	if err != nil {
		return nil, err
	}
	if v.Status != template.StatusPublished {
		return nil, problems(invalid("version", problem.NotAllowed, "Publish the version before propagating it."))
	}
	docs, err := q.ListTemplateLinkedDocuments(ctx, db.ListTemplateLinkedDocumentsParams{TemplateID: v.TemplateID, VersionID: &versionID})
	if err != nil {
		return nil, fmt.Errorf("membaca dokumen tertaut: %w", err)
	}
	out := make([]PreviewItem, 0, len(docs))
	for _, d := range docs {
		plan, err := document.PlanTemplate(ctx, q, d.ID, versionID)
		if err != nil {
			return nil, err
		}
		item := PreviewItem{TemplatePlan: plan}
		taken, err := q.DocumentHasTemplateChange(ctx, db.DocumentHasTemplateChangeParams{DocumentID: d.ID, TemplateVersionID: &versionID})
		if err != nil {
			return nil, err
		}
		switch {
		case plan.Mode == document.ModeDetached:
			item.Skipped = "detached"
		case plan.Mode == document.ModePinned && scope != ScopeAll:
			item.Skipped = "pinned"
		case plan.Updates+plan.Conflicts+plan.Removed == 0:
			item.Skipped = "no_change"
		case taken:
			item.Skipped = "already_proposed"
		default:
			item.Included = true
		}
		out = append(out, item)
	}
	return out, nil
}

// Propagate menjalankan propagasi: satu perubahan bermekanisme template per
// customer yang punya dokumen ikut (E-6). Perubahan menunggu verifikasi.
func (s *RequestService) Propagate(ctx context.Context, actor access.Actor, in Propagation) ([]Change, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return nil, err
	}
	in.Reason, in.SourceReference = strings.TrimSpace(in.Reason), strings.TrimSpace(in.SourceReference)
	var list []*problem.Problem
	if in.Scope != ScopeFollowing && in.Scope != ScopeAll {
		list = append(list, invalid("scope", problem.InvalidFormat, "Choose following or all."))
	}
	if in.ConflictPolicy != ConflictKeep && in.ConflictPolicy != ConflictOverwrite {
		list = append(list, invalid("conflict_policy", problem.InvalidFormat, "Choose keep or overwrite."))
	}
	reason, p := problem.Text("reason", in.Reason, true, maxReason)
	list = append(list, p)
	if in.Scope == ScopeAll {
		if in.SourceReference == "" {
			list = append(list, invalid("source_reference", problem.Required, "Write the SQAM revision that asks to update every active document."))
		}
		if in.EffectiveDate == nil {
			list = append(list, invalid("effective_date", problem.Required, "Choose the effective date."))
		}
	}
	ok := false
	if db.IsUUID(in.ChangeTypeID) {
		var err error
		if ok, err = s.q.ChangeTypeExists(ctx, in.ChangeTypeID); err != nil {
			return nil, err
		}
	}
	if !ok {
		list = append(list, invalid("change_type_id", problem.Required, "Choose the change type."))
	}
	if err := problems(list...); err != nil {
		return nil, err
	}
	var ids []string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		items, err := preview(ctx, q, in.VersionID, in.Scope)
		if err != nil {
			return err
		}
		byCustomer := map[string][]document.TemplatePlan{}
		var customers []string
		for _, it := range items {
			if !it.Included {
				continue
			}
			if _, seen := byCustomer[it.CustomerID]; !seen {
				customers = append(customers, it.CustomerID)
			}
			byCustomer[it.CustomerID] = append(byCustomer[it.CustomerID], it.TemplatePlan)
		}
		if len(customers) == 0 {
			return problems(invalid("version", problem.NotAllowed, "No document takes this version: nothing to propagate."))
		}
		date := pgtype.Date{}
		if in.EffectiveDate != nil {
			date = pgtype.Date{Time: *in.EffectiveDate, Valid: true}
		}
		scope, policy := in.Scope, in.ConflictPolicy
		for _, customerID := range customers {
			id, err := q.CreateTemplateChange(ctx, db.CreateTemplateChangeParams{
				CustomerID: customerID, ChangeTypeID: in.ChangeTypeID, Reason: reason, SourceReference: optional(in.SourceReference),
				RequestedByPersonID: actor.PersonID, RequestedByAccountID: actor.AccountID,
				TemplateVersionID: &in.VersionID, PropagationScope: &scope, ConflictPolicy: &policy, EffectiveDate: date,
			})
			if err != nil {
				return err
			}
			numbers := make([]string, 0, len(byCustomer[customerID]))
			for _, plan := range byCustomer[customerID] {
				if err := q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: id, DocumentID: plan.DocumentID}); err != nil {
					return err
				}
				numbers = append(numbers, plan.Number)
			}
			details := map[string]string{"mechanism": MechanismTemplate, "template_version": in.VersionID, "scope": scope, "documents": strings.Join(numbers, ", ")}
			if err := audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionChangeCreated, TargetType: audit.TargetChange, TargetID: id, Details: details}); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, result(err, "menjalankan propagasi")
	}
	return s.many(ctx, ids)
}

// applyTemplate menulis isi versi template perubahan ke draft satu dokumen
// yang baru dibuka atau sudah terbuka (dipanggil openDrafts).
func applyTemplate(ctx context.Context, q *db.Queries, actor access.Actor, c Change, docID string) error {
	if c.Template == nil {
		return errors.New("perubahan template tanpa versi template")
	}
	updated, conflicts, removed, err := document.ApplyTemplate(ctx, q, docID, c.Template.VersionID, c.Template.ConflictPolicy == ConflictOverwrite)
	if err != nil {
		return err
	}
	u, k, r := int32(updated), int32(conflicts), int32(removed)
	if err := q.SetChangeDocumentResult(ctx, db.SetChangeDocumentResultParams{ChangeID: c.ID, DocumentID: docID, RowsUpdated: &u, RowsConflict: &k, RowsRemoved: &r}); err != nil {
		return err
	}
	details := map[string]string{"document": docID, "updated": strconv.Itoa(updated), "conflicts": strconv.Itoa(conflicts), "removed": strconv.Itoa(removed)}
	return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionChangeTemplateApplied, TargetType: audit.TargetChange, TargetID: c.ID, Details: details})
}

// TemplateInfo adalah data propagasi milik perubahan bermekanisme template.
type TemplateInfo struct {
	VersionID      string
	Scope          string
	ConflictPolicy string
	EffectiveDate  *time.Time
}
