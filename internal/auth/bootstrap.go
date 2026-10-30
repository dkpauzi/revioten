package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// adminBootstrapLock adalah nomor kunci advisory PostgreSQL untuk pembuatan
// admin pertama. Nomornya bebas, asal tidak dipakai kunci lain.
const adminBootstrapLock int64 = 11001

// CreateFirstAdmin membuat orang dan akun admin pertama di database yang
// belum punya admin (E-07). Bila admin sudah ada, hasilnya ErrAdminExists.
// Dipanggil dari perintah CLI `revioten create-admin`, bukan dari API, jadi
// tidak ada halaman yang terbuka tanpa login.
func (s *Service) CreateFirstAdmin(ctx context.Context, name, login, password string) (access.Actor, error) {
	name = strings.TrimSpace(name)
	login = normalizeLogin(login)

	var problems []problem.Problem
	if p, ok := checkPersonName("name", name); !ok {
		problems = append(problems, p)
	}
	if p, ok := checkLogin(login); !ok {
		problems = append(problems, p)
	}
	problems = append(problems, checkNewPassword("password", password, login, name)...)
	if len(problems) > 0 {
		return access.Actor{}, &problem.ValidationError{Problems: problems}
	}

	hash, err := hashPassword(password)
	if err != nil {
		return access.Actor{}, err
	}

	var actor access.Actor
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Kunci dipegang sampai transaksi selesai. Perintah kedua yang datang
		// bersamaan menunggu di sini, lalu melihat admin yang sudah dibuat.
		if err := q.LockAdminBootstrap(ctx, adminBootstrapLock); err != nil {
			return err
		}
		exists, err := q.AdminAccountExists(ctx)
		if err != nil {
			return err
		}
		if exists {
			return ErrAdminExists
		}
		person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: name})
		if err != nil {
			return err
		}
		acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: person.ID, Kind: KindAdmin, Login: login, PasswordHash: hash,
		})
		if err != nil {
			return err
		}
		// Akun admin selalu memegang peran Admin, dan hanya itu (D-72, M1.2).
		grants := []access.RoleGrant{{Role: access.RoleAdmin, Scope: access.ScopeAll}}
		if err := access.ReplaceRoles(ctx, q, acc.ID, grants); err != nil {
			return err
		}
		// Belum ada akun yang bisa disebut pelakunya, jadi dicatat dengan
		// aktor sistem: perintah yang dijalankan di komputer server (E-07).
		err = audit.Record(ctx, q, access.Actor{Type: access.ActorSystem}, audit.Event{
			Action: audit.ActionAccountCreated, TargetType: audit.TargetAccount, TargetID: acc.ID,
			Details: map[string]string{"login": acc.Login, "kind": acc.Kind, "roles": describeGrants(grants), "via": "create-admin"},
		})
		if err != nil {
			return err
		}
		actor = access.Actor{
			Type: access.ActorHuman, AccountID: acc.ID, AccountKind: acc.Kind, Login: acc.Login,
			PersonID: person.ID, PersonName: person.Name,
		}
		return nil
	})
	switch {
	case errors.Is(err, ErrAdminExists):
		return access.Actor{}, err
	case db.IsUniqueViolation(err, "accounts_login_key"):
		return access.Actor{}, &problem.ConflictError{Problems: []problem.Problem{{
			Field: "login", Code: problem.Taken, Message: "This login name is already used.",
		}}}
	case err != nil:
		return access.Actor{}, fmt.Errorf("membuat admin pertama: %w", err)
	}
	slog.Info("akun admin pertama dibuat", "login", actor.Login)
	return actor, nil
}

// ResetAdminPassword membuka kembali akun admin yang lupa password, terkunci,
// atau nonaktif (E-22 opsi a, D-101): password sementara dipasang dan wajib
// diganti saat login berikutnya, hitungan gagal dan kunci dihapus, akun
// diaktifkan lagi, dan semua sesinya berakhir. Hasilnya password sementara,
// yang hanya ditampilkan sekali.
//
// Dipanggil dari perintah CLI `revioten reset-admin-password`, yang hanya bisa
// dijalankan di komputer server. Orang yang bisa menjalankannya sudah memegang
// database, jadi perintah ini tidak membuka jalan baru. Peristiwanya dicatat
// dengan aktor sistem, seperti create-admin (E-07). Akun kerja tetap direset
// admin lewat aplikasi (A-1), jadi login yang bukan akun admin ditolak.
func (s *Service) ResetAdminPassword(ctx context.Context, login string) (string, error) {
	acc, err := s.q.GetAccountByLogin(ctx, normalizeLogin(login))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && acc.Kind != KindAdmin) {
		return "", ErrNotAdminAccount
	}
	if err != nil {
		return "", fmt.Errorf("mengambil akun: %w", err)
	}

	temporary := newTemporaryPassword()
	hash, err := hashPassword(temporary)
	if err != nil {
		return "", err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.ResetAccountPassword(ctx, db.ResetAccountPasswordParams{ID: acc.ID, PasswordHash: hash}); err != nil {
			return err
		}
		details := map[string]string{"via": "reset-admin-password"}
		if acc.DisabledAt != nil {
			if err := q.EnableAccount(ctx, acc.ID); err != nil {
				return err
			}
			details["enabled"] = "true"
		}
		if err := q.DeleteAccountSessions(ctx, acc.ID); err != nil {
			return err
		}
		// Password sementara tidak pernah masuk audit log (E-18, D-95).
		return audit.Record(ctx, q, access.Actor{Type: access.ActorSystem}, audit.Event{
			Action: audit.ActionPasswordReset, TargetType: audit.TargetAccount, TargetID: acc.ID, Details: details,
		})
	})
	if err != nil {
		return "", fmt.Errorf("mereset password admin: %w", err)
	}
	slog.Info("password akun admin direset dari komputer server", "login", acc.Login)
	return temporary, nil
}
