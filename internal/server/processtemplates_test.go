package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// TestProcessTemplateEndpoints: handler process template (M3.2) meneruskan
// isian ke layanan dan menulis jawaban sesuai kontrak. Isian yang tidak
// dikirim tetap nil, supaya layanan membiarkan nilainya (D-111).
func TestProcessTemplateEndpoints(t *testing.T) {
	created := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	iqc := template.ProcessRef{ID: "p-1", Name: "IQC"}
	fake := &fakeProcessTemplates{
		templates: []template.ProcessTemplate{{
			ID: "pt-1", Process: iqc, CreatedAt: created,
			Versions: []template.ProcessVersionSummary{{ID: "v-1", Version: 1, Status: template.StatusDraft}},
		}},
		version: template.ProcessVersion{
			ID: "v-1", TemplateID: "pt-1", Process: iqc, Version: 1, Status: template.StatusDraft, CreatedAt: created,
			Subprocesses: []template.Subprocess{
				{ID: "s-1", Name: "IQC MSD", Position: 1, Points: []template.Point{{
					ID: "pp-1", DocumentType: template.DocumentTypeRef{ID: "dt-3", Key: "cp", Name: "CP"},
					Text: "Check MSL label", Position: 1, Scope: template.Scope{Level: template.ScopeGeneral},
					Variants: []template.Variant{{
						ID: "va-1", Text: "Check MSL label level 3", Hidden: false,
						Scope: template.Scope{Level: template.ScopeModel, TargetID: "m-1", Label: "K-125"},
					}},
				}}},
				{ID: "s-2", Name: "IQC PCB", Position: 2, Hidden: true, Points: []template.Point{}},
			},
		},
	}
	srv := newTestServer(t, nil, Deps{ProcessTemplates: fake})

	rec := srv.do(http.MethodGet, "/api/v1/process-templates", "")
	want := `{"items":[{"created_at":"2026-10-31T08:00:00Z","id":"pt-1","process":{"id":"p-1","name":"IQC"},` +
		`"versions":[{"id":"v-1","status":"draft","version":1}]}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("daftar: status %d\n%s\ningin\n%s", rec.Code, rec.Body.String(), want)
	}

	rec = srv.do(http.MethodGet, "/api/v1/process-template-versions/v-1", "")
	want = `{"created_at":"2026-10-31T08:00:00Z","id":"v-1","process":{"id":"p-1","name":"IQC"},"status":"draft",` +
		`"subprocesses":[{"hidden":false,"id":"s-1","name":"IQC MSD","position":1,"points":[{"document_type":{"id":"dt-3","key":"cp","name":"CP"},` +
		`"hidden":false,"id":"pp-1","position":1,"text":"Check MSL label","content":{},"scope":{"level":"general"},` +
		`"variants":[{"hidden":false,"id":"va-1","scope":{"label":"K-125","level":"model","target_id":"m-1"},"text":"Check MSL label level 3","content":{}}]}]},` +
		`{"hidden":true,"id":"s-2","name":"IQC PCB","position":2,"points":[]}],"template_id":"pt-1","version":1}`
	if rec.Code != http.StatusOK || fake.got != (called{method: "GetProcessVersion", id: "v-1"}) || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("versi: status %d, %+v\n%s\ningin\n%s", rec.Code, fake.got, rec.Body.String(), want)
	}

	rec = srv.do(http.MethodPost, "/api/v1/process-templates", `{"process_id":"p-1"}`)
	if rec.Code != http.StatusCreated || fake.got != (called{method: "CreateProcessTemplate", input: "p-1"}) {
		t.Errorf("buat: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/subprocesses", `{"name":"IQC Connector"}`)
	if rec.Code != http.StatusCreated || fake.got != (called{method: "AddSubprocess", id: "v-1", input: "IQC Connector"}) {
		t.Errorf("tambah sub-proses: status %d, %+v", rec.Code, fake.got)
	}

	// Hanya nama yang dikirim: position dan hidden tetap nil.
	rec = srv.do(http.MethodPut, "/api/v1/process-template-versions/v-1/subprocesses/s-2", `{"name":"IQC Board"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateSubprocess", id: "v-1", input: "s-2"}) ||
		!reflect.DeepEqual(fake.gotSub, template.SubprocessUpdate{Name: "IQC Board"}) {
		t.Errorf("ubah sub-proses tanpa isian lain: status %d, %+v", rec.Code, fake.gotSub)
	}
	rec = srv.do(http.MethodPut, "/api/v1/process-template-versions/v-1/subprocesses/s-2", `{"name":"IQC Board","hidden":false,"position":1}`)
	if g := fake.gotSub; rec.Code != http.StatusOK || g.Hidden == nil || *g.Hidden || g.Position == nil || *g.Position != 1 {
		t.Errorf("ubah sub-proses: status %d, %+v", rec.Code, g)
	}

	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/points",
		`{"subprocess_id":"s-1","document_type_id":"dt-2","text":"Floor life exceeded"}`)
	wantPoint := template.NewPoint{SubprocessID: "s-1", DocumentTypeID: "dt-2", Text: "Floor life exceeded"}
	if rec.Code != http.StatusCreated || fake.got.method != "AddPoint" || !reflect.DeepEqual(fake.gotNew, wantPoint) {
		t.Errorf("tambah poin: status %d, %+v", rec.Code, fake.gotNew)
	}
	rec = srv.do(http.MethodPut, "/api/v1/process-template-versions/v-1/points/pp-1", `{"text":"Check MSL label","hidden":true,"content":{"failure_mode":"Wrong label"}}`)
	if g := fake.gotPoint; rec.Code != http.StatusOK || fake.got != (called{method: "UpdatePoint", id: "v-1", input: "pp-1"}) ||
		g.Text != "Check MSL label" || g.Hidden == nil || !*g.Hidden || g.Position != nil || g.Content == nil || (*g.Content)["failure_mode"] != "Wrong label" {
		t.Errorf("ubah poin: status %d, %+v", rec.Code, g)
	}

	// Poin dengan cakupan (M3.3): tingkat dan sasaran diteruskan apa adanya.
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/points",
		`{"subprocess_id":"s-1","document_type_id":"dt-2","text":"Cek khusus","scope_level":"customer","scope_target_id":"c-1"}`)
	wantPoint.Text, wantPoint.Scope = "Cek khusus", template.ScopeInput{Level: "customer", TargetID: "c-1"}
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotNew, wantPoint) {
		t.Errorf("tambah poin dengan cakupan: status %d, %+v", rec.Code, fake.gotNew)
	}

	// Varian (M3.3, D-129).
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/points/pp-1/variants",
		`{"scope_level":"model","scope_target_id":"m-1","text":"Laser type: fiber 20 W"}`)
	wantVariant := template.NewVariant{Scope: template.ScopeInput{Level: "model", TargetID: "m-1"}, Text: "Laser type: fiber 20 W"}
	if rec.Code != http.StatusCreated || fake.got != (called{method: "AddVariant", id: "v-1", input: "pp-1"}) || !reflect.DeepEqual(fake.gotVariant, wantVariant) {
		t.Errorf("tambah varian: status %d, %+v, %+v", rec.Code, fake.got, fake.gotVariant)
	}
	rec = srv.do(http.MethodPut, "/api/v1/process-template-versions/v-1/variants/va-1", `{"text":"Laser type: fiber 30 W"}`)
	if g := fake.gotVariantUpdate; rec.Code != http.StatusOK || fake.got != (called{method: "UpdateVariant", id: "v-1", input: "va-1"}) ||
		g.Text != "Laser type: fiber 30 W" || g.Hidden != nil {
		t.Errorf("ubah varian: status %d, %+v", rec.Code, g)
	}

	// Pratinjau poin untuk satu model: model tanpa group tidak mengirim isian group.
	cp := template.DocumentTypeRef{ID: "dt-3", Key: "cp", Name: "CP"}
	fake.resolved = template.Resolved{
		Model: template.ResolvedModel{ID: "m-2", Name: "K-200", CustomerID: "c-1", CustomerCode: "AB"},
		Subprocesses: []template.ResolvedSubprocess{{ID: "s-1", Name: "IQC MSD", Points: []template.ResolvedPoint{{
			PointID: "va-2", OriginID: "pp-1", DocumentType: cp, Text: "Customer AB check",
			Scope: template.Scope{Level: template.ScopeCustomer, TargetID: "c-1", Label: "AB"},
		}}}},
	}
	rec = srv.do(http.MethodGet, "/api/v1/process-template-versions/v-1/resolved?model_id=m-2", "")
	want = `{"model":{"customer_code":"AB","customer_id":"c-1","id":"m-2","name":"K-200"},` +
		`"subprocesses":[{"id":"s-1","name":"IQC MSD","points":[{"document_type":{"id":"dt-3","key":"cp","name":"CP"},` +
		`"origin_id":"pp-1","point_id":"va-2","scope":{"label":"AB","level":"customer","target_id":"c-1"},"text":"Customer AB check","content":{}}]}]}`
	if rec.Code != http.StatusOK || fake.got != (called{method: "ResolvePoints", id: "v-1", input: "m-2"}) || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("pratinjau: status %d, %+v\n%s\ningin\n%s", rec.Code, fake.got, rec.Body.String(), want)
	}

	// Menerbitkan dan membuat versi baru (M3.4): waktu terbit ikut di jawaban.
	published := time.Date(2026, 11, 10, 9, 30, 0, 0, time.UTC)
	fake.version.Status, fake.version.PublishedAt = template.StatusPublished, &published
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/publish", "")
	var body struct {
		Status      string `json:"status"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || fake.got != (called{method: "PublishProcessVersion", id: "v-1"}) ||
		body.Status != "published" || body.PublishedAt != "2026-11-10T09:30:00Z" {
		t.Errorf("terbitkan: status %d, %+v, %+v", rec.Code, fake.got, body)
	}
	rec = srv.do(http.MethodPost, "/api/v1/process-templates/pt-1/versions", "")
	if rec.Code != http.StatusCreated || fake.got != (called{method: "NewProcessVersion", id: "pt-1"}) {
		t.Errorf("versi baru: status %d, %+v", rec.Code, fake.got)
	}

	// Penolakan layanan diteruskan dalam format error API (R-API-13).
	fake.err = &problem.ConflictError{Problems: []problem.Problem{{Field: "name", Code: problem.Taken, Message: "Taken."}}}
	rec = srv.do(http.MethodPost, "/api/v1/process-template-versions/v-1/subprocesses", `{"name":"IQC MSD"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("nama kembar: status %d, %s", rec.Code, rec.Body.String())
	}
}
