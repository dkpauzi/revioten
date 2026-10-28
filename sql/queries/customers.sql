-- name: ListCustomers :many
-- ListCustomers mengambil semua customer, urut menurut kode.
SELECT id, code, name, created_at
FROM customers
ORDER BY code;

-- name: CreateCustomer :one
-- CreateCustomer menyimpan satu customer baru dan mengembalikan barisnya,
-- termasuk id dan created_at yang diisi database.
INSERT INTO customers (code, name)
VALUES ($1, $2)
RETURNING id, code, name, created_at;
