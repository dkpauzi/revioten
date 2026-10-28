package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

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
func (s *Service) CreateFirstAdmin(ctx context.Context, name, login, password string) (Actor, error) {
	name = strings.TrimSpace(name)
	login = normalizeLogin(login)

	var problems []problem.Problem
	if p, ok := checkPersonName(name); !ok {
		problems = append(problems, p)
	}
	if p, ok := checkLogin(login); !ok {
		problems = append(problems, p)
	}
	problems = append(problems, checkNewPassword("password", password, login, name)...)
	if len(problems) > 0 {
		return Actor{}, &problem.ValidationError{Problems: problems}
	}

	hash, err := hashPassword(password)
	if err != nil {
		return Actor{}, err
	}

	var actor Actor
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
		actor = Actor{AccountID: acc.ID, AccountKind: acc.Kind, Login: acc.Login, PersonID: person.ID, PersonName: person.Name}
		return nil
	})
	switch {
	case errors.Is(err, ErrAdminExists):
		return Actor{}, err
	case db.IsUniqueViolation(err, "accounts_login_key"):
		return Actor{}, &problem.ConflictError{Problems: []problem.Problem{{
			Field: "login", Code: problem.Taken, Message: "This login name is already used.",
		}}}
	case err != nil:
		return Actor{}, fmt.Errorf("membuat admin pertama: %w", err)
	}
	// M1.3: peristiwa ini juga dicatat di audit log dengan aktor sistem (E-07).
	slog.Info("akun admin pertama dibuat", "login", actor.Login)
	return actor, nil
}
