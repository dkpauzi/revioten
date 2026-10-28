package change

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi master perubahan terhadap database.
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

// Melihat memakai hak L-2 seperti master lain (ASUMSI E-24), dan mengubah
// memakai A-2 (UC-05).

// ListChangeTypes mengembalikan jenis perubahan menurut urutan tetapnya.
func (s *Service) ListChangeTypes(ctx context.Context, actor access.Actor) ([]ChangeType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListChangeTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil jenis perubahan: %w", err)
	}
	list := make([]ChangeType, 0, len(rows))
	for _, r := range rows {
		list = append(list, ChangeType{
			ID: r.ID, Key: r.Key, Name: r.Name,
			ApprovalRouteID: deref(r.ApprovalRouteID), ApprovalRouteName: deref(r.ApprovalRouteName),
		})
	}
	return list, nil
}

// RenameChangeType mengganti nama satu jenis perubahan. Key-nya tetap.
func (s *Service) RenameChangeType(ctx context.Context, actor access.Actor, id, name string) (ChangeType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ChangeType{}, err
	}
	name, err := cleanName(name)
	if err != nil {
		return ChangeType{}, err
	}
	if !db.IsUUID(id) {
		return ChangeType{}, problem.ErrNotFound
	}

	var renamed ChangeType
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockChangeType(ctx, id); err != nil {
			return err
		}
		row, err := q.GetChangeType(ctx, id)
		if err != nil {
			return notFound(err)
		}
		renamed = ChangeType{
			ID: row.ID, Key: row.Key, Name: name,
			ApprovalRouteID: deref(row.ApprovalRouteID), ApprovalRouteName: deref(row.ApprovalRouteName),
		}
		if row.Name == name {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.RenameChangeType(ctx, db.RenameChangeTypeParams{ID: id, Name: name}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionChangeTypeUpdated, TargetType: audit.TargetChangeType, TargetID: id,
			Details: map[string]string{"name": audit.Change(row.Name, name)},
		})
	})
	return renamed, result(err, "mengganti nama jenis perubahan")
}

// SetApprovalRoute mengatur rute approval untuk satu jenis perubahan
// (R-TPL-10, M1.8). routeID kosong berarti rute default customer, sehingga
// semua jenis memakai alur yang sama sampai diatur lain (Q-15, ASUMSI E-35).
func (s *Service) SetApprovalRoute(ctx context.Context, actor access.Actor, id, routeID string) (ChangeType, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ChangeType{}, err
	}
	if !db.IsUUID(id) {
		return ChangeType{}, problem.ErrNotFound
	}

	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockChangeType(ctx, id); err != nil {
			return err
		}
		before, err := q.GetChangeType(ctx, id)
		if err != nil {
			return notFound(err)
		}
		var routeName string
		if routeID != "" {
			noRoute := &problem.ValidationError{Problems: []problem.Problem{{
				Field: "route_id", Code: problem.InvalidFormat, Message: "This approval route does not exist.",
			}}}
			if !db.IsUUID(routeID) {
				return noRoute
			}
			route, err := q.GetApprovalRoute(ctx, routeID)
			if errors.Is(err, pgx.ErrNoRows) {
				return noRoute
			}
			if err != nil {
				return err
			}
			routeName = route.Name
		}
		if deref(before.ApprovalRouteID) == routeID {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		var route *string
		if routeID != "" {
			route = &routeID
		}
		if err := q.SetChangeTypeRoute(ctx, db.SetChangeTypeRouteParams{ID: id, ApprovalRouteID: route}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionChangeTypeRouteChanged, TargetType: audit.TargetChangeType, TargetID: id,
			Details: map[string]string{"approval_route": audit.Change(deref(before.ApprovalRouteName), routeName)},
		})
	})
	if err := result(err, "mengatur rute jenis perubahan"); err != nil {
		return ChangeType{}, err
	}
	row, err := s.q.GetChangeType(ctx, id)
	if err != nil {
		return ChangeType{}, fmt.Errorf("membaca jenis perubahan: %w", err)
	}
	return ChangeType{
		ID: row.ID, Key: row.Key, Name: row.Name,
		ApprovalRouteID: deref(row.ApprovalRouteID), ApprovalRouteName: deref(row.ApprovalRouteName),
	}, nil
}

// ListChangeSources mengembalikan semua sumber perubahan, urut menurut nama.
// Sumber nonaktif ikut tampil, karena perubahan lama tetap menunjuknya.
func (s *Service) ListChangeSources(ctx context.Context, actor access.Actor) ([]ChangeSource, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListChangeSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil sumber perubahan: %w", err)
	}
	list := make([]ChangeSource, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromSource(r))
	}
	return list, nil
}

// CreateChangeSource menambah pemicu perubahan baru. Sumber baru tidak pernah
// punya mekanisme sistem.
func (s *Service) CreateChangeSource(ctx context.Context, actor access.Actor, name string) (ChangeSource, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ChangeSource{}, err
	}
	name, err := cleanName(name)
	if err != nil {
		return ChangeSource{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if id, err = q.CreateChangeSource(ctx, name); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionChangeSourceCreated, TargetType: audit.TargetChangeSource, TargetID: id,
			Details: map[string]string{"name": name},
		})
	})
	if err := result(err, "menyimpan sumber perubahan"); err != nil {
		return ChangeSource{}, err
	}
	return getSource(ctx, s.q, id)
}

