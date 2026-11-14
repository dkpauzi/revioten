package printout

// Layout sebagai data (M6.1, R-PRT-02). Developer menulis layout bawaan
// (migrasi 00040); admin (A-2) bisa menambah layout, mengubah isinya, dan
// memilih layout lain untuk satu customer, tanpa mengubah kode.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// PrintLayout adalah satu layout tersimpan.
type PrintLayout struct {
	ID              string
	Name            string
	DocumentTypeID  string
	DocumentTypeKey string
	IsDefault       bool
	Layout          Layout
	UpdatedAt       time.Time
}

// LayoutInput adalah isian layout baru atau yang diubah.
type LayoutInput struct {
	Name           string
	DocumentTypeID string // hanya untuk layout baru
	Layout         json.RawMessage
}

// Layouts memberi semua layout (L-2).
func (s *Service) Layouts(ctx context.Context, actor access.Actor) ([]PrintLayout, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListPrintLayouts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PrintLayout, 0, len(rows))
	for _, r := range rows {
		l, err := ParseLayout(r.Layout)
		if err != nil {
			return nil, fmt.Errorf("layout %s rusak: %w", r.Name, err)
		}
		out = append(out, PrintLayout{ID: r.ID, Name: r.Name, DocumentTypeID: r.DocumentTypeID, DocumentTypeKey: deref(r.DocumentTypeKey), IsDefault: r.IsDefault, Layout: l, UpdatedAt: r.UpdatedAt.UTC()})
	}
	return out, nil
}

// CreateLayout menambah layout baru (A-2).
func (s *Service) CreateLayout(ctx context.Context, actor access.Actor, in LayoutInput) (PrintLayout, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return PrintLayout{}, err
	}
	name, layout, err := checkLayout(in)
	if err != nil {
		return PrintLayout{}, err
	}
	if !db.IsUUID(in.DocumentTypeID) {
		return PrintLayout{}, invalid("document_type_id", problem.Required, "Choose the document type.")
	}
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if id, err = q.CreatePrintLayout(ctx, db.CreatePrintLayoutParams{Name: name, DocumentTypeID: in.DocumentTypeID, Layout: layout, UpdatedByPersonID: &actor.PersonID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionPrintLayoutChanged, Details: map[string]string{"layout": name, "created": "true"}})
	})
	if err != nil {
		return PrintLayout{}, layoutResult(err)
	}
	return s.layout(ctx, id)
}

// UpdateLayout mengubah nama dan isi satu layout (A-2).
func (s *Service) UpdateLayout(ctx context.Context, actor access.Actor, id string, in LayoutInput) (PrintLayout, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return PrintLayout{}, err
	}
	if !db.IsUUID(id) {
		return PrintLayout{}, problem.ErrNotFound
	}
	name, layout, err := checkLayout(in)
	if err != nil {
		return PrintLayout{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.UpdatePrintLayout(ctx, db.UpdatePrintLayoutParams{ID: id, Name: name, Layout: layout, UpdatedByPersonID: &actor.PersonID})
		if err != nil {
			return err
		}
		if n == 0 {
			return problem.ErrNotFound
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionPrintLayoutChanged, Details: map[string]string{"layout": name, "content": string(layout)}})
	})
	if err != nil {
		return PrintLayout{}, layoutResult(err)
	}
	return s.layout(ctx, id)
}

// SetCustomerLayout memilih layout satu customer untuk satu jenis dokumen
// (A-2). layoutID kosong kembali ke layout bawaan.
func (s *Service) SetCustomerLayout(ctx context.Context, actor access.Actor, customerID, documentTypeID, layoutID string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return err
	}
	if !db.IsUUID(customerID) || !db.IsUUID(documentTypeID) {
		return problem.ErrNotFound
	}
	if layoutID != "" {
		l, err := s.layout(ctx, layoutID)
		if err != nil {
			return invalid("layout_id", problem.InvalidFormat, "Choose a layout from the list.")
		}
		if l.DocumentTypeID != documentTypeID {
			return invalid("layout_id", problem.NotAllowed, "Choose a layout of the same document type.")
		}
	}
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if layoutID == "" {
			err = q.ClearCustomerPrintLayout(ctx, db.ClearCustomerPrintLayoutParams{CustomerID: customerID, DocumentTypeID: documentTypeID})
		} else {
			err = q.SetCustomerPrintLayout(ctx, db.SetCustomerPrintLayoutParams{CustomerID: customerID, DocumentTypeID: documentTypeID, LayoutID: layoutID})
		}
		if err != nil {
			return layoutResult(err)
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionCustomerLayoutChanged, TargetType: audit.TargetCustomer, TargetID: customerID,
			Details: map[string]string{"document_type": documentTypeID, "layout": layoutID}})
	})
}

// CustomerLayouts memberi layout pilihan satu customer per jenis dokumen (L-2).
func (s *Service) CustomerLayouts(ctx context.Context, actor access.Actor, customerID string) (map[string]string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return nil, err
	}
	if !db.IsUUID(customerID) {
		return nil, problem.ErrNotFound
	}
	rows, err := s.q.ListCustomerPrintLayouts(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.DocumentTypeID] = r.LayoutID
	}
	return out, nil
}

func (s *Service) layout(ctx context.Context, id string) (PrintLayout, error) {
	if !db.IsUUID(id) {
		return PrintLayout{}, problem.ErrNotFound
	}
	r, err := s.q.GetPrintLayout(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PrintLayout{}, problem.ErrNotFound
	}
	if err != nil {
		return PrintLayout{}, err
	}
	l, err := ParseLayout(r.Layout)
	if err != nil {
		return PrintLayout{}, err
	}
	return PrintLayout{ID: r.ID, Name: r.Name, DocumentTypeID: r.DocumentTypeID, DocumentTypeKey: deref(r.DocumentTypeKey), IsDefault: r.IsDefault, Layout: l, UpdatedAt: r.UpdatedAt.UTC()}, nil
}

// checkLayout memeriksa nama dan isi layout, lalu menulis ulang isinya
// dalam bentuk baku.
func checkLayout(in LayoutInput) (string, []byte, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > 100 {
		return "", nil, invalid("name", problem.Required, "Write a name of at most 100 characters.")
	}
	l, err := ParseLayout(in.Layout)
	if err != nil {
		return "", nil, err
	}
	data, err := json.Marshal(l)
	return name, data, err
}

func layoutResult(err error) error {
	if db.IsUniqueViolation(err, "print_layouts_name_key") {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: "name", Code: problem.Taken, Message: "Another layout has this name."}}}
	}
	return err
}
