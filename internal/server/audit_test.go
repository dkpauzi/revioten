package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
)

// sampleEntries adalah dua entri contoh: oleh akun admin, dan oleh sistem.
var sampleEntries = []audit.Entry{
	{
		Seq: 7, OccurredAt: time.Date(2026, 10, 28, 1, 2, 3, 0, time.UTC), ActorType: "human",
		ActorAccountID: "akun-admin", ActorLogin: "admin.uji", ActorPersonID: "orang-admin", ActorPersonName: "Admin Uji",
		RequestID: "REQ-7", SourceIP: "127.0.0.1", Action: audit.ActionAccountDisabled,
		TargetType: audit.TargetAccount, TargetID: "akun-2", TargetLabel: "penyusun.uji",
		Details: map[string]string{}, PrevHash: "aa", Hash: "bb",
	},
	{
		Seq: 6, OccurredAt: time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC), ActorType: "system",
		Action: audit.ActionAccountCreated, Details: map[string]string{"login": "admin.uji"}, PrevHash: "99", Hash: "aa",
	},
}

// TestListAuditLogEndpoint: entri diterjemahkan ke bentuk kontrak, dengan
// akun dan orang aktornya (D-74), dan halaman berikutnya lewat next_before.
func TestListAuditLogEndpoint(t *testing.T) {
	fake := &fakeAudit{entries: sampleEntries}
	srv := newTestServer(t, nil, Deps{Audit: fake})

	rec := srv.do(http.MethodGet, "/api/v1/audit-log?before=8&limit=2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if fake.gotBefore != 8 || fake.gotLimit != 2 {
		t.Errorf("List menerima before %d limit %d", fake.gotBefore, fake.gotLimit)
	}
	var list AuditEntryList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 || list.NextBefore == nil || *list.NextBefore != 6 {
		t.Fatalf("isi %s", rec.Body.String())
	}
	first := list.Items[0]
	if first.Actor.Type != AuditActorTypeHuman || value(first.Actor.Login) != "admin.uji" ||
		value(first.Actor.PersonName) != "Admin Uji" || value(first.RequestId) != "REQ-7" {
		t.Errorf("entri pertama: %+v", first)
	}
	if first.Target == nil || first.Target.Id != "akun-2" || value(first.Target.Label) != "penyusun.uji" {
		t.Errorf("sasaran: %+v", first.Target)
	}
	second := list.Items[1]
	if second.Actor.Type != AuditActorTypeSystem || second.Actor.Login != nil || second.Target != nil ||
		second.Details["login"] != "admin.uji" {
		t.Errorf("entri sistem: %+v", second)
	}

	// Tanpa parameter: entri terbaru, 50 per halaman. Halaman yang tidak
	// penuh berarti tidak ada entri yang lebih lama.
	rec = srv.do(http.MethodGet, "/api/v1/audit-log", "")
	if fake.gotBefore != 0 || fake.gotLimit != 50 {
		t.Errorf("bawaan: before %d limit %d", fake.gotBefore, fake.gotLimit)
	}
	list = AuditEntryList{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || list.NextBefore != nil {
		t.Errorf("next_before pada halaman yang tidak penuh: %s", rec.Body.String())
	}

	// Parameter yang bukan angka ditolak sebelum sampai ke layanan.
	if rec := srv.do(http.MethodGet, "/api/v1/audit-log?limit=banyak", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("limit bukan angka: status %d", rec.Code)
	}
}

// TestVerifyAuditLogEndpoint: hasil pemeriksaan rantai, utuh atau rusak.
func TestVerifyAuditLogEndpoint(t *testing.T) {
	fake := &fakeAudit{verification: audit.Verification{Valid: true, Entries: 12, LastSeq: 12, LastHash: "cc"}}
	srv := newTestServer(t, nil, Deps{Audit: fake})

	var got AuditVerification
	rec := srv.do(http.MethodGet, "/api/v1/audit-log/verification", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if !got.Valid || got.Entries != 12 || value(got.LastHash) != "cc" || got.Reason != nil || got.BrokenSeq != nil {
		t.Errorf("rantai utuh: %s", rec.Body.String())
	}

	fake.verification = audit.Verification{Entries: 3, LastSeq: 2, LastHash: "bb", BrokenSeq: 3, Reason: audit.ReasonHashMismatch}
	got = AuditVerification{}
	rec = srv.do(http.MethodGet, "/api/v1/audit-log/verification", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Valid || value(got.BrokenSeq) != 3 || got.Reason == nil || *got.Reason != AuditVerificationReasonHashMismatch {
		t.Errorf("rantai rusak: %s", rec.Body.String())
	}
}

// TestAuditLogForbidden: tanpa hak A-4 jawabannya 403 (D-75).
func TestAuditLogForbidden(t *testing.T) {
	srv := newTestServer(t, nil, Deps{Audit: &fakeAudit{err: access.ErrForbidden}})
	for _, path := range []string{"/api/v1/audit-log", "/api/v1/audit-log/verification"} {
		rec := srv.do(http.MethodGet, path, "")
		if rec.Code != http.StatusForbidden || decodeError(t, rec).Code != ErrorCodeForbidden {
			t.Errorf("%s: status %d", path, rec.Code)
		}
	}
}

// TestRequestReachesAudit: request ID dan alamat IP pengirim sampai ke layanan
// domain, yang mencatatnya di audit log (R-API-03).
func TestRequestReachesAudit(t *testing.T) {
	fake := &fakeCustomers{}
	srv := newTestServer(t, nil, Deps{Customers: fake})
	// httptest.NewRequest memakai alamat pengirim 192.0.2.1:1234.
	srv.do(http.MethodGet, "/api/v1/customers", "", "X-Request-Id", "uji-audit-1")
	if fake.gotRequest != (audit.Request{ID: "uji-audit-1", IP: "192.0.2.1"}) {
		t.Errorf("asal permintaan di layanan: %+v", fake.gotRequest)
	}
}
