// Package document berisi dokumen PFD, PFMEA, dan CP beserta barisnya
// (Tahap 4, M4.1).
//
// Satu dokumen:
//   - memakai satu nomor register (satu nomor satu dokumen), dan nomor itu
//     menjadi In use;
//   - milik satu customer (hook portal, brief bagian 17) dan mencakup satu
//     atau lebih model serta part number (R-DOC-08);
//   - memakai versi terbit terakhir form set customer itu, sehingga format
//     setiap sheet sudah pasti dan tidak berubah (D-133);
//   - memakai rute approval bawaan customer, atau rute lain dengan alasan
//     yang tercatat di audit log (D-47);
//   - berisi header dan baris. Isinya JSONB per ID field format, diperiksa
//     terhadap versi format itu (R-DOC-01). Baris punya ID tetap dan urutan
//     sendiri (R-DOC-02), serta proses master dan nomor operasi sebagai
//     atribut baris, tempat kolom terisi otomatis membaca isinya (D-122, D-123).
//
// Dokumen baru selalu draft. Revisi, rilis, dan snapshot datang di Tahap 5 dan 6.
package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Status dokumen (M5.4): draft bila ada draft revisi terbuka, released bila
// tidak ada; dokumen released hanya bisa diubah lewat perubahan yang diterima.
const (
	StatusDraft    = "draft"
	StatusReleased = "released"
)

// Batas isian. ASUMSI E-90 (Terbuka): nomor operasi paling banyak 20 karakter, alasan
// rute 500 karakter, dan paling banyak 1.000 baris per dokumen.
const (
	maxOperationNumber = 20
	maxRouteReason     = 500
	maxRows            = 1000
)

// TypeRef adalah jenis dokumen.
type TypeRef struct {
	ID   string
	Key  string // pfd, pfmea, atau cp
	Name string
}

// Ref adalah rujukan bernama: model atau rute.
type Ref struct {
	ID   string
	Name string
}

// PartRef adalah satu part number yang dicakup dokumen.
type PartRef struct {
	ID     string
	Number string
	Name   string
}

// Row adalah satu baris dokumen.
type Row struct {
	ID              string // tetap selamanya (R-DOC-02)
	Position        int    // urutan, mulai 1
	ProcessID       string // "" bila belum dipilih
	ProcessName     string // nama tampilan proses yang berlaku sekarang (D-06)
	OperationNumber string
	Content         map[string]any // per ID kolom format
	// Version naik setiap kali baris disimpan (penguncian optimistik, M4.3).
	Version int
	// TemplatePointID adalah poin process template asal baris ini; "" untuk
	// baris yang diketik atau ditempel sendiri (R-CMP-04, M4.5).
	TemplatePointID   string
	TemplateVersionID string
	// OperationManual: nomor operasi diketik atau ditempel penyusun, jadi
	// penomoran otomatis tidak menimpanya (M4.6).
	OperationManual bool
	// Detached: tautan ke poin template diputus, dengan alasan (M4.8, R-CMP-04).
	Detached     bool
	DetachReason string
	// SectionID adalah bagian pemilik baris (D-155); "" untuk dokumen tanpa bagian.
	SectionID string
}

