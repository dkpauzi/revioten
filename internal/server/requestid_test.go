package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestRequestIDGenerated: tanpa header dari klien, server membuat request ID
// sendiri, berbeda untuk setiap permintaan.
func TestRequestIDGenerated(t *testing.T) {
	srv := newTestServer(t, testUI(), Deps{})

	first := srv.do(http.MethodGet, "/health", "").Header().Get("X-Request-Id")
	second := srv.do(http.MethodGet, "/health", "").Header().Get("X-Request-Id")

	if first == "" || second == "" {
		t.Fatalf("request ID kosong: %q dan %q", first, second)
	}
	if first == second {
		t.Errorf("dua permintaan mendapat request ID yang sama: %q", first)
	}
	if !validRequestID(first) {
		t.Errorf("request ID buatan server %q tidak lolos aturannya sendiri", first)
	}
}

// TestRequestIDFromClient: request ID yang wajar dari klien dipakai apa adanya,
// juga di halaman web, bukan hanya di API.
func TestRequestIDFromClient(t *testing.T) {
	srv := newTestServer(t, testUI(), Deps{})

	for _, path := range []string{"/health", "/", "/api/v1/tidak-ada"} {
		rec := srv.do(http.MethodGet, path, "", "X-Request-Id", "uji-Klien_01.a")
		if got := rec.Header().Get("X-Request-Id"); got != "uji-Klien_01.a" {
			t.Errorf("%s: X-Request-Id %q, ingin uji-Klien_01.a", path, got)
		}
	}
}

// TestRequestIDInvalidReplaced: request ID yang terlalu panjang atau berisi
// karakter lain diganti, supaya log tidak bisa disusupi baris palsu.
func TestRequestIDInvalidReplaced(t *testing.T) {
	srv := newTestServer(t, nil, Deps{})

	tests := map[string]string{
		"terlalu panjang":   strings.Repeat("a", 65),
		"berisi spasi":      "uji klien",
		"berisi baris baru": "uji\nlevel=ERROR msg=palsu",
		"berisi tanda =":    "a=b",
	}
	for name, bad := range tests {
		t.Run(name, func(t *testing.T) {
			got := srv.do(http.MethodGet, "/health", "", "X-Request-Id", bad).Header().Get("X-Request-Id")
			if got == bad || !validRequestID(got) {
				t.Errorf("request ID %q dipakai atau diganti dengan nilai tidak valid %q", bad, got)
			}
		})
	}

	// Batas atas: 64 karakter masih diterima.
	ok := strings.Repeat("a", 64)
	if got := srv.do(http.MethodGet, "/health", "", "X-Request-Id", ok).Header().Get("X-Request-Id"); got != ok {
		t.Errorf("request ID 64 karakter diganti menjadi %q", got)
	}
}
