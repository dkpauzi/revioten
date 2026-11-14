package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/printout"
)

// fakePrinting meniru printout.Service (Tahap 6).
type fakePrinting struct {
	got    called
	gotReq printout.Request
	err    error
}

func (f *fakePrinting) Print(_ context.Context, _ access.Actor, id string, req printout.Request) (printout.File, error) {
	f.got, f.gotReq = called{method: "Print", id: id}, req
	return printout.File{Name: "SC-AB-001-00-draft.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Body: []byte("PK")}, f.err
}

func (f *fakePrinting) Logs(_ context.Context, _ access.Actor, id string) ([]printout.Log, error) {
	n := 2
	return []printout.Log{{ID: "l-1", Revision: 0, Purpose: "controlled", Format: "pdf", CopyNumber: &n, SHA256: "ab", PrintedBy: "DCC", PrintedAt: time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)}}, f.err
}

func (f *fakePrinting) UploadScan(_ context.Context, _ access.Actor, id, fileName string, r io.Reader) (printout.Scan, error) {
	body, _ := io.ReadAll(r)
	f.got = called{method: "UploadScan", id: id, input: fileName + ":" + string(body)}
	return printout.Scan{FileName: fileName}, f.err
}

func (f *fakePrinting) OpenScan(_ context.Context, _ access.Actor, id string, revision int) (printout.Scan, *os.File, error) {
	return printout.Scan{}, nil, f.err
}

func (f *fakePrinting) Verify(_ context.Context, _ access.Actor, revisionID string) (printout.Verification, error) {
	return printout.Verification{RevisionID: revisionID, Number: "SC-AB-001", Status: "effective"}, f.err
}

func (f *fakePrinting) Layouts(context.Context, access.Actor) ([]printout.PrintLayout, error) {
	return nil, f.err
}

func (f *fakePrinting) CreateLayout(_ context.Context, _ access.Actor, in printout.LayoutInput) (printout.PrintLayout, error) {
	f.got = called{method: "CreateLayout", input: in.Name + ":" + string(in.Layout)}
	return printout.PrintLayout{Name: in.Name}, f.err
}

func (f *fakePrinting) UpdateLayout(_ context.Context, _ access.Actor, id string, in printout.LayoutInput) (printout.PrintLayout, error) {
	return printout.PrintLayout{ID: id}, f.err
}

func (f *fakePrinting) SetCustomerLayout(_ context.Context, _ access.Actor, customerID, documentTypeID, layoutID string) error {
	f.got = called{method: "SetCustomerLayout", id: customerID, input: documentTypeID + ":" + layoutID}
	return f.err
}

func (f *fakePrinting) CustomerLayouts(context.Context, access.Actor, string) (map[string]string, error) {
	return map[string]string{"t-3": "l-9"}, f.err
}

// TestPrintEndpoints: cetak mengirim berkas sebagai unduhan, catatan cetak,
// scan multipart, verifikasi, dan layout customer (Tahap 6).
func TestPrintEndpoints(t *testing.T) {
	fake := &fakePrinting{}
	srv := newTestServer(t, nil, Deps{Printing: fake})
	rec := srv.do(http.MethodPost, "/api/v1/documents/d-1/print", `{"purpose":"controlled","format":"pdf","revision":1}`)
	if rec.Code != http.StatusOK || rec.Body.String() != "PK" || rec.Header().Get("Content-Disposition") != "attachment; filename*=UTF-8''SC-AB-001-00-draft.xlsx" ||
		fake.gotReq.Purpose != "controlled" || fake.gotReq.Format != "pdf" || fake.gotReq.Revision == nil || *fake.gotReq.Revision != 1 {
		t.Errorf("cetak: status %d, %+v, %v", rec.Code, fake.gotReq, rec.Header())
	}
	rec = srv.do(http.MethodGet, "/api/v1/documents/d-1/prints", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"copy_number":2`) {
		t.Errorf("catatan cetak: %d %s", rec.Code, rec.Body.String())
	}
	body := "--b\r\nContent-Disposition: form-data; name=\"file\"; filename=\"signed.pdf\"\r\n\r\n%PDF\r\n--b--\r\n"
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/scan", body, "Content-Type", "multipart/form-data; boundary=b")
	if rec.Code != http.StatusCreated || fake.got != (called{method: "UploadScan", id: "d-1", input: "signed.pdf:%PDF"}) {
		t.Errorf("scan: %d %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodGet, "/api/v1/verify/r-1", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"effective"`) {
		t.Errorf("verifikasi: %d %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/customers/cu-1/print-layouts/t-3", `{"layout_id":"l-9"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "SetCustomerLayout", id: "cu-1", input: "t-3:l-9"}) || !strings.Contains(rec.Body.String(), `"layout_id":"l-9"`) {
		t.Errorf("layout customer: %d %+v", rec.Code, fake.got)
	}
}
