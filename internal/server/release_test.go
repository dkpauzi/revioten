package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/change"
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
	want := `{"items":[{"id":"r-1","revision":1,"status":"draft","created_by":"Verifier","created_at":"2026-11-14T08:00:00Z","changes":["CHG-0002"],"migrated":false},` +
		`{"id":"r-0","revision":0,"status":"released","created_by":"Author","created_at":"2026-11-14T08:00:00Z",` +
		`"released_by":"DCC","released_at":"2026-11-14T08:00:00Z","sha256":"ab12","changes":[],"migrated":false}]}`
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

// TestPropagationEndpoints: pratinjau dan propagasi template, serta mode
// tautan dokumen (M5.8), diteruskan ke layanan.
func TestPropagationEndpoints(t *testing.T) {
	changes := &fakeChangeRequests{preview: []change.PreviewItem{{
		TemplatePlan: document.TemplatePlan{DocumentID: "d-1", Number: "SC-AB-001", CustomerID: "cu-1", Status: "draft", Mode: "follow", Updates: 1,
			Rows: []document.PlanRow{{RowID: "r-1", Position: 1, Status: "update", PointText: "Check part number", Fields: []document.FieldDifference{{FieldID: "f-1", Label: "Method", Template: "SPI", Row: "Visual"}}}}},
		Included: true,
	}}}
	docs := &fakeDocuments{templates: []document.TemplateLink{{TemplateID: "pt-1", ProcessName: "IQC", Version: 2, Mode: "pinned", Reason: "Customer approved v2"}}}
	srv := newTestServer(t, nil, Deps{ChangeRequests: changes, Documents: docs})
	rec := srv.do(http.MethodGet, "/api/v1/process-template-versions/v-2/propagation?scope=following", "")
	want := `{"items":[{"document_id":"d-1","number":"SC-AB-001","customer_id":"cu-1","status":"draft","mode":"follow","included":true,"updates":1,"conflicts":0,"removed":0,` +
		`"rows":[{"row_id":"r-1","position":1,"status":"update","point_text":"Check part number","fields":[{"field_id":"f-1","label":"Method","template":"SPI","row":"Visual"}]}]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) || changes.got != (called{method: "Preview", id: "v-2", input: "following"}) {
		t.Errorf("pratinjau: status %d, %+v\n%s", rec.Code, changes.got, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-2/propagation",
		`{"scope":"all","conflict_policy":"overwrite","change_type_id":"t-1","reason":"SQAM rev 5","source_reference":"SQAM-CD rev 5","effective_date":"2026-12-01"}`)
	got := changes.gotPropagation
	if rec.Code != http.StatusCreated || got.Scope != "all" || got.ConflictPolicy != "overwrite" || got.SourceReference != "SQAM-CD rev 5" ||
		got.EffectiveDate == nil || !got.EffectiveDate.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("propagasi: status %d, %+v\n%s", rec.Code, got, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/templates/pt-1", `{"mode":"pinned","reason":"Customer approved v2"}`)
	want = `{"items":[{"process_template_id":"pt-1","process_name":"IQC","version":2,"mode":"pinned","reason":"Customer approved v2"}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) || docs.got != (called{method: "SetTemplateMode", id: "d-1", input: "pt-1:pinned"}) || docs.gotReason != "Customer approved v2" {
		t.Errorf("mode: status %d, %+v\n%s", rec.Code, docs.got, rec.Body.String())
	}
}
