package template

// Form set (langkah D, D-133, D-139, D-140).
//
// Satu form set adalah paket versi format template untuk tiga sheet: PFD,
// PFMEA, dan Control Plan. Dua set bawaan:
//
//	AIAG 4th set   PFD, PFMEA AIAG 4th (RPN), CP
//	AIAG-VDA set   PFD yang sama, PFMEA AIAG-VDA 2019 (AP), Control Plan AIAG 2024
//
// Setiap customer punya form set bawaan untuk dokumen baru; tanpa pilihan
// berarti set bawaan sistem (AIAG 4th). Dokumen yang sudah ada nanti tetap
// memakai set-nya sendiri (M4.1).
//
// Versi form set berjalan dari draft ke terbit seperti versi template (M3.4).
// Bedanya ada di siapa yang menerbitkan (D-139):
//   - admin (A-2) menyusun draft;
//   - pemegang hak V-3 (peran Penyetuju form, untuk saat ini PPAP Supervisor)
//     menerbitkan;
//   - penyetuju tidak boleh orang yang menyusun drafnya. Yang dibandingkan
//     orang, bukan akun, karena satu orang bisa punya akun admin dan akun kerja.
//
// Aturan yang sama berlaku untuk versi format template bermetode AP, karena
// tabel AP menentukan prioritas tindakan.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// SheetRef adalah satu sheet di versi form set: versi format template yang dipakai.
type SheetRef struct {
	FormatVersionID string
	TemplateName    string
	Version         int
	Status          string // draft atau published
}

// FormSetVersion adalah satu versi form set.
type FormSetVersion struct {
	ID          string
	Version     int
	Status      string
	PublishedAt *time.Time // nil selama draft; selalu UTC
	PFD         SheetRef
	PFMEA       SheetRef
	CP          SheetRef
}

// FormSet adalah satu form set beserta versinya.
type FormSet struct {
	ID       string
	Key      string // tetap, misalnya aiag_4th
	Name     string
	Default  bool             // set bawaan sistem untuk customer yang belum memilih
	Versions []FormSetVersion // menurut nomor versi, tidak pernah nil
}

// CustomerFormSet adalah pilihan form set satu customer. FormSetID kosong
// berarti customer itu memakai set bawaan sistem.
type CustomerFormSet struct {
	CustomerID   string
	CustomerCode string
	CustomerName string
	FormSetID    string
}

// FormSetList adalah isi layar Form sets.
type FormSetList struct {
	Sets     []FormSet
	Defaults []CustomerFormSet // menurut kode customer
}

// FormSetSheets adalah ID versi format untuk ketiga sheet.
type FormSetSheets struct {
	PFD   string
	PFMEA string
	CP    string
}

// NewFormSet adalah isian form set baru.
type NewFormSet struct {
	Name   string
	Sheets FormSetSheets
}

// errSamePerson adalah penolakan D-139: penyetuju sama dengan penyusun.
func errSamePerson() error {
	return &problem.ConflictError{Problems: []problem.Problem{{
		Field: "approver", Code: problem.NotAllowed, Message: "The person who prepared this draft cannot approve it. Ask another approver.",
	}}}
}

// ListFormSets mengembalikan semua form set dan pilihan setiap customer (L-2).
func (s *Service) ListFormSets(ctx context.Context, actor access.Actor) (FormSetList, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return FormSetList{}, err
	}
	sets, err := listFormSets(ctx, s.q)
	if err != nil {
		return FormSetList{}, err
	}
	rows, err := s.q.ListCustomerFormSets(ctx)
	if err != nil {
		return FormSetList{}, fmt.Errorf("mengambil form set customer: %w", err)
	}
	defaults := make([]CustomerFormSet, 0, len(rows))
	for _, r := range rows {
		defaults = append(defaults, CustomerFormSet{CustomerID: r.ID, CustomerCode: r.Code, CustomerName: r.Name, FormSetID: derefText(r.DefaultFormSetID)})
	}
	return FormSetList{Sets: sets, Defaults: defaults}, nil
}

