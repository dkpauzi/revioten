package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/problem"
)

// sampleAccount adalah akun contoh yang dikembalikan fakeAuth.
var sampleAccount = auth.AccountInfo{
	ID: "akun-1", Kind: access.AccountWork, Login: "penyusun.uji",
	PersonID: "orang-1", PersonName: "Penyusun Uji", EmployeeNumber: "K-001",
	Roles:              []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}},
	MustChangePassword: true,
	CreatedAt:          waktuUji,
}

func TestListRolesEndpoint(t *testing.T) {
	fake := &fakeAccess{roles: []access.Role{
		{ID: access.RoleAuthor, Level: "edit", Permissions: []access.Permission{access.ViewReleased, access.EditDrafts}},
	}}
	rec := newTestServer(t, nil, Deps{Access: fake}).do(http.MethodGet, "/api/v1/roles", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body RoleList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Id != "author" || body.Items[0].Level != RoleLevelEdit ||
		!slices.Equal(body.Items[0].Permissions, []string{"L-1", "E-2"}) {
		t.Errorf("isi %+v", body)
	}
}

func TestListAccountsEndpoint(t *testing.T) {
	locked := waktuUji.Add(15 * time.Minute)
	other := sampleAccount
	other.LockedUntil = &locked
	other.EmployeeNumber = ""
	fake := newFakeAuth()
	fake.accounts = []auth.AccountInfo{sampleAccount, other}
	rec := newTestServer(t, nil, Deps{Auth: fake}).do(http.MethodGet, "/api/v1/accounts", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body AccountList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("%d akun, ingin 2", len(body.Items))
	}
	first := body.Items[0]
	if first.Id != "akun-1" || first.Kind != AccountInfoKindWork || first.Person.Name != "Penyusun Uji" ||
		first.Person.EmployeeNumber == nil || *first.Person.EmployeeNumber != "K-001" ||
		!first.MustChangePassword || first.Disabled || first.LockedUntil != nil ||
		!slices.Equal(first.Roles, []RoleGrant{{Role: "author", Scope: RoleGrantScopeOwnProjectGroup}}) {
		t.Errorf("akun pertama %+v", first)
	}
	if body.Items[1].Person.EmployeeNumber != nil || body.Items[1].LockedUntil == nil {
		t.Errorf("akun kedua %+v", body.Items[1])
	}
}

func TestCreateAccountEndpoint(t *testing.T) {
	fake := newFakeAuth()
	fake.account = sampleAccount
	fake.temporary = "SEMENTARA2026"
	rec := newTestServer(t, nil, Deps{Auth: fake}).do(http.MethodPost, "/api/v1/accounts",
		`{"person_name":"Penyusun Uji","employee_number":"K-001","kind":"work","login":"penyusun.uji",
		  "roles":[{"role":"author","scope":"own_project_group"}]}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	want := auth.NewAccount{
		PersonName: "Penyusun Uji", EmployeeNumber: "K-001", Kind: "work", Login: "penyusun.uji",
		Roles: []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}},
	}
	if fake.newAccount == nil || fake.newAccount.PersonName != want.PersonName || fake.newAccount.Kind != want.Kind ||
		fake.newAccount.Login != want.Login || fake.newAccount.EmployeeNumber != want.EmployeeNumber ||
		!slices.Equal(fake.newAccount.Roles, want.Roles) {
		t.Errorf("input ke service %+v, ingin %+v", fake.newAccount, want)
	}
	var body CreatedAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TemporaryPassword != "SEMENTARA2026" || body.Account.Login != "penyusun.uji" {
		t.Errorf("isi %+v", body)
	}
}

func TestSetAccountRolesEndpoint(t *testing.T) {
	fakeA := &fakeAccess{}
	fakeAu := newFakeAuth()
	fakeAu.account = sampleAccount
	rec := newTestServer(t, nil, Deps{Auth: fakeAu, Access: fakeA}).do(http.MethodPut, "/api/v1/accounts/akun-1/roles",
		`{"roles":[{"role":"verifier","scope":"all"}]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if fakeA.setAccount != "akun-1" || !slices.Equal(fakeA.setGrants, []access.RoleGrant{{Role: access.RoleVerifier, Scope: access.ScopeAll}}) {
		t.Errorf("SetRoles menerima %q %+v", fakeA.setAccount, fakeA.setGrants)
	}
	if fakeAu.targetAccount != "akun-1" {
		t.Errorf("akun yang dibaca ulang %q", fakeAu.targetAccount)
	}
}

func TestAccountActionsEndpoints(t *testing.T) {
	fake := newFakeAuth()
	fake.account = sampleAccount
	fake.temporary = "SEMENTARA2026"
	srv := newTestServer(t, nil, Deps{Auth: fake})

	rec := srv.do(http.MethodPost, "/api/v1/accounts/akun-1/password-reset", "")
	var reset TemporaryPassword
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &reset) != nil || reset.TemporaryPassword != "SEMENTARA2026" {
		t.Errorf("reset: status %d, isi %s", rec.Code, rec.Body.String())
	}
	for _, action := range []string{"disable", "enable"} {
		fake.targetAccount = ""
		rec := srv.do(http.MethodPost, "/api/v1/accounts/akun-1/"+action, "")
		if rec.Code != http.StatusOK || fake.targetAccount != "akun-1" {
			t.Errorf("%s: status %d, akun %q", action, rec.Code, fake.targetAccount)
		}
	}
}

// TestDomainErrors: error dari paket domain diterjemahkan ke satu format error
// API dengan status yang sesuai (R-API-13), tanpa detail teknis.
func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   ErrorCode
	}{
		{"tidak punya hak", access.ErrForbidden, http.StatusForbidden, ErrorCodeForbidden},
		{"tidak ditemukan", problem.ErrNotFound, http.StatusNotFound, ErrorCodeNotFound},
		{"input tidak valid", &problem.ValidationError{Problems: []problem.Problem{
			{Field: "account_id", Code: problem.NotAllowed, Message: "You cannot disable your own account."}}},
			http.StatusUnprocessableEntity, ErrorCodeValidationFailed},
		{"bentrok", &problem.ConflictError{Problems: []problem.Problem{
			{Field: "login", Code: problem.Taken, Message: "This login name is already used."}}},
			http.StatusConflict, ErrorCodeConflict},
		{"error lain", errors.New("koneksi putus"), http.StatusInternalServerError, ErrorCodeInternalError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeAuth()
			fake.adminErr = tt.err
			rec := newTestServer(t, nil, Deps{Auth: fake}).do(http.MethodPost, "/api/v1/accounts/akun-1/disable", "")

			if rec.Code != tt.wantStatus {
				t.Errorf("status %d, ingin %d", rec.Code, tt.wantStatus)
			}
			if body := decodeError(t, rec); body.Code != tt.wantCode {
				t.Errorf("code %q, ingin %q", body.Code, tt.wantCode)
			}
		})
	}
}
