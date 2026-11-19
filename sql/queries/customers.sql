-- name: ListCustomers :many
-- ListCustomers mengambil semua customer beserta Project Group dan rute
-- default-nya (M1.8), urut menurut kode.
SELECT c.id, c.code, c.name, c.active, c.created_at, c.project_group_id, g.code AS project_group_code,
       c.default_route_id, r.name AS default_route_name, c.signature_mode
FROM customers c
LEFT JOIN project_groups g ON g.id = c.project_group_id
LEFT JOIN approval_routes r ON r.id = c.default_route_id
ORDER BY c.code;

-- name: GetCustomer :one
SELECT c.id, c.code, c.name, c.active, c.created_at, c.project_group_id, g.code AS project_group_code,
       c.default_route_id, r.name AS default_route_name, c.signature_mode
FROM customers c
LEFT JOIN project_groups g ON g.id = c.project_group_id
LEFT JOIN approval_routes r ON r.id = c.default_route_id
WHERE c.id = $1;

-- name: LockCustomer :exec
-- LockCustomer mengunci baris customer sampai transaksi selesai, supaya isi
-- "sebelum" yang dicatat di audit log tidak berubah di tengah jalan.
-- FOR NO KEY UPDATE tidak bentrok dengan pemeriksaan foreign key (lihat LockAccount).
SELECT id FROM customers WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateCustomer :one
-- CreateCustomer menyimpan satu customer baru dan mengembalikan barisnya,
-- termasuk id dan created_at yang diisi database.
INSERT INTO customers (code, name, project_group_id)
VALUES ($1, $2, $3)
RETURNING id, code, name, active, created_at, project_group_id;

-- name: UpdateCustomer :exec
UPDATE customers SET name = $2, project_group_id = $3, active = $4 WHERE id = $1;

-- name: SetCustomerCode :exec
UPDATE customers SET code = $2 WHERE id = $1;

-- name: ListCustomerCodes :many
-- ListCustomerCodes mengambil riwayat kode semua customer, lama ke baru.
SELECT customer_id, code, valid_from, valid_until
FROM customer_codes
ORDER BY customer_id, valid_from;

-- name: AddCustomerCode :exec
INSERT INTO customer_codes (code, customer_id, valid_from) VALUES ($1, $2, $3);

-- name: CloseCustomerCode :exec
-- CloseCustomerCode mengakhiri masa berlaku kode yang sedang berlaku.
UPDATE customer_codes SET valid_until = $2 WHERE customer_id = $1 AND valid_until IS NULL;

-- name: ListCodesOfCustomer :many
-- ListCodesOfCustomer mengambil riwayat kode satu customer, lama ke baru.
SELECT code, valid_from, valid_until
FROM customer_codes
WHERE customer_id = $1
ORDER BY valid_from;
