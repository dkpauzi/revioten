// Package masterimport mengimpor Project Group, customer, group model, model,
// dan part number dari satu berkas Excel (I5). Permintaan pemilik (24
// November 2026): sesudah database direset, master data cukup diunggah lagi,
// supaya import akun (I1) dan import nomor lama (I4) tidak gagal karena
// customer atau model belum ada.
//
// Berkas punya dua sheet: Customers (Project Group dan customer) dan Models
// (group model, model, dan part number). Preview melaporkan setiap baris
// tanpa menyimpan. Import menyimpan baris demi baris lewat layanan master
// data yang biasa (customer.Service dan product.Service), jadi aturan, hak,
// dan audit log sama dengan mengisi layar satu per satu. Data yang sudah ada
// dilewati, sehingga berkas yang sama aman diunggah lagi.
package masterimport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/product"
)

// Batas berkas import.
const (
	MaxBytes = 5 << 20 // 5 MB
	MaxRows  = 2000    // per sheet
)

// Hasil satu baris.
const (
	RowNew      = "new"      // ada yang dibuat (bila Import)
	RowExisting = "existing" // semua sudah ada, tidak diubah
	RowInvalid  = "invalid"  // ada isian yang salah; lihat Problems
)

// Yang dibuat satu baris, untuk laporan.
const (
	CreatesProjectGroup = "project_group"
	CreatesCustomer     = "customer"
	CreatesModelGroup   = "model_group"
	CreatesModel        = "model"
	CreatesPartNumber   = "part_number"
)

// Nama sheet dan judul kolom template. Urutan kolom di berkas bebas.
const (
	SheetCustomers = "Customers"
	SheetModels    = "Models"
)

var (
	customerHeaders = []string{"Project group code", "Project group name", "Customer code", "Customer name"}
	modelHeaders    = []string{"Customer code", "Model group", "Model", "Part number", "Part name"}
)

// Row adalah satu baris salah satu sheet beserta hasilnya.
type Row struct {
	Sheet    string
	Row      int               // nomor baris di Excel, mulai dari 2
	Cells    map[string]string // isi kolom menurut judulnya, sudah dirapikan
	Status   string
	Creates  []string // yang dibuat (Import) atau akan dibuat (Preview)
	Problems []problem.Problem
}

// Report adalah laporan satu Preview atau Import.
type Report struct {
	Customers []Row
	Models    []Row
	New       int
	Existing  int
	Invalid   int
	Committed bool
}

// Service menjalankan import lewat layanan master data.
type Service struct {
	authz     access.Authorizer
	customers *customer.Service
	products  *product.Service
}

// NewService membuat Service.
func NewService(authz access.Authorizer, customers *customer.Service, products *product.Service) *Service {
	return &Service{authz: authz, customers: customers, products: products}
}

// state adalah master data yang sudah ada, ditambah yang dibuat (atau akan
// dibuat) baris sebelumnya, supaya baris berikutnya bisa merujuknya.
type state struct {
	groups      map[string]string            // kode Project Group -> ID ("" bila baru di Preview)
	customers   map[string]string            // kode customer -> ID ("" bila baru di Preview)
	modelGroups map[string]map[string]string // kode customer -> nama group model -> ID
	models      map[string]map[string]string // kode customer -> nama model -> ID
	parts       map[string]map[string]string // kode customer -> part number -> nama model
}

// Template membuat berkas Excel template (E-11): sheet Customers, Models, dan
// Help. Tanpa baris contoh, supaya tidak ada data contoh yang ikut terimpor.
func (s *Service) Template(ctx context.Context, actor access.Actor) ([]byte, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	defer f.Close()
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	header := func(sheet string, cols []string, widths []float64) {
		for i, h := range cols {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			_ = f.SetCellStr(sheet, cell, h)
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = f.SetColWidth(sheet, col, col, widths[i])
		}
		last, _ := excelize.CoordinatesToCellName(len(cols), 1)
		_ = f.SetCellStyle(sheet, "A1", last, bold)
	}
	if err := f.SetSheetName("Sheet1", SheetCustomers); err != nil {
		return nil, err
	}
	header(SheetCustomers, customerHeaders, []float64{20, 30, 16, 36})
	if _, err := f.NewSheet(SheetModels); err != nil {
		return nil, err
	}
	header(SheetModels, modelHeaders, []float64{16, 24, 28, 24, 36})
	if _, err := f.NewSheet("Help"); err != nil {
		return nil, err
	}
	help := [][]any{
		{"Sheet and column", "What to fill in"},
		{"Customers", "One customer per row."},
		{"Project group code", "Required, for example GR1. A project group that does not exist yet is created."},
		{"Project group name", "Required only for a new project group."},
		{"Customer code", "Required: the two-letter code, for example TR. A customer that already exists is skipped."},
		{"Customer name", "Required for a new customer."},
		{"Models", "One part number per row. Repeat the model on the next row for its next part number."},
		{"Customer code", "Required. The customer may be new in the Customers sheet of the same file."},
		{"Model group", "Optional. A model group that does not exist yet is created."},
		{"Model", "Required. A model that already exists is not changed."},
		{"Part number", "Optional. A part number that already exists is skipped."},
		{"Part name", "Optional."},
	}
	for i, line := range help {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Help", cell, &line)
	}
	_ = f.SetCellStyle("Help", "A1", "B1", bold)
	_ = f.SetColWidth("Help", "A", "A", 22)
	_ = f.SetColWidth("Help", "B", "B", 100)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("menulis template master data: %w", err)
	}
	return buf.Bytes(), nil
}

