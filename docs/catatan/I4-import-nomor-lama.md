# I4 · Import nomor lama dari Excel, dengan revisi

- **Tanggal:** 25 November 2026
- **Persetujuan pemilik:** "I1, apakah bisa saya import menggunakan excel? ... ini juga berlaku untuk import data nomor dokumen lama", lalu "opsi B" (revisi di belakang nomor disimpan dan mengisi menu Legacy documents), lalu "Setuju, mulai I4".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian I4.
- **Aturan yang dipakai:** M2.5 dan R-NUM-06 (impor nomor lama), D-216 dan E-140 (satu baris satu set, satu batch), M2.4 (revisi awal), D-215 dan D-217 (menu Legacy documents, revisi pindah ke kolomnya), R-POR-01 (kode customer di nomor), E-11 (kode lama customer), E-67 (customer nonaktif).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Pemisahan revisi | `Pattern.SplitRevision` di `internal/numbering/format.go`: `SC-KT-001-05` menjadi `SC-KT-001` dan revisi 5, menurut jumlah digit revisi di Document types. Berlaku untuk impor tempel dan impor berkas. |
| Inti impor | `internal/numbering/import.go`: `importRows` dipakai bersama oleh impor tempel dan impor berkas. Status dan revisi dibaca per baris. Revisi disimpan di `initial_revision` dan dicatat di audit log. |
| Impor berkas | `internal/numbering/import_file.go`: `ImportTemplate` (sheet Numbers dan Help) dan `ImportFile`. Baris dikelompokkan per customer; setiap customer satu transaksi. |
| Menu Legacy documents | Daftar nomor tanpa dokumen (`/api/v1/migration/numbers`) sekarang membawa revisi. Saat nomor diketik, kolom revisi yang kosong terisi, juga untuk nomor yang mengikuti (satu set). |
| API | `GET /api/v1/document-number-imports/template` dan `POST /api/v1/document-number-imports/file?commit=&one_set=` (multipart, kolom `file`). Hanya pemegang E-8. Laporan impor memuat `revision`, dan untuk berkas juga `customer_code` dan `status` per baris. |
| Layar | Numbers → **Import legacy numbers**: bagian **From an Excel file** di atas, **Paste from the register** di bawah. Laporannya satu komponen (`web/src/lib/numbers/ImportReport.svelte`) dan sekarang menampilkan semua nomor beserta revisinya. |
| Help | Tip baru di bab nomor lama, dua bahasa. |
| Test | `format_test.go` (`TestSplitRevision`), `import_file_test.go` (`TestImportRevision`, `TestImportFile`, `TestNumberImportTemplate`), baris izin di `permissions_test.go`, `numbers.spec.ts`, `web/e2e/numbers-import.test.ts` dengan fixture `web/e2e/fixtures/numbers-import.xlsx`, dan halaman ini di `responsive.test.ts`. |

## 2. Format berkas

Sheet **Numbers**, satu set per baris. Urutan kolom bebas, asal judulnya sama dengan template.

| Kolom | Isi |
|---|---|
| Customer code | Wajib. Kode sekarang atau kode lama customer. |
| PFD, PFMEA, CP | Nomor satu set, boleh diakhiri revisi, misalnya `SH-TR-020-02`. Dengan centang **One set per row**, cukup satu nomor. |
| Status | `Released` atau `Reserved`. Kosong berarti Released. |

Batasnya 5 MB dan 5000 nomor sekali impor.

## 3. Urutan sesudah reset database

1. `create-admin`, lalu login sebagai admin.
2. Import akun (I1). Preview bisa kapan saja; Import butuh Migration mode.
3. Login sebagai PPAP Engineer, lalu import customer dan model (I5), dan nama proses (I2).
4. Import nomor lama dari Excel (I4).

## 4. Cara mengulang dari nol

1. Tulis `TestSplitRevision`, lalu `SplitRevision`.
2. Tulis `TestImportRevision` dan `TestImportFile`, lalu pisahkan `importRows` dari `Import`, tambah status dan revisi per baris, dan buat `import_file.go`.
3. Tambah `n.initial_revision` ke `ListMigratableNumbers`, lalu `go generate ./...`.
4. Tulis kontrak OpenAPI, buat `internal/server/numberimport.go`, perbarui `toAPIImportReport`, antarmuka layanan, dan baris izin.
5. `npm --prefix web run gen:api`, lalu klien, komponen laporan, halaman, menu Legacy documents, teks dua bahasa, dan Help.
6. Buat fixture Excel, tulis test E2E, jalankan semua test dan `help:shots`.

## 5. Konsep yang dipelajari

- **Satu inti, dua pintu masuk.** Impor tempel dan impor berkas hanya berbeda cara membaca barisnya. Sesudah itu keduanya memanggil `importRows`, jadi aturan register (kunci penghitung, nomor ganda, laporan lubang) hanya ditulis sekali.
- **Memakai kolom yang sudah ada.** Revisi terakhir nomor lama sama dengan revisi awal dokumen di sistem, jadi `initial_revision` cukup. Batasnya dicatat sebagai E-146.

## 6. Asumsi

- **`-5` dan `-05` sama-sama revisi 5.** Akhiran 1 sampai sebanyak digit revisi diterima; yang lebih panjang membuat nomor tidak dikenali.
- **Nomor yang sudah ada tidak diubah,** termasuk revisinya, supaya impor ulang aman.
- **Nomor yang mengikuti (satu set) tidak punya revisi.** Revisinya diisi di menu Legacy documents.
- **Satu customer satu transaksi.** Customer yang sudah tersimpan tetap tersimpan bila customer lain gagal.
- **E-146, diputuskan D-225 (opsi a, 25 November 2026):** revisi disimpan di `initial_revision`; revisi 0 tidak mengisi kolom revisi di menu Legacy documents.
- **Model tidak ikut di berkas** (berbeda dengan usulan awal di rencana); model tetap dipilih saat dokumen dimigrasi.

## 7. Temuan desain baru

- **E-146** · Revisi nomor lama disimpan sebagai revisi awal (Ambigu, tidak menghambat). Diputuskan D-225: opsi a.

## 8. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | 194 lulus |
| `npm --prefix web run test:e2e` | 77 lulus |
| help-shots | Lulus, gambar diperbarui |

## 9. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `splitRevision(text string, digits int) (string, int, bool)` untuk pola tetap `XX-YY-NNN`: pisahkan akhiran `-NN` bila panjangnya 1 sampai `digits` angka. Uji dengan `SC-KT-001-05`, `SC-KT-001`, dan `SC-KT-001-123`. Bandingkan dengan `Pattern.SplitRevision` di `internal/numbering/format.go`.
