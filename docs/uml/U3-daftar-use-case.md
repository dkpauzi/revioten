# U3 · Daftar use case

- Paket UML: draft 0.8, 30 Oktober 2026. Sumber: brief desain v1.16.
- Peta jabatan ke peran dan matriks hak akses ada di `U3-peta-pengguna.md`.
- Isi: daftar use case tanpa detail (S2 langkah 2). Spesifikasi lengkap (prasyarat, alur utama, alur alternatif, hasil) dibuat di langkah 5.
- Diagram: `U3-use-case.puml`, terdiri dari satu diagram ringkasan dan tujuh diagram per kelompok, diekspor ke `export/`.
- Setiap use case merujuk ke ID brief dan milestone, supaya bisa ditelusuri (R-UML-01). Matriks keterlacakan lengkap (U13) dibuat dari tabel ini.

**Status**
- **MVP:** dibangun sebagai bagian dari MVP, dari aturan yang sudah [DECIDED].
- **Usulan:** berasal dari aturan yang masih [PROPOSED].
- **Opsional:** milestone opsional.
- **Saklar:** fiturnya dibangun, tetapi saat ini dimatikan.
- **Masa depan:** latihan tahap 9 dan 10 (R-UML-06).

## Aktor

| Aktor | Jabatan saat ini | Rujukan |
|---|---|---|
| Pengguna internal | Semua pengguna yang login. Aktor induk dari aktor internal lain. | R-SEC-01 |
| Penyusun | PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer (peran author). Project Leader dan Project Engineer hanya untuk customer di Project Group-nya. | D-63, D-67, D-71 |
| PPAP Engineer | Penyusun yang juga mengelola master proses dan process template, serta memberi tanda dicek PFMEA–CP | D-64, D-65 |
| Verifikator | Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller | D-51, D-54 |
| Pengecek | Project Engineer atau Project Leader dari Project Group customer itu | D-48, D-54 |
| Dokumen kontrol | PPAP Staff dan PPAP Engineer, nanti bisa ditambah DCC | D-60, D-68 |
| Admin | Akun admin terpisah yang hanya memegang hak admin, saat ini dipegang PPAP Staff dan ditautkan ke pemegangnya | R-SEC-10, R-SEC-13, D-72, D-74 |
| Viewer | QA, QC, internal audit, approver (NPI Manager, QA QC Manager, Advisor, Factory Manager), DCC, tim WI, dan ECN & SAP Coordinator | D-41, D-70 |
| Petugas migrasi | PPAP Staff dan PPAP Engineer, untuk saat ini | R-MIG-06, D-69 |
| Penjadwal | Proses waktu di dalam sistem, untuk pengingat | R-NUM-04 |
| Microsoft Excel | Sistem lain di komputer pengguna | R-MIG-02, R-PRT-01 |
| Aplikasi internal lain | Aplikasi yang digabung lewat API (masa depan) | D-61, bagian 16 |
| Customer user | Akun customer (masa depan) | R-POR-04 |

Pengecek dan approver menandatangani cetakan di kertas, jadi tanda tangan itu tidak menjadi use case. Pengecek hanya muncul di UC-85 (masa depan). Approver tidak punya use case sendiri: mereka menandatangani di kertas dan memakai akun Viewer untuk UC-56 dan UC-57 (R-APR-13, D-70).

