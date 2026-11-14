// Package settings berisi pengaturan sistem (UC-07, M1.8): saklar yang
// mengubah perilaku sistem tanpa mengubah kode. Untuk sekarang hanya
// tanggal tanda tangan per slot, yang bisa dimatikan dan disembunyikan, dan
// saat ini mati (D-52, R-APR-14). Saklar QR (D-62), aturan tanggal berlaku
// (R-REV-08), dan mode migrasi menyusul di milestone masing-masing.
package settings

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Settings adalah pengaturan sistem yang berlaku.
type Settings struct {
	// SignDatesEnabled true berarti setiap slot tanda tangan punya tanggal.
	// Bila false, isiannya disembunyikan dan hanya tanggal rilis yang dicatat (D-52).
	SignDatesEnabled bool
	// FormApprovalMode adalah aturan persetujuan form set dan tabel AP (D-139):
	// ApprovalAll (bawaan) berarti semua penyetuju form harus setuju,
	// ApprovalAny berarti satu persetujuan cukup.
	FormApprovalMode string
	// QREnabled menyalakan QR di cetakan (D-62); bawaannya mati.
	QREnabled bool
}

// Aturan persetujuan form (D-139). Nilainya sama dengan di internal/template.
const (
	ApprovalAll = "all"
	ApprovalAny = "any"
)

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
// (E-24, D-103), karena layar dan cetakan mengikutinya.
func (s *Service) Get(ctx context.Context, actor access.Actor) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Settings{}, err
	}
	row, err := s.q.GetSystemSettings(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("membaca pengaturan sistem: %w", err)
	}
	return Settings{SignDatesEnabled: row.SignDatesEnabled, FormApprovalMode: row.FormApprovalMode, QREnabled: row.QrEnabled}, nil
}

// Update adalah perubahan pengaturan. Saklar yang nil tetap seperti sebelumnya.
type Update struct {
	SignDatesEnabled *bool
	FormApprovalMode *string
	QREnabled        *bool
}

// Apply mengubah pengaturan sistem (A-3). Hak diperiksa dulu, juga bila
// tidak ada saklar yang dikirim.
func (s *Service) Apply(ctx context.Context, actor access.Actor, in Update) (Settings, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Settings{}, err
	}
	if in.FormApprovalMode != nil && *in.FormApprovalMode != ApprovalAll && *in.FormApprovalMode != ApprovalAny {
		return Settings{}, &problem.ValidationError{Problems: []problem.Problem{{
			Field: "form_approval_mode", Code: problem.InvalidFormat, Message: "Choose all approvers or any one approver.",
		}}}
	}
	var after Settings
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		before, err := q.LockSystemSettings(ctx)
		if err != nil {
			return err
		}
		after = Settings{SignDatesEnabled: before.SignDatesEnabled, FormApprovalMode: before.FormApprovalMode, QREnabled: before.QrEnabled}
		if in.QREnabled != nil {
			after.QREnabled = *in.QREnabled
		}
		if in.SignDatesEnabled != nil {
			after.SignDatesEnabled = *in.SignDatesEnabled
		}
		if in.FormApprovalMode != nil {
			after.FormApprovalMode = *in.FormApprovalMode
		}
		details := map[string]string{}
		if after.QREnabled != before.QrEnabled {
			if err := q.SetQREnabled(ctx, after.QREnabled); err != nil {
				return err
			}
			details["qr_enabled"] = audit.Change(strconv.FormatBool(before.QrEnabled), strconv.FormatBool(after.QREnabled))
		}
		if after.SignDatesEnabled != before.SignDatesEnabled {
			if err := q.SetSignDates(ctx, after.SignDatesEnabled); err != nil {
				return err
			}
			details["sign_dates_enabled"] = audit.Change(strconv.FormatBool(before.SignDatesEnabled), strconv.FormatBool(after.SignDatesEnabled))
		}
		if after.FormApprovalMode != before.FormApprovalMode {
			if err := q.SetFormApprovalMode(ctx, after.FormApprovalMode); err != nil {
				return err
			}
			details["form_approval_mode"] = audit.Change(before.FormApprovalMode, after.FormApprovalMode)
		}
		if len(details) == 0 {
			return nil // tidak ada yang berubah, jadi tidak dicatat
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSystemSettingsUpdated, TargetType: audit.TargetSystemSettings, TargetID: before.ID, Details: details,
		})
	})
	var invalid *problem.ValidationError
	if errors.As(err, &invalid) {
		return Settings{}, err
	}
	if err != nil {
		return Settings{}, fmt.Errorf("mengubah pengaturan sistem: %w", err)
	}
	return after, nil
}
