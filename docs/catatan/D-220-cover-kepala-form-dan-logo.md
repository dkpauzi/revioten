# D-220 · Cover, kepala form, dan logo seperti TR-020 (bagian C)

- **Tanggal:** 22 sampai 23 November 2026
- **Keputusan:** D-220 (brief v1.49), menutup temuan E-144 bersama D-219.
- **Persetujuan pemilik:** "Ya, bagian C". Logo: "jadikan png, dan juga buat logo bisa di atur oleh admin".
- **Aturan yang dipakai:** R-PRT-07 (cover dan slot dari rute), R-REV-04 dan M6.8 (riwayat revisi), R-TPL-02 (ID tetap, label hanya tampilan), M3.4 (versi terbit tidak berubah).
- **Acuan:** Cover, lembar data, sel gabungan, dan gambar di SH-TR-020-02, SF-TR-020-03, dan SC-TR-020-04.

## 1. Tujuan

Cover dan kepala lembar cetakan dibuat sama dengan dokumen Tokairika, dengan lebar sel yang boleh menyesuaikan.

## 2. Temuan dari referensi

| Bagian | Isi referensi |
|---|---|
| Cover (ketiganya) | Logo SIIX di kiri atas, "PCP/PFMEA/PFD No :" dan nomor di kanan atas. Judul, lalu Customer, Model Name, Part Name, Part Number, dan Process or Line. Document Revision History. INHOUSE APPROVAL: tahap, ruang tanda tangan, nama, jabatan. CUSTOMER APPROVAL, nama customer. |
| Kepala CP | Logo dan PT. SIIX EMS INDONESIA, CONTROL PLAN, tabel Department / Name / Signature / Date (General Mgr sampai Prepared by). Control Plan No., Product, Core Team. Part number dan nama part besar di tengah. Remark, Customer Name, Rev. Description. Grup kolom PROCESS, CHARACTERISTICS ( Key Parameter ), CONTROL METHOD. |
| Kepala PFMEA | Nama customer besar, judul tiga baris. Part Name, Part No., Model, Process, Process Responsibility, Key Date, Revision, FMEA NO., Prepared by, Core Team, Remark. Grup kolom Current Process dan Action Results. |
| Kepala PFD | DOC. NO, "Process Flow Chart for D660A Unit", CUSTOMER, MODEL, PART NAME, PART NUMBER, PROCESS LINE. |
| Legenda simbol CP | Ada di lembar CP, tetapi di luar area cetak, jadi tidak dicetak dan tidak dibuat. |

## 3. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Migrasi 00056 | Nama perusahaan dan logo di `system_settings`. Key tetap untuk field header bawaan. Field Remark untuk PFMEA v1 dan Control Plan v1. Judul grup kolom. Field Remark dan judul grup hanya ditambahkan ke versi draft. |
| `internal/settings/company.go` | `Company`, `SetCompanyName`, `SetLogo`, `ResetLogo` (admin, A-3), dan `Logo` (logo unggahan atau bawaan). PNG diperiksa dari isinya, paling besar 1 MB dan 2000 piksel. Audit `system_settings.logo_changed` menyimpan hash, bukan gambar. |
| Snapshot | `part_names` ikut dibekukan (tambahan; snapshot lama tanpa nama part). |
| `internal/printout/head.go` | `FormHead`, `HeadHTML`, dan `CoverHTML` untuk PDF dan pratinjau. Tinggi kepala per jenis ada di `headMM`. Cover PDF berupa halaman A4 tegak bernama (`@page cover`). |
| `internal/printout/xlsxhead.go` | Kepala dan cover Excel. Kolom logis dipetakan ke kolom lembar, lalu sel digabung. Logo ditempel dengan jangkar satu sel. |
| API dan layar | `GET`/`PUT /api/v1/company`, `GET`/`PUT`/`DELETE /api/v1/company/logo`. Pratinjau memakai `head_html`. Admin → Settings → **Company name and logo**. |
| Test | `company_test.go`, `print_head_test.go`, contoh TR-020 untuk CP, PFMEA, dan PFD di `plan_test.go`, tabel izin, dan E2E `company-logo.test.ts`. |

## 4. Cara mengulang dari nol

1. Baca referensi dengan openpyxl: lebar kolom, tinggi baris, sel gabungan, isi, ukuran huruf. Lalu baca gambar di `xl/drawings/*.xml`.
2. Migrasi 00056, lalu `print_head_test.go`.
3. `company_test.go`, lalu `company.go` dan query `GetCompany`, `GetCompanyLogo`, `SetCompanyName`, `SetCompanyLogo`.
4. `head.go`, lalu sambungkan ke template HTML dan `BuildPlan` (tinggi kepala).
5. `xlsxhead.go`, lalu `writeData` dan `writeDiagram` memanggil `writeFormHead`.
6. API, `CompanyLogo.svelte`, `PreviewSheet.svelte`, teks dua bahasa, Help, dan E2E.
7. Periksa mata:
   - `REVIOTEN_SAMPLE_DIR=<folder> go test -short ./internal/printout -run TestSampleTokairika`;
   - HTML dipotret dengan Chrome headless;
   - Excel disimpan sebagai HTML lewat Excel (lembar cover dihapus dulu di memori), lalu dipotret.

## 5. Konsep yang dipelajari

- **Satu HTML untuk dua tempat.** Kepala dibuat sekali di server. PDF memakainya, dan pratinjau menerimanya sebagai `head_html`, jadi tampilan layar dan kertas tidak bisa berbeda. html/template tetap meng-escape isinya.
- **Kolom logis.** Kepala form punya tujuh kolom (CP), sedangkan lembar CP punya enam belas kolom data. Kolom logis dipetakan menurut lebar relatif, lalu sel digabung.
- **Halaman bernama di CSS.** `@page cover { size: A4 portrait }` membuat cover tegak walau lembar isinya A3 mendatar.

## 6. Asumsi

- Cetakan memakai logo yang berlaku saat dicetak, juga untuk cetak ulang revisi lama (pertanyaan 2 di rencana, belum dijawab pemilik).
- Nama perusahaan bawaan "PT. SIIX EMS INDONESIA", dari kepala CP referensi.
- Field Remark dan judul grup kolom hanya ditambahkan ke versi draft. Bila Control Plan v1 atau PFMEA v1 di database pemilik sudah terbit, keduanya diisi lewat versi baru di Format templates.
- Kolom "No" karakteristik (F15, I15 di referensi) tidak ada di form sistem, jadi tidak dibuat.
- Isi slot tanda tangan dan tabel persetujuan CP mengikuti rute customer. Rute "Up to QA QC Manager" tidak punya General Mgr.

## 7. Temuan desain baru

Tidak ada. E-144 sudah diputuskan seluruhnya (D-219, D-220).

## 8. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run check` | 0 error |
| `npm --prefix web test` | 191 lulus |
| `npm --prefix web run test:e2e` | 72 lulus (baru: `company-logo.test.ts`) |
| Tur (`fast`) dan help-shots | Lulus |
| Pemeriksaan mata | HTML dan Excel untuk CP, PFMEA, PFD, dan cover; pratinjau CP di aplikasi |

## 9. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `checkLogo(data []byte) error` yang menolak berkas lebih dari 1 MB, berkas yang tidak diawali tanda tangan PNG `\x89PNG\r\n\x1a\n`, dan gambar yang lebih lebar dari 2000 piksel (pakai `png.DecodeConfig`). Lalu tulis tiga kasus test-nya.
