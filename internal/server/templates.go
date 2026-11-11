package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/template"
)

// ListFormatTemplates menjawab GET /api/v1/format-templates (M3.1).
func (a *apiServer) ListFormatTemplates(ctx context.Context, _ ListFormatTemplatesRequestObject) (ListFormatTemplatesResponseObject, error) {
	list, err := a.templates.ListFormatTemplates(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]FormatTemplate, 0, len(list))
	for _, t := range list {
		items = append(items, toAPIFormatTemplate(t))
	}
	return ListFormatTemplates200JSONResponse{Items: items}, nil
}

// CreateFormatTemplate menjawab POST /api/v1/format-templates.
func (a *apiServer) CreateFormatTemplate(ctx context.Context, req CreateFormatTemplateRequestObject) (CreateFormatTemplateResponseObject, error) {
	b := req.Body
	in := template.NewTemplate{
		Name: b.Name, DocumentTypeID: b.DocumentTypeId, CustomerID: deref(b.CustomerId), CopyFromVersionID: deref(b.CopyFromVersionId),
	}
	if b.Methodology != nil {
		in.Methodology = string(*b.Methodology)
	}
	if b.RiskMethod != nil {
		in.RiskMethod = string(*b.RiskMethod)
	}
	created, err := a.templates.CreateFormatTemplate(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateFormatTemplate201JSONResponse(toAPIFormatTemplate(created)), nil
}

// RenameFormatTemplate menjawab PUT /api/v1/format-templates/{formatTemplateId}.
func (a *apiServer) RenameFormatTemplate(ctx context.Context, req RenameFormatTemplateRequestObject) (RenameFormatTemplateResponseObject, error) {
	renamed, err := a.templates.RenameFormatTemplate(ctx, actorFrom(ctx), req.FormatTemplateId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return RenameFormatTemplate200JSONResponse(toAPIFormatTemplate(renamed)), nil
}

// GetFormatVersion menjawab GET /api/v1/format-template-versions/{formatVersionId}.
func (a *apiServer) GetFormatVersion(ctx context.Context, req GetFormatVersionRequestObject) (GetFormatVersionResponseObject, error) {
	v, err := a.templates.GetFormatVersion(ctx, actorFrom(ctx), req.FormatVersionId)
	if err != nil {
		return nil, err
	}
	return GetFormatVersion200JSONResponse(toAPIFormatVersion(v)), nil
}

// SetFormatApRules menjawab PUT /api/v1/format-template-versions/{formatVersionId}/ap-rules (D-134).
func (a *apiServer) SetFormatApRules(ctx context.Context, req SetFormatApRulesRequestObject) (SetFormatApRulesResponseObject, error) {
	rules := make([]template.APRule, 0, len(req.Body.Items))
	for _, r := range req.Body.Items {
		rules = append(rules, template.APRule{
			SMin: r.SMin, SMax: r.SMax, OMin: r.OMin, OMax: r.OMax, DMin: r.DMin, DMax: r.DMax, Priority: string(r.Priority),
		})
	}
	v, err := a.templates.SetAPRules(ctx, actorFrom(ctx), req.FormatVersionId, rules)
	if err != nil {
		return nil, err
	}
	return SetFormatApRules200JSONResponse(toAPIFormatVersion(v)), nil
}

// GetActionPriority menjawab GET /api/v1/format-template-versions/{formatVersionId}/action-priority (D-134).
func (a *apiServer) GetActionPriority(ctx context.Context, req GetActionPriorityRequestObject) (GetActionPriorityResponseObject, error) {
	p := req.Params
	priority, err := a.templates.ActionPriorityOf(ctx, actorFrom(ctx), req.FormatVersionId, p.Severity, p.Occurrence, p.Detection)
	if err != nil {
		return nil, err
	}
	return GetActionPriority200JSONResponse{Priority: ActionPriorityResultPriority(priority)}, nil
}

// PublishFormatVersion menjawab POST /api/v1/format-template-versions/{formatVersionId}/publish (M3.4).
func (a *apiServer) PublishFormatVersion(ctx context.Context, req PublishFormatVersionRequestObject) (PublishFormatVersionResponseObject, error) {
	v, err := a.templates.PublishFormatVersion(ctx, actorFrom(ctx), req.FormatVersionId)
	if err != nil {
		return nil, err
	}
	return PublishFormatVersion200JSONResponse(toAPIFormatVersion(v)), nil
}

// CreateFormatVersion menjawab POST /api/v1/format-templates/{formatTemplateId}/versions (M3.4).
func (a *apiServer) CreateFormatVersion(ctx context.Context, req CreateFormatVersionRequestObject) (CreateFormatVersionResponseObject, error) {
	v, err := a.templates.NewFormatVersion(ctx, actorFrom(ctx), req.FormatTemplateId)
	if err != nil {
		return nil, err
	}
	return CreateFormatVersion201JSONResponse(toAPIFormatVersion(v)), nil
}

// UpdateFormatHighlights menjawab PUT /api/v1/format-template-versions/{formatVersionId}.
// Aturan yang tidak dikirim tetap (D-111).
func (a *apiServer) UpdateFormatHighlights(ctx context.Context, req UpdateFormatHighlightsRequestObject) (UpdateFormatHighlightsResponseObject, error) {
	in := template.HighlightSettings{RPNAbove: req.Body.HighlightRpnAbove, SeverityAtLeast: req.Body.HighlightSeverityAtLeast}
	v, err := a.templates.SetHighlights(ctx, actorFrom(ctx), req.FormatVersionId, in)
	if err != nil {
		return nil, err
	}
	return UpdateFormatHighlights200JSONResponse(toAPIFormatVersion(v)), nil
}

// AddFormatField menjawab POST /api/v1/format-template-versions/{formatVersionId}/fields.
func (a *apiServer) AddFormatField(ctx context.Context, req AddFormatFieldRequestObject) (AddFormatFieldResponseObject, error) {
	b := req.Body
	in := template.NewField{Area: string(b.Area), Kind: string(b.Kind), Label: b.Label, Options: fromAPIOptions(b.Options)}
	if b.Required != nil {
		in.Required = *b.Required
	}
	if b.RatingScale != nil {
		in.RatingScale = string(*b.RatingScale)
	}
	if b.RatingSet != nil {
		in.RatingSet = *b.RatingSet
	}
	v, err := a.templates.AddField(ctx, actorFrom(ctx), req.FormatVersionId, in)
	if err != nil {
		return nil, err
	}
	return AddFormatField201JSONResponse(toAPIFormatVersion(v)), nil
}

// UpdateFormatField menjawab PUT /api/v1/format-template-versions/{formatVersionId}/fields/{formatFieldId}.
// Isian yang tidak dikirim tetap (D-111).
func (a *apiServer) UpdateFormatField(ctx context.Context, req UpdateFormatFieldRequestObject) (UpdateFormatFieldResponseObject, error) {
	b := req.Body
	in := template.FieldUpdate{Label: b.Label, Required: b.Required, Hidden: b.Hidden, Position: b.Position, Options: fromAPIOptions(b.Options)}
	v, err := a.templates.UpdateField(ctx, actorFrom(ctx), req.FormatVersionId, req.FormatFieldId, in)
	if err != nil {
		return nil, err
	}
	return UpdateFormatField200JSONResponse(toAPIFormatVersion(v)), nil
}

// SetFormatRatingCriteria menjawab PUT /api/v1/format-template-versions/{formatVersionId}/rating-criteria.
// Item tanpa criteria tidak mengubah apa pun (D-111).
func (a *apiServer) SetFormatRatingCriteria(ctx context.Context, req SetFormatRatingCriteriaRequestObject) (SetFormatRatingCriteriaResponseObject, error) {
	ratings := make([]template.Rating, 0, len(req.Body.Items))
	for _, r := range req.Body.Items {
		if r.Criteria != nil {
			ratings = append(ratings, template.Rating{Scale: string(r.Scale), Value: r.Value, Criteria: *r.Criteria})
		}
	}
	v, err := a.templates.SetRatingCriteria(ctx, actorFrom(ctx), req.FormatVersionId, ratings)
	if err != nil {
		return nil, err
	}
	return SetFormatRatingCriteria200JSONResponse(toAPIFormatVersion(v)), nil
}

// SaveFormatSymbols menjawab PUT /api/v1/format-template-versions/{formatVersionId}/symbols.
func (a *apiServer) SaveFormatSymbols(ctx context.Context, req SaveFormatSymbolsRequestObject) (SaveFormatSymbolsResponseObject, error) {
	symbols := make([]template.Symbol, 0, len(req.Body.Items))
	for _, sym := range req.Body.Items {
		symbols = append(symbols, template.Symbol{Key: deref(sym.Key), Name: sym.Name})
	}
	v, err := a.templates.SaveSymbols(ctx, actorFrom(ctx), req.FormatVersionId, symbols)
	if err != nil {
		return nil, err
	}
	return SaveFormatSymbols200JSONResponse(toAPIFormatVersion(v)), nil
}

// toAPIFormatTemplate mengubah FormatTemplate domain menjadi FormatTemplate kontrak API.
func toAPIFormatTemplate(t template.FormatTemplate) FormatTemplate {
	out := FormatTemplate{
		Id: t.ID, Name: t.Name, CreatedAt: t.CreatedAt,
		DocumentType: DocumentTypeRef{Id: t.DocumentTypeID, Key: DocumentTypeRefKey(t.DocumentTypeKey), Name: t.DocumentTypeName},
		Versions:     make([]FormatVersionSummary, 0, len(t.Versions)),
	}
	if t.Methodology != "" {
		m := FormatTemplateMethodology(t.Methodology)
		out.Methodology = &m
	}
	if t.CustomerID != "" {
		out.CustomerId, out.CustomerCode = &t.CustomerID, &t.CustomerCode
	}
	for _, v := range t.Versions {
		out.Versions = append(out.Versions, FormatVersionSummary{
			Id: v.ID, Version: v.Version, Status: FormatVersionSummaryStatus(v.Status), RiskMethod: FormatVersionSummaryRiskMethod(v.RiskMethod), PublishedAt: v.PublishedAt,
		})
	}
	return out
}

// toAPIFormatVersion mengubah FormatVersion domain menjadi FormatVersion kontrak API.
// Daftar kosong dikirim sebagai [] dan bukan null.
func toAPIFormatVersion(v template.FormatVersion) FormatVersion {
	out := FormatVersion{
		Id: v.ID, TemplateId: v.TemplateID, Version: v.Version, Status: FormatVersionStatus(v.Status),
		RiskMethod: FormatVersionRiskMethod(v.RiskMethod), HighlightRpnAbove: v.HighlightRPNAbove,
		HighlightSeverityAtLeast: v.HighlightSeverityAtLeast, CreatedAt: v.CreatedAt, PublishedAt: v.PublishedAt,
		Fields:  make([]FormatField, 0, len(v.Fields)),
		Ratings: make([]RatingCriterion, 0, len(v.Ratings)),
		Symbols: make([]FormatSymbol, 0, len(v.Symbols)),
		ApRules: make([]ApRule, 0, len(v.APRules)),
	}
	for _, r := range v.APRules {
		out.ApRules = append(out.ApRules, ApRule{
			SMin: r.SMin, SMax: r.SMax, OMin: r.OMin, OMax: r.OMax, DMin: r.DMin, DMax: r.DMax, Priority: ApRulePriority(r.Priority),
		})
	}
	for _, f := range v.Fields {
		field := FormatField{
			Id: f.ID, Area: FormatFieldArea(f.Area), Kind: FormatFieldKind(f.Kind), Label: f.Label,
			Position: f.Position, Required: f.Required, Hidden: f.Hidden,
		}
		if f.RatingScale != "" {
			scale := FormatFieldRatingScale(f.RatingScale)
			field.RatingScale = &scale
		}
		if f.RatingSet != 0 {
			field.RatingSet = &f.RatingSet
		}
		if f.Key != "" {
			field.Key = &f.Key
		}
		if f.Group != "" {
			field.Group = &f.Group
		}
		if f.Kind == template.KindSelect {
			options := make([]FieldOption, 0, len(f.Options))
			for _, o := range f.Options {
				options = append(options, FieldOption{Key: &o.Key, Label: o.Label})
			}
			field.Options = &options
		}
		out.Fields = append(out.Fields, field)
	}
	for _, r := range v.Ratings {
		out.Ratings = append(out.Ratings, RatingCriterion{Scale: RatingCriterionScale(r.Scale), Value: r.Value, Criteria: r.Criteria})
	}
	for _, sym := range v.Symbols {
		out.Symbols = append(out.Symbols, FormatSymbol{Key: &sym.Key, Name: sym.Name, Automatic: &sym.Automatic})
	}
	return out
}

// fromAPIOptions mengubah pilihan dari kontrak API. nil berarti tidak dikirim.
func fromAPIOptions(options *[]FieldOption) []template.Option {
	if options == nil {
		return nil
	}
	out := make([]template.Option, 0, len(*options))
	for _, o := range *options {
		out = append(out, template.Option{Key: deref(o.Key), Label: o.Label})
	}
	return out
}
