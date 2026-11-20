# Rancangan · Form Control Submission per dokumen

- **Dibuat:** 21 November 2026, dari permintaan pemilik dan desain `Desain-Submission-Control.zip` (artboard "05 SubmissionControl per dokumen").
- **Status:** **usulan, menunggu persetujuan pemilik.** Belum ada kode.

## 1. Permintaan pemilik

> "bantu saya untuk membuat Form control submission nya, tidak harus seperti referensi, namun setidaknya mencakup semua point yang di butuhkan, dan buat control ini jadi per dokumen saja. artinya Control plan punya control nya sendiri, PFMEA punya control nya sendiri, PFD punya control nya sendiri."

Form referensi (kertas): judul "Form Control Submission Document", header Customer dan Model, kolom No, Document Type, Document number, Document Revision, Submission Document (Date of Start, Date of Finish, Suitable to customer), Remark. Catatan form: setiap dokumen sebelum migrasi MP atau dari change request yang butuh tanda tangan top management dan customer dikendalikan dengan form ini.

## 2. Bentuknya di sistem

**Satu daftar per dokumen.** PFD, PFMEA, dan CP masing-masing punya daftarnya. Satu baris = satu revisi dokumen itu. Karena dokumennya sudah pasti, kolom Document Type dan Document number pindah ke header form.

| Kolom | Isi | Dari mana |
|---|---|---|
| No | Urutan | Otomatis |
| Revision | Nomor revisi dan statusnya (draft, menunggu tanda tangan, terbit) | Otomatis |
| Reason | Nomor perubahan (CHG-, YKT-) dan pemicunya; "First release" atau "Migration to MP" untuk revisi awal | Otomatis dari perubahan |
| Date of start | Perubahan diterima (atau draft dibuat untuk revisi awal) | Otomatis |
| Date of finish | Tanggal rilis | Otomatis |
| Top management sign | Tanda tangan digital terakhir (fase 3), atau scan bertanda tangan diunggah (basah) | Otomatis |
| Submit to customer | Tanggal dikirim ke customer dan isi paketnya, atau "Not required" dengan alasan | **Diketik** |
| Customer result | Approved / Rejected / Waiting, tanggalnya, dan scan bukti (opsional) | **Diketik** |
| Remark | Catatan bebas | **Diketik** |

Kolom referensi "Suitable to customer" menjadi dua kolom: **Submit to customer** dan **Customer result**, supaya terlihat apakah sudah dikirim dan apakah sudah disetujui.

## 3. Aturan (usulan)

| No | Aturan |
|---|---|
| SC-1 | Setiap revisi muncul di daftar dokumennya. Centang "Show revisions without customer submission" menyembunyikan revisi yang tidak perlu dikirim |
| SC-2 | "Perlu dikirim ke customer" diusulkan dari pengaturan customer (Master data → Customers, kolom baru **Customer submission**: Required / Not required). Per revisi bisa diubah, dengan alasan |
| SC-3 | Kolom otomatis tidak bisa diketik; isinya diambil dari perubahan, rilis, tanda tangan, dan scan |
| SC-4 | Catatan submission adalah data terpisah dari revisi terbit, jadi tetap bisa diisi sesudah rilis. Setiap perubahan tercatat di audit log |
| SC-5 | Cetak Excel dan PDF per dokumen: A4 landscape, 30 baris per halaman, header Customer, Model, Document type, Document number, catatan form di bawah, dan nomor form |
| SC-6 | Panel "Revision N needs": langkah yang sudah, sedang, dan belum (perubahan diterima, edit dan cek, rilis dan tanda tangan, kirim ke customer) |

## 4. Layar

- **Halaman dokumen → bagian "Submission control"**, dengan tombol pindah PFD · PFMEA · Control Plan satu set (seperti desain).
- Tabel di atas, kolom yang diketik bisa diubah langsung di baris (pemegang hak di pertanyaan 2).
- Panel samping: **Print control form** (PDF, Excel), **Revision N needs**, dan penjelasan "When a revision goes on this list".

## 5. Yang berbeda dari desain

| Di desain | Usulan | Alasan |
|---|---|---|
| "Day 8, target 5 working days" | "Day 8" saja, hari kalender | D-201: tanpa target; kalender libur belum ada (E-130) |
| Tab "Cover · PFD · PFMEA · Control Plan · Submission control" | Bagian di halaman dokumen | Halaman dokumen sekarang memakai bagian, bukan tab; isinya sama |

## 6. Pertanyaan untuk pemilik

1. **Nomor form.** Desain menulis "QR-ENG-24-K015 (Rev.01)", form kertas memakai nomor lain. Nomor mana yang dicetak? Usulan: diatur admin di Settings, karena bisa berubah.
2. **Siapa yang mengisi kolom yang diketik?** Usulan: pemegang hak E-7 (catat rilis; PPAP Engineer dan Supervisor), karena ini pekerjaan dokumen kontrol.
3. **Scan bukti customer.** Perlu unggah scan persetujuan customer per revisi, atau cukup tanggal dan hasil? Usulan: opsional, memakai penyimpanan lampiran yang sudah ada.

## 7. Urutan pembangunan (usulan, satu fase)

1. Migrasi: `customers.submission_required`, tabel `document_submissions` (per revisi: perlu dikirim, alasan, tanggal kirim, paket, hasil, tanggal hasil, scan, remark). Layanan dan test.
2. API, layar bagian Submission control, pengaturan customer.
3. Cetak Excel dan PDF, Help, E2E, catatan.
