# I2 · Import nama proses dari Excel

- **Tanggal:** 25 November 2026
- **Persetujuan pemilik:** "Ya, rencanakan I2", lalu "Setuju, mulai I2".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian I2.
- **Aturan yang dipakai:** M1.5 dan D-06 (master proses, ID tetap), D-64 (hanya PPAP Engineer dengan E-6 yang mengubah proses), R-API-13 (format error), D-142 (pesan hasil aksi).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Layanan | `internal/process/import.go`: `ImportTemplate` (sheet Processes dan Help) dan `Import`. Setiap baris yang benar disimpan lewat `Create`, jadi aturan nama, hak, dan audit log sama dengan layar Processes. Satu baris satu transaksi. |
| API | `GET /api/v1/process-imports/template` dan `POST /api/v1/process-imports?commit=` (multipart, kolom `file`). Hanya pemegang E-6. Tidak butuh Migration mode. |
| Layar | Template library → **Import processes** (`/processes/import`), juga tautan **Import processes from Excel** di halaman Processes. Laporan per baris: proses baru, sudah ada, atau salah beserta alasannya. |
| Help | Tip baru di bab Processes, dua bahasa. |
| Test | `internal/process/import_test.go` (`TestImportProcesses`, `TestProcessImportTemplate`), baris izin di `permissions_test.go`, `menu.spec.ts`, `web/e2e/processes-import.test.ts` dengan fixture `web/e2e/fixtures/processes-import.xlsx` (nama fiktif berawalan IMP), dan halaman ini di `responsive.test.ts`. |

## 2. Format berkas

Sheet **Processes** (atau sheet pertama), satu proses per baris. Judul kolom tidak membedakan huruf besar kecil, dan urutannya bebas.

| Kolom | Isi |
|---|---|
| Process name | Wajib, paling panjang 200 karakter. |
| Aliases | Opsional. Nama lain proses yang sama, dipisah titik koma, paling banyak 20. |

Batasnya 5 MB dan 2000 baris sekali import. Baris yang kosong seluruhnya dilewati.

## 3. Hasil per baris

| Hasil | Kapan |
|---|---|
| Proses baru | Nama dan semua alias belum dipakai. Saat Import, prosesnya dibuat. |
| Sudah ada | Nama sama dengan nama tampilan proses yang sudah ada. Proses itu tidak diubah. Alias di baris yang belum dimilikinya disebut, tetapi tidak ditambahkan. |
| Salah | Nama sudah menjadi alias proses lain, alias dipakai proses lain, alias ditulis dua kali, nama kosong atau terlalu panjang, atau alias lebih dari 20. |

Preview juga melihat proses yang akan dibuat baris sebelumnya, jadi hasil Preview sama dengan hasil Import.

## 4. Urutan sesudah reset database

1. `create-admin`, lalu import akun (I1).
2. Sebagai PPAP Engineer: import customer dan model (I5), import nama proses (I2), lalu import nomor lama (I4).

## 5. Cara mengulang dari nol

1. Tulis `TestImportProcesses` dan `TestProcessImportTemplate`, lalu `import.go`.
2. Tulis kontrak OpenAPI, `go generate ./...`, lalu `internal/server/processimport.go`, antarmuka `ProcessService`, layanan tiruan di `helpers_test.go`, dan baris izin.
3. `npm --prefix web run gen:api`, lalu `importProcesses` di `client.ts`, halaman, menu, tautan di Processes, teks dua bahasa, dan Help.
4. Buat fixture Excel, tulis test E2E, jalankan semua test, `help:shots`, dan build.

## 6. Konsep yang dipelajari

- **Memakai ulang aturan yang sudah ada.** Import tidak menulis ke tabel sendiri; setiap baris memanggil `Create`. Pemeriksaan nama di import hanya untuk laporan yang jelas, dan database tetap penjaga terakhir (`name_key` unik).
- **Preview yang jujur.** Peta nama diisi juga oleh baris yang baru akan dibuat, jadi tabrakan antarbaris terlihat sebelum Import.

## 7. Asumsi

- **Nama yang sama dengan nama tampilan proses yang sudah ada dilewati, bukan ditolak** (rencana I2 yang disetujui), supaya berkas yang sama aman diunggah lagi. Butir di dokumen rencana yang menyebut "ditolak" sudah diberi catatan.
- **Alias baru untuk proses yang sudah ada tidak ditambahkan.** Mengubah proses tetap lewat tombol Edit di Processes.
- **Error database menghentikan import;** baris yang sudah tersimpan tetap tersimpan, dan unggahan ulang melewatinya.

## 8. Temuan desain baru

Tidak ada.

## 9. Status test

| Test | Hasil |
|---|---|
| `go test ./...`, `go vet`, `gofmt` | Semua lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | 195 lulus |
| `npm --prefix web run test:e2e` | 78 lulus |
| help-shots | Lulus, gambar diperbarui |

## 10. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `splitAliases(text string) []string` yang memotong teks menurut titik koma, membuang spasi di tepi, dan melewati bagian kosong. Uji dengan `"A; B ;;C"` (hasil `A`, `B`, `C`) dan `""` (tanpa hasil). Bandingkan dengan `splitAliases` di `internal/process/import.go`.
