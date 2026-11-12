-- name: GetSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode FROM system_settings;

-- name: LockSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode FROM system_settings FOR NO KEY UPDATE;

-- name: SetSignDates :exec
UPDATE system_settings SET sign_dates_enabled = $1;
