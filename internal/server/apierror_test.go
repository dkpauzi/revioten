package server

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestAPIErrorFormat: semua kegagalan di API memakai satu format error JSON
// (R-API-13), termasuk alamat dan method yang tidak ada di kontrak.
func TestAPIErrorFormat(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   ErrorCode
		wantAllow  string
	}{
		{"alamat API yang tidak ada", http.MethodGet, "/api/v1/tidak-ada", "", http.StatusNotFound, ErrorCodeNotFound, ""},
		{"method yang tidak ada di kontrak", http.MethodDelete, "/api/v1/customers", "", http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed, "GET, HEAD, POST"},
		{"POST ke /health", http.MethodPost, "/health", "", http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed, "GET, HEAD"},
		{"body bukan JSON", http.MethodPost, "/api/v1/customers", "{bukan json", http.StatusBadRequest, ErrorCodeInvalidRequest, ""},
		{"body kosong", http.MethodPost, "/api/v1/customers", "", http.StatusBadRequest, ErrorCodeInvalidRequest, ""},
		{"body salah tipe", http.MethodPost, "/api/v1/customers", `{"code": 12, "name": "X"}`, http.StatusBadRequest, ErrorCodeInvalidRequest, ""},
		{"body terlalu besar", http.MethodPost, "/api/v1/customers", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusRequestEntityTooLarge, ErrorCodeInvalidRequest, ""},
	}

	srv := newTestServer(t, testUI(), Deps{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Request ID dibuat tetap. Request ID acak bisa kebetulan memuat
			// teks seperti "EOF", sehingga pemeriksaan kebocoran di bawah
			// salah menandainya.
			rec := srv.do(tt.method, tt.path, tt.body, "X-Request-Id", "uji-format")

			if rec.Code != tt.wantStatus {
				t.Errorf("status %d, ingin %d", rec.Code, tt.wantStatus)
			}
			body := decodeError(t, rec)
			if body.Code != tt.wantCode {
				t.Errorf("code %q, ingin %q", body.Code, tt.wantCode)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("header Allow %q, ingin %q", got, tt.wantAllow)
			}
			// Pesan bawaan Go seperti "invalid character" tidak boleh sampai ke pengguna.
			for _, leak := range []string{"invalid character", "json:", "EOF", "http:"} {
				if strings.Contains(rec.Body.String(), leak) {
					t.Errorf("detail teknis %q bocor ke jawaban: %s", leak, rec.Body.String())
				}
			}
		})
	}
}

// TestAPIErrorHidesDetails: error tak terduga dijawab 500 tanpa detail teknis.
// Detailnya dicatat di log server bersama request ID (R-API-13, E-18).
func TestAPIErrorHidesDetails(t *testing.T) {
	broken := &fakeCustomers{listErr: errors.New("koneksi putus ke 10.0.0.9:5432")}
	srv := newTestServer(t, nil, Deps{Customers: broken})

	rec := srv.do(http.MethodGet, "/api/v1/customers", "", "X-Request-Id", "uji-500")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d, ingin 500", rec.Code)
	}
	body := decodeError(t, rec)
	if body.Code != ErrorCodeInternalError {
		t.Errorf("code %q, ingin internal_error", body.Code)
	}
	if strings.Contains(rec.Body.String(), "koneksi putus") || strings.Contains(rec.Body.String(), "10.0.0.9") {
		t.Errorf("detail teknis bocor ke jawaban: %s", rec.Body.String())
	}
	log := srv.log.String()
	if !strings.Contains(log, "koneksi putus") || !strings.Contains(log, "uji-500") {
		t.Errorf("log server harus memuat detail error dan request ID, dapat:\n%s", log)
	}
}
