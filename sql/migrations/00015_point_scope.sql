-- M3.3: cakupan poin dan prioritas (D-129, R-TPL-03, D-39).
--
-- Satu process template per proses (keputusan E-72). Setiap poin punya
-- cakupan: umum, customer, group model, atau model. Di layar cakupan ini
-- tampil sebagai "biasanya dipakai untuk". Poin boleh punya varian untuk
-- cakupan yang lebih sempit, misalnya laser type yang berbeda per model
-- walaupun poinnya biasanya umum. Untuk satu model, varian yang paling
-- spesifik menang: model, lalu group model, lalu customer, lalu umum.
--
--   template_points               identitas poin mendapat cakupan, dan varian
--                                 menunjuk poin asalnya (variant_of)
--   process_template_variants     isi varian dalam satu versi: teks dan
--                                 tersembunyi, mengikuti urutan poin asalnya
--
-- Cakupan bagian dari identitas poin, jadi tidak pernah diubah (role
-- aplikasi tidak punya UPDATE untuk kolom ini). D-131 (keputusan E-78): cakupan
-- poin yang salah dibetulkan dengan menyembunyikan poin itu dan membuat poin baru.

-- +goose Up
ALTER TABLE template_points
    ADD COLUMN scope text NOT NULL DEFAULT 'general',
    ADD COLUMN customer_id uuid REFERENCES customers (id),
    ADD COLUMN model_group_id uuid REFERENCES model_groups (id),
    ADD COLUMN model_id uuid REFERENCES models (id),
    ADD COLUMN variant_of uuid,
    -- Tepat satu sasaran sesuai tingkat cakupannya; umum tanpa sasaran.
    ADD CONSTRAINT template_points_scope_check CHECK (
        (scope = 'general' AND customer_id IS NULL AND model_group_id IS NULL AND model_id IS NULL)
        OR (scope = 'customer' AND customer_id IS NOT NULL AND model_group_id IS NULL AND model_id IS NULL)
        OR (scope = 'model_group' AND customer_id IS NULL AND model_group_id IS NOT NULL AND model_id IS NULL)
        OR (scope = 'model' AND customer_id IS NULL AND model_group_id IS NULL AND model_id IS NOT NULL)
    ),
    -- Varian selalu lebih sempit dari umum.
    ADD CONSTRAINT template_points_variant_scope_check CHECK (variant_of IS NULL OR scope <> 'general'),
    -- Varian tetap di sub-proses dan jenis dokumen yang sama dengan poin asalnya.
    ADD CONSTRAINT template_points_variant_fkey FOREIGN KEY (variant_of, subprocess_id, document_type_id)
        REFERENCES template_points (id, subprocess_id, document_type_id),
    -- Sasaran foreign key gabungan dari process_template_variants.
    ADD CONSTRAINT template_points_id_variant_key UNIQUE (id, variant_of);

-- Satu poin paling banyak punya satu varian untuk setiap cakupan.
CREATE UNIQUE INDEX template_points_variant_scope_key
    ON template_points (variant_of, scope, (coalesce(customer_id, model_group_id, model_id)))
    WHERE variant_of IS NOT NULL;

-- Isi varian dalam satu versi. Poin asalnya harus ada di versi yang sama, dan
-- variant_of identitas varian harus sama dengan origin_point_id.
CREATE TABLE process_template_variants (
    version_id uuid NOT NULL,
    point_id uuid NOT NULL,
    origin_point_id uuid NOT NULL,
    text text NOT NULL,
    hidden boolean NOT NULL DEFAULT false,

    PRIMARY KEY (version_id, point_id),
    CONSTRAINT process_template_variants_origin_fkey FOREIGN KEY (version_id, origin_point_id)
        REFERENCES process_template_points (version_id, point_id),
    CONSTRAINT process_template_variants_point_fkey FOREIGN KEY (point_id, origin_point_id)
        REFERENCES template_points (id, variant_of),
    CONSTRAINT process_template_variants_text_check CHECK (
        text <> '' AND text = btrim(text) AND char_length(text) <= 500
    )
);

CREATE INDEX process_template_variants_origin_idx ON process_template_variants (version_id, origin_point_id);

-- Hak sekecil mungkin (E-06): varian tidak pernah dihapus, hanya teks dan
-- tersembunyi yang bisa diubah. Hak INSERT tingkat tabel template_points
-- sudah mencakup kolom baru; UPDATE-nya tetap tidak ada.
GRANT SELECT, INSERT ON process_template_variants TO revioten_app;
GRANT UPDATE (text, hidden) ON process_template_variants TO revioten_app;

-- +goose Down
DROP TABLE process_template_variants;
DROP INDEX template_points_variant_scope_key;
ALTER TABLE template_points
    DROP CONSTRAINT template_points_id_variant_key,
    DROP CONSTRAINT template_points_variant_fkey,
    DROP CONSTRAINT template_points_variant_scope_check,
    DROP CONSTRAINT template_points_scope_check,
    DROP COLUMN variant_of,
    DROP COLUMN model_id,
    DROP COLUMN model_group_id,
    DROP COLUMN customer_id,
    DROP COLUMN scope;
