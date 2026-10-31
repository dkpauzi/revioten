package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/template"
)

// ListProcessTemplates menjawab GET /api/v1/process-templates (M3.2).
func (a *apiServer) ListProcessTemplates(ctx context.Context, _ ListProcessTemplatesRequestObject) (ListProcessTemplatesResponseObject, error) {
	list, err := a.processTemplates.ListProcessTemplates(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]ProcessTemplate, 0, len(list))
	for _, t := range list {
		items = append(items, toAPIProcessTemplate(t))
	}
	return ListProcessTemplates200JSONResponse{Items: items}, nil
}

// CreateProcessTemplate menjawab POST /api/v1/process-templates.
func (a *apiServer) CreateProcessTemplate(ctx context.Context, req CreateProcessTemplateRequestObject) (CreateProcessTemplateResponseObject, error) {
	created, err := a.processTemplates.CreateProcessTemplate(ctx, actorFrom(ctx), req.Body.ProcessId)
	if err != nil {
		return nil, err
	}
	return CreateProcessTemplate201JSONResponse(toAPIProcessTemplate(created)), nil
}

// GetProcessTemplateVersion menjawab GET /api/v1/process-template-versions/{processVersionId}.
func (a *apiServer) GetProcessTemplateVersion(ctx context.Context, req GetProcessTemplateVersionRequestObject) (GetProcessTemplateVersionResponseObject, error) {
	v, err := a.processTemplates.GetProcessVersion(ctx, actorFrom(ctx), req.ProcessVersionId)
	if err != nil {
		return nil, err
	}
	return GetProcessTemplateVersion200JSONResponse(toAPIProcessVersion(v)), nil
}

// AddSubprocess menjawab POST /api/v1/process-template-versions/{processVersionId}/subprocesses.
func (a *apiServer) AddSubprocess(ctx context.Context, req AddSubprocessRequestObject) (AddSubprocessResponseObject, error) {
	v, err := a.processTemplates.AddSubprocess(ctx, actorFrom(ctx), req.ProcessVersionId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return AddSubprocess201JSONResponse(toAPIProcessVersion(v)), nil
}

// UpdateSubprocess menjawab PUT /api/v1/process-template-versions/{processVersionId}/subprocesses/{subprocessId}.
// Isian yang tidak dikirim tetap (D-111).
func (a *apiServer) UpdateSubprocess(ctx context.Context, req UpdateSubprocessRequestObject) (UpdateSubprocessResponseObject, error) {
	b := req.Body
	in := template.SubprocessUpdate{Name: b.Name, Position: b.Position, Hidden: b.Hidden}
	v, err := a.processTemplates.UpdateSubprocess(ctx, actorFrom(ctx), req.ProcessVersionId, req.SubprocessId, in)
	if err != nil {
		return nil, err
	}
	return UpdateSubprocess200JSONResponse(toAPIProcessVersion(v)), nil
}

// AddTemplatePoint menjawab POST /api/v1/process-template-versions/{processVersionId}/points.
func (a *apiServer) AddTemplatePoint(ctx context.Context, req AddTemplatePointRequestObject) (AddTemplatePointResponseObject, error) {
	b := req.Body
	in := template.NewPoint{SubprocessID: b.SubprocessId, DocumentTypeID: b.DocumentTypeId, Text: b.Text}
	v, err := a.processTemplates.AddPoint(ctx, actorFrom(ctx), req.ProcessVersionId, in)
	if err != nil {
		return nil, err
	}
	return AddTemplatePoint201JSONResponse(toAPIProcessVersion(v)), nil
}

// UpdateTemplatePoint menjawab PUT /api/v1/process-template-versions/{processVersionId}/points/{pointId}.
// Isian yang tidak dikirim tetap (D-111).
func (a *apiServer) UpdateTemplatePoint(ctx context.Context, req UpdateTemplatePointRequestObject) (UpdateTemplatePointResponseObject, error) {
	b := req.Body
	in := template.PointUpdate{Text: b.Text, Position: b.Position, Hidden: b.Hidden}
	v, err := a.processTemplates.UpdatePoint(ctx, actorFrom(ctx), req.ProcessVersionId, req.PointId, in)
	if err != nil {
		return nil, err
	}
	return UpdateTemplatePoint200JSONResponse(toAPIProcessVersion(v)), nil
}

// toAPIProcessTemplate mengubah ProcessTemplate domain menjadi ProcessTemplate kontrak API.
func toAPIProcessTemplate(t template.ProcessTemplate) ProcessTemplate {
	out := ProcessTemplate{
		Id: t.ID, Process: ProcessRef{Id: t.Process.ID, Name: t.Process.Name}, CreatedAt: t.CreatedAt,
		Versions: make([]ProcessVersionSummary, 0, len(t.Versions)),
	}
	for _, v := range t.Versions {
		out.Versions = append(out.Versions, ProcessVersionSummary{Id: v.ID, Version: v.Version, Status: ProcessVersionSummaryStatus(v.Status)})
	}
	return out
}

// toAPIProcessVersion mengubah ProcessVersion domain menjadi ProcessTemplateVersion
// kontrak API. Daftar kosong dikirim sebagai [] dan bukan null.
func toAPIProcessVersion(v template.ProcessVersion) ProcessTemplateVersion {
	out := ProcessTemplateVersion{
		Id: v.ID, TemplateId: v.TemplateID, Process: ProcessRef{Id: v.Process.ID, Name: v.Process.Name},
		Version: v.Version, Status: ProcessTemplateVersionStatus(v.Status), CreatedAt: v.CreatedAt,
		Subprocesses: make([]Subprocess, 0, len(v.Subprocesses)),
	}
	for _, s := range v.Subprocesses {
		sub := Subprocess{Id: s.ID, Name: s.Name, Position: s.Position, Hidden: s.Hidden, Points: make([]TemplatePoint, 0, len(s.Points))}
		for _, p := range s.Points {
			sub.Points = append(sub.Points, TemplatePoint{
				Id: p.ID, Text: p.Text, Position: p.Position, Hidden: p.Hidden,
				DocumentType: DocumentTypeRef{Id: p.DocumentType.ID, Key: DocumentTypeRefKey(p.DocumentType.Key), Name: p.DocumentType.Name},
			})
		}
		out.Subprocesses = append(out.Subprocesses, sub)
	}
	return out
}
