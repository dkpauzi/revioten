# U1 · Glosarium

- Paket UML: draft 0.7, 27 Oktober 2026
- Sumber: brief desain v1.15, termasuk temuan dari dokumen TR-020 (brief bagian 21). Nama jabatan mengikuti bagan organisasi NPI (update 22 Januari 2026). Nama orang tidak dicantumkan. Setiap istilah merujuk ke ID di brief (R-UML-01).
- Bahasa: penjelasan dalam bahasa Indonesia. Nama jabatan, nama standar, dan tahap tanda tangan memakai istilah yang sama dengan brief (R-UML-05, usulan).
- Kolom **Cek WI**:
  - **Belum**: istilah domain yang harus dicocokkan dengan WI dan prosedur yang berlaku. Diisi pemilik.
  - **Tidak perlu**: istilah sistem yang tidak muncul di WI.
- Tanda di kolom Arti:
  - **(usulan)**: aturannya masih [PROPOSED] di brief.
  - **(masa depan)**: hanya untuk latihan tahap 9 dan 10 (R-UML-06).
  - **Q-xx**: masih menunggu jawaban pemilik (brief bagian 12). Bagian ini tidak diisi dengan tebakan.

## 1. Dokumen dan standar

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| PFD | Process Flow Diagram: diagram alur proses produksi satu model. Job description di bagan organisasi menyebutnya PFC; sistem dan prosedur memakai PFD. | Diagram alir proses, PFC | R-DOC-01, R-DOC-04, D-57 | Belum |
| PFMEA | Process Failure Mode and Effects Analysis: analisis mode kegagalan proses beserta risikonya. | | R-DOC-01 | Belum |
| CP | Control Plan: rencana kontrol per proses. Cover SIIX menyebutnya Process Control Plan (PCP). | Control Plan, PCP | R-DOC-01 | Belum |
| Set (satu set) | PFD, PFMEA, dan CP satu model yang berbagi satu nomor urut. | | R-NUM-05 | Belum |
| AIAG 4th | Buku panduan FMEA AIAG edisi ke-4, berbasis RPN. | | R-TPL-02 | Belum |
| AIAG-VDA | Buku panduan FMEA gabungan AIAG dan VDA, berbasis Action Priority. Sistem saat ini hanya kompatibel dengan form-nya. Struktur penuhnya ditunda. | | R-TPL-02, R-DOC-03 | Belum |
| RPN / AP | Risk Priority Number / Action Priority. | | R-TPL-02 | Belum |
| SQAM | Supplier Quality Assurance Manual milik customer. Bisa mengatur format dan aturan dokumen, dan perubahannya sering berlaku surut. | | R-TPL-07 | Belum |
| PPAP | Production Part Approval Process. | | D-34 | Belum |
| 4M change | Perubahan Man, Machine, Material, atau Method. Persetujuan customer untuk perubahan ini diurus di luar sistem. | Perubahan 4M | R-CHG-01, D-34 | Belum |
| WI | Work Instruction: dokumen di luar sistem yang bisa terdampak perubahan. Istilah di sistem harus sama dengan WI. | Instruksi kerja | R-CHG-05, R-UML-05 | Belum |
| Standar inspeksi | Dokumen di luar sistem yang bisa terdampak perubahan. | Inspection standard | R-CHG-05 | Belum |