// Import membaca berkas (E-11). commit false memberi laporan saja (Preview);
// commit true juga menyimpan baris yang benar, satu baris satu transaksi.
func (s *Service) Import(ctx context.Context, actor access.Actor, data []byte, commit bool) (Report, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return Report{}, err
	}
	customersRows, modelRows, err := readBook(data)
	if err != nil {
		return Report{}, err
	}
	st, err := s.load(ctx, actor)
	if err != nil {
		return Report{}, err
	}
	report := Report{Customers: customersRows, Models: modelRows, Committed: commit}
	for i := range report.Customers {
		if err := s.customerRow(ctx, actor, st, &report.Customers[i], commit); err != nil {
			return Report{}, err
		}
	}
	for i := range report.Models {
		if err := s.modelRow(ctx, actor, st, &report.Models[i], commit); err != nil {
			return Report{}, err
		}
	}
	for _, list := range [][]Row{report.Customers, report.Models} {
		for _, r := range list {
			switch r.Status {
			case RowNew:
				report.New++
			case RowExisting:
				report.Existing++
			default:
				report.Invalid++
			}
		}
	}
	return report, nil
}

// load membaca master data yang sudah ada.
func (s *Service) load(ctx context.Context, actor access.Actor) (*state, error) {
	st := &state{groups: map[string]string{}, customers: map[string]string{},
		modelGroups: map[string]map[string]string{}, models: map[string]map[string]string{}, parts: map[string]map[string]string{}}
	groups, err := s.customers.ListProjectGroups(ctx, actor)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		st.groups[g.Code] = g.ID
	}
	list, err := s.customers.List(ctx, actor)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		st.customers[c.Code] = c.ID
		if err := s.loadCustomer(ctx, actor, st, c.Code, c.ID); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// loadCustomer membaca group model, model, dan part number satu customer.
func (s *Service) loadCustomer(ctx context.Context, actor access.Actor, st *state, code, id string) error {
	st.modelGroups[code], st.models[code], st.parts[code] = map[string]string{}, map[string]string{}, map[string]string{}
	if id == "" {
		return nil
	}
	groups, err := s.products.ListModelGroups(ctx, actor, id)
	if err != nil {
		return err
	}
	for _, g := range groups {
		st.modelGroups[code][g.Name] = g.ID
	}
	models, err := s.products.ListModels(ctx, actor, id)
	if err != nil {
		return err
	}
	for _, m := range models {
		st.models[code][m.Name] = m.ID
	}
	parts, err := s.products.ListPartNumbers(ctx, actor, id)
	if err != nil {
		return err
	}
	for _, p := range parts {
		st.parts[code][p.Number] = p.ModelName
	}
	return nil
}

