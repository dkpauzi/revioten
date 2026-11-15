// Package integration berisi aturan API integrasi (Tahap 9): service account
// dan token (M9.1), bacaan data rilis (M9.3), usulan perubahan dari aplikasi
// lain (M9.4), dan ekspor asinkron (M9.5).
//
// Aktornya bukan orang, melainkan aplikasi lain (access.ActorService). Izinnya
// tidak lewat peran, tetapi lewat dua hal yang melekat pada token: scope
// (misalnya documents:read) dan daftar customer yang diizinkan (R-API-05).
// Customer di luar daftar dijawab "tidak ada", supaya keberadaannya tidak bocor.
//
// Aplikasi lain tidak pernah bisa memverifikasi, menyetujui, atau merilis:
// paket ini memang tidak punya operasi itu (R-API-07, RK-26).
package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
)

// Scope yang dikenal (R-API-05). Bentuknya resource:action.
const (
	ScopeDocumentsRead  = "documents:read"
	ScopeRevisionsRead  = "revisions:read"
	ScopeChangesRead    = "changes:read"
	ScopeChangesPropose = "changes:propose"
	ScopeExportsCreate  = "exports:create"
)

// Scopes adalah semua scope yang bisa diberikan, menurut urutan tampil.
var Scopes = []string{ScopeDocumentsRead, ScopeRevisionsRead, ScopeChangesRead, ScopeChangesPropose, ScopeExportsCreate}

// ErrUnauthenticated berarti token tidak ada, salah, kedaluwarsa, dicabut,
// atau datang dari jaringan yang tidak diizinkan. Pemanggil tidak diberi tahu
// yang mana.
var ErrUnauthenticated = errors.New("token integrasi tidak berlaku")

const (
	tokenPrefix     = "rvt"
	defaultTokenAge = 90 // hari (R-API-04)
	maxTokenAge     = 365
)

// Account adalah satu service account beserta tokennya.
type Account struct {
	ID            string
	Name          string
	OwnerTeam     string
	Contact       string
	Purpose       string
	Scopes        []string
	CustomerIDs   []string
	CustomerCodes []string
	Networks      []string // CIDR; kosong berarti semua alamat
	Disabled      bool
	CreatedAt     time.Time
	Tokens        []Token // yang terbaru dulu; tidak pernah nil
}

// Token adalah keterangan satu token. Nilai tokennya sendiri tidak disimpan.
type Token struct {
	ID         string
	Prefix     string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// NewAccount adalah isian service account baru.
type NewAccount struct {
	Name        string
	OwnerTeam   string
	Contact     string
	Purpose     string
	Scopes      []string
	CustomerIDs []string
	Networks    []string
}

// Client adalah aplikasi yang sudah dikenali dari tokennya.
type Client struct {
	ServiceAccountID string
	Name             string
	TokenID          string
	TokenPrefix      string
	Scopes           []string
	CustomerIDs      []string
}

// actor memberi aktor audit untuk aplikasi ini.
func (c Client) actor() access.Actor {
	return access.Actor{Type: access.ActorService, Login: c.Name}
}

// details menambahkan service account dan awalan token ke rincian audit (R-API-03).
func (c Client) details(extra map[string]string) map[string]string {
	out := map[string]string{"service_account": c.Name, "token": c.TokenPrefix}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// require memeriksa scope token.
func (c Client) require(scope string) error {
	if !slices.Contains(c.Scopes, scope) {
		return access.ErrForbidden
	}
	return nil
}

// allows melaporkan apakah customer itu termasuk yang diizinkan untuk token.
func (c Client) allows(customerID string) bool {
	return slices.Contains(c.CustomerIDs, customerID)
}

// Service menjalankan aturan API integrasi.
type Service struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	authz   access.Authorizer
	store   *storage.Store
	printer *printout.Service
	now     func() time.Time
}

// NewService membuat Service. store dan printer dipakai ekspor; keduanya
// boleh nil bila ekspor tidak dipakai.
func NewService(pool *pgxpool.Pool, authz access.Authorizer, store *storage.Store, printer *printout.Service) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, store: store, printer: printer, now: time.Now}
}

