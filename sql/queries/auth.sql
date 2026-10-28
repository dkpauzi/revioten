-- name: LockAdminBootstrap :exec
-- LockAdminBootstrap memegang kunci advisory sampai transaksi selesai, supaya
-- dua perintah create-admin yang berjalan bersamaan tidak sama-sama lolos.
SELECT pg_advisory_xact_lock($1);

-- name: AdminAccountExists :one
SELECT EXISTS (SELECT 1 FROM accounts WHERE kind = 'admin');

-- name: CreatePerson :one
INSERT INTO people (name, employee_number)
VALUES ($1, $2)
RETURNING id, name, employee_number, created_at;

-- name: CreateAccount :one
INSERT INTO accounts (person_id, kind, login, password_hash, must_change_password)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, person_id, kind, login, must_change_password, created_at;

-- name: GetAccountByLogin :one
-- GetAccountByLogin mengambil akun beserta nama orangnya untuk proses login.
SELECT a.id, a.person_id, a.kind, a.login, a.password_hash, a.must_change_password,
       a.failed_logins, a.locked_until, a.disabled_at, p.name AS person_name
FROM accounts a
JOIN people p ON p.id = a.person_id
WHERE a.login = $1;

-- name: GetAccountByID :one
SELECT a.id, a.person_id, a.kind, a.login, a.password_hash, a.must_change_password,
       a.failed_logins, a.locked_until, a.disabled_at, p.name AS person_name
FROM accounts a
JOIN people p ON p.id = a.person_id
WHERE a.id = $1;

-- name: RecordLoginFailure :one
-- RecordLoginFailure menambah hitungan gagal login dan mengembalikan hitungan barunya.
UPDATE accounts
SET failed_logins = failed_logins + 1
WHERE id = $1
RETURNING failed_logins;

-- name: LockAccountUntil :exec
UPDATE accounts SET locked_until = $2 WHERE id = $1;

-- name: DisableAccount :exec
UPDATE accounts SET disabled_at = $2 WHERE id = $1;

-- name: RecordLoginSuccess :exec
UPDATE accounts SET failed_logins = 0, locked_until = NULL WHERE id = $1;

-- name: SetPasswordHash :exec
UPDATE accounts SET password_hash = $2, must_change_password = $3 WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, account_id, created_at, last_seen_at, expires_at)
VALUES ($1, $2, $3, $3, $4);

-- name: GetSession :one
-- GetSession mengambil sesi beserta akun dan orangnya.
SELECT s.account_id, s.created_at, s.last_seen_at, s.expires_at,
       a.person_id, a.kind, a.login, a.must_change_password, a.disabled_at,
       p.name AS person_name
FROM sessions s
JOIN accounts a ON a.id = s.account_id
JOIN people p ON p.id = a.person_id
WHERE s.token_hash = $1;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = $2 WHERE token_hash = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteAccountSessions :exec
DELETE FROM sessions WHERE account_id = $1;

-- name: DeleteExpiredSessions :exec
-- DeleteExpiredSessions membuang sesi yang sudah melewati batas mutlak atau
-- sudah terlalu lama tidak dipakai.
DELETE FROM sessions WHERE expires_at <= $1 OR last_seen_at <= $2;