// FormSetFor mengembalikan form set yang dipakai dokumen baru satu customer:
// pilihan customer itu, atau set bawaan sistem (L-2 untuk customer itu).
func (s *Service) FormSetFor(ctx context.Context, actor access.Actor, customerID string) (FormSet, error) {
	if !db.IsUUID(customerID) {
		return FormSet{}, problem.ErrNotFound
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{CustomerID: customerID}); err != nil {
		return FormSet{}, err
	}
	c, err := s.q.GetCustomerFormSet(ctx, customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return FormSet{}, problem.ErrNotFound
	}
	if err != nil {
		return FormSet{}, fmt.Errorf("membaca form set customer: %w", err)
	}
	if c.DefaultFormSetID != nil {
		return getFormSet(ctx, s.q, *c.DefaultFormSetID)
	}
	row, err := s.q.GetDefaultFormSet(ctx)
	if err != nil {
		return FormSet{}, fmt.Errorf("membaca form set bawaan: %w", err)
	}
	return getFormSet(ctx, s.q, row.ID)
}

// SetCustomerFormSet memilih form set bawaan satu customer (A-2). formSetID
// kosong berarti kembali ke set bawaan sistem.
func (s *Service) SetCustomerFormSet(ctx context.Context, actor access.Actor, customerID, formSetID string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return err
	}
	if !db.IsUUID(customerID) {
		return problem.ErrNotFound
	}
	missing := problems(&problem.Problem{Field: "form_set_id", Code: problem.InvalidFormat, Message: "Choose an existing form set."})
	if formSetID != "" && !db.IsUUID(formSetID) {
		return missing
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		c, err := q.GetCustomerFormSet(ctx, customerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		name := func(id *string) (string, error) {
			if id == nil {
				return "-", nil
			}
			fs, err := q.GetFormSet(ctx, *id)
			if errors.Is(err, pgx.ErrNoRows) {
				return "", missing
			}
			return fs.Name, err
		}
		before, err := name(c.DefaultFormSetID)
		if err != nil {
			return err
		}
		after, err := name(optional(formSetID))
		if err != nil {
			return err
		}
		if before == after {
			return nil
		}
		if err := q.SetCustomerFormSet(ctx, db.SetCustomerFormSetParams{ID: customerID, DefaultFormSetID: optional(formSetID)}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionCustomerFormSetChanged, TargetType: audit.TargetCustomer, TargetID: customerID,
			Details: map[string]string{"form_set": audit.Change(before, after)},
		})
	})
	return result(err, "menyimpan form set customer")
}

// CreateFormSet membuat form set baru dengan versi 1 berstatus draft (A-2).
func (s *Service) CreateFormSet(ctx context.Context, actor access.Actor, in NewFormSet) (FormSet, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormSet{}, err
	}
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	list := append([]*problem.Problem{nameProblem}, s.checkSheets(ctx, in.Sheets)...)
	if err := problems(list...); err != nil {
		return FormSet{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		keys, err := q.ListFormSetKeys(ctx)
		if err != nil {
			return err
		}
		if id, err = q.CreateFormSet(ctx, db.CreateFormSetParams{Key: keyFrom(name, keys), Name: name}); err != nil {
			return err
		}
		_, err = q.CreateFormSetVersion(ctx, db.CreateFormSetVersionParams{
			FormSetID: id, Version: 1, PfdFormatVersionID: in.Sheets.PFD, PfmeaFormatVersionID: in.Sheets.PFMEA,
			CpFormatVersionID: in.Sheets.CP, PreparedByPersonID: optional(actor.PersonID),
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormSetCreated, TargetType: audit.TargetFormSet, TargetID: id,
			Details: map[string]string{"name": name, "version": "1"},
		})
	})
	if err := result(err, "membuat form set"); err != nil {
		return FormSet{}, err
	}
	return getFormSet(ctx, s.q, id)
}

