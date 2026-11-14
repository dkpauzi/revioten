# Tahap 7 · Migrasi dan bukti nilai (M7.1 sampai M7.7)

- **Tanggal:** 15 November 2026
- **Aturan yang dipakai:** R-MIG-01 sampai R-MIG-07, R-REV-01, R-REV-05, R-CMP-06, R-SEC-11, D-69, D-151, D-173, RK-12, RK-16, RK-50, UC-07, UC-60 sampai UC-64.
- **Persetujuan pemilik:** "lanjutkan ke M7 hingga selesai" (15 November 2026).
- **Tidak dibangun:** migrasi saat disentuh untuk versi perusahaan (R-MIG-08, usulan, hanya dirancang di UML), arsip berkas Excel lama per dokumen (R-REV-05 bagian opsional), impor isi dari Excel (D-151).
- **Belum selesai menurut kriteria milestone:** M7.6 "estimasi usaha migrasi tersedia" butuh pemilik memigrasikan dua sampai tiga model sungguhan dan mengisi menitnya. Alatnya sudah ada.

## 1. Apa yang dibuat

| Milestone | Hasil |
|---|---|
| M7.1 Form dokumen lama | Documents → **Enter a legacy document** (E-10, mode migrasi menyala): nomor asli dari register, model, revisi saat ini, tanggal berlaku, penyusun, dan approver. Dokumen masuk sebagai draft di revisi itu, misalnya 4 untuk -04, dengan asal `migrated`. |
| M7.2 Riwayat revisi sebagai teks | Di halaman dokumen, bagian **Migration baseline**: baris riwayat (revisi, tanggal, deskripsi) dari tabel lama. Tercetak di tabel riwayat revisi sebelum revisi buatan sistem. |
| M7.3 Pembanding Excel | **Compare with the legacy Excel**: berkas `.xlsx` hanya dibandingkan, tidak diimpor dan tidak disimpan. Hasilnya dua daftar selisih (ASUMSI E-112). |
| M7.4 Penautan baris | **Link rows to process templates**: usulan poin lewat kemiripan teks, dikonfirmasi orang. Baris tertaut menerima propagasi versi template berikutnya, dan isi yang berbeda tampil sebagai perbedaan lokal. |
| M7.5 Verifikasi hardcopy | Scan hardcopy diunggah, lalu petugas migrasi lain menekan **Verify against the hard copy**. Revisi terbit sebagai Released (migrated) dengan snapshot dan SHA-256. Saklar **Migration mode** di Settings, bawaannya mati. |
| M7.6 Pilot | Menit kerja per dokumen diisi saat verifikasi dan dijumlahkan di dasbor. Pilotnya sendiri tugas pemilik. |
| M7.7 Dasbor ukuran | Documents → **Dashboard** (L-2): baris yang mengikuti template, dokumen per status, target yokoten, waktu revisi, dan usaha migrasi, untuk semua customer atau satu customer. |

Perbaikan ikutan: daftar di halaman Documents selalu menulis status Draft; sekarang menulis Draft, Released, atau Released (migrated).

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00041_migration.sql` |
| Query | `sql/queries/migration.sql`, tambahan di `documents.sql`, `revisions.sql`, `printing.sql`, `settings.sql` |
| Aturan | `internal/document/migrate.go`, `link.go`, `excelcompare.go`; `internal/metrics`; `internal/settings` (saklar); `internal/printout/service.go` (riwayat teks, scan oleh petugas migrasi) |
| API | `GET /migration/numbers`, `POST /migration/documents`, `PUT /documents/{id}/history`, `POST /documents/{id}/migration/verify`, `GET /documents/{id}/link-suggestions`, `POST /documents/{id}/links`, `POST /documents/{id}/excel-comparison`, `GET /metrics` |
| Layar | `MigrationPanel.svelte`, `ExcelComparePanel.svelte`, `LinkPanel.svelte`, `/dashboard`, form di `/documents`, saklar di Settings, bab Help "Legacy documents" dan "Dashboard" |
| Test | `migrate_test.go`, `excelcompare_test.go`, `internal/metrics`, tabel izin, `m7-migration.test.ts` |

## 2. Cara mengulang dari nol

1. Tulis `TestMigrateLegacyDocument`: mode mati ditolak, tanpa E-10 ditolak, dokumen masuk di revisi 4, riwayat tersimpan, pemasuk tidak bisa memverifikasi, tanpa scan ditolak, sesudah verifikasi tidak bisa diubah lewat layanan maupun SQL.
2. Migrasi 00041: saklar, `documents.origin`, `document_baselines`, `document_history_entries`, `document_revisions.migrated`, dan trigger penolak.
3. `migrate.go` sampai test lulus.
4. `TestSimilarity` dan `TestLinkRows`, lalu `link.go`.
5. `TestCompareExcel` (berkas Excel dibuat di dalam test), lalu `excelcompare.go`.
6. `TestMetrics`, lalu query dan `internal/metrics`.
7. Kontrak API, handler, tabel izin, layar, teks dua bahasa, Help, E2E.

## 3. Konsep yang dipelajari

- **Baseline migrasi.** Dokumen lama tidak dibuat ulang dari revisi 0. Ia masuk di revisinya yang sekarang, sama dengan hardcopy; riwayat sebelumnya cukup sebagai teks.
- **Empat mata.** Orang yang mengetik bukan orang yang menyatakan "sudah sama dengan hardcopy". Aturan ini dijaga dua kali: di kode dan di CHECK database.
- **Kemiripan teks.** Koefisien Dice atas pasangan dua huruf: dua kali jumlah pasangan yang sama, dibagi jumlah pasangan kedua teks. "Parts Receiving" dan "Part Receiving" bernilai tinggi walau tidak sama persis. Sistem hanya mengusulkan; orang yang memutuskan.
- **Membandingkan sebagai kumpulan.** Bila bentuk dua sumber berbeda, bandingkan isinya sebagai kumpulan teks dulu. Hasilnya lebih kasar, tetapi tidak butuh pemetaan kolom.
- **Ukuran dihitung, bukan diketik.** Dasbor hanya menjalankan query agregat. Median dipakai di samping rata-rata, karena satu nilai ekstrem menggeser rata-rata.

## 4. Asumsi

- ASUMSI E-112 (pembanding Excel), E-113 (verifikasi hardcopy), E-114 (kemiripan), E-115 (saklar), E-116 (form dan riwayat), E-117 (definisi ukuran).

## 5. Status test

- `go test -p 1 ./...` lulus, Vitest 156 lulus, E2E 58 lulus (15 November 2026).

## 6. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `similarity(a, b string) int` dengan koefisien Dice atas pasangan dua huruf, beserta test tabelnya: dua teks sama (100), satu teks kosong (0), "Parts Receiving" dan "Part Receiving" (di atas 85), dan dua teks yang tidak berhubungan (di bawah 30).
