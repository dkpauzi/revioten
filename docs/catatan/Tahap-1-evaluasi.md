# Evaluasi Tahap 1 · Fondasi

- **Tanggal:** 28 Oktober 2026
- **Dasar:** instruksi pemilik "untuk saat ini lanjutkan hingga M1 selesai. baru setelah itu kita akan evaluasi" (28 Oktober 2026), dan D-77 (kode dipakai untuk menguji desain).
- **Isi:** bahan keputusan untuk pemilik. Claude Code tidak memutuskan apa pun di sini. Rincian setiap temuan (opsi, dampak, dan asumsi) ada di `docs/evaluasi-desain.md`.
- **Status:** menunggu keputusan pemilik. M2.1 belum dimulai.

## 1. Yang sudah dibangun

| Milestone | Selesai bila (milestone) | Hasil |
|---|---|---|
| M0.1 sampai M0.5 | Alat dan versi, server Go, PostgreSQL, SvelteKit, dan irisan customer | Selesai |
| M1.1 Login lokal | Halaman tanpa login ditolak | Selesai |
| M1.2 Titik pemeriksaan izin | Semua endpoint lewat satu fungsi izin, dan test membuktikan setiap peran hanya bisa melakukan aksi yang diizinkan | Selesai. Tabel izin menguji 58 operasi (semua operasi kontrak kecuali 2 yang publik) dengan tiga pelaku |
| M1.3 Audit log tahan manipulasi | Percobaan mengubah log gagal | Selesai: role aplikasi tanpa UPDATE dan DELETE, trigger penolak, dan hash chain |
| M1.4 Master customer, model, part number | Relasi model ke customer berjalan | Selesai, termasuk Project Group dan group model |
| M1.5 Master proses | Nama proses bisa diubah tanpa memutus rujukan | Selesai |
| M1.6 Jenis dokumen dan perubahan | Semuanya bisa diedit admin | Selesai. Kebijakan approval per jenis perubahan berupa rute, dipasang di M1.8 (ASUMSI E-35) |
| M1.7 Backup data lokal | Restore berhasil dan langkahnya tertulis | Belum dibangun, karena masih "Tambah (usulan)". Lihat bagian 5 |
| M1.8 Jabatan dan rute approval | Admin bisa mengatur rute default customer tanpa mengubah kode | Selesai |

Hasilnya 8 migrasi database dan 60 operasi di kontrak `api/openapi.yaml`. Setiap milestone punya catatan di folder ini.

**Status test**

| Perintah | Hasil |
|---|---|
| `go vet ./...`, `gofmt -l .` | Bersih. Dijalankan ulang 28 Oktober 2026 untuk evaluasi ini |
| `go test ./...` | Lulus di 14 paket. Dijalankan ulang 28 Oktober 2026. Isinya 162 fungsi test dan 271 sub-test (catatan M1.8 bagian 8) |
| `npm --prefix web test`, `npm --prefix web run check`, `npm --prefix web run test:e2e` | Lulus di akhir M1.8: 65 test Vitest, svelte-check tanpa error, dan 18 test E2E. Tidak dijalankan ulang, karena kode tidak berubah sejak commit `786ed23` |

Semua perintah Windows di bagian 2 setiap catatan belum dicoba di komputer pemilik.

## 2. Apa yang ditunjukkan Tahap 1 tentang desain (D-77)

