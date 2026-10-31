# Sistem PFD, PFMEA, dan CP · ringkasan untuk desain UI/UX

- **Dibuat:** 31 Oktober 2026, sesudah Tahap 2 (penomoran dokumen) selesai; diperbarui 31 Oktober 2026 sesudah M3.1 (format template) dan M3.2 (pustaka proses).
- **Untuk:** Claude chat, atau siapa pun yang mendesain tampilan web ini, supaya konsepnya bisa dipahami tanpa membaca kode.
- **Sumber:**
  - brief desain v1.17 (`docs/brief/pfmea-system-design-brief.md`);
  - paket UML draft 0.9 (`docs/uml/`);
  - catatan milestone (`docs/catatan/`) dan lembar evaluasi desain (`docs/evaluasi-desain.md`);
  - kode yang berjalan di repo.

  Lampiran A sampai D, serta bagian 10.2 dan 10.3, disalin otomatis dari sumber itu.
- **Tanpa data asli.** Kode customer seperti KT, NB, atau AB hanya contoh. Nama orang, nama customer, nama perusahaan, dan nomor dokumen internal sengaja tidak ditulis, karena dokumen asli perusahaan tidak boleh keluar dari repo private dan komputer pemilik.

## Daftar isi

1. Cara memakai dokumen ini
2. Proyek dalam satu halaman
3. Konsep domain untuk desainer
4. Pengguna, jabatan, peran, dan hak akses
5. Arsitektur dan batasan teknis yang memengaruhi desain
6. Halaman yang sudah dibangun
7. Data yang sudah disimpan
8. Alur kerja dan layar yang akan datang
9. Aturan yang wajib dihormati desain
10. Keputusan, temuan, dan pertanyaan terbuka
11. Rencana tahap dan statusnya
12. Saran untuk sesi desain UI/UX (usulan)
- Lampiran A. Semua keputusan (D-01 sampai D-120)
- Lampiran B. Daftar use case (U3)
- Lampiran C. Glosarium dan istilah layar (U1)
- Lampiran D. Operasi API yang sudah ada

## 1. Cara memakai dokumen ini

- **Penanda status** (dari brief):

  | Penanda | Arti |
  |---|---|
  | [DECIDED] | Sudah diputuskan pemilik dan mengikat |
  | [PROPOSED] | Rencana kerja, masih bisa berubah |
  | [OPEN] | Belum terjawab; ada asumsi kerja |
  | [DEFERRED] | Sengaja tidak dibangun sekarang |
  | [SUPERSEDED] | Sudah diganti keputusan lain |

- **Bahasa layar Inggris** (D-96). Semua teks layar ada di satu katalog, jadi bahasa Indonesia bisa ditambah nanti dengan kunci yang sama. Label di layar mengikuti kolom "Istilah layar" di Lampiran C (D-98). Istilah baru ditandai sebagai usulan, tidak dipakai diam-diam.
- **Yang sudah ada dipisah dari yang direncanakan.** Bagian 6 berisi halaman yang sudah berjalan; bagian 8 berisi layar yang belum dibangun.
- **Desain tidak mengubah aturan.** Bila sebuah desain butuh aturan lain dari bagian 9, tulis sebagai usulan untuk pemilik. Pemilik yang memutuskan, lalu keputusannya dicatat di brief.
- **Desain ini memang direncanakan.** D-92 sudah memutuskan bahwa user flow dan wireframe layar Tahap 4 dan 5 dibuat sebelum Tahap 4, lalu disimpan bersama paket UML. Hasil desain dibawa kembali ke Claude Code dan dibangun milestone demi milestone, dengan test.

## 2. Proyek dalam satu halaman

- **Apa:** sistem web internal (self-hosted) untuk menyusun, merevisi, menomori, mencetak, dan mencatat rilis tiga dokumen mutu yang saling terkait di pabrik EMS (perakitan elektronik): PFD, PFMEA, dan Control Plan (CP).
- **Keadaan sekarang di pabrik:** dokumen dibuat di Excel dan beredar sebagai hardcopy bertanda tangan basah.
- **Masalah yang ingin dihilangkan:**
  - revisi lambat dan rawan salah, terutama revisi yokoten (satu perbaikan diterapkan ke banyak model);
  - sulit melihat apa yang berubah, kapan, kenapa, dan atas permintaan siapa;
  - nomor dokumen yang didaftarkan di Excel oleh orang berbeda sering tidak sejajar antar dokumen satu model;
  - perubahan template customer (SQAM), yang sering berlaku surut, harus diterapkan tangan ke banyak file lama.
- **Tujuan utama pemilik: mempermudah revisi yokoten.** Satu perubahan sampai ke banyak dokumen, dengan keputusan per dokumen dan papan status.
- **Tujuan lain (brief G1 sampai G10):**
  - riwayat perubahan mudah dilihat (diff baris dan sel, terhubung ke alasan dan lampiran);
  - dokumen dirakit dari template proses;
  - nomor tanpa bentrok;
  - cetak Excel dan PDF ukuran A3 dan A4 sebagai fitur utama;
  - migrasi dokumen Excel lama tanpa dibersihkan dulu;
  - persetujuan tetap manual, tetapi catatan rilisnya tepercaya;
  - aman sejak awal;
  - spesifikasi dan UML cukup lengkap untuk tim IT perusahaan.
- **Persetujuan tetap di kertas (D-07).**
  - Sistem mencetak dokumen beserta kotak tanda tangannya.
  - Orang menandatangani di kertas.
  - Dokumen kontrol mengunggah scan dan mencatat rilis.

  Tidak ada tombol "Approve" untuk approver di sistem; approval dan tanda tangan digital baru sebatas rancangan.
- **Bukan tujuan sekarang (NG1 sampai NG8):**
  - pemeriksaan otomatis kesesuaian PFMEA dan CP (orang yang memeriksa);
  - login AD atau LDAP;
  - analisis jaringan kegagalan AIAG-VDA penuh;
  - hak akses yang sangat rinci;
  - QMS lengkap;
  - akses dari luar perusahaan;
  - API integrasi (sudah dirancang, dibangun setelah MVP).
- **Konteks:**
  - proyek belajar pribadi pemilik sekaligus usulan QCC (quality control circle) di perusahaan;
  - perusahaan menerima spesifikasi dan UML, lalu tim IT membangun versinya sendiri;
  - versi pemilik berjalan di localhost sebagai demo (D-36);
  - skala yang diasumsikan 10 sampai 50 pengguna internal bersamaan.
- **Ukuran keberhasilan** (baseline belum diukur):
  - waktu merakit set dokumen model baru;
  - waktu menerapkan satu yokoten ke N model;
  - persentase baris yang masih mengikuti template;
  - jumlah bentrok nomor dan set yang tidak sejajar;
  - waktu menemukan riwayat satu perubahan;
  - penerimaan hasil cetak oleh pengguna.

## 3. Konsep domain untuk desainer

### 3.1 Tiga dokumen yang saling terkait

| Dokumen | Isi | Contoh nomor |
|---|---|---|
| PFD (Process Flow Diagram) | Urutan langkah proses dengan nomor bertingkat (misalnya 5.7.1), simbol alur (process, alternate process, decision, connector, merge, document), dan disposisi NG untuk langkah decision (scrap, NCP, rework). | SH-KT-001 |
| PFMEA (Process Failure Mode and Effects Analysis) | Per proses: fungsi, mode kegagalan, efek, penyebab, kontrol, rating, dan tindakan. Form AIAG 4th memakai RPN; AIAG-VDA memakai AP. Aturan aksi di form sekarang (RPN di atas 100 atau severity 8) hanya menyorot, tidak memblokir. | SF-KT-001 |
| CP (Control Plan) | Per proses: mesin dan alat, karakteristik produk dan proses beserta spesifikasinya, karakteristik khusus, metode kontrol (alat, metode, frekuensi, PIC), dokumen terkait (WI, SOP), pokayoke, dan reaction plan. | SC-KT-001 |

- **Dicocokkan per proses.** Proses punya ID tetap di master proses; namanya hanya tampilan, dan alias menampung nama lama. Baris PFMEA dan CP memilih proses dari daftar PFD, bukan mengetik nama.
- **Simbol di form:** Critical Process, Revision, Important Character, dan Regulation. Baris yang berubah di satu revisi diberi tanda segitiga berisi nomor revisi; rencananya tanda ini dibuat otomatis dari diff.
- **Cover cetak:**
  - nomor dokumen di atas, judul, dan field header: Customer, Model Name, Part Name, Part Number, Process or Line;
  - tabel riwayat revisi: No, Date, Revision, Description of Change;
  - slot tanda tangan internal dari rute approval, ditambah area approval customer;
  - nama customer di footer.

  Halaman dalam mengulang blok tanda tangan dengan tulisan "Refer approval sheet".
- **Satu set dokumen per model.** Urutan master data: Project Group, customer, group model (opsional), model, lalu part number. Satu model paling banyak ada di satu group model.

### 3.2 Nomor dokumen (sudah dibangun, Tahap 2)

- **Bentuk nomor:** `SC-KT-001`, tersusun dari tiga bagian.
  - Awalan: kode pabrik `S` ditambah huruf jenis dokumen (`H` untuk PFD, `F` untuk PFMEA, `C` untuk CP).
  - Kode customer: dua huruf, atau kode khusus seperti `ZZ` untuk dokumen umum.
  - Nomor berjalan: 001 sampai 999, lalu A001 sampai Z999.

  Revisi (`-00`, `-01`, dan seterusnya) bukan bagian dari nomor terdaftar.
- **Pola, kode pabrik, dan digit revisi adalah master data.** Nomor yang sudah terdaftar disimpan sebagai teks dan tidak pernah dihitung ulang.
- **Alokasi oleh server**, tanpa bentrok dan tanpa lubang, dengan satu penghitung per kode customer dan jenis dokumen.
- **Daftar sebagai set:** PFD, PFMEA, dan CP satu model mendapat nomor berjalan yang sama. Nomor yang dilewati penghitung yang tertinggal dicatat dengan status Skipped.
- **Status nomor** (tidak pernah dihapus atau dipakai ulang):

  | Status | Arti |
  |---|---|
  | Reserved | Sudah didaftarkan, dokumennya belum dibuat |
  | In use | Dokumen sudah dibuat (Tahap 4) |
  | Released | Rilis sudah dicatat (Tahap 6) |
  | Obsolete | Ditarik |
  | Void | Dibatalkan dengan alasan; hanya dari Reserved atau In use |
  | Skipped | Dilewati saat daftar sebagai set |

- **Kode customer bisa berganti.** Nomor lama tetap memakai kode lama dan tetap milik customer itu; kode baru mulai lagi dari 001.
- **Impor nomor lama.** Register Excel lama ditempel dan nomornya masuk dengan teks aslinya. Penghitung naik ke nomor tertinggi. Lubang, nomor ganda, dan set yang tidak sejajar hanya dilaporkan.

### 3.3 Template (Tahap 3; format template dibangun di M3.1, pustaka proses di M3.2)

- **Format template:**
  - metodologi: AIAG 4th, AIAG-VDA, atau format SQAM customer;
  - kolom ber-ID tetap, dengan label yang hanya tampilan;
  - field header, metode risiko (RPN atau AP) beserta tabel skalanya, simbol, dan referensi layout ekspor.

  Format template berversi: draft lalu terbit, dan versi terbit tidak bisa diubah.
- **Process template:** pustaka proses, sub-proses, dan poin dengan ID tetap. Ada empat tingkat cakupan: umum, customer, group model, dan model. Yang paling spesifik menang (model, lalu group model, lalu customer, lalu umum).

  Yang sudah dibangun di M3.2: satu template per proses dengan cakupan umum saja (ASUMSI E-72). Sub-proses dan poin punya ID tetap; nama dan teksnya bisa diganti, urutannya bisa diubah, dan yang tidak dipakai disembunyikan, tidak dihapus. Poin dibuat untuk PFD, PFMEA, atau CP, dan untuk sementara berisi satu baris teks (ASUMSI E-73). Pencocokan PFD, PFMEA, dan CP tetap di tingkat proses (ASUMSI E-33).
- **Perubahan template tidak pernah mengubah dokumen langsung.** Perubahan selalu lewat draft revisi yang disetujui, dengan pratinjau dampak sebelum diterapkan.

### 3.4 Dokumen dan perakitan (Tahap 4, belum dibangun)

- **Baris dokumen datar:** satu baris sama dengan satu baris lembar kerja, dikelompokkan per proses. Isinya divalidasi terhadap versi format template, dan ID baris tetap antar revisi.
- **Perakitan dua panel:**
  - panel kiri berisi pohon template dengan kotak centang;
  - panel kanan berisi struktur model, diurutkan dengan drag and drop;
  - nomor operasi dibuat otomatis tapi bisa diubah;
  - satu perakitan menghasilkan baris PFD, PFMEA, dan CP sekaligus;
  - header diisi tangan.
- **Tautan ke template:** baris tetap tertaut ke poin template, perbedaan lokal ditandai, dan memutus tautan butuh alasan.
- **Editor tabel (grid):** edit sel, tambah dan hapus baris, dengan penguncian optimistik supaya dua orang tidak saling menimpa. Tempel dari Excel lewat pratinjau dan pemetaan kolom.
- **Cek berdampingan:** PFMEA dan CP ditampilkan berdampingan per proses, dengan tanda "dicek oleh, tanggal" per proses per revisi.
- **Diagram PFD** dibentuk dari daftar proses (opsional).

### 3.5 Perubahan, revisi, dan yokoten (Tahap 5, belum dibangun)

- **Satu entitas perubahan** untuk semua penyebab revisi: permintaan manual, yokoten, propagasi template, dan perubahan surut SQAM. Isinya:

  | Field | Nilai |
  |---|---|
  | Sumber | Master: PP (pre-production), Customer claim, ECR, Yokoten, Template propagation, SQAM, Audit finding |
  | Jenis | Content, Layout only, Administrative |
  | Kategori 4M (opsional) | Man, Machine, Material, Method |
  | Lainnya | Alasan, rujukan sumber, lampiran, dan status |

- **Status perubahan** [PROPOSED]:
  1. Draft.
  2. Submitted: lampiran sudah lengkap.
  3. Accepted atau Rejected oleh verifikator, dengan catatan wajib.
  4. Implementing: draft revisi dibuat per target.
  5. Released.
  6. Closed.
- **Verifikasi.** Yang boleh memverifikasi: Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller. Mereka menerima atau menolak dengan catatan wajib, dan pemohon tidak boleh memverifikasi permintaannya sendiri.
- **Draft revisi.** Setiap dokumen terdampak mendapat draft revisi; revisi yang berlaku tidak berubah sampai draft dirilis.
- **Yokoten:**
  - pilih target per customer, group model, model, proses, atau pencarian teks;
  - pemilik setiap target memutuskan terapkan, sesuaikan, atau tidak berlaku (dengan alasan);
  - papan status menunjukkan kemajuan per model.
