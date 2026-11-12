-- D-139 diperbarui 12 November 2026: aturan persetujuan form set dan tabel AP
-- bisa diatur admin. "all" (bawaan, keputusan pemilik): semua penyetuju form
-- harus setuju. "any": satu persetujuan cukup.
--
--   system_settings.form_approval_mode   aturan yang berlaku
--   form_approvals                        persetujuan per orang untuk satu
--                                         versi form set atau versi format.
--                                         Hanya bisa ditambah, tidak diubah
--                                         atau dihapus, seperti audit log.
--
-- "Semua penyetuju" dihitung saat persetujuan diberikan: orang dengan akun
-- kerja aktif yang memegang hak V-3, kecuali penyusun drafnya.

-- +goose Up
ALTER TABLE system_settings
    ADD COLUMN form_approval_mode text NOT NULL DEFAULT 'all',
    ADD CONSTRAINT system_settings_form_approval_mode_check CHECK (form_approval_mode IN ('any', 'all'));

CREATE TABLE form_approvals (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    -- form_set_version atau format_version
    target_kind text NOT NULL,
    target_id uuid NOT NULL,
    person_id uuid NOT NULL REFERENCES people (id),
    account_id uuid NOT NULL REFERENCES accounts (id),
    approved_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT form_approvals_kind_check CHECK (target_kind IN ('form_set_version', 'format_version')),
    CONSTRAINT form_approvals_person_key UNIQUE (target_kind, target_id, person_id)
);

GRANT SELECT, INSERT ON form_approvals TO revioten_app;

-- +goose Down
DROP TABLE form_approvals;
ALTER TABLE system_settings DROP COLUMN form_approval_mode;