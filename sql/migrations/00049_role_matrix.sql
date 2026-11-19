-- D-199 · Matriks peran (fase 2, gaya Spatie Laravel Permission).
--
--   roles              punya nama sendiri dalam dua bahasa, tanda aktif,
--                      tanda bawaan, dan tanda boleh dibatasi ke Project
--                      Group sendiri (D-71). Admin boleh menambah peran,
--                      mengganti nama, dan menonaktifkan peran yang tidak
--                      dipegang akun mana pun. Peran tidak pernah dihapus,
--                      supaya audit log tetap terbaca.
--   permissions        hak baru S-1 (tanda tangan digital, dipakai fase 3,
--                      D-198) dan G-1 (catatan tampilan genba, dipakai saat
--                      tampilan genba dibangun, D-137).
--   peran baru         PPAP Engineer, PPAP Engineer Supervisor, Project
--                      Leader, dan Manager, dengan isi awal dari
--                      docs/rencana/matriks-peran-dan-hak.md bagian 4.
--   akun               dipindah dari peran lama ke peran baru; peran lama
--                      dinonaktifkan.
--   viewer             hak peran Viewer hanya ikut ke akun kerja. Akun admin
--                      sudah memegang L-1 sampai L-3 lewat peran Admin, dan
--                      tidak memegang G-1 (D-72).

-- +goose Up
ALTER TABLE roles
    ADD COLUMN name_en text,
    ADD COLUMN name_id text,
    ADD COLUMN active boolean NOT NULL DEFAULT true,
    ADD COLUMN builtin boolean NOT NULL DEFAULT false,
    ADD COLUMN group_scope boolean NOT NULL DEFAULT false;

UPDATE roles SET builtin = true;
UPDATE roles SET group_scope = true WHERE id IN ('author', 'verifier');
UPDATE roles SET name_en = v.en, name_id = v.id_
FROM (VALUES
    ('author', 'Author', 'Penyusun'),
    ('template_maintainer', 'Template maintainer', 'Pengelola template'),
    ('pfmea_cp_checker', 'PFMEA–CP checker', 'Pemeriksa PFMEA–CP'),
    ('verifier', 'Verifier', 'Verifikator'),
    ('form_approver', 'Form approver', 'Penyetuju form'),
    ('document_control', 'Document control', 'Dokumen kontrol'),
    ('migration', 'Migration', 'Petugas migrasi'),
    ('viewer', 'Viewer', 'Viewer'),
    ('admin', 'Admin', 'Admin')
) AS v (role, en, id_)
WHERE roles.id = v.role;

ALTER TABLE roles
    ALTER COLUMN name_en SET NOT NULL,
    ALTER COLUMN name_id SET NOT NULL,
    ADD CONSTRAINT roles_name_en_not_blank CHECK (btrim(name_en) <> ''),
    ADD CONSTRAINT roles_name_id_not_blank CHECK (btrim(name_id) <> '');
CREATE UNIQUE INDEX roles_name_en_unique ON roles (lower(name_en));
CREATE UNIQUE INDEX roles_name_id_unique ON roles (lower(name_id));

-- Hak baru: S-1 sesudah V-3, G-1 sesudah S-1, hak admin bergeser.
UPDATE permissions SET sort_order = sort_order + 2 WHERE level = 'admin';
INSERT INTO permissions (code, level, sort_order)
SELECT 'S-1', 'verify', max(sort_order) + 1 FROM permissions WHERE level <> 'admin';
INSERT INTO permissions (code, level, sort_order)
SELECT 'G-1', 'view', max(sort_order) + 1 FROM permissions WHERE level <> 'admin';

-- Peran baru. Peran lama digeser ke belakang, sesudah Viewer dan Admin.
UPDATE roles SET sort_order = sort_order + 100 WHERE id NOT IN ('viewer', 'admin');
UPDATE roles SET sort_order = 5 WHERE id = 'viewer';
UPDATE roles SET sort_order = 6 WHERE id = 'admin';
INSERT INTO roles (id, level, sort_order, name_en, name_id, builtin, group_scope) VALUES
    ('ppap_engineer', 'edit', 1, 'PPAP Engineer', 'PPAP Engineer', true, false),
    ('ppap_engineer_supervisor', 'verify', 2, 'PPAP Engineer Supervisor', 'PPAP Engineer Supervisor', true, false),
    ('project_leader', 'verify', 3, 'Project Leader', 'Project Leader', true, true),
    ('manager', 'verify', 4, 'Manager', 'Manager', true, false);

