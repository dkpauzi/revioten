package integrationapi

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/integration"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Contract test API integrasi (M9.2, R-API-15).
//
// Bentuk setiap jawaban (nama field dan jenis nilainya) dibandingkan dengan
// berkas acuan api/integration.contract.json. Perubahan yang mengubah bentuk
// itu membuat test ini gagal, juga bila perubahannya datang dari tempat lain,
// misalnya dari UI API atau dari layanan di bawahnya. Menambah field boleh
// (perubahan aditif), tetapi harus disengaja: jalankan
//
//	go test ./internal/integrationapi/ -run TestContract -update
//
// lalu periksa selisih berkas acuannya sebelum di-commit. Menghapus atau
// mengganti jenis field butuh versi mayor baru (/integration/v2).

var update = flag.Bool("update", false, "tulis ulang api/integration.contract.json")

const (
	contractFile = "../../api/integration.contract.json"
	specFile     = "../../api/integration.yaml"
	goodToken    = "rvt_contoh00_token"
)

// fake meniru integration.Service dengan jawaban tetap.
type fake struct {
	fail error // bila terisi, semua operasi sesudah autentikasi mengembalikannya
}

var at = time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)

func (f *fake) Authenticate(_ context.Context, token, _ string) (integration.Client, error) {
	if token != goodToken {
		return integration.Client{}, integration.ErrUnauthenticated
	}
	return integration.Client{ServiceAccountID: "sa-1", Name: "MES", TokenID: "tok-1", TokenPrefix: "contoh00"}, nil
}

func (f *fake) ListDocuments(context.Context, integration.Client, integration.ListFilter) (integration.Page, error) {
	return integration.Page{NextCursor: "cursor-2", Items: []integration.ReleasedDocument{{
		ID: "d-1", Number: "SC-AB-001", CustomerID: "c-1", CustomerCode: "AB", DocumentType: "cp", Revision: 1, ReleasedAt: at, SHA256: "ab12",
	}}}, f.fail
}

func (f *fake) GetRevision(context.Context, integration.Client, string, *int) (integration.Revision, error) {
	return integration.Revision{
		DocumentID: "d-1", Number: "SC-AB-001", CustomerID: "c-1", Revision: 1, Current: true, ReleasedAt: at, ReleaseDate: "2026-11-15", SHA256: "ab12",
		Fields: []integration.Field{{ID: "f-1", Key: "product_characteristic", Label: "Product Characteristics", Area: "row"}},
		Snapshot: document.Snapshot{DocumentType: "cp", Models: []string{"K-125"}, PartNumbers: []string{"PN-1"}, Header: map[string]any{"f-0": "Line 1"},
			Sections: []document.SnapshotSection{{ID: "s-1", Name: "PCB A", PartNumbers: []string{"PN-1"}}},
			Rows:     []document.SnapshotRow{{ID: "r-1", Position: 1, SectionID: "s-1", ProcessID: "p-1", ProcessName: "SMT", OperationNumber: "10", Content: map[string]any{"f-1": "Paste"}}}},
	}, f.fail
}

func (f *fake) Propose(_ context.Context, _ integration.Client, key string, _ integration.Proposal) (integration.ProposedChange, bool, error) {
	return f.change(), key == "ulang", f.fail
}

func (f *fake) change() integration.ProposedChange {
	return integration.ProposedChange{ID: "ch-1", Ref: "CHG-0007", Status: "accepted", CreatedAt: at, VerifiedAt: &at, VerificationNote: "ok"}
}

func (f *fake) ChangeStatus(context.Context, integration.Client, string) (integration.ProposedChange, error) {
	return f.change(), f.fail
}

func (f *fake) job() integration.Job {
	return integration.Job{ID: "j-1", DocumentID: "d-1", Revision: 1, Status: "done", FileName: "SC-AB-001-01-export.xlsx", SizeBytes: 10, CreatedAt: at, FinishedAt: &at}
}

func (f *fake) CreateExport(context.Context, integration.Client, string, *int) (integration.Job, error) {
	return f.job(), f.fail
}