- **Daftar periksa dampak** per perubahan: dokumen terkait mana yang terdampak (PFD, PFMEA, CP, WI, standar inspeksi), keputusannya, dan alasannya.
- **Riwayat seperti Git:** diff baris dan sel antar revisi, per dokumen dan per proses, terhubung ke perubahan, alasan, dan lampiran.
- **Revisi rilis dibekukan** sebagai snapshot dengan hash SHA-256 dan tidak bisa diubah siapa pun.

### 3.6 Cetak, tanda tangan, dan rilis (Tahap 6, belum dibangun)

- **Format cetak:** Excel (A3 dan A4 landscape) dan PDF wajib ada. Layout adalah data, terpisah dari isi.
- **Cetak untuk tanda tangan:**
  - revisi dikunci (menunggu approval), dan tanggal rilis terisi otomatis;
  - mengedit setelah cetak membatalkan cetakan itu;
  - bila approver meminta perubahan di kertas, penyusun mencatat "dokumen dikembalikan" dan revisi kembali ke draft.
- **Rute approval menentukan kotak tanda tangan.**
  - Slot berurutan: Prepared, Checked, lalu satu atau lebih Approved. Setiap slot punya judul cetak dan jabatan yang boleh mengisi.
  - Dua rute awal: "Up to QA QC Manager" dan "Up to GM".
  - Setiap customer punya rute default; penyusun boleh memilih rute lain dengan alasan.
- **Urutan tanda tangan:** Checked, NPI Manager, QA QC Manager, lalu slot GM bila rutenya memuatnya. Satu orang hanya mengisi satu slot dalam satu rute.
- **Catat rilis:** dokumen kontrol mengunggah scan bertanda tangan dan mengonfirmasi slot sesuai urutan. Pencatat harus berbeda dari penyusun.
- **Salinan:**
  - watermark UNCONTROLLED COPY untuk draft dan CONTROLLED COPY untuk rilis;
  - catatan cetak berisi siapa, kapan, dan nomor salinan;
  - daftar salinan yang harus ditarik saat revisi baru rilis.
- **Saklar yang sekarang mati:**
  - QR code beserta halaman verifikasi (berlaku, digantikan, atau ditarik);
  - tanggal tanda tangan per slot (sekarang hanya tanggal rilis yang dicatat).

### 3.7 Migrasi (Tahap 7, belum dibangun)

- **Dokumen lama masuk tanpa dibersihkan**, sebagai baseline di revisi saat ini (misalnya -04), sama dengan hardcopy bertanda tangan. Ada tiga jalan: form header, tempel dari Excel, dan impor file; isinya juga bisa dirakit dari template.
- **Layar pembanding:** Excel lama hanya dipakai untuk dibandingkan dengan hasil perakitan.
- **Penautan baris:** sistem mengusulkan pasangan baris dan poin template lewat kemiripan teks, lalu manusia mengonfirmasi.
- **Released (migrated):** status ini diberikan setelah dokumen diverifikasi terhadap hardcopy. Mode migrasi bisa dimatikan admin.
- **Dasbor ukuran:** persentase baris yang mengikuti template, status dokumen dan yokoten, serta waktu revisi.

### 3.8 Masa depan (latihan Tahap 9 dan 10)

- **API integrasi** untuk aplikasi internal lain: token service account, baca data rilis, dan usulan perubahan yang tetap menunggu verifikasi manusia.
- **Portal customer** (hanya baca): customer melihat dan mengunduh dokumen rilis miliknya, dengan izin tampil per revisi.

## 4. Pengguna, jabatan, peran, dan hak akses

### 4.1 Jabatan

Jabatan adalah master data dari bagan organisasi NPI. ID-nya tetap dan namanya bisa diganti; satu orang boleh merangkap beberapa jabatan.

| Jabatan | Aturan pemegang | Peran di sistem saat ini | Cakupan customer |
|---|---|---|---|
| PPAP Engineer | Bebas | Author, Template maintainer, PFMEA–CP checker, Document control, Migration | Semua |
| PPAP Staff | Bebas | Author, Document control, Migration. Juga memegang akun admin terpisah | Semua |
| Project Leader (level Supervisor) | Satu per Project Group | Author, Verifier | Customer di Project Group sendiri |
| Project Engineer (level Leader) | Satu per Project Group | Author, Verifier | Customer di Project Group sendiri |
| APQP/PPAP Controller (PPAP Supervisor) | Satu orang | Verifier | Semua |
| NPI Manager, QA QC Manager, Advisor (slot GM), Factory Manager | Satu orang per jabatan | Viewer; tetap menandatangani di kertas | Semua, dokumen rilis |

Pengguna lain mendapat akun Viewer: QA, QC, internal audit, DCC (Dept Document Control, nanti bisa diberi peran Document control), tim WI, serta ECN & SAP Coordinator.

### 4.2 Peran dan hak

- **Katalog hak tetap; isi setiap peran diatur admin** (D-66). Ada empat level akses: Buat/Edit, Verifikasi, Lihat, dan Admin.
- **Delapan peran** dengan nama layar: Author, Template maintainer, PFMEA–CP checker, Verifier, Document control, Migration, Viewer, dan Admin.
- **Semua akun yang login memegang hak Viewer** (L-1 sampai L-3).
- **Setiap peran yang diberikan punya cakupan:** All customers, atau Own project group. Cakupan Own project group dibaca dari pemegang jabatan Project Leader dan Project Engineer.

| Kode | Hak | Peran pemegang |
|---|---|---|
| L-1 | Lihat dokumen rilis dan halaman verifikasi | Semua |
| L-2 | Lihat register nomor, riwayat dan diff, diagram PFD, dasbor, dan master data | Semua |
| L-3 | Ekspor Excel dan PDF dokumen rilis | Semua |
| L-4 | Lihat draft dan permintaan perubahan yang berjalan (usulan) | Semua peran kerja selain Viewer |
| E-1 | Daftarkan nomor dokumen, termasuk sebagai set | Author |
| E-2 | Buat, rakit, dan edit dokumen draft | Author |
| E-3 | Ajukan perubahan dan isi daftar periksa dampak | Author |
| E-4 | Kerjakan draft revisi, yokoten, dan perubahan surut SQAM | Author |
| E-5 | Cetak untuk tanda tangan dan catat dokumen dikembalikan | Author |
| E-6 | Kelola master proses dan process template | Template maintainer |
| E-7 | Catat rilis | Document control |
| E-8 | Ubah status nomor dan impor nomor lama | Document control |
| E-9 | Cetak salinan terkendali dan lihat salinan yang harus ditarik | Document control |
| E-10 | Migrasikan dokumen lama | Migration |
| V-1 | Verifikasi permintaan perubahan | Verifier |
| V-2 | Beri tanda dicek di perbandingan PFMEA dan CP | PFMEA–CP checker |
| A-1 | Kelola pengguna, hak akses, dan pemegang jabatan | Admin |
| A-2 | Kelola master data, jenis dokumen dan perubahan, rute approval, dokumen terkait, dan format template | Admin |
| A-3 | Kelola pengaturan sistem | Admin |
| A-4 | Lihat audit log | Admin |

- **Akun admin adalah akun terpisah** (D-72, D-120). Akun ini hanya memegang hak admin ditambah L-1 sampai L-3, dan tidak pernah memegang peran kerja.
- **Akun admin terakhir tidak bisa dihapus.** Akun tidak pernah dihapus, hanya dinonaktifkan (D-73, D-105).
- **Audit log hanya dilihat lewat akun admin** (D-75).

### 4.3 Aturan tetap yang tidak bisa dimatikan admin

1. Pemohon perubahan tidak memverifikasi permintaannya sendiri.
2. Satu orang hanya mengisi satu slot dalam satu rute, jadi penyusun tidak bisa sekaligus menjadi pengecek.
3. Pencatat rilis berbeda dari penyusun.
4. Revisi yang sudah rilis tidak bisa diubah siapa pun, termasuk admin.
5. Audit log tidak bisa diubah atau dihapus.
6. Hak admin tidak mencakup mengedit isi dokumen.
7. Akun admin terakhir tidak bisa dihapus atau kehilangan peran admin.
8. Akun admin tidak memegang peran kerja, dan akun kerja tidak memegang peran admin.
9. Setiap akun boleh mengurus akunnya sendiri: sesi, logout, dan password.
10. Hak bercakupan Project Group hanya berlaku untuk operasi pada satu customer. Daftar lintas customer disaring per customer.

### 4.4 Akibatnya untuk desain

- **Menu dan tombol mengikuti hak akun.**
  - Menu Accounts hanya untuk A-1, Audit log untuk A-4, dan Settings untuk A-3.
  - Tombol Void hanya untuk E-8.

  Server tetap memeriksa setiap operasi dan menjawab 403 bila tidak berhak, dan layar menampilkan pesan yang jelas.
- **Master data terlihat oleh semua akun** (L-2), tetapi tombol ubah hanya untuk pemegang hak.
- **Akun admin perlu dibedakan dari akun kerja.** Sekarang header menampilkan label "Admin account".
- **Tidak ada tombol hapus** untuk akun, customer, nomor, proses, atau revisi rilis. Penggantinya: nonaktifkan, Void atau Obsolete, atau revisi baru.
- **Layar perlu memberi tahu kenapa sebuah aksi tidak boleh.** Contoh: penyusun tidak bisa menjadi pengecek dokumennya sendiri, dan customer tanpa rute default belum bisa dicetak untuk tanda tangan.

## 5. Arsitektur dan batasan teknis yang memengaruhi desain

- **Satu server di localhost.** Backend Go dengan PostgreSQL. Frontend SvelteKit 3 (Svelte 5) dibangun sebagai SPA statis lalu di-embed ke binary Go.
- **API-first.** Setiap layar adalah klien API `/api/v1`, yang dijelaskan kontrak OpenAPI 3.0.4. Desain yang butuh data di luar API ditulis sebagai usulan perubahan kontrak.
- **Satu bentuk error API:** kode, pesan, dan request ID. Masalah per field (`field:kode`) diterjemahkan ke teks layar di katalog.
  - Masalah per field tampil di bawah isian yang bersangkutan.
  - Error tak terduga tampil sebagai pesan umum beserta request ID.
- **Login lokal (D-85):**
  - password 15 sampai 128 karakter, dicek terhadap daftar password umum;
  - sesi berakhir setelah 30 menit diam atau 8 jam;
  - akun terkunci 15 menit setiap 10 kali login gagal;
  - akun kerja dinonaktifkan pada kegagalan ke-100 (akun admin tidak);
  - admin mereset password dengan password sementara yang tampil sekali dan wajib diganti saat login.
- **Teks layar** ada di satu katalog (`web/src/lib/i18n/en.ts`, sekitar 520 kunci).
- **Format waktu** `YYYY-MM-DD HH:mm` waktu lokal; audit log memakai detik.
- **Test E2E (Playwright) mencari elemen lewat label dan role yang terlihat**, misalnya tombol "Register" atau isian berlabel "Customer". Desain baru perlu mempertahankan:
  - label teks yang terlihat untuk setiap isian dan tombol;
  - elemen semantik: table, form, button, `role="status"` untuk pesan berhasil, dan `role="alert"` untuk error.
- **Pustaka yang direncanakan:** tabulator-tables untuk grid, serta @xyflow/svelte dengan elkjs untuk diagram PFD. Belum ada pustaka komponen atau CSS framework. Pustaka baru harus versi stabil terbaru, tanpa konflik peer dependency.
- **Tampilan sekarang sengaja polos:** HTML semantik dengan sedikit CSS per halaman, yaitu bar header dengan tautan, tabel bergaris bawah, teks error merah, dan teks petunjuk abu-abu. Belum ada design system, ikon, tema, atau tata letak khusus layar kecil.

## 6. Halaman yang sudah dibangun (Tahap 0 sampai M3.2)

**Header di setiap halaman setelah login:**
- judul aplikasi "PFD, PFMEA, and CP System";
- menu Home, Customers, Models, Processes, Format templates, Process templates, Document types, Change types, Positions, Approval routes, Numbers, lalu Accounts (A-1), Audit log (A-4), Settings (A-3), dan About;
- di kanan: "Logged in as {nama} ({login})", label "Admin account" bila akun admin, tautan "Change password", dan tombol "Log out".

Menu disembunyikan selama password wajib diganti.

