package process

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi master proses terhadap database.
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

// Melihat master proses memakai hak L-2 seperti master lain (E-24, D-103), dan
// mengubahnya memakai E-6 milik Pengelola template (D-64). Master ini berlaku
// untuk semua customer (E-32, D-110), jadi Resource-nya kosong.

// List mengembalikan semua proses beserta aliasnya, urut menurut nama.
func (s *Service) List(ctx context.Context, actor access.Actor) ([]Process, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil proses: %w", err)
	}
	aliases, err := s.q.ListProcessAliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil alias proses: %w", err)
	}
	// Alias dikelompokkan per proses, dengan urutan dari database.
	byProcess := make(map[string][]string)
	for _, a := range aliases {
		byProcess[a.ProcessID] = append(byProcess[a.ProcessID], a.Name)
	}
	list := make([]Process, 0, len(rows))
	for _, r := range rows {
		list = append(list, Process{ID: r.ID, Name: r.Name, Aliases: orEmpty(byProcess[r.ID]), CreatedAt: r.CreatedAt.UTC()})
	}
	return list, nil
}

// Get mengembalikan satu proses menurut ID tetapnya.
func (s *Service) Get(ctx context.Context, actor access.Actor, id string) (Process, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Process{}, err
	}
	return get(ctx, s.q, id)
}

// Create menambah proses baru beserta nama tampilan dan aliasnya.
func (s *Service) Create(ctx context.Context, actor access.Actor, in Input) (Process, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return Process{}, err
	}
	name, aliases, err := clean(in)
	if err != nil {
		return Process{}, err
	}

	var created Process
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		id, err := q.CreateProcess(ctx)
		if err != nil {
			return err
		}
		err = q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: id, Kind: kindDisplay, Name: name})
		if err != nil {
			return taken("name", err)
		}
		if err := addAliases(ctx, q, id, aliases); err != nil {
			return err
		}
		// Dibaca ulang di dalam transaksi, supaya urutan alias sama dengan daftar.
		if created, err = get(ctx, q, id); err != nil {
			return err
		}
		details := map[string]string{"name": created.Name}
		if len(created.Aliases) > 0 {
			details["aliases"] = strings.Join(created.Aliases, ", ")
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProcessCreated, TargetType: audit.TargetProcess, TargetID: id, Details: details,
		})
	})
	return created, result(err, "menyimpan proses")
}

// Update mengganti nama tampilan dan seluruh alias satu proses. ID-nya tetap,
// jadi semua yang menunjuk proses ini ikut menampilkan nama baru (D-06).
func (s *Service) Update(ctx context.Context, actor access.Actor, id string, in Input) (Process, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return Process{}, err
	}
	name, aliases, err := clean(in)
	if err != nil {
		return Process{}, err
	}
	if !db.IsUUID(id) {
		return Process{}, problem.ErrNotFound
	}

	var updated Process
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockProcess(ctx, id); err != nil {
			return err
		}
		before, err := get(ctx, q, id)
		if err != nil {
			return err
		}
		// Urutannya penting: alias lama dilepas dulu, baru nama tampilan
		// diganti dan alias baru dipasang. Dengan begitu nama dan alias bisa
		// bertukar dalam satu perubahan.
		if err := q.DeleteProcessAliases(ctx, id); err != nil {
			return err
		}
		if err := q.RenameProcess(ctx, db.RenameProcessParams{ProcessID: id, Name: name}); err != nil {
			return taken("name", err)
		}
		if err := addAliases(ctx, q, id, aliases); err != nil {
			return err
		}
		if updated, err = get(ctx, q, id); err != nil {
			return err
		}

		details := map[string]string{}
		if before.Name != updated.Name {
			details["name"] = audit.Change(before.Name, updated.Name)
		}
		// Dibandingkan per alias, bukan sebagai teks gabungan, karena alias
		// sendiri bisa memuat koma.
		if !slices.Equal(before.Aliases, updated.Aliases) {
			details["aliases"] = audit.Change(strings.Join(before.Aliases, ", "), strings.Join(updated.Aliases, ", "))
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProcessUpdated, TargetType: audit.TargetProcess, TargetID: id, Details: details,
		})
	})
	return updated, result(err, "mengubah proses")
}

// addAliases memasang alias satu proses.
func addAliases(ctx context.Context, q *db.Queries, id string, aliases []string) error {
	for _, alias := range aliases {
		err := q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: id, Kind: kindAlias, Name: alias})
		if err != nil {
			return taken("aliases", err)
		}
	}
	return nil
}

// get membaca satu proses beserta aliasnya. ID yang tidak ada (atau bukan
// UUID) memberi problem.ErrNotFound.
func get(ctx context.Context, q *db.Queries, id string) (Process, error) {
	if !db.IsUUID(id) {
		return Process{}, problem.ErrNotFound
	}
	row, err := q.GetProcess(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Process{}, problem.ErrNotFound
	}
	if err != nil {
		return Process{}, fmt.Errorf("membaca proses: %w", err)
	}
	aliases, err := q.ListAliasesOfProcess(ctx, id)
	if err != nil {
		return Process{}, fmt.Errorf("membaca alias proses: %w", err)
	}
	return Process{ID: row.ID, Name: row.Name, Aliases: orEmpty(aliases), CreatedAt: row.CreatedAt.UTC()}, nil
}

// taken mengubah penolakan primary key process_names menjadi masalah input
// pada field itu: namanya sudah dipakai proses lain (sebagai nama atau alias),
// atau ditulis dua kali dalam satu permintaan. Error lain diteruskan apa adanya.
func taken(field string, err error) error {
	if !db.IsUniqueViolation(err, "process_names_pkey") {
		return err
	}
	message := "This name is already used by a process or an alias."
	if field == "aliases" {
		message = "An alias is already used by a process or an alias, or it is listed twice."
	}
	return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
}

// result meneruskan penolakan input dan "tidak ada" apa adanya, dan
// membungkus error lain dengan keterangan doing.
func result(err error, doing string) error {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &invalid), errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// orEmpty mengganti daftar nil dengan daftar kosong, supaya API selalu
// mengirim [] untuk proses tanpa alias.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