func (f *fake) GetExport(context.Context, integration.Client, string) (integration.Job, error) {
	return f.job(), f.fail
}

func (f *fake) OpenExport(context.Context, integration.Client, string) (integration.Job, *os.File, error) {
	if f.fail != nil {
		return integration.Job{}, nil, f.fail
	}
	file, err := os.CreateTemp("", "export-*.xlsx")
	if err != nil {
		return integration.Job{}, nil, err
	}
	_, _ = file.WriteString("PK contoh")
	_, _ = file.Seek(0, 0)
	return f.job(), file, nil
}

// shape memberi bentuk satu nilai JSON: nama field dan jenis nilainya.
// Untuk array, bentuk elemen pertamanya.
func shape(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, child := range x {
			out[k] = shape(child)
		}
		return out
	case []any:
		if len(x) == 0 {
			return []any{}
		}
		return []any{shape(x[0])}
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	default:
		return "null"
	}
}

type operation struct {
	id, method, path, body string
	headers                map[string]string
	status                 int
}

// operations mencakup setiap operasi di api/integration.yaml.
var operations = []operation{
	{id: "listReleasedDocuments", method: http.MethodGet, path: "/integration/v1/documents?limit=10", status: 200},
	{id: "getReleasedRevision", method: http.MethodGet, path: "/integration/v1/documents/d-1/revisions/current", status: 200},
	{id: "proposeChange", method: http.MethodPost, path: "/integration/v1/changes", status: 201, headers: map[string]string{"Idempotency-Key": "k-1"},
		body: `{"document_id":"d-1","base_revision":1,"source_application":"MES","reason":"Uji","operations":[{"op":"modify","row_id":"r-1","content":{"f-1":"Paste 120"}}]}`},
	{id: "getProposedChange", method: http.MethodGet, path: "/integration/v1/changes/ch-1", status: 200},
	{id: "createExport", method: http.MethodPost, path: "/integration/v1/exports", status: 202, body: `{"document_id":"d-1"}`},
	{id: "getExport", method: http.MethodGet, path: "/integration/v1/exports/j-1", status: 200},
	{id: "downloadExport", method: http.MethodGet, path: "/integration/v1/exports/j-1/file", status: 200},
}