1. **Desain Tahap 1 bisa dibangun.** Semua kriteria "Selesai bila" tercapai. Temuan E-19 sampai E-41 tidak ada yang menghentikan pekerjaan; semuanya bisa dilewati dengan asumsi paling aman.
2. **Dari 41 temuan, 18 sudah diputuskan dan 23 masih terbuka.** Yang terbuka terdiri dari 17 celah, 5 ambigu, 1 sulit dibangun, dan tidak ada kontradiksi. Kelima kontradiksi (E-01, E-04, E-10, E-11, E-13) sudah diputuskan.
3. **Kebanyakan celah ada di tingkat rincian.** Brief menetapkan hubungan antar entitas, tapi belum field, batas, dan aturan tepinya: field dan batas (E-21, E-30, E-32, E-40), aturan tepi (E-31, E-37, E-39), dan jumlah pemegang (E-38). Tempat untuk menutupnya adalah diagram kelas domain U5 dan ERD U8, yang keduanya belum ada (bagian 6).
4. **Model akses D-66 bisa dibangun, tapi butuh aturan tetap di luar katalog hak:** melihat master data (E-24), mengurus akun sendiri (E-25), dan cakupan Project Group untuk operasi tanpa customer (E-41).
5. **Pertanyaan terbuka brief kini menahan tahap tertentu:** Q-15 (E-35, E-40) sebelum Tahap 5, Q-46 sebelum group model dipakai sebagai target yokoten (D-40), dan butir 4 E-37 di awal M2.1.
6. **Server hasil generate tidak memeriksa field wajib di kontrak (E-34).** Sejak M1.6, boolean baru yang nilai kosongnya berbahaya dibuat opsional dengan arti "tidak dikirim berarti tetap": `active` di sumber perubahan (M1.6) dan `sign_dates_enabled` di pengaturan (M1.8). Field `active` di customer (M1.4) masih wajib, dan dibaca `false` bila tidak dikirim.

## 3. Keputusan yang cukup dikonfirmasi

Untuk temuan di bawah, kode sudah memakai rekomendasi Claude Code. Jawab "setuju bagian 3" bila semuanya diterima, atau sebut ID yang ingin diubah.

| ID | Pertanyaan | Rekomendasi (sudah di kode) | Paling lambat |
|---|---|---|---|
| E-21 | Field dan batas customer | (a) Nama wajib, paling banyak 200 karakter, dan boleh sama; kode unik dua huruf | Sudah dipakai sejak M1.4 |
| E-23 | Daftar password terlarang tanpa layanan luar | (a) Daftar offline; (b) API Pwned Passwords dibahas dengan IT | — |
| E-24 | Hak melihat master data | (a) Termasuk L-2 | — |
| E-25 | Mengurus akun sendiri | (a) Aturan tetap di luar katalog hak | — |
| E-26 | Hapus akun | (a) Akun tidak pernah dihapus, hanya dinonaktifkan | — |
| E-27 | Aktor tanpa akun | (a) Jenis aktor `system` dan `anonymous` | — |
| E-28 | Isi audit log | (a) Setiap perubahan data, membaca tidak dicatat; (b) ekspor dan cetak ditambah di Tahap 6 | Tahap 6 |
| E-29 | Kredensial pemilik database | (a) Untuk demo; (c) ditulis untuk IT di M8.4 | M8.4 |
| E-31 | Kode customer lama | (a) Satu kode hanya pernah milik satu customer, tidak bisa kembali, dan berlaku saat dicatat | M2.1 |
| E-32 | Nama dan alias proses | (a) Satu master, satu ruang nama tanpa membedakan huruf besar kecil, proses tidak dihapus, dan nama lama tidak otomatis menjadi alias | M3.2 |
| E-36 | Sumber perubahan | (a) Satu daftar dengan mekanisme sistem yang selalu aktif; (b) mekanisme dan pemicu menjadi dua field di Tahap 5 | Tahap 5 |
| E-37 | Kode khusus | (a) Dua atau tiga huruf, bisa ditambah Admin, dan tidak boleh sama dengan kode customer; (c) satu tabel untuk semua kode dipertimbangkan di M2.1 | M2.1 |
| E-39 | Customer tanpa rute default | (a) Boleh kosong, dan tampil "Not set" di daftar customer pada layar Approval routes; cetak ditolak sampai rute diatur | Tahap 6 |
| E-40 | Bentuk rute | (a) Satu Prepared, satu Checked, lalu satu atau lebih Approved; paling banyak 10 slot; judul cetak paling banyak 100 karakter | Sampai Q-15 dijawab |
| E-41 | Hak bercakupan untuk operasi tanpa customer | (a) Ditolak; (c) dipertimbangkan di Tahap 2 | Tahap 2 |

## 4. Keputusan yang butuh pilihan atau bahan dari pemilik

