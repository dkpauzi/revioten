package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan operasi format template terhadap database.
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

// Melihat memakai hak L-2 seperti master lain (D-103). Format template diubah
// lewat A-2 (UC-20, U3: kelola master data dan format template).

// ListFormatTemplates mengembalikan semua format template beserta ringkasan
// versinya, menurut urutan jenis dokumen lalu nama.
func (s *Service) ListFormatTemplates(ctx context.Context, actor access.Actor) ([]FormatTemplate, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListFormatTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil format template: %w", err)
	}
	versions, err := s.q.ListFormatVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil versi format template: %w", err)
	}
	byTemplate := map[string][]VersionSummary{}
	for _, v := range versions {
		byTemplate[v.TemplateID] = append(byTemplate[v.TemplateID], summary(v))
	}
	list := make([]FormatTemplate, 0, len(rows))
	for _, r := range rows {
		list = append(list, fromTemplate(db.GetFormatTemplateRow(r), byTemplate[r.ID]))
	}
	return list, nil
}

// GetFormatVersion mengembalikan satu versi lengkap: field, tabel skala, dan simbol.
func (s *Service) GetFormatVersion(ctx context.Context, actor access.Actor, id string) (FormatVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return FormatVersion{}, err
	}
	if !db.IsUUID(id) {
		return FormatVersion{}, problem.ErrNotFound
	}
	return getVersion(ctx, s.q, id)
}

// CreateFormatTemplate membuat format template baru dengan versi 1 berstatus
// draft: kosong, atau salinan versi lain dengan ID field baru.
func (s *Service) CreateFormatTemplate(ctx context.Context, actor access.Actor, in NewTemplate) (FormatTemplate, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormatTemplate{}, err
	}
	in, source, err := s.cleanNewTemplate(ctx, in)
	if err != nil {
		return FormatTemplate{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		id, err = q.CreateFormatTemplate(ctx, db.CreateFormatTemplateParams{
			Name: in.Name, DocumentTypeID: in.DocumentTypeID, Methodology: optional(in.Methodology), CustomerID: optional(in.CustomerID),
		})
		if err != nil {
			return err
		}
		details := map[string]string{"name": in.Name}
		if source != nil {
			err = copyVersion(ctx, q, id, *source)
			details["copied_from"] = in.CopyFromVersionID
		} else {
			err = emptyVersion(ctx, q, id, in.RiskMethod)
		}
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormatTemplateCreated, TargetType: audit.TargetFormatTemplate, TargetID: id, Details: details,
		})
	})
	if err := result(err, "membuat format template"); err != nil {
		return FormatTemplate{}, err
	}
	return getTemplate(ctx, s.q, id)
}

// RenameFormatTemplate mengganti nama format template. ID dan isinya tetap.
func (s *Service) RenameFormatTemplate(ctx context.Context, actor access.Actor, id, name string) (FormatTemplate, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormatTemplate{}, err
	}
	name, p := problem.Text("name", name, true, maxNameLength)
	if err := problems(p); err != nil {
		return FormatTemplate{}, err
	}
	if !db.IsUUID(id) {
		return FormatTemplate{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockFormatTemplate(ctx, id); err != nil {
			return err
		}
		before, err := getTemplate(ctx, q, id)
		if err != nil {
			return err
		}
		if before.Name == name {
			return nil
		}
		if err := q.RenameFormatTemplate(ctx, db.RenameFormatTemplateParams{ID: id, Name: name}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormatTemplateRenamed, TargetType: audit.TargetFormatTemplate, TargetID: id,
			Details: map[string]string{"name": audit.Change(before.Name, name)},
		})
	})
	if err := result(err, "mengganti nama format template"); err != nil {
		return FormatTemplate{}, err
	}
	return getTemplate(ctx, s.q, id)
}

// AddField menambah field header atau kolom baris di akhir areanya.
func (s *Service) AddField(ctx context.Context, actor access.Actor, versionID string, in NewField) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatFieldAdded, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		in, list := cleanNewField(v, in)
		if err := problems(list...); err != nil {
			return nil, err
		}
		position := 1
		for _, f := range v.Fields {
			if f.Area == in.Area {
				position = max(position, f.Position+1)
			}
		}
		fieldID, err := q.CreateFormatField(ctx, db.CreateFormatFieldParams{
			TemplateID: v.TemplateID, Area: in.Area, Kind: in.Kind,
			RatingScale: optional(in.RatingScale), RatingSet: optionalSmall(in.RatingSet),
		})
		if err != nil {
			return nil, err
		}
		err = q.AddVersionField(ctx, db.AddVersionFieldParams{
			VersionID: v.ID, FieldID: fieldID, TemplateID: v.TemplateID, Area: in.Area, Label: in.Label,
			Position: int16(position), Required: in.Required, Options: optionsJSON(in.Options),
		})
		return map[string]string{"label": in.Label, "area": in.Area, "kind": in.Kind}, err
	})
}

