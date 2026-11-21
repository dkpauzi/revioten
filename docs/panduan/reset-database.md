# Panduan reset database

Panduan ini mengosongkan database pengembangan `revioten`. Hasilnya database baru yang bersih: tabel dibuat ulang oleh migrasi, tanpa akun, customer, nomor, atau dokumen. Pakai panduan ini bila data uji coba lama mengganggu, misalnya bentrok dengan alur baru.

- Ditulis 22 November 2026.
- Semua perintah dijalankan di **PowerShell**, dari folder proyek `C:\E_Project\revioten-main`.
- Docker Desktop harus menyala.

## Yang terhapus dan yang tidak

| Terhapus | Tetap |
|---|---|
| Semua isi database `revioten`: akun (juga akun admin), peran, customer, model, proses, template, nomor, dokumen, revisi, perubahan, audit log | File `.env` dan password di dalamnya |
| | Database test E2E `revioten_e2e` (dibuat ulang sendiri setiap test berjalan) |
| | Role database `revioten_app` (dipakai lagi oleh database baru) |
| | Kode dan dokumen di repo |
| | Lampiran lama: tidak dihapus, hanya dipindah ke folder lain (langkah 4) |

**Jangan memakai `docker compose down -v`.** Perintah itu menghapus volume `revioten_pgdata`, yang berisi semua database di container, termasuk yang bukan milik aplikasi ini. Panduan ini hanya menghapus satu database.

## Langkah

### 1. Hentikan server aplikasi

Di jendela PowerShell tempat `go run ./cmd/revioten` berjalan, tekan **Ctrl+C**. Bila server tetap berjalan, server itu akan menampilkan error, karena tabelnya hilang di tengah jalan.

### 2. Pastikan PostgreSQL menyala

```powershell
docker compose up -d
```

```powershell
docker compose ps
```

Kolom STATUS harus berisi `healthy`.

### 3. Buat backup lebih dulu

Backup dibuat supaya data lama masih bisa dibuka bila ternyata dibutuhkan.

**a. Bila `BACKUP_DIR` dan `BACKUP_PASSPHRASE` sudah diisi di `.env`** (backup terenkripsi, M1.7):

```powershell
go run ./cmd/revioten backup
```

**b. Bila belum diisi**, buat salinan biasa ke folder `data` (folder ini tidak di-commit). Salinan dibuat di dalam container lalu disalin keluar, karena `>` di PowerShell merusak berkas biner:

```powershell
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -Fc revioten > /tmp/backup.dump'; docker compose cp db:/tmp/backup.dump data\backup-sebelum-reset.dump
```

Salinan ini **tidak terenkripsi**, tetapi berisi hash password dan data asli. Simpan hanya di komputer ini, dan hapus bila sudah tidak perlu.

### 4. Pindahkan lampiran lama

Lampiran (scan, berkas perubahan) disimpan di luar database, di folder `data\attachments`. Sesudah reset, berkas itu tidak punya pasangan lagi di database. Pindahkan foldernya, jangan dihapus:

```powershell
if (Test-Path data\attachments) { Move-Item data\attachments data\attachments-sebelum-reset }
```

Bila `ATTACHMENT_DIR` di `.env` menunjuk folder lain, pindahkan folder itu.

### 5. Hapus dan buat ulang database

```powershell
docker compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres -c "DROP DATABASE IF EXISTS revioten WITH (FORCE);" -c "CREATE DATABASE revioten;"'
```

Jawabannya `DROP DATABASE` lalu `CREATE DATABASE`. `WITH (FORCE)` memutus koneksi yang masih tersisa, misalnya server yang lupa dihentikan.

Bila `POSTGRES_DB` di `.env` bukan `revioten`, ganti kedua nama `revioten` di perintah itu.

### 6. Jalankan server

```powershell
npm.cmd --prefix web run build
```

```powershell
go run ./cmd/revioten
```

Migrasi berjalan sendiri saat server mulai dan membuat semua tabel beserta isi awalnya (jenis dokumen, form set bawaan, peran). Biarkan jendela ini terbuka.

### 7. Buat akun admin pertama lagi

Buka jendela PowerShell **kedua** di folder proyek:

```powershell
go run ./cmd/revioten create-admin -name "Nama Anda" -login admin.nama
```

Ketik password dua kali (tidak tampil di layar, minimal 8 karakter). Lalu buka http://127.0.0.1:8080 dan login dengan akun itu.

### 8. Isi lagi data awal

Urutan yang disarankan:

1. **Admin → Accounts:** buat akun kerja, misalnya PPAP Engineer.
2. **Login dengan PPAP Engineer.**
3. **Master data:** Customers dan Project Group, lalu Models.
4. **Template library:** Processes, lalu Process templates, lalu terbitkan form set.
5. **Documents:** Numbers, lalu Documents.

## Memeriksa hasilnya

```powershell
docker compose exec db psql -U revioten -d revioten -c "select count(*) from accounts;"
```

Sebelum langkah 7 jawabannya `0`, sesudahnya `1`.

## Membuka data lama dari backup

Backup tidak pernah menimpa database yang ada. Untuk melihat data lama:

- **Backup terenkripsi (3a):**

  ```powershell
  go run ./cmd/revioten restore -file <BACKUP_DIR>\revioten-<waktu>.dump.age
  ```

  Hasilnya database terpisah `revioten_restore`.
- **Salinan biasa (3b):**

  ```powershell
  docker compose cp data\backup-sebelum-reset.dump db:/tmp/backup.dump; docker compose exec -T db sh -c 'createdb -U "$POSTGRES_USER" revioten_lama && pg_restore -U "$POSTGRES_USER" -d revioten_lama /tmp/backup.dump'
  ```

Periksa isinya dengan `docker compose exec db psql -U revioten -d revioten_lama`. Buang bila sudah tidak perlu:

```powershell
docker compose exec db dropdb -U revioten revioten_lama
```