## 2. Isi dokumen dan template

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Proses | Langkah produksi, misalnya IQC. Punya ID tetap; namanya hanya tampilan dan boleh punya alias. | | R-DOC-04 | Belum |
| Sub-proses | Varian dari satu proses, misalnya IQC General, IQC MSD, IQC PCB, IQC Wireharness, IQC Connector. | | Brief bagian 3 | Belum |
| Poin | Satu butir setingkat baris di dalam sub-proses, dengan ID tetap. | | R-TPL-04 | Belum |
| IQC | Incoming Quality Control. | | Brief bagian 3 | Belum |
| Nomor operasi | Nomor urut proses di dokumen. Dibuat otomatis dan bisa diubah. | | R-CMP-01 | Belum |
| Baris dokumen | Satu baris lembar kerja PFD, PFMEA, atau CP. ID-nya tetap lintas revisi. | Row | R-DOC-01, R-DOC-02 | Tidak perlu |
| Format template | Template metodologi: kolom, metode risiko, tabel skala, field header, dan layout ekspor. Berversi. | | R-TPL-01, R-TPL-02 | Tidak perlu |
| Process template | Pustaka proses, sub-proses, dan poin. Berversi. | Template proses | R-TPL-01 | Tidak perlu |
| Cakupan template | Tingkat berlakunya process template: umum, customer, group model, dan model. Yang paling spesifik menang: model, lalu group model, lalu customer, lalu umum. | Scope | R-TPL-03, D-39 | Tidak perlu |
| Template General | Process template umum yang tidak terikat customer atau model. | Template umum | R-TPL-03 | Belum |
| Group model | Kumpulan model dalam satu customer yang berbagi isi template. Satu model paling banyak masuk satu group. | Model group | D-39, D-42 | Belum |
| Perakitan | Menyusun dokumen dari process template dengan kotak centang dan drag and drop. | Composition | R-CMP-01 | Tidak perlu |
| Poin dikecualikan | Poin template yang sengaja tidak dipakai satu model. Tetap tercatat, tidak dihapus diam-diam. (usulan) | Excluded point | R-CMP-03 | Tidak perlu |
| Perbedaan lokal | Baris yang diubah hanya untuk satu dokumen dan ditandai berbeda dari template. (usulan) | Local override | R-CMP-04 | Tidak perlu |
| Putus tautan | Dokumen atau baris yang dilepas dari tautan template, dengan alasan. (usulan) | Detach | R-CMP-04 | Tidak perlu |
| Mode tautan template | Hubungan dokumen dengan template: ikuti, kunci versi, atau lepas. (usulan) | Follow, pinned, detached | R-TPL-08 | Tidak perlu |
| Tautan PFMEA ke CP | Hubungan opsional antara baris PFMEA dan baris CP, dibuat manual atau dari template. | Linked counterpart | R-DOC-05, R-TPL-04 | Tidak perlu |
| Simbol alur PFD | Bentuk langkah di PFD: process, alternate process, decision, connector, merge, dan document. (usulan) | | R-DOC-09 | Belum |
| Disposisi NG | Tujuan produk NG dari langkah decision: scrap, NCP, atau rework. (usulan) | | R-DOC-09 | Belum |
| NCP | Nonconforming product, salah satu disposisi NG. | | R-DOC-09 | Belum |
| Simbol karakteristik | Simbol di PFMEA dan CP. Saat ini: Critical Process, Revision, Important Character (IC), dan Regulation. Disimpan di format template per customer. (usulan) | | R-TPL-11 | Belum |
| Core Team | Field header PFMEA dan CP yang berisi anggota tim beserta departemennya. | | Brief bagian 21, F-10, F-11 | Belum |
| Dokumen terkait | Nomor dan judul WI, SOP, QEP, dan dokumen lain yang dirujuk di kolom Relevant Document CP. Diambil dari master supaya sama dengan sumbernya. (usulan) | Relevant document | R-DOC-11 | Belum |
| Dokumen saling bergantung | Dokumen terpisah yang dirujuk sebagai pengganti langkah yang diulang, misalnya ESD control plan SC-ESD-K001. (usulan) | Interdependent document | R-DOC-10 | Belum |