// WithClock memberi salinan Service yang memakai jam now. Dipakai test
// untuk memeriksa token kedaluwarsa tanpa menunggu.
func (s *Service) WithClock(now func() time.Time) *Service {
	copy := *s
	copy.now = now
	return &copy
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

// hashToken memberi SHA-256 token. Token berisi 256 bit acak, jadi hash
// cepat tanpa garam sudah cukup; ini berbeda dari password buatan orang.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newToken membuat token acak: rvt_<awalan 8 huruf>_<256 bit acak>.
func newToken() (token, prefix string, err error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", "", err
	}
	id := make([]byte, 5)
	if _, err := rand.Read(id); err != nil {
		return "", "", err
	}
	prefix = strings.ToLower(base64.RawURLEncoding.EncodeToString(id))
	prefix = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return 'x'
	}, prefix+"00000000")[:8]
	return tokenPrefix + "_" + prefix + "_" + base64.RawURLEncoding.EncodeToString(secret), prefix, nil
}

// CreateAccount membuat service account (A-1). Token diterbitkan terpisah.
func (s *Service) CreateAccount(ctx context.Context, actor access.Actor, in NewAccount) (Account, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return Account{}, err
	}
	var list []problem.Problem
	text := func(field, value string, max int) string {
		clean, p := problem.Text(field, value, true, max)
		if p != nil {
			list = append(list, *p)
		}
		return clean
	}
	name := text("name", in.Name, 100)
	team := text("owner_team", in.OwnerTeam, 100)
	contact := text("contact", in.Contact, 200)
	purpose := text("purpose", in.Purpose, 500)
	if len(in.Scopes) == 0 {
		list = append(list, problem.Problem{Field: "scopes", Code: problem.Required, Message: "Choose at least one scope."})
	}
	for _, sc := range in.Scopes {
		if !slices.Contains(Scopes, sc) {
			list = append(list, problem.Problem{Field: "scopes", Code: problem.InvalidFormat, Message: "Unknown scope: " + sc + "."})
			break
		}
	}
	if len(in.CustomerIDs) == 0 {
		list = append(list, problem.Problem{Field: "customer_ids", Code: problem.Required, Message: "Choose at least one customer."})
	}
	for _, c := range in.CustomerIDs {
		if !db.IsUUID(c) {
			list = append(list, problem.Problem{Field: "customer_ids", Code: problem.InvalidFormat, Message: "Choose existing customers."})
			break
		}
	}
	networks := make([]string, 0, len(in.Networks))
	for _, n := range in.Networks {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(n)
		if err != nil {
			list = append(list, problem.Problem{Field: "networks", Code: problem.InvalidFormat, Message: "Write each network as CIDR, for example 192.168.10.0/24."})
			break
		}
		networks = append(networks, prefix.Masked().String())
	}
	if len(list) > 0 {
		return Account{}, &problem.ValidationError{Problems: list}
	}
	scopes := make([]string, 0, len(in.Scopes))
	for _, sc := range Scopes { // urutan tetap, tanpa ganda
		if slices.Contains(in.Scopes, sc) {
			scopes = append(scopes, sc)
		}
	}

	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		taken, err := q.ServiceAccountNameExists(ctx, name)
		if err != nil {
			return err
		}
		if taken {
			return &problem.ConflictError{Problems: []problem.Problem{{Field: "name", Code: problem.Taken, Message: "A service account with this name already exists."}}}
		}
		id, err = q.CreateServiceAccount(ctx, db.CreateServiceAccountParams{
			Name: name, OwnerTeam: team, Contact: contact, Purpose: purpose, Scopes: scopes, AllowedNetworks: networks, CreatedByPersonID: actor.PersonID,
		})
		if err != nil {
			return err
		}
		for _, c := range slices.Compact(slices.Sorted(slices.Values(in.CustomerIDs))) {
			if err := q.AddServiceAccountCustomer(ctx, db.AddServiceAccountCustomerParams{ServiceAccountID: id, CustomerID: c}); err != nil {
				var pg *pgconn.PgError
				if errors.As(err, &pg) && pg.Code == "23503" { // foreign_key_violation: customer tidak ada
					return invalid("customer_ids", problem.InvalidFormat, "Choose existing customers.")
				}
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionServiceAccountCreated,
			Details: map[string]string{"service_account": name, "scopes": strings.Join(scopes, ", "), "customers": strconv.Itoa(len(in.CustomerIDs))}})
	})
	if err != nil {
		return Account{}, pass(err, "membuat service account")
	}
	return s.account(ctx, id)
}

