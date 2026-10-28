package approval

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi jabatan dan rute approval terhadap database.
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

// Melihat memakai hak L-2 seperti master lain (ASUMSI E-24). Jabatan dan
// pemegangnya diubah lewat A-1 (UC-02: kelola pengguna, peran, dan jabatan),
// dan rute lewat A-2 (UC-06).

// ListPositions mengembalikan semua jabatan beserta pemegangnya, menurut
// urutan bagan organisasi.
func (s *Service) ListPositions(ctx context.Context, actor access.Actor) ([]Position, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil jabatan: %w", err)
	}
	holders, err := s.q.ListPositionHolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil pemegang jabatan: %w", err)
	}
	byPosition := make(map[string][]Holder)
	for _, h := range holders {
		byPosition[h.PositionID] = append(byPosition[h.PositionID], fromHolder(db.GetPositionHolderRow(h)))
	}
	list := make([]Position, 0, len(rows))
	for _, r := range rows {
		p := Position{ID: r.ID, Key: r.Key, Name: r.Name, PerProjectGroup: r.PerProjectGroup, Holders: byPosition[r.ID]}
		if p.Holders == nil {
			p.Holders = []Holder{}
		}
		list = append(list, p)
	}
	return list, nil
}

// RenamePosition mengganti nama jabatan, misalnya NPI Manager menjadi PE
// Manager (D-55). Key-nya tetap, jadi slot rute dan pemegangnya tidak berubah.
func (s *Service) RenamePosition(ctx context.Context, actor access.Actor, id, name string) (Position, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return Position{}, err
	}
	name, p := problem.Text("name", name, true, maxNameLength)
	if p != nil {
		return Position{}, &problem.ValidationError{Problems: []problem.Problem{*p}}
	}
	if !db.IsUUID(id) {
		return Position{}, problem.ErrNotFound
	}

	var renamed Position
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockPosition(ctx, id); err != nil {
			return err
		}
		row, err := q.GetPosition(ctx, id)
		if err != nil {
			return notFound(err)
		}
		renamed = Position{ID: row.ID, Key: row.Key, Name: name, PerProjectGroup: row.PerProjectGroup}
		if row.Name == name {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.RenamePosition(ctx, db.RenamePositionParams{ID: id, Name: name}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPositionUpdated, TargetType: audit.TargetPosition, TargetID: id,
			Details: map[string]string{"name": audit.Change(row.Name, name)},
		})
	})
	if err := result(err, "mengganti nama jabatan"); err != nil {
		return Position{}, err
	}
	return s.positionWithHolders(ctx, renamed)
}

// AddHolder mencatat satu orang sebagai pemegang jabatan. Untuk Project
// Leader dan Project Engineer, Project Group wajib diisi dan setiap group
// hanya punya satu pemegang (D-54). Satu orang boleh merangkap jabatan (D-58).
func (s *Service) AddHolder(ctx context.Context, actor access.Actor, positionID string, in NewHolder) (Holder, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return Holder{}, err
	}
	position, err := getPosition(ctx, s.q, positionID)
	if err != nil {
		return Holder{}, err
	}
	person, personProblem, err := s.personFor(ctx, in.PersonID)
	if err != nil {
		return Holder{}, err
	}
	groupID, groupCode, groupProblem, err := s.groupFor(ctx, position, in.ProjectGroupID)
	if err != nil {
		return Holder{}, err
	}
	if err := problems(personProblem, groupProblem); err != nil {
		return Holder{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.AddPositionHolder(ctx, db.AddPositionHolderParams{
			PositionID: position.ID, PerProjectGroup: position.PerProjectGroup, PersonID: person.ID, ProjectGroupID: nullable(groupID),
		})
		if err != nil {
			return err
		}
		details := map[string]string{"person": person.Name}
		if groupCode != "" {
			details["project_group"] = groupCode
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPositionHolderAdded, TargetType: audit.TargetPosition, TargetID: position.ID, Details: details,
		})
	})
	if err := result(err, "mencatat pemegang jabatan"); err != nil {
		return Holder{}, err
	}
	row, err := s.q.GetPositionHolder(ctx, id)
	if err != nil {
		return Holder{}, fmt.Errorf("membaca pemegang jabatan: %w", err)
	}
	return fromHolder(row), nil
}

// RemoveHolder melepas satu pemegang jabatan. Riwayatnya tetap ada di audit log.
func (s *Service) RemoveHolder(ctx context.Context, actor access.Actor, positionID, holderID string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return err
	}
	if !db.IsUUID(positionID) || !db.IsUUID(holderID) {
		return problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		h, err := q.GetPositionHolder(ctx, holderID)
		if err != nil {
			return notFound(err)
		}
		if h.PositionID != positionID {
			return problem.ErrNotFound
		}
		if err := q.DeletePositionHolder(ctx, holderID); err != nil {
			return err
		}
		details := map[string]string{"person": h.PersonName}
		if h.ProjectGroupCode != nil {
			details["project_group"] = *h.ProjectGroupCode
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPositionHolderRemoved, TargetType: audit.TargetPosition, TargetID: positionID, Details: details,
		})
	})
	return result(err, "melepas pemegang jabatan")
}

