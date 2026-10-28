package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/settings"
)

// TestPositionEndpoints: jabatan beserta pemegangnya, ganti nama, catat dan
// lepas pemegang (M1.8). Pemegang tanpa Project Group dikirim tanpa field itu.
func TestPositionEndpoints(t *testing.T) {
	leader := approval.Position{
		ID: "pos-1", Key: approval.KeyProjectLeader, Name: "Project Leader", PerProjectGroup: true,
		Holders: []approval.Holder{{ID: "h-1", PersonID: "p-1", PersonName: "Orang Contoh", ProjectGroupID: "g-1", ProjectGroupCode: "GR1", CreatedAt: waktuUji}},
	}
	npi := approval.Position{ID: "pos-2", Key: approval.KeyNPIManager, Name: "NPI Manager", Holders: []approval.Holder{}}
	fake := &fakeApproval{positions: []approval.Position{leader, npi}, position: npi, holder: leader.Holders[0]}
	srv := newTestServer(t, nil, Deps{Approval: fake})

	var list PositionList
	rec := srv.do(http.MethodGet, "/api/v1/positions", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil {
		t.Fatalf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	want := []Position{
		{Id: "pos-1", Key: PositionKeyProjectLeader, Name: "Project Leader", PerProjectGroup: true, Holders: []PositionHolder{{
			Id: "h-1", Person: Person{Id: "p-1", Name: "Orang Contoh"}, ProjectGroup: &ProjectGroupRef{Id: "g-1", Code: "GR1"}, CreatedAt: waktuUji,
		}}},
		{Id: "pos-2", Key: PositionKeyNpiManager, Name: "NPI Manager", Holders: []PositionHolder{}},
	}
	if !reflect.DeepEqual(list.Items, want) {
		t.Errorf("daftar: %s", rec.Body.String())
	}

	rec = srv.do(http.MethodPut, "/api/v1/positions/pos-2", `{"name":"PE Manager"}`)
	if rec.Code != http.StatusOK || fake.got != (called{method: "RenamePosition", id: "pos-2", input: "PE Manager"}) {
		t.Errorf("ganti nama: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodPost, "/api/v1/positions/pos-1/holders", `{"person_id":"p-1","project_group_id":"g-1"}`)
	if rec.Code != http.StatusCreated || fake.got != (called{method: "AddHolder", id: "pos-1", input: approval.NewHolder{PersonID: "p-1", ProjectGroupID: "g-1"}}) {
		t.Errorf("catat pemegang: status %d, %+v", rec.Code, fake.got)
	}
	rec = srv.do(http.MethodDelete, "/api/v1/positions/pos-1/holders/h-1", "")
	if rec.Code != http.StatusNoContent || fake.got.id != "pos-1" || fake.gotHolderID != "h-1" {
		t.Errorf("lepas pemegang: status %d, %+v %s", rec.Code, fake.got, fake.gotHolderID)
	}
}

// TestRouteEndpoints: rute approval beserta slotnya, tambah, dan ubah.
func TestRouteEndpoints(t *testing.T) {
	route := approval.Route{ID: "r-1", Name: "Up to GM", CreatedAt: waktuUji, Slots: []approval.Slot{
		{Stage: approval.StagePrepared, PrintTitle: "Process. Eng", Positions: []approval.PositionRef{{ID: "pos-1", Name: "PPAP Engineer"}}},
	}}
	fake := &fakeApproval{routes: []approval.Route{route}, route: route}
	srv := newTestServer(t, nil, Deps{Approval: fake})

	var list ApprovalRouteList
	rec := srv.do(http.MethodGet, "/api/v1/approval-routes", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &list) != nil {
		t.Fatalf("daftar: status %d: %s", rec.Code, rec.Body.String())
	}
	want := []ApprovalRoute{{Id: "r-1", Name: "Up to GM", CreatedAt: waktuUji, Slots: []RouteSlot{
		{Stage: RouteSlotStagePrepared, PrintTitle: "Process. Eng", Positions: []PositionRef{{Id: "pos-1", Name: "PPAP Engineer"}}},
	}}}
	if !reflect.DeepEqual(list.Items, want) {
		t.Errorf("daftar: %s", rec.Body.String())
	}

	body := `{"name":"Up to GM","slots":[{"stage":"prepared","print_title":"Process. Eng","position_ids":["pos-1"]}]}`
	wantInput := approval.RouteInput{Name: "Up to GM", Slots: []approval.SlotInput{
		{Stage: approval.StagePrepared, PrintTitle: "Process. Eng", PositionIDs: []string{"pos-1"}},
	}}
	rec = srv.do(http.MethodPost, "/api/v1/approval-routes", body)
	if rec.Code != http.StatusCreated || !reflect.DeepEqual(fake.gotRoute, wantInput) {
		t.Errorf("tambah: status %d, %+v", rec.Code, fake.gotRoute)
	}
	rec = srv.do(http.MethodPut, "/api/v1/approval-routes/r-1", body)
	if rec.Code != http.StatusOK || fake.got.id != "r-1" || !reflect.DeepEqual(fake.gotRoute, wantInput) {
		t.Errorf("ubah: status %d, %+v", rec.Code, fake.got)
	}
}

// TestDefaultRouteEndpoints: rute default customer (D-47) dan rute per jenis
// perubahan (ASUMSI E-35). route_id yang tidak dikirim di jenis perubahan
// berarti rute default customer.
func TestDefaultRouteEndpoints(t *testing.T) {
	customers := &fakeCustomers{created: customer.Customer{
		ID: "c-1", Code: "AB", Name: "Customer A", Active: true, CreatedAt: waktuUji,
		DefaultRouteID: "r-1", DefaultRouteName: "Up to GM",
	}}
	changes := &fakeChanges{changeType: change.ChangeType{ID: "ct-2", Key: change.KeyLayout, Name: "Layout only", ApprovalRouteID: "r-2", ApprovalRouteName: "Short"}}
	srv := newTestServer(t, nil, Deps{Customers: customers, Changes: changes})

	var got Customer
	rec := srv.do(http.MethodPut, "/api/v1/customers/c-1/default-route", `{"route_id":"r-1"}`)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil || customers.gotID != "c-1" || customers.gotRoute != "r-1" ||
		got.DefaultRoute == nil || *got.DefaultRoute != (ApprovalRouteRef{Id: "r-1", Name: "Up to GM"}) {
		t.Errorf("rute default customer: status %d: %s", rec.Code, rec.Body.String())
	}

	var ct ChangeType
	rec = srv.do(http.MethodPut, "/api/v1/change-types/ct-2/approval-route", `{"route_id":"r-2"}`)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &ct) != nil ||
		changes.got != (called{method: "SetApprovalRoute", id: "ct-2", input: "r-2"}) ||
		ct.ApprovalRoute == nil || ct.ApprovalRoute.Name != "Short" {
		t.Errorf("rute jenis perubahan: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/change-types/ct-2/approval-route", `{}`)
	if rec.Code != http.StatusOK || changes.got.input != "" {
		t.Errorf("kembali ke rute default: status %d, %+v", rec.Code, changes.got)
	}
}

// TestSystemSettingsEndpoints: saklar tanggal tanda tangan (D-52).
func TestSystemSettingsEndpoints(t *testing.T) {
	fake := &fakeSettings{current: settings.Settings{SignDatesEnabled: false}}
	srv := newTestServer(t, nil, Deps{Settings: fake})

	rec := srv.do(http.MethodGet, "/api/v1/system-settings", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"sign_dates_enabled":false}`+"\n" {
		t.Errorf("baca: status %d: %s", rec.Code, rec.Body.String())
	}
	rec = srv.do(http.MethodPut, "/api/v1/system-settings", `{"sign_dates_enabled":true}`)
	if rec.Code != http.StatusOK || fake.got == nil || !*fake.got {
		t.Errorf("ubah: status %d, %v", rec.Code, fake.got)
	}
	// Saklar yang tidak dikirim diteruskan sebagai nil: tetap seperti sebelumnya.
	rec = srv.do(http.MethodPut, "/api/v1/system-settings", `{}`)
	if rec.Code != http.StatusOK || fake.got != nil || rec.Body.String() != `{"sign_dates_enabled":true}`+"\n" {
		t.Errorf("tanpa saklar: status %d, %v: %s", rec.Code, fake.got, rec.Body.String())
	}
}
