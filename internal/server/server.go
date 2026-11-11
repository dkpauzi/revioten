// Package server adalah adapter HTTP: daftar route, handler, dan format error.
// Paket ini tipis dan tidak memuat aturan bisnis (R-API-01). Aturan ada di
// paket domain seperti internal/customer.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/settings"
	"github.com/dkpauzi/revioten/internal/template"
)

// Pinger adalah bagian database yang dipakai /health. *pgxpool.Pool memenuhinya.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Setiap operasi domain di bawah menerima aktor dan memeriksa izinnya sendiri
// lewat access.Authorize (R-API-01, M1.2). Adapter HTTP ini hanya meneruskan
// aktor dari sesi.

// CustomerService adalah operasi customer dan Project Group yang dipakai API.
// *customer.Service memenuhinya; test memakai versi palsu.
type CustomerService interface {
	List(ctx context.Context, actor access.Actor) ([]customer.Customer, error)
	Create(ctx context.Context, actor access.Actor, in customer.NewCustomer) (customer.Customer, error)
	Update(ctx context.Context, actor access.Actor, id string, in customer.CustomerUpdate) (customer.Customer, error)
	ChangeCode(ctx context.Context, actor access.Actor, id, code string) (customer.Customer, error)
	ListProjectGroups(ctx context.Context, actor access.Actor) ([]customer.ProjectGroup, error)
	CreateProjectGroup(ctx context.Context, actor access.Actor, in customer.ProjectGroupInput) (customer.ProjectGroup, error)
	UpdateProjectGroup(ctx context.Context, actor access.Actor, id string, in customer.ProjectGroupInput) (customer.ProjectGroup, error)
	SetDefaultRoute(ctx context.Context, actor access.Actor, id, routeID string) (customer.Customer, error)
}

// ProductService adalah operasi group model, model, dan part number yang
// dipakai API (M1.4). *product.Service memenuhinya; test memakai versi palsu.
type ProductService interface {
	ListModelGroups(ctx context.Context, actor access.Actor, customerID string) ([]product.ModelGroup, error)
	CreateModelGroup(ctx context.Context, actor access.Actor, in product.NewModelGroup) (product.ModelGroup, error)
	RenameModelGroup(ctx context.Context, actor access.Actor, id, name string) (product.ModelGroup, error)
	ListModels(ctx context.Context, actor access.Actor, customerID string) ([]product.Model, error)
	CreateModel(ctx context.Context, actor access.Actor, in product.NewModel) (product.Model, error)
	UpdateModel(ctx context.Context, actor access.Actor, id string, in product.ModelUpdate) (product.Model, error)
	ListPartNumbers(ctx context.Context, actor access.Actor, customerID string) ([]product.PartNumber, error)
	CreatePartNumber(ctx context.Context, actor access.Actor, in product.NewPartNumber) (product.PartNumber, error)
	UpdatePartNumber(ctx context.Context, actor access.Actor, id string, in product.PartNumberUpdate) (product.PartNumber, error)
}

// ProcessService adalah operasi master proses yang dipakai API (M1.5).
// *process.Service memenuhinya; test memakai versi palsu.
type ProcessService interface {
	List(ctx context.Context, actor access.Actor) ([]process.Process, error)
	Get(ctx context.Context, actor access.Actor, id string) (process.Process, error)
	Create(ctx context.Context, actor access.Actor, in process.Input) (process.Process, error)
	Update(ctx context.Context, actor access.Actor, id string, in process.Input) (process.Process, error)
}