INSERT INTO role_permissions (role_id, permission_code)
SELECT r.role, p.code
FROM (VALUES
    ('ppap_engineer', 'L-1,L-2,L-3,L-4,L-5,E-1,E-2,E-3,E-4,E-5,E-6,E-7,E-8,E-9,E-10,V-2,S-1,G-1'),
    ('ppap_engineer_supervisor', 'L-1,L-2,L-3,L-4,L-5,E-1,E-2,E-3,E-4,E-5,E-6,E-7,E-8,E-9,E-10,V-1,V-2,V-3,S-1,G-1'),
    ('project_leader', 'L-1,L-2,L-3,L-4,E-1,E-2,E-3,E-4,E-5,V-1,S-1,G-1'),
    ('manager', 'L-1,L-2,L-3,L-4,L-5,S-1,G-1')
) AS r (role, codes)
CROSS JOIN LATERAL unnest(string_to_array(r.codes, ',')) AS c (code)
JOIN permissions p ON p.code = c.code;

-- G-1 untuk Viewer. Isi Viewer dikunci trigger (R-SEC-11), jadi trigger
-- dimatikan sebentar di sini, dengan sengaja.
ALTER TABLE role_permissions DISABLE TRIGGER role_permissions_guard;
INSERT INTO role_permissions (role_id, permission_code) VALUES ('viewer', 'G-1');
ALTER TABLE role_permissions ENABLE TRIGGER role_permissions_guard;

-- Pemindahan akun (matriks bagian 3):
--   Penyetuju form, atau Verifikator semua customer      -> PPAP Engineer Supervisor
--   Penyusun, Pengelola template, Pemeriksa PFMEA–CP,
--   Petugas migrasi, Dokumen kontrol (semua customer)    -> PPAP Engineer
--   peran apa pun bercakupan Project Group sendiri       -> Project Leader (Project Group sendiri)
CREATE TEMPORARY TABLE role_moves ON COMMIT DROP AS
SELECT a.account_id,
       bool_or(a.role_id = 'form_approver' OR (a.role_id = 'verifier' AND a.scope = 'all')) AS supervisor,
       bool_or(a.role_id IN ('author', 'template_maintainer', 'pfmea_cp_checker', 'migration', 'document_control')
               AND a.scope = 'all') AS engineer,
       bool_or(a.scope = 'own_project_group') AS leader
FROM account_roles a
WHERE a.role_id NOT IN ('viewer', 'admin')
GROUP BY a.account_id;

INSERT INTO account_roles (account_id, role_id, scope)
SELECT account_id, 'ppap_engineer_supervisor', 'all' FROM role_moves WHERE supervisor;
INSERT INTO account_roles (account_id, role_id, scope)
SELECT account_id, 'ppap_engineer', 'all' FROM role_moves WHERE engineer AND NOT supervisor;
INSERT INTO account_roles (account_id, role_id, scope)
SELECT account_id, 'project_leader', 'own_project_group' FROM role_moves WHERE leader AND NOT supervisor;
DELETE FROM account_roles WHERE role_id IN
    ('author', 'template_maintainer', 'pfmea_cp_checker', 'verifier', 'form_approver', 'document_control', 'migration');

UPDATE roles SET active = false WHERE id IN
    ('author', 'template_maintainer', 'pfmea_cp_checker', 'verifier', 'form_approver', 'document_control', 'migration');

