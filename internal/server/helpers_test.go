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

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/product"
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
	// gotInput dan gotActor adalah input dan aktor terakhir yang diterima.
	gotInput *customer.NewCustomer
	gotActor access.Actor
	// gotRequest adalah asal permintaan yang sampai ke layanan (untuk audit log).
	gotRequest audit.Request
	// Untuk operasi ubah, ganti kode, dan Project Group.
	groups    []customer.ProjectGroup
	group     customer.ProjectGroup
	gotID     string
	gotUpdate *customer.CustomerUpdate
	gotCode   string
	gotGroup  *customer.ProjectGroupInput
}

func (f *fakeCustomers) List(ctx context.Context, actor access.Actor) ([]customer.Customer, error) {
	f.gotActor = actor
	f.gotRequest = audit.RequestFrom(ctx)
	return f.list, f.listErr
}

func (f *fakeCustomers) Create(_ context.Context, actor access.Actor, in customer.NewCustomer) (customer.Customer, error) {
	f.gotActor = actor
	f.gotInput = &in
	return f.created, f.createErr
}

func (f *fakeCustomers) Update(_ context.Context, actor access.Actor, id string, in customer.CustomerUpdate) (customer.Customer, error) {
	f.gotActor, f.gotID, f.gotUpdate = actor, id, &in
	return f.created, f.createErr
}

func (f *fakeCustomers) ChangeCode(_ context.Context, actor access.Actor, id, code string) (customer.Customer, error) {
	f.gotActor, f.gotID, f.gotCode = actor, id, code
	return f.created, f.createErr
}

func (f *fakeCustomers) ListProjectGroups(_ context.Context, actor access.Actor) ([]customer.ProjectGroup, error) {
	f.gotActor = actor
	return f.groups, f.listErr
}

func (f *fakeCustomers) CreateProjectGroup(_ context.Context, actor access.Actor, in customer.ProjectGroupInput) (customer.ProjectGroup, error) {
	f.gotActor, f.gotGroup = actor, &in
	return f.group, f.createErr
}

func (f *fakeCustomers) UpdateProjectGroup(_ context.Context, actor access.Actor, id string, in customer.ProjectGroupInput) (customer.ProjectGroup, error) {
	f.gotActor, f.gotID, f.gotGroup = actor, id, &in
	return f.group, f.createErr
}

// fakeProducts meniru product.Service. Setiap method mencatat argumennya di
// got, lalu mengembalikan nilai yang sudah diisi test.
type fakeProducts struct {
	groups []product.ModelGroup
	group  product.ModelGroup
	models []product.Model
	model  product.Model
	parts  []product.PartNumber
	part   product.PartNumber
	err    error
	// got berisi argumen panggilan terakhir: nama method, ID, dan input.
	got struct {
		method string
		id     string
		input  any
	}
}

func (f *fakeProducts) record(method, id string, input any) {
	f.got.method, f.got.id, f.got.input = method, id, input
}

func (f *fakeProducts) ListModelGroups(_ context.Context, _ access.Actor, customerID string) ([]product.ModelGroup, error) {
	f.record("ListModelGroups", customerID, nil)
	return f.groups, f.err
}

func (f *fakeProducts) CreateModelGroup(_ context.Context, _ access.Actor, in product.NewModelGroup) (product.ModelGroup, error) {
	f.record("CreateModelGroup", "", in)
	return f.group, f.err
}

func (f *fakeProducts) RenameModelGroup(_ context.Context, _ access.Actor, id, name string) (product.ModelGroup, error) {
	f.record("RenameModelGroup", id, name)
	return f.group, f.err
}

func (f *fakeProducts) ListModels(_ context.Context, _ access.Actor, customerID string) ([]product.Model, error) {
	f.record("ListModels", customerID, nil)
	return f.models, f.err
}

func (f *fakeProducts) CreateModel(_ context.Context, _ access.Actor, in product.NewModel) (product.Model, error) {
	f.record("CreateModel", "", in)
	return f.model, f.err
}

func (f *fakeProducts) UpdateModel(_ context.Context, _ access.Actor, id string, in product.ModelUpdate) (product.Model, error) {
	f.record("UpdateModel", id, in)
	return f.model, f.err
}

func (f *fakeProducts) ListPartNumbers(_ context.Context, _ access.Actor, customerID string) ([]product.PartNumber, error) {
	f.record("ListPartNumbers", customerID, nil)
	return f.parts, f.err
}

func (f *fakeProducts) CreatePartNumber(_ context.Context, _ access.Actor, in product.NewPartNumber) (product.PartNumber, error) {
	f.record("CreatePartNumber", "", in)
	return f.part, f.err
}

