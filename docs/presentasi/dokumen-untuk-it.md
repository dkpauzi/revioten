# Dokumen untuk tim IT · Sistem PFD, PFMEA, dan Control Plan

- **Versi:** 1.0, 15 November 2026 (M8.4). Mengikuti brief desain v1.32 dan kode sampai Tahap 8.
- **Untuk siapa:** tim IT yang menilai dan nanti membangun versi perusahaan dari spesifikasi dan UML (D-33). Dokumen ini bisa dibaca tanpa melihat kode; kode demo tidak diserahkan (R-SEC-05).
- **Pendamping:** `ringkasan-untuk-it.md` (bahan presentasi 12 slide), paket UML di `docs/uml/`, kontrak API `api/openapi.yaml`, dan `docs/panduan/panduan-operasional.md`.
- **Tentang angka:** dokumen ini tidak memuat perkiraan ukuran server. Angka yang tertulis adalah hasil pengamatan di demo dan ditandai begitu; sisanya ditulis "belum diukur" (brief bagian 15, butir 5).

## 1. Apa sistem ini

Sistem web untuk menyusun, merevisi, menomori, mencetak, dan mencatat rilis PFD, PFMEA, dan Control Plan. Persetujuan tetap tanda tangan basah di kertas (D-07); sistem mencatat rilis sesudah scan bertanda tangan diunggah. Tujuan utamanya mempermudah revisi dan yokoten: satu perubahan di process template bisa disebarkan ke dokumen banyak model dengan jejak yang lengkap.

Versi yang ada sekarang adalah demo di localhost milik pemilik proyek (D-36). Fungsinya mematangkan konsep dan alur kerja, bukan dipakai harian.

## 2. Arsitektur

| Bagian | Di demo | Catatan untuk versi IT |
|---|---|---|
| Browser | Aplikasi satu halaman (SPA) statis | Hanya klien dari API; tidak menyimpan aturan bisnis |
| Reverse proxy | Caddy 2.11.7, HTTPS dengan CA lokal | Ganti dengan proxy dan sertifikat perusahaan |
| Aplikasi | Satu binary (Go 1.27.3) yang memuat API `/api/v1` dan berkas SPA | Tanpa status di memori selain sesi di database, jadi bisa dijalankan ulang kapan saja |
| Database | PostgreSQL 18.6 | Satu-satunya tempat data. Antrean job juga direncanakan di PostgreSQL, tanpa Redis |
| Penyimpanan berkas | Folder privat; nama berkas adalah SHA-256 isinya | Tidak pernah dilayani langsung; unduhan lewat API yang memeriksa hak |
| Pemindai antivirus | ClamAV 1.5.4 (opsional di demo) | Bisa diganti antivirus perusahaan (lihat bagian 8) |
| PDF | Chrome atau Edge yang terpasang di komputer server | Image container demo tidak memuat browser; cetak Excel selalu tersedia |

Alur satu permintaan: browser → proxy (HTTPS) → aplikasi → pemeriksaan sesi → pemeriksaan izin → database. Setiap jawaban error memakai satu format berisi kode, pesan, dan request ID; rincian teknis hanya ada di log server (R-API-13).

API-first: semua layar memakai API yang sama, yang dijelaskan kontrak OpenAPI 3.0.4 (D-38, D-61). Tim IT bisa membangun klien atau server sendiri dari kontrak itu.

## 3. Data

- Baris dokumen disimpan datar dengan isi JSON yang divalidasi terhadap versi format template (D-02). Form customer baru tidak butuh perubahan skema.
- ID tetap, nama hanya tampilan: proses, jabatan, kolom format, dan baris dokumen (D-06, D-58).
- Revisi terbit dibekukan sebagai snapshot beserta SHA-256, dan dijaga trigger database (R-REV-02).
- Nomor dokumen dialokasikan server dalam transaksi dengan baris penghitung, bukan sequence, supaya tidak ada lubang dan tidak ada nomor ganda (R-NUM-03).
- Skema dibangun dari 42 berkas migrasi berurutan. Diagram tabelnya ada di U8.

## 4. Keamanan