// SetFormSetSheets mengganti versi format ketiga sheet satu versi draft (A-2).
func (s *Service) SetFormSetSheets(ctx context.Context, actor access.Actor, versionID string, sheets FormSetSheets) (FormSet, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormSet{}, err
	}
	if !db.IsUUID(versionID) {
		return FormSet{}, problem.ErrNotFound
	}
	var setID string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		v, err := lockFormSetVersion(ctx, q, versionID)
		if err != nil {
			return err
		}
		setID = v.FormSetID
		if v.Status != StatusDraft {
			return statusConflict("A published version cannot be changed. Make a new version instead.")
		}
		if err := problems(s.checkSheets(ctx, sheets)...); err != nil {
			return err
		}
		if v.PfdFormatVersionID == sheets.PFD && v.PfmeaFormatVersionID == sheets.PFMEA && v.CpFormatVersionID == sheets.CP {
			return nil
		}
		err = q.SetFormSetSheets(ctx, db.SetFormSetSheetsParams{
			ID: versionID, PfdFormatVersionID: sheets.PFD, PfmeaFormatVersionID: sheets.PFMEA, CpFormatVersionID: sheets.CP,
			PreparedByPersonID: optional(actor.PersonID),
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormSetChanged, TargetType: audit.TargetFormSet, TargetID: v.FormSetID,
			Details: map[string]string{"version": strconv.Itoa(int(v.Version))},
		})
	})
	if err := result(err, "mengubah form set"); err != nil {
		return FormSet{}, err
	}
	return getFormSet(ctx, s.q, setID)
}

// NewFormSetVersion membuat versi draft berikutnya dari versi terbit terakhir (A-2).
func (s *Service) NewFormSetVersion(ctx context.Context, actor access.Actor, formSetID string) (FormSet, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormSet{}, err
	}
	if !db.IsUUID(formSetID) {
		return FormSet{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		last, err := q.LastFormSetVersion(ctx, formSetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if last.Status != StatusPublished {
			return statusConflict(draftExists)
		}
		_, err = q.CreateFormSetVersion(ctx, db.CreateFormSetVersionParams{
			FormSetID: formSetID, Version: last.Version + 1, PfdFormatVersionID: last.PfdFormatVersionID,
			PfmeaFormatVersionID: last.PfmeaFormatVersionID, CpFormatVersionID: last.CpFormatVersionID,
			PreparedByPersonID: optional(actor.PersonID),
		})
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormSetVersionCreated, TargetType: audit.TargetFormSet, TargetID: formSetID,
			Details: map[string]string{"version": strconv.Itoa(int(last.Version) + 1), "copied_from": strconv.Itoa(int(last.Version))},
		})
	})
	if err := result(err, "membuat versi form set"); err != nil {
		return FormSet{}, err
	}
	return getFormSet(ctx, s.q, formSetID)
}

// PublishFormSetVersion menerbitkan satu versi draft form set. Hanya
// pemegang V-3, bukan penyusun drafnya (D-139), dan hanya bila ketiga versi
// format di dalamnya sudah terbit, supaya isi set tidak berubah lagi.
func (s *Service) PublishFormSetVersion(ctx context.Context, actor access.Actor, versionID string) (FormSet, error) {
	if err := s.authz.Authorize(ctx, actor, access.ApproveForms, access.Resource{}); err != nil {
		return FormSet{}, err
	}
	if !db.IsUUID(versionID) {
		return FormSet{}, problem.ErrNotFound
	}
	var setID string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		v, err := lockFormSetVersion(ctx, q, versionID)
		if err != nil {
			return err
		}
		setID = v.FormSetID
		if v.Status != StatusDraft {
			return statusConflict(alreadyPublished)
		}
		if v.PreparedByPersonID != nil && *v.PreparedByPersonID == actor.PersonID {
			return errSamePerson()
		}
		for _, id := range []string{v.PfdFormatVersionID, v.PfmeaFormatVersionID, v.CpFormatVersionID} {
			sheet, err := q.GetFormatVersionSheet(ctx, id)
			if err != nil {
				return err
			}
			if sheet.Status != StatusPublished {
				return &problem.ConflictError{Problems: []problem.Problem{{
					Field: "sheets", Code: problem.NotAllowed,
					Message: "Publish the format template versions of this form set first: " + sheet.TemplateName + " is still a draft.",
				}}}
			}
		}
		if _, err := q.PublishFormSetVersion(ctx, versionID); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormSetVersionPublished, TargetType: audit.TargetFormSet, TargetID: v.FormSetID,
			Details: map[string]string{"version": strconv.Itoa(int(v.Version))},
		})
	})
	if err := result(err, "menerbitkan form set"); err != nil {
		return FormSet{}, err
	}
	return getFormSet(ctx, s.q, setID)
}