// ListRoutes mengembalikan semua rute approval beserta slotnya, urut menurut nama.
func (s *Service) ListRoutes(ctx context.Context, actor access.Actor) ([]Route, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListApprovalRoutes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil rute approval: %w", err)
	}
	slots, err := listSlots(ctx, s.q, nil)
	if err != nil {
		return nil, err
	}
	list := make([]Route, 0, len(rows))
	for _, r := range rows {
		list = append(list, Route{ID: r.ID, Name: r.Name, Slots: slots[r.ID], CreatedAt: r.CreatedAt.UTC()})
	}
	return list, nil
}

// CreateRoute menambah rute approval baru (UC-06).
func (s *Service) CreateRoute(ctx context.Context, actor access.Actor, in RouteInput) (Route, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Route{}, err
	}
	in, err := s.clean(ctx, in)
	if err != nil {
		return Route{}, err
	}

	var created Route
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		id, err := q.CreateApprovalRoute(ctx, in.Name)
		if err != nil {
			return err
		}
		if err := saveSlots(ctx, q, id, in.Slots); err != nil {
			return err
		}
		if created, err = getRoute(ctx, q, id); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionApprovalRouteCreated, TargetType: audit.TargetApprovalRoute, TargetID: id,
			Details: map[string]string{"name": created.Name, "slots": describeSlots(created.Slots)},
		})
	})
	return created, result(err, "menyimpan rute approval")
}

// UpdateRoute mengganti nama dan semua slot satu rute. Contohnya slot GM
// dipindah dari Advisor ke Factory Manager (D-56, R-APR-15). Revisi yang
// sudah dicetak nanti menyimpan salinan rutenya sendiri (R-APR-11), jadi
// tidak ikut berubah.
func (s *Service) UpdateRoute(ctx context.Context, actor access.Actor, id string, in RouteInput) (Route, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Route{}, err
	}
	in, err := s.clean(ctx, in)
	if err != nil {
		return Route{}, err
	}
	if !db.IsUUID(id) {
		return Route{}, problem.ErrNotFound
	}

	var updated Route
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockApprovalRoute(ctx, id); err != nil {
			return err
		}
		before, err := getRoute(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.RenameApprovalRoute(ctx, db.RenameApprovalRouteParams{ID: id, Name: in.Name}); err != nil {
			return err
		}
		if err := q.DeleteRouteSlots(ctx, id); err != nil {
			return err
		}
		if err := saveSlots(ctx, q, id, in.Slots); err != nil {
			return err
		}
		if updated, err = getRoute(ctx, q, id); err != nil {
			return err
		}
		details := map[string]string{}
		if before.Name != updated.Name {
			details["name"] = audit.Change(before.Name, updated.Name)
		}
		if b, a := describeSlots(before.Slots), describeSlots(updated.Slots); b != a {
			details["slots"] = audit.Change(b, a)
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionApprovalRouteUpdated, TargetType: audit.TargetApprovalRoute, TargetID: id, Details: details,
		})
	})
	return updated, result(err, "mengubah rute approval")
}

// clean memeriksa isian rute terhadap jabatan yang ada di database.
func (s *Service) clean(ctx context.Context, in RouteInput) (RouteInput, error) {
	rows, err := s.q.ListPositions(ctx)
	if err != nil {
		return RouteInput{}, fmt.Errorf("mengambil jabatan: %w", err)
	}
	known := make(map[string]string, len(rows))
	for _, r := range rows {
		known[r.ID] = r.Name
	}
	return cleanRoute(in, known)
}

// saveSlots menyimpan slot rute menurut urutannya, beserta jabatan setiap slot.
func saveSlots(ctx context.Context, q *db.Queries, routeID string, slots []SlotInput) error {
	for i, s := range slots {
		slotID, err := q.AddRouteSlot(ctx, db.AddRouteSlotParams{
			RouteID: routeID, SlotOrder: int16(i + 1), Stage: s.Stage, PrintTitle: s.PrintTitle,
		})
		if err != nil {
			return err
		}
		for _, positionID := range s.PositionIDs {
			if err := q.AddRouteSlotPosition(ctx, db.AddRouteSlotPositionParams{SlotID: slotID, PositionID: positionID}); err != nil {
				return err
			}
		}
	}
	return nil
}

// getRoute membaca satu rute beserta slotnya.
func getRoute(ctx context.Context, q *db.Queries, id string) (Route, error) {
	r, err := q.GetApprovalRoute(ctx, id)
	if err != nil {
		return Route{}, notFound(err)
	}
	slots, err := listSlots(ctx, q, &id)
	if err != nil {
		return Route{}, err
	}
	return Route{ID: r.ID, Name: r.Name, Slots: slots[r.ID], CreatedAt: r.CreatedAt.UTC()}, nil
}