| Halaman (URL) | Siapa | Isi dan perilaku |
|---|---|---|
| Log in (`/login`) | Siapa pun | Login name dan Password. Pesan gagal sama untuk login atau password yang salah. Petunjuk: bila lupa password, minta admin meresetnya. |
| Change password (`/account/password`) | Setiap akun untuk dirinya | Current password, New password, dan pengulangannya. Wajib setelah password direset; sesi lain akun itu berakhir setelah diganti. |
| Home (`/`) | Semua | Judul dan status server (OK, database tidak terjangkau, atau server tidak terjangkau) dengan tombol Check again. |
| About (`/about`) | Semua | Penjelasan singkat sistem. |
| Accounts (`/admin/accounts`) | Admin (A-1) | Buat akun: orang baru atau orang yang sudah ada, nomor karyawan opsional, jenis akun (Work account atau Admin account), login name, dan peran beserta cakupannya (All customers atau Own project group). Password sementara tampil sekali. Daftar akun menampilkan status (Active, Disabled, Locked until, Must change password) dengan aksi Edit roles, Reset password, Disable, dan Enable. Reset password dan Disable meminta konfirmasi. |
| Audit log (`/admin/audit`) | Admin (A-4) | Entri terbaru dulu, dengan kolom No., Time, By, Event, About, Details, dan Request ID. Ada tombol "Show older entries", dan tombol "Check the chain" yang memeriksa hash chain lalu melaporkan rantainya utuh atau putus di entri mana. |
| System settings (`/admin/settings`) | Admin (A-3) | Saklar "Sign date in every signature slot" (sekarang mati). |
| Customers (`/customers`) | Semua melihat; Admin (A-2) mengubah | Project groups (kode seperti GR1, nama). Customers: kode dua huruf, nama, Project Group, status Active atau Inactive, dan kode sebelumnya. Aksinya Edit dan Change code (kode lama tetap di riwayat). Ada juga formulir tambah customer. |
| Models and part numbers (`/models`) | Semua melihat; Admin mengubah | Pilih customer, lalu kelola Model groups (tambah, ganti nama), Models (tambah, ubah nama dan group), dan Part numbers (tambah, ubah; nama part opsional). |
| Format templates (`/format-templates`) | Semua melihat; Admin (A-2) mengubah | Pilih template (PFD, PFMEA AIAG 4th, CP, atau buatan sendiri). Terlihat jenis dokumen, metodologi, customer, metode risiko, dan versi (Version 1 · Draft). Tabel Header fields dan Columns: No., Label, Kind, Required, Hidden, Options, dengan aksi Edit, naik, dan turun. Bagian lain: Add a header field or column, Highlight rule (RPN di atas, severity paling sedikit), Rating tables (Severity, Occurrence, Detection nilai 1 sampai 10 dengan teks kriteria), Symbols (satu per baris), dan Add a format template (kosong atau salinan). Kolom disembunyikan, tidak pernah dihapus. |
| Processes (`/processes`) | Semua melihat; Template maintainer (E-6) mengubah | Daftar proses beserta aliasnya, tambah proses (nama, dan alias satu per baris), dan ubah. Nama dan alias berbagi satu ruang nama tanpa membedakan huruf besar kecil. |
| Process templates (`/process-templates`) | Semua melihat; Template maintainer (E-6) mengubah | Pilih proses. Bila belum ada template: teks "This process has no process template yet." dan tombol Add process template. Terlihat versi (Version 1 · Draft). Tabel Sub-processes: No., Sub-process, Points (jumlah per jenis dokumen, misalnya "PFD 1 · PFMEA 2 · CP 1"), Hidden, dengan aksi Points, Edit, naik, dan turun; formulir Add a sub-process. Setelah Points ditekan: judul "Points of {sub-proses}", satu tabel per PFD, PFMEA, dan CP (No., Text, Hidden, aksi Edit, naik, dan turun), dan formulir Add a point (Document type, Text). Tidak ada tombol hapus; yang tidak dipakai disembunyikan. |
| Document types and number codes (`/document-types`) | Semua melihat; Admin mengubah | Number settings: kode pabrik, pola nomor (`{prefix}-{customer}-{seq}`), digit revisi, dan keterangan format nomor berjalan. Document types: nama, huruf, awalan, dan tanda "Built in this system" untuk PFD, PFMEA, CP; bisa menambah jenis lain dari WI. Special codes: kode dan artinya (ZZ, HSE). |
| Change types, sources, and 4M (`/change-types`) | Semua melihat; Admin mengubah | Change types (ganti nama). Change sources (tambah, ganti nama, aktif atau nonaktif; sumber milik sistem selalu aktif). 4M categories (ganti nama). |
| Positions (`/positions`) | Semua melihat; Admin (A-1) mengubah | Jabatan beserta pemegangnya. Aksinya: ganti nama, tambah pemegang (orang, dan Project Group untuk jabatan per group), dan lepas pemegang. Label "One holder" atau "One holder per project group". |
| Approval routes (`/approval-routes`) | Semua melihat; Admin (A-2) mengubah | Rute beserta slotnya (No., Stage, Print title, Positions). Ubah rute (tambah, hapus, naikkan, atau turunkan slot) atau tambah rute baru. Default route per customer (bisa "Not set"). Route per change type (kosong berarti rute default customer). |
| Numbers (`/numbers`) | Semua melihat register; Author (E-1) mendaftar; Document control (E-8) mengubah status | **Formulir Register a number:**<br>- kotak "Register as a set: PFD, PFMEA, and CP with one number" (dicentang lebih dulu);<br>- Customer (hanya yang aktif), Document type (bila bukan set), dan Initial revision;<br>- pratinjau "Next number" untuk satu nomor;<br>- pesan hasil, beserta nomor yang dilewati.<br>**Register:**<br>- saringan Customer, Document type, Status, dan Number contains;<br>- kolom Number (dengan label revisi awal), Customer, Type, Status (dengan "Reserved for N days"), Registered by, Registered, dan Reason;<br>- aksi Void (alasan wajib) dan Make obsolete;<br>- tombol Show more.<br>Tautan Import legacy numbers tampil untuk pemegang E-8. |
| Import legacy numbers (`/numbers/import`) | Document control (E-8) | **Isian:** Customer, Status of these numbers, dan kotak teks "Numbers, one per line" (kolom pertama tempelan Excel).<br>**Preview menampilkan:**<br>- ringkasan;<br>- Lines not imported, dengan alasan: sudah ada di register, nomor sama dengan baris lain, kode customer bukan milik customer ini, atau bukan nomor menurut pola;<br>- Register after the import: jumlah, nomor tertinggi, dan lubang per kode dan jenis;<br>- Sets not aligned.<br>Tombol Import muncul setelah pratinjau dan hilang bila isian diubah. |

**Pola interaksi yang sudah dipakai** (pertahankan, atau ubah dengan sengaja):
- formulir tambah di atas daftar; pengubahan di baris yang sama (inline) dengan Save dan Cancel;
- pesan berhasil di atas formulir; masalah per field di bawah isian; error umum disertai request ID;
- konfirmasi browser untuk aksi berisiko (reset password, disable);
- hasil yang hanya tampil sekali (password sementara) ditampilkan jelas dengan petunjuk;
- daftar panjang dimuat per halaman dengan tombol "Show more" atau "Show older entries";
- pratinjau sebelum perubahan massal (impor nomor).

## 7. Data yang sudah disimpan

- **Master customer:** Project Group, customer, riwayat kode customer, group model, model, dan part number.
- **Master proses:** proses dengan ID tetap; nama tampilan dan alias.
- **Orang dan akses:** orang, akun (kerja atau admin), sesi, peran, katalog hak, dan peran per akun beserta cakupannya.
- **Jabatan dan rute:** jabatan, pemegang jabatan (per Project Group bila perlu), rute approval, slot, dan jabatan per slot. Rute default disimpan per customer, dan rute per jenis perubahan disimpan di jenis perubahan.
- **Master dokumen dan perubahan:** jenis dokumen, kode khusus, pengaturan nomor, jenis perubahan, sumber perubahan, kategori 4M, dan pengaturan sistem.
- **Register nomor:** penghitung, batch permintaan (satu nomor, set, atau impor), dan nomor dokumen.
- **Format template (M3.1):** template, versi, identitas field, field per versi, tabel skala, dan simbol.
- **Process template (M3.2):** satu template per proses, versi, identitas sub-proses dan poin, serta nama, teks, urutan, dan tanda tersembunyi per versi.
- **Audit log:** append-only dengan hash chain.
- **Belum ada:** dokumen, baris, revisi, cakupan dan penerbitan template, perubahan, lampiran, dan cetakan.

Diagram kelas domain (U5) dan ERD (U8) ada di `docs/uml/` sebagai sumber PlantUML.

## 8. Alur kerja dan layar yang akan datang

Desain UI/UX sebaiknya mengikuti alur di bawah. Yang ditandai (sudah) sudah berjalan; selebihnya rencana, dengan nomor milestone.

1. **Model baru:**
   1. daftarkan nomor sebagai set (sudah);
   2. buat dokumen dan rakit dari template di layar dua panel (M4.1, M4.5);
   3. edit di grid (M4.2, M4.3), atau tempel dari Excel (M4.4);
   4. atur urutan proses dan nomor operasi (M4.6);
   5. cek PFMEA dan CP berdampingan (M4.9);
   6. cetak untuk tanda tangan (M6.1 sampai M6.3);
   7. tanda tangan di kertas;
   8. catat rilis dengan scan (M6.7).
2. **Revisi yokoten (tujuan utama):**
   1. ajukan perubahan dengan sumber Yokoten dan lampirannya (M5.1, M5.2);
   2. verifikasi (M5.3);
   3. pilih target per customer, group model, model, proses, atau teks; setiap target mendapat draft revisi dan keputusan terapkan, sesuaikan, atau tidak berlaku, dengan papan status per model (M5.9);
   4. isi daftar periksa dampak (M5.7);
   5. periksa diff dan riwayat (M5.6);
   6. cetak dan rilis per dokumen.
3. **Revisi biasa dari permintaan** (PP, Customer claim, ECR, Audit finding): alurnya sama seperti nomor 2, untuk satu atau beberapa dokumen.
4. **Template dan SQAM surut:**
   1. terbitkan versi template (M3.4);
   2. pratinjau dampak;
   3. buat perubahan dengan draft revisi untuk dokumen terdampak (M5.8, M5.10).
5. **Migrasi dokumen lama:** form dokumen lama, riwayat revisi sebagai teks, pembanding Excel, penautan baris, dan verifikasi hardcopy (M7.1 sampai M7.5).
6. **Dasbor ukuran** (M7.7) dan halaman verifikasi QR (M6.5, saklar).

**Layar baru yang paling berat untuk didesain:**

| Layar | Milestone | Catatan |
|---|---|---|
| Editor grid dokumen | M4.2 | Banyak kolom, ukuran A3, dikelompokkan per proses, simbol, dan penguncian optimistik |
| Perakitan dua panel | M4.5 | Pohon template dengan kotak centang, dan struktur model dengan drag and drop |
| Papan status yokoten | M5.9 | Banyak model sekaligus, dengan keputusan per dokumen |
| Diff revisi | M5.6 | Baris dan sel, per dokumen dan per proses |
| PFMEA dan CP berdampingan | M4.9 | Per proses, dengan tanda dicek |
| Pratinjau cetak dan blok tanda tangan | Tahap 6 | Dibentuk dari rute |
| Catat rilis | M6.7 | Unggah scan, konfirmasi slot per urutan |
| Formulir permintaan perubahan | M5.1 sampai M5.3 | Dengan lampiran, verifikasi, dan daftar periksa dampak |

## 9. Aturan yang wajib dihormati desain

- **Tanda tangan basah:** tidak ada tombol approve untuk approver, karena persetujuan terjadi di kertas (D-07).
- **Tidak ada hapus untuk data penting:** akun, customer, proses, nomor, dan revisi rilis hanya bisa dinonaktifkan, Void atau Obsolete, atau diganti revisi baru.
- **Nomor dialokasikan server.** Layar tidak boleh membiarkan pengguna mengetik nomor baru sendiri; "Next number" hanya pratinjau.
- **ID tetap, nama hanya tampilan:** proses, jabatan, kolom format template, dan baris dokumen. Mengganti nama tidak memutus rujukan.
- **Perubahan template dan SQAM selalu lewat revisi yang disetujui,** tidak pernah langsung mengubah dokumen.
- **Revisi rilis tidak bisa diedit.** Perubahan berarti revisi baru.
- **Pemisahan tugas:**
  - pemohon bukan verifikator;
  - penyusun bukan pengecek;
  - pencatat rilis bukan penyusun.
- **Cakupan Project Group:** Project Leader dan Project Engineer hanya bekerja untuk customer di Project Group sendiri.
- **Istilah layar mengikuti glosarium (Lampiran C), dalam bahasa Inggris.** Bahasa Indonesia mungkin ditambah nanti, jadi teks layar jangan ditulis langsung di desain komponen.
- **Hak dicek server.** Menyembunyikan tombol hanya kenyamanan, bukan pengamanan.

## 10. Keputusan, temuan, dan pertanyaan terbuka

### 10.1 Keputusan yang paling memengaruhi desain

Teks lengkap semua keputusan ada di Lampiran A.

- **Approval, layar, dan bahasa:**
  - D-07: approval tanda tangan basah.
  - D-92: user flow dan wireframe Tahap 4 dan 5 dibuat sebelum Tahap 4.
  - D-96: bahasa layar Inggris, dari satu katalog teks.
  - D-98: glosarium U1 memuat istilah layar.
- **Akses:**
  - D-66: katalog hak tetap, isi peran diatur admin, empat level.
  - D-78: delapan peran dari peta pengguna.
  - D-71, D-81, D-82, D-118: cakupan Project Group.
  - D-72, D-120: akun admin terpisah.
  - D-73, D-105: akun tidak pernah dihapus.
  - D-75: audit log hanya lewat akun admin.
  - D-103: master data terlihat lewat L-2.
  - D-104: setiap akun mengurus akunnya sendiri.
- **Master data:**
  - D-06, D-110: master proses ber-ID tetap, satu ruang nama.
  - D-58: jabatan ber-ID tetap, boleh rangkap.
  - D-54, D-115: Project Group dengan Project Leader dan Project Engineer; approver satu pemegang.
  - D-86, D-88, D-109: customer tidak dihapus, kode bisa diganti dengan riwayat.
- **Penomoran:**
  - D-89: struktur nomor.
  - D-93: penghitung per teks kode customer.
  - D-114: kode khusus.
- **Template:**
  - D-39: empat tingkat template, yang paling spesifik menang.
  - D-40: group model sebagai target yokoten.
  - D-42: satu model paling banyak di satu group.
- **Dokumen dan perubahan:**
  - D-02: baris datar.
  - D-03: kesesuaian PFMEA dan CP diperiksa manusia.
  - D-04: satu entitas perubahan.
  - D-05: perubahan surut dan template selalu menjadi revisi baru lewat approval.
  - D-10: layar perakitan dipakai juga untuk migrasi.
- **Rute dan cetak:**
  - D-45 sampai D-49, D-117: rute approval sebagai data.
  - D-47, D-116: rute default per customer, dan "Not set" menolak cetak.
  - D-112: rute per jenis perubahan.
  - D-52, D-53, D-59: tanggal tanda tangan per slot mati; tanggal rilis terisi saat cetak.
  - D-62: QR mati.
  - D-76: judul cetak slot.
- **Lainnya:**
  - D-30: dokumen dikembalikan dicatat penyusun.
  - D-34: approval customer di luar sistem.
  - D-31: field internal dan filter visibilitas customer ditunda.
  - D-85: aturan login.

### 10.2 Temuan desain yang masih terbuka

Disalin dari `docs/evaluasi-desain.md`. Kode memakai asumsi paling aman untuk setiap temuan, ditandai `ASUMSI E-nn`.

| ID | Temuan | Jenis | Status |
|---|---|---|---|
| E-30 | Field dan aturan Project Group, group model, model, dan part number belum ditetapkan | Celah | Ditunda (menunggu bahan) |
| E-33 | Pencocokan PFD, PFMEA, dan CP: di tingkat proses atau sub-proses | Ambigu | Ditunda (menunggu bahan) |
| E-35 | Bentuk kebijakan approval per jenis perubahan belum ditentukan (Q-15) | Celah | Diputuskan sebagian (D-112) |
| E-37 | Kode khusus: bentuk, penambahan, dan tumpang tindih dengan kode customer | Celah | Diputuskan sebagian (D-114) |
| E-44 | Nomor berjalan pertama: 000 atau 001 | Ambigu | Terbuka |
| E-54 | Tanggal di kepala milestone tidak sesuai dengan tanggal brief v1.15 | Kontradiksi | Ditunda (sesi desain) |
| E-60 | Jadwal, retensi, dan penyimpanan passphrase backup belum ditetapkan | Celah | Terbuka |
| E-61 | Satu pola nomor untuk semua jenis dokumen dan customer | Ambigu | Terbuka |
| E-62 | Isi register dan pengingat nomor Reserved yang terlalu lama | Celah | Terbuka |
| E-63 | Kunci idempoten di UI API: cakupan, umur, dan pemakaian ulang | Celah | Terbuka |
| E-64 | Nomor yang dilewati saat daftar sebagai set | Kontradiksi | Terbuka |
| E-65 | Bentuk impor nomor lama dan isi laporannya | Ambigu | Terbuka |
| E-66 | Nomor lama yang tidak bisa masuk apa adanya: bentuk lain dan nomor ganda | Celah | Terbuka |
| E-67 | Pendaftaran nomor untuk customer nonaktif | Celah | Terbuka |
| E-68 | Perubahan status nomor yang dikerjakan tangan di Tahap 2, dan alasannya | Ambigu | Terbuka |
| E-69 | Urutan kunci transaksi penomoran: deadlock dengan ganti kode customer | Sulit dibangun | Terbuka |
| E-70 | Isi awal format template dan aturan field yang belum ditetapkan | Celah | Terbuka |
| E-71 | Aturan sorot PFMEA: arti "severity 8" dan nilai mana yang disorot | Ambigu | Terbuka |
| E-72 | Satuan process template: satu per proses, atau satu pustaka untuk semua proses | Ambigu | Terbuka |
| E-73 | Isi poin process template dan format template yang diikutinya | Celah | Terbuka |