### 4.1 Akun dan sesi

- Akun lokal dengan password yang di-hash argon2id. Belum ada SSO; itu keputusan IT untuk versi perusahaan.
- Sesi berupa cookie `HttpOnly`, `Secure`, `SameSite=Lax`. Permintaan yang mengubah data dari situs lain ditolak (pemeriksaan Origin).
- Akun admin terpisah dari akun kerja, dan tidak pernah memegang peran kerja (D-72). Akun admin terakhir tidak bisa dinonaktifkan.

### 4.2 Hak akses, dua pagar

1. **Fungsi izin di aplikasi.** Setiap operasi melewati satu fungsi pemeriksaan (aktor, hak, resource). Peran adalah kumpulan hak; admin mengatur isi peran di layar tanpa perubahan kode (D-66). Project Leader dan Project Engineer hanya bekerja untuk customer di Project Group sendiri (D-71).
2. **Row-level security di PostgreSQL.** Tabel dokumen, baris, dan revisi disaring database menurut akun yang sedang dilayani. Query yang lupa memeriksa izin tetap tidak melihat customer lain. Akun tanpa hak mendapat "tidak ada" (404), bukan "ditolak", supaya keberadaan dokumen pun tidak bocor.

Aturan yang tidak bisa dimatikan peran atau pengaturan apa pun (R-SEC-11): pemohon tidak memverifikasi permintaannya sendiri, pencatat rilis bukan penyusun, revisi terbit tidak bisa diubah, audit log hanya bisa ditambah, peran kerja tidak memegang hak admin. Aturan ini dijaga di kode dan sekali lagi di database (constraint dan trigger).

### 4.3 Audit log

Setiap perubahan dicatat dalam transaksi yang sama dengan perubahannya. Role database aplikasi tidak punya UPDATE dan DELETE pada tabel audit, ada trigger penolak, dan setiap entri membawa hash entri sebelumnya (hash chain). Saran untuk versi IT: pemeriksaan rantai harian dan jangkar di luar sistem (D-108).

### 4.4 Database

- Dua role: pemilik (hanya untuk migrasi) dan role aplikasi dengan hak sekecil mungkin per tabel dan per kolom.
- Di demo, migrasi berjalan saat program mulai. Untuk versi IT disarankan langkah migrasi terpisah, dan kredensial pemilik hanya dipakai saat deploy (D-108).

### 4.5 Unggahan

Lampiran dan scan bertanda tangan dibatasi ukurannya (20 MB dan 30 MB), disimpan dengan SHA-256, dan bila pemindai dipasang dipindai sebelum disimpan. Bila pemindai tidak bisa dihubungi, unggahan ditolak.

### 4.6 Rahasia

Password, token, dan alamat koneksi tidak pernah masuk log, pesan error, atau audit log, dan ini dibuktikan dengan test (R-SEC-14). Rahasia dibaca dari environment atau berkas `.env` yang tidak masuk repositori.

## 4.7 Email notifikasi (D-194)

Sistem mengirim email saat ada yang menunggu seseorang. Email ditulis ke kotak keluar di database bersama peristiwanya, lalu dikirim pekerja di latar belakang. Email tidak pernah memuat password atau token.

Yang perlu disiapkan IT, salah satu:

| Jalur | Pengaturan | Catatan |
|---|---|---|
| Microsoft Graph (disarankan) | `MAIL_PROVIDER=graph`, `MAIL_FROM`, `GRAPH_TENANT_ID`, `GRAPH_CLIENT_ID`, `GRAPH_CLIENT_SECRET` | App registration di Microsoft Entra ID dengan izin aplikasi Mail.Send. Sebaiknya dibatasi ke kotak surat pengirim saja. |
| SMTP | `MAIL_PROVIDER=smtp`, `MAIL_FROM`, `SMTP_HOST`, `SMTP_PORT` (587), `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_STARTTLS` | Misalnya smtp.office365.com atau relay Exchange internal. Login SMTP dengan password biasa mungkin dibatasi di tenant perusahaan; mohon dicek. |

Belum diuji dengan tenant perusahaan (E-126). Sesudah diatur, admin menekan **Send me a test email** di Admin → Notifications.

