# Tahap 8 · Penguatan dan serah terima (M8.1 sampai M8.5)

- **Tanggal:** 15 November 2026
- **Aturan yang dipakai:** R-SEC-02, R-SEC-04, R-SEC-10, R-SEC-11, R-SEC-12, R-API-13, D-19, D-33, D-36, D-66, D-71, D-81, D-108, D-172, RK-14, RK-50, RK-58, RK-63.
- **Persetujuan pemilik:** "Lanjutkan ke M8 kemudian jangan lupa atur ulang tur otomatisnya dan halaman help nya" (15 November 2026).
- **Belum selesai menurut kriteria milestone:** M8.2 "restore berhasil di mesin lain", M8.4 "IT bisa menilai tanpa melihat kode", dan M8.5 "semua angka berasal dari pengukuran nyata". Ketiganya butuh pemilik atau pihak lain (E-121).

## 1. Apa yang dibuat

| Milestone | Hasil |
|---|---|
| M8.1 Peran dan hak akses | Admin → **Roles**: matriks peran dan hak (D-66). Row-level security pada `documents`, `document_rows`, dan `document_revisions` (migrasi 00042). Test lintas customer: sepuluh percobaan Project Leader terhadap customer Project Group lain ditolak fungsi izin, dan query langsung ke tabel tidak melihat barisnya. |
| M8.2 Unggahan aman dan backup | Klien ClamAV (`internal/storage/scan.go`): lampiran dan scan dipindai sebelum disimpan bila `CLAMAV_ADDR` diisi; pemindai mati berarti unggahan ditolak. Backup terenkripsi dan uji restore sudah ada sejak M1.7 dan D-163; langkah uji di mesin lain ada di panduan operasional. |
| M8.3 Deploy | `Dockerfile`, `compose.deploy.yaml`, dan `deploy/Caddyfile`: PostgreSQL, aplikasi, dan Caddy dengan HTTPS lokal di https://localhost:8443. Terbukti jalan pada 15 November 2026 (`/health` menjawab 200 lewat Caddy; image aplikasi 166 MB). |
| M8.4 Dokumen untuk IT | `docs/presentasi/dokumen-untuk-it.md` versi 1.0 dan `docs/panduan/panduan-operasional.md`. |
| M8.5 Presentasi QCC | `docs/presentasi/kerangka-qcc.md`: alur cerita dan tabel angka dengan kotak [ukur] yang belum diisi. |
| Tur otomatis | `web/tour/alur.tour.ts` sekarang 32 langkah: sampai cetak, rilis, perubahan, verifikasi, revisi 1, dokumen lama, dasbor, dan Roles. |
| Halaman Help | Bab baru "Roles and rights"; gambar baru untuk Documents, Document control, Changes, Print, Legacy documents, Dashboard, dan Roles. |

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00042_access_hardening.sql` |
| Subjek RLS | `internal/db/subject.go`, `internal/db/db.go` (PrepareConn), `internal/server/session.go` |
| Isi peran | `internal/auth/roles.go`, `internal/server/roles.go`, `web/src/routes/admin/roles/+page.svelte` |
| Pemindai | `internal/storage/scan.go`, `storage.go` (PutScanned) |
| Deploy | `Dockerfile`, `.dockerignore`, `compose.deploy.yaml`, `deploy/Caddyfile` |
| Test | `internal/auth/roles_test.go`, `internal/document/rls_test.go`, `internal/storage/scan_test.go`, `internal/printout` (scan berbahaya), tabel izin, `web/e2e/m8-roles.test.ts` |

## 2. Cara mengulang dari nol

1. Tulis `TestSetRolePermissions`, lalu trigger penjaga di migrasi 00042 dan `auth/roles.go`.
2. Tulis `TestCrossCustomerAccess`: dua Project Group, satu Project Leader, sepuluh percobaan lewat layanan, lalu query langsung dengan `db.WithSubject`.
3. Fungsi `app_sees_document` dan tiga kebijakan RLS di migrasi 00042; `subject.go` dan `PrepareConn`; sesi menandai subjek.
4. Jalankan tabel izin: akun tanpa hak sekarang mendapat 404 untuk dokumen draft.
5. `TestScanOnPut` dengan clamd tiruan (listener TCP di dalam test), lalu `scan.go`.
6. `Dockerfile` tiga tahap, `compose.deploy.yaml`, `Caddyfile`; jalankan dan panggil `/health` lewat HTTPS.
7. Layar Roles, teks dua bahasa, Help, tur, E2E.

## 3. Konsep yang dipelajari

- **Pertahanan berlapis.** Fungsi izin di aplikasi bisa terlupa di satu query. Row-level security membuat database sendiri menolak, jadi satu kelalaian tidak langsung menjadi kebocoran.
- **Memberi tahu database siapa yang dilayani.** Dengan connection pool, satu koneksi dipakai bergantian. ID akun ditulis ke variabel sesi setiap kali koneksi dipinjam, dan kebijakan RLS membacanya.
- **404 lawan 403.** Menjawab "tidak ada" untuk hal di luar cakupan tidak membocorkan bahwa hal itu ada.
- **Gagal tertutup.** Bila pemindai antivirus mati, unggahan ditolak. Lawannya, gagal terbuka, membiarkan berkas masuk tanpa dipindai.
- **Build bertahap.** Image akhir hanya berisi binary; Node dan kompilator Go tinggal di tahap build.
- **Proxy di depan.** Caddy mengurus HTTPS; aplikasi cukup bicara HTTP di jaringan dalam container.

## 4. Asumsi

- ASUMSI E-118 (cakupan RLS), E-119 (aturan isi peran), E-120 (pemindai opsional, PDF tidak ada di image), E-121 (kriteria yang butuh pemilik).

## 5. Status test

- 15 November 2026: `go test -p 1 ./...` lulus, Vitest 156 lulus, E2E 59 lulus, tur otomatis (cepat, tanpa jendela) lulus, dan skrip gambar Help berhasil.
- Belum dijalankan: container ClamAV sungguhan, login di deploy Docker, dan restore di mesin lain.

## 6. Latihan tulis tangan

Tanpa AI, di psql sebagai pemilik database: buat tabel `catatan (id serial, pemilik text, isi text)`, nyalakan row-level security, dan tulis satu kebijakan yang hanya menampilkan baris dengan `pemilik = current_setting('app.pemilik', true)`. Buat role kedua, beri `SELECT`, lalu buktikan dengan `SET app.pemilik = 'a'` bahwa role itu hanya melihat baris milik `a`.