## A. Akses dan pengaturan

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-01 | Masuk dan keluar sistem | Pengguna internal | | Login lokal, sesi, semua halaman butuh login. | R-SEC-01 | M1.1 | MVP |
| UC-02 | Kelola pengguna, peran, dan jabatan | Admin | | Akun, peran, dan jabatan yang dipegang, termasuk rangkap jabatan. Admin juga mengubah isi peran (hak akses), sesuai peta pengguna. | R-SEC-10, D-66, D-115 | M1.2, M1.8, M8.1 | MVP |
| UC-03 | Kelola customer, Project Group, group model, model, dan part number | Admin | | Master data dengan kode customer, alokasi ke Project Group, dan satu group per model. | R-DOC-08, D-42, D-54 | M1.4 | MVP |
| UC-04 | Kelola master proses | PPAP Engineer | | ID tetap, nama tampilan, dan alias. | R-DOC-04 | M1.5, D-64 | MVP |
| UC-05 | Kelola jenis dokumen, jenis perubahan, dan sumber perubahan | Admin | | Awalan SH, SF, SC, kebijakan approval per jenis perubahan, dan pemicu PP, customer claim, ECR. | R-NUM-02, R-TPL-10, R-CHG-06 | M1.6 | MVP |
| UC-06 | Kelola rute approval | Admin | | Slot berurutan, jabatan per slot, judul cetak, dan rute default per customer. | R-APR-08, R-APR-09, R-APR-17 | M1.8 | MVP |
| UC-07 | Kelola pengaturan sistem | Admin | | Saklar QR, tanggal tanda tangan per slot, aturan tanggal berlaku, dan mode migrasi. | R-APR-14, R-APR-16, R-REV-08, R-MIG-06 | M1.8, M6.5, M7.5 | MVP |
| UC-08 | Lihat audit log | Admin | | Menelusuri siapa melakukan apa dan kapan. Hanya lewat akun admin. Setiap entri tindakan admin menampilkan akun dan pemegangnya. | R-SEC-03, D-72, D-74, D-75 | M1.3 | MVP |
| UC-09 | Kelola master dokumen terkait | Admin | | Nomor dan judul WI, SOP, QEP yang dirujuk CP. | R-DOC-11 | M4.1 | Usulan |

## B. Penomoran dan template

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-10 | Daftarkan nomor dokumen | Penyusun | | Nomor dialokasikan di server tanpa bentrok; kode customer di nomor dicek. | R-NUM-01, R-NUM-03, R-POR-01 | M2.1, M2.2 | MVP |
| UC-11 | Daftarkan nomor sebagai set | Penyusun | | PFD, PFMEA, dan CP mendapat nomor urut yang sama. | R-NUM-05 | M2.4 | Usulan |
| UC-12 | Cari dan ekspor register nomor | Pengguna internal | | Master document list dengan filter status. | R-NUM-04 | M2.3 | Usulan |
| UC-13 | Ubah status nomor | Dokumen kontrol | | Void atau Obsolete. Nomor tidak pernah dihapus atau dipakai ulang. | R-NUM-04 | M2.3 | Usulan |
| UC-14 | Impor nomor lama | Dokumen kontrol | Admin | Nomor lama masuk apa adanya, beserta laporan lubang, ganda, dan set tidak sejajar. | R-NUM-06 | M2.5 | Usulan |
| UC-15 | Ingatkan nomor Reserved yang terlalu lama | Penjadwal | Penyusun | Pengingat untuk nomor yang dipesan tetapi tidak dipakai. | R-NUM-04 | M2.3 | Usulan |
| UC-20 | Kelola format template | Admin | PPAP Engineer | Kolom ber-ID tetap, metode risiko, tabel skala, simbol, dan layout ekspor. | R-TPL-01, R-TPL-02, R-TPL-11 | M3.1 | MVP |
| UC-21 | Kelola process template | PPAP Engineer | | Proses, sub-proses, dan poin di empat tingkat: umum, customer, group model, model. | R-TPL-03, D-39 | M3.2, M3.3, D-64 | MVP |
| UC-22 | Terbitkan versi template | PPAP Engineer | | Versi draft menjadi terbit, dan tidak bisa diubah lagi. | R-TPL-05 | M3.4, D-64 | Usulan |
| UC-23 | Propagasikan versi template | PPAP Engineer | | Pratinjau dampak, pilih cakupan, tangani konflik lokal. Hasilnya sebuah perubahan, bukan edit langsung. | R-TPL-06, R-TPL-08, R-TPL-09 | M5.8, D-64 | MVP |

