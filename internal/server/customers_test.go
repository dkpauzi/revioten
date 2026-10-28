package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/customer"
)

// waktuUji adalah waktu tetap supaya isi JSON bisa dibandingkan persis.
var waktuUji = time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)

func TestListCustomers(t *testing.T) {
	fake := &fakeCustomers{list: []customer.Customer{
		{ID: "id-a", Code: "AA", Name: "Customer A", CreatedAt: waktuUji},
		{ID: "id-b", Code: "BB", Name: "Customer B", CreatedAt: waktuUji},
	}}
	rec := newTestServer(t, nil, Deps{Customers: fake}).do(http.MethodGet, "/api/v1/customers", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, ingin 200: %s", rec.Code, rec.Body.String())
	}
	var body CustomerList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	want := []Customer{
		{Id: "id-a", Code: "AA", Name: "Customer A", CreatedAt: waktuUji},
		{Id: "id-b", Code: "BB", Name: "Customer B", CreatedAt: waktuUji},
	}
	if !slices.Equal(body.Items, want) {
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
	fake := &fakeCustomers{created: customer.Customer{ID: "id-ab", Code: "AB", Name: "Example Customer", CreatedAt: waktuUji}}
	srv := newTestServer(t, nil, Deps{Customers: fake})

	rec := srv.do(http.MethodPost, "/api/v1/customers", `{"code":" ab ","name":"Example Customer"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, ingin 201: %s", rec.Code, rec.Body.String())
	}
	// Adapter HTTP meneruskan input apa adanya; merapikan adalah tugas domain.
	if fake.gotInput == nil || *fake.gotInput != (customer.NewCustomer{Code: " ab ", Name: "Example Customer"}) {
		t.Errorf("input ke service %+v, ingin kode \" ab \" dan nama Example Customer", fake.gotInput)
	}
	var body Customer
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	want := Customer{Id: "id-ab", Code: "AB", Name: "Example Customer", CreatedAt: waktuUji}
	if body != want {
		t.Errorf("isi %+v, ingin %+v", body, want)
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
			err: &customer.ValidationError{Problems: []customer.Problem{
				{Field: "code", Code: customer.ProblemRequired, Message: "Enter the customer code."},
				{Field: "name", Code: customer.ProblemTooLong, Message: "Use at most 200 characters."},
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
			err: &customer.ConflictError{Problems: []customer.Problem{
				{Field: "code", Code: customer.ProblemTaken, Message: "This code is already used by another customer."},
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
	for _, code := range []string{
		customer.ProblemRequired,
		customer.ProblemInvalidFormat,
		customer.ProblemTooLong,
		customer.ProblemReserved,
		customer.ProblemTaken,
	} {
		if !ErrorDetailCode(code).Valid() {
			t.Errorf("kode masalah %q tidak ada di kontrak API", code)
		}
	}
}
