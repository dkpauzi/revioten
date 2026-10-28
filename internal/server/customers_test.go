package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/problem"
)

// waktuUji adalah waktu tetap supaya isi JSON bisa dibandingkan persis.
var waktuUji = time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)

// sampleCustomer adalah customer contoh dengan riwayat dua kode.
var sampleCustomer = customer.Customer{
	ID: "id-ab", Code: "CD", Name: "Example Customer", ProjectGroupID: "gr-1", ProjectGroupCode: "GR1",
	Active: true, CreatedAt: waktuUji,
	Codes: []customer.CodePeriod{
		{Code: "AB", ValidFrom: waktuUji, ValidUntil: &waktuUji2},
		{Code: "CD", ValidFrom: waktuUji2},
	},
}

var waktuUji2 = waktuUji.Add(24 * time.Hour)

// sampleAPICustomer adalah sampleCustomer dalam bentuk kontrak API.
var sampleAPICustomer = Customer{
	Id: "id-ab", Code: "CD", Name: "Example Customer", Active: true, CreatedAt: waktuUji,
	ProjectGroup: &ProjectGroupRef{Id: "gr-1", Code: "GR1"},
	Codes: []CustomerCode{
		{Code: "AB", ValidFrom: waktuUji, ValidUntil: &waktuUji2},
		{Code: "CD", ValidFrom: waktuUji2},
	},
}

func TestListCustomers(t *testing.T) {
	// Customer lama tanpa Project Group tetap tampil, tanpa field project_group.
	legacy := customer.Customer{ID: "id-ee", Code: "EE", Name: "Customer E", Active: true, CreatedAt: waktuUji,
		Codes: []customer.CodePeriod{{Code: "EE", ValidFrom: waktuUji}}}
	fake := &fakeCustomers{list: []customer.Customer{sampleCustomer, legacy}}
	rec := newTestServer(t, nil, Deps{Customers: fake}).do(http.MethodGet, "/api/v1/customers", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, ingin 200: %s", rec.Code, rec.Body.String())
	}
	var body CustomerList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	want := []Customer{sampleAPICustomer, {Id: "id-ee", Code: "EE", Name: "Customer E", Active: true, CreatedAt: waktuUji,
		Codes: []CustomerCode{{Code: "EE", ValidFrom: waktuUji}}}}
	if !reflect.DeepEqual(body.Items, want) {
		t.Errorf("items %+v, ingin %+v", body.Items, want)
	}
	// Waktu ditulis dalam UTC dengan format ISO 8601 (R-API-13).
	if !strings.Contains(rec.Body.String(), `"created_at":"2026-10-28T03:00:00Z"`) {
		t.Errorf("format created_at salah: %s", rec.Body.String())
	}
}

func TestListCustomersEmpty(t *testing.T) {
	rec := newTestServer(t, nil, Deps{}).do(http.MethodGet, "/api/v1/customers", "")

	// Daftar kosong ditulis [], bukan null, supaya klien tidak perlu kasus khusus.
	if got := strings.TrimSpace(rec.Body.String()); got != `{"items":[]}` {
		t.Errorf("isi %s, ingin {\"items\":[]}", got)
	}
}