// customerRow memeriksa, dan bila commit menyimpan, satu baris sheet Customers.
func (s *Service) customerRow(ctx context.Context, actor access.Actor, st *state, r *Row, commit bool) error {
	code := customer.NormalizeCode(r.Cells["customer code"])
	if _, exists := st.customers[code]; exists && code != "" {
		// Customer yang sudah ada dilewati tanpa memeriksa kolom lain.
		r.Status = RowExisting
		return nil
	}
	group, list := customer.CheckNewProjectGroup(customer.ProjectGroupInput{Code: r.Cells["project group code"], Name: r.Cells["project group name"]})
	groupID, groupExists := st.groups[group.Code]
	if groupExists {
		// Project Group yang sudah ada cukup ditunjuk kodenya; namanya tidak dipakai.
		list = nil
	} else {
		r.Creates = append(r.Creates, CreatesProjectGroup)
	}
	r.Problems = append(r.Problems, renamed(list, map[string]string{"code": "project_group_code", "name": "project_group_name"})...)
	_, _, cList, err := s.customers.CheckNewCustomer(ctx, code, r.Cells["customer name"])
	if err != nil {
		return err
	}
	r.Problems = append(r.Problems, renamed(cList, map[string]string{"code": "customer_code", "name": "customer_name"})...)
	r.Creates = append(r.Creates, CreatesCustomer)
	if len(r.Problems) > 0 {
		r.Status, r.Creates = RowInvalid, nil
		return nil
	}
	customerExists := false
	var customerID string
	r.Status = RowNew
	if !commit {
		// Baris berikutnya boleh merujuk yang akan dibuat di sini.
		if !groupExists {
			st.groups[group.Code] = ""
		}
		if !customerExists {
			st.customers[code] = ""
			_ = s.loadCustomer(ctx, actor, st, code, "")
		}
		return nil
	}
	if !groupExists {
		g, err := s.customers.CreateProjectGroup(ctx, actor, customer.ProjectGroupInput{Code: group.Code, Name: r.Cells["project group name"]})
		if err != nil {
			return failed(r, err, map[string]string{"code": "project_group_code", "name": "project_group_name"})
		}
		groupID = g.ID
		st.groups[group.Code] = g.ID
	}
	if !customerExists {
		c, err := s.customers.Create(ctx, actor, customer.NewCustomer{Code: code, Name: r.Cells["customer name"], ProjectGroupID: groupID})
		if err != nil {
			return failed(r, err, map[string]string{"code": "customer_code", "name": "customer_name"})
		}
		customerID = c.ID
		st.customers[code] = c.ID
		if err := s.loadCustomer(ctx, actor, st, code, customerID); err != nil {
			return err
		}
	}
	return nil
}

// modelRow memeriksa, dan bila commit menyimpan, satu baris sheet Models.
func (s *Service) modelRow(ctx context.Context, actor access.Actor, st *state, r *Row, commit bool) error {
	code := customer.NormalizeCode(r.Cells["customer code"])
	customerID, known := st.customers[code]
	if code == "" {
		r.Problems = append(r.Problems, problem.Problem{Field: "customer_code", Code: problem.Required, Message: "Enter the customer code."})
	} else if !known {
		r.Problems = append(r.Problems, problem.Problem{Field: "customer_code", Code: problem.InvalidFormat,
			Message: "Customer " + code + " does not exist. Add it to the Customers sheet."})
	}
	groupName := strings.TrimSpace(r.Cells["model group"])
	if groupName != "" {
		if _, p := product.CheckName("model_group", groupName); p != nil {
			r.Problems = append(r.Problems, *p)
		}
	}
	modelName, p := product.CheckName("model", r.Cells["model"])
	if p != nil {
		r.Problems = append(r.Problems, *p)
	}
	partNumber, partName := strings.TrimSpace(r.Cells["part number"]), ""
	if partNumber != "" || strings.TrimSpace(r.Cells["part name"]) != "" {
		var list []problem.Problem
		partNumber, partName, list = product.CheckPartNumber(partNumber, r.Cells["part name"])
		r.Problems = append(r.Problems, list...)
	}
	if len(r.Problems) > 0 {
		r.Status = RowInvalid
		return nil
	}

	groupID, groupExists := st.modelGroups[code][groupName]
	modelID, modelExists := st.models[code][modelName]
	if groupName != "" && !groupExists && !modelExists {
		r.Creates = append(r.Creates, CreatesModelGroup)
	}
	if !modelExists {
		r.Creates = append(r.Creates, CreatesModel)
	}
	if partNumber != "" {
		owner, partExists := st.parts[code][partNumber]
		switch {
		case partExists && owner != modelName:
			r.Status, r.Creates = RowInvalid, nil
			r.Problems = append(r.Problems, problem.Problem{Field: "part_number", Code: problem.Taken,
				Message: "This part number already belongs to model " + owner + "."})
			return nil
		case !partExists:
			r.Creates = append(r.Creates, CreatesPartNumber)
		}
	}
	if len(r.Creates) == 0 {
		r.Status = RowExisting
		return nil
	}
	r.Status = RowNew
	if !commit {
		if groupName != "" && !groupExists && !modelExists {
			st.modelGroups[code][groupName] = ""
		}
		if !modelExists {
			st.models[code][modelName] = ""
		}
		if partNumber != "" {
			st.parts[code][partNumber] = modelName
		}
		return nil
	}
	if groupName != "" && !groupExists && !modelExists {
		g, err := s.products.CreateModelGroup(ctx, actor, product.NewModelGroup{CustomerID: customerID, Name: groupName})
		if err != nil {
			return failed(r, err, map[string]string{"name": "model_group"})
		}
		groupID = g.ID
		st.modelGroups[code][groupName] = g.ID
	}
	if !modelExists {
		m, err := s.products.CreateModel(ctx, actor, product.NewModel{CustomerID: customerID, Name: modelName, ModelGroupID: groupID})
		if err != nil {
			return failed(r, err, map[string]string{"name": "model"})
		}
		modelID = m.ID
		st.models[code][modelName] = m.ID
	}
	if partNumber != "" && st.parts[code][partNumber] == "" {
		if _, err := s.products.CreatePartNumber(ctx, actor, product.NewPartNumber{ModelID: modelID, Number: partNumber, Name: partName}); err != nil {
			return failed(r, err, map[string]string{"number": "part_number", "name": "part_name"})
		}
		st.parts[code][partNumber] = modelName
	}
	return nil
}

