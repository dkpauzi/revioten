-- Tahap 9 · API integrasi. Lihat migrasi 00044.

-- name: CreateServiceAccount :one
INSERT INTO service_accounts (name, owner_team, contact, purpose, scopes, allowed_networks, created_by_person_id)
VALUES ($1, $2, $3, $4, $5, sqlc.arg(allowed_networks)::text[]::cidr[], $6)
RETURNING id;

-- name: ServiceAccountNameExists :one
SELECT EXISTS (SELECT 1 FROM service_accounts WHERE lower(name) = lower($1));

-- name: AddServiceAccountCustomer :exec
INSERT INTO service_account_customers (service_account_id, customer_id) VALUES ($1, $2);

-- name: ListServiceAccounts :many
SELECT sa.id, sa.name, sa.owner_team, sa.contact, sa.purpose, sa.scopes, sa.allowed_networks::text[] AS allowed_networks,
       sa.disabled_at, sa.created_at,
       coalesce((SELECT array_agg(c.code ORDER BY c.code) FROM service_account_customers sc JOIN customers c ON c.id = sc.customer_id
                 WHERE sc.service_account_id = sa.id), '{}')::text[] AS customer_codes,
       coalesce((SELECT array_agg(sc.customer_id::text ORDER BY sc.customer_id) FROM service_account_customers sc
                 WHERE sc.service_account_id = sa.id), '{}')::text[] AS customer_ids
FROM service_accounts sa
ORDER BY sa.name;

-- name: GetServiceAccount :one
SELECT sa.id, sa.name, sa.owner_team, sa.contact, sa.purpose, sa.scopes, sa.allowed_networks::text[] AS allowed_networks,
       sa.disabled_at, sa.created_at,
       coalesce((SELECT array_agg(c.code ORDER BY c.code) FROM service_account_customers sc JOIN customers c ON c.id = sc.customer_id
                 WHERE sc.service_account_id = sa.id), '{}')::text[] AS customer_codes,
       coalesce((SELECT array_agg(sc.customer_id::text ORDER BY sc.customer_id) FROM service_account_customers sc
                 WHERE sc.service_account_id = sa.id), '{}')::text[] AS customer_ids
FROM service_accounts sa
WHERE sa.id = $1;

-- name: DisableServiceAccount :execrows
UPDATE service_accounts SET disabled_at = now() WHERE id = $1 AND disabled_at IS NULL;

