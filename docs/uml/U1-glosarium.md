# U1 · Glosarium

- Paket UML: draft 0.14, 4 November 2026 (draft 0.13 dan 0.12: 3 November 2026; draft 0.11, 0.10, 0.9, 0.8, dan 0.7: 31, 31, 30, 30, dan 27 Oktober 2026). Draft 0.12 menambah istilah dari desain UI 2 November 2026 dan temuan register nomor 3 November 2026, bertanda UX-xx (`usulan-desain-ui.md`). Draft 0.13 mencatat simbol Revision yang dipasang sistem (migrasi 00013). Draft 0.14 menambah kolom terisi otomatis dan aturan cetak tanda revisi (D-121 sampai D-123). Draft 0.15 (8 November 2026) menambah nama kelompok menu (D-127), cakupan poin yang sudah diputuskan, dan varian poin (D-129).
- Sumber: brief desain v1.22, termasuk temuan dari dokumen TR-020 (brief bagian 21). Nama jabatan mengikuti bagan organisasi NPI (update 22 Januari 2026). Nama orang tidak dicantumkan. Setiap istilah merujuk ke ID di brief (R-UML-01).
- Bahasa: penjelasan dalam bahasa Indonesia. Nama jabatan, nama standar, dan tahap tanda tangan memakai istilah yang sama dengan brief (R-UML-05, usulan).
- Kolom **Istilah layar (Inggris)** (E-19, D-98): istilah yang dipakai di layar berbahasa Inggris dan di katalog teks `web/src/lib/i18n/en.ts`. Layar juga punya bahasa Indonesia (D-128, menggantikan D-96); katalognya `web/src/lib/i18n/id.ts` memakai kolom **Istilah**. Istilah baru di layar ditambahkan ke sini lebih dulu.
- Kolom **Cek WI**:
  - **Belum**: istilah domain yang harus dicocokkan dengan WI dan prosedur yang berlaku. Diisi pemilik.
  - **Tidak perlu**: istilah sistem yang tidak muncul di WI.
- Tanda di kolom Arti:
  - **(usulan)**: aturannya masih [PROPOSED] di brief.
  - **(masa depan)**: hanya untuk latihan tahap 9 dan 10 (R-UML-06).
  - **Q-xx**: masih menunggu jawaban pemilik (brief bagian 12). Bagian ini tidak diisi dengan tebakan.

## 1. Dokumen dan standar

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| PFD | PFD | Process Flow Diagram: diagram alur proses produksi satu model. Job description di bagan organisasi menyebutnya PFC; sistem dan prosedur memakai PFD. | Diagram alir proses, PFC | R-DOC-01, R-DOC-04, D-57 | Belum |
| PFMEA | PFMEA | Process Failure Mode and Effects Analysis: analisis mode kegagalan proses beserta risikonya. | | R-DOC-01 | Belum |
| CP | CP | Control Plan: rencana kontrol per proses. Cover SIIX menyebutnya Process Control Plan (PCP). | Control Plan, PCP | R-DOC-01 | Belum |
| Set (satu set) | Set | PFD, PFMEA, dan CP satu model yang berbagi satu nomor urut. | | R-NUM-05 | Belum |
| AIAG 4th | AIAG 4th | Buku panduan FMEA AIAG edisi ke-4, berbasis RPN. | | R-TPL-02 | Belum |
| AIAG-VDA | AIAG-VDA | Buku panduan FMEA gabungan AIAG dan VDA, berbasis Action Priority. Sistem saat ini hanya kompatibel dengan form-nya. Struktur penuhnya ditunda. | | R-TPL-02, R-DOC-03 | Belum |
| RPN / AP | RPN / AP | Risk Priority Number / Action Priority. | | R-TPL-02 | Belum |
| SQAM | SQAM | Supplier Quality Assurance Manual milik customer. Bisa mengatur format dan aturan dokumen, dan perubahannya sering berlaku surut. | | R-TPL-07 | Belum |
| PPAP | PPAP | Production Part Approval Process. | | D-34 | Belum |
| 4M change | 4M change | Perubahan Man, Machine, Material, atau Method. Persetujuan customer untuk perubahan ini diurus di luar sistem. | Perubahan 4M | R-CHG-01, D-34 | Belum |
| WI | WI | Work Instruction: dokumen di luar sistem yang bisa terdampak perubahan. Istilah di sistem harus sama dengan WI. | Instruksi kerja | R-CHG-05, R-UML-05 | Belum |
| Standar inspeksi | Inspection standard | Dokumen di luar sistem yang bisa terdampak perubahan. | Inspection standard | R-CHG-05 | Belum |