// listSlots membaca slot satu rute (routeID) atau semua rute (nil), dikelompokkan per rute.
func listSlots(ctx context.Context, q *db.Queries, routeID *string) (map[string][]Slot, error) {
	rows, err := q.ListRouteSlots(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("mengambil slot rute: %w", err)
	}
	positions, err := q.ListRouteSlotPositions(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("mengambil jabatan slot: %w", err)
	}
	bySlot := make(map[string][]PositionRef)
	for _, p := range positions {
		bySlot[p.SlotID] = append(bySlot[p.SlotID], PositionRef{ID: p.PositionID, Name: p.PositionName})
	}
	byRoute := make(map[string][]Slot)
	for _, r := range rows {
		byRoute[r.RouteID] = append(byRoute[r.RouteID], Slot{Stage: r.Stage, PrintTitle: r.PrintTitle, Positions: bySlot[r.ID]})
	}
	return byRoute, nil
}

// personFor mencari orang yang akan memegang jabatan.
func (s *Service) personFor(ctx context.Context, id string) (db.Person, *problem.Problem, error) {
	if id == "" {
		return db.Person{}, &problem.Problem{Field: "person_id", Code: problem.Required, Message: "Choose the person."}, nil
	}
	if !db.IsUUID(id) {
		return db.Person{}, &problem.Problem{Field: "person_id", Code: problem.InvalidFormat, Message: "This person does not exist."}, nil
	}
	person, err := s.q.GetPerson(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Person{}, &problem.Problem{Field: "person_id", Code: problem.InvalidFormat, Message: "This person does not exist."}, nil
	}
	if err != nil {
		return db.Person{}, nil, fmt.Errorf("membaca orang: %w", err)
	}
	return person, nil, nil
}

// groupFor memeriksa Project Group pemegang: wajib untuk jabatan per Project
// Group, dan harus kosong untuk jabatan lain.
func (s *Service) groupFor(ctx context.Context, position Position, id string) (string, string, *problem.Problem, error) {
	switch {
	case !position.PerProjectGroup && id != "":
		return "", "", &problem.Problem{Field: "project_group_id", Code: problem.NotAllowed,
			Message: "This position is not held per project group."}, nil
	case !position.PerProjectGroup:
		return "", "", nil, nil
	case id == "":
		return "", "", &problem.Problem{Field: "project_group_id", Code: problem.Required, Message: "Choose the project group."}, nil
	}
	invalid := &problem.Problem{Field: "project_group_id", Code: problem.InvalidFormat, Message: "This project group does not exist."}
	if !db.IsUUID(id) {
		return "", "", invalid, nil
	}
	group, err := s.q.GetProjectGroup(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", invalid, nil
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("membaca Project Group: %w", err)
	}
	return group.ID, group.Code, nil, nil
}

// positionWithHolders melengkapi jabatan dengan pemegangnya.
func (s *Service) positionWithHolders(ctx context.Context, p Position) (Position, error) {
	holders, err := s.q.ListPositionHolders(ctx)
	if err != nil {
		return Position{}, fmt.Errorf("mengambil pemegang jabatan: %w", err)
	}
	p.Holders = []Holder{}
	for _, h := range holders {
		if h.PositionID == p.ID {
			p.Holders = append(p.Holders, fromHolder(db.GetPositionHolderRow(h)))
		}
	}
	return p, nil
}

// getPosition membaca satu jabatan.
func getPosition(ctx context.Context, q *db.Queries, id string) (Position, error) {
	if !db.IsUUID(id) {
		return Position{}, problem.ErrNotFound
	}
	r, err := q.GetPosition(ctx, id)
	if err != nil {
		return Position{}, notFound(err)
	}
	return Position{ID: r.ID, Key: r.Key, Name: r.Name, PerProjectGroup: r.PerProjectGroup}, nil
}

// fromHolder mengubah baris hasil sqlc menjadi Holder.
func fromHolder(r db.GetPositionHolderRow) Holder {
	return Holder{
		ID: r.ID, PersonID: r.PersonID, PersonName: r.PersonName,
		ProjectGroupID: deref(r.ProjectGroupID), ProjectGroupCode: deref(r.ProjectGroupCode), CreatedAt: r.CreatedAt.UTC(),
	}
}

// problems mengumpulkan masalah input yang ada menjadi satu ValidationError.
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

// notFound mengubah "baris tidak ada" menjadi problem.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return problem.ErrNotFound
	}
	return err
}

// result menerjemahkan error dari transaksi: nama atau pemegang yang sudah
// ada menjadi ConflictError, penolakan input dan "tidak ada" diteruskan apa
// adanya, dan error lain dibungkus dengan keterangan doing.
func result(err error, doing string) error {
	taken := func(field, message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
	}
	var invalid *problem.ValidationError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "positions_name_key"):
		return taken("name", "This name is already used by another position.")
	case db.IsUniqueViolation(err, "approval_routes_name_key"):
		return taken("name", "This name is already used by another approval route.")
	case db.IsUniqueViolation(err, "position_holders_one_per_group"):
		return taken("project_group_id", "This project group already has a holder for this position.")
	case db.IsUniqueViolation(err, "position_holders_unique"):
		return taken("person_id", "This person already holds this position.")
	case errors.As(err, &invalid), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
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
