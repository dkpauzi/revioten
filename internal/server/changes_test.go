package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/dkpauzi/revioten/internal/change"
)

// TestChangeMasterEndpoints: jenis perubahan, sumber perubahan, dan kategori
// 4M (M1.6). Sumber tanpa mekanisme dikirim tanpa field mechanism.
func TestChangeMasterEndpoints(t *testing.T) {
	content := change.ChangeType{ID: "ct-1", Key: change.KeyContent, Name: "Content"}
	claim := change.ChangeSource{ID: "cs-1", Name: "Customer claim", Active: true, CreatedAt: waktuUji}
	yokoten := change.ChangeSource{ID: "cs-2", Name: "Yokoten", Mechanism: change.MechanismYokoten, Active: true, CreatedAt: waktuUji}
	man := change.FourMCategory{ID: "fm-1", Key: "man", Name: "Man"}
	fake := &fakeChanges{
		types: []change.ChangeType{content}, changeType: content,
		sources: []change.ChangeSource{claim, yokoten}, source: claim,
		categories: []change.FourMCategory{man}, category: man,
	}
	srv := newTestServer(t, nil, Deps{Changes: fake})

	var types ChangeTypeList
	rec := srv.do(http.MethodGet, "/api/v1/change-types", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &types) != nil ||
		!reflect.DeepEqual(types.Items, []ChangeType{{Id: "ct-1", Key: ChangeTypeKeyContent, Name: "Content"}}) {
		t.Errorf("jenis perubahan: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/change-types/ct-1", `{"name":"Content change"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "RenameChangeType", id: "ct-1", input: "Content change"}) {
		t.Errorf("ganti nama jenis perubahan: status %d, %+v", rec.Code, fake.got)
	}

	var sources ChangeSourceList
	rec = srv.do(http.MethodGet, "/api/v1/change-sources", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &sources) != nil {
		t.Fatalf("sumber perubahan: status %d: %s", rec.Code, rec.Body.String())
	}
	mechanism := ChangeSourceMechanismYokoten
	want := []ChangeSource{
		{Id: "cs-1", Name: "Customer claim", Active: true, CreatedAt: waktuUji},
		{Id: "cs-2", Name: "Yokoten", Mechanism: &mechanism, Active: true, CreatedAt: waktuUji},
	}
	if !reflect.DeepEqual(sources.Items, want) {
		t.Errorf("sumber perubahan: %s", rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/change-sources", `{"name":"Kaizen"}`)
	if rec.Code != http.StatusCreated || fake.got != (called{method: "CreateChangeSource", input: "Kaizen"}) {
		t.Errorf("tambah sumber: status %d, %+v", rec.Code, fake.got)
	}
	// active yang tidak dikirim berarti status tetap.
	rec = srv.do(http.MethodPut, "/api/v1/change-sources/cs-1", `{"name":"Customer complaint"}`)
	in, _ := fake.got.input.(change.ChangeSourceUpdate)
	if rec.Code != http.StatusOK || fake.got.id != "cs-1" || in.Name != "Customer complaint" || in.Active != nil {
		t.Errorf("ubah nama sumber: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/change-sources/cs-1", `{"name":"Customer claim","active":false}`)
	in, _ = fake.got.input.(change.ChangeSourceUpdate)
	if rec.Code != http.StatusOK || in.Active == nil || *in.Active {
		t.Errorf("nonaktifkan sumber: status %d, %+v", rec.Code, fake.got)
	}

	var categories FourMCategoryList
	rec = srv.do(http.MethodGet, "/api/v1/four-m-categories", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &categories) != nil ||
		!reflect.DeepEqual(categories.Items, []FourMCategory{{Id: "fm-1", Key: FourMCategoryKeyMan, Name: "Man"}}) {
		t.Errorf("kategori 4M: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/four-m-categories/fm-1", `{"name":"Manpower"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "RenameFourMCategory", id: "fm-1", input: "Manpower"}) {
		t.Errorf("ganti nama kategori 4M: status %d, %+v", rec.Code, fake.got)
	}
}
