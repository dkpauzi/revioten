package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// testUI meniru hasil build SvelteKit di memori, jadi test tidak butuh npm.
func testUI() fstest.MapFS {
	return fstest.MapFS{
		"200.html":              {Data: []byte("<html>spa</html>")},
		"robots.txt":            {Data: []byte("User-agent: *")},
		"_app/immutable/app.js": {Data: []byte("console.log('app')")},
	}
}

// TestSPA memeriksa cara server melayani tampilan web.
func TestSPA(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "beranda memakai halaman fallback", method: http.MethodGet, path: "/", wantStatus: http.StatusOK, wantBody: "<html>spa</html>"},
		{name: "halaman lain memakai halaman fallback", method: http.MethodGet, path: "/about", wantStatus: http.StatusOK, wantBody: "<html>spa</html>"},
		{name: "halaman bertingkat memakai halaman fallback", method: http.MethodGet, path: "/customers/12", wantStatus: http.StatusOK, wantBody: "<html>spa</html>"},
		{name: "file statis dikirim apa adanya", method: http.MethodGet, path: "/robots.txt", wantStatus: http.StatusOK, wantBody: "User-agent: *"},
		{name: "aset JavaScript dikirim apa adanya", method: http.MethodGet, path: "/_app/immutable/app.js", wantStatus: http.StatusOK, wantBody: "console.log('app')"},
		{name: "aset yang tidak ada", method: http.MethodGet, path: "/_app/immutable/hilang.js", wantStatus: http.StatusNotFound},
		{name: "file yang tidak ada", method: http.MethodGet, path: "/favicon.ico", wantStatus: http.StatusNotFound},
		{name: "API yang tidak ada tidak diganti halaman", method: http.MethodGet, path: "/api/v1/tidak-ada", wantStatus: http.StatusNotFound},
		{name: "POST ke halaman ditolak", method: http.MethodPost, path: "/about", wantStatus: http.StatusMethodNotAllowed},
	}

	handler := NewHandler(testUI())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d, seharusnya %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("isi %q, seharusnya %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestSPANotBuilt: bila frontend belum di-build, halaman menjawab 503
// dengan petunjuk cara membangunnya.
func TestSPANotBuilt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	NewHandler(fstest.MapFS{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, seharusnya %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rec.Body.String(), "npm --prefix web run build") {
		t.Errorf("isi %q tidak memuat petunjuk build", rec.Body.String())
	}
}
