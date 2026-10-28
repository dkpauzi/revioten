package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/dkpauzi/revioten/internal/process"
)

// TestProcessEndpoints: daftar, satu proses menurut ID tetapnya, tambah, dan
// ubah. Alias selalu dikirim sebagai daftar, juga bila kosong.
func TestProcessEndpoints(t *testing.T) {
	iqc := process.Process{ID: "pr-1", Name: "IQC", Aliases: []string{"Incoming QC"}, CreatedAt: waktuUji}
	smt := process.Process{ID: "pr-2", Name: "SMT", Aliases: []string{}, CreatedAt: waktuUji}
	fake := &fakeProcesses{list: []process.Process{iqc, smt}, one: iqc}
	srv := newTestServer(t, nil, Deps{Processes: fake})

	rec := srv.do(http.MethodGet, "/api/v1/processes", "")
	var list ProcessList
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil {
		t.Fatalf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	want := []Process{
		{Id: "pr-1", Name: "IQC", Aliases: []string{"Incoming QC"}, CreatedAt: waktuUji},
		{Id: "pr-2", Name: "SMT", Aliases: []string{}, CreatedAt: waktuUji},
	}
	if !reflect.DeepEqual(list.Items, want) || !strings.Contains(rec.Body.String(), `"aliases":[]`) {
		t.Errorf("daftar: %s", rec.Body.String())
	}

	rec = srv.do(http.MethodGet, "/api/v1/processes/pr-1", "")
	if rec.Code != http.StatusOK || fake.got.method != "Get" || fake.got.id != "pr-1" {
		t.Errorf("satu proses: status %d, %+v", rec.Code, fake.got)
	}

	rec = srv.do(http.MethodPost, "/api/v1/processes", `{"name":"IQC","aliases":["Incoming QC"]}`)
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.got.input, process.Input{Name: "IQC", Aliases: []string{"Incoming QC"}}) {
		t.Errorf("tambah: status %d, input %+v", rec.Code, fake.got.input)
	}
	// Alias boleh tidak dikirim saat menambah.
	rec = srv.do(http.MethodPost, "/api/v1/processes", `{"name":"AOI"}`)
	if rec.Code != http.StatusCreated || fake.got.input.Name != "AOI" || len(fake.got.input.Aliases) != 0 {
		t.Errorf("tambah tanpa alias: status %d, input %+v", rec.Code, fake.got.input)
	}

	rec = srv.do(http.MethodPut, "/api/v1/processes/pr-1", `{"name":"Incoming QC","aliases":["IQC"]}`)
	if rec.Code != http.StatusOK || fake.got.method != "Update" || fake.got.id != "pr-1" ||
		!reflect.DeepEqual(fake.got.input, process.Input{Name: "Incoming QC", Aliases: []string{"IQC"}}) {
		t.Errorf("ubah: status %d, %+v", rec.Code, fake.got)
	}
	// [] berarti semua alias dilepas.
	rec = srv.do(http.MethodPut, "/api/v1/processes/pr-1", `{"name":"IQC","aliases":[]}`)
	if rec.Code != http.StatusOK || fake.got.input.Aliases == nil || len(fake.got.input.Aliases) != 0 {
		t.Errorf("lepas semua alias: status %d, input %#v", rec.Code, fake.got.input)
	}
}

// TestUpdateProcessNeedsAliases: saat mengubah, aliases wajib dikirim (kontrak).
// Tanpa aturan ini, klien yang lupa field itu akan menghapus semua alias.
func TestUpdateProcessNeedsAliases(t *testing.T) {
	fake := &fakeProcesses{}
	srv := newTestServer(t, nil, Deps{Processes: fake})

	rec := srv.do(http.MethodPut, "/api/v1/processes/pr-1", `{"name":"IQC"}`)
	var body Error
	if rec.Code != http.StatusUnprocessableEntity || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if body.Code != ErrorCodeValidationFailed || body.Details == nil || len(*body.Details) != 1 ||
		(*body.Details)[0].Field != "aliases" || (*body.Details)[0].Code != ErrorDetailCodeRequired {
		t.Errorf("jawaban: %s", rec.Body.String())
	}
	if fake.got.method != "" {
		t.Errorf("service tetap dipanggil: %+v", fake.got)
	}
}
