# Rancangan · Review berkala PFD, PFMEA, dan Control Plan setiap 3 tahun

- **Dibuat:** 10 November 2026, dari permintaan pemilik.
- **Keputusan yang sudah dicatat:** D-138 di brief.
- **Status:** rancangan (usulan), **belum dibangun**. Rincian yang belum diputuskan ada di temuan E-86.

## 1. Permintaan pemilik

> "di WI general control plan perusahaan, ada informasi bahwa Control plan akan di review per 3 tahun, jadi tambahkan fitur untuk Control plan, PFD dan PFMEA yang tidak ada update / revisi selama 3 tahun, munculnya notifikasi di dashboard sebagai peringatan bahwa model tersebut harus di review. kemudian untuk saat ini buat model yang sudah muncul peringatan tersebut bisa di review dengan adanya perubahan, dan di review namun tidak ada perubahaan"

## 2. Mengapa belum bisa dibangun sekarang

Peringatan dihitung dari **tanggal rilis revisi terakhir**. Tanggal itu baru ada setelah sistem bisa mencatat rilis (Tahap 6). Saat ini sistem baru punya register nomor, yang hanya mencatat kapan nomor didaftarkan, bukan kapan dokumen terakhir direvisi. Bila peringatan dihitung dari tanggal pendaftaran nomor, hasilnya salah untuk setiap dokumen yang pernah direvisi.

Jadi fitur ini saya tempatkan setelah pencatatan rilis ada. Yang bisa disiapkan lebih awal: tanggal rilis revisi terakhir ikut diisi saat nomor lama diimpor dan saat dokumen lama dimigrasi (Tahap 7), supaya peringatan langsung benar untuk dokumen lama.

## 3. Rancangan

### 3.1 Kapan sebuah set dokumen jatuh tempo

- Jam dihitung **per set dokumen** (PFD, PFMEA, dan CP dengan nomor berjalan yang sama), karena ketiganya direview bersama untuk satu model.
- Tanggal acuan = yang paling baru di antara:
  1. tanggal rilis revisi terakhir dari salah satu dokumen di set itu, dan
  2. tanggal review berkala terakhir yang hasilnya "tidak ada perubahan".
- Jatuh tempo = tanggal acuan + **3 tahun**. Lamanya disimpan sebagai pengaturan sistem (Admin › Settings), bawaan 36 bulan, supaya mengikuti WI bila WI berubah.
- Tahap peringatan:

| Status | Kapan | Tampilan |
|---|---|---|
| Akan jatuh tempo | 90 hari sebelum jatuh tempo (usulan) | Kuning, "Review due in N days" |
| Jatuh tempo | Lewat 3 tahun | Merah, "Review overdue by N days" |

- Dokumen Obsolete, Void, dan model EOL tidak diberi peringatan.

### 3.2 Di mana peringatan tampil

- **Home (dashboard):** kartu "Periodic review" berisi jumlah set yang jatuh tempo dan yang akan jatuh tempo. Setiap akun hanya melihat customer yang boleh dilihatnya (D-71).
- **Daftar review:** model, customer, nomor dokumen, revisi terakhir, tanggal rilis terakhir, tanggal jatuh tempo, dan selisih hari. Bisa disaring per customer dan diekspor ke Excel sebagai bukti daftar.
- Tidak ada email (belum ada layanan email di sistem). Peringatan hanya di layar.

### 3.3 Dua hasil review

**A. Review dengan perubahan.** Tombol "Review with changes" membuka permintaan perubahan biasa (Tahap 5) dengan sumber perubahan baru **"Periodic review"**. Alurnya sama dengan perubahan lain: verifikasi, draft revisi, cetak, tanda tangan, rilis. Saat revisi baru rilis, jam 3 tahun mulai lagi dari tanggal rilis itu.