## 3. Perubahan dan revisi

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Perubahan | Satu entitas untuk semua sumber perubahan. Berisi sumber, jenis, alasan, rujukan, lampiran, dokumen terdampak, dan daftar periksa dampak. | Change | R-CHG-01 | Tidak perlu |
| Permintaan perubahan | Perubahan yang diajukan manual oleh PPAP Engineer dengan dokumen pendukung. | Revision request | R-CHG-02 | Belum |
| Sumber perubahan | Asal perubahan: permintaan manual, yokoten, propagasi template, atau perubahan surut SQAM. | | R-CHG-01 | Belum |
| Pemicu perubahan | Pemicu yang dipakai sekarang: PP (pre-production), customer claim, ECR, dan temuan audit. (usulan) | | R-CHG-06 | Belum |
| ECR | Engineering Change Request: permintaan perubahan yang diajukan ke customer. Salah satu pemicu perubahan. | | R-CHG-06 | Belum |
| Yokoten | Penerapan satu perubahan atau pelajaran dari satu model, proses, atau dokumen ke yang lain. Target dipilih per customer, group model, model, proses, atau pencarian teks. | Horizontal deployment | R-CHG-04, D-40 | Belum |
| Propagasi template | Versi template baru yang diteruskan ke dokumen pengikutnya sebagai draft revisi. Tidak pernah mengubah dokumen secara langsung. | Template propagation | R-TPL-06, R-TPL-08 | Tidak perlu |
| Perubahan surut SQAM | Versi SQAM baru yang berlaku untuk dokumen aktif. Dibuat sebagai revisi baru; revisi yang sudah terbit tidak pernah ditulis ulang. | Retroactive change | R-TPL-07 | Belum |
| Jenis perubahan | Isi, tata letak saja, atau administratif. Menentukan jalur approval. | Change type | R-TPL-10 | Belum |
| Daftar periksa dampak | Catatan per perubahan tentang dokumen terkait yang terdampak (PFD, PFMEA, CP, WI, standar inspeksi), keputusannya, dan alasannya. (usulan) | Impact checklist | R-CHG-05 | Belum |
| Verifikasi | Penerimaan atau penolakan permintaan perubahan dengan catatan wajib, oleh Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller. Siapa yang memverifikasi tercatat. | | R-CHG-02, D-51, D-54 | Belum |
| Revisi | Versi dokumen bernomor bulat, ditampilkan -00, -01, dan seterusnya. Dokumen hasil migrasi mulai dari revisi saat ini, misalnya -04. | | R-REV-01 | Belum |
| Draft | Revisi yang masih bisa diedit dan belum berlaku. | | R-CHG-03 | Belum |
| Menunggu approval | Revisi yang sudah dicetak untuk tanda tangan dan dikunci. Edit sesudahnya membatalkan cetakan itu. (usulan) | Awaiting approval | R-REV-06 | Tidak perlu |
| Dikembalikan | Approver minta perubahan di kertas. Penyusun mencatatnya, dan revisi kembali ke draft. | Return | D-30 | Belum |
| Released | Revisi yang berlaku. Isinya dibekukan sebagai snapshot dan tidak bisa diubah. | Rilis | R-REV-02 | Belum |
| Obsolete | Revisi yang sudah digantikan. Tidak pernah berubah. | | R-REV-02 | Belum |
| Snapshot | Isi revisi yang dibekukan saat rilis, beserta hash SHA-256. | | R-REV-02 | Tidak perlu |
| Hash | Sidik jari isi. Ada tiga: snapshot_hash, customer_content_hash, dan file_hash. (usulan) | | R-REV-07 | Tidak perlu |
| Diff | Perbandingan baris dan sel antar revisi. | | R-REV-03 | Tidak perlu |
| Riwayat revisi | Tabel revisi otomatis di cetakan. Riwayat sebelum baseline migrasi disimpan sebagai teks dari tabel Excel lama. (usulan) | Revision history | R-REV-04, R-REV-05 | Belum |

## 4. Penomoran

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Nomor dokumen | Nomor terdaftar untuk PFD, PFMEA, atau CP per customer dan jenis dokumen, misalnya SC-KT-001. Disimpan sebagai teks dan tidak pernah dipakai ulang. | | R-NUM-01, R-NUM-02 | Belum |
| Awalan | SH untuk PFD, SF untuk PFMEA, dan SC untuk CP (dokumen TR-020), sebagai master data yang bisa diedit. | Prefix | R-NUM-01, R-NUM-02 | Belum |
| Kode customer | Kode singkat customer di nomor, misalnya KT. Harus sama dengan kode di master customer. (usulan) | | R-POR-01 | Belum |
| Register nomor | Daftar semua nomor beserta statusnya. Bisa diekspor sebagai master document list. (usulan) | Master document list | R-NUM-04 | Belum |
| Status nomor | Reserved, In use, Released, Obsolete, Void. (usulan) | | R-NUM-04 | Belum |
| Daftarkan sebagai set | Mengambil nomor bebas berikutnya di ketiga jenis dokumen supaya nomornya sejajar. (usulan) | Register as set | R-NUM-05 | Tidak perlu |
| Penghitung | Nomor terakhir per customer dan jenis dokumen, dinaikkan di dalam transaksi. | Counter | R-NUM-03 | Tidak perlu |

