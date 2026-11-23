package masterimport

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/product"
)

// I5: PPAP Engineer mengimpor Project Group, customer, group model, model, dan
// part number dari satu berkas Excel. Nama fiktif.

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func newActor(t *testing.T, pool *pgxpool.Pool, login string, roles ...string) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{PersonID: person.ID, Kind: "work", Login: login, PasswordHash: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: r, Scope: access.ScopeAll}); err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{Type: access.ActorHuman, AccountID: acc.ID, AccountKind: "work", Login: login, PersonID: person.ID}
}

// book membuat berkas Excel dengan sheet dan baris yang diberikan; baris
// pertama setiap sheet adalah judul kolom.
func book(t *testing.T, sheets map[string][][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	first := true
	for name, rows := range sheets {
		if first {
			_ = f.SetSheetName("Sheet1", name)
			first = false
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		for y, row := range rows {
			for x, text := range row {
				cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
				_ = f.SetCellStr(name, cell, text)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func statuses(rows []Row) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Status)
	}
	return out
}

func codes(err error) []string {
	var v *problem.ValidationError
	if !errors.As(err, &v) {
		return nil
	}
	var out []string
	for _, p := range v.Problems {
		out = append(out, p.Field+":"+p.Code)
	}
	return out
}

func TestImportMasterData(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	checker := access.NewChecker(pool)
	customers := customer.NewService(pool, checker)
	products := product.NewService(pool, checker)
	s := NewService(checker, customers, products)
	engineer := newActor(t, pool, "ppap.uji", access.RolePPAPEngineer)
	viewer := newActor(t, pool, "lihat.uji")

	// Satu customer sudah ada, beserta satu model dan part number-nya.
	group, err := customers.CreateProjectGroup(ctx, engineer, customer.ProjectGroupInput{Code: "GR1", Name: "Group satu"})
	if err != nil {
		t.Fatal(err)
	}
	ab, err := customers.Create(ctx, engineer, customer.NewCustomer{Code: "AB", Name: "Customer AB", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	k1, err := products.CreateModel(ctx, engineer, product.NewModel{CustomerID: ab.ID, Name: "K-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := products.CreatePartNumber(ctx, engineer, product.NewPartNumber{ModelID: k1.ID, Number: "P-100"}); err != nil {
		t.Fatal(err)
	}

	data := book(t, map[string][][]string{
		SheetCustomers: {
			customerHeaders,
			{"GR1", "", "AB", "Customer AB"},          // sudah ada
			{"GR2", "Group dua", "tr", "Customer TR"}, // Project Group dan customer baru
			{"GR2", "", "CD", "Customer CD"},          // Project Group dari baris sebelumnya
			{"", "", "", ""},                          // kosong, diabaikan
			{"GR3", "", "EF", "Customer EF"},          // nama Project Group baru wajib
			{"GR1", "", "X1", "Kode salah"},           // kode customer bukan dua huruf
		},
		SheetModels: {
			modelHeaders,
			{"AB", "", "K-1", "P-100", ""},               // semua sudah ada
			{"AB", "", "K-1", "P-101", "Main board"},     // part number baru untuk model lama
			{"TR", "Tokai", "T-20", "SH-1", "PCB utama"}, // customer, group, model, part baru
			{"TR", "Tokai", "T-20", "SH-2", ""},          // part kedua model yang sama
			{"ZZ", "", "Z-1", "", ""},                    // customer tidak ada
			{"AB", "", "K-2", "P-100", ""},               // part number milik model lain
		},
	})

	if _, err := s.Import(ctx, viewer, data, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-11: %v", err)
	}
	if _, err := s.Import(ctx, engineer, []byte("bukan excel"), false); !slices.Equal(codes(err), []string{"file:invalid_format"}) {
		t.Errorf("bukan Excel: %v", err)
	}

	preview, err := s.Import(ctx, engineer, data, false)
	if err != nil {
		t.Fatal(err)
	}
	wantCustomers := []string{RowExisting, RowNew, RowNew, RowInvalid, RowInvalid}
	wantModels := []string{RowExisting, RowNew, RowNew, RowNew, RowInvalid, RowInvalid}
	if !slices.Equal(statuses(preview.Customers), wantCustomers) || !slices.Equal(statuses(preview.Models), wantModels) || preview.Committed {
		t.Fatalf("preview: %v %v", statuses(preview.Customers), statuses(preview.Models))
	}
	if !slices.Equal(preview.Customers[1].Creates, []string{CreatesProjectGroup, CreatesCustomer}) ||
		!slices.Equal(preview.Models[2].Creates, []string{CreatesModelGroup, CreatesModel, CreatesPartNumber}) ||
		!slices.Equal(preview.Models[3].Creates, []string{CreatesPartNumber}) {
		t.Errorf("yang akan dibuat: %v, %v, %v", preview.Customers[1].Creates, preview.Models[2].Creates, preview.Models[3].Creates)
	}
	if preview.Customers[3].Row != 6 || preview.Customers[3].Problems[0].Field != "project_group_name" {
		t.Errorf("baris EF: %+v", preview.Customers[3])
	}
	if list, _ := customers.List(ctx, engineer); len(list) != 1 {
		t.Fatalf("Preview menyimpan customer: %d", len(list))
	}

	done, err := s.Import(ctx, engineer, data, true)
	if err != nil {
		t.Fatal(err)
	}
	if !done.Committed || done.New != 5 || done.Existing != 2 || done.Invalid != 4 {
		t.Fatalf("import: new %d, existing %d, invalid %d", done.New, done.Existing, done.Invalid)
	}
	list, _ := customers.List(ctx, engineer)
	byCode := map[string]customer.Customer{}
	for _, c := range list {
		byCode[c.Code] = c
	}
	if byCode["TR"].ProjectGroupCode != "GR2" || byCode["CD"].ProjectGroupCode != "GR2" || len(list) != 3 {
		t.Fatalf("customer: %+v", list)
	}
	models, _ := products.ListModels(ctx, engineer, byCode["TR"].ID)
	if len(models) != 1 || models[0].Name != "T-20" || models[0].ModelGroupName != "Tokai" {
		t.Errorf("model TR: %+v", models)
	}
	parts, _ := products.ListPartNumbers(ctx, engineer, byCode["TR"].ID)
	if len(parts) != 2 {
		t.Errorf("part number TR: %+v", parts)
	}

	// Berkas yang sama diunggah lagi: yang benar sudah ada semua.
	again, err := s.Import(ctx, engineer, data, true)
	if err != nil || again.New != 0 || again.Existing != 7 {
		t.Errorf("import ulang: new %d, existing %d, %v", again.New, again.Existing, err)
	}
}

func TestMasterDataTemplate(t *testing.T) {
	pool := dbtest.NewPool(t)
	checker := access.NewChecker(pool)
	s := NewService(checker, customer.NewService(pool, checker), product.NewService(pool, checker))
	engineer := newActor(t, pool, "ppap.uji", access.RolePPAPEngineer)
	data, err := s.Template(t.Context(), engineer)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for sheet, want := range map[string][]string{SheetCustomers: customerHeaders, SheetModels: modelHeaders} {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) != 1 || !slices.Equal(rows[0], want) {
			t.Errorf("sheet %s: %v, %v", sheet, rows, err)
		}
	}
}
