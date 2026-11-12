package template

// Cakupan poin dan prioritas (M3.3, D-129, R-TPL-03, D-39). Satu process
// template per proses (keputusan E-72). Setiap poin punya cakupan: umum,
// customer, group model, atau model. Di layar cakupan ini tampil sebagai
// "biasanya dipakai untuk".
//
// Poin boleh punya varian untuk cakupan yang lebih sempit. Contoh pemilik:
// poin laser type di proses laser marking biasanya umum, tetapi laser type
// sebenarnya berbeda per model, jadi model K-125 mendapat varian sendiri.
// Untuk satu model, varian yang paling spesifik menang: model, lalu group
// model, lalu customer, lalu poin umumnya.

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Tingkat cakupan poin.
const (
	ScopeGeneral    = "general"
	ScopeCustomer   = "customer"
	ScopeModelGroup = "model_group"
	ScopeModel      = "model"
)

// scopeRank memberi nilai kespesifikan tiap tingkat: makin besar makin
// spesifik. Tingkat yang tidak dikenal bernilai 0.
func scopeRank(level string) int {
	switch level {
	case ScopeGeneral:
		return 1
	case ScopeCustomer:
		return 2
	case ScopeModelGroup:
		return 3
	case ScopeModel:
		return 4
	}
	return 0
}

// Scope adalah cakupan satu poin atau varian.
type Scope struct {
	Level    string // general, customer, model_group, atau model
	TargetID string // id customer, group model, atau model; kosong untuk umum
	Label    string // kode customer, nama group model, atau nama model; kosong untuk umum
}

// ScopeInput adalah isian cakupan dari pengguna. Level kosong berarti umum.
type ScopeInput struct {
	Level    string
	TargetID string
}

// Variant adalah isi satu poin untuk cakupan yang lebih sempit. ID-nya tetap
// lintas versi, sama seperti poin.
type Variant struct {
	ID      string
	Scope   Scope
	Text    string
	Hidden  bool
	Content map[string]any // isi per kolom (D-146); tidak pernah nil
}

// NewVariant adalah isian varian baru. Cakupannya tetap setelah dibuat.
type NewVariant struct {
	Scope   ScopeInput
	Text    string
	Content map[string]any // boleh nil (D-146)
}

// VariantUpdate adalah isian perubahan varian. Teks wajib dikirim; isian nil
// berarti tetap (D-111).
type VariantUpdate struct {
	Text    string
	Hidden  *bool
	Content *map[string]any // nil berarti isi tetap (D-146)
}

// modelScope adalah letak satu model: dirinya, customer-nya, dan group
// model-nya (boleh kosong, D-42).
type modelScope struct {
	ID         string
	CustomerID string
	GroupID    string
}

// ResolvedModel adalah model yang dipakai untuk pratinjau poin.
type ResolvedModel struct {
	ID             string
	Name           string
	CustomerID     string
	CustomerCode   string
	ModelGroupID   string // kosong bila model tanpa group
	ModelGroupName string
}

// ResolvedPoint adalah poin yang berlaku untuk satu model: poin asal itu
// sendiri atau variannya yang paling spesifik.
type ResolvedPoint struct {
	PointID      string // id poin atau varian yang terpilih
	OriginID     string // id poin asal; baris dokumen nanti tertaut ke sini (R-CMP-04)
	DocumentType DocumentTypeRef
	Text         string
	Scope        Scope
	Content      map[string]any // isi per kolom milik poin atau varian yang terpilih (D-146)
}

// ResolvedSubprocess adalah sub-proses beserta poin yang berlaku untuk satu model.
type ResolvedSubprocess struct {
	ID     string
	Name   string
	Points []ResolvedPoint // tidak pernah nil
}

// Resolved adalah hasil pratinjau satu versi untuk satu model.
type Resolved struct {
	Model        ResolvedModel
	Subprocesses []ResolvedSubprocess // tidak pernah nil
}

