package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// ListProjectGroups mengembalikan semua Project Group, urut menurut kode
// (L-2, ASUMSI E-24).
func (s *Service) ListProjectGroups(ctx context.Context, actor access.Actor) ([]ProjectGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListProjectGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar Project Group: %w", err)
	}
	list := make([]ProjectGroup, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromGroupRow(r))
	}
	return list, nil
}

// CreateProjectGroup menambah Project Group (A-2, E-10).
func (s *Service) CreateProjectGroup(ctx context.Context, actor access.Actor, in ProjectGroupInput) (ProjectGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ProjectGroup{}, err
	}
	in, problems := validateProjectGroup(in)
	if len(problems) > 0 {
		return ProjectGroup{}, &problem.ValidationError{Problems: problems}
	}

	var group ProjectGroup
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		row, err := q.CreateProjectGroup(ctx, db.CreateProjectGroupParams{Code: in.Code, Name: in.Name})
		if err != nil {
			return err
		}
		group = fromGroupRow(row)
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProjectGroupCreated, TargetType: audit.TargetProjectGroup, TargetID: row.ID,
			Details: map[string]string{"code": row.Code, "name": row.Name},
		})
	})
	if conflict := groupCodeConflict(err); conflict != nil {
		return ProjectGroup{}, conflict
	}
	if err != nil {
		return ProjectGroup{}, fmt.Errorf("menyimpan Project Group: %w", err)
	}
	return group, nil
}

// UpdateProjectGroup mengubah kode dan nama Project Group (A-2). Customer yang
// dialokasikan menunjuk ID-nya, jadi ikut memakai kode yang baru.
func (s *Service) UpdateProjectGroup(ctx context.Context, actor access.Actor, id string, in ProjectGroupInput) (ProjectGroup, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ProjectGroup{}, err
	}
	in, problems := validateProjectGroup(in)
	if len(problems) > 0 {
		return ProjectGroup{}, &problem.ValidationError{Problems: problems}
	}
	if !db.IsUUID(id) {
		return ProjectGroup{}, problem.ErrNotFound
	}

	var group ProjectGroup
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		if err := q.LockProjectGroup(ctx, id); err != nil {
			return err
		}
		before, err := getProjectGroup(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.UpdateProjectGroup(ctx, db.UpdateProjectGroupParams{ID: before.ID, Code: in.Code, Name: in.Name}); err != nil {
			return err
		}
		group = ProjectGroup{ID: before.ID, Code: in.Code, Name: in.Name, CreatedAt: before.CreatedAt}
		details := map[string]string{}
		if before.Code != in.Code {
			details["code"] = audit.Change(before.Code, in.Code)
		}
		if before.Name != in.Name {
			details["name"] = audit.Change(before.Name, in.Name)
		}
		if len(details) == 0 {
			return nil
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProjectGroupUpdated, TargetType: audit.TargetProjectGroup, TargetID: before.ID, Details: details,
		})
	})
	if conflict := groupCodeConflict(err); conflict != nil {
		return ProjectGroup{}, conflict
	}
	if errors.Is(err, problem.ErrNotFound) {
		return ProjectGroup{}, err
	}
	if err != nil {
		return ProjectGroup{}, fmt.Errorf("mengubah Project Group: %w", err)
	}
	return group, nil
}

// groupCodeConflict menerjemahkan kode Project Group yang sudah dipakai.
func groupCodeConflict(err error) error {
	if db.IsUniqueViolation(err, "project_groups_code_key") {
		return &problem.ConflictError{Problems: []problem.Problem{{
			Field: "code", Code: problem.Taken, Message: "This code is already used by another project group.",
		}}}
	}
	return nil
}

// getProjectGroup membaca satu Project Group. ID yang tidak ada (atau bukan
// UUID) memberi problem.ErrNotFound.
func getProjectGroup(ctx context.Context, q *db.Queries, id string) (ProjectGroup, error) {
	if !db.IsUUID(id) {
		return ProjectGroup{}, problem.ErrNotFound
	}
	row, err := q.GetProjectGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProjectGroup{}, problem.ErrNotFound
	}
	if err != nil {
		return ProjectGroup{}, fmt.Errorf("membaca Project Group: %w", err)
	}
	return fromGroupRow(row), nil
}

// fromGroupRow mengubah baris hasil sqlc menjadi ProjectGroup.
func fromGroupRow(r db.ProjectGroup) ProjectGroup {
	return ProjectGroup{ID: r.ID, Code: r.Code, Name: r.Name, CreatedAt: r.CreatedAt.UTC()}
}
