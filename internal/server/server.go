// Package server adalah adapter HTTP: daftar route, handler, dan format error.
// Paket ini tipis dan tidak memuat aturan bisnis (R-API-01). Aturan ada di
// paket domain seperti internal/customer.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

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

// Deps berisi semua yang dibutuhkan handler.
type Deps struct {
	DB        Pinger
	Customers CustomerService
	// Logger untuk log server. Bila nil, dipakai slog.Default().
	Logger *slog.Logger
}

// NewHandler menyusun semua route server:
//   - route API dari kontrak api/openapi.yaml (lihat api.gen.go), termasuk /health;
//   - tampilan web (SvelteKit) dari ui untuk alamat lainnya.
//
// Setiap permintaan mendapat request ID (requestid.go).
func NewHandler(ui fs.FS, deps Deps) http.Handler {
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	a := &apiServer{db: deps.DB, customers: deps.Customers, log: log}

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
	mux.Handle("/", handleSPA(ui))
	return withRequestID(mux)
}

// apiServer menjalankan operasi di kontrak (StrictServerInterface di api.gen.go).
// Isinya dibagi per file: health.go, customers.go, dan apierror.go.
type apiServer struct {
	db        Pinger
	customers CustomerService
	log       *slog.Logger
}