// appliesTo melaporkan apakah cakupan berlaku untuk model m. Model tanpa
// group tidak pernah cocok dengan cakupan group model.
func appliesTo(s Scope, m modelScope) bool {
	switch s.Level {
	case ScopeGeneral:
		return true
	case ScopeCustomer:
		return s.TargetID == m.CustomerID
	case ScopeModelGroup:
		return m.GroupID != "" && s.TargetID == m.GroupID
	case ScopeModel:
		return s.TargetID == m.ID
	}
	return false
}

// resolve memilih poin yang berlaku untuk model m dari satu versi, tanpa
// database. Sub-proses dan poin yang tersembunyi dilewati. Poin yang
// cakupannya tidak berlaku untuk model ini juga dilewati. Dari poin yang
// berlaku, varian yang paling spesifik dan tidak tersembunyi menang; tanpa
// varian yang cocok, poin itu sendiri yang dipakai.
func resolve(v ProcessVersion, m modelScope) []ResolvedSubprocess {
	out := []ResolvedSubprocess{}
	for _, s := range v.Subprocesses {
		if s.Hidden {
			continue
		}
		points := []ResolvedPoint{}
		for _, p := range s.Points {
			if p.Hidden || !appliesTo(p.Scope, m) {
				continue
			}
			best := ResolvedPoint{PointID: p.ID, OriginID: p.ID, DocumentType: p.DocumentType, Text: p.Text, Scope: p.Scope, Content: p.Content}
			for _, variant := range p.Variants {
				if variant.Hidden || !appliesTo(variant.Scope, m) || scopeRank(variant.Scope.Level) <= scopeRank(best.Scope.Level) {
					continue
				}
				best.PointID, best.Text, best.Scope, best.Content = variant.ID, variant.Text, variant.Scope, variant.Content
			}
			points = append(points, best)
		}
		if len(points) > 0 {
			out = append(out, ResolvedSubprocess{ID: s.ID, Name: s.Name, Points: points})
		}
	}
	return out
}

// ResolveFor memilih poin satu versi yang berlaku untuk satu model, seperti
// ResolvePoints tetapi tanpa database. groupID kosong untuk model tanpa
// group. Dipakai perakitan dokumen (M4.5).
func ResolveFor(v ProcessVersion, modelID, customerID, groupID string) []ResolvedSubprocess {
	return resolve(v, modelScope{ID: modelID, CustomerID: customerID, GroupID: groupID})
}

// scopeTarget adalah cakupan yang sudah diperiksa, beserta letaknya: customer
// dan group model pemilik sasarannya. Letak ini dipakai untuk memeriksa
// bahwa varian berada di dalam poin asalnya.
type scopeTarget struct {
	Scope      Scope
	CustomerID string // customer sasaran; kosong untuk umum
	GroupID    string // group model sasaran; kosong untuk umum, customer, dan model tanpa group
}

