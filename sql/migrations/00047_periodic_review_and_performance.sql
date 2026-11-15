-- D-138 dan D-196 · Review berkala 3 tahun; D-197 · laporan kinerja PPAP Engineer.
--
--   system_settings.review_period_months   lama satu siklus review (bawaan 36
--                                          bulan, WI general control plan).
--   system_settings.review_warning_days    peringatan kuning mulai N hari
--                                          sebelum jatuh tempo (bawaan 90).
--   change_sources 'Periodic review'       pemicu untuk "review dengan perubahan".
--   periodic_reviews                       catatan "review tanpa perubahan" per
--                                          set dokumen: dicatat satu orang
--                                          (E-3), dikonfirmasi orang kedua (V-1).
--                                          Hanya catatan terkonfirmasi yang
--                                          memulai lagi jam 3 tahun (ASUMSI E-86).
--   periodic_review_notices                email jatuh tempo dikirim sekali per
--                                          set per tahap (akan jatuh tempo,
--                                          lewat jatuh tempo).
--   permissions L-5                        lihat laporan kinerja (D-197).

-- +goose Up
ALTER TABLE system_settings
    ADD COLUMN review_period_months smallint NOT NULL DEFAULT 36 CHECK (review_period_months BETWEEN 1 AND 120),
    ADD COLUMN review_warning_days smallint NOT NULL DEFAULT 90 CHECK (review_warning_days BETWEEN 0 AND 365);

INSERT INTO change_sources (name, mechanism) VALUES ('Periodic review', NULL)
ON CONFLICT DO NOTHING;

CREATE TABLE periodic_reviews (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    seq bigint GENERATED ALWAYS AS IDENTITY,
    -- Set dokumen: PFD, PFMEA, dan CP satu customer dengan nomor berjalan yang
    -- sama (M2.4). Kuncinya kode customer di nomor dan nomor berjalan, sama
    -- seperti keunikan register (kode customer bisa berganti, D-88). Nomor
    -- lama hasil impor juga punya nomor berjalan.
    customer_id uuid NOT NULL REFERENCES customers (id),
    customer_code text NOT NULL,
    sequence integer NOT NULL,
    reviewed_on date NOT NULL,
    participants text NOT NULL CHECK (participants <> '' AND participants = btrim(participants) AND char_length(participants) <= 500),
    conclusion text NOT NULL CHECK (conclusion <> '' AND conclusion = btrim(conclusion) AND char_length(conclusion) <= 2000),
    -- Daftar periksa yang dijawab: kunci butir yang dicentang.
    checklist jsonb NOT NULL CHECK (jsonb_typeof(checklist) = 'array'),
    -- Revisi yang direview: nomor, revisi, dan SHA-256 snapshot setiap dokumen.
    revisions jsonb NOT NULL CHECK (jsonb_typeof(revisions) = 'array'),
    status text NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted', 'confirmed')),
    recorded_by_person_id uuid NOT NULL REFERENCES people (id),
    recorded_by_account_id uuid NOT NULL REFERENCES accounts (id),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    confirmed_by_person_id uuid REFERENCES people (id),
    confirmed_by_account_id uuid REFERENCES accounts (id),
    confirmed_at timestamptz,
    CONSTRAINT periodic_reviews_seq_key UNIQUE (seq),
    CONSTRAINT periodic_reviews_confirmed_check CHECK (
        (status = 'confirmed') = (confirmed_at IS NOT NULL)
        AND (confirmed_at IS NULL) = (confirmed_by_person_id IS NULL)
        AND (confirmed_at IS NULL) = (confirmed_by_account_id IS NULL)),
    -- Orang yang mencatat tidak mengonfirmasi catatannya sendiri (R-SEC-11).
    CONSTRAINT periodic_reviews_second_person CHECK (confirmed_by_person_id IS NULL OR confirmed_by_person_id <> recorded_by_person_id)
);
-- Paling banyak satu catatan menunggu konfirmasi per set.
CREATE UNIQUE INDEX periodic_reviews_one_open ON periodic_reviews (customer_code, sequence) WHERE status = 'submitted';
CREATE INDEX periodic_reviews_set_idx ON periodic_reviews (customer_code, sequence, reviewed_on DESC);
GRANT SELECT, INSERT ON periodic_reviews TO revioten_app;
GRANT UPDATE (status, confirmed_by_person_id, confirmed_by_account_id, confirmed_at) ON periodic_reviews TO revioten_app;

-- Catatan yang sudah dikonfirmasi tidak bisa diubah atau dihapus.
-- +goose StatementBegin
CREATE FUNCTION periodic_reviews_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' OR OLD.status = 'confirmed' THEN
        RAISE EXCEPTION 'a confirmed periodic review cannot be changed' USING ERRCODE = '55000';
    END IF;
    IF NEW.customer_code <> OLD.customer_code OR NEW.sequence <> OLD.sequence OR NEW.reviewed_on <> OLD.reviewed_on OR NEW.participants <> OLD.participants
       OR NEW.conclusion <> OLD.conclusion OR NEW.checklist <> OLD.checklist OR NEW.revisions <> OLD.revisions THEN
        RAISE EXCEPTION 'the content of a periodic review cannot be changed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER periodic_reviews_guard BEFORE UPDATE OR DELETE ON periodic_reviews
    FOR EACH ROW EXECUTE FUNCTION periodic_reviews_guard();

CREATE TABLE periodic_review_notices (
    customer_code text NOT NULL,
    sequence integer NOT NULL,
    due_on date NOT NULL,
    stage text NOT NULL CHECK (stage IN ('due_soon', 'overdue')),
    sent_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (customer_code, sequence, due_on, stage)
);
GRANT SELECT, INSERT ON periodic_review_notices TO revioten_app;

INSERT INTO notification_events (key, position) VALUES
    ('review_due', 9),        -- set dokumen akan atau sudah lewat jatuh tempo review: penyusun customer itu
    ('review_waiting', 10)    -- catatan review tanpa perubahan menunggu konfirmasi: verifikator
ON CONFLICT DO NOTHING;
UPDATE notification_events SET position = 11 WHERE key = 'test';

-- Hak baru: lihat laporan kinerja (D-197). Bawaannya untuk PPAP Engineer
-- (Penyusun) dan supervisornya (Penyetuju form); admin bisa mengubahnya di Roles.
UPDATE permissions SET sort_order = sort_order + 1 WHERE sort_order >= 5;
INSERT INTO permissions (code, level, sort_order) VALUES ('L-5', 'view', 5);
INSERT INTO role_permissions (role_id, permission_code) VALUES
    ('author', 'L-5'), ('form_approver', 'L-5');

-- +goose Down
DELETE FROM role_permissions WHERE permission_code = 'L-5';
DELETE FROM permissions WHERE code = 'L-5';
UPDATE permissions SET sort_order = sort_order - 1 WHERE sort_order > 5;
UPDATE notification_events SET position = 9 WHERE key = 'test';
DELETE FROM notification_outbox WHERE event IN ('review_due', 'review_waiting');
DELETE FROM notification_events WHERE key IN ('review_due', 'review_waiting');
DROP TABLE periodic_review_notices;
DROP TRIGGER periodic_reviews_guard ON periodic_reviews;
DROP FUNCTION periodic_reviews_guard();
DROP TABLE periodic_reviews;
DELETE FROM change_sources WHERE name = 'Periodic review' AND NOT EXISTS (SELECT 1 FROM changes c WHERE c.trigger_source_id = change_sources.id);
ALTER TABLE system_settings DROP COLUMN review_warning_days, DROP COLUMN review_period_months;
