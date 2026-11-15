# Panduan operasional (M8.4)

Untuk orang yang menjalankan sistem ini di komputernya. Semua perintah dijalankan dari folder proyek. Perintah yang belum pernah dicoba di komputer pemilik ditandai.

## 1. Dua cara menjalankan

| Cara | Kapan dipakai | Alamat |
|---|---|---|
| Pengembangan: database di Docker, aplikasi lewat `go run` | Saat menulis kode | http://127.0.0.1:8080 |
| Deploy lokal: semuanya di Docker, HTTPS lewat Caddy (M8.3) | Saat demo, tanpa Go dan Node di komputer | https://localhost:8443 |

Keduanya memakai berkas `.env` yang sama, tetapi database dan foldernya terpisah. Data di cara pertama tidak muncul di cara kedua.

## 2. Deploy lokal dengan Docker Compose

1. Pastikan Docker Desktop menyala dan berkas `.env` sudah terisi (contohnya `.env.example`).
2. Bangun dan jalankan: `docker compose -f compose.deploy.yaml up -d --build`. Pertama kali butuh beberapa menit.
3. Periksa: `docker compose -f compose.deploy.yaml ps`. Tiga layanan harus `Up`, dan `db` `healthy`.
4. Buat akun admin pertama (sekali): `docker compose -f compose.deploy.yaml exec app /app/revioten create-admin -name "Nama Pemegang" -login admin.nama`, lalu ketik password dua kali. *(Belum dicoba di komputer pemilik.)*
5. Buka https://localhost:8443. Browser memberi peringatan sertifikat, karena sertifikatnya dibuat Caddy sendiri. Pilih lanjut.
6. Lihat log: `docker compose -f compose.deploy.yaml logs -f app`.
7. Matikan, data tetap: `docker compose -f compose.deploy.yaml down`.

Catatan:
- Port bisa diganti dengan `HTTPS_PORT` di `.env`.
- Cetak PDF tidak tersedia di cara ini, karena image tidak memuat Chrome. Cetak Excel tetap jalan.
- Jangan menambahkan `-v` pada perintah `down`: itu menghapus database dan lampiran.

## 3. Pemindai antivirus (M8.2)

Tanpa pengaturan, unggahan tidak dipindai dan server menulis peringatan saat mulai.

- Deploy lokal: isi `DEPLOY_CLAMAV_ADDR=clamav:3310` di `.env`, lalu `docker compose -f compose.deploy.yaml --profile clamav up -d`. Pertama kali, ClamAV mengunduh basis tanda virus dan butuh beberapa menit sebelum siap. *(Belum dicoba di komputer pemilik; butuh internet dan RAM cukup besar.)*
- Pengembangan: jalankan clamd sendiri, lalu isi `CLAMAV_ADDR=127.0.0.1:3310` di `.env`.
- Bila pemindai dipasang tetapi mati, setiap unggahan ditolak dengan pesan yang jelas. Itu disengaja: berkas yang belum dipindai tidak pernah disimpan.
- Uji: unggah berkas uji EICAR sebagai lampiran perubahan. Harus ditolak.

## 4. Backup harian

1. Isi `BACKUP_DIR` (folder di disk lain) dan `BACKUP_PASSPHRASE` (minimal 15 karakter) di `.env`.
2. Jalankan `go run ./cmd/revioten backup` saat PostgreSQL menyala.
3. Hasilnya tiga berkas: `revioten-<waktu>.dump.age`, `revioten-<waktu>.manifest.json`, dan `revioten-<waktu>.files.tar.age`.
4. Simpan salinan passphrase di tempat lain. Tanpa passphrase, backup tidak bisa dibuka.

Perintah backup sekarang memakai database pengembangan (`compose.yaml`). Backup untuk deploy lokal belum dibuat.

## 5. Uji pemulihan di mesin lain (kriteria M8.2, tugas pemilik)

1. Di komputer kedua: pasang Docker Desktop dan Go, salin folder proyek (atau `git clone`), dan buat `.env` dengan password baru.
2. Salin ketiga berkas backup ke komputer itu, dan isi `BACKUP_PASSPHRASE` yang sama.
3. `docker compose up -d`.
4. `go run ./cmd/revioten restore -file <folder>/revioten-<waktu>.dump.age -files data/attachments-restore`.
5. `POSTGRES_DB=revioten_restore go run ./cmd/revioten` (PowerShell: `$env:POSTGRES_DB="revioten_restore"; go run ./cmd/revioten`).
6. Login, buka satu dokumen terbit, unduh scan-nya, lalu di **Admin → Audit log** tekan **Check the chain**.
7. Catat tanggal dan hasilnya di catatan milestone.

## 6. Pemeriksaan rutin

| Kapan | Yang diperiksa | Cara |
|---|---|---|
| Harian | Backup berhasil | Ada tiga berkas baru di `BACKUP_DIR` |
| Mingguan | Rantai audit utuh | Admin → Audit log → **Check the chain** |
| Bulanan | Pemulihan masih bisa | Bagian 5, di komputer yang sama cukup |
| Setiap ganti orang | Peran dan pemegang jabatan | Admin → Accounts, Admin → Roles, Master data → Positions |
| Sesudah migrasi selesai | Mode migrasi mati | Admin → Settings |

## 7. Bila ada masalah

| Gejala | Penyebab yang biasa | Yang dilakukan |
|---|---|---|
| Server tidak mau mulai: nilai `.env` belum diisi | `.env` belum lengkap | Bandingkan dengan `.env.example` |
| Server tidak mau mulai: database tidak bisa dihubungi | PostgreSQL belum menyala | `docker compose up -d`, tunggu `healthy` |
| Admin lupa password atau terkunci | | `go run ./cmd/revioten reset-admin-password -login admin.nama` |
| Dokumen "tidak ada" padahal ada | Akun tidak punya hak untuk customer itu (row-level security) | Periksa peran dan jabatan orangnya |
| Unggahan selalu ditolak | Pemindai antivirus mati | Nyalakan lagi, atau kosongkan `CLAMAV_ADDR` dengan sadar |
| PDF gagal dibuat | Chrome atau Edge tidak ditemukan | Isi `PDF_BROWSER`, atau cetak Excel |