## 5. Tanda tangan, approval, dan rilis

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Prepared | Tahap tanda tangan penyusun: PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer. | | R-APR-07, D-63, D-67 | Belum |
| Checked | Tahap tanda tangan pengecek. Cukup satu orang: Project Engineer atau Project Leader dari Project Group customer itu. | | R-APR-07, D-48, D-54 | Belum |
| Approved | Tahap tanda tangan approver, berurutan: NPI Manager, lalu QA QC Manager, lalu slot GM bila ada di rute. | | R-APR-07, D-49 | Belum |
| Slot GM | Slot approver terakhir. Saat ini diisi Advisor; nanti bisa diisi Factory Manager. Cover mencetak "General Mgr." dengan nama pemegangnya, mengikuti cover TR-020 (D-76). | | D-56, D-76 | Belum |
| Judul cetak slot | Tulisan di bawah nama pada setiap slot, terpisah dari nama jabatan. Mengikuti cover TR-020: Process. Eng, Engineer, NPI Manager, QA / QC Mgr., General Mgr. | | R-APR-17, D-76 | Belum |
| Rute approval | Daftar slot tanda tangan berurutan untuk satu dokumen. Ada dua rute: sampai QA QC Manager dan sampai slot GM. Default-nya per customer, dan bisa diganti per dokumen dengan alasan. | Approval route | R-APR-08, R-APR-09, D-46, D-47 | Tidak perlu |
| Slot | Satu kotak tanda tangan di rute, dengan tahap dan jabatan yang boleh mengisinya. | | R-APR-08 | Tidak perlu |
| Tanggal tanda tangan | Tanggal per slot. Disiapkan dengan saklar untuk mematikan dan menyembunyikannya, dan saat ini mati. | | R-APR-14, D-52 | Tidak perlu |
| Tanggal rilis | Tanggal dokumen dicetak untuk tanda tangan, diisi otomatis. Secara resmi tanggal rilis adalah tanggung jawab DCC. Saat ini satu-satunya tanggal yang dicatat. | Release date | D-52, D-53, D-59 | Belum |
| Tanggal berlaku | Tanggal revisi mulai berlaku. Saat ini sama dengan tanggal rilis, dan aturannya harus bisa diubah nanti. | Effective date | D-59, R-REV-08 | Belum |
| Tanda tangan basah | Tanda tangan tangan di kertas. Approval saat ini manual. | Wet signature | R-APR-01, R-APR-04 | Belum |
| Pencatatan rilis | Dokumen kontrol mencatat scan bertanda tangan, lalu status revisi menjadi Released. Pencatat bukan penyusun. Berbeda dari tanggal rilis. (usulan) | Release recording | R-APR-02 | Belum |
| Dokumen kontrol | Fungsi yang mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer, yang saling mencatat rilis dokumen yang disusun yang lain. Nanti bisa ditambah DCC. | Document control | R-APR-02, D-60, D-68 | Belum |
| Salinan terkendali | Salinan cetak yang didistribusikan dan dilacak, dengan watermark CONTROLLED COPY. (usulan) | Controlled copy | R-PRT-04 | Belum |
| Salinan tidak terkendali | Salinan untuk referensi atau draft, dengan watermark UNCONTROLLED COPY. (usulan) | Uncontrolled copy | R-PRT-04 | Belum |
| Catatan cetak | Siapa mencetak, kapan, dan nomor salinannya. (usulan) | Print log | R-PRT-04 | Tidak perlu |
| QR | Kode di cetakan yang membuka halaman verifikasi. Opsional dengan saklar, dan saat ini mati karena belum dipakai. | | D-62, R-APR-16 | Tidak perlu |
| Halaman verifikasi | Halaman yang menampilkan status dokumen: berlaku, digantikan, atau dibatalkan. Bisa dibuka dari QR bila QR menyala, atau dari dalam sistem. (usulan) | | R-APR-03, R-APR-16 | Tidak perlu |
| Tanda revisi | Segitiga berisi nomor revisi di samping baris yang berubah pada revisi itu, dibuat otomatis dari diff. (usulan) | Revision mark | R-REV-09 | Belum |
| Layout | Template Excel beserta pemetaan sel, terpisah dari data baris. | | R-PRT-02 | Tidak perlu |

