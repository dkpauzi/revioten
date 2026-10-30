-- Jabatan dan rute approval (M1.8, UC-02, UC-06, UC-07). Konsep: alur kerja
-- sebagai data. Jabatan punya ID tetap dan nama yang bisa diganti (D-58).
-- Rute approval adalah daftar slot berurutan per tahap Prepared, Checked, dan
-- Approved (D-45 sampai D-49, R-APR-08), dan setiap customer punya rute
-- default (D-47).

-- +goose Up
-- Jabatan sesuai bagan organisasi NPI (D-54 sampai D-56). key tetap; nama
-- hanya tampilan, jadi NPI Manager bisa diganti PE Manager lewat master data
-- (D-55, R-APR-15). per_project_group menandai jabatan yang dipegang satu
-- orang per Project Group: Project Leader dan Project Engineer (D-54).
CREATE TABLE positions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    key text NOT NULL,
    name text NOT NULL,
    per_project_group boolean NOT NULL,
    sort_order smallint NOT NULL,

    CONSTRAINT positions_key_key UNIQUE (key),
    CONSTRAINT positions_key_check CHECK (key IN (
        'ppap_engineer', 'ppap_staff', 'project_engineer', 'project_leader',
        'apqp_ppap_controller', 'npi_manager', 'qa_qc_manager', 'advisor', 'factory_manager'
    )),
    -- Sasaran foreign key gabungan di tabel position_holders.
    CONSTRAINT positions_id_per_group_key UNIQUE (id, per_project_group),
    CONSTRAINT positions_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX positions_name_key ON positions (lower(name));
INSERT INTO positions (key, name, per_project_group, sort_order) VALUES
    ('ppap_engineer', 'PPAP Engineer', false, 1),
    ('ppap_staff', 'PPAP Staff', false, 2),
    ('project_engineer', 'Project Engineer', true, 3),
    ('project_leader', 'Project Leader', true, 4),
    ('apqp_ppap_controller', 'APQP/PPAP Controller', false, 5),
    ('npi_manager', 'NPI Manager', false, 6),
    ('qa_qc_manager', 'QA QC Manager', false, 7),
    ('advisor', 'Advisor', false, 8),
    ('factory_manager', 'Factory Manager', false, 9);

-- Pemegang jabatan: orang, bukan akun (E-02). Satu orang boleh memegang lebih
-- dari satu jabatan (D-58), misalnya Project Leader GR3 yang merangkap
-- APQP/PPAP Controller (D-54).
-- Project Leader dan Project Engineer dibatasi satu orang per Project Group
-- (D-54). Batas satu pemegang untuk jabatan approver dan APQP/PPAP
-- Controller ditambah migrasi 00009 (E-38, D-115).
CREATE TABLE position_holders (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    position_id uuid NOT NULL,
    -- Salinan positions.per_project_group, dijaga foreign key gabungan di bawah.
    per_project_group boolean NOT NULL,
    person_id uuid NOT NULL REFERENCES people (id),
    project_group_id uuid REFERENCES project_groups (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT position_holders_position_fkey FOREIGN KEY (position_id, per_project_group)
        REFERENCES positions (id, per_project_group),
    -- Project Group diisi tepat untuk jabatan per Project Group.
    CONSTRAINT position_holders_group_check CHECK (per_project_group = (project_group_id IS NOT NULL)),
    -- Orang yang sama tidak dicatat dua kali untuk jabatan dan Project Group yang sama.
    CONSTRAINT position_holders_unique UNIQUE NULLS NOT DISTINCT (position_id, person_id, project_group_id)
);
-- Satu Project Leader dan satu Project Engineer per Project Group (D-54).
CREATE UNIQUE INDEX position_holders_one_per_group ON position_holders (position_id, project_group_id)
    WHERE per_project_group;

-- Rute approval: daftar slot berurutan (R-APR-08). Isi awalnya dua rute
-- (D-46): sampai QA QC Manager, dan sampai slot GM yang saat ini dipegang
-- Advisor (D-56).
CREATE TABLE approval_routes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT approval_routes_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
CREATE UNIQUE INDEX approval_routes_name_key ON approval_routes (lower(name));

-- Slot satu rute, urut menurut slot_order. Setiap slot punya tahap dan judul
-- cetak yang terpisah dari nama jabatan (R-APR-17, D-76). Slot diganti
-- sekaligus saat rute diubah; revisi yang dicetak nanti menyimpan salinan
-- rutenya sendiri (R-APR-11), jadi tidak menunjuk baris ini.
CREATE TABLE route_slots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    route_id uuid NOT NULL REFERENCES approval_routes (id),
    slot_order smallint NOT NULL,
    stage text NOT NULL,
    print_title text NOT NULL,

    CONSTRAINT route_slots_order_key UNIQUE (route_id, slot_order),
    CONSTRAINT route_slots_stage_check CHECK (stage IN ('prepared', 'checked', 'approved')),
    CONSTRAINT route_slots_print_title_check CHECK (
        print_title <> '' AND print_title = btrim(print_title) AND char_length(print_title) <= 100
    )
);

