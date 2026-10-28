package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
)

// fakePinger meniru database untuk /health. err nil berarti database hidup.
type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

// fakeCustomers meniru customer.Service, jadi test HTTP tidak butuh database.
type fakeCustomers struct {
	list      []customer.Customer
	listErr   error
	created   customer.Customer
	createErr error
	// gotInput adalah input terakhir yang diterima Create.
	gotInput *customer.NewCustomer
}

func (f *fakeCustomers) List(context.Context) ([]customer.Customer, error) {
	return f.list, f.listErr
}

func (f *fakeCustomers) Create(_ context.Context, in customer.NewCustomer) (customer.Customer, error) {
	f.gotInput = &in
	return f.created, f.createErr
}

// testToken adalah token sesi yang diterima fakeAuth secara bawaan.
const testToken = "token-uji"

// testActor adalah pemilik sesi testToken.
var testActor = auth.Actor{
	AccountID: "akun-uji", AccountKind: auth.KindAdmin, Login: "admin.uji",
	PersonID: "orang-uji", PersonName: "Admin Uji",
}

// fakeAuth meniru auth.Service tanpa database. sessions memetakan token ke actor.
type fakeAuth struct {
	sessions map[string]auth.Actor
	// loginToken dan loginActor dikembalikan Login bila password-nya "password-benar".
	loginToken string
	loginActor auth.Actor
	// changeErr dikembalikan ChangePassword bila diisi.
	changeErr error
	// Catatan panggilan, untuk diperiksa test.
	loggedOut     []string
	changeCurrent string
	changeNew     string
}

func newFakeAuth() *fakeAuth {
	return &fakeAuth{sessions: map[string]auth.Actor{testToken: testActor}}
}

func (f *fakeAuth) Login(_ context.Context, login, password string) (auth.Actor, string, error) {
	if login == f.loginActor.Login && password == "password-benar" {
		return f.loginActor, f.loginToken, nil
	}
	return auth.Actor{}, "", auth.ErrInvalidCredentials
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (auth.Actor, error) {
	actor, ok := f.sessions[token]
	if !ok {
		return auth.Actor{}, auth.ErrUnauthenticated
	}
	return actor, nil
}

func (f *fakeAuth) Logout(_ context.Context, token string) error {
	f.loggedOut = append(f.loggedOut, token)
	return nil
}

func (f *fakeAuth) ChangePassword(_ context.Context, _ auth.Actor, current, newPassword string) (string, error) {
	f.changeCurrent, f.changeNew = current, newPassword
	if f.changeErr != nil {
		return "", f.changeErr
	}
	return "token-baru", nil
}

// testServer adalah handler lengkap dengan dependensi palsu. Log server
// ditampung di log, supaya test bisa memeriksa apa yang dicatat.
type testServer struct {
	handler http.Handler
	log     *bytes.Buffer
}

// newTestServer mengisi dependensi yang kosong dengan versi palsu yang sehat.
func newTestServer(t *testing.T, ui fs.FS, deps Deps) testServer {
	t.Helper()
	if ui == nil {
		ui = fstest.MapFS{}
	}
	if deps.DB == nil {
		deps.DB = fakePinger{}
	}
	if deps.Customers == nil {
		deps.Customers = &fakeCustomers{list: []customer.Customer{}}
	}
	if deps.Auth == nil {
		deps.Auth = newFakeAuth()
	}
	var log bytes.Buffer
	deps.Logger = slog.New(slog.NewTextHandler(&log, nil))
	return testServer{handler: NewHandler(ui, deps), log: &log}
}

// do mengirim satu permintaan sebagai pengguna yang sudah login (cookie sesi
// testToken), lalu mengembalikan rekaman jawabannya. header berisi pasangan
// nama dan nilai, misalnya "X-Request-Id", "abc".
func (s testServer) do(method, path, body string, header ...string) *httptest.ResponseRecorder {
	return s.doWithCookie(testToken, method, path, body, header...)
}

// doAnon mengirim permintaan tanpa cookie sesi, seperti pengguna yang belum login.
func (s testServer) doAnon(method, path, body string, header ...string) *httptest.ResponseRecorder {
	return s.doWithCookie("", method, path, body, header...)
}

// doWithCookie mengirim permintaan dengan token sesi tertentu ("" berarti tanpa cookie).
func (s testServer) doWithCookie(token, method, path, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

// decodeError membaca jawaban sebagai Error dari kontrak dan memeriksa
// bagian yang wajib ada di setiap error (R-API-13).
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) Error {
	t.Helper()
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, ingin application/json", got)
	}
	var body Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi error bukan JSON yang valid: %v\n%s", err, rec.Body.String())
	}
	if body.Message == "" {
		t.Error("message kosong")
	}
	if id := rec.Header().Get("X-Request-Id"); body.RequestId == "" || body.RequestId != id {
		t.Errorf("request_id %q, ingin sama dengan header X-Request-Id %q", body.RequestId, id)
	}
	return body
}
