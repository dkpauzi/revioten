package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/dkpauzi/revioten/internal/auth"
)

// sessionCookieName adalah nama cookie sesi. Awalan __Host- membuat browser
// hanya menerimanya dengan Secure, Path=/, dan tanpa Domain, jadi cookie ini
// tidak bisa dipasang atau ditimpa subdomain lain (ASVS 3.3.3).
const sessionCookieName = "__Host-revioten_session"

// publicRoutes adalah route API yang boleh dipanggil tanpa login. Semua route
// lain butuh sesi, termasuk route baru yang lupa didaftarkan (tolak bila tidak
// disebut). Isinya harus sama dengan operasi bertanda `security: []` di
// api/openapi.yaml; TestPublicRoutesMatchContract memeriksanya.
var publicRoutes = map[string]bool{
	"GET /health":          true,
	"POST /api/v1/session": true,
}

// passwordChangeRoutes adalah route yang masih boleh dipakai akun yang wajib
// mengganti password (E-08): lihat sesi, logout, dan ganti password.
var passwordChangeRoutes = map[string]bool{
	"GET /api/v1/session":          true,
	"DELETE /api/v1/session":       true,
	"PUT /api/v1/account/password": true,
}

// sessionKey adalah kunci sesi di context permintaan.
type sessionKey struct{}

// session adalah sesi yang sudah diperiksa requireSession.
type session struct {
	actor auth.Actor
	token string
}

// sessionFrom mengambil sesi dari context. Hanya route yang tidak publik yang
// punya sesi.
func sessionFrom(ctx context.Context) session {
	s, _ := ctx.Value(sessionKey{}).(session)
	return s
}

// sessionToken membaca token dari cookie sesi, atau "" bila tidak ada.
func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// requireSession memeriksa sesi untuk setiap route API yang tidak publik.
// Middleware ini dipasang di dalam kode hasil generate, setelah ServeMux memilih
// route, jadi r.Pattern sudah berisi pola route, misalnya "GET /api/v1/customers".
func (a *apiServer) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicRoutes[r.Pattern] {
			next.ServeHTTP(w, r)
			return
		}

		token := sessionToken(r)
		actor, err := a.auth.Authenticate(r.Context(), token)
		switch {
		case errors.Is(err, auth.ErrUnauthenticated):
			if token != "" {
				// Cookie lama yang sudah tidak berlaku dihapus dari browser.
				w.Header().Add("Set-Cookie", expiredSessionCookie())
			}
			a.writeError(w, r, http.StatusUnauthorized, ErrorCodeUnauthenticated, "Log in to continue.", nil)
			return
		case err != nil:
			a.responseError(w, r, err)
			return
		case actor.MustChangePassword && !passwordChangeRoutes[r.Pattern]:
			a.writeError(w, r, http.StatusForbidden, ErrorCodePasswordChangeRequired,
				"Change your password before you continue.", nil)
			return
		}

		ctx := context.WithValue(r.Context(), sessionKey{}, session{actor: actor, token: token})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggedIn melaporkan apakah permintaan halaman membawa sesi yang berlaku.
// Dipakai penjaga halaman di spa.go. Hanya "belum login" yang dianggap tidak
// login; error lain (misalnya database mati) tetap menampilkan halaman, karena
// halaman itu sendiri tidak berisi data dan setiap data tetap dijaga API.
func (a *apiServer) loggedIn(r *http.Request) bool {
	_, err := a.auth.Authenticate(r.Context(), sessionToken(r))
	return !errors.Is(err, auth.ErrUnauthenticated)
}

// sessionCookie menyusun header Set-Cookie untuk token sesi:
//   - HttpOnly: tidak bisa dibaca JavaScript (ASVS 3.3.4);
//   - Secure: hanya dikirim lewat HTTPS. Browser menganggap localhost dan
//     127.0.0.1 aman, jadi demo localhost tetap jalan (D-36, ASVS 3.3.1);
//   - SameSite=Lax: tidak dikirim pada POST dari situs lain, tapi tetap dikirim
//     saat pengguna membuka tautan ke sistem ini (ASVS 3.3.2);
//   - tanpa Max-Age: cookie hilang saat browser ditutup. Batas waktu sesi
//     tetap dijaga server.
func sessionCookie(token string) string {
	c := &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	}
	return c.String()
}

// expiredSessionCookie menyusun header Set-Cookie yang menghapus cookie sesi.
func expiredSessionCookie() string {
	c := &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	}
	return c.String()
}

// GetSession menjawab GET /api/v1/session: siapa yang sedang login.
func (a *apiServer) GetSession(ctx context.Context, _ GetSessionRequestObject) (GetSessionResponseObject, error) {
	return GetSession200JSONResponse(toAPISession(sessionFrom(ctx).actor)), nil
}

// Login menjawab POST /api/v1/session.
func (a *apiServer) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	actor, token, err := a.auth.Login(ctx, req.Body.Login, req.Body.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		// Nama login dicatat untuk memantau serangan; password tidak pernah dicatat.
		a.log.Info("login gagal", "request_id", requestIDFrom(ctx), "login", req.Body.Login)
		return Login401JSONResponse(errorBody(ctx, ErrorCodeInvalidCredentials,
			"The login name or password is not correct.", nil)), nil
	}
	if err != nil {
		return nil, err
	}
	a.log.Info("login berhasil", "request_id", requestIDFrom(ctx), "login", actor.Login)
	cookie := sessionCookie(token)
	return Login200JSONResponse{Body: toAPISession(actor), Headers: Login200ResponseHeaders{SetCookie: &cookie}}, nil
}

// Logout menjawab DELETE /api/v1/session.
func (a *apiServer) Logout(ctx context.Context, _ LogoutRequestObject) (LogoutResponseObject, error) {
	if err := a.auth.Logout(ctx, sessionFrom(ctx).token); err != nil {
		return nil, err
	}
	cookie := expiredSessionCookie()
	return Logout204Response{Headers: Logout204ResponseHeaders{SetCookie: &cookie}}, nil
}

// ChangePassword menjawab PUT /api/v1/account/password.
func (a *apiServer) ChangePassword(ctx context.Context, req ChangePasswordRequestObject) (ChangePasswordResponseObject, error) {
	token, err := a.auth.ChangePassword(ctx, sessionFrom(ctx).actor, req.Body.CurrentPassword, req.Body.NewPassword)
	if status, body, ok := problemResponse(ctx, err); ok && status == http.StatusUnprocessableEntity {
		return ChangePassword422JSONResponse(body), nil
	}
	if err != nil {
		return nil, err
	}
	cookie := sessionCookie(token)
	return ChangePassword204Response{Headers: ChangePassword204ResponseHeaders{SetCookie: &cookie}}, nil
}

// toAPISession mengubah actor menjadi Session kontrak API.
func toAPISession(actor auth.Actor) Session {
	return Session{
		Account:            Account{Id: actor.AccountID, Login: actor.Login, Kind: AccountKind(actor.AccountKind)},
		Person:             Person{Id: actor.PersonID, Name: actor.PersonName},
		MustChangePassword: actor.MustChangePassword,
	}
}