// Document adalah satu dokumen lengkap dengan header dan barisnya.
type Document struct {
	ID               string
	NumberID         string
	Number           string
	CustomerID       string
	CustomerCode     string
	DocumentType     TypeRef
	Revision         int  // revisi yang dikerjakan, atau revisi terakhir yang terbit
	ReleasedRevision *int // revisi yang berlaku; nil bila belum pernah terbit (M5.4)
	Status           string
	FormSetVersionID string
	FormSetName      string
	FormatVersionID  string // versi format sheet dokumen ini
	Models           []Ref  // tidak pernah nil
	PartNumbers      []PartRef
	Route            *Ref   // nil bila belum ada rute
	RouteReason      string // "" bila memakai rute bawaan
	Header           map[string]any
	HeaderVersion    int   // naik setiap kali header disimpan (M4.3)
	Rows             []Row // menurut urutan, tidak pernah nil
	// Sections adalah bagian per PCB menurut urutan (D-155); kosong untuk
	// dokumen biasa. Tidak pernah nil.
	Sections []Section
	// Set berisi dokumen satu set, termasuk dokumen ini, menurut urutan jenis
	// dokumen (M4.2). Dokumen dari nomor tunggal hanya berisi dirinya sendiri.
	Set       []SetMember
	CreatedAt time.Time
	// Origin: new, atau migrated untuk dokumen lama (Tahap 7).
	Origin string
	// ReleasedMigrated: revisi yang berlaku adalah baseline migrasi yang
	// sudah dicocokkan dengan hardcopy, "Released (migrated)" (R-MIG-06).
	ReleasedMigrated bool
	// Baseline adalah data hardcopy dokumen migrasi; nil untuk dokumen baru.
	Baseline  *Baseline
	UpdatedAt time.Time
	// Signing: status permintaan tanda tangan digital untuk draft terbuka
	// (D-198): "" bila tidak ada, waiting, atau completed. Selama ada, isi
	// draft dikunci (R-REV-06).
	Signing string
}

// SetMember adalah satu dokumen di set yang sama (M4.2): nomornya didaftarkan
// dalam satu pendaftaran set PFD, PFMEA, dan CP (M2.4). Workbook memakainya
// sebagai tab sheet.
type SetMember struct {
	ID           string
	Number       string
	DocumentType TypeRef
}

// Summary adalah satu dokumen di daftar.
type Summary struct {
	ID           string
	Number       string
	DocumentType TypeRef
	Revision     int
	Status       string
	FormSetName  string
	RowCount     int
	Models       []string // nama model, menurut abjad (M4.7); tidak pernah nil
	Origin       string
	// ReleasedMigrated: lihat Document.
	ReleasedMigrated bool
	UpdatedAt        time.Time
}

// NewDocument adalah isian dokumen baru.
type NewDocument struct {
	NumberID      string
	ModelIDs      []string
	PartNumberIDs []string
	// RouteID kosong berarti rute bawaan customer. Rute lain butuh alasan (D-47).
	RouteID     string
	RouteReason string
}

// NewRow adalah isian baris baru; baris baru ditaruh di akhir.
type NewRow struct {
	ProcessID       string
	OperationNumber string
	Content         map[string]any
	// SectionID wajib bila dokumen punya bagian (D-155).
	SectionID string
}

// RowUpdate adalah isi baru satu baris. Semua isian dikirim utuh, bersama
// versi baris yang dibaca penyusun (M4.3).
type RowUpdate struct {
	ProcessID       string
	OperationNumber string
	Content         map[string]any
	Version         int
}

// conflict adalah penolakan karena isi sudah diubah orang lain (M4.3, R-DOC-07).
func conflict(field, message string) error {
	return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Changed, Message: message}}}
}

// Service membuat, membaca, dan mengubah dokumen.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database.
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz}
}

