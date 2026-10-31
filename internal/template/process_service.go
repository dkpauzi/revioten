package template

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

// ProcessService menjalankan operasi process template (M3.2) terhadap
// database. Melihat memakai hak L-2 seperti master lain (D-103). Mengubah
// memakai E-6 milik Pengelola template, sama dengan master proses (D-64).
// Template umum berlaku untuk semua customer, jadi Resource-nya kosong.
type ProcessService struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewProcessService membuat ProcessService yang memakai pool database. authz
// memeriksa izin setiap operasi (M1.2).
func NewProcessService(pool *pgxpool.Pool, authz access.Authorizer) *ProcessService {
	return &ProcessService{pool: pool, q: db.New(pool), authz: authz}
}

// ListProcessTemplates mengembalikan semua process template beserta
// ringkasan versinya, urut menurut nama proses.
func (s *ProcessService) ListProcessTemplates(ctx context.Context, actor access.Actor) ([]ProcessTemplate, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListProcessTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil process template: %w", err)
	}
	versions, err := s.q.ListAllProcessVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil versi process template: %w", err)
	}
	byTemplate := make(map[string][]ProcessVersionSummary)
	for _, v := range versions {
		byTemplate[v.TemplateID] = append(byTemplate[v.TemplateID],
			ProcessVersionSummary{ID: v.ID, Version: int(v.Version), Status: v.Status})
	}
	list := make([]ProcessTemplate, 0, len(rows))
	for _, r := range rows {
		list = append(list, ProcessTemplate{
			ID: r.ID, Process: ProcessRef{ID: r.ProcessID, Name: r.ProcessName},
			Versions: orNone(byTemplate[r.ID]), CreatedAt: r.CreatedAt.UTC(),
		})
	}
	return list, nil
}

// GetProcessVersion mengembalikan satu versi process template lengkap.
func (s *ProcessService) GetProcessVersion(ctx context.Context, actor access.Actor, id string) (ProcessVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return ProcessVersion{}, err
	}
	if !db.IsUUID(id) {
		return ProcessVersion{}, problem.ErrNotFound
	}
	return getProcessVersion(ctx, s.q, id)
}

// CreateProcessTemplate membuat process template untuk satu proses master,
// beserta versi 1 yang masih kosong dan berstatus draft. Satu proses hanya
// punya satu template (ASUMSI E-72).
func (s *ProcessService) CreateProcessTemplate(ctx context.Context, actor access.Actor, processID string) (ProcessTemplate, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ProcessTemplate{}, err
	}
	process, p, err := s.checkProcess(ctx, processID)
	if err != nil {
		return ProcessTemplate{}, err
	}
	if err := problems(p); err != nil {
		return ProcessTemplate{}, err
	}

	var templateID string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if templateID, err = q.CreateProcessTemplate(ctx, process.ID); err != nil {
			return err
		}
		_, err = q.CreateProcessVersion(ctx, db.CreateProcessVersionParams{TemplateID: templateID, ProcessID: process.ID, Version: 1})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProcessTemplateCreated, TargetType: audit.TargetProcessTemplate, TargetID: templateID,
			Details: map[string]string{"process": process.Name, "version": "1"},
		})
	})
	if err := result(err, "menyimpan process template"); err != nil {
		return ProcessTemplate{}, err
	}
	return getProcessTemplate(ctx, s.q, templateID)
}

// AddSubprocess menambah sub-proses di akhir versi.
func (s *ProcessService) AddSubprocess(ctx context.Context, actor access.Actor, versionID, name string) (ProcessVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionSubprocessAdded, func(q *db.Queries, v ProcessVersion) (map[string]string, error) {
		name, p := problem.Text("name", name, true, maxSubprocessName)
		if p == nil && len(v.Subprocesses) >= maxSubprocesses {
			p = &problem.Problem{Field: "name", Code: problem.NotAllowed,
				Message: fmt.Sprintf("A version has at most %d sub-processes.", maxSubprocesses)}
		}
		if err := problems(p); err != nil {
			return nil, err
		}
		position := len(v.Subprocesses) + 1
		subprocessID, err := q.CreateSubprocess(ctx, v.Process.ID)
		if err != nil {
			return nil, err
		}
		err = q.AddVersionSubprocess(ctx, db.AddVersionSubprocessParams{
			VersionID: v.ID, ProcessID: v.Process.ID, SubprocessID: subprocessID, Name: name, Position: int16(position),
		})
		return map[string]string{"name": name, "position": strconv.Itoa(position)}, err
	})
}