## 2. Isi dokumen dan template

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Proses | Process | Langkah produksi, misalnya IQC. Punya ID tetap; namanya hanya tampilan dan boleh punya alias. | | R-DOC-04 | Belum |
| Sub-proses | Sub-process | Varian dari satu proses, misalnya IQC General, IQC MSD, IQC PCB, IQC Wireharness, IQC Connector. ID-nya tetap dan milik satu proses master; namanya disimpan per versi process template (ASUMSI E-33). | | Brief bagian 3, E-33 | Belum |
| Poin | Point | Satu butir setingkat baris di dalam sub-proses, untuk PFD, PFMEA, atau CP, dengan ID tetap. Untuk sementara berisi satu baris teks (ASUMSI E-73). | | R-TPL-04 | Belum |
| IQC | IQC | Incoming Quality Control. | | Brief bagian 3 | Belum |
| Nomor operasi | Operation number | Nomor urut proses di dokumen. Dibuat otomatis dan bisa diubah. | | R-CMP-01 | Belum |
| Baris dokumen | Row | Satu baris lembar kerja PFD, PFMEA, atau CP. ID-nya tetap lintas revisi. | Row | R-DOC-01, R-DOC-02 | Tidak perlu |
| Format template | Format template | Template metodologi: kolom, metode risiko, tabel skala, field header, dan layout ekspor. Berversi. | | R-TPL-01, R-TPL-02 | Tidak perlu |
| Process template | Process template | Pustaka proses, sub-proses, dan poin. Satu template per proses (ASUMSI E-72). Berversi. | Template proses, pustaka proses | R-TPL-01 | Tidak perlu |
| Sub-proses dan poin tersembunyi | Hidden | Sub-proses atau poin yang tidak dipakai lagi. Tidak dihapus: ID-nya tetap, jadi rujukan dari dokumen tetap utuh. | | R-TPL-04 | Tidak perlu |
| Cakupan template | Template scope | Tingkat berlakunya process template: umum, customer, group model, dan model. Yang paling spesifik menang: model, lalu group model, lalu customer, lalu umum. | Scope | R-TPL-03, D-39 | Tidak perlu |
| Template General | Template General | Process template umum yang tidak terikat customer atau model. | Template umum | R-TPL-03 | Belum |
| Group model | Model group | Kumpulan model dalam satu customer yang berbagi isi template. Satu model paling banyak masuk satu group. Bisa dipilih sebagai target yokoten. | Model group | D-39, D-40, D-42, D-87 | Belum |
| Model | Model | Produk customer, misalnya nama di kolom Model pada PFMEA. Milik satu customer dan paling banyak satu group model. | | R-DOC-08, D-42 | Belum |
| Part number | Part number | Nomor part customer, milik satu model. | | R-DOC-08 | Belum |
| Kolom | Column | Satu kolom baris dokumen di format template. ID-nya tetap; labelnya hanya tampilan, jadi bisa diganti tanpa merusak data. | Column | R-TPL-02 | Tidak perlu |
| Field header | Header field | Isian di bagian header dokumen yang diketik tangan, misalnya Core Team atau Key Date. ID-nya tetap, seperti kolom. | | R-TPL-01, R-CMP-01 | Tidak perlu |
| Kolom terisi otomatis | Filled from the row | Kolom yang tampil dan tercetak seperti kolom lain, tetapi isinya tidak diketik: nomor operasi dan nama proses dari baris (D-123), Process Flow dari langkah PFD (D-122). RPN juga tidak diketik karena dihitung. | | D-122, D-123, R-TPL-02 | Tidak perlu |
| Kolom tersembunyi | Hidden | Kolom atau field header yang tidak dipakai lagi. Tidak dihapus: datanya tetap tersimpan dan tetap sah. | | R-TPL-02 | Tidak perlu |
| Metode risiko | Risk method | RPN (AIAG 4th), AP (AIAG-VDA), atau tanpa metode risiko (PFD dan CP). Tetap sejak versi format template dibuat (ASUMSI E-70). | | R-TPL-02 | Tidak perlu |
| Tabel skala | Rating tables | Kriteria nilai 1 sampai 10 untuk severity, occurrence, dan detection di satu versi format template. Teksnya diisi dari form yang dipakai. | | R-TPL-01 | Belum |
| Nilai sekarang dan hasil tindakan | Current, action results | Dua set rating dan RPN di PFMEA AIAG 4th: nilai sekarang, dan nilai sesudah tindakan (kolom Action Results). | Action Results | F-10 | Belum |
| Aturan sorot | Highlight rule | Tanda untuk RPN di atas ambang atau severity paling sedikit ambang, misalnya RPN di atas 100 atau severity 8 (ASUMSI E-71). Hanya menandai, tidak memblokir. (usulan) | Aturan aksi | R-TPL-11, R-DOC-06 | Belum |
| Perakitan | Composition | Menyusun dokumen dari process template dengan kotak centang dan drag and drop. | Composition | R-CMP-01 | Tidak perlu |
| Poin dikecualikan | Excluded point | Poin template yang sengaja tidak dipakai satu model. Tetap tercatat, tidak dihapus diam-diam. (usulan) | Excluded point | R-CMP-03 | Tidak perlu |
| Perbedaan lokal | Local override | Baris yang diubah hanya untuk satu dokumen dan ditandai berbeda dari template. (usulan) | Local override | R-CMP-04 | Tidak perlu |
| Putus tautan | Detach | Dokumen atau baris yang dilepas dari tautan template, dengan alasan. (usulan) | Detach | R-CMP-04 | Tidak perlu |
| Mode tautan template | Follow, Pinned, Detached | Hubungan dokumen dengan template: ikuti, kunci versi, atau lepas. (usulan) | Follow, pinned, detached | R-TPL-08 | Tidak perlu |
| Tautan PFMEA ke CP | Linked counterpart | Hubungan opsional antara baris PFMEA dan baris CP, dibuat manual atau dari template. | Linked counterpart | R-DOC-05, R-TPL-04 | Tidak perlu |
| Simbol alur PFD | Flow symbol | Bentuk langkah di PFD: process, alternate process, decision, connector, merge, dan document. (usulan) | | R-DOC-09 | Belum |
| Disposisi NG | NG disposition | Tujuan produk NG dari langkah decision: scrap, NCP, atau rework. (usulan) | | R-DOC-09 | Belum |
| NCP | NCP | Nonconforming product, salah satu disposisi NG. | | R-DOC-09 | Belum |
| Simbol karakteristik | Characteristic symbol | Simbol di PFMEA dan CP. Saat ini: Critical Process, Revision, Important Character (IC), dan Regulation. Disimpan di format template per customer. Revision dipasang sistem pada baris yang berubah dan tidak dipilih tangan di kolom simbol (R-REV-09, dibangun di migrasi 00013). (usulan) | | R-TPL-11, R-REV-09 | Belum |
| Simbol internal | Internal symbol | Simbol milik pabrik yang berlaku untuk semua customer: Critical Process (default) dan Revision (dipasang otomatis pada baris yang berubah). (usulan) | | R-REV-09, UX-03 | Belum |
| Simbol customer | Customer symbol | Simbol yang bentuk dan artinya ditentukan customer, misalnya Important Character dan Regulation. Dicatat sebagai nama teks dan gambar yang diunggah, berversi, dan tidak dihapus. (usulan) | | R-TPL-11, UX-03 | Belum |
| Cakupan poin | Use point for | Untuk siapa poin biasanya dipakai: umum, customer, group model, atau model. Ditetapkan saat poin dibuat dan tidak berubah (D-129, M3.3). Pilihan cakupan saat menyusun dokumen, termasuk usulan ke Pengelola template, masih usulan sampai M4.5 (UX-01). | Point scope | R-TPL-03, D-39, D-129, UX-01 | Tidak perlu |
| Varian poin | Variant | Teks pengganti satu poin untuk cakupan yang lebih sempit, misalnya laser type untuk satu model walaupun poinnya biasanya umum. Untuk satu model, varian yang paling spesifik menang: model, lalu group model, lalu customer, lalu poin umum. Varian disembunyikan, tidak pernah dihapus (D-129, M3.3). | | R-TPL-03, D-129 | Tidak perlu |
| Gambar di sel | Picture | Gambar di dalam satu sel CP, misalnya drawing PCB untuk item cek IQC PCB. Ikut tercetak di dalam sel. (usulan) | | UX-02 | Belum |
| Tabel spesifikasi | Spec table | Tabel kecil di dalam satu sel CP, misalnya parameter ROM writing (program, checksum) atau item uji FCT dengan spesifikasi dan satuannya. Bisa ditempel dari Excel. (usulan) | Spec table | UX-02 | Belum |
| Item cek khusus CP | CP-only check item | Baris CP yang tidak punya pasangan proses di PFD atau PFMEA, misalnya IPQC 1st Printing Inspection. (usulan) | | UX-04 | Belum |
| Cek sinkronisasi | Sync check | Tampilan yang menjajarkan proses PFD, PFMEA, dan CP menurut nomor operasi dan ID proses master, dengan hasil per proses: sama, nama berbeda, CP dipecah, item cek khusus CP, atau tidak ada di salah satu dokumen. Keputusan tetap di tangan manusia. (usulan) | Sync check | R-DOC-06, D-65, UX-04 | Tidak perlu |
| Workbook | Workbook | Satu layar untuk satu set dokumen satu model, dengan tab sheet di bawah: Cover, PFD, PFMEA, Control Plan, dan Check, meniru file Excel yang dipakai sekarang. (usulan) | | UX-05 | Tidak perlu |
| Core Team | Core team | Field header PFMEA dan CP yang berisi anggota tim beserta departemennya. | | Brief bagian 21, F-10, F-11 | Belum |
| Dokumen terkait | Relevant document | Nomor dan judul WI, SOP, QEP, dan dokumen lain yang dirujuk di kolom Relevant Document CP. Diambil dari master supaya sama dengan sumbernya. (usulan) | Relevant document | R-DOC-11 | Belum |
| Dokumen saling bergantung | Interdependent document | Dokumen terpisah yang dirujuk sebagai pengganti langkah yang diulang, misalnya ESD control plan SC-ESD-K001. (usulan) | Interdependent document | R-DOC-10 | Belum |