## 6. Jabatan

Nama jabatan mengikuti bagan organisasi NPI (D-54). Kolom "Sebutan lain" memuat nama yang dipakai di brief versi awal.

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Project Group | Kelompok customer yang dipegang satu Project Leader dan satu Project Engineer. Setiap customer dialokasikan ke satu Project Group, saat ini GR1 sampai GR4 (Tabel 1 di bagan organisasi). | GR | D-54 | Belum |
| PPAP Engineer | Level Leader. Menyusun dan memperbarui PFD, PFMEA, dan CP, mengelola master proses dan process template, dan untuk saat ini memberi tanda dicek PFMEA–CP. Juga memegang peran dokumen kontrol dan petugas migrasi. | | D-45, D-63, D-64, D-65, D-68, D-69 | Belum |
| PPAP Staff | Level SOP/Operator. Boleh menyusun dokumen, dan saat ini memegang peran dokumen kontrol dan petugas migrasi. Juga memegang akun admin terpisah, sehingga punya dua akun. | | D-60, D-63, D-69, D-72 | Belum |
| DCC | Dept Document Control. Secara resmi bertanggung jawab atas tanggal rilis, dan nanti bisa memegang peran dokumen kontrol. | Document Control Department | D-59, D-60 | Belum |
| Project Engineer | Level Leader, satu per Project Group. Boleh menyusun dokumen (D-67), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). | Project Leader, Production Engineer (brief versi awal) | D-48, D-51, D-54 | Belum |
| Project Leader | Level Supervisor, satu per Project Group. Boleh menyusun dokumen (D-63), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). | Project Leader GR, Project Leader Supervisor, project supervisor (brief versi awal) | D-48, D-51, D-54 | Belum |
| APQP/PPAP Controller | Level Supervisor, satu orang, saat ini dirangkap Project Leader GR3. Boleh memverifikasi permintaan perubahan semua customer. | PPAP Supervisor | D-51, D-54 | Belum |
| NPI Manager | Approver pertama. Menurut bagan, menyetujui dokumen resmi departemen NPI. Bila struktur berganti nama, jabatan ini bisa bernama PE Manager. | PE Manager (nama lama atau nama cadangan) | D-45, D-55 | Belum |
| QA QC Manager | Approver kedua. Rute bisa berakhir di sini. Jabatan ini di luar bagan NPI. | | D-45, D-46 | Belum |
| Advisor | Level General Manager. Saat ini mengisi slot GM. | | D-56 | Belum |
| Factory Manager | Level General Manager. Nanti bisa mengisi slot GM. | | D-56 | Belum |
| QA, QC, internal audit | Pengguna halaman verifikasi QR dan dokumen rilis, dengan peran viewer. | | D-32, D-41 | Belum |

## 7. Peran sistem dan keamanan

