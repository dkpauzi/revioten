-- Process template atau pustaka proses (Tahap 3, M3.2, UC-21, R-TPL-01,
-- R-TPL-04, D-64). Isinya proses, sub-proses, dan poin, masing-masing dengan
-- ID tetap. Nama dan teks hanya tampilan, jadi bisa diganti tanpa memutus
-- rujukan. Baris dokumen (M4.1) nanti menunjuk proses dan poin lewat ID-nya
-- (R-DOC-02, R-DOC-04), dan perakitan (M4.5) memilih sub-proses dan poin dari
-- pustaka ini.
--
--   process_templates               satu template per proses master (ASUMSI E-72)
--   process_template_versions       versi template; draft bisa diubah, published di M3.4
--   subprocesses                    identitas sub-proses yang tetap, milik satu proses (ASUMSI E-33)
--   process_template_subprocesses   nama, urutan, dan tersembunyi per versi
--   template_points                 identitas poin yang tetap: sub-proses dan jenis dokumen
--   process_template_points         teks, urutan, dan tersembunyi per versi
--
-- Pola yang sama dengan format template (M3.1): identitas dipisah dari sifat
-- per versi, foreign key gabungan menjaga semua bagian milik proses yang sama,
-- dan tidak ada DELETE (yang tidak dipakai disembunyikan).

-- +goose Up
-- ASUMSI E-72: satu process template per proses master, dengan cakupan umum.
-- M3.3 menambah cakupan customer, group model, dan model (D-39); saat itu
-- kunci unik process_id diganti dengan kunci per proses dan cakupan.
CREATE TABLE process_templates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    process_id uuid NOT NULL REFERENCES processes (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT process_templates_process_key UNIQUE (process_id),
    -- Sasaran foreign key gabungan dari process_template_versions.
    CONSTRAINT process_templates_id_process_key UNIQUE (id, process_id)
);

-- Versi template (R-TPL-05). Versi draft bisa diubah; penerbitan dan versi
-- berikutnya dibangun di M3.4. process_id disalin dari template supaya
-- tabel di bawah bisa memeriksa bahwa sub-prosesnya milik proses yang sama.
CREATE TABLE process_template_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    template_id uuid NOT NULL,
    process_id uuid NOT NULL,
    version integer NOT NULL,
    status text NOT NULL DEFAULT 'draft',
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT process_template_versions_template_fkey FOREIGN KEY (template_id, process_id)
        REFERENCES process_templates (id, process_id),
    CONSTRAINT process_template_versions_version_key UNIQUE (template_id, version),
    -- Sasaran foreign key gabungan dari process_template_subprocesses.
    CONSTRAINT process_template_versions_id_process_key UNIQUE (id, process_id),
    CONSTRAINT process_template_versions_version_check CHECK (version >= 1),
    CONSTRAINT process_template_versions_status_check CHECK (status IN ('draft', 'published'))
);