// pass meneruskan penolakan apa adanya dan membungkus error lain.
func pass(err error, doing string) error {
	if err == nil {
		return nil
	}
	var v *problem.ValidationError
	var c *problem.ConflictError
	if errors.As(err, &v) || errors.As(err, &c) || errors.Is(err, problem.ErrNotFound) || errors.Is(err, access.ErrForbidden) || errors.Is(err, ErrUnauthenticated) {
		return err
	}
	return fmt.Errorf("%s: %w", doing, err)
}

func (s *Service) account(ctx context.Context, id string) (Account, error) {
	if !db.IsUUID(id) {
		return Account{}, problem.ErrNotFound
	}
	r, err := s.q.GetServiceAccount(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, problem.ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("membaca service account: %w", err)
	}
	a := Account{ID: r.ID, Name: r.Name, OwnerTeam: r.OwnerTeam, Contact: r.Contact, Purpose: r.Purpose, Scopes: r.Scopes,
		CustomerIDs: r.CustomerIds, CustomerCodes: r.CustomerCodes, Networks: r.AllowedNetworks, Disabled: r.DisabledAt != nil, CreatedAt: r.CreatedAt.UTC(), Tokens: []Token{}}
	tokens, err := s.q.ListServiceTokens(ctx)
	if err != nil {
		return Account{}, err
	}
	for _, t := range tokens {
		if t.ServiceAccountID == id {
			a.Tokens = append(a.Tokens, Token{ID: t.ID, Prefix: t.Prefix, ExpiresAt: t.ExpiresAt.UTC(), RevokedAt: t.RevokedAt, LastUsedAt: t.LastUsedAt, CreatedAt: t.CreatedAt.UTC()})
		}
	}
	return a, nil
}

// ListAccounts memberi semua service account beserta tokennya (A-1).
func (s *Service) ListAccounts(ctx context.Context, actor access.Actor) ([]Account, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListServiceAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca service account: %w", err)
	}
	tokens, err := s.q.ListServiceTokens(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		a := Account{ID: r.ID, Name: r.Name, OwnerTeam: r.OwnerTeam, Contact: r.Contact, Purpose: r.Purpose, Scopes: r.Scopes,
			CustomerIDs: r.CustomerIds, CustomerCodes: r.CustomerCodes, Networks: r.AllowedNetworks, Disabled: r.DisabledAt != nil, CreatedAt: r.CreatedAt.UTC(), Tokens: []Token{}}
		for _, t := range tokens {
			if t.ServiceAccountID == r.ID {
				a.Tokens = append(a.Tokens, Token{ID: t.ID, Prefix: t.Prefix, ExpiresAt: t.ExpiresAt.UTC(), RevokedAt: t.RevokedAt, LastUsedAt: t.LastUsedAt, CreatedAt: t.CreatedAt.UTC()})
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// IssueToken menerbitkan token baru untuk satu service account (A-1). Nilai
// token dikembalikan sekali ini saja; yang disimpan hanya hash-nya. Rotasi
// berarti menerbitkan token baru, memindahkan aplikasinya, lalu mencabut
// token lama: selama itu keduanya berlaku (R-API-04).
func (s *Service) IssueToken(ctx context.Context, actor access.Actor, accountID string, days int) (string, Token, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return "", Token{}, err
	}
	if days == 0 {
		days = defaultTokenAge
	}
	if days < 1 || days > maxTokenAge {
		return "", Token{}, invalid("days", problem.InvalidFormat, fmt.Sprintf("Choose a lifetime from 1 to %d days.", maxTokenAge))
	}
	a, err := s.account(ctx, accountID)
	if err != nil {
		return "", Token{}, err
	}
	if a.Disabled {
		return "", Token{}, invalid("service_account", problem.NotAllowed, "This service account is disabled.")
	}
	token, prefix, err := newToken()
	if err != nil {
		return "", Token{}, fmt.Errorf("membuat token: %w", err)
	}
	expires := s.now().Add(time.Duration(days) * 24 * time.Hour).UTC()
	var id string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if id, err = q.CreateServiceToken(ctx, db.CreateServiceTokenParams{
			ServiceAccountID: a.ID, Prefix: prefix, TokenHash: hashToken(token), ExpiresAt: expires, CreatedByPersonID: actor.PersonID,
		}); err != nil {
			return err
		}
		// Nilai token tidak pernah masuk audit log (R-SEC-14); hanya awalannya.
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionServiceTokenIssued,
			Details: map[string]string{"service_account": a.Name, "token": prefix, "expires_at": expires.Format(time.RFC3339)}})
	})
	if err != nil {
		return "", Token{}, pass(err, "menerbitkan token")
	}
	return token, Token{ID: id, Prefix: prefix, ExpiresAt: expires, CreatedAt: s.now().UTC()}, nil
}

