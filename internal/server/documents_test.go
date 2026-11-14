package server

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
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
		Route: &document.Ref{ID: "r-1", Name: "Up to GM"}, Header: map[string]any{"f-1": "Line 1"}, HeaderVersion: 4,
		Rows:      []document.Row{{ID: "row-1", Position: 1, Version: 2, ProcessID: "p-1", ProcessName: "SMT", OperationNumber: "10", Content: map[string]any{"f-2": "Paste"}}},
		CreatedAt: created, UpdatedAt: created,
	}}
	srv := newTestServer(t, nil, Deps{Documents: fake})

	rec := srv.do(http.MethodGet, "/api/v1/documents/d-1", "")
	want := `{"id":"d-1","document_number_id":"n-1","number":"SC-AB-001","customer_id":"c-1","customer_code":"AB",` +
		`"document_type":{"id":"dt-3","key":"cp","name":"CP"},"revision":0,"status":"draft","form_set_version_id":"fsv-1",` +
		`"form_set_name":"AIAG 4th set","format_version_id":"v-3","models":[{"id":"m-1","name":"K-125"}],"part_numbers":[],` +
		`"route":{"id":"r-1","name":"Up to GM"},"header":{"f-1":"Line 1"},"header_version":4,` +
		`"rows":[{"id":"row-1","position":1,"version":2,"operation_manual":false,"detached":false,"process_id":"p-1","process_name":"SMT","operation_number":"10","content":{"f-2":"Paste"}}],"set":[],` +
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
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/rows/batch", `{"rows":[{"operation_number":"20","content":{}},{"content":{"f-2":"Height"}}]}`)
	if rec.Code != http.StatusCreated || len(fake.gotRows) != 2 || fake.gotRows[0].OperationNumber != "20" || fake.gotRows[1].Content["f-2"] != "Height" {
		t.Errorf("tempel baris: status %d, %+v", rec.Code, fake.gotRows)
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/rows/row-1", `{"content":{},"version":2}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateRow", id: "d-1", input: "row-1"}) || fake.gotUpd.Version != 2 {
		t.Errorf("ubah baris: status %d, %+v, versi %d", rec.Code, fake.got, fake.gotUpd.Version)
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/header", `{"header":{"f-1":"Line 2"},"header_version":4}`)
	if rec.Code != http.StatusOK || fake.header["f-1"] != "Line 2" || fake.headerVersion != 4 {
		t.Errorf("header: status %d, %+v, versi %d", rec.Code, fake.header, fake.headerVersion)
	}

	// M4.3 (R-DOC-07): konflik dari layanan menjadi 409 dengan kode changed.
	fake.err = &problem.ConflictError{Problems: []problem.Problem{{Field: "version", Code: problem.Changed, Message: "changed"}}}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/rows/row-1", `{"content":{},"version":1}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"code":"changed"`) {
		t.Errorf("konflik: status %d, %s", rec.Code, rec.Body.String())
	}
	fake.err = nil
	rec = srv.do(http.MethodDelete, "/api/v1/documents/d-1/rows/row-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "DeleteRow", id: "d-1", input: "row-1"}) {
		t.Errorf("hapus baris: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodGet, "/api/v1/documents?customer_id=c-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "List", id: "c-1"}) {
		t.Errorf("daftar: status %d, %+v", rec.Code, fake.got)
	}

	// M4.5: tampilan perakitan dan keputusan penyusun.
	fake.composition = document.Composition{
		VersionID: "pv-1", Version: 2, Process: document.Ref{ID: "p-1", Name: "IQC"},
		Subprocesses: []document.ComposedSubprocess{{ID: "s-1", Name: "IQC MSD", Points: []document.ComposedPoint{
			{PointID: "pp-1", DocumentType: document.TypeRef{ID: "dt-3", Key: "cp", Name: "CP"}, Text: "Check MSL label", DocumentID: "d-1", State: document.StateExcluded, Reason: "No MSD parts"},
		}}},
	}
	rec = srv.do(http.MethodGet, "/api/v1/documents/d-1/composition?process_template_version_id=pv-1", "")
	wantView := `{"process_template_version_id":"pv-1","version":2,"process":{"id":"p-1","name":"IQC"},"subprocesses":[{"id":"s-1","name":"IQC MSD",` +
		`"points":[{"point_id":"pp-1","document_type":{"id":"dt-3","key":"cp","name":"CP"},"text":"Check MSL label","document_id":"d-1","state":"excluded","reason":"No MSD parts"}]}]}`
	if rec.Code != http.StatusOK || fake.got != (called{method: "Composition", id: "d-1", input: "pv-1"}) || !sameJSON(t, rec.Body.String(), wantView) {
		t.Errorf("perakitan: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/composition", `{"process_template_version_id":"pv-1","include":["pp-2"],"exclude":[{"point_id":"pp-1","reason":"No MSD parts"}]}`)
	wantCompose := document.Compose{VersionID: "pv-1", Include: []string{"pp-2"}, Exclude: []document.Exclusion{{PointID: "pp-1", Reason: "No MSD parts"}}}
	if rec.Code != http.StatusOK || !reflect.DeepEqual(fake.gotCompose, wantCompose) {
		t.Errorf("rakit: status %d, %+v", rec.Code, fake.gotCompose)
	}

	// M4.6: urutan baru dan penomoran otomatis.
	rec = srv.do(http.MethodPut, "/api/v1/documents/d-1/order", `{"row_ids":["row-2","row-1"],"renumber":true}`)
	if rec.Code != http.StatusOK || !reflect.DeepEqual(fake.gotOrder, document.Order{RowIDs: []string{"row-2", "row-1"}, Renumber: true}) {
		t.Errorf("urut: status %d, %+v", rec.Code, fake.gotOrder)
	}

	// M4.7: salin komposisi dari dokumen model lain.
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/copy", `{"source_document_id":"d-9"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "CopyFrom", id: "d-1", input: "d-9"}) {
		t.Errorf("salin: status %d, %+v", rec.Code, fake.got)
	}

	// M4.8: perbandingan dengan template dan putus tautan.
	fake.differences = []document.RowDifference{{
		RowID: "row-1", Position: 1, PointID: "pp-1", PointText: "Check MSL label", Status: document.LinkDifferent,
		Fields: []document.FieldDifference{{FieldID: "f-2", Label: "Product Characteristics", Template: "MSL label", Row: ""}},
	}}
	rec = srv.do(http.MethodGet, "/api/v1/documents/d-1/differences", "")
	wantDiff := `{"items":[{"row_id":"row-1","position":1,"point_id":"pp-1","point_text":"Check MSL label","status":"different",` +
		`"fields":[{"field_id":"f-2","label":"Product Characteristics","template":"MSL label","row":""}]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), wantDiff) {
		t.Errorf("perbandingan: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/rows/row-1/detach", `{"reason":"Customer drawing"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "Detach", id: "d-1", input: "row-1"}) || fake.gotReason != "Customer drawing" {
		t.Errorf("putus tautan: status %d, %+v %q", rec.Code, fake.got, fake.gotReason)
	}
}