// checkScope memeriksa isian cakupan dan membaca sasarannya. Level kosong
// berarti umum. Sasaran yang tidak ada menjadi masalah input, bukan error.
func checkScope(ctx context.Context, q *db.Queries, in ScopeInput) (scopeTarget, *problem.Problem, error) {
	level := in.Level
	if level == "" {
		level = ScopeGeneral
	}
	if scopeRank(level) == 0 {
		return scopeTarget{}, &problem.Problem{Field: "scope_level", Code: problem.InvalidFormat,
			Message: "Choose general, customer, model group, or model."}, nil
	}
	if level == ScopeGeneral {
		if in.TargetID != "" {
			return scopeTarget{}, &problem.Problem{Field: "scope_target_id", Code: problem.NotAllowed,
				Message: "A general point is not tied to a customer, model group, or model."}, nil
		}
		return scopeTarget{Scope: Scope{Level: ScopeGeneral}}, nil, nil
	}
	if in.TargetID == "" {
		return scopeTarget{}, &problem.Problem{Field: "scope_target_id", Code: problem.Required,
			Message: "Choose the customer, model group, or model."}, nil
	}
	missing := &problem.Problem{Field: "scope_target_id", Code: problem.InvalidFormat,
		Message: "Choose an existing customer, model group, or model."}
	if !db.IsUUID(in.TargetID) {
		return scopeTarget{}, missing, nil
	}
	out := scopeTarget{Scope: Scope{Level: level, TargetID: in.TargetID}}
	var err error
	switch level {
	case ScopeCustomer:
		var c db.GetCustomerRow
		if c, err = q.GetCustomer(ctx, in.TargetID); err == nil {
			out.Scope.Label, out.CustomerID = c.Code, c.ID
		}
	case ScopeModelGroup:
		var g db.ModelGroup
		if g, err = q.GetModelGroup(ctx, in.TargetID); err == nil {
			out.Scope.Label, out.CustomerID, out.GroupID = g.Name, g.CustomerID, g.ID
		}
	case ScopeModel:
		var m db.GetModelRow
		if m, err = q.GetModel(ctx, in.TargetID); err == nil {
			out.Scope.Label, out.CustomerID, out.GroupID = m.Name, m.CustomerID, derefText(m.ModelGroupID)
		}
	}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return scopeTarget{}, missing, nil
	case err != nil:
		return scopeTarget{}, nil, fmt.Errorf("membaca sasaran cakupan: %w", err)
	}
	return out, nil, nil
}

// withinOrigin memeriksa bahwa varian lebih sempit dari poin asalnya dan
// berada di dalamnya: varian poin customer A harus milik customer A, dan
// varian poin group model hanya untuk model di group itu.
func withinOrigin(origin Scope, t scopeTarget) *problem.Problem {
	if scopeRank(t.Scope.Level) <= scopeRank(origin.Level) {
		return &problem.Problem{Field: "scope_level", Code: problem.NotAllowed,
			Message: "A variant is for a narrower scope than its point."}
	}
	outside := &problem.Problem{Field: "scope_target_id", Code: problem.NotAllowed,
		Message: "The variant must be inside the scope of its point."}
	switch origin.Level {
	case ScopeCustomer:
		if t.CustomerID != origin.TargetID {
			return outside
		}
	case ScopeModelGroup:
		if t.GroupID != origin.TargetID {
			return outside
		}
	}
	return nil
}

// params mengisi parameter identitas poin baru dengan cakupan ini. variantOf
// berisi poin asal untuk varian, atau nil untuk poin biasa.
func (t scopeTarget) params(subprocessID, documentTypeID string, variantOf *string) db.CreateTemplatePointParams {
	p := db.CreateTemplatePointParams{
		SubprocessID: subprocessID, DocumentTypeID: documentTypeID, Scope: t.Scope.Level, VariantOf: variantOf,
	}
	target := &t.Scope.TargetID
	switch t.Scope.Level {
	case ScopeCustomer:
		p.CustomerID = target
	case ScopeModelGroup:
		p.ModelGroupID = target
	case ScopeModel:
		p.ModelID = target
	}
	return p
}

// scopeFromColumns menyusun cakupan dari kolom database: tingkat, tiga
// sasaran yang paling banyak satu terisi, dan nama tampilan sasarannya.
func scopeFromColumns(level string, customerID, groupID, modelID, customerCode, groupName, modelName *string) Scope {
	switch level {
	case ScopeCustomer:
		return Scope{Level: level, TargetID: derefText(customerID), Label: derefText(customerCode)}
	case ScopeModelGroup:
		return Scope{Level: level, TargetID: derefText(groupID), Label: derefText(groupName)}
	case ScopeModel:
		return Scope{Level: level, TargetID: derefText(modelID), Label: derefText(modelName)}
	}
	return Scope{Level: ScopeGeneral}
}

// describe menulis cakupan untuk rincian audit, misalnya "model: K-125".
func (t scopeTarget) describe() string {
	if t.Scope.Level == ScopeGeneral || t.Scope.Level == "" {
		return ScopeGeneral
	}
	return t.Scope.Level + ": " + t.Scope.Label
}
