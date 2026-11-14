-- M5.4 · Draft revisi (R-CHG-03, R-REV-01, R-REV-02, UC-42).
--
--   document_revisions   satu baris per revisi satu dokumen: nomor revisi,
--                        status (draft, lalu released di M5.5), pembuat, dan
--                        waktu. Satu dokumen paling banyak punya satu draft.
--   documents            revision sekarang berarti revisi yang sedang
--                        dikerjakan (draft) atau revisi terakhir yang terbit;
--                        released_revision adalah revisi yang berlaku.
--                        status 'released' berarti tidak ada draft terbuka.
--   change_documents     draft_revision_id: draft yang dihasilkan perubahan
--                        untuk dokumen itu; merged true bila perubahan masuk
--                        ke draft yang sudah terbuka (R-CHG-03).
--
-- Penjaga di database: baris dan header dokumen yang tidak punya draft
-- terbuka (status released) tidak bisa diubah. Aplikasi memeriksa ini lebih
-- dulu dengan pesan jelas; trigger ini pengaman terakhir (R-REV-02).

-- +goose Up
CREATE TABLE document_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES documents (id),
    revision smallint NOT NULL CHECK (revision >= 0),
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'released')),
    created_by_person_id uuid NOT NULL REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    released_at timestamptz,
    CONSTRAINT document_revisions_revision_key UNIQUE (document_id, revision),
    CONSTRAINT document_revisions_released_check CHECK ((status = 'released') = (released_at IS NOT NULL))
);
-- Satu draft terbuka per dokumen.
CREATE UNIQUE INDEX document_revisions_one_draft ON document_revisions (document_id) WHERE status = 'draft';
GRANT SELECT, INSERT ON document_revisions TO revioten_app;
GRANT UPDATE (status, released_at) ON document_revisions TO revioten_app;

-- Dokumen yang sudah ada adalah draft revisi awalnya.
INSERT INTO document_revisions (document_id, revision, status, created_by_person_id, created_at)
SELECT id, revision, 'draft', created_by_person_id, created_at FROM documents;

ALTER TABLE documents DROP CONSTRAINT documents_status_check;
ALTER TABLE documents ADD CONSTRAINT documents_status_check CHECK (status IN ('draft', 'released'));
ALTER TABLE documents ADD COLUMN released_revision smallint CHECK (released_revision IS NULL OR released_revision >= 0);
GRANT UPDATE (revision, status, released_revision) ON documents TO revioten_app;

ALTER TABLE change_documents
    ADD COLUMN draft_revision_id uuid REFERENCES document_revisions (id),
    ADD COLUMN merged boolean NOT NULL DEFAULT false;
GRANT UPDATE (draft_revision_id, merged) ON change_documents TO revioten_app;

-- +goose StatementBegin
CREATE FUNCTION document_reject_when_released() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    doc_id uuid;
    doc_status text;
BEGIN
    IF TG_TABLE_NAME = 'documents' THEN
        -- Header hanya bisa diubah selama ada draft terbuka.
        IF OLD.status = 'released' AND NEW.status = 'released'
           AND (NEW.header IS DISTINCT FROM OLD.header OR NEW.approval_route_id IS DISTINCT FROM OLD.approval_route_id) THEN
            RAISE EXCEPTION 'dokumen tanpa draft terbuka tidak bisa diubah (R-REV-02)' USING ERRCODE = '55000';
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP = 'DELETE' THEN
        doc_id := OLD.document_id;
    ELSE
        doc_id := NEW.document_id;
    END IF;
    SELECT status INTO doc_status FROM documents WHERE id = doc_id;
    IF doc_status = 'released' THEN
        RAISE EXCEPTION 'baris dokumen tanpa draft terbuka tidak bisa diubah: % ditolak (R-REV-02)', TG_OP USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_rows_reject_when_released BEFORE INSERT OR UPDATE OR DELETE ON document_rows
    FOR EACH ROW EXECUTE FUNCTION document_reject_when_released();
CREATE TRIGGER documents_reject_when_released BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE FUNCTION document_reject_when_released();

-- +goose Down
DROP TRIGGER documents_reject_when_released ON documents;
DROP TRIGGER document_rows_reject_when_released ON document_rows;
DROP FUNCTION document_reject_when_released();
REVOKE UPDATE (draft_revision_id, merged) ON change_documents FROM revioten_app;
ALTER TABLE change_documents DROP COLUMN merged, DROP COLUMN draft_revision_id;
REVOKE UPDATE (revision, status, released_revision) ON documents FROM revioten_app;
ALTER TABLE documents DROP COLUMN released_revision;
ALTER TABLE documents DROP CONSTRAINT documents_status_check;
ALTER TABLE documents ADD CONSTRAINT documents_status_check CHECK (status IN ('draft'));
DROP TABLE document_revisions;
