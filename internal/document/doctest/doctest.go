// Package doctest menyiapkan data untuk test yang butuh dokumen nyata (Tahap
// 5): customer dengan model, form set AIAG 4th yang sudah terbit, akun, dan
// set dokumen PFD, PFMEA, dan CP. Hanya dipakai file _test.go.
package doctest

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/template"
)

// Env adalah data dasar satu test. Customer AB punya model K-125 dan K-200;
// customer CD punya model B-1. Customer AB dan CD ada di Project Group GR1.
type Env struct {
	Pool    *pgxpool.Pool
	Q       *db.Queries
	Checker *access.Checker
	Docs    *document.Service
	Numbers *numbering.Service

	Admin  access.Actor
	Author access.Actor // Penyusun untuk semua customer
	Viewer access.Actor // akun kerja tanpa peran

	GroupID      string
	CustomerAB   string
	CustomerCD   string
	ModelK125    string
	ModelK200    string
	ModelB1      string
	ProcessSMT   string // proses master "SMT Printing"
	CPFormat     template.FormatVersion
	PFMEAFormat  template.FormatVersion
	CustomerCode map[string]string // ID customer ke kode
}

// NewActor membuat akun beserta orangnya dan memberi peran.
func NewActor(t *testing.T, pool *pgxpool.Pool, login, kind string, grants ...access.RoleGrant) access.Actor {
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

// All memberi satu peran untuk semua customer.
func All(role string) access.RoleGrant { return access.RoleGrant{Role: role, Scope: access.ScopeAll} }

// Setup menyiapkan database baru dengan data dasar.
func Setup(t *testing.T) Env {
	t.Helper()
	ctx := t.Context()
	pool := dbtest.NewPool(t)
	checker := access.NewChecker(pool)
	e := Env{
		Pool: pool, Q: db.New(pool), Checker: checker,
		Docs: document.NewService(pool, checker), Numbers: numbering.NewService(pool, checker),
		CustomerCode: map[string]string{},
	}
	e.Admin = NewActor(t, pool, "admin.uji", access.AccountAdmin, All(access.RoleAdmin))
	e.Author = NewActor(t, pool, "penyusun.uji", access.AccountWork, All(access.RoleAuthor))
	e.Viewer = NewActor(t, pool, "viewer.uji", access.AccountWork)
	approver := NewActor(t, pool, "ppap.supervisor", access.AccountWork, All(access.RoleFormApprover))

	customers := customer.NewService(pool, checker)
	group, err := customers.CreateProjectGroup(ctx, e.Admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	e.GroupID = group.ID
	newCustomer := func(code string) string {
		c, err := customers.Create(ctx, e.Admin, customer.NewCustomer{Code: code, Name: "Customer " + code, ProjectGroupID: group.ID})
		if err != nil {
			t.Fatal(err)
		}
		e.CustomerCode[c.ID] = code
		return c.ID
	}
	model := func(customerID, name string) string {
		id, err := e.Q.CreateModel(ctx, db.CreateModelParams{CustomerID: customerID, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	e.CustomerAB, e.CustomerCD = newCustomer("AB"), newCustomer("CD")
	e.ModelK125, e.ModelK200, e.ModelB1 = model(e.CustomerAB, "K-125"), model(e.CustomerAB, "K-200"), model(e.CustomerCD, "B-1")
	if e.ProcessSMT, err = e.Q.CreateProcess(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.Q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: e.ProcessSMT, Kind: "display", Name: "SMT Printing"}); err != nil {
		t.Fatal(err)
	}

	templates := template.NewService(pool, checker)
	sets, err := templates.ListFormSets(ctx, e.Admin)
	if err != nil {
		t.Fatal(err)
	}
	v1 := sets.Sets[0].Versions[0]
	for _, id := range []string{v1.PFD.FormatVersionID, v1.PFMEA.FormatVersionID, v1.CP.FormatVersionID} {
		if _, err := templates.PublishFormatVersion(ctx, e.Admin, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := templates.PublishFormSetVersion(ctx, approver, v1.ID); err != nil {
		t.Fatal(err)
	}
	if e.CPFormat, err = templates.GetFormatVersion(ctx, e.Admin, v1.CP.FormatVersionID); err != nil {
		t.Fatal(err)
	}
	if e.PFMEAFormat, err = templates.GetFormatVersion(ctx, e.Admin, v1.PFMEA.FormatVersionID); err != nil {
		t.Fatal(err)
	}
	return e
}

// NewSet mendaftarkan satu set nomor untuk customer dan membuat dokumen PFD,
// PFMEA, dan CP untuk satu model. Hasilnya menurut key jenis dokumen.
func (e Env) NewSet(t *testing.T, key, customerID, modelID string) map[string]document.Document {
	t.Helper()
	ctx := t.Context()
	reg, err := e.Numbers.RegisterSet(ctx, e.Author, numbering.NewSet{CustomerID: customerID, IdempotencyKey: key + "-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]document.Document{}
	for _, n := range reg.Numbers {
		if n.Status != "reserved" {
			continue
		}
		d, err := e.Docs.Create(ctx, e.Author, document.NewDocument{NumberID: n.ID, ModelIDs: []string{modelID}})
		if err != nil {
			t.Fatal(err)
		}
		out[d.DocumentType.Key] = d
	}
	return out
}

// FieldID mencari ID kolom baris format menurut label.
func FieldID(t *testing.T, v template.FormatVersion, label string) string {
	t.Helper()
	for _, f := range v.Fields {
		if f.Area == template.AreaRow && f.Label == label {
			return f.ID
		}
	}
	t.Fatalf("kolom %q tidak ada", label)
	return ""
}

// AttachScan mencatat scan bertanda tangan palsu pada draft terbuka dokumen
// (M6.7), supaya test bisa mencatat rilis tanpa berkas sungguhan.
func AttachScan(t *testing.T, pool *pgxpool.Pool, documentID string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `UPDATE document_revisions
		SET scan_file_name = 'signed.pdf', scan_content_type = 'application/pdf', scan_size_bytes = 10,
		    scan_sha256 = repeat('a', 64), scan_uploaded_by_person_id = created_by_person_id, scan_uploaded_at = now()
		WHERE document_id = $1 AND status = 'draft'`, documentID)
	if err != nil {
		t.Fatal(err)
	}
}
