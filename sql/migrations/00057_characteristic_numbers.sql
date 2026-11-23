-- D-222: nomor karakteristik Control Plan, seperti kolom F dan I di
-- SC-TR-020-04 (No di depan PRODUCT CHARACTERISTICS dan di depan PROCESS
-- CHARACTERISTICS).
--
-- Dua jenis kolom terisi otomatis baru: product_char_number dan
-- process_char_number. Isinya tidak diketik; cetakan dan workbook menomori
-- baris yang Product atau Process Characteristics-nya diisi, berurutan dan
-- mulai lagi dari 1 di setiap proses (internal/printout/charno.go).
--
-- Label di format "Product No" dan "Process No", karena label dalam satu versi
-- harus unik dan "No" sudah dipakai nomor langkah; cetakan menulis "No".
--
-- Control Plan v1 mendapat kedua kolom di versi draft-nya (versi terbit tidak
-- bisa diubah, M3.4; template yang sudah terbit mendapatnya lewat versi baru
-- di Format templates, Add a field).

-- +goose Up
ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn', 'ap',
             'operation_number', 'process', 'process_flow', 'document_number',
             'product_char_number', 'process_char_number')
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_filled_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_filled_check CHECK (
    (kind NOT IN ('operation_number', 'process', 'process_flow', 'product_char_number', 'process_char_number') OR area = 'row')
    AND (kind <> 'document_number' OR area = 'header')
);

-- +goose StatementBegin
DO $$
DECLARE
    v record;
    f record;
    product_no uuid;
    process_no uuid;
    pos smallint;
BEGIN
    FOR v IN
        SELECT tv.id, tv.template_id
        FROM format_template_versions tv
        JOIN format_templates t ON t.id = tv.template_id
        WHERE t.name = 'Control Plan v1' AND t.customer_id IS NULL AND tv.status = 'draft'
          AND NOT EXISTS (SELECT 1 FROM format_fields o WHERE o.template_id = t.id AND o.kind = 'product_char_number')
    LOOP
        INSERT INTO format_fields (template_id, area, kind) VALUES (v.template_id, 'row', 'product_char_number')
        RETURNING id INTO product_no;
        INSERT INTO format_fields (template_id, area, kind) VALUES (v.template_id, 'row', 'process_char_number')
        RETURNING id INTO process_no;
        -- Geser semua kolom sementara, supaya urutan baru tidak bentrok dengan
        -- UNIQUE (version_id, area, position).
        UPDATE format_version_fields SET position = position + 1000 WHERE version_id = v.id AND area = 'row';
        pos := 0;
        FOR f IN
            SELECT vf.field_id, ff.key FROM format_version_fields vf JOIN format_fields ff ON ff.id = vf.field_id
            WHERE vf.version_id = v.id AND vf.area = 'row' ORDER BY vf.position
        LOOP
            IF f.key = 'product_characteristic' THEN
                pos := pos + 1;
                INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, group_label)
                VALUES (v.id, product_no, v.template_id, 'row', 'Product No', pos, 'CHARACTERISTICS ( Key Parameter )');
            ELSIF f.key = 'process_characteristic' THEN
                pos := pos + 1;
                INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position, group_label)
                VALUES (v.id, process_no, v.template_id, 'row', 'Process No', pos, 'CHARACTERISTICS ( Key Parameter )');
            END IF;
            pos := pos + 1;
            UPDATE format_version_fields SET position = pos WHERE version_id = v.id AND field_id = f.field_id;
        END LOOP;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Kolom yang sudah dipakai versi tidak dihapus; turunkan hanya bila belum dipakai.
ALTER TABLE format_fields DROP CONSTRAINT format_fields_filled_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_filled_check CHECK (
    (kind NOT IN ('operation_number', 'process', 'process_flow') OR area = 'row')
    AND (kind <> 'document_number' OR area = 'header')
);
ALTER TABLE format_fields DROP CONSTRAINT format_fields_kind_check;
ALTER TABLE format_fields ADD CONSTRAINT format_fields_kind_check CHECK (
    kind IN ('text', 'date', 'select', 'symbol', 'rating', 'rpn', 'ap',
             'operation_number', 'process', 'process_flow', 'document_number')
);
