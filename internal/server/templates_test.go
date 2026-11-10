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
			ID: "ft-1", Name: "PFMEA AIAG 4th", DocumentTypeID: "dt-2", DocumentTypeKey: "pfmea", DocumentTypeName: "PFMEA",
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
		`"id":"ft-1","methodology":"aiag_4th","name":"PFMEA AIAG 4th","versions":[{"id":"v-1","risk_method":"rpn","status":"draft","version":1}]}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("daftar: status %d\n%s\ningin\n%s", rec.Code, rec.Body.String(), want)
	}

	rec = srv.do(http.MethodGet, "/api/v1/format-template-versions/v-1", "")
	want = `{"created_at":"2026-10-31T08:00:00Z","fields":[{"hidden":false,"id":"f-1","kind":"select","label":"Flow Symbol",` +
		`"options":[{"key":"process","label":"Process"}],"position":1,"required":false,"area":"row"}],` +
		`"highlight_rpn_above":0,"highlight_severity_at_least":0,"id":"v-1","ratings":[],"risk_method":"none","status":"draft",` +
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
