package product

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi master produk terhadap database.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database. authz memeriksa izin
// setiap operasi (M1.2).
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz}
}

// Melihat master produk memakai hak L-2 (E-24, D-103), dan mengubahnya memakai
// E-11 (E-10). Customer dikirim ke Authorize, supaya cakupan customer bisa
// berlaku tanpa mengubah pemanggil (E-04).

// ListModelGroups mengembalikan group model satu customer, urut menurut nama.
func (s *Service) ListModelGroups(ctx context.Context, actor access.Actor, customerID string) ([]ModelGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	if _, err := getCustomer(ctx, s.q, customerID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListModelGroups(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("mengambil group model: %w", err)
	}
	list := make([]ModelGroup, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromGroupRow(r))
	}
	return list, nil
}

// CreateModelGroup menambah group model untuk satu customer yang aktif.
func (s *Service) CreateModelGroup(ctx context.Context, actor access.Actor, in NewModelGroup) (ModelGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return ModelGroup{}, err
	}
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	customer, customerProblem, err := s.customerForChange(ctx, in.CustomerID)
	if err != nil {
		return ModelGroup{}, err
	}
	if err := problems(nameProblem, customerProblem); err != nil {
		return ModelGroup{}, err
	}

	var group ModelGroup
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.CreateModelGroup(ctx, db.CreateModelGroupParams{CustomerID: customer.ID, Name: name})
		if err != nil {
			return err
		}
		group = fromGroupRow(row)
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionModelGroupCreated, TargetType: audit.TargetModelGroup, TargetID: row.ID,
			Details: map[string]string{"customer": customer.Code, "name": name},
		})
	})
	return group, result(err, "menyimpan group model")
}

// RenameModelGroup mengganti nama group model. Customer group tetap (D-42).
func (s *Service) RenameModelGroup(ctx context.Context, actor access.Actor, id, name string) (ModelGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ModelGroup{}, err
	}
	name, nameProblem := problem.Text("name", name, true, maxNameLength)
	if err := problems(nameProblem); err != nil {
		return ModelGroup{}, err
	}
	if !db.IsUUID(id) {
		return ModelGroup{}, problem.ErrNotFound
	}

	var group ModelGroup
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockModelGroup(ctx, id); err != nil {
			return err
		}
		before, err := getGroup(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.RenameModelGroup(ctx, db.RenameModelGroupParams{ID: before.ID, Name: name}); err != nil {
			return err
		}
		group = before
		group.Name = name
		if before.Name == name {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionModelGroupUpdated, TargetType: audit.TargetModelGroup, TargetID: before.ID,
			Details: map[string]string{"name": audit.Change(before.Name, name)},
		})
	})
	return group, result(err, "mengganti nama group model")
}

// ListModels mengembalikan model satu customer beserta group-nya, urut menurut nama.
func (s *Service) ListModels(ctx context.Context, actor access.Actor, customerID string) ([]Model, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	if _, err := getCustomer(ctx, s.q, customerID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListModels(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("mengambil model: %w", err)
	}
	list := make([]Model, 0, len(rows))
	for _, r := range rows {
		list = append(list, Model{
			ID: r.ID, CustomerID: r.CustomerID, ModelGroupID: deref(r.ModelGroupID),
			ModelGroupName: deref(r.ModelGroupName), Name: r.Name, CreatedAt: r.CreatedAt.UTC(), EOLAt: utc(r.EolAt),
		})
	}
	return list, nil
}

// CreateModel menambah model untuk satu customer yang aktif, dengan atau tanpa
// group model. Group yang dipilih harus milik customer yang sama (D-42).
func (s *Service) CreateModel(ctx context.Context, actor access.Actor, in NewModel) (Model, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Model{}, err
	}
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	customer, customerProblem, err := s.customerForChange(ctx, in.CustomerID)
	if err != nil {
		return Model{}, err
	}
	group, groupProblem, err := s.groupForModel(ctx, s.q, in.ModelGroupID, customer.ID)
	if err != nil {
		return Model{}, err
	}
	if err := problems(nameProblem, customerProblem, groupProblem); err != nil {
		return Model{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreateModel(ctx, db.CreateModelParams{CustomerID: customer.ID, ModelGroupID: nullable(group.ID), Name: name})
		if err != nil {
			return err
		}
		details := map[string]string{"customer": customer.Code, "name": name}
		if group.ID != "" {
			details["model_group"] = group.Name
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionModelCreated, TargetType: audit.TargetModel, TargetID: id, Details: details,
		})
	})
	if err := result(err, "menyimpan model"); err != nil {
		return Model{}, err
	}
	return getModel(ctx, s.q, id)
}