// UpdateField mengubah label, wajib, tersembunyi, urutan, dan pilihan satu
// field. Area dan jenisnya tetap, jadi isi baris yang sudah ada tetap sah.
func (s *Service) UpdateField(ctx context.Context, actor access.Actor, versionID, fieldID string, in FieldUpdate) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatFieldChanged, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		i := slices.IndexFunc(v.Fields, func(f Field) bool { return f.ID == fieldID })
		if i < 0 {
			return nil, problem.ErrNotFound
		}
		f := v.Fields[i]
		change, list := cleanFieldUpdate(v, f, in)
		if err := problems(list...); err != nil {
			return nil, err
		}
		if change.Position != f.Position {
			// Field di antaranya bergeser satu tempat ke arah posisi lama.
			first, last, delta := moveRange(f.Position, change.Position)
			shift := db.ShiftFieldPositionsParams{VersionID: v.ID, Area: f.Area, Delta: delta, FirstPosition: first, LastPosition: last}
			if err := q.ShiftFieldPositions(ctx, shift); err != nil {
				return nil, err
			}
			if err := q.SetFieldPosition(ctx, db.SetFieldPositionParams{VersionID: v.ID, FieldID: f.ID, Position: int16(change.Position)}); err != nil {
				return nil, err
			}
		}
		err := q.UpdateVersionField(ctx, db.UpdateVersionFieldParams{
			VersionID: v.ID, FieldID: f.ID, Label: change.Label, Required: change.Required, Hidden: change.Hidden, Options: optionsJSON(change.Options),
		})
		if err != nil {
			return nil, err
		}
		details := map[string]string{}
		note := func(key, before, after string) {
			if before != after {
				details[key] = audit.Change(before, after)
			}
		}
		note("label", f.Label, change.Label)
		note("required", strconv.FormatBool(f.Required), strconv.FormatBool(change.Required))
		note("hidden", strconv.FormatBool(f.Hidden), strconv.FormatBool(change.Hidden))
		note("position", strconv.Itoa(f.Position), strconv.Itoa(change.Position))
		note("options", optionLabels(f.Options), optionLabels(change.Options))
		if len(details) > 0 {
			details["field"] = f.Label
		}
		return details, nil
	})
}

// SetRatingCriteria mengganti teks kriteria tabel skala.
func (s *Service) SetRatingCriteria(ctx context.Context, actor access.Actor, versionID string, ratings []Rating) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatRatingsChanged, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		ratings, p := cleanRatings(v, ratings)
		if err := problems(p); err != nil {
			return nil, err
		}
		var changed []string
		for _, r := range ratings {
			i := slices.IndexFunc(v.Ratings, func(old Rating) bool { return old.Scale == r.Scale && old.Value == r.Value })
			if i < 0 || v.Ratings[i].Criteria == r.Criteria {
				continue
			}
			err := q.SetRatingCriterion(ctx, db.SetRatingCriterionParams{VersionID: v.ID, Scale: r.Scale, Value: int16(r.Value), Criteria: r.Criteria})
			if err != nil {
				return nil, err
			}
			changed = append(changed, fmt.Sprintf("%s %d", r.Scale, r.Value))
		}
		if len(changed) == 0 {
			return nil, nil
		}
		return map[string]string{"changed": strings.Join(changed, ", ")}, nil
	})
}