## 3. Perubahan dan revisi

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Perubahan | Change | Satu entitas untuk semua sumber perubahan. Berisi sumber, jenis, alasan, rujukan, lampiran, dokumen terdampak, dan daftar periksa dampak. | Change | R-CHG-01 | Tidak perlu |
| Permintaan perubahan | Revision request | Perubahan yang diajukan manual oleh Penyusun (PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer) dengan dokumen pendukung. | Revision request | R-CHG-02 | Belum |
| Sumber perubahan | Change source | Asal perubahan: permintaan manual, yokoten, propagasi template, atau perubahan surut SQAM. | | R-CHG-01 | Belum |
| Pemicu perubahan | Change source (E-36) | Pemicu yang dipakai sekarang: PP (pre-production), customer claim, ECR, dan temuan audit. (usulan) | | R-CHG-06 | Belum |
| ECR | ECR | Engineering Change Request: permintaan perubahan yang diajukan ke customer. Salah satu pemicu perubahan. | | R-CHG-06 | Belum |
| Yokoten | Yokoten | Penerapan satu perubahan atau pelajaran dari satu model, proses, atau dokumen ke yang lain. Target dipilih per customer, group model, model, proses, atau pencarian teks. | Horizontal deployment | R-CHG-04, D-40 | Belum |
| Propagasi template | Template propagation | Versi template baru yang diteruskan ke dokumen pengikutnya sebagai draft revisi. Tidak pernah mengubah dokumen secara langsung. | Template propagation | R-TPL-06, R-TPL-08 | Tidak perlu |
| Perubahan surut SQAM | Retroactive SQAM change | Versi SQAM baru yang berlaku untuk dokumen aktif. Dibuat sebagai revisi baru; revisi yang sudah terbit tidak pernah ditulis ulang. | Retroactive change | R-TPL-07 | Belum |
| Jenis perubahan | Change type | Isi, tata letak saja, atau administratif. Menentukan jalur approval. | Change type | R-TPL-10 | Belum |
| Daftar periksa dampak | Impact checklist | Catatan per perubahan tentang dokumen terkait yang terdampak (PFD, PFMEA, CP, WI, standar inspeksi), keputusannya, dan alasannya. (usulan) | Impact checklist | R-CHG-05 | Belum |
| Verifikasi | Verification | Penerimaan atau penolakan permintaan perubahan dengan catatan wajib, oleh Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller. Siapa yang memverifikasi tercatat. | | R-CHG-02, D-51, D-54 | Belum |
| Revisi | Revision | Versi dokumen bernomor bulat, ditampilkan -00, -01, dan seterusnya. Dokumen hasil migrasi mulai dari revisi saat ini, misalnya -04. | | R-REV-01 | Belum |
| Draft | Draft | Revisi yang masih bisa diedit dan belum berlaku. | | R-CHG-03 | Belum |
| Menunggu approval | Awaiting approval | Revisi yang sudah dicetak untuk tanda tangan dan dikunci. Edit sesudahnya membatalkan cetakan itu. (usulan) | Awaiting approval | R-REV-06 | Tidak perlu |
| Dikembalikan | Returned | Approver minta perubahan di kertas. Penyusun mencatatnya, dan revisi kembali ke draft. | Return | D-30 | Belum |
| Released | Released | Revisi yang berlaku. Isinya dibekukan sebagai snapshot dan tidak bisa diubah. | Rilis | R-REV-02 | Belum |
| Obsolete | Obsolete | Revisi yang sudah digantikan. Tidak pernah berubah. | | R-REV-02 | Belum |
| Snapshot | Snapshot | Isi revisi yang dibekukan saat rilis, beserta hash SHA-256. | | R-REV-02 | Tidak perlu |
| Hash | Hash | Sidik jari isi. Ada tiga: snapshot_hash, customer_content_hash, dan file_hash. (usulan) | | R-REV-07 | Tidak perlu |
| Diff | Diff | Perbandingan baris dan sel antar revisi. | | R-REV-03 | Tidak perlu |
| Cakupan yokoten | Yokoten coverage | Persentase dokumen target satu yokoten yang revisinya sudah rilis. Dokumen yang diputuskan tidak berlaku dihitung selesai bila alasannya tercatat. Ukuran utama QCC. (usulan) | | R-CHG-04, UX-07 | Belum |
| Riwayat revisi | Revision history | Tabel revisi otomatis di cetakan. Riwayat sebelum baseline migrasi disimpan sebagai teks dari tabel Excel lama. (usulan) | Revision history | R-REV-04, R-REV-05 | Belum |

