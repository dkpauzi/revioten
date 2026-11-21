-- D-207 · Form Control Submission per dokumen.
--
--   customers.submission_required   bawaan "perlu dikirim ke customer" untuk
--                                   revisi dokumen customer itu.
--   system_settings.submission_form_number
--                                   nomor form yang dicetak, diatur admin.
--   document_submissions            satu baris per revisi: perlu dikirim atau
--                                   tidak (bila berbeda dari bawaan customer,
--                                   dengan alasan), tanggal kirim dan isi paket,
--                                   hasil customer dan tanggalnya, scan bukti
--                                   (opsional), dan catatan. Kolom lain di form
--                                   (alasan, tanggal mulai, selesai, tanda
--                                   tangan) dihitung dari data yang sudah ada.

-- +goose Up
ALTER TABLE customers ADD COLUMN submission_required boolean NOT NULL DEFAULT true;
GRANT UPDATE (submission_required) ON customers TO revioten_app;

ALTER TABLE system_settings ADD COLUMN submission_form_number text NOT NULL DEFAULT ''
    CHECK (char_length(submission_form_number) <= 100);

CREATE TABLE document_submissions (
    revision_id uuid PRIMARY KEY REFERENCES document_revisions (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    required_override boolean,
    override_reason text CHECK (override_reason IS NULL OR (btrim(override_reason) <> '' AND char_length(override_reason) <= 500)),
    submitted_on date,
    package text CHECK (package IS NULL OR char_length(package) <= 500),
    result text CHECK (result IN ('waiting', 'approved', 'rejected')),
    result_on date,
    result_scan_name text,
    result_scan_type text,
    result_scan_size bigint,
    result_scan_sha256 text CHECK (result_scan_sha256 IS NULL OR result_scan_sha256 ~ '^[0-9a-f]{64}$'),
    remark text CHECK (remark IS NULL OR char_length(remark) <= 1000),
    updated_by_person_id uuid NOT NULL REFERENCES people (id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Pengecualian dari bawaan customer selalu beralasan.
    CONSTRAINT document_submissions_override_check CHECK ((required_override IS NULL) = (override_reason IS NULL)),
    -- Tanggal hasil hanya untuk hasil yang sudah diputuskan.
    CONSTRAINT document_submissions_result_date_check CHECK (result_on IS NULL OR result IN ('approved', 'rejected')),
    CONSTRAINT document_submissions_scan_pair_check CHECK ((result_scan_sha256 IS NULL) = (result_scan_name IS NULL))
);
CREATE INDEX document_submissions_document_idx ON document_submissions (document_id);
GRANT SELECT, INSERT ON document_submissions TO revioten_app;
GRANT UPDATE (required_override, override_reason, submitted_on, package, result, result_on, result_scan_name, result_scan_type,
    result_scan_size, result_scan_sha256, remark, updated_by_person_id, updated_at) ON document_submissions TO revioten_app;

-- Disaring seperti dokumennya (RLS, M8.1).
ALTER TABLE document_submissions ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_submissions_customer_scope ON document_submissions TO revioten_app
    USING (EXISTS (SELECT 1 FROM documents d WHERE d.id = document_submissions.document_id));

-- +goose Down
DROP TABLE document_submissions;
ALTER TABLE system_settings DROP COLUMN submission_form_number;
REVOKE UPDATE (submission_required) ON customers FROM revioten_app;
ALTER TABLE customers DROP COLUMN submission_required;