1. **E-38 · Jumlah pemegang jabatan.** Di sini rekomendasi berbeda dengan kode. Kode sekarang memakai (a): hanya Project Leader dan Project Engineer yang dibatasi satu orang per Project Group. Rekomendasi Claude Code adalah (b) untuk jabatan approver (NPI Manager, QA QC Manager, Advisor, dan Factory Manager) dan APQP/PPAP Controller, yaitu satu pemegang per jabatan, karena satu slot mencetak satu nama (D-76). Jabatan lain tetap (a). Bila (b) dipilih, perlu satu migrasi kecil dan test-nya sebelum Tahap 6. **Pertanyaan:** (a) atau (b)?
2. **E-34 · Field wajib yang tidak diperiksa server.** Rekomendasi (b): `active` di `PUT /api/v1/customers/{id}` dibuat opsional dengan arti "tidak dikirim berarti tetap", seperti field yang sama sejak M1.6. Middleware pemeriksa kontrak (a) diputuskan sebelum M8.1. Pekerjaannya kecil: kontrak, handler, dan test. **Pertanyaan:** boleh dikerjakan sebelum M2.1?
3. **E-22 · Akun admin terakhir lupa password atau terkunci.** Kode belum punya jalan keluar. Rekomendasi (a) dan (c): perintah `reset-admin-password` yang hanya jalan di komputer server dan tercatat di audit log dengan aktor `system`, ditambah akun admin tidak dinonaktifkan pada kegagalan ke-100 (kunci 15 menit tetap berlaku). **Pertanyaan:** setuju, dan kapan dibangun? Usulan: sebelum data asli dipakai sehari-hari di laptop.
4. **E-20 · Tabel kode error.** **Pertanyaan:** apakah spec lama `pfmea-system-phase1-spec-v2` masih ada? Bila ada, (a): tabelnya disalin ke brief dan kontrak disesuaikan. Bila tidak ada, (b): tabel dari M0.5 menjadi baseline.
5. **E-19 · Istilah layar bahasa Inggris.** Rekomendasi (a): kolom "Istilah layar (Inggris)" di U1, lalu `web/src/lib/i18n/en.ts` mengikutinya. Ini mengubah UML, jadi digabung dengan bagian 6. **Pertanyaan:** (a) atau (b)?
6. **E-35 dan Q-15 · Kebijakan approval per jenis perubahan.** Bentuk (a) sudah dibangun di M1.8: setiap jenis perubahan boleh menunjuk satu rute, dan kosong berarti rute default customer. Q-15 tetap memakai asumsi kerja "alur yang sama untuk semua". **Pertanyaan (sebelum Tahap 5):** bila penyusun memilih rute lain untuk satu dokumen (R-APR-09), mana yang menang: rute dari jenis perubahan, atau pilihan penyusun?
7. **E-30 · Field Project Group, group model, model, dan part number.** Rekomendasi (b): field ditetapkan dari header PFD, PFMEA, dan CP yang dipakai sekarang (F-10, F-11), sebelum M4.1. Bahannya form asli. Bila Anda memberikannya lagi, saya bisa menyusun daftar usulan field tanpa nama orang dan tanpa menyimpan form-nya di repo.
8. **E-33 · Pencocokan di tingkat proses atau sub-proses.** Rekomendasi (c): bandingkan satu set PFD, PFMEA, dan CP asli baris per baris sebelum M3.2. Bahannya juga dokumen asli, dengan cara yang sama seperti butir 7.
9. **E-37 butir 4 · Nomor dengan kode khusus.** **Pertanyaan (awal M2.1):** apakah PFD, PFMEA, dan CP boleh dinomori dengan kode khusus, misalnya PFD umum dengan ZZ? R-NUM-01 hanya menyebut nomor per customer.
10. **Q-46 · Group model.** E-10 memutuskan group model dikelola Admin (A-2) untuk M1.4, sedangkan asumsi kerja Q-46 di brief masih "Template authors maintain groups". Definisi group model (misalnya platform PCB yang sama) juga belum ada. **Pertanyaan (sebelum group model dipakai sebagai target yokoten, D-40):** tetap Admin, atau dipindah ke Pengelola template (E-6) seperti master proses (D-64)?
11. **Q-47 · Generator TypeScript.** Hasil uji di M0.5: @hey-api/openapi-ts 0.99.0 (plugin tipe saja) terpasang tanpa konflik, sedangkan openapi-typescript 7.13.0 gagal karena peer dependency TypeScript 5. Sejak M0.5 yang dipakai adalah @hey-api/openapi-ts. **Pertanyaan:** dikonfirmasi?