Peran adalah hak akses di aplikasi, berbeda dari jabatan. Pemetaan jabatan ke peran mengikuti D-51, D-54, dan D-63 sampai D-66, dan peta lengkapnya ada di `U3-peta-pengguna.md`. Satu orang boleh memegang lebih dari satu jabatan (D-58).

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Level akses | Buat / Edit, Verifikasi, Lihat, dan Admin. | | R-SEC-10, D-66 | Tidak perlu |
| Hak akses | Satu aksi di sistem, misalnya mencatat rilis. Peran adalah kumpulan hak yang isinya diatur admin. | Permission | R-SEC-10, D-66 | Tidak perlu |
| Penyusun (author) | Peran PPAP Engineer, PPAP Staff, Project Leader, dan Project Engineer: menyusun dokumen, mendaftarkan nomor, mengajukan perubahan, dan mencetak untuk tanda tangan. | Author | R-SEC-09, D-63, D-67 | Tidak perlu |
| Pengelola template | Peran PPAP Engineer: mengelola master proses dan process template. | | D-64 | Tidak perlu |
| Pemeriksa PFMEA–CP | Peran PPAP Engineer untuk saat ini: memberi tanda dicek di perbandingan PFMEA dan CP. | | D-65, R-DOC-06 | Tidak perlu |
| verifier | Peran Project Engineer, Project Leader, dan APQP/PPAP Controller: memverifikasi permintaan perubahan. | | R-SEC-09, D-51, D-54 | Tidak perlu |
| approver | Peran untuk approval di aplikasi nanti. Selama approval masih di kertas, approver memakai akun Viewer. | | R-SEC-09, R-APR-13, D-70 | Tidak perlu |
| document control | Peran dokumen kontrol: mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer. | | R-SEC-09, R-APR-02, D-60, D-68 | Tidak perlu |
| admin | Mengelola pengguna, hak akses, master data, rute approval, dan pengaturan, serta melihat audit log. Dipakai lewat akun admin terpisah yang tidak memegang peran kerja; saat ini dipegang PPAP Staff. Akun admin ditautkan ke pemegangnya, dan akun admin terakhir tidak bisa dihapus. | Akun admin | R-SEC-10, R-SEC-13, D-72, D-73, D-74 | Tidak perlu |
| viewer | Hanya baca. Diberikan ke semua jabatan sampai level GM, termasuk approver, DCC, QA, QC, internal audit, dan tim WI. | | R-APR-06, D-41, D-70 | Tidak perlu |
| Audit log | Catatan semua aksi yang tidak bisa diubah atau dihapus, dilindungi hash chain. (usulan) | | R-SEC-03 | Tidak perlu |
| Cakupan customer | Batas data per customer pada setiap dokumen. Ditegakkan penuh di milestone M8.1. | Customer scope | R-SEC-02 | Tidak perlu |

## 8. Migrasi

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| Baseline migrasi | Dokumen lama yang dimasukkan di revisi saat ini (misalnya -04), sama dengan hardcopy bertanda tangan. (usulan) | | R-MIG-03 | Belum |
| Released (migrated) | Status dokumen migrasi setelah diverifikasi terhadap hardcopy bertanda tangan. (usulan) | | R-MIG-06 | Tidak perlu |
| Mode migrasi | Saklar yang membuka fitur migrasi untuk peran khusus. Bisa dimatikan admin. (usulan) | | R-MIG-06 | Tidak perlu |
| Penautan baris | Sistem mengusulkan pasangan baris hasil tempel dengan poin template lewat kemiripan teks, lalu manusia mengonfirmasi. | Linking pass | R-MIG-05 | Tidak perlu |

## 9. Masa depan (latihan tahap 9 dan 10)

| Istilah | Arti | Sebutan lain | Rujukan | Cek WI |
|---|---|---|---|---|
| API integrasi | Antarmuka untuk aplikasi internal lain: membaca data dan mengirim usulan perubahan. (masa depan) | | Brief bagian 16 | Tidak perlu |
| Service account | Akun aplikasi lain dengan token, scope, dan masa berlaku. (masa depan) | | R-API-04 | Tidak perlu |
| Usulan perubahan | Perubahan yang dikirim aplikasi lain dan menunggu verifikasi manusia. (masa depan) | Change proposal | R-API-07 | Tidak perlu |
| Portal customer | Halaman baca saja bagi customer untuk melihat dan mengunduh dokumen rilis miliknya. (masa depan) | | D-18, R-POR-08 | Tidak perlu |
| Customer user | Akun customer, terpisah dari akun internal. (masa depan) | | R-POR-04 | Tidak perlu |
| Izin tampil | Konfirmasi internal sebelum revisi tampil di portal. (masa depan) | Display permission | R-PUB-03 | Tidak perlu |
| Sumber file | Kebijakan per customer: scan_then_pdf (default), scan_only, atau pdf_only. (masa depan) | File source | R-PUB-02 | Tidak perlu |
| Field internal | Field yang tidak dikirim ke customer. Fiturnya ditahan. | | D-31 | Tidak perlu |
