// Package server adalah adapter HTTP: daftar route, handler, dan format error.
// Paket ini tipis dan tidak memuat aturan bisnis (R-API-01). Aturan ada di
// paket domain seperti internal/customer.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
)

// Pinger adalah bagian database yang dipakai /health. *pgxpool.Pool memenuhinya.
type Pinger interface {
	Ping(ctx context.Context) error
}

// CustomerService adalah operasi customer yang dipakai API.
// *customer.Service memenuhinya; test memakai versi palsu.
type CustomerService interface {
	List(ctx context.Context) ([]customer.Customer, error)
	Create(ctx context.Context, in customer.NewCustomer) (customer.Customer, error)
}

// AuthService adalah operasi login dan sesi yang dipakai API.
// *auth.Service memenuhinya; test memakai versi palsu.
type AuthService interface {
	Login(ctx context.Context, login, password string) (auth.Actor, string, error)
	Authenticate(ctx context.Context, token string) (auth.Actor, error)
	Logout(ctx context.Context, token string) error
	ChangePassword(ctx context.Context, actor auth.Actor, current, newPassword string) (string, error)
}

// Deps berisi semua yang dibutuhkan handler.
type Deps struct {
	DB        Pinger
	Customers CustomerService
	Auth      AuthService
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
	a := &apiServer{db: deps.DB, customers: deps.Customers, auth: deps.Auth, log: log}

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
// Isinya dibagi per file: health.go, session.go, customers.go, dan apierror.go.
type apiServer struct {
	db        Pinger
	customers CustomerService
	auth      AuthService
	log       *slog.Logger
}
