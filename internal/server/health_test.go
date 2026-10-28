package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
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

	// Tampilan web tidak dipakai di test ini, jadi newTestServer memakai
	// sistem file kosong dan database palsu yang sehat.
	srv := newTestServer(t, nil, Deps{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := srv.do(tt.method, tt.path, "")

			if rec.Code != tt.wantStatus {
				t.Errorf("status %d, seharusnya %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestHealthBody memeriksa header dan isi JSON dari GET /health.
func TestHealthBody(t *testing.T) {
	rec := newTestServer(t, nil, Deps{}).do(http.MethodGet, "/health", "")

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, seharusnya %q", got, "application/json")
	}
	var body Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi respons bukan JSON yang valid: %v", err)
	}
	if body.Status != HealthStatusOk || body.Database != HealthDatabaseOk {
		t.Errorf("isi %+v, seharusnya status ok dan database ok", body)
	}
}

// TestHealthDatabaseDown: server tetap menjawab, tapi melaporkan database
// tidak bisa dihubungi. Detail error hanya masuk log server (R-API-13).
func TestHealthDatabaseDown(t *testing.T) {
	srv := newTestServer(t, nil, Deps{DB: fakePinger{err: errors.New("dial tcp 127.0.0.1:5432: connection refused")}})

	rec := srv.do(http.MethodGet, "/health", "")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d, seharusnya 503", rec.Code)
	}
	var body Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi respons bukan JSON yang valid: %v", err)
	}
	if body.Status != HealthStatusUnavailable || body.Database != HealthDatabaseUnreachable {
		t.Errorf("isi %+v, seharusnya status unavailable dan database unreachable", body)
	}
	if strings.Contains(rec.Body.String(), "dial tcp") {
		t.Errorf("detail teknis bocor ke jawaban: %s", rec.Body.String())
	}
	log := srv.log.String()
	if !strings.Contains(log, "dial tcp") || !strings.Contains(log, rec.Header().Get("X-Request-Id")) {
		t.Errorf("log server harus memuat detail error dan request ID, dapat:\n%s", log)
	}
}