// UpdateSubprocess mengganti nama, urutan, atau tersembunyi satu sub-proses.
// ID-nya tetap, jadi poin dan rujukan lain tetap utuh.
func (s *ProcessService) UpdateSubprocess(ctx context.Context, actor access.Actor, versionID, subprocessID string, in SubprocessUpdate) (ProcessVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionSubprocessChanged, func(q *db.Queries, v ProcessVersion) (map[string]string, error) {
		sub, ok := findSubprocess(v, subprocessID)
		if !ok {
			return nil, problem.ErrNotFound
		}
		old := itemChange{Text: sub.Name, Position: sub.Position, Hidden: sub.Hidden}
		change, list := cleanItemUpdate("name", in.Name, maxSubprocessName, in.Position, in.Hidden, old, len(v.Subprocesses))
		if err := problems(list...); err != nil {
			return nil, err
		}
		if change.Position != old.Position {
			// Sub-proses di antaranya bergeser satu tempat ke arah posisi lama.
			first, last, delta := moveRange(old.Position, change.Position)
			err := q.ShiftSubprocessPositions(ctx, db.ShiftSubprocessPositionsParams{
				VersionID: v.ID, Delta: delta, FirstPosition: first, LastPosition: last,
			})
			if err != nil {
				return nil, err
			}
			err = q.SetSubprocessPosition(ctx, db.SetSubprocessPositionParams{VersionID: v.ID, SubprocessID: sub.ID, Position: int16(change.Position)})
			if err != nil {
				return nil, err
			}
		}
		err := q.UpdateVersionSubprocess(ctx, db.UpdateVersionSubprocessParams{
			VersionID: v.ID, SubprocessID: sub.ID, Name: change.Text, Hidden: change.Hidden,
		})
		if err != nil {
			return nil, err
		}
		details := changeDetails("name", old, change)
		if len(details) > 0 {
			details["subprocess"] = sub.Name
		}
		return details, nil
	})
}

// AddPoint menambah poin di akhir daftar satu sub-proses untuk satu jenis
// dokumen.
func (s *ProcessService) AddPoint(ctx context.Context, actor access.Actor, versionID string, in NewPoint) (ProcessVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionTemplatePointAdded, func(q *db.Queries, v ProcessVersion) (map[string]string, error) {
		text, textProblem := problem.Text("text", in.Text, true, maxPointText)
		sub, found := findSubprocess(v, in.SubprocessID)
		var subProblem *problem.Problem
		if !found {
			subProblem = &problem.Problem{Field: "subprocess_id", Code: problem.InvalidFormat, Message: "Choose a sub-process of this version."}
		}
		typeProblem, err := checkDocumentType(ctx, q, in.DocumentTypeID, "Only PFD, PFMEA, and CP have process template points.")
		if err != nil {
			return nil, err
		}
		count := countPoints(sub, in.DocumentTypeID)
		if found && typeProblem == nil && count >= maxPointsPerList {
			subProblem = &problem.Problem{Field: "subprocess_id", Code: problem.NotAllowed,
				Message: fmt.Sprintf("A sub-process has at most %d points for one document type.", maxPointsPerList)}
		}
		if err := problems(textProblem, subProblem, typeProblem); err != nil {
			return nil, err
		}
		documentType, err := q.GetDocumentType(ctx, in.DocumentTypeID)
		if err != nil {
			return nil, err
		}
		pointID, err := q.CreateTemplatePoint(ctx, db.CreateTemplatePointParams{SubprocessID: sub.ID, DocumentTypeID: in.DocumentTypeID})
		if err != nil {
			return nil, err
		}
		position := count + 1
		err = q.AddVersionPoint(ctx, db.AddVersionPointParams{
			VersionID: v.ID, SubprocessID: sub.ID, DocumentTypeID: in.DocumentTypeID, PointID: pointID,
			Text: text, Position: int16(position),
		})
		return map[string]string{
			"subprocess": sub.Name, "document_type": documentType.Name, "text": text, "position": strconv.Itoa(position),
		}, err
	})
}

