-- Fase 3 · Tanda tangan digital (D-198, D-203, D-204).
--
--   customers.signature_mode  digital (bawaan): boleh tanda tangan digital;
--                             wet_only: customer mewajibkan tanda tangan basah.
--   signature_requests        satu permintaan tanda tangan untuk satu revisi
--                             draft: isi dibekukan (snapshot dan SHA-256), rute
--                             ikut dibekukan, tanggal rilis = tanggal dikirim
--                             (D-204). Status: waiting, completed, rejected,
--                             withdrawn. Hanya status dan penutupnya yang bisa
--                             berubah, sekali, dari waiting.
--   signature_slots           penanda tangan yang dipilih untuk setiap slot
--                             rute, dicatat saat dikirim. Tidak bisa diubah.
--   signatures                tanda tangan yang diberikan. Tidak bisa diubah
--                             atau dihapus siapa pun (seperti audit log).
--   penguncian draft          selama permintaan menunggu atau sudah lengkap,
--                             isi draft (baris, bagian, header, rute) tidak
--                             bisa diubah (R-REV-06, R-APR-11). Ditolak atau
--                             ditarik kembali membuka kuncinya.

-- +goose Up
ALTER TABLE customers ADD COLUMN signature_mode text NOT NULL DEFAULT 'digital'
    CHECK (signature_mode IN ('digital', 'wet_only'));
GRANT UPDATE (signature_mode) ON customers TO revioten_app;

CREATE TABLE signature_requests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES documents (id),
    revision_id uuid NOT NULL REFERENCES document_revisions (id),
    revision smallint NOT NULL CHECK (revision >= 0),
    customer_id uuid NOT NULL REFERENCES customers (id),
    route_id uuid NOT NULL REFERENCES approval_routes (id),
    route_name text NOT NULL,
    snapshot text NOT NULL,
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    release_date date NOT NULL,
    note text CHECK (note IS NULL OR char_length(note) <= 1000),
    status text NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'completed', 'rejected', 'withdrawn')),
    sent_by_person_id uuid NOT NULL REFERENCES people (id),
    sent_by_account_id uuid NOT NULL REFERENCES accounts (id),
    sent_at timestamptz NOT NULL DEFAULT now(),
    closed_at timestamptz,
    closed_by_person_id uuid REFERENCES people (id),
    reject_reason text CHECK (reject_reason IS NULL OR (btrim(reject_reason) <> '' AND char_length(reject_reason) <= 1000)),
    CONSTRAINT signature_requests_closed_check CHECK ((status = 'waiting') = (closed_at IS NULL)),
    CONSTRAINT signature_requests_reason_check CHECK ((status = 'rejected') = (reject_reason IS NOT NULL))
);
-- Satu permintaan berjalan atau lengkap per revisi.
CREATE UNIQUE INDEX signature_requests_active ON signature_requests (revision_id) WHERE status IN ('waiting', 'completed');
CREATE INDEX signature_requests_document_idx ON signature_requests (document_id, sent_at DESC);
GRANT SELECT, INSERT ON signature_requests TO revioten_app;
GRANT UPDATE (status, closed_at, closed_by_person_id, reject_reason) ON signature_requests TO revioten_app;

-- Isi dokumen: disaring seperti dokumennya (RLS, M8.1).
ALTER TABLE signature_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY signature_requests_customer_scope ON signature_requests TO revioten_app
    USING (EXISTS (SELECT 1 FROM documents d WHERE d.id = signature_requests.document_id));

CREATE TABLE signature_slots (
    request_id uuid NOT NULL REFERENCES signature_requests (id),
    slot_order smallint NOT NULL CHECK (slot_order >= 1),
    stage text NOT NULL CHECK (stage IN ('prepared', 'checked', 'approved')),
    print_title text NOT NULL,
    position_name text NOT NULL,
    person_id uuid NOT NULL REFERENCES people (id),
    person_name text NOT NULL,
    PRIMARY KEY (request_id, slot_order),
    -- Satu orang hanya satu slot dalam satu rute (R-APR-12).
    CONSTRAINT signature_slots_one_slot_per_person UNIQUE (request_id, person_id)
);
GRANT SELECT, INSERT ON signature_slots TO revioten_app;

