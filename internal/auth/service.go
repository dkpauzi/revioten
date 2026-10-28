package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Service menjalankan login, sesi, dan ganti password terhadap database.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
	// now bisa diganti di test supaya batas waktu bisa dites tanpa menunggu.
	now func() time.Time
	// afterAdminCount hanya diisi test: dipanggil di dalam transaksi
	// DisableAccount setelah jumlah admin aktif dihitung, untuk memaksa dua
	// transaksi berjalan bersamaan (lihat TestDisableLastAdminConcurrent).
	afterAdminCount func()
}

// NewService membuat Service yang memakai pool database. authz memeriksa izin
// setiap operasi yang punya aktor (M1.2).
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, now: time.Now}
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
func (s *Service) Login(ctx context.Context, login, password string) (access.Actor, string, error) {
	acc, err := s.q.GetAccountByLogin(ctx, normalizeLogin(login))
	if errors.Is(err, pgx.ErrNoRows) {
		_, _, _ = verifyPassword(password, dummyHash())
		return access.Actor{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return access.Actor{}, "", fmt.Errorf("mengambil akun: %w", err)
	}

	now := s.now()
	match, needsRehash, err := verifyPassword(password, acc.PasswordHash)
	if err != nil {
		return access.Actor{}, "", fmt.Errorf("memeriksa password akun %s: %w", acc.Login, err)
	}
	switch {
	case acc.DisabledAt != nil:
		return access.Actor{}, "", ErrInvalidCredentials
	case acc.LockedUntil != nil && now.Before(*acc.LockedUntil):
		// Selama terkunci, percobaan selalu ditolak dan tidak dihitung, jadi
		// penyerang paling banyak mendapat 10 tebakan per 15 menit.
		return access.Actor{}, "", ErrInvalidCredentials
	case !match:
		// Pelakunya belum login, jadi dicatat sebagai aktor anonim.
		anonymous := access.Actor{Type: access.ActorAnonymous}
		if err := s.recordFailure(ctx, acc.ID, acc.Login, anonymous, audit.ActionLoginFailed, now); err != nil {
			return access.Actor{}, "", err
		}
		return access.Actor{}, "", ErrInvalidCredentials
	}

	actor := access.Actor{
		Type:               access.ActorHuman,
		AccountID:          acc.ID,
		AccountKind:        acc.Kind,
		Login:              acc.Login,
		PersonID:           acc.PersonID,
		PersonName:         acc.PersonName,
		MustChangePassword: acc.MustChangePassword,
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
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionLogin, TargetType: audit.TargetAccount, TargetID: acc.ID,
		})
	})
	if err != nil {
		return access.Actor{}, "", fmt.Errorf("membuat sesi: %w", err)
	}
	return actor, token, nil
}

// recordFailure menambah hitungan gagal login, lalu mengunci atau
// menonaktifkan akun bila hitungannya mencapai batas. Perubahan itu dan entri
// auditnya (action, oleh actor) disimpan dalam satu transaksi.
func (s *Service) recordFailure(ctx context.Context, accountID, login string, actor access.Actor, action string, now time.Time) error {
	var count int32
	var lockedUntil *time.Time
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		count, err = q.RecordLoginFailure(ctx, accountID)
		if err != nil {
			return err
		}
		details := map[string]string{"failed_logins": strconv.Itoa(int(count))}
		switch {
		case count >= disableThreshold:
			if err := q.DisableAccount(ctx, db.DisableAccountParams{ID: accountID, DisabledAt: &now}); err != nil {
				return err
			}
			details["disabled"] = "true"
		case count%lockThreshold == 0:
			until := now.Add(lockDuration)
			if err := q.LockAccountUntil(ctx, db.LockAccountUntilParams{ID: accountID, LockedUntil: &until}); err != nil {
				return err
			}
			lockedUntil = &until
			details["locked_until"] = until.UTC().Format(time.RFC3339)
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: action, TargetType: audit.TargetAccount, TargetID: accountID, Details: details,
		})
	})
	if err != nil {
		return fmt.Errorf("mencatat gagal login: %w", err)
	}
	// Peringatan di log server untuk yang memantau server; catatan resminya di audit log.
	switch {
	case count >= disableThreshold:
		slog.Warn("akun dinonaktifkan karena gagal login berulang", "login", login, "gagal", count)
	case lockedUntil != nil:
		slog.Warn("akun dikunci sementara karena gagal login berulang", "login", login, "gagal", count, "sampai", *lockedUntil)
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
func (s *Service) Authenticate(ctx context.Context, token string) (access.Actor, error) {
	if token == "" {
		return access.Actor{}, ErrUnauthenticated
	}
	hash := hashToken(token)
	row, err := s.q.GetSession(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.Actor{}, ErrUnauthenticated
	}
	if err != nil {
		return access.Actor{}, fmt.Errorf("mengambil sesi: %w", err)
	}

	now := s.now()
	expired := !now.Before(row.ExpiresAt) || !now.Before(row.LastSeenAt.Add(idleTimeout))
	if expired || row.DisabledAt != nil {
		if err := s.q.DeleteSession(ctx, hash); err != nil {
			return access.Actor{}, fmt.Errorf("menghapus sesi: %w", err)
		}
		return access.Actor{}, ErrUnauthenticated
	}
	if err := s.q.TouchSession(ctx, db.TouchSessionParams{TokenHash: hash, LastSeenAt: now}); err != nil {
		return access.Actor{}, fmt.Errorf("memperbarui sesi: %w", err)
	}
	return access.Actor{
		Type:               access.ActorHuman,
		AccountID:          row.AccountID,
		AccountKind:        row.Kind,
		Login:              row.Login,
		PersonID:           row.PersonID,
		PersonName:         row.PersonName,
		MustChangePassword: row.MustChangePassword,
	}, nil
}

// Logout mengakhiri sesi actor di server (ASVS 7.4.1). Token yang tidak
// dikenal tidak dianggap error.
func (s *Service) Logout(ctx context.Context, actor access.Actor, token string) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: actor.AccountID}); err != nil {
		return err
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteSession(ctx, hashToken(token)); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionLogout, TargetType: audit.TargetAccount, TargetID: actor.AccountID,
		})
	})
	if err != nil {
		return fmt.Errorf("menghapus sesi: %w", err)
	}
	return nil
}

// ChangePassword mengganti password akun actor. Password lama wajib benar
// (ASVS 6.2.3). Setelah berhasil, semua sesi akun itu berakhir dan sesi baru
// dibuat untuk pemanggil (ASVS 7.4.3 dan 7.2.4); token barunya dikembalikan.
func (s *Service) ChangePassword(ctx context.Context, actor access.Actor, current, newPassword string) (string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: actor.AccountID}); err != nil {
		return "", err
	}
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
		if err := s.recordFailure(ctx, acc.ID, acc.Login, actor, audit.ActionPasswordChangeFailed, now); err != nil {
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
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPasswordChanged, TargetType: audit.TargetAccount, TargetID: acc.ID,
		})
	})
	if err != nil {
		return "", fmt.Errorf("menyimpan password baru: %w", err)
	}
	return token, nil
}