func invalid(field, code, message string) *problem.Problem {
	return &problem.Problem{Field: field, Code: code, Message: message}
}

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

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Create membuat dokumen draft dari satu nomor register (E-2 untuk customer nomor itu).
func (s *Service) Create(ctx context.Context, actor access.Actor, in NewDocument) (Document, error) {
	if in.NumberID == "" {
		return Document{}, problems(invalid("document_number_id", problem.Required, "Choose the document number."))
	}
	if !db.IsUUID(in.NumberID) {
		return Document{}, problems(invalid("document_number_id", problem.InvalidFormat, "Choose an existing document number."))
	}
	number, err := s.q.GetNumberForDocument(ctx, in.NumberID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, problems(invalid("document_number_id", problem.InvalidFormat, "Choose an existing document number."))
	}
	if err != nil {
		return Document{}, fmt.Errorf("membaca nomor: %w", err)
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: number.CustomerID}); err != nil {
		return Document{}, err
	}

	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Nomor dikunci, supaya dua permintaan tidak memakai nomor yang sama.
		number, err := q.GetNumberForDocument(ctx, in.NumberID)
		if err != nil {
			return err
		}
		used, err := q.DocumentExistsForNumber(ctx, number.ID)
		if err != nil {
			return err
		}
		if used || number.Status != "reserved" {
			return problems(invalid("document_number_id", problem.NotAllowed, "Choose a Reserved number that no document uses yet."))
		}
		customer, err := q.GetCustomerForDocument(ctx, number.CustomerID)
		if err != nil {
			return err
		}
		list := s.checkScope(ctx, q, number.CustomerID, in)
		route, reason, routeProblem := checkRoute(ctx, q, customer.DefaultRouteID, in)
		list = append(list, routeProblem)
		set, setProblem := formSetOf(ctx, q, customer.DefaultFormSetID)
		list = append(list, setProblem)
		if err := problems(list...); err != nil {
			return err
		}
		formatID := map[string]string{"pfd": set.PfdFormatVersionID, "pfmea": set.PfmeaFormatVersionID, "cp": set.CpFormatVersionID}[deref(number.DocumentTypeKey)]
		if formatID == "" {
			return problems(invalid("document_number_id", problem.NotAllowed, "Only PFD, PFMEA, and CP numbers make documents."))
		}

		id, err = q.CreateDocument(ctx, db.CreateDocumentParams{
			DocumentNumberID: number.ID, CustomerID: number.CustomerID, DocumentTypeID: number.DocumentTypeID,
			Revision: number.InitialRevision, FormSetVersionID: set.ID, FormatVersionID: formatID,
			ApprovalRouteID: route, RouteReason: optional(reason), CreatedByPersonID: actor.PersonID, Origin: OriginNew,
		})
		if err != nil {
			return err
		}
		// Dokumen baru adalah draft revisi awalnya (M5.4).
		if _, err := q.CreateDocumentRevision(ctx, db.CreateDocumentRevisionParams{DocumentID: id, Revision: number.InitialRevision, CreatedByPersonID: actor.PersonID}); err != nil {
			return err
		}
		for _, m := range in.ModelIDs {
			if err := q.AddDocumentModel(ctx, db.AddDocumentModelParams{DocumentID: id, ModelID: m}); err != nil {
				return err
			}
		}
		for _, p := range in.PartNumberIDs {
			if err := q.AddDocumentPartNumber(ctx, db.AddDocumentPartNumberParams{DocumentID: id, PartNumberID: p}); err != nil {
				return err
			}
		}
		// Nomor yang sudah dipakai dokumen menjadi In use (R-NUM-04).
		now := time.Now()
		if err := q.SetDocumentNumberStatus(ctx, db.SetDocumentNumberStatusParams{ID: number.ID, Status: "in_use", StatusChangedAt: &now}); err != nil {
			return err
		}
		details := map[string]string{"document_number": number.Number, "form_set": set.Name}
		if reason != "" {
			details["route_reason"] = reason
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionDocumentCreated, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return Document{}, result(err, "membuat dokumen")
	}
	return getDocument(ctx, s.q, id)
}

// checkScope memeriksa model dan part number: minimal satu model, semua milik
// customer nomor itu, dan part number milik salah satu model yang dipilih.
func (s *Service) checkScope(ctx context.Context, q *db.Queries, customerID string, in NewDocument) []*problem.Problem {
	var list []*problem.Problem
	if len(in.ModelIDs) == 0 {
		return append(list, invalid("model_ids", problem.Required, "Choose at least one model."))
	}
	for _, m := range in.ModelIDs {
		owner, err := q.GetModelOwner(ctx, m)
		if !db.IsUUID(m) || err != nil || owner != customerID {
			list = append(list, invalid("model_ids", problem.NotAllowed, "Choose models of the customer of this document number."))
			break
		}
	}
	for _, p := range in.PartNumberIDs {
		owner, err := q.GetPartNumberOwner(ctx, p)
		if !db.IsUUID(p) || err != nil || owner.CustomerID != customerID || !slices.Contains(in.ModelIDs, owner.ModelID) {
			list = append(list, invalid("part_number_ids", problem.NotAllowed, "Choose part numbers of the chosen models."))
			break
		}
	}
	return list
}