### 10.3 Pertanyaan terbuka di brief

Pertanyaan yang belum selesai, beserta asumsi kerjanya (bahasa Inggris, dari brief bagian 12):

| ID | Question | Working assumption |
|---|---|---|
| Q-01 | Do customers and auditors accept electronic approval, and what is the legal position in Indonesia? | Manual wet signature only. |
| Q-02 | Active Directory or local accounts? What does IT require (server request process, security checklist, AD policy)? | Local accounts. |
| Q-03 | Hosting target, server specification, backup location. | On-premise Docker, specification to be defined. |
| Q-04 | Who owns Template General and who approves template changes? | A template owner role. Affected documents still go through normal approval. |
| Q-05 | Final numbering pattern per customer. Any customer with a special pattern? Counters per document type, or shared across a set? How are numbers skipped by "register as set" recorded (R-NUM-05)? | Per document type with a "register as set" action. |
| Q-06 | Do all legacy Excel documents have a complete revision history table? | Most do. Missing ones start with an empty text history. |
| Q-07 | What do real SQAM formats look like, and how are retroactive changes communicated (effective dates, evidence)? | Collect samples first (milestone S3). |
| Q-08 | Which Excel forms, fonts, and paper sizes are used in practice? | A3 landscape, A4 as an option. |
| Q-09 | How many customers, models, and documents must be migrated? Which pilot models? | Pilot of 2 to 3 models of one customer. |
| Q-10 | Which PDF route (A, B, C, or D)? | Decided after S5. |
| Q-11 | Does a document control role exist? Will approvers ever use the web? | Document control records releases. |
| Q-12 | Do customers require 4M change notification? | Optional field. |
| Q-13 | Keep real-time push updates and dashboards from the earlier spec? | Drop for now. Optimistic locking only. |
| Q-14 | PARTLY RESOLVED 2026-10-28: the screens are in English for now, from one text catalog (D-96). File naming conventions are still open. | Code comments in Indonesian. |
| Q-15 | Which approval steps apply to content, layout-only, and administrative changes? | Same flow for all until defined. The data shape is a route per change type (D-112). |
| Q-16 | QCC schedule, deadline, and baseline measurements. | None. Baselines to be measured by the owner. "After" numbers come from timed demo runs (D-36). |
| Q-18 | Retention and archive rules for obsolete revisions and attachments. | Keep everything. |
| Q-19 | More than one plant or line? Numbering per plant? | Single plant. |
| Q-20 | Company data classification rules for customer-confidential documents. | Treat all documents as confidential. |
| Q-21 | Real roles, headcount per role, and concurrency. | 10 to 50 concurrent users, the eight roles of U3-peta-pengguna (D-78). |
| Q-22 | Which internal application connects first, what exactly does it write (data, frequency, trigger), and who owns it? | Writes are change proposals. Candidate list in R-API-08. |
| Q-23 | Can one customer account see several customer codes (for example a customer group)? Who administers customer accounts? | One customer, one code. Company admin manages accounts. |
| Q-24 | Portal policy: may customers see history or older revisions? Hosting model (separate DMZ portal, same application with row-level security, or VPN)? Any customer requirements of their own? | Latest released revision only, no history, separate DMZ portal preferred. |
| Q-25 | Which application-to-application authentication does IT prefer (tokens, OAuth2 client credentials, mutual TLS, an API gateway)? | Bearer tokens of service accounts. |
| Q-26 | Notice period for breaking API changes. | Six months. |
| Q-28 | Should customers be notified when a new revision is released? | No notifications. |
| Q-29 | Which fields (columns, header fields) must be hidden, and for which customers? | Standard columns visible, custom columns hidden until marked visible. |
| Q-30 | May display permission be given before approval is complete (after_check), or only after release is recorded (after_release)? | After release only. |
| Q-31 | Should the verification page need a login, and which details does it show (roles or names of those who permitted display)? | Login required. Roles, not names. |
| Q-32 | Who are the preparer and the checker who give display permission: the persons in the Prepared and Checked slots of the revision's route, or anyone entitled in the authority matrix? | The persons in the Prepared and Checked slots of the revision's route (D-45). |
| Q-33 | Do customers accept an unsigned, system-verified PDF as the effective document? | Not confirmed. Keep the default policy (scan then PDF) until confirmed per customer. |
| Q-44 | PARTLY ANSWERED 2026-10-24: not decided yet. IT will probably review after the whole design is finished. Who reviews is still open. | The owner reviews with the spec-reviewer agent until IT names a reviewer. |
| Q-46 | What defines a model group (for example the same PCB platform or product family), and who creates and maintains groups? | The admin account maintains groups for now (D-87), with no fixed definition. |

## 11. Rencana tahap dan statusnya

| Tahap | Isi | Status |
|---|---|---|
| S | Matangkan spesifikasi dan UML, sejajar dengan kode | Berjalan |
| 0 | Persiapan: alat, server Go pertama, PostgreSQL, SvelteKit, irisan customer | Selesai |
| 1 | Fondasi: login, titik pemeriksaan izin, audit log, master data, jabatan dan rute approval, backup terenkripsi | Selesai |
| 2 | Penomoran: pola, penghitung atomik, register, set, impor nomor lama | Selesai (Gerbang 2 menunggu pemilik) |
| 3 | Template: format template, pustaka proses, empat tingkat cakupan, versi template | Berjalan: M3.1 dan M3.2 selesai (M3.2 dengan ASUMSI E-33); berikutnya M3.3 |
| 4 | Dokumen dan perakitan: baris, editor grid, penguncian, tempel Excel, perakitan dua panel, urutan, perbedaan lokal, cek berdampingan | Belum |
| 5 | Perubahan, revisi, dan riwayat: entitas perubahan, lampiran, verifikasi, draft revisi, snapshot, diff, daftar periksa dampak, propagasi, yokoten, SQAM | Belum |
| 6 | Cetak dan rilis manual: layout sebagai data, Excel, PDF, watermark, QR dan hash, catatan cetak, catat rilis, tabel riwayat revisi | Belum |
| 7 | Migrasi dan bukti nilai: form dokumen lama, pembanding, penautan baris, verifikasi hardcopy, pilot, dasbor | Belum |
| 8 | Penguatan dan serah terima: peran dan hak rinci, unggahan aman, deploy, dokumen IT, presentasi QCC | Belum |
| 9 | API integrasi (latihan, setelah MVP) | Belum |
| 10 | Portal customer (latihan) | Belum |

## 12. Saran untuk sesi desain UI/UX (usulan)

Bagian ini usulan Claude Code, bukan keputusan.

1. **Mulai dari struktur navigasi.** Menu sekarang datar (sembilan menu kerja ditambah menu admin). Usulan pengelompokan:
   - Documents (Tahap 4 sampai 6);
   - Changes (Tahap 5);
   - Numbers;
   - Master data: Customers, Models, Processes, Document types, Change types, Positions, Approval routes;
   - Admin: Accounts, Audit log, Settings.
2. **Buat design system kecil lebih dulu:**
   - warna;
   - tipografi;
   - jarak;
   - tabel dengan saringan;
   - formulir dengan pesan per field;
   - tombol utama dan sekunder;
   - konfirmasi;
   - pesan berhasil dan error;
   - lencana status nomor (Reserved, In use, Released, Obsolete, Void, Skipped);
   - penanda "hanya baca" untuk pengguna tanpa hak ubah.
3. **Rancang ulang dulu dua layar yang sudah ada:** Numbers dan Import legacy numbers. Keduanya sudah lengkap fungsinya, jadi cocok untuk menguji design system.
4. **Lalu wireframe layar Tahap 4 dan 5** sesuai D-92: editor grid, perakitan dua panel, permintaan perubahan, papan yokoten, dan diff.
5. **Prioritaskan desktop dan layar lebar** untuk editor dokumen ukuran A3. Halaman verifikasi QR dipakai QA, QC, dan internal audit, dan pemimpin ke atas boleh membawa ponsel di area produksi (D-32), jadi halaman itu perlu nyaman di layar kecil.
6. **Aset dibundel bersama aplikasi** (font, ikon), bukan diambil dari CDN, karena aplikasi self-hosted di jaringan internal.
7. **Simpan hasil desain** (user flow, wireframe, dan keputusan desain) di repo bersama paket UML (`docs/uml/`) atau di folder desain baru. Istilah baru yang muncul dari desain dicatat sebagai usulan untuk glosarium U1.

## Lampiran A. Semua keputusan (D-01 sampai D-120)

Teks keputusan disalin dari brief bagian 6 (bahasa Inggris), tanpa kolom sumbernya. Nomor dokumen internal diganti keterangan umum.

