-- name: ListAccounts :many
-- ListAccounts mengambil semua akun beserta orangnya, untuk layar admin (A-1).
SELECT a.id, a.kind, a.login, a.must_change_password, a.locked_until, a.disabled_at, a.created_at,
       p.id AS person_id, p.name AS person_name, p.employee_number
FROM accounts a
JOIN people p ON p.id = a.person_id
ORDER BY p.name, a.kind;

-- name: GetAccountInfo :one
SELECT a.id, a.kind, a.login, a.must_change_password, a.locked_until, a.disabled_at, a.created_at,
       p.id AS person_id, p.name AS person_name, p.employee_number
FROM accounts a
JOIN people p ON p.id = a.person_id
WHERE a.id = $1;

-- name: GetPerson :one
SELECT id, name, employee_number, created_at FROM people WHERE id = $1;

-- name: ResetAccountPassword :exec
-- ResetAccountPassword memasang password sementara, mewajibkan ganti
-- password, dan membuka kunci akun.
UPDATE accounts
SET password_hash = $2, must_change_password = true, failed_logins = 0, locked_until = NULL
WHERE id = $1;

-- name: EnableAccount :exec
UPDATE accounts SET disabled_at = NULL, failed_logins = 0, locked_until = NULL WHERE id = $1;

-- name: CountEnabledAdmins :one
SELECT count(*) FROM accounts WHERE kind = 'admin' AND disabled_at IS NULL;
