package server

// Handler yokoten (M5.9). Aturannya ada di internal/change/yokoten.go.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/change"
)

// ListYokotens menjawab GET /api/v1/yokotens.
func (a *apiServer) ListYokotens(ctx context.Context, _ ListYokotensRequestObject) (ListYokotensResponseObject, error) {
	list, err := a.changeRequests.ListYokotens(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := YokotenList{Items: make([]Yokoten, 0, len(list))}
	for _, y := range list {
		out.Items = append(out.Items, toAPIYokoten(y))
	}
	return ListYokotens200JSONResponse(out), nil
}

// CreateYokoten menjawab POST /api/v1/yokotens.
func (a *apiServer) CreateYokoten(ctx context.Context, req CreateYokotenRequestObject) (CreateYokotenResponseObject, error) {
	b := req.Body
	y, err := a.changeRequests.CreateYokoten(ctx, actorFrom(ctx), change.NewYokoten{SourceChangeID: b.SourceChangeId, Reason: b.Reason, Targets: toTargetFilter(b.Targets)})
	if err != nil {
		return nil, err
	}
	return CreateYokoten201JSONResponse(toAPIYokoten(y)), nil
}

// SearchYokotenTargets menjawab POST /api/v1/yokoten-targets/search (tanpa menyimpan).
func (a *apiServer) SearchYokotenTargets(ctx context.Context, req SearchYokotenTargetsRequestObject) (SearchYokotenTargetsResponseObject, error) {
	b := req.Body
	// Alasan hanya syarat penyimpanan; pencarian memakai teks pengganti.
	y, err := a.changeRequests.CreateYokoten(ctx, actorFrom(ctx), change.NewYokoten{SourceChangeID: b.SourceChangeId, Reason: "search", Targets: toTargetFilter(b.Targets), DryRun: true})
	if err != nil {
		return nil, err
	}
	out := YokotenTargetList{Items: make([]YokotenTarget, 0, len(y.Targets))}
	for _, t := range y.Targets {
		out.Items = append(out.Items, toAPIYokotenTarget(t))
	}
	return SearchYokotenTargets200JSONResponse(out), nil
}

// GetYokoten menjawab GET /api/v1/yokotens/{yokotenId}.
func (a *apiServer) GetYokoten(ctx context.Context, req GetYokotenRequestObject) (GetYokotenResponseObject, error) {
	y, err := a.changeRequests.GetYokoten(ctx, actorFrom(ctx), req.YokotenId)
	if err != nil {
		return nil, err
	}
	return GetYokoten200JSONResponse(toAPIYokoten(y)), nil
}

// DecideYokotenTarget menjawab POST /api/v1/yokotens/{yokotenId}/targets/{targetId}/decision.
func (a *apiServer) DecideYokotenTarget(ctx context.Context, req DecideYokotenTargetRequestObject) (DecideYokotenTargetResponseObject, error) {
	y, err := a.changeRequests.Decide(ctx, actorFrom(ctx), req.YokotenId, req.TargetId, req.Body.Decision, deref(req.Body.Reason))
	if err != nil {
		return nil, err
	}
	return DecideYokotenTarget200JSONResponse(toAPIYokoten(y)), nil
}

func toTargetFilter(f YokotenTargetFilter) change.TargetFilter {
	out := change.TargetFilter{CustomerID: deref(f.CustomerId), ModelGroupID: deref(f.ModelGroupId), ProcessID: deref(f.ProcessId), Text: deref(f.Text)}
	if f.ModelIds != nil {
		out.ModelIDs = *f.ModelIds
	}
	return out
}

func toAPIYokoten(y change.Yokoten) Yokoten {
	out := Yokoten{
		Id: y.ID, Ref: y.Ref, SourceChange: ChangeRef{Id: y.SourceID, Name: y.SourceChange}, Reason: y.Reason,
		ProcessFilter: optional(y.ProcessFilter), TextFilter: optional(y.TextFilter), CreatedBy: y.CreatedBy, CreatedAt: y.CreatedAt,
		Targets: make([]YokotenTarget, 0, len(y.Targets)),
	}
	for _, t := range y.Targets {
		out.Targets = append(out.Targets, toAPIYokotenTarget(t))
	}
	return out
}

func toAPIYokotenTarget(t change.YokotenTarget) YokotenTarget {
	out := YokotenTarget{
		Id: optional(t.ID), ModelId: t.ModelID, ModelName: t.ModelName, CustomerId: t.CustomerID, CustomerCode: t.CustomerCode,
		Decision: YokotenTargetDecision(t.Decision), Reason: optional(t.Reason), DecidedBy: optional(t.DecidedBy), DecidedAt: t.DecidedAt,
		Progress: YokotenTargetProgress(t.Progress), Documents: make([]YokotenDocument, 0, len(t.Documents)),
	}
	if t.ChangeID != "" {
		out.Change = &ChangeRef{Id: t.ChangeID, Name: t.ChangeRef}
	}
	for _, d := range t.Documents {
		out.Documents = append(out.Documents, YokotenDocument{
			Id: d.ID, Number: d.Number, DocumentType: d.DocumentType, Revision: d.Revision, Status: d.Status, ReleasedRevision: d.ReleasedRevision,
		})
	}
	return out
}