// NumberingService adalah operasi nomor dokumen yang dipakai API: bagian nomor
// (M1.6) dan register nomor (Tahap 2). *numbering.Service memenuhinya; test
// memakai versi palsu.
type NumberingService interface {
	GetSettings(ctx context.Context, actor access.Actor) (numbering.Settings, error)
	UpdateSettings(ctx context.Context, actor access.Actor, in numbering.SettingsInput) (numbering.Settings, error)
	Register(ctx context.Context, actor access.Actor, in numbering.NewNumber) (numbering.Registration, error)
	RegisterSet(ctx context.Context, actor access.Actor, in numbering.NewSet) (numbering.Registration, error)
	Import(ctx context.Context, actor access.Actor, in numbering.ImportInput) (numbering.ImportReport, error)
	NextNumber(ctx context.Context, actor access.Actor, customerID, documentTypeID string) (string, error)
	ListNumbers(ctx context.Context, actor access.Actor, f numbering.ListFilter) ([]numbering.DocumentNumber, string, error)
	Void(ctx context.Context, actor access.Actor, id, reason string) (numbering.DocumentNumber, error)
	MakeObsolete(ctx context.Context, actor access.Actor, id, reason string) (numbering.DocumentNumber, error)
	ListDocumentTypes(ctx context.Context, actor access.Actor) ([]numbering.DocumentType, error)
	CreateDocumentType(ctx context.Context, actor access.Actor, in numbering.DocumentTypeInput) (numbering.DocumentType, error)
	UpdateDocumentType(ctx context.Context, actor access.Actor, id string, in numbering.DocumentTypeInput) (numbering.DocumentType, error)
	ListSpecialCodes(ctx context.Context, actor access.Actor) ([]numbering.SpecialCode, error)
	CreateSpecialCode(ctx context.Context, actor access.Actor, in numbering.NewSpecialCode) (numbering.SpecialCode, error)
	RenameSpecialCode(ctx context.Context, actor access.Actor, id, name string) (numbering.SpecialCode, error)
}

// ChangeMasterService adalah operasi daftar untuk perubahan yang dipakai API
// (M1.6): jenis perubahan, sumber perubahan, dan kategori 4M.
// *change.Service memenuhinya; test memakai versi palsu.
type ChangeMasterService interface {
	ListChangeTypes(ctx context.Context, actor access.Actor) ([]change.ChangeType, error)
	RenameChangeType(ctx context.Context, actor access.Actor, id, name string) (change.ChangeType, error)
	ListChangeSources(ctx context.Context, actor access.Actor) ([]change.ChangeSource, error)
	CreateChangeSource(ctx context.Context, actor access.Actor, name string) (change.ChangeSource, error)
	UpdateChangeSource(ctx context.Context, actor access.Actor, id string, in change.ChangeSourceUpdate) (change.ChangeSource, error)
	ListFourMCategories(ctx context.Context, actor access.Actor) ([]change.FourMCategory, error)
	RenameFourMCategory(ctx context.Context, actor access.Actor, id, name string) (change.FourMCategory, error)
	SetApprovalRoute(ctx context.Context, actor access.Actor, id, routeID string) (change.ChangeType, error)
}

// ApprovalService adalah operasi jabatan dan rute approval yang dipakai API
// (M1.8). *approval.Service memenuhinya; test memakai versi palsu.
type ApprovalService interface {
	ListPositions(ctx context.Context, actor access.Actor) ([]approval.Position, error)
	RenamePosition(ctx context.Context, actor access.Actor, id, name string) (approval.Position, error)
	AddHolder(ctx context.Context, actor access.Actor, positionID string, in approval.NewHolder) (approval.Holder, error)
	RemoveHolder(ctx context.Context, actor access.Actor, positionID, holderID string) error
	ListRoutes(ctx context.Context, actor access.Actor) ([]approval.Route, error)
	CreateRoute(ctx context.Context, actor access.Actor, in approval.RouteInput) (approval.Route, error)
	UpdateRoute(ctx context.Context, actor access.Actor, id string, in approval.RouteInput) (approval.Route, error)
}

// TemplateService adalah operasi format template yang dipakai API (M3.1).
// *template.Service memenuhinya; test memakai versi palsu.
type TemplateService interface {
	ListFormatTemplates(ctx context.Context, actor access.Actor) ([]template.FormatTemplate, error)
	GetFormatVersion(ctx context.Context, actor access.Actor, id string) (template.FormatVersion, error)
	CreateFormatTemplate(ctx context.Context, actor access.Actor, in template.NewTemplate) (template.FormatTemplate, error)
	RenameFormatTemplate(ctx context.Context, actor access.Actor, id, name string) (template.FormatTemplate, error)
	AddField(ctx context.Context, actor access.Actor, versionID string, in template.NewField) (template.FormatVersion, error)
	UpdateField(ctx context.Context, actor access.Actor, versionID, fieldID string, in template.FieldUpdate) (template.FormatVersion, error)
	SetRatingCriteria(ctx context.Context, actor access.Actor, versionID string, ratings []template.Rating) (template.FormatVersion, error)
	SaveSymbols(ctx context.Context, actor access.Actor, versionID string, symbols []template.Symbol) (template.FormatVersion, error)
	SetHighlights(ctx context.Context, actor access.Actor, versionID string, in template.HighlightSettings) (template.FormatVersion, error)
	// Tabel AP (form set langkah A, D-134).
	SetAPRules(ctx context.Context, actor access.Actor, versionID string, rules []template.APRule) (template.FormatVersion, error)
	ActionPriorityOf(ctx context.Context, actor access.Actor, versionID string, severity, occurrence, detection int) (string, error)
	// Versi terbit dan versi baru (M3.4, R-TPL-05).
	PublishFormatVersion(ctx context.Context, actor access.Actor, versionID string) (template.FormatVersion, error)
	NewFormatVersion(ctx context.Context, actor access.Actor, templateID string) (template.FormatVersion, error)
}

