-- Tahap 9 · API integrasi (M9.1 sampai M9.5; R-API-03 sampai R-API-15, D-19,
-- D-37). Latihan di localhost; konsumen nyata belum ditetapkan (Q-22).
--
--   service_accounts            aplikasi lain yang boleh memanggil API
--                               integrasi: nama, tim pemilik, kontak, tujuan,
--                               scope, jaringan yang diizinkan (R-API-04).
--   service_account_customers   customer yang boleh disentuh (R-API-05).
--   service_tokens              token: hanya awalan dan hash SHA-256 yang
--                               disimpan; nilainya tampil sekali. Bisa
--                               kedaluwarsa dan dicabut.
--   integration_requests        kunci idempoten untuk penulisan (R-API-10).
--   change_proposals            usulan perubahan dari luar: operasi patch
--                               terhadap revisi dasar (R-API-07, R-API-11).
--   export_jobs                 ekspor asinkron (R-API-14).
--   changes                     pemohon boleh berupa service account;
--                               mekanisme baru 'integration'.
--   audit_log                   jenis aktor baru 'service' (R-API-03).

-- +goose Up
CREATE TABLE service_accounts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL CHECK (name <> '' AND name = btrim(name) AND char_length(name) <= 100),
    owner_team text NOT NULL CHECK (owner_team <> '' AND owner_team = btrim(owner_team) AND char_length(owner_team) <= 100),
    contact text NOT NULL CHECK (contact <> '' AND contact = btrim(contact) AND char_length(contact) <= 200),
    purpose text NOT NULL CHECK (purpose <> '' AND purpose = btrim(purpose) AND char_length(purpose) <= 500),
    scopes text[] NOT NULL CHECK (cardinality(scopes) >= 1),
    -- Jaringan asal yang diizinkan; kosong berarti semua alamat.
    allowed_networks cidr[] NOT NULL DEFAULT '{}',
    disabled_at timestamptz,
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT service_accounts_name_key UNIQUE (name)
);
GRANT SELECT, INSERT ON service_accounts TO revioten_app;
GRANT UPDATE (disabled_at) ON service_accounts TO revioten_app;

CREATE TABLE service_account_customers (
    service_account_id uuid NOT NULL REFERENCES service_accounts (id),
    customer_id uuid NOT NULL REFERENCES customers (id),
    PRIMARY KEY (service_account_id, customer_id)
);
GRANT SELECT, INSERT ON service_account_customers TO revioten_app;

CREATE TABLE service_tokens (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    service_account_id uuid NOT NULL REFERENCES service_accounts (id),
    -- Awalan untuk mengenali token di layar dan log; bukan rahasia.
    prefix text NOT NULL CHECK (prefix ~ '^[a-z0-9]{8}$'),
    token_hash text NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    last_used_at timestamptz,
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT service_tokens_prefix_key UNIQUE (prefix),
    CONSTRAINT service_tokens_hash_key UNIQUE (token_hash)
);
GRANT SELECT, INSERT ON service_tokens TO revioten_app;
GRANT UPDATE (revoked_at, last_used_at) ON service_tokens TO revioten_app;

CREATE TABLE integration_requests (
    service_account_id uuid NOT NULL REFERENCES service_accounts (id),
    idempotency_key text NOT NULL CHECK (idempotency_key <> '' AND char_length(idempotency_key) <= 200),
    -- SHA-256 isi permintaan: kunci yang sama dengan isi berbeda ditolak.
    request_hash text NOT NULL,
    change_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (service_account_id, idempotency_key)
);
GRANT SELECT, INSERT ON integration_requests TO revioten_app;
GRANT UPDATE (change_id) ON integration_requests TO revioten_app;

-- Pemohon perubahan: orang, atau service account.
ALTER TABLE changes
    ALTER COLUMN requested_by_person_id DROP NOT NULL,
    ALTER COLUMN requested_by_account_id DROP NOT NULL,
    ADD COLUMN requested_by_service_account_id uuid REFERENCES service_accounts (id),
    ADD CONSTRAINT changes_requester_check CHECK (
        (requested_by_service_account_id IS NULL AND requested_by_person_id IS NOT NULL AND requested_by_account_id IS NOT NULL)
        OR (requested_by_service_account_id IS NOT NULL AND requested_by_person_id IS NULL AND requested_by_account_id IS NULL)),
    DROP CONSTRAINT changes_mechanism_check,
    ADD CONSTRAINT changes_mechanism_check CHECK (mechanism IN ('manual', 'yokoten', 'template', 'sqam', 'integration')),
    ADD CONSTRAINT changes_integration_check CHECK ((mechanism = 'integration') = (requested_by_service_account_id IS NOT NULL));

CREATE TABLE change_proposals (
    change_id uuid PRIMARY KEY REFERENCES changes (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    -- Revisi terbit yang menjadi dasar usulan (R-API-11).
    base_revision smallint NOT NULL CHECK (base_revision >= 0),
    source_application text NOT NULL CHECK (source_application <> '' AND char_length(source_application) <= 100),
    -- Orang yang disebut aplikasi pemanggil; dicatat, tidak dipercaya untuk izin (R-API-09).
    on_behalf_of text CHECK (on_behalf_of IS NULL OR (on_behalf_of <> '' AND char_length(on_behalf_of) <= 200)),
    operations jsonb NOT NULL CHECK (jsonb_typeof(operations) = 'array'),
    applied_at timestamptz
);
GRANT SELECT, INSERT ON change_proposals TO revioten_app;
GRANT UPDATE (applied_at) ON change_proposals TO revioten_app;

CREATE TABLE export_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    service_account_id uuid NOT NULL REFERENCES service_accounts (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    revision smallint NOT NULL CHECK (revision >= 0),
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
    file_name text,
    file_sha256 text,
    size_bytes bigint,
    error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz,
    CONSTRAINT export_jobs_done_check CHECK ((status = 'done') = (file_sha256 IS NOT NULL))
);
CREATE INDEX export_jobs_queue_idx ON export_jobs (created_at) WHERE status = 'queued';
GRANT SELECT, INSERT ON export_jobs TO revioten_app;
GRANT UPDATE (status, file_name, file_sha256, size_bytes, error, started_at, finished_at) ON export_jobs TO revioten_app;

ALTER TABLE audit_log DROP CONSTRAINT audit_log_actor_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_actor_type_check CHECK (actor_type IN ('human', 'system', 'anonymous', 'service'));

-- +goose Down
ALTER TABLE audit_log DROP CONSTRAINT audit_log_actor_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_actor_type_check CHECK (actor_type IN ('human', 'system', 'anonymous')) NOT VALID;
DROP TABLE export_jobs;
DROP TABLE change_proposals;
ALTER TABLE changes
    DROP CONSTRAINT changes_integration_check,
    DROP CONSTRAINT changes_mechanism_check,
    ADD CONSTRAINT changes_mechanism_check CHECK (mechanism IN ('manual', 'yokoten', 'template', 'sqam')) NOT VALID,
    DROP CONSTRAINT changes_requester_check,
    DROP COLUMN requested_by_service_account_id;
DROP TABLE integration_requests;
DROP TABLE service_tokens;
DROP TABLE service_account_customers;
DROP TABLE service_accounts;
