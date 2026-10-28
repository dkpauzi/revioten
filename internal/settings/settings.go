// Package settings berisi pengaturan sistem (UC-07, M1.8): saklar yang
// mengubah perilaku sistem tanpa mengubah kode. Untuk sekarang hanya
// tanggal tanda tangan per slot, yang bisa dimatikan dan disembunyikan, dan
// saat ini mati (D-52, R-APR-14). Saklar QR (D-62), aturan tanggal berlaku
// (R-REV-08), dan mode migrasi menyusul di milestone masing-masing.
package settings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
)

// Settings adalah pengaturan sistem yang berlaku.
type Settings struct {
	// SignDatesEnabled true berarti setiap slot tanda tangan punya tanggal.
	// Bila false, isiannya disembunyikan dan hanya tanggal rilis yang dicatat (D-52).
	SignDatesEnabled bool
}

// Service membaca dan mengubah pengaturan sistem.
type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database. authz memeriksa izin
// setiap operasi (M1.2).
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz}
}

// Get mengembalikan pengaturan yang berlaku. Semua akun boleh membacanya
// (ASUMSI E-24), karena layar dan cetakan mengikutinya.
func (s *Service) Get(ctx context.Context, actor access.Actor) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Settings{}, err
	}
	row, err := s.q.GetSystemSettings(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("membaca pengaturan sistem: %w", err)
	}
	return Settings{SignDatesEnabled: row.SignDatesEnabled}, nil
}

// Update adalah perubahan pengaturan. Saklar yang nil tetap seperti sebelumnya.
type Update struct {
	SignDatesEnabled *bool
}

// Apply mengubah pengaturan sistem (A-3). Hak diperiksa dulu, juga bila
// tidak ada saklar yang dikirim.
func (s *Service) Apply(ctx context.Context, actor access.Actor, in Update) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Settings{}, err
	}
	var after Settings
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		before, err := q.LockSystemSettings(ctx)
		if err != nil {
			return err
		}
		enabled := before.SignDatesEnabled
		if in.SignDatesEnabled != nil {
			enabled = *in.SignDatesEnabled
		}
		after = Settings{SignDatesEnabled: enabled}
		if before.SignDatesEnabled == enabled {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		if err := q.SetSignDates(ctx, enabled); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSystemSettingsUpdated, TargetType: audit.TargetSystemSettings, TargetID: before.ID,
			Details: map[string]string{
				"sign_dates_enabled": audit.Change(strconv.FormatBool(before.SignDatesEnabled), strconv.FormatBool(enabled)),
			},
		})
	})
	if err != nil {
		return Settings{}, fmt.Errorf("mengubah pengaturan sistem: %w", err)
	}
	return after, nil
}