## 4.8 Web ECR dan aplikasi lain (Tahap 9, D-193)

Web ECR atau aplikasi lain terhubung lewat `/integration/v1` dengan token service account (kontrak `api/integration.yaml`). Penggunanya tidak butuh akun di sistem ini; nama orangnya dikirim sebagai `on_behalf_of` dan hanya dicatat. Setiap usulan menunggu verifikasi orang yang punya akun di sistem ini. Tidak ada jalur API untuk verifikasi, approval, atau rilis.

## 5. Backup dan pemulihan

- Backup: `pg_dump` format custom dan arsip folder lampiran, keduanya dienkripsi dengan passphrase (age), ditambah manifest berisi SHA-256.
- Pemulihan selalu ke database baru; database yang ada tidak pernah ditimpa.
- Uji pemulihan di mesin yang sama sudah terbukti. **Uji pemulihan di mesin lain belum dilakukan** (kriteria M8.2); langkahnya ada di panduan operasional.
- RPO dan RTO belum ditetapkan. Itu keputusan IT dan pemilik proses.

## 6. Deployment

Demo: `docker compose -f compose.deploy.yaml up -d --build` menjalankan PostgreSQL, aplikasi, dan Caddy. Terbukti pada 15 November 2026: `https://localhost:8443/health` menjawab 200 lewat Caddy. Image aplikasi 166 MB (pengamatan di komputer pemilik).

Yang berbeda untuk versi IT: server dan sertifikat perusahaan, nama host, alamat yang terbuka ke jaringan pabrik, pemantauan, dan rotasi log. Semua itu di luar cakupan demo (D-36).

## 7. Kebutuhan server

| Hal | Status |
|---|---|
| CPU dan RAM aplikasi | Belum diukur. Demo berjalan di laptop pemilik bersama Docker Desktop |
| RAM pemindai antivirus | Belum diukur. ClamAV memuat seluruh basis tanda virus ke memori |
| Ukuran database | Belum diukur dengan data asli. Jumlah dokumen yang akan dimigrasi belum diketahui (Q-09); Control Plan saja lebih dari 380 |
| Ukuran penyimpanan berkas | Bergantung pada jumlah scan bertanda tangan (paling besar 30 MB per revisi) |
| Jumlah pengguna serentak | Belum diuji beban. Penomoran diuji dengan 50 pendaftaran paralel |

## 8. Yang kami minta dari IT

1. Tinjauan atas desain dan UML: siapa peninjaunya dan kapan (Q-44).
2. Keputusan cara login untuk versi perusahaan: akun lokal atau SSO (misalnya Microsoft Entra ID).
2a. Email notifikasi: jalur Microsoft Graph atau SMTP dan pengaturannya (bagian 4.7).
2b. Web ECR: siapa pemiliknya, dan apakah ia akan memakai API integrasi (bagian 4.8).
3. Server uji, nama host, dan sertifikat.
4. Antivirus: apakah pemindaian unggahan memakai antivirus perusahaan. ClamAV tanpa akses internet tidak mendapat pembaruan tanda virus (RK-58).
5. Tempat backup di luar server, jadwalnya, dan siapa yang menyimpan passphrase.
6. Huruf untuk cetakan: huruf Windows belum tentu boleh dipakai di container Linux (RK-58).
7. Aturan retensi untuk audit log dan scan bertanda tangan.
8. Bila nanti ada portal customer atau integrasi: zona jaringan (DMZ) dan uji penetrasi (Q-24, RK-31). Keduanya belum dibangun.

## 9. Yang belum ada

- Approval di dalam aplikasi (tanda tangan elektronik): tidak dibangun; persetujuan tetap di kertas.
- API integrasi `/integration/v1` dan token (Tahap 9) dan portal customer (Tahap 10): latihan sesudah MVP, belum dibangun.
- Konfirmasi slot saat rilis (R-APR-10) dan pembekuan rute saat cetak (R-APR-11): masih usulan.
- Diagram U4, U6, U7, U9, U12, U13, dan U14 di paket UML belum dibuat.
