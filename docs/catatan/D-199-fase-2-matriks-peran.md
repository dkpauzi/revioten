# D-199 · Fase 2: matriks peran yang bisa ditambah admin

- **Tanggal:** 20 November 2026
- **Keputusan:** D-199 (matriks peran, gaya Spatie Laravel Permission), D-203 baru (Engineer dipilih saat mengajukan tanda tangan; memutuskan E-132). Brief v1.37.
- **Migrasi baru:** 00049 (`roles` dengan nama dua bahasa, aktif, bawaan, cakupan Project Group; hak S-1 dan G-1; peran baru; pemindahan akun; trigger `roles_guard`).
- **Rancangan acuan:** `docs/rencana/matriks-peran-dan-hak.md`.

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| **Peran baru** | PPAP Engineer, PPAP Engineer Supervisor, Project Leader (cakupan Project Group sendiri), Manager. Viewer dan Admin tetap |
| **Hak baru** | S-1 tanda tangan digital (dipakai fase 3), G-1 catatan genba (dipakai saat tampilan genba dibangun) |
| **Pemindahan akun** | Penyetuju form atau Verifikator semua customer → PPAP Engineer Supervisor; peran kerja lain semua customer → PPAP Engineer; peran bercakupan Project Group → Project Leader. Peran lama dinonaktifkan, tidak dihapus |
| **Admin → Roles** | Matriks centang dengan nama peran dan jumlah pemegang; **Add a role** (nama Inggris dan Indonesia, salin hak, boleh dibatasi ke Project Group); **Rename**; **Deactivate** / **Activate**; **Show inactive roles** |
| **Admin → Accounts** | Hanya peran aktif yang ditawarkan; Project Leader otomatis bercakupan Project Group sendiri |
| **API** | `POST /api/v1/roles`, `PATCH /api/v1/roles/{roleId}`, `PUT /api/v1/roles/{roleId}/active`; `Role` mendapat `name_en`, `name_id`, `active`, `builtin`, `group_scope`, `holders` |
| **Database** | Trigger `roles_guard`: peran tidak pernah dihapus, Admin dan Viewer tetap, ID dan level tetap, peran yang masih dipegang tidak bisa dinonaktifkan |
| **Viewer** | Hak peran Viewer (L-1 sampai L-3, G-1) hanya ikut ke akun kerja; akun admin tidak memegang G-1 |

Perbaikan kecil di luar matriks: tanggal rilis (`release_date`) kini dihitung menurut tanggal WIB, bukan tanggal UTC database. Sebelumnya rilis antara pukul 00.00 dan 07.00 WIB tercatat sebagai tanggal kemarin, dan jatuh tempo review 3 tahun ikut mundur sehari. Ditemukan test review yang gagal pada 20 November dini hari.

## 2. Yang berbeda dari rancangan, dan alasannya

| Di rancangan | Di kode | Alasan |
|---|---|---|
| Peran lama dinonaktifkan | Nonaktif, tetapi haknya masih dihitung bila ada akun yang memegangnya lewat SQL langsung | Test unit memakai peran lama untuk menguji satu hak terpisah. Di data nyata tidak ada pemegang, karena migrasi memindahkannya dan database menolak menonaktifkan peran yang masih dipegang (ASUMSI E-133) |
| Peran baru tanpa level | Peran tambahan admin berlevel "edit" | Level tidak dipakai pemeriksaan izin; hanya keterangan |
| — | Setiap PPAP Engineer Supervisor memegang V-3, jadi bawaannya semua supervisor harus menyetujui form set (D-148) | Mengikuti matriks. Bila cukup satu orang, admin mengubah aturan di Settings, atau mencabut V-3 dari peran ini |

## 3. Tampilan genba dan kolom Engineer

- **Tampilan genba belum dibangun.** Yang sudah ada: Released documents dengan tab **Check** (PFMEA dan CP terbit berdampingan, baca saja, D-195). Tampilan genba per proses dengan catatan saran perbaikan adalah langkah L di `docs/rencana/form-set-aiag-vda.md` (D-137) dan belum dikerjakan. Hak G-1 sudah ada supaya bisa diatur lebih dulu.
- **Kolom Engineer** dipilih saat dokumen diajukan untuk tanda tangan, dari akun berperan PPAP Engineer atau Project Leader (D-203). Dibangun bersama tanda tangan digital di fase 3.

## 4. Cara mengulang dari nol

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
python docs/uml/tools/u8_dari_database.py revioten
```

## 5. Konsep yang dipelajari

- **Peran sebagai data.** Kode hanya bertanya "apakah akun ini punya hak E-3?", tidak pernah "apakah ia Penyusun?". Karena itu peran bisa ditambah dan diganti nama tanpa mengubah kode.
- **ID tetap, nama tampilan.** ID peran dibuat sekali dari nama Inggris (`QA Auditor` → `qa_auditor`) dan tidak berubah saat nama diganti, sama seperti proses dan jabatan (D-06, D-58).
- **Menonaktifkan, bukan menghapus.** Audit log menyebut ID peran; bila peran dihapus, catatan lama tidak bisa dibaca lagi.
- **Dua lapis penjaga.** Layanan Go menolak lebih dulu dengan pesan yang jelas; trigger database menolak lagi bila ada jalan lain (psql, kode baru yang lupa memeriksa).
- **Kunci penasihat (advisory lock).** Dua admin yang menambah peran bersamaan bergantian lewat `pg_advisory_xact_lock`, tanpa hak UPDATE tambahan.

## 6. Asumsi

- **ASUMSI E-133:** peran lama nonaktif tetap memberi hak bila dipegang (hanya mungkin lewat SQL langsung).
- **ASUMSI E-132 (sampai fase 3):** engineer set = pembuat dokumen pertama set itu.

## 7. Temuan desain

- E-132 diputuskan (D-203).
- E-133 baru · Peran lama sesudah matriks D-199.

## 8. Status test

- Go: semua paket lulus, termasuk `TestRoleMatrixSeed`, `TestCreateRenameDeactivateRole`, `TestRoleSlug`, dan tabel izin per operasi (3 operasi baru).
- Vitest: 164 lulus (`roles.spec.ts` ditulis ulang).
- E2E: 65 lulus (test baru: tambah, ganti nama, dan nonaktifkan peran).
- Tur dan gambar Help: lulus; gambar diperbarui.

## 9. Latihan tulis tangan

Tanpa melihat kode, tulis fungsi `roleSlug(name)` yang mengubah `"  Leader / QC  "` menjadi `leader_qc`: huruf kecil, huruf dan angka saja, satu garis bawah untuk setiap rangkaian karakter lain, tanpa garis bawah di awal dan akhir, dan `role` bila hasilnya kosong. Bandingkan dengan `internal/auth/role_catalog.go`.