// checkRoute menentukan rute: bawaan customer, atau rute lain dengan alasan (D-47).
func checkRoute(ctx context.Context, q *db.Queries, defaultRoute *string, in NewDocument) (*string, string, *problem.Problem) {
	if in.RouteID == "" || (defaultRoute != nil && *defaultRoute == in.RouteID) {
		return defaultRoute, "", nil
	}
	exists, err := q.RouteExists(ctx, in.RouteID)
	if !db.IsUUID(in.RouteID) || err != nil || !exists {
		return nil, "", invalid("route_id", problem.InvalidFormat, "Choose an existing approval route.")
	}
	reason, p := problem.Text("route_reason", in.RouteReason, true, maxRouteReason)
	if p != nil {
		if p.Code == problem.Required {
			p.Message = "Give the reason for not using the default route of the customer."
		}
		return nil, "", p
	}
	return &in.RouteID, reason, nil
}

// formSetOf mengambil versi terbit terakhir form set customer, atau form set bawaan.
func formSetOf(ctx context.Context, q *db.Queries, chosen *string) (db.LatestPublishedFormSetVersionRow, *problem.Problem) {
	setID := deref(chosen)
	if setID == "" {
		def, err := q.GetDefaultFormSet(ctx)
		if err != nil {
			return db.LatestPublishedFormSetVersionRow{}, invalid("form_set", problem.NotAllowed, "There is no default form set.")
		}
		setID = def.ID
	}
	v, err := q.LatestPublishedFormSetVersion(ctx, setID)
	if err != nil {
		return db.LatestPublishedFormSetVersionRow{}, invalid("form_set", problem.NotAllowed,
			"The form set of this customer has no published version yet. Ask a form approver to publish it.")
	}
	return v, nil
}

// List mengembalikan dokumen satu customer (L-4: lihat draft).
func (s *Service) List(ctx context.Context, actor access.Actor, customerID string) ([]Summary, error) {
	if !db.IsUUID(customerID) {
		return nil, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: customerID}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListDocuments(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("mengambil dokumen: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, r := range rows {
		out = append(out, Summary{
			ID: r.ID, Number: r.Number, DocumentType: TypeRef{Key: deref(r.DocumentTypeKey), Name: r.DocumentTypeName},
			Revision: int(r.Revision), Status: r.Status, FormSetName: r.FormSetName, RowCount: int(r.RowCount), UpdatedAt: r.UpdatedAt.UTC(), Models: r.ModelNames,
			Origin: r.Origin, ReleasedMigrated: r.ReleasedMigrated,
		})
	}
	return out, nil
}

// Get mengembalikan satu dokumen lengkap (L-4).
func (s *Service) Get(ctx context.Context, actor access.Actor, id string) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	return d, nil
}

// load membaca dokumen tanpa memeriksa hak; dipakai untuk mencari customer-nya dulu.
// Load membaca satu dokumen tanpa memeriksa hak. Hanya untuk pemanggil yang
// sudah memeriksa hak atas dokumen lain di set yang sama, misalnya cetakan CP
// yang membaca langkah PFD satu set (C3). Row-level security tetap berlaku.
func (s *Service) Load(ctx context.Context, id string) (Document, error) {
	return s.load(ctx, id)
}

func (s *Service) load(ctx context.Context, id string) (Document, error) {
	if !db.IsUUID(id) {
		return Document{}, problem.ErrNotFound
	}
	return getDocument(ctx, s.q, id)
}

