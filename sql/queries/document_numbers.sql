-- name: LockCustomerForNumber :one
-- LockCustomerForNumber membaca kode customer yang sedang berlaku dan
-- menguncinya FOR SHARE sampai transaksi selesai. Pendaftaran nomor untuk
-- customer yang sama tidak saling menunggu, tapi ganti kode customer (FOR NO
-- KEY UPDATE) menunggu pendaftaran selesai, jadi kode di nomor selalu kode
-- yang berlaku saat nomor dibuat (R-POR-01). Harus menjadi perintah pertama
-- transaksi penomoran, sebelum INSERT yang merujuk customer: kunci KEY SHARE
-- dari foreign key yang dinaikkan menjadi FOR SHARE bisa deadlock dengan ganti
-- kode (TestRegisterDuringCodeChange).
SELECT id, code, active FROM customers WHERE id = $1 FOR SHARE;

-- name: InsertNumberBatch :one
-- InsertNumberBatch mencatat satu permintaan pendaftaran beserta kunci
-- idempotennya. Bila akun yang sama sudah memakai kunci itu, tidak ada baris
-- baru dan hasilnya pgx.ErrNoRows. Permintaan kedua yang datang bersamaan
-- menunggu di sini sampai permintaan pertama selesai (R-NUM-03).
INSERT INTO number_batches (kind, customer_id, account_id, person_id, idempotency_key, request_hash)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (account_id, idempotency_key) DO NOTHING
RETURNING id;

-- name: GetNumberBatchByKey :one
SELECT id, kind, request_hash FROM number_batches WHERE account_id = $1 AND idempotency_key = $2;

-- name: NextCounterValue :one
-- NextCounterValue menaikkan penghitung satu kode customer dan jenis dokumen,
-- lalu mengembalikan nomor berjalan yang baru. Baris penghitung terkunci
-- sampai transaksi selesai, jadi pendaftaran lain untuk kode dan jenis yang
-- sama menunggu. Bila transaksi batal, nilainya ikut kembali, jadi tidak ada
-- lubang (R-NUM-03).
INSERT INTO number_counters (customer_code, document_type_id, last_value)
VALUES (sqlc.arg(customer_code), sqlc.arg(document_type_id), sqlc.arg(first_value)::integer)
ON CONFLICT (customer_code, document_type_id)
DO UPDATE SET last_value = number_counters.last_value + 1
RETURNING last_value;

-- name: GetCounterValue :one
SELECT last_value FROM number_counters WHERE customer_code = $1 AND document_type_id = $2;

