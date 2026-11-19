# D-198 · Fase 3: tanda tangan digital

- **Tanggal:** 20 November 2026
- **Keputusan:** D-198 (tanda tangan digital boleh, tanpa sertifikat; cetak basah tetap ada), D-203 (diperbarui: Engineer = slot Prepared "Process. Eng"; Checked hanya Project Leader tingkat Leader atau Supervisor), D-204 (tanggal rilis = tanggal dikirim; berurutan). Brief v1.37.
- **Migrasi baru:** 00050 (`customers.signature_mode`, `signature_requests`, `signature_slots`, `signatures`, `document_revisions.signature_request_id`, penguncian draft, tiga peristiwa email).
- **Rancangan:** `docs/rencana/tanda-tangan-digital.md`; desain di kanvas "Desain UI Sistem PFD PFMEA CP", baris "Digital signature".

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| **Kirim untuk tanda tangan** | Panel **Digital signature** di halaman dokumen draft (hak E-5): pilih Prepared (pengirim atau Project Leader tingkat Leader), Checked (Project Leader tingkat Leader atau Supervisor; otomatis Supervisor bila Leader yang Prepared), slot Approved terisi pemegang jabatannya, catatan, ketik ulang password, **Send for digital signature** |
| **Pembekuan** | Isi draft dibekukan dengan SHA-256, rute ikut dibekukan, tanggal rilis = tanggal kirim WIB (D-204). Layanan dan trigger database menolak perubahan baris, bagian, header, dan rute selama menunggu atau sesudah lengkap |
| **Tanda tangan** | Documents → **Signatures** (tab menunggu saya, saya kirim, siap dirilis) dan halaman tanda tangan: ketik ulang password, **Sign** atau **Reject** dengan alasan. Berurutan (D-49). Password salah dihitung seperti gagal login |
| **Tarik kembali** | Pengirim menekan **Withdraw** selama belum lengkap; tanggal rilis dikosongkan |
| **Rilis** | Sesudah lengkap, PPAP Engineer lain mencatat rilis tanpa scan. Database memeriksa bahwa revisi itu memang punya permintaan yang lengkap |
| **Cetak** | Slot tertulis "Signed digitally <tanggal>" dengan nama penanda tangan. **For signature** tetap bisa untuk revisi bertanda tangan digital (cetak basah), tanpa keterangan itu. Tanggal revisi tidak ditimpa tanggal cetak |
| **Customer** | Master data → Customers: kolom **Signatures** (Digital allowed / Wet only), diatur admin |
| **Home dan email** | Tugas **Sign** di "Waiting for you"; email ke penanda tangan berikutnya, ke pengirim bila ditolak, dan ke pencatat rilis bila lengkap |
| **Review 3 tahun** | Kolom Engineer kini orang Prepared dari tanda tangan digital terakhir set itu (D-203); bila belum ada, pembuat dokumen pertama |
| **Tidak bisa diubah** | Slot dan tanda tangan tidak bisa diubah atau dihapus siapa pun; permintaan hanya berubah sekali dari waiting |

## 2. Yang berbeda dari desain, dan alasannya

| Di desain | Di kode | Alasan |
|---|---|---|
| Halaman tanda tangan menampilkan tabel baris yang berubah | Ringkasan (SHA-256, nomor perubahan, catatan) dan tombol **Open the document** | Label kolom bergantung pada format setiap jenis dokumen; halaman dokumen sudah menampilkan isi yang dibekukan (baca saja) dan **Compare revisions**. Tabel di halaman tanda tangan bisa ditambah nanti |
| Pilihan "Digital / Wet" di formulir kirim | Formulir hanya untuk digital; tanda tangan basah tetap lewat **Print** → For signature | Alur basah sudah ada dan teruji; tidak perlu dua jalan ke hal yang sama |
| Password dan kode sekali pakai (brief bagian 8) | Password saja | Belum ada kode sekali pakai di sistem; tanda tangan tidak harus tersertifikasi (D-198) |
| Tur otomatis | Tur belum memuat tanda tangan digital; test E2E `m6-signatures.test.ts` menjalankan alurnya | Tur butuh lima akun dan pemegang jabatan tambahan; ditunda supaya tur tetap pendek |

## 3. Cara mengulang dari nol

```bash
docker compose up -d
go generate ./internal/db/... ./internal/server/...
npm --prefix web run gen:api
go test -p 1 ./...
npm --prefix web test
npm --prefix web run check
npm --prefix web run test:e2e
```

Mencoba di layar: buat akun berperan PPAP Engineer, Project Leader (cakupan Project Group sendiri), dan dua Manager; catat mereka sebagai pemegang jabatan PPAP Engineer, Project Leader Project Group customer, NPI Manager, dan QA QC Manager; atur rute default customer; buat dokumen; lalu ikuti bab Help "Digital signature".

## 4. Konsep yang dipelajari

- **Membekukan sebelum menandatangani.** Yang ditandatangani adalah hash isinya. Bila isi berubah, hash berubah, jadi isi dikunci dan rilis memeriksa hash itu lagi.
- **Tabel tambah saja (append-only).** Tanda tangan, seperti audit log, hanya bisa ditambah. Trigger menolak UPDATE dan DELETE, juga dari pemilik database.
- **Mesin status kecil.** Permintaan berpindah sekali: waiting → completed, rejected, atau withdrawn. Trigger menolak perpindahan lain.
- **Dua lapis penjaga.** Layanan Go menolak dengan pesan yang jelas; trigger database menolak bila ada jalan lain.
- **Mengetik ulang password** membuktikan orangnya sendiri yang menekan tombol, bukan sesi yang lupa ditutup.

## 5. Asumsi

- Penanda tangan membutuhkan hak S-1 dan L-4 untuk customer itu (L-4 supaya bisa membuka draft).
- Hanya orang yang slotnya sedang ditunggu yang bisa menolak.
- Tanggal "Signed digitally" di cetakan memakai tanggal WIB saat tanda tangan diberikan.

## 6. Temuan desain

- Tidak ada temuan E- baru. Usulan dasbor admin dicatat di `docs/rencana/dasbor-admin.md` (menunggu keputusan pemilik).

## 7. Status test

- Go: semua paket lulus, termasuk paket baru `internal/signature` (rencana slot D-203, kirim, urutan, password, tolak, tarik, customer wajib basah, rilis tanpa scan, cetak Excel, trigger tambah saja) dan tabel izin (9 operasi baru).
- Vitest: 164 lulus; cek tipe bersih.
- E2E: 66 lulus, termasuk `m6-signatures.test.ts` baru (kirim, tiga tanda tangan berurutan, rilis tanpa scan) dan `/signatures` di test tampilan HP.
- Tur otomatis dan gambar Help: lulus.

## 8. Latihan tulis tangan

Tanpa melihat kode, tulis fungsi yang menerima daftar slot (urutan, sudah ditandatangani atau belum) dan mengembalikan urutan slot yang sedang ditunggu, atau 0 bila semua sudah. Lalu tulis aturan: orang boleh menandatangani hanya bila slotnya adalah slot itu. Bandingkan dengan query `NextSignerOf` di `sql/queries/signatures.sql`.