## 4. Penomoran

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Nomor dokumen | Document number | Nomor terdaftar untuk PFD, PFMEA, atau CP per customer dan jenis dokumen, misalnya SC-KT-001. Disimpan sebagai teks dan tidak pernah dipakai ulang. | | R-NUM-01, R-NUM-02 | Belum |
| Awalan | Prefix | SH untuk PFD, SF untuk PFMEA, dan SC untuk CP (dokumen TR-020), sebagai master data yang bisa diedit. | Prefix | R-NUM-01, R-NUM-02 | Belum |
| Kode customer | Customer code | Kode singkat customer di nomor, misalnya KT. Harus kode customer itu, sekarang atau dulu: nomor lama tetap memakai kode lamanya. (usulan) | | R-POR-01, D-88, E-11 | Belum |
| Register nomor | Number register | Daftar semua nomor beserta statusnya. Bisa diekspor sebagai master document list. (usulan) | Master document list | R-NUM-04 | Belum |
| Status nomor | Number status | Reserved, In use, Released, Obsolete, Void, ditambah Skipped untuk nomor yang dilewati saat daftar sebagai set (ASUMSI E-64). Nomor tidak pernah dihapus; yang berubah hanya statusnya. (usulan) | | R-NUM-04 | Belum |
| Daftarkan sebagai set | Register as a set | Mengambil nomor bebas berikutnya di ketiga jenis dokumen supaya nomornya sejajar. Nomor yang dilewati penghitung yang tertinggal dicatat Skipped. (usulan) | | R-NUM-05 | Tidak perlu |
| Nomor dilewati | Skipped | Nomor yang dilewati saat daftar sebagai set. Tetap di register dan tidak pernah diberikan lagi, supaya register tanpa lubang. (ASUMSI E-64) | | R-NUM-03, R-NUM-05 | Tidak perlu |
| Pola nomor | Number pattern | Susunan nomor dari {prefix}, {customer}, dan {seq} dengan pemisah, misalnya {prefix}-{customer}-{seq} untuk SC-KT-001. Satu pola untuk semua jenis dokumen dan customer (ASUMSI E-61), dan hanya berlaku untuk nomor baru. | | R-NUM-02, D-89 | Tidak perlu |
| Nomor berjalan | Running number | Bagian urut di nomor: 001 sampai 999, lalu A001 sampai Z999. | DDD (WI SW-ZZ-000) | D-89, E-12 | Belum |
| Revisi awal | Initial revision | Revisi pertama dokumen yang memakai nomor itu, misalnya 4 (-04) untuk dokumen lama yang masuk di revisi saat ini. | | R-REV-01 | Tidak perlu |
| Digit revisi | Revision digits | Jumlah digit label revisi, saat ini 2 (-04). Diatur di master data. | | D-89 | Belum |
| Kunci idempoten | (tidak tampil di layar) | Kunci acak yang dikirim bersama permintaan pendaftaran. Permintaan yang diulang dengan kunci yang sama mendapat nomor yang sama, bukan nomor baru. Berlaku per akun (ASUMSI E-63). | Idempotency-Key | R-NUM-03 | Tidak perlu |
| Nomor lama | Legacy number | Nomor dari sistem penomoran sebelumnya, misalnya CP-KT-14-K001 atau PFD-HL-18-K001, yang di register Excel tercatat di kolom No. (Before). Disimpan sebagai rujukan yang bisa dicari, tidak menjadi nomor aktif. (usulan) | No. (Before) | R-NUM-06, UX-06 | Belum |
| Impor nomor lama | Import legacy numbers | Memasukkan nomor dari register Excel lama dengan teks aslinya. Penghitung naik ke nomor tertinggi; lubang, nomor ganda, dan set tidak sejajar hanya dilaporkan. (usulan) | | R-NUM-06, UC-14 | Tidak perlu |
| Pratinjau impor | Preview | Laporan impor tanpa menyimpan apa pun. | Dry run | R-NUM-06 | Tidak perlu |
| Lubang | Gaps | Nomor berjalan yang tidak ada di register, di antara 001 dan nomor tertinggi. Dilaporkan, tidak pernah diisi. | | R-NUM-06 | Tidak perlu |
| Set tidak sejajar | Sets not aligned | Nomor berjalan yang hanya dimiliki sebagian dari PFD, PFMEA, dan CP. Nomor Skipped dianggap tidak ada (ASUMSI E-65). | | R-NUM-06, RK-24 | Tidak perlu |
| Penghitung | Counter | Nomor terakhir per kode customer dan jenis dokumen, dinaikkan di dalam transaksi. | Counter | R-NUM-03, D-93 | Tidak perlu |
| Kode pabrik | Factory code | Huruf pertama awalan nomor, saat ini S (WI SW-ZZ-000). | Factory code | D-89 | Belum |
| Jenis dokumen | Document type | Huruf kedua awalan nomor: H untuk PFD, F untuk PFMEA, C untuk CP, ditambah jenis lain dari WI seperti W. Hanya PFD, PFMEA, dan CP yang dinomori sistem. | Document type | D-89 | Belum |
| Kode khusus | Special code | Kode di posisi kode customer untuk dokumen yang bukan milik satu customer: ZZ untuk General/Common dan HSE. Tidak pernah sama dengan kode customer. | Special code | D-89, D-114 | Belum |

