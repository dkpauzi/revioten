-- M3.4: versi template terbit dan tidak bisa diubah (R-TPL-05).
--
-- Versi format template dan process template berjalan dari draft ke
-- published. Sesudah terbit, isi versi itu tidak bisa diubah siapa pun; untuk
-- perubahan berikutnya dibuat versi baru yang menyalin isinya. Propagasi ke
-- dokumen belum ada (M5.8).
--
-- Tiga penjaga di database, supaya aturan ini tidak bergantung pada kode
-- layanan saja (pola yang sama dengan audit log, R-SEC-03):
--   1. CHECK: status published selalu punya waktu terbit, dan sebaliknya.
--   2. Indeks unik: satu template paling banyak punya satu versi draft.
--   3. Trigger: baris versi terbit dan seluruh isinya menolak INSERT, UPDATE,
--      DELETE, dan TRUNCATE, juga untuk pemilik database.
--
-- Catatan untuk migrasi berikutnya: mengisi kolom baru pada versi terbit
-- (backfill) akan ditolak trigger ini. Migrasi seperti itu harus mematikan
-- triggernya dengan sengaja (ALTER TABLE ... DISABLE TRIGGER), lalu
-- menyalakannya lagi, dan alasannya ditulis di migrasinya.

-- +goose Up
ALTER TABLE format_template_versions
    ADD COLUMN published_at timestamptz,
    ADD CONSTRAINT format_template_versions_published_check
        CHECK ((status = 'published') = (published_at IS NOT NULL));
CREATE UNIQUE INDEX format_template_versions_draft_key
    ON format_template_versions (template_id) WHERE status = 'draft';

ALTER TABLE process_template_versions
    ADD COLUMN published_at timestamptz,
    ADD CONSTRAINT process_template_versions_published_check
        CHECK ((status = 'published') = (published_at IS NOT NULL));
CREATE UNIQUE INDEX process_template_versions_draft_key
    ON process_template_versions (template_id) WHERE status = 'draft';

-- SQLSTATE 55000 (object_not_in_prerequisite_state) dipakai ketiga fungsi di
-- bawah, supaya test bisa membedakannya dari penolakan hak (42501).

-- Baris versi: yang sudah terbit tidak bisa diubah atau dihapus. Menerbitkan
-- (draft menjadi published) tetap lewat, karena status lamanya draft.
-- +goose StatementBegin
CREATE FUNCTION template_version_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'published' THEN
        RAISE EXCEPTION 'versi template yang sudah terbit tidak bisa diubah: % pada % ditolak (R-TPL-05)', TG_OP, TG_TABLE_NAME
            USING ERRCODE = 'object_not_in_prerequisite_state';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

-- Isi versi (field, sub-proses, poin, dan seterusnya): ditolak bila versinya
-- sudah terbit. Nama tabel versinya diberikan lewat argumen trigger.
-- Untuk UPDATE, versi lama dan versi baru baris itu sama-sama diperiksa,
-- supaya baris tidak bisa dipindah keluar dari versi terbit.
-- +goose StatementBegin
CREATE FUNCTION template_content_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    version_ids uuid[];
    found boolean;
BEGIN
    IF TG_OP = 'INSERT' THEN
        version_ids := ARRAY[NEW.version_id];
    ELSIF TG_OP = 'DELETE' THEN
        version_ids := ARRAY[OLD.version_id];
    ELSE
        version_ids := ARRAY[OLD.version_id, NEW.version_id];
    END IF;
    EXECUTE format('SELECT EXISTS (SELECT 1 FROM %I WHERE id = ANY($1) AND status = ''published'')', TG_ARGV[0])
        INTO found USING version_ids;
    IF found THEN
        RAISE EXCEPTION 'isi versi template yang sudah terbit tidak bisa diubah: % pada % ditolak (R-TPL-05)', TG_OP, TG_TABLE_NAME
            USING ERRCODE = 'object_not_in_prerequisite_state';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd

-- TRUNCATE tidak menjalankan trigger per baris, jadi ditolak seluruhnya.
-- +goose StatementBegin
CREATE FUNCTION template_reject_truncate() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'tabel versi template tidak bisa dikosongkan: TRUNCATE pada % ditolak (R-TPL-05)', TG_TABLE_NAME
        USING ERRCODE = 'object_not_in_prerequisite_state';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER format_template_versions_published
    BEFORE UPDATE OR DELETE ON format_template_versions
    FOR EACH ROW EXECUTE FUNCTION template_version_reject_change();
