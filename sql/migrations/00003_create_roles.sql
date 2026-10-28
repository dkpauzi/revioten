-- Peran dan hak akses sebagai data (M1.2, E-01 opsi a, D-66, R-SEC-10).
-- Isi awal mengikuti matriks U3-peta-pengguna bagian 3, baseline yang
-- disetujui pemilik. ID peran dan kode hak tetap; nama tampilannya ada di
-- katalog teks layar (web/src/lib/i18n). Mengubah isi peran lewat layar baru
-- di M8.1 (E-01).

-- +goose Up
CREATE TABLE permissions (
    code text PRIMARY KEY,
    level text NOT NULL CHECK (level IN ('view', 'edit', 'verify', 'admin')),
    sort_order integer NOT NULL
);

INSERT INTO permissions (code, level, sort_order) VALUES
    ('L-1', 'view', 1),    -- lihat dokumen rilis dan halaman verifikasi
    ('L-2', 'view', 2),    -- lihat register nomor, riwayat, diff, diagram PFD, dasbor
    ('L-3', 'view', 3),    -- ekspor Excel dan PDF dokumen rilis
    ('L-4', 'view', 4),    -- lihat draft dan permintaan perubahan yang berjalan (usulan)
    ('E-1', 'edit', 5),    -- daftarkan nomor dokumen, termasuk sebagai set
    ('E-2', 'edit', 6),    -- buat, rakit, dan edit dokumen draft
    ('E-3', 'edit', 7),    -- ajukan perubahan dan isi daftar periksa dampak
    ('E-4', 'edit', 8),    -- draft revisi, yokoten, perubahan surut SQAM
    ('E-5', 'edit', 9),    -- cetak untuk tanda tangan, catat dokumen dikembalikan
    ('E-6', 'edit', 10),   -- master proses dan process template
    ('E-7', 'edit', 11),   -- catat rilis
    ('E-8', 'edit', 12),   -- ubah status nomor dan impor nomor lama
    ('E-9', 'edit', 13),   -- salinan terkendali
    ('E-10', 'edit', 14),  -- migrasikan dokumen lama
    ('V-1', 'verify', 15), -- verifikasi permintaan perubahan
    ('V-2', 'verify', 16), -- tanda dicek perbandingan PFMEA dan CP
    ('A-1', 'admin', 17),  -- kelola pengguna dan hak akses
    ('A-2', 'admin', 18),  -- kelola master data dan pengaturan dokumen
    ('A-3', 'admin', 19),  -- kelola pengaturan sistem
    ('A-4', 'admin', 20);  -- lihat audit log

CREATE TABLE roles (
    id text PRIMARY KEY,
    level text NOT NULL CHECK (level IN ('view', 'edit', 'verify', 'admin')),
    sort_order integer NOT NULL
);

INSERT INTO roles (id, level, sort_order) VALUES
    ('author', 'edit', 1),              -- Penyusun
    ('template_maintainer', 'edit', 2), -- Pengelola template
    ('pfmea_cp_checker', 'verify', 3),  -- Pemeriksa PFMEA–CP
    ('verifier', 'verify', 4),          -- Verifikator
    ('document_control', 'edit', 5),    -- Dokumen kontrol
    ('migration', 'edit', 6),           -- Petugas migrasi
    ('viewer', 'view', 7),              -- Viewer: dimiliki semua akun yang login
    ('admin', 'admin', 8);              -- Admin: hanya akun admin (D-72)

CREATE TABLE role_permissions (
    role_id text NOT NULL REFERENCES roles (id),
    permission_code text NOT NULL REFERENCES permissions (code),
    PRIMARY KEY (role_id, permission_code)
);

-- L-1 sampai L-3 untuk semua peran.
INSERT INTO role_permissions (role_id, permission_code)
SELECT r.id, p.code
FROM roles r CROSS JOIN permissions p
WHERE p.code IN ('L-1', 'L-2', 'L-3');

-- L-4 untuk semua peran kerja, tidak untuk Viewer dan Admin.
INSERT INTO role_permissions (role_id, permission_code)
SELECT id, 'L-4' FROM roles WHERE id NOT IN ('viewer', 'admin');

INSERT INTO role_permissions (role_id, permission_code) VALUES
    ('author', 'E-1'), ('author', 'E-2'), ('author', 'E-3'), ('author', 'E-4'), ('author', 'E-5'),
    ('template_maintainer', 'E-6'),
    ('document_control', 'E-7'), ('document_control', 'E-8'), ('document_control', 'E-9'),
    ('migration', 'E-10'),
    ('verifier', 'V-1'),
    ('pfmea_cp_checker', 'V-2'),
    ('admin', 'A-1'), ('admin', 'A-2'), ('admin', 'A-3'), ('admin', 'A-4');

-- Peran yang dipegang setiap akun. Viewer tidak disimpan di sini, karena
-- semua akun yang login otomatis punya hak dasar level Lihat (U3 bagian 1).
CREATE TABLE account_roles (
    account_id uuid NOT NULL REFERENCES accounts (id),
    role_id text NOT NULL REFERENCES roles (id),
    -- all: semua customer. own_project_group: hanya customer di Project Group
    -- tempat orang pemegang akun menjadi Project Leader atau Project Engineer
    -- (D-71; sumbernya master Project Group, E-05).
    scope text NOT NULL DEFAULT 'all' CHECK (scope IN ('all', 'own_project_group')),
    PRIMARY KEY (account_id, role_id)
);

-- Akun admin yang sudah ada (dibuat create-admin di M1.1) mendapat peran Admin.
INSERT INTO account_roles (account_id, role_id, scope)
SELECT id, 'admin', 'all' FROM accounts WHERE kind = 'admin';

-- +goose Down
DROP TABLE account_roles;
DROP TABLE role_permissions;
DROP TABLE roles;
DROP TABLE permissions;
