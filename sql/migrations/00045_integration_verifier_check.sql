-- M9.4 · Pemohon perubahan kini boleh service account, jadi kolom orang
-- pemohon bisa kosong. Aturan "verifikator bukan pemohon" di migrasi 00032
-- memakai IS DISTINCT FROM, yang menganggap dua nilai kosong sama, sehingga
-- usulan dari luar yang belum diverifikasi ikut tertolak. Di sini aturan itu
-- ditulis ulang: hanya berlaku bila perubahan sudah punya verifikator.

-- +goose Up
ALTER TABLE changes DROP CONSTRAINT changes_verifier_not_requester;
ALTER TABLE changes ADD CONSTRAINT changes_verifier_not_requester
    CHECK (verified_by_person_id IS NULL OR verified_by_person_id IS DISTINCT FROM requested_by_person_id);

-- +goose Down
ALTER TABLE changes DROP CONSTRAINT changes_verifier_not_requester;
ALTER TABLE changes ADD CONSTRAINT changes_verifier_not_requester
    CHECK (verified_by_person_id IS DISTINCT FROM requested_by_person_id) NOT VALID;
