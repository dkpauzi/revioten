-- Master proses (M1.5, UC-04): ID tetap, nama hanya tampilan, dan boleh punya
-- alias (D-06, R-DOC-04). Baris PFD, PFMEA, dan CP nanti menunjuk proses lewat
-- ID-nya (R-DOC-02), jadi nama bisa diganti tanpa memutus rujukan.
-- Dikelola PPAP Engineer lewat hak E-6 (D-64).

-- +goose Up
-- Tabel processes hanya berisi identitas proses. Semua namanya ada di
-- process_names, jadi mengganti nama tidak pernah menyentuh baris ini.
-- E-32 (D-110): satu master untuk semua customer, dan proses tidak dihapus.
CREATE TABLE processes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Nama tampilan dan alias setiap proses. Keduanya disimpan di satu tabel
-- supaya satu kunci menjaga semua nama: satu nama hanya menunjuk satu proses,
-- baik sebagai nama tampilan maupun sebagai alias (E-32, D-110).
CREATE TABLE process_names (
    process_id uuid NOT NULL REFERENCES processes (id),
    -- 'display' untuk nama tampilan, 'alias' untuk nama lain proses yang sama,
    -- misalnya nama yang dipakai dokumen lama (F-05).
    kind text NOT NULL,
    name text NOT NULL,
    -- Kunci pembanding: nama dalam huruf kecil, dihitung database dari name.
    -- "IQC" dan "iqc" dianggap nama yang sama.
    name_key text NOT NULL GENERATED ALWAYS AS (lower(name)) STORED,

    CONSTRAINT process_names_pkey PRIMARY KEY (name_key),
    CONSTRAINT process_names_kind_check CHECK (kind IN ('display', 'alias')),
    CONSTRAINT process_names_name_check CHECK (
        name <> '' AND name = btrim(name) AND char_length(name) <= 200
    )
);
-- Setiap proses paling banyak punya satu nama tampilan. Aplikasi membuat
-- proses dan nama tampilannya dalam satu transaksi, jadi selalu tepat satu.
CREATE UNIQUE INDEX process_names_display_key ON process_names (process_id) WHERE kind = 'display';

-- Hak role aplikasi (E-06). Proses tidak pernah dihapus. Alias boleh dihapus
-- karena tidak ada data yang menunjuknya; nama tampilan hanya diganti.
GRANT SELECT, INSERT ON processes TO revioten_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON process_names TO revioten_app;

-- +goose Down
DROP TABLE process_names;
DROP TABLE processes;