## C. Dokumen dan perakitan

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-30 | Rakit dokumen baru dari template | Penyusun | | Header diisi manual, isi dirakit dengan kotak centang; satu perakitan menghasilkan baris PFD, PFMEA, dan CP. | R-CMP-01, R-CMP-02 | M4.1, M4.5 | MVP |
| UC-31 | Mulai dari komposisi model lain | Penyusun | | Salinan menjadi dokumen mandiri. | R-CMP-05 | M4.7 | Usulan |
| UC-32 | Edit baris di editor tabel | Penyusun | | Edit sel, tambah dan hapus baris, dengan penguncian optimistik. | R-DOC-01, R-DOC-07 | M4.2, M4.3 | MVP |
| UC-33 | Tempel blok sel dari Excel | Penyusun | Microsoft Excel | Pratinjau dan pemetaan kolom; baris janggal ditandai. | R-MIG-02 | M4.4 | MVP |
| UC-34 | Atur urutan proses dan nomor operasi | Penyusun | | Drag and drop, nomor otomatis yang bisa diubah. | R-CMP-01 | M4.6 | MVP |
| UC-35 | Tandai perbedaan lokal atau putus tautan | Penyusun | | Putus tautan butuh alasan. | R-CMP-04 | M4.8 | Usulan |
| UC-36 | Cek PFMEA dan CP berdampingan | PPAP Engineer |  | Tampilan per proses, beserta tanda dicek oleh siapa dan kapan. | R-DOC-06, D-65 | M4.9 | MVP |
| UC-37 | Lihat diagram PFD | Pengguna internal | | Diagram dibentuk dari daftar proses, simbol, dan disposisi NG. | R-DOC-09 | M4.10 | Opsional |
| UC-38 | Tautkan dokumen yang saling bergantung | Penyusun | | Misalnya ESD control plan SC-ESD-K001, dengan titik masuk dan keluar. | R-DOC-10 | M4.1 | Usulan |

## D. Perubahan dan revisi

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-40 | Ajukan permintaan perubahan | Penyusun | | Sumber, jenis, alasan, rujukan, lampiran, dan dokumen terdampak. | R-CHG-01, R-CHG-02 | M5.1, M5.2 | MVP |
| UC-41 | Verifikasi permintaan perubahan | Verifikator | | Terima atau tolak dengan catatan wajib. Pemohon tidak boleh memverifikasi; verifikator tercatat. | R-CHG-02, D-51 | M5.3 | MVP |
| UC-42 | Isi daftar periksa dampak | Penyusun | | Dokumen terkait yang terdampak, termasuk WI, beserta keputusan dan alasannya. | R-CHG-05 | M5.7 | Usulan |
| UC-43 | Kerjakan draft revisi | Penyusun | | Setiap dokumen terdampak mendapat draft; revisi yang berlaku tidak berubah. | R-CHG-03 | M5.4 | MVP |
| UC-44 | Jalankan yokoten | Penyusun | | Pilih target per customer, group model, model, proses, atau teks; papan status per model. | R-CHG-04, D-40 | M5.9 | MVP |
| UC-45 | Putuskan yokoten per dokumen target | Penyusun | | Terapkan, sesuaikan, atau tidak berlaku dengan alasan. | R-CHG-04 | M5.9 | MVP |
| UC-46 | Terapkan perubahan surut SQAM | Penyusun | | Semua dokumen aktif mendapat draft baru; revisi terbit tidak ditulis ulang. | R-TPL-07 | M5.10 | MVP |
| UC-47 | Lihat riwayat dan diff revisi | Pengguna internal | | Per dokumen dan per proses, terhubung ke perubahan dan lampirannya. | R-REV-03 | M5.6 | MVP |

