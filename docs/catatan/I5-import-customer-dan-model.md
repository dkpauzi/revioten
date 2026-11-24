# I5 · Import customer dan model dari Excel

- **Tanggal:** 24 November 2026
- **Persetujuan pemilik:** "untuk import akun, karna memuat data customer, berarti buatkan juga untuk import customer", lalu "oke saya setuju dengan usulan anda" (model dan part number ikut di berkas customer), lalu "lanjut I5".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian I5.
- **Aturan yang dipakai:** D-209 (master data milik PPAP Engineer, hak E-11), D-54 (setiap customer punya Project Group), D-42 (model paling banyak masuk satu group model), E-12 (kode customer dua huruf), dan M1.6 (kode khusus).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Paket `internal/masterimport` | `Template` membuat berkas Excel dengan sheet Customers, Models, dan Help, tanpa baris contoh. `Import` membaca kedua sheet. Preview hanya melaporkan; Import menyimpan baris demi baris lewat `customer.Service` dan `product.Service`, jadi aturan, hak, dan audit log sama dengan mengisi layar satu per satu. Data yang sudah ada dilewati. |
| Pemeriksaan tanpa menyimpan | `customer.CheckNewProjectGroup`, `(*customer.Service).CheckNewCustomer` (termasuk kode khusus), `product.CheckName`, dan `product.CheckPartNumber`. Aturannya sama dengan Create, sehingga hasil Preview sama dengan hasil Import. |
| API | `GET /api/v1/master-data/import-template` dan `POST /api/v1/master-data/import?commit=` (multipart, kolom `file`). Hanya pemegang E-11. |
| Layar | Master data → **Import customers and models** (`/customers/import`), dengan tautan dari halaman Customers. Laporannya satu tabel per sheet, dan setiap baris menyebut apa yang dibuat. |
| Help | Tip baru di bab Customers, dua bahasa. |
| Test | `internal/masterimport/masterimport_test.go`, baris izin di `permissions_test.go`, `web/e2e/master-import.test.ts` dengan fixture nama fiktif `web/e2e/fixtures/master-import.xlsx`, dan halaman baru di `responsive.test.ts`. |

## 2. Format berkas

**Sheet Customers:** satu customer per baris.

| Kolom | Isi |
|---|---|
| Project group code | Wajib, misalnya GR1. Project Group yang belum ada dibuat. |
| Project group name | Wajib hanya untuk Project Group baru. |
| Customer code | Wajib, dua huruf. Customer yang sudah ada dilewati tanpa memeriksa kolom lain. |
| Customer name | Wajib untuk customer baru. |

**Sheet Models:** satu part number per baris. Model yang punya beberapa part number ditulis di beberapa baris.

| Kolom | Isi |
|---|---|
| Customer code | Wajib. Boleh customer yang baru ada di sheet Customers pada berkas yang sama. |
| Model group | Opsional. Dibuat bila belum ada. |
| Model | Wajib. Model yang sudah ada tidak diubah. |
| Part number | Opsional. Part number yang sudah ada dilewati. Part number milik model lain ditolak. |
| Part name | Opsional. |

Urutan kolom bebas, asal judulnya sama dengan template. Sheet boleh hanya salah satu. Batasnya 5 MB dan 2000 baris per sheet.

## 3. Urutan sesudah reset database

1. `create-admin`, lalu login sebagai admin.
2. Nyalakan Migration mode, lalu import akun (I1).
3. Login sebagai PPAP Engineer, lalu import customer dan model (I5).
4. Import nomor lama dari Excel (I4).

## 4. Cara mengulang dari nol

1. Tulis `TestImportMasterData` dan `TestMasterDataTemplate`.
2. Buat fungsi pemeriksa di `internal/customer/check.go` dan `internal/product/check.go`.
3. Buat `internal/masterimport/masterimport.go`, lalu sambungkan di `cmd/revioten/main.go`.
4. Tulis kontrak OpenAPI, jalankan `go generate ./internal/server/...`, buat `internal/server/masterimport.go`, tambah baris izin, dan isi `Deps.MasterImport`.
5. Jalankan `npm --prefix web run gen:api`, lalu buat `importMasterData` di klien, halaman, entri menu, teks dua bahasa, dan Help.
6. Buat fixture Excel dan tulis test E2E.

## 5. Konsep yang dipelajari

- **Memakai ulang layanan, bukan menyalin aturan.** Import memanggil layanan master data yang sama dengan layar. Aturan baru di layanan otomatis berlaku juga untuk import.
- **Keadaan sementara untuk Preview.** Preview mengingat apa yang akan dibuat baris sebelumnya, misalnya customer baru, sehingga baris berikutnya di sheet Models boleh merujuknya walaupun belum tersimpan.

## 6. Asumsi

- **Tidak butuh Migration mode,** berbeda dengan import akun. Master data tidak memuat password.
- **Satu baris satu transaksi.** Baris yang sudah tersimpan tetap tersimpan walaupun baris lain gagal. Unggah lagi berkas yang sudah diperbaiki; data yang sudah ada akan dilewati.
- **Data yang sudah ada tidak diubah:** nama customer, Project Group customer, group model sebuah model, dan nama part tidak dibandingkan.
- **Nama model dan group model dicocokkan persis,** termasuk huruf besar kecilnya, seperti batasan unik di database.

## 7. Temuan desain baru

Tidak ada.

## 8. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | 193 lulus |
| `npm --prefix web run test:e2e` | 76 lulus |
| help-shots | Lulus, gambar diperbarui |

## 9. Latihan tulis tangan

Tanpa AI, tulis fungsi Go yang membaca sheet Models dan mengelompokkan part number per model: hasilnya `map[string][]string` dari nama model ke daftar part number, dengan baris tanpa part number dilewati. Bandingkan dengan `readSheet` dan `modelRow` di `internal/masterimport/masterimport.go`.