// SaveSymbols mengganti nama simbol dan menambah simbol baru. Simbol tidak
// bisa dihapus, karena isi baris bisa memakainya.
func (s *Service) SaveSymbols(ctx context.Context, actor access.Actor, versionID string, symbols []Symbol) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatSymbolsChanged, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		symbols, p := cleanSymbols(symbols, v.Symbols)
		if err := problems(p); err != nil {
			return nil, err
		}
		position := len(v.Symbols)
		for _, sym := range symbols {
			i := slices.IndexFunc(v.Symbols, func(old Symbol) bool { return old.Key == sym.Key })
			var err error
			switch {
			case i < 0:
				position++
				err = q.AddFormatSymbol(ctx, db.AddFormatSymbolParams{VersionID: v.ID, Key: sym.Key, Name: sym.Name, Position: int16(position), Automatic: false})
			case v.Symbols[i].Name != sym.Name:
				err = q.RenameFormatSymbol(ctx, db.RenameFormatSymbolParams{VersionID: v.ID, Key: sym.Key, Name: sym.Name})
			}
			if err != nil {
				return nil, err
			}
		}
		before, after := symbolNames(v.Symbols), symbolNames(symbols)
		if before == after && len(symbols) == len(v.Symbols) {
			return nil, nil
		}
		return map[string]string{"symbols": audit.Change(before, after)}, nil
	})
}

// SetHighlights mengganti aturan sorot satu versi (R-TPL-11); 0 berarti mati.
func (s *Service) SetHighlights(ctx context.Context, actor access.Actor, versionID string, in HighlightSettings) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatSettingsChanged, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		rpnAbove, severityAtLeast, p := cleanHighlights(v, in)
		if err := problems(p); err != nil {
			return nil, err
		}
		err := q.SetFormatHighlights(ctx, db.SetFormatHighlightsParams{
			ID: v.ID, HighlightRpnAbove: optionalSmall(rpnAbove), HighlightSeverityAtLeast: optionalSmall(severityAtLeast),
		})
		if err != nil {
			return nil, err
		}
		details := map[string]string{}
		if v.HighlightRPNAbove != rpnAbove {
			details["highlight_rpn_above"] = audit.Change(threshold(v.HighlightRPNAbove), threshold(rpnAbove))
		}
		if v.HighlightSeverityAtLeast != severityAtLeast {
			details["highlight_severity_at_least"] = audit.Change(threshold(v.HighlightSeverityAtLeast), threshold(severityAtLeast))
		}
		return details, nil
	})
}

// editVersion menjalankan satu perubahan pada versi draft di dalam
// transaksi: hak A-2, versi dikunci, versi published ditolak, lalu entri
// audit bila ada yang berubah (details tidak kosong).
func (s *Service) editVersion(ctx context.Context, actor access.Actor, versionID, action string,
	change func(q *db.Queries, v FormatVersion) (map[string]string, error)) (FormatVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormatVersion{}, err
	}
	if !db.IsUUID(versionID) {
		return FormatVersion{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockFormatVersion(ctx, versionID); err != nil {
			return err
		}
		v, err := getVersion(ctx, q, versionID)
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
			Action: action, TargetType: audit.TargetFormatTemplate, TargetID: v.TemplateID, Details: details,
		})
	})
	if err := result(err, "mengubah format template"); err != nil {
		return FormatVersion{}, err
	}
	return getVersion(ctx, s.q, versionID)
}

// cleanNewTemplate memeriksa isian template baru. Bila menyalin, versi
// sumbernya dikembalikan.
func (s *Service) cleanNewTemplate(ctx context.Context, in NewTemplate) (NewTemplate, *FormatVersion, error) {
	name, nameProblem := problem.Text("name", in.Name, true, maxNameLength)
	list := []*problem.Problem{nameProblem}
	out := NewTemplate{Name: name, DocumentTypeID: in.DocumentTypeID, Methodology: in.Methodology,
		CustomerID: in.CustomerID, RiskMethod: in.RiskMethod, CopyFromVersionID: in.CopyFromVersionID}

	typeProblem, err := checkDocumentType(ctx, s.q, in.DocumentTypeID, "Only PFD, PFMEA, and CP have format templates.")
	if err != nil {
		return NewTemplate{}, nil, err
	}
	list = append(list, typeProblem)
	if in.Methodology != "" && !slices.Contains(methods, in.Methodology) {
		list = append(list, &problem.Problem{Field: "methodology", Code: problem.InvalidFormat, Message: "Choose AIAG 4th, AIAG-VDA, or SQAM."})
	}
	customerProblem, err := s.checkCustomer(ctx, in.CustomerID)
	if err != nil {
		return NewTemplate{}, nil, err
	}
	list = append(list, customerProblem)

	var source *FormatVersion
	if in.CopyFromVersionID != "" {
		v, sourceProblem, err := s.copySource(ctx, in.CopyFromVersionID, in.DocumentTypeID)
		if err != nil {
			return NewTemplate{}, nil, err
		}
		list = append(list, sourceProblem)
		if sourceProblem == nil {
			source = &v
			out.RiskMethod = v.RiskMethod
		}
	} else {
		switch {
		case in.RiskMethod == "":
			list = append(list, &problem.Problem{Field: "risk_method", Code: problem.Required, Message: "Choose RPN, AP, or none."})
		case !slices.Contains(risks, in.RiskMethod):
			list = append(list, &problem.Problem{Field: "risk_method", Code: problem.InvalidFormat, Message: "Choose RPN, AP, or none."})
		}
	}
	return out, source, problems(list...)
}