**B. Review tanpa perubahan.** Tombol "Reviewed, no change" membuat **catatan review** (bukan revisi baru):
- revisi yang direview (nomor dan revisi PFD, PFMEA, dan CP, beserta SHA-256 snapshot-nya, R-REV-02);
- tanggal review;
- peserta review (nama dan jabatan);
- daftar periksa yang dijawab (bagian 4.2);
- kesimpulan dan catatan;
- lampiran bukti (foto atau PDF lembar review bertanda tangan), opsional.

Catatan review tidak bisa diubah setelah disimpan, masuk audit log, dan memulai lagi jam 3 tahun. Nomor revisi dokumen **tidak naik**, karena isi dokumen tidak berubah.

### 3.4 Siapa yang mereview

- Dibuat oleh pemegang hak E-3 (Penyusun), karena hasilnya bisa menjadi permintaan perubahan.
- Untuk hasil "tanpa perubahan", catatan dikonfirmasi oleh orang kedua dengan hak V-1 (Verifikator), mengikuti aturan "pemohon tidak memverifikasi permintaannya sendiri" (R-CHG-02). Ini usulan; lihat E-86.

## 4. Cara membuktikan review sudah dilakukan (saran)

Auditor biasanya menanyakan tiga hal: kapan direview, siapa yang mereview, dan apa yang diperiksa. "Tidak ada perubahan" tanpa catatan apa pun sulit dibuktikan.

### 4.1 Tiga pilihan bukti

| Pilihan | Bentuk bukti | Kelebihan | Kekurangan |
|---|---|---|---|
| (a) Catatan review di sistem (rekomendasi) | Catatan review dengan daftar periksa, peserta, dan konfirmasi orang kedua; bisa dicetak sebagai lembar "Periodic review record" | Tidak mengubah dokumen, cepat, bisa ditelusuri dari daftar review | Tanda tangan basah hanya ada bila lembarnya dicetak dan dipindai |
| (b) Revisi administratif | Dokumen naik revisi, riwayat revisi berisi "Periodic review, no change" | Bukti ada di dokumen itu sendiri, sesuai kebiasaan tanda tangan basah (D-07) | Enam tanda tangan untuk dokumen yang tidak berubah; nomor revisi naik tanpa perubahan isi; salinan terkendali harus diganti |
| (c) Gabungan | Catatan review di sistem, lalu lembar cetaknya ditandatangani dan dipindai sebagai lampiran | Ada tanda tangan basah tanpa menaikkan revisi | Satu langkah cetak dan pindai tambahan |

Rekomendasi saya (a), dengan (c) bila auditor atau customer meminta tanda tangan. Jenis perubahan "Administrative" yang sudah ada di master (M1.6) tetap bisa dipakai untuk (b) bila WI mewajibkan revisi.

### 4.2 Daftar periksa review (usulan isi awal, bisa diubah admin)

1. Proses di PFD masih sama dengan proses di line (urutan, mesin, jig).
2. Ada komplain customer, temuan audit, atau NG internal sejak revisi terakhir yang belum masuk PFMEA.
3. Nilai Occurrence dan Detection masih sesuai data NG dan hasil inspeksi terbaru.
4. Karakteristik khusus dan simbolnya masih sesuai drawing dan spesifikasi customer terbaru.
5. Metode kontrol, frekuensi, dan ukuran sampel di CP masih dijalankan seperti tertulis.
6. Alat ukur dan dokumen rujukan (WI, SOP, QEP) di Doc ref masih berlaku dan revisinya benar.
7. Reaction plan masih sesuai prosedur yang berlaku.
8. PFMEA dan CP masih sinkron (cek PFMEA–CP, UC-36).

Setiap butir dijawab "OK", "Perlu perubahan", atau "Tidak berlaku", dengan catatan. Satu saja jawaban "Perlu perubahan" membuat hasil review menjadi A (dengan perubahan).

## 5. Kaitan dengan IATF 16949

**Peringatan:** teks IATF 16949:2016 berbayar dan tidak saya miliki, jadi nomor klausul di bawah berasal dari ingatan model dan sumber sekunder di web (10 November 2026). **Belum diverifikasi ke teks standar.** Cocokkan dengan salinan standar milik perusahaan sebelum dipakai di prosedur atau di depan auditor.

