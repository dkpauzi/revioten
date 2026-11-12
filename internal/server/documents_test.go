package server

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/document"
)

// TestDocumentEndpoints: handler dokumen (M4.1) meneruskan isian ke layanan
// dan menulis jawaban sesuai kontrak.
func TestDocumentEndpoints(t *testing.T) {
	created := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	fake := &fakeDocuments{doc: document.Document{
		ID: "d-1", NumberID: "n-1", Number: "SC-AB-001", CustomerID: "c-1", CustomerCode: "AB",
		DocumentType: document.TypeRef{ID: "dt-3", Key: "cp", Name: "CP"}, Status: document.StatusDraft,
		FormSetVersionID: "fsv-1", FormSetName: "AIAG 4th set", FormatVersionID: "v-3",
		Models: []document.Ref{{ID: "m-1", Name: "K-125"}}, PartNumbers: []document.PartRef{},
		Route: &document.Ref{ID: "r-1", Name: "Up to GM"}, Header: map[string]any{"f-1": "Line 1"},
		Rows:      []document.Row{{ID: "row-1", Position: 1, ProcessID: "p-1", ProcessName: "SMT", OperationNumber: "10", Content: map[string]any{"f-2": "Paste"}}},
		CreatedAt: created, UpdatedAt: created,
	}}
	srv := newTestServer(t, nil, Deps{Documents: fake})

	rec := srv.do(http.MethodGet, "/api/v1/documents/d-1", "")
	want := `{"id":"d-1","document_number_id":"n-1","number":"SC-AB-001","customer_id":"c-1","customer_code":"AB",` +
		`"document_type":{"id":"dt-3","key":"cp","name":"CP"},"revision":0,"status":"draft","form_set_version_id":"fsv-1",` +
		`"form_set_name":"AIAG 4th set","format_version_id":"v-3","models":[{"id":"m-1","name":"K-125"}],"part_numbers":[],` +
		`"route":{"id":"r-1","name":"Up to GM"},"header":{"f-1":"Line 1"},` +
		`"rows":[{"id":"row-1","position":1,"process_id":"p-1","process_name":"SMT","operation_number":"10","content":{"f-2":"Paste"}}],"set":[],` +
		`"created_at":"2026-11-12T08:00:00Z","updated_at":"2026-11-12T08:00:00Z"}`
	if rec.Code != http.StatusOK || fake.got != (called{method: "Get", id: "d-1"}) || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("baca: status %d\n%s\ningin\n%s", rec.Code, rec.Body.String(), want)
	}

	rec = srv.do(http.MethodPost, "/api/v1/documents", `{"document_number_id":"n-1","model_ids":["m-1"],"route_id":"r-2","route_reason":"Customer"}`)
	wantNew := document.NewDocument{NumberID: "n-1", ModelIDs: []string{"m-1"}, RouteID: "r-2", RouteReason: "Customer"}
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotNew, wantNew) {
		t.Errorf("buat: status %d, %+v", rec.Code, fake.gotNew)
	}
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/rows", `{"process_id":"p-1","operation_number":"20","content":{"f-2":"Height"}}`)
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotRow, document.NewRow{ProcessID: "p-1", OperationNumber: "20", Content: map[string]any{"f-2": "Height"}}) {
		t.Errorf("tambah baris: status %d, %+v", rec.Code, fake.gotRow)
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/rows/row-1", `{"content":{}}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateRow", id: "d-1", input: "row-1"}) {
		t.Errorf("ubah baris: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/header", `{"header":{"f-1":"Line 2"}}`)
	if rec.Code != http.StatusOK || fake.header["f-1"] != "Line 2" {
		t.Errorf("header: status %d, %+v", rec.Code, fake.header)
	}
	rec = srv.do(http.MethodDelete, "/api/v1/documents/d-1/rows/row-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "DeleteRow", id: "d-1", input: "row-1"}) {
		t.Errorf("hapus baris: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodGet, "/api/v1/documents?customer_id=c-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "List", id: "c-1"}) {
		t.Errorf("daftar: status %d, %+v", rec.Code, fake.got)
	}
}