// RevokeToken mencabut satu token (A-1). Token yang dicabut langsung ditolak.
func (s *Service) RevokeToken(ctx context.Context, actor access.Actor, accountID, tokenID string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return err
	}
	a, err := s.account(ctx, accountID)
	if err != nil {
		return err
	}
	if !db.IsUUID(tokenID) {
		return problem.ErrNotFound
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		prefix, err := q.RevokeServiceToken(ctx, db.RevokeServiceTokenParams{ID: tokenID, ServiceAccountID: a.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionServiceTokenRevoked, Details: map[string]string{"service_account": a.Name, "token": prefix}})
	})
	return pass(err, "mencabut token")
}

// DisableAccount menonaktifkan service account (A-1): semua tokennya ditolak.
// Seperti akun orang, service account tidak pernah dihapus (D-105).
func (s *Service) DisableAccount(ctx context.Context, actor access.Actor, accountID string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return err
	}
	a, err := s.account(ctx, accountID)
	if err != nil {
		return err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.DisableServiceAccount(ctx, a.ID)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionServiceAccountDisabled, Details: map[string]string{"service_account": a.Name}})
	})
	return pass(err, "menonaktifkan service account")
}

// Authenticate mengenali aplikasi dari tokennya. ip adalah alamat asal
// permintaan. Token yang salah bentuk, tidak dikenal, kedaluwarsa, dicabut,
// milik akun nonaktif, atau datang dari luar jaringan yang diizinkan selalu
// ditolak dengan ErrUnauthenticated (kriteria M9.1).
func (s *Service) Authenticate(ctx context.Context, token, ip string) (Client, error) {
	if !strings.HasPrefix(token, tokenPrefix+"_") || len(token) > 200 {
		return Client{}, ErrUnauthenticated
	}
	sum := hashToken(token)
	r, err := s.q.FindServiceToken(ctx, sum)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrUnauthenticated
	}
	if err != nil {
		return Client{}, fmt.Errorf("mencari token: %w", err)
	}
	// Perbandingan waktu tetap, walau pencarian sudah lewat indeks hash.
	if subtle.ConstantTimeCompare([]byte(sum), []byte(hashToken(token))) != 1 {
		return Client{}, ErrUnauthenticated
	}
	client := Client{ServiceAccountID: r.ServiceAccountID, Name: r.Name, TokenID: r.ID, TokenPrefix: r.Prefix, Scopes: r.Scopes, CustomerIDs: r.CustomerIds}
	reason := ""
	switch {
	case r.RevokedAt != nil:
		reason = "revoked"
	case !s.now().Before(r.ExpiresAt):
		reason = "expired"
	case r.DisabledAt != nil:
		reason = "disabled"
	case !networkAllowed(r.AllowedNetworks, ip):
		reason = "network"
	}
	if reason != "" {
		// Penolakan token yang dikenal dicatat, supaya penyalahgunaan terlihat.
		err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			return audit.Record(ctx, s.q.WithTx(tx), client.actor(), audit.Event{Action: audit.ActionServiceTokenRefused, Details: client.details(map[string]string{"reason": reason})})
		})
		if err != nil {
			return Client{}, fmt.Errorf("mencatat penolakan token: %w", err)
		}
		return Client{}, ErrUnauthenticated
	}
	if err := s.q.TouchServiceToken(ctx, r.ID); err != nil {
		return Client{}, err
	}
	return client, nil
}

// networkAllowed melaporkan apakah ip termasuk salah satu jaringan. Daftar
// kosong berarti semua alamat boleh.
func networkAllowed(networks []string, ip string) bool {
	if len(networks) == 0 {
		return true
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, n := range networks {
		if prefix, err := netip.ParsePrefix(n); err == nil && prefix.Contains(addr.Unmap()) {
			return true
		}
	}
	return false
}