| Klausul (belum diverifikasi) | Isi singkat | Kaitan dengan fitur ini |
|---|---|---|
| 8.5.1.1 Control plan | Control plan ditinjau dan diperbarui bila ada produk tidak sesuai yang terkirim, bila ada perubahan yang memengaruhi produk atau proses, setelah komplain customer, dan **pada frekuensi yang ditetapkan berdasarkan analisis risiko** | Dasar review berkala. Sumber sekunder tidak menyebut angka tahun; 3 tahun adalah pilihan WI perusahaan. Auditor bisa menanyakan dasar risikonya |
| 7.5.3 (ISO 9001) Pengendalian informasi terdokumentasi | Dokumen ditinjau, disetujui, dan dikendalikan perubahannya; rekaman dijaga | Catatan review adalah rekaman yang harus bisa ditemukan dan tidak bisa diubah |
| 7.5.3.2.1 Retensi rekaman | Rekaman disimpan selama masa yang ditetapkan | Catatan review dan lampirannya ikut aturan retensi |
| 6.1.2.1 Analisis risiko | Analisis risiko memuat pelajaran dari recall, audit, retur, komplain, scrap, dan rework | Butir 2 dan 3 daftar periksa |
| 10.2.3 Problem solving | Tindakan perbaikan mencakup tinjauan dan pembaruan PFMEA dan control plan bila perlu | Review karena kejadian (bukan berkala) tetap lewat permintaan perubahan |
| 9.2.2.3 Audit proses manufaktur | Semua proses manufaktur diaudit dalam siklus tiga tahun, termasuk penerapan PFMEA dan control plan | Kemungkinan asal angka 3 tahun di banyak perusahaan; audit ini bisa menjadi pemicu review yang sama |

Buku AIAG & VDA FMEA Handbook dan manual Control Plan AIAG 2024 juga menyebut FMEA dan control plan sebagai dokumen hidup, tetapi halaman dan bunyinya belum saya verifikasi.

Sumber sekunder yang dibaca:
- https://blog.aiag.org/core-tools-and-iatf-169492016-overview
- https://www.sgs.com/en-ph/news/2022/04/a-guide-to-risk-based-thinking-in-iatf-16949
- https://memastermind4u.blogspot.com/2024/05/procedure-of-control-plan.html (blog tidak resmi)

## 6. Model data (usulan)

- `periodic_reviews`: id, document_set_id, tanggal review, hasil (`no_change` atau `change_requested`), change_request_id (boleh kosong), kesimpulan, dibuat oleh, dikonfirmasi oleh, waktu konfirmasi.
- `periodic_review_revisions`: review_id, document_id, revision_id, snapshot_sha256.
- `periodic_review_participants`: review_id, person_id, position_id.
- `periodic_review_answers`: review_id, butir daftar periksa, jawaban, catatan.
- `periodic_review_attachments`: review_id, file_id.
- Pengaturan sistem: `periodic_review_months` (bawaan 36) dan `periodic_review_warning_days` (bawaan 90).
- Role aplikasi tanpa UPDATE dan DELETE pada tabel ini, kecuali kolom konfirmasi.

## 7. Milestone (usulan)

| Langkah | Isi | Kapan |
|---|---|---|
| R1 | Pengaturan lamanya review, sumber perubahan "Periodic review", dan kolom tanggal rilis terakhir saat impor nomor lama | Bisa sekarang, tetapi belum berguna tanpa R2 |
| R2 | Hitung jatuh tempo, kartu di Home, dan daftar review | Setelah catat rilis (Tahap 6) |
| R3 | Review tanpa perubahan: catatan, daftar periksa, konfirmasi, lembar cetak | Setelah R2 |
| R4 | Review dengan perubahan: membuka permintaan perubahan | Setelah Tahap 5 dan R2 |
