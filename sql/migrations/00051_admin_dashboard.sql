-- Fase 4 · Dasbor admin: kesehatan web (D-205).
--
--   error_reports    setiap error server (HTTP 500) dicatat dengan request ID,
--                    alamat, dan pesan teknis yang sama dengan log server.
--                    Pesan error tidak memuat password, token, atau isi
--                    permintaan (R-SEC-14, R-API-13). Tambah saja.
--   problem_reports  laporan masalah dari pengguna lewat tombol "Report a
--                    problem": halaman, uraian, dan request ID terakhir.
--                    Admin mengubah statusnya: new, checking, done.

-- +goose Up
CREATE TABLE error_reports (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    request_id text NOT NULL,
    method text NOT NULL,
    path text NOT NULL CHECK (char_length(path) <= 500),
    account_id uuid REFERENCES accounts (id),
    message text NOT NULL CHECK (char_length(message) <= 2000)
);
CREATE INDEX error_reports_time_idx ON error_reports (occurred_at DESC);
GRANT SELECT, INSERT ON error_reports TO revioten_app;

CREATE TABLE problem_reports (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    created_at timestamptz NOT NULL DEFAULT now(),
    person_id uuid NOT NULL REFERENCES people (id),
    account_id uuid NOT NULL REFERENCES accounts (id),
    page text NOT NULL CHECK (char_length(page) <= 500),
    description text NOT NULL CHECK (btrim(description) <> '' AND char_length(description) <= 2000),
    last_request_id text CHECK (last_request_id IS NULL OR char_length(last_request_id) <= 100),
    status text NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'checking', 'done')),
    handled_by_person_id uuid REFERENCES people (id),
    handled_at timestamptz,
    admin_note text CHECK (admin_note IS NULL OR char_length(admin_note) <= 2000)
);
CREATE INDEX problem_reports_time_idx ON problem_reports (created_at DESC);
GRANT SELECT, INSERT ON problem_reports TO revioten_app;
GRANT UPDATE (status, handled_by_person_id, handled_at, admin_note) ON problem_reports TO revioten_app;

-- +goose Down
DROP TABLE problem_reports;
DROP TABLE error_reports;