// checkDocumentType memeriksa jenis dokumen: hanya PFD, PFMEA, dan CP yang
// punya format template dan poin process template (R-TPL-01). notAllowed
// adalah pesan untuk jenis dokumen lain.
func checkDocumentType(ctx context.Context, q *db.Queries, id, notAllowed string) (*problem.Problem, error) {
	if id == "" {
		return &problem.Problem{Field: "document_type_id", Code: problem.Required, Message: "Choose the document type."}, nil
	}
	if !db.IsUUID(id) {
		return &problem.Problem{Field: "document_type_id", Code: problem.InvalidFormat, Message: "Choose an existing document type."}, nil
	}
	d, err := q.GetDocumentType(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return &problem.Problem{Field: "document_type_id", Code: problem.InvalidFormat, Message: "Choose an existing document type."}, nil
	case err != nil:
		return nil, fmt.Errorf("membaca jenis dokumen: %w", err)
	case d.Key == nil:
		return &problem.Problem{Field: "document_type_id", Code: problem.NotAllowed, Message: notAllowed}, nil
	}
	return nil, nil
}

// checkCustomer memeriksa customer format SQAM: boleh kosong, dan bila diisi
// harus customer aktif (seperti E-09 dan ASUMSI E-67).
func (s *Service) checkCustomer(ctx context.Context, id string) (*problem.Problem, error) {
	if id == "" {
		return nil, nil
	}
	if !db.IsUUID(id) {
		return &problem.Problem{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}, nil
	}
	c, err := s.q.GetCustomer(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return &problem.Problem{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}, nil
	case err != nil:
		return nil, fmt.Errorf("membaca customer: %w", err)
	case !c.Active:
		return &problem.Problem{Field: "customer_id", Code: problem.NotAllowed, Message: "This customer is inactive."}, nil
	}
	return nil, nil
}

// copySource membaca versi sumber salinan dan memeriksa jenis dokumennya.
func (s *Service) copySource(ctx context.Context, versionID, documentTypeID string) (FormatVersion, *problem.Problem, error) {
	notFound := &problem.Problem{Field: "copy_from_version_id", Code: problem.InvalidFormat, Message: "Choose an existing format template version."}
	if !db.IsUUID(versionID) {
		return FormatVersion{}, notFound, nil
	}
	v, err := getVersion(ctx, s.q, versionID)
	if errors.Is(err, problem.ErrNotFound) {
		return FormatVersion{}, notFound, nil
	}
	if err != nil {
		return FormatVersion{}, nil, err
	}
	t, err := s.q.GetFormatTemplate(ctx, v.TemplateID)
	if err != nil {
		return FormatVersion{}, nil, fmt.Errorf("membaca format template sumber: %w", err)
	}
	if t.DocumentTypeID != documentTypeID {
		return FormatVersion{}, &problem.Problem{Field: "copy_from_version_id", Code: problem.NotAllowed,
			Message: "Copy a format template of the same document type."}, nil
	}
	return v, nil, nil
}

