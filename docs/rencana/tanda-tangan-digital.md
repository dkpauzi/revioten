# Rancangan · Fase 3: tanda tangan digital (D-198, D-203)

- **Dibuat:** 20 November 2026.
- **Status:** **usulan, menunggu persetujuan pemilik.** Belum ada kode. D-198 meminta desain layar lebih dulu; desainnya ada di kanvas "Desain UI Sistem PFD PFMEA CP" (https://claude.ai/artifact/UzNGNNQq69sPLzHS9fQPAU), baris "Digital signature": SendForSignature, SignDocument, SignatureStatus.
- **Keputusan yang dipakai:** D-198 (tanda tangan digital boleh, tanpa sertifikat; cetak basah tetap ada; tanggal revisi tidak ikut tanggal cetak), D-199 (hak S-1), D-203 (Engineer dipilih saat mengajukan), D-47 dan D-116 (rute default customer), D-49 (urutan berurutan), D-52 (tanggal tanda tangan per slot, saklar), R-APR-07 sampai R-APR-12, R-REV-06, R-REV-08.

## 1. Alur

```
Draft ──Ready to print──► Draft siap
   │
   └─ Kirim untuk tanda tangan (PPAP Engineer / Project Leader, hak E-5)
        pilih: cara (Digital / Basah), rute, penanda tangan setiap slot
        isi dibekukan: snapshot + SHA-256, rute ikut dibekukan
        slot Prepared = pengirim, ditandatangani saat mengirim
              │
              ▼
      Menunggu tanda tangan ──► Checked ──► NPI Manager ──► QA QC Mgr ──► (GM)
              │                   satu per satu, berurutan (D-49)
              │   setiap penanda tangan: buka dokumen, baca, ketik ulang password, Sign
              │
              ├── Tolak (alasan wajib) ──► kembali ke Draft, semua tanda tangan batal
              ├── Pengirim menarik kembali ──► kembali ke Draft
              │
              ▼
      Ditandatangani lengkap ──► Catat rilis (hak E-7, bukan penyusun) ──► Released
                                  tanpa scan; tanggal rilis = tanggal tanda tangan terakhir (usulan)
```

Cara **Basah** tetap seperti sekarang: cetak untuk tanda tangan, unggah scan, catat rilis.

## 2. Aturan (usulan)

| No | Aturan | Dasar |
|---|---|---|
| T-1 | Setiap customer punya **cara tanda tangan**: *Digital boleh* (bawaan) atau *Wajib basah*. Diatur admin di Master data → Customers | D-198 |
| T-2 | Saat mengirim, isi revisi dibekukan seperti cetak untuk tanda tangan (R-REV-06): snapshot dan SHA-256. Edit sesudahnya tidak mungkin sampai ditolak atau ditarik | R-REV-06, R-APR-11 |
| T-3 | Pengirim memilih penanda tangan setiap slot dari pemegang jabatan yang diizinkan slot itu, yang juga memegang hak S-1. Slot jabatan satu orang (NPI Manager, QA QC Manager, GM) terisi otomatis | D-203, R-APR-08, D-115 |
| T-4 | Satu orang hanya satu slot dalam satu rute | R-APR-12 |
| T-5 | Tanda tangan berurutan; slot berikutnya baru bisa menandatangani sesudah slot sebelumnya | D-49 |
| T-6 | Saat menandatangani, penanda tangan mengetik ulang password (tanpa kode sekali pakai untuk sekarang) | Brief bagian 8, level 1 |
| T-7 | Catatan tanda tangan tidak bisa diubah: siapa, jabatan dan judul slot saat itu, arti (Prepared, Checked, Approved), waktu server, SHA-256 isi, alamat IP. Dijaga trigger seperti audit log | Brief bagian 8 |
| T-8 | Tolak: alasan wajib; revisi kembali ke Draft; semua tanda tangan putaran itu batal tetapi tetap tercatat di riwayat | R-REV-06 |
| T-9 | Rilis digital tidak butuh scan. Pencatat rilis bukan penyusun | R-APR-02 |
| T-10 | Cetak Excel dan PDF revisi bertanda tangan digital: setiap slot menulis nama dan "Signed digitally" dengan tanggalnya; tanggal revisi dan tanggal rilis tetap tanggal aslinya | D-198 |
| T-11 | Cetak untuk tanda tangan basah tetap tersedia untuk dokumen apa pun, juga yang sudah bertanda tangan digital; tanggal revisinya tetap | D-198 |
| T-12 | Penanda tangan yang menunggu muncul di Home → Waiting for you, dan mendapat email (peristiwa baru `signature_requested`) | D-194 |

## 3. Layar

1. **Halaman dokumen → Revisions → Send for signature** (panel): cara tanda tangan, rute, pilihan orang per slot (Engineer: PPAP Engineer atau Project Leader), catatan untuk penanda tangan, tombol Send.
2. **Status tanda tangan di halaman dokumen:** daftar slot dengan status (Signed, Waiting, Not yet), waktu, dan tombol Withdraw untuk pengirim.
3. **Halaman tanda tangan** (dibuka dari Home atau email): ringkasan dokumen, isi revisi baca saja dengan perubahan ditandai, SHA-256 pendek, kotak password, tombol **Sign** dan **Reject**.
4. **Documents → Signatures:** daftar "Menunggu tanda tangan saya" dan "Yang saya kirim".
5. **Master data → Customers:** kolom cara tanda tangan.
6. **Cetakan:** blok tanda tangan dengan "Signed digitally" dan tanggal.

## 4. Pertanyaan untuk pemilik

1. **Kolom Engineer.** Di cover TR-020 slot kedua (Checked) berjudul cetak **"Engineer"**. Apakah "kolom Engineer" yang Anda maksud adalah slot ini? Usulan: ya, slot Checked boleh dipilih dari PPAP Engineer atau Project Leader (sekarang hanya Project Engineer atau Project Leader), dan orang yang dipilih juga menjadi "engineer" set di daftar review 3 tahun.
2. **Tanggal rilis digital.** Usulan: tanggal tanda tangan terakhir. Alternatif: tanggal dikirim untuk tanda tangan (seperti cetak basah, R-REV-08).
3. **Slot Prepared.** Usulan: pengirim otomatis menjadi Prepared dan tanda tangannya tercatat saat mengirim.
4. **Penandatanganan berurutan atau bebas urutan** untuk slot Approved. Usulan: berurutan (D-49).

## 5. Urutan pembangunan (usulan, satu langkah per sesi bila panjang)

| Langkah | Isi |
|---|---|
| 3A | Migrasi: cara tanda tangan customer, tabel `signature_requests` dan `signatures` (append-only), status revisi "awaiting signature". Layanan kirim, tanda tangan, tolak, tarik, catat rilis digital. Test termasuk pemisahan tugas dan urutan |
| 3B | Layar: panel Send for signature, status slot, halaman tanda tangan, Documents → Signatures, Home, email |
| 3C | Cetak dengan "Signed digitally", Help, tur, UML, catatan |
