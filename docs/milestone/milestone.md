<!-- Ekspor otomatis dari Milestone_Pembangunan_Web_PFMEA_Control_Plan_dan_PFD.docx (tracked changes diterima, sinkron dengan brief v1.15, 27 Oktober 2026). Jangan diedit; sumbernya docx di folder ini. "Keputusan:" = diputuskan pemilik, "Usulan:" dan "Tambah (usulan):" = belum. -->

# Milestone Pembangunan Web PFMEA, Control Plan, dan PFD

Disusun 22 Oktober 2026 · bahan belajar dan peta kerja · Go + PostgreSQL + SvelteKit · diperbarui 23 Oktober 2026 (sinkron dengan brief v1.15)

## Cara memakai dokumen ini

- Setiap milestone kecil, utuh dari database sampai layar, dan cukup untuk satu sesi kerja.

- Urutan kerja: tulis test dulu, buat kodenya berjalan, lalu rapikan.

- Mode mengajar: sebelum menulis kode, Claude Code menjelaskan konsep barunya. Sesudahnya ia memberi ringkasan, langkah yang bisa diulang, dan catatan keputusan beserta alasannya.

- Latihan wajib: tiap tahap ditutup dengan menulis ulang satu bagian kecil dengan tangan sendiri, tanpa AI. Itu cara paling jujur untuk menguji tujuan membangun ulang sendiri.

- Satu milestone selesai bila test lulus, hasilnya bisa didemokan, dan catatan keputusan sudah ditulis.

- Setiap tahap berakhir dengan gerbang: tinjauan singkat sebelum lanjut.

- **Urutan prioritas.** Bila dokumen ini berbeda dengan brief, yang berlaku adalah brief (brief bagian 0). Rujukan seperti R-API-03 atau Q-22 menunjuk ke ID di brief.

- **Keputusan:** **Kode dan UML.** Kode adalah proyek belajar pribadi, bukan pengembangan perusahaan. Perusahaan menerima spesifikasi dan UML, dan tim IT akan membangun versi mereka sendiri dari UML itu (D-33). Aplikasi berjalan di localhost sebagai demo untuk mematangkan konsep dan alur kerja (D-36), dan semua kode, termasuk tahap 9 dan 10, adalah latihan (D-37).

- **Keputusan:** **Kode sejajar dengan UML.** Mulai 27 Oktober 2026 kode dibangun milestone demi milestone bersama Claude Code, sejajar dengan UML yang tersisa, dan dipakai untuk mengevaluasi desain (D-77). Ini mengubah D-11 (spesifikasi dan UML sebelum kode).

- **Tambah (usulan):** **Catatan evaluasi desain.** Setiap temuan dari kode (celah, kontradiksi, aturan yang ambigu atau sulit dibangun) dicatat Claude Code di docs/evaluasi-desain.md dengan ID E-nn. Claude Code tidak memutuskannya sendiri dan tidak mengubah brief, UML, atau milestone. Anda yang memutuskan, lalu keputusannya masuk ke brief (brief bagian 15, butir 11).

- **Tambah (usulan):** **Urutan UML.** Selesaikan diagram kelas domain U5 dan state machine U6 sebelum Tahap 1, karena Tahap 0 hanya butuh entitas customer (RK-64).

- **Tambah (usulan):** **Kapasitas.** Catat jumlah sesi kerja nyata per minggu sejak tahap 0, lalu hitung rata-rata waktu per milestone setelah Gerbang 0. Angka itu dipakai untuk memperkirakan jadwal dan memilih cakupan QCC, bukan tebakan di awal.

## Hasil pengecekan versi (24 Oktober 2026)

- **Terverifikasi ada dan stabil:** Go 1.27.3, PostgreSQL 18.6, Node 24.22.1 LTS, SvelteKit 3.1.0, Svelte 5.57.2, Vite 8.3.5, TypeScript 6.0.3, Vitest 5.0.5, Playwright 1.64.1, pgx v5.11.0, sqlc v1.31.1, goose v3.29.0, River v0.49.0, excelize v2.11.0, PlantUML v1.2026.9. Daftar lengkap beserta tanggal rilisnya ada di brief bagian 7.