CREATE TABLE signatures (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    request_id uuid NOT NULL,
    slot_order smallint NOT NULL,
    person_id uuid NOT NULL REFERENCES people (id),
    account_id uuid NOT NULL REFERENCES accounts (id),
    person_name text NOT NULL,
    print_title text NOT NULL,
    stage text NOT NULL,
    content_sha256 text NOT NULL,
    signed_at timestamptz NOT NULL DEFAULT now(),
    source_ip text,
    CONSTRAINT signatures_slot_fkey FOREIGN KEY (request_id, slot_order) REFERENCES signature_slots (request_id, slot_order),
    CONSTRAINT signatures_one_per_slot UNIQUE (request_id, slot_order)
);
GRANT SELECT, INSERT ON signatures TO revioten_app;

-- Revisi yang terbit lewat tanda tangan digital menyimpan permintaannya, dan
-- tidak butuh scan (D-198). Trigger memastikan permintaannya lengkap dan
-- milik revisi itu.
ALTER TABLE document_revisions ADD COLUMN signature_request_id uuid REFERENCES signature_requests (id);
GRANT UPDATE (signature_request_id) ON document_revisions TO revioten_app;
ALTER TABLE document_revisions DROP CONSTRAINT document_revisions_released_scan_check;
ALTER TABLE document_revisions ADD CONSTRAINT document_revisions_released_scan_check
    CHECK (status <> 'released' OR migrated OR scan_sha256 IS NOT NULL OR signature_request_id IS NOT NULL) NOT VALID;

-- +goose StatementBegin
CREATE FUNCTION revision_signature_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.signature_request_id IS NOT NULL AND NEW.signature_request_id IS DISTINCT FROM OLD.signature_request_id
       AND NOT EXISTS (SELECT 1 FROM signature_requests sr
                       WHERE sr.id = NEW.signature_request_id AND sr.revision_id = NEW.id AND sr.status = 'completed') THEN
        RAISE EXCEPTION 'revisi % hanya bisa memakai permintaan tanda tangan miliknya yang sudah lengkap (D-198)', NEW.id USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER document_revisions_signature_guard BEFORE UPDATE OF signature_request_id ON document_revisions
    FOR EACH ROW EXECUTE FUNCTION revision_signature_guard();

-- +goose StatementBegin
-- signature_append_only: slot dan tanda tangan tidak pernah diubah atau
-- dihapus, juga oleh pemilik database (brief bagian 8).
CREATE FUNCTION signature_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% tidak bisa diubah atau dihapus (D-198)', TG_TABLE_NAME USING ERRCODE = '55000';
END;
$$;

-- signature_request_guard: hanya status dan penutupnya yang berubah, sekali,
-- dari waiting ke completed, rejected, atau withdrawn.
CREATE FUNCTION signature_request_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'permintaan tanda tangan tidak dihapus (D-198)' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'waiting' THEN
        RAISE EXCEPTION 'permintaan tanda tangan % sudah selesai (%)', OLD.id, OLD.status USING ERRCODE = '55000';
    END IF;
    IF (NEW.id, NEW.document_id, NEW.revision_id, NEW.revision, NEW.customer_id, NEW.route_id, NEW.route_name, NEW.snapshot,
        NEW.content_sha256, NEW.release_date, NEW.note, NEW.sent_by_person_id, NEW.sent_by_account_id, NEW.sent_at)
       IS DISTINCT FROM
       (OLD.id, OLD.document_id, OLD.revision_id, OLD.revision, OLD.customer_id, OLD.route_id, OLD.route_name, OLD.snapshot,
        OLD.content_sha256, OLD.release_date, OLD.note, OLD.sent_by_person_id, OLD.sent_by_account_id, OLD.sent_at) THEN
        RAISE EXCEPTION 'isi permintaan tanda tangan tidak bisa diubah (D-198)' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