// SetHeader mengganti isi header dokumen (E-2).
// version adalah versi header yang dibaca penyusun (M4.3).
func (s *Service) SetHeader(ctx context.Context, actor access.Actor, id string, header map[string]any, version int) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error) {
		if header == nil {
			header = map[string]any{}
		}
		if err := checkContent(format, template.AreaHeader, header); err != nil {
			return "", nil, err
		}
		data, err := json.Marshal(header)
		if err != nil {
			return "", nil, err
		}
		changed, err := q.SetDocumentHeader(ctx, db.SetDocumentHeaderParams{ID: d.ID, Header: data, ExpectedVersion: int32(version)})
		if err != nil {
			return "", nil, err
		}
		if changed == 0 {
			return "", nil, conflict("header_version", "Someone else changed the header. The latest header is shown; enter your change again.")
		}
		return audit.ActionDocumentHeaderChanged, map[string]string{"fields": strconv.Itoa(len(header))}, nil
	})
}

// AddRow menambah satu baris di akhir dokumen (E-2).
func (s *Service) AddRow(ctx context.Context, actor access.Actor, id string, in NewRow) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error) {
		if len(d.Rows) >= maxRows {
			return "", nil, problems(invalid("rows", problem.NotAllowed, "A document has at most 1000 rows."))
		}
		section, p := rowSection(d, in.SectionID)
		if err := problems(p); err != nil {
			return "", nil, err
		}
		operation, content, err := checkRow(ctx, q, format, in.ProcessID, in.OperationNumber, in.Content)
		if err != nil {
			return "", nil, err
		}
		row, err := q.AddDocumentRow(ctx, db.AddDocumentRowParams{DocumentID: d.ID, ProcessID: optional(in.ProcessID), OperationNumber: optional(operation), Content: content, SectionID: section})
		if err != nil {
			return "", nil, err
		}
		return audit.ActionDocumentRowAdded, map[string]string{"position": strconv.Itoa(int(row.Position))}, nil
	})
}

// AddRows menambah beberapa baris sekaligus di akhir dokumen (E-2, M4.4,
// R-MIG-02): blok baris yang ditempel dari Excel. Semua baris masuk dalam
// satu transaksi; satu baris yang ditolak membatalkan semuanya, dan
// masalahnya menunjuk barisnya, misalnya "rows[2].<id kolom>".
func (s *Service) AddRows(ctx context.Context, actor access.Actor, id string, rows []NewRow) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error) {
		if len(rows) == 0 {
			return "", nil, problems(invalid("rows", problem.Required, "Send at least one row."))
		}
		if len(d.Rows)+len(rows) > maxRows {
			return "", nil, problems(invalid("rows", problem.NotAllowed, "A document has at most 1000 rows."))
		}
		var first, last int32
		for i, in := range rows {
			section, p := rowSection(d, in.SectionID)
			if err := problems(p); err != nil {
				return "", nil, atRow(i, err)
			}
			operation, content, err := checkRow(ctx, q, format, in.ProcessID, in.OperationNumber, in.Content)
			if err != nil {
				return "", nil, atRow(i, err)
			}
			row, err := q.AddDocumentRow(ctx, db.AddDocumentRowParams{DocumentID: d.ID, ProcessID: optional(in.ProcessID), OperationNumber: optional(operation), Content: content, SectionID: section})
			if err != nil {
				return "", nil, err
			}
			if i == 0 {
				first = row.Position
			}
			last = row.Position
		}
		return audit.ActionDocumentRowsPasted, map[string]string{
			"count": strconv.Itoa(len(rows)), "from_position": strconv.Itoa(int(first)), "to_position": strconv.Itoa(int(last)),
		}, nil
	})
}

// atRow menambahkan nomor baris (mulai 0) ke nama field setiap masalah input.
func atRow(i int, err error) error {
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) {
		return err
	}
	out := make([]problem.Problem, len(invalid.Problems))
	for n, p := range invalid.Problems {
		p.Field = fmt.Sprintf("rows[%d].%s", i, p.Field)
		out[n] = p
	}
	return &problem.ValidationError{Problems: out}
}

