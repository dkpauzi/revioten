# D-207 · Form Control Submission per dokumen, dan D-208 peta Laravel/React untuk IT

- **Tanggal:** 21 November 2026
- **Keputusan:** D-207 (Form Control Submission per dokumen; jawaban pemilik: semua usulan di `docs/rencana/submission-control.md`), D-208 (demo tetap Go/SvelteKit; peta komponen Laravel dan React di `docs/presentasi/dokumen-untuk-it.md` bagian 10). Brief v1.39.
- **Migrasi baru:** 00052 (`customers.submission_required`, `system_settings.submission_form_number`, `document_submissions` dengan RLS).
- **Desain acuan:** `Desain-Submission-Control.zip`, artboard "05 SubmissionControl per dokumen", dan form kertas "Form Control Submission Document".

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| **Bagian Submission control di halaman dokumen** | Satu daftar per dokumen, satu baris per revisi; tombol PFD · PFMEA · Control Plan satu set |
| **Kolom otomatis** | Revision dan statusnya, Reason (CHG- dan pemicunya, "First release", "Legacy document"), Date of start (perubahan diterima, atau draft dibuat), Date of finish (tanggal rilis) dengan lama hari, Top management sign (tanda tangan digital terakhir, atau scan bertanda tangan diunggah) |
| **Kolom yang diketik (E-7)** | Perlu dikirim ke customer (bawaan customer, pengecualian dengan alasan), tanggal kirim, yang dikirim, hasil customer (Waiting, Approved, Rejected) dan tanggalnya, catatan, bukti persetujuan customer (opsional) |
| **Panel samping** | Cetak form (PDF, Excel, A4 landscape, 30 baris per halaman), "Revision N needs", dan penjelasan kapan revisi masuk daftar |
| **Master data → Customers** | Kolom **Customer submission**: Required / Not required (A-2) |
| **Admin → Settings** | Nomor form yang dicetak di bawah form |
| **Dokumen untuk IT** | Bagian 10: peta komponen Go/Svelte → Laravel/React dan risikonya |

## 2. Yang berbeda dari desain, dan alasannya

| Di desain | Di kode | Alasan |
|---|---|---|
| "Day 8, target 5 working days" | "Day 8", hari kalender | D-201 (tanpa target); kalender libur belum ada (E-130) |
| Tab di atas halaman dokumen | Bagian di halaman dokumen | Halaman dokumen memakai bagian, bukan tab |
| Kolom "Suitable to customer" (form kertas) | Dua kolom: Submit to customer dan Customer result | Terlihat apakah sudah dikirim dan apakah sudah disetujui |
| Reason "Migration to MP" | "First release" untuk revisi awal | Sistem belum mencatat kapan model masuk MP; bila perlu, tulis di Remark |

## 3. Cara mengulang dari nol

```bash
docker compose up -d
go generate ./internal/db/... ./internal/server/...
npm --prefix web run gen:api
go test -p 1 ./...
npm --prefix web test
npm --prefix web run test:e2e
```

## 4. Konsep yang dipelajari

- **Data turunan vs data yang diketik.** Kolom tanggal dihitung saat dibaca dari perubahan, rilis, dan tanda tangan, jadi tidak pernah berbeda dari kenyataan. Hanya yang memang keputusan orang yang disimpan.
- **Catatan terpisah dari revisi terbit.** Revisi terbit tidak bisa diubah (R-REV-02), jadi data pengiriman disimpan di tabel sendiri yang ditautkan ke revisinya.
- **Bawaan dengan pengecualian beralasan.** Pengaturan customer menjadi bawaan; pengecualian per revisi wajib alasan, dan database menjaga pasangan itu dengan CHECK.
- **`INSERT ... ON CONFLICT`** membuat baris bila belum ada dan menguncinya dalam satu langkah.

## 5. Asumsi

- Membaca form butuh L-4; mengisi butuh E-7; bawaan customer butuh A-2.
- Tanggal ditulis dalam tanggal WIB.
- Revisi yang tidak perlu dikirim tidak boleh punya tanggal kirim atau hasil.

## 6. Temuan desain

- Tidak ada temuan E- baru.

## 7. Status test

- Go: semua 28 paket lulus, termasuk paket baru `internal/submission` (kolom otomatis, validasi, hak, bawaan customer, scan, Excel) dan tabel izin (6 operasi baru).
- Vitest: 165 lulus. E2E: 68 lulus, termasuk `m6-submission.test.ts` baru. Tur dan gambar Help lulus.

## 8. Latihan tulis tangan

Tanpa melihat kode, tulis aturan validasi isian satu revisi: hasil butuh tanggal kirim; tanggal hasil hanya untuk Approved atau Rejected dan tidak boleh sebelum tanggal kirim; revisi yang tidak perlu dikirim tidak boleh punya tanggal kirim. Bandingkan dengan fungsi `clean` di `internal/submission/submission.go`.