## 5. Tanda tangan, approval, dan rilis

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Prepared | Prepared | Tahap tanda tangan penyusun: PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer. | | R-APR-07, D-63, D-67 | Belum |
| Checked | Checked | Tahap tanda tangan pengecek. Cukup satu orang: Project Engineer atau Project Leader dari Project Group customer itu. | | R-APR-07, D-48, D-54 | Belum |
| Approved | Approved | Tahap tanda tangan approver, berurutan: NPI Manager, lalu QA QC Manager, lalu slot GM bila ada di rute. | | R-APR-07, D-49 | Belum |
| Slot GM | GM slot | Slot approver terakhir. Saat ini diisi Advisor; nanti bisa diisi Factory Manager. Cover mencetak "General Mgr." dengan nama pemegangnya, mengikuti cover TR-020 (D-76). | | D-56, D-76 | Belum |
| Judul cetak slot | Print title | Tulisan di bawah nama pada setiap slot, terpisah dari nama jabatan. Mengikuti cover TR-020: Process. Eng, Engineer, NPI Manager, QA / QC Mgr., General Mgr. | | R-APR-17, D-76 | Belum |
| Rute approval | Approval route | Daftar slot tanda tangan berurutan untuk satu dokumen. Ada dua rute: sampai QA QC Manager dan sampai slot GM. Default-nya per customer, dan bisa diganti per dokumen dengan alasan. | Approval route | R-APR-08, R-APR-09, D-46, D-47 | Tidak perlu |
| Slot | Slot | Satu kotak tanda tangan di rute, dengan tahap dan jabatan yang boleh mengisinya. | | R-APR-08 | Tidak perlu |
| Tanggal tanda tangan | Sign date | Tanggal per slot. Disiapkan dengan saklar untuk mematikan dan menyembunyikannya, dan saat ini mati. | | R-APR-14, D-52 | Tidak perlu |
| Tanggal rilis | Release date | Tanggal dokumen dicetak untuk tanda tangan, diisi otomatis. Secara resmi tanggal rilis adalah tanggung jawab DCC. Saat ini satu-satunya tanggal yang dicatat. | Release date | D-52, D-53, D-59 | Belum |
| Tanggal berlaku | Effective date | Tanggal revisi mulai berlaku. Saat ini sama dengan tanggal rilis, dan aturannya harus bisa diubah nanti. | Effective date | D-59, R-REV-08 | Belum |
| Tanda tangan basah | Wet signature | Tanda tangan tangan di kertas. Approval saat ini manual. | Wet signature | R-APR-01, R-APR-04 | Belum |
| Pencatatan rilis | Release recording | Dokumen kontrol mencatat scan bertanda tangan, lalu status revisi menjadi Released. Pencatat bukan penyusun. Berbeda dari tanggal rilis. (usulan) | Release recording | R-APR-02 | Belum |
| Dokumen kontrol | Document control | Fungsi yang mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer, yang saling mencatat rilis dokumen yang disusun yang lain. Nanti bisa ditambah DCC. | Document control | R-APR-02, D-60, D-68 | Belum |
| Salinan terkendali | Controlled copy | Salinan cetak yang didistribusikan dan dilacak, dengan watermark CONTROLLED COPY. (usulan) | Controlled copy | R-PRT-04 | Belum |
| Salinan tidak terkendali | Uncontrolled copy | Salinan untuk referensi atau draft, dengan watermark UNCONTROLLED COPY. (usulan) | Uncontrolled copy | R-PRT-04 | Belum |
| Catatan cetak | Print log | Siapa mencetak, kapan, dan nomor salinannya. (usulan) | Print log | R-PRT-04 | Tidak perlu |
| QR | QR | Kode di cetakan yang membuka halaman verifikasi. Opsional dengan saklar, dan saat ini mati karena belum dipakai. | | D-62, R-APR-16 | Tidak perlu |
| Halaman verifikasi | Verification page | Halaman yang menampilkan status dokumen: berlaku, digantikan, atau dibatalkan. Bisa dibuka dari QR bila QR menyala, atau dari dalam sistem. (usulan) | | R-APR-03, R-APR-16 | Tidak perlu |
| Tanda revisi | Revision mark | Segitiga berisi nomor revisi di samping baris yang berubah pada revisi itu, dibuat otomatis dari diff. Yang tercetak hanya tanda revisi yang berlaku; tanda revisi lama hilang dari cetakan, tetapi perubahannya tetap bisa dilihat di riwayat dan diff (D-121). | Revision mark | R-REV-09, D-121 | Belum |
| Layout | Layout | Template Excel beserta pemetaan sel, terpisah dari data baris. | | R-PRT-02 | Tidak perlu |