-- Penjaga peran (R-SEC-11): Admin dan Viewer tetap, ID dan sifat bawaan
-- tidak berubah, dan peran yang masih dipegang akun tidak bisa dinonaktifkan.
-- +goose StatementBegin
CREATE FUNCTION roles_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'peran tidak pernah dihapus (D-199)' USING ERRCODE = '55000';
    END IF;
    IF OLD.id IN ('admin', 'viewer') AND (NEW.active IS DISTINCT FROM OLD.active OR NEW.name_en IS DISTINCT FROM OLD.name_en OR NEW.name_id IS DISTINCT FROM OLD.name_id) THEN
        RAISE EXCEPTION 'peran % tidak bisa diubah (R-SEC-11)', OLD.id USING ERRCODE = '55000';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id OR NEW.builtin IS DISTINCT FROM OLD.builtin OR NEW.level IS DISTINCT FROM OLD.level THEN
        RAISE EXCEPTION 'ID, level, dan tanda bawaan peran tetap (D-199)' USING ERRCODE = '55000';
    END IF;
    IF OLD.active AND NOT NEW.active AND EXISTS (SELECT 1 FROM account_roles WHERE role_id = OLD.id) THEN
        RAISE EXCEPTION 'peran % masih dipegang akun, jadi tidak bisa dinonaktifkan (D-199)', OLD.id USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER roles_guard BEFORE UPDATE OR DELETE ON roles
    FOR EACH ROW EXECUTE FUNCTION roles_guard();

GRANT INSERT ON roles TO revioten_app;
GRANT UPDATE (name_en, name_id, active, group_scope) ON roles TO revioten_app;

-- +goose Down
REVOKE UPDATE (name_en, name_id, active, group_scope) ON roles FROM revioten_app;
REVOKE INSERT ON roles FROM revioten_app;
DROP TRIGGER roles_guard ON roles;
DROP FUNCTION roles_guard();
UPDATE roles SET active = true;
-- Akun peran baru kembali ke peran lama yang paling dekat.
INSERT INTO account_roles (account_id, role_id, scope)
SELECT account_id, v.old, 'all'
FROM account_roles ar
JOIN (VALUES ('ppap_engineer_supervisor', 'author'), ('ppap_engineer_supervisor', 'verifier'),
             ('ppap_engineer_supervisor', 'form_approver'), ('ppap_engineer', 'author'),
             ('ppap_engineer', 'document_control')) AS v (new, old) ON v.new = ar.role_id
ON CONFLICT DO NOTHING;
INSERT INTO account_roles (account_id, role_id, scope)
SELECT account_id, v.old, 'own_project_group'
FROM account_roles ar
JOIN (VALUES ('project_leader', 'author'), ('project_leader', 'verifier')) AS v (new, old) ON v.new = ar.role_id
ON CONFLICT DO NOTHING;
DELETE FROM account_roles WHERE role_id IN ('ppap_engineer', 'ppap_engineer_supervisor', 'project_leader', 'manager')
    OR role_id NOT IN (SELECT id FROM roles WHERE builtin);
ALTER TABLE role_permissions DISABLE TRIGGER role_permissions_guard;
DELETE FROM role_permissions WHERE permission_code IN ('S-1', 'G-1')
    OR role_id IN ('ppap_engineer', 'ppap_engineer_supervisor', 'project_leader', 'manager')
    OR role_id NOT IN (SELECT id FROM roles WHERE builtin);
ALTER TABLE role_permissions ENABLE TRIGGER role_permissions_guard;
DELETE FROM roles WHERE id IN ('ppap_engineer', 'ppap_engineer_supervisor', 'project_leader', 'manager') OR NOT builtin;
UPDATE roles SET sort_order = sort_order - 100 WHERE sort_order > 100;
UPDATE roles SET sort_order = 8 WHERE id = 'viewer';
UPDATE roles SET sort_order = 9 WHERE id = 'admin';
DELETE FROM permissions WHERE code IN ('S-1', 'G-1');
UPDATE permissions SET sort_order = sort_order - 2 WHERE level = 'admin';
DROP INDEX roles_name_id_unique;
DROP INDEX roles_name_en_unique;
ALTER TABLE roles
    DROP COLUMN group_scope,
    DROP COLUMN builtin,
    DROP COLUMN active,
    DROP COLUMN name_id,
    DROP COLUMN name_en;