// UpdatePoint mengganti teks, urutan, atau tersembunyi satu poin. Sub-proses
// dan jenis dokumennya tetap.
func (s *ProcessService) UpdatePoint(ctx context.Context, actor access.Actor, versionID, pointID string, in PointUpdate) (ProcessVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionTemplatePointChanged, func(q *db.Queries, v ProcessVersion) (map[string]string, error) {
		sub, point, ok := findPoint(v, pointID)
		if !ok {
			return nil, problem.ErrNotFound
		}
		old := itemChange{Text: point.Text, Position: point.Position, Hidden: point.Hidden}
		count := countPoints(sub, point.DocumentType.ID)
		change, list := cleanItemUpdate("text", in.Text, maxPointText, in.Position, in.Hidden, old, count)
		if err := problems(list...); err != nil {
			return nil, err
		}
		if change.Position != old.Position {
			first, last, delta := moveRange(old.Position, change.Position)
			err := q.ShiftPointPositions(ctx, db.ShiftPointPositionsParams{
				VersionID: v.ID, SubprocessID: sub.ID, DocumentTypeID: point.DocumentType.ID,
				Delta: delta, FirstPosition: first, LastPosition: last,
			})
			if err != nil {
				return nil, err
			}
			err = q.SetPointPosition(ctx, db.SetPointPositionParams{VersionID: v.ID, PointID: point.ID, Position: int16(change.Position)})
			if err != nil {
				return nil, err
			}
		}
		err := q.UpdateVersionPoint(ctx, db.UpdateVersionPointParams{
			VersionID: v.ID, PointID: point.ID, Text: change.Text, Hidden: change.Hidden,
		})
		if err != nil {
			return nil, err
		}
		details := changeDetails("text", old, change)
		if len(details) > 0 {
			details["subprocess"] = sub.Name
			details["document_type"] = point.DocumentType.Name
			details["point"] = point.Text
		}
		return details, nil
	})
}

// editVersion menjalankan satu perubahan pada versi draft di dalam
// transaksi: hak E-6, versi dikunci, versi published ditolak, lalu entri
// audit bila ada yang berubah (details tidak kosong). Pola yang sama dengan
// format template.
func (s *ProcessService) editVersion(ctx context.Context, actor access.Actor, versionID, action string,
	change func(q *db.Queries, v ProcessVersion) (map[string]string, error)) (ProcessVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ProcessVersion{}, err
	}
	if !db.IsUUID(versionID) {
		return ProcessVersion{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockProcessVersion(ctx, versionID); err != nil {
			return err
		}
		v, err := getProcessVersion(ctx, q, versionID)
		if err != nil {
			return err
		}
		if v.Status != StatusDraft {
			return &problem.ConflictError{Problems: []problem.Problem{{
				Field: "status", Code: problem.NotAllowed, Message: "A published version cannot be changed. Make a new version instead.",
			}}}
		}
		details, err := change(q, v)
		if err != nil || len(details) == 0 {
			return err
		}
		details["version"] = strconv.Itoa(v.Version)
		return audit.Record(ctx, q, actor, audit.Event{
			Action: action, TargetType: audit.TargetProcessTemplate, TargetID: v.TemplateID, Details: details,
		})
	})
	if err := result(err, "mengubah process template"); err != nil {
		return ProcessVersion{}, err
	}
	return getProcessVersion(ctx, s.q, versionID)
}

// checkProcess memeriksa proses master untuk template baru.
func (s *ProcessService) checkProcess(ctx context.Context, id string) (ProcessRef, *problem.Problem, error) {
	if id == "" {
		return ProcessRef{}, &problem.Problem{Field: "process_id", Code: problem.Required, Message: "Choose a process."}, nil
	}
	missing := &problem.Problem{Field: "process_id", Code: problem.InvalidFormat, Message: "Choose an existing process."}
	if !db.IsUUID(id) {
		return ProcessRef{}, missing, nil
	}
	p, err := s.q.GetProcess(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ProcessRef{}, missing, nil
	case err != nil:
		return ProcessRef{}, nil, fmt.Errorf("membaca proses: %w", err)
	}
	return ProcessRef{ID: p.ID, Name: p.Name}, nil, nil
}