## E. Cetak, tanda tangan, dan rilis

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-50 | Cetak untuk tanda tangan | Penyusun | | Revisi dikunci, tanggal rilis diisi otomatis, kotak tanda tangan dibentuk dari rute. | R-REV-06, R-REV-08, R-APR-01, D-59 | M6.1, M6.2, M6.3 | MVP |
| UC-51 | Catat dokumen dikembalikan | Penyusun | | Approver minta perubahan di kertas; revisi kembali ke draft. | D-30, R-REV-06 | M5.5a (usulan); bila M5.5a ditolak, M5.4 | MVP |
| UC-52 | Catat rilis | Dokumen kontrol | | Unggah scan, konfirmasi slot sesuai urutan rute. Pencatat bukan penyusun. | R-APR-02, R-APR-10 | M6.7 | MVP |
| UC-53 | Cetak salinan terkendali | Dokumen kontrol | | Watermark CONTROLLED COPY, beserta catatan cetak. | R-PRT-04 | M6.4, M6.6 | Usulan |
| UC-54 | Ekspor Excel dan PDF | Pengguna internal | Microsoft Excel | Excel A3 dan A4, PDF yang sama rapinya. | R-PRT-01 | M6.2, M6.3 | MVP |
| UC-55 | Lihat salinan yang harus ditarik | Dokumen kontrol | | Daftar dari catatan cetak saat revisi baru rilis. | R-PRT-06 | M6.6 | Usulan |
| UC-56 | Buka halaman verifikasi | Viewer | | Status berlaku, digantikan, atau dibatalkan. Lewat QR bila QR menyala. | R-APR-03, R-APR-16, D-62 | M6.5 | Saklar |
| UC-57 | Baca dokumen rilis | Viewer | | Hanya baca. | R-APR-06, D-41 | M1.2 (hak L-1), M5.5 | MVP |

## F. Migrasi dan ukuran

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-60 | Masukkan dokumen lama sebagai baseline | Petugas migrasi | | Nomor asli, revisi saat ini, dan riwayat revisi sebagai teks. | R-MIG-02, R-MIG-03, R-REV-05 | M7.1, M7.2 | MVP |
| UC-61 | Bandingkan dengan Excel lama | Petugas migrasi | Microsoft Excel | Excel lama hanya untuk dibandingkan dengan hasil perakitan. | R-MIG-04 | M7.3 | Usulan |
| UC-62 | Tautkan baris hasil tempel ke template | Petugas migrasi | | Sistem mengusulkan pasangan, manusia mengonfirmasi. | R-MIG-05 | M7.4 | MVP |
| UC-63 | Verifikasi terhadap hardcopy bertanda tangan | Petugas migrasi | | Status Released (migrated). | R-MIG-06 | M7.5 | Usulan |
| UC-64 | Lihat dasbor ukuran | Pengguna internal | | Persentase baris yang mengikuti template, status dokumen dan yokoten, dan waktu revisi. | Brief bagian 2 | M7.7 | MVP |

## G. Masa depan (latihan tahap 9 dan 10)

| ID | Use case | Aktor utama | Aktor lain | Ringkasan | Rujukan | Milestone | Status |
|---|---|---|---|---|---|---|---|
| UC-80 | Kelola service account dan token | Admin | | Scope, customer yang diizinkan, dan masa berlaku. | R-API-04 | M9.1 | Masa depan |
| UC-81 | Baca data rilis lewat API | Aplikasi internal lain | | Data rilis, paginasi, filter updated_since. | R-API-12 | M9.3 | Masa depan |
| UC-82 | Kirim usulan perubahan lewat API | Aplikasi internal lain | Verifikator | Menjadi permintaan perubahan yang menunggu verifikasi manusia. | R-API-07 | M9.4 | Masa depan |
| UC-83 | Ekspor data secara asinkron | Aplikasi internal lain | | Job ekspor, polling, unduh. | R-API-14 | M9.5 | Masa depan |
| UC-84 | Lihat dan unduh dokumen di portal | Customer user | | Hanya dokumen rilis milik customer itu. | R-POR-08 | M10.2, M10.6 | Masa depan |
| UC-85 | Beri izin tampil | Penyusun | Pengecek | Konfirmasi per peran, terikat content hash. | R-PUB-03 | M10.4 | Masa depan |
| UC-86 | Kelola kebijakan publikasi customer | Admin | Dokumen kontrol | Sumber file dan tahap paling awal. | R-PUB-02 | M10.3 | Masa depan |

## Yang masih menunggu jawaban

Tidak ada pertanyaan terbuka khusus untuk U3. Temuan dari kode yang menyentuh use case dicatat di `docs/evaluasi-desain.md`.
