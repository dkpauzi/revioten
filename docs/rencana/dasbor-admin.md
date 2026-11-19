# Rancangan · Dasbor admin: kesehatan web, bukan isi dokumen

- **Dibuat:** 20 November 2026, dari pertanyaan pemilik.
- **Status:** **usulan, menunggu keputusan pemilik.** Belum ada kode.

## 1. Permintaan pemilik

> "apakah sebaiknya untuk dashboard admin menampilkan informasi yang berbeda? karna peran admin lebih ke mengatur web, saya rasa yang perlu di tampilkan adalah kinerja web nya. contoh nya seperti berapa lama tanda tangan digital tertahan (yang tampil di dashboard top 5 saja, namun sediakan untuk detailnya), laporan error web (misalkan ada kasus terjadi error atau bug oleh user), berapa banyak dokumen yang di proses (dibagi lagi menjadi beberapa kategori seperti yokoten, ecr, dan lainnya)"

## 2. Pendapat Claude

Setuju. Akun admin terpisah dari akun kerja (D-72) dan tidak menyusun dokumen, jadi Home admin yang sekarang (kartu kerja, review 3 tahun) kurang berguna. Dasbor admin sebaiknya menjawab satu pertanyaan: **"apakah web ini berjalan sehat, dan di mana pekerjaan tertahan?"** Isinya angka dan daftar singkat, bukan isi dokumen.

## 3. Isi yang diusulkan

| Kartu | Isi di dasbor | Detail (halaman sendiri) | Sumber data | Sudah ada? |
|---|---|---|---|---|
| **Tanda tangan tertahan** | Top 5 permintaan yang paling lama menunggu: dokumen, slot, orang yang ditunggu, lama menunggu | Semua permintaan yang menunggu, saring customer dan orang, unduh Excel | `signature_requests`, `signature_slots`, `signatures` (fase 3) | Data ada, layar belum |
| **Error web** | Jumlah error server (HTTP 5xx) hari ini dan 7 hari, 5 terakhir dengan request ID | Daftar error dengan waktu, halaman, request ID, akun; tanpa password atau isi rahasia (R-SEC-14) | Baru: tabel `error_reports` diisi server saat error 5xx | Belum; sekarang error hanya di log server |
| **Laporan masalah dari pengguna** | Jumlah laporan terbuka | Daftar laporan, status (baru, diperiksa, selesai) | Baru: tombol **Report a problem** di setiap halaman (teks, halaman, request ID terakhir otomatis) | Belum |
| **Dokumen yang diproses** | Jumlah revisi terbit minggu ini dan bulan ini, per kategori: ECR/ECN, Customer claim, Internal audit, Yokoten, Review 3 tahun, Lainnya | Grafik per minggu atau bulan | Sama dengan laporan kinerja PPAP (D-197, `report_key`) | Data ada, dihitung ulang tanpa nama orang |
| **Email** | Email gagal atau tertahan di kotak keluar | Halaman Notifications yang sudah ada | `notification_outbox` (D-194) | Ada |
| **Akun** | Akun terkunci, akun nonaktif, gagal login 24 jam | Halaman Accounts | `accounts`, audit log | Sebagian |
| **Backup** | Waktu backup terakhir dan hasilnya | Panduan operasional | Manifest backup (D-119) | Belum terbaca layar |
| **Audit log** | Hasil "Check the chain" terakhir | Halaman Audit log | Audit log (R-SEC-03) | Ada, manual |

Usulan tambahan dari Claude (di luar contoh pemilik): kartu Email, Akun, Backup, dan Audit log. Semuanya hal yang memang diurus admin.

## 4. Batas

- Admin tidak melihat isi dokumen draft di dasbor ini (D-72, RLS). Daftar tanda tangan tertahan cukup nomor dokumen, slot, nama orang, dan lama menunggu.
- Laporan error tidak memuat password, token, atau isi permintaan (R-SEC-14, R-API-13).
- Dasbor ini menggantikan Home untuk akun admin; akun kerja tetap memakai Home yang sekarang.

## 5. Pertanyaan untuk pemilik

1. Setuju dasbor admin menggantikan Home untuk akun admin?
2. Tombol **Report a problem** untuk semua pengguna: perlu?
3. "Tertahan" dihitung sejak slot itu mulai ditunggu (tanda tangan sebelumnya), atau sejak dokumen dikirim? Usulan: sejak slot itu mulai ditunggu, supaya terlihat siapa yang menahan.
4. Kapan dibangun: sesudah fase 3 (usulan: sebagai fase 4).