-- name: InsertDocumentNumber :one
INSERT INTO document_numbers (number, customer_id, customer_code, document_type_id, sequence, status, initial_revision, batch_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: GetDocumentNumber :one
-- GetDocumentNumber mengambil satu nomor beserta customer, jenis dokumen,
-- asal, dan orang yang mendaftarkannya.
SELECT n.id, n.number, n.customer_id, c.code AS customer_current_code, c.name AS customer_name,
       n.customer_code, n.document_type_id, d.name AS document_type_name, d.key AS document_type_key,
       n.sequence, n.status, n.initial_revision, n.batch_id, b.kind AS origin,
       b.person_id AS registered_by_id, p.name AS registered_by_name,
       n.status_reason, n.status_changed_at, n.created_at
FROM document_numbers n
JOIN customers c ON c.id = n.customer_id
JOIN document_types d ON d.id = n.document_type_id
JOIN number_batches b ON b.id = n.batch_id
JOIN people p ON p.id = b.person_id
WHERE n.id = $1;

-- name: ListBatchNumbers :many
-- ListBatchNumbers mengambil nomor hasil satu permintaan, menurut urutan dibuat.
SELECT n.id, n.number, n.customer_id, c.code AS customer_current_code, c.name AS customer_name,
       n.customer_code, n.document_type_id, d.name AS document_type_name, d.key AS document_type_key,
       n.sequence, n.status, n.initial_revision, n.batch_id, b.kind AS origin,
       b.person_id AS registered_by_id, p.name AS registered_by_name,
       n.status_reason, n.status_changed_at, n.created_at
FROM document_numbers n
JOIN customers c ON c.id = n.customer_id
JOIN document_types d ON d.id = n.document_type_id
JOIN number_batches b ON b.id = n.batch_id
JOIN people p ON p.id = b.person_id
WHERE n.batch_id = $1
ORDER BY n.id;

-- name: ListDocumentNumbers :many
-- ListDocumentNumbers mengambil register dengan saringan yang boleh kosong
-- (M2.3), terbaru dulu. ID buatan uuidv7() urut menurut waktu dibuat, jadi
-- halaman berikutnya diminta dengan before = ID terakhir di halaman ini.
-- Pencarian teks memakai strpos, jadi % dan _ dari pengguna tidak berarti apa-apa.
SELECT n.id, n.number, n.customer_id, c.code AS customer_current_code, c.name AS customer_name,
       n.customer_code, n.document_type_id, d.name AS document_type_name, d.key AS document_type_key,
       n.sequence, n.status, n.initial_revision, n.batch_id, b.kind AS origin,
       b.person_id AS registered_by_id, p.name AS registered_by_name,
       n.status_reason, n.status_changed_at, n.created_at
FROM document_numbers n
JOIN customers c ON c.id = n.customer_id
JOIN document_types d ON d.id = n.document_type_id
JOIN number_batches b ON b.id = n.batch_id
JOIN people p ON p.id = b.person_id
WHERE (sqlc.narg(customer_id)::uuid IS NULL OR n.customer_id = sqlc.narg(customer_id)::uuid)
  AND (sqlc.narg(document_type_id)::uuid IS NULL OR n.document_type_id = sqlc.narg(document_type_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR n.status = sqlc.narg(status)::text)
  AND (sqlc.narg(query)::text IS NULL OR strpos(lower(n.number), lower(sqlc.narg(query)::text)) > 0)
  AND (sqlc.narg(before)::uuid IS NULL OR n.id < sqlc.narg(before)::uuid)
ORDER BY n.id DESC
LIMIT sqlc.arg(max_items);

-- name: LockDocumentNumber :exec
-- LockDocumentNumber mengunci baris nomor sampai transaksi selesai, supaya
-- status yang diperiksa tidak berubah oleh transaksi lain di tengah jalan.
SELECT id FROM document_numbers WHERE id = $1 FOR NO KEY UPDATE;

-- name: SetDocumentNumberStatus :exec
UPDATE document_numbers SET status = $2, status_reason = $3, status_changed_at = $4 WHERE id = $1;

-- name: LockCounter :one
-- LockCounter membuat baris penghitung bila belum ada (nilai awal
-- first_value), lalu menguncinya sampai transaksi selesai dan mengembalikan
-- nilainya. Dipakai daftar sebagai set (M2.4) dan impor (M2.5), yang
-- mengunci beberapa penghitung sekaligus, selalu menurut urutan kode customer
-- lalu ID jenis dokumen, supaya dua transaksi tidak saling menunggu (deadlock).
INSERT INTO number_counters (customer_code, document_type_id, last_value)
VALUES (sqlc.arg(customer_code), sqlc.arg(document_type_id), sqlc.arg(first_value)::integer)
ON CONFLICT (customer_code, document_type_id)
DO UPDATE SET last_value = number_counters.last_value
RETURNING last_value;

-- name: SetCounter :exec
-- SetCounter mengganti nilai penghitung yang sudah dikunci. Penghitung tidak
-- pernah turun, jadi nomor yang sudah ada tidak pernah diberikan lagi.
UPDATE number_counters SET last_value = greatest(last_value, sqlc.arg(last_value)::integer)
WHERE customer_code = sqlc.arg(customer_code) AND document_type_id = sqlc.arg(document_type_id);

-- name: ListRegisterOfCodes :many
-- ListRegisterOfCodes mengambil kode, jenis dokumen, nomor berjalan, dan
-- status semua nomor dengan kode customer itu, untuk pemeriksaan dan laporan
-- impor (M2.5). Kode customer unik di semua customer, jadi hasilnya adalah
-- register satu customer bila kodenya kode customer itu.
SELECT number, customer_code, document_type_id, sequence, status
FROM document_numbers
WHERE customer_code = ANY(sqlc.arg(codes)::text[])
ORDER BY customer_code, document_type_id, sequence;

-- name: ListNumbersByText :many
-- ListNumbersByText mengambil teks nomor yang sudah ada di register dari
-- daftar teks (impor, M2.5).
SELECT number FROM document_numbers WHERE number = ANY(sqlc.arg(numbers)::text[]);
