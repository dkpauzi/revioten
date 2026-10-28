package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dkpauzi/revioten/internal/problem"
)

// maxBodyBytes adalah ukuran body permintaan API paling besar (1 MiB).
// Unggahan file nanti punya batasnya sendiri.
// ASUMSI E-20: brief belum menyebut batas ini.
const maxBodyBytes = 1 << 20

// limitBody membatasi ukuran body permintaan, supaya klien tidak bisa
// menghabiskan memori server dengan body yang sangat besar.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
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

// responseError menangani error tak terduga dari handler, misalnya database
// putus. Pengguna hanya menerima pesan umum dan request ID; detailnya dicatat
// di log server (R-API-13, E-18).
func (a *apiServer) responseError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("permintaan API gagal",
		"request_id", requestIDFrom(r.Context()), "method", r.Method, "path", r.URL.Path, "error", err)
	a.writeError(w, r, http.StatusInternalServerError, ErrorCodeInternalError,
		"Something went wrong on the server. Give the request ID to the administrator.", nil)
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

// problemResponse menerjemahkan penolakan input dari paket domain
// (internal/problem) menjadi status dan isi error API: ValidationError
// menjadi 422 validation_failed, ConflictError menjadi 409 conflict.
// ok bernilai false bila err bukan penolakan input.
func problemResponse(ctx context.Context, err error) (status int, body Error, ok bool) {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		return http.StatusUnprocessableEntity,
			errorBody(ctx, ErrorCodeValidationFailed, "Some fields are not valid.", toDetails(invalid.Problems)), true
	case errors.As(err, &conflict):
		return http.StatusConflict,
			errorBody(ctx, ErrorCodeConflict, "The data conflicts with data that already exists.", toDetails(conflict.Problems)), true
	}
	return 0, Error{}, false
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
