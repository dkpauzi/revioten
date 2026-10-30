package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"go.yaml.in/yaml/v3"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/settings"
)

// TestMain menyalakan PostgreSQL untuk test integrasi di file ini. Test lain
// di paket ini memakai service palsu dan tetap berjalan dengan -short.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// operationCase adalah satu operasi kontrak beserta status yang diharapkan
// untuk akun Viewer (tanpa peran), akun PPAP Engineer (Penyusun, Pengelola
// template, dan Dokumen kontrol, seperti di U3 bagian 2), dan akun admin.
type operationCase struct {
	method, path, body      string
	viewer, engineer, admin int
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
	processes := process.NewService(pool, checker)
	numbers := numbering.NewService(pool, checker)
	changes := change.NewService(pool, checker)
	approvals := approval.NewService(pool, checker)
	handler := NewHandler(fstest.MapFS{}, Deps{
		DB: pool, Customers: customers, Auth: authSvc, Access: checker,
		Audit: audit.NewService(pool, checker), Products: products, Processes: processes,
		Numbering: numbers, Changes: changes, Approval: approvals, Settings: settings.NewService(pool, checker),
	})

	// Admin pertama, satu akun kerja tanpa peran (hanya hak dasar Viewer), dan
	// satu akun kerja PPAP Engineer dengan peran Pengelola template (D-64).
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
	_, temporary, err = authSvc.CreateAccount(ctx, admin, auth.NewAccount{
		PersonName: "PPAP Engineer Uji", Kind: auth.KindWork, Login: "ppap.engineer.uji",
		Roles: []access.RoleGrant{
			{Role: access.RoleAuthor, Scope: access.ScopeAll},
			{Role: access.RoleTemplateMaintainer, Scope: access.ScopeAll},
			{Role: access.RoleDocumentControl, Scope: access.ScopeAll},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	engineer, _, err := authSvc.Login(ctx, "ppap.engineer.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	engineerToken, err := authSvc.ChangePassword(ctx, engineer, temporary, "perahu kertas di sungai jernih")
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
	// Master proses (M1.5).
	proc, err := processes.Create(ctx, engineer, process.Input{Name: "IQC"})
	if err != nil {
		t.Fatal(err)
	}
	// Isi awal master M1.6.
	docTypes, err := numbers.ListDocumentTypes(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	specialCodes, err := numbers.ListSpecialCodes(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	changeTypes, err := changes.ListChangeTypes(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	sources, err := changes.ListChangeSources(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	categories, err := changes.ListFourMCategories(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	// Isi awal M1.8, dan satu pemegang jabatan untuk operasi lepas pemegang.
	positions, err := approvals.ListPositions(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	routes, err := approvals.ListRoutes(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	// Operasi pemegang memakai PPAP Engineer, yang boleh punya beberapa
	// pemegang; NPI Manager hanya satu (E-38, D-115).
	var npi, ppapEngineer approval.Position
	for _, p := range positions {
		switch p.Key {
		case approval.KeyNPIManager:
			npi = p
		case approval.KeyPPAPEngineer:
			ppapEngineer = p
		}
	}
	holder, err := approvals.AddHolder(ctx, admin, ppapEngineer.ID, approval.NewHolder{PersonID: admin.PersonID})
	if err != nil {
		t.Fatal(err)
	}
	routeBody := `{"name":"Route X","slots":[` +
		`{"stage":"prepared","print_title":"Process. Eng","position_ids":["` + positions[0].ID + `"]},` +
		`{"stage":"checked","print_title":"Engineer","position_ids":["` + positions[2].ID + `"]},` +
		`{"stage":"approved","print_title":"NPI Manager","position_ids":["` + npi.ID + `"]}]}`
	// Satu nomor terdaftar untuk operasi status (Tahap 2).
	reg, err := numbers.Register(ctx, engineer, numbering.NewNumber{CustomerID: cust.ID, DocumentTypeID: docTypes[2].ID, IdempotencyKey: "setup"})
	if err != nil {
		t.Fatal(err)
	}
	// params mengisi {nama} di alamat dan isi permintaan dengan ID sungguhan.
	params := map[string]string{
		"positionId": ppapEngineer.ID, "holderId": holder.ID, "approvalRouteId": routes[0].ID, "routeId": routes[1].ID,
		"viewerPersonId": viewerInfo.PersonID,
		"accountId":      viewerInfo.ID, "projectGroupId": group.ID, "customerId": cust.ID,
		"modelGroupId": modelGroup.ID, "modelId": model.ID, "partNumberId": part.ID,
		"processId": proc.ID, "documentTypeId": docTypes[0].ID, "cpTypeId": docTypes[2].ID, "documentNumberId": reg.Numbers[0].ID, "specialCodeId": specialCodes[0].ID,
		"changeTypeId": changeTypes[0].ID, "changeSourceId": sources[0].ID, "fourMCategoryId": categories[0].ID,
	}

	// Alamat ditulis seperti di kontrak. Urutan penting: operasi yang
	// mengakhiri sesi atau menonaktifkan akun ada di akhir.
	cases := []operationCase{
		{http.MethodGet, "/api/v1/session", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/account/password", `{"current_password":"salah","new_password":"x"}`, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity},
		{http.MethodGet, "/api/v1/customers", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/customers", `{"code":"AB","name":"Customer Baru","project_group_id":"{projectGroupId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/customers/{customerId}", `{"name":"Customer Ubah","project_group_id":"{projectGroupId}","active":true}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/customers/{customerId}/code", `{"code":"XZ"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Register nomor (Tahap 2): mendaftarkan lewat E-1 (Penyusun), melihat lewat L-2.
		{http.MethodGet, "/api/v1/document-numbers/next?customer_id={customerId}&document_type_id={cpTypeId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/document-numbers", `{"customer_id":"{customerId}","document_type_id":"{cpTypeId}"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/document-numbers?customer_id={customerId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		// Status nomor diubah dokumen kontrol (E-8). Nomor yang dibuat setup ini Reserved.
		{http.MethodPost, "/api/v1/document-numbers/{documentNumberId}/void", `{"reason":"Uji izin"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/document-numbers/{documentNumberId}/obsolete", `{}`, http.StatusForbidden, http.StatusConflict, http.StatusForbidden},
		{http.MethodGet, "/api/v1/project-groups", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/project-groups", `{"code":"GR2","name":"Project Group 2"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/project-groups/{projectGroupId}", `{"code":"GR1","name":"Satu"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/model-groups?customer_id={customerId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/model-groups", `{"customer_id":"{customerId}","name":"Family 2"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/model-groups/{modelGroupId}", `{"name":"Family B"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/models?customer_id={customerId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/models", `{"customer_id":"{customerId}","name":"X200"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/models/{modelId}", `{"name":"X100A","model_group_id":"{modelGroupId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/part-numbers?customer_id={customerId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/part-numbers", `{"model_id":"{modelId}","number":"PN-2"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/part-numbers/{partNumberId}", `{"number":"PN-1A"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Master proses dikelola Pengelola template (E-6, D-64), bukan admin.
		{http.MethodGet, "/api/v1/processes", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/processes", `{"name":"SMT","aliases":["Surface Mount"]}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/processes/{processId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/processes/{processId}", `{"name":"Incoming QC","aliases":["IQC"]}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Jenis dokumen dan perubahan diubah Admin (A-2, UC-05).
		{http.MethodGet, "/api/v1/number-settings", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/number-settings", `{"factory_code":"T"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/document-types", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/document-types", `{"name":"Work Instruction","letter":"W"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/document-types/{documentTypeId}", `{"name":"Process Flow Diagram","letter":"H"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/special-codes", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/special-codes", `{"code":"QA","name":"Quality"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/special-codes/{specialCodeId}", `{"name":"Health and Safety"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/change-types", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/change-types/{changeTypeId}", `{"name":"Content change"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/change-sources", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/change-sources", `{"name":"Kaizen"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/change-sources/{changeSourceId}", `{"name":"Internal audit finding"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/four-m-categories", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/four-m-categories/{fourMCategoryId}", `{"name":"Manpower"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Jabatan lewat A-1 (UC-02), rute lewat A-2 (UC-06), pengaturan lewat A-3 (UC-07).
		{http.MethodGet, "/api/v1/positions", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/positions/{positionId}", `{"name":"PPAP Engineer Senior"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/positions/{positionId}/holders", `{"person_id":"{viewerPersonId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodDelete, "/api/v1/positions/{positionId}/holders/{holderId}", "", http.StatusForbidden, http.StatusForbidden, http.StatusNoContent},
		{http.MethodGet, "/api/v1/approval-routes", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/approval-routes", routeBody, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/approval-routes/{approvalRouteId}", strings.Replace(routeBody, "Route X", "Route Y", 1), http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPut, "/api/v1/customers/{customerId}/default-route", `{"route_id":"{routeId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPut, "/api/v1/change-types/{changeTypeId}/approval-route", `{"route_id":"{routeId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/system-settings", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/system-settings", `{"sign_dates_enabled":true}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/roles", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/accounts", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts", `{"person_name":"Orang Baru","kind":"work","login":"orang.baru"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/accounts/{accountId}/roles", `{"roles":[{"role":"author","scope":"all"}]}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/password-reset", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/disable", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/accounts/{accountId}/enable", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Audit log hanya lewat akun admin (D-72, D-75).
		{http.MethodGet, "/api/v1/audit-log", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/audit-log/verification", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodDelete, "/api/v1/session", "", http.StatusNoContent, http.StatusNoContent, http.StatusNoContent},
	}
	checkCoversContract(t, cases)

	fill := func(text string) string {
		for name, id := range params {
			text = strings.ReplaceAll(text, "{"+name+"}", id)
		}
		return text
	}
	run := func(who, token string, want func(operationCase) int) {
		for i, tc := range cases {
			req := httptest.NewRequest(tc.method, fill(tc.path), strings.NewReader(fill(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			// Kunci idempoten baru untuk setiap permintaan (pendaftaran nomor, R-NUM-03).
			req.Header.Set("Idempotency-Key", fmt.Sprintf("%s-%d", who, i))
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != want(tc) {
				t.Errorf("%s: %s %s status %d, ingin %d: %s", who, tc.method, tc.path, rec.Code, want(tc), rec.Body.String())
			}
		}
	}
	// Admin terakhir: operasinya ikut mereset dan menonaktifkan akun viewer.
	run("viewer", viewerToken, func(c operationCase) int { return c.viewer })
	run("engineer", engineerToken, func(c operationCase) int { return c.engineer })
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