## 6. Jabatan

Nama jabatan mengikuti bagan organisasi NPI (D-54). Kolom "Sebutan lain" memuat nama yang dipakai di brief versi awal.

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Pemegang jabatan | Holder | Orang yang tercatat memegang satu jabatan. Project Leader dan Project Engineer satu per Project Group; jabatan approver dan APQP/PPAP Controller satu orang; jabatan lain boleh beberapa orang. | Holder | D-82, D-115 | Belum |
| Project Group | Project group | Kelompok customer yang dipegang satu Project Leader dan satu Project Engineer. Setiap customer dialokasikan ke satu Project Group, saat ini GR1 sampai GR4 (Tabel 1 di bagan organisasi). | GR | D-54 | Belum |
| PPAP Engineer | PPAP Engineer | Level Leader. Menyusun dan memperbarui PFD, PFMEA, dan CP, mengelola master proses dan process template, dan untuk saat ini memberi tanda dicek PFMEA–CP. Juga memegang peran dokumen kontrol dan petugas migrasi. | | D-45, D-63, D-64, D-65, D-68, D-69 | Belum |
| PPAP Staff | PPAP Staff | Level SOP/Operator. Boleh menyusun dokumen, dan saat ini memegang peran dokumen kontrol dan petugas migrasi. Juga memegang akun admin terpisah, sehingga punya dua akun. | | D-60, D-63, D-69, D-72 | Belum |
| DCC | DCC | Dept Document Control. Secara resmi bertanggung jawab atas tanggal rilis, dan nanti bisa memegang peran dokumen kontrol. | Document Control Department | D-59, D-60 | Belum |
| Project Engineer | Project Engineer | Level Leader, satu per Project Group. Boleh menyusun dokumen (D-67), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). | Project Leader, Production Engineer (brief versi awal) | D-48, D-51, D-54 | Belum |
| Project Leader | Project Leader | Level Supervisor, satu per Project Group. Boleh menyusun dokumen (D-63), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). | Project Leader GR, Project Leader Supervisor, project supervisor (brief versi awal) | D-48, D-51, D-54 | Belum |
| APQP/PPAP Controller | APQP/PPAP Controller | Level Supervisor, satu orang, saat ini dirangkap Project Leader GR3. Boleh memverifikasi permintaan perubahan semua customer. | PPAP Supervisor | D-51, D-54 | Belum |
| NPI Manager | NPI Manager | Approver pertama. Menurut bagan, menyetujui dokumen resmi departemen NPI. Bila struktur berganti nama, jabatan ini bisa bernama PE Manager. | PE Manager (nama lama atau nama cadangan) | D-45, D-55 | Belum |
| QA QC Manager | QA QC Manager | Approver kedua. Rute bisa berakhir di sini. Jabatan ini di luar bagan NPI. | | D-45, D-46 | Belum |
| Advisor | Advisor | Level General Manager. Saat ini mengisi slot GM. | | D-56 | Belum |
| Factory Manager | Factory Manager | Level General Manager. Nanti bisa mengisi slot GM. | | D-56 | Belum |
| QA, QC, internal audit | QA, QC, internal audit | Pengguna halaman verifikasi QR dan dokumen rilis, dengan peran viewer. | | D-32, D-41 | Belum |