// failed mencatat penolakan layanan pada baris itu. Error lain (database)
// menghentikan import; baris yang sudah tersimpan tetap tersimpan.
func failed(r *Row, err error, fields map[string]string) error {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		r.Problems = append(r.Problems, renamed(invalid.Problems, fields)...)
	case errors.As(err, &conflict):
		r.Problems = append(r.Problems, renamed(conflict.Problems, fields)...)
	default:
		return err
	}
	r.Status, r.Creates = RowInvalid, nil
	return nil
}

// renamed mengganti nama field layanan menjadi nama kolom berkas.
func renamed(list []problem.Problem, fields map[string]string) []problem.Problem {
	out := make([]problem.Problem, 0, len(list))
	for _, p := range list {
		if name, ok := fields[p.Field]; ok {
			p.Field = name
		}
		out = append(out, p)
	}
	return out
}

// readBook membaca sheet Customers dan Models. Sheet yang tidak ada dianggap
// kosong, tetapi berkas tanpa keduanya ditolak.
func readBook(data []byte) ([]Row, []Row, error) {
	if len(data) == 0 {
		return nil, nil, fileProblem(problem.Required, "Choose the Excel file.")
	}
	if len(data) > MaxBytes {
		return nil, nil, fileProblem(problem.TooLong, "The Excel file can be at most 5 MB.")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fileProblem(problem.InvalidFormat, "Upload an Excel workbook (.xlsx). Save an old .xls file as .xlsx first.")
	}
	defer book.Close()
	customers, err := readSheet(book, SheetCustomers, customerHeaders, []string{"project group code", "customer code"})
	if err != nil {
		return nil, nil, err
	}
	models, err := readSheet(book, SheetModels, modelHeaders, []string{"customer code", "model"})
	if err != nil {
		return nil, nil, err
	}
	if customers == nil && models == nil {
		return nil, nil, fileProblem(problem.InvalidFormat, "The Excel file needs a Customers or Models sheet. Download the template.")
	}
	if len(customers)+len(models) == 0 {
		return nil, nil, fileProblem(problem.Required, "The Excel file has no rows to import.")
	}
	return customers, models, nil
}

// readSheet membaca satu sheet. Hasil nil berarti sheet itu tidak ada.
func readSheet(book *excelize.File, sheet string, headers, required []string) ([]Row, error) {
	index, err := book.GetSheetIndex(sheet)
	if err != nil || index < 0 {
		return nil, nil
	}
	lines, err := book.GetRows(sheet)
	if err != nil {
		return nil, fileProblem(problem.InvalidFormat, "The "+sheet+" sheet cannot be read.")
	}
	column := map[string]int{}
	if len(lines) > 0 {
		for i, h := range lines[0] {
			column[strings.ToLower(strings.TrimSpace(h))] = i
		}
	}
	for _, need := range required {
		if _, ok := column[need]; !ok {
			return nil, fileProblem(problem.InvalidFormat,
				"The first row of the "+sheet+" sheet must have the columns "+strings.Join(headers, ", ")+". Download the template.")
		}
	}
	rows := []Row{}
	for y, line := range lines[min(1, len(lines)):] {
		cells := map[string]string{}
		empty := true
		for _, h := range headers {
			key := strings.ToLower(h)
			if i, ok := column[key]; ok && i < len(line) {
				cells[key] = strings.TrimSpace(line[i])
				if cells[key] != "" {
					empty = false
				}
			}
		}
		if empty {
			continue
		}
		rows = append(rows, Row{Sheet: sheet, Row: y + 2, Cells: cells, Creates: []string{}})
	}
	if len(rows) > MaxRows {
		return nil, fileProblem(problem.TooLong, fmt.Sprintf("Import at most %d rows per sheet at a time.", MaxRows))
	}
	return rows, nil
}

func fileProblem(code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: code, Message: message}}}
}
