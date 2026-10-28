# Evaluasi desain dari kode

Mulai 27 Oktober 2026 kode dibangun sejajar dengan UML, dan dipakai untuk menguji desain (D-77). File ini mencatat setiap temuan dari kode yang menyentuh brief, UML, atau milestone.

## Aturan

- Claude Code menulis temuan, dan pemilik yang memutuskan.
- Claude Code tidak mengubah brief, UML, atau milestone. Keputusan pemilik dicatat di brief sebagai item D, lalu ID-nya ditulis di kolom Status.
- Satu temuan untuk satu entri. ID berurutan (E-01, E-02, dan seterusnya) dan tidak pernah dipakai ulang.
- Asumsi sementara di kode ditandai dengan komentar `// ASUMSI E-nn`, supaya mudah dicari setelah temuannya diputuskan.

**Jenis temuan**
- **Kontradiksi:** dua aturan atau dokumen saling bertentangan.
- **Celah:** kode butuh aturan yang belum ada.
- **Ambigu:** aturan bisa dibaca lebih dari satu cara.
- **Sulit dibangun:** aturannya jelas, tapi biayanya tinggi atau berisiko. Sertakan alternatifnya.
- **Penyederhanaan:** desain bisa dibuat lebih sederhana tanpa kehilangan tujuannya.

**Status:** Terbuka, Diputuskan (D-xx), Ditolak, atau Ditunda.

## Ringkasan

| ID | Judul | Jenis | Milestone | Menghambat? | Status |
|---|---|---|---|---|---|
| E-01 | Daftar peran M1.2 berbeda dengan peta pengguna U3 | Kontradiksi | M1.2 | Ya, untuk M1.2 | Diputuskan, D menunggu brief |
| E-02 | Akun dan orang belum dibedakan di model data | Celah | M1.1, M1.2, M1.8 | Ya, untuk M1.1 | Diputuskan, D menunggu brief |
| E-03 | Apakah Tahap 1 menunggu U5 dan U6 | Ambigu | Tahap 1 | Ya, untuk mulai Tahap 1 | Diputuskan, D menunggu brief |
| E-04 | Cakupan Project Group dibutuhkan sebelum M8.1 | Kontradiksi | M1.2, M5.3, M8.1 | Tidak (perlu sebelum M5.3) | Diputuskan, D menunggu brief |
| E-05 | Dua sumber data untuk cakupan Project Group | Ambigu | M1.2, M1.4, M1.8 | Tidak (perlu sebelum M1.4) | Diputuskan, D menunggu brief |
| E-06 | Hash chain audit log pada penulisan paralel | Sulit dibangun | M1.3, M2.2 | Tidak (perlu sebelum M1.3) | Diputuskan, D menunggu brief |
| E-07 | Cara membuat akun admin pertama belum ada | Celah | M1.1, M1.2 | Tidak | Diputuskan, D menunggu brief |
| E-08 | Aturan login dan sesi belum ditetapkan | Celah | M1.1 | Tidak | Diputuskan, D menunggu brief |
| E-09 | Aturan kode customer belum lengkap | Celah | M0.5, M1.4 | Tidak | Diputuskan, D menunggu brief |
| E-10 | Siapa yang mengelola group model dan model | Kontradiksi | M1.2, M1.4 | Tidak | Diputuskan, D menunggu brief |
| E-11 | Kode customer pernah diganti (WI SW-ZZ-000) | Kontradiksi | M1.4, M2.1, M2.5 | Tidak (perlu sebelum M1.4) | Diputuskan, D menunggu brief |
| E-12 | Struktur nomor di WI SW-ZZ-000 lebih rinci daripada pola di brief | Celah | M2.1, M2.4, M2.5 | Tidak (perlu sebelum M2.1) | Diputuskan, D menunggu brief |
| E-13 | Versi terbaru berbeda dari brief bagian 7 | Kontradiksi | M0.1 | Tidak | Diputuskan, D menunggu brief |
| E-14 | R-VER-04 belum menjawab line tunggal dan versi 0.x | Ambigu | M0.1, M0.4 | Tidak untuk M0.1, ya untuk M0.4 | Diputuskan, D menunggu brief |
| E-15 | Belum ada langkah desain layar (wireframe) di milestone dan paket UML | Celah | Tahap 4, paket UML | Tidak (perlu sebelum Tahap 4) | Diputuskan, D menunggu brief |
| E-16 | Memindah dokumen aktif ke nomor baru setelah kode customer berganti | Celah | M2.1, M2.3, M2.5 | Tidak (perlu sebelum M2.1) | Diputuskan, D menunggu brief |
| E-17 | Kedudukan /health terhadap /api/v1 dan kontrak OpenAPI | Ambigu | M0.2, M0.4, M0.5 | Tidak (perlu sebelum M0.5) | Terbuka |

## Rincian

<!--
Salin blok ini untuk setiap temuan baru.

### E-nn · Judul singkat

- **Tanggal dan milestone:**
- **Jenis:**
- **Rujukan:** ID di brief, UML, atau milestone (R-, D-, UC-, U, M)
- **Temuan:** apa yang ditemukan saat menulis kode.
- **Dampak bila dibiarkan:**
- **Opsi:** (a) ... (b) ...
- **Rekomendasi Claude Code (usulan):**
- **Asumsi sementara di kode:** isi bila temuan tidak menghambat.
- **Menghambat milestone?** Ya atau Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**
-->