func call(h http.Handler, op operation, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(op.method, op.path, strings.NewReader(op.body))
	if op.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range op.headers {
		req.Header.Set(k, v)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestContract(t *testing.T) {
	h := NewHandler(&fake{}, 1000, nil)

	// Setiap operasi di kontrak ada di tabel, dan sebaliknya.
	spec, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range operations {
		if !strings.Contains(string(spec), "operationId: "+op.id+"\n") {
			t.Errorf("operasi %s tidak ada di %s", op.id, specFile)
		}
	}
	if n := strings.Count(string(spec), "operationId: "); n != len(operations) {
		t.Errorf("kontrak punya %d operasi, tabel test punya %d", n, len(operations))
	}

	got := map[string]any{}
	for _, op := range operations {
		rec := call(h, op, goodToken)
		if rec.Code != op.status {
			t.Fatalf("%s: status %d, ingin %d: %s", op.id, rec.Code, op.status, rec.Body.String())
		}
		entry := map[string]any{"method": op.method, "path": strings.SplitN(op.path, "?", 2)[0], "status": op.status}
		if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			var body any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("%s: jawaban bukan JSON: %v", op.id, err)
			}
			entry["response"] = shape(body)
		} else {
			entry["response"] = rec.Header().Get("Content-Type")
		}
		got[op.id] = entry
	}
	// Bentuk error juga bagian dari kontrak (R-API-13).
	rec := call(NewHandler(&fake{fail: &problem.ValidationError{Problems: []problem.Problem{{Field: "reason", Code: problem.Required, Message: "Enter a value."}}}}, 1000, nil), operations[3], goodToken)
	var body any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	got["error"] = map[string]any{"status": rec.Code, "response": shape(body)}

	text, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	text = append(text, '\n')
	if *update {
		if err := os.WriteFile(contractFile, text, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(contractFile)
	if err != nil {
		t.Fatalf("berkas acuan belum ada, jalankan dengan -update: %v", err)
	}
	if strings.ReplaceAll(string(want), "\r\n", "\n") != string(text) {
		t.Errorf("bentuk API integrasi berubah dari %s.\nBila perubahan ini aditif dan disengaja, jalankan test dengan -update dan periksa selisihnya.\nSekarang:\n%s", contractFile, text)
	}
}

// TestErrorsAndLimits: token salah 401, scope kurang 403, data tidak ada 404,
// konflik 409, alamat di luar kontrak 404, dan batas laju 429 dengan Retry-After.
func TestErrorsAndLimits(t *testing.T) {
	code := func(rec *httptest.ResponseRecorder) string {
		var e Error
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		return fmt.Sprintf("%d %s", rec.Code, e.Code)
	}
	h := NewHandler(&fake{}, 1000, nil)
	for _, token := range []string{"", "rvt_salah"} {
		rec := call(h, operations[0], token)
		if code(rec) != "401 unauthenticated" || rec.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Errorf("token %q: %s", token, code(rec))
		}
	}
	if rec := call(h, operation{method: http.MethodGet, path: "/integration/v1/accounts"}, goodToken); code(rec) != "404 not_found" {
		t.Errorf("alamat di luar kontrak: %s", code(rec))
	}
	if rec := call(h, operation{method: http.MethodPost, path: "/integration/v1/changes", body: "{rusak", headers: map[string]string{"Idempotency-Key": "k"}}, goodToken); code(rec) != "400 invalid_request" {
		t.Errorf("JSON rusak: %s", code(rec))
	}
	for want, err := range map[string]error{
		"403 forbidden":      access.ErrForbidden,
		"404 not_found":      problem.ErrNotFound,
		"409 conflict":       &problem.ConflictError{Problems: []problem.Problem{{Field: "base_revision", Code: problem.Changed, Message: "x"}}},
		"500 internal_error": fmt.Errorf("postgres://user:rahasia@db/x gagal"),
	} {
		rec := call(NewHandler(&fake{fail: err}, 1000, nil), operations[3], goodToken)
		if code(rec) != want {
			t.Errorf("ingin %s, dapat %s", want, code(rec))
		}
		// Rincian internal tidak pernah sampai ke pemanggil (R-SEC-14).
		if strings.Contains(rec.Body.String(), "rahasia") {
			t.Errorf("rincian internal bocor: %s", rec.Body.String())
		}
	}
	// Usulan yang diulang dijawab 200, bukan 201.
	replay := operations[2]
	replay.headers = map[string]string{"Idempotency-Key": "ulang"}
	if rec := call(h, replay, goodToken); rec.Code != http.StatusOK {
		t.Errorf("ulangan: %d", rec.Code)
	}

	limited := NewHandler(&fake{}, 3, nil)
	for i := range 3 {
		if rec := call(limited, operations[0], goodToken); rec.Code != http.StatusOK {
			t.Fatalf("permintaan %d: %d", i+1, rec.Code)
		}
	}
	rec := call(limited, operations[0], goodToken)
	if code(rec) != "429 rate_limited" || rec.Header().Get("Retry-After") == "" {
		t.Errorf("batas laju: %s, Retry-After %q", code(rec), rec.Header().Get("Retry-After"))
	}
}

func TestLimiterWindow(t *testing.T) {
	now := at
	l := newLimiter(2, time.Minute, func() time.Time { return now })
	for range 2 {
		if _, limited := l.take("a"); limited {
			t.Fatal("ditolak sebelum batas")
		}
	}
	wait, limited := l.take("a")
	if !limited || wait < 1 || wait > 61 {
		t.Errorf("batas: %d, %v", wait, limited)
	}
	if _, limited := l.take("b"); limited {
		t.Error("token lain ikut dibatasi")
	}
	now = now.Add(time.Minute)
	if _, limited := l.take("a"); limited {
		t.Error("jendela baru masih dibatasi")
	}
	// Kunci map diurutkan hanya supaya test ini tidak bergantung urutan.
	keys := make([]string, 0, len(l.seen))
	for k := range l.seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "a,b" {
		t.Errorf("isi penghitung: %v", keys)
	}
}