-- Jabatan yang boleh mengisi satu slot.
CREATE TABLE route_slot_positions (
    slot_id uuid NOT NULL REFERENCES route_slots (id) ON DELETE CASCADE,
    position_id uuid NOT NULL REFERENCES positions (id),

    PRIMARY KEY (slot_id, position_id)
);

-- Isi awal dua rute. Judul cetak mengikuti cover TR-020 (F-02, D-76).
-- Prepared: PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer
-- (D-63, D-67). Checked: Project Engineer atau Project Leader (D-48, D-54).
-- Approved: NPI Manager, lalu QA QC Manager, lalu slot GM (D-49, D-56).
INSERT INTO approval_routes (name) VALUES ('Up to QA QC Manager'), ('Up to GM');
INSERT INTO route_slots (route_id, slot_order, stage, print_title)
SELECT r.id, s.slot_order, s.stage, s.print_title
FROM approval_routes r
CROSS JOIN (VALUES
    (1, 'prepared', 'Process. Eng'),
    (2, 'checked', 'Engineer'),
    (3, 'approved', 'NPI Manager'),
    (4, 'approved', 'QA / QC Mgr.'),
    (5, 'approved', 'General Mgr.')
) AS s (slot_order, stage, print_title)
WHERE r.name = 'Up to GM' OR s.slot_order <= 4;
INSERT INTO route_slot_positions (slot_id, position_id)
SELECT s.id, p.id
FROM route_slots s
JOIN positions p ON
    (s.slot_order = 1 AND p.key IN ('ppap_engineer', 'ppap_staff', 'project_leader', 'project_engineer'))
    OR (s.slot_order = 2 AND p.key IN ('project_engineer', 'project_leader'))
    OR (s.slot_order = 3 AND p.key = 'npi_manager')
    OR (s.slot_order = 4 AND p.key = 'qa_qc_manager')
    OR (s.slot_order = 5 AND p.key = 'advisor');

-- Rute default customer (D-47). Kosong untuk customer yang belum diatur
-- (E-39, D-116); Admin mengaturnya di layar Approval routes.
ALTER TABLE customers ADD COLUMN default_route_id uuid REFERENCES approval_routes (id);

-- Kebijakan approval per jenis perubahan (R-TPL-10): rute yang dipakai.
-- Kosong berarti rute default customer, jadi semua jenis memakai alur yang
-- sama sampai diatur lain (Q-15, E-35, D-112).
ALTER TABLE change_types ADD COLUMN approval_route_id uuid REFERENCES approval_routes (id);

-- Pengaturan sistem, satu baris (UC-07). Tanggal tanda tangan per slot bisa
-- dimatikan dan disembunyikan, dan saat ini mati (D-52, R-APR-14).
CREATE TABLE system_settings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    sign_dates_enabled boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX system_settings_single_row ON system_settings ((true));
INSERT INTO system_settings DEFAULT VALUES;

-- Hak role aplikasi (E-06). Jabatan dan rute tidak dihapus. Pemegang jabatan
-- dan slot rute boleh dihapus: riwayatnya ada di audit log, dan revisi yang
-- dicetak menyimpan salinan rutenya sendiri.
GRANT SELECT, UPDATE ON positions TO revioten_app;
GRANT SELECT, INSERT, DELETE ON position_holders TO revioten_app;
GRANT SELECT, INSERT, UPDATE ON approval_routes TO revioten_app;
GRANT SELECT, INSERT, DELETE ON route_slots TO revioten_app;
GRANT SELECT, INSERT, DELETE ON route_slot_positions TO revioten_app;
GRANT SELECT, UPDATE ON system_settings TO revioten_app;

-- +goose Down
DROP TABLE system_settings;
ALTER TABLE change_types DROP COLUMN approval_route_id;
ALTER TABLE customers DROP COLUMN default_route_id;
DROP TABLE route_slot_positions;
DROP TABLE route_slots;
DROP TABLE approval_routes;
DROP TABLE position_holders;
DROP TABLE positions;
