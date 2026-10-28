package server

import (
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// TestAPIRequiresSession: semua operasi API butuh sesi, kecuali yang publik.
func TestAPIRequiresSession(t *testing.T) {
	protected := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/customers", ""},
		{http.MethodPost, "/api/v1/customers", `{"code":"AB","name":"X"}`},
		{http.MethodGet, "/api/v1/session", ""},
		{http.MethodDelete, "/api/v1/session", ""},
		{http.MethodPut, "/api/v1/account/password", `{"current_password":"a","new_password":"b"}`},
		{http.MethodGet, "/api/v1/roles", ""},
		{http.MethodGet, "/api/v1/accounts", ""},
		{http.MethodPost, "/api/v1/accounts", `{"kind":"work","login":"x.uji"}`},
		{http.MethodPut, "/api/v1/accounts/akun-1/roles", `{"roles":[]}`},
		{http.MethodPost, "/api/v1/accounts/akun-1/password-reset", ""},
		{http.MethodPost, "/api/v1/accounts/akun-1/disable", ""},
		{http.MethodPost, "/api/v1/accounts/akun-1/enable", ""},
		{http.MethodGet, "/api/v1/audit-log", ""},
		{http.MethodGet, "/api/v1/audit-log/verification", ""},
		{http.MethodPut, "/api/v1/customers/c-1", `{"name":"X","project_group_id":"g","active":true}`},
		{http.MethodPost, "/api/v1/customers/c-1/code", `{"code":"AB"}`},
		{http.MethodGet, "/api/v1/project-groups", ""},
		{http.MethodPost, "/api/v1/project-groups", `{"code":"GR1","name":"X"}`},
		{http.MethodPut, "/api/v1/project-groups/g-1", `{"code":"GR1","name":"X"}`},
		{http.MethodGet, "/api/v1/model-groups?customer_id=c-1", ""},
		{http.MethodPost, "/api/v1/model-groups", `{"customer_id":"c-1","name":"X"}`},
		{http.MethodPut, "/api/v1/model-groups/mg-1", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/models?customer_id=c-1", ""},
		{http.MethodPost, "/api/v1/models", `{"customer_id":"c-1","name":"X"}`},
		{http.MethodPut, "/api/v1/models/m-1", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/part-numbers?customer_id=c-1", ""},
		{http.MethodPost, "/api/v1/part-numbers", `{"model_id":"m-1","number":"X"}`},
		{http.MethodPut, "/api/v1/part-numbers/p-1", `{"number":"X"}`},
		{http.MethodGet, "/api/v1/processes", ""},
		{http.MethodPost, "/api/v1/processes", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/processes/pr-1", ""},
		{http.MethodPut, "/api/v1/processes/pr-1", `{"name":"X","aliases":[]}`},
		{http.MethodGet, "/api/v1/number-settings", ""},
		{http.MethodPut, "/api/v1/number-settings", `{"factory_code":"S"}`},
		{http.MethodGet, "/api/v1/document-types", ""},
		{http.MethodPost, "/api/v1/document-types", `{"name":"X","letter":"X"}`},
		{http.MethodPut, "/api/v1/document-types/dt-1", `{"name":"X","letter":"X"}`},
		{http.MethodGet, "/api/v1/special-codes", ""},
		{http.MethodPost, "/api/v1/special-codes", `{"code":"XX","name":"X"}`},
		{http.MethodPut, "/api/v1/special-codes/sc-1", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/change-types", ""},
		{http.MethodPut, "/api/v1/change-types/ct-1", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/change-sources", ""},
		{http.MethodPost, "/api/v1/change-sources", `{"name":"X"}`},
		{http.MethodPut, "/api/v1/change-sources/cs-1", `{"name":"X"}`},
		{http.MethodGet, "/api/v1/four-m-categories", ""},
		{http.MethodPut, "/api/v1/four-m-categories/fm-1", `{"name":"X"}`},
	}
	srv := newTestServer(t, nil, Deps{})
	for _, tt := range protected {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := srv.doAnon(tt.method, tt.path, tt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status %d, ingin 401", rec.Code)
			}
			if body := decodeError(t, rec); body.Code != ErrorCodeUnauthenticated {
				t.Errorf("code %q, ingin unauthenticated", body.Code)
			}
		})
	}

	// Cookie dengan token yang tidak berlaku lagi ditolak, dan cookie-nya dihapus.
	rec := srv.doWithCookie("token-kedaluwarsa", http.MethodGet, "/api/v1/customers", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("token tidak berlaku: status %d, ingin 401", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("cookie lama tidak dihapus: %q", rec.Header().Get("Set-Cookie"))
	}
}

// TestPublicRoutesMatchContract: daftar route publik di kode harus sama dengan
// operasi bertanda `security: []` di kontrak, supaya keduanya tidak berbeda.
func TestPublicRoutesMatchContract(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Paths map[string]map[string]struct {
			Security *[]map[string][]string `yaml:"security"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &contract); err != nil {
		t.Fatalf("membaca kontrak: %v", err)
	}

	var inContract []string
	for path, operations := range contract.Paths {
		for method, op := range operations {
			if op.Security != nil && len(*op.Security) == 0 {
				inContract = append(inContract, strings.ToUpper(method)+" "+path)
			}
		}
	}
	var inCode []string
	for route := range publicRoutes {
		inCode = append(inCode, route)
	}
	slices.Sort(inContract)
	slices.Sort(inCode)
	if !slices.Equal(inContract, inCode) {
		t.Errorf("route publik di kontrak %v, di kode %v", inContract, inCode)
	}
}

func TestLogin(t *testing.T) {
	fake := newFakeAuth()
	fake.loginActor = testActor
	fake.loginToken = "token-dari-login"
	srv := newTestServer(t, nil, Deps{Auth: fake})

	rec := srv.doAnon(http.MethodPost, "/api/v1/session", `{"login":"admin.uji","password":"password-benar"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, ingin 200: %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Header().Get("Set-Cookie")
	// Atribut cookie sesi: ASVS 3.3.1, 3.3.2, 3.3.3, 3.3.4.
	for _, want := range []string{"__Host-revioten_session=token-dari-login", "Path=/", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("Set-Cookie %q tidak memuat %q", cookie, want)
		}
	}
	var body Session
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	if body.Account.Login != "admin.uji" || body.Account.Kind != AccountKindAdmin || body.Person.Name != "Admin Uji" {
		t.Errorf("isi sesi %+v", body)
	}
}

// TestLoginFailed: login gagal dijawab 401 dengan pesan umum dan tanpa cookie.
// Log server hanya mencatat request ID. Nama login yang diketik tidak dicatat,
// karena pengguna kadang mengetik password di kolom nama login (E-18).
// Percobaan terhadap akun yang ada tercatat di audit log dengan ID akunnya.
func TestLoginFailed(t *testing.T) {
	srv := newTestServer(t, nil, Deps{})

	rec := srv.doAnon(http.MethodPost, "/api/v1/session", `{"login":"admin.uji","password":"rahasia-salah-sekali"}`,
		"X-Request-Id", "uji-login")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, ingin 401", rec.Code)
	}
	if body := decodeError(t, rec); body.Code != ErrorCodeInvalidCredentials {
		t.Errorf("code %q, ingin invalid_credentials", body.Code)
	}
	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("login gagal tetap memasang cookie %q", got)
	}
	log := srv.log.String()
	if !strings.Contains(log, "login gagal") || !strings.Contains(log, "uji-login") {
		t.Errorf("log harus memuat peristiwa dan request ID:\n%s", log)
	}
	if strings.Contains(log, "admin.uji") || strings.Contains(log, "rahasia-salah-sekali") {
		t.Errorf("nama login atau password masuk log:\n%s", log)
	}
}

func TestGetSession(t *testing.T) {
	rec := newTestServer(t, nil, Deps{}).do(http.MethodGet, "/api/v1/session", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, ingin 200: %s", rec.Code, rec.Body.String())
	}
	var body Session
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	if body.Account != (Account{Id: "akun-uji", Login: "admin.uji", Kind: AccountKindAdmin}) ||
		body.Person != (Person{Id: "orang-uji", Name: "Admin Uji"}) || body.MustChangePassword {
		t.Errorf("isi %+v", body)
	}
	// Daftar hak dari Permissions, untuk menampilkan atau menyembunyikan menu.
	if !slices.Equal(body.Permissions, []string{"L-1", "A-1"}) {
		t.Errorf("permissions %v, ingin [L-1 A-1]", body.Permissions)
	}
}

func TestLogout(t *testing.T) {
	fake := newFakeAuth()
	rec := newTestServer(t, nil, Deps{Auth: fake}).do(http.MethodDelete, "/api/v1/session", "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d, ingin 204", rec.Code)
	}
	if !slices.Equal(fake.loggedOut, []string{testToken}) {
		t.Errorf("Logout dipanggil dengan %v, ingin [%s]", fake.loggedOut, testToken)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("cookie tidak dihapus: %q", rec.Header().Get("Set-Cookie"))
	}
}

// TestMustChangePassword: selama wajib ganti password, hanya sesi, logout,
// dan ganti password yang boleh dipakai (E-08).
func TestMustChangePassword(t *testing.T) {
	fake := newFakeAuth()
	actor := testActor
	actor.MustChangePassword = true
	fake.sessions[testToken] = actor
	srv := newTestServer(t, nil, Deps{Auth: fake})

	rec := srv.do(http.MethodGet, "/api/v1/customers", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, ingin 403", rec.Code)
	}
	if body := decodeError(t, rec); body.Code != ErrorCodePasswordChangeRequired {
		t.Errorf("code %q, ingin password_change_required", body.Code)
	}

	if rec := srv.do(http.MethodGet, "/api/v1/session", ""); rec.Code != http.StatusOK {
		t.Errorf("GET sesi: status %d, ingin 200", rec.Code)
	}
	body := `{"current_password":"lama-lama-lama-lama","new_password":"baru-baru-baru-baru"}`
	if rec := srv.do(http.MethodPut, "/api/v1/account/password", body); rec.Code != http.StatusNoContent {
		t.Errorf("ganti password: status %d, ingin 204", rec.Code)
	}
}

func TestChangePassword(t *testing.T) {
	fake := newFakeAuth()
	srv := newTestServer(t, nil, Deps{Auth: fake})

	rec := srv.do(http.MethodPut, "/api/v1/account/password",
		`{"current_password":"password lama yang benar","new_password":"password baru yang panjang"}`)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d, ingin 204: %s", rec.Code, rec.Body.String())
	}
	if fake.changeCurrent != "password lama yang benar" || fake.changeNew != "password baru yang panjang" {
		t.Errorf("ChangePassword menerima %q dan %q", fake.changeCurrent, fake.changeNew)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "__Host-revioten_session=token-baru") {
		t.Errorf("cookie sesi baru tidak dipasang: %q", rec.Header().Get("Set-Cookie"))
	}

	fake.changeErr = &problem.ValidationError{Problems: []problem.Problem{
		{Field: "current_password", Code: problem.Incorrect, Message: "The current password is not correct."},
	}}
	rec = srv.do(http.MethodPut, "/api/v1/account/password", `{"current_password":"salah","new_password":"x"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, ingin 422", rec.Code)
	}
	body := decodeError(t, rec)
	want := []ErrorDetail{{Field: "current_password", Code: ErrorDetailCodeIncorrect, Message: "The current password is not correct."}}
	if body.Code != ErrorCodeValidationFailed || body.Details == nil || !slices.Equal(*body.Details, want) {
		t.Errorf("isi error %+v", body)
	}
}

// TestCrossOriginRejected: permintaan yang mengubah data dari situs lain
// ditolak (cek Origin, brief bagian 16), termasuk login.
func TestCrossOriginRejected(t *testing.T) {
	srv := newTestServer(t, nil, Deps{})

	tests := []struct {
		name   string
		header []string
		want   int
	}{
		{"dari situs lain menurut Sec-Fetch-Site", []string{"Sec-Fetch-Site", "cross-site"}, http.StatusForbidden},
		{"dari situs lain menurut Origin", []string{"Origin", "http://situs-lain.example"}, http.StatusForbidden},
		{"dari situs sendiri", []string{"Sec-Fetch-Site", "same-origin"}, http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := srv.do(http.MethodPost, "/api/v1/customers", `{"code":"AB","name":"X"}`, tt.header...)
			if rec.Code != tt.want {
				t.Fatalf("status %d, ingin %d", rec.Code, tt.want)
			}
			if tt.want == http.StatusForbidden {
				if body := decodeError(t, rec); body.Code != ErrorCodeForbidden {
					t.Errorf("code %q, ingin forbidden", body.Code)
				}
			}
		})
	}

	rec := srv.doAnon(http.MethodPost, "/api/v1/session", `{"login":"a","password":"b"}`, "Sec-Fetch-Site", "cross-site")
	if rec.Code != http.StatusForbidden {
		t.Errorf("login dari situs lain: status %d, ingin 403", rec.Code)
	}
}

// TestPageGate: halaman web tanpa sesi diarahkan ke /login, dengan alamat asal
// di parameter next. Halaman login dan aset tetap terbuka.
func TestPageGate(t *testing.T) {
	srv := newTestServer(t, testUI(), Deps{})

	tests := []struct {
		name         string
		path         string
		loggedIn     bool
		wantStatus   int
		wantLocation string
	}{
		{"halaman tanpa sesi", "/customers?x=1", false, http.StatusSeeOther, "/login?next=%2Fcustomers%3Fx%3D1"},
		{"beranda tanpa sesi", "/", false, http.StatusSeeOther, "/login?next=%2F"},
		{"halaman login tanpa sesi", "/login", false, http.StatusOK, ""},
		{"aset tanpa sesi", "/_app/immutable/app.js", false, http.StatusOK, ""},
		{"halaman dengan sesi", "/customers", true, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := ""
			if tt.loggedIn {
				token = testToken
			}
			rec := srv.doWithCookie(token, http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d, ingin %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location %q, ingin %q", got, tt.wantLocation)
			}
		})
	}
}

// TestSessionActorInContext: handler API menerima actor dari sesi.
func TestSessionActorInContext(t *testing.T) {
	fake := newFakeAuth()
	other := access.Actor{Type: access.ActorHuman, AccountID: "akun-lain", AccountKind: access.AccountWork, Login: "penyusun.uji", PersonID: "orang-lain", PersonName: "Penyusun Uji"}
	fake.sessions["token-lain"] = other
	rec := newTestServer(t, nil, Deps{Auth: fake}).doWithCookie("token-lain", http.MethodGet, "/api/v1/session", "")

	var body Session
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi bukan JSON yang valid: %v", err)
	}
	if body.Account.Login != "penyusun.uji" || body.Account.Kind != AccountKindWork {
		t.Errorf("sesi %+v, ingin milik penyusun.uji", body)
	}
}
