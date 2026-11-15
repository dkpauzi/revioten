# ECR lewat API integrasi dan email notifikasi (D-193, D-194)

- **Tanggal:** 15 November 2026
- **Aturan yang dipakai:** D-188 sampai D-194, R-API-07, R-API-09, R-CHG-06, R-SEC-14, D-51, D-54, D-104, D-139.
- **Permintaan pemilik:** "untuk perubahaan karna ECR ... jika bisa bikin web kita agar bisa terhubung nantinya", "kita juga butuh email notifikasi ... siapkan agar nanti bisa di setting oleh pihak IT", "apakah UML sudah update" (15 November 2026).
- **Juga di sesi ini:** E-122 opsi (c): job ekspor yang tertahan diantrekan lagi saat server mulai (D-188); paket UML draft 0.16.

## 1. Apa yang dibuat

| Bagian | Hasil |
|---|---|
| ECR (D-193) | Usulan lewat `/integration/v1/changes` boleh menyebut `trigger` (misalnya ECR), `reference` (nomor ECR), `include_set` (seluruh PFD, PFMEA, CP satu set), dan tanpa `operations`. Perubahan menunggu verifikasi orang; sesudah diterima, draft ketiga dokumen terbuka dan penyusun mengubahnya. |
| Email (D-194) | Kotak keluar `notification_outbox` ditulis di transaksi yang sama dengan peristiwanya, lewat pendengar audit log. Pekerja mengirim setiap 15 detik lewat Microsoft Graph atau SMTP, mencoba ulang 5 kali (2, 4, 8, 16 menit). Delapan jenis peristiwa dengan saklar per jenis. Alamat email orang diisi admin atau orangnya sendiri. Token integrasi yang kedaluwarsa dalam 7 hari diberitahukan ke admin, sekali. |
| Layar | Admin → **Notifications** (server email, saklar, email uji, kotak keluar), tombol **Email** di Accounts, bagian **Email for notifications** di halaman akun sendiri. Bab Help "Email notifications". |
| UML draft 0.16 | U1, U2, U3 (status pembangunan, notifikasi, ECR), U3 peta pengguna (siapa yang butuh akun), U5-F sampai U5-I, U6 baru, U8 dibangkitkan dari database dengan `docs/uml/tools/u8_dari_database.py`. Belum dirender, karena Java dan PlantUML belum terpasang. |

| Bagian | File |
|---|---|
| Migrasi | `00046_notifications.sql` |
| Aturan | `internal/notify` (`notify.go` peristiwa ke penerima, `mail.go` Graph dan SMTP, `service.go` pengaturan dan pekerja), `internal/audit` (pendengar), `internal/integration/write.go` (ECR) |
| Adapter dan layar | `internal/server/notifications.go`, `web/src/routes/admin/notifications/+page.svelte`, Accounts, halaman akun |
| Test | `internal/notify` (peristiwa, saklar, percobaan ulang, tanpa pengirim, server SMTP tiruan, server Graph tiruan), test perubahan, dokumen, cetak, integrasi, dan server berjalan dengan pendengar notifikasi dalam mode ketat, `web/e2e/m9-notifications.test.ts` |

## 2. Cara mengulang dari nol

1. Tulis `TestNotifications` dengan pengirim tiruan, lalu migrasi 00046, query, dan `notify.go`.
2. Pasang pendengar di `audit.Record`, satu titik untuk semua peristiwa.
3. Tulis `TestSMTPMailer` dengan server SMTP tiruan dan `TestGraphMailer` dengan server HTTP tiruan, lalu `mail.go`.
4. Pekerja kotak keluar dan saklar, lalu kontrak API, layar, dan teks dua bahasa.
5. ECR: tambahkan kasus di `TestPropose`, lalu pemicu, `include_set`, dan usulan tanpa operasi.

## 3. Konsep yang dipelajari

- **Outbox pattern.** Email ditulis ke tabel di transaksi yang sama dengan aksinya, lalu dikirim terpisah. Aksi yang batal tidak pernah mengirim email, dan server email yang mati tidak menggagalkan aksi orang.
- **Satu titik pendengar.** Semua peristiwa sudah lewat audit log; notifikasi cukup mendengarkan di sana, tanpa mengubah setiap layanan.
- **Mode ketat di test.** Di produksi, sasaran yang tidak ditemukan tidak menggagalkan aksi; di test hal itu dijadikan error, sehingga peristiwa yang salah dipetakan langsung ketahuan. Mode ini menemukan satu kesalahan sungguhan selama pembangunan: peristiwa yokoten menyimpan ID yokoten di rincian, bukan di sasaran.
- **Client credentials.** Aplikasi meminta token akses ke Microsoft Entra ID dengan ID dan rahasia aplikasi, lalu memanggil Graph atas nama kotak surat pengirim.

## 4. Asumsi

- ASUMSI E-126 (jalur Microsoft 365), E-127 (penerima dan isi email). E-128 (akses Viewer ke dokumen terbit) belum diubah di kode.

## 5. Status test

- 15 November 2026: `go test -p 1 ./...` lulus (test layanan dengan pendengar notifikasi mode ketat), Vitest 156 lulus, E2E 62 lulus, tur otomatis lulus, skrip gambar Help berhasil.

## 6. Latihan tulis tangan

Tanpa AI, tambahkan satu jenis notifikasi baru, misalnya "lampiran ditambahkan ke perubahan" untuk verifikator: baris di `notification_events` (migrasi baru), satu `case` di `dispatch`, satu fungsi penyusun email, teks dua bahasa untuk namanya, dan satu kasus test di `TestNotifications`.
