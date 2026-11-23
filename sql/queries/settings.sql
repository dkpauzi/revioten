-- name: GetSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled, migration_mode, submission_form_number FROM system_settings;

-- name: LockSystemSettings :one
SELECT id, sign_dates_enabled, form_approval_mode, qr_enabled, migration_mode, submission_form_number FROM system_settings FOR NO KEY UPDATE;

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


-- Bagian C (E-144): nama perusahaan dan logo cetakan (migrasi 00056).

-- name: GetCompany :one
SELECT company_name, company_logo_sha256, company_logo_updated_at FROM system_settings;

-- name: GetCompanyLogo :one
SELECT company_logo FROM system_settings;

-- name: SetCompanyName :exec
UPDATE system_settings SET company_name = $1;

-- name: SetCompanyLogo :exec
UPDATE system_settings
SET company_logo = $1, company_logo_sha256 = $2, company_logo_updated_at = now(), company_logo_updated_by_person_id = $3;


-- D-223: riwayat logo dan penggantian logo (migrasi 00058).

-- name: SaveCompanyLogo :exec
INSERT INTO company_logos (sha256, png, created_by_person_id) VALUES ($1, $2, $3)
ON CONFLICT (sha256) DO NOTHING;

-- name: GetCompanyLogoBySHA :one
SELECT png FROM company_logos WHERE sha256 = $1;

-- name: RecordLogoChange :one
INSERT INTO logo_changes (logo_sha256, revise, created_by_person_id) VALUES ($1, $2, $3)
RETURNING id;