| ID | Decision | Status |
|---|---|---|
| D-01 | Keep Go + PostgreSQL + SvelteKit even though it means learning two new technologies. | [DECIDED] |
| D-02 | Flat rows with JSONB content instead of the normalized failure-mode, effect, cause structure. | [DECIDED] |
| D-03 | PFMEA to CP consistency is checked manually. | [DECIDED] |
| D-04 | One unified change entity. | [DECIDED] |
| D-05 | Retroactive and template-driven updates always create new revisions through approval. | [DECIDED] |
| D-06 | Process master with fixed IDs. Names are display only. | [DECIDED] |
| D-07 | Wet-signature approval is primary. In-app approval is design only. | [DECIDED] |
| D-08 | PDF route is decided after a test. | [DECIDED] |
| D-09 | Local accounts first, internal only, roles later. | [DECIDED] |
| D-10 | Composition screen serves both new and migrated documents. | [DECIDED] |
| D-11 | Specification first, with UML, before any code. | [DECIDED], amended by D-77 |
| D-12 | Milestones are small and used as learning material. | [DECIDED] |
| D-13 | Layouts are written by developers now but stored as data. | [DECIDED] |
| D-14 | VDA support is form compatible for now and will be re-evaluated. | [DECIDED] |
| D-15 | The API is designed before building, because the first internal consumer must write into the system, not only read. | [DECIDED] |
| D-16 | Three API surfaces: UI API, integration API, customer portal. | [PROPOSED] |
| D-17 | External writes are change proposals only. They never release, verify, or approve. | [PROPOSED] |
| D-18 | The customer portal is a future read-only feature: only the internal team writes, and customers can only read and download their own documents. | [DECIDED] as a future feature |
| D-19 | No integration token is issued before roles and customer scope are enforced (M8.1). | [PROPOSED] |
| D-20 | Portal v1 is a document register plus PDF download, with no in-browser row view. | [PROPOSED] |
| D-21 | By default customers receive the signed scan when it exists, otherwise the system-generated PDF. Never Excel. The default can be switched per customer (D-25). | [DECIDED] |
| D-22 | A visibility filter lets the company hide selected information from customers. | [DEFERRED] (D-31), was [DECIDED] |
| D-23 | The filter works at field level (columns and header fields) with per-customer profiles. Row and process hiding are not supported in v1. | [DEFERRED] (D-31) |
| D-24 | The release print used for signature uses the customer-audience layout. Internal-only information stays web-only or on internal prints. | [DEFERRED] (D-31) |
| D-25 | The customer file source is a per-customer policy that can be switched: scan then PDF (default), scan only, or generated unsigned PDF only. | [DECIDED] as a requirement |
| D-26 | Customers see a new revision only after internal roles (default: preparer and checker) give display permission in the web system. Until then the portal keeps showing the old file, even if editing is finished and the revision is printed for manual approval. | [DECIDED] |
| D-27 | Display permission is separate from approval and release, and is bound to the content hash of the customer-visible content. | [PROPOSED] |
| D-28 | Showing a revision before approval is complete (earliest stage after_check) is opt-in per customer. The default is after release. | [PROPOSED], see Q-30 |
| D-29 | Develop with real company and customer documents, using Claude Code connected to a private GitHub repository. Replaces the earlier proposal to sanitize data. | [DECIDED] |
| D-30 | When the approver asks for changes on the paper copy, the preparer records the return in the system. | [DECIDED] |
| D-31 | Internal-only fields and the customer visibility filter are on hold (section 18, D-22 to D-24, R-PRT-05). Customer outputs carry all fields until the feature resumes. | [DECIDED], feature [DEFERRED] |
| D-32 | QR verification users are QA, QC, and internal audit. Leaders and above may carry phones in production areas. | [DECIDED] |
| D-33 | The owner's code is personal and serves learning, not company development. Company IT will use the UML to develop its own version. | [DECIDED] |
| D-34 | Customer approval of changes (4M, PPAP resubmission) stays outside the system, as today. | [DECIDED] |
| D-35 | Coarse roles are enforced from milestone M1.2, amending R-SEC-02. | [DECIDED] |
| D-36 | The personal version runs on localhost. For now it is a demo, used to mature the concept and the workflow of the website. | [DECIDED] |
| D-37 | All code is practice, including stages 9 and 10. For the practice build, a stage starts when the previous stage is demonstrated and its gate is passed. Conditions that need real users, management, legal, or IT (Q-22, RK-33, RK-35) apply to IT's version only. | [DECIDED] |
| D-38 | The UML package for IT follows current best practice: UML 2.5.1, the C4 model, a crow's foot ERD, OpenAPI, decision records, acceptance scenarios, and a traceability matrix, as diagrams as code (section 20). | [DECIDED] by delegation |
| D-39 | Process templates exist at four levels: general, customer, model group, and model. | [DECIDED] |
| D-40 | A model group can be chosen as a yokoten target, so one choice reaches every model in the group. | [DECIDED] |
| D-41 | A read-only viewer role exists for QA, QC, and internal audit (R-APR-06). | [DECIDED] |
| D-42 | A model group belongs to exactly one customer, and a model belongs to at most one group. | [DECIDED] |
| D-43 | The work starts with the UML (milestone S2), built directly from this brief. | [DECIDED], extended by D-77 |
| D-44 | Only stable versions of the tech stack are used. | [DECIDED], criteria [PROPOSED] |
| D-45 | Signature stages and positions: Prepared by the PPAP Engineer; Checked by the Project Leader, the Production Engineer, or the Project Leader Supervisor; Approved by the NPI Manager or PE Manager (NPI Manager today), the QA QC Manager, and the GM. | [DECIDED], names refined by D-54 to D-56 |
| D-46 | The signature page is flexible: the approval route ends either at the QA QC Manager or at the GM. | [DECIDED] |
| D-47 | Each customer has a default route. The preparer may override it per document with a recorded reason. | [DECIDED] |
| D-48 | The Checked stage has one signature, from any one of the three checker positions. | [DECIDED], positions refined by D-54 (two positions) |
| D-49 | Signing order is sequential: Checked, NPI or PE Manager, QA QC Manager, GM. | [DECIDED] |
| D-50 | Change requests (R-CHG-02) are verified by the Project Leader Supervisor only. | [SUPERSEDED] by D-51 |
| D-51 | Each customer has one Project Leader and one Project Leader Supervisor. A change request is verified by any one of three people: that customer's Project Leader, that customer's Project Leader Supervisor, or the PPAP Supervisor. The system records who verified. | [DECIDED], positions refined by D-54 |
| D-52 | Per-slot sign dates are prepared with a switch to turn them off and hide them. For now only the release date is recorded. | [DECIDED] |
| D-53 | The release date (tanggal rilis) is the date the document is created or printed. | [DECIDED] |
| D-54 | Position names follow the NPI organization chart. The earlier "Project Leader / Production Engineer" is the Project Engineer (Leader level), and the earlier "Project Leader Supervisor" is the Project Leader (Supervisor level). Each project group has one of each, and each customer is allocated to one project group. The PPAP Supervisor is the APQP/PPAP Controller (Supervisor level, one person, today held concurrently by the Project Leader of GR3). | [DECIDED] |
| D-55 | PE Manager is not a separate position. PE (Production Engineering) was the earlier name of a department with the same role as NPI, so PE Manager is only a possible future name of the NPI Manager. PPAP and the Project Leaders sit under NPI. | [DECIDED] |
| D-56 | The final Approved slot (GM) is held by the Advisor today and may be held by the Factory Manager later. | [DECIDED] |
| D-57 | The system keeps the term PFD, as the WIs and procedures do. PFC is recorded as an alternative name. | [DECIDED] |
| D-58 | Positions have fixed IDs and editable names, and one person may hold more than one position (R-APR-15). | [DECIDED] |
| D-59 | The release date is filled in when the document is printed for signature. The effective date equals the release date for now. The release date belongs to DCC, so this rule must be changeable later. | [DECIDED] |
| D-60 | The document control role is held by the PPAP Staff today and may move to DCC later. | [DECIDED], extended by D-68 |
| D-61 | Scope stays PFD, PFMEA, and CP. The system is API-first, so other applications can be combined later. | [DECIDED] |
| D-62 | The QR code on documents is optional, with an on and off switch. It is off for now because QR codes are not used today. | [DECIDED] |
| D-63 | Preparing documents is allowed for the PPAP Staff, the PPAP Engineer, and the Project Leader. | [DECIDED], extended by D-67 |
| D-64 | The PPAP Engineer maintains the process master and the process templates. | [DECIDED] |
| D-65 | The PPAP Engineer sets the "checked by, date" mark in the PFMEA to CP comparison for now. | [DECIDED] |
| D-66 | The admin can change users' access rights, so access can be granted later (for example edit rights for DCC) without code changes. Roles are first mapped into create and edit, verify, and view only. | [DECIDED] |
| D-67 | The Project Engineer may also prepare documents. "Project leader" in the answer to Q-55 meant both the Project Leader (Supervisor level) and the Project Engineer. | [DECIDED] |
| D-68 | The document control role is also given to the PPAP Engineer, so a release is always recorded by someone other than the preparer. | [DECIDED] |
| D-69 | The admin role and the migration role are held by the PPAP Staff and the PPAP Engineer for now. | [DECIDED] for the migration role; admin part [SUPERSEDED] by D-72 |
| D-70 | Every position up to GM level gets a viewer account, including the approvers (NPI Manager, QA QC Manager, Advisor, Factory Manager), DCC, QA, QC, internal audit, and the WI team. | [DECIDED] |
| D-71 | The Project Leader and the Project Engineer prepare documents only for the customers of their own project group. | [DECIDED] |
| D-72 | The admin has a separate admin account. For now the PPAP Staff holds it, so that person has two different accounts. The audit log is viewed through the admin account. | [DECIDED] |
| D-73 | The last admin account cannot be removed and cannot lose its admin role. "Removed" means disabled, because accounts are never deleted (D-105). | [DECIDED] |
| D-74 | Each admin account is linked to its named holder, and the audit log shows both the account and the holder. | [DECIDED] |
| D-75 | The NPI Manager does not get a view of the log of access changes. The audit log, access changes included, stays visible only through the admin account (D-72). | [DECIDED] |
| D-76 | Covers and print titles follow the covers of the current documents: each slot prints its print title ("Process. Eng", "Engineer", "NPI Manager", "QA / QC Mgr.", "General Mgr.") with the holder's name. | [DECIDED] |
| D-77 | Coding starts now, in parallel with the remaining UML work (U4 to U14). The code is built milestone by milestone with Claude Code and is used to evaluate the design. | [DECIDED] |
| D-78 | The eight roles of U3-peta-pengguna and the permission catalog L-1 to A-4 are stored as data from M1.2, seeded from the U3 matrix. No approver role until in-app approval (R-APR-04). Amends R-SEC-09 and the role list in section 1. | [DECIDED] |
| D-79 | Person and account are separate entities. A person (name, optional employee number) has at most one work account and one admin account. Positions are held by persons, and separation of duties compares persons. | [DECIDED] |
| D-80 | Stage 1 does not wait for U5 and U6. The UML is updated from the code as it is built. Replaces the mitigation proposed in RK-64. | [DECIDED] |
| D-81 | The project group scope (D-71) is checked in the permission function from M1.2. Row-level security stays in M8.1. Amends R-SEC-02 and R-SEC-12. | [DECIDED] |
| D-82 | The project group master and its position holders are the single source of the Project Leader and Project Engineer scope. A role grant only says "all customers" or "own project group". | [DECIDED] |
| D-83 | One hash chain for the whole audit log. The chain tip is locked, entries are written by the Go service layer in the same transaction as the change, and two database roles exist: the owner for migrations, and an application role without UPDATE and DELETE on the audit log. | [DECIDED] |
| D-84 | The first admin account is created with a one-time command on the server machine (`revioten create-admin`), recorded with the system actor. | [DECIDED] |
| D-85 | Login and session rules come from open standards (OWASP, ASVS 5.0, NIST SP 800-63B-4): argon2id, passwords of 15 to 128 characters checked against a blocklist, a 30-minute idle and an 8-hour absolute session limit, a 15-minute lock after every 10 failed logins, disabling at the 100th failure except for admin accounts (D-101), and a reset by the admin with a forced change. The exact values are in E-08. | [DECIDED] |
| D-86 | Customer codes are unique, two letters A to Z, and never equal a special code. Customers are never deleted, only deactivated. Each customer belongs to one project group, required from M1.4. | [DECIDED] |
| D-87 | The masters of M1.4 (customer, project group, model group, model, part number) are managed by the admin account (A-2). Who should own model groups later stays open (Q-46). | [DECIDED] |
| D-88 | A customer code can be changed through master data. Old codes stay in a history with validity dates; old numbers keep their old code (R-NUM-02) and match the customer the code belonged to. After a change, the running number of the new code starts again, and active documents move to new numbers by hand (D-93). | [DECIDED] |
| D-89 | The number structure follows the numbering WI and is kept as master data: the factory code (S) and the document type letter (H for PFD, F for PFMEA, C for CP) form the prefix, then the customer code or a special code (ZZ, HSE), then the running number (three digits up to 999, then A001 to A999, B001, and so on, "for now"), and a two-digit revision outside the registered number. An ESD control plan is not a process control plan; the system does not number such documents and records them as referenced documents (R-DOC-10, R-DOC-11). | [DECIDED] |
| D-90 | Use the newest stable or LTS versions. `docs/versions.md` holds the exact versions with dates and sources; section 7 names only the lines. | [DECIDED] |
| D-91 | R-VER-04 exception: a line with a single release may be used when no other line satisfies the peer dependencies and the release is at least 14 days old. A 0.x package uses its newest release that is at least 14 days old when first installed. Exceptions are recorded in `docs/versions.md`. | [DECIDED] |
| D-92 | User flows and wireframes for the screens of stages 4 and 5 are made before stage 4 and stored with the UML package. The UI language is decided first (D-96). | [DECIDED] |
| D-93 | After a customer code change, an active document moves to a new number through a document control action, per document: the revision continues, the old number becomes Obsolete with a link to the new one, and the move needs a new revision with new signatures. Counters are kept per customer code text and document type. | [DECIDED] |
| D-94 | `/health` stays at the root as an operational endpoint, outside `/api/v1`, and is documented in the OpenAPI contract. | [DECIDED] |
| D-95 | Passwords, tokens, keys, and connection strings with passwords are never written to logs, error messages, or the audit log. Error messages to users carry no internal details; the details go to the server log with the request ID. Tests prove it (R-SEC-14). | [DECIDED] |
| D-96 | The screens are in English for now. All screen texts sit in one catalog, so Indonesian can be added with the same keys. | [DECIDED] |
| D-97 | TypeScript types for the OpenAPI contract are generated with @hey-api/openapi-ts (types plugin only); the fetch calls are written by hand. openapi-typescript 7.13.0 fails R-VER-03 with TypeScript 6. | [DECIDED] |
| D-98 | The glossary U1 gets a column with the English screen term, and the screen text catalog follows it. | [DECIDED] |
| D-99 | The earlier spec package is not available, so the error codes built in M0.5 are the baseline for R-API-13. | [DECIDED] |
| D-100 | Customer name: required, at most 200 characters, one line, and may repeat. Customer code: unique, two letters A to Z. | [DECIDED] |
| D-101 | A forgotten or locked last admin account is recovered with the command `revioten reset-admin-password` on the server machine, recorded with the system actor. Admin accounts are not disabled by failed logins; the 15-minute lock still applies (R-SEC-15). | [DECIDED] |
| D-102 | The demo checks new passwords against an offline list of common passwords. Checking breached passwords through the Pwned Passwords API is discussed with IT for the company version. | [DECIDED] |
| D-103 | Viewing master data is part of permission L-2. | [DECIDED] |
| D-104 | Every account may manage itself (sessions, logout, password). This is a fixed rule outside the permission catalog, still checked by the permission function. | [DECIDED] |
| D-105 | Accounts are never deleted, only disabled. "Removed" in D-73 means disabled. | [DECIDED] |
| D-106 | Two more actor types for the audit log: system (a command on the server machine) and anonymous (not logged in). Neither has an account, and the permission function always refuses them. | [DECIDED] |
| D-107 | The audit log records every change of data, including sessions and failed logins; reading is not recorded. Exports and prints are added in stage 6. | [DECIDED] |
| D-108 | The demo runs migrations at program start with one `.env`. The document for IT recommends a separate migration step, owner credentials only at deploy time, a daily chain check, and an external chain anchor. | [DECIDED] |
| D-109 | A customer code belongs to one customer only, ever, never returns to a customer, and takes effect when recorded. | [DECIDED] |
| D-110 | One process master for all customers. Names and aliases share one case-insensitive namespace, with at most 200 characters and 20 aliases. Processes are never deleted, and an old name does not become an alias by itself. | [DECIDED] |
| D-111 | Contract fields whose empty value would change data (for example `active`) are optional and mean "unchanged" when absent. A middleware that validates requests against the contract is decided before M8.1. | [DECIDED] |
| D-112 | The approval policy per change type (R-TPL-10) is a route per change type; empty means the customer default route. Which route wins when the preparer chooses another route (R-APR-09) is still open, to be answered before stage 5. | [DECIDED], precedence [OPEN] |
| D-113 | Change sources stay one list for now; entries used by system mechanisms (yokoten, template propagation, SQAM) are marked and always active. At stage 5 a change gets two fields: mechanism and trigger. | [DECIDED] |
| D-114 | Special codes have two or three letters, the admin may add them, a code cannot be changed, and it never equals a customer code, now or before. Whether PFD, PFMEA, and CP may be numbered with a special code is still open. | [DECIDED], numbering with special codes [OPEN] |
| D-115 | The approver positions (NPI Manager, QA QC Manager, Advisor, Factory Manager) and the APQP/PPAP Controller have one holder each. The Project Leader and the Project Engineer have one per project group. Other positions may have several. The holder history is the audit log. | [DECIDED] |
| D-116 | A customer may have no default route yet ("Not set"). Printing for signature is refused until a route is set, unless the preparer chooses another route with a reason. | [DECIDED] |
| D-117 | A route has exactly one Prepared slot first, one Checked slot next, then one or more Approved slots; at most 10 slots; print titles of at most 100 characters. Revisit after Q-15. | [DECIDED] |
| D-118 | Project-group-scoped rights do not apply to operations without a customer. Lists across customers filter per customer. The yokoten target search (M5.9) gets the allowed customers from the permission function, so scoped users keep finding targets in their own project group. | [DECIDED] |
| D-119 | Local backup (M1.7) is built before M2.1: encrypted backups to another disk, the audit chain tip copied into the backup manifest, and a tested restore into an empty database. Disk encryption of the laptop is the owner's step. | [DECIDED] |
| D-120 | The admin account holds the admin rights (A-1 to A-4) plus the basic view rights every account has (L-1 to L-3), and never holds work roles. Amends R-SEC-13. | [DECIDED] |

## Lampiran B. Daftar use case (U3)

Disalin dari `docs/uml/U3-daftar-use-case.md`, ditambah kolom status kode per 31 Oktober 2026.

### A. Akses dan pengaturan

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-01 | Masuk dan keluar sistem | Pengguna internal | Login lokal, sesi, semua halaman butuh login. | M1.1 | MVP | Sudah |
| UC-02 | Kelola pengguna, peran, dan jabatan | Admin | Akun, peran, dan jabatan yang dipegang, termasuk rangkap jabatan. Admin juga mengubah isi peran (hak akses), sesuai peta pengguna. | M1.2, M1.8, M8.1 | MVP | Sudah |
| UC-03 | Kelola customer, Project Group, group model, model, dan part number | Admin | Master data dengan kode customer, alokasi ke Project Group, dan satu group per model. | M1.4 | MVP | Sudah |
| UC-04 | Kelola master proses | PPAP Engineer | ID tetap, nama tampilan, dan alias. | M1.5, D-64 | MVP | Sudah |
| UC-05 | Kelola jenis dokumen, jenis perubahan, dan sumber perubahan | Admin | Awalan SH, SF, SC, kebijakan approval per jenis perubahan, dan pemicu PP, customer claim, ECR. | M1.6 | MVP | Sudah |
| UC-06 | Kelola rute approval | Admin | Slot berurutan, jabatan per slot, judul cetak, dan rute default per customer. | M1.8 | MVP | Sudah |
| UC-07 | Kelola pengaturan sistem | Admin | Saklar QR, tanggal tanda tangan per slot, aturan tanggal berlaku, dan mode migrasi. | M1.8, M6.5, M7.5 | MVP | Sebagian: saklar tanggal tanda tangan |
| UC-08 | Lihat audit log | Admin | Menelusuri siapa melakukan apa dan kapan. Hanya lewat akun admin. Setiap entri tindakan admin menampilkan akun dan pemegangnya. | M1.3 | MVP | Sudah |
| UC-09 | Kelola master dokumen terkait | Admin | Nomor dan judul WI, SOP, QEP yang dirujuk CP. | M4.1 | Usulan | Belum |

