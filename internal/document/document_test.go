package document

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// M4.1 · Dokumen dan baris: header dokumen terhubung ke nomor register,
// customer, model, part number, form set, dan rute approval; baris datar
// berisi JSONB yang diperiksa terhadap versi format template (R-DOC-01),
// dengan ID tetap dan urutan (R-DOC-02).

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func newActor(t *testing.T, pool *pgxpool.Pool, login, kind string, grants ...access.RoleGrant) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range grants {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: g.Role, Scope: g.Scope}); err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{Type: access.ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login, PersonID: person.ID}
}

func all(role string) access.RoleGrant { return access.RoleGrant{Role: role, Scope: access.ScopeAll} }

// fixture: customer AB dengan model K-125 (part PN-1) dan K-200, customer CD
// dengan model B-1, form set AIAG 4th yang sudah terbit, dan satu nomor CP
// berstatus Reserved untuk AB.
type fixture struct {
	s        *Service
	pool     *pgxpool.Pool
	q        *db.Queries
	numbers  *numbering.Service
	admin    access.Actor
	author   access.Actor
	viewer   access.Actor
	custAB   string
	custCD   string
	k125     string
	k200     string
	b1       string
	pn1      string
	process  string
	cpNumber numbering.DocumentNumber
	cpFormat template.FormatVersion
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.NewPool(t)
	checker := access.NewChecker(pool)
	f := fixture{s: NewService(pool, checker), pool: pool, q: db.New(pool), numbers: numbering.NewService(pool, checker)}
	f.admin = newActor(t, pool, "admin.uji", access.AccountAdmin, all(access.RoleAdmin))
	f.author = newActor(t, pool, "penyusun.uji", access.AccountWork, all(access.RoleAuthor))
	f.viewer = newActor(t, pool, "viewer.uji", access.AccountWork)
	approver := newActor(t, pool, "ppap.supervisor", access.AccountWork, all(access.RoleFormApprover))

	// Customer lewat layanannya, supaya kode customer untuk penomoran ikut tercatat.
	customers := customer.NewService(pool, checker)
	group, err := customers.CreateProjectGroup(ctx, f.admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	newCustomer := func(code string) string {
		c, err := customers.Create(ctx, f.admin, customer.NewCustomer{Code: code, Name: "Customer " + code, ProjectGroupID: group.ID})
		if err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	model := func(customerID, name string) string {
		id, err := f.q.CreateModel(ctx, db.CreateModelParams{CustomerID: customerID, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	f.custAB, f.custCD = newCustomer("AB"), newCustomer("CD")
	f.k125, f.k200, f.b1 = model(f.custAB, "K-125"), model(f.custAB, "K-200"), model(f.custCD, "B-1")
	var err2 error
	if f.pn1, err2 = f.q.CreatePartNumber(ctx, db.CreatePartNumberParams{CustomerID: f.custAB, ModelID: f.k125, Number: "PN-1"}); err2 != nil {
		t.Fatal(err2)
	}
	if f.process, err = f.q.CreateProcess(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: f.process, Kind: "display", Name: "SMT Printing"}); err != nil {
		t.Fatal(err)
	}

	// Form set AIAG 4th diterbitkan: format oleh admin, set oleh penyetuju form.
	templates := template.NewService(pool, checker)
	sets, err := templates.ListFormSets(ctx, f.admin)
	if err != nil {
		t.Fatal(err)
	}
	v1 := sets.Sets[0].Versions[0]
	for _, id := range []string{v1.PFD.FormatVersionID, v1.PFMEA.FormatVersionID, v1.CP.FormatVersionID} {
		if _, err := templates.PublishFormatVersion(ctx, f.admin, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := templates.PublishFormSetVersion(ctx, approver, v1.ID); err != nil {
		t.Fatal(err)
	}
	if f.cpFormat, err = templates.GetFormatVersion(ctx, f.admin, v1.CP.FormatVersionID); err != nil {
		t.Fatal(err)
	}
	f.cpNumber = f.register(t, "cp")
	return f
}

// register mendaftarkan satu nomor untuk customer AB.
func (f fixture) register(t *testing.T, key string) numbering.DocumentNumber {
	t.Helper()
	var typeID string
	if err := f.pool.QueryRow(t.Context(), "SELECT id FROM document_types WHERE key = $1", key).Scan(&typeID); err != nil {
		t.Fatal(err)
	}
	reg, err := f.numbers.Register(t.Context(), f.author, numbering.NewNumber{CustomerID: f.custAB, DocumentTypeID: typeID, IdempotencyKey: key + "-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	return reg.Numbers[0]
}

// fieldID mencari ID field format menurut label.
func fieldID(t *testing.T, v template.FormatVersion, area, label string) string {
	t.Helper()
	for _, fd := range v.Fields {
		if fd.Area == area && fd.Label == label {
			return fd.ID
		}
	}
	t.Fatalf("field %q tidak ada", label)
	return ""
}

func codes(t *testing.T, err error) []string {
	t.Helper()
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	var list []problem.Problem
	switch {
	case errors.As(err, &invalid):
		list = invalid.Problems
	case errors.As(err, &conflict):
		list = conflict.Problems
	default:
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	var out []string
	for _, p := range list {
		out = append(out, p.Field+":"+p.Code)
	}
	return out
}

// TestCreateDocument: dokumen dibuat dari nomor Reserved, memakai form set
// customer itu, dan nomornya menjadi In use.
func TestCreateDocument(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	in := NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125, f.k200}, PartNumberIDs: []string{f.pn1}}

	if _, err := f.s.Create(ctx, f.viewer, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer membuat dokumen: %v", err)
	}
	d, err := f.s.Create(ctx, f.author, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Number != f.cpNumber.Number || d.CustomerCode != "AB" || d.DocumentType.Key != "cp" || d.Status != StatusDraft || d.Revision != 0 {
		t.Errorf("dokumen: %+v", d)
	}
	if d.FormSetName != "AIAG 4th set" || d.FormatVersionID != f.cpFormat.ID {
		t.Errorf("form set %q, format %s", d.FormSetName, d.FormatVersionID)
	}
	var models []string
	for _, m := range d.Models {
		models = append(models, m.Name)
	}
	if !slices.Equal(models, []string{"K-125", "K-200"}) || len(d.PartNumbers) != 1 || d.PartNumbers[0].Number != "PN-1" {
		t.Errorf("model %v, part %+v", models, d.PartNumbers)
	}
	if len(d.Rows) != 0 || len(d.Header) != 0 {
		t.Errorf("dokumen baru tidak kosong: %+v", d)
	}
	var status string
	if err := f.pool.QueryRow(ctx, "SELECT status FROM document_numbers WHERE id = $1", f.cpNumber.ID).Scan(&status); err != nil || status != "in_use" {
		t.Errorf("status nomor: %q, %v", status, err)
	}
	// Satu nomor untuk satu dokumen.
	if _, err := f.s.Create(ctx, f.author, in); !slices.Equal(codes(t, err), []string{"document_number_id:not_allowed"}) {
		t.Errorf("nomor dipakai dua kali: %v", err)
	}
	// Daftar dokumen customer.
	list, err := f.s.List(ctx, f.author, f.custAB)
	if err != nil || len(list) != 1 || list[0].ID != d.ID {
		t.Errorf("daftar: %+v, %v", list, err)
	}
}

// TestCreateDocumentRules: isian dokumen baru diperiksa.
func TestCreateDocumentRules(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tests := []struct {
		name string
		in   NewDocument
		want []string
	}{
		{"tanpa nomor", NewDocument{ModelIDs: []string{f.k125}}, []string{"document_number_id:required"}},
		{"tanpa model", NewDocument{NumberID: f.cpNumber.ID}, []string{"model_ids:required"}},
		{"model customer lain", NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.b1}}, []string{"model_ids:not_allowed"}},
		{"part number model lain", NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k200}, PartNumberIDs: []string{f.pn1}}, []string{"part_number_ids:not_allowed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.s.Create(ctx, f.author, tt.in); !slices.Equal(codes(t, err), tt.want) {
				t.Errorf("masalah: %v, ingin %v", err, tt.want)
			}
		})
	}
	// Nomor yang sudah Void tidak bisa dipakai.
	voided := f.register(t, "pfd")
	if _, err := f.pool.Exec(ctx, "UPDATE document_numbers SET status = 'void' WHERE id = $1", voided.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Create(ctx, f.author, NewDocument{NumberID: voided.ID, ModelIDs: []string{f.k125}}); !slices.Equal(codes(t, err), []string{"document_number_id:not_allowed"}) {
		t.Errorf("nomor Void: %v", err)
	}
}

// TestRoute: rute bawaan dari customer; rute lain butuh alasan yang dicatat (D-47).
func TestRoute(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	var first, second string
	rows, err := f.pool.Query(ctx, "SELECT id FROM approval_routes ORDER BY created_at, id")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	first = ids[0]
	if err := f.q.SetCustomerDefaultRoute(ctx, db.SetCustomerDefaultRouteParams{ID: f.custAB, DefaultRouteID: &first}); err != nil {
		t.Fatal(err)
	}
	d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Route == nil || d.Route.ID != first || d.RouteReason != "" {
		t.Errorf("rute bawaan: %+v %q", d.Route, d.RouteReason)
	}
	// Rute lain tanpa alasan ditolak.
	second = f.makeRoute(t)
	other := f.register(t, "pfmea")
	if _, err := f.s.Create(ctx, f.author, NewDocument{NumberID: other.ID, ModelIDs: []string{f.k125}, RouteID: second}); !slices.Equal(codes(t, err), []string{"route_reason:required"}) {
		t.Errorf("rute lain tanpa alasan: %v", err)
	}
	d, err = f.s.Create(ctx, f.author, NewDocument{NumberID: other.ID, ModelIDs: []string{f.k125}, RouteID: second, RouteReason: "Customer minta tanda tangan GM"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Route == nil || d.Route.ID != second || d.RouteReason != "Customer minta tanda tangan GM" {
		t.Errorf("rute lain: %+v %q", d.Route, d.RouteReason)
	}
	var details string
	if err := f.pool.QueryRow(ctx, "SELECT details FROM audit_log WHERE action = 'document.created' ORDER BY seq DESC LIMIT 1").Scan(&details); err != nil {
		t.Fatal(err)
	}
	if want := `"route_reason":"Customer minta tanda tangan GM"`; !contains(details, want) {
		t.Errorf("audit: %s, ingin memuat %s", details, want)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// makeRoute membuat rute kedua lewat query (slotnya tidak dibutuhkan di sini).
func (f fixture) makeRoute(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(t.Context(), "INSERT INTO approval_routes (name) VALUES ('Route uji') RETURNING id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestRows: baris disimpan dan dibaca dengan ID tetap dan urutan; isinya
// diperiksa terhadap versi format; header juga.
func TestRows(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")
	method := fieldID(t, f.cpFormat, template.AreaRow, "Method")

	if _, err := f.s.AddRow(ctx, f.viewer, d.ID, NewRow{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer menambah baris: %v", err)
	}
	d, err = f.s.AddRow(ctx, f.author, d.ID, NewRow{ProcessID: f.process, OperationNumber: "10", Content: map[string]any{product: "Solder paste height"}})
	if err != nil {
		t.Fatal(err)
	}
	d, err = f.s.AddRow(ctx, f.author, d.ID, NewRow{ProcessID: f.process, OperationNumber: "10", Content: map[string]any{method: "SPI 100%"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rows) != 2 || d.Rows[0].Position != 1 || d.Rows[1].Position != 2 || d.Rows[0].ProcessName != "SMT Printing" || d.Rows[0].Content[product] != "Solder paste height" {
		t.Fatalf("baris: %+v", d.Rows)
	}
	first := d.Rows[0].ID

	// Ubah isi: ID tetap.
	d, err = f.s.UpdateRow(ctx, f.author, d.ID, first, RowUpdate{ProcessID: f.process, OperationNumber: "20", Content: map[string]any{product: "Paste height 120 µm"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Rows[0].ID != first || d.Rows[0].OperationNumber != "20" || d.Rows[0].Content[product] != "Paste height 120 µm" {
		t.Errorf("sesudah ubah: %+v", d.Rows[0])
	}

	// Isi yang tidak cocok dengan format ditolak.
	if _, err := f.s.AddRow(ctx, f.author, d.ID, NewRow{Content: map[string]any{"bukan-field": "x"}}); !slices.Equal(codes(t, err), []string{"bukan-field:not_allowed"}) {
		t.Errorf("field tidak dikenal: %v", err)
	}
	if _, err := f.s.AddRow(ctx, f.author, d.ID, NewRow{ProcessID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"}); !slices.Equal(codes(t, err), []string{"process_id:invalid_format"}) {
		t.Errorf("proses tidak ada: %v", err)
	}

	// Header.
	line := fieldID(t, f.cpFormat, template.AreaHeader, "Core Team")
	d, err = f.s.SetHeader(ctx, f.author, d.ID, map[string]any{line: "Process Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Header[line] != "Process Engineer" {
		t.Errorf("header: %+v", d.Header)
	}

	// Baca ulang dari database.
	again, err := f.s.Get(ctx, f.author, d.ID)
	if err != nil || len(again.Rows) != 2 || again.Rows[0].ID != first || again.Header[line] != "Process Engineer" {
		t.Errorf("baca ulang: %+v, %v", again, err)
	}
	// Viewer tidak membaca draft (L-4).
	if _, err := f.s.Get(ctx, f.viewer, d.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer membaca draft: %v", err)
	}
}

// TestDocumentGuards: role aplikasi tidak bisa menghapus dokumen atau
// mengubah identitasnya.
func TestDocumentGuards(t *testing.T) {
	f := newFixture(t)
	for _, stmt := range []string{
		"DELETE FROM documents WHERE false",
		"UPDATE documents SET document_number_id = document_number_id WHERE false",
		"UPDATE documents SET customer_id = customer_id WHERE false",
		"UPDATE document_rows SET document_id = document_id WHERE false",
	} {
		_, err := f.pool.Exec(t.Context(), stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: %v, ingin permission denied", stmt, err)
		}
	}
}

// TestDeleteRow (M4.2): baris draft bisa dihapus; urutan baris sesudahnya
// bergeser naik, dan ID baris lain tetap.
func TestDeleteRow(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"10", "20", "30"} {
		if d, err = f.s.AddRow(ctx, f.author, d.ID, NewRow{OperationNumber: op}); err != nil {
			t.Fatal(err)
		}
	}
	middle, last := d.Rows[1].ID, d.Rows[2].ID
	if _, err := f.s.DeleteRow(ctx, f.viewer, d.ID, middle); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer menghapus baris: %v", err)
	}
	d, err = f.s.DeleteRow(ctx, f.author, d.ID, middle)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rows) != 2 || d.Rows[1].ID != last || d.Rows[1].Position != 2 || d.Rows[1].OperationNumber != "30" {
		t.Errorf("sesudah hapus: %+v", d.Rows)
	}
	if _, err := f.s.DeleteRow(ctx, f.author, d.ID, middle); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("hapus baris yang sudah tidak ada: %v", err)
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "document.row_removed" || details != `{"operation_number":"20","position":"2"}` {
		t.Errorf("audit: %s %s", action, details)
	}
}

// TestDocumentSet (M4.2): dokumen dari nomor yang didaftarkan sebagai set
// saling mengenal, supaya workbook bisa menampilkan tab PFD, PFMEA, dan CP.
func TestDocumentSet(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	reg, err := f.numbers.RegisterSet(ctx, f.author, numbering.NewSet{CustomerID: f.custAB, IdempotencyKey: "set-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, n := range reg.Numbers {
		if n.Status != "reserved" {
			continue
		}
		d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: n.ID, ModelIDs: []string{f.k125}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, d.ID)
	}
	d, err := f.s.Get(ctx, f.author, ids[1])
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, m := range d.Set {
		keys = append(keys, m.DocumentType.Key)
	}
	if !slices.Equal(keys, []string{"pfd", "pfmea", "cp"}) {
		t.Errorf("anggota set: %v", keys)
	}
	// Dokumen dari nomor tunggal hanya mengenal dirinya sendiri.
	single, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	if len(single.Set) != 1 || single.Set[0].ID != single.ID {
		t.Errorf("set dokumen tunggal: %+v", single.Set)
	}
}