### Keputusan yang belum tercatat di brief

E-01 sampai E-18 sudah Anda putuskan, tapi belum punya nomor D di brief (status "Diputuskan, D menunggu brief"). D terakhir di brief adalah D-77. Brief bagian 15 butir 11 meminta keputusan dicatat di brief sebagai item D, dan aturan kerja melarang Claude Code mengubah brief tanpa permintaan pemilik. **Pertanyaan:** apakah saya boleh menyusun item D-78 dan seterusnya di brief, beserta keputusan dari bagian 3 dan 4? Bila ya, versi brief naik, dan baris "dari brief desain v1.15" di `CLAUDE.md` ikut diperbarui.

### Seberapa luas kode tersentuh oleh setiap keputusan

Jumlah penanda `// ASUMSI E-nn` di kode untuk temuan terbuka (di luar `docs/`). Bila keputusan berbeda dengan asumsi, semua tempat ini diperiksa ulang.

| Penanda | ID |
|---|---|
| 15 | E-24 |
| 12 | E-30 |
| 11 | E-35 |
| 9 | E-32, E-37 |
| 5 | E-21, E-36 |
| 3 | E-20, E-26, E-31, E-39 |
| 2 | E-19, E-27, E-40, E-41 |
| 1 | E-25, E-28, E-29, E-33, E-34, E-38 |
| 0 | E-22 (belum dibangun), E-23 (dicatat di `internal/auth/common_passwords.txt`) |

Cara mencari: `git grep -n "ASUMSI E-24"` (ganti nomornya).

## 5. M1.7 Backup data lokal (usulan)

Isi usulan di milestone: backup harian terenkripsi ke disk lain, enkripsi disk laptop, dan satu uji restore ke database kosong, "segera setelah login dan audit log jadi", karena data asli dipakai di localhost sejak awal (D-29, D-36). Login (M1.1) dan audit log (M1.3) sudah jadi. Usulan M1.3 juga menaruh hash rantai audit terakhir di manifest backup harian, dan E-29 (c) memakainya.

- **(a)** Bangun sekarang, sebelum M2.1.
- **(b)** Bangun sebelum data asli pertama masuk database di laptop, paling lambat sebelum M2.5 (impor nomor lama).
- **(c)** Tunda ke M8.2 seperti rencana semula.

**Rekomendasi Claude Code (usulan):** (a). Syarat di usulannya sudah terpenuhi, dan register nomor di Tahap 2 adalah data yang paling mahal bila hilang, karena nomor tidak pernah dipakai ulang. Enkripsi disk laptop adalah langkah Anda sendiri dan mungkin butuh izin IT. Kodenya hanya menyiapkan backup, enkripsi file backup, dan uji restore. **Pertanyaan:** (a), (b), atau (c)?

## 6. UML yang tertinggal dari kode

Keputusan E-03: Tahap 1 tidak menunggu U5 dan U6, dan UML diperbarui mengikuti kode. Paket UML masih draft 0.7 (27 Oktober 2026), yang dibuat sebelum kode Tahap 1.

| Bagian UML | Keadaan sekarang | Yang perlu diperbarui |
|---|---|---|
| U1 glosarium | Belum ada entri sendiri untuk model, part number, kode pabrik, kode khusus, dan pemegang jabatan, padahal semuanya tampil di layar Tahap 1 | Tambah entri, dan kolom istilah Inggris bila E-19 (a) dipilih |
| U3 peta pengguna | Belum memuat E-24, E-25, dan E-41 | Tambah aturan tetap di bagian 4, setelah keputusannya ada |
| U5 diagram kelas domain | Belum ada | Buat untuk entitas Tahap 1, dari 8 migrasi dan paket `internal/` |
| U8 ERD fisik | Belum ada ("per milestone") | Buat dari 8 migrasi |
| U10 file OpenAPI | README UML masih menulis "Belum", padahal `api/openapi.yaml` sudah berisi 60 operasi | Rujuk file kontrak di README |

