-- name: ListApprovalRoutes :many
-- ListApprovalRoutes mengambil semua rute approval, urut menurut nama.
SELECT id, name, created_at FROM approval_routes ORDER BY lower(name);

-- name: GetApprovalRoute :one
SELECT id, name, created_at FROM approval_routes WHERE id = $1;

-- name: LockApprovalRoute :exec
-- LockApprovalRoute memakai FOR NO KEY UPDATE, karena customer dan jenis
-- perubahan menunjuk rute lewat foreign key (pelajaran deadlock M1.3).
SELECT id FROM approval_routes WHERE id = $1 FOR NO KEY UPDATE;

-- name: CreateApprovalRoute :one
INSERT INTO approval_routes (name) VALUES ($1) RETURNING id;

-- name: RenameApprovalRoute :exec
UPDATE approval_routes SET name = $2 WHERE id = $1;

-- name: ListRouteSlots :many
-- ListRouteSlots mengambil slot semua rute (route_id kosong) atau satu rute,
-- urut menurut rute lalu urutan slot.
SELECT id, route_id, slot_order, stage, print_title
FROM route_slots
WHERE sqlc.narg(route_id)::uuid IS NULL OR route_id = sqlc.narg(route_id)::uuid
ORDER BY route_id, slot_order;

-- name: ListRouteSlotPositions :many
-- ListRouteSlotPositions mengambil jabatan setiap slot, urut menurut bagan organisasi.
SELECT sp.slot_id, p.id AS position_id, p.name AS position_name
FROM route_slot_positions sp
JOIN positions p ON p.id = sp.position_id
JOIN route_slots s ON s.id = sp.slot_id
WHERE sqlc.narg(route_id)::uuid IS NULL OR s.route_id = sqlc.narg(route_id)::uuid
ORDER BY sp.slot_id, p.sort_order;

-- name: DeleteRouteSlots :exec
-- DeleteRouteSlots membuang semua slot satu rute. Jabatan per slot ikut
-- terhapus (ON DELETE CASCADE).
DELETE FROM route_slots WHERE route_id = $1;

-- name: AddRouteSlot :one
INSERT INTO route_slots (route_id, slot_order, stage, print_title) VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: AddRouteSlotPosition :exec
INSERT INTO route_slot_positions (slot_id, position_id) VALUES ($1, $2);

-- name: SetCustomerDefaultRoute :exec
UPDATE customers SET default_route_id = $2 WHERE id = $1;

-- name: SetChangeTypeRoute :exec
UPDATE change_types SET approval_route_id = $2 WHERE id = $1;