// getProcessTemplate membaca satu process template beserta versinya.
func getProcessTemplate(ctx context.Context, q *db.Queries, id string) (ProcessTemplate, error) {
	r, err := q.GetProcessTemplate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessTemplate{}, problem.ErrNotFound
	}
	if err != nil {
		return ProcessTemplate{}, fmt.Errorf("membaca process template: %w", err)
	}
	rows, err := q.ListProcessVersionsOfTemplate(ctx, id)
	if err != nil {
		return ProcessTemplate{}, fmt.Errorf("membaca versi process template: %w", err)
	}
	versions := make([]ProcessVersionSummary, 0, len(rows))
	for _, v := range rows {
		versions = append(versions, ProcessVersionSummary{ID: v.ID, Version: int(v.Version), Status: v.Status})
	}
	return ProcessTemplate{
		ID: r.ID, Process: ProcessRef{ID: r.ProcessID, Name: r.ProcessName}, Versions: versions, CreatedAt: r.CreatedAt.UTC(),
	}, nil
}

// getProcessVersion membaca satu versi beserta sub-proses dan poinnya.
func getProcessVersion(ctx context.Context, q *db.Queries, id string) (ProcessVersion, error) {
	r, err := q.GetProcessVersion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProcessVersion{}, problem.ErrNotFound
	}
	if err != nil {
		return ProcessVersion{}, fmt.Errorf("membaca versi process template: %w", err)
	}
	subRows, err := q.ListVersionSubprocesses(ctx, id)
	if err != nil {
		return ProcessVersion{}, fmt.Errorf("membaca sub-proses: %w", err)
	}
	pointRows, err := q.ListVersionPoints(ctx, id)
	if err != nil {
		return ProcessVersion{}, fmt.Errorf("membaca poin: %w", err)
	}
	// Poin sudah urut menurut sub-proses, jenis dokumen, lalu urutan poin.
	points := make(map[string][]Point)
	for _, p := range pointRows {
		points[p.SubprocessID] = append(points[p.SubprocessID], Point{
			ID:           p.PointID,
			DocumentType: DocumentTypeRef{ID: p.DocumentTypeID, Key: derefText(p.DocumentTypeKey), Name: p.DocumentTypeName},
			Text:         p.Text, Position: int(p.Position), Hidden: p.Hidden,
		})
	}
	subs := make([]Subprocess, 0, len(subRows))
	for _, sr := range subRows {
		subs = append(subs, Subprocess{
			ID: sr.SubprocessID, Name: sr.Name, Position: int(sr.Position), Hidden: sr.Hidden,
			Points: orNone(points[sr.SubprocessID]),
		})
	}
	return ProcessVersion{
		ID: r.ID, TemplateID: r.TemplateID, Process: ProcessRef{ID: r.ProcessID, Name: r.ProcessName},
		Version: int(r.Version), Status: r.Status, Subprocesses: subs, CreatedAt: r.CreatedAt.UTC(),
	}, nil
}

// moveRange menghitung rentang urutan yang bergeser saat satu butir pindah
// dari urutan from ke urutan to: butir di antaranya bergeser satu tempat ke
// arah urutan lama.
func moveRange(from, to int) (first, last, delta int16) {
	if to < from {
		return int16(to), int16(from - 1), 1
	}
	return int16(from + 1), int16(to), -1
}

// changeDetails menulis rincian audit perubahan sub-proses atau poin. field
// adalah nama isian teksnya ("name" atau "text").
func changeDetails(field string, old, change itemChange) map[string]string {
	details := map[string]string{}
	if old.Text != change.Text {
		details[field] = audit.Change(old.Text, change.Text)
	}
	if old.Position != change.Position {
		details["position"] = audit.Change(strconv.Itoa(old.Position), strconv.Itoa(change.Position))
	}
	if old.Hidden != change.Hidden {
		details["hidden"] = audit.Change(strconv.FormatBool(old.Hidden), strconv.FormatBool(change.Hidden))
	}
	return details
}

// orNone mengganti daftar nil dengan daftar kosong, supaya JSON-nya [] dan
// bukan null.
func orNone[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}
