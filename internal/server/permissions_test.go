package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"go.yaml.in/yaml/v3"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestMain menyalakan PostgreSQL untuk test integrasi di file ini. Test lain
// di paket ini memakai service palsu dan tetap berjalan dengan -short.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// operationCase adalah satu operasi kontrak beserta status yang diharapkan
// untuk akun Viewer (tanpa peran) dan untuk akun admin.
type operationCase struct {
	method, path, body string
	viewer, admin      int
}

// TestPermissionsPerOperation membuktikan kriteria selesai M1.2 dengan service
// sungguhan dan database: setiap operasi API diperiksa izinnya, dan setiap
// peran hanya bisa menjalankan aksi yang diizinkan. Tabelnya wajib mencakup
// semua operasi di api/openapi.yaml, jadi endpoint baru tidak bisa terlewat.
func TestPermissionsPerOperation(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	checker := access.NewChecker(pool)
	authSvc := auth.NewService(pool, checker)
	handler := NewHandler(fstest.MapFS{}, Deps{
		DB: pool, Customers: customer.NewService(pool, checker), Auth: authSvc, Access: checker,
		Audit: audit.NewService(pool, checker),
	})

	// Admin pertama, dan satu akun kerja tanpa peran (hanya hak dasar Viewer).
	if _, err := authSvc.CreateFirstAdmin(ctx, "Admin Uji", "admin.uji", "kuda terbang di atas sawah"); err != nil {
		t.Fatal(err)
	}
	admin, adminToken, err := authSvc.Login(ctx, "admin.uji", "kuda terbang di atas sawah")
	if err != nil {
		t.Fatal(err)
	}
	viewerInfo, temporary, err := authSvc.CreateAccount(ctx, admin, auth.NewAccount{PersonName: "Viewer Uji", Kind: auth.KindWork, Login: "viewer.uji"})
	if err != nil {
		t.Fatal(err)
	}
	viewer, _, err := authSvc.Login(ctx, "viewer.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	viewerToken, err := authSvc.ChangePassword(ctx, viewer, temporary, "burung hantu di pohon mangga")
	if err != nil {
		t.Fatal(err)
	}

	target := "/api/v1/accounts/" + viewerInfo.ID
	// Urutan penting: operasi yang mengakhiri sesi atau menonaktifkan akun ada di akhir.
	cases := []operationCase{
		{http.MethodGet, "/api/v1/session", "", http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/account/password", `{"current_password":"salah","new_password":"x"}`, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity},
		{http.MethodGet, "/api/v1/customers", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/customers", `{"code":"AB","name":"Customer Uji"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodGet, "/api/v1/roles", "", http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/accounts", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts", `{"person_name":"Orang Baru","kind":"work","login":"orang.baru"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "{target}/roles", `{"roles":[{"role":"author","scope":"all"}]}`, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "{target}/password-reset", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "{target}/disable", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "{target}/enable", "", http.StatusForbidden, http.StatusOK},
		// Audit log hanya lewat akun admin (D-72, D-75).
		{http.MethodGet, "/api/v1/audit-log", "", http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/audit-log/verification", "", http.StatusForbidden, http.StatusOK},
		{http.MethodDelete, "/api/v1/session", "", http.StatusNoContent, http.StatusNoContent},
	}
	checkCoversContract(t, cases)

	run := func(who, token string, want func(operationCase) int) {
		for _, tc := range cases {
			path := strings.Replace(tc.path, "{target}", target, 1)
			req := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != want(tc) {
				t.Errorf("%s: %s %s status %d, ingin %d: %s", who, tc.method, tc.path, rec.Code, want(tc), rec.Body.String())
			}
		}
	}
	// Viewer dulu: operasi admin di bawah ikut mereset dan menonaktifkan akun viewer.
	run("viewer", viewerToken, func(c operationCase) int { return c.viewer })
	run("admin", adminToken, func(c operationCase) int { return c.admin })
}

// checkCoversContract memastikan tabel test mencakup semua operasi kontrak
// yang tidak publik.
func checkCoversContract(t *testing.T, cases []operationCase) {
	t.Helper()
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
		t.Fatal(err)
	}
	covered := map[string]bool{}
	for _, c := range cases {
		path := strings.Replace(c.path, "{target}", "/api/v1/accounts/{accountId}", 1)
		covered[c.method+" "+path] = true
	}
	for path, operations := range contract.Paths {
		for method, op := range operations {
			public := op.Security != nil && len(*op.Security) == 0
			route := strings.ToUpper(method) + " " + path
			if !public && !covered[route] {
				t.Errorf("operasi %s belum ada di tabel TestPermissionsPerOperation", route)
			}
		}
	}
}