// lockFormSetVersion mengunci dan membaca satu versi form set.
func lockFormSetVersion(ctx context.Context, q *db.Queries, id string) (db.GetFormSetVersionRow, error) {
	if err := q.LockFormSetVersion(ctx, id); err != nil {
		return db.GetFormSetVersionRow{}, err
	}
	v, err := q.GetFormSetVersion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetFormSetVersionRow{}, problem.ErrNotFound
	}
	return v, err
}

// checkSheets memeriksa ketiga sheet: wajib diisi, versinya ada, dan jenis
// dokumennya cocok dengan sheet-nya.
func (s *Service) checkSheets(ctx context.Context, sheets FormSetSheets) []*problem.Problem {
	var list []*problem.Problem
	for _, sheet := range []struct{ field, id, documentType string }{
		{"pfd_format_version_id", sheets.PFD, "pfd"},
		{"pfmea_format_version_id", sheets.PFMEA, "pfmea"},
		{"cp_format_version_id", sheets.CP, "cp"},
	} {
		if sheet.id == "" {
			list = append(list, &problem.Problem{Field: sheet.field, Code: problem.Required, Message: "Choose the format template version of this sheet."})
			continue
		}
		missing := &problem.Problem{Field: sheet.field, Code: problem.InvalidFormat, Message: "Choose an existing format template version."}
		if !db.IsUUID(sheet.id) {
			list = append(list, missing)
			continue
		}
		row, err := s.q.GetFormatVersionSheet(ctx, sheet.id)
		switch {
		case err != nil:
			list = append(list, missing)
		case derefText(row.DocumentTypeKey) != sheet.documentType:
			list = append(list, &problem.Problem{Field: sheet.field, Code: problem.NotAllowed, Message: "Choose a format template of the document type of this sheet."})
		}
	}
	return list
}

// listFormSets membaca semua form set beserta versinya.
func listFormSets(ctx context.Context, q *db.Queries) ([]FormSet, error) {
	rows, err := q.ListFormSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil form set: %w", err)
	}
	versions, err := q.ListFormSetVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil versi form set: %w", err)
	}
	bySet := map[string][]FormSetVersion{}
	for _, v := range versions {
		bySet[v.FormSetID] = append(bySet[v.FormSetID], FormSetVersion{
			ID: v.ID, Version: int(v.Version), Status: v.Status, PublishedAt: utcTime(v.PublishedAt),
			PFD:   SheetRef{FormatVersionID: v.PfdFormatVersionID, TemplateName: v.PfdTemplateName, Version: int(v.PfdVersion), Status: v.PfdStatus},
			PFMEA: SheetRef{FormatVersionID: v.PfmeaFormatVersionID, TemplateName: v.PfmeaTemplateName, Version: int(v.PfmeaVersion), Status: v.PfmeaStatus},
			CP:    SheetRef{FormatVersionID: v.CpFormatVersionID, TemplateName: v.CpTemplateName, Version: int(v.CpVersion), Status: v.CpStatus},
		})
	}
	sets := make([]FormSet, 0, len(rows))
	for _, r := range rows {
		sets = append(sets, FormSet{ID: r.ID, Key: r.Key, Name: r.Name, Default: r.IsDefault, Versions: orNone(bySet[r.ID])})
	}
	return sets, nil
}

// getFormSet membaca satu form set beserta versinya.
func getFormSet(ctx context.Context, q *db.Queries, id string) (FormSet, error) {
	sets, err := listFormSets(ctx, q)
	if err != nil {
		return FormSet{}, err
	}
	for _, fs := range sets {
		if fs.ID == id {
			return fs, nil
		}
	}
	return FormSet{}, problem.ErrNotFound
}