### E-01 · Daftar peran M1.2 berbeda dengan peta pengguna U3

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1 (sebelum kode). Berdampak ke M1.2.
- **Jenis:** Kontradiksi
- **Rujukan:** R-SEC-09, D-35, D-41, brief bagian 1 (System roles) dan bagian 9, M1.2, R-SEC-10, D-66, D-70, U3-peta-pengguna bagian 1 sampai 3, U1 bagian 7.
- **Temuan:** R-SEC-09, brief bagian 1, dan M1.2 menyebut enam peran kasar yang ditegakkan sejak M1.2: author, verifier, approver, document control, admin, dan viewer. U3-peta-pengguna, yang sudah disetujui sebagai baseline (D-66), memakai delapan peran: Penyusun, Pengelola template, Pemeriksa PFMEA–CP, Verifikator, Dokumen kontrol, Petugas migrasi, Viewer, dan Admin. Peran approver tidak ada di U3 (approver memakai Viewer, D-70), sedangkan tiga peran U3 tidak ada di R-SEC-09. Selain itu brief bagian 9 masih menulis "every logged-in user can do everything until M8.1", yang bertentangan dengan D-35. Terakhir, tidak jelas apakah M1.2 sudah menyimpan peran sebagai data (peran berisi hak L-1 sampai A-4, sesuai D-66) atau memakai peran tetap di kode, karena layar admin untuk mengubah isi peran baru dijadwalkan di M8.1.
- **Dampak bila dibiarkan:** test M1.2 ("setiap peran hanya bisa melakukan aksi yang diizinkan") tidak bisa ditulis tanpa daftar peran dan hak yang pasti. Bila M1.2 memakai enam peran tetap di kode, M8.1 harus membongkarnya.
- **Opsi:** (a) M1.2 memakai delapan peran U3 dan katalog hak L-1 sampai A-4 sebagai data awal (seed). Isi peran belum bisa diedit lewat layar sampai M8.1. R-SEC-09 dan brief bagian 9 diperbarui. (b) M1.2 memakai enam peran R-SEC-09 sebagai konstanta di kode, lalu M8.1 menggantinya dengan model D-66. (c) Seperti (a), tetapi peran approver tetap dibuat di katalog dan belum diberikan ke siapa pun sampai approval di aplikasi (R-APR-04).
- **Rekomendasi Claude Code (usulan):** (a). U3 adalah baseline yang disetujui, dan tabel peran–hak tidak jauh lebih sulit dibuat daripada konstanta, tapi menghindari bongkar ulang di M8.1. Peran approver tidak dibuat dulu karena approval di aplikasi ditunda.
- **Asumsi sementara di kode:** belum ada; temuan ini tidak menyentuh Tahap 0.
- **Menghambat milestone?** Ya, untuk M1.2. Tidak untuk Tahap 0.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: delapan peran U3 dan katalog hak L-1 sampai A-4 sebagai data awal di M1.2, tanpa peran approver dulu ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-02 · Akun dan orang belum dibedakan di model data

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.1, M1.2, dan M1.8.
- **Jenis:** Celah
- **Rujukan:** D-58, D-72, D-74, R-SEC-11, R-SEC-13, R-APR-12, R-CHG-02, R-APR-02, brief bagian 4 (role, permission, user_role), brief bagian 8 (external identity field, employee number), M1.1, M1.2, M1.8.
- **Temuan:** PPAP Staff punya dua akun, akun kerja dan akun admin (D-72). Akun admin ditautkan ke "pemegangnya" (D-74). Satu orang boleh memegang beberapa jabatan (D-58). Aturan pemisahan tugas membandingkan orang, bukan akun: "one person fills at most one slot" (R-APR-12) dan "a requester never verifies their own request" (R-CHG-02). Model domain di brief bagian 4 hanya punya user, role, dan user_role. Tidak ada entitas orang yang terpisah dari akun, dan tidak jelas "pemegang" itu apa: akun kerja, data orang (nama, nomor karyawan), atau teks nama.
- **Dampak bila dibiarkan:** tabel pengguna yang dibuat di M1.1 harus diubah lagi di M1.2 atau M1.8. Pemeriksaan pemisahan tugas bisa terlewat bila yang dibandingkan akun, karena satu orang bisa memakai dua akun.
- **Opsi:** (a) Dua entitas: Orang (nama, nomor karyawan opsional) dan Akun (jenis kerja atau admin, login, hash password, kolom identitas eksternal untuk LDAP nanti). Satu orang punya paling banyak satu akun kerja dan paling banyak satu akun admin. Jabatan dipegang Orang. Aturan pemisahan tugas membandingkan Orang. (b) Hanya Akun. Akun admin punya rujukan ke akun kerja pemegangnya, dan pemisahan tugas membandingkan akun kerja. (c) Hanya Akun, dengan nama pemegang sebagai teks.
- **Rekomendasi Claude Code (usulan):** (a). Paling jelas untuk diagram kelas U5 dan untuk pemisahan tugas, dan cocok dengan hook "identity is bound to an employee number" di brief bagian 8.
- **Asumsi sementara di kode:** belum ada; temuan ini tidak menyentuh Tahap 0.
- **Menghambat milestone?** Ya, untuk M1.1 (skema pengguna). Tidak untuk Tahap 0.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: entitas Orang dan Akun dipisah, pemisahan tugas membandingkan Orang ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-03 · Apakah Tahap 1 menunggu U5 dan U6

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke awal Tahap 1.
- **Jenis:** Ambigu
- **Rujukan:** D-77, D-11, D-43, RK-64, R-UML-03, milestone "Cara memakai dokumen ini" (Tambah (usulan): Urutan UML), docs/uml/README.md (U5 dan U6 berstatus Belum).
- **Temuan:** D-77 memulai kode sejajar dengan UML. Usulan di milestone dan mitigasi RK-64 meminta diagram kelas domain U5 dan state machine U6 selesai sebelum Tahap 1. Di paket UML draft 0.7, U5 dan U6 masih berstatus Belum. Karena usulan ini belum diputuskan, tidak jelas apakah M1.1 boleh dimulai setelah Gerbang 0 bila U5 dan U6 belum ada. Beberapa temuan di file ini (E-01, E-02, E-05) sebenarnya pertanyaan kelas domain yang akan dijawab U5.
- **Dampak bila dibiarkan:** bila usulan diterima, Tahap 1 berhenti menunggu pekerjaan desain. Bila ditolak, kode Tahap 1 menentukan struktur entitas lebih dulu dan U5 mengikuti kode, dengan risiko yang disebut RK-64.
- **Opsi:** (a) Terima usulan: U5 dan U6 selesai sebelum M1.1. (b) Tolak usulan: Tahap 1 jalan, temuan dicatat di sini, dan U5 disusun dari kode beserta keputusan pemilik. (c) Jalan tengah: sebelum M1.1, U5 dibuat hanya untuk entitas yang disentuh Tahap 1 (akun dan orang, peran dan hak, jabatan, Project Group, rute approval, audit log). U6 menyusul sebelum Tahap 2.
- **Rekomendasi Claude Code (usulan):** (c). Tahap 0 tidak terhambat karena hanya butuh entitas customer.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak untuk Tahap 0. Ya, untuk mulai Tahap 1.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (b): Tahap 1 tidak menunggu U5 dan U6, dan UML diperbarui mengikuti kode yang dikembangkan ("untuk UML akan kita update sesuai code dikembangkan", 27 Oktober 2026). Ini berbeda dari rekomendasi (c).

