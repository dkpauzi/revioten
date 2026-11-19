# D-205 · Fase 4: dasbor admin (kesehatan web), dan D-206 Project Leader pembantu

- **Tanggal:** 20 November 2026
- **Keputusan:** D-205 (dasbor admin, menurut `docs/rencana/dasbor-admin.md`), D-206 (batas Project Group untuk Project Leader bisa dilepas per akun, termasuk tanda tangan dan verifikasi). Brief v1.38.
- **Migrasi baru:** 00051 (`error_reports`, `problem_reports`).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| **Home akun admin = Web health** | Kartu: tanda tangan tertahan, error server (24 jam dan 7 hari), laporan masalah terbuka, email gagal dan menunggu, akun terkunci, nonaktif, dan gagal login 24 jam, backup terakhir (dari manifest di `BACKUP_DIR`), rantai audit log. Lalu lima tanda tangan paling lama tertahan, error terakhir, laporan masalah terakhir, dan dokumen yang diproses minggu laporan ini dan bulan ini per pemicu |
| **Admin → Web health** | Tab tanda tangan tertahan (semua, paling lama dulu), error server (200 terakhir, dengan request ID, alamat, akun, pesan), dan laporan masalah (status New, Being checked, Done, dengan catatan admin) |
| **Report a problem** | Tombol di bilah atas untuk semua akun: uraian, alamat halaman, dan request ID error terakhir ikut terkirim |
| **Pencatat error** | Setiap error 500 dicatat di `error_reports` dengan pesan yang sama dengan log server; permintaan yang dibatalkan browser tidak dicatat |
| **Project Leader pembantu (D-206)** | Peran Project Leader dengan cakupan **All customers**: orang itu muncul sebagai calon slot Prepared dan Checked untuk Project Group lain (label "Project Leader GR5"), dan boleh memverifikasi perubahan Project Group lain. Cakupan **Own project group** tetap bawaan |

## 2. Yang berbeda dari usulan, dan alasannya

| Di usulan | Di kode | Alasan |
|---|---|---|
| Email ke admin saat ada laporan masalah | Tidak ada email; laporan tampil di dasbor | Admin membuka dasbor sebagai Home; email bisa ditambah lewat `notification_events` bila perlu |
| Jumlah dokumen tanpa nama orang | Sama | Laporan per orang tetap di PPAP performance (L-5) |
| Rantai audit log "hasil cek terakhir" | Rantai diperiksa ulang setiap dasbor dibuka | Data masih kecil; bila lambat nanti, hasilnya bisa disimpan |

## 3. Cara mengulang dari nol

```bash
docker compose up -d
go generate ./internal/db/... ./internal/server/...
npm --prefix web run gen:api
go test -p 1 ./...
npm --prefix web test
npm --prefix web run test:e2e
npm --prefix web run help:shots
```

## 4. Konsep yang dipelajari

- **Mencatat error tanpa membocorkan rahasia.** Pengguna hanya melihat pesan umum dan request ID; detail teknis ke log dan ke tabel admin, tanpa isi permintaan.
- **`context.WithoutCancel`.** Error dicatat dengan context yang tidak ikut batal walau permintaannya selesai.
- **`db.AsSystem` dengan alasan.** Admin tidak memegang L-4, jadi draft disaring RLS; dasbor hanya membaca nomor, slot, dan nama, jadi boleh lintas customer.
- **Cakupan sebagai satu saklar.** D-206 tidak butuh tabel baru: cakupan peran yang sudah ada (Own project group / All customers) kini juga menentukan calon tanda tangan dan verifikator.

## 5. Asumsi

- Dasbor butuh hak A-3 (kelola pengaturan sistem).
- Lama tertahan dihitung sejak tanda tangan sebelumnya, atau sejak dikirim bila slot pertama yang ditunggu.
- Project Leader dengan cakupan semua customer mendapat email verifikasi untuk semua customer.

## 6. Temuan desain

- Tidak ada temuan E- baru.

## 7. Status test

- Go: semua paket lulus, termasuk paket baru `internal/health`, test Project Leader pembantu (`TestHelperProjectLeader`), test error 500 tercatat, dan tabel izin (6 operasi baru).
- Vitest: 165 lulus. E2E: 67 lulus, termasuk `m9-admin-health.test.ts` baru. Gambar Help: `admin-health` baru, `home` kini dari akun kerja.

## 8. Latihan tulis tangan

Tanpa melihat kode, tulis fungsi `waited(hours)` yang menulis "5 h" di bawah 24 jam dan "2 d 3 h" sesudahnya. Bandingkan dengan `web/src/lib/admin/health.ts`.
