# C4 · PFD dulu, lalu PFMEA dan CP dirakit per langkah PFD

- **Tanggal:** 22 November 2026
- **Keputusan:** D-211 (akar masalah, urutan C), D-212 (langkah PFD dibekukan di snapshot CP), E-136 (setiap langkah PFD muncul di CP, "--" tanpa kontrol). Temuan baru: E-138, E-139.
- **Rencana:** `docs/rencana/workbook-form.md` bagian 6, disetujui pemilik ("Setuju, mulai C4").

## Tujuan

Alur menurut pemilik:

- Proses yang tidak bergantung model, misalnya Part Receiving, punya template umum.
- PFD dibuat per dokumen tanpa template.
- Langkah PFD menjadi dasar proses di PFMEA dan CP.

## Yang dibuat

| Bagian | Isi |
|---|---|
| Langkah PFD (`internal/document/pfdflow.go`) | `FlowStep` (bagian, nomor langkah, proses, simbol, NG) disusun dari baris PFD. Baris berurutan dengan nomor dan proses yang sama adalah satu langkah. CP membawa langkah PFD set-nya di `Document.Flow`. |
| D-212 | `Snapshot.Flow`: langkah PFD ikut ditulis saat CP ditandatangani dan dirilis. Cetakan dan pratinjau CP memakai `snap.Flow`. Snapshot lama tanpa langkah memakai PFD yang sekarang. |
| Rakit dari PFD (`internal/document/compose_pfd.go`) | `PFDComposition` menampilkan setiap langkah dengan versi terbit terakhir process template prosesnya dan keadaannya di PFMEA dan CP (`present`, `to_add`, `no_template`, `not_draft`). Juga `Extra`: nomor langkah PFMEA atau CP yang tidak ada di PFD. `ComposeFromPFD` membuat baris untuk langkah yang dipilih dalam satu transaksi, lalu mengurutkan baris mengikuti PFD. |
| Baris baru | Nomor langkah dan proses dari PFD. Nomornya ditandai manual, supaya **Order and operation numbers** tidak menggantinya dengan 10, 20, 30. Langkah tanpa template mendapat satu baris dengan "--" di setiap kolom teks. Kolom Relevant Document terisi seperti perakitan biasa. |
| API | `GET` dan `POST /api/v1/documents/{id}/pfd-composition` (L-4 dan E-2). Query baru: `LatestPublishedProcessVersions`, `AddStepRow`. |
| Layar | Tombol **From PFD** (Compose from PFD) di PFMEA dan CP yang set-nya punya PFD. Panel di laci (`PfdComposePanel.svelte`, logika di `pfdcompose.ts`) memuat tabel langkah, pemeriksaan, dan tombol rakit. Sheet PFD tidak lagi punya tombol rakit dari template. |
| Langkah berikutnya | PFD kosong: "isi PFD". PFMEA/CP kosong dengan PFD kosong: "isi PFD dulu", dengan tautan **Open the PFD**. Dengan langkah PFD: "rakit dari langkah PFD". |
| Pratinjau | Pemeriksaan langkah tampil di atas halaman PFMEA atau CP. |
| Help | Bab Documents (en dan id): satu langkah tentang rakit dari PFD. Help-shots mengisi PFD contoh lewat API, karena tombol rakit di PFD sudah tidak ada. |

## Cara mencoba

1. Pastikan ada process template terbit, misalnya Part Receiving dengan poin PFMEA dan CP.
2. Buat dokumen PFD, PFMEA, dan CP satu set. Buka CP: langkah berikutnya meminta Anda mengisi PFD dulu.
3. Buka PFD. Tambah baris: nomor langkah, proses, dan Flow Symbol.
4. Kembali ke CP, lalu tekan **From PFD**. Periksa keadaan setiap langkah, centang, lalu tekan **Compose the chosen steps**.
5. PFMEA dan CP terisi berurutan mengikuti PFD. Langkah tanpa template berisi "--".
6. Buka panel lagi: semua langkah sudah ada, dan pemeriksaannya bersih.
7. Tambah baris CP dengan nomor langkah yang tidak ada di PFD, lalu tekan **Preview**: pemeriksaan tampil di atas halaman.