CREATE TRIGGER process_template_versions_published
    BEFORE UPDATE OR DELETE ON process_template_versions
    FOR EACH ROW EXECUTE FUNCTION template_version_reject_change();

CREATE TRIGGER format_version_fields_published
    BEFORE INSERT OR UPDATE OR DELETE ON format_version_fields
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('format_template_versions');
CREATE TRIGGER format_rating_criteria_published
    BEFORE INSERT OR UPDATE OR DELETE ON format_rating_criteria
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('format_template_versions');
CREATE TRIGGER format_symbols_published
    BEFORE INSERT OR UPDATE OR DELETE ON format_symbols
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('format_template_versions');

CREATE TRIGGER process_template_subprocesses_published
    BEFORE INSERT OR UPDATE OR DELETE ON process_template_subprocesses
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('process_template_versions');
CREATE TRIGGER process_template_points_published
    BEFORE INSERT OR UPDATE OR DELETE ON process_template_points
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('process_template_versions');
CREATE TRIGGER process_template_variants_published
    BEFORE INSERT OR UPDATE OR DELETE ON process_template_variants
    FOR EACH ROW EXECUTE FUNCTION template_content_reject_change('process_template_versions');

CREATE TRIGGER format_template_versions_truncate
    BEFORE TRUNCATE ON format_template_versions FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER format_version_fields_truncate
    BEFORE TRUNCATE ON format_version_fields FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER format_rating_criteria_truncate
    BEFORE TRUNCATE ON format_rating_criteria FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER format_symbols_truncate
    BEFORE TRUNCATE ON format_symbols FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER process_template_versions_truncate
    BEFORE TRUNCATE ON process_template_versions FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER process_template_subprocesses_truncate
    BEFORE TRUNCATE ON process_template_subprocesses FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER process_template_points_truncate
    BEFORE TRUNCATE ON process_template_points FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();
CREATE TRIGGER process_template_variants_truncate
    BEFORE TRUNCATE ON process_template_variants FOR EACH STATEMENT EXECUTE FUNCTION template_reject_truncate();

-- Role aplikasi hanya mendapat kolom yang diubah saat menerbitkan (E-06).
-- Process template sudah punya UPDATE (status) sejak migrasi 00012.
GRANT UPDATE (status, published_at) ON format_template_versions TO revioten_app;
GRANT UPDATE (published_at) ON process_template_versions TO revioten_app;

-- +goose Down
REVOKE UPDATE (status, published_at) ON format_template_versions FROM revioten_app;
REVOKE UPDATE (published_at) ON process_template_versions FROM revioten_app;

DROP TRIGGER process_template_variants_truncate ON process_template_variants;
DROP TRIGGER process_template_points_truncate ON process_template_points;
DROP TRIGGER process_template_subprocesses_truncate ON process_template_subprocesses;
DROP TRIGGER process_template_versions_truncate ON process_template_versions;
DROP TRIGGER format_symbols_truncate ON format_symbols;
DROP TRIGGER format_rating_criteria_truncate ON format_rating_criteria;
DROP TRIGGER format_version_fields_truncate ON format_version_fields;
DROP TRIGGER format_template_versions_truncate ON format_template_versions;

DROP TRIGGER process_template_variants_published ON process_template_variants;
DROP TRIGGER process_template_points_published ON process_template_points;
DROP TRIGGER process_template_subprocesses_published ON process_template_subprocesses;
DROP TRIGGER format_symbols_published ON format_symbols;
DROP TRIGGER format_rating_criteria_published ON format_rating_criteria;
DROP TRIGGER format_version_fields_published ON format_version_fields;
DROP TRIGGER process_template_versions_published ON process_template_versions;
DROP TRIGGER format_template_versions_published ON format_template_versions;

DROP FUNCTION template_reject_truncate();
DROP FUNCTION template_content_reject_change();
DROP FUNCTION template_version_reject_change();

DROP INDEX process_template_versions_draft_key;
DROP INDEX format_template_versions_draft_key;

-- Versi yang sudah terbit kembali menjadi draft, karena skema lama tidak
-- menyimpan waktu terbit. Bagian Down hanya untuk pengembangan.
ALTER TABLE process_template_versions DROP CONSTRAINT process_template_versions_published_check;
ALTER TABLE format_template_versions DROP CONSTRAINT format_template_versions_published_check;
UPDATE process_template_versions SET status = 'draft' WHERE status = 'published';
UPDATE format_template_versions SET status = 'draft' WHERE status = 'published';
ALTER TABLE process_template_versions DROP COLUMN published_at;
ALTER TABLE format_template_versions DROP COLUMN published_at;
