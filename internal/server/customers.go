package server

import (
	"context"
	"net/http"

	"github.com/dkpauzi/revioten/internal/customer"
)

// ListCustomers menjawab GET /api/v1/customers.
func (a *apiServer) ListCustomers(ctx context.Context, _ ListCustomersRequestObject) (ListCustomersResponseObject, error) {
	list, err := a.customers.List(ctx)
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
	created, err := a.customers.Create(ctx, customer.NewCustomer{Code: req.Body.Code, Name: req.Body.Name})
	if status, body, ok := problemResponse(ctx, err); ok {
		if status == http.StatusConflict {
			return CreateCustomer409JSONResponse(body), nil
		}
		return CreateCustomer422JSONResponse(body), nil
	}
	if err != nil {
		return nil, err
	}
	return CreateCustomer201JSONResponse(toAPICustomer(created)), nil
}

// toAPICustomer mengubah Customer domain menjadi Customer kontrak API.
func toAPICustomer(c customer.Customer) Customer {
	return Customer{Id: c.ID, Code: c.Code, Name: c.Name, CreatedAt: c.CreatedAt}
}
