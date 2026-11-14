-- Tahap 7 · Migrasi dan bukti nilai (M7.1 sampai M7.7; R-MIG-01 sampai
-- R-MIG-07, R-REV-01, R-REV-05, D-69).
--
--   system_settings.migration_mode   saklar mode migrasi (R-MIG-06). Bawaannya
--                                    mati; admin menyalakannya selama migrasi.
--   documents.origin                 new atau migrated (brief bagian 4).
--   document_baselines               data hardcopy dokumen lama: revisi saat
--                                    ini, tanggal berlaku, penyusun, dan
--                                    approver sebagai teks (M7.1), lalu siapa
--                                    yang memverifikasi terhadap hardcopy dan
--                                    berapa menit usahanya (M7.5, M7.6).
--   document_history_entries         riwayat revisi sebelum baseline sebagai
--                                    teks dari tabel Excel lama (M7.2, R-REV-05).
--   document_revisions.migrated      revisi ini terbit lewat verifikasi
--                                    hardcopy: "Released (migrated)" (R-MIG-06).
--
-- Penjaga di database: baseline dan riwayat teks tidak bisa diubah lagi
-- sesudah diverifikasi terhadap hardcopy (SQLSTATE 55000).

-- +goose Up
ALTER TABLE system_settings ADD COLUMN migration_mode boolean NOT NULL DEFAULT false;

ALTER TABLE documents ADD COLUMN origin text NOT NULL DEFAULT 'new'
    CONSTRAINT documents_origin_check CHECK (origin IN ('new', 'migrated'));

ALTER TABLE document_revisions ADD COLUMN migrated boolean NOT NULL DEFAULT false;
GRANT UPDATE (migrated) ON document_revisions TO revioten_app;

CREATE TABLE document_baselines (
    document_id uuid PRIMARY KEY REFERENCES documents (id),
    revision smallint NOT NULL CHECK (revision >= 0),
    effective_date date NOT NULL,
    -- Nama di hardcopy, ditulis apa adanya; orangnya belum tentu punya akun.
    prepared_by text NOT NULL CHECK (prepared_by <> '' AND prepared_by = btrim(prepared_by) AND char_length(prepared_by) <= 200),
    approved_by text NOT NULL CHECK (approved_by <> '' AND approved_by = btrim(approved_by) AND char_length(approved_by) <= 500),
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    verified_by_person_id uuid REFERENCES people (id),
    verified_at timestamptz,
    -- Menit kerja migrasi dokumen ini, diisi orangnya saat verifikasi (M7.6).
    effort_minutes integer CHECK (effort_minutes IS NULL OR effort_minutes BETWEEN 1 AND 100000),
    CONSTRAINT document_baselines_verified_check CHECK ((verified_at IS NULL) = (verified_by_person_id IS NULL)),
    -- Pemeriksa hardcopy bukan orang yang memasukkan dokumen (R-SEC-11, ASUMSI E-113).
    CONSTRAINT document_baselines_other_person CHECK (verified_by_person_id IS NULL OR verified_by_person_id <> created_by_person_id)
);
GRANT SELECT, INSERT ON document_baselines TO revioten_app;
GRANT UPDATE (verified_by_person_id, verified_at, effort_minutes) ON document_baselines TO revioten_app;

CREATE TABLE document_history_entries (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES document_baselines (document_id),
    position integer NOT NULL CHECK (position >= 1),
    -- Teks revisi seperti di tabel lama, misalnya -03 atau C.
    revision_label text NOT NULL CHECK (revision_label <> '' AND revision_label = btrim(revision_label) AND char_length(revision_label) <= 20),
    entry_date date,
    description text NOT NULL CHECK (description <> '' AND description = btrim(description) AND char_length(description) <= 1000),
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_history_entries_position_key UNIQUE (document_id, position)
);
GRANT SELECT, INSERT, DELETE ON document_history_entries TO revioten_app;

-- Tautan baris ke poin template dipasang saat penautan (M7.4, R-MIG-05).
GRANT UPDATE (template_point_id, template_version_id) ON document_rows TO revioten_app;

-- +goose StatementBegin
CREATE FUNCTION document_baseline_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    doc_id uuid;
    verified timestamptz;
BEGIN
    IF TG_TABLE_NAME = 'document_baselines' THEN
        IF TG_OP = 'DELETE' OR OLD.verified_at IS NOT NULL THEN
            RAISE EXCEPTION 'baseline migrasi tidak bisa dihapus, dan tidak bisa diubah sesudah diverifikasi: % ditolak (R-MIG-03)', TG_OP USING ERRCODE = '55000';
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP = 'INSERT' THEN
        doc_id := NEW.document_id;
    ELSE
        doc_id := OLD.document_id;
    END IF;
    SELECT verified_at INTO verified FROM document_baselines WHERE document_id = doc_id;
    IF verified IS NOT NULL THEN
        RAISE EXCEPTION 'riwayat teks tidak bisa diubah sesudah baseline diverifikasi: % ditolak (R-REV-05)', TG_OP USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_baselines_reject_change BEFORE UPDATE OR DELETE ON document_baselines
    FOR EACH ROW EXECUTE FUNCTION document_baseline_reject_change();
CREATE TRIGGER document_history_entries_reject_change BEFORE INSERT OR UPDATE OR DELETE ON document_history_entries
    FOR EACH ROW EXECUTE FUNCTION document_baseline_reject_change();

-- +goose Down
DROP TRIGGER document_history_entries_reject_change ON document_history_entries;
DROP TRIGGER document_baselines_reject_change ON document_baselines;
DROP FUNCTION document_baseline_reject_change();
DROP TABLE document_history_entries;
DROP TABLE document_baselines;
REVOKE UPDATE (migrated) ON document_revisions FROM revioten_app;
ALTER TABLE document_revisions DROP COLUMN migrated;
ALTER TABLE documents DROP COLUMN origin;
ALTER TABLE system_settings DROP COLUMN migration_mode;
