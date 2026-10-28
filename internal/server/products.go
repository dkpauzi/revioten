package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/product"
)

// ListModelGroups menjawab GET /api/v1/model-groups?customer_id=...
func (a *apiServer) ListModelGroups(ctx context.Context, req ListModelGroupsRequestObject) (ListModelGroupsResponseObject, error) {
	groups, err := a.products.ListModelGroups(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	items := make([]ModelGroup, 0, len(groups))
	for _, g := range groups {
		items = append(items, toAPIModelGroup(g))
	}
	return ListModelGroups200JSONResponse{Items: items}, nil
}

// CreateModelGroup menjawab POST /api/v1/model-groups.
func (a *apiServer) CreateModelGroup(ctx context.Context, req CreateModelGroupRequestObject) (CreateModelGroupResponseObject, error) {
	in := product.NewModelGroup{CustomerID: req.Body.CustomerId, Name: req.Body.Name}
	group, err := a.products.CreateModelGroup(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateModelGroup201JSONResponse(toAPIModelGroup(group)), nil
}

// UpdateModelGroup menjawab PUT /api/v1/model-groups/{modelGroupId}.
func (a *apiServer) UpdateModelGroup(ctx context.Context, req UpdateModelGroupRequestObject) (UpdateModelGroupResponseObject, error) {
	group, err := a.products.RenameModelGroup(ctx, actorFrom(ctx), req.ModelGroupId, req.Body.Name)
	if err != nil {
		return nil, err
	}
	return UpdateModelGroup200JSONResponse(toAPIModelGroup(group)), nil
}

// ListModels menjawab GET /api/v1/models?customer_id=...
func (a *apiServer) ListModels(ctx context.Context, req ListModelsRequestObject) (ListModelsResponseObject, error) {
	models, err := a.products.ListModels(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	items := make([]Model, 0, len(models))
	for _, m := range models {
		items = append(items, toAPIModel(m))
	}
	return ListModels200JSONResponse{Items: items}, nil
}

// CreateModel menjawab POST /api/v1/models.
func (a *apiServer) CreateModel(ctx context.Context, req CreateModelRequestObject) (CreateModelResponseObject, error) {
	in := product.NewModel{CustomerID: req.Body.CustomerId, Name: req.Body.Name, ModelGroupID: value(req.Body.ModelGroupId)}
	model, err := a.products.CreateModel(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateModel201JSONResponse(toAPIModel(model)), nil
}

// UpdateModel menjawab PUT /api/v1/models/{modelId}.
func (a *apiServer) UpdateModel(ctx context.Context, req UpdateModelRequestObject) (UpdateModelResponseObject, error) {
	in := product.ModelUpdate{Name: req.Body.Name, ModelGroupID: value(req.Body.ModelGroupId)}
	model, err := a.products.UpdateModel(ctx, actorFrom(ctx), req.ModelId, in)
	if err != nil {
		return nil, err
	}
	return UpdateModel200JSONResponse(toAPIModel(model)), nil
}

// ListPartNumbers menjawab GET /api/v1/part-numbers?customer_id=...
func (a *apiServer) ListPartNumbers(ctx context.Context, req ListPartNumbersRequestObject) (ListPartNumbersResponseObject, error) {
	parts, err := a.products.ListPartNumbers(ctx, actorFrom(ctx), req.Params.CustomerId)
	if err != nil {
		return nil, err
	}
	items := make([]PartNumber, 0, len(parts))
	for _, p := range parts {
		items = append(items, toAPIPartNumber(p))
	}
	return ListPartNumbers200JSONResponse{Items: items}, nil
}

// CreatePartNumber menjawab POST /api/v1/part-numbers.
func (a *apiServer) CreatePartNumber(ctx context.Context, req CreatePartNumberRequestObject) (CreatePartNumberResponseObject, error) {
	in := product.NewPartNumber{ModelID: req.Body.ModelId, Number: req.Body.Number, Name: value(req.Body.Name)}
	part, err := a.products.CreatePartNumber(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreatePartNumber201JSONResponse(toAPIPartNumber(part)), nil
}

// UpdatePartNumber menjawab PUT /api/v1/part-numbers/{partNumberId}.
func (a *apiServer) UpdatePartNumber(ctx context.Context, req UpdatePartNumberRequestObject) (UpdatePartNumberResponseObject, error) {
	in := product.PartNumberUpdate{Number: req.Body.Number, Name: value(req.Body.Name)}
	part, err := a.products.UpdatePartNumber(ctx, actorFrom(ctx), req.PartNumberId, in)
	if err != nil {
		return nil, err
	}
	return UpdatePartNumber200JSONResponse(toAPIPartNumber(part)), nil
}

// toAPIModelGroup mengubah ModelGroup domain menjadi bentuk kontrak API.
func toAPIModelGroup(g product.ModelGroup) ModelGroup {
	return ModelGroup{Id: g.ID, CustomerId: g.CustomerID, Name: g.Name, CreatedAt: g.CreatedAt}
}

// toAPIModel mengubah Model domain menjadi bentuk kontrak API. Group hanya
// ditulis bila model masuk group.
func toAPIModel(m product.Model) Model {
	out := Model{Id: m.ID, CustomerId: m.CustomerID, Name: m.Name, CreatedAt: m.CreatedAt}
	if m.ModelGroupID != "" {
		out.ModelGroup = &ModelGroupRef{Id: m.ModelGroupID, Name: m.ModelGroupName}
	}
	return out
}

// toAPIPartNumber mengubah PartNumber domain menjadi bentuk kontrak API.
func toAPIPartNumber(p product.PartNumber) PartNumber {
	return PartNumber{
		Id: p.ID, CustomerId: p.CustomerID, Model: ModelRef{Id: p.ModelID, Name: p.ModelName},
		Number: p.Number, Name: optional(p.Name), CreatedAt: p.CreatedAt,
	}
}
