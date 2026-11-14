-- M5.7 · Daftar periksa dampak (R-CHG-05, R-CHG-01, UC-40).
--
--   change_impacts   satu baris per dokumen terkait yang diperiksa untuk satu
--                    perubahan: jenis dokumen (PFD, PFMEA, CP, WI, standar
--                    inspeksi, lainnya), nomor atau nama dokumennya sebagai
--                    teks, keputusan (terdampak atau tidak), dan alasan.
--                    Alasan wajib bila tidak terdampak (ASUMSI E-104).
--
-- Daftar disimpan utuh: simpan berarti mengganti semua baris perubahan itu,
-- dan setiap penyimpanan tercatat di audit log beserta isinya.

-- +goose Up
CREATE TABLE change_impacts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    change_id uuid NOT NULL REFERENCES changes (id),
    position integer NOT NULL CHECK (position >= 1),
    kind text NOT NULL CHECK (kind IN ('pfd', 'pfmea', 'cp', 'wi', 'inspection_standard', 'other')),
    document_reference text CHECK (document_reference IS NULL OR (document_reference <> '' AND document_reference = btrim(document_reference) AND char_length(document_reference) <= 200)),
    decision text NOT NULL CHECK (decision IN ('affected', 'not_affected')),
    reason text CHECK (reason IS NULL OR (reason <> '' AND reason = btrim(reason) AND char_length(reason) <= 500)),
    updated_by_person_id uuid NOT NULL REFERENCES people (id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT change_impacts_position_key UNIQUE (change_id, position),
    CONSTRAINT change_impacts_reason_needed CHECK (decision = 'affected' OR reason IS NOT NULL)
);
GRANT SELECT, INSERT, DELETE ON change_impacts TO revioten_app;

-- +goose Down
DROP TABLE change_impacts;
