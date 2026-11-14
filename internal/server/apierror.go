package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// maxBodyBytes adalah ukuran body permintaan API paling besar (1 MiB).
// Unggahan lampiran (M5.2) punya batasnya sendiri, maxUploadBytes.
// E-20 (D-99): bagian dari baseline konvensi API M0.5 (brief R-API-13).
const maxBodyBytes = 1 << 20

// maxUploadBytes adalah batas body unggahan lampiran: 20 MB isi berkas
// ditambah ruang untuk bagian multipart lain (ASUMSI E-99).
const maxUploadBytes = 21 << 20

// limitBody membatasi ukuran body permintaan, supaya klien tidak bisa
// menghabiskan memori server dengan body yang sangat besar.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxBodyBytes)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/attachments") {
			limit = maxUploadBytes
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

// routeAPI meneruskan permintaan ke route dari kontrak. Bila alamat atau
// method tidak ada di kontrak, jawabannya tetap memakai format error JSON,
// bukan teks polos bawaan ServeMux (R-API-13).
func (a *apiServer) routeAPI(apiMux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler mengembalikan pola route yang cocok, atau "" bila tidak ada.
		if _, pattern := apiMux.Handler(r); pattern != "" {
			apiMux.ServeHTTP(w, r)
			return
		}
		if allowed := allowedMethods(apiMux, r); len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			a.writeError(w, r, http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed,
				"This method is not allowed for this address.", nil)
			return
		}
		a.writeError(w, r, http.StatusNotFound, ErrorCodeNotFound, "This address does not exist in the API.", nil)
	})
}

// allowedMethods mencari method lain yang punya route untuk alamat ini,
// dengan mencoba setiap method ke apiMux. Route GET juga melayani HEAD.
func allowedMethods(apiMux *http.ServeMux, r *http.Request) []string {
	var allowed []string
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		probe := r.Clone(r.Context())
		probe.Method = method
		if _, pattern := apiMux.Handler(probe); pattern != "" {
			allowed = append(allowed, method)
		}
	}
	return allowed
}

// requestError menangani permintaan yang tidak bisa dibaca, misalnya body yang
// bukan JSON. Detail error hanya dicatat di log, karena pesan bawaan Go bisa
// memuat potongan isi permintaan.
func (a *apiServer) requestError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Info("permintaan API tidak bisa dibaca",
		"request_id", requestIDFrom(r.Context()), "method", r.Method, "path", r.URL.Path, "error", err)

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		a.writeError(w, r, http.StatusRequestEntityTooLarge, ErrorCodeInvalidRequest,
			"The request body is larger than 1 MiB.", nil)
		return
	}
	a.writeError(w, r, http.StatusBadRequest, ErrorCodeInvalidRequest,
		"The request body is not valid JSON for this address.", nil)
}

// responseError menerjemahkan error dari handler ke format error API. Semua
// handler cukup mengembalikan error dari paket domain; status dan kodenya
// ditentukan di satu tempat ini:
//   - access.ErrForbidden: 403 forbidden (M1.2);
//   - problem.ErrNotFound: 404 not_found;
//   - *problem.ValidationError: 422 validation_failed, dengan rincian per field;
//   - *problem.ConflictError: 409 conflict, dengan rincian per field;
//   - error lain, misalnya database putus: 500 internal_error. Pengguna hanya
//     menerima pesan umum dan request ID; detailnya dicatat di log server
//     (R-API-13, E-18).
func (a *apiServer) responseError(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.Is(err, access.ErrForbidden):
		a.log.Info("akses ditolak", "request_id", requestIDFrom(r.Context()), "login", actorFrom(r.Context()).Login,
			"method", r.Method, "path", r.URL.Path)
		a.writeError(w, r, http.StatusForbidden, ErrorCodeForbidden, "You do not have the right to do this.", nil)
	case errors.Is(err, problem.ErrNotFound):
		a.writeError(w, r, http.StatusNotFound, ErrorCodeNotFound, "The requested data does not exist.", nil)
	case errors.As(err, &invalid):
		a.writeError(w, r, http.StatusUnprocessableEntity, ErrorCodeValidationFailed,
			"Some fields are not valid.", toDetails(invalid.Problems))
	case errors.As(err, &conflict):
		a.writeError(w, r, http.StatusConflict, ErrorCodeConflict,
			"The data conflicts with data that already exists.", toDetails(conflict.Problems))
	default:
		a.log.Error("permintaan API gagal",
			"request_id", requestIDFrom(r.Context()), "method", r.Method, "path", r.URL.Path, "error", err)
		a.writeError(w, r, http.StatusInternalServerError, ErrorCodeInternalError,
			"Something went wrong on the server. Give the request ID to the administrator.", nil)
	}
}

// writeError menulis satu error dalam format kontrak (skema Error).
func (a *apiServer) writeError(w http.ResponseWriter, r *http.Request, status int, code ErrorCode, message string, details []ErrorDetail) {
	body := errorBody(r.Context(), code, message, details)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Status sudah terkirim, jadi kegagalan ini hanya bisa dicatat.
		a.log.Error("gagal menulis error API", "request_id", body.RequestId, "error", err)
	}
}

// crossOriginDenied menjawab permintaan yang mengubah data dari situs lain
// (CSRF), yang ditolak http.CrossOriginProtection di server.go.
func (a *apiServer) crossOriginDenied(w http.ResponseWriter, r *http.Request) {
	a.log.Warn("permintaan dari situs lain ditolak",
		"request_id", requestIDFrom(r.Context()), "method", r.Method, "path", r.URL.Path,
		"origin", r.Header.Get("Origin"), "sec_fetch_site", r.Header.Get("Sec-Fetch-Site"))
	a.writeError(w, r, http.StatusForbidden, ErrorCodeForbidden,
		"Requests that change data must come from this application.", nil)
}

// toDetails mengubah masalah input dari domain menjadi ErrorDetail kontrak.
func toDetails(problems []problem.Problem) []ErrorDetail {
	details := make([]ErrorDetail, 0, len(problems))
	for _, p := range problems {
		details = append(details, ErrorDetail{Field: p.Field, Code: ErrorDetailCode(p.Code), Message: p.Message})
	}
	return details
}

// errorBody menyusun isi error dengan request ID dari context.
func errorBody(ctx context.Context, code ErrorCode, message string, details []ErrorDetail) Error {
	body := Error{Code: code, Message: message, RequestId: requestIDFrom(ctx)}
	if len(details) > 0 {
		body.Details = &details
	}
	return body
}