### E-04 · Cakupan Project Group dibutuhkan sebelum M8.1

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.2, M5.3, dan M8.1.
- **Jenis:** Kontradiksi
- **Rujukan:** D-71, R-SEC-12, R-CHG-02, R-APR-07, D-51, D-54, R-SEC-02, D-35, D-19, R-API-06, M1.2, M5.3, M8.1.
- **Temuan:** D-71 (DECIDED) membatasi Project Leader dan Project Engineer ke customer di Project Group sendiri. R-CHG-02 (DECIDED) mensyaratkan verifikator dari Project Group customer itu. Tetapi R-SEC-02, R-SEC-12, dan M8.1 menjadwalkan penegakan cakupan customer di M8.1, dan D-19 menyatakan cakupan customer belum ditegakkan sebelum M8.1. Jadi antara M1.2 dan M8.1, termasuk saat verifikasi di M5.3, dua aturan DECIDED itu tidak ditegakkan, kecuali cakupan ditegakkan lebih awal dari jadwal.
- **Dampak bila dibiarkan:** test M5.3 tidak bisa membuktikan R-CHG-02 sepenuhnya, dan pada pilot di localhost seorang Project Leader bisa menyusun atau memverifikasi dokumen customer group lain.
- **Opsi:** (a) Sejak M1.2 fungsi izin menerima resource beserta customer-nya, dan sudah mengecek cakupan Project Group untuk peran yang punya cakupan. M8.1 tinggal menambah row-level security dan matriks wewenang detail. (b) Cakupan baru ditegakkan di M8.1. Sampai itu, D-71 dan bagian Project Group dari R-CHG-02 dicatat sebagai pengecualian yang diketahui. (c) Cakupan ditegakkan mulai M5.3, khusus untuk verifikasi.
- **Rekomendasi Claude Code (usulan):** (a). Pengecekan cakupan di lapisan layanan cukup murah (customer, lalu Project Group, lalu apakah orang itu Project Leader atau Project Engineer group itu). Row-level security di PostgreSQL tetap di M8.1.
- **Asumsi sementara di kode:** bila belum diputuskan di M1.2, fungsi izin tetap menerima resource lengkap supaya opsi (a) bisa ditambah tanpa mengubah pemanggilnya.
- **Menghambat milestone?** Tidak untuk Tahap 0 dan M1.2. Harus diputuskan sebelum M5.3.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: cakupan Project Group dicek di fungsi izin sejak M1.2; row-level security tetap di M8.1 ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-05 · Dua sumber data untuk cakupan Project Group

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.2, M1.4, dan M1.8.
- **Jenis:** Ambigu
- **Rujukan:** D-54, D-66, D-71, brief bagian 4 (user_role dengan cakupan; "Each project group names its Project Leader and Project Engineer"), U3-peta-pengguna bagian 2 (rangkap jabatan Project Leader GR3 dan APQP/PPAP Controller), M1.4, M1.8.
- **Temuan:** Cakupan seorang Project Leader atau Project Engineer bisa diambil dari dua tempat: (1) cakupan pada peran pengguna (semua customer, atau satu atau lebih Project Group) yang diatur admin (D-66), dan (2) master Project Group yang menyebut siapa Project Leader dan Project Engineer-nya (D-54). Bila admin mengganti Project Leader di master Project Group tetapi tidak mengubah cakupan perannya, atau sebaliknya, keduanya tidak sejalan. Pada rangkap jabatan Project Leader GR3 dan APQP/PPAP Controller, cakupannya berbeda per peran (menyusun hanya GR3, memverifikasi semua customer).
- **Dampak bila dibiarkan:** data yang sama tersimpan di dua tempat dan bisa tidak sinkron, sehingga hasil pemeriksaan izin bergantung pada tabel mana yang dibaca.
- **Opsi:** (a) Master Project Group adalah sumber tunggal untuk cakupan Project Leader dan Project Engineer. Cakupan pada peran hanya dipakai untuk peran yang tidak terikat group, misalnya "semua customer". (b) Cakupan pada peran adalah sumber tunggal. Master Project Group hanya dipakai untuk nama yang dicetak di slot (R-APR-17), bukan untuk izin. (c) Keduanya disimpan, dan sistem memaksa keduanya sejalan saat salah satu diubah.
- **Rekomendasi Claude Code (usulan):** (a). D-54 sudah menetapkan satu Project Leader dan satu Project Engineer per group, jadi jabatan itu sendiri sudah menyiratkan cakupannya.
- **Asumsi sementara di kode:** belum ada; temuan ini tidak menyentuh Tahap 0.
- **Menghambat milestone?** Tidak untuk Tahap 0. Perlu diputuskan sebelum M1.4.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: master Project Group adalah sumber tunggal cakupan Project Leader dan Project Engineer ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-06 · Hash chain audit log pada penulisan paralel

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.3 dan M2.2.
- **Jenis:** Sulit dibangun
- **Rujukan:** R-SEC-03, R-SEC-11, R-API-01, R-API-03, R-API-18, R-NUM-03, D-75, M1.3, M2.2.
- **Temuan:** R-SEC-03 meminta hash chain: setiap entri menyimpan hash entri sebelumnya. Dengan satu rantai untuk seluruh sistem, setiap transaksi yang menulis audit harus menunggu giliran, karena urutan entri harus pasti sebelum hash dihitung. Uji M2.2 (50 pendaftaran nomor paralel) juga menulis 50 entri audit, sehingga semuanya antre di ujung rantai. Brief juga belum menyebut: (1) apakah entri ditulis trigger database atau lapisan layanan Go (R-API-01), padahal jenis aktor dan request ID (R-API-03) hanya diketahui aplikasi; (2) isi apa yang di-hash dan dalam bentuk baku apa, karena JSONB yang dibaca ulang sebagai teks tidak menjamin bentuk yang sama dengan saat ditulis; (3) role database mana yang menjalankan migrasi dan memiliki tabel, dan role mana yang dipakai aplikasi tanpa UPDATE dan DELETE.
- **Dampak bila dibiarkan:** M1.3 mudah menghasilkan rantai yang putus karena dua transaksi membaca ujung rantai yang sama (race condition), atau penulisan menjadi lambat tanpa disadari.
- **Opsi:** (a) Satu rantai global. Entri ditulis di transaksi yang sama dengan perubahan datanya, dan ujung rantai dikunci (satu baris penanda dengan `SELECT ... FOR UPDATE`, atau advisory lock). Sederhana dan benar; penulisan audit menjadi berurutan. (b) Entri ditulis tanpa hash di transaksi, lalu job latar (River) menyegel entri secara berurutan berdasarkan ID. Rantai tertinggal beberapa detik. (c) Satu rantai per entitas atau per tabel.
- **Rekomendasi Claude Code (usulan):** (a) untuk demo localhost dengan 10 sampai 50 pengguna. Entri ditulis lapisan layanan Go dengan aktor dan request ID dari aplikasi. Hash dihitung dari serialisasi baku yang ditetapkan di kode. Dua role database: pemilik untuk migrasi, dan role aplikasi tanpa UPDATE dan DELETE pada tabel audit. Uji paralel M2.2 sekaligus membuktikan rantai tetap utuh.
- **Asumsi sementara di kode:** belum ada; temuan ini tidak menyentuh Tahap 0.
- **Menghambat milestone?** Tidak untuk Tahap 0. Perlu diputuskan sebelum M1.3.
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: satu rantai global dengan ujung rantai dikunci, entri ditulis lapisan layanan Go, dua role database ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-07 · Cara membuat akun admin pertama belum ada

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.1 dan M1.2.
- **Jenis:** Celah
- **Rujukan:** D-72, D-73, D-74, R-SEC-13, UC-02 (hak A-1), M1.1, M1.2.
- **Temuan:** Hanya admin yang membuat pengguna dan memberi peran (UC-02, A-1), dan akun admin harus ditautkan ke pemegangnya (D-74). Belum ada aturan bagaimana akun admin pertama dibuat di database kosong, dan bagaimana menautkannya ke pemegang yang belum punya data atau akun kerja.
- **Dampak bila dibiarkan:** M1.1 tidak bisa diuji dari database kosong tanpa jalur awal. Jalur yang dibuat seadanya, misalnya password bawaan di kode, menjadi celah keamanan.
- **Opsi:** (a) Perintah CLI satu kali di binary Go yang hanya bisa jalan bila belum ada akun admin. Perintah ini menanyakan nama pemegang dan password, lalu mencatatnya di audit log dengan aktor sistem. (b) Akun admin pertama dibuat dari variabel di `.env` saat aplikasi pertama kali jalan. (c) Halaman setup di web yang terbuka sampai admin pertama dibuat.
- **Rekomendasi Claude Code (usulan):** (a). Tidak ada password di file dan tidak ada halaman yang terbuka tanpa login.
- **Asumsi sementara di kode:** bila belum diputuskan saat M1.1, pakai (a) dan tandai `// ASUMSI E-07`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: perintah CLI satu kali untuk akun admin pertama ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-08 · Aturan login dan sesi belum ditetapkan

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.1.
- **Jenis:** Celah
- **Rujukan:** R-SEC-01, UC-01, M1.1, brief bagian 16 (cookie session with Origin check), Q-02.
- **Temuan:** M1.1 membangun login lokal dengan argon2id dan sesi. Brief belum menetapkan aturan password (panjang minimal, daftar password yang dilarang), lama sesi dan batas tidak aktif, penanganan gagal login berulang, siapa yang mereset password yang lupa, parameter argon2id, dan apa yang dipakai sebagai nama login (misalnya nomor karyawan).
- **Dampak bila dibiarkan:** nilai-nilai ini akan ditentukan sendiri di kode, padahal CLAUDE.md melarang mengarang. Spesifikasi UC-01 untuk tim IT juga butuh aturan ini.
- **Opsi:** (a) Pakai standar terbuka sebagai dasar (OWASP ASVS, OWASP Password Storage Cheat Sheet, NIST SP 800-63B). Nilai tepatnya dikutip dari dokumen standar saat M1.1, lalu dipilih pemilik. Reset password oleh admin, dengan wajib ganti password saat login berikutnya. (b) Tunggu kebijakan IT perusahaan (Q-02). Kurang cocok, karena versi ini demo pribadi (D-36).
- **Rekomendasi Claude Code (usulan):** (a). Dicatat sebagai keputusan di catatan M1.1.
- **Asumsi sementara di kode:** bila belum diputuskan saat M1.1, pakai nilai dari standar di opsi (a) dengan sumbernya, dan tandai `// ASUMSI E-08`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: aturan dari standar terbuka, nilai tepat dipilih saat M1.1 ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-09 · Aturan kode customer belum lengkap

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M0.5 dan M1.4.
- **Jenis:** Celah
- **Rujukan:** R-NUM-01, R-NUM-02, R-POR-01, D-54, UC-03, R-DOC-10, brief bagian 1 (contoh KT dan KO), M0.5, M1.4, M2.1.
- **Temuan:** M0.5 membuat entitas customer pertama. Brief hanya memberi contoh kode (KT, KO) dan aturan bahwa kode di nomor harus sama dengan kode di master (R-POR-01, usulan). Belum ada aturan tentang: huruf dan angka yang boleh dipakai, panjang, apakah kode boleh diubah setelah ada nomor terdaftar, dan apakah customer boleh dihapus atau hanya dinonaktifkan. Nomor disimpan sebagai teks dan tidak dihitung ulang (R-NUM-02), jadi mengganti kode customer membuat nomor lama tidak cocok lagi menurut R-POR-01. Customer juga wajib masuk satu Project Group (D-54), padahal master Project Group baru dibuat di M1.4. Contoh SC-ESD-K001 (R-DOC-10) memakai ESD di posisi kode customer, padahal ESD bukan customer.
- **Dampak bila dibiarkan:** validasi dan batasan unik di M0.5 harus ditebak. Migrasi M1.4 harus menambah kolom wajib ke tabel yang sudah berisi data.
- **Opsi:** (a) Kode unik, tidak boleh kosong, pola dan panjangnya diambil dari register nomor yang ada, dan tidak bisa diubah setelah ada nomor terdaftar. Customer tidak pernah dihapus, hanya dinonaktifkan. Kolom Project Group ditambah di M1.4. (b) Kode boleh diubah. Nomor lama tetap memakai kode lama dan dilaporkan sebagai pengecualian R-POR-01.
- **Rekomendasi Claude Code (usulan):** (a). Pola kode dicek pemilik dari register nyata, termasuk kasus seperti ESD.
- **Asumsi sementara di kode:** bila belum diputuskan saat M0.5, kode hanya huruf besar dan angka, unik, tidak kosong, dan belum bisa diubah. Ditandai `// ASUMSI E-09`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi ("saya setuju dengan semua saran anda", 27 Oktober 2026). Catatan: WI SW-ZZ-000 rev 09 menunjukkan kode customer pernah diganti, jadi bagian "kode tidak bisa diubah" ditinjau ulang di E-11. Bagian itu digantikan keputusan E-11 (28 Oktober 2026): kode customer bisa diubah lewat master data, dengan riwayat kode.