### B. Penomoran dan template

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-10 | Daftarkan nomor dokumen | Penyusun | Nomor dialokasikan di server tanpa bentrok; kode customer di nomor dicek. | M2.1, M2.2 | MVP | Sudah |
| UC-11 | Daftarkan nomor sebagai set | Penyusun | PFD, PFMEA, dan CP mendapat nomor urut yang sama. | M2.4 | Usulan | Sudah |
| UC-12 | Cari dan ekspor register nomor | Pengguna internal | Master document list dengan filter status. | M2.3 | Usulan | Sebagian: cari dan saring; ekspor belum |
| UC-13 | Ubah status nomor | Dokumen kontrol | Void atau Obsolete. Nomor tidak pernah dihapus atau dipakai ulang. | M2.3 | Usulan | Sudah |
| UC-14 | Impor nomor lama | Dokumen kontrol | Nomor lama masuk apa adanya, beserta laporan lubang, ganda, dan set tidak sejajar. | M2.5 | Usulan | Sudah |
| UC-15 | Ingatkan nomor Reserved yang terlalu lama | Penjadwal | Pengingat untuk nomor yang dipesan tetapi tidak dipakai. | M2.3 | Usulan | Belum; umur nomor Reserved sudah tampil |
| UC-20 | Kelola format template | Admin | Kolom ber-ID tetap, metode risiko, tabel skala, simbol, dan layout ekspor. | M3.1 | MVP | Sebagian: M3.1 sudah; layout ekspor menyusul di M6.1 |
| UC-21 | Kelola process template | PPAP Engineer | Proses, sub-proses, dan poin di empat tingkat: umum, customer, group model, model. | M3.2, M3.3, D-64 | MVP | Sebagian: cakupan umum sudah (M3.2); empat tingkat cakupan di M3.3 |
| UC-22 | Terbitkan versi template | PPAP Engineer | Versi draft menjadi terbit, dan tidak bisa diubah lagi. | M3.4, D-64 | Usulan | Belum |
| UC-23 | Propagasikan versi template | PPAP Engineer | Pratinjau dampak, pilih cakupan, tangani konflik lokal. Hasilnya sebuah perubahan, bukan edit langsung. | M5.8, D-64 | MVP | Belum |

### C. Dokumen dan perakitan

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-30 | Rakit dokumen baru dari template | Penyusun | Header diisi manual, isi dirakit dengan kotak centang; satu perakitan menghasilkan baris PFD, PFMEA, dan CP. | M4.1, M4.5 | MVP | Belum |
| UC-31 | Mulai dari komposisi model lain | Penyusun | Salinan menjadi dokumen mandiri. | M4.7 | Usulan | Belum |
| UC-32 | Edit baris di editor tabel | Penyusun | Edit sel, tambah dan hapus baris, dengan penguncian optimistik. | M4.2, M4.3 | MVP | Belum |
| UC-33 | Tempel blok sel dari Excel | Penyusun | Pratinjau dan pemetaan kolom; baris janggal ditandai. | M4.4 | MVP | Belum |
| UC-34 | Atur urutan proses dan nomor operasi | Penyusun | Drag and drop, nomor otomatis yang bisa diubah. | M4.6 | MVP | Belum |
| UC-35 | Tandai perbedaan lokal atau putus tautan | Penyusun | Putus tautan butuh alasan. | M4.8 | Usulan | Belum |
| UC-36 | Cek PFMEA dan CP berdampingan | PPAP Engineer | Tampilan per proses, beserta tanda dicek oleh siapa dan kapan. | M4.9 | MVP | Belum |
| UC-37 | Lihat diagram PFD | Pengguna internal | Diagram dibentuk dari daftar proses, simbol, dan disposisi NG. | M4.10 | Opsional | Belum |
| UC-38 | Tautkan dokumen yang saling bergantung | Penyusun | Misalnya ESD control plan, dengan titik masuk dan keluar. | M4.1 | Usulan | Belum |

### D. Perubahan dan revisi

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-40 | Ajukan permintaan perubahan | Penyusun | Sumber, jenis, alasan, rujukan, lampiran, dan dokumen terdampak. | M5.1, M5.2 | MVP | Belum |
| UC-41 | Verifikasi permintaan perubahan | Verifikator | Terima atau tolak dengan catatan wajib. Pemohon tidak boleh memverifikasi; verifikator tercatat. | M5.3 | MVP | Belum |
| UC-42 | Isi daftar periksa dampak | Penyusun | Dokumen terkait yang terdampak, termasuk WI, beserta keputusan dan alasannya. | M5.7 | Usulan | Belum |
| UC-43 | Kerjakan draft revisi | Penyusun | Setiap dokumen terdampak mendapat draft; revisi yang berlaku tidak berubah. | M5.4 | MVP | Belum |
| UC-44 | Jalankan yokoten | Penyusun | Pilih target per customer, group model, model, proses, atau teks; papan status per model. | M5.9 | MVP | Belum |
| UC-45 | Putuskan yokoten per dokumen target | Penyusun | Terapkan, sesuaikan, atau tidak berlaku dengan alasan. | M5.9 | MVP | Belum |
| UC-46 | Terapkan perubahan surut SQAM | Penyusun | Semua dokumen aktif mendapat draft baru; revisi terbit tidak ditulis ulang. | M5.10 | MVP | Belum |
| UC-47 | Lihat riwayat dan diff revisi | Pengguna internal | Per dokumen dan per proses, terhubung ke perubahan dan lampirannya. | M5.6 | MVP | Belum |

### E. Cetak, tanda tangan, dan rilis

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-50 | Cetak untuk tanda tangan | Penyusun | Revisi dikunci, tanggal rilis diisi otomatis, kotak tanda tangan dibentuk dari rute. | M6.1, M6.2, M6.3 | MVP | Belum |
| UC-51 | Catat dokumen dikembalikan | Penyusun | Approver minta perubahan di kertas; revisi kembali ke draft. | M5.5a (usulan); bila M5.5a ditolak, M5.4 | MVP | Belum |
| UC-52 | Catat rilis | Dokumen kontrol | Unggah scan, konfirmasi slot sesuai urutan rute. Pencatat bukan penyusun. | M6.7 | MVP | Belum |
| UC-53 | Cetak salinan terkendali | Dokumen kontrol | Watermark CONTROLLED COPY, beserta catatan cetak. | M6.4, M6.6 | Usulan | Belum |
| UC-54 | Ekspor Excel dan PDF | Pengguna internal | Excel A3 dan A4, PDF yang sama rapinya. | M6.2, M6.3 | MVP | Belum |
| UC-55 | Lihat salinan yang harus ditarik | Dokumen kontrol | Daftar dari catatan cetak saat revisi baru rilis. | M6.6 | Usulan | Belum |
| UC-56 | Buka halaman verifikasi | Viewer | Status berlaku, digantikan, atau dibatalkan. Lewat QR bila QR menyala. | M6.5 | Saklar | Belum |
| UC-57 | Baca dokumen rilis | Viewer | Hanya baca. | M1.2 (hak L-1), M5.5 | MVP | Belum |

### F. Migrasi dan ukuran

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-60 | Masukkan dokumen lama sebagai baseline | Petugas migrasi | Nomor asli, revisi saat ini, dan riwayat revisi sebagai teks. | M7.1, M7.2 | MVP | Belum |
| UC-61 | Bandingkan dengan Excel lama | Petugas migrasi | Excel lama hanya untuk dibandingkan dengan hasil perakitan. | M7.3 | Usulan | Belum |
| UC-62 | Tautkan baris hasil tempel ke template | Petugas migrasi | Sistem mengusulkan pasangan, manusia mengonfirmasi. | M7.4 | MVP | Belum |
| UC-63 | Verifikasi terhadap hardcopy bertanda tangan | Petugas migrasi | Status Released (migrated). | M7.5 | Usulan | Belum |
| UC-64 | Lihat dasbor ukuran | Pengguna internal | Persentase baris yang mengikuti template, status dokumen dan yokoten, dan waktu revisi. | M7.7 | MVP | Belum |

### G. Masa depan (latihan tahap 9 dan 10)

| ID | Use case | Aktor utama | Ringkasan | Milestone | Status rencana | Status kode |
|---|---|---|---|---|---|---|
| UC-80 | Kelola service account dan token | Admin | Scope, customer yang diizinkan, dan masa berlaku. | M9.1 | Masa depan | Belum |
| UC-81 | Baca data rilis lewat API | Aplikasi internal lain | Data rilis, paginasi, filter updated_since. | M9.3 | Masa depan | Belum |
| UC-82 | Kirim usulan perubahan lewat API | Aplikasi internal lain | Menjadi permintaan perubahan yang menunggu verifikasi manusia. | M9.4 | Masa depan | Belum |
| UC-83 | Ekspor data secara asinkron | Aplikasi internal lain | Job ekspor, polling, unduh. | M9.5 | Masa depan | Belum |
| UC-84 | Lihat dan unduh dokumen di portal | Customer user | Hanya dokumen rilis milik customer itu. | M10.2, M10.6 | Masa depan | Belum |
| UC-85 | Beri izin tampil | Penyusun | Konfirmasi per peran, terikat content hash. | M10.4 | Masa depan | Belum |
| UC-86 | Kelola kebijakan publikasi customer | Admin | Sumber file dan tahap paling awal. | M10.3 | Masa depan | Belum |

## Lampiran C. Glosarium dan istilah layar (U1)

Disalin dari `docs/uml/U1-glosarium.md`. Kolom Istilah layar adalah teks yang dipakai di layar berbahasa Inggris. (usulan) berarti aturannya belum diputuskan; (masa depan) hanya untuk latihan Tahap 9 dan 10.

### 1. Dokumen dan standar

| Istilah | Istilah layar | Arti |
|---|---|---|
| PFD | PFD | Process Flow Diagram: diagram alur proses produksi satu model. Job description di bagan organisasi menyebutnya PFC; sistem dan prosedur memakai PFD. |
| PFMEA | PFMEA | Process Failure Mode and Effects Analysis: analisis mode kegagalan proses beserta risikonya. |
| CP | CP | Control Plan: rencana kontrol per proses. Cover perusahaan menyebutnya Process Control Plan (PCP). |
| Set (satu set) | Set | PFD, PFMEA, dan CP satu model yang berbagi satu nomor urut. |
| AIAG 4th | AIAG 4th | Buku panduan FMEA AIAG edisi ke-4, berbasis RPN. |
| AIAG-VDA | AIAG-VDA | Buku panduan FMEA gabungan AIAG dan VDA, berbasis Action Priority. Sistem saat ini hanya kompatibel dengan form-nya. Struktur penuhnya ditunda. |
| RPN / AP | RPN / AP | Risk Priority Number / Action Priority. |
| SQAM | SQAM | Supplier Quality Assurance Manual milik customer. Bisa mengatur format dan aturan dokumen, dan perubahannya sering berlaku surut. |
| PPAP | PPAP | Production Part Approval Process. |
| 4M change | 4M change | Perubahan Man, Machine, Material, atau Method. Persetujuan customer untuk perubahan ini diurus di luar sistem. |
| WI | WI | Work Instruction: dokumen di luar sistem yang bisa terdampak perubahan. Istilah di sistem harus sama dengan WI. |
| Standar inspeksi | Inspection standard | Dokumen di luar sistem yang bisa terdampak perubahan. |

### 2. Isi dokumen dan template

| Istilah | Istilah layar | Arti |
|---|---|---|
| Proses | Process | Langkah produksi, misalnya IQC. Punya ID tetap; namanya hanya tampilan dan boleh punya alias. |
| Sub-proses | Sub-process | Varian dari satu proses, misalnya IQC General, IQC MSD, IQC PCB, IQC Wireharness, IQC Connector. ID-nya tetap dan milik satu proses master; namanya disimpan per versi process template (ASUMSI E-33). |
| Poin | Point | Satu butir setingkat baris di dalam sub-proses, untuk PFD, PFMEA, atau CP, dengan ID tetap. Untuk sementara berisi satu baris teks (ASUMSI E-73). |
| IQC | IQC | Incoming Quality Control. |
| Nomor operasi | Operation number | Nomor urut proses di dokumen. Dibuat otomatis dan bisa diubah. |
| Baris dokumen | Row | Satu baris lembar kerja PFD, PFMEA, atau CP. ID-nya tetap lintas revisi. |
| Format template | Format template | Template metodologi: kolom, metode risiko, tabel skala, field header, dan layout ekspor. Berversi. |
| Process template | Process template | Pustaka proses, sub-proses, dan poin. Satu template per proses (ASUMSI E-72). Berversi. |
| Sub-proses dan poin tersembunyi | Hidden | Sub-proses atau poin yang tidak dipakai lagi. Tidak dihapus: ID-nya tetap, jadi rujukan dari dokumen tetap utuh. |
| Cakupan template | Template scope | Tingkat berlakunya process template: umum, customer, group model, dan model. Yang paling spesifik menang: model, lalu group model, lalu customer, lalu umum. |
| Template General | Template General | Process template umum yang tidak terikat customer atau model. |
| Group model | Model group | Kumpulan model dalam satu customer yang berbagi isi template. Satu model paling banyak masuk satu group. Bisa dipilih sebagai target yokoten. |
| Model | Model | Produk customer, misalnya nama di kolom Model pada PFMEA. Milik satu customer dan paling banyak satu group model. |
| Part number | Part number | Nomor part customer, milik satu model. |
| Kolom | Column | Satu kolom baris dokumen di format template. ID-nya tetap; labelnya hanya tampilan, jadi bisa diganti tanpa merusak data. |
| Field header | Header field | Isian di bagian header dokumen yang diketik tangan, misalnya Core Team atau Key Date. ID-nya tetap, seperti kolom. |
| Kolom tersembunyi | Hidden | Kolom atau field header yang tidak dipakai lagi. Tidak dihapus: datanya tetap tersimpan dan tetap sah. |
| Metode risiko | Risk method | RPN (AIAG 4th), AP (AIAG-VDA), atau tanpa metode risiko (PFD dan CP). Tetap sejak versi format template dibuat (ASUMSI E-70). |
| Tabel skala | Rating tables | Kriteria nilai 1 sampai 10 untuk severity, occurrence, dan detection di satu versi format template. Teksnya diisi dari form yang dipakai. |
| Nilai sekarang dan hasil tindakan | Current, action results | Dua set rating dan RPN di PFMEA AIAG 4th: nilai sekarang, dan nilai sesudah tindakan (kolom Action Results). |
| Aturan sorot | Highlight rule | Tanda untuk RPN di atas ambang atau severity paling sedikit ambang, misalnya RPN di atas 100 atau severity 8 (ASUMSI E-71). Hanya menandai, tidak memblokir. (usulan) |
| Perakitan | Composition | Menyusun dokumen dari process template dengan kotak centang dan drag and drop. |
| Poin dikecualikan | Excluded point | Poin template yang sengaja tidak dipakai satu model. Tetap tercatat, tidak dihapus diam-diam. (usulan) |
| Perbedaan lokal | Local override | Baris yang diubah hanya untuk satu dokumen dan ditandai berbeda dari template. (usulan) |
| Putus tautan | Detach | Dokumen atau baris yang dilepas dari tautan template, dengan alasan. (usulan) |
| Mode tautan template | Follow, Pinned, Detached | Hubungan dokumen dengan template: ikuti, kunci versi, atau lepas. (usulan) |
| Tautan PFMEA ke CP | Linked counterpart | Hubungan opsional antara baris PFMEA dan baris CP, dibuat manual atau dari template. |
| Simbol alur PFD | Flow symbol | Bentuk langkah di PFD: process, alternate process, decision, connector, merge, dan document. (usulan) |
| Disposisi NG | NG disposition | Tujuan produk NG dari langkah decision: scrap, NCP, atau rework. (usulan) |
| NCP | NCP | Nonconforming product, salah satu disposisi NG. |
| Simbol karakteristik | Characteristic symbol | Simbol di PFMEA dan CP. Saat ini: Critical Process, Revision, Important Character (IC), dan Regulation. Disimpan di format template per customer. (usulan) |
| Core Team | Core team | Field header PFMEA dan CP yang berisi anggota tim beserta departemennya. |
| Dokumen terkait | Relevant document | Nomor dan judul WI, SOP, QEP, dan dokumen lain yang dirujuk di kolom Relevant Document CP. Diambil dari master supaya sama dengan sumbernya. (usulan) |
| Dokumen saling bergantung | Interdependent document | Dokumen terpisah yang dirujuk sebagai pengganti langkah yang diulang, misalnya ESD control plan. (usulan) |

