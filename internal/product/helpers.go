package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// getCustomer membaca customer pemilik master produk. ID yang tidak ada (atau
// bukan UUID) memberi problem.ErrNotFound.
func getCustomer(ctx context.Context, q *db.Queries, id string) (db.GetCustomerRow, error) {
	if !db.IsUUID(id) {
		return db.GetCustomerRow{}, problem.ErrNotFound
	}
	row, err := q.GetCustomer(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetCustomerRow{}, problem.ErrNotFound
	}
	if err != nil {
		return db.GetCustomerRow{}, fmt.Errorf("membaca customer: %w", err)
	}
	return row, nil
}

// customerForChange mencari customer untuk master baru. Hasilnya masalah input
// bila customer kosong, tidak ada, atau sudah dinonaktifkan (E-09).
func (s *Service) customerForChange(ctx context.Context, id string) (db.GetCustomerRow, *problem.Problem, error) {
	if id == "" {
		return db.GetCustomerRow{}, &problem.Problem{Field: "customer_id", Code: problem.Required, Message: "Choose the customer."}, nil
	}
	row, err := getCustomer(ctx, s.q, id)
	switch {
	case errors.Is(err, problem.ErrNotFound):
		return row, &problem.Problem{Field: "customer_id", Code: problem.InvalidFormat, Message: "This customer does not exist."}, nil
	case err != nil:
		return row, nil, err
	case !row.Active:
		return row, &problem.Problem{Field: "customer_id", Code: problem.NotAllowed, Message: "The customer is not active."}, nil
	}
	return row, nil, nil
}

// groupForModel mencari group model untuk sebuah model milik customerID. ID
// kosong berarti tanpa group. Group harus milik customer yang sama (D-42).
func (s *Service) groupForModel(ctx context.Context, q *db.Queries, id, customerID string) (ModelGroup, *problem.Problem, error) {
	if id == "" {
		return ModelGroup{}, nil, nil
	}
	group, err := getGroup(ctx, q, id)
	switch {
	case errors.Is(err, problem.ErrNotFound):
		return group, &problem.Problem{Field: "model_group_id", Code: problem.InvalidFormat, Message: "This model group does not exist."}, nil
	case err != nil:
		return group, nil, err
	case group.CustomerID != customerID:
		return group, &problem.Problem{Field: "model_group_id", Code: problem.NotAllowed,
			Message: "The model group belongs to another customer."}, nil
	}
	return group, nil, nil
}

// modelForPart mencari model untuk part number baru. Customer modelnya harus aktif.
func (s *Service) modelForPart(ctx context.Context, id string) (Model, *problem.Problem, error) {
	if id == "" {
		return Model{}, &problem.Problem{Field: "model_id", Code: problem.Required, Message: "Choose the model."}, nil
	}
	model, err := getModel(ctx, s.q, id)
	if errors.Is(err, problem.ErrNotFound) {
		return model, &problem.Problem{Field: "model_id", Code: problem.InvalidFormat, Message: "This model does not exist."}, nil
	}
	if err != nil {
		return model, nil, err
	}
	customer, err := getCustomer(ctx, s.q, model.CustomerID)
	if err != nil {
		return model, nil, err
	}
	if !customer.Active {
		return model, &problem.Problem{Field: "model_id", Code: problem.NotAllowed, Message: "The customer of this model is not active."}, nil
	}
	return model, nil, nil
}

// getGroup membaca satu group model.
func getGroup(ctx context.Context, q *db.Queries, id string) (ModelGroup, error) {
	if !db.IsUUID(id) {
		return ModelGroup{}, problem.ErrNotFound
	}
	row, err := q.GetModelGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelGroup{}, problem.ErrNotFound
	}
	if err != nil {
		return ModelGroup{}, fmt.Errorf("membaca group model: %w", err)
	}
	return fromGroupRow(row), nil
}

// getModel membaca satu model beserta nama group-nya.
func getModel(ctx context.Context, q *db.Queries, id string) (Model, error) {
	if !db.IsUUID(id) {
		return Model{}, problem.ErrNotFound
	}
	r, err := q.GetModel(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Model{}, problem.ErrNotFound
	}
	if err != nil {
		return Model{}, fmt.Errorf("membaca model: %w", err)
	}
	return Model{
		ID: r.ID, CustomerID: r.CustomerID, ModelGroupID: deref(r.ModelGroupID),
		ModelGroupName: deref(r.ModelGroupName), Name: r.Name, CreatedAt: r.CreatedAt.UTC(),
	}, nil
}

// getPartNumber membaca satu part number beserta nama modelnya.
func getPartNumber(ctx context.Context, q *db.Queries, id string) (PartNumber, error) {
	if !db.IsUUID(id) {
		return PartNumber{}, problem.ErrNotFound
	}
	r, err := q.GetPartNumber(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PartNumber{}, problem.ErrNotFound
	}
	if err != nil {
		return PartNumber{}, fmt.Errorf("membaca part number: %w", err)
	}
	return PartNumber{
		ID: r.ID, CustomerID: r.CustomerID, ModelID: r.ModelID, ModelName: r.ModelName,
		Number: r.Number, Name: deref(r.Name), CreatedAt: r.CreatedAt.UTC(),
	}, nil
}

// problems mengumpulkan masalah input yang ada menjadi satu ValidationError.
// Hasilnya nil bila semua masalah nil.
func problems(ps ...*problem.Problem) error {
	var list []problem.Problem
	for _, p := range ps {
		if p != nil {
			list = append(list, *p)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return &problem.ValidationError{Problems: list}
}

// result menerjemahkan error dari transaksi: nama atau nomor yang sudah
// dipakai menjadi ConflictError, penolakan input dan "tidak ada" diteruskan
// apa adanya, dan error lain dibungkus dengan keterangan doing.
func result(err error, doing string) error {
	taken := func(field, message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
	}
	var invalid *problem.ValidationError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "model_groups_name_key"):
		return taken("name", "This name is already used by another model group of this customer.")
	case db.IsUniqueViolation(err, "models_name_key"):
		return taken("name", "This name is already used by another model of this customer.")
	case db.IsUniqueViolation(err, "part_numbers_number_key"):
		return taken("number", "This part number is already used by this customer.")
	case errors.As(err, &invalid), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// fromGroupRow mengubah baris hasil sqlc menjadi ModelGroup.
func fromGroupRow(r db.ModelGroup) ModelGroup {
	return ModelGroup{ID: r.ID, CustomerID: r.CustomerID, Name: r.Name, CreatedAt: r.CreatedAt.UTC()}
}

// nullable mengubah teks kosong menjadi NULL di database.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// deref mengembalikan isi pointer teks, atau "" bila nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
