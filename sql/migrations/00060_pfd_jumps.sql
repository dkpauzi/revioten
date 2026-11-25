-- D-227 (E-147): lompatan di PFD. Langkah inspeksi (Decision) bisa menuju
-- langkah lain selain langkah berikutnya: maju (OK dari AOI langsung ke ICT,
-- melewati VMI) atau mundur (No di Finish 2nd side kembali ke Loading PCB).
--
--   jump_to     nomor langkah tujuan di PFD yang sama (teks; workbook memberi
--               pilihan dari nomor langkah dokumen itu)
--   jump_label  label garis lompatan: OK, NG, Yes, atau No. Garis lurus ke
--               bawah mendapat label lawannya.
--
-- Hanya versi draft PFD v1 yang diubah: versi terbit tidak bisa diubah
-- (M3.4). Template PFD yang versinya sudah terbit mendapat kedua field ini
-- lewat versi baru di Format templates (Add a field, dengan key yang sama).

-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    v record;
    jump_to uuid;
    jump_label uuid;
    pos smallint;
BEGIN
    FOR v IN
        SELECT tv.id, tv.template_id
        FROM format_template_versions tv
        JOIN format_templates t ON t.id = tv.template_id
        WHERE t.name = 'PFD v1' AND t.customer_id IS NULL AND tv.status = 'draft'
          AND NOT EXISTS (SELECT 1 FROM format_fields o WHERE o.template_id = t.id AND o.area = 'row' AND o.key IN ('jump_to', 'jump_label'))
    LOOP
        INSERT INTO format_fields (template_id, area, kind, key) VALUES (v.template_id, 'row', 'text', 'jump_to')
        RETURNING id INTO jump_to;
        INSERT INTO format_fields (template_id, area, kind, key) VALUES (v.template_id, 'row', 'select', 'jump_label')
        RETURNING id INTO jump_label;
        SELECT COALESCE(max(position), 0) INTO pos FROM format_version_fields WHERE version_id = v.id AND area = 'row';
        INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, options)
        VALUES (v.id, jump_to, v.template_id, 'row', 'Jump to', pos + 1, NULL),
               (v.id, jump_label, v.template_id, 'row', 'Jump label', pos + 2,
                '[{"key": "ok", "label": "OK"}, {"key": "ng", "label": "NG"}, {"key": "yes", "label": "Yes"}, {"key": "no", "label": "No"}]'::jsonb);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Field yang sudah dipakai versi tidak dihapus (R-TPL-02).
SELECT 1;
