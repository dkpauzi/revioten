-- M5.5 · Snapshot rilis (R-REV-02, R-APR-02, R-SEC-11, brief bagian 17).
--
-- Saat revisi dicatat terbit, isi dokumen (header, model, part number,
-- bagian, dan semua baris) dibekukan sebagai teks JSON di snapshot, beserta
-- SHA-256-nya. CHECK memastikan hash selalu cocok dengan isi, dan trigger
-- menolak setiap perubahan atau penghapusan revisi yang sudah terbit.
--
-- Hook portal customer (brief bagian 17, DEFERRED): customer_visible dan
-- portal_published_at disiapkan, belum dipakai dan belum bisa diubah aplikasi.

-- +goose Up
ALTER TABLE document_revisions
    ADD COLUMN snapshot text,
    ADD COLUMN snapshot_sha256 text,
    ADD COLUMN released_by_person_id uuid REFERENCES people (id),
    ADD COLUMN released_by_account_id uuid REFERENCES accounts (id),
    ADD COLUMN customer_visible boolean NOT NULL DEFAULT false,
    ADD COLUMN portal_published_at timestamptz,
    ADD CONSTRAINT document_revisions_snapshot_check CHECK (
        (status = 'released') = (snapshot IS NOT NULL)
        AND (snapshot IS NULL) = (snapshot_sha256 IS NULL)
        AND (snapshot IS NULL) = (released_by_person_id IS NULL)
        AND (snapshot IS NULL) = (released_by_account_id IS NULL)),
    -- Hash dihitung ulang database dari isi snapshot, jadi tidak bisa palsu.
    ADD CONSTRAINT document_revisions_sha256_check CHECK (
        snapshot IS NULL OR snapshot_sha256 = encode(sha256(convert_to(snapshot, 'UTF8')), 'hex'));
GRANT UPDATE (snapshot, snapshot_sha256, released_by_person_id, released_by_account_id) ON document_revisions TO revioten_app;

-- +goose StatementBegin
CREATE FUNCTION document_revision_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'released' THEN
        RAISE EXCEPTION 'revisi yang sudah terbit tidak bisa diubah: % ditolak (R-REV-02)', TG_OP USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_revisions_reject_change BEFORE UPDATE OR DELETE ON document_revisions
    FOR EACH ROW EXECUTE FUNCTION document_revision_reject_change();

-- +goose Down
DROP TRIGGER document_revisions_reject_change ON document_revisions;
DROP FUNCTION document_revision_reject_change();
REVOKE UPDATE (snapshot, snapshot_sha256, released_by_person_id, released_by_account_id) ON document_revisions FROM revioten_app;
ALTER TABLE document_revisions
    DROP CONSTRAINT document_revisions_sha256_check,
    DROP CONSTRAINT document_revisions_snapshot_check,
    DROP COLUMN portal_published_at, DROP COLUMN customer_visible,
    DROP COLUMN released_by_account_id, DROP COLUMN released_by_person_id,
    DROP COLUMN snapshot_sha256, DROP COLUMN snapshot;
