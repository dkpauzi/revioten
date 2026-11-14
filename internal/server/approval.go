package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/settings"
)

// ListPositions menjawab GET /api/v1/positions.
func (a *apiServer) ListPositions(ctx context.Context, _ ListPositionsRequestObject) (ListPositionsResponseObject, error) {
	list, err := a.approval.ListPositions(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]Position, 0, len(list))
	for _, p := range list {
		items = append(items, toAPIPosition(p))
	}
	return ListPositions200JSONResponse{Items: items}, nil
}

// UpdatePosition menjawab PUT /api/v1/positions/{positionId}.
func (a *apiServer) UpdatePosition(ctx context.Context, req UpdatePositionRequestObject) (UpdatePositionResponseObject, error) {
	renamed, err := a.approval.RenamePosition(ctx, actorFrom(ctx), req.PositionId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return UpdatePosition200JSONResponse(toAPIPosition(renamed)), nil
}

// AddPositionHolder menjawab POST /api/v1/positions/{positionId}/holders.
func (a *apiServer) AddPositionHolder(ctx context.Context, req AddPositionHolderRequestObject) (AddPositionHolderResponseObject, error) {
	in := approval.NewHolder{PersonID: req.Body.PersonId}
	if req.Body.ProjectGroupId != nil {
		in.ProjectGroupID = *req.Body.ProjectGroupId
	}
	holder, err := a.approval.AddHolder(ctx, actorFrom(ctx), req.PositionId, in)
	if err != nil {
		return nil, err
	}
	return AddPositionHolder201JSONResponse(toAPIHolder(holder)), nil
}

// RemovePositionHolder menjawab DELETE /api/v1/positions/{positionId}/holders/{holderId}.
func (a *apiServer) RemovePositionHolder(ctx context.Context, req RemovePositionHolderRequestObject) (RemovePositionHolderResponseObject, error) {
	if err := a.approval.RemoveHolder(ctx, actorFrom(ctx), req.PositionId, req.HolderId); err != nil {
		return nil, err
	}
	return RemovePositionHolder204Response{}, nil
}

// ListApprovalRoutes menjawab GET /api/v1/approval-routes.
func (a *apiServer) ListApprovalRoutes(ctx context.Context, _ ListApprovalRoutesRequestObject) (ListApprovalRoutesResponseObject, error) {
	list, err := a.approval.ListRoutes(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]ApprovalRoute, 0, len(list))
	for _, r := range list {
		items = append(items, toAPIRoute(r))
	}
	return ListApprovalRoutes200JSONResponse{Items: items}, nil
}

// CreateApprovalRoute menjawab POST /api/v1/approval-routes.
func (a *apiServer) CreateApprovalRoute(ctx context.Context, req CreateApprovalRouteRequestObject) (CreateApprovalRouteResponseObject, error) {
	created, err := a.approval.CreateRoute(ctx, actorFrom(ctx), fromAPIRouteInput(*req.Body))
	if err != nil {
		return nil, err
	}
	return CreateApprovalRoute201JSONResponse(toAPIRoute(created)), nil
}

// UpdateApprovalRoute menjawab PUT /api/v1/approval-routes/{approvalRouteId}.
func (a *apiServer) UpdateApprovalRoute(ctx context.Context, req UpdateApprovalRouteRequestObject) (UpdateApprovalRouteResponseObject, error) {
	updated, err := a.approval.UpdateRoute(ctx, actorFrom(ctx), req.ApprovalRouteId, fromAPIRouteInput(*req.Body))
	if err != nil {
		return nil, err
	}
	return UpdateApprovalRoute200JSONResponse(toAPIRoute(updated)), nil
}

// GetSystemSettings menjawab GET /api/v1/system-settings.
func (a *apiServer) GetSystemSettings(ctx context.Context, _ GetSystemSettingsRequestObject) (GetSystemSettingsResponseObject, error) {
	current, err := a.settings.Get(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return GetSystemSettings200JSONResponse(toAPISettings(current)), nil
}

// UpdateSystemSettings menjawab PUT /api/v1/system-settings. Saklar yang
// tidak dikirim tetap seperti sebelumnya (E-34).
func (a *apiServer) UpdateSystemSettings(ctx context.Context, req UpdateSystemSettingsRequestObject) (UpdateSystemSettingsResponseObject, error) {
	in := settings.Update{SignDatesEnabled: req.Body.SignDatesEnabled, QREnabled: req.Body.QrEnabled}
	if req.Body.FormApprovalMode != nil {
		mode := string(*req.Body.FormApprovalMode)
		in.FormApprovalMode = &mode
	}
	updated, err := a.settings.Apply(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return UpdateSystemSettings200JSONResponse(toAPISettings(updated)), nil
}

// toAPISettings mengubah Settings domain menjadi SystemSettings kontrak API.
func toAPISettings(s settings.Settings) SystemSettings {
	return SystemSettings{SignDatesEnabled: s.SignDatesEnabled, QrEnabled: s.QREnabled, FormApprovalMode: SystemSettingsFormApprovalMode(s.FormApprovalMode)}
}

// toAPIPosition mengubah Position domain menjadi Position kontrak API.
func toAPIPosition(p approval.Position) Position {
	out := Position{Id: p.ID, Key: PositionKey(p.Key), Name: p.Name, PerProjectGroup: p.PerProjectGroup, SingleHolder: p.SingleHolder, Holders: make([]PositionHolder, 0, len(p.Holders))}
	for _, h := range p.Holders {
		out.Holders = append(out.Holders, toAPIHolder(h))
	}
	return out
}

// toAPIHolder mengubah Holder domain menjadi PositionHolder kontrak API.
// Pemegang tanpa Project Group dikirim tanpa field project_group.
func toAPIHolder(h approval.Holder) PositionHolder {
	out := PositionHolder{Id: h.ID, Person: Person{Id: h.PersonID, Name: h.PersonName}, CreatedAt: h.CreatedAt}
	if h.ProjectGroupID != "" {
		out.ProjectGroup = &ProjectGroupRef{Id: h.ProjectGroupID, Code: h.ProjectGroupCode}
	}
	return out
}

// toAPIRoute mengubah Route domain menjadi ApprovalRoute kontrak API.
func toAPIRoute(r approval.Route) ApprovalRoute {
	out := ApprovalRoute{Id: r.ID, Name: r.Name, CreatedAt: r.CreatedAt, Slots: make([]RouteSlot, 0, len(r.Slots))}
	for _, s := range r.Slots {
		slot := RouteSlot{Stage: RouteSlotStage(s.Stage), PrintTitle: s.PrintTitle, Positions: make([]PositionRef, 0, len(s.Positions))}
		for _, p := range s.Positions {
			slot.Positions = append(slot.Positions, PositionRef{Id: p.ID, Name: p.Name})
		}
		out.Slots = append(out.Slots, slot)
	}
	return out
}

// fromAPIRouteInput mengubah isian rute dari kontrak API menjadi RouteInput domain.
func fromAPIRouteInput(in ApprovalRouteInput) approval.RouteInput {
	out := approval.RouteInput{Name: in.Name}
	for _, s := range in.Slots {
		out.Slots = append(out.Slots, approval.SlotInput{Stage: string(s.Stage), PrintTitle: s.PrintTitle, PositionIDs: s.PositionIds})
	}
	return out
}