### E-10 · Siapa yang mengelola group model dan model

- **Tanggal dan milestone:** 27 Oktober 2026, evaluasi awal sesi 1. Berdampak ke M1.2 (isi awal peran) dan M1.4.
- **Jenis:** Kontradiksi
- **Rujukan:** Q-46, UC-03, U3-peta-pengguna hak A-2 dan E-6, D-39, D-42, D-64, D-72, M1.4.
- **Temuan:** UC-03 dan hak A-2 menempatkan pengelolaan customer, Project Group, group model, model, dan part number di akun Admin. Asumsi kerja Q-46 menyebut group model dikelola penyusun template ("Template authors maintain groups"), yaitu PPAP Engineer (D-64). Akun admin dipegang PPAP Staff (D-72), jadi keduanya orang yang berbeda. Selain itu, menambah model dan part number terjadi setiap ada model baru. Bila hanya akun admin yang boleh, PPAP Staff harus berpindah akun setiap kali, dan penyusun lain harus meminta kepadanya.
- **Dampak bila dibiarkan:** isi awal peran di M1.2 dan hak di layar M1.4 berbeda tergantung dokumen mana yang diikuti.
- **Opsi:** (a) Ikuti U3: semua master di M1.4 hanya untuk Admin, dan asumsi Q-46 disesuaikan. (b) Group model dikelola Pengelola template (hak E-6 diperluas), master lain tetap Admin. (c) Model dan part number dikelola Penyusun lewat hak baru, sedangkan customer dan Project Group tetap Admin.
- **Rekomendasi Claude Code (usulan):** (a) untuk M1.4, karena U3 adalah baseline yang disetujui. Opsi (c) layak dipertimbangkan bila alur kerja nyata sering menambah model baru.
- **Asumsi sementara di kode:** bila belum diputuskan saat M1.4, pakai (a) dan tandai `// ASUMSI E-10`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 27 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: semua master di M1.4 hanya untuk Admin ("saya setuju dengan semua saran anda", 27 Oktober 2026).

### E-11 · Kode customer pernah diganti (WI SW-ZZ-000)

