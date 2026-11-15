# D-195 sampai D-197 · Dokumen terbit untuk Viewer, review berkala 3 tahun, dan laporan kinerja PPAP Engineer

- **Tanggal:** 15 November 2026
- **Keputusan:** D-195 (jawaban E-128 opsi a), D-196 (review berkala, mewujudkan D-138), D-197 (laporan kinerja). Brief v1.35.
- **Migrasi baru:** 00047 (`periodic_reviews`, `periodic_review_notices`, pengaturan siklus review, sumber perubahan "Periodic review", hak L-5, dua peristiwa email).

## 1. Tujuan dan apa yang dibuat

### D-195 · Dokumen terbit untuk akun Viewer (UC-57)
- Halaman baru **Documents → Released documents** (`/documents/released`): daftar dokumen satu customer yang punya revisi berlaku, dan halaman bacanya (`/documents/released/view?id=`).
- Isi dibaca dari **snapshot** revisi yang berlaku, bukan dari baris dokumen. Jadi dokumen yang sedang punya draft revisi 1 tetap menampilkan revisi 0 apa adanya.
- Tab **Check** bisa dibuka dengan `?released=1`: PFMEA dan CP terbit berdampingan, tanda dicek tampil, tetapi tidak bisa diubah.
- API: `GET /api/v1/released-documents`, `GET /api/v1/released-documents/{id}`, `GET /api/v1/released-documents/{id}/pfmea-cp` (L-1).

### D-196 · Review berkala 3 tahun (D-138, UC-58)
Jawaban untuk pertanyaan "dimanakah hal itu muncul?":

| Tempat | Isi |
|---|---|
| **Home** | Kartu "Periodic review": jumlah set yang lewat jatuh tempo (merah) dan yang akan jatuh tempo (kuning) |
| **Documents → Dashboard** | Kartu yang sama, mengikuti pilihan customer |
| **Halaman dokumen** (draft maupun terbit) | Spanduk "Periodic review of this document set is due on …" bila set dokumen itu jatuh tempo |
| **Documents → Periodic review** | Daftar lengkap, tombol **Review with changes** dan **Reviewed, no change**, catatan yang menunggu konfirmasi, dan riwayat |
| **Email** | Ke penyusun (E-3) customer itu, sekali saat masuk masa kuning dan sekali saat lewat jatuh tempo. Catatan review baru juga memberi email ke verifikator |

Aturannya:
- Satu **set** = PFD, PFMEA, dan CP satu customer dengan nomor berjalan yang sama (kode customer di nomor + nomor berjalan). Nomor Void dan Obsolete tidak ikut.
- Jam mulai dari tanggal rilis terakhir di set (tanggal di dokumen bertanda tangan, atau tanggal berlaku hardcopy untuk dokumen lama), atau dari review tanpa perubahan terakhir yang sudah dikonfirmasi.
- Jatuh tempo = acuan + 36 bulan; kuning mulai 90 hari sebelumnya. Kedua angka disimpan di `system_settings` (belum bisa diubah di layar).
- **Review dengan perubahan**: membuka halaman Changes dengan customer, pemicu "Periodic review", dan dokumen set sudah terisi. Alurnya perubahan biasa.
- **Review tanpa perubahan**: tanggal, peserta, delapan butir daftar periksa (OK atau Tidak berlaku), kesimpulan. Revisi yang direview dibekukan beserta SHA-256-nya. Catatan baru memulai jam setelah orang kedua (V-1) menekan **Confirm**; pencatat tidak bisa mengonfirmasi sendiri (dijaga kode dan constraint database), dan catatan terkonfirmasi tidak bisa diubah (trigger, SQLSTATE 55000).

### D-197 · Laporan kinerja PPAP Engineer (UC-65)
- Halaman **Documents → Performance report** (`/performance`), hak baru **L-5**. Bawaannya untuk peran Penyusun (PPAP Engineer) dan Penyetuju form (supervisor). Admin bisa memberikan ke peran lain di Admin → Roles.
- Periode: **minggu Jumat sampai Kamis**, bulan, atau tahun, dalam WIB. Tombol Previous dan Next berpindah satu periode.
- Angka: permintaan perubahan diajukan, diterima, ditolak; diajukan per pemicu (Customer claim, Audit finding, ECR, Periodic review, dan lainnya); yokoten dibuat, model target, dan perubahan dari yokoten; untuk revisi yang terbit lewat perubahan: waktu diterima sampai rilis, mulai edit sampai siap cetak, dan cetak sampai ditandatangani. Semua total dan per orang, ditambah daftar revisi yang terbit.

