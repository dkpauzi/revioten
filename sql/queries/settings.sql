-- name: GetSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled FROM system_settings;

-- name: LockSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled FROM system_settings FOR NO KEY UPDATE;

-- name: SetSignDates :exec
UPDATE system_settings SET sign_dates_enabled = $1;

-- name: SetQREnabled :exec
-- SetQREnabled menyalakan atau mematikan QR di cetakan (D-62, R-APR-16).
UPDATE system_settings SET qr_enabled = $1;