- **Tanggal dan milestone:** 27 Oktober 2026, M0.1. Ditemukan saat membaca WI yang diberikan pemilik. Berdampak ke M1.4, M2.1, dan M2.5.
- **Jenis:** Kontradiksi
- **Rujukan:** E-09 (keputusan opsi a), R-NUM-02, R-NUM-03, R-NUM-06, R-POR-01, D-54, M1.4, M2.1, M2.5. Sumber: WI SW-ZZ-000 rev 09 "Sistem Penomoran & Penulisan Dokumen Level 3 & 4". Dokumen ini diberikan pemilik pada 27 Oktober 2026 dan tidak disimpan di repo.
- **Temuan:** Riwayat revisi WI mencatat dua kali penggantian kode customer: rev 04 (HL menjadi MS) dan rev 09 (BA menjadi RB). Keputusan E-09 opsi (a) menyatakan kode customer tidak bisa diubah setelah ada nomor terdaftar, jadi aturan sistem lebih ketat daripada praktik perusahaan. WI tidak menyebut apa yang terjadi pada nomor dokumen yang sudah terbit dengan kode lama.
- **Dampak bila dibiarkan:** bila kode dikunci, penggantian seperti rev 09 tidak bisa dicatat di sistem. Bila kode boleh diubah tanpa aturan, R-POR-01 (kode di nomor sama dengan kode master) gagal untuk semua nomor lama. Penghitung per customer dan jenis dokumen (R-NUM-03) juga butuh aturan: melanjutkan nomor terakhir kode lama, atau mulai dari awal.
- **Opsi:** (a) Kode customer disimpan beserta riwayatnya (kode lama tetap tercatat dengan tanggal berlaku). Nomor lama tetap teks dengan kode lama (R-NUM-02) dan dianggap cocok bila kode itu pernah berlaku untuk customer tersebut. Nomor baru memakai kode yang sedang berlaku. (b) Admin bisa mengganti kode tanpa riwayat, dan nomor lama dilaporkan sebagai pengecualian R-POR-01. (c) Penggantian kode dianggap customer baru, dan customer lama dinonaktifkan.
- **Rekomendasi Claude Code (usulan):** (a). Cocok dengan R-NUM-02 (nomor tidak pernah dihitung ulang) dan dengan praktik di WI. Pemilik perlu menjawab dua hal: setelah kode diganti, apakah nomor berjalan melanjutkan atau mulai dari awal, dan apakah dokumen lama mendapat nomor baru saat revisi berikutnya.
- **Asumsi sementara di kode:** untuk M0.5, kode customer disimpan di tabel customer dan belum bisa diubah lewat API. Ditandai `// ASUMSI E-11`.
- **Menghambat milestone?** Tidak untuk M0.1 dan M0.5. Perlu diputuskan sebelum M1.4.
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Kode customer dibuat sebagai master data yang bisa diubah nanti ("E-11 buatkan master data nya, sehingga kode customer bisa diubah nantinya", 28 Oktober 2026). Ini mengikuti opsi (a): kode lama disimpan sebagai riwayat dengan tanggal berlaku, nomor lama tetap teks dengan kode lama (R-NUM-02), dan nomor baru memakai kode yang berlaku. Urutan pembangunan: M0.5 menyimpan kode customer, lalu mengubah kode beserta riwayatnya dibangun di M1.4. Tambahan 28 Oktober 2026: setelah kode diganti, nomor berjalan untuk kode baru mulai lagi dari awal, dan dokumen model yang masih aktif dipindah secara manual ke nomor baru. Ini mengikuti pengalaman perusahaan saat struktur penomoran pernah berubah (pernyataan pemilik). Rincian pemindahan ke nomor baru dicatat di E-16.

### E-12 · Struktur nomor di WI SW-ZZ-000 lebih rinci daripada pola di brief

- **Tanggal dan milestone:** 27 Oktober 2026, M0.1. Ditemukan saat membaca WI yang diberikan pemilik. Berdampak ke M2.1, M2.4, dan M2.5.
- **Jenis:** Celah
- **Rujukan:** R-NUM-01, R-NUM-02, R-NUM-05, R-NUM-06, R-REV-01, R-DOC-10, R-DOC-11, Q-05, Q-19, brief bagian 1 (pola SH-KT-nnn), U1 (Awalan), M2.1, M2.4, M2.5. Sumber: WI SW-ZZ-000 rev 09, halaman 2 dan 3.
- **Temuan:** WI menetapkan struktur nomor A-B-CC-DDD-EE untuk dokumen Level 3.
  1. A adalah kode pabrik (1 huruf, S), dan B adalah jenis dokumen (1 huruf: C untuk Control Plan, F untuk FMEA, H untuk Process Flow Diagram, ditambah banyak jenis lain seperti W, I, dan R). Brief memperlakukan SH, SF, dan SC sebagai satu awalan per jenis dokumen.
  2. CC adalah kode customer 2 huruf (sesuai kode raw material), ZZ untuk General/Common, dan HSE (3 huruf) untuk Health, Safety, and Environment. Jadi panjang kode tidak selalu 2.
  3. DDD adalah nomor berjalan 000 sampai 999. Di atas 999 ada kode tambahan dua karakter (00 untuk ribuan, A1 dan seterusnya untuk puluhan ribu, B1 dan seterusnya untuk ratusan ribu), dan rev 07 menambah aturan nomor 1000 sampai 9999 menjadi 4 karakter. Cara penulisannya belum jelas: struktur menyebut DD-DDD (6 karakter), sedangkan tabel menulis 00-XXXX. Brief hanya menulis nnn.
  4. EE adalah nomor revisi 2 digit (00 sampai 99), ditulis sebagai bagian nomor (misalnya SW-ZZ-001-00) dan di nama file softcopy. Ini cocok dengan R-REV-01 (revisi berupa angka dengan format tampilan -00), tapi batas 99 belum tercatat di brief.
  5. Contoh SC-ESD-K001 di R-DOC-10 tidak cocok dengan struktur ini: ESD bukan kode customer, dan K001 bukan pola DDD. Pola K001 mirip penomoran dokumen Level 4 di WI (D = K untuk Karawang, lalu nomor berjalan).
  6. Daftar jenis dokumen B (W untuk Work Instruction, I untuk Inspection Standard, dan lainnya) bisa menjadi isi awal master dokumen terkait (R-DOC-11).