// ProcessTemplateService adalah operasi process template yang dipakai API
// (M3.2). *template.ProcessService memenuhinya; test memakai versi palsu.
type ProcessTemplateService interface {
	ListProcessTemplates(ctx context.Context, actor access.Actor) ([]template.ProcessTemplate, error)
	GetProcessVersion(ctx context.Context, actor access.Actor, id string) (template.ProcessVersion, error)
	CreateProcessTemplate(ctx context.Context, actor access.Actor, processID string) (template.ProcessTemplate, error)
	AddSubprocess(ctx context.Context, actor access.Actor, versionID, name string) (template.ProcessVersion, error)
	UpdateSubprocess(ctx context.Context, actor access.Actor, versionID, subprocessID string, in template.SubprocessUpdate) (template.ProcessVersion, error)
	AddPoint(ctx context.Context, actor access.Actor, versionID string, in template.NewPoint) (template.ProcessVersion, error)
	UpdatePoint(ctx context.Context, actor access.Actor, versionID, pointID string, in template.PointUpdate) (template.ProcessVersion, error)
	// Cakupan poin dan varian (M3.3, D-129).
	AddVariant(ctx context.Context, actor access.Actor, versionID, pointID string, in template.NewVariant) (template.ProcessVersion, error)
	UpdateVariant(ctx context.Context, actor access.Actor, versionID, variantID string, in template.VariantUpdate) (template.ProcessVersion, error)
	ResolvePoints(ctx context.Context, actor access.Actor, versionID, modelID string) (template.Resolved, error)
	// Versi terbit dan versi baru (M3.4, R-TPL-05).
	PublishProcessVersion(ctx context.Context, actor access.Actor, versionID string) (template.ProcessVersion, error)
	NewProcessVersion(ctx context.Context, actor access.Actor, templateID string) (template.ProcessVersion, error)
}

// SettingsService adalah operasi pengaturan sistem yang dipakai API (M1.8).
// *settings.Service memenuhinya; test memakai versi palsu.
type SettingsService interface {
	Get(ctx context.Context, actor access.Actor) (settings.Settings, error)
	Apply(ctx context.Context, actor access.Actor, in settings.Update) (settings.Settings, error)
}

// AuthService adalah operasi login, sesi, dan pengelolaan akun yang dipakai API.
// *auth.Service memenuhinya; test memakai versi palsu.
type AuthService interface {
	Login(ctx context.Context, login, password string) (access.Actor, string, error)
	Authenticate(ctx context.Context, token string) (access.Actor, error)
	Logout(ctx context.Context, actor access.Actor, token string) error
	ChangePassword(ctx context.Context, actor access.Actor, current, newPassword string) (string, error)
	ListAccounts(ctx context.Context, actor access.Actor) ([]auth.AccountInfo, error)
	GetAccount(ctx context.Context, actor access.Actor, accountID string) (auth.AccountInfo, error)
	CreateAccount(ctx context.Context, actor access.Actor, in auth.NewAccount) (auth.AccountInfo, string, error)
	ResetPassword(ctx context.Context, actor access.Actor, accountID string) (string, error)
	DisableAccount(ctx context.Context, actor access.Actor, accountID string) (auth.AccountInfo, error)
	EnableAccount(ctx context.Context, actor access.Actor, accountID string) (auth.AccountInfo, error)
	SetRoles(ctx context.Context, actor access.Actor, accountID string, grants []access.RoleGrant) error
}

// AccessService adalah operasi hak akses yang dipakai API.
// *access.Checker memenuhinya; test memakai versi palsu.
type AccessService interface {
	Permissions(ctx context.Context, actor access.Actor) ([]access.Permission, error)
	ListRoles(ctx context.Context, actor access.Actor) ([]access.Role, error)
}