### 3. Perubahan dan revisi

| Istilah | Istilah layar | Arti |
|---|---|---|
| Perubahan | Change | Satu entitas untuk semua sumber perubahan. Berisi sumber, jenis, alasan, rujukan, lampiran, dokumen terdampak, dan daftar periksa dampak. |
| Permintaan perubahan | Revision request | Perubahan yang diajukan manual oleh Penyusun (PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer) dengan dokumen pendukung. |
| Sumber perubahan | Change source | Asal perubahan: permintaan manual, yokoten, propagasi template, atau perubahan surut SQAM. |
| Pemicu perubahan | Change source (E-36) | Pemicu yang dipakai sekarang: PP (pre-production), customer claim, ECR, dan temuan audit. (usulan) |
| ECR | ECR | Engineering Change Request: permintaan perubahan yang diajukan ke customer. Salah satu pemicu perubahan. |
| Yokoten | Yokoten | Penerapan satu perubahan atau pelajaran dari satu model, proses, atau dokumen ke yang lain. Target dipilih per customer, group model, model, proses, atau pencarian teks. |
| Propagasi template | Template propagation | Versi template baru yang diteruskan ke dokumen pengikutnya sebagai draft revisi. Tidak pernah mengubah dokumen secara langsung. |
| Perubahan surut SQAM | Retroactive SQAM change | Versi SQAM baru yang berlaku untuk dokumen aktif. Dibuat sebagai revisi baru; revisi yang sudah terbit tidak pernah ditulis ulang. |
| Jenis perubahan | Change type | Isi, tata letak saja, atau administratif. Menentukan jalur approval. |
| Daftar periksa dampak | Impact checklist | Catatan per perubahan tentang dokumen terkait yang terdampak (PFD, PFMEA, CP, WI, standar inspeksi), keputusannya, dan alasannya. (usulan) |
| Verifikasi | Verification | Penerimaan atau penolakan permintaan perubahan dengan catatan wajib, oleh Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller. Siapa yang memverifikasi tercatat. |
| Revisi | Revision | Versi dokumen bernomor bulat, ditampilkan -00, -01, dan seterusnya. Dokumen hasil migrasi mulai dari revisi saat ini, misalnya -04. |
| Draft | Draft | Revisi yang masih bisa diedit dan belum berlaku. |
| Menunggu approval | Awaiting approval | Revisi yang sudah dicetak untuk tanda tangan dan dikunci. Edit sesudahnya membatalkan cetakan itu. (usulan) |
| Dikembalikan | Returned | Approver minta perubahan di kertas. Penyusun mencatatnya, dan revisi kembali ke draft. |
| Released | Released | Revisi yang berlaku. Isinya dibekukan sebagai snapshot dan tidak bisa diubah. |
| Obsolete | Obsolete | Revisi yang sudah digantikan. Tidak pernah berubah. |
| Snapshot | Snapshot | Isi revisi yang dibekukan saat rilis, beserta hash SHA-256. |
| Hash | Hash | Sidik jari isi. Ada tiga: snapshot_hash, customer_content_hash, dan file_hash. (usulan) |
| Diff | Diff | Perbandingan baris dan sel antar revisi. |
| Riwayat revisi | Revision history | Tabel revisi otomatis di cetakan. Riwayat sebelum baseline migrasi disimpan sebagai teks dari tabel Excel lama. (usulan) |

### 4. Penomoran

| Istilah | Istilah layar | Arti |
|---|---|---|
| Nomor dokumen | Document number | Nomor terdaftar untuk PFD, PFMEA, atau CP per customer dan jenis dokumen, misalnya SC-KT-001. Disimpan sebagai teks dan tidak pernah dipakai ulang. |
| Awalan | Prefix | SH untuk PFD, SF untuk PFMEA, dan SC untuk CP (dokumen saat ini), sebagai master data yang bisa diedit. |
| Kode customer | Customer code | Kode singkat customer di nomor, misalnya KT. Harus kode customer itu, sekarang atau dulu: nomor lama tetap memakai kode lamanya. (usulan) |
| Register nomor | Number register | Daftar semua nomor beserta statusnya. Bisa diekspor sebagai master document list. (usulan) |
| Status nomor | Number status | Reserved, In use, Released, Obsolete, Void, ditambah Skipped untuk nomor yang dilewati saat daftar sebagai set (ASUMSI E-64). Nomor tidak pernah dihapus; yang berubah hanya statusnya. (usulan) |
| Daftarkan sebagai set | Register as a set | Mengambil nomor bebas berikutnya di ketiga jenis dokumen supaya nomornya sejajar. Nomor yang dilewati penghitung yang tertinggal dicatat Skipped. (usulan) |
| Nomor dilewati | Skipped | Nomor yang dilewati saat daftar sebagai set. Tetap di register dan tidak pernah diberikan lagi, supaya register tanpa lubang. (ASUMSI E-64) |
| Pola nomor | Number pattern | Susunan nomor dari {prefix}, {customer}, dan {seq} dengan pemisah, misalnya {prefix}-{customer}-{seq} untuk SC-KT-001. Satu pola untuk semua jenis dokumen dan customer (ASUMSI E-61), dan hanya berlaku untuk nomor baru. |
| Nomor berjalan | Running number | Bagian urut di nomor: 001 sampai 999, lalu A001 sampai Z999. |
| Revisi awal | Initial revision | Revisi pertama dokumen yang memakai nomor itu, misalnya 4 (-04) untuk dokumen lama yang masuk di revisi saat ini. |
| Digit revisi | Revision digits | Jumlah digit label revisi, saat ini 2 (-04). Diatur di master data. |
| Kunci idempoten | (tidak tampil di layar) | Kunci acak yang dikirim bersama permintaan pendaftaran. Permintaan yang diulang dengan kunci yang sama mendapat nomor yang sama, bukan nomor baru. Berlaku per akun (ASUMSI E-63). |
| Impor nomor lama | Import legacy numbers | Memasukkan nomor dari register Excel lama dengan teks aslinya. Penghitung naik ke nomor tertinggi; lubang, nomor ganda, dan set tidak sejajar hanya dilaporkan. (usulan) |
| Pratinjau impor | Preview | Laporan impor tanpa menyimpan apa pun. |
| Lubang | Gaps | Nomor berjalan yang tidak ada di register, di antara 001 dan nomor tertinggi. Dilaporkan, tidak pernah diisi. |
| Set tidak sejajar | Sets not aligned | Nomor berjalan yang hanya dimiliki sebagian dari PFD, PFMEA, dan CP. Nomor Skipped dianggap tidak ada (ASUMSI E-65). |
| Penghitung | Counter | Nomor terakhir per kode customer dan jenis dokumen, dinaikkan di dalam transaksi. |
| Kode pabrik | Factory code | Huruf pertama awalan nomor, saat ini S (WI penomoran). |
| Jenis dokumen | Document type | Huruf kedua awalan nomor: H untuk PFD, F untuk PFMEA, C untuk CP, ditambah jenis lain dari WI seperti W. Hanya PFD, PFMEA, dan CP yang dinomori sistem. |
| Kode khusus | Special code | Kode di posisi kode customer untuk dokumen yang bukan milik satu customer: ZZ untuk General/Common dan HSE. Tidak pernah sama dengan kode customer. |

### 5. Tanda tangan, approval, dan rilis

| Istilah | Istilah layar | Arti |
|---|---|---|
| Prepared | Prepared | Tahap tanda tangan penyusun: PPAP Engineer, PPAP Staff, Project Leader, atau Project Engineer. |
| Checked | Checked | Tahap tanda tangan pengecek. Cukup satu orang: Project Engineer atau Project Leader dari Project Group customer itu. |
| Approved | Approved | Tahap tanda tangan approver, berurutan: NPI Manager, lalu QA QC Manager, lalu slot GM bila ada di rute. |
| Slot GM | GM slot | Slot approver terakhir. Saat ini diisi Advisor; nanti bisa diisi Factory Manager. Cover mencetak "General Mgr." dengan nama pemegangnya, mengikuti cover dokumen saat ini (D-76). |
| Judul cetak slot | Print title | Tulisan di bawah nama pada setiap slot, terpisah dari nama jabatan. Mengikuti cover dokumen saat ini: Process. Eng, Engineer, NPI Manager, QA / QC Mgr., General Mgr. |
| Rute approval | Approval route | Daftar slot tanda tangan berurutan untuk satu dokumen. Ada dua rute: sampai QA QC Manager dan sampai slot GM. Default-nya per customer, dan bisa diganti per dokumen dengan alasan. |
| Slot | Slot | Satu kotak tanda tangan di rute, dengan tahap dan jabatan yang boleh mengisinya. |
| Tanggal tanda tangan | Sign date | Tanggal per slot. Disiapkan dengan saklar untuk mematikan dan menyembunyikannya, dan saat ini mati. |
| Tanggal rilis | Release date | Tanggal dokumen dicetak untuk tanda tangan, diisi otomatis. Secara resmi tanggal rilis adalah tanggung jawab DCC. Saat ini satu-satunya tanggal yang dicatat. |
| Tanggal berlaku | Effective date | Tanggal revisi mulai berlaku. Saat ini sama dengan tanggal rilis, dan aturannya harus bisa diubah nanti. |
| Tanda tangan basah | Wet signature | Tanda tangan tangan di kertas. Approval saat ini manual. |
| Pencatatan rilis | Release recording | Dokumen kontrol mencatat scan bertanda tangan, lalu status revisi menjadi Released. Pencatat bukan penyusun. Berbeda dari tanggal rilis. (usulan) |
| Dokumen kontrol | Document control | Fungsi yang mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer, yang saling mencatat rilis dokumen yang disusun yang lain. Nanti bisa ditambah DCC. |
| Salinan terkendali | Controlled copy | Salinan cetak yang didistribusikan dan dilacak, dengan watermark CONTROLLED COPY. (usulan) |
| Salinan tidak terkendali | Uncontrolled copy | Salinan untuk referensi atau draft, dengan watermark UNCONTROLLED COPY. (usulan) |
| Catatan cetak | Print log | Siapa mencetak, kapan, dan nomor salinannya. (usulan) |
| QR | QR | Kode di cetakan yang membuka halaman verifikasi. Opsional dengan saklar, dan saat ini mati karena belum dipakai. |
| Halaman verifikasi | Verification page | Halaman yang menampilkan status dokumen: berlaku, digantikan, atau dibatalkan. Bisa dibuka dari QR bila QR menyala, atau dari dalam sistem. (usulan) |
| Tanda revisi | Revision mark | Segitiga berisi nomor revisi di samping baris yang berubah pada revisi itu, dibuat otomatis dari diff. (usulan) |
| Layout | Layout | Template Excel beserta pemetaan sel, terpisah dari data baris. |

### 6. Jabatan

| Istilah | Istilah layar | Arti |
|---|---|---|
| Pemegang jabatan | Holder | Orang yang tercatat memegang satu jabatan. Project Leader dan Project Engineer satu per Project Group; jabatan approver dan APQP/PPAP Controller satu orang; jabatan lain boleh beberapa orang. |
| Project Group | Project group | Kelompok customer yang dipegang satu Project Leader dan satu Project Engineer. Setiap customer dialokasikan ke satu Project Group, saat ini GR1 sampai GR4 (Tabel 1 di bagan organisasi). |
| PPAP Engineer | PPAP Engineer | Level Leader. Menyusun dan memperbarui PFD, PFMEA, dan CP, mengelola master proses dan process template, dan untuk saat ini memberi tanda dicek PFMEA–CP. Juga memegang peran dokumen kontrol dan petugas migrasi. |
| PPAP Staff | PPAP Staff | Level SOP/Operator. Boleh menyusun dokumen, dan saat ini memegang peran dokumen kontrol dan petugas migrasi. Juga memegang akun admin terpisah, sehingga punya dua akun. |
| DCC | DCC | Dept Document Control. Secara resmi bertanggung jawab atas tanggal rilis, dan nanti bisa memegang peran dokumen kontrol. |
| Project Engineer | Project Engineer | Level Leader, satu per Project Group. Boleh menyusun dokumen (D-67), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). |
| Project Leader | Project Leader | Level Supervisor, satu per Project Group. Boleh menyusun dokumen (D-63), mengisi slot Checked, dan memverifikasi permintaan perubahan, semuanya hanya untuk customer di group-nya (D-71). |
| APQP/PPAP Controller | APQP/PPAP Controller | Level Supervisor, satu orang, saat ini dirangkap Project Leader GR3. Boleh memverifikasi permintaan perubahan semua customer. |
| NPI Manager | NPI Manager | Approver pertama. Menurut bagan, menyetujui dokumen resmi departemen NPI. Bila struktur berganti nama, jabatan ini bisa bernama PE Manager. |
| QA QC Manager | QA QC Manager | Approver kedua. Rute bisa berakhir di sini. Jabatan ini di luar bagan NPI. |
| Advisor | Advisor | Level General Manager. Saat ini mengisi slot GM. |
| Factory Manager | Factory Manager | Level General Manager. Nanti bisa mengisi slot GM. |
| QA, QC, internal audit | QA, QC, internal audit | Pengguna halaman verifikasi QR dan dokumen rilis, dengan peran viewer. |

