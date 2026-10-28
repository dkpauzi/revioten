package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// TestHealthStatus memeriksa kode status untuk beberapa permintaan sekaligus.
// Bentuk ini disebut table-driven test: setiap baris tabel adalah satu kasus.
func TestHealthStatus(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "GET /health berhasil", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK},
		{name: "POST /health ditolak", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed},
		{name: "API yang tidak ada", method: http.MethodGet, path: "/api/v1/tidak-ada", wantStatus: http.StatusNotFound},
	}

	// Tampilan web tidak dipakai di test ini, jadi cukup sistem file kosong.
	handler := NewHandler(fstest.MapFS{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// httptest membuat permintaan palsu dan merekam jawabannya,
			// tanpa membuka port jaringan.
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status %d, seharusnya %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestHealthBody memeriksa header dan isi JSON dari GET /health.
func TestHealthBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	NewHandler(fstest.MapFS{}).ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, seharusnya %q", got, "application/json")
	}

	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi respons bukan JSON yang valid: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status di JSON %q, seharusnya %q", body.Status, "ok")
	}
}