-- name: CreateServiceToken :one
INSERT INTO service_tokens (service_account_id, prefix, token_hash, expires_at, created_by_person_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: ListServiceTokens :many
SELECT id, service_account_id, prefix, expires_at, revoked_at, last_used_at, created_at
FROM service_tokens
ORDER BY created_at DESC;

-- name: RevokeServiceToken :one
UPDATE service_tokens SET revoked_at = now()
WHERE id = $1 AND service_account_id = $2 AND revoked_at IS NULL
RETURNING prefix;

-- name: FindServiceToken :one
-- FindServiceToken mencari token menurut hash-nya, beserta keadaan akun
-- pemiliknya. Kedaluwarsa, dicabut, dan nonaktif diputuskan pemanggil,
-- supaya penolakannya bisa dicatat.
SELECT t.id, t.prefix, t.expires_at, t.revoked_at, sa.id AS service_account_id, sa.name, sa.scopes,
       sa.allowed_networks::text[] AS allowed_networks, sa.disabled_at,
       coalesce((SELECT array_agg(sc.customer_id::text) FROM service_account_customers sc WHERE sc.service_account_id = sa.id), '{}')::text[] AS customer_ids
FROM service_tokens t
JOIN service_accounts sa ON sa.id = t.service_account_id
WHERE t.token_hash = $1;

-- name: TouchServiceToken :exec
UPDATE service_tokens SET last_used_at = now() WHERE id = $1;

-- M9.3: bacaan. Hanya dokumen yang punya revisi terbit, dan hanya customer
-- yang diizinkan untuk token itu.

-- name: ListReleasedDocuments :many
-- ListReleasedDocuments memberi dokumen terbit menurut waktu rilis revisi
-- yang berlaku, lalu ID, supaya paginasi cursor stabil (R-API-13).
SELECT d.id, n.number, c.id AS customer_id, c.code AS customer_code, t.key AS document_type_key,
       d.released_revision, r.released_at, r.snapshot_sha256, r.migrated
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN customers c ON c.id = d.customer_id
JOIN document_types t ON t.id = d.document_type_id
JOIN document_revisions r ON r.document_id = d.id AND r.revision = d.released_revision
WHERE d.customer_id::text = ANY (sqlc.arg(customer_ids)::text[])
  AND (sqlc.narg(customer_id)::uuid IS NULL OR d.customer_id = sqlc.narg(customer_id)::uuid)
  AND (sqlc.narg(updated_since)::timestamptz IS NULL OR r.released_at >= sqlc.narg(updated_since)::timestamptz)
  AND (sqlc.narg(after_time)::timestamptz IS NULL
       OR (r.released_at, d.id) > (sqlc.narg(after_time)::timestamptz, sqlc.narg(after_id)::uuid))
ORDER BY r.released_at, d.id
LIMIT sqlc.arg(page_size);

-- name: GetReleasedRevision :one
-- GetReleasedRevision memberi satu revisi terbit beserta snapshot-nya.
-- revision NULL berarti revisi yang berlaku.
SELECT d.id, d.customer_id, n.number, d.released_revision, r.revision, r.released_at, r.release_date, r.snapshot, r.snapshot_sha256, r.migrated
FROM documents d
JOIN document_numbers n ON n.id = d.document_number_id
JOIN document_revisions r ON r.document_id = d.id AND r.status = 'released'
     AND r.revision = coalesce(sqlc.narg(revision)::smallint, d.released_revision)
WHERE d.id = $1;

-- M9.4: usulan perubahan dari luar.

-- name: GetIntegrationRequest :one
SELECT request_hash, change_id FROM integration_requests
WHERE service_account_id = $1 AND idempotency_key = $2
FOR UPDATE;

-- name: AddIntegrationRequest :exec
INSERT INTO integration_requests (service_account_id, idempotency_key, request_hash, change_id) VALUES ($1, $2, $3, $4);

-- name: CreateIntegrationChange :one
INSERT INTO changes (customer_id, mechanism, change_type_id, reason, source_reference, requested_by_service_account_id)
VALUES ($1, 'integration', $2, $3, $4, $5)
RETURNING id, seq;

-- name: CreateChangeProposal :exec
INSERT INTO change_proposals (change_id, document_id, base_revision, source_application, on_behalf_of, operations)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetChangeProposal :one
SELECT cp.change_id, cp.document_id, cp.base_revision, cp.source_application, cp.on_behalf_of, cp.operations, cp.applied_at
FROM change_proposals cp WHERE cp.change_id = $1;

-- name: MarkProposalApplied :exec
UPDATE change_proposals SET applied_at = now() WHERE change_id = $1;

-- name: GetIntegrationChangeStatus :one
SELECT c.id, c.seq, c.status, c.customer_id, c.created_at, c.verified_at, c.verification_note
FROM changes c
WHERE c.id = $1 AND c.requested_by_service_account_id = $2;

-- name: ChangeTypeByKey :one
SELECT id FROM change_types WHERE key = $1;

-- name: PatchRowContent :execrows
-- PatchRowContent menulis isi baru satu baris draft dari usulan luar.
UPDATE document_rows SET content = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND document_id = $2;

-- M9.5: ekspor asinkron.

-- name: CreateExportJob :one
INSERT INTO export_jobs (service_account_id, document_id, revision) VALUES ($1, $2, $3)
RETURNING id, created_at;

-- name: GetExportJob :one
SELECT id, document_id, revision, status, file_name, file_sha256, size_bytes, error, created_at, finished_at
FROM export_jobs WHERE id = $1 AND service_account_id = $2;

-- name: ClaimExportJob :one
-- ClaimExportJob mengambil satu job yang antre. SKIP LOCKED membuat dua
-- pekerja tidak mengambil job yang sama.
UPDATE export_jobs SET status = 'running', started_at = now()
WHERE id = (SELECT id FROM export_jobs WHERE status = 'queued' ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING id, document_id, revision;

-- name: FinishExportJob :exec
UPDATE export_jobs SET status = 'done', file_name = $2, file_sha256 = $3, size_bytes = $4, finished_at = now() WHERE id = $1;

-- name: FailExportJob :exec
UPDATE export_jobs SET status = 'failed', error = $2, finished_at = now() WHERE id = $1;

-- name: RequeueRunningExports :execrows
-- RequeueRunningExports mengembalikan job yang tertahan berstatus running
-- (server mati di tengah pekerjaan) ke antrean (D-188).
UPDATE export_jobs SET status = 'queued', started_at = NULL WHERE status = 'running';