// UpdateChangeSource mengganti nama sumber perubahan, dan menonaktifkan atau
// mengaktifkannya lagi. Sumber milik mekanisme sistem tidak bisa dinonaktifkan,
// karena sistem membuat perubahan dengan sumber itu (ASUMSI E-36).
func (s *Service) UpdateChangeSource(ctx context.Context, actor access.Actor, id string, in ChangeSourceUpdate) (ChangeSource, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return ChangeSource{}, err
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return ChangeSource{}, err
	}
	if !db.IsUUID(id) {
		return ChangeSource{}, problem.ErrNotFound
	}

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockChangeSource(ctx, id); err != nil {
			return err
		}
		before, err := getSource(ctx, q, id)
		if err != nil {
			return err
		}
		active := before.Active
		if in.Active != nil {
			active = *in.Active
		}
		if before.Mechanism != "" && !active {
			return &problem.ValidationError{Problems: []problem.Problem{{
				Field: "active", Code: problem.NotAllowed, Message: "This source is used by the system and stays active.",
			}}}
		}
		if err := q.UpdateChangeSource(ctx, db.UpdateChangeSourceParams{ID: id, Name: name, Active: active}); err != nil {
			return err
		}
		details := map[string]string{}
		if before.Name != name {
			details["name"] = audit.Change(before.Name, name)
		}
		if before.Active != active {
			details["active"] = audit.Change(strconv.FormatBool(before.Active), strconv.FormatBool(active))
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionChangeSourceUpdated, TargetType: audit.TargetChangeSource, TargetID: id, Details: details,
		})
	})
	if err := result(err, "mengubah sumber perubahan"); err != nil {
		return ChangeSource{}, err
	}
	return getSource(ctx, s.q, id)
}

// ListFourMCategories mengembalikan kategori 4M menurut urutan tetapnya.
func (s *Service) ListFourMCategories(ctx context.Context, actor access.Actor) ([]FourMCategory, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListFourMCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil kategori 4M: %w", err)
	}
	list := make([]FourMCategory, 0, len(rows))
	for _, r := range rows {
		list = append(list, FourMCategory(r))
	}
	return list, nil
}

// RenameFourMCategory mengganti nama satu kategori 4M. Key-nya tetap.
func (s *Service) RenameFourMCategory(ctx context.Context, actor access.Actor, id, name string) (FourMCategory, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FourMCategory{}, err
	}
	name, err := cleanName(name)
	if err != nil {
		return FourMCategory{}, err
	}
	if !db.IsUUID(id) {
		return FourMCategory{}, problem.ErrNotFound
	}

	var renamed FourMCategory
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockFourMCategory(ctx, id); err != nil {
			return err
		}
		row, err := q.GetFourMCategory(ctx, id)
		if err != nil {
			return notFound(err)
		}
		renamed = FourMCategory{ID: row.ID, Key: row.Key, Name: name}
		if row.Name == name {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.RenameFourMCategory(ctx, db.RenameFourMCategoryParams{ID: id, Name: name}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFourMCategoryUpdated, TargetType: audit.TargetFourMCategory, TargetID: id,
			Details: map[string]string{"name": audit.Change(row.Name, name)},
		})
	})
	return renamed, result(err, "mengganti nama kategori 4M")
}

// cleanName merapikan dan memeriksa nama dengan aturan teks master biasa.
func cleanName(value string) (string, error) {
	name, p := problem.Text("name", value, true, maxNameLength)
	if p != nil {
		return "", &problem.ValidationError{Problems: []problem.Problem{*p}}
	}
	return name, nil
}

// getSource membaca satu sumber perubahan.
func getSource(ctx context.Context, q *db.Queries, id string) (ChangeSource, error) {
	row, err := q.GetChangeSource(ctx, id)
	if err != nil {
		return ChangeSource{}, notFound(err)
	}
	return fromSource(row), nil
}

// fromSource mengubah baris hasil sqlc menjadi ChangeSource.
func fromSource(r db.ChangeSource) ChangeSource {
	mechanism := ""
	if r.Mechanism != nil {
		mechanism = *r.Mechanism
	}
	return ChangeSource{ID: r.ID, Name: r.Name, Mechanism: mechanism, Active: r.Active, CreatedAt: r.CreatedAt.UTC()}
}

// notFound mengubah "baris tidak ada" menjadi problem.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return problem.ErrNotFound
	}
	return err
}

// result menerjemahkan error dari transaksi: nama yang sudah dipakai menjadi
// ConflictError, penolakan input dan "tidak ada" diteruskan apa adanya, dan
// error lain dibungkus dengan keterangan doing.
func result(err error, doing string) error {
	taken := func(message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: "name", Code: problem.Taken, Message: message}}}
	}
	var invalid *problem.ValidationError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "change_types_name_key"):
		return taken("This name is already used by another change type.")
	case db.IsUniqueViolation(err, "change_sources_name_key"):
		return taken("This name is already used by another change source.")
	case db.IsUniqueViolation(err, "four_m_categories_name_key"):
		return taken("This name is already used by another 4M category.")
	case errors.As(err, &invalid), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// deref mengembalikan isi pointer teks, atau "" bila nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