## 7. Peran sistem dan keamanan

Peran adalah hak akses di aplikasi, berbeda dari jabatan. Pemetaan jabatan ke peran mengikuti D-51, D-54, dan D-63 sampai D-66, dan peta lengkapnya ada di `U3-peta-pengguna.md`. Satu orang boleh memegang lebih dari satu jabatan (D-58).

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Level akses | Access level | Buat / Edit, Verifikasi, Lihat, dan Admin. | | R-SEC-10, D-66 | Tidak perlu |
| Hak akses | Permission | Satu aksi di sistem, misalnya mencatat rilis. Peran adalah kumpulan hak yang isinya diatur admin. | Permission | R-SEC-10, D-66 | Tidak perlu |
| Penyusun | Author | Peran PPAP Engineer, PPAP Staff, Project Leader, dan Project Engineer: menyusun dokumen, mendaftarkan nomor, mengajukan perubahan, dan mencetak untuk tanda tangan. Project Leader dan Project Engineer hanya untuk customer di Project Group sendiri. | Author | R-SEC-09, D-63, D-67, D-71, D-78 | Tidak perlu |
| Pengelola template | Template maintainer | Peran PPAP Engineer: mengelola master proses dan process template. | | D-64, D-78 | Tidak perlu |
| Pemeriksa PFMEA–CP | PFMEA–CP checker | Peran PPAP Engineer untuk saat ini: memberi tanda dicek di perbandingan PFMEA dan CP. | | D-65, R-DOC-06, D-78 | Tidak perlu |
| Verifikator | Verifier | Peran Project Engineer, Project Leader, dan APQP/PPAP Controller: memverifikasi permintaan perubahan. | Verifier | R-SEC-09, D-51, D-54, D-78 | Tidak perlu |
| Dokumen kontrol (peran) | Document control | Peran yang mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer. | Document control | R-SEC-09, R-APR-02, D-60, D-68 | Tidak perlu |
| Petugas migrasi | Migration | Peran yang memasukkan dokumen lama. Saat ini dipegang PPAP Staff dan PPAP Engineer. | Migration | R-MIG-06, D-69, D-78 | Tidak perlu |
| Viewer | Viewer | Hak dasar Lihat (L-1 sampai L-3) yang dimiliki semua akun yang login. Jabatan sampai level GM yang tidak menyusun, termasuk approver, DCC, QA, QC, internal audit, dan tim WI, hanya memegang ini. | | R-APR-06, D-41, D-70 | Tidak perlu |
| Admin | Admin | Peran akun admin: mengelola pengguna, hak akses, master data, rute approval, dan pengaturan, serta melihat audit log (A-1 sampai A-4), ditambah hak dasar Lihat. Tidak pernah memegang peran kerja. | | R-SEC-10, R-SEC-13, D-72, D-120 | Tidak perlu |
| Approver (peran) | Approver | Belum dibuat. Disiapkan untuk approval di aplikasi nanti (R-APR-04); selama approval di kertas, approver memakai hak dasar Lihat. | Approver | R-APR-13, D-70, D-78 | Tidak perlu |
| Orang | Person | Seseorang dengan nama dan nomor karyawan opsional. Jabatan dipegang orang, dan aturan pemisahan tugas membandingkan orang, bukan akun. | Person | D-79 | Tidak perlu |
| Action Priority (AP) | Action Priority | Prioritas tindakan PFMEA AIAG-VDA: H (tinggi), M (sedang), atau L (rendah). Dibaca dari tabel AP menurut Severity, Occurrence, dan Detection, tidak dihitung dengan perkalian seperti RPN. Tabelnya disimpan di versi format template. | AP | D-134 | Belum |
| Kata sandi | Password | Kata rahasia untuk login. Minimal 8 karakter (D-132). Kata sandi sementara dari admin tampil satu kali, disalin otomatis ke clipboard, dan wajib diganti saat login pertama. | Password | D-85, D-132, D-142 | Tidak perlu |
| Atur ulang kata sandi | Reset password | Admin memberi kata sandi sementara baru untuk satu akun kerja. Semua sesi akun itu berakhir. | Reset | D-85, D-101 | Tidak perlu |
| Versi template: Draft dan Terbit | Draft / Published | Versi format template atau process template. Draft masih bisa diubah; versi terbit tidak bisa diubah siapa pun, dan perubahan berikutnya masuk ke versi baru yang menyalin isinya dengan ID yang sama (M3.4). | Version | R-TPL-05 | Tidak perlu |
| Akun kerja / akun admin | Work account / Admin account | Login milik satu orang. Satu orang paling banyak punya satu akun kerja dan satu akun admin. Akun tidak pernah dihapus, hanya dinonaktifkan. | Work account / Admin account | D-72, D-79, D-105 | Tidak perlu |
| Kelompok menu Dokumen | Documents | Kelompok menu atas untuk register nomor dan impor nomor lama. Dokumen PFD, PFMEA, dan CP masuk ke sini saat tahapnya dibangun. | | D-127, UX-05 | Tidak perlu |
| Kelompok menu Pustaka template | Template library | Kelompok menu atas untuk format template, process template, dan master proses. | | D-127, UX-05 | Tidak perlu |
| Kelompok menu Master data | Master data | Kelompok menu atas untuk customer, model, jenis dokumen, jenis perubahan, jabatan, dan rute approval. | | D-127, UX-05 | Tidak perlu |
| Kelompok menu Admin | Admin | Kelompok menu atas untuk akun, audit log, dan pengaturan. Hanya tampil bagi pemegang hak admin; server tetap memeriksa setiap operasi. | | D-127, R-SEC-09 | Tidak perlu |
| Audit log | Audit log | Catatan setiap perubahan data, termasuk login, yang tidak bisa diubah atau dihapus dan dilindungi satu hash chain. Hanya dilihat lewat akun admin. | | R-SEC-03, D-75, D-83, D-107 | Tidak perlu |
| Cakupan customer | Customer scope | Batas data per customer. Cakupan Project Group untuk Project Leader dan Project Engineer diperiksa fungsi izin sejak M1.2; row-level security di M8.1. | Customer scope | R-SEC-02, D-81, D-118 | Tidak perlu |

