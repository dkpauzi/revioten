# I3 · Import isi process template dari Excel

- **Tanggal:** 25 November 2026
- **Persetujuan pemilik:** "Own template first" (template import sendiri dulu; bentuk Tokairika menyusul bila diperlukan), lalu "Setuju, mulai I3".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian I3.
- **Aturan yang dipakai:** M3.2 dan R-TPL-01 (process template), M3.3 dan D-129 (cakupan poin), M3.4 dan R-TPL-05 (versi terbit tidak berubah), D-146 (isi poin per kolom dengan key tetap), D-130 (Relevant Document dari master dokumen terkait), D-214 (PFD tanpa poin template), D-218 (format tersembunyi tidak ikut), D-64 (E-6).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Layanan | `internal/template/process_import.go`: `ContentImportTemplate` dan `ImportContent` di `ProcessService`. Poin disimpan lewat `CreateProcessTemplate`, `NewProcessVersion`, `AddSubprocess`, dan `AddPoint`, jadi aturan, batas, hak, dan audit log sama dengan layar Process templates. |
| API | `GET /api/v1/process-template-imports/template` dan `POST /api/v1/process-template-imports?commit=` (multipart, kolom `file`). Hanya pemegang E-6. Tidak butuh Migration mode. |
| Layar | Template library → **Import process template content** (`/process-templates/import`), juga tautan **Import points from Excel** di halaman Process templates. Laporan per baris dan daftar versi yang menerima poin baru. |
| Help | Tip baru di bab Process templates, dua bahasa. |
| Test | `internal/template/process_import_test.go` (`TestImportContent`, `TestContentImportTemplate`), baris izin di `permissions_test.go`, `menu.spec.ts`, `web/e2e/process-content-import.test.ts` dengan fixture `process-content-processes.xlsx` dan `process-content-import.xlsx` (nama fiktif berawalan IMP3), dan halaman ini di `responsive.test.ts`. |

## 2. Format berkas

Sheet **PFMEA** dan **Control Plan**; paling tidak satu harus ada. Satu poin per baris. Urutan kolom bebas, dan judul kolom tidak membedakan huruf besar kecil.

| Kolom | Isi |
|---|---|
| Process | Wajib. Nama atau alias proses yang terdaftar (import I2 dulu). |
| Sub-process | Wajib. Sub-proses dengan nama itu dipakai, atau dibuat bila belum ada. |
| Point text | Wajib, paling panjang 500 karakter. |
| Use point for | `General` (atau kosong), `Customer`, `Model group`, `Model`. |
| Customer code, Model group, Model | Untuk Customer, Model group, dan Model. Kode lama customer juga dikenali. Group model dan model dicari di customer itu. |
| Kolom isi | Label kolom teks dan rating dari versi terakhir setiap format yang tidak disembunyikan, atau key tetapnya (`failure_mode`, `severity`). Rating angka bulat 1 sampai 10. |

Template unduhan dibuat dari format yang berlaku saat itu. Kolom yang terisi sendiri (nomor operasi, proses, Process Flow, RPN, AP), Relevant Document, dan kolom simbol tidak ada di template; kolom yang tidak dikenal membuat berkas ditolak, supaya salah ketik judul kolom tidak diam-diam membuang isinya.

Batasnya 10 MB dan 5000 baris sekali import.

## 3. Hasil

- **Poin baru:** belum ada poin dengan jenis dokumen, sub-proses, teks (tanpa membedakan huruf besar kecil), dan cakupan yang sama di versi target.
- **Sudah ada:** poin itu sudah ada (juga bila disembunyikan). Dilewati dan tidak diubah.
- **Salah:** proses, customer, group model, atau model tidak ditemukan; rating di luar 1 sampai 10; teks terlalu panjang; sub-proses yang sama namanya sedang disembunyikan; atau batas 50 sub-proses atau 100 poin terlewati.
- **Versi:** proses tanpa template mendapat template dan versi 1; proses dengan versi terakhir terbit mendapat versi berikutnya sebagai salinan; draft yang sudah ada langsung dipakai. Laporan menyebut versi itu per proses. Penerbitan tetap lewat **Publish version N**.

## 4. Urutan sesudah reset database

1. `create-admin`, lalu import akun (I1).
2. Sebagai PPAP Engineer: import customer dan model (I5), nama proses (I2), isi process template (I3), lalu nomor lama (I4).
3. Terbitkan versi process template di halaman Process templates.

## 5. Cara mengulang dari nol

1. Tulis `TestImportContent` dan `TestContentImportTemplate`, lalu `process_import.go`.
2. Tulis kontrak OpenAPI, `go generate ./...`, lalu `internal/server/processcontentimport.go`, antarmuka `ProcessTemplateService`, layanan tiruan, dan baris izin.
3. `npm --prefix web run gen:api`, lalu `importProcessContent` di `client.ts`, halaman, menu, tautan, teks dua bahasa, dan Help.
4. Buat fixture, tulis test E2E, jalankan semua test, `help:shots`, dan build.

## 6. Konsep yang dipelajari

- **Simulasi untuk Preview.** Importer menyimpan keadaan setiap proses di memori (versi target, sub-proses, poin). Preview hanya memperbarui keadaan itu; Import juga memanggil layanan lalu memuat ulang versinya. Karena keputusan per baris memakai kode yang sama, hasil Preview sama dengan hasil Import.
- **Kolom mengikuti data, bukan kode.** Daftar kolom isi dibaca dari format yang berlaku, sama seperti tombol Columns di layar, jadi kolom baru di format langsung muncul di template import.

## 7. Asumsi

- **Satu poin satu transaksi,** bukan satu proses satu transaksi seperti di rencana. Import memakai ulang layanan yang sudah ada, dan setiap layanan punya transaksinya sendiri. Bila import berhenti di tengah, poin yang sudah tersimpan tetap tersimpan, dan unggahan ulang melewatinya.
- **Poin yang sudah ada tidak diubah,** termasuk isinya. Perubahan isi tetap lewat layar.
- **Varian poin tidak diimpor** (usulan di rencana yang disetujui). Baris dengan Use point for selain General menjadi poin khusus, bukan varian.
- **Laporan diurutkan per sheet:** PFMEA dulu, lalu Control Plan.

## 8. Temuan desain baru

Tidak ada.

## 9. Status test

| Test | Hasil |
|---|---|
| `go test ./...`, `go vet`, `gofmt` | Semua lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | 195 lulus |
| `npm --prefix web run test:e2e` | 79 lulus |
| help-shots | Lulus, gambar diperbarui |

## 10. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `pointKey(typeID, text, level, targetID string) string` dan gunakan sebuah `map[string]bool` untuk menandai baris kedua `Paste volume` dan `paste volume` (jenis dan cakupan sama) sebagai "sudah ada". Bandingkan dengan `pointKey` di `internal/template/process_import.go`.
