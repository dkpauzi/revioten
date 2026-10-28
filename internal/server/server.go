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
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
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
		audit: deps.Audit, products: deps.Products, processes: deps.Processes, log: log,
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
	log       *slog.Logger
}