## 2. Cara mengulang dari nol

```bash
docker compose up -d
go generate ./internal/db/... ./internal/server/...
npm --prefix web run gen:api
go test -p 1 ./...
npm --prefix web test
npm --prefix web run check
npm --prefix web run test:e2e
npm --prefix web run help:shots
TOUR_SPEED=fast TOUR_HEADLESS=1 npm --prefix web run tour
```

Mencoba review berkala tanpa menunggu 3 tahun: test `internal/review/review_test.go` memakai `WithNow` untuk memajukan jam. Di layar, centang **Show sets that are not due** untuk melihat set yang belum jatuh tempo.

## 3. Konsep yang dipelajari

- **Snapshot sebagai sumber baca.** Revisi terbit tidak dibaca ulang dari tabel baris, karena tabel itu bisa sedang diedit. Snapshot JSON beserta SHA-256 adalah isi yang ditandatangani.
- **Nilai turunan tidak disimpan.** Status jatuh tempo dihitung setiap kali dari tanggal rilis dan tanggal review. Yang disimpan hanya kejadian: catatan review dan email yang sudah dikirim.
- **Empat mata di database.** `CHECK (confirmed_by_person_id <> recorded_by_person_id)` memastikan aturan orang kedua tidak bisa dilewati, juga oleh bug di kode.
- **Minggu laporan.** Awal minggu = tanggal dikurangi `(hari - Jumat + 7) % 7` hari. Zona waktu tetap WIB, supaya pukul 23.30 UTC hari Kamis sudah dihitung Jumat.
- **Median** lebih tahan terhadap satu revisi yang sangat lama daripada rata-rata; laporan menampilkan keduanya.

## 4. Keputusan beserta alasannya

- D-195 sesuai jawaban pemilik "E-128 OK".
- D-196 dibangun dengan opsi (a) E-86 sebagai asumsi kerja, karena pemilik sudah meminta kedua hasil review dan bertanya di mana peringatan muncul. E-86 tetap Terbuka untuk dipastikan.
- D-197: hak baru L-5, bukan L-2, karena angka per orang tidak untuk semua akun.

## 5. Asumsi

- **ASUMSI E-86:** opsi (a), jam per set, konfirmasi orang kedua, peringatan 90 hari, daftar periksa tetap di kode.
- **ASUMSI E-129:** tanda "perlu dicek ulang" pada PFMEA dan CP terbit hanya dihitung selama tidak ada draft terbuka.
- **ASUMSI E-130:** siapa yang dihitung dan titik waktu setiap tahap (lihat temuan).
- **ASUMSI E-131:** zona waktu WIB tetap di kode (`internal/clock`).

## 6. Temuan desain baru

- E-129 · Tanda "perlu dicek ulang" pada PFMEA dan CP terbit (Celah).
- E-130 · Laporan kinerja: siapa yang dihitung dan titik waktunya (Ambigu).
- E-131 · Zona waktu pabrik (Celah).

## 7. Status test

- Go: semua paket lulus, termasuk test baru `TestReleasedView`, `TestPeriodicReview`, `TestPeriodOf`, `TestPerformance`, dan tabel izin per operasi (8 operasi baru).
- Vitest: 157 lulus.
- E2E: 64 lulus, termasuk test baru `web/e2e/released.test.ts`. Potret Help diperbarui (tiga gambar baru per bahasa).
- Tur: 37 langkah (langkah 31 sampai 33 baru).

## 8. Yang belum

- Pengaturan 36 bulan dan 90 hari di layar Settings.
- Lampiran bukti dan cetak lembar "Periodic review record" (opsi c E-86).
- Ekspor laporan kinerja ke Excel untuk laporan hari Jumat.
- Cetak atau ekspor Excel salinan referensi dari halaman dokumen terbit (L-3) untuk akun Viewer.

## 9. Latihan tulis tangan

Tanpa melihat kode, tulis fungsi Go `weekStart(d time.Time) time.Time` yang memberi hari Jumat awal minggu laporan untuk tanggal d (minggu Jumat sampai Kamis). Uji dengan tanggal Kamis 19 November 2026 (harus 13 November) dan Jumat 20 November 2026 (harus 20 November). Lalu bandingkan dengan `PeriodOf` di `internal/metrics/performance.go`.
