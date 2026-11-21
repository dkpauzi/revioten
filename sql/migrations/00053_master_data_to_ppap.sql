-- D-209: master data menjadi pekerjaan PPAP Engineer, bukan admin.
--
-- Hak admin A-2 (kelola master data) diganti hak kerja E-11 dengan isi yang
-- sama: customer, Project Group, model, jenis dokumen, jenis perubahan,
-- jabatan dan pemegangnya, rute approval, format template, form set, layout
-- cetak, dan pengaturan dokumen (tanggal di slot, QR, nomor form submission).
-- E-11 diberikan ke PPAP Engineer dan PPAP Engineer Supervisor. Admin tetap
-- melihat master data lewat hak dasar, tetapi tidak mengubahnya.
--
-- Yang tetap di admin (A-1, A-3): akun, peran, kesehatan web, audit log,
-- email, service account, mode migrasi, dan aturan penyetujuan form.
-- Pemisahan akun admin dan akun kerja (D-72, R-SEC-13) tidak berubah: hak
-- A- tetap hanya milik akun admin.

-- +goose Up
-- E-11 masuk sesudah E-10; hak sesudahnya bergeser satu.
UPDATE permissions SET sort_order = sort_order + 1
WHERE sort_order > (SELECT sort_order FROM permissions WHERE code = 'E-10');
INSERT INTO permissions (code, level, sort_order)
SELECT 'E-11', 'edit', sort_order + 1 FROM permissions WHERE code = 'E-10';

-- Isi peran Admin dikunci trigger (R-SEC-11); migrasi ini sengaja membuka
-- kuncinya sebentar untuk melepas A-2.
ALTER TABLE role_permissions DISABLE TRIGGER role_permissions_guard;
DELETE FROM role_permissions WHERE permission_code = 'A-2';
ALTER TABLE role_permissions ENABLE TRIGGER role_permissions_guard;
DELETE FROM permissions WHERE code = 'A-2';

INSERT INTO role_permissions (role_id, permission_code)
VALUES ('ppap_engineer', 'E-11'), ('ppap_engineer_supervisor', 'E-11');

-- +goose Down
INSERT INTO permissions (code, level, sort_order)
SELECT 'A-2', 'admin', sort_order + 1 FROM permissions WHERE code = 'A-1';
DELETE FROM role_permissions WHERE permission_code = 'E-11';
ALTER TABLE role_permissions DISABLE TRIGGER role_permissions_guard;
INSERT INTO role_permissions (role_id, permission_code) VALUES ('admin', 'A-2');
ALTER TABLE role_permissions ENABLE TRIGGER role_permissions_guard;
DELETE FROM permissions WHERE code = 'E-11';
