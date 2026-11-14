-- Tahap 6 · Cetak dan rilis manual (M6.1 sampai M6.9; R-PRT-01 sampai R-PRT-07,
-- R-APR-08, R-APR-16, R-REV-04, R-REV-08, D-46, D-52, D-53, D-59, D-62, D-76).
--
--   print_layouts            layout cetak sebagai data (R-PRT-02): kertas,
--                            arah, ukuran huruf, margin, cover, tanda revisi,
--                            dan lebar kolom per key field. Satu layout bawaan
--                            per jenis dokumen; layout lain dipilih per customer.
--   customer_print_layouts   layout yang dipakai satu customer untuk satu jenis
--                            dokumen. Tanpa baris berarti layout bawaan.
--   print_logs               setiap cetakan: dokumen, revisi, tujuan (draft,
--                            tanda tangan, referensi, salinan terkendali),
--                            format, nomor salinan, pencetak, waktu, dan hash
--                            isi yang dicetak (R-PRT-04, M6.6).
--   document_revisions       release_date diisi saat dicetak untuk tanda
--                            tangan (D-59); scan bertanda tangan yang wajib ada
--                            sebelum rilis dicatat (M6.7).
--   system_settings          qr_enabled: saklar QR di cetakan, bawaannya mati
--                            (D-62, R-APR-16).

-- +goose Up
CREATE TABLE print_layouts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL CHECK (name <> '' AND name = btrim(name) AND char_length(name) <= 100),
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    is_default boolean NOT NULL DEFAULT false,
    layout jsonb NOT NULL CHECK (jsonb_typeof(layout) = 'object'),
    updated_by_person_id uuid REFERENCES people (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT print_layouts_name_key UNIQUE (name)
);
CREATE UNIQUE INDEX print_layouts_one_default ON print_layouts (document_type_id) WHERE is_default;
GRANT SELECT, INSERT ON print_layouts TO revioten_app;
GRANT UPDATE (name, layout, updated_by_person_id, updated_at) ON print_layouts TO revioten_app;

-- Layout bawaan dari form asli: PFMEA dan CP A3 landscape, PFD A4 landscape
-- (Q-08, asumsi kerja A3 landscape dengan A4 sebagai pilihan).
INSERT INTO print_layouts (name, document_type_id, is_default, layout)
SELECT 'Standard ' || upper(t.key), t.id, true,
       jsonb_build_object('paper', CASE WHEN t.key = 'pfd' THEN 'A4' ELSE 'A3' END, 'orientation', 'landscape',
                          'font_size', CASE WHEN t.key = 'pfd' THEN 8 ELSE 7 END,
                          'margins_mm', jsonb_build_object('top', 10, 'right', 8, 'bottom', 12, 'left', 8),
                          'cover', true, 'revision_marks', true, 'widths', '{}'::jsonb)
FROM document_types t WHERE t.key IN ('pfd', 'pfmea', 'cp');

CREATE TABLE customer_print_layouts (
    customer_id uuid NOT NULL REFERENCES customers (id),
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    layout_id uuid NOT NULL REFERENCES print_layouts (id),
    PRIMARY KEY (customer_id, document_type_id)
);
GRANT SELECT, INSERT, DELETE ON customer_print_layouts TO revioten_app;
GRANT UPDATE (layout_id) ON customer_print_layouts TO revioten_app;

CREATE TABLE print_logs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    document_id uuid NOT NULL REFERENCES documents (id),
    revision_id uuid NOT NULL REFERENCES document_revisions (id),
    revision smallint NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('draft', 'signature', 'reference', 'controlled')),
    format text NOT NULL CHECK (format IN ('xlsx', 'pdf')),
    copy_number integer CHECK (copy_number IS NULL OR copy_number >= 1),
    content_sha256 text NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    printed_by_person_id uuid NOT NULL REFERENCES people (id),
    printed_by_account_id uuid NOT NULL REFERENCES accounts (id),
    printed_at timestamptz NOT NULL DEFAULT now(),
    -- Salinan terkendali selalu bernomor; cetakan lain tidak.
    CONSTRAINT print_logs_copy_check CHECK ((purpose = 'controlled') = (copy_number IS NOT NULL))
);
CREATE UNIQUE INDEX print_logs_copy_key ON print_logs (revision_id, copy_number) WHERE copy_number IS NOT NULL;
CREATE INDEX print_logs_document_idx ON print_logs (document_id, printed_at DESC);
-- Catatan cetak tidak pernah diubah atau dihapus.
GRANT SELECT, INSERT ON print_logs TO revioten_app;

ALTER TABLE document_revisions
    ADD COLUMN release_date date,
    ADD COLUMN scan_file_name text CHECK (scan_file_name IS NULL OR (scan_file_name <> '' AND char_length(scan_file_name) <= 200)),
    ADD COLUMN scan_content_type text,
    ADD COLUMN scan_size_bytes bigint CHECK (scan_size_bytes IS NULL OR scan_size_bytes > 0),
    ADD COLUMN scan_sha256 text CHECK (scan_sha256 IS NULL OR scan_sha256 ~ '^[0-9a-f]{64}$'),
    ADD COLUMN scan_uploaded_by_person_id uuid REFERENCES people (id),
    ADD COLUMN scan_uploaded_at timestamptz,
    ADD CONSTRAINT document_revisions_scan_check CHECK (
        (scan_sha256 IS NULL) = (scan_file_name IS NULL)
        AND (scan_sha256 IS NULL) = (scan_uploaded_at IS NULL)
        AND (scan_sha256 IS NULL) = (scan_uploaded_by_person_id IS NULL)),
    -- Revisi terbit selalu punya scan bertanda tangan (M6.7).
    ADD CONSTRAINT document_revisions_released_scan_check CHECK (status <> 'released' OR scan_sha256 IS NOT NULL) NOT VALID;
GRANT UPDATE (release_date, scan_file_name, scan_content_type, scan_size_bytes, scan_sha256, scan_uploaded_by_person_id, scan_uploaded_at)
    ON document_revisions TO revioten_app;

ALTER TABLE system_settings ADD COLUMN qr_enabled boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE system_settings DROP COLUMN qr_enabled;
REVOKE UPDATE (release_date, scan_file_name, scan_content_type, scan_size_bytes, scan_sha256, scan_uploaded_by_person_id, scan_uploaded_at)
    ON document_revisions FROM revioten_app;
ALTER TABLE document_revisions
    DROP CONSTRAINT document_revisions_released_scan_check,
    DROP CONSTRAINT document_revisions_scan_check,
    DROP COLUMN scan_uploaded_at, DROP COLUMN scan_uploaded_by_person_id, DROP COLUMN scan_sha256,
    DROP COLUMN scan_size_bytes, DROP COLUMN scan_content_type, DROP COLUMN scan_file_name, DROP COLUMN release_date;
DROP TABLE print_logs;
DROP TABLE customer_print_layouts;
DROP TABLE print_layouts;
