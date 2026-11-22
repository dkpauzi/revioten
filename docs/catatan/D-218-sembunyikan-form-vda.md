# D-218 · Format template dan form set tersembunyi, nama v1 dan v2 (bagian A)

- **Tanggal:** 22 November 2026
- **Keputusan:** D-218 (brief v1.47).
- **Persetujuan pemilik:** "ya, mulai bagian A", setelah pemeriksaan cetakan Tokairika (temuan E-144) dan rencana tiga bagian A, B, C.
- **Aturan yang dipakai:** R-TPL-02 (ID tetap, nama hanya tampilan), M3.4 (versi terbit tidak berubah), D-133 dan D-139 (form set), D-209 (E-11).

## 1. Tujuan

Pekerjaan sekarang fokus ke form yang dipakai hari ini (bentuk Tokairika). Form AIAG-VDA dan Control Plan AIAG 2024 tidak dihapus, tetapi disembunyikan supaya tidak ikut ditawarkan. Nama format template diganti menjadi versi yang mudah dibedakan.

## 2. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Migrasi 00054 | Kolom `hidden` di `format_templates` dan `form_sets` (dengan `GRANT UPDATE (hidden)`), constraint "set bawaan tidak boleh tersembunyi", dan nama baru: PFD → PFD v1, CP → Control Plan v1, Control Plan AIAG 2024 → Control Plan v2, PFMEA AIAG 4th → PFMEA v1, PFMEA AIAG-VDA 2019 → PFMEA v2. PFMEA v2, Control Plan v2, dan set AIAG-VDA disembunyikan (set hanya bila belum dipilih customer). |
| Layanan (`internal/template`) | `SetFormatTemplateHidden` dan `SetFormSetHidden` (E-11, audit `format_template.hidden_changed` dan `form_set.hidden_changed`). Form set bawaan dan form set yang dipilih customer ditolak (409, `hidden:not_allowed`). `SetCustomerFormSet` menolak set tersembunyi (`form_set_id:not_allowed`). Baris form set dikunci di kedua jalur supaya tidak bersamaan. |
| API | `PUT /api/v1/format-templates/{id}/hidden` dan `PUT /api/v1/form-sets/{id}/hidden` dengan `{"hidden": bool}`. Field `hidden` di `FormatTemplate` dan `FormSet`. |
| Layar | Format templates: centang **Show hidden**, tanda *Hidden* di pilihan, tombol **Hide this template** / **Show this template**. Form sets: centang **Show hidden**, badge *Hidden*, tombol **Hide/Show {nama}** (bukan untuk set bawaan). Pilihan customer, sheet form set, sumber salinan, dan kolom **Columns** di Process templates tidak menawarkan yang tersembunyi. |
| Help dan tur | Bab Format templates dan Form sets (en dan id). Keterangan langkah 8 tur. |

## 3. Cara mengulang dari nol

1. Tulis `hidden_test.go`: nama baru dan isi awal, hak E-11, audit, set bawaan, set yang dipilih customer, dan customer memilih set tersembunyi.
2. Migrasi 00054, query (`SetFormatTemplateHidden`, `LockFormSet`, `SetFormSetHidden`, `CustomersUsingFormSet`), `go generate ./internal/db/`.
3. Fungsi layanan sampai test lulus; sesuaikan test lama ke nama baru, dan tampilkan set AIAG-VDA dulu di test yang memakainya.
4. Kontrak OpenAPI, `go generate ./internal/server/`, handler, dan dua baris di tabel izin.
5. `formSets.spec.ts` (`visibleItems`, `sheetOptions` dengan versi yang sedang dipakai), `formats.spec.ts` (tanda Hidden), lalu kedua halaman, teks dua bahasa, Help, E2E.

## 4. Konsep yang dipelajari

- **Menyembunyikan, bukan menghapus.** Data tetap utuh dan bisa dimunculkan lagi. Yang berubah hanya pilihan yang ditawarkan. Dokumen lama tidak terpengaruh, karena dokumen menunjuk versi format lewat ID.
- **Pengecualian "yang sedang dipakai".** Daftar pilihan tetap memuat item tersembunyi yang sedang dipilih (`keepId`), supaya isian lama tidak tiba-tiba kosong.
- **Kunci baris di dua jalur.** Memilih set untuk customer dan menyembunyikan set sama-sama mengunci baris form set, jadi tidak mungkin terjadi "disembunyikan tepat saat dipilih".

## 5. Asumsi

- Nama form set (AIAG 4th set, AIAG-VDA set) tidak diganti, karena pemilik hanya menyebut nama format.
- Migrasi hanya mengganti nama template bawaan yang namanya belum diubah pengguna.
- Server tidak menolak versi format tersembunyi sebagai sheet form set; penyaringan dilakukan di layar. Form set yang sudah memakainya tetap sah.

## 6. Temuan desain baru

- **E-143:** password standar untuk akun hasil import berbenturan dengan D-132. Terbuka.
- **E-144:** cetakan dan simbol alur belum sama dengan dokumen Tokairika. Terbuka, menunggu bagian B dan C.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus (baru: `hidden_test.go`; test lama memakai nama baru) |
| `npm --prefix web test` | 191 lulus |
| `npm --prefix web run check` | 0 error |
| `npm --prefix web run test:e2e` | 71 lulus (`formats.test.ts` dan `formsets.test.ts` memakai **Show hidden**) |
| Tur (`fast`, tanpa jendela) dan help-shots | Lulus; gambar Format templates dan Form sets diperbarui |

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi TypeScript `visibleItems(items, showHidden, keepId)` yang mengembalikan item yang tidak tersembunyi, semua item bila `showHidden` benar, dan selalu menyertakan item dengan `id === keepId`. Lalu tulis tiga kasus test-nya.
