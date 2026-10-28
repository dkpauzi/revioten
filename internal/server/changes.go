package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/change"
)

// ListChangeTypes menjawab GET /api/v1/change-types.
func (a *apiServer) ListChangeTypes(ctx context.Context, _ ListChangeTypesRequestObject) (ListChangeTypesResponseObject, error) {
	list, err := a.changes.ListChangeTypes(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]ChangeType, 0, len(list))
	for _, ct := range list {
		items = append(items, toAPIChangeType(ct))
	}
	return ListChangeTypes200JSONResponse{Items: items}, nil
}

// UpdateChangeType menjawab PUT /api/v1/change-types/{changeTypeId}.
func (a *apiServer) UpdateChangeType(ctx context.Context, req UpdateChangeTypeRequestObject) (UpdateChangeTypeResponseObject, error) {
	renamed, err := a.changes.RenameChangeType(ctx, actorFrom(ctx), req.ChangeTypeId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return UpdateChangeType200JSONResponse(toAPIChangeType(renamed)), nil
}

// ListChangeSources menjawab GET /api/v1/change-sources.
func (a *apiServer) ListChangeSources(ctx context.Context, _ ListChangeSourcesRequestObject) (ListChangeSourcesResponseObject, error) {
	list, err := a.changes.ListChangeSources(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]ChangeSource, 0, len(list))
	for _, s := range list {
		items = append(items, toAPIChangeSource(s))
	}
	return ListChangeSources200JSONResponse{Items: items}, nil
}

// CreateChangeSource menjawab POST /api/v1/change-sources.
func (a *apiServer) CreateChangeSource(ctx context.Context, req CreateChangeSourceRequestObject) (CreateChangeSourceResponseObject, error) {
	created, err := a.changes.CreateChangeSource(ctx, actorFrom(ctx), req.Body.Name)
	if err != nil {
		return nil, err
	}
	return CreateChangeSource201JSONResponse(toAPIChangeSource(created)), nil
}

// UpdateChangeSource menjawab PUT /api/v1/change-sources/{changeSourceId}.
// active yang tidak dikirim berarti status tetap (lihat E-34).
func (a *apiServer) UpdateChangeSource(ctx context.Context, req UpdateChangeSourceRequestObject) (UpdateChangeSourceResponseObject, error) {
	in := change.ChangeSourceUpdate{Name: req.Body.Name, Active: req.Body.Active}
	updated, err := a.changes.UpdateChangeSource(ctx, actorFrom(ctx), req.ChangeSourceId, in)
	if err != nil {
		return nil, err
	}
	return UpdateChangeSource200JSONResponse(toAPIChangeSource(updated)), nil
}

// ListFourMCategories menjawab GET /api/v1/four-m-categories.
func (a *apiServer) ListFourMCategories(ctx context.Context, _ ListFourMCategoriesRequestObject) (ListFourMCategoriesResponseObject, error) {
	list, err := a.changes.ListFourMCategories(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]FourMCategory, 0, len(list))
	for _, c := range list {
		items = append(items, toAPIFourMCategory(c))
	}
	return ListFourMCategories200JSONResponse{Items: items}, nil
}

// UpdateFourMCategory menjawab PUT /api/v1/four-m-categories/{fourMCategoryId}.
func (a *apiServer) UpdateFourMCategory(ctx context.Context, req UpdateFourMCategoryRequestObject) (UpdateFourMCategoryResponseObject, error) {
	renamed, err := a.changes.RenameFourMCategory(ctx, actorFrom(ctx), req.FourMCategoryId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return UpdateFourMCategory200JSONResponse(toAPIFourMCategory(renamed)), nil
}

// toAPIChangeType mengubah ChangeType domain menjadi ChangeType kontrak API.
func toAPIChangeType(ct change.ChangeType) ChangeType {
	return ChangeType{Id: ct.ID, Key: ChangeTypeKey(ct.Key), Name: ct.Name}
}

// toAPIChangeSource mengubah ChangeSource domain menjadi ChangeSource kontrak
// API. Sumber tanpa mekanisme sistem dikirim tanpa field mechanism.
func toAPIChangeSource(s change.ChangeSource) ChangeSource {
	out := ChangeSource{Id: s.ID, Name: s.Name, Active: s.Active, CreatedAt: s.CreatedAt}
	if s.Mechanism != "" {
		mechanism := ChangeSourceMechanism(s.Mechanism)
		out.Mechanism = &mechanism
	}
	return out
}

// toAPIFourMCategory mengubah FourMCategory domain menjadi FourMCategory kontrak API.
func toAPIFourMCategory(c change.FourMCategory) FourMCategory {
	return FourMCategory{Id: c.ID, Key: FourMCategoryKey(c.Key), Name: c.Name}
}
