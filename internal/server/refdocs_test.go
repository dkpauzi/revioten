package server

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/template"
)

// TestRefDocEndpoints: handler master dokumen terkait (D-130) meneruskan
// isian ke layanan dan menulis jawaban sesuai kontrak.
func TestRefDocEndpoints(t *testing.T) {
	created := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	fake := &fakeRefDocs{doc: template.RefDocument{
		ID: "rd-1", Number: "SW-ZZ-103", Title: "Material Receiving", Level: "WI (Level 3)", Revision: "19", RevisionDate: "2026-06-26",
		Active: true, CreatedAt: created,
		Links: []template.RefLink{{ID: "l-1", Process: template.ProcessRef{ID: "p-1", Name: "Receiving"}, Scope: template.Scope{Level: template.ScopeGeneral}}},
	}}
	fake.list = []template.RefDocument{fake.doc}
	srv := newTestServer(t, nil, Deps{RefDocs: fake})

	rec := srv.do(http.MethodGet, "/api/v1/referenced-documents", "")
	want := `{"items":[{"id":"rd-1","number":"SW-ZZ-103","title":"Material Receiving","level":"WI (Level 3)","revision":"19",` +
		`"revision_date":"2026-06-26","active":true,"links":[{"id":"l-1","process":{"id":"p-1","name":"Receiving"},"scope":{"level":"general"}}],` +
		`"created_at":"2026-11-14T08:00:00Z"}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("daftar: status %d\n%s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/referenced-documents", `{"number":"sw-zz-103","title":"Material Receiving","revision_date":"2026-06-26"}`)
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotNew, template.NewRefDocument{Number: "sw-zz-103", Title: "Material Receiving", RevisionDate: "2026-06-26"}) {
		t.Errorf("buat: status %d, %+v", rec.Code, fake.gotNew)
	}
	rec = srv.do(http.MethodPut, "/api/v1/referenced-documents/rd-1", `{"title":"Material Receiving","active":false}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "Update", id: "rd-1"}) || fake.gotUpd.Active == nil || *fake.gotUpd.Active {
		t.Errorf("ubah: status %d, %+v", rec.Code, fake.gotUpd)
	}
	rec = srv.do(http.MethodPost, "/api/v1/referenced-documents/rd-1/links", `{"process_id":"p-1","scope_level":"model","scope_target_id":"m-1"}`)
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotLink, template.NewRefLink{ProcessID: "p-1", Scope: template.ScopeInput{Level: "model", TargetID: "m-1"}}) {
		t.Errorf("tautan: status %d, %+v", rec.Code, fake.gotLink)
	}
	rec = srv.do(http.MethodDelete, "/api/v1/referenced-documents/rd-1/links/l-1", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "RemoveLink", id: "rd-1", input: "l-1"}) {
		t.Errorf("lepas tautan: status %d, %+v", rec.Code, fake.got)
	}

	docs := &fakeDocuments{}
	srv = newTestServer(t, nil, Deps{Documents: docs})
	rec = srv.do(http.MethodPost, "/api/v1/documents/d-1/references", "")
	if docs.got != (called{method: "FillReferences", id: "d-1"}) {
		t.Errorf("isi dokumen terkait: status %d, %+v", rec.Code, docs.got)
	}
}
