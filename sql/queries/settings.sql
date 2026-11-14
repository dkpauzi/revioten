-- name: GetSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled, migration_mode FROM system_settings;

-- name: LockSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled, migration_mode FROM system_settings FOR NO KEY UPDATE;

-- name: SetSignDates :exec
UPDATE system_settings SET sign_dates_enabled = $1;

-- name: SetQREnabled :exec
-- SetQREnabled menyalakan atau mematikan QR di cetakan (D-62, R-APR-16).
UPDATE system_settings SET qr_enabled = $1;

-- name: SetMigrationMode :exec
-- SetMigrationMode menyalakan atau mematikan mode migrasi (R-MIG-06).
UPDATE system_settings SET migration_mode = $1;

-- name: GetMigrationMode :one
SELECT migration_mode FROM system_settings;
