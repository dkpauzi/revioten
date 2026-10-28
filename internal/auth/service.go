package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan login, sesi, dan ganti password terhadap database.
type Service struct {
	pool *pgxpool.Pool
	q    *db.Queries
	// now bisa diganti di test supaya batas waktu bisa dites tanpa menunggu.
	now func() time.Time
}

// NewService membuat Service yang memakai pool database.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: db.New(pool), now: time.Now}
}

// dummyHash dipakai saat nama login tidak ada: hash tetap dihitung, supaya
// lama proses sama dengan login yang ada, dan waktu jawab tidak membocorkan
// nama login mana yang terdaftar.
var dummyHash = sync.OnceValue(func() string {
	hash, err := hashPassword("password pengganti yang tidak pernah dipakai")
	if err != nil {
		panic(err) // hanya gagal bila crypto/rand rusak
	}
	return hash
})

// Login memeriksa nama login dan password, lalu membuat sesi baru. Hasilnya
// actor dan token sesi untuk cookie. Semua kegagalan memberi ErrInvalidCredentials.
func (s *Service) Login(ctx context.Context, login, password string) (Actor, string, error) {
	acc, err := s.q.GetAccountByLogin(ctx, normalizeLogin(login))
	if errors.Is(err, pgx.ErrNoRows) {
		_, _, _ = verifyPassword(password, dummyHash())
		return Actor{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return Actor{}, "", fmt.Errorf("mengambil akun: %w", err)
	}

	now := s.now()
	match, needsRehash, err := verifyPassword(password, acc.PasswordHash)
	if err != nil {
		return Actor{}, "", fmt.Errorf("memeriksa password akun %s: %w", acc.Login, err)
	}
	switch {
	case acc.DisabledAt != nil:
		return Actor{}, "", ErrInvalidCredentials
	case acc.LockedUntil != nil && now.Before(*acc.LockedUntil):
		// Selama terkunci, percobaan selalu ditolak dan tidak dihitung, jadi
		// penyerang paling banyak mendapat 10 tebakan per 15 menit.
		return Actor{}, "", ErrInvalidCredentials
	case !match:
		if err := s.recordFailure(ctx, acc.ID, acc.Login, now); err != nil {
			return Actor{}, "", err
		}
		return Actor{}, "", ErrInvalidCredentials
	}

	var token string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.RecordLoginSuccess(ctx, acc.ID); err != nil {
			return err
		}
		if needsRehash {
			// Parameter argon2id sudah dinaikkan sejak hash ini dibuat.
			hash, err := hashPassword(password)
			if err != nil {
				return err
			}
			err = q.SetPasswordHash(ctx, db.SetPasswordHashParams{
				ID: acc.ID, PasswordHash: hash, MustChangePassword: acc.MustChangePassword,
			})
			if err != nil {
				return err
			}
		}
		// Sesi yang sudah lewat batas dibuang sekalian, supaya tabel tidak terus membesar.
		err := q.DeleteExpiredSessions(ctx, db.DeleteExpiredSessionsParams{
			ExpiresAt: now, LastSeenAt: now.Add(-idleTimeout),
		})
		if err != nil {
			return err
		}
		token, err = startSession(ctx, q, acc.ID, now)
		return err
	})
	if err != nil {
		return Actor{}, "", fmt.Errorf("membuat sesi: %w", err)
	}
	return Actor{
		AccountID:          acc.ID,
		AccountKind:        acc.Kind,
		Login:              acc.Login,
		PersonID:           acc.PersonID,
		PersonName:         acc.PersonName,
		MustChangePassword: acc.MustChangePassword,
	}, token, nil
}

// recordFailure menambah hitungan gagal login, lalu mengunci atau
// menonaktifkan akun bila hitungannya mencapai batas.
func (s *Service) recordFailure(ctx context.Context, accountID, login string, now time.Time) error {
	count, err := s.q.RecordLoginFailure(ctx, accountID)
	if err != nil {
		return fmt.Errorf("mencatat gagal login: %w", err)
	}
	switch {
	case count >= disableThreshold:
		err = s.q.DisableAccount(ctx, db.DisableAccountParams{ID: accountID, DisabledAt: &now})
		slog.Warn("akun dinonaktifkan karena gagal login berulang", "login", login, "gagal", count)
	case count%lockThreshold == 0:
		until := now.Add(lockDuration)
		err = s.q.LockAccountUntil(ctx, db.LockAccountUntilParams{ID: accountID, LockedUntil: &until})
		slog.Warn("akun dikunci sementara karena gagal login berulang", "login", login, "gagal", count, "sampai", until)
	}
	if err != nil {
		return fmt.Errorf("mengunci akun: %w", err)
	}
	return nil
}

