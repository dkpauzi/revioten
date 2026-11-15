-- M8.1 · Peran, hak akses, dan row-level security (R-SEC-02, R-SEC-10,
-- R-SEC-11, R-SEC-12, D-66, D-71, RK-14, RK-63).
--
--   role_permissions     role aplikasi boleh menambah dan menghapus isi peran,
--                        karena admin sekarang mengubahnya di layar (D-66).
--                        Trigger menjaga aturan yang tidak bisa dimatikan:
--                        peran admin dan viewer tetap, peran kerja tidak
--                        pernah memegang hak admin (A-), dan hak dasar L-1
--                        sampai L-3 tidak bisa dicabut.
--   row-level security   documents, document_rows, dan document_revisions
--                        disaring database menurut akun di variabel sesi
--                        app.account_id (diisi internal/db/subject.go):
--                        dokumen yang sudah pernah terbit terlihat semua akun
--                        (L-1, R-SEC-12); dokumen yang belum pernah terbit
--                        hanya terlihat akun yang memegang L-4 untuk customer
--                        itu, termasuk cakupan Project Group sendiri (D-71).
--                        Variabel kosong (CLI, migrasi, job) tidak disaring.

-- +goose Up
GRANT INSERT, DELETE ON role_permissions TO revioten_app;

-- +goose StatementBegin
CREATE FUNCTION role_permissions_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    r text;
    p text;
BEGIN
    IF TG_OP = 'DELETE' THEN
        r := OLD.role_id; p := OLD.permission_code;
    ELSE
        r := NEW.role_id; p := NEW.permission_code;
    END IF;
    IF r IN ('admin', 'viewer') THEN
        RAISE EXCEPTION 'isi peran % tidak bisa diubah (R-SEC-11)', r USING ERRCODE = '55000';
    END IF;
    IF TG_OP <> 'DELETE' AND p LIKE 'A-%' THEN
        RAISE EXCEPTION 'peran kerja tidak boleh memegang hak admin % (R-SEC-11)', p USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' AND p IN ('L-1', 'L-2', 'L-3') THEN
        RAISE EXCEPTION 'hak dasar % tidak bisa dicabut (R-SEC-11)', p USING ERRCODE = '55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER role_permissions_guard BEFORE INSERT OR UPDATE OR DELETE ON role_permissions
    FOR EACH ROW EXECUTE FUNCTION role_permissions_guard();

-- +goose StatementBegin
-- app_sees_document: apakah akun di app.account_id boleh melihat dokumen
-- customer c. SECURITY DEFINER, supaya fungsi ini sendiri tidak disaring RLS
-- dan role aplikasi tidak perlu hak baca tambahan.
CREATE FUNCTION app_sees_document(c uuid, released boolean) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
    SELECT CASE
        WHEN coalesce(current_setting('app.account_id', true), '') = '' THEN true
        WHEN released THEN true
        ELSE EXISTS (
            SELECT 1
            FROM accounts a
            JOIN account_roles ar ON ar.account_id = a.id
            JOIN role_permissions rp ON rp.role_id = ar.role_id
            WHERE a.id = nullif(current_setting('app.account_id', true), '')::uuid
              AND a.disabled_at IS NULL
              AND rp.permission_code = 'L-4'
              AND (ar.scope = 'all' OR EXISTS (
                    SELECT 1
                    FROM position_holders h
                    JOIN positions p ON p.id = h.position_id
                    JOIN customers cu ON cu.project_group_id = h.project_group_id
                    WHERE h.person_id = a.person_id AND cu.id = c
                      AND p.key IN ('project_leader', 'project_engineer')))
        )
    END
$$;
-- +goose StatementEnd

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY documents_customer_scope ON documents TO revioten_app
    USING (app_sees_document(customer_id, released_revision IS NOT NULL));

-- Baris dan revisi mengikuti dokumennya: subquery ini sendiri disaring kebijakan di atas.
ALTER TABLE document_rows ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_rows_customer_scope ON document_rows TO revioten_app
    USING (EXISTS (SELECT 1 FROM documents d WHERE d.id = document_rows.document_id));

ALTER TABLE document_revisions ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_revisions_customer_scope ON document_revisions TO revioten_app
    USING (EXISTS (SELECT 1 FROM documents d WHERE d.id = document_revisions.document_id));

-- +goose Down
DROP POLICY document_revisions_customer_scope ON document_revisions;
ALTER TABLE document_revisions DISABLE ROW LEVEL SECURITY;
DROP POLICY document_rows_customer_scope ON document_rows;
ALTER TABLE document_rows DISABLE ROW LEVEL SECURITY;
DROP POLICY documents_customer_scope ON documents;
ALTER TABLE documents DISABLE ROW LEVEL SECURITY;
DROP FUNCTION app_sees_document(uuid, boolean);
DROP TRIGGER role_permissions_guard ON role_permissions;
DROP FUNCTION role_permissions_guard();
REVOKE INSERT, DELETE ON role_permissions FROM revioten_app;