// UpdateRow mengganti isi satu baris (E-2). ID dan urutannya tetap.
func (s *Service) UpdateRow(ctx context.Context, actor access.Actor, id, rowID string, in RowUpdate) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error) {
		i := slices.IndexFunc(d.Rows, func(r Row) bool { return r.ID == rowID })
		if i < 0 {
			return "", nil, problem.ErrNotFound
		}
		if in.Version < 1 {
			return "", nil, problems(invalid("version", problem.Required, "Send the version of the row that was read."))
		}
		changedMessage := "Someone else changed this row. The latest row is shown; enter your change again."
		if d.Rows[i].Version != in.Version {
			return "", nil, conflict("version", changedMessage)
		}
		operation, content, err := checkRow(ctx, q, format, in.ProcessID, in.OperationNumber, in.Content)
		if err != nil {
			return "", nil, err
		}
		changed, err := q.UpdateDocumentRow(ctx, db.UpdateDocumentRowParams{
			ID: rowID, DocumentID: d.ID, ProcessID: optional(in.ProcessID), OperationNumber: optional(operation), Content: content,
			ExpectedVersion: int32(in.Version), OperationManual: isManual(d.Rows[i], operation),
		})
		if err != nil {
			return "", nil, err
		}
		// Penjaga kedua di database: versi diperiksa juga di perintah UPDATE.
		if changed == 0 {
			return "", nil, conflict("version", changedMessage)
		}
		return audit.ActionDocumentRowChanged, map[string]string{"position": strconv.Itoa(d.Rows[i].Position)}, nil
	})
}

// DeleteRow menghapus satu baris draft (E-2, M4.2). Baris sesudahnya bergeser
// naik; ID baris lain tetap.
func (s *Service) DeleteRow(ctx context.Context, actor access.Actor, id, rowID string) (Document, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, d Document, _ template.FormatVersion) (string, map[string]string, error) {
		if !db.IsUUID(rowID) || !slices.ContainsFunc(d.Rows, func(r Row) bool { return r.ID == rowID }) {
			return "", nil, problem.ErrNotFound
		}
		gone, err := q.DeleteDocumentRow(ctx, db.DeleteDocumentRowParams{ID: rowID, DocumentID: d.ID})
		if err != nil {
			return "", nil, err
		}
		if err := q.ShiftRowsUp(ctx, db.ShiftRowsUpParams{DocumentID: d.ID, Position: gone.Position}); err != nil {
			return "", nil, err
		}
		// Baris tertaut yang dihapus berarti poinnya dikecualikan (R-CMP-03).
		if gone.TemplatePointID != nil {
			if err := q.ExcludePoint(ctx, db.ExcludePointParams{DocumentID: d.ID, PointID: *gone.TemplatePointID}); err != nil {
				return "", nil, err
			}
		}
		details := map[string]string{"position": strconv.Itoa(int(gone.Position))}
		if gone.OperationNumber != nil {
			details["operation_number"] = *gone.OperationNumber
		}
		return audit.ActionDocumentRowRemoved, details, nil
	})
}

// edit menjalankan satu perubahan dokumen draft di dalam transaksi: hak E-2
// untuk customer dokumen itu, dokumen dikunci, lalu entri audit.
func (s *Service) edit(ctx context.Context, actor access.Actor, id string,
	change func(q *db.Queries, d Document, format template.FormatVersion) (string, map[string]string, error)) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockDocument(ctx, id); err != nil {
			return err
		}
		d, err := getDocument(ctx, q, id)
		if err != nil {
			return err
		}
		if err := requireDraft(d); err != nil {
			return err
		}
		format, err := template.LoadFormatVersion(ctx, q, d.FormatVersionID)
		if err != nil {
			return err
		}
		action, details, err := change(q, d, format)
		if err != nil {
			return err
		}
		if err := q.TouchDocument(ctx, id); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return Document{}, result(err, "mengubah dokumen")
	}
	return getDocument(ctx, s.q, id)
}