// startSession membuat sesi baru untuk akun dan mengembalikan tokennya.
func startSession(ctx context.Context, q *db.Queries, accountID string, now time.Time) (string, error) {
	token, hash, err := newToken()
	if err != nil {
		return "", err
	}
	err = q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash, AccountID: accountID, CreatedAt: now, ExpiresAt: now.Add(absoluteTimeout),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// Authenticate mencari sesi dari token cookie. Sesi yang melewati batas tidak
// aktif atau batas mutlak, atau milik akun yang dinonaktifkan, dihapus dan
// dijawab ErrUnauthenticated. Sesi yang berlaku diperpanjang batas tidak aktifnya.
func (s *Service) Authenticate(ctx context.Context, token string) (Actor, error) {
	if token == "" {
		return Actor{}, ErrUnauthenticated
	}
	hash := hashToken(token)
	row, err := s.q.GetSession(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrUnauthenticated
	}
	if err != nil {
		return Actor{}, fmt.Errorf("mengambil sesi: %w", err)
	}

	now := s.now()
	expired := !now.Before(row.ExpiresAt) || !now.Before(row.LastSeenAt.Add(idleTimeout))
	if expired || row.DisabledAt != nil {
		if err := s.q.DeleteSession(ctx, hash); err != nil {
			return Actor{}, fmt.Errorf("menghapus sesi: %w", err)
		}
		return Actor{}, ErrUnauthenticated
	}
	if err := s.q.TouchSession(ctx, db.TouchSessionParams{TokenHash: hash, LastSeenAt: now}); err != nil {
		return Actor{}, fmt.Errorf("memperbarui sesi: %w", err)
	}
	return Actor{
		AccountID:          row.AccountID,
		AccountKind:        row.Kind,
		Login:              row.Login,
		PersonID:           row.PersonID,
		PersonName:         row.PersonName,
		MustChangePassword: row.MustChangePassword,
	}, nil
}

// Logout mengakhiri sesi di server (ASVS 7.4.1). Token yang tidak dikenal
// tidak dianggap error.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.q.DeleteSession(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("menghapus sesi: %w", err)
	}
	return nil
}

// ChangePassword mengganti password akun actor. Password lama wajib benar
// (ASVS 6.2.3). Setelah berhasil, semua sesi akun itu berakhir dan sesi baru
// dibuat untuk pemanggil (ASVS 7.4.3 dan 7.2.4); token barunya dikembalikan.
func (s *Service) ChangePassword(ctx context.Context, actor Actor, current, newPassword string) (string, error) {
	acc, err := s.q.GetAccountByID(ctx, actor.AccountID)
	if err != nil {
		return "", fmt.Errorf("mengambil akun: %w", err)
	}
	now := s.now()

	var problems []problem.Problem
	match, _, err := verifyPassword(current, acc.PasswordHash)
	if err != nil {
		return "", fmt.Errorf("memeriksa password akun %s: %w", acc.Login, err)
	}
	if !match {
		// Dihitung seperti gagal login, supaya sesi yang dicuri tidak bisa dipakai
		// untuk menebak password tanpa batas.
		if err := s.recordFailure(ctx, acc.ID, acc.Login, now); err != nil {
			return "", err
		}
		problems = append(problems, problem.Problem{
			Field: "current_password", Code: problem.Incorrect, Message: "The current password is not correct.",
		})
	}
	problems = append(problems, checkNewPassword("new_password", newPassword, acc.Login, acc.PersonName)...)
	if len(problems) > 0 {
		return "", &problem.ValidationError{Problems: problems}
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return "", err
	}
	var token string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		err := q.SetPasswordHash(ctx, db.SetPasswordHashParams{ID: acc.ID, PasswordHash: hash, MustChangePassword: false})
		if err != nil {
			return err
		}
		if err := q.RecordLoginSuccess(ctx, acc.ID); err != nil {
			return err
		}
		if err := q.DeleteAccountSessions(ctx, acc.ID); err != nil {
			return err
		}
		token, err = startSession(ctx, q, acc.ID, now)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("menyimpan password baru: %w", err)
	}
	return token, nil
}
