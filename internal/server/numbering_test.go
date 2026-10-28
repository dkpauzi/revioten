package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/dkpauzi/revioten/internal/numbering"
)

// TestNumberingEndpoints: kode pabrik, jenis dokumen dengan awalannya, dan
// kode khusus (M1.6). Jenis dokumen tanpa key dikirim tanpa field key.
func TestNumberingEndpoints(t *testing.T) {
	pfd := numbering.DocumentType{ID: "dt-1", Key: numbering.KeyPFD, Name: "PFD", Letter: "H", Prefix: "SH"}
	wi := numbering.DocumentType{ID: "dt-4", Name: "Work Instruction", Letter: "W", Prefix: "SW"}
	zz := numbering.SpecialCode{ID: "sc-1", Code: "ZZ", Name: "General/Common", CreatedAt: waktuUji}
	fake := &fakeNumbering{
		settings: numbering.Settings{ID: "ns-1", FactoryCode: "S"},
		types:    []numbering.DocumentType{pfd, wi}, docType: wi,
		codes: []numbering.SpecialCode{zz}, code: zz,
	}
	srv := newTestServer(t, nil, Deps{Numbering: fake})

	rec := srv.do(http.MethodGet, "/api/v1/number-settings", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"factory_code":"S"}`+"\n" {
		t.Errorf("pengaturan nomor: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/number-settings", `{"factory_code":"t"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "SetFactoryCode", input: "t"}) {
		t.Errorf("ganti kode pabrik: status %d, %+v", rec.Code, fake.got)
	}

	var types DocumentTypeList
	rec = srv.do(http.MethodGet, "/api/v1/document-types", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &types) != nil {
		t.Fatalf("jenis dokumen: status %d: %s", rec.Code, rec.Body.String())
	}
	key := DocumentTypeKeyPfd
	want := []DocumentType{
		{Id: "dt-1", Key: &key, Name: "PFD", Letter: "H", Prefix: "SH"},
		{Id: "dt-4", Name: "Work Instruction", Letter: "W", Prefix: "SW"},
	}
	if !reflect.DeepEqual(types.Items, want) {
		t.Errorf("jenis dokumen: %s", rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/document-types", `{"name":"Work Instruction","letter":"w"}`)
	if rec.Code != http.StatusCreated || fake.got.input != (numbering.DocumentTypeInput{Name: "Work Instruction", Letter: "w"}) {
		t.Errorf("tambah jenis dokumen: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/document-types/dt-1", `{"name":"Process Flow Diagram","letter":"H"}`)
	if rec.Code != http.StatusOK || fake.got.id != "dt-1" || fake.got.input != (numbering.DocumentTypeInput{Name: "Process Flow Diagram", Letter: "H"}) {
		t.Errorf("ubah jenis dokumen: status %d, %+v", rec.Code, fake.got)
	}

	var codes SpecialCodeList
	rec = srv.do(http.MethodGet, "/api/v1/special-codes", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &codes) != nil ||
		!reflect.DeepEqual(codes.Items, []SpecialCode{{Id: "sc-1", Code: "ZZ", Name: "General/Common", CreatedAt: waktuUji}}) {
		t.Errorf("kode khusus: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPost, "/api/v1/special-codes", `{"code":"qa","name":"Quality"}`)
	if rec.Code != http.StatusCreated || fake.got.input != (numbering.NewSpecialCode{Code: "qa", Name: "Quality"}) {
		t.Errorf("tambah kode khusus: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/special-codes/sc-1", `{"name":"General"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "RenameSpecialCode", id: "sc-1", input: "General"}) {
		t.Errorf("ganti nama kode khusus: status %d, %+v", rec.Code, fake.got)
	}
}