- **Dampak bila dibiarkan:** token pola nomor di M2.1 dan test "SC-KT-001" bisa terlalu sederhana. Nomor di atas 999 dan kode HSE tidak tertangani, dan impor nomor lama (R-NUM-06) bisa salah membaca nomor.
- **Opsi:** (a) Pola nomor di M2.1 disusun dari token terpisah: kode pabrik, jenis dokumen, kode customer (panjang bervariasi), nomor berjalan dengan aturan di atas 999, dan revisi. Isi awalnya diambil dari WI SW-ZZ-000. (b) Tetap memakai pola sederhana {awalan}-{customer}-{nnn} sesuai brief. Aturan di atas 999 dan kode HSE ditunda, karena nomor PFD, PFMEA, dan CP per customer mungkin belum mendekati 999.
- **Rekomendasi Claude Code (usulan):** (a) untuk struktur token. Aturan di atas 999 baru dibangun setelah pemilik memastikan cara penulisannya dan apakah ada customer yang mendekati 999. Pemilik juga perlu menjelaskan contoh SC-ESD-K001.
- **Asumsi sementara di kode:** untuk M2.1, nomor berjalan mulai dari 001, huruf A sampai Z dipakai berurutan, dan setelah Z999 sistem menolak dengan pesan yang jelas. Dikonfirmasi di rencana M2.1. Bila belum dikonfirmasi, ditandai `// ASUMSI E-12`.
- **Menghambat milestone?** Tidak untuk Tahap 0 dan 1. Perlu diputuskan sebelum M2.1.
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Pakai aturan WI SW-ZZ-000, dan aturannya disimpan sebagai master data ("E-12 gunakan aturan sesuai dengan yang ada di WI, dan cantumkan juga aturannya di master data", 28 Oktober 2026). Ini opsi (a). Isi master data: kode pabrik (S); jenis dokumen (C, F, H, dan jenis lain di WI); kode customer 2 huruf beserta kode khusus ZZ dan HSE; format nomor berjalan; dan jumlah digit revisi (2). Dibangun di M1.4 (kode customer), M1.6 (kode pabrik dan jenis dokumen), dan M2.1 (struktur nomor). Dokumen referensi set TR-021 (SH, SF, SC) sesuai dengan struktur ini: nomor di cover ditulis tanpa revisi, revisi ada di tabel riwayat, dan nama file menyertakan revisi. Dua rincian dijawab pemilik pada 28 Oktober 2026: (1) Nomor di atas 999 memakai huruf dan 3 digit: A001 sampai A999, lalu B001, dan seterusnya. Aturan ini berlaku "untuk saat ini" dan berbeda dari tabel WI SW-ZZ-000 rev 09 (00-XXXX, A1-XXXX); pernyataan pemilik yang dipakai. Karena bisa berubah, format nomor berjalan disimpan sebagai master data. (2) SC-ESD-K001 diabaikan karena bukan control plan proses. Sistem tidak menomori dokumen seperti itu. Bila dirujuk dari CP, dokumen itu dicatat apa adanya sebagai dokumen terkait (R-DOC-10, R-DOC-11).

### E-13 · Versi terbaru berbeda dari brief bagian 7

- **Tanggal dan milestone:** 27 Oktober 2026, M0.1.
- **Jenis:** Kontradiksi
- **Rujukan:** brief bagian 7, R-VER-01, R-VER-04, R-VER-05, Q-17, M0.1, `docs/versions.md`.
- **Temuan:** Saat dicek 27 Oktober 2026, empat versi lebih baru daripada brief. Semuanya rilis stabil (R-VER-01):
  - Svelte 5.57.3 (2026-10-26; brief 5.57.2);
  - oapi-codegen v2.8.1 (2026-10-24; brief v2.8.0);
  - River v0.50.0 (2026-10-27; brief v0.49.0);
  - Gotenberg v8.39.0 (2026-10-26; brief v8.38.1).

  Svelte dan oapi-codegen adalah patch di line yang sama, jadi R-VER-05 meminta versi terbaru. River dan Gotenberg adalah minor baru.
- **Dampak bila dibiarkan:** kecil. Versi patch di brief cepat tertinggal karena ada rilis baru hampir setiap minggu, sehingga brief dan `docs/versions.md` mudah tidak sejalan.
- **Opsi:** (a) Perbarui brief bagian 7 dengan versi terbaru sekarang. (b) Versi dikunci saat paketnya pertama kali dipasang dan dicek ulang di milestone itu (M0.2 untuk modul Go, M0.4 untuk paket frontend, M0.5 untuk generator OpenAPI, S5 untuk Gotenberg), lalu dicatat di `docs/versions.md`. Brief bagian 7 cukup menyebut line-nya (misalnya Go 1.27, PostgreSQL 18, SvelteKit 3), dan versi patch tepatnya hanya ada di `docs/versions.md`.
- **Rekomendasi Claude Code (usulan):** (b). Lebih sedikit dokumen yang harus disinkronkan, dan `docs/versions.md` sudah mencatat tanggal dan sumbernya.
- **Asumsi sementara di kode:** sampai diputuskan, kolom "Versi yang dipakai" di `docs/versions.md` mengikuti brief, dan versi terbaru ditulis di kolom Catatan.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Pakai versi stabil atau LTS terbaru ("E-13 gunakan versi yang sudah stabil atau LTS", 28 Oktober 2026). Versi dicek ulang 28 Oktober 2026 dan tidak berubah sejak 27 Oktober. `docs/versions.md` dan CLAUDE.md bagian 4 diperbarui ke Svelte 5.57.3, oapi-codegen v2.8.1, dan Gotenberg v8.39.0. River tetap v0.49.0 sampai E-14 diputuskan, karena v0.50.0 baru berumur satu hari dan termasuk kasus 0.x di E-14. E-14 diputuskan pada hari yang sama, dengan masa tunggu 14 hari, sehingga River tetap v0.49.0 sampai v0.50.0 memenuhi syarat (10 November 2026).

### E-14 · R-VER-04 belum menjawab line tunggal dan versi 0.x

- **Tanggal dan milestone:** 27 Oktober 2026, M0.1. Berdampak ke M0.4.
- **Jenis:** Ambigu
- **Rujukan:** R-VER-04, R-VER-03, R-VER-05, D-01, D-44, RK-59, brief bagian 7 (watch item adapter-static; River "still a 0.x project"), M0.4.
- **Temuan:** (1) Saat dicek 27 Oktober 2026, `@sveltejs/adapter-static` 4.0.0 masih satu-satunya rilis di line 4. Line itu juga satu-satunya yang menerima SvelteKit 3 (peer `@sveltejs/kit ^3.0.0-next.0`). Bila R-VER-04 dibaca harfiah, adapter-static 4.0.0 tidak boleh dipakai, sehingga SPA statis SvelteKit 3 (D-01, brief bagian 7) tidak bisa dibangun di M0.4. R-VER-04 tidak menyebut tindakan bila tidak ada alternatif. (2) Pada paket 0.x (River, `@hey-api/openapi-ts`), setiap minor bisa membawa perubahan yang tidak kompatibel, dan tidak jelas apakah River 0.50 dihitung sebagai line baru menurut R-VER-04.
- **Dampak bila dibiarkan:** M0.4 terhambat bila adapter-static belum mendapat patch dan aturannya dibaca harfiah.
- **Opsi:** (a) Tambah pengecualian di R-VER-04: line dengan satu rilis boleh dipakai bila tidak ada line lain yang memenuhi R-VER-03. Pengecualian itu dicatat di `docs/versions.md` dan dicek ulang di setiap milestone yang memakainya. Untuk paket 0.x, minor dihitung sebagai line baru. (b) Tunda M0.4 sampai adapter-static 4.0.1 rilis. (c) Pakai SvelteKit 2 dengan adapter-static 3. Ini bertentangan dengan stack di brief bagian 7, dan line 2 sudah tidak mendapat rilis sejak 2.70.3.
- **Rekomendasi Claude Code (usulan):** (a), dengan rincian berikut (diperjelas 28 Oktober 2026 atas pertanyaan pemilik):
  1. Line dengan satu rilis boleh dipakai bila tidak ada line lain yang memenuhi R-VER-03, dan rilisnya sudah berumur minimal 14 hari. adapter-static 4.0.0 (rilis 1 Oktober 2026) memenuhi syarat ini. Pengecualian dicatat di `docs/versions.md`, dan begitu patch 4.0.x keluar, pindah ke patch terbaru (R-VER-05).
  2. Untuk paket 0.x, syarat "tunggu patch" diganti dengan masa tunggu yang sama (14 hari). Alasannya, proyek seperti River merilis minor tanpa patch (v0.45.0 sampai v0.50.0 semuanya .0), sehingga syarat patch tidak pernah terpenuhi. Per 28 Oktober 2026, River v0.49.0 (22 hari) memenuhi syarat, sedangkan v0.50.0 (1 hari) belum. Versi River dikunci saat pertama kali dipakai dan dicek ulang saat itu.
  3. Angka 14 hari adalah usulan dan bisa diganti pemilik.