## Konsep yang dipelajari

- **Satu sumber data.** Langkah PFD dibaca dari satu tempat (`StepsOfPFD`). Rakit, pemeriksaan, snapshot, dan cetakan memakai hasil yang sama.
- **Membekukan isi yang dipinjam.** Alur CP dipinjam dari dokumen lain (PFD). Supaya revisi terbit tidak berubah (R-REV-02), isi pinjaman itu ikut ditulis ke snapshot. Hash tanda tangan digital juga mencakupnya. Akibatnya, bila PFD diubah sesudah CP dikirim untuk tanda tangan, rilis menolak karena isinya berbeda dengan yang ditandatangani.
- **Urutan stabil.** Baris baru disisipkan dengan pengurutan stabil menurut peringkat langkah di PFD. Baris lama satu langkah tetap dalam urutannya, dan baris baru menyusul di belakangnya.

## Asumsi

- **ASUMSI E-138:**
  - Proses yang sama di dua langkah PFD: baris langkah kedua memakai isi poin yang sama, tetapi tidak tertaut ke template, karena indeks unik hanya mengizinkan satu baris per poin per bagian.
  - Baris PFD tanpa nomor langkah dilewati.
- **Langkah yang sudah ada:** langkah dianggap sudah ada bila dokumen punya baris dengan nomor langkah dan proses yang sama di bagian yang sama. Langkah itu tidak dirakit lagi, juga bila barisnya hanya satu.
- **Baris di luar PFD:** baris yang langkahnya tidak ada di PFD ikut bergeser bersama baris di atasnya. Bila ada di awal, baris itu pindah ke akhir.
- **Poin yang berlaku:** dihitung untuk model pertama dokumen (ASUMSI E-93, sama dengan M4.5).
- **Hanya CP yang membawa `Flow`:** alasannya, hanya CP yang menggambar Process Flow.
- **E-139:** Compose from process templates di PFMEA atau CP masih mengisi poin PFD ke PFD satu set. Ini tidak diubah sampai pemilik memutuskannya.
- **Tur:** tidak diubah di C4. Tur masih merakit CP dari template dan tetap jalan. Penulisan ulang dengan data TR-020 dan alur baru dijadwalkan di C5 (D-211).

## Temuan desain baru

- **E-138:** proses yang sama di dua langkah PFD, dan baris PFD tanpa nomor langkah. Terbuka.
- **E-139:** poin PFD di process template bertentangan dengan D-211 (PFD tanpa template). Terbuka; rekomendasinya rakit dari template tidak lagi mengisi PFD.

## Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus. Test baru: `TestComposeFromPFD` (PFD kosong ditolak, keadaan langkah, urutan, baris "--", proses yang sama di dua langkah, rakit ulang tanpa perubahan, pemeriksaan, `Flow` di snapshot CP), `TestStepsOfPFD`, `TestFlowFrozenAtRelease` (alur revisi terbit tetap sesudah PFD diubah), handler dan izin endpoint baru |
| `npm --prefix web test` | 180 lulus (baru: langkah berikutnya C4 dan `pfdcompose.spec.ts`) |
| `npm --prefix web run test:e2e` | 71 lulus; `m4-pfd.test.ts` memeriksa alur PFD → CP dan PFMEA di layar |
| Help-shots | Lulus; gambar diperbarui |

## Latihan tulis tangan

PFD berisi langkah 1 Part Receiving, 2 IQC, dan 3 Storage. CP sudah punya dua baris IQC (langkah 2) dan satu baris langkah 9. Hanya Part Receiving yang punya template, dengan dua poin CP. Tulis urutan baris CP sesudah semua langkah dirakit, dan tandai baris mana yang baru, mana yang berisi "--", dan mana yang muncul di pemeriksaan.