func TestCreateCustomer(t *testing.T) {
	fake := &fakeCustomers{created: sampleCustomer}
	srv := newTestServer(t, nil, Deps{Customers: fake})

	rec := srv.do(http.MethodPost, "/api/v1/customers", `{"code":" ab ","name":"Example Customer","project_group_id":"gr-1"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, ingin 201: %s", rec.Code, rec.Body.String())
	}
	// Adapter HTTP meneruskan input apa adanya; merapikan adalah tugas domain.
	want := customer.NewCustomer{Code: " ab ", Name: "Example Customer", ProjectGroupID: "gr-1"}
	if fake.gotInput == nil || *fake.gotInput != want {
		t.Errorf("input ke service %+v, ingin %+v", fake.gotInput, want)
	}
	// Aktor dari sesi diteruskan ke service, yang memeriksa izinnya (M1.2).
	if fake.gotActor != testActor {
		t.Errorf("aktor ke service %+v, ingin %+v", fake.gotActor, testActor)
	}
	var body Customer
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	if !reflect.DeepEqual(body, sampleAPICustomer) {
		t.Errorf("isi %+v, ingin %+v", body, sampleAPICustomer)
	}
}

// TestUpdateCustomerAndCode: ubah customer dan ganti kode diteruskan ke
// service dengan ID dari alamat.
func TestUpdateCustomerAndCode(t *testing.T) {
	fake := &fakeCustomers{created: sampleCustomer}
	srv := newTestServer(t, nil, Deps{Customers: fake})

	rec := srv.do(http.MethodPut, "/api/v1/customers/id-ab", `{"name":"Baru","project_group_id":"gr-2","active":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ubah: status %d: %s", rec.Code, rec.Body.String())
	}
	if fake.gotID != "id-ab" || fake.gotUpdate == nil ||
		*fake.gotUpdate != (customer.CustomerUpdate{Name: "Baru", ProjectGroupID: "gr-2", Active: false}) {
		t.Errorf("Update menerima %q %+v", fake.gotID, fake.gotUpdate)
	}

	fake.gotID = ""
	rec = srv.do(http.MethodPost, "/api/v1/customers/id-ab/code", `{"code":"cd"}`)
	if rec.Code != http.StatusOK || fake.gotID != "id-ab" || fake.gotCode != "cd" {
		t.Errorf("ganti kode: status %d, ID %q, kode %q", rec.Code, fake.gotID, fake.gotCode)
	}
}

// TestProjectGroupEndpoints: daftar, tambah, dan ubah Project Group.
func TestProjectGroupEndpoints(t *testing.T) {
	gr1 := customer.ProjectGroup{ID: "gr-1", Code: "GR1", Name: "Project Group 1", CreatedAt: waktuUji}
	fake := &fakeCustomers{groups: []customer.ProjectGroup{gr1}, group: gr1}
	srv := newTestServer(t, nil, Deps{Customers: fake})
	apiGR1 := ProjectGroup{Id: "gr-1", Code: "GR1", Name: "Project Group 1", CreatedAt: waktuUji}

	var list ProjectGroupList
	rec := srv.do(http.MethodGet, "/api/v1/project-groups", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil || !slices.Equal(list.Items, []ProjectGroup{apiGR1}) {
		t.Errorf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}

	rec = srv.do(http.MethodPost, "/api/v1/project-groups", `{"code":"gr1","name":"Project Group 1"}`)
	if rec.Code != http.StatusCreated || *fake.gotGroup != (customer.ProjectGroupInput{Code: "gr1", Name: "Project Group 1"}) {
		t.Errorf("tambah: status %d, input %+v", rec.Code, fake.gotGroup)
	}

	rec = srv.do(http.MethodPut, "/api/v1/project-groups/gr-1", `{"code":"GR01","name":"Satu"}`)
	if rec.Code != http.StatusOK || fake.gotID != "gr-1" || *fake.gotGroup != (customer.ProjectGroupInput{Code: "GR01", Name: "Satu"}) {
		t.Errorf("ubah: status %d, ID %q, input %+v", rec.Code, fake.gotID, fake.gotGroup)
	}
}

func TestCreateCustomerRejected(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    ErrorCode
		wantDetails []ErrorDetail
	}{
		{
			name: "input tidak valid",
			err: &problem.ValidationError{Problems: []problem.Problem{
				{Field: "code", Code: problem.Required, Message: "Enter the customer code."},
				{Field: "name", Code: problem.TooLong, Message: "Use at most 200 characters."},
			}},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   ErrorCodeValidationFailed,
			wantDetails: []ErrorDetail{
				{Field: "code", Code: ErrorDetailCodeRequired, Message: "Enter the customer code."},
				{Field: "name", Code: ErrorDetailCodeTooLong, Message: "Use at most 200 characters."},
			},
		},
		{
			name: "kode sudah dipakai",
			err: &problem.ConflictError{Problems: []problem.Problem{
				{Field: "code", Code: problem.Taken, Message: "This code is already used by another customer."},
			}},
			wantStatus: http.StatusConflict,
			wantCode:   ErrorCodeConflict,
			wantDetails: []ErrorDetail{
				{Field: "code", Code: ErrorDetailCodeTaken, Message: "This code is already used by another customer."},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, nil, Deps{Customers: &fakeCustomers{createErr: tt.err}})

			rec := srv.do(http.MethodPost, "/api/v1/customers", `{"code":"","name":"X"}`)

			if rec.Code != tt.wantStatus {
				t.Errorf("status %d, ingin %d", rec.Code, tt.wantStatus)
			}
			body := decodeError(t, rec)
			if body.Code != tt.wantCode {
				t.Errorf("code %q, ingin %q", body.Code, tt.wantCode)
			}
			if body.Details == nil || !slices.Equal(*body.Details, tt.wantDetails) {
				t.Errorf("details %+v, ingin %+v", body.Details, tt.wantDetails)
			}
		})
	}
}

// TestProblemCodesMatchContract: setiap kode masalah di domain harus ada di
// enum ErrorDetail.code di kontrak, supaya klien bisa menerjemahkannya.
func TestProblemCodesMatchContract(t *testing.T) {
	for _, code := range problem.Codes {
		if !ErrorDetailCode(code).Valid() {
			t.Errorf("kode masalah %q tidak ada di kontrak API", code)
		}
	}
}