- **Syarat SvelteKit 3:** Node 22.17 atau lebih baru, TypeScript 6, Svelte 5.57.1 atau lebih baru, Vite 8.0.12 atau lebih baru. Konfigurasi pindah ke vite.config.ts (bukan svelte.config.js), dan alias impor \$lib menjadi \#lib.

  - **Usulan:** tulis perbedaan SvelteKit 3 (konfigurasi di vite.config.ts, alias \#lib) di CLAUDE.md, karena kode buatan AI bisa masih memakai pola SvelteKit 2 dan itu membingungkan saat belajar. Sudah dimasukkan ke CLAUDE.md yang dibuat 27 Oktober 2026.

- **Perlu perhatian:** TypeScript 7.0.2 sudah rilis tetapi belum diterima SvelteKit 3 dan svelte-check, jadi pakai 6.0.3. openapi-typescript 7.13.0 hanya menerima TypeScript 5 (Q-47). oapi-codegen v2.8.0 baru mendukung OpenAPI 3.1 secara awal, jadi kontrak memakai OpenAPI 3.0.4 (D-38). adapter-static 4.0.0 baru satu rilis, cek ulang di M0.1. Node 26 belum LTS. PostgreSQL 19 masih RC. River v0.49 dan Vitest 5 sekarang sudah stabil.

- **Belum dicek:** tidak ada lagi. Semua komponen di brief bagian 7 sudah dicek pada 24 Oktober 2026.

- **Aturan:** jangan menulis versi minimal yang belum dicek. Di M0.1 jalankan go list -m -versions dan npm view \<paket\> versions, pilih versi stabil terbaru, lalu catat dengan tanggal.

  - **Keputusan:** hanya versi stabil yang dipakai (D-44): tanpa alpha, beta, rc, next, atau nightly (R-VER-01).

  - **Usulan:** kriteria tambahan di brief bagian 7: runtime memakai jalur LTS atau yang masih didukung (R-VER-02), semua peer dependency cocok tanpa --force atau --legacy-peer-deps (R-VER-03), major baru dipakai setelah ada minimal satu rilis patch (R-VER-04), dan pakai patch terbaru lalu cek ulang di M0.1 (R-VER-05).

## Tahap S · Matangkan spesifikasi (sejajar dengan kode mulai 27 Oktober 2026, D-77)

- **S1 Perbarui spesifikasi.** Baris fleksibel, entitas perubahan tunggal, pewarisan template, penomoran, cetak, migrasi, dan mesin aturan dinonaktifkan. Selesai bila tidak ada kontradiksi antar dokumen. Termasuk bagian 16 sampai 19 brief v1.4 (API integrasi, portal customer, filter visibilitas, izin tampil) sebagai rancangan saja, tanpa dibangun.

- **S2 Terjemahkan ke UML.** Use case, diagram kelas atau ERD, diagram status untuk perubahan, revisi, dan nomor dokumen, serta diagram urutan untuk propagasi template dan yokoten. Selesai bila setiap skenario nyata bisa ditelusuri di diagram. Tambahkan jenis aktor manusia, service account, dan customer user (R-API-03), serta tiga gerbang tampil customer (brief bagian 19) sebagai diagram rancangan.

  - **Keputusan:** **langkah pertama pekerjaan adalah S2**, disusun langsung dari brief v1.7 (D-43). Alatnya PlantUML v1.2026.9 dengan C4-PlantUML v2.14.0, lewat image Docker resmi atau ekstensi editor (brief bagian 7).

  - **Usulan:** S1 dikerjakan sesudah S2, sebagai penyesuaian file spesifikasi lama terhadap UML, karena brief sudah menjadi spesifikasi terbaru.

  - **Usulan:** urutan di dalam S2: (1) glosarium U1 dan diagram konteks C4 U2, untuk batas sistem dan aktor; (2) daftar use case U3 tanpa detail; (3) diagram kelas domain U5; (4) state machine U6; (5) spesifikasi use case U3 dan skenario penerimaan U12; (6) activity diagram U4; (7) sequence diagram U7; (8) matriks keterlacakan U13, lalu tinjauan S6.

  - **Keputusan:** UML menjadi hasil utama untuk perusahaan, jadi harus bisa dipahami dan dipakai tim IT untuk membangun versi mereka tanpa melihat kode (D-33). Isi, notasi, dan alat mengikuti praktik terbaik saat ini (brief bagian 20, D-38): UML 2.5.1, model C4, ERD crow's foot, OpenAPI, catatan keputusan, skenario penerimaan, dan matriks keterlacakan, ditulis sebagai kode dengan PlantUML lalu diekspor ke SVG dan PDF. Di S2 dibuat diagram konsep dan perilaku (U1 sampai U7, U13). Diagram fisik menyusul per milestone dan difinalkan di M8.4.

  - **Usulan:** tambahkan status revisi Menunggu approval di diagram status revisi (draft, menunggu approval, released), dengan jalur dikembalikan ke draft yang dicatat oleh penyusun (D-30, R-REV-06). R-APR-01 mencetak hash sebelum rilis, dan R-PUB-07 serta R-PUB-08 perlu tahu approval sedang berjalan atau ditolak.

- **S3 Bahan nyata dari Anda.** Form Excel asli PFD, PFMEA, dan CP, satu contoh format SQAM customer, dan satu paket dokumen model nyata. Selesai bila data pilot siap.

  - **Keputusan:** data asli dipakai. Pengembangan memakai Claude Code yang terhubung ke repository GitHub private (D-29).

  - **Usulan:** simpan data asli di repository private terpisah dari kode (atau folder yang tidak di-commit bila bekerja lokal). Repo kode tetap bersih dari dokumen customer, dan data bisa dihapus sekaligus tanpa menulis ulang riwayat git (R-SEC-08).

  - **Usulan:** isi awal pustaka proses (M3.2) bisa diambil dari master template proses dan pustaka CP Generator yang sudah ada.

- **S4 Skenario di atas kertas.** Tulis 10 sampai 15 skenario (merakit model baru, yokoten ke lima model, dua orang mendaftar nomor bersamaan, migrasi revisi -04, perubahan surut SQAM) dan telusuri ke UML. Sertakan juga skenario SC-01 sampai SC-04 dari brief bagian 19.

  - **Keputusan:** tulis skenario dalam bentuk Given, When, Then, supaya tim IT bisa memakainya sebagai kriteria penerimaan (U12, D-38).

- **S5 Uji coba PDF sekali pakai.** Bandingkan Excel ke PDF lewat LibreOffice dengan HTML ke PDF lewat Chromium memakai form asli, font yang sama, dan teks panjang. Putuskan jalur utama dan opsi cadangan, lalu buang prototipenya.

  - **Usulan:** tambahkan kriteria uji karakter non-Latin (misalnya Jepang) bila form customer memakainya, dan pastikan lisensi font. Font bawaan Windows tidak boleh disalin sembarangan ke container Linux, jadi pilih font berlisensi jelas atau yang metriknya kompatibel.

  - **Usulan:** karena kode sekarang berjalan sejajar (D-77), uji ini cukup selesai sebelum Tahap 6, bukan sebelum menulis kode (R-PRT-03).

- **S6 Tinjauan.** Jalankan agen spec-reviewer, selesaikan temuannya, dan tulis catatan keputusan.

- **Tambah (usulan):** **S7 Ukur baseline sekarang.** Dengan cara kerja Excel saat ini, ukur waktu merakit satu set dokumen model baru, waktu satu yokoten ke beberapa model, jumlah nomor bentrok atau tidak sejajar, dan waktu mencari riwayat satu perubahan. Selesai bila angka sebelum tercatat beserta tanggal dan cara mengukurnya. Tanpa ini QCC tidak punya pembanding.

  - **Usulan:** contoh nyata untuk QCC dari dokumen TR-020: nama proses yang sama ditulis berbeda di PFD, PFMEA, dan CP; file SF-TR-020-03 berisi PFMEA SF-TR-022 untuk part lain; nomor baris riwayat revisi CP tidak urut (brief bagian 21, F-04 sampai F-06).

- **Tambah (usulan):** **S8 Konfirmasi dengan atasan lebih awal.** Minta konfirmasi tertulis atas dua hal: kode sebagai proyek belajar pribadi dengan UML sebagai hasil untuk perusahaan (D-33), dan pemakaian data asli dalam pengembangan (D-29). Tetapkan juga jadwal dan cakupan QCC (Q-16). Selesai bila keduanya terjawab tertulis.

## Tahap 0 · Persiapan dan dasar belajar

- **M0.1 Alat dan versi.** Pasang Go, Node 24 LTS, Docker, dan Git, lalu verifikasi versi paket dan catat di docs/versions.md. Konsep: toolchain dan penguncian versi. Selesai bila go version, node -v, dan PostgreSQL 18 di Docker berjalan.

  - **Keputusan:** cek ulang semua versi terhadap R-VER-01 sampai R-VER-05. Kontrak OpenAPI ditulis dalam 3.0.4 sampai generator di kedua sisi mendukung 3.1 penuh (D-38).

  - **Usulan:** sesi pertama Claude Code memakai docs/claude-code/prompt-sesi-01.md: orientasi, evaluasi awal desain, lalu M0.1. Aturan tetapnya ada di CLAUDE.md di root repo.

- **M0.2 Server Go pertama.** Endpoint /health dengan net/http dan test httptest. Konsep: package, struct, error, handler. Selesai bila test lulus.

- **M0.3 PostgreSQL dan pgx.** Buat tabel, isi, dan baca lewat pgx. Coba JSONB, indeks, dan transaksi di psql. Konsep: SQL dasar, connection pool, transaksi. Selesai bila satu fungsi menyimpan dan membaca baris dalam transaksi.

- **M0.4 SvelteKit pertama.** Halaman SPA dengan komponen dan state Svelte 5 yang memanggil /health. Konsep: runes, routing, fetch. Selesai bila halaman menampilkan status server.

- **M0.5 Irisan pertama: customer.** Migrasi goose, query sqlc, API daftar dan tambah customer, serta layar daftar. Konsep: migrasi, kode hasil generate, kontrak API. Selesai bila customer yang ditambah di layar tersimpan dan test lulus. Kontrak OpenAPI ditulis dulu (R-API-02), dengan satu format error berisi kode, pesan, dan request ID (R-API-13).

  - **Keputusan:** API-first: layar web memakai API terdokumentasi yang sama dengan yang nanti dipakai aplikasi lain, misalnya untuk dokumen PPAP lain seperti PSW (D-61, R-API-01, R-API-02).

  - **Usulan:** pilih generator TypeScript untuk OpenAPI di sini lewat uji kecil: openapi-typescript (hanya menerima TypeScript 5) atau @hey-api/openapi-ts (menerima TypeScript 6, masih 0.x) (Q-47).

- **Gerbang 0.** Tulis ulang versi kecil M0.5 untuk entitas lain tanpa AI.

## Tahap 1 · Fondasi

- **M1.1 Login lokal.** Pengguna, kata sandi argon2id, sesi, dan semua halaman butuh login. Konsep: hashing, cookie sesi, middleware. Selesai bila halaman tanpa login ditolak.

- **M1.2 Titik pemeriksaan izin.** Satu fungsi yang dipanggil di semua operasi, yang sejak awal menegakkan peran kasar: author, verifier, approver, dokumen kontrol, admin, dan viewer untuk QA, QC, dan internal audit (D-35, D-41). Selesai bila semua endpoint melewati fungsi itu dan test membuktikan setiap peran hanya bisa melakukan aksi yang diizinkan. Fungsi menerima aktor, aksi, dan resource; jenis aktor manusia, service, atau customer user (R-API-03), walau saat ini baru manusia yang dipakai.

  - **Keputusan:** pemetaan jabatan sesuai bagan organisasi NPI: author (penyusun) adalah PPAP Engineer, PPAP Staff, Project Leader, dan Project Engineer (D-63, D-67); verifier adalah Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller (D-45, D-51, D-54).

  - **Keputusan:** semua jabatan sampai level GM mendapat akun Viewer, termasuk approver, DCC, QA, QC, internal audit, dan tim WI. Approver tetap menandatangani di kertas (D-70, R-APR-13).

  - **Keputusan:** admin memakai akun admin terpisah yang hanya memegang hak admin. Untuk saat ini akun itu dipegang PPAP Staff, sehingga PPAP Staff punya dua akun: akun kerja dan akun admin. Akun admin tidak pernah memegang peran kerja (D-72, R-SEC-13).

  - **Keputusan:** aturan hak akses dikelola admin: peran adalah kumpulan hak yang isinya diatur admin, dengan empat level akses: Buat/Edit, Verifikasi, Lihat, dan Admin. Peta pengguna U3-peta-pengguna menjadi baseline (D-66, R-SEC-10).

  - **Keputusan:** akun admin terakhir tidak bisa dihapus atau kehilangan peran admin (D-73). Setiap akun admin ditautkan ke pemegangnya, sehingga audit log menampilkan akun dan orangnya (D-74). Catatan perubahan hak hanya dilihat lewat akun admin, tidak dibuka untuk NPI Manager (D-75).

  - **Usulan:** aturan pemisahan tugas lainnya tetap tidak bisa diubah siapa pun, dan setiap perubahan hak tercatat di audit log (R-SEC-11).

- **M1.3 Audit log tahan manipulasi.** Tabel append-only, role database terpisah tanpa hak UPDATE dan DELETE, trigger penolak, dan hash chain. Konsep: role database, trigger. Selesai bila percobaan mengubah log gagal. Setiap entri mencatat jenis aktor dan request ID (R-API-03, R-API-18). Audit log hanya dilihat lewat akun admin terpisah (D-72).

  - **Usulan:** simpan hash rantai terakhir di luar database secara berkala (misalnya di manifest backup harian), supaya penulisan ulang seluruh rantai oleh admin database tetap ketahuan.

- **M1.4 Master customer, model, part number.** CRUD dan layar, termasuk kode customer. Selesai bila relasi model ke customer berjalan. Termasuk master group model dan relasi model ke group model (D-39). Termasuk master Project Group (GR) dan alokasi setiap customer ke satu Project Group (D-54).

  - **Keputusan:** satu group hanya berada di dalam satu customer, dan satu model hanya boleh masuk satu group (D-42). Siapa yang membuat dan mengelola group masih terbuka (Q-46).

- **M1.5 Master proses.** ID tetap, nama tampilan, dan alias. Konsep: nama sebagai tampilan, ID sebagai identitas. Selesai bila nama proses bisa diubah tanpa memutus rujukan. Dikelola oleh PPAP Engineer (D-64).

- **M1.6 Master jenis dokumen dan jenis perubahan.** Awalan SH, SF, SC, jenis perubahan (isi, tata letak, administratif), sumber perubahan, kategori 4M, dan kebijakan approval per jenis sebagai data. Selesai bila semuanya bisa diedit admin.

  - **Usulan:** isi awal sumber perubahan mengikuti pemicu yang sudah dipakai sekarang, yaitu PP (pre-production), customer claim, dan ECR, selain yokoten, template, SQAM, dan temuan audit.

- **Tambah (usulan):** **M1.7 Backup data lokal.** Backup harian terenkripsi ke disk lain, enkripsi disk laptop, dan satu uji restore ke database kosong, segera setelah login dan audit log jadi. Konsep: backup dan restore PostgreSQL. Selesai bila restore berhasil dan langkahnya tertulis. Ini memindahkan sebagian M8.2 ke depan, karena data asli dipakai di localhost sejak awal (D-29, D-36).

- **Keputusan:** **M1.8 Jabatan dan rute approval.** Master jabatan sesuai bagan organisasi NPI (PPAP Engineer, PPAP Staff, keduanya pemegang peran dokumen kontrol (D-60, D-68), Project Engineer, Project Leader, APQP/PPAP Controller, NPI Manager, QA QC Manager, Advisor, Factory Manager), satu Project Engineer dan satu Project Leader per Project Group (D-54), dan rute approval berurutan dengan slot per tahap Prepared, Checked, dan Approved. Checked cukup satu tanda tangan dari Project Engineer atau Project Leader. Dua rute: sampai QA QC Manager dan sampai slot GM, yang saat ini diisi Advisor (D-56). Setiap customer punya rute default (D-45 sampai D-49). Setiap slot punya tanggal tanda tangan opsional yang bisa dimatikan dan disembunyikan, dan saat ini mati (D-52). Konsep: alur kerja sebagai data. Selesai bila admin bisa mengatur rute default customer tanpa mengubah kode.

  - **Usulan:** satu orang hanya boleh mengisi satu slot dalam satu rute, jadi pengecek berbeda dari penyusun (R-APR-12).

  - **Keputusan:** ID jabatan tetap dan namanya bisa diganti, sehingga NPI Manager bisa bernama PE Manager lagi atau slot GM pindah ke Factory Manager cukup lewat master data. Satu orang boleh memegang lebih dari satu jabatan, misalnya Project Leader GR3 yang merangkap APQP/PPAP Controller (D-58, R-APR-15).

- **Gerbang 1.** Tulis sendiri satu master data baru beserta test dan audit-nya.

## Tahap 2 · Penomoran dokumen

- **M2.1 Pola dan penghitung.** Pola nomor dengan token, penghitung per customer dan jenis dokumen. Selesai bila pola menghasilkan nomor seperti SC-KT-001. Kode customer di nomor harus sama dengan kode customer di master dan dicek saat pendaftaran (R-POR-01).

- **M2.2 Alokasi atomik.** Penghitung dinaikkan dalam transaksi, batasan unik pada nomor, dan kunci idempoten. Konsep: penguncian baris, idempotensi. Selesai bila 50 pendaftaran paralel menghasilkan 50 nomor berbeda tanpa lubang.

- **M2.3 Register dan status.** Reserved, In use, Released, Obsolete, Void, dengan pencarian dan filter. Selesai bila nomor tidak pernah dipakai ulang atau dihapus.

- **M2.4 Set dan format revisi.** Tombol daftarkan sebagai set yang menyejajarkan nomor, dan format revisi -00, -01 dari master. Selesai bila revisi awal bisa -04.

  - **Usulan:** tetapkan aturan untuk nomor yang dilompati saat daftarkan sebagai set. R-NUM-05 mengambil nomor bebas berikutnya di ketiga jenis, sehingga penghitung yang tertinggal melompati nomor, padahal R-NUM-03 menuntut tanpa lubang. Pilihan: nomor yang dilompati dicatat dengan status Dilewati, atau satu penghitung per customer untuk set (Q-05).

- **M2.5 Impor nomor lama.** Penghitung diatur ke nomor tertinggi, laporan lubang, nomor ganda, dan nomor tidak sejajar. Selesai bila impor satu customer menghasilkan laporan yang benar.

- **Gerbang 2.** Ulangi uji paralel dengan skenario baru yang Anda tulis sendiri.

## Tahap 3 · Template

- **M3.1 Format template.** Kolom ber-ID tetap, versi, metode risiko (RPN atau AP) dan tabel skala. Isi awal AIAG 4th. Selesai bila label kolom bisa diganti tanpa merusak data.

  - **Usulan:** isi awal format template diambil dari form nyata TR-020: PFMEA AIAG 4th (RPN) dan form CP SIIX, lengkap dengan simbol Critical Process, Revision, Important Character, dan Regulation. Aturan aksi PFMEA (RPN di atas 100 atau severity 8) hanya memberi tanda, tidak memblokir (R-TPL-11). Field fase CP dari manual AIAG 2024 bisa ditambahkan (R-TPL-12).

- **M3.2 Pustaka proses.** Proses, sub-proses, dan poin dengan ID tetap, beserta editor sederhana. Selesai bila contoh IQC dengan lima sub-proses tersimpan. Dikelola oleh PPAP Engineer (D-64).

- **M3.3 Cakupan dan prioritas.** Umum, customer, group model, dan model, dengan aturan yang paling spesifik menang. Selesai bila test prioritas lulus. Urutan: model, lalu group model, lalu customer, lalu umum (D-39).

  - **Usulan:** tambahkan test untuk model tanpa group dan untuk group tanpa template, karena keduanya harus jatuh ke tingkat customer.

- **M3.4 Versi template.** Draft dan terbit, tanpa propagasi dulu. Selesai bila versi terbit tidak bisa diubah.

- **Gerbang 3.** Buat satu template baru sendiri dari awal.

## Tahap 4 · Dokumen dan perakitan

- **M4.1 Dokumen dan baris.** Header dokumen terhubung ke nomor register, customer, model, dan part. Baris disimpan JSONB dengan ID tetap dan urutan. Selesai bila baris bisa disimpan dan dibaca. Setiap dokumen wajib punya customer_id (hook portal, brief bagian 17). Header dokumen menyimpan rute approval: default dari customer, bisa diganti dengan alasan yang tercatat di audit log (D-47).

  - **Usulan:** dokumen bisa menautkan dokumen yang saling bergantung, misalnya ESD control plan SC-ESD-K001, dan nomor WI atau SOP dipilih dari master dokumen terkait (R-DOC-10, R-DOC-11).

- **M4.2 Editor tabel.** Tabulator untuk PFMEA, CP, dan PFD: edit sel, tambah dan hapus baris. Konsep: komponen, state, simpan ke API. Selesai bila perubahan sel tersimpan.

- **M4.3 Penguncian optimistik.** Dua pengguna mengedit bersamaan tidak saling menimpa. Selesai bila konflik terdeteksi dan ditampilkan.

- **M4.4 Tempel dari Excel.** Pratinjau dan pemetaan kolom, baris janggal ditandai tapi tidak ditolak. Selesai bila blok sel dari Excel masuk benar.

- **M4.5 Perakitan dua panel.** Kotak centang sub-proses dan poin membuat baris tertaut ke template, dengan status dimasukkan atau dikecualikan. Selesai bila IQC General, MSD, dan PCB dengan sebagian poin dicoret menghasilkan dokumen yang benar.

- **M4.6 Urutan dan nomor operasi.** Drag and drop urutan proses, nomor operasi otomatis, dan bisa diubah manual. Selesai bila urutan baru tersimpan.

- **M4.7 Salin komposisi model lain.** Mulai dari komposisi model serupa lalu sesuaikan. Selesai bila salinan menjadi dokumen mandiri.

- **M4.8 Perbedaan lokal dan putus tautan.** Baris yang diubah ditandai berbeda dari template, putus tautan butuh alasan. Selesai bila perbedaan terlihat di tampilan banding.

- **M4.9 PFMEA dan CP berdampingan.** Tampilan per proses dan tanda dicek oleh siapa dan kapan. Selesai bila tanda tersimpan per proses per revisi. Untuk saat ini tanda dicek diberikan oleh PPAP Engineer (D-65).

- **M4.10 Diagram PFD (opsional).** Svelte Flow dan ELK. Selesai bila diagram terbentuk otomatis dari daftar proses.

  - **Usulan:** pakai simbol dan disposisi NG dari PFD TR-020 (process, alternate process, decision, connector, merge, document; scrap, NCP, rework), dan bentuk kolom Process Flow di CP dari PFD (R-DOC-09).

- **Gerbang 4.** Demo ke satu atau dua rekan dan ukur waktu merakit satu model. Angka ini bahan QCC.

## Tahap 5 · Perubahan, revisi, dan riwayat

- **M5.1 Entitas perubahan.** Jenis sumber, alasan, dan daftar dokumen terdampak, belum ada approval. Selesai bila satu perubahan bisa menunjuk beberapa dokumen.

- **M5.2 Lampiran.** Unggah ke storage privat dengan hash. Selesai bila lampiran tertaut ke perubahan.

- **M5.3 Verifikasi atasan.** Terima atau tolak dengan catatan wajib, pemohon tidak boleh memverifikasi dirinya sendiri. Selesai bila aturan itu diuji. Verifikatornya salah satu dari Project Engineer atau Project Leader dari Project Group customer itu, atau APQP/PPAP Controller, dan sistem mencatat siapa yang memverifikasi (D-51, D-54).

- **M5.4 Draft revisi dan versi baris.** Setiap dokumen terdampak mendapat draft. Selesai bila draft bisa diedit tanpa mengubah revisi yang berlaku.

- **M5.5 Snapshot rilis.** Isi dibekukan dengan hash saat rilis. Selesai bila revisi terbit tidak bisa diubah. Kolom customer-visible dan waktu terbit di revisi disiapkan sebagai hook (brief bagian 17), belum dipakai.

- **Tambah (usulan):** **M5.5a Status menunggu approval.** Draft beserta rute approval-nya dikunci saat dicetak untuk tanda tangan (R-APR-11), hash di cetakan sama dengan isi yang dikunci, edit sesudahnya membatalkan cetakan itu, dan penyusun (prepared) mencatat dikembalikan bila approver minta perubahan di kertas (D-30, R-REV-06). Selesai bila draft yang sedang dalam approval tidak bisa diedit tanpa membatalkan cetakannya.

- **M5.6 Diff dan riwayat.** Perbandingan per baris dan sel antar revisi, tampilan riwayat per dokumen dan per proses. Selesai bila perubahan antar dua revisi terlihat.

- **M5.7 Daftar periksa dampak.** Dokumen terdampak (PFD, PFMEA, CP, WI, standar inspeksi), keputusan, dan alasan, serta kategori 4M. Selesai bila daftar tersimpan di perubahan.

  - **Keputusan:** persetujuan customer atas perubahan (4M, PPAP ulang) diurus di luar sistem seperti sekarang (D-34).

  - **Usulan:** dokumen terdampak, termasuk WI, dipilih dari master dokumen terkait supaya nomornya sama dengan sumbernya (R-DOC-11).

- **M5.8 Propagasi template.** Pratinjau dampak, mode ikuti, kunci versi, dan lepas, cakupan penerbitan, penanganan konflik lokal, dan draft yang sedang terbuka. Selesai bila perubahan template menghasilkan draft di dokumen yang mengikuti.

- **M5.9 Yokoten.** Sumber perubahan, pemilihan target (customer, model, proses, pencarian teks), draft per target, dan papan status. Selesai bila status per model terlihat.

  - **Keputusan:** group model bisa dipilih sebagai target, supaya yokoten ke semua model dalam satu group cukup satu pilihan (D-40).

- **M5.10 Perubahan surut SQAM.** Rujukan revisi SQAM, tanggal berlaku, dan jenis perubahan. Revisi terbit tidak pernah ditulis ulang. Selesai bila semua dokumen aktif mendapat draft baru.

- **Gerbang 5.** Tulis sendiri satu alur perubahan kecil dari awal sampai rilis.

## Tahap 6 · Cetak dan rilis manual

- **M6.1 Layout sebagai data.** Layout dipisah dari data baris, ditulis developer dulu dengan skema yang siap diedit admin. Selesai bila satu layout customer bisa diganti tanpa mengubah kode.

  - **Keputusan:** kotak tanda tangan di cover dan di halaman bertanda tangan dibentuk dari rute approval revisi, tidak ditulis tetap di layout (D-46, R-APR-08). Selesai bila layout yang sama tercetak benar dengan rute sampai QA QC Manager dan rute sampai GM.

  - **Keputusan:** ikuti layout cover TR-020: judul cetak per slot (Process. Eng, Engineer, NPI Manager, QA / QC Mgr., General Mgr.) dengan nama pemegang, area CUSTOMER APPROVAL, dan blok tanda tangan berulang di setiap halaman dengan tulisan Refer approval sheet (R-PRT-07, R-APR-17, D-76).

- **M6.2 Excel A3 dan A4.** Isi layout, judul berulang, area cetak, dan footer. Selesai bila hasil cetak di printer asli rapi.

- **M6.3 PDF sesuai hasil uji S5.** Jalur utama dan opsi cadangan: (A) Excel ke PDF lewat LibreOffice, (B) HTML ke PDF lewat Chromium, (C) pustaka PDF Go dengan tata letak di kode, (D) Excel siap cetak dan PDF lewat Print to PDF manual sebagai cadangan terakhir. Selesai bila PDF form asli sama rapi dengan Excel.

- **M6.4 Watermark.** Controlled copy dan uncontrolled copy. Selesai bila draft selalu berwatermark.

- **M6.5 QR dan hash.** Halaman verifikasi menampilkan status berlaku, digantikan, atau dibatalkan. Selesai bila QR bisa dinyalakan dan dimatikan, dan saat menyala QR dari cetakan membuka halaman yang benar.

  - **Keputusan:** QR opsional dengan saklar, dan saat ini mati karena QR belum dipakai. Halaman verifikasi tetap bisa dibuka dari dalam sistem (D-62).

  - **Usulan:** satu saklar global untuk admin, dan hash pendek di cetakan ikut saklar yang sama (R-APR-16).

  - **Keputusan:** pengguna halaman verifikasi adalah QA, QC, dan internal audit. Leader ke atas boleh membawa HP di area produksi (D-32).

  - **Usulan:** jadikan alamat dasar di QR sebagai pengaturan. Di demo localhost, HP hanya bisa membuka halaman verifikasi bila HP dan laptop ada di jaringan yang sama dan QR memakai alamat IP laptop, bukan localhost (D-36).

- **M6.6 Catatan cetak.** Siapa, kapan, dan nomor salinan. Selesai bila setiap cetakan rilis tercatat.

  - **Usulan:** dari catatan cetak, buat daftar salinan terkendali yang harus ditarik saat revisi baru rilis.

- **M6.7 Rilis manual.** Kotak tanda tangan di cetakan, unggah scan, dan dokumen kontrol mencatat rilis dengan pencatat berbeda dari penyusun. Selesai bila rilis tanpa scan ditolak.

  - **Keputusan:** urutan tanda tangan berurutan: Checked, NPI Manager, QA QC Manager, lalu slot GM bila ada di rute (D-49, D-56).

  - **Keputusan:** tanggal tanda tangan per slot disiapkan tetapi dimatikan. Saat ini hanya tanggal rilis yang dicatat (D-52, D-53).

  - **Keputusan:** tanggal rilis diisi otomatis saat dicetak untuk tanda tangan, dan tanggal berlaku sama dengan tanggal rilis. Aturan ini harus bisa diubah nanti, misalnya tanggal berlaku diisi DCC (D-59, R-REV-08).

  - **Keputusan:** peran dokumen kontrol saat ini dipegang PPAP Staff dan PPAP Engineer, yang saling mencatat rilis dokumen yang disusun yang lain, sehingga pencatat selalu berbeda dari penyusun. Nanti bisa ditambah DCC (D-60, D-68).

  - **Usulan:** saat mencatat rilis, dokumen kontrol mengonfirmasi setiap slot sesuai urutan rute, dan rilis ditolak bila ada slot kosong (R-APR-10). Bila rute dokumen berbeda dari default customer, alasannya ditampilkan di layar ini (RK-60).

  - **Usulan:** (opsional, bila QR menyala) saat scan diunggah, sistem membaca QR di scan dan mencocokkan nomor dokumen, revisi, dan hash. Scan yang tidak cocok ditolak.

- **M6.8 Tabel riwayat revisi di cetakan.** Otomatis dari riwayat. Selesai bila tabel tampil di Excel dan PDF.

- **M6.9 Label status dan footer rilis di PDF.** Label draft, pending, atau effective, content hash (R-PUB-05), dan catatan rilis pengganti tanda tangan di PDF buatan sistem (R-PUB-06). Selesai bila PDF menampilkan status, hash, dan catatan rilis yang benar.

  - **Usulan:** label pending hanya dibutuhkan bila after_check dipakai (R-PUB-07, default mati), jadi bagian itu bisa ikut ditunda ke tahap 10.

  - **Usulan:** nomor baris dan format tanggal di tabel riwayat dibuat otomatis, dan baris yang berubah diberi tanda segitiga dengan nomor revisi yang diambil dari diff (R-REV-04, R-REV-09).

- **Gerbang 6.** Cetak satu paket lengkap di printer pabrik dan tinjau bersama pengguna.

## Tahap 7 · Migrasi dan bukti nilai

- **M7.1 Form dokumen lama.** Nomor asli, revisi saat ini, tanggal berlaku, penyusun, dan approver, dengan status baseline migrasi. Selesai bila dokumen lama masuk di revisi -04.

- **M7.2 Riwayat revisi sebagai teks.** Revisi, tanggal, dan deskripsi dari tabel Excel lama. Selesai bila riwayat tampil lengkap di cetakan.

- **M7.3 Pembanding Excel.** Unggah Excel lama hanya untuk dibandingkan dengan hasil perakitan. Selesai bila selisih tampil.

- **M7.4 Penautan baris.** Sistem mengusulkan pasangan baris dengan poin template lewat kemiripan teks, manusia mengonfirmasi. Selesai bila dokumen hasil tempel bisa menerima propagasi.

- **M7.5 Verifikasi hardcopy.** Konfirmasi terhadap dokumen bertanda tangan, status Released (migrated), peran khusus, dan saklar mode migrasi. Selesai bila mode bisa dimatikan admin. Peran khusus migrasi dipegang PPAP Staff dan PPAP Engineer untuk saat ini (D-69).

- **M7.6 Pilot.** Migrasikan dua sampai tiga model dari satu customer dan catat waktu per dokumen. Selesai bila estimasi usaha migrasi tersedia.

  - **Usulan:** pilot berjalan di localhost dengan backup terenkripsi (M1.7, D-36). Untuk versi perusahaan, rancang migrasi saat disentuh di UML: register nomor dimigrasi semua di awal, sedangkan isi dokumen dimigrasi ketika dokumen itu perlu revisi (R-MIG-08). Dengan 380+ CP saja, belum termasuk PFMEA dan PFD, migrasi sekaligus sangat berat.

- **M7.7 Dasbor ukuran.** Persentase baris yang mengikuti template, dokumen per status, status yokoten, dan waktu revisi. Selesai bila angka bisa dipakai di presentasi QCC.

## Tahap 8 · Penguatan dan serah terima

- **M8.1 Peran dan hak akses.** Hak per peran dan cakupan customer, row-level security di PostgreSQL, dan test yang mencoba mengakses customer lain. Selesai bila semua percobaan lintas customer ditolak. Ini juga syarat sebelum tahap 9: tidak ada token integrasi sebelum M8.1 selesai (D-19, R-API-06).

  - **Keputusan:** peran kasar sudah ditegakkan sejak M1.2 (D-35). Di sini tinggal matriks wewenang, cakupan customer, row-level security, dan test lintas customer.

  - **Keputusan:** cakupan: Project Leader dan Project Engineer, baik saat menyusun maupun memverifikasi, hanya customer di Project Group-nya (D-71, R-SEC-12). Layar admin untuk mengubah isi peran juga dibuat di sini (D-66).

  - **Usulan:** PPAP Engineer, PPAP Staff, APQP/PPAP Controller, dan dokumen kontrol bekerja untuk semua customer (R-SEC-12).

- **M8.2 Unggahan aman dan backup.** Pemindaian antivirus, backup terenkripsi, dan uji restore. Selesai bila restore berhasil di mesin lain.

  - **Usulan:** di localhost, ClamAV bisa dijalankan di Docker sebagai latihan, tapi butuh RAM cukup besar. Untuk versi IT, catat di dokumen IT bahwa pemindaian bisa memakai antivirus perusahaan.

- **M8.3 Deploy.** Docker Compose dan Caddy dengan HTTPS lokal. Selesai bila aplikasi berjalan di localhost lewat Docker Compose.

  - **Keputusan:** server uji dan sertifikat perusahaan berlaku untuk versi IT, bukan untuk demo (D-36).

- **M8.4 Dokumen untuk IT dan buku panduan operasional.** Tanpa kode: arsitektur, kebutuhan server, keamanan, backup, dan daftar permintaan ke IT. Selesai bila IT bisa menilai tanpa melihat kode. Setelah tahap 9, ditambah kontrak OpenAPI integrasi dan catatan keamanannya (R-API-17).

  - **Keputusan:** isi utamanya paket UML final (U1 sampai U14, brief bagian 20) dan spesifikasi, cukup bagi tim IT untuk membangun versi mereka sendiri (D-33, D-38). Tim IT kemungkinan meninjau setelah seluruh desain selesai. Peninjau dan waktunya belum diputuskan (Q-44).

  - **Usulan:** versi awal dokumen ini sudah dibawa di S8; di sini tinggal versi final.

- **M8.5 Presentasi QCC.** Kerangka cerita QCC dengan data dari M4 sampai M7. Selesai bila semua angka berasal dari pengukuran nyata.

  - **Usulan:** angka sebelum diambil dari S7. Angka sesudah berasal dari demo terukur di localhost (M7.7, D-36) dengan tugas yang sama seperti baseline, dan disebut sebagai hasil demo di presentasi.

## Tahap 9 · API integrasi (latihan, setelah MVP)

- **Keputusan:** **Syarat masuk.** MVP sudah didemokan dan M8.1 selesai. Semua kode tahap ini adalah latihan di localhost (D-37). Konsumen nyata (Q-22) baru ditetapkan untuk versi IT.

  - **Usulan:** untuk latihan, buat konsumen tiruan berupa skrip kecil yang membaca data rilis dan mengirim satu usulan perubahan.

- **M9.1 Service account dan token.** Nama, tim pemilik, tujuan, scope, customer yang diizinkan, jaringan, dan masa berlaku. Token acak ditampilkan sekali, disimpan sebagai hash, bisa dirotasi dan dicabut (R-API-04, R-API-05). Selesai bila token kedaluwarsa atau dicabut selalu ditolak.

- **M9.2 Kontrak integrasi terpisah.** /integration/v1 dengan file OpenAPI sendiri, aturan versi, dan contract test di CI (R-API-02, R-API-15). Selesai bila perubahan UI API tidak memecahkan contract test integrasi.

- **M9.3 Endpoint baca.** Data rilis saja secara default, paginasi cursor, filter updated_since, dan satu format error (R-API-12, R-API-13). Selesai bila sinkronisasi bertahap berjalan dan token satu customer tidak bisa membaca customer lain.

- **M9.4 Usulan perubahan.** Penulisan dari luar hanya membuat perubahan berisi operasi patch terhadap revisi dasar, dengan Idempotency-Key, konflik 409, dan on_behalf_of yang hanya dicatat (R-API-07, R-API-09 sampai R-API-11). Selesai bila usulan dari luar menunggu verifikasi manusia dan tidak ada jalur API untuk verifikasi, approval, atau rilis.

- **M9.5 Ekspor asinkron dan batas laju.** Job ekspor, polling, unduh, dan 429 dengan Retry-After (R-API-14). Selesai bila ekspor besar tidak memperlambat layar pengguna.

- **M9.6 Webhook (opsional).** Ditandatangani HMAC, dikirim dari outbox lewat antrean job, dan konsumen harus idempoten (R-API-16).

- **Gerbang 9.** Tulis sendiri satu endpoint baca baru beserta contract test-nya.

## Tahap 10 · Portal customer (latihan)

- **Keputusan:** **Syarat masuk.** Tahap 9 selesai. Untuk latihan, Q-23, Q-24, dan Q-29 sampai Q-33 memakai jawaban sementara di brief. Persetujuan manajemen dan legal (RK-35) serta tinjauan keamanan IT (RK-33) berlaku untuk versi IT (D-37).

- **Tambah (usulan):** **M10.0a Paket kiriman customer.** Latihan kecil sebelum portal: ekspor paket per customer berisi register dokumen dan PDF revisi yang berlaku. Di UML untuk IT, catat paket ini sebagai alternatif yang jauh lebih murah daripada portal, karena tidak ada server yang terbuka ke luar.

- **M10.1 Identitas customer user.** Kumpulan identitas terpisah, hanya lewat undangan, MFA, dan sesi pendek (R-POR-04). Selesai bila akun customer tidak bisa masuk ke UI internal.

- **M10.2 Read model dan register portal.** Data rilis yang boleh tampil dipublikasikan ke portal terpisah dan dibatasi customer_id (R-POR-01, R-POR-06, R-POR-08). Selesai bila test lintas customer (akun KT membuka dokumen KO) selalu gagal.

- **M10.3 Kebijakan publikasi per customer.** file_source, permission_roles, dan earliest_stage, berversi dan tercatat di audit log (R-PUB-02). Selesai bila perubahan kebijakan berlaku sejak waktu efektifnya.

- **M10.4 Izin tampil.** Konfirmasi per peran dengan autentikasi ulang, terikat content hash, dan gagal tertutup (R-PUB-03 sampai R-PUB-05, R-PUB-08). Selesai bila edit setelah izin otomatis membatalkan izin dan portal tetap menampilkan file lama.

- **M10.5 Antrean izin dan monitor keterlambatan** (R-PUB-10). Selesai bila dokumen yang tampilan portalnya tertinggal terlihat beserta umurnya.

- **M10.6 Unduhan berwatermark dan log akses.** Watermark customer, pengguna, dan tanggal; setiap lihat dan unduh tercatat (R-POR-05, R-PUB-12).

- **M10.7 Halaman verifikasi.** Status, kecocokan hash, dan waktu izin tampil diberikan, dengan login (R-PUB-09, Q-31).

- **M10.8 Ganti sumber file.** Batch per customer dengan status per dokumen, bisa dibalik, dan scan tetap internal (R-PUB-11, SC-02). Selesai bila SC-02 berjalan dari awal sampai akhir di data uji.

- **Gerbang 10.** Uji penetrasi oleh pihak yang ditunjuk IT, lalu tulis sendiri satu test lintas customer baru.

  - **Usulan:** untuk latihan di localhost, ganti uji penetrasi oleh IT dengan pemindaian otomatis sendiri, misalnya OWASP ZAP. Uji penetrasi tetap syarat untuk versi IT.

## Ditunda, tidak untuk sekarang

- Mesin aturan 32 aturan (dinonaktifkan, bisa dipakai nanti sebagai audit opsional).

- Login Active Directory atau LDAP.

- Approval di aplikasi dan tanda tangan digital (hanya rancangan UML, dievaluasi lagi nanti).

- Struktur AIAG-VDA penuh (saat ini kompatibel dengan form saja).

- **Keputusan:** Field internal dan filter visibilitas customer ditahan (D-31): atribut audiens field, layout customer dan internal, watermark INTERNAL, print manifest, dan profil visibilitas (brief bagian 18). Selama ditahan, keluaran untuk customer memuat semua field.

- Desainer layout untuk admin (layout tetap disimpan sebagai data, R-PRT-02).

- Penyembunyian baris atau proses dari customer (R-VIS-06) dan profil visibilitas per model (R-VIS-11).

- Pembaruan real-time lewat WebSocket (Q-13); penguncian optimistik dulu.

- **Keputusan:** Tahap 9 dan 10 tidak dimulai sebelum MVP didemokan dan M8.1 selesai (D-37).

- Hak akses detail sebelum M8.1.
