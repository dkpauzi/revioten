package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/customer"
)

// ListCustomers menjawab GET /api/v1/customers.
func (a *apiServer) ListCustomers(ctx context.Context, _ ListCustomersRequestObject) (ListCustomersResponseObject, error) {
	list, err := a.customers.List(ctx, actorFrom(ctx))
	if err != nil {
		// Error dikembalikan ke strict handler, yang meneruskannya ke responseError (500).
		return nil, err
	}
	items := make([]Customer, 0, len(list))
	for _, c := range list {
		items = append(items, toAPICustomer(c))
	}
	return ListCustomers200JSONResponse{Items: items}, nil
}

// CreateCustomer menjawab POST /api/v1/customers. Input diteruskan apa adanya;
// merapikan dan memeriksa input adalah tugas paket customer.
func (a *apiServer) CreateCustomer(ctx context.Context, req CreateCustomerRequestObject) (CreateCustomerResponseObject, error) {
	in := customer.NewCustomer{Code: req.Body.Code, Name: req.Body.Name, ProjectGroupID: req.Body.ProjectGroupId}
	created, err := a.customers.Create(ctx, actorFrom(ctx), in)
	if err != nil {
		// Tanpa hak (403), input salah (422), atau kode ganda (409):
		// semuanya diterjemahkan responseError.
		return nil, err
	}
	return CreateCustomer201JSONResponse(toAPICustomer(created)), nil
}

// UpdateCustomer menjawab PUT /api/v1/customers/{customerId}.
func (a *apiServer) UpdateCustomer(ctx context.Context, req UpdateCustomerRequestObject) (UpdateCustomerResponseObject, error) {
	in := customer.CustomerUpdate{Name: req.Body.Name, ProjectGroupID: req.Body.ProjectGroupId, Active: req.Body.Active}
	updated, err := a.customers.Update(ctx, actorFrom(ctx), req.CustomerId, in)
	if err != nil {
		return nil, err
	}
	return UpdateCustomer200JSONResponse(toAPICustomer(updated)), nil
}

// ChangeCustomerCode menjawab POST /api/v1/customers/{customerId}/code.
func (a *apiServer) ChangeCustomerCode(ctx context.Context, req ChangeCustomerCodeRequestObject) (ChangeCustomerCodeResponseObject, error) {
	updated, err := a.customers.ChangeCode(ctx, actorFrom(ctx), req.CustomerId, req.Body.Code)
	if err != nil {
		return nil, err
	}
	return ChangeCustomerCode200JSONResponse(toAPICustomer(updated)), nil
}

// SetCustomerDefaultRoute menjawab PUT /api/v1/customers/{customerId}/default-route.
func (a *apiServer) SetCustomerDefaultRoute(ctx context.Context, req SetCustomerDefaultRouteRequestObject) (SetCustomerDefaultRouteResponseObject, error) {
	updated, err := a.customers.SetDefaultRoute(ctx, actorFrom(ctx), req.CustomerId, req.Body.RouteId)
	if err != nil {
		return nil, err
	}
	return SetCustomerDefaultRoute200JSONResponse(toAPICustomer(updated)), nil
}

// ListProjectGroups menjawab GET /api/v1/project-groups.
func (a *apiServer) ListProjectGroups(ctx context.Context, _ ListProjectGroupsRequestObject) (ListProjectGroupsResponseObject, error) {
	groups, err := a.customers.ListProjectGroups(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]ProjectGroup, 0, len(groups))
	for _, g := range groups {
		items = append(items, toAPIProjectGroup(g))
	}
	return ListProjectGroups200JSONResponse{Items: items}, nil
}

// CreateProjectGroup menjawab POST /api/v1/project-groups.
func (a *apiServer) CreateProjectGroup(ctx context.Context, req CreateProjectGroupRequestObject) (CreateProjectGroupResponseObject, error) {
	in := customer.ProjectGroupInput{Code: req.Body.Code, Name: req.Body.Name}
	group, err := a.customers.CreateProjectGroup(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreateProjectGroup201JSONResponse(toAPIProjectGroup(group)), nil
}

// UpdateProjectGroup menjawab PUT /api/v1/project-groups/{projectGroupId}.
func (a *apiServer) UpdateProjectGroup(ctx context.Context, req UpdateProjectGroupRequestObject) (UpdateProjectGroupResponseObject, error) {
	in := customer.ProjectGroupInput{Code: req.Body.Code, Name: req.Body.Name}
	group, err := a.customers.UpdateProjectGroup(ctx, actorFrom(ctx), req.ProjectGroupId, in)
	if err != nil {
		return nil, err
	}
	return UpdateProjectGroup200JSONResponse(toAPIProjectGroup(group)), nil
}

// toAPICustomer mengubah Customer domain menjadi Customer kontrak API.
func toAPICustomer(c customer.Customer) Customer {
	out := Customer{
		Id: c.ID, Code: c.Code, Name: c.Name, Active: c.Active, CreatedAt: c.CreatedAt,
		Codes: make([]CustomerCode, 0, len(c.Codes)),
	}
	if c.ProjectGroupID != "" {
		out.ProjectGroup = &ProjectGroupRef{Id: c.ProjectGroupID, Code: c.ProjectGroupCode}
	}
	if c.DefaultRouteID != "" {
		out.DefaultRoute = &ApprovalRouteRef{Id: c.DefaultRouteID, Name: c.DefaultRouteName}
	}
	for _, p := range c.Codes {
		out.Codes = append(out.Codes, CustomerCode{Code: p.Code, ValidFrom: p.ValidFrom, ValidUntil: p.ValidUntil})
	}
	return out
}

// toAPIProjectGroup mengubah ProjectGroup domain menjadi ProjectGroup kontrak API.
func toAPIProjectGroup(g customer.ProjectGroup) ProjectGroup {
	return ProjectGroup{Id: g.ID, Code: g.Code, Name: g.Name, CreatedAt: g.CreatedAt}
}
