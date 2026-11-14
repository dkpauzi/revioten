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
		{ID: "r-1", Revision: 1, Status: "draft", CreatedBy: "Verifier", CreatedAt: at, Changes: []string{"CHG-0002"}},
		{ID: "r-0", Revision: 0, Status: "released", CreatedBy: "Author", CreatedAt: at, ReleasedBy: "DCC", ReleasedAt: &at, SHA256: "ab12", Changes: []string{}},
	}}
	srv := newTestServer(t, nil, Deps{Documents: fake})
	rec := srv.do(http.MethodGet, "/api/v1/documents/d-1/revisions", "")
	want := `{"items":[{"id":"r-1","revision":1,"status":"draft","created_by":"Verifier","created_at":"2026-11-14T08:00:00Z","changes":["CHG-0002"]},` +
		`{"id":"r-0","revision":0,"status":"released","created_by":"Author","created_at":"2026-11-14T08:00:00Z",` +
		`"released_by":"DCC","released_at":"2026-11-14T08:00:00Z","sha256":"ab12","changes":[]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("revisi: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/release", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "Release", id: "d-1"}) {
		t.Errorf("rilis: status %d, %+v", rec.Code, fake.got)
	}
}

// TestDiffEndpoint: diff dua revisi (M5.6) menulis sel kosong sebagai field
// yang tidak ada, dan meneruskan saringan proses.
func TestDiffEndpoint(t *testing.T) {
	fake := &fakeDocuments{diff: document.Diff{From: 0, ToDraft: true, Header: []document.CellChange{}, Rows: []document.RowChange{
		{RowID: "row-1", Status: document.RowChanged, OldPosition: 1, NewPosition: 1, ProcessName: "SMT Printing",
			Cells: []document.CellChange{{FieldID: "f-1", Label: "Method", New: "SPI"}}},
		{RowID: "row-2", Status: document.RowAdded, NewPosition: 2, Cells: []document.CellChange{}},
	}}}
	srv := newTestServer(t, nil, Deps{Documents: fake})
	rec := srv.do(http.MethodGet, "/api/v1/documents/d-1/diff?from=0&process_id=p-1", "")
	want := `{"from":0,"to_draft":true,"header":[],"rows":[` +
		`{"row_id":"row-1","status":"changed","old_position":1,"new_position":1,"process_name":"SMT Printing","cells":[{"field_id":"f-1","label":"Method","new":"SPI"}]},` +
		`{"row_id":"row-2","status":"added","new_position":2,"cells":[]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) || fake.got != (called{method: "Diff", id: "d-1", input: "0:false:p-1"}) {
		t.Errorf("diff: status %d, %+v\n%s", rec.Code, fake.got, rec.Body.String())
	}
}