-- signature_freeze: isi draft yang sedang atau sudah ditandatangani tidak
-- bisa diubah (R-REV-06). Layanan Go menolak lebih dulu dengan pesan yang
-- jelas; trigger ini penjaga terakhir.
CREATE FUNCTION signature_frozen(doc uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
    SELECT EXISTS (
        SELECT 1 FROM signature_requests sr
        JOIN document_revisions r ON r.id = sr.revision_id
        WHERE sr.document_id = doc AND sr.status IN ('waiting', 'completed') AND r.status = 'draft')
$$;

CREATE FUNCTION signature_freeze_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    doc uuid;
BEGIN
    IF TG_TABLE_NAME = 'documents' THEN
        doc := NEW.id;
    ELSIF TG_OP = 'DELETE' THEN
        doc := OLD.document_id;
    ELSE
        doc := NEW.document_id;
    END IF;
    IF signature_frozen(doc) THEN
        RAISE EXCEPTION 'draft dokumen % sedang atau sudah ditandatangani dan tidak bisa diubah (R-REV-06)', doc USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER signature_slots_append_only BEFORE UPDATE OR DELETE ON signature_slots
    FOR EACH ROW EXECUTE FUNCTION signature_append_only();
CREATE TRIGGER signatures_append_only BEFORE UPDATE OR DELETE ON signatures
    FOR EACH ROW EXECUTE FUNCTION signature_append_only();
CREATE TRIGGER signature_requests_guard BEFORE UPDATE OR DELETE ON signature_requests
    FOR EACH ROW EXECUTE FUNCTION signature_request_guard();
CREATE TRIGGER document_rows_signature_freeze BEFORE INSERT OR UPDATE OR DELETE ON document_rows
    FOR EACH ROW EXECUTE FUNCTION signature_freeze_guard();
CREATE TRIGGER document_sections_signature_freeze BEFORE INSERT OR UPDATE OR DELETE ON document_sections
    FOR EACH ROW EXECUTE FUNCTION signature_freeze_guard();
CREATE TRIGGER documents_signature_freeze BEFORE UPDATE OF header, header_version, approval_route_id, route_reason ON documents
    FOR EACH ROW EXECUTE FUNCTION signature_freeze_guard();

-- Email: permintaan tanda tangan ke penanda tangan berikutnya, penolakan ke
-- pengirim, dan tanda tangan lengkap ke pencatat rilis (D-194).
INSERT INTO notification_events (key, position) VALUES
    ('signature_requested', 11),
    ('signature_rejected', 12),
    ('signature_completed', 13)
ON CONFLICT DO NOTHING;
UPDATE notification_events SET position = 14 WHERE key = 'test';

-- +goose Down
UPDATE notification_events SET position = 11 WHERE key = 'test';
DELETE FROM notification_outbox WHERE event IN ('signature_requested', 'signature_rejected', 'signature_completed');
DELETE FROM notification_events WHERE key IN ('signature_requested', 'signature_rejected', 'signature_completed');
DROP TRIGGER documents_signature_freeze ON documents;
DROP TRIGGER document_revisions_signature_guard ON document_revisions;
DROP FUNCTION revision_signature_guard();
ALTER TABLE document_revisions DROP CONSTRAINT document_revisions_released_scan_check;
ALTER TABLE document_revisions ADD CONSTRAINT document_revisions_released_scan_check
    CHECK (status <> 'released' OR migrated OR scan_sha256 IS NOT NULL) NOT VALID;
REVOKE UPDATE (signature_request_id) ON document_revisions FROM revioten_app;
ALTER TABLE document_revisions DROP COLUMN signature_request_id;
DROP TRIGGER document_sections_signature_freeze ON document_sections;
DROP TRIGGER document_rows_signature_freeze ON document_rows;
DROP TABLE signatures;
DROP TABLE signature_slots;
DROP TABLE signature_requests;
DROP FUNCTION signature_freeze_guard();
DROP FUNCTION signature_frozen(uuid);
DROP FUNCTION signature_request_guard();
DROP FUNCTION signature_append_only();
REVOKE UPDATE (signature_mode) ON customers FROM revioten_app;
ALTER TABLE customers DROP COLUMN signature_mode;
