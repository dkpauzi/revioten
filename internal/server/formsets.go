package server

// Handler form set (langkah D, D-133, D-139). Aturannya ada di
// internal/template/formset.go; handler hanya menerjemahkan bentuk data.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/template"
)

// ListFormSets menjawab GET /api/v1/form-sets.
func (a *apiServer) ListFormSets(ctx context.Context, _ ListFormSetsRequestObject) (ListFormSetsResponseObject, error) {
	list, err := a.templates.ListFormSets(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := FormSetList{Items: make([]FormSet, 0, len(list.Sets)), CustomerDefaults: make([]CustomerFormSet, 0, len(list.Defaults))}
	for _, fs := range list.Sets {
		out.Items = append(out.Items, toAPIFormSet(fs))
	}
	for _, d := range list.Defaults {
		out.CustomerDefaults = append(out.CustomerDefaults, CustomerFormSet{
			CustomerId: d.CustomerID, CustomerCode: d.CustomerCode, CustomerName: d.CustomerName, FormSetId: optional(d.FormSetID),
		})
	}
	return ListFormSets200JSONResponse(out), nil
}

// CreateFormSet menjawab POST /api/v1/form-sets.
func (a *apiServer) CreateFormSet(ctx context.Context, req CreateFormSetRequestObject) (CreateFormSetResponseObject, error) {
	b := req.Body
	fs, err := a.templates.CreateFormSet(ctx, actorFrom(ctx), template.NewFormSet{
		Name: b.Name, Sheets: template.FormSetSheets{PFD: b.PfdFormatVersionId, PFMEA: b.PfmeaFormatVersionId, CP: b.CpFormatVersionId},
	})
	if err != nil {
		return nil, err
	}
	return CreateFormSet201JSONResponse(toAPIFormSet(fs)), nil
}

// SetFormSetSheets menjawab PUT /api/v1/form-set-versions/{formSetVersionId}.
func (a *apiServer) SetFormSetSheets(ctx context.Context, req SetFormSetSheetsRequestObject) (SetFormSetSheetsResponseObject, error) {
	b := req.Body
	sheets := template.FormSetSheets{PFD: b.PfdFormatVersionId, PFMEA: b.PfmeaFormatVersionId, CP: b.CpFormatVersionId}
	fs, err := a.templates.SetFormSetSheets(ctx, actorFrom(ctx), req.FormSetVersionId, sheets)
	if err != nil {
		return nil, err
	}
	return SetFormSetSheets200JSONResponse(toAPIFormSet(fs)), nil
}

// CreateFormSetVersion menjawab POST /api/v1/form-sets/{formSetId}/versions.
func (a *apiServer) CreateFormSetVersion(ctx context.Context, req CreateFormSetVersionRequestObject) (CreateFormSetVersionResponseObject, error) {
	fs, err := a.templates.NewFormSetVersion(ctx, actorFrom(ctx), req.FormSetId)
	if err != nil {
		return nil, err
	}
	return CreateFormSetVersion201JSONResponse(toAPIFormSet(fs)), nil
}

// PublishFormSetVersion menjawab POST /api/v1/form-set-versions/{formSetVersionId}/publish.
func (a *apiServer) PublishFormSetVersion(ctx context.Context, req PublishFormSetVersionRequestObject) (PublishFormSetVersionResponseObject, error) {
	fs, err := a.templates.PublishFormSetVersion(ctx, actorFrom(ctx), req.FormSetVersionId)
	if err != nil {
		return nil, err
	}
	return PublishFormSetVersion200JSONResponse(toAPIFormSet(fs)), nil
}

// SetCustomerFormSet menjawab PUT /api/v1/customers/{customerId}/default-form-set.
// Tanpa form_set_id berarti customer kembali memakai set bawaan.
func (a *apiServer) SetCustomerFormSet(ctx context.Context, req SetCustomerFormSetRequestObject) (SetCustomerFormSetResponseObject, error) {
	if err := a.templates.SetCustomerFormSet(ctx, actorFrom(ctx), req.CustomerId, deref(req.Body.FormSetId)); err != nil {
		return nil, err
	}
	return SetCustomerFormSet204Response{}, nil
}

// toAPIFormSet mengubah FormSet domain menjadi FormSet kontrak API.
func toAPIFormSet(fs template.FormSet) FormSet {
	sheet := func(s template.SheetRef) FormSetSheet {
		return FormSetSheet{FormatVersionId: s.FormatVersionID, TemplateName: s.TemplateName, Version: s.Version, Status: FormSetSheetStatus(s.Status)}
	}
	out := FormSet{Id: fs.ID, Key: fs.Key, Name: fs.Name, Default: fs.Default, Versions: make([]FormSetVersion, 0, len(fs.Versions))}
	for _, v := range fs.Versions {
		out.Versions = append(out.Versions, FormSetVersion{
			Id: v.ID, Version: v.Version, Status: FormSetVersionStatus(v.Status), PublishedAt: v.PublishedAt,
			Pfd: sheet(v.PFD), Pfmea: sheet(v.PFMEA), Cp: sheet(v.CP),
		})
	}
	return out
}
