-- Bagian B (E-144): PFD dicetak sebagai diagram A4 tegak, seperti
-- SH-TR-020-02. Layout bawaan PFD (Standard PFD) yang masih memakai arah
-- bawaan lama (mendatar) diubah menjadi tegak. Layout yang sudah diubah
-- pengguna ke arah lain tidak disentuh, karena hanya nilai landscape yang
-- diganti, dan layout lain tetap bisa diatur di Template library → Print
-- layouts.

-- +goose Up
UPDATE print_layouts
SET layout = jsonb_set(layout, '{orientation}', '"portrait"')
WHERE name = 'Standard PFD' AND layout ->> 'orientation' = 'landscape';

-- +goose Down
UPDATE print_layouts
SET layout = jsonb_set(layout, '{orientation}', '"landscape"')
WHERE name = 'Standard PFD' AND layout ->> 'orientation' = 'portrait';