### 7. Peran sistem dan keamanan

| Istilah | Istilah layar | Arti |
|---|---|---|
| Level akses | Access level | Buat / Edit, Verifikasi, Lihat, dan Admin. |
| Hak akses | Permission | Satu aksi di sistem, misalnya mencatat rilis. Peran adalah kumpulan hak yang isinya diatur admin. |
| Penyusun | Author | Peran PPAP Engineer, PPAP Staff, Project Leader, dan Project Engineer: menyusun dokumen, mendaftarkan nomor, mengajukan perubahan, dan mencetak untuk tanda tangan. Project Leader dan Project Engineer hanya untuk customer di Project Group sendiri. |
| Pengelola template | Template maintainer | Peran PPAP Engineer: mengelola master proses dan process template. |
| Pemeriksa PFMEA–CP | PFMEA–CP checker | Peran PPAP Engineer untuk saat ini: memberi tanda dicek di perbandingan PFMEA dan CP. |
| Verifikator | Verifier | Peran Project Engineer, Project Leader, dan APQP/PPAP Controller: memverifikasi permintaan perubahan. |
| Dokumen kontrol (peran) | Document control | Peran yang mencatat rilis serta mengelola nomor dan salinan. Saat ini dipegang PPAP Staff dan PPAP Engineer. |
| Petugas migrasi | Migration | Peran yang memasukkan dokumen lama. Saat ini dipegang PPAP Staff dan PPAP Engineer. |
| Viewer | Viewer | Hak dasar Lihat (L-1 sampai L-3) yang dimiliki semua akun yang login. Jabatan sampai level GM yang tidak menyusun, termasuk approver, DCC, QA, QC, internal audit, dan tim WI, hanya memegang ini. |
| Admin | Admin | Peran akun admin: mengelola pengguna, hak akses, master data, rute approval, dan pengaturan, serta melihat audit log (A-1 sampai A-4), ditambah hak dasar Lihat. Tidak pernah memegang peran kerja. |
| Approver (peran) | Approver | Belum dibuat. Disiapkan untuk approval di aplikasi nanti (R-APR-04); selama approval di kertas, approver memakai hak dasar Lihat. |
| Orang | Person | Seseorang dengan nama dan nomor karyawan opsional. Jabatan dipegang orang, dan aturan pemisahan tugas membandingkan orang, bukan akun. |
| Akun kerja / akun admin | Work account / Admin account | Login milik satu orang. Satu orang paling banyak punya satu akun kerja dan satu akun admin. Akun tidak pernah dihapus, hanya dinonaktifkan. |
| Audit log | Audit log | Catatan setiap perubahan data, termasuk login, yang tidak bisa diubah atau dihapus dan dilindungi satu hash chain. Hanya dilihat lewat akun admin. |
| Cakupan customer | Customer scope | Batas data per customer. Cakupan Project Group untuk Project Leader dan Project Engineer diperiksa fungsi izin sejak M1.2; row-level security di M8.1. |

### 8. Migrasi

| Istilah | Istilah layar | Arti |
|---|---|---|
| Baseline migrasi | Migration baseline | Dokumen lama yang dimasukkan di revisi saat ini (misalnya -04), sama dengan hardcopy bertanda tangan. (usulan) |
| Released (migrated) | Released (migrated) | Status dokumen migrasi setelah diverifikasi terhadap hardcopy bertanda tangan. (usulan) |
| Mode migrasi | Migration mode | Saklar yang membuka fitur migrasi untuk peran khusus. Bisa dimatikan admin. (usulan) |
| Penautan baris | Linking pass | Sistem mengusulkan pasangan baris hasil tempel dengan poin template lewat kemiripan teks, lalu manusia mengonfirmasi. |

### 9. Masa depan (latihan tahap 9 dan 10)

| Istilah | Istilah layar | Arti |
|---|---|---|
| API integrasi | Integration API | Antarmuka untuk aplikasi internal lain: membaca data dan mengirim usulan perubahan. (masa depan) |
| Service account | Service account | Akun aplikasi lain dengan token, scope, dan masa berlaku. (masa depan) |
| Usulan perubahan | Change proposal | Perubahan yang dikirim aplikasi lain dan menunggu verifikasi manusia. (masa depan) |
| Portal customer | Customer portal | Halaman baca saja bagi customer untuk melihat dan mengunduh dokumen rilis miliknya. (masa depan) |
| Customer user | Customer user | Akun customer, terpisah dari akun internal. (masa depan) |
| Izin tampil | Display permission | Konfirmasi internal sebelum revisi tampil di portal. (masa depan) |
| Sumber file | File source | Kebijakan per customer: scan_then_pdf (default), scan_only, atau pdf_only. (masa depan) |
| Field internal | Internal field | Field yang tidak dikirim ke customer. Fiturnya ditahan. |

## Lampiran D. Operasi API yang sudah ada

Disalin dari kontrak `api/openapi.yaml` (82 operasi). Semua butuh login kecuali `/health` dan login. Kode hak di ringkasan menunjukkan hak yang diperiksa server.

| Method | Path | Ringkasan |
|---|---|---|
| GET | `/health` | Server and database status |
| GET | `/api/v1/session` | The session of the caller |
| POST | `/api/v1/session` | Log in with a login name and password |
| DELETE | `/api/v1/session` | Log out and end the session |
| PUT | `/api/v1/account/password` | Change the password of the caller |
| GET | `/api/v1/roles` | The role catalog with the rights of each role (A-1) |
| GET | `/api/v1/accounts` | All accounts with their person and roles (A-1) |
| POST | `/api/v1/accounts` | Create an account with a temporary password (A-1) |
| PUT | `/api/v1/accounts/{accountId}/roles` | Replace the roles of a work account (A-1) |
| POST | `/api/v1/accounts/{accountId}/password-reset` | Set a new temporary password for another account (A-1) |
| POST | `/api/v1/accounts/{accountId}/disable` | Disable another account and end its sessions (A-1) |
| POST | `/api/v1/accounts/{accountId}/enable` | Enable an account again and unlock it (A-1) |
| GET | `/api/v1/customers` | List all customers, ordered by code |
| POST | `/api/v1/customers` | Add a customer |
| PUT | `/api/v1/customers/{customerId}` | Change the name, project group, and status of a customer (A-2) |
| POST | `/api/v1/customers/{customerId}/code` | Give a customer a new code; the old code stays in its history (A-2, E-11) |
| GET | `/api/v1/project-groups` | All project groups, ordered by code |
| POST | `/api/v1/project-groups` | Add a project group (A-2) |
| PUT | `/api/v1/project-groups/{projectGroupId}` | Change the code and name of a project group (A-2) |
| GET | `/api/v1/model-groups` | The model groups of one customer, ordered by name |
| POST | `/api/v1/model-groups` | Add a model group to a customer (A-2) |
| PUT | `/api/v1/model-groups/{modelGroupId}` | Rename a model group (A-2) |
| GET | `/api/v1/models` | The models of one customer, ordered by name |
| POST | `/api/v1/models` | Add a model to a customer, in at most one model group (A-2, D-42) |
| PUT | `/api/v1/models/{modelId}` | Rename a model or move it to another model group of the same customer (A-2) |
| GET | `/api/v1/part-numbers` | The part numbers of one customer, ordered by model and number |
| POST | `/api/v1/part-numbers` | Add a part number to a model (A-2) |
| PUT | `/api/v1/part-numbers/{partNumberId}` | Change the number and name of a part number (A-2) |
| GET | `/api/v1/processes` | All processes with their aliases, ordered by name (UC-04) |
| POST | `/api/v1/processes` | Add a process with a fixed ID, a display name, and aliases (E-6, D-06, D-64) |
| GET | `/api/v1/processes/{processId}` | One process by its fixed ID, with the name in use today (D-06) |
| PUT | `/api/v1/processes/{processId}` | Rename a process or replace its aliases. The ID stays, so references keep working (E-6, D-06) |
| GET | `/api/v1/number-settings` | Number settings - factory code, number pattern, running number format, and revision digits (D-89) |
| PUT | `/api/v1/number-settings` | Change the number settings. Issued numbers keep their text (A-2, R-NUM-02) |
| GET | `/api/v1/document-numbers` | The number register, newest first, with filters (L-2, UC-12, R-NUM-04) |
| POST | `/api/v1/document-numbers` | Register one PFD, PFMEA, or CP number for a customer (E-1, UC-10, R-NUM-03) |
| POST | `/api/v1/document-numbers/sets` | Register PFD, PFMEA, and CP of one model with the same running number (E-1, UC-11, R-NUM-05, M2.4) |
| POST | `/api/v1/document-numbers/{documentNumberId}/void` | Void a Reserved or In use number, with a reason (E-8, UC-13) |
| POST | `/api/v1/document-numbers/{documentNumberId}/obsolete` | Make a Released number obsolete, for example when it is withdrawn (E-8, UC-13) |
| GET | `/api/v1/document-numbers/next` | The number the next registration would get, without registering it (L-2) |
| POST | `/api/v1/document-number-imports` | Import legacy numbers of one customer, with a report (E-8, UC-14, R-NUM-06) |
| GET | `/api/v1/document-types` | Document types with their number prefix (R-NUM-01, E-12) |
| POST | `/api/v1/document-types` | Add another document type from the WI, for example Work Instruction (A-2) |
| PUT | `/api/v1/document-types/{documentTypeId}` | Change the name and letter of a document type (A-2) |
| GET | `/api/v1/special-codes` | Special codes used in place of a customer code, ordered by code (E-12) |
| POST | `/api/v1/special-codes` | Add a special code. It cannot equal any customer code, now or before (A-2) |
| PUT | `/api/v1/special-codes/{specialCodeId}` | Rename a special code. The code itself never changes (A-2) |
| GET | `/api/v1/change-types` | The change types content, layout only, and administrative (R-TPL-10) |
| PUT | `/api/v1/change-types/{changeTypeId}` | Rename a change type (A-2) |
| GET | `/api/v1/change-sources` | Change sources, active and inactive, ordered by name (R-CHG-06) |
| POST | `/api/v1/change-sources` | Add a change source (A-2) |
| PUT | `/api/v1/change-sources/{changeSourceId}` | Rename a change source, or switch it off or on (A-2) |
| GET | `/api/v1/four-m-categories` | The 4M categories Man, Machine, Material, and Method |
| PUT | `/api/v1/four-m-categories/{fourMCategoryId}` | Rename a 4M category (A-2) |
| GET | `/api/v1/positions` | Positions of the NPI organization chart with their holders (D-54, D-58) |
| PUT | `/api/v1/positions/{positionId}` | Rename a position. Its key and holders stay (A-1, D-58) |
| POST | `/api/v1/positions/{positionId}/holders` | Record a person as holder of a position (A-1). One Project Leader and one Project Engineer per project group (D-54) |
| DELETE | `/api/v1/positions/{positionId}/holders/{holderId}` | Remove a holder from a position (A-1) |
| GET | `/api/v1/approval-routes` | Approval routes with their slots in signing order, ordered by name (R-APR-08) |
| POST | `/api/v1/approval-routes` | Add an approval route (A-2, UC-06) |
| PUT | `/api/v1/approval-routes/{approvalRouteId}` | Rename an approval route and replace its slots (A-2, R-APR-15) |
| PUT | `/api/v1/customers/{customerId}/default-route` | Set the default approval route of a customer (A-2, D-47) |
| PUT | `/api/v1/change-types/{changeTypeId}/approval-route` | Set the approval route of a change type, or use the customer default (A-2, R-TPL-10) |
| GET | `/api/v1/format-templates` | Format templates with a summary of their versions (L-2, UC-20, R-TPL-01) |
| POST | `/api/v1/format-templates` | Add a format template with a draft version 1, empty or copied from another version (A-2, UC-20) |
| PUT | `/api/v1/format-templates/{formatTemplateId}` | Rename a format template (A-2) |
| GET | `/api/v1/format-template-versions/{formatVersionId}` | One format template version with its fields, rating criteria, and symbols (L-2) |
| PUT | `/api/v1/format-template-versions/{formatVersionId}` | Change the highlight rule of a draft version (A-2, R-TPL-11) |
| POST | `/api/v1/format-template-versions/{formatVersionId}/fields` | Add a header field or a column at the end of its area in a draft version (A-2, R-TPL-02) |
| PUT | `/api/v1/format-template-versions/{formatVersionId}/fields/{formatFieldId}` | Change the label, required flag, hidden flag, position, or options of a field in a draft version (A-2, R-TPL-02) |
| PUT | `/api/v1/format-template-versions/{formatVersionId}/rating-criteria` | Change the criteria text of rating values in a draft version (A-2, R-TPL-01) |
| PUT | `/api/v1/format-template-versions/{formatVersionId}/symbols` | Rename symbols and add new ones in a draft version (A-2, R-TPL-11) |
| GET | `/api/v1/process-templates` | Process templates with a summary of their versions (L-2, UC-21, R-TPL-01) |
| POST | `/api/v1/process-templates` | Add the process template of a process, with an empty draft version 1 (E-6, UC-21, D-64) |
| GET | `/api/v1/process-template-versions/{processVersionId}` | One process template version with its sub-processes and points (L-2) |
| POST | `/api/v1/process-template-versions/{processVersionId}/subprocesses` | Add a sub-process at the end of a draft version (E-6) |
| PUT | `/api/v1/process-template-versions/{processVersionId}/subprocesses/{subprocessId}` | Rename, move, hide, or show a sub-process in a draft version (E-6) |
| POST | `/api/v1/process-template-versions/{processVersionId}/points` | Add a point for PFD, PFMEA, or CP at the end of a sub-process in a draft version (E-6, R-TPL-04) |
| PUT | `/api/v1/process-template-versions/{processVersionId}/points/{pointId}` | Change the text of a point, move, hide, or show it in a draft version (E-6, R-TPL-04) |
| GET | `/api/v1/system-settings` | System settings, today the sign date switch (D-52) |
| PUT | `/api/v1/system-settings` | Change the system settings (A-3, UC-07) |
| GET | `/api/v1/audit-log` | Audit log entries, newest first (A-4, admin account only) |
| GET | `/api/v1/audit-log/verification` | Check the whole hash chain of the audit log (A-4, admin account only) |