-- Identitas sub-proses, tetap lintas versi dan nanti lintas cakupan (M3.3):
-- template customer bisa menunjuk sub-proses yang sama dengan template umum.
-- ASUMSI E-33: pencocokan PFD, PFMEA, dan CP tetap di tingkat proses (opsi a),
-- dan nama sub-proses disimpan per versi. Bila pemilik memilih opsi (b), nama
-- dan alias sub-proses pindah ke master dengan ID yang sama, jadi rujukan ke
-- tabel ini tidak berubah.
CREATE TABLE subprocesses (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    process_id uuid NOT NULL REFERENCES processes (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    -- Sasaran foreign key gabungan dari process_template_subprocesses.
    CONSTRAINT subprocesses_id_process_key UNIQUE (id, process_id)
);

-- Sub-proses dalam satu versi: nama tampilan, urutan, dan tersembunyi.
CREATE TABLE process_template_subprocesses (
    version_id uuid NOT NULL,
    process_id uuid NOT NULL,
    subprocess_id uuid NOT NULL,
    name text NOT NULL,
    position smallint NOT NULL,
    hidden boolean NOT NULL DEFAULT false,

    PRIMARY KEY (version_id, subprocess_id),
    -- Versi dan sub-proses harus milik proses yang sama.
    CONSTRAINT process_template_subprocesses_version_fkey FOREIGN KEY (version_id, process_id)
        REFERENCES process_template_versions (id, process_id),
    CONSTRAINT process_template_subprocesses_subprocess_fkey FOREIGN KEY (subprocess_id, process_id)
        REFERENCES subprocesses (id, process_id),
    -- Urutan unik per versi, diperiksa di akhir transaksi supaya bisa digeser.
    CONSTRAINT process_template_subprocesses_position_key UNIQUE (version_id, position)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT process_template_subprocesses_position_check CHECK (position >= 1),
    CONSTRAINT process_template_subprocesses_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
-- Dua sub-proses dalam satu versi tidak boleh bernama sama, tanpa membedakan
-- huruf besar dan kecil.
CREATE UNIQUE INDEX process_template_subprocesses_name_key
    ON process_template_subprocesses (version_id, lower(name));

-- Identitas poin (R-TPL-04): satu butir setingkat baris di dalam satu
-- sub-proses, untuk satu jenis dokumen (PFD, PFMEA, atau CP). Sub-proses dan
-- jenis dokumennya tetap, seperti area dan jenis field di format template.
CREATE TABLE template_points (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    subprocess_id uuid NOT NULL REFERENCES subprocesses (id),
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    created_at timestamptz NOT NULL DEFAULT now(),

    -- Sasaran foreign key gabungan dari process_template_points.
    CONSTRAINT template_points_identity_key UNIQUE (id, subprocess_id, document_type_id)
);

-- Poin dalam satu versi. ASUMSI E-73: di M3.2 poin hanya berisi teks satu
-- baris; isi baris lengkap menurut format template diputuskan sebelum M4.5.
CREATE TABLE process_template_points (
    version_id uuid NOT NULL,
    subprocess_id uuid NOT NULL,
    document_type_id uuid NOT NULL,
    point_id uuid NOT NULL,
    text text NOT NULL,
    position smallint NOT NULL,
    hidden boolean NOT NULL DEFAULT false,

    PRIMARY KEY (version_id, point_id),
    -- Sub-prosesnya harus ada di versi yang sama, dan sub-proses serta jenis
    -- dokumen di sini sama dengan identitas poin.
    CONSTRAINT process_template_points_subprocess_fkey FOREIGN KEY (version_id, subprocess_id)
        REFERENCES process_template_subprocesses (version_id, subprocess_id),
    CONSTRAINT process_template_points_point_fkey FOREIGN KEY (point_id, subprocess_id, document_type_id)
        REFERENCES template_points (id, subprocess_id, document_type_id),
    -- Urutan unik per sub-proses dan jenis dokumen.
    CONSTRAINT process_template_points_position_key UNIQUE (version_id, subprocess_id, document_type_id, position)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT process_template_points_position_check CHECK (position >= 1),
    CONSTRAINT process_template_points_text_check CHECK (
        text <> '' AND text = btrim(text) AND char_length(text) <= 500
    )
);

-- Hak role aplikasi (E-06), sekecil mungkin. Tidak ada DELETE di semua tabel
-- process template, dan identitas sub-proses serta poin tidak bisa diubah.
GRANT SELECT, INSERT ON process_templates TO revioten_app;
GRANT SELECT, INSERT ON process_template_versions TO revioten_app;
-- Mengunci baris versi (SELECT ... FOR NO KEY UPDATE) butuh hak UPDATE pada
-- paling sedikit satu kolom. Kolom status dipilih karena penerbitan (M3.4)
-- memang mengubahnya; kode M3.2 belum mengubah status.
GRANT UPDATE (status) ON process_template_versions TO revioten_app;
GRANT SELECT, INSERT ON subprocesses TO revioten_app;
GRANT SELECT, INSERT ON process_template_subprocesses TO revioten_app;
GRANT UPDATE (name, position, hidden) ON process_template_subprocesses TO revioten_app;
GRANT SELECT, INSERT ON template_points TO revioten_app;
GRANT SELECT, INSERT ON process_template_points TO revioten_app;
GRANT UPDATE (text, position, hidden) ON process_template_points TO revioten_app;

-- +goose Down
DROP TABLE process_template_points;
DROP TABLE template_points;
DROP TABLE process_template_subprocesses;
DROP TABLE subprocesses;
DROP TABLE process_template_versions;
DROP TABLE process_templates;
