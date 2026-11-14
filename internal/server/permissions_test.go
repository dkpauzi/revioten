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
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/settings"
	"github.com/dkpauzi/revioten/internal/storage"
	"github.com/dkpauzi/revioten/internal/template"
)

// TestMain menyalakan PostgreSQL untuk test integrasi di file ini. Test lain
// di paket ini memakai service palsu dan tetap berjalan dengan -short.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// operationCase adalah satu operasi kontrak beserta status yang diharapkan
// untuk akun Viewer (tanpa peran), akun PPAP Engineer (Penyusun, Pengelola
// template, dan Dokumen kontrol, seperti di U3 bagian 2), dan akun admin.
// multipartBody adalah unggahan satu berkas teks kecil (M5.2).
const multipartBody = "--revioten\r\nContent-Disposition: form-data; name=\"file\"; filename=\"uji.txt\"\r\nContent-Type: text/plain\r\n\r\nisi uji\r\n--revioten--\r\n"

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
	attachmentStore, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
		Templates: template.NewService(pool, checker), ProcessTemplates: template.NewProcessService(pool, checker),
		Documents: document.NewService(pool, checker), RefDocs: template.NewRefDocService(pool, checker),
		ChangeRequests: change.NewRequestService(pool, checker).WithStorage(attachmentStore),
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
			// Penyetuju form (V-3, D-139), untuk operasi menerbitkan form set.
			{Role: access.RoleFormApprover, Scope: access.ScopeAll},
			// Pemeriksa PFMEA-CP (V-2, D-65), untuk tanda dicek (M4.9).
			{Role: access.RolePFMEACPChecker, Scope: access.ScopeAll},
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
	// Format template isian awal (M3.1): PFMEA AIAG 4th, versi 1, field pertamanya.
	templates := template.NewService(pool, checker)
	formats, err := templates.ListFormatTemplates(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	pfmeaFormat := formats[1]
	pfmeaVersion, err := templates.GetFormatVersion(ctx, admin, pfmeaFormat.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	// Form set bawaan (langkah D): versi 1 set AIAG 4th masih draft.
	formSets, err := templates.ListFormSets(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	formSet := formSets.Sets[0]
	// Dokumen (M4.1) butuh form set yang sudah terbit: set AIAG-VDA diterbitkan
	// (format oleh admin, versi AP dan set oleh engineer sebagai penyetuju
	// form) dan menjadi form set customer. Set AIAG 4th tetap draft.
	vdaSet := formSets.Sets[1]
	vdaV := vdaSet.Versions[0]
	for _, id := range []string{vdaV.PFD.FormatVersionID, vdaV.CP.FormatVersionID} {
		if _, err := templates.PublishFormatVersion(ctx, admin, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := templates.PublishFormatVersion(ctx, engineer, vdaV.PFMEA.FormatVersionID); err != nil {
		t.Fatal(err)
	}
	if _, err := templates.PublishFormSetVersion(ctx, engineer, vdaV.ID); err != nil {
		t.Fatal(err)
	}
	if err := templates.SetCustomerFormSet(ctx, admin, cust.ID, vdaSet.ID); err != nil {
		t.Fatal(err)
	}
	docNumbers, err := numbers.RegisterSet(ctx, engineer, numbering.NewSet{CustomerID: cust.ID, IdempotencyKey: "setup-dokumen"})
	if err != nil {
		t.Fatal(err)
	}
	// Nomor Reserved dari set itu (nomor Skipped tidak bisa dipakai dokumen).
	var reserved []string
	for _, n := range docNumbers.Numbers {
		if n.Status == "reserved" {
			reserved = append(reserved, n.ID)
		}
	}
	documents := document.NewService(pool, checker)
	doc, err := documents.Create(ctx, engineer, document.NewDocument{NumberID: reserved[0], ModelIDs: []string{model.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if doc, err = documents.AddRow(ctx, engineer, doc.ID, document.NewRow{}); err != nil {
		t.Fatal(err)
	}
	// Process template IQC (M3.2) dengan satu sub-proses dan satu poin CP,
	// dibuat PPAP Engineer. Proses Reflow belum punya template, untuk operasi
	// membuat template.
	processTemplates := template.NewProcessService(pool, checker)
	iqcTemplate, err := processTemplates.CreateProcessTemplate(ctx, engineer, proc.ID)
	if err != nil {
		t.Fatal(err)
	}
	iqcVersion, err := processTemplates.AddSubprocess(ctx, engineer, iqcTemplate.Versions[0].ID, "IQC MSD")
	if err != nil {
		t.Fatal(err)
	}
	iqcVersion, err = processTemplates.AddPoint(ctx, engineer, iqcVersion.ID, template.NewPoint{
		SubprocessID: iqcVersion.Subprocesses[0].ID, DocumentTypeID: docTypes[2].ID, Text: "Check MSL label",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Satu varian customer dari poin itu (M3.3), untuk operasi mengubah varian.
	iqcVersion, err = processTemplates.AddVariant(ctx, engineer, iqcVersion.ID, iqcVersion.Subprocesses[0].Points[0].ID, template.NewVariant{
		Scope: template.ScopeInput{Level: template.ScopeCustomer, TargetID: cust.ID}, Text: "Check MSL label customer",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Satu dokumen terkait dengan satu tautan ke IQC (D-130), untuk operasi ubah dan lepas tautan.
	refDocs := template.NewRefDocService(pool, checker)
	refDoc, err := refDocs.Create(ctx, engineer, template.NewRefDocument{Number: "SW-ZZ-103", Title: "Material Receiving"})
	if err != nil {
		t.Fatal(err)
	}
	if refDoc, err = refDocs.AddLink(ctx, engineer, refDoc.ID, template.NewRefLink{ProcessID: proc.ID}); err != nil {
		t.Fatal(err)
	}
	// Satu perubahan untuk dokumen uji (M5.1), dengan pemicu Customer claim dan jenis isi.
	var triggerID, contentTypeID string
	if err := pool.QueryRow(ctx, "SELECT id FROM change_sources WHERE name = 'Customer claim'").Scan(&triggerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT id FROM change_types WHERE key = 'content'").Scan(&contentTypeID); err != nil {
		t.Fatal(err)
	}
	changeReq, err := change.NewRequestService(pool, checker).Create(ctx, engineer, change.NewChange{
		CustomerID: cust.ID, TriggerID: triggerID, ChangeTypeID: contentTypeID, Reason: "Uji izin", DocumentIDs: []string{doc.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	reflow, err := processes.Create(ctx, engineer, process.Input{Name: "Reflow"})
	if err != nil {
		t.Fatal(err)
	}
	// params mengisi {nama} di alamat dan isi permintaan dengan ID sungguhan.
	params := map[string]string{
		"positionId": ppapEngineer.ID, "holderId": holder.ID, "approvalRouteId": routes[0].ID, "routeId": routes[1].ID,
		"viewerPersonId": viewerInfo.PersonID,
		"accountId":      viewerInfo.ID, "projectGroupId": group.ID, "customerId": cust.ID,
		"modelGroupId": modelGroup.ID, "modelId": model.ID, "partNumberId": part.ID,
		"processId": proc.ID, "documentTypeId": docTypes[0].ID, "cpTypeId": docTypes[2].ID, "documentNumberId": reg.Numbers[0].ID, "documentId": doc.ID, "rowId": doc.Rows[0].ID, "freeNumberId": reserved[1], "specialCodeId": specialCodes[0].ID,
		"changeTypeId": changeTypes[0].ID, "changeSourceId": sources[0].ID, "fourMCategoryId": categories[0].ID,
		"formatTemplateId": pfmeaFormat.ID, "formatVersionId": pfmeaVersion.ID, "formSetId": formSet.ID, "formSetVersionId": formSet.Versions[0].ID, "pfdSheetId": formSet.Versions[0].PFD.FormatVersionID, "pfmeaSheetId": formSet.Versions[0].PFMEA.FormatVersionID, "cpSheetId": formSet.Versions[0].CP.FormatVersionID, "formatFieldId": pfmeaVersion.Fields[0].ID,
		"freeProcessId": reflow.ID, "refDocId": refDoc.ID, "linkId": refDoc.Links[0].ID, "changeId": changeReq.ID, "triggerId": triggerID, "contentTypeId": contentTypeID, "sectionId": "01900000-0000-7000-8000-000000000000", "attachmentId": "01900000-0000-7000-8000-000000000000", "processVersionId": iqcVersion.ID, "processTemplateId": iqcVersion.TemplateID, "subprocessId": iqcVersion.Subprocesses[0].ID,
		"pointId": iqcVersion.Subprocesses[0].Points[0].ID, "variantId": iqcVersion.Subprocesses[0].Points[0].Variants[0].ID,
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
		{http.MethodPost, "/api/v1/document-numbers/sets", `{"customer_id":"{customerId}","initial_revision":4}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		// Status nomor diubah dokumen kontrol (E-8). Nomor yang dibuat setup ini Reserved.
		{http.MethodPost, "/api/v1/document-numbers/{documentNumberId}/void", `{"reason":"Uji izin"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/document-numbers/{documentNumberId}/obsolete", `{}`, http.StatusForbidden, http.StatusConflict, http.StatusForbidden},
		// Impor nomor lama juga lewat E-8 (UC-14); pratinjau, supaya register tidak berubah.
		{http.MethodPost, "/api/v1/document-number-imports", `{"customer_id":"{customerId}","status":"released","text":"SC-XY-900","dry_run":true}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
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
		// Process template (M3.2): semua akun melihat (L-2), Pengelola template mengubah (E-6, D-64).
		{http.MethodGet, "/api/v1/process-templates", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/process-templates", `{"process_id":"{freeProcessId}"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/process-template-versions/{processVersionId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/process-template-versions/{processVersionId}/subprocesses", `{"name":"IQC PCB"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodPut, "/api/v1/process-template-versions/{processVersionId}/subprocesses/{subprocessId}", `{"name":"IQC MSD"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/process-template-versions/{processVersionId}/points", `{"subprocess_id":"{subprocessId}","document_type_id":"{cpTypeId}","text":"Check HIC"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodPut, "/api/v1/process-template-versions/{processVersionId}/points/{pointId}", `{"text":"Check MSL label"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Varian poin diubah Pengelola template (E-6, M3.3); pratinjau per model memakai L-2.
		{http.MethodPost, "/api/v1/process-template-versions/{processVersionId}/points/{pointId}/variants", `{"scope_level":"model","scope_target_id":"{modelId}","text":"Check MSL label model"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodPut, "/api/v1/process-template-versions/{processVersionId}/variants/{variantId}", `{"text":"Check MSL label customer"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodGet, "/api/v1/process-template-versions/{processVersionId}/resolved?model_id={modelId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		// Menerbitkan dan membuat versi baru lewat E-6 (M3.4). Di akhir kelompok
		// ini, karena versi terbit menolak perubahan di baris-baris sebelumnya.
		{http.MethodPost, "/api/v1/process-template-versions/{processVersionId}/publish", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/process-templates/{processTemplateId}/versions", "", http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		// Jenis dokumen dan perubahan diubah Admin (A-2, UC-05).
		{http.MethodGet, "/api/v1/number-settings", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/number-settings", `{"factory_code":"T"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/document-types", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/document-types", `{"name":"Work Instruction","letter":"W"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/document-types/{documentTypeId}", `{"name":"Process Flow Diagram","letter":"H"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/special-codes", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/special-codes", `{"code":"QA","name":"Quality"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/special-codes/{specialCodeId}", `{"name":"Health and Safety"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Format template (M3.1): semua akun melihat (L-2), admin mengubah (A-2, UC-20).
		{http.MethodGet, "/api/v1/format-templates", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/format-templates", `{"name":"PFD copy","document_type_id":"{documentTypeId}","risk_method":"none"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/format-templates/{formatTemplateId}", `{"name":"PFMEA AIAG 4th"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodGet, "/api/v1/format-template-versions/{formatVersionId}", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPut, "/api/v1/format-template-versions/{formatVersionId}", `{"highlight_rpn_above":100}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/format-template-versions/{formatVersionId}/fields", `{"area":"row","kind":"text","label":"Note"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/format-template-versions/{formatVersionId}/fields/{formatFieldId}", `{"label":"Process or Line"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPut, "/api/v1/format-template-versions/{formatVersionId}/rating-criteria", `{"items":[{"scale":"severity","value":1,"criteria":""}]}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPut, "/api/v1/format-template-versions/{formatVersionId}/symbols", `{"items":[{"key":"critical_process","name":"Critical Process"},{"key":"revision","name":"Revision"},{"key":"important_character","name":"Important Character"},{"key":"regulation","name":"Regulation"}]}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		// Dokumen (M4.1): melihat draft lewat L-4, mengubah lewat E-2. Admin tidak memegang keduanya.
		{http.MethodGet, "/api/v1/documents?customer_id={customerId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/documents", `{"document_number_id":"{freeNumberId}","model_ids":["{modelId}"]}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Rilis (M5.5) lewat E-7. Engineer memegang E-7 tetapi juga menyusun dokumen uji,
		// jadi aturan pencatat bukan penyusun menolaknya (R-APR-02).
		{http.MethodPost, "/api/v1/documents/{documentId}/release", "", http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}/revisions", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Diff (M5.6): dokumen uji belum pernah terbit, jadi revisi 0 ditolak sesudah haknya lolos.
		{http.MethodGet, "/api/v1/documents/{documentId}/diff?from=0", "", http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodPut, "/api/v1/documents/{documentId}/header", `{"header":{},"header_version":1}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/documents/{documentId}/rows", `{"content":{}}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}/composition?process_template_version_id={processVersionId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Versi IQC sudah diterbitkan oleh baris publish di atas.
		{http.MethodPut, "/api/v1/documents/{documentId}/composition", `{"process_template_version_id":"{processVersionId}"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Dokumen uji sudah punya beberapa baris, jadi akun yang berhak mendapat 422 (bukan 403): haknya lolos, daftar barisnya yang ditolak.
		{http.MethodPut, "/api/v1/documents/{documentId}/order", `{"row_ids":["{rowId}"]}`, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		// Menyalin dari dirinya sendiri ditolak aturan (422), sesudah haknya lolos.
		{http.MethodPost, "/api/v1/documents/{documentId}/copy", `{"source_document_id":"{documentId}"}`, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		// Dokumen uji adalah dokumen tunggal tanpa pasangan PFMEA dan CP, jadi akun yang berhak mendapat 422 sesudah haknya lolos.
		// Master dokumen terkait (D-130): semua melihat, Pengelola template (E-6) mengubah.
		{http.MethodGet, "/api/v1/referenced-documents", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/referenced-documents", `{"number":"SW-ZZ-318","title":"Incoming SAP System Barcode"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodPut, "/api/v1/referenced-documents/{refDocId}", `{"title":"Material Receiving","revision":"19"}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/referenced-documents/{refDocId}/links", `{"process_id":"{freeProcessId}"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/referenced-documents/{refDocId}/links/{linkId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Dokumen uji memakai form AIAG-VDA (CP tanpa kolom Relevant Document), jadi akun yang berhak mendapat 422.
		{http.MethodPost, "/api/v1/documents/{documentId}/references", "", http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}/pfmea-cp", "", http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodPut, "/api/v1/documents/{documentId}/pfmea-cp", `{"process_id":"{processId}","checked":true}`, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}/differences", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Baris uji tidak tertaut ke template, jadi akun yang berhak mendapat 422 sesudah haknya lolos.
		{http.MethodPost, "/api/v1/documents/{documentId}/rows/{rowId}/detach", `{"reason":"x"}`, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		{http.MethodPost, "/api/v1/documents/{documentId}/rows/batch", `{"rows":[{"content":{}}]}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodPut, "/api/v1/documents/{documentId}/rows/{rowId}", `{"content":{},"version":1}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/documents/{documentId}/rows/{rowId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Form set (langkah D): melihat lewat L-2, menyusun lewat A-2, menerbitkan lewat V-3 (D-139).
		// Engineer memegang V-3, tetapi format di dalam set masih draft, jadi jawabannya 409, bukan 403.
		{http.MethodGet, "/api/v1/form-sets", "", http.StatusOK, http.StatusOK, http.StatusOK},
		{http.MethodPost, "/api/v1/form-sets", `{"name":"Set uji izin","pfd_format_version_id":"{pfdSheetId}","pfmea_format_version_id":"{pfmeaSheetId}","cp_format_version_id":"{cpSheetId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
		{http.MethodPut, "/api/v1/form-set-versions/{formSetVersionId}", `{"pfd_format_version_id":"{pfdSheetId}","pfmea_format_version_id":"{pfmeaSheetId}","cp_format_version_id":"{cpSheetId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/form-set-versions/{formSetVersionId}/publish", "", http.StatusForbidden, http.StatusConflict, http.StatusForbidden},
		{http.MethodPost, "/api/v1/form-sets/{formSetId}/versions", "", http.StatusForbidden, http.StatusForbidden, http.StatusConflict},
		{http.MethodPut, "/api/v1/customers/{customerId}/default-form-set", `{"form_set_id":"{formSetId}"}`, http.StatusForbidden, http.StatusForbidden, http.StatusNoContent},
		// Tabel AP: versi PFMEA di sini memakai RPN, jadi admin mendapat 422; hitungan AP boleh semua akun tetapi ditolak 422 dengan alasan yang sama.
		{http.MethodPut, "/api/v1/format-template-versions/{formatVersionId}/ap-rules", `{"items":[{"s_min":1,"s_max":10,"o_min":1,"o_max":10,"d_min":1,"d_max":10,"priority":"L"}]}`, http.StatusForbidden, http.StatusForbidden, http.StatusUnprocessableEntity},
		{http.MethodGet, "/api/v1/format-template-versions/{formatVersionId}/action-priority?severity=8&occurrence=6&detection=4", "", http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity},
		// Menerbitkan dan membuat versi baru lewat A-2 (M3.4), sesudah semua perubahan versi itu.
		{http.MethodPost, "/api/v1/format-template-versions/{formatVersionId}/publish", "", http.StatusForbidden, http.StatusForbidden, http.StatusOK},
		{http.MethodPost, "/api/v1/format-templates/{formatTemplateId}/versions", "", http.StatusForbidden, http.StatusForbidden, http.StatusCreated},
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
		// Perubahan (M5.1). Dokumen uji sudah tertaut ke perubahan, jadi menambahnya lagi 409 dan
		// melepasnya (dokumen terakhir) 422: haknya lolos, aturannya menolak.
		{http.MethodGet, "/api/v1/changes?customer_id={customerId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/changes", `{"customer_id":"{customerId}","trigger_id":"{triggerId}","change_type_id":"{contentTypeId}","reason":"Uji","document_ids":["{documentId}"]}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/changes/{changeId}", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		{http.MethodPost, "/api/v1/changes/{changeId}/documents", `{"document_id":"{documentId}"}`, http.StatusForbidden, http.StatusConflict, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/changes/{changeId}/documents/{documentId}", "", http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusForbidden},
		// Daftar periksa dampak (M5.7) lewat E-3; engineer (Penyusun) boleh selama perubahan diajukan.
		{http.MethodPut, "/api/v1/changes/{changeId}/impacts", `{"items":[{"kind":"wi","document_reference":"SW-ZZ-116","decision":"affected"}]}`, http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Verifikasi (M5.3): engineer (Penyusun) tanpa V-1 ditolak, seperti admin dan viewer.
		{http.MethodPost, "/api/v1/changes/{changeId}/verification", `{"decision":"accept","note":"OK"}`, http.StatusForbidden, http.StatusForbidden, http.StatusForbidden},
		// Lampiran (M5.2): body multipart ditandai awalan --revioten. ID lampiran palsu: 404.
		{http.MethodPost, "/api/v1/changes/{changeId}/attachments", multipartBody, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		{http.MethodGet, "/api/v1/changes/{changeId}/attachments/{attachmentId}", "", http.StatusForbidden, http.StatusNotFound, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/changes/{changeId}/attachments/{attachmentId}", "", http.StatusForbidden, http.StatusNotFound, http.StatusForbidden},
		{http.MethodGet, "/api/v1/documents/{documentId}/changes", "", http.StatusForbidden, http.StatusOK, http.StatusForbidden},
		// Bagian per PCB (D-155), sengaja di akhir: sesudah dokumen punya bagian, baris baru wajib menyebut
		// bagiannya. Dokumen uji tidak punya part number, jadi bagian dibuat tanpa part number.
		{http.MethodPost, "/api/v1/documents/{documentId}/sections", `{"name":"PCB A"}`, http.StatusForbidden, http.StatusCreated, http.StatusForbidden},
		// ID bagian palsu: akun yang berhak mendapat 404 sesudah haknya lolos.
		{http.MethodPut, "/api/v1/documents/{documentId}/sections/{sectionId}", `{"name":"PCB B"}`, http.StatusForbidden, http.StatusNotFound, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/documents/{documentId}/sections/{sectionId}", "", http.StatusForbidden, http.StatusNotFound, http.StatusForbidden},
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
			if strings.HasPrefix(tc.body, "--revioten") {
				req.Header.Set("Content-Type", "multipart/form-data; boundary=revioten")
			}
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