func (f *fakeProducts) UpdatePartNumber(_ context.Context, _ access.Actor, id string, in product.PartNumberUpdate) (product.PartNumber, error) {
	f.record("UpdatePartNumber", id, in)
	return f.part, f.err
}

// testToken adalah token sesi yang diterima fakeAuth secara bawaan.
const testToken = "token-uji"

// testActor adalah pemilik sesi testToken.
var testActor = access.Actor{
	Type: access.ActorHuman, AccountID: "akun-uji", AccountKind: access.AccountAdmin, Login: "admin.uji",
	PersonID: "orang-uji", PersonName: "Admin Uji",
}

// fakeAuth meniru auth.Service tanpa database. sessions memetakan token ke actor.
type fakeAuth struct {
	sessions map[string]access.Actor
	// loginToken dan loginActor dikembalikan Login bila password-nya "password-benar".
	loginToken string
	loginActor access.Actor
	// changeErr dikembalikan ChangePassword bila diisi.
	changeErr error
	// Hasil dan error untuk operasi akun (A-1).
	accounts  []auth.AccountInfo
	account   auth.AccountInfo
	temporary string
	adminErr  error
	// Catatan panggilan, untuk diperiksa test.
	loggedOut     []string
	changeCurrent string
	changeNew     string
	newAccount    *auth.NewAccount
	targetAccount string
	setGrants     []access.RoleGrant
}

func newFakeAuth() *fakeAuth {
	return &fakeAuth{sessions: map[string]access.Actor{testToken: testActor}}
}

func (f *fakeAuth) Login(_ context.Context, login, password string) (access.Actor, string, error) {
	if login == f.loginActor.Login && password == "password-benar" {
		return f.loginActor, f.loginToken, nil
	}
	return access.Actor{}, "", auth.ErrInvalidCredentials
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (access.Actor, error) {
	actor, ok := f.sessions[token]
	if !ok {
		return access.Actor{}, auth.ErrUnauthenticated
	}
	return actor, nil
}

func (f *fakeAuth) Logout(_ context.Context, _ access.Actor, token string) error {
	f.loggedOut = append(f.loggedOut, token)
	return nil
}

func (f *fakeAuth) ChangePassword(_ context.Context, _ access.Actor, current, newPassword string) (string, error) {
	f.changeCurrent, f.changeNew = current, newPassword
	if f.changeErr != nil {
		return "", f.changeErr
	}
	return "token-baru", nil
}

func (f *fakeAuth) ListAccounts(context.Context, access.Actor) ([]auth.AccountInfo, error) {
	return f.accounts, f.adminErr
}

func (f *fakeAuth) GetAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) CreateAccount(_ context.Context, _ access.Actor, in auth.NewAccount) (auth.AccountInfo, string, error) {
	f.newAccount = &in
	return f.account, f.temporary, f.adminErr
}

func (f *fakeAuth) ResetPassword(_ context.Context, _ access.Actor, accountID string) (string, error) {
	f.targetAccount = accountID
	return f.temporary, f.adminErr
}

func (f *fakeAuth) DisableAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) EnableAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) SetRoles(_ context.Context, _ access.Actor, accountID string, grants []access.RoleGrant) error {
	f.targetAccount, f.setGrants = accountID, grants
	return f.adminErr
}

// fakeAccess meniru access.Checker untuk hak dan katalog peran.
type fakeAccess struct {
	permissions []access.Permission
	roles       []access.Role
	err         error
}

func (f *fakeAccess) Permissions(context.Context, access.Actor) ([]access.Permission, error) {
	return f.permissions, f.err
}

func (f *fakeAccess) ListRoles(context.Context, access.Actor) ([]access.Role, error) {
	return f.roles, f.err
}

// fakeAudit meniru audit.Service tanpa database.
type fakeAudit struct {
	entries      []audit.Entry
	verification audit.Verification
	err          error
	// Catatan panggilan List.
	gotBefore int64
	gotLimit  int
}

func (f *fakeAudit) List(_ context.Context, _ access.Actor, before int64, limit int) ([]audit.Entry, error) {
	f.gotBefore, f.gotLimit = before, limit
	return f.entries, f.err
}

func (f *fakeAudit) Verify(context.Context, access.Actor) (audit.Verification, error) {
	return f.verification, f.err
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
	if deps.Access == nil {
		deps.Access = &fakeAccess{permissions: []access.Permission{access.ViewReleased, access.ManageUsers}}
	}
	if deps.Audit == nil {
		deps.Audit = &fakeAudit{}
	}
	if deps.Products == nil {
		deps.Products = &fakeProducts{}
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