// emptyVersion membuat versi 1 tanpa field. Metode risiko RPN dan AP
// mendapat tabel skala 1 sampai 10 dengan kriteria kosong.
func emptyVersion(ctx context.Context, q *db.Queries, templateID, risk string) error {
	versionID, err := q.CreateFormatVersion(ctx, db.CreateFormatVersionParams{TemplateID: templateID, Version: 1, RiskMethod: risk})
	if err != nil || risk == RiskNone {
		return err
	}
	for _, scale := range scales {
		for value := 1; value <= maxRating; value++ {
			if err := q.AddRatingCriterion(ctx, db.AddRatingCriterionParams{VersionID: versionID, Scale: scale, Value: int16(value)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyVersion membuat versi 1 template baru dari versi sumber. Setiap field
// mendapat identitas baru milik template baru, dengan label, urutan, dan
// pilihan yang sama.
func copyVersion(ctx context.Context, q *db.Queries, templateID string, source FormatVersion) error {
	versionID, err := q.CreateFormatVersion(ctx, db.CreateFormatVersionParams{
		TemplateID: templateID, Version: 1, RiskMethod: source.RiskMethod,
		HighlightRpnAbove: optionalSmall(source.HighlightRPNAbove), HighlightSeverityAtLeast: optionalSmall(source.HighlightSeverityAtLeast),
	})
	if err != nil {
		return err
	}
	for _, f := range source.Fields {
		fieldID, err := q.CreateFormatField(ctx, db.CreateFormatFieldParams{
			TemplateID: templateID, Area: f.Area, Kind: f.Kind, RatingScale: optional(f.RatingScale), RatingSet: optionalSmall(f.RatingSet),
		})
		if err != nil {
			return err
		}
		err = q.AddVersionField(ctx, db.AddVersionFieldParams{
			VersionID: versionID, FieldID: fieldID, TemplateID: templateID, Area: f.Area, Label: f.Label,
			Position: int16(f.Position), Required: f.Required, Hidden: f.Hidden, Options: optionsJSON(f.Options),
		})
		if err != nil {
			return err
		}
	}
	for _, r := range source.Ratings {
		err := q.AddRatingCriterion(ctx, db.AddRatingCriterionParams{VersionID: versionID, Scale: r.Scale, Value: int16(r.Value), Criteria: r.Criteria})
		if err != nil {
			return err
		}
	}
	for i, sym := range source.Symbols {
		err := q.AddFormatSymbol(ctx, db.AddFormatSymbolParams{
			VersionID: versionID, Key: sym.Key, Name: sym.Name, Position: int16(i + 1), Automatic: sym.Automatic,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// getTemplate membaca satu format template beserta ringkasan versinya.
func getTemplate(ctx context.Context, q *db.Queries, id string) (FormatTemplate, error) {
	row, err := q.GetFormatTemplate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return FormatTemplate{}, problem.ErrNotFound
	}
	if err != nil {
		return FormatTemplate{}, fmt.Errorf("membaca format template: %w", err)
	}
	versions, err := q.ListVersionsOfTemplate(ctx, id)
	if err != nil {
		return FormatTemplate{}, fmt.Errorf("membaca versi format template: %w", err)
	}
	summaries := make([]VersionSummary, 0, len(versions))
	for _, v := range versions {
		summaries = append(summaries, summary(v))
	}
	return fromTemplate(row, summaries), nil
}

// getVersion membaca satu versi lengkap.
func getVersion(ctx context.Context, q *db.Queries, id string) (FormatVersion, error) {
	row, err := q.GetFormatVersion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return FormatVersion{}, problem.ErrNotFound
	}
	if err != nil {
		return FormatVersion{}, fmt.Errorf("membaca versi format template: %w", err)
	}
	v := FormatVersion{
		ID: row.ID, TemplateID: row.TemplateID, Version: int(row.Version), Status: row.Status, RiskMethod: row.RiskMethod,
		HighlightRPNAbove: deref(row.HighlightRpnAbove), HighlightSeverityAtLeast: deref(row.HighlightSeverityAtLeast),
		Fields: []Field{}, Ratings: []Rating{}, Symbols: []Symbol{}, CreatedAt: row.CreatedAt.UTC(),
	}
	fields, err := q.ListVersionFields(ctx, id)
	if err != nil {
		return FormatVersion{}, fmt.Errorf("membaca field: %w", err)
	}
	for _, f := range fields {
		field := Field{
			ID: f.FieldID, Area: f.Area, Kind: f.Kind, RatingScale: derefText(f.RatingScale), RatingSet: deref(f.RatingSet),
			Label: f.Label, Position: int(f.Position), Required: f.Required, Hidden: f.Hidden,
		}
		if field.Options, err = parseOptions(f.Options); err != nil {
			return FormatVersion{}, err
		}
		v.Fields = append(v.Fields, field)
	}
	sortFields(v.Fields)
	ratings, err := q.ListRatingCriteria(ctx, id)
	if err != nil {
		return FormatVersion{}, fmt.Errorf("membaca tabel skala: %w", err)
	}
	for _, r := range ratings {
		v.Ratings = append(v.Ratings, Rating{Scale: r.Scale, Value: int(r.Value), Criteria: r.Criteria})
	}
	symbols, err := q.ListFormatSymbols(ctx, id)
	if err != nil {
		return FormatVersion{}, fmt.Errorf("membaca simbol: %w", err)
	}
	for _, sym := range symbols {
		v.Symbols = append(v.Symbols, Symbol{Key: sym.Key, Name: sym.Name, Automatic: sym.Automatic})
	}
	return v, nil
}

// fromTemplate mengubah baris hasil sqlc menjadi FormatTemplate.
func fromTemplate(r db.GetFormatTemplateRow, versions []VersionSummary) FormatTemplate {
	if versions == nil {
		versions = []VersionSummary{}
	}
	return FormatTemplate{
		ID: r.ID, Name: r.Name, DocumentTypeID: r.DocumentTypeID, DocumentTypeKey: derefText(r.DocumentTypeKey),
		DocumentTypeName: r.DocumentTypeName, Methodology: derefText(r.Methodology), CustomerID: derefText(r.CustomerID),
		CustomerCode: derefText(r.CustomerCode), Versions: versions, CreatedAt: r.CreatedAt.UTC(),
	}
}

func summary(v db.FormatTemplateVersion) VersionSummary {
	return VersionSummary{ID: v.ID, Version: int(v.Version), Status: v.Status, RiskMethod: v.RiskMethod}
}

// optionJSON adalah bentuk pilihan di kolom options (jsonb).
type optionJSON struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// optionsJSON menulis pilihan untuk kolom options; nil bila tidak ada.
func optionsJSON(options []Option) []byte {
	if len(options) == 0 {
		return nil
	}
	list := make([]optionJSON, 0, len(options))
	for _, o := range options {
		list = append(list, optionJSON{Key: o.Key, Label: o.Label})
	}
	data, _ := json.Marshal(list) // struct teks biasa: Marshal tidak bisa gagal
	return data
}

// parseOptions membaca kolom options.
func parseOptions(data []byte) ([]Option, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var list []optionJSON
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("membaca pilihan field: %w", err)
	}
	options := make([]Option, 0, len(list))
	for _, o := range list {
		options = append(options, Option{Key: o.Key, Label: o.Label})
	}
	return options, nil
}

// optionLabels dan symbolNames menulis daftar untuk rincian audit log.
func optionLabels(options []Option) string {
	labels := make([]string, 0, len(options))
	for _, o := range options {
		labels = append(labels, o.Label)
	}
	return strings.Join(labels, ", ")
}

func symbolNames(symbols []Symbol) string {
	names := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		names = append(names, sym.Name)
	}
	return strings.Join(names, ", ")
}

// threshold menulis ambang sorot untuk audit log; 0 berarti mati.
func threshold(n int) string {
	if n == 0 {
		return "off"
	}
	return strconv.Itoa(n)
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalSmall(n int) *int16 {
	if n == 0 {
		return nil
	}
	v := int16(n)
	return &v
}

func deref(n *int16) int {
	if n == nil {
		return 0
	}
	return int(*n)
}

func derefText(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// problems menggabungkan masalah input menjadi satu ValidationError, atau nil.
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

// result menerjemahkan error transaksi format template dan process template:
// nama atau label yang sudah dipakai menjadi ConflictError, error input dan
// "tidak ada" diteruskan apa adanya.
func result(err error, doing string) error {
	taken := func(field, message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
	}
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case err == nil:
		return nil
	case db.IsUniqueViolation(err, "format_templates_name_key"):
		return taken("name", "This name is already used by another format template.")
	case db.IsUniqueViolation(err, "format_version_fields_label_key"):
		return taken("label", "This label is already used by another field in this area.")
	case db.IsUniqueViolation(err, "format_symbols_name_key"):
		return taken("symbols", "Two symbols have the same name.")
	case db.IsUniqueViolation(err, "process_templates_process_key"):
		return taken("process_id", "This process already has a process template.")
	case db.IsUniqueViolation(err, "process_template_subprocesses_name_key"):
		return taken("name", "This name is already used by another sub-process in this version.")
	case db.IsUniqueViolation(err, "template_points_variant_scope_key"):
		return taken("scope_target_id", "This point already has a variant for this scope.")
	case errors.As(err, &invalid), errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}