- **Asumsi sementara di kode:** tidak ada untuk M0.1. Bila belum diputuskan saat M0.4 dan adapter-static masih 4.0.0, berhenti dan tanya pemilik.
- **Menghambat milestone?** Tidak untuk M0.1. Ya untuk M0.4 bila adapter-static masih 4.0.0.
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a) dengan masa tunggu 14 hari, sesuai rekomendasi ("saya setuju untuk E-14" dan "saya setuju dengan usulan anda", 28 Oktober 2026). Line dengan satu rilis boleh dipakai bila tidak ada line lain yang memenuhi R-VER-03 dan rilisnya sudah berumur minimal 14 hari. Paket 0.x memakai versi terbaru yang sudah berumur minimal 14 hari saat pertama kali dipasang. Akibatnya: `@sveltejs/adapter-static` 4.0.0 boleh dipakai di M0.4 sebagai pengecualian yang tercatat di `docs/versions.md`, dan River memakai versi terbaru yang memenuhi masa tunggu saat pertama kali dipakai (per 28 Oktober 2026: v0.49.0; v0.50.0 memenuhi mulai 10 November 2026).

### E-15 · Belum ada langkah desain layar (wireframe) di milestone dan paket UML

- **Tanggal dan milestone:** 28 Oktober 2026, setelah M0.1. Muncul dari pertanyaan pemilik tentang desain UI/UX. Berdampak ke Tahap 4 dan 5 serta paket UML untuk tim IT.
- **Jenis:** Celah
- **Rujukan:** brief bagian 20 (isi paket U1 sampai U14), D-33, D-38, R-UML-02, Q-14, U3 (UC-30 sampai UC-36, UC-44, UC-47), M4.2, M4.5, M5.6, M5.9.
- **Temuan:** Paket UML (U1 sampai U14) dan milestone tidak punya langkah desain layar. Layar yang paling rumit ada di Tahap 4 dan 5: editor tabel (M4.2), perakitan dua panel (M4.5), diff riwayat (M5.6), dan papan status yokoten (M5.9). Tanpa wireframe, tata letak layar baru ditentukan saat menulis kode, dan tim IT tidak mendapat gambaran layar dari paket UML. Bahasa UI juga belum diputuskan (Q-14).
- **Dampak bila dibiarkan:** layar harus dibangun ulang bila alurnya tidak cocok dengan cara kerja pengguna, dan tim IT harus menebak tata letaknya.
- **Opsi:** (a) Tambah satu langkah desain layar sebelum Tahap 4: alur pengguna dan wireframe untuk layar Tahap 4 dan 5, ditinjau pemilik, lalu disimpan sebagai bagian paket UML. (b) Desain layar dibuat per milestone, saat layarnya dibangun. (c) Tanpa wireframe.
- **Rekomendasi Claude Code (usulan):** (a). Bila memungkinkan, wireframe juga diperlihatkan ke satu atau dua calon pengguna sebelum Tahap 4.
- **Asumsi sementara di kode:** tidak ada. Layar Tahap 0 sampai 3 berupa daftar dan form sederhana.
- **Menghambat milestone?** Tidak untuk Tahap 0 sampai 3. Perlu selesai sebelum Tahap 4.
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief). Posisi di milestone dan nomor artefak UML-nya ditentukan di sesi desain.
- **Keputusan pemilik:** Opsi (a): wireframe dibuat sebelum Tahap 4 ("saya setuju dengan usulan anda", 28 Oktober 2026). Bahasa UI (Q-14) perlu diputuskan sebelum wireframe dibuat.

### E-16 · Memindah dokumen aktif ke nomor baru setelah kode customer berganti

- **Tanggal dan milestone:** 28 Oktober 2026, setelah M0.1. Muncul dari keputusan pemilik di E-11. Berdampak ke M2.1, M2.3, dan M2.5.
- **Jenis:** Celah
- **Rujukan:** E-11, R-NUM-02, R-NUM-03, R-NUM-04, R-NUM-05, R-NUM-06, R-POR-01, R-REV-01, R-TPL-10, R-CHG-01, R-PRT-07, brief bagian 4 (document_number terhubung ke satu dokumen; diagram status nomor dokumen), M2.1, M2.3, M2.5.
- **Temuan:** Pemilik menetapkan (28 Oktober 2026) bahwa setelah kode customer diganti, nomor berjalan untuk kode baru mulai lagi dari awal, dan dokumen model yang masih aktif dipindah secara manual ke nomor baru. Desain saat ini belum punya aksi untuk itu. Model domain menghubungkan satu nomor ke paling banyak satu dokumen, dokumen menunjuk satu nomor, dan diagram status nomor tidak punya status untuk nomor yang digantikan nomor lain. Belum jelas juga:
  1. Apakah revisi berlanjut di nomor baru (misalnya -06 menjadi -07), atau mulai lagi dari -00.
  2. Apa status nomor lama: Obsolete, atau status baru seperti "Diganti".
  3. Apakah pemindahan butuh revisi baru dan tanda tangan, karena cover harus dicetak ulang dengan nomor baru (perubahan administratif, R-TPL-10).
  4. Apakah pemindahan dilakukan per dokumen atau per set, supaya nomor PFD, PFMEA, dan CP satu model tetap sejajar (R-NUM-05).
  5. Bagaimana penghitung kode baru menghindari nomor ganda bila kode itu pernah dipakai sebelumnya.
