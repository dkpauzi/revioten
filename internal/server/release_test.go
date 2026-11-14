package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/document"
)

// TestReleaseEndpoints: rilis dan daftar revisi (M5.5) diteruskan ke layanan.
func TestReleaseEndpoints(t *testing.T) {
	at := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	fake := &fakeDocuments{revisions: []document.Revision{
		{ID: "r-1", Revision: 1, Status: "draft", CreatedBy: "Verifier", CreatedAt: at},
		{ID: "r-0", Revision: 0, Status: "released", CreatedBy: "Author", CreatedAt: at, ReleasedBy: "DCC", ReleasedAt: &at, SHA256: "ab12"},
	}}
	srv := newTestServer(t, nil, Deps{Documents: fake})
	rec := srv.do(http.MethodGet, "/api/v1/documents/d-1/revisions", "")
	want := `{"items":[{"id":"r-1","revision":1,"status":"draft","created_by":"Verifier","created_at":"2026-11-14T08:00:00Z"},` +
		`{"id":"r-0","revision":0,"status":"released","created_by":"Author","created_at":"2026-11-14T08:00:00Z",` +
		`"released_by":"DCC","released_at":"2026-11-14T08:00:00Z","sha256":"ab12"}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("revisi: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/release", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "Release", id: "d-1"}) {
		t.Errorf("rilis: status %d, %+v", rec.Code, fake.got)
	}
}