// AuditService adalah operasi membaca audit log yang dipakai API (M1.3).
// *audit.Service memenuhinya; test memakai versi palsu.
type AuditService interface {
	List(ctx context.Context, actor access.Actor, before int64, limit int) ([]audit.Entry, error)
	Verify(ctx context.Context, actor access.Actor) (audit.Verification, error)
}

// Deps berisi semua yang dibutuhkan handler.
type Deps struct {
	DB        Pinger
	Customers CustomerService
	Auth      AuthService
	Access    AccessService
	Audit     AuditService
	Products  ProductService
	Processes ProcessService
	Numbering NumberingService
	Changes   ChangeMasterService
	Approval  ApprovalService
	Settings  SettingsService
	Templates TemplateService
	// ProcessTemplates adalah pustaka proses (M3.2).
	ProcessTemplates ProcessTemplateService
	// Logger untuk log server. Bila nil, dipakai slog.Default().
	Logger *slog.Logger
}

// NewHandler menyusun semua route server:
//   - route API dari kontrak api/openapi.yaml (lihat api.gen.go), termasuk /health;
//   - tampilan web (SvelteKit) dari ui untuk alamat lainnya.
//
// Setiap permintaan mendapat request ID (requestid.go), dan permintaan yang
// mengubah data dari situs lain ditolak (cek Origin, brief bagian 16).
func NewHandler(ui fs.FS, deps Deps) http.Handler {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	a := &apiServer{
		db: deps.DB, customers: deps.Customers, auth: deps.Auth, access: deps.Access,
		audit: deps.Audit, products: deps.Products, processes: deps.Processes,
		numbering: deps.Numbering, changes: deps.Changes, approval: deps.Approval, settings: deps.Settings,
		templates: deps.Templates, processTemplates: deps.ProcessTemplates, log: log,
	}

	// Route dari kontrak didaftarkan oleh kode hasil generate ke mux khusus API.
	// Strict handler membaca body JSON dan menulis jawaban bertipe; error-nya
	// diserahkan ke handler error kita supaya formatnya sesuai R-API-13.
	apiMux := http.NewServeMux()
	strict := NewStrictHandlerWithOptions(a, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  a.requestError,
		ResponseErrorHandlerFunc: a.responseError,
	})
	HandlerWithOptions(strict, StdHTTPServerOptions{
		BaseRouter:       apiMux,
		ErrorHandlerFunc: a.requestError,
		// Semua route API butuh sesi, kecuali publicRoutes (session.go).
		Middlewares: []MiddlewareFunc{a.requireSession},
	})
	api := limitBody(a.routeAPI(apiMux))

	mux := http.NewServeMux()
	// E-17 (diputuskan): /health ada di root sebagai endpoint operasional,
	// di luar /api/v1, dan tercatat di kontrak sejak M0.5.
	mux.Handle("/health", api)
	mux.Handle("/api/", api)
	// Pola "/" sengaja tanpa method. Pola "GET /" ditolak ServeMux karena
	// bentrok dengan "/api/": untuk GET /api/x, yang satu lebih sempit di
	// method, yang lain lebih sempit di path, jadi tidak ada yang menang.
	// Method selain GET dan HEAD ditolak di dalam handleSPA.
	mux.Handle("/", handleSPA(ui, a.loggedIn))

	// CrossOriginProtection (Go 1.25+) menolak POST, PUT, PATCH, dan DELETE
	// yang datang dari situs lain, dengan membaca header Sec-Fetch-Site dan
	// Origin dari browser. Ini perlindungan CSRF untuk cookie sesi.
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(a.crossOriginDenied))
	return withRequestID(csrf.Handler(mux))
}

// apiServer menjalankan operasi di kontrak (StrictServerInterface di api.gen.go).
// Isinya dibagi per file: health.go, session.go, accounts.go, customers.go,
// dan apierror.go.
type apiServer struct {
	db        Pinger
	customers CustomerService
	auth      AuthService
	access    AccessService
	audit     AuditService
	products  ProductService
	processes ProcessService
	numbering NumberingService
	changes   ChangeMasterService
	approval  ApprovalService
	settings  SettingsService
	templates TemplateService
	// processTemplates adalah pustaka proses (M3.2).
	processTemplates ProcessTemplateService
	log              *slog.Logger
}
