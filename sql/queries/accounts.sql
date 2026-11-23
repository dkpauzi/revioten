-- name: ListAccounts :many
-- ListAccounts mengambil semua akun beserta orangnya, untuk layar admin (A-1).
SELECT a.id, a.kind, a.login, a.must_change_password, a.locked_until, a.disabled_at, a.created_at,
       a.temporary_password_expires_at,
       p.id AS person_id, p.name AS person_name, p.employee_number, p.email
FROM accounts a
JOIN people p ON p.id = a.person_id
ORDER BY p.name, a.kind;

-- name: GetAccountInfo :one
SELECT a.id, a.kind, a.login, a.must_change_password, a.locked_until, a.disabled_at, a.created_at,
       a.temporary_password_expires_at,
       p.id AS person_id, p.name AS person_name, p.employee_number, p.email
FROM accounts a
JOIN people p ON p.id = a.person_id
WHERE a.id = $1;

-- name: GetPerson :one
SELECT id, name, employee_number, created_at, email FROM people WHERE id = $1;

-- name: ResetAccountPassword :exec
-- ResetAccountPassword memasang password sementara, mewajibkan ganti
-- password, dan membuka kunci akun.
UPDATE accounts
SET password_hash = $2, must_change_password = true, failed_logins = 0, locked_until = NULL,
    temporary_password_expires_at = NULL
WHERE id = $1;

-- name: EnableAccount :exec
UPDATE accounts SET disabled_at = NULL, failed_logins = 0, locked_until = NULL WHERE id = $1;

-- name: CountEnabledAdmins :one
SELECT count(*) FROM accounts WHERE kind = 'admin' AND disabled_at IS NULL;

-- name: LockAccount :exec
-- LockAccount mengunci baris akun sampai transaksi selesai, supaya isi akun
-- yang dibaca untuk audit log (misalnya peran sebelum diubah) tidak berubah
-- oleh transaksi lain di tengah jalan. FOR NO KEY UPDATE, bukan FOR UPDATE:
-- kunci ini tidak bentrok dengan pemeriksaan foreign key saat transaksi lain
-- menulis entri audit tentang akun yang sama, jadi tidak bisa terjadi deadlock
-- dengan kunci rantai audit (lihat TestLockAccountAllowsAudit).
SELECT id FROM accounts WHERE id = $1 FOR NO KEY UPDATE;

-- name: SetTemporaryPasswordExpiry :exec
-- SetTemporaryPasswordExpiry memasang batas password sementara akun hasil
-- import (D-224).
UPDATE accounts SET temporary_password_expires_at = $2 WHERE id = $1;

-- name: EmployeeNumberTaken :one
-- EmployeeNumberTaken melaporkan apakah nomor karyawan sudah dipakai orang lain.
SELECT EXISTS (SELECT 1 FROM people WHERE employee_number = $1);
