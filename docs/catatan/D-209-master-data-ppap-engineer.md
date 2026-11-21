# D-209 · Master data milik PPAP Engineer

- **Tanggal:** 21 November 2026
- **Keputusan:** D-209 (brief v1.40). Temuan desain: E-134.
- **Migrasi:** 00053_master_data_to_ppap.sql

## Tujuan

Saat uji manual, pemilik merasa peran utama admin adalah memantau kesehatan dan performa web serta mengelola akun. Data seperti master data, nomor dokumen, dan model seharusnya menjadi pekerjaan PPAP Engineer dan PPAP Staff.

Keputusan pemilik:
- semua data kerja pindah ("Semua data kerja");
- mode migrasi dan aturan penyetujuan form "untuk saat ini tetap di admin".

## Yang dibuat

| Bagian | Isi |
|---|---|
| Hak | Hak kerja baru **E-11** "Kelola master data dan pengaturan dokumen" menggantikan hak admin **A-2**. E-11 diberikan ke PPAP Engineer dan PPAP Engineer Supervisor. A-2 dihapus dari katalog. |
| Migrasi 00053 | Menyisipkan E-11 sesudah E-10, melepas A-2 dari peran Admin (trigger penjaga dimatikan sebentar dengan sengaja), lalu menghapus A-2 dan memberi E-11 ke kedua peran PPAP. |
| Go | `access.ManageMasterData` sekarang bernilai `"E-11"`. Semua operasi yang memakainya langsung ikut. Jabatan dan pemegangnya (dulu A-1) juga pindah ke E-11. |
| Daftar orang | Endpoint baru `GET /api/v1/position-candidates` (E-11): orang dengan akun kerja aktif, hanya ID dan nama. Halaman Positions memakainya, karena daftar akun tetap hak admin (A-1). |
| Settings dibagi dua | `settings.Apply` memeriksa hak per kelompok isian (`authorizeUpdate`). Pengaturan dokumen (tanggal di slot, QR, nomor form submission) butuh E-11; pengaman (mode migrasi, aturan penyetujuan form) butuh A-3. Permintaan yang mencampur keduanya butuh kedua hak, jadi ditolak untuk siapa pun. |
| Layar | Halaman baru **Master data → Document settings** (`/document-settings`). **Admin → Settings** sekarang hanya berisi dua pengaman dan petunjuk ke halaman baru. Semua tombol ubah master data memakai `can('E-11')`. |
| Teks | Kunci baru di `en.ts` dan `id.ts`: `permE11`, `navDocumentSettings`, `docSettingsTitle`, `docSettingsIntro`, `settingsMovedHint`. `permA2` dihapus. |
| Help | Pelaku bab customer, jabatan, format template, form set, cetak, tanda tangan, dan submission diganti PPAP Engineer. Bab Accounts pindah ke depan (urutan kerja: akun dulu). Bab baru **Document settings** dengan gambarnya. |
| Tur dan skenario | Langkah 1 sampai 3 oleh admin (login, akun, pengaman, logout). Langkah 4 sampai 8 oleh PPAP Engineer (login pertama, customer, model, jabatan dan rute serta Document settings, format template). Langkah 9 dan seterusnya tidak bergeser. |
| UML | U3 peta pengguna (matriks: E-11 menggantikan A-2), U3 daftar use case dan diagram (UC-03, 05, 06, 20, 24 milik PPAP Engineer), README draft 0.23. |

## Cara mengulang dari nol

1. Jalankan server. Migrasi 00053 berjalan sendiri.
2. Login dengan akun admin, buka **Admin → Roles**. Kolom PPAP Engineer dan PPAP Engineer Supervisor sekarang punya centang E-11, dan Admin tidak punya A-2.
3. Login dengan akun PPAP Engineer, lalu buka **Master data → Customers**. Formulir tambah customer tampil.
4. Login lagi dengan akun admin dan buka halaman yang sama. Halamannya hanya bisa dibaca.
5. Coba langgar aturan lewat psql (harus gagal, karena peran Admin dikunci):
   ```powershell
   docker compose exec db psql -U revioten -d revioten -c "INSERT INTO role_permissions (role_id, permission_code) VALUES ('admin', 'E-11')"
   ```

## Konsep yang dipelajari

- **Satu titik izin membuat perpindahan hak murah.** Sekitar 40 operasi Go memanggil `Authorize(..., access.ManageMasterData, ...)`. Yang diganti hanya nilai konstantanya, ditambah satu migrasi data.
- **Hak dicek per isian, bukan per halaman.** Satu endpoint (`PUT /system-settings`) melayani dua pemilik hak. Server memeriksa hak menurut isian yang dikirim, jadi layar bisa dipecah tanpa endpoint baru.
- **Pengaman dipegang pihak di luar yang diawasi.** Mode migrasi membuka jalan pintas ke status terbit. Aturan penyetujuan form mengatur pengawasan terhadap penyetuju form. Keduanya tetap di admin.
- **Data minimal untuk tugas.** PPAP Engineer butuh nama orang untuk mencatat pemegang jabatan, bukan seluruh daftar akun. Endpoint kandidat hanya memberi ID dan nama.

## Keputusan dan alasan

- **E-11, bukan A-2 untuk akun kerja (E-134, opsi a).** D-72 tetap utuh: hak A- hanya milik akun admin, dan database menolak sebaliknya.
- **E-11 juga untuk Supervisor.** Supervisor kini bisa menyusun sekaligus menyetujui form set. Pemisahan tugas tetap dijaga aturan "yang terakhir mengubah tidak menerbitkan" (D-139), dibuktikan `TestSeparationOfDuties`.
- **Format template tanpa AP** tetap boleh diterbitkan penyusunnya sendiri (D-143), sekarang oleh PPAP Engineer.

## Asumsi

- Calon pemegang jabatan adalah orang dengan akun kerja aktif. Orang yang hanya punya akun admin tidak ikut dalam daftar.
- Peran buatan admin (lewat **Add a role**) tidak otomatis mendapat E-11. Admin mencentangnya sendiri di Roles bila perlu.

## Temuan desain baru

- **E-134** · Hak master data pindah dari admin ke akun kerja (kontradiksi dengan D-72, diputuskan opsi a).

## Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | 28 paket lulus. Test baru: `TestCandidates`, `TestSettingsSplit`; `TestPermissions` (approval) juga menolak admin |
| `npm --prefix web test` (Vitest) | 165 lulus |
| `npm --prefix web run check` | 0 error |
| `npm --prefix web run test:e2e` | 68 lulus. Akun bersama `ppap.e2e` lewat `logInAsMaster` dan `asMaster` di `web/e2e/helpers.ts` |
| `npm --prefix web run help:shots` | Lulus, termasuk gambar baru `document-settings` |
| Tur (`TOUR_SPEED=fast TOUR_HEADLESS=1`) | Lulus |

## Latihan tulis tangan

Tanpa melihat kode, tulis di kertas:

1. Isi migrasi 00053 bagian Up: urutan perintahnya, dan mengapa trigger `role_permissions_guard` harus dimatikan sebentar.
2. Fungsi `authorizeUpdate` di `internal/settings`. Hasilnya untuk tiga kasus: admin mengirim QR, PPAP Engineer mengirim mode migrasi, dan PPAP Engineer mengirim isian kosong.
