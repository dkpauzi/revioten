# D-222 · Kolom No karakteristik di Control Plan

- **Tanggal:** 23 November 2026
- **Keputusan:** D-222 (brief v1.51).
- **Persetujuan pemilik:** "jika iya, maka itu wajib ada, karena itu penomorannya", dan "isi sesuai kolom mana yang di isi ... untuk nomornya buat berurutan saja tidak masalah".
- **Aturan yang dipakai:** D-122 dan D-123 (kolom terisi otomatis), R-TPL-02 (ID tetap), M3.4 (versi terbit tidak berubah).

## 1. Temuan dari referensi

Di SC-TR-020-04, grup CHARACTERISTICS ( Key Parameter ) punya kolom No (F) sebelum PRODUCT CHARACTERISTICS dan kolom No (I) sebelum PROCESS CHARACTERISTICS.
- Kolom F menomori semua baris 1, 2, 3, dan mulai lagi di setiap proses. Beberapa baris ESD dan Strain Test mulai lagi dari 1, tampaknya diatur tangan.
- Kolom I selalu "--".
- Form Control Plan v1 belum punya kedua kolom itu.

Aturan pemilik untuk sistem: nomor diisi pada kolom karakteristik yang memang diisi, dan cukup berurutan.

## 2. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Jenis kolom baru | `product_char_number` dan `process_char_number`: kolom terisi otomatis, satu per versi, hanya kolom baris, dan tidak bisa diketik. |
| Migrasi 00057 | Batasan jenis kolom diperbarui. Control Plan v1 versi draft mendapat kolom **Product No** sebelum Product Characteristics dan **Process No** sebelum Process Characteristics, dalam grup CHARACTERISTICS ( Key Parameter ). Kolom lain digeser urutannya. |
| Cetakan | `internal/printout/charno.go`: nomor berurutan, dan mulai lagi dari 1 di setiap proses (baris berurutan dengan bagian, nomor langkah, dan nama proses yang sama). Isian "--" tercetak "--", isian kosong dibiarkan kosong. Judul kolom tercetak "No". |
| Workbook | `charNumbers` di `web/src/lib/documents.ts` menghitung dengan aturan yang sama. Kolom hanya bisa dibaca, dan nomor baris lain dihitung ulang sesudah satu sel disimpan. |
| Layar format | Kedua jenis tampil di daftar jenis kolom (**Add a field**), jadi template yang sudah terbit bisa menambahkannya lewat versi baru. |
| Test | `internal/printout/charno_test.go`, `internal/template/charno_test.go`, `documents.spec.ts`, dan urutan kolom di `formats.test.ts`. |

## 3. Cara mengulang dari nol

1. Tulis `TestCharacteristicNumbers` (aturan penomoran) dan `TestCharacteristicNumberColumns` (kolom di template).
2. Tambah jenis kolom di `template.go`, lalu `charno.go` dan pemakaiannya di `Build`.
3. Migrasi 00057, lalu sesuaikan test yang menghitung urutan kolom.
4. Kontrak OpenAPI (enum jenis kolom), `go generate`, `gen:api`, `charNumbers`, workbook, layar format, teks dua bahasa, dan Help.

## 4. Konsep yang dipelajari

- **Nilai yang dihitung, bukan disimpan.** Nomor karakteristik bergantung pada baris lain, jadi tidak disimpan di baris. Nomor dihitung saat ditampilkan dan dicetak. Snapshot revisi terbit tetap tidak berubah, karena isinya sama dan cara hitungnya sama.
- **Satu aturan di dua tempat.** Server (Go) dan workbook (TypeScript) memakai aturan yang sama, dan keduanya punya test dengan contoh yang sama.

## 5. Asumsi

- **Mulai lagi:** nomor mulai lagi dari 1 di setiap proses. Mulai ulang khusus pada baris ESD di referensi tidak ditiru, sesuai jawaban pemilik bahwa berurutan sudah cukup.
- **Label:** di format kedua kolom berlabel "Product No" dan "Process No", karena label dalam satu versi harus unik. Di cetakan judulnya "No".
- **Versi:** hanya versi draft yang diubah migrasi. Bila Control Plan v1 di database pemilik sudah terbit, kedua kolom ditambahkan lewat versi baru (Add a field, jenis Product characteristic number dan Process characteristic number).

## 6. Temuan desain baru

Tidak ada.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | Lulus |
| `npm --prefix web run test:e2e` | 73 lulus |
| Tur (`fast`) dan help-shots | Lulus |

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi Go yang menerima daftar baris (nomor langkah, nama proses, isi karakteristik), lalu mengembalikan nomor urut untuk baris yang isinya bukan kosong dan bukan "--". Nomor mulai lagi dari 1 setiap kali nomor langkah atau nama proses berganti.