## 8. Migrasi

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| Baseline migrasi | Migration baseline | Dokumen lama yang dimasukkan di revisi saat ini (misalnya -04), sama dengan hardcopy bertanda tangan. (usulan) | | R-MIG-03 | Belum |
| Released (migrated) | Released (migrated) | Status dokumen migrasi setelah diverifikasi terhadap hardcopy bertanda tangan. (usulan) | | R-MIG-06 | Tidak perlu |
| Mode migrasi | Migration mode | Saklar yang membuka fitur migrasi untuk peran khusus. Bisa dimatikan admin. (usulan) | | R-MIG-06 | Tidak perlu |
| Penautan baris | Linking pass | Sistem mengusulkan pasangan baris hasil tempel dengan poin template lewat kemiripan teks, lalu manusia mengonfirmasi. | Linking pass | R-MIG-05 | Tidak perlu |

## 9. Masa depan (latihan tahap 9 dan 10)

| Istilah | Istilah layar (Inggris) | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|---|
| API integrasi | Integration API | Antarmuka untuk aplikasi internal lain: membaca data dan mengirim usulan perubahan. (masa depan) | | Brief bagian 16 | Tidak perlu |
| Service account | Service account | Akun aplikasi lain dengan token, scope, dan masa berlaku. (masa depan) | | R-API-04 | Tidak perlu |
| Usulan perubahan | Change proposal | Perubahan yang dikirim aplikasi lain dan menunggu verifikasi manusia. (masa depan) | Change proposal | R-API-07 | Tidak perlu |
| Portal customer | Customer portal | Halaman baca saja bagi customer untuk melihat dan mengunduh dokumen rilis miliknya. (masa depan) | | D-18, R-POR-08 | Tidak perlu |
| Customer user | Customer user | Akun customer, terpisah dari akun internal. (masa depan) | | R-POR-04 | Tidak perlu |
| Izin tampil | Display permission | Konfirmasi internal sebelum revisi tampil di portal. (masa depan) | Display permission | R-PUB-03 | Tidak perlu |
| Sumber file | File source | Kebijakan per customer: scan_then_pdf (default), scan_only, atau pdf_only. (masa depan) | File source | R-PUB-02 | Tidak perlu |
| Field internal | Internal field | Field yang tidak dikirim ke customer. Fiturnya ditahan. | | D-31 | Tidak perlu |
