-- name: LockNumberCodes :exec
-- LockNumberCodes memegang kunci advisory 11003 sampai transaksi selesai.
-- Kode customer dan kode khusus menempati posisi yang sama di nomor dokumen
-- (E-12), dan disimpan di dua tabel, jadi UNIQUE tidak bisa menjaga keduanya.
-- Semua pendaftaran kode di posisi itu mengambil kunci ini dulu, supaya
-- berjalan bergantian (E-37, D-114). Kunci lain: 11001 admin pertama, 11002
-- rantai audit.
SELECT pg_advisory_xact_lock(11003);

-- name: SpecialCodeExists :one
SELECT EXISTS (SELECT 1 FROM special_codes WHERE code = $1);

-- name: CustomerCodeExists :one
-- CustomerCodeExists melaporkan apakah kode sedang atau pernah dipakai customer (E-11).
SELECT EXISTS (SELECT 1 FROM customer_codes WHERE code = $1);

-- name: GetNumberSettings :one
SELECT id, factory_code FROM number_settings;

-- name: LockNumberSettings :one
SELECT id, factory_code FROM number_settings FOR NO KEY UPDATE;

-- name: SetFactoryCode :exec
UPDATE number_settings SET factory_code = $1;

-- name: ListDocumentTypes :many
-- ListDocumentTypes mengambil jenis dokumen beserta awalan nomornya: kode
-- pabrik ditambah huruf jenis (E-12).
SELECT d.id, d.key, d.name, d.letter, (s.factory_code || d.letter)::text AS prefix
FROM document_types d
CROSS JOIN number_settings s
ORDER BY d.position, d.letter;

-- name: GetDocumentType :one
SELECT d.id, d.key, d.name, d.letter, (s.factory_code || d.letter)::text AS prefix
FROM document_types d
CROSS JOIN number_settings s
WHERE d.id = $1;

-- name: LockDocumentType :exec
SELECT id FROM document_types WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateDocumentType :one
-- CreateDocumentType menambah jenis dokumen di urutan terakhir.
INSERT INTO document_types (name, letter, position)
SELECT sqlc.arg(name)::text, sqlc.arg(letter)::text, coalesce(max(position), 0) + 1
FROM document_types
RETURNING id;

-- name: UpdateDocumentType :exec
UPDATE document_types SET name = $2, letter = $3 WHERE id = $1;

-- name: ListSpecialCodes :many
SELECT id, code, name, created_at FROM special_codes ORDER BY code;

-- name: GetSpecialCode :one
SELECT id, code, name, created_at FROM special_codes WHERE id = $1;

-- name: LockSpecialCode :exec
SELECT id FROM special_codes WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateSpecialCode :one
INSERT INTO special_codes (code, name) VALUES ($1, $2)
RETURNING id, code, name, created_at;

-- name: RenameSpecialCode :exec
UPDATE special_codes SET name = $2 WHERE id = $1;