**Rekomendasi Claude Code (usulan):** kerjakan sebelum M2.1 dalam satu sesi tersendiri. Tim IT membangun dari UML (D-33), dan rincian Tahap 1 paling mudah ditulis selagi masih baru. **Pertanyaan:** boleh, dan apakah sebelum atau sesudah M2.1?

## 7. Gerbang dan latihan tulis tangan

**Gerbang 0** (Tahap 0): "Tulis ulang versi kecil M0.5 untuk entitas lain tanpa AI." **Pertanyaan:** sudah Anda kerjakan?

**Gerbang 1** (Tahap 1): "Tulis sendiri satu master data baru beserta test dan audit-nya." Supaya gerbang ini menguji seluruh pola Tahap 1, isinya paling sedikit (usulan):
1. migrasi goose dengan `GRANT` untuk `revioten_app`, dengan hak sekecil mungkin (E-06);
2. query sqlc;
3. layanan yang memanggil `Authorize` dan menulis audit log dalam satu transaksi;
4. operasi di kontrak beserta handler-nya, dan barisnya di `internal/server/permissions_test.go`;
5. test yang ditulis dan dilihat gagal lebih dulu.

Pilihan entitas (usulan):
- **(a) Master dokumen terkait** (UC-09, R-DOC-11, usulan untuk M4.1): nomor dan judul WI, SOP, dan QEP. Kebutuhannya nyata dan entitasnya kecil. Dikerjakan di luar `main` dulu. Masuk `main` atau tidak, Anda yang memutuskan, karena menyentuh M4.1.
- **(b) Kategori 4M ditulis ulang dari nol** di folder latihan di luar repo, lalu dibandingkan dengan kode di repo.

**Latihan per milestone** (rincian dan kriteria lulus di bagian latihan setiap catatan):

| Catatan | Latihan |
|---|---|
| M0.1 | `compose.yaml` dan `.env` untuk PostgreSQL 18.6, ditulis dari nol |
| M0.2 | Endpoint `/halo` dengan table-driven test, di modul baru |
| M0.3 | JSONB, indeks, dan transaksi di skema `latihan` lewat psql |
| M0.4 | Katalog bahasa Indonesia `id.ts` dan hitungan dengan `$state` dan `$derived` |
| M0.5 | `GET /api/v1/customers/{code}` dengan alur yang sama |
| M1.1 | `DELETE /api/v1/session/all`: logout dari semua perangkat |
| M1.2 | `GET /api/v1/accounts/{accountId}` |
| M1.3 | Perintah `revioten verify-audit` |
| M1.4 | Customer nonaktif juga membatasi perubahan model |
| M1.5 | Cari proses menurut nama atau alias |
| M1.6 | Jenis dokumen tambahan bisa dinonaktifkan |
| M1.8 | Satu orang hanya satu slot dalam satu rute (R-APR-12) |

## 8. Langkah berikutnya (usulan, menunggu persetujuan)

1. Anda menjawab bagian 3, 4, 5, dan 6. Jawaban boleh singkat, misalnya "setuju bagian 3; E-38 (b); E-34 ya; M1.7 (a)".
2. Claude Code mencatat keputusan di `docs/evaluasi-desain.md`, memperbarui penanda `// ASUMSI E-nn`, dan, bila diizinkan, menulis item D di brief.
3. Perbaikan kecil dari keputusan: E-34, E-38 bila (b) dipilih, dan E-22 bila disetujui.
4. M1.7, bila (a) dipilih.
5. Pembaruan UML Tahap 1, bila disetujui.
6. Bagian Anda: Gerbang 0 dan Gerbang 1, serta mengulang langkah catatan di Windows.
7. M2.1, setelah E-31 dan E-37 (termasuk butir 4) diputuskan.

Mulai sekarang aturan "satu milestone per sesi, dengan rencana singkat dan persetujuan sebelum menulis kode" (CLAUDE.md bagian 3 butir 4) berlaku lagi, kecuali Anda meminta lain.