- **Dampak bila dibiarkan:** M2.1 sampai M2.3 bisa membangun hubungan nomor dan dokumen yang tidak bisa menampung pindah nomor, dan riwayat dokumen putus karena nomor lama tidak bisa dicari dari dokumen yang sama.
- **Opsi:** (a) Aksi "pindah ke nomor baru" oleh dokumen kontrol, per set atau per dokumen: sistem mendaftarkan nomor baru dengan kode yang berlaku dan menautkannya ke dokumen yang sama. Nomor lama mendapat status "Diganti" beserta tautan ke nomor barunya, tetap tersimpan sebagai teks (R-NUM-02), dan tetap bisa dicari. Pemindahan dicatat sebagai perubahan administratif (R-CHG-01, R-TPL-10) yang menghasilkan revisi baru, karena cover dicetak ulang. (b) Dokumen di nomor baru dianggap dokumen baru yang mulai dari -00, dengan tautan "berasal dari" ke dokumen lama, dan dokumen lama menjadi Obsolete. (c) Pemindahan dilakukan di luar sistem, lalu nomor dan dokumen diimpor ulang (R-NUM-06, M7.1).
- **Rekomendasi Claude Code (usulan):** (a). Dokumen tetap satu entitas dengan riwayat utuh; yang berganti hanya nomornya, dan jejaknya tercatat. Ini tidak bertentangan dengan R-NUM-02 dan R-NUM-06, karena nomor lama tidak diubah atau dihapus dan pemindahannya manual. Revisi berlanjut, bukan mulai dari -00, karena isinya sama. Pemilik perlu mengonfirmasi butir 1 sampai 4.
- **Asumsi sementara di kode:** untuk M2.1, penghitung dikunci per teks kode customer dan jenis dokumen. Dengan begitu kode baru mulai dari 001 secara alami, dan kode yang pernah dipakai tidak menghasilkan nomor ganda. Bila belum diputuskan, ditandai `// ASUMSI E-16`.
- **Menghambat milestone?** Tidak untuk Tahap 0 dan 1. Perlu diputuskan sebelum M2.1.
- **Status:** Diputuskan pemilik 28 Oktober 2026 (nomor D menunggu brief)
- **Keputusan pemilik:** Opsi (a), dengan jawaban pemilik atas empat butir (28 Oktober 2026): (1) nomor revisi berlanjut di nomor baru; (2) nomor lama menjadi tidak aktif dan tidak bisa diteruskan, artinya tidak ada revisi baru di nomor lama; (3) pemindahan butuh revisi baru dan tanda tangan baru; (4) pemindahan dilakukan per dokumen. Asumsi untuk M2.3, dikonfirmasi saat itu: status "tidak aktif" memakai status Obsolete yang sudah ada (R-NUM-04), ditambah tautan ke nomor baru, supaya tidak muncul istilah baru. Butir 5 (penghitung per teks kode customer) tetap asumsi di M2.1.

### E-17 · Kedudukan /health terhadap /api/v1 dan kontrak OpenAPI

- **Tanggal dan milestone:** 28 Oktober 2026, M0.2. Berdampak ke M0.4 dan M0.5.
- **Jenis:** Ambigu
- **Rujukan:** M0.2, M0.4, M0.5, R-API-02 (DECIDED, D-61), R-API-13, R-API-15, brief bagian 16 (UI API `/api/v1` dengan kontrak OpenAPI, khusus untuk SPA), D-38, M8.3.
- **Temuan:** Milestone menaruh endpoint pertama di `/health` (M0.2), lalu layar SPA memanggilnya di M0.4. Brief menyatakan API untuk layar ada di `/api/v1` dan kontraknya ditulis lebih dulu (R-API-02). Akibatnya `/health` dibuat sebelum ada kontrak (M0.5), dan tidak jelas apakah `/health` termasuk API untuk layar (di bawah `/api/v1` dan tercatat di kontrak OpenAPI) atau endpoint operasional di luar API, misalnya untuk healthcheck Docker di M8.3. Selain itu, jawaban 404 dan 405 bawaan `ServeMux` masih berupa teks biasa, belum memakai format error R-API-13. Format error itu memang dijadwalkan di M0.5.
- **Dampak bila dibiarkan:** di M0.5 bisa muncul dua gaya endpoint: satu ditulis tangan di luar kontrak, satu dihasilkan dari kontrak. Layar M0.4 juga bisa memanggil endpoint yang tidak terdokumentasi.
- **Opsi:** (a) `/health` tetap di root sebagai endpoint operasional, dan di M0.5 ikut dicatat di file OpenAPI untuk layar, sehingga panggilan dari layar tetap terdokumentasi. (b) `/health` tetap di root dan di luar kontrak. Mulai M0.5 layar hanya memanggil endpoint yang ada di kontrak, dan panggilan `/health` di M0.4 dianggap latihan. (c) `/health` dipindah ke `/api/v1/health`, di dalam kontrak.
- **Rekomendasi Claude Code (usulan):** (a). Path di milestone tetap, healthcheck Docker nanti bisa memakai path yang sama, dan semua endpoint yang dipanggil layar tetap tercatat di kontrak (R-API-02).
- **Asumsi sementara di kode:** sampai M0.5, `/health` ada di root, di luar `/api/v1`, dan belum masuk kontrak, sesuai milestone M0.2. Ditandai `// ASUMSI E-17` di `internal/server/server.go`.
- **Menghambat milestone?** Tidak. Perlu diputuskan sebelum M0.5.
- **Status:** Terbuka
- **Keputusan pemilik:**

## Bahan dari dokumen referensi (untuk brief bagian 21)

Bagian ini bukan temuan E. Isinya pengamatan dari dokumen referensi yang diberikan pemilik, untuk dibawa ke sesi desain dan dicatat sebagai item F di brief bagian 21 (usulan Claude Code, disetujui pemilik 28 Oktober 2026). Dokumennya tidak disimpan di repo, dan nama orang tidak ditulis.

**Sumber:** set TR-021 dari customer TR, diberikan pemilik 28 Oktober 2026: SH-TR-021 revisi 02 (PFD), SF-TR-021 revisi 03 (PFMEA, di file bernama SF-TR-021-02), dan SC-TR-021 revisi 06 (CP). Set ini model saudara dari set TR-020 di brief bagian 21.

| No | Pengamatan | Hubungan dengan brief |
|---|---|---|
| 1 | Pola yang sama dengan set TR-020 muncul lagi: nomor 021 dipakai bersama oleh ketiga dokumen; revisi berjalan per dokumen (PFD -02, PFMEA -03, CP -06); nama model ditulis berbeda di PFD dan di PFMEA serta CP; nomor baris riwayat revisi CP tidak urut; dan satu perubahan (AOI, rujukan SW-ZZ-116) muncul di PFMEA dan CP dengan nomor revisi berbeda. | Menguatkan F-03, F-05, F-06, dan F-07 dengan set kedua. |
| 2 | Nama file PFMEA tertinggal satu revisi dari isinya: file bernama SF-TR-021-02 berisi revisi 03. | Varian baru dari F-04. Usulan: nama file cetak dibuat sistem dari nomor dan revisi (R-PRT-01, E-12). |
| 3 | Isian header "Process or Line" berbeda di dalam satu set: isian di PFD berbeda dengan isian di PFMEA dan CP. | Varian baru dari F-05. Usulan: field header satu set diambil dari master data, bukan diketik per dokumen (R-DOC-08). |
| 4 | Nomor di cover ditulis tanpa revisi (contoh: "PFD No : SH-TR-021"), revisi ada di tabel riwayat, dan nama file menyertakan revisi. | Sesuai WI SW-ZZ-000 dan R-REV-01. Dipakai di E-12. |
| 5 | Riwayat revisi CP merujuk nomor WI (SW-ZZ-006, SW-ZZ-023, SW-ZZ-053, SW-ZZ-116, dan lainnya) dan dokumen QES-SSC-K049, yang polanya (K dan 3 digit) mirip SC-ESD-K001. | R-DOC-11 (master dokumen terkait). Dokumen seperti ini dicatat apa adanya dan tidak dinomori sistem (E-12). |
