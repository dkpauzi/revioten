package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/template"
)

// TestFormatTemplateEndpoints: handler format template (M3.1) meneruskan
// isian ke layanan dan menulis jawaban sesuai kontrak. Isian yang tidak
// dikirim tetap nil, supaya layanan membiarkan nilainya (D-111).
func TestFormatTemplateEndpoints(t *testing.T) {
	created := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	fake := &fakeTemplates{
		templates: []template.FormatTemplate{{
			ID: "ft-1", Name: "PFMEA v1", DocumentTypeID: "dt-2", DocumentTypeKey: "pfmea", DocumentTypeName: "PFMEA",
			Methodology: template.MethodAIAG4th, CreatedAt: created,
			Versions: []template.VersionSummary{{ID: "v-1", Version: 1, Status: template.StatusDraft, RiskMethod: template.RiskRPN}},
		}},
		version: template.FormatVersion{
			ID: "v-1", TemplateID: "ft-1", Version: 1, Status: template.StatusDraft, RiskMethod: template.RiskNone, CreatedAt: created,
			Fields: []template.Field{{ID: "f-1", Area: template.AreaRow, Kind: template.KindSelect, Label: "Flow Symbol", Position: 1,
				Options: []template.Option{{Key: "process", Label: "Process"}}}},
			Symbols: []template.Symbol{{Key: "revision", Name: "Revision", Automatic: true}, {Key: "regulation", Name: "Regulation"}},
		},
	}
	srv := newTestServer(t, nil, Deps{Templates: fake})

	rec := srv.do(http.MethodGet, "/api/v1/format-templates", "")
	want := `{"items":[{"created_at":"2026-10-31T08:00:00Z","document_type":{"id":"dt-2","key":"pfmea","name":"PFMEA"},` +
		`"hidden":false,"id":"ft-1","methodology":"aiag_4th","name":"PFMEA v1","versions":[{"id":"v-1","risk_method":"rpn","status":"draft","version":1}]}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("daftar: status %d\n%s\ningin\n%s", rec.Code, rec.Body.String(), want)
	}

	rec = srv.do(http.MethodGet, "/api/v1/format-template-versions/v-1", "")
	want = `{"created_at":"2026-10-31T08:00:00Z","fields":[{"hidden":false,"id":"f-1","kind":"select","label":"Flow Symbol",` +
		`"options":[{"key":"process","label":"Process"}],"position":1,"required":false,"area":"row"}],` +
		`"ap_rules":[],"highlight_rpn_above":0,"highlight_severity_at_least":0,"id":"v-1","ratings":[],"risk_method":"none","status":"draft",` +
		`"symbols":[{"automatic":true,"key":"revision","name":"Revision"},{"automatic":false,"key":"regulation","name":"Regulation"}],` +
		`"template_id":"ft-1","version":1}` + "\n"
	if rec.Code != http.StatusOK || fake.got != (called{method: "GetFormatVersion", id: "v-1"}) {
		t.Errorf("versi: status %d, %+v", rec.Code, fake.got)
	}
	if got := rec.Body.String(); !sameJSON(t, got, want) {
		t.Errorf("isi versi:\n%s\ningin\n%s", got, want)
	}

	rec = srv.do(http.MethodPost, "/api/v1/format-templates", `{"name":"PFMEA SQAM KT","document_type_id":"dt-2","methodology":"sqam","customer_id":"c-1","risk_method":"ap"}`)
	wantNew := template.NewTemplate{Name: "PFMEA SQAM KT", DocumentTypeID: "dt-2", Methodology: "sqam", CustomerID: "c-1", RiskMethod: "ap"}
	if rec.Code != http.StatusCreated || fake.got != (called{method: "CreateFormatTemplate", input: wantNew}) {
		t.Errorf("buat: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/format-templates/ft-1", `{"name":"PFMEA baru"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "RenameFormatTemplate", id: "ft-1", input: "PFMEA baru"}) {
		t.Errorf("ganti nama: status %d, %+v", rec.Code, fake.got)
	}

	rec = srv.do(http.MethodPost, "/api/v1/format-template-versions/v-1/fields",
		`{"area":"row","kind":"select","label":"Phase","options":[{"label":"Prototype"},{"label":"Production"}]}`)
	wantField := template.NewField{Area: "row", Kind: "select", Label: "Phase", Options: []template.Option{{Label: "Prototype"}, {Label: "Production"}}}
	if rec.Code != http.StatusCreated || fake.got.method != "AddField" || !reflect.DeepEqual(fake.gotNew, wantField) {
		t.Errorf("tambah field: status %d, %+v", rec.Code, fake.gotNew)
	}

	// Hanya label yang dikirim: required, hidden, position, dan options tetap nil.
	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1/fields/f-1", `{"label":"Flow"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateField", id: "v-1", input: "f-1"}) ||
		!reflect.DeepEqual(fake.gotField, template.FieldUpdate{Label: "Flow"}) {
		t.Errorf("ubah field tanpa isian lain: status %d, %+v", rec.Code, fake.gotField)
	}
	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1/fields/f-1", `{"label":"Flow","hidden":true,"position":2}`)
	if g := fake.gotField; rec.Code != http.StatusOK || g.Hidden == nil || !*g.Hidden || g.Position == nil || *g.Position != 2 || g.Required != nil {
		t.Errorf("ubah field: status %d, %+v", rec.Code, g)
	}

	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1",
		`{"highlight_severity_at_least":0}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "SetHighlights", id: "v-1", input: "-/0"}) {
		t.Errorf("aturan sorot: status %d, %+v", rec.Code, fake.got)
	}

	// Item tanpa criteria dilewati (D-111).
	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1/rating-criteria",
		`{"items":[{"scale":"severity","value":10,"criteria":"Hazardous"},{"scale":"detection","value":1}]}`)
	if rec.Code != http.StatusOK || !slices.Equal(fake.gotRatings, []template.Rating{{Scale: "severity", Value: 10, Criteria: "Hazardous"}}) {
		t.Errorf("tabel skala: status %d, %+v", rec.Code, fake.gotRatings)
	}
	// Tanda automatic dari klien diabaikan; layanan memakai tanda simbol lama.
	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1/symbols",
		`{"items":[{"key":"revision","name":"Revision Mark"},{"name":"Safety","automatic":true}]}`)
	if rec.Code != http.StatusOK || !slices.Equal(fake.gotSymbols, []template.Symbol{{Key: "revision", Name: "Revision Mark"}, {Name: "Safety"}}) {
		t.Errorf("simbol: status %d, %+v", rec.Code, fake.gotSymbols)
	}

	// Tabel AP dan hitungannya (form set langkah A, D-134).
	rec = srv.do(http.MethodPut, "/api/v1/format-template-versions/v-1/ap-rules",
		`{"items":[{"s_min":1,"s_max":10,"o_min":1,"o_max":10,"d_min":1,"d_max":10,"priority":"L"}]}`)
	wantRules := []template.APRule{{SMin: 1, SMax: 10, OMin: 1, OMax: 10, DMin: 1, DMax: 10, Priority: "L"}}
	if rec.Code != http.StatusOK || fake.got != (called{method: "SetAPRules", id: "v-1"}) || !slices.Equal(fake.gotAPRules, wantRules) {
		t.Errorf("tabel AP: status %d, %+v", rec.Code, fake.gotAPRules)
	}
	rec = srv.do(http.MethodGet, "/api/v1/format-template-versions/v-1/action-priority?severity=8&occurrence=6&detection=4", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "ActionPriorityOf", id: "v-1", input: [3]int{8, 6, 4}}) ||
		!sameJSON(t, rec.Body.String(), `{"priority":"H"}`) {
		t.Errorf("hitung AP: status %d, %+v, %s", rec.Code, fake.got, rec.Body.String())
	}

	// Form set (langkah D, D-133, D-139).
	sheet := func(id, name string) template.SheetRef {
		return template.SheetRef{FormatVersionID: id, TemplateName: name, Version: 1, Status: template.StatusDraft}
	}
	fake.formSet = template.FormSet{ID: "fs-1", Key: "aiag_4th", Name: "AIAG 4th set", Default: true, Versions: []template.FormSetVersion{{
		ID: "fsv-1", Version: 1, Status: template.StatusDraft, PFD: sheet("v-1", "PFD"), PFMEA: sheet("v-2", "PFMEA v1"), CP: sheet("v-3", "CP"),
	}}}
	fake.formSets = template.FormSetList{
		Sets:     []template.FormSet{fake.formSet},
		Defaults: []template.CustomerFormSet{{CustomerID: "c-1", CustomerCode: "AB", CustomerName: "Customer AB"}, {CustomerID: "c-2", CustomerCode: "CD", CustomerName: "Customer CD", FormSetID: "fs-1"}},
	}
	rec = srv.do(http.MethodGet, "/api/v1/form-sets", "")
	want = `{"items":[{"id":"fs-1","key":"aiag_4th","name":"AIAG 4th set","default":true,"hidden":false,"versions":[{"id":"fsv-1","version":1,"status":"draft",` +
		`"pfd":{"format_version_id":"v-1","template_name":"PFD","version":1,"status":"draft"},` +
		`"pfmea":{"format_version_id":"v-2","template_name":"PFMEA v1","version":1,"status":"draft"},` +
		`"cp":{"format_version_id":"v-3","template_name":"CP","version":1,"status":"draft"}}]}],` +
		`"customer_defaults":[{"customer_id":"c-1","customer_code":"AB","customer_name":"Customer AB"},` +
		`{"customer_id":"c-2","customer_code":"CD","customer_name":"Customer CD","form_set_id":"fs-1"}]}`
	if rec.Code != http.StatusOK || !sameJSON(t, rec.Body.String(), want) {
		t.Errorf("daftar form set: status %d\n%s\ningin\n%s", rec.Code, rec.Body.String(), want)
	}
	wantSheets := template.FormSetSheets{PFD: "v-1", PFMEA: "v-2", CP: "v-3"}
	sheetsBody := `"pfd_format_version_id":"v-1","pfmea_format_version_id":"v-2","cp_format_version_id":"v-3"}`
	rec = srv.do(http.MethodPost, "/api/v1/form-sets", `{"name":"Set baru",`+sheetsBody)
	if rec.Code != http.StatusCreated || fake.got != (called{method: "CreateFormSet", input: template.NewFormSet{Name: "Set baru", Sheets: wantSheets}}) {
		t.Errorf("buat form set: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/form-set-versions/fsv-1", `{`+sheetsBody)
	if rec.Code != http.StatusOK || fake.got != (called{method: "SetFormSetSheets", id: "fsv-1", input: wantSheets}) {
		t.Errorf("ubah sheet: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/form-sets/fs-1/versions", "")
	if rec.Code != http.StatusCreated || fake.got != (called{method: "NewFormSetVersion", id: "fs-1"}) {
		t.Errorf("versi form set baru: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/form-set-versions/fsv-1/publish", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "PublishFormSetVersion", id: "fsv-1"}) {
		t.Errorf("terbitkan form set: status %d, %+v", rec.Code, fake.got)
	}
	// Tanpa form_set_id berarti kembali ke set bawaan.
	rec = srv.do(http.MethodPut, "/api/v1/customers/c-1/default-form-set", `{"form_set_id":"fs-1"}`)
	if rec.Code != http.StatusNoContent || fake.got != (called{method: "SetCustomerFormSet", id: "c-1", input: "fs-1"}) {
		t.Errorf("form set customer: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/customers/c-1/default-form-set", `{}`)
	if rec.Code != http.StatusNoContent || fake.got != (called{method: "SetCustomerFormSet", id: "c-1", input: ""}) {
		t.Errorf("form set customer dikosongkan: status %d, %+v", rec.Code, fake.got)
	}

	// Menerbitkan dan membuat versi baru (M3.4).
	rec = srv.do(http.MethodPost, "/api/v1/format-template-versions/v-1/publish", "")
	if rec.Code != http.StatusOK || fake.got != (called{method: "PublishFormatVersion", id: "v-1"}) {
		t.Errorf("terbitkan: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/format-templates/t-1/versions", "")
	if rec.Code != http.StatusCreated || fake.got != (called{method: "NewFormatVersion", id: "t-1"}) {
		t.Errorf("versi baru: status %d, %+v", rec.Code, fake.got)
	}
}

// sameJSON membandingkan dua teks JSON tanpa memedulikan urutan key.
func sameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var a, b any
	if err := json.Unmarshal([]byte(got), &a); err != nil {
		t.Fatalf("jawaban bukan JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatalf("JSON yang diinginkan salah: %v", err)
	}
	return reflect.DeepEqual(a, b)
}