// checkRow memeriksa proses, nomor operasi, dan isi satu baris, lalu
// mengembalikan isi dalam bentuk JSON.
func checkRow(ctx context.Context, q *db.Queries, format template.FormatVersion, processID, operation string, content map[string]any) (string, []byte, error) {
	var list []*problem.Problem
	if processID != "" {
		exists, err := q.ProcessExists(ctx, processID)
		if !db.IsUUID(processID) || err != nil || !exists {
			list = append(list, invalid("process_id", problem.InvalidFormat, "Choose an existing process."))
		}
	}
	operation, p := problem.Text("operation_number", operation, false, maxOperationNumber)
	list = append(list, p)
	if err := problems(list...); err != nil {
		return "", nil, err
	}
	if content == nil {
		content = map[string]any{}
	}
	if err := checkContent(format, template.AreaRow, content); err != nil {
		return "", nil, err
	}
	data, err := json.Marshal(content)
	return operation, data, err
}

// checkContent memeriksa isi terhadap versi format. Isian wajib yang masih
// kosong tidak ditolak, karena draft boleh belum lengkap.
func checkContent(format template.FormatVersion, area string, content map[string]any) error {
	check := template.CheckContent(format, area, content)
	if check.Valid() {
		return nil
	}
	return &problem.ValidationError{Problems: check.Problems}
}

// getDocument membaca satu dokumen beserta model, part number, dan barisnya.
func getDocument(ctx context.Context, q *db.Queries, id string) (Document, error) {
	r, err := q.GetDocument(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, problem.ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("membaca dokumen: %w", err)
	}
	d := Document{
		ID: r.ID, NumberID: r.DocumentNumberID, Number: r.Number, CustomerID: r.CustomerID, CustomerCode: r.CustomerCode,
		DocumentType: TypeRef{ID: r.DocumentTypeID, Key: deref(r.DocumentTypeKey), Name: r.DocumentTypeName},
		Revision:     int(r.Revision), Status: r.Status, FormSetVersionID: r.FormSetVersionID, FormSetName: r.FormSetName,
		FormatVersionID: r.FormatVersionID, RouteReason: deref(r.RouteReason),
		Models: []Ref{}, PartNumbers: []PartRef{}, Rows: []Row{}, CreatedAt: r.CreatedAt.UTC(), UpdatedAt: r.UpdatedAt.UTC(),
		HeaderVersion: int(r.HeaderVersion), Origin: r.Origin, ReleasedMigrated: r.ReleasedMigrated,
	}
	if d.Origin == OriginMigrated {
		if d.Baseline, err = loadBaseline(ctx, q, id); err != nil {
			return Document{}, err
		}
	}
	if r.ReleasedRevision != nil {
		v := int(*r.ReleasedRevision)
		d.ReleasedRevision = &v
	}
	if d.Status == StatusDraft {
		sig, err := q.ActiveSignatureRequestOfDraft(ctx, id)
		switch {
		case err == nil:
			d.Signing = sig.Status
		case !errors.Is(err, pgx.ErrNoRows):
			return Document{}, fmt.Errorf("membaca permintaan tanda tangan: %w", err)
		}
	}
	if r.ApprovalRouteID != nil {
		d.Route = &Ref{ID: *r.ApprovalRouteID, Name: deref(r.RouteName)}
	}
	if err := json.Unmarshal(r.Header, &d.Header); err != nil {
		return Document{}, fmt.Errorf("membaca header: %w", err)
	}
	models, err := q.ListDocumentModels(ctx, id)
	if err != nil {
		return Document{}, err
	}
	for _, m := range models {
		d.Models = append(d.Models, Ref{ID: m.ID, Name: m.Name})
	}
	parts, err := q.ListDocumentPartNumbers(ctx, id)
	if err != nil {
		return Document{}, err
	}
	for _, p := range parts {
		d.PartNumbers = append(d.PartNumbers, PartRef{ID: p.ID, Number: p.Number, Name: deref(p.Name)})
	}
	members, err := q.ListDocumentSet(ctx, id)
	if err != nil {
		return Document{}, err
	}
	d.Set = make([]SetMember, 0, len(members))
	for _, m := range members {
		d.Set = append(d.Set, SetMember{ID: m.ID, Number: m.Number, DocumentType: TypeRef{ID: m.DocumentTypeID, Key: deref(m.DocumentTypeKey), Name: m.DocumentTypeName}})
	}
	sections, err := q.ListDocumentSections(ctx, id)
	if err != nil {
		return Document{}, err
	}
	d.Sections = make([]Section, 0, len(sections))
	for _, s := range sections {
		item := Section{ID: s.ID, Key: s.SectionKey, Position: int(s.Position), Name: s.Name, PartNumbers: []PartRef{}}
		for i, pid := range s.PartNumberIds {
			item.PartNumbers = append(item.PartNumbers, PartRef{ID: pid, Number: s.PartNumbers[i], Name: s.PartNames[i]})
		}
		d.Sections = append(d.Sections, item)
	}
	rows, err := q.ListDocumentRows(ctx, id)
	if err != nil {
		return Document{}, err
	}
	for _, row := range rows {
		item := Row{ID: row.ID, Position: int(row.Position), ProcessID: deref(row.ProcessID), ProcessName: deref(row.ProcessName), OperationNumber: deref(row.OperationNumber), Version: int(row.Version),
			TemplatePointID: deref(row.TemplatePointID), TemplateVersionID: deref(row.TemplateVersionID), OperationManual: row.OperationManual,
			Detached: row.DetachedAt != nil, DetachReason: deref(row.DetachReason), SectionID: deref(row.SectionID)}
		if err := json.Unmarshal(row.Content, &item.Content); err != nil {
			return Document{}, fmt.Errorf("membaca baris: %w", err)
		}
		d.Rows = append(d.Rows, item)
	}
	return d, nil
}

