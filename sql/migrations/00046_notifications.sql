-- D-194 · Email notifikasi.
--
--   people.email                 alamat email orang (opsional). Diisi admin
--                                (A-1) atau oleh orangnya sendiri (D-104).
--   notification_events          jenis peristiwa yang bisa dikirimi email,
--                                dan saklar nyala-mati per jenis (admin, A-3).
--   notification_outbox          kotak keluar: satu baris per email per
--                                penerima, ditulis dalam transaksi yang sama
--                                dengan peristiwanya. Pekerja di latar
--                                belakang mengirimnya lewat Microsoft 365.
--   service_tokens.expiry_notified_at  token yang segera kedaluwarsa hanya
--                                diberitahukan sekali.
--
-- Isi email tidak pernah memuat password, password sementara, atau token.

-- +goose Up
ALTER TABLE people ADD COLUMN email text
    CONSTRAINT people_email_check CHECK (email IS NULL OR (char_length(email) <= 254 AND email = btrim(email) AND email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$'));
GRANT UPDATE (email) ON people TO revioten_app;

CREATE TABLE notification_events (
    key text PRIMARY KEY,
    position smallint NOT NULL,
    enabled boolean NOT NULL DEFAULT true
);
INSERT INTO notification_events (key, position) VALUES
    ('change_submitted', 1),       -- perubahan diajukan: verifikator customer itu
    ('change_verified', 2),        -- perubahan diterima atau ditolak: pemohon
    ('release_ready', 3),          -- scan bertanda tangan diunggah: dokumen kontrol
    ('revision_released', 4),      -- revisi terbit: pemohon perubahan di revisi itu
    ('form_approval_waiting', 5),  -- form set atau tabel AP menunggu persetujuan lain
    ('yokoten_created', 6),        -- yokoten dibuat: Project Leader dan Engineer target
    ('migration_waiting', 7),      -- dokumen lama masuk: petugas migrasi lain
    ('token_expiring', 8),         -- token integrasi kedaluwarsa dalam 7 hari: admin
    ('test', 9);                   -- email uji dari halaman Notifications
GRANT SELECT ON notification_events TO revioten_app;
GRANT UPDATE (enabled) ON notification_events TO revioten_app;

CREATE TABLE notification_outbox (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    event text NOT NULL REFERENCES notification_events (key),
    recipient_person_id uuid NOT NULL REFERENCES people (id),
    recipient_email text,
    subject text NOT NULL CHECK (subject <> '' AND char_length(subject) <= 300),
    body text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'skipped')),
    -- Alasan dilewati atau gagal, untuk layar admin. Tidak memuat rahasia.
    reason text,
    attempts smallint NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_at timestamptz,
    CONSTRAINT notification_outbox_sent_check CHECK ((status = 'sent') = (sent_at IS NOT NULL))
);
CREATE INDEX notification_outbox_queue_idx ON notification_outbox (next_attempt_at) WHERE status = 'pending';
CREATE INDEX notification_outbox_recent_idx ON notification_outbox (created_at DESC);
GRANT SELECT, INSERT ON notification_outbox TO revioten_app;
GRANT UPDATE (status, reason, attempts, next_attempt_at, sent_at) ON notification_outbox TO revioten_app;

ALTER TABLE service_tokens ADD COLUMN expiry_notified_at timestamptz;
GRANT UPDATE (expiry_notified_at) ON service_tokens TO revioten_app;

-- +goose Down
REVOKE UPDATE (expiry_notified_at) ON service_tokens FROM revioten_app;
ALTER TABLE service_tokens DROP COLUMN expiry_notified_at;
DROP TABLE notification_outbox;
DROP TABLE notification_events;
REVOKE UPDATE (email) ON people FROM revioten_app;
ALTER TABLE people DROP COLUMN email;
