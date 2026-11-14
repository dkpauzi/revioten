-- M5.3 · Verifikasi atasan (R-CHG-02, D-51, D-54, UC-41).
--
-- Perubahan yang diajukan diterima atau ditolak oleh satu verifikator dengan
-- catatan wajib. Yang dicatat: orang dan akun verifikator, waktu, dan
-- catatannya. Pemohon tidak pernah memverifikasi perubahannya sendiri; aturan
-- ini juga dijaga CHECK di bawah (R-SEC-11).

-- +goose Up
ALTER TABLE changes DROP CONSTRAINT changes_status_check;
ALTER TABLE changes ADD CONSTRAINT changes_status_check CHECK (status IN ('submitted', 'accepted', 'rejected'));
ALTER TABLE changes
    ADD COLUMN verified_by_person_id uuid REFERENCES people (id),
    ADD COLUMN verified_by_account_id uuid REFERENCES accounts (id),
    ADD COLUMN verified_at timestamptz,
    ADD COLUMN verification_note text CHECK (verification_note IS NULL OR (verification_note <> '' AND verification_note = btrim(verification_note) AND char_length(verification_note) <= 2000)),
    -- Diterima atau ditolak selalu dengan verifikator, waktu, dan catatan.
    ADD CONSTRAINT changes_verified_check CHECK (
        (status = 'submitted') = (verified_by_person_id IS NULL)
        AND (verified_by_person_id IS NULL) = (verified_at IS NULL)
        AND (verified_by_person_id IS NULL) = (verification_note IS NULL)
        AND (verified_by_person_id IS NULL) = (verified_by_account_id IS NULL)),
    -- Pemohon tidak memverifikasi permintaannya sendiri (R-CHG-02, R-SEC-11).
    ADD CONSTRAINT changes_verifier_not_requester CHECK (verified_by_person_id IS DISTINCT FROM requested_by_person_id);
GRANT UPDATE (verified_by_person_id, verified_by_account_id, verified_at, verification_note) ON changes TO revioten_app;

-- +goose Down
REVOKE UPDATE (verified_by_person_id, verified_by_account_id, verified_at, verification_note) ON changes FROM revioten_app;
ALTER TABLE changes DROP CONSTRAINT changes_verifier_not_requester;
ALTER TABLE changes DROP CONSTRAINT changes_verified_check;
ALTER TABLE changes DROP COLUMN verification_note, DROP COLUMN verified_at, DROP COLUMN verified_by_account_id, DROP COLUMN verified_by_person_id;
ALTER TABLE changes DROP CONSTRAINT changes_status_check;
ALTER TABLE changes ADD CONSTRAINT changes_status_check CHECK (status IN ('submitted'));