// result meneruskan penolakan input dan "tidak ada" apa adanya, dan
// membungkus error lain.
func result(err error, doing string) error {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	if errors.As(err, &invalid) || errors.As(err, &conflict) || errors.Is(err, problem.ErrNotFound) || errors.Is(err, access.ErrForbidden) {
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

// requireDraft menolak perubahan isi dokumen yang tidak punya draft terbuka
// (M5.4). Revisi yang terbit hanya berubah lewat perubahan yang diterima,
// yang membuka draft revisi berikutnya (R-CHG-03, R-REV-02).
func requireDraft(d Document) error {
	if d.Status != StatusDraft {
		return problems(invalid("status", problem.NotAllowed,
			fmt.Sprintf("%s has no open draft. Submit a change; when it is accepted, the next revision opens as a draft.", d.Number)))
	}
	return requireUnsigned(d)
}

// requireUnsigned menolak perubahan isi draft yang sedang atau sudah
// ditandatangani digital (R-REV-06, D-198). Database menjaga hal yang sama
// (migrasi 00050).
func requireUnsigned(d Document) error {
	switch d.Signing {
	case SigningWaiting:
		return problems(invalid("status", problem.NotAllowed,
			fmt.Sprintf("%s is waiting for signatures and cannot be edited. Withdraw the request first.", d.Number)))
	case SigningCompleted:
		return problems(invalid("status", problem.NotAllowed,
			fmt.Sprintf("%s is signed and waits for its release; it cannot be edited.", d.Number)))
	}
	return nil
}

// Status permintaan tanda tangan digital pada draft (Document.Signing).
const (
	SigningWaiting   = "waiting"
	SigningCompleted = "completed"
)

// Load membaca satu dokumen lengkap tanpa memeriksa hak. Hanya untuk layanan
// lain yang sudah memeriksa izin dengan caranya sendiri, misalnya API
// integrasi yang memeriksa scope dan customer token (Tahap 9).
func Load(ctx context.Context, q *db.Queries, id string) (Document, error) {
	if !db.IsUUID(id) {
		return Document{}, problem.ErrNotFound
	}
	return getDocument(ctx, q, id)
}