// UpdateModel mengganti nama model dan group-nya. ModelGroupID kosong berarti
// model dikeluarkan dari group. Customer model tetap.
func (s *Service) UpdateModel(ctx context.Context, actor access.Actor, id string, in ModelUpdate) (Model, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Model{}, err
	}
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	if err := problems(nameProblem); err != nil {
		return Model{}, err
	}
	if !db.IsUUID(id) {
		return Model{}, problem.ErrNotFound
	}

	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockModel(ctx, id); err != nil {
			return err
		}
		before, err := getModel(ctx, q, id)
		if err != nil {
			return err
		}
		group, groupProblem, err := s.groupForModel(ctx, q, in.ModelGroupID, before.CustomerID)
		if err != nil {
			return err
		}
		if err := problems(groupProblem); err != nil {
			return err
		}
		err = q.UpdateModel(ctx, db.UpdateModelParams{ID: before.ID, Name: name, ModelGroupID: nullable(group.ID)})
		if err != nil {
			return err
		}
		details := map[string]string{}
		if before.Name != name {
			details["name"] = audit.Change(before.Name, name)
		}
		if before.ModelGroupID != group.ID {
			details["model_group"] = audit.Change(before.ModelGroupName, group.Name)
		}
		if len(details) == 0 {
			return nil
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionModelUpdated, TargetType: audit.TargetModel, TargetID: before.ID, Details: details,
		})
	})
	if err := result(err, "mengubah model"); err != nil {
		return Model{}, err
	}
	return getModel(ctx, s.q, id)
}

// ListPartNumbers mengembalikan part number satu customer beserta modelnya.
func (s *Service) ListPartNumbers(ctx context.Context, actor access.Actor, customerID string) ([]PartNumber, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	if _, err := getCustomer(ctx, s.q, customerID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPartNumbers(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("mengambil part number: %w", err)
	}
	list := make([]PartNumber, 0, len(rows))
	for _, r := range rows {
		list = append(list, PartNumber{
			ID: r.ID, CustomerID: r.CustomerID, ModelID: r.ModelID, ModelName: r.ModelName,
			Number: r.Number, Name: deref(r.Name), CreatedAt: r.CreatedAt.UTC(),
		})
	}
	return list, nil
}

// CreatePartNumber menambah part number untuk satu model. Customer part number
// selalu customer modelnya, dan customer itu harus aktif.
func (s *Service) CreatePartNumber(ctx context.Context, actor access.Actor, in NewPartNumber) (PartNumber, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return PartNumber{}, err
	}
	number, numberProblem := problem.Text("number", in.Number, true, maxNumberLength)
	name, nameProblem := problem.Text("name", in.Name, false, maxNameLength)
	model, modelProblem, err := s.modelForPart(ctx, in.ModelID)
	if err != nil {
		return PartNumber{}, err
	}
	if err := problems(numberProblem, nameProblem, modelProblem); err != nil {
		return PartNumber{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreatePartNumber(ctx, db.CreatePartNumberParams{
			CustomerID: model.CustomerID, ModelID: model.ID, Number: number, Name: nullable(name),
		})
		if err != nil {
			return err
		}
		details := map[string]string{"model": model.Name, "number": number}
		if name != "" {
			details["name"] = name
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPartNumberCreated, TargetType: audit.TargetPartNumber, TargetID: id, Details: details,
		})
	})
	if err := result(err, "menyimpan part number"); err != nil {
		return PartNumber{}, err
	}
	return getPartNumber(ctx, s.q, id)
}

// UpdatePartNumber mengganti nomor dan nama part. Model part number tetap.
func (s *Service) UpdatePartNumber(ctx context.Context, actor access.Actor, id string, in PartNumberUpdate) (PartNumber, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return PartNumber{}, err
	}
	number, numberProblem := problem.Text("number", in.Number, true, maxNumberLength)
	name, nameProblem := problem.Text("name", in.Name, false, maxNameLength)
	if err := problems(numberProblem, nameProblem); err != nil {
		return PartNumber{}, err
	}
	if !db.IsUUID(id) {
		return PartNumber{}, problem.ErrNotFound
	}

	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockPartNumber(ctx, id); err != nil {
			return err
		}
		before, err := getPartNumber(ctx, q, id)
		if err != nil {
			return err
		}
		err = q.UpdatePartNumber(ctx, db.UpdatePartNumberParams{ID: before.ID, Number: number, Name: nullable(name)})
		if err != nil {
			return err
		}
		details := map[string]string{}
		if before.Number != number {
			details["number"] = audit.Change(before.Number, number)
		}
		if before.Name != name {
			details["name"] = audit.Change(before.Name, name)
		}
		if len(details) == 0 {
			return nil
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPartNumberUpdated, TargetType: audit.TargetPartNumber, TargetID: before.ID, Details: details,
		})
	})
	if err := result(err, "mengubah part number"); err != nil {
		return PartNumber{}, err
	}
	return getPartNumber(ctx, s.q, id)
}
