package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/dkpauzi/revioten/internal/product"
)

// TestModelGroupEndpoints: daftar per customer, tambah, dan ganti nama group model.
func TestModelGroupEndpoints(t *testing.T) {
	group := product.ModelGroup{ID: "mg-1", CustomerID: "c-1", Name: "Family", CreatedAt: waktuUji}
	fake := &fakeProducts{groups: []product.ModelGroup{group}, group: group}
	srv := newTestServer(t, nil, Deps{Products: fake})
	want := ModelGroup{Id: "mg-1", CustomerId: "c-1", Name: "Family", CreatedAt: waktuUji}

	var list ModelGroupList
	rec := srv.do(http.MethodGet, "/api/v1/model-groups?customer_id=c-1", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil ||
		!reflect.DeepEqual(list.Items, []ModelGroup{want}) || fake.got.id != "c-1" {
		t.Errorf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/model-groups", `{"customer_id":"c-1","name":"Family"}`)
	if rec.Code != http.StatusCreated || fake.got.input != (product.NewModelGroup{CustomerID: "c-1", Name: "Family"}) {
		t.Errorf("tambah: status %d, input %+v", rec.Code, fake.got.input)
	}
	rec = srv.do(http.MethodPut, "/api/v1/model-groups/mg-1", `{"name":"Family B"}`)
	if rec.Code != http.StatusOK || fake.got.method != "RenameModelGroup" || fake.got.id != "mg-1" || fake.got.input != "Family B" {
		t.Errorf("ganti nama: status %d, %+v", rec.Code, fake.got)
	}
	// customer_id wajib di alamat daftar.
	if rec := srv.do(http.MethodGet, "/api/v1/model-groups", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("tanpa customer_id: status %d, ingin 400", rec.Code)
	}
}

// TestModelEndpoints: model ditampilkan dengan group-nya bila ada (D-42).
func TestModelEndpoints(t *testing.T) {
	inGroup := product.Model{ID: "m-1", CustomerID: "c-1", ModelGroupID: "mg-1", ModelGroupName: "Family", Name: "X100", CreatedAt: waktuUji}
	alone := product.Model{ID: "m-2", CustomerID: "c-1", Name: "X200", CreatedAt: waktuUji}
	fake := &fakeProducts{models: []product.Model{inGroup, alone}, model: inGroup}
	srv := newTestServer(t, nil, Deps{Products: fake})

	var list ModelList
	rec := srv.do(http.MethodGet, "/api/v1/models?customer_id=c-1", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil {
		t.Fatalf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	want := []Model{
		{Id: "m-1", CustomerId: "c-1", Name: "X100", CreatedAt: waktuUji, ModelGroup: &ModelGroupRef{Id: "mg-1", Name: "Family"}},
		{Id: "m-2", CustomerId: "c-1", Name: "X200", CreatedAt: waktuUji},
	}
	if !reflect.DeepEqual(list.Items, want) {
		t.Errorf("items %+v, ingin %+v", list.Items, want)
	}
	rec = srv.do(http.MethodPost, "/api/v1/models", `{"customer_id":"c-1","name":"X100","model_group_id":"mg-1"}`)
	if rec.Code != http.StatusCreated || fake.got.input != (product.NewModel{CustomerID: "c-1", Name: "X100", ModelGroupID: "mg-1"}) {
		t.Errorf("tambah: status %d, input %+v", rec.Code, fake.got.input)
	}
	// Tanpa model_group_id berarti model dikeluarkan dari group-nya.
	rec = srv.do(http.MethodPut, "/api/v1/models/m-1", `{"name":"X100A"}`)
	if rec.Code != http.StatusOK || fake.got.id != "m-1" || fake.got.input != (product.ModelUpdate{Name: "X100A"}) {
		t.Errorf("ubah: status %d, %+v", rec.Code, fake.got)
	}
}

// TestPartNumberEndpoints: part number ditampilkan dengan modelnya.
func TestPartNumberEndpoints(t *testing.T) {
	part := product.PartNumber{ID: "p-1", CustomerID: "c-1", ModelID: "m-1", ModelName: "X100", Number: "PN-1", Name: "Main Board", CreatedAt: waktuUji}
	fake := &fakeProducts{parts: []product.PartNumber{part}, part: part}
	srv := newTestServer(t, nil, Deps{Products: fake})

	var list PartNumberList
	rec := srv.do(http.MethodGet, "/api/v1/part-numbers?customer_id=c-1", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil {
		t.Fatalf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	name := "Main Board"
	want := []PartNumber{{Id: "p-1", CustomerId: "c-1", Model: ModelRef{Id: "m-1", Name: "X100"}, Number: "PN-1", Name: &name, CreatedAt: waktuUji}}
	if !reflect.DeepEqual(list.Items, want) {
		t.Errorf("items %+v, ingin %+v", list.Items, want)
	}
	rec = srv.do(http.MethodPost, "/api/v1/part-numbers", `{"model_id":"m-1","number":"PN-1"}`)
	if rec.Code != http.StatusCreated || fake.got.input != (product.NewPartNumber{ModelID: "m-1", Number: "PN-1"}) {
		t.Errorf("tambah: status %d, input %+v", rec.Code, fake.got.input)
	}
	rec = srv.do(http.MethodPut, "/api/v1/part-numbers/p-1", `{"number":"PN-1A","name":"Board"}`)
	if rec.Code != http.StatusOK || fake.got.id != "p-1" || fake.got.input != (product.PartNumberUpdate{Number: "PN-1A", Name: "Board"}) {
		t.Errorf("ubah: status %d, %+v", rec.Code, fake.got)
	}
}
