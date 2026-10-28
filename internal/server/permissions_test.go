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
	"github.com/dkpauzi/revioten/internal/product"
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
	customers := customer.NewService(pool, checker)
	products := product.NewService(pool, checker)
	handler := NewHandler(fstest.MapFS{}, Deps{
		DB: pool, Customers: customers, Auth: authSvc, Access: checker,
		Audit: audit.NewService(pool, checker), Products: products,
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

	// Master data yang dirujuk operasi di bawah (M1.4).
	group, err := customers.CreateProjectGroup(ctx, admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	cust, err := customers.Create(ctx, admin, customer.NewCustomer{Code: "XY", Name: "Customer Uji", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	modelGroup, err := products.CreateModelGroup(ctx, admin, product.NewModelGroup{CustomerID: cust.ID, Name: "Family"})
	if err != nil {
		t.Fatal(err)
	}
	model, err := products.CreateModel(ctx, admin, product.NewModel{CustomerID: cust.ID, Name: "X100"})
	if err != nil {
		t.Fatal(err)
	}
	part, err := products.CreatePartNumber(ctx, admin, product.NewPartNumber{ModelID: model.ID, Number: "PN-1"})
	if err != nil {
		t.Fatal(err)
	}
	// params mengisi {nama} di alamat dan isi permintaan dengan ID sungguhan.
	params := map[string]string{
		"accountId": viewerInfo.ID, "projectGroupId": group.ID, "customerId": cust.ID,
		"modelGroupId": modelGroup.ID, "modelId": model.ID, "partNumberId": part.ID,
	}

	// Alamat ditulis seperti di kontrak. Urutan penting: operasi yang
	// mengakhiri sesi atau menonaktifkan akun ada di akhir.
	cases := []operationCase{
		{http.MethodGet, "/api/v1/session", "", http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/account/password", `{"current_password":"salah","new_password":"x"}`, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity},
		{http.MethodGet, "/api/v1/customers", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/customers", `{"code":"AB","name":"Customer Baru","project_group_id":"{projectGroupId}"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/customers/{customerId}", `{"name":"Customer Ubah","project_group_id":"{projectGroupId}","active":true}`, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/customers/{customerId}/code", `{"code":"XZ"}`, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/project-groups", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/project-groups", `{"code":"GR2","name":"Project Group 2"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/project-groups/{projectGroupId}", `{"code":"GR1","name":"Satu"}`, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/model-groups?customer_id={customerId}", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/model-groups", `{"customer_id":"{customerId}","name":"Family 2"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/model-groups/{modelGroupId}", `{"name":"Family B"}`, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/models?customer_id={customerId}", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/models", `{"customer_id":"{customerId}","name":"X200"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/models/{modelId}", `{"name":"X100A","model_group_id":"{modelGroupId}"}`, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/part-numbers?customer_id={customerId}", "", http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/part-numbers", `{"model_id":"{modelId}","number":"PN-2"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/part-numbers/{partNumberId}", `{"number":"PN-1A"}`, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/roles", "", http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/accounts", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts", `{"person_name":"Orang Baru","kind":"work","login":"orang.baru"}`, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/accounts/{accountId}/roles", `{"roles":[{"role":"author","scope":"all"}]}`, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/password-reset", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/disable", "", http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/enable", "", http.StatusForbidden, http.StatusOK},
		// Audit log hanya lewat akun admin (D-72, D-75).
		{http.MethodGet, "/api/v1/audit-log", "", http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/audit-log/verification", "", http.StatusForbidden, http.StatusOK},
		{http.MethodDelete, "/api/v1/session", "", http.StatusNoContent, http.StatusNoContent},
	}
	checkCoversContract(t, cases)

	fill := func(text string) string {
		for name, id := range params {
			text = strings.ReplaceAll(text, "{"+name+"}", id)
		}
		return text
	}
	run := func(who, token string, want func(operationCase) int) {
		for _, tc := range cases {
			req := httptest.NewRequest(tc.method, fill(tc.path), strings.NewReader(fill(tc.body)))
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
		// Parameter query bukan bagian dari route di kontrak.
		path, _, _ := strings.Cut(c.path, "?")
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
