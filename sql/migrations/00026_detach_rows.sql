-- M4.8 · Perbedaan lokal dan putus tautan (R-CMP-04).
--
--   document_rows.detached_at    kapan baris diputus dari poin template-nya.
--   document_rows.detach_reason  alasannya, wajib. Baris yang putus tidak lagi
--                                dibandingkan dengan template, tetapi tetap
--                                mengingat poin asalnya (template_point_id).
--
-- Perbedaan lokal tidak disimpan: dihitung saat dilihat, dengan membandingkan
-- isi baris dan isi poin di versi template yang dipakai merakit.

-- +goose Up
ALTER TABLE document_rows
    ADD COLUMN detached_at timestamptz,
    ADD COLUMN detach_reason text,
    ADD CONSTRAINT document_rows_detach_check CHECK (
        (detached_at IS NULL AND detach_reason IS NULL)
        OR (detached_at IS NOT NULL AND template_point_id IS NOT NULL
            AND detach_reason <> '' AND detach_reason = btrim(detach_reason) AND char_length(detach_reason) <= 500)
    );
GRANT UPDATE (detached_at, detach_reason) ON document_rows TO revioten_app;

-- +goose Down
REVOKE UPDATE (detached_at, detach_reason) ON document_rows FROM revioten_app;
ALTER TABLE document_rows DROP CONSTRAINT document_rows_detach_check, DROP COLUMN detach_reason, DROP COLUMN detached_at;