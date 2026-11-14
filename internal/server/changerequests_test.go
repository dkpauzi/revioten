package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/change"
)

// TestChangeEndpoints: handler perubahan (M5.1) meneruskan isian ke layanan
// dan menulis jawaban sesuai kontrak.
func TestChangeEndpoints(t *testing.T) {
	at := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	fake := &fakeChangeRequests{c: change.Change{
		ID: "c-1", Ref: "CHG-0001", CustomerID: "cu-1", CustomerCode: "AB", Mechanism: change.MechanismManual,
		Trigger: &change.Ref{ID: "s-1", Name: "Customer claim"}, ChangeType: change.Ref{ID: "t-1", Key: "content", Name: "Content"},
		Reason: "Add AOI check", Status: change.StatusSubmitted, RequestedBy: "PPAP Engineer",
		Documents: []change.AffectedDocument{{ID: "d-1", Number: "SC-AB-001", DocumentType: change.Ref{ID: "dt-3", Key: "cp", Name: "CP"}, Status: "draft"}},
		CreatedAt: at, UpdatedAt: at,
	}}
	fake.list = []change.Change{fake.c}
	srv := newTestServer(t, nil, Deps{ChangeRequests: fake})

	rec := srv.do(http.MethodGet, "/api/v1/changes/c-1", "")
	want := `{"id":"c-1","ref":"CHG-0001","customer_id":"cu-1","customer_code":"AB","mechanism":"manual",` +
		`"trigger":{"id":"s-1","name":"Customer claim"},"change_type":{"id":"t-1","key":"content","name":"Content"},` +
		`"reason":"Add AOI check","status":"submitted","requested_by":"PPAP Engineer",` +
		`"documents":[{"id":"d-1","number":"SC-AB-001","document_type":{"id":"dt-3","key":"cp","name":"CP"},"revision":0,"status":"draft"}],` +
		`"attachments":[],"impacts":[],"created_at":"2026-11-14T08:00:00Z","updated_at":"2026-11-14T08:00:00Z"}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("baca: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/changes", `{"customer_id":"cu-1","trigger_id":"s-1","change_type_id":"t-1","reason":"Add AOI check","document_ids":["d-1","d-2"]}`)
	wantNew := change.NewChange{CustomerID: "cu-1", TriggerID: "s-1", ChangeTypeID: "t-1", Reason: "Add AOI check", DocumentIDs: []string{"d-1", "d-2"}}
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotNew, wantNew) {
		t.Errorf("ajukan: status %d, %+v", rec.Code, fake.gotNew)
	}
	for _, c := range []struct {
		method, path, body string
		want               called
	}{
		{http.MethodGet, "/api/v1/changes?customer_id=cu-1", "", called{method: "List", id: "cu-1"}},
		{http.MethodPost, "/api/v1/changes/c-1/documents", `{"document_id":"d-2"}`, called{method: "AddDocument", id: "c-1", input: "d-2"}},
		{http.MethodDelete, "/api/v1/changes/c-1/documents/d-2", "", called{method: "RemoveDocument", id: "c-1", input: "d-2"}},
		{http.MethodGet, "/api/v1/documents/d-1/changes", "", called{method: "ForDocument", id: "d-1"}},
		{http.MethodPut, "/api/v1/changes/c-1/impacts", `{"items":[{"kind":"cp","decision":"not_affected","reason":"Same method"}]}`, called{method: "SaveImpacts", id: "c-1", input: "[{cp  not_affected Same method}]"}},
		{http.MethodPost, "/api/v1/changes/c-1/verification", `{"decision":"reject","note":"No 8D"}`, called{method: "Verify", id: "c-1", input: "false:No 8D"}},
	} {
		rec := srv.do(c.method, c.path, c.body)
		if rec.Code >= 300 || fake.got != c.want {
			t.Errorf("%s %s: status %d, %+v", c.method, c.path, rec.Code, fake.got)
		}
	}
}

// TestChangeAttachmentEndpoints: unggah lewat multipart, unduh sebagai
// attachment dengan nama asli, dan body di atas 1 MiB diterima (M5.2).
func TestChangeAttachmentEndpoints(t *testing.T) {
	fake := &fakeChangeRequests{c: change.Change{ID: "c-1"}, file: "%PDF-1.4"}
	srv := newTestServer(t, nil, Deps{ChangeRequests: fake})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("note", "abaikan")
	fw, _ := mw.CreateFormFile("file", "8D report.pdf")
	content := strings.Repeat("x", 2<<20) // 2 MB, di atas batas JSON 1 MiB
	_, _ = fw.Write([]byte(content))
	_ = mw.Close()
	rec := srv.do(http.MethodPost, "/api/v1/changes/c-1/attachments", body.String(), "Content-Type", mw.FormDataContentType())
	if rec.Code != http.StatusCreated || fake.got != (called{method: "AddAttachment", id: "c-1", input: "8D report.pdf:" + content}) {
		t.Errorf("unggah: status %d, %s, %s", rec.Code, fake.got.method, rec.Body.String())
	}

	// Tanpa bagian "file": 422 dengan field file.
	body.Reset()
	mw = multipart.NewWriter(&body)
	_ = mw.WriteField("note", "abaikan")
	_ = mw.Close()
	rec = srv.do(http.MethodPost, "/api/v1/changes/c-1/attachments", body.String(), "Content-Type", mw.FormDataContentType())
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), `"field":"file"`) {
		t.Errorf("tanpa file: status %d, %s", rec.Code, rec.Body.String())
	}

	rec = srv.do(http.MethodGet, "/api/v1/changes/c-1/attachments/a-1", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "%PDF-1.4" ||
		rec.Header().Get("Content-Disposition") != "attachment; filename*=UTF-8''8D%20report.pdf" ||
		rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("unduh: status %d, %v, %q", rec.Code, rec.Header(), rec.Body.String())
	}
	rec = srv.do(http.MethodDelete, "/api/v1/changes/c-1/attachments/a-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "RemoveAttachment", id: "c-1", input: "a-1"}) {
		t.Errorf("lepas: status %d, %+v", rec.Code, fake.got)
	}
}

// TestYokotenEndpoints: yokoten (M5.9) diteruskan ke layanan dan ditulis
// sesuai kontrak; pencarian target tidak menyimpan.
func TestYokotenEndpoints(t *testing.T) {
	at := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	fake := &fakeChangeRequests{yokoten: change.Yokoten{
		ID: "y-1", Ref: "YKT-0001", SourceID: "c-1", SourceChange: "CHG-0001", Reason: "Spread AOI fix", CreatedBy: "PPAP Engineer", CreatedAt: at,
		Targets: []change.YokotenTarget{{ID: "t-1", ModelID: "m-1", ModelName: "K-200", CustomerID: "cu-1", CustomerCode: "AB", Decision: "apply",
			ChangeID: "c-2", ChangeRef: "CHG-0002", Progress: "waiting_verification",
			Documents: []change.TargetDocument{{ID: "d-1", Number: "SC-AB-002", DocumentType: "cp", Status: "draft"}}}},
	}}
	srv := newTestServer(t, nil, Deps{ChangeRequests: fake})
	rec := srv.do(http.MethodGet, "/api/v1/yokotens/y-1", "")
	want := `{"id":"y-1","ref":"YKT-0001","source_change":{"id":"c-1","name":"CHG-0001"},"reason":"Spread AOI fix","created_by":"PPAP Engineer","created_at":"2026-11-14T08:00:00Z",` +
		`"targets":[{"id":"t-1","model_id":"m-1","model_name":"K-200","customer_id":"cu-1","customer_code":"AB","decision":"apply","change":{"id":"c-2","name":"CHG-0002"},` +
		`"progress":"waiting_verification","documents":[{"id":"d-1","number":"SC-AB-002","document_type":"cp","revision":0,"status":"draft"}]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("baca: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/yokoten-targets/search", `{"source_change_id":"c-1","targets":{"model_group_id":"g-1","text":"aoi"}}`)
	if rec.Code != http.StatusOK || !fake.gotYokoten.DryRun || fake.gotYokoten.Targets.ModelGroupID != "g-1" || fake.gotYokoten.Targets.Text != "aoi" {
		t.Errorf("cari: status %d, %+v", rec.Code, fake.gotYokoten)
	}
	rec = srv.do(http.MethodPost, "/api/v1/yokotens", `{"source_change_id":"c-1","reason":"Spread","targets":{"model_ids":["m-1","m-2"]}}`)
	if rec.Code != http.StatusCreated || fake.gotYokoten.DryRun || !reflect.DeepEqual(fake.gotYokoten.Targets.ModelIDs, []string{"m-1", "m-2"}) {
		t.Errorf("buat: status %d, %+v", rec.Code, fake.gotYokoten)
	}
	rec = srv.do(http.MethodPost, "/api/v1/yokotens/y-1/targets/t-1/decision", `{"decision":"not_applicable","reason":"No AOI"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "Decide", id: "y-1", input: "t-1:not_applicable:No AOI"}) {
		t.Errorf("putus: status %d, %+v", rec.Code, fake.got)
	}
}
