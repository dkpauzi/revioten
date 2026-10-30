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
		settings: numbering.Settings{ID: "ns-1", FactoryCode: "S", Pattern: numbering.DefaultPattern, SequenceFormat: "letter_block", RevisionDigits: 2},
		types:    []numbering.DocumentType{pfd, wi}, docType: wi,
		codes: []numbering.SpecialCode{zz}, code: zz,
	}
	srv := newTestServer(t, nil, Deps{Numbering: fake})

	rec := srv.do(http.MethodGet, "/api/v1/number-settings", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"factory_code":"S","pattern":"{prefix}-{customer}-{seq}","revision_digits":2,"sequence_format":"letter_block"}`+"\n" {
		t.Errorf("pengaturan nomor: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/number-settings", `{"factory_code":"t"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateSettings", input: "t|-|-"}) {
		t.Errorf("ganti kode pabrik: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPut, "/api/v1/number-settings", `{"factory_code":"S","pattern":"{customer}/{prefix}/{seq}","revision_digits":3}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "UpdateSettings", input: "S|{customer}/{prefix}/{seq}|3"}) {
		t.Errorf("ganti pola: status %d, %+v", rec.Code, fake.got)
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

// TestDocumentNumberEndpoints: pendaftaran nomor (M2.1, M2.2). Idempotency-Key
// diteruskan dari header, permintaan baru dijawab 201, dan permintaan yang
// diulang dengan kunci yang sama dijawab 200 dengan nomor lamanya.
func TestDocumentNumberEndpoints(t *testing.T) {
	number := numbering.DocumentNumber{
		ID: "dn-1", Number: "SC-AB-001", CustomerID: "c-1", CustomerCurrentCode: "AB", CustomerName: "Customer AB",
		CustomerCode: "AB", DocumentTypeID: "dt-3", DocumentTypeKey: numbering.KeyCP, DocumentTypeName: "CP",
		Sequence: 1, Status: numbering.StatusReserved, InitialRevision: 4, RevisionLabel: "-04", BatchID: "b-1",
		Origin: numbering.OriginSingle, RegisteredByID: "p-1", RegisteredByName: "Penyusun Contoh", CreatedAt: waktuUji,
	}
	fake := &fakeNumbering{registration: numbering.Registration{Numbers: []numbering.DocumentNumber{number}}, next: "SC-AB-002"}
	srv := newTestServer(t, nil, Deps{Numbering: fake})

	rec := srv.do(http.MethodPost, "/api/v1/document-numbers", `{"customer_id":"c-1","document_type_id":"dt-3","initial_revision":4}`,
		"Idempotency-Key", "kunci-1")
	want := numbering.NewNumber{CustomerID: "c-1", DocumentTypeID: "dt-3", InitialRevision: 4, IdempotencyKey: "kunci-1"}
	if rec.Code != http.StatusCreated || fake.got != (called{method: "Register", input: want}) {
		t.Fatalf("daftar: status %d, %+v: %s", rec.Code, fake.got, rec.Body.String())
	}
	var body NumberRegistration
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	wantBody := NumberRegistration{Items: []DocumentNumber{{
		Id: "dn-1", Number: "SC-AB-001", Customer: CustomerRef{Id: "c-1", Code: "AB", Name: "Customer AB"}, CustomerCode: "AB",
		DocumentType: DocumentTypeRef{Id: "dt-3", Key: DocumentTypeRefKeyCp, Name: "CP"}, Sequence: 1,
		Status: DocumentNumberStatusReserved, InitialRevision: 4, RevisionLabel: "-04", Origin: DocumentNumberOriginSingle,
		BatchId: "b-1", RegisteredBy: Person{Id: "p-1", Name: "Penyusun Contoh"}, CreatedAt: waktuUji,
	}}}
	if !reflect.DeepEqual(body, wantBody) {
		t.Errorf("isi %+v", body)
	}

	fake.registration.Replayed = true
	if rec := srv.do(http.MethodPost, "/api/v1/document-numbers", `{"customer_id":"c-1","document_type_id":"dt-3"}`); rec.Code != http.StatusOK {
		t.Errorf("permintaan ulang: status %d, ingin 200", rec.Code)
	}

	rec = srv.do(http.MethodGet, "/api/v1/document-numbers/next?customer_id=c-1&document_type_id=dt-3", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"number":"SC-AB-002"}`+"\n" ||
		fake.got != (called{method: "NextNumber", id: "c-1", input: "dt-3"}) {
		t.Errorf("nomor berikutnya: status %d, %+v: %s", rec.Code, fake.got, rec.Body.String())
	}
}

// TestRegisterEndpoints: daftar register dengan saringan dari query, dan
// perubahan status Void dan Obsolete (M2.3).
func TestRegisterEndpoints(t *testing.T) {
	number := numbering.DocumentNumber{ID: "dn-1", Number: "SC-AB-001", Status: numbering.StatusVoid, StatusReason: "Salah daftar", CreatedAt: waktuUji}
	fake := &fakeNumbering{registration: numbering.Registration{Numbers: []numbering.DocumentNumber{number}}, number: number, next: "dn-1"}
	srv := newTestServer(t, nil, Deps{Numbering: fake})

	rec := srv.do(http.MethodGet, "/api/v1/document-numbers?customer_id=c-1&status=void&q=sc-ab&before=dn-9&limit=2", "")
	want := numbering.ListFilter{CustomerID: "c-1", Status: "void", Query: "sc-ab", Before: "dn-9", Limit: 2}
	if rec.Code != http.StatusOK || fake.got != (called{method: "ListNumbers", input: want}) {
		t.Fatalf("daftar: status %d, %+v: %s", rec.Code, fake.got, rec.Body.String())
	}
	var list DocumentNumberList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Items) != 1 || list.NextBefore == nil || *list.NextBefore != "dn-1" ||
		list.Items[0].StatusReason == nil || *list.Items[0].StatusReason != "Salah daftar" {
		t.Errorf("isi daftar: %s", rec.Body.String())
	}

	rec = srv.do(http.MethodPost, "/api/v1/document-numbers/dn-1/void", `{"reason":"Salah daftar"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "Void", id: "dn-1", input: "Salah daftar"}) {
		t.Errorf("void: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/document-numbers/dn-1/obsolete", `{}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "MakeObsolete", id: "dn-1", input: ""}) {
		t.Errorf("obsolete: status %d, %+v", rec.Code, fake.got)
	}
}

// TestDocumentNumberSetEndpoint: daftar sebagai set (M2.4) meneruskan
// customer, revisi awal, dan Idempotency-Key.
func TestDocumentNumberSetEndpoint(t *testing.T) {
	fake := &fakeNumbering{}
	srv := newTestServer(t, nil, Deps{Numbering: fake})
	rec := srv.do(http.MethodPost, "/api/v1/document-numbers/sets", `{"customer_id":"c-1","initial_revision":4}`, "Idempotency-Key", "set-1")
	want := numbering.NewSet{CustomerID: "c-1", InitialRevision: 4, IdempotencyKey: "set-1"}
	if rec.Code != http.StatusCreated || fake.got != (called{method: "RegisterSet", input: want}) {
		t.Errorf("set: status %d, %+v: %s", rec.Code, fake.got, rec.Body.String())
	}
}
