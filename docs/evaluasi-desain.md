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

**Status:** Terbuka, Diputuskan (D-xx), Ditolak, Ditunda, atau Diperbaiki. Diperbaiki dipakai untuk temuan yang hanya berupa teks dokumen yang salah atau usang: teksnya dibetulkan langsung, tanpa keputusan desain dan tanpa item D.

## Ringkasan

| ID | Judul | Jenis | Milestone | Menghambat? | Status |
|---|---|---|---|---|---|
| E-01 | Daftar peran M1.2 berbeda dengan peta pengguna U3 | Kontradiksi | M1.2 | Ya, untuk M1.2 | Diputuskan (D-78) |
| E-02 | Akun dan orang belum dibedakan di model data | Celah | M1.1, M1.2, M1.8 | Ya, untuk M1.1 | Diputuskan (D-79) |
| E-03 | Apakah Tahap 1 menunggu U5 dan U6 | Ambigu | Tahap 1 | Ya, untuk mulai Tahap 1 | Diputuskan (D-80) |
| E-04 | Cakupan Project Group dibutuhkan sebelum M8.1 | Kontradiksi | M1.2, M5.3, M8.1 | Tidak (perlu sebelum M5.3) | Diputuskan (D-81) |
| E-05 | Dua sumber data untuk cakupan Project Group | Ambigu | M1.2, M1.4, M1.8 | Tidak (perlu sebelum M1.4) | Diputuskan (D-82) |
| E-06 | Hash chain audit log pada penulisan paralel | Sulit dibangun | M1.3, M2.2 | Tidak (perlu sebelum M1.3) | Diputuskan (D-83) |
| E-07 | Cara membuat akun admin pertama belum ada | Celah | M1.1, M1.2 | Tidak | Diputuskan (D-84) |
| E-08 | Aturan login dan sesi belum ditetapkan | Celah | M1.1 | Tidak | Diputuskan (D-85) |
| E-09 | Aturan kode customer belum lengkap | Celah | M0.5, M1.4 | Tidak | Diputuskan (D-86) |
| E-10 | Siapa yang mengelola group model dan model | Kontradiksi | M1.2, M1.4 | Tidak | Diputuskan (D-87) |
| E-11 | Kode customer pernah diganti (WI SW-ZZ-000) | Kontradiksi | M1.4, M2.1, M2.5 | Tidak (perlu sebelum M1.4) | Diputuskan (D-88) |
| E-12 | Struktur nomor di WI SW-ZZ-000 lebih rinci daripada pola di brief | Celah | M2.1, M2.4, M2.5 | Tidak (perlu sebelum M2.1) | Diputuskan (D-89) |
| E-13 | Versi terbaru berbeda dari brief bagian 7 | Kontradiksi | M0.1 | Tidak | Diputuskan (D-90) |
| E-14 | R-VER-04 belum menjawab line tunggal dan versi 0.x | Ambigu | M0.1, M0.4 | Tidak untuk M0.1, ya untuk M0.4 | Diputuskan (D-91) |
| E-15 | Belum ada langkah desain layar (wireframe) di milestone dan paket UML | Celah | Tahap 4, paket UML | Tidak (perlu sebelum Tahap 4) | Diputuskan (D-92) |
| E-16 | Memindah dokumen aktif ke nomor baru setelah kode customer berganti | Celah | M2.1, M2.3, M2.5 | Tidak (perlu sebelum M2.1) | Diputuskan (D-93) |
| E-17 | Kedudukan /health terhadap /api/v1 dan kontrak OpenAPI | Ambigu | M0.2, M0.4, M0.5 | Tidak (perlu sebelum M0.5) | Diputuskan (D-94) |
| E-18 | Belum ada aturan tentang rahasia di log dan pesan error | Celah | M0.3, M0.5, M1.1, M1.3 | Tidak | Diputuskan (D-95) |
| E-19 | Glosarium U1 belum punya istilah bahasa Inggris untuk layar | Celah | M0.4, M0.5, Tahap 1 | Tidak | Diputuskan (D-98) |
| E-20 | Konvensi API R-API-13 merujuk tabel kode error yang tidak ada di repo | Celah | M0.5, M9.3 | Tidak | Diputuskan (D-99) |
| E-21 | Field dan batas master customer belum ditentukan | Celah | M0.5, M1.4 | Tidak (perlu sebelum M1.4) | Diputuskan (D-100) |
| E-22 | Tidak ada jalan keluar bila akun admin terakhir lupa password atau terkunci | Celah | M1.1, M1.2 | Tidak | Diputuskan (D-101) |
| E-23 | Daftar password yang dilarang belum memenuhi ASVS 6.2.4 tanpa layanan luar | Sulit dibangun | M1.1 | Tidak | Diputuskan (D-102) |
| E-24 | U3 belum punya hak untuk melihat master data | Celah | M1.2, M1.4 sampai M1.8 | Tidak (perlu sebelum M1.4) | Diputuskan (D-103) |
| E-25 | Hak mengurus akun sendiri tidak ada di katalog U3 | Celah | M1.1, M1.2 | Tidak | Diputuskan (D-104) |
| E-26 | Akun admin terakhir "tidak bisa dihapus", padahal hapus akun belum diatur | Ambigu | M1.2, M1.3 | Tidak | Diputuskan (D-105) |
| E-27 | Jenis aktor untuk perintah server dan orang yang belum login | Celah | M1.3, M8.1 | Tidak | Diputuskan (D-106) |
| E-28 | Peristiwa apa saja yang masuk audit log | Ambigu | M1.3, Tahap 2 sampai 7 | Tidak | Diputuskan (D-107) |
| E-29 | Kredensial pemilik database dan siapa yang menjalankan migrasi | Celah | M1.3, M8.3, M8.4 | Tidak | Diputuskan (D-108) |
| E-30 | Field dan aturan Project Group, group model, model, dan part number belum ditetapkan | Celah | M1.4, M4.1 | Tidak | Ditunda (menunggu bahan) |
| E-31 | Kode customer lama: boleh dipakai customer lain, dan tanggal berlaku penggantian | Ambigu | M1.4, M2.1 | Tidak | Diputuskan (D-109) |
| E-32 | Aturan nama dan alias master proses belum ditetapkan | Celah | M1.5, M3.2, Tahap 7 | Tidak | Diputuskan (D-110) |
| E-33 | Pencocokan PFD, PFMEA, dan CP: di tingkat proses atau sub-proses | Ambigu | M1.5, M3.2, Tahap 3, M4.1 | Tidak (M3.2 memakai asumsi; perlu sebelum M4.1) | Ditunda (menunggu bahan) |
| E-34 | Field wajib di kontrak tidak diperiksa oleh server hasil generate | Celah | M1.5, M8.1, M9.3 | Tidak | Diputuskan (D-111) |
| E-35 | Bentuk kebijakan approval per jenis perubahan belum ditentukan (Q-15) | Celah | M1.6, M1.8, Tahap 5 dan 6 | Tidak (perlu sebelum Tahap 5) | Diputuskan sebagian (D-112) |
| E-36 | Sumber perubahan mencampur mekanisme sistem (R-CHG-01) dan pemicu (R-CHG-06) | Ambigu | M1.6, Tahap 5 | Tidak (perlu sebelum Tahap 5) | Diputuskan (D-113) |
| E-37 | Kode khusus: bentuk, penambahan, dan tumpang tindih dengan kode customer | Celah | M1.6, M2.1 | Tidak (perlu sebelum M2.1) | Diputuskan sebagian (D-114) |
| E-38 | Jumlah pemegang setiap jabatan dan riwayatnya | Celah | M1.8, Tahap 6 | Tidak (perlu sebelum Tahap 6) | Diputuskan (D-115) |
| E-39 | Customer tanpa rute default | Celah | M1.8, Tahap 6 | Tidak (perlu sebelum Tahap 6) | Diputuskan (D-116) |
| E-40 | Batas bentuk rute dan isian slot | Celah | M1.8, Tahap 6 | Tidak | Diputuskan (D-117) |
| E-41 | Hak bercakupan Project Group untuk operasi tanpa customer | Celah | M1.8, Tahap 2 sampai 5, M8.1 | Tidak | Diputuskan (D-118) |
| E-42 | Hak akun admin: R-SEC-13 menyebut hanya A-1 sampai A-4, U3 dan kode memberi juga L-1 sampai L-3 | Kontradiksi | M1.2 | Tidak | Diputuskan (D-120) |
| E-43 | Q-21 menyebut lima peran, bagian 1 enam, U3 delapan | Kontradiksi | M1.2 | Tidak | Diperbaiki |
| E-44 | Nomor berjalan pertama: 000 atau 001 | Ambigu | M2.1, M2.5 | Tidak | Terbuka |
| E-45 | Q-50 mengasumsikan satu QA QC Manager, kode M1.8 membolehkan beberapa | Kontradiksi | M1.8 | Tidak | Diputuskan bersama E-38 (D-115) |
| E-46 | U1 menyebut permintaan perubahan diajukan PPAP Engineer saja | Kontradiksi | UML | Tidak | Diperbaiki |
| E-47 | Nama peran di U1 bagian 7 berbeda dengan U3 | Kontradiksi | UML | Tidak | Diperbaiki |
| E-48 | Glosarium brief tidak memuat jabatan PPAP Staff | Celah | Brief | Tidak | Diperbaiki |
| E-49 | Model domain brief tertinggal dari keputusan dan kode | Celah | Brief, U5 | Tidak | Diperbaiki |
| E-50 | UC-51 berstatus MVP, padahal M5.5a masih usulan | Ambigu | UML, M5.5a | Tidak | Diperbaiki |
| E-51 | UC-57 dijadwalkan di M1.2, padahal dokumen rilis baru ada di Tahap 5 | Celah | UML | Tidak | Diperbaiki |
| E-52 | Nilai aturan login E-08 belum dikonfirmasi, dan 10 penanda ASUMSI tersisa untuk temuan yang sudah diputuskan | Celah | M1.1 | Tidak | Diputuskan (D-85) |
| E-53 | Tanggal rilis PostgreSQL 19 RC1 berbeda antara brief dan versions.md | Kontradiksi | M0.1 | Tidak | Diperbaiki |
| E-54 | Tanggal di kepala milestone tidak sesuai dengan tanggal brief v1.15 | Kontradiksi | Milestone | Tidak | Ditunda (sesi desain) |
| E-55 | Perintah memasang browser Playwright berbeda antara CLAUDE.md dan versions.md | Kontradiksi | M0.5 | Tidak | Diperbaiki |
| E-56 | E-41 menyebut M1.7 yang belum dibangun | Ambigu | M1.8 | Tidak | Diperbaiki |
| E-57 | Kolom baris UC-02 di daftar use case bergeser | Celah | UML | Tidak | Diperbaiki |
| E-58 | README UML usang (U10 dan pertanyaan terbuka) | Celah | UML | Tidak | Diperbaiki |
| E-59 | README di root repo hampir kosong | Celah | Repo | Tidak | Diperbaiki |
| E-60 | Jadwal, retensi, dan penyimpanan passphrase backup belum ditetapkan | Celah | M1.7, M8.2, M8.4 | Tidak | Terbuka |
| E-61 | Satu pola nomor untuk semua jenis dokumen dan customer | Ambigu | M2.1, M2.5 | Tidak | Terbuka |
| E-62 | Isi register dan pengingat nomor Reserved yang terlalu lama | Celah | M2.3, M4.1 | Tidak | Terbuka |
| E-63 | Kunci idempoten di UI API: cakupan, umur, dan pemakaian ulang | Celah | M2.2, M9.4 | Tidak | Terbuka |
| E-64 | Nomor yang dilewati saat daftar sebagai set | Kontradiksi | M2.4 | Tidak | Terbuka |
| E-65 | Bentuk impor nomor lama dan isi laporannya | Ambigu | M2.5, M7.1 | Tidak | Terbuka |
| E-66 | Nomor lama yang tidak bisa masuk apa adanya: bentuk lain dan nomor ganda | Celah | M2.5, M4.1, M7.1 | Tidak | Terbuka |
| E-67 | Pendaftaran nomor untuk customer nonaktif | Celah | M2.1 | Tidak | Terbuka |
| E-68 | Perubahan status nomor yang dikerjakan tangan di Tahap 2, dan alasannya | Ambigu | M2.3, M4.1, M6.7 | Tidak | Terbuka |
| E-69 | Urutan kunci transaksi penomoran: deadlock dengan ganti kode customer | Sulit dibangun | M2.1, M2.2, M2.4 | Tidak (sudah diperbaiki di kode) | Terbuka |
| E-70 | Isi awal format template dan aturan field yang belum ditetapkan | Celah | M3.1, M4.1 | Tidak (perlu sebelum M4.1) | Diputuskan sebagian (D-124) |
| E-71 | Aturan sorot PFMEA: arti "severity 8" dan nilai mana yang disorot | Ambigu | M3.1 | Tidak | Terbuka |
| E-72 | Satuan process template: satu per proses, atau satu pustaka untuk semua proses | Ambigu | M3.2, M3.3, M3.4, M5.8 | Tidak (perlu sebelum M3.3) | Diputuskan (D-129) |
| E-73 | Isi poin process template dan format template yang diikutinya | Celah | M3.2, M4.5, M5.8, M7.4 | Tidak (perlu sebelum M4.5) | Terbuka |
| E-74 | Kode customer tiga huruf di register (AJI, AVI) bertentangan dengan D-100 | Kontradiksi | M1.4, M2.1, M2.5 | Tidak (perlu sebelum register asli diimpor) | Diputuskan (D-125) |
| E-75 | Register terpisah PL1 dan PL3: plant, line, atau Project Leader (Q-19) | Ambigu | M2.1, M2.5 | Tidak (perlu sebelum register asli diimpor) | Diputuskan (D-126) |
| E-76 | Proses baru diminta saat menyusun, dan urutan standar proses di pustaka | Celah | M1.5, M3.3, M4.5, M4.6 | Tidak | Terbuka |
| E-77 | Doc ref (Relevant Document) berbeda per model, padahal poinnya umum | Celah | M3.3, M4.1, M4.5, R-DOC-11 | Tidak (perlu sebelum M4.5) | Diputuskan (D-130) |
| E-78 | Cakupan poin yang salah pilih: diubah, atau poin disembunyikan lalu dibuat ulang | Celah | M3.3, M3.4 | Tidak | Diputuskan (D-131) |
| E-79 | Daftar password umum belum mencakup password 8 sampai 14 karakter dari sumber terverifikasi | Celah | M1.1 (D-132) | Tidak | Terbuka |
| E-80 | Prompt form set mengandaikan editor, wizard, yokoten, dan ekspor yang belum dibangun | Kontradiksi | Tahap 3 sampai 6 | Ya, untuk mulai form set | Diputuskan (D-140) |
| E-81 | Isi poin PFMEA di process template untuk form AIAG-VDA | Celah | M3.2, M4.5 | Tidak (perlu sebelum M4.5) | Terbuka |
| E-82 | Siapa yang menyetujui form set baru dan perubahan tabel AP | Celah | Form set langkah A dan D | Ya, untuk langkah A dan D | Diputuskan (D-139) |
| E-83 | Perpindahan dokumen set 1 ke set 2: satu per satu atau massal | Celah | Tahap 5 | Tidak | Terbuka |
| E-84 | Genba: penulis catatan selain tim PPAP, dan lama penyimpanan foto | Celah | Tahap 6 | Tidak | Terbuka |
| E-85 | Alias proses: siapa yang mengelola | Ambigu | M1.5, pencarian per proses | Tidak | Terbuka |
| E-86 | Review berkala 3 tahun: rincian yang belum diputuskan | Celah | Tahap 5, 6, 7 | Tidak (perlu sebelum Tahap 6) | Terbuka |
| E-87 | Menerbitkan versi template: perlu orang kedua atau tidak | Celah | M3.4, form set | Tidak | Terbuka |

## Keputusan 4 November 2026

Pemilik menjawab tinjauan desain UI (`docs/catatan/Tinjauan-desain-UI-UML-0.12.md`). Kutipan:

> "1. tambahan untuk simbol revision yang tercetak di control plan / PFD / PFMEA. jika revision naik dari revisi 1 ke revisi 2, maka simbol revisi 1 nya di hilangkan. namun apakah memungkinkan untuk tetap menyimpan version control nya? artinya kita tetap punya tempat untuk cek semua perubahan
> 2. namun untuk di print tidak boleh di sembunyikan. untuk template yang saya kirimkan memang memuat flow process nya, dan hal itu yang membuat edit PCP jadi sangat lama dan menjadi tergate utama kita adalah mempermudah process penyesuaian flow process, terutama di simbol connector karna perpindahan page
> 3. kenapa nama process di sembunyikan? dari mana user tau nama process nya ketika melihat / membaca dokumen PFMEA
> E-74 untuk saat ini ikuti WI hanya mengizinkan dua huruf
> E-75 PL = project leader
> E-70 ya, itu sudah sesuai file excel asli"

Yang dicatat di brief v1.21:

| Keputusan | Isi | Dampak ke kode |
|---|---|---|
| D-121 | Hanya tanda revisi yang berlaku yang tercetak. Tanda revisi lama hilang dari cetakan, tetapi semua perubahan tetap bisa dilihat: revisi rilis dibekukan (R-REV-02), dan diff serta riwayat menunjukkan perubahan antara dua revisi mana pun (R-REV-03). R-REV-09 menjadi DECIDED. | Tidak ada sekarang. Tanda dihitung dari diff saat cetak (Tahap 5 dan 6) |
| D-122 | Kolom Process Flow di CP selalu tercetak, terisi dari langkah PFD. Memudahkan penyesuaian alur, terutama konektor di perpindahan halaman, menjadi tujuan utama (G11). | Migrasi 00014: Process Flow tidak lagi disembunyikan, tetapi menjadi kolom terisi otomatis `process_flow` |
| D-123 | Nama proses dan nomor operasi selalu tampil dan tercetak di PFD, PFMEA, dan CP, terisi dari baris, tidak diketik. | Migrasi 00014: jenis kolom `process` dan `operation_number`; Process Step / Function di PFMEA tampil lagi |
| D-124 | Susunan kolom di artboard "current form layout" sama dengan file Excel asli; isi awal mengikutinya (E-70). | Migrasi 00014: kolom isi awal PFD, PFMEA, dan CP |
| D-125 | Kode customer tetap dua huruf sesuai WI (E-74). | Tidak ada; D-100 tetap |
| D-126 | Register PL1 dan PL3 adalah register Project Leader (E-75). | Tidak ada; penomoran tetap per kode customer dan jenis dokumen |

Temuan lain dari tinjauan (E-33, E-72, E-73, E-76) dan usulan UX-01 sampai UX-07 belum dijawab, jadi tetap Terbuka.

## Keputusan 30 Oktober 2026

Pemilik menjawab lembar evaluasi Tahap 1 (`docs/catatan/Tahap-1-evaluasi.md`) dan laporan pemeriksaan dokumen dengan pernyataan ini:

> "untuk saat ini saya setujui semua saran anda asalkan tidak ada pengaruhnya ke tujuan utama website yaitu mempermudah revisi yokoten. anda juga saya izinkan untuk merubah brief sesuai dengan aktual code. jika tidak ada kendala untuk lanjut ke M2, silahkan selesaikan hingga M2" (30 Oktober 2026)

Cara membacanya:
1. **Yang disetujui** adalah saran yang sudah pernah ditulis Claude Code: rekomendasi setiap temuan E-19 sampai E-41, rekomendasi di lembar evaluasi Tahap 1 (M1.7 dibangun sekarang, UML Tahap 1 diperbarui sebelum M2.1, item D ditulis di brief), dan saran di laporan pemeriksaan dokumen (temuan E-42 sampai E-59).
2. **Yang tidak ikut disetujui** adalah pertanyaan yang belum pernah diberi saran. Pertanyaan itu tetap terbuka: butir 4 E-37, pertanyaan sisa E-35, E-44, Q-46, dan isi E-30 serta E-33 (yang menunggu bahan dari pemilik).
3. **Pemeriksaan terhadap tujuan yokoten.** Setiap saran diperiksa: apakah membuat yokoten lebih sulit? Hasilnya:
   - Membantu yokoten: E-24 (semua peran bisa melihat customer, group model, model, dan proses sebagai target), E-32 (ID proses tidak pernah ambigu saat target dipilih per proses, R-CHG-04), dan E-36 opsi (b) di Tahap 5 (perubahan yokoten yang dipicu customer claim bisa mencatat keduanya).
   - Perlu dijaga: E-41. Opsi (a) menolak hak bercakupan untuk operasi tanpa customer. Pencarian target yokoten lintas customer (M5.9) karena itu wajib memakai opsi (c), supaya Project Leader dan Project Engineer tetap bisa mencari target di customer Project Group-nya. Catatan ini ditulis di keputusan E-41.
   - Perlu dijaga: E-39. Cetak untuk tanda tangan ditolak bila customer belum punya rute default, kecuali penyusun memilih rute lain dengan alasan. Revisi yokoten ke banyak customer tidak terhenti, karena jalan keluar itu tetap ada.
   - Saran lain tidak menyentuh alur yokoten (akun, audit log, versi, backup, teks dokumen).
4. **Brief** diperbarui ke v1.16 dengan item D-78 sampai D-120 dan teks yang mengikuti kode.

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-78)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: delapan peran U3 dan katalog hak L-1 sampai A-4 sebagai data awal di M1.2, tanpa peran approver dulu ("saya setuju dengan semua saran anda", 27 Oktober 2026).
- **Pelaksanaan (M1.2, 28 Oktober 2026):** tabel `permissions`, `roles`, `role_permissions`, dan `account_roles` di `sql/migrations/00003_create_roles.sql`, diisi dari matriks U3 bagian 3. `TestRoleMatrix` di `internal/access` membandingkan isi database dengan matriks U3 yang ditulis ulang di test. Isi peran belum bisa diubah lewat layar sampai M8.1.

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-79)
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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-80)
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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-81)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: cakupan Project Group dicek di fungsi izin sejak M1.2; row-level security tetap di M8.1 ("saya setuju dengan semua saran anda", 27 Oktober 2026).
- **Pelaksanaan (M1.2, 28 Oktober 2026):** `access.Authorize` menerima `Resource{CustomerID}` dan memeriksa cakupan `own_project_group` untuk Penyusun dan Verifikator. Sumber datanya (master Project Group dan pemegang jabatan) baru ada di M1.4 dan M1.8, jadi sampai itu `inOwnProjectGroup` selalu menjawab tidak (ASUMSI E-04, pilihan paling aman). Saat ini belum ada operasi yang menyentuh satu customer, jadi aturan ini belum berpengaruh ke layar.
- **Pelaksanaan (M1.8, 28 Oktober 2026):** `inOwnProjectGroup` kini membaca pemegang jabatan: hak bercakupan Project Group berlaku untuk customer di Project Group tempat orang itu menjadi Project Leader atau Project Engineer. ASUMSI E-04 sudah dihapus dari kode. Operasi tanpa customer tidak lagi diloloskan untuk hak bercakupan (ASUMSI E-41). Dibuktikan `TestAuthorizeScope`, termasuk hak yang langsung hilang saat jabatan dilepas.

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-82)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: master Project Group adalah sumber tunggal cakupan Project Leader dan Project Engineer ("saya setuju dengan semua saran anda", 27 Oktober 2026).
- **Pelaksanaan (M1.8, 28 Oktober 2026):** pemegang jabatan disimpan di tabel `position_holders`. Project Leader dan Project Engineer dicatat per Project Group, satu orang per group (D-54), dan menjadi satu-satunya sumber cakupan. Kolom cakupan di peran hanya menyatakan "semua customer" atau "Project Group sendiri", tanpa menyebut group-nya.

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-83)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: satu rantai global dengan ujung rantai dikunci, entri ditulis lapisan layanan Go, dua role database ("saya setuju dengan semua saran anda", 27 Oktober 2026).
- **Pelaksanaan (M1.3, 28 Oktober 2026):**
  - Kunci rantai: advisory lock 11002 dipegang sampai transaksi selesai. Nomor urut dibagikan aplikasi di dalam kunci itu, bukan dari sequence, jadi lubang nomor berarti ada entri yang hilang.
  - Entri ditulis `audit.Record` sebagai langkah terakhir transaksi yang mengubah datanya.
  - Bentuk baku yang di-hash: setiap field ditulis `<panjang>:<isi>,` dengan urutan tetap, diawali `prev_hash`. Waktu dalam UTC dengan 6 angka pecahan detik. Rincian disimpan sebagai teks JSON buatan Go, bukan `jsonb`. Contoh lengkapnya di `TestCanonicalForm` (`internal/audit/hash_test.go`).
  - Dua role: pemilik (`POSTGRES_USER`) hanya untuk migrasi saat program mulai; `revioten_app` untuk semua operasi, dengan hak minimum per tabel (`audit_log` hanya `SELECT` dan `INSERT`). Ditambah trigger `audit_log_append_only` yang menolak `UPDATE`, `DELETE`, dan `TRUNCATE` dari siapa pun yang tidak mematikan trigger.
  - Uji paralel: 20 penulisan bersamaan memberi nomor 1 sampai 20 tanpa lubang (`TestRecordConcurrent`). Uji paralel penomoran M2.2 nanti ikut membuktikannya.

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-84)
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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-85)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi: aturan dari standar terbuka, nilai tepat dipilih saat M1.1 ("saya setuju dengan semua saran anda", 27 Oktober 2026).
- **Pelaksanaan M1.1 (28 Oktober 2026; nilai di bawah dikonfirmasi pemilik 30 Oktober 2026 lewat persetujuan umum, E-52, D-85; sejak itu akun admin tidak dinonaktifkan pada kegagalan ke-100, E-22):** sumber dibaca dari repo resminya di GitHub: OWASP Cheat Sheet Series (commit 9d41c05, 26 Oktober 2026), OWASP ASVS 5.0 (folder `5.0/en`, commit 9ccce0e), dan NIST SP 800-63B-4 (repo usnistgov/800-63-4, commit 4f2487b, 26 Agustus 2025).

  | Aturan | Nilai | Dasar |
  |---|---|---|
  | Hash password | argon2id m=19456 KiB, t=2, p=1, salt 16 byte, hash 32 byte | OWASP Password Storage Cheat Sheet (konfigurasi minimum) |
  | Panjang password | 15 sampai 128 karakter Unicode | NIST: minimal 15 untuk password tanpa faktor kedua; ASVS 6.2.1 dan 6.2.9 |
  | Komposisi dan ganti berkala | Tidak ada aturan komposisi, tidak ada kewajiban ganti berkala | NIST; ASVS 6.2.5 dan 6.2.10 |
  | Password yang dilarang | 329 password umum (15 karakter atau lebih) dari daftar NCSC top 100.000, ditambah nama login, nama orang, dan nama sistem | ASVS 6.2.4, 6.2.11, 6.2.12; kekurangannya di E-23 |
  | Normalisasi Unicode | Tidak dilakukan; password dibandingkan apa adanya | ASVS 6.2.8. NIST menyarankan NFC (SHOULD); tidak diikuti supaya sederhana |
  | Token sesi | 256 bit acak, disimpan sebagai SHA-256 | ASVS 7.2.3 (minimal 128 bit) |
  | Batas tidak aktif | 30 menit | OWASP Session Management: 15 sampai 30 menit untuk aplikasi berisiko rendah; NIST AAL2: paling lama 1 jam |
  | Batas mutlak sesi | 8 jam | OWASP: 4 sampai 8 jam untuk pekerja kantor; NIST AAL1: paling lama 30 hari |
  | Sesi bersamaan | Tidak dibatasi; ganti password mengakhiri semua sesi akun | ASVS 7.1.2 (harus didokumentasikan), 7.4.3 |
  | Gagal login | Setiap 10 kali gagal berturut-turut, akun dikunci 15 menit; kegagalan ke-100 menonaktifkan akun | OWASP Authentication Cheat Sheet (ambang, lama kunci); NIST: paling banyak 100 |
  | Pesan login gagal | Selalu sama, tanpa menyebut penyebabnya | OWASP Authentication Cheat Sheet |
  | Nama login | 3 sampai 64 karakter a-z, 0-9, titik, garis bawah, tanda hubung; tidak membedakan huruf besar dan kecil | Belum ada sumber; dipilih supaya mudah diketik |
  | Cookie sesi | `__Host-revioten_session`, HttpOnly, Secure, SameSite=Lax, tanpa Max-Age | ASVS 3.3.1 sampai 3.3.4 |
  | CSRF | `http.CrossOriginProtection` (Sec-Fetch-Site dan Origin) untuk POST, PUT, PATCH, DELETE | Brief bagian 16 ("cookie session with Origin check"), ASVS 3.5 |
  | `/health` | Terbuka tanpa login, karena tidak berisi data | Belum ada sumber |

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-86)
- **Keputusan pemilik:** Opsi (a) sesuai rekomendasi ("saya setuju dengan semua saran anda", 27 Oktober 2026). Catatan: WI SW-ZZ-000 rev 09 menunjukkan kode customer pernah diganti, jadi bagian "kode tidak bisa diubah" ditinjau ulang di E-11. Bagian itu digantikan keputusan E-11 (28 Oktober 2026): kode customer bisa diubah lewat master data, dengan riwayat kode.
- **Pelaksanaan (M1.4, 28 Oktober 2026):** customer tidak bisa dihapus (role aplikasi tanpa `DELETE`), hanya dinonaktifkan lewat kolom `active`. Customer nonaktif tidak mendapat group model, model, atau part number baru. Kolom `project_group_id` boleh kosong hanya untuk customer yang dibuat sebelum M1.4 (ASUMSI E-09); customer baru dan customer yang diubah wajib punya Project Group. Pola kode tetap dua huruf A sampai Z dari WI (E-12).

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
- **Status:** Diputuskan pemilik 27 Oktober 2026 (D-87)
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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-88)
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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-89)
- **Keputusan pemilik:** Pakai aturan WI SW-ZZ-000, dan aturannya disimpan sebagai master data ("E-12 gunakan aturan sesuai dengan yang ada di WI, dan cantumkan juga aturannya di master data", 28 Oktober 2026). Ini opsi (a). Isi master data: kode pabrik (S); jenis dokumen (C, F, H, dan jenis lain di WI); kode customer 2 huruf beserta kode khusus ZZ dan HSE; format nomor berjalan; dan jumlah digit revisi (2). Dibangun di M1.4 (kode customer), M1.6 (kode pabrik dan jenis dokumen), dan M2.1 (struktur nomor). Dokumen referensi set TR-021 (SH, SF, SC) sesuai dengan struktur ini: nomor di cover ditulis tanpa revisi, revisi ada di tabel riwayat, dan nama file menyertakan revisi. Dua rincian dijawab pemilik pada 28 Oktober 2026: (1) Nomor di atas 999 memakai huruf dan 3 digit: A001 sampai A999, lalu B001, dan seterusnya. Aturan ini berlaku "untuk saat ini" dan berbeda dari tabel WI SW-ZZ-000 rev 09 (00-XXXX, A1-XXXX); pernyataan pemilik yang dipakai. Karena bisa berubah, format nomor berjalan disimpan sebagai master data. (2) SC-ESD-K001 diabaikan karena bukan control plan proses. Sistem tidak menomori dokumen seperti itu. Bila dirujuk dari CP, dokumen itu dicatat apa adanya sebagai dokumen terkait (R-DOC-10, R-DOC-11).
- **Pelaksanaan (M1.4, 28 Oktober 2026):** kode customer dua huruf dengan riwayat sudah menjadi master data (E-11). Kode khusus ZZ dan HSE belum menjadi tabel master: ZZ tetap ditolak sebagai kode customer, dan HSE tidak bisa bentrok karena tiga huruf. Kode khusus disimpan sebagai master data di M1.6, bersama kode pabrik dan jenis dokumen, karena ketiganya bagian struktur nomor.
- **Pelaksanaan (M2.1, 30 Oktober 2026):** pola nomor (`{prefix}-{customer}-{seq}`), bentuk nomor berjalan (001 sampai 999, lalu A001 sampai Z999), dan jumlah digit revisi (2) kini master data di `number_settings`, diubah Admin di layar Document types. Nomor berjalan pertama 001 (ASUMSI E-44), dan pola memakai satu pola untuk semua jenis dan customer (ASUMSI E-61).
- **Pelaksanaan (M1.6, 28 Oktober 2026):** kode pabrik (S), jenis dokumen dengan hurufnya (H untuk PFD, F untuk PFMEA, C untuk CP), dan kode khusus (ZZ dan HSE) sekarang master data yang diubah Admin. Awalan nomor dihitung dari kode pabrik dan huruf, jadi tidak disimpan. Jenis dokumen lain dari WI (misalnya W dan I) bisa ditambah Admin; isinya tidak disalin dari WI, karena WI tidak disimpan di repo. Kode customer kini diperiksa terhadap daftar kode khusus di database, bukan "ZZ" yang ditulis tetap di kode (E-37). Format nomor berjalan dan jumlah digit revisi tetap dibangun di M2.1.

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
- **Rekomendasi Claude Code (usulan):** (b). Lebih sedikit dokumen yang harus disinkronkan, dan `docs/versions.md` sudah mencatat tanggal dan sumbernya. Rekomendasi (b) ikut disetujui lewat persetujuan umum 30 Oktober 2026: brief v1.16 bagian 7 hanya menyebut line, dan versi tepatnya ada di `docs/versions.md`.
- **Asumsi sementara di kode:** sampai diputuskan, kolom "Versi yang dipakai" di `docs/versions.md` mengikuti brief, dan versi terbaru ditulis di kolom Catatan.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-90)
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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-91)
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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-92). Posisi di milestone dan nomor artefak UML-nya ditentukan di sesi desain.
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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-93)
- **Keputusan pemilik:** Opsi (a), dengan jawaban pemilik atas empat butir (28 Oktober 2026): (1) nomor revisi berlanjut di nomor baru; (2) nomor lama menjadi tidak aktif dan tidak bisa diteruskan, artinya tidak ada revisi baru di nomor lama; (3) pemindahan butuh revisi baru dan tanda tangan baru; (4) pemindahan dilakukan per dokumen. Status "tidak aktif" memakai status Obsolete yang sudah ada (R-NUM-04), ditambah tautan ke nomor baru, supaya tidak muncul istilah baru. Rangkuman ini, termasuk pemakaian status Obsolete, disetujui pemilik ("E-16 setuju", 28 Oktober 2026). Butir 5 (penghitung per teks kode customer) tetap asumsi di M2.1. Asumsi itu ikut disetujui lewat persetujuan umum 30 Oktober 2026, dan dipakai di M2.1.

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
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-94)
- **Keputusan pemilik:** Opsi (a), sesuai rekomendasi ("E-17 setuju", 28 Oktober 2026): `/health` tetap di root sebagai endpoint operasional, di luar `/api/v1`, dan mulai M0.5 ikut dicatat di kontrak OpenAPI untuk layar. Komentar `// ASUMSI E-17` di `internal/server/server.go` diganti dengan keterangan keputusan ini.

### E-18 · Belum ada aturan tentang rahasia di log dan pesan error

- **Tanggal dan milestone:** 28 Oktober 2026, M0.3. Berdampak ke M0.5, M1.1, M1.3, dan tahap 9.
- **Jenis:** Celah
- **Rujukan:** G9, R-SEC-01, R-SEC-03, R-API-03, R-API-04, R-API-13, brief bagian 7 (pgx), M0.3, M0.5, M1.1, M1.3.
- **Temuan:** Saat membuat `Connect`, terlihat bahwa pesan error pgx untuk alamat database yang salah format bisa memuat alamat lengkap, termasuk password. Dokumentasi pgx v5.11.0 menyebut penyamaran password di pesan itu hanya "best effort". Brief belum punya aturan bahwa password, token, dan alamat koneksi tidak boleh muncul di log, di pesan error API (R-API-13), atau di audit log (R-SEC-03). Aturan seperti ini dibutuhkan kode mulai sekarang: koneksi database (M0.3), format error API (M0.5), login dan password (M1.1), audit log (M1.3), dan token integrasi (R-API-04, tahap 9).
- **Dampak bila dibiarkan:** rahasia bisa tercatat di log atau terkirim ke layar lewat pesan error. Bila rahasia sempat masuk ke audit log, ia tidak bisa dihapus lagi, karena audit log append-only (R-SEC-03).
- **Opsi:** (a) Tambah aturan keamanan baru: password, token, kunci, dan alamat koneksi yang memuat password tidak pernah ditulis ke log, pesan error, atau audit log. Pesan error ke pengguna tidak memuat detail teknis internal; detailnya dicatat di log server bersama request ID (R-API-13). Aturan ini dibuktikan dengan test, seperti `TestConnectError`. (b) Tanpa aturan tertulis, cukup diserahkan ke kehati-hatian saat menulis kode.
- **Rekomendasi Claude Code (usulan):** (a). Murah ditegakkan sejak awal, dan sangat sulit diperbaiki setelah rahasia terlanjur masuk ke audit log yang tidak bisa diubah.
- **Asumsi sementara di kode:** `Connect` tidak meneruskan pesan error pgx untuk alamat yang tidak bisa dibaca, dan `TestConnectError` memastikan password tidak muncul di pesan error. Ditandai `// ASUMSI E-18` di `internal/db/db.go`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 28 Oktober 2026 (D-95)
- **Keputusan pemilik:** Opsi (a), sesuai rekomendasi ("E-18 setuju", 28 Oktober 2026). Aturan keamanan baru: password, token, kunci, dan alamat koneksi yang memuat password tidak pernah ditulis ke log, pesan error, atau audit log. Pesan error ke pengguna tidak memuat detail teknis internal; detailnya dicatat di log server bersama request ID (R-API-13). Aturan ini dibuktikan dengan test. Komentar `// ASUMSI E-18` di `internal/db/db.go` diganti dengan keterangan keputusan, dan aturannya ditambahkan ke CLAUDE.md bagian 7.

### E-19 · Glosarium U1 belum punya istilah bahasa Inggris untuk layar

- **Tanggal dan milestone:** 28 Oktober 2026, M0.4. Berdampak ke semua layar mulai M0.5.
- **Jenis:** Celah
- **Rujukan:** Q-14 (diputuskan: layar berbahasa Inggris untuk saat ini), CLAUDE.md bagian 3 butir 12, U1 (kolom "Sebutan lain"), brief bagian 3 (glosarium berbahasa Inggris), R-UML-05.
- **Temuan:** Pemilik memutuskan layar memakai bahasa Inggris untuk saat ini (Q-14). Aturan kerja meminta istilah di layar mengikuti glosarium U1 dan istilah di WI, tanpa membuat istilah baru. U1 ditulis dalam bahasa Indonesia, dan kolom "Sebutan lain" hanya memuat padanan Inggris untuk sebagian istilah (misalnya Change, Revision request, Impact checklist). Istilah lain yang segera muncul di layar, seperti Project Group, group model, kode customer, atau penyusun, belum punya padanan Inggris yang ditetapkan.
- **Dampak bila dibiarkan:** setiap layar memilih kata Inggris sendiri, sehingga istilah bisa tidak konsisten antarlayar dan berbeda dari brief serta form.
- **Opsi:** (a) Tambah kolom "Istilah layar (Inggris)" di U1, diisi dari brief bagian 3 dan 4 (yang berbahasa Inggris) serta form dan WI. Katalog teks di `web/src/lib/i18n/en.ts` mengikuti kolom itu. (b) Tanpa kolom baru; istilah Inggris diambil langsung dari brief bagian 3 dan 4 saat layarnya dibuat.
- **Rekomendasi Claude Code (usulan):** (a). Satu sumber istilah untuk layar, cetakan, dan UML, dan memudahkan peralihan ke bahasa Indonesia nanti, karena setiap kunci teks punya pasangan istilah yang jelas.
- **Asumsi sementara di kode:** sampai diputuskan, teks layar memakai istilah Inggris dari brief bagian 3 dan 4 (misalnya customer, model group, part number, project group, approval route), dan tidak membuat istilah baru. Ditandai di `web/src/lib/i18n/en.ts`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-98), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): kolom "Istilah layar (Inggris)" di U1, dan `web/src/lib/i18n/en.ts` mengikutinya. U1 diperbarui 30 Oktober 2026.

### E-20 · Konvensi API R-API-13 merujuk tabel kode error yang tidak ada di repo

- **Tanggal dan milestone:** 28 Oktober 2026, M0.5. Berdampak ke semua endpoint berikutnya, dan ke API integrasi (M9.3).
- **Jenis:** Celah
- **Rujukan:** R-API-13 ([PROPOSED]), R-API-18, R-API-02, brief bagian 13 (spec lama `pfmea-system-phase1-spec-v2`), E-17, E-18.
- **Temuan:** R-API-13 meminta satu format error berisi code, message, details, dan request ID, dan meminta tabel kode error dari spec lama dipakai ulang. Spec lama tidak ada di repo ini, jadi tabelnya tidak bisa dipakai. R-API-13 juga belum menyebut: status HTTP untuk input yang salah (400 atau 422), bentuk `details`, bentuk `X-Request-Id` yang diterima dari klien, gaya nama field JSON, dan ukuran body permintaan paling besar.
- **Dampak bila dibiarkan:** kode error dan bentuknya ditetapkan oleh kode, bukan oleh desain. Bila tabel spec lama ternyata berbeda, klien (layar web dan nanti aplikasi lain) harus diubah.
- **Opsi:** (a) Pemilik menyalin tabel kode error spec lama ke brief, lalu kontrak disesuaikan. (b) Tabel yang dibuat di M0.5 dijadikan baseline dan dicatat di brief. (c) Pakai standar RFC 9457 (Problem Details, `application/problem+json`). Bentuknya berbeda dari yang diminta R-API-13 (code, message, details, request ID), jadi R-API-13 harus diubah.
- **Rekomendasi Claude Code (usulan):** (a) bila tabel spec lama masih ada dan masih sesuai. Bila tidak ada, pakai (b).
- **Asumsi sementara di kode (ASUMSI E-20):** dipakai di `api/openapi.yaml` dan `internal/server`:
  - Kode error dan statusnya: `invalid_request` 400 (body bukan JSON) atau 413 (body lebih dari 1 MiB); `validation_failed` 422; `conflict` 409; `not_found` 404; `method_not_allowed` 405 dengan header Allow; `internal_error` 500.
  - `details` berisi daftar `{field, code, message}`. Kode detail saat ini: `required`, `invalid_format`, `too_long`, `reserved`, `taken`. Layar menerjemahkan dari `field` dan `code`, bukan dari `message`.
  - `X-Request-Id` dari klien diterima bila panjangnya 1 sampai 64 karakter dan hanya berisi huruf, angka, titik, garis bawah, dan tanda hubung. Selain itu server membuat ID sendiri (26 karakter acak).
  - Nama field JSON memakai snake_case, mengikuti contoh di brief (`updated_since`, `on_behalf_of`).
  - Teks `message` dan `description` di kontrak berbahasa Inggris, sama dengan layar (Q-14).
  - Body permintaan paling besar 1 MiB. Unggahan file nanti punya batas sendiri.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-99), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (b): spec lama `pfmea-system-phase1-spec-v2` tidak ada di repo, jadi tabel kode error dari M0.5 (rincian asumsi di atas) menjadi baseline dan dicatat di brief R-API-13. Penanda `ASUMSI E-20` di kode diganti keterangan keputusan.

### E-21 · Field dan batas master customer belum ditentukan

- **Tanggal dan milestone:** 28 Oktober 2026, M0.5. Berdampak ke M1.4.
- **Jenis:** Celah
- **Rujukan:** brief bagian 4 (model data: customer punya kode dan satu Project Group), D-54, E-09, E-11, E-12, U1 (Kode customer).
- **Temuan:** Brief hanya menyebut bahwa customer punya kode (misalnya KT) dan masuk satu Project Group. Belum ada aturan untuk nama customer: wajib atau tidak, panjang paling banyak, boleh ganda atau tidak, dan apakah ada nama pendek di samping nama resmi. Field lain juga belum disebut, misalnya status aktif (E-09: customer tidak dihapus, hanya dinonaktifkan).
- **Dampak bila dibiarkan:** batas yang dipilih kode bisa terlalu sempit untuk nama resmi customer, atau ada field yang dibutuhkan cetakan dan baru ketahuan di Tahap 6.
- **Opsi:** (a) Pakai asumsi M0.5 di bawah, lalu tetapkan daftar field lengkap sebelum M1.4. (b) Pemilik menetapkan daftar field dan batasnya sekarang, dari data customer yang ada (misalnya daftar kode customer di WI SW-ZZ-000).
- **Rekomendasi Claude Code (usulan):** (a). Field lain ditambah lewat migrasi baru di M1.4 tanpa mengubah API yang sudah ada (R-API-15).
- **Asumsi sementara di kode (ASUMSI E-21):** nama wajib diisi, spasi di tepi dibuang, paling banyak 200 karakter, satu baris tanpa karakter kontrol, dan boleh sama dengan nama customer lain. Kode unik, tepat dua huruf A sampai Z, dan ZZ ditolak (ASUMSI E-12: ZZ untuk dokumen General/Common). Aturan ini dijaga di `internal/customer` dan oleh batasan tabel `customers`.
- **Menghambat milestone?** Tidak (perlu sebelum M1.4)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-100), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): nama wajib, paling banyak 200 karakter, satu baris, dan boleh sama dengan nama customer lain; kode unik, dua huruf A sampai Z. Field lain ditambah lewat migrasi baru saat dibutuhkan.

### E-22 · Tidak ada jalan keluar bila akun admin terakhir lupa password atau terkunci

- **Tanggal dan milestone:** 28 Oktober 2026, M1.1. Berdampak ke M1.2.
- **Jenis:** Celah
- **Rujukan:** E-07, E-08, D-72, D-73, R-SEC-13, UC-02.
- **Temuan:** Perintah `create-admin` hanya jalan bila belum ada akun admin sama sekali (E-07), dan akun admin terakhir tidak bisa dihapus (D-73). Reset password dilakukan admin (E-08). Bila hanya ada satu akun admin dan pemegangnya lupa password, tidak ada yang bisa mereset password itu. Hal yang sama terjadi bila akun admin dinonaktifkan setelah 100 kali gagal login (ASUMSI E-08). Siapa pun yang tahu nama login admin bisa memicunya dengan sengaja, karena OWASP sendiri memperingatkan bahwa kunci akun bisa dipakai untuk mengunci pengguna lain (denial of service).
- **Dampak bila dibiarkan:** sistem tidak bisa dikelola lagi tanpa mengubah database langsung, dan mengubah database langsung melewati audit log (M1.3).
- **Opsi:** (a) Perintah CLI kedua, misalnya `revioten reset-admin-password -login ...`. Perintah ini hanya bisa dijalankan di komputer server, membuka akun, menghapus kunci, memasang password sementara dengan wajib ganti, dan mencatat peristiwanya di audit log dengan aktor sistem. (b) Wajib ada dua akun admin yang dipegang dua orang berbeda, sehingga satu admin bisa mereset yang lain. (c) Akun admin dikecualikan dari penonaktifan setelah 100 kali gagal (tetap kena kunci 15 menit). Bisa digabung dengan (a) atau (b).
- **Rekomendasi Claude Code (usulan):** (a) dan (c). Akses ke komputer server sudah setara dengan akses admin, jadi (a) tidak membuka jalan baru. (c) menutup cara mudah untuk mengunci admin dari luar.
- **Asumsi sementara di kode:** belum ada perintah reset. Akun admin diperlakukan sama dengan akun lain (dikunci 15 menit tiap 10 gagal, dinonaktifkan pada kegagalan ke-100).
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-101), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) dan (c): perintah `revioten reset-admin-password` yang hanya jalan di komputer server dan tercatat di audit log dengan aktor `system`, ditambah akun admin tidak dinonaktifkan pada kegagalan ke-100 (kunci 15 menit tetap berlaku). Dibangun 30 Oktober 2026, sebelum M2.1.

### E-23 · Daftar password yang dilarang belum memenuhi ASVS 6.2.4 tanpa layanan luar

- **Tanggal dan milestone:** 28 Oktober 2026, M1.1.
- **Jenis:** Sulit dibangun
- **Rujukan:** E-08, ASVS 5.0 6.2.4 dan 6.2.12, NIST SP 800-63B-4 (blocklist), Q-02 (kebijakan IT).
- **Temuan:** ASVS 6.2.4 meminta password baru dicek terhadap minimal 3000 password teratas yang memenuhi aturan password (di sini: panjang minimal 15). ASVS 6.2.12 meminta cek terhadap kumpulan password yang pernah bocor. Daftar terbuka yang dipakai (NCSC top 100.000, dari Pwned Passwords) hanya berisi 329 password yang panjangnya 15 karakter atau lebih. Kumpulan yang lengkap ada di layanan Pwned Passwords (API k-anonymity, server mengirim 5 karakter awal hash SHA-1 ke internet) atau unduhan datanya (puluhan GB).
- **Dampak bila dibiarkan:** cek password umum lebih lemah dari ASVS 6.2.4. Panjang minimal 15 sudah menutup sebagian besar password umum, jadi risikonya rendah.
- **Opsi:** (a) Tetap memakai daftar offline yang ada (329 entri) dan mencatat kekurangannya. (b) Memakai API Pwned Passwords bila jaringan perusahaan mengizinkan server menghubungi internet (perlu jawaban Q-02). (c) Mengunduh dataset Pwned Passwords dan memasang filter lokal (berat untuk demo localhost).
- **Rekomendasi Claude Code (usulan):** (a) untuk demo localhost, dan (b) dibahas dengan IT untuk versi perusahaan.
- **Asumsi sementara di kode:** (a). File `internal/auth/common_passwords.txt` mencatat sumber dan cara penyaringannya.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-102), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) untuk demo localhost. Opsi (b), API Pwned Passwords, dibahas dengan IT untuk versi perusahaan dan dicatat untuk dokumen IT (M8.4).

### E-24 · U3 belum punya hak untuk melihat master data

- **Tanggal dan milestone:** 28 Oktober 2026, M1.2. Berdampak ke M1.4 sampai M1.8.
- **Jenis:** Celah
- **Rujukan:** U3-peta-pengguna bagian 3 (L-1 sampai L-4, A-2), UC-03 sampai UC-06, E-10, D-66, R-SEC-09.
- **Temuan:** Matriks U3 hanya punya A-2 "Kelola master data", yang dipegang Admin saja. Tidak ada hak untuk melihat master data. Padahal penyusun perlu memilih customer, model, proses, dan jenis dokumen saat mendaftarkan nomor dan menyusun dokumen, dan Viewer perlu melihat nama customer di register. Endpoint `GET /api/v1/customers` butuh satu hak untuk diperiksa (D-35).
- **Dampak bila dibiarkan:** setiap master baru di M1.4 sampai M1.8 memilih haknya sendiri, sehingga aturannya tidak seragam.
- **Opsi:** (a) Melihat master data termasuk L-2 (register, riwayat, dan diagram butuh nama customer, model, dan proses). Semua peran sudah punya L-2. (b) Hak baru di katalog, misalnya L-5 "Lihat master data", diberikan ke semua peran. (c) Master data terbuka untuk semua akun yang login, tanpa hak di katalog.
- **Rekomendasi Claude Code (usulan):** (a). Tidak menambah hak baru, dan hasilnya sama dengan isi awal U3. Bila nanti ada data customer yang harus dibatasi, cakupan customer (D-71) yang membatasinya.
- **Asumsi sementara di kode (ASUMSI E-24):** `GET /api/v1/customers` memeriksa L-2, di `internal/customer/service.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M1.4)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-103), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): melihat master data termasuk hak L-2.

### E-25 · Hak mengurus akun sendiri tidak ada di katalog U3

- **Tanggal dan milestone:** 28 Oktober 2026, M1.2. Berdampak ke M1.1 (sesi dan ganti password).
- **Jenis:** Celah
- **Rujukan:** U3-peta-pengguna bagian 1, 3, dan 4, UC-01, D-35, R-SEC-09, R-SEC-11, E-08.
- **Temuan:** Semua operasi lewat satu fungsi izin (D-35). Beberapa operasi tidak cocok dengan hak mana pun di U3: melihat sesi sendiri, logout, dan ganti password sendiri. Operasi ini harus selalu boleh untuk setiap akun, juga akun tanpa peran dan akun yang wajib ganti password (E-08).
- **Dampak bila dibiarkan:** bila dimasukkan ke katalog sebagai hak biasa, admin bisa mencabutnya dari sebuah peran, sehingga pengguna tidak bisa logout atau mengganti password yang direset. Bila tidak lewat fungsi izin sama sekali, ada operasi yang melewati titik pemeriksaan.
- **Opsi:** (a) Aturan tetap di luar katalog: setiap akun boleh mengurus akunnya sendiri, dan tidak ada yang bisa mematikannya. Tetap diperiksa fungsi izin, dengan resource berupa akun itu. Dicatat di U3 bagian 4. (b) Hak baru di katalog (misalnya L-0) yang diberikan ke semua peran dan tidak bisa dicabut.
- **Rekomendasi Claude Code (usulan):** (a). Hak yang tidak boleh dicabut lebih jelas ditulis sebagai aturan tetap daripada sebagai baris katalog dengan pengecualian.
- **Asumsi sementara di kode (ASUMSI E-25):** `access.ManageOwnAccount` di `internal/access/access.go`. `Authorize` mengizinkannya hanya bila resource adalah akun pemanggil sendiri.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-104), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): mengurus akun sendiri adalah aturan tetap di luar katalog hak, tetap lewat fungsi izin dengan resource akun itu sendiri. Dicatat di U3 bagian 4.

### E-26 · Akun admin terakhir "tidak bisa dihapus", padahal hapus akun belum diatur

- **Tanggal dan milestone:** 28 Oktober 2026, M1.2. Berdampak ke M1.3 (audit log menunjuk akun).
- **Jenis:** Ambigu
- **Rujukan:** D-73, R-SEC-11, U3-peta-pengguna bagian 4 aturan 7, D-74, R-SEC-03, UC-02, E-02.
- **Temuan:** D-73 dan R-SEC-11 menyebut akun admin terakhir tidak bisa dihapus (removed). Belum ada aturan apakah akun boleh dihapus sama sekali. Audit log (D-74) dan riwayat dokumen menunjuk akun dan orangnya, sehingga menghapus baris akun memutus rujukan itu, atau audit log harus menyimpan salinan nama.
- **Dampak bila dibiarkan:** bila operasi hapus akun dibuat, audit log dan riwayat bisa kehilangan pelaku, atau penghapusan ditolak foreign key dan admin tidak tahu kenapa.
- **Opsi:** (a) Akun tidak pernah dihapus, hanya dinonaktifkan (semua sesi berakhir, tidak bisa login). "Dihapus" di D-73 dibaca sebagai "dinonaktifkan". (b) Hapus diizinkan hanya untuk akun yang belum pernah dipakai. (c) Hapus dengan anonimisasi data orang.
- **Rekomendasi Claude Code (usulan):** (a). (b) atau (c) baru perlu bila ada kewajiban menghapus data pribadi, dan itu pertanyaan untuk perusahaan.
- **Asumsi sementara di kode (ASUMSI E-26):** tidak ada operasi hapus akun. Akun admin aktif terakhir tidak bisa dinonaktifkan, dan peran akun admin tidak bisa diubah sama sekali (`internal/auth/accounts.go`, `internal/access/roles.go`).
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-105), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): akun tidak pernah dihapus, hanya dinonaktifkan. "Removed" di D-73 dibaca "dinonaktifkan".

### E-27 · Jenis aktor untuk perintah server dan orang yang belum login

- **Tanggal dan milestone:** 28 Oktober 2026, M1.3. Berdampak ke M8.1 (service account).
- **Jenis:** Celah
- **Rujukan:** R-API-03, R-API-18, R-SEC-03, E-07, D-74.
- **Temuan:** R-API-03 menyebut tiga jenis aktor: human, service, dan customer user. Audit log butuh pelaku untuk dua peristiwa yang tidak punya akun: admin pertama yang dibuat perintah `create-admin` di komputer server (E-07), dan login gagal oleh orang yang belum login. Login gagal mengubah data (hitungan gagal, kunci akun), jadi harus tercatat.
- **Dampak bila dibiarkan:** peristiwa itu dicatat seolah-olah dilakukan akun yang menjadi sasarannya, sehingga audit log menuduh orang yang salah.
- **Opsi:** (a) Dua jenis aktor tambahan: `system` (perintah di komputer server) dan `anonymous` (belum login). Keduanya tidak punya akun, dan selalu ditolak fungsi izin. (b) Satu jenis tambahan `system` untuk keduanya. (c) Akun sistem khusus di tabel akun.
- **Rekomendasi Claude Code (usulan):** (a). Perbedaannya penting saat membaca log: perintah server dijalankan orang yang punya akses ke komputer server, sedangkan login gagal bisa datang dari siapa saja. Akun sistem (c) bisa disalahgunakan untuk login.
- **Asumsi sementara di kode (ASUMSI E-27):** `access.ActorSystem` dan `access.ActorAnonymous` di `internal/access/access.go`, serta batasan `actor_type` di `sql/migrations/00004_create_audit_log.sql`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-106), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): jenis aktor `system` (perintah di komputer server) dan `anonymous` (belum login). Keduanya tidak punya akun dan selalu ditolak fungsi izin.

### E-28 · Peristiwa apa saja yang masuk audit log

- **Tanggal dan milestone:** 28 Oktober 2026, M1.3. Berdampak ke semua milestone yang menambah operasi (Tahap 2 sampai 7).
- **Jenis:** Ambigu
- **Rujukan:** brief bagian 4 (`audit_log: Append-only record of every change to data`), R-SEC-03, R-SEC-10, R-APR-09, R-PUB-12, D-75.
- **Temuan:** Brief menyebut audit log sebagai catatan setiap perubahan data. Belum jelas apakah login, logout, dan login gagal termasuk, karena peristiwa itu mengubah data sesi dan hitungan gagal, tapi bukan data dokumen. Belum jelas juga apakah membaca atau mengekspor dokumen dicatat. R-PUB-12 hanya menyebut log akses untuk customer di portal.
- **Dampak bila dibiarkan:** setiap milestone memutuskan sendiri apa yang dicatat. Bila auditor nanti menanyakan siapa yang mengekspor dokumen rilis, jawabannya tidak ada.
- **Opsi:** (a) Setiap perubahan data yang disimpan, termasuk sesi dan hitungan gagal login; membaca tidak dicatat. (b) Seperti (a), ditambah ekspor dan cetak dokumen (L-3, E-5, E-9), karena salinan dokumen yang keluar sistem penting untuk pengendalian dokumen. (c) Semua akses, termasuk membaca.
- **Rekomendasi Claude Code (usulan):** (a) sekarang, dan (b) saat ekspor dan cetak dibangun (Tahap 6), karena salinan terkendali memang perlu dilacak. (c) membuat log sangat besar tanpa banyak manfaat untuk demo.
- **Asumsi sementara di kode (ASUMSI E-28):** (a). Login dengan nama yang tidak ada tidak mengubah data, jadi tidak dicatat. Pembersihan sesi yang habis waktunya juga tidak. Ditandai di `internal/audit/audit.go`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-107), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) sekarang: setiap perubahan data dicatat, membaca tidak. Opsi (b), ekspor dan cetak, ditambah saat Tahap 6.

### E-29 · Kredensial pemilik database dan siapa yang menjalankan migrasi

- **Tanggal dan milestone:** 28 Oktober 2026, M1.3. Berdampak ke M8.3 (deploy) dan M8.4 (dokumen untuk IT).
- **Jenis:** Celah
- **Rujukan:** E-06, R-SEC-03, R-SEC-11 butir 5, D-33, D-36, M8.3, M8.4.
- **Temuan:** E-06 memisahkan role pemilik (untuk migrasi) dan role aplikasi (tanpa `UPDATE` dan `DELETE` di audit log). Belum diatur di mana kredensial pemilik disimpan, dan siapa yang menjalankan migrasi. Di demo, program yang sama menjalankan migrasi saat mulai, jadi proses server sempat memegang password pemilik, dan keduanya ada di file `.env` yang sama.
- **Dampak bila dibiarkan:** penyerang yang menguasai proses server, atau yang bisa membaca `.env`, mendapat password pemilik. Dengan password itu ia bisa mematikan trigger, dan hanya hash chain yang tersisa sebagai pengaman.
- **Opsi:** (a) Seperti sekarang: migrasi saat program mulai, satu file `.env`. Sederhana untuk demo. (b) Migrasi sebagai langkah deploy terpisah (misalnya `revioten migrate`) dengan kredensial pemilik yang hanya ada saat deploy. Server hanya memegang password role aplikasi. (c) Seperti (b), ditambah pemeriksaan rantai harian dan hash terakhir yang dicatat di luar database (usulan R-SEC-03).
- **Rekomendasi Claude Code (usulan):** (a) untuk demo localhost (D-36), dan (c) dicatat di dokumen untuk IT (M8.4).
- **Asumsi sementara di kode (ASUMSI E-29):** (a). Ditandai di `cmd/revioten/main.go` (`openDatabase`).
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-108), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) untuk demo localhost. Opsi (c) ditulis di dokumen untuk IT (M8.4). Hash ujung rantai audit sudah dicatat di manifest backup M1.7 (D-119).

### E-30 · Field dan aturan Project Group, group model, model, dan part number belum ditetapkan

- **Tanggal dan milestone:** 28 Oktober 2026, M1.4. Berdampak ke M4.1 (header dokumen) dan Tahap 6 (cetakan).
- **Jenis:** Celah
- **Rujukan:** brief bagian 3 dan 4 (customer, model_group, model, part_number), D-39, D-42, D-54, R-DOC-08, UC-03, F-04, F-05, F-10, F-11, E-21.
- **Temuan:** Brief hanya menyebut hubungan: group model milik satu customer, model paling banyak satu group (D-42), dan customer masuk satu Project Group (D-54). Belum ada field dan aturan:
  1. Project Group: apakah punya nama selain kode GR1 sampai GR4, dan bentuk kodenya.
  2. Group model: nama, dan apakah unik dalam satu customer.
  3. Model: field apa saja (nama seperti "660A" di PFMEA, nomor model, deskripsi), dan apakah model selalu milik satu customer.
  4. Part number: apakah selalu milik satu model, apakah unik per customer atau seluruh sistem, dan apakah nama part (misalnya deskripsi di PFMEA) disimpan.
  5. Apakah model dan part number bisa dinonaktifkan seperti customer (E-09).
- **Dampak bila dibiarkan:** header dokumen di M4.1 dan cetakan di Tahap 6 butuh field yang mungkin belum disimpan, misalnya nama part atau nomor model.
- **Opsi:** (a) Pakai asumsi M1.4 di bawah, lalu tambah field lewat migrasi saat M4.1 butuh. (b) Pemilik menetapkan daftar field dari header PFD, PFMEA, dan CP yang ada sekarang (F-10, F-11) sebelum M4.1.
- **Rekomendasi Claude Code (usulan):** (b), dibahas bersama pemilik saat evaluasi Tahap 1, karena header form sudah menunjukkan field yang dicetak.
- **Asumsi sementara di kode (ASUMSI E-30):**
  - Project Group: kode huruf besar A sampai Z dan angka, 1 sampai 10 karakter, dan nama wajib.
  - Group model dan model: nama wajib, paling banyak 200 karakter, unik dalam satu customer.
  - Part number: milik satu model, nomor wajib (paling banyak 100 karakter, unik dalam satu customer), dan nama part boleh kosong.
  - Model tidak pindah customer, dan part number tidak pindah model. Belum ada status nonaktif untuk group model, model, dan part number.
  - Ditandai di `sql/migrations/00005_create_master_data.sql`, `internal/customer/validate.go`, dan `internal/product/product.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M4.1)
- **Status:** Ditunda, menunggu bahan pemilik (paling lambat sebelum M4.1). Cara (b) disetujui pemilik 30 Oktober 2026 lewat persetujuan umum: field Project Group, group model, model, dan part number ditetapkan dari header PFD, PFMEA, dan CP asli (F-10, F-11). Sampai bahannya ada, asumsi M1.4 di atas tetap berlaku.
- **Keputusan pemilik:** belum, menunggu bahan.

### E-31 · Kode customer lama: boleh dipakai customer lain, dan tanggal berlaku penggantian

- **Tanggal dan milestone:** 28 Oktober 2026, M1.4. Berdampak ke M2.1 (penghitung nomor) dan R-POR-01.
- **Jenis:** Ambigu
- **Rujukan:** E-11, E-16, R-NUM-02, R-NUM-03, R-POR-01, WI SW-ZZ-000 rev 04 dan rev 09 (penggantian HL menjadi MS dan BA menjadi RB).
- **Temuan:** E-11 menetapkan riwayat kode dengan tanggal berlaku. Belum jelas:
  1. Apakah kode lama satu customer boleh dipakai customer lain nanti. Bila boleh, nomor lama dengan kode itu menunjuk dua customer.
  2. Apakah customer boleh kembali ke kode lamanya sendiri.
  3. Apakah tanggal berlaku penggantian boleh dipilih (misalnya mengikuti tanggal revisi WI), atau selalu saat penggantian dicatat.
- **Dampak bila dibiarkan:** pencocokan nomor lama dengan customer (R-POR-01) dan penghitung per kode (E-16 butir 5) bisa memberi hasil yang salah.
- **Opsi:** (a) Satu kode hanya pernah milik satu customer, tidak bisa kembali, dan berlaku saat dicatat. (b) Seperti (a), tapi tanggal berlaku bisa diisi admin. (c) Kode bebas dipakai lagi setelah masa berlakunya habis, dan pencocokan nomor lama memakai tanggal nomor.
- **Rekomendasi Claude Code (usulan):** (a). Paling sederhana dan tidak pernah ambigu. Tanggal berlaku yang dipilih (b) bisa ditambah bila nanti dibutuhkan untuk mengimpor riwayat WI.
- **Asumsi sementara di kode (ASUMSI E-31):** (a). Kode menjadi primary key `customer_codes`, jadi database menolak kode yang pernah dipakai. Ditandai di `sql/migrations/00005_create_master_data.sql` dan `internal/customer/service.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M2.1)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-109), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): satu kode hanya pernah milik satu customer, tidak bisa kembali, dan berlaku saat dicatat.

### E-32 · Aturan nama dan alias master proses belum ditetapkan

- **Tanggal dan milestone:** 28 Oktober 2026, M1.5. Berdampak ke M3.2 (process template) dan impor dokumen lama (R-DOC-04, R-MIG).
- **Jenis:** Celah
- **Rujukan:** D-06, D-64, R-DOC-04, UC-04, brief bagian 3 dan 4 (process), U1 (Proses), F-05.
- **Temuan:** Brief menetapkan ID tetap, nama hanya tampilan, dan alias boleh ada. Belum ada aturan:
  1. Apakah master proses satu untuk semua customer, atau per customer.
  2. Apakah nama dan alias harus unik, dan di mana: antar nama saja, atau nama dan alias berbagi satu ruang nama. Bila tidak unik, satu nama di dokumen lama bisa menunjuk dua proses saat impor (R-DOC-04).
  3. Apakah perbedaan huruf besar kecil dan spasi ganda dianggap nama yang sama. F-05 menunjukkan nama yang hampir sama ("Parts Receiving" dan "Part Receiving").
  4. Batas panjang nama dan jumlah alias.
  5. Apakah proses bisa dihapus atau dinonaktifkan.
  6. Apakah nama lama otomatis menjadi alias saat proses diganti nama.
- **Dampak bila dibiarkan:** pencocokan nama ke ID saat impor (R-DOC-04) bisa ambigu, dan daftar pilihan proses di PFMEA dan CP bisa berisi dua proses dengan nama yang sama.
- **Opsi:** (a) Satu master untuk semua customer; nama dan alias berbagi satu ruang nama tanpa membedakan huruf besar kecil dan spasi ganda; proses tidak dihapus; nama lama tidak otomatis menjadi alias. (b) Seperti (a), tapi nama lama otomatis menjadi alias saat diganti. (c) Master per customer, dengan ruang nama per customer.
- **Rekomendasi Claude Code (usulan):** (a). Ruang nama tunggal membuat pencocokan impor tidak pernah ambigu. Nama lama tidak otomatis menjadi alias, karena ganti nama sering untuk membetulkan salah ketik, dan salah ketik tidak perlu dipertahankan. Status nonaktif ditambah saat Tahap 3 membutuhkan daftar pilihan proses.
- **Asumsi sementara di kode (ASUMSI E-32):** (a), dengan batas nama dan alias 200 karakter dan paling banyak 20 alias per proses. Database menegakkan ruang nama tunggal lewat primary key `name_key` (nama dalam huruf kecil) di tabel `process_names`. Ditandai di `sql/migrations/00006_create_processes.sql` dan `internal/process/process.go`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-110), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): satu master untuk semua customer, nama dan alias berbagi satu ruang nama tanpa membedakan huruf besar kecil, proses tidak dihapus, dan nama lama tidak otomatis menjadi alias. Batas 200 karakter dan 20 alias per proses ikut disetujui.

### E-33 · Pencocokan PFD, PFMEA, dan CP: di tingkat proses atau sub-proses

- **Tanggal dan milestone:** 28 Oktober 2026, M1.5. Berdampak ke M3.2 (process template) dan Tahap 3 (baris dokumen).
- **Jenis:** Ambigu
- **Rujukan:** R-DOC-04, R-TPL-01, R-TPL-04, brief bagian 3 (Process / sub-process / point), U1 (Proses, Sub-proses), F-05.
- **Temuan:** R-DOC-04 berbunyi: PFD, PFMEA, dan CP dicocokkan per proses, dan ID-nya dipegang master proses. R-TPL-01 menaruh sub-proses (misalnya IQC General, IQC MSD, IQC PCB) di process template, bukan di master proses. Belum jelas apakah baris PFMEA dan CP cukup menunjuk proses (IQC), atau harus menunjuk sub-proses (IQC MSD). Beberapa contoh F-05 ("LED Test Lighting" dan "LED Lighting Test") tampak seperti nama setingkat sub-proses.
- **Dampak bila dibiarkan:** bila pencocokan perlu sampai sub-proses, sub-proses juga butuh ID tetap dan alias, dan master proses M1.5 harus diperluas sebelum M3.2.
- **Opsi:** (a) Pencocokan di tingkat proses. Sub-proses tetap varian di process template, dengan ID tetap dari template. (b) Sub-proses juga masuk master proses, masing-masing dengan ID tetap, alias, dan induk (proses). (c) Diputuskan setelah membandingkan satu set PFD, PFMEA, dan CP asli baris per baris.
- **Rekomendasi Claude Code (usulan):** (c) sebelum M3.2, karena jawabannya bergantung pada cara dokumen sekarang ditulis. Skema M1.5 bisa diperluas ke (b) dengan kolom induk tanpa mengubah ID proses yang sudah ada.
- **Asumsi sementara di kode:** master proses M1.5 hanya berisi proses.
- **Asumsi sementara di kode sejak M3.2 (ASUMSI E-33):** opsi (a). Pencocokan tetap di tingkat proses. Sub-proses punya identitas sendiri (`subprocesses`) dengan ID tetap, milik satu proses master, dan namanya disimpan per versi process template. Bila pemilik memilih (b), nama dan alias sub-proses pindah ke master dengan ID yang sama, jadi rujukan dari template dan dokumen tidak berubah. Ditandai di `sql/migrations/00012_create_process_templates.sql` dan `internal/template/process.go`.
- **Menghambat milestone?** Tidak. M3.2 dibangun dengan asumsi di atas; keputusan dibutuhkan sebelum M4.1, karena baris dokumen menunjuk proses (R-DOC-04).
- **Status:** Ditunda, menunggu bahan pemilik. Cara (c) disetujui pemilik 30 Oktober 2026 lewat persetujuan umum: satu set PFD, PFMEA, dan CP asli dibandingkan baris per baris, lalu dipilih (a) atau (b). Pada 31 Oktober 2026 pemilik meminta proyek dilanjutkan dulu ("lanjutkan proyek untuk saat ini"), karena dokumen asli hanya ada di komputer pemilik dan sesi cloud tidak bisa mengaksesnya; M3.2 dibangun dengan ASUMSI E-33.
- **Bukti baru (tinjauan desain UI, 3 November 2026):** artboard kanvas "Desain UI Sistem PFD PFMEA CP" dibuat dari dokumen TR-020, dan memperlihatkan pola yang sama di tiga dokumen (UX-04):
  - nomor operasi sama di PFD, PFMEA, dan CP, sedangkan namanya sering berbeda (F-05), jadi pencocokan memakai nomor operasi dan ID proses master;
  - PFMEA bekerja di tingkat proses (IQC hanya satu kelompok baris);
  - CP memecah satu proses menjadi sub-proses (IQC: General Material, Material MSD, PCB) dan punya item cek khusus CP seperti IPQC 1st Printing Inspection;
  - di pustaka (artboard Compose), satu sub-proses membawa poin PFMEA dan poin CP sekaligus.

  Pola ini cocok dengan opsi (a) dan dengan ASUMSI E-33 yang sudah dibangun di M3.2: sub-proses adalah varian di process template, PFMEA menampilkan barisnya di bawah proses, CP menampilkannya sebagai pecahan proses, dan item cek khusus CP adalah sub-proses yang hanya berisi poin CP.
- **Rekomendasi Claude Code (usulan, diperbarui 3 November 2026):** (a), sesuai bukti di atas. Tidak perlu perubahan kode; ASUMSI E-33 cukup dinyatakan sebagai keputusan.
- **Keputusan pemilik:** belum, menunggu bahan.

### E-34 · Field wajib di kontrak tidak diperiksa oleh server hasil generate

- **Tanggal dan milestone:** 28 Oktober 2026, M1.5. Berdampak ke M8.1 (klien integrasi) dan M9.3 (konvensi API).
- **Jenis:** Celah
- **Rujukan:** R-API-02, R-API-13, R-API-15, D-61.
- **Temuan:** oapi-codegen membuat tipe dari kontrak, tapi tidak memeriksa apakah field wajib (`required`) benar-benar dikirim. Field yang tidak dikirim menjadi nilai kosong. Untuk teks, layanan domain tetap menolaknya (wajib diisi). Untuk boolean dan daftar, nilai kosong punya arti:
  1. `PUT /api/v1/customers/{id}` tanpa `active` dibaca sebagai `false`, sehingga customer dinonaktifkan.
  2. `PUT /api/v1/processes/{id}` tanpa `aliases` akan menghapus semua alias. Di M1.5 kasus ini sudah ditolak handler dengan `aliases:required`.
  Batas lain di kontrak, misalnya `maxItems` dan `enum`, juga belum diperiksa server; layanan domain yang memeriksa.
- **Dampak bila dibiarkan:** klien API yang lupa satu field bisa mengubah data tanpa sengaja. Layar web selalu mengirim semua field, jadi risikonya muncul saat klien integrasi ada (M8.1).
- **Opsi:** (a) Pasang middleware yang memeriksa setiap permintaan terhadap kontrak (misalnya kin-openapi lewat middleware oapi-codegen). Butuh paket baru dan pengecekan versi (R-VER). (b) Periksa di handler hanya untuk field yang nilai kosongnya berbahaya, seperti `aliases` di M1.5. (c) Ubah kontrak supaya field seperti itu bertipe pointer, sehingga "tidak dikirim" bisa dibedakan.
- **Rekomendasi Claude Code (usulan):** (b) sekarang, termasuk `active` di customer, lalu (a) diputuskan sebelum M8.1, karena klien integrasi tidak bisa diandalkan selalu mengirim semua field.
- **Asumsi sementara di kode (ASUMSI E-34):** (b) untuk `aliases` di `internal/server/processes.go`. `active` di customer belum diubah, supaya M1.5 tidak melebar ke M1.4.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-111), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (b) sekarang, termasuk `active` di `PUT /api/v1/customers/{id}` yang kini opsional dengan arti "tidak dikirim berarti tetap" (dibangun 30 Oktober 2026). Opsi (a), middleware pemeriksa kontrak, diputuskan sebelum M8.1.

### E-35 · Bentuk kebijakan approval per jenis perubahan belum ditentukan (Q-15)

- **Tanggal dan milestone:** 28 Oktober 2026, M1.6. Berdampak ke M1.8 (rute approval), Tahap 5 (alur perubahan), dan Tahap 6 (cetak untuk tanda tangan).
- **Jenis:** Celah
- **Rujukan:** R-TPL-10, Q-15, D-46, D-47, R-APR-07, R-APR-08, R-APR-09, RK-05, UC-05.
- **Temuan:** R-TPL-10 (diputuskan) berbunyi: jenis perubahan (isi, tata letak saja, administratif) menentukan jalur approval, dan kebijakan per jenis adalah master data. Q-15 (terbuka) menanyakan langkah approval mana yang berlaku untuk setiap jenis, dengan asumsi kerja "alur yang sama untuk semua". Bentuk datanya belum ada, dan bisa dibaca dua cara:
  1. Kebijakan memilih rute approval (D-46 punya dua rute: sampai QA QC Manager, dan sampai GM).
  2. Kebijakan mematikan tahap tertentu (Checked atau Approved). Ini bertentangan dengan R-APR-07 (diputuskan), yang menetapkan tahap Prepared, Checked, dan Approved.
- **Dampak bila dibiarkan:** M1.6 tidak bisa menyimpan kebijakan yang berarti sebelum rute ada (M1.8), dan alur perubahan serta pencetakan butuh jawabannya.
- **Opsi:** (a) Kebijakan adalah rute per jenis perubahan: kosong berarti rute default customer (D-47), atau satu rute tertentu. Bila Q-15 meminta alur yang lebih pendek (misalnya hanya Prepared dan Checked), Admin membuatnya sebagai rute baru. (b) Kebijakan adalah saklar per tahap (Checked, Approved) untuk setiap jenis. (c) Satu alur untuk semua jenis sampai Q-15 dijawab, tanpa data kebijakan.
- **Rekomendasi Claude Code (usulan):** (a), karena hanya memakai konsep yang sudah diputuskan (rute D-46 dan rute default D-47), dan alur pendek tetap bisa dibuat sebagai data. Yang masih perlu ditetapkan pemilik: mana yang menang bila rute dari jenis perubahan berbeda dengan rute yang dipilih penyusun untuk satu dokumen (R-APR-09).
- **Asumsi sementara di kode (ASUMSI E-35):** (a), dibangun di M1.8 bersama rute approval. Sampai itu semua jenis perubahan memakai rute default customer (asumsi kerja Q-15). Ditandai di `sql/migrations/00007_create_document_and_change_types.sql` dan `internal/change/change.go`.
- **Menghambat milestone?** Tidak (perlu sebelum Tahap 5)
- **Status:** Diputuskan sebagian pemilik 30 Oktober 2026 (D-112), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026"). Pertanyaan sisa terbuka sampai sebelum Tahap 5
- **Keputusan pemilik:** Opsi (a), yang sudah dibangun di M1.8: setiap jenis perubahan boleh menunjuk satu rute, dan kosong berarti rute default customer. Pertanyaan sisanya, mana yang menang bila penyusun memilih rute lain untuk satu dokumen (R-APR-09), tetap terbuka sampai sebelum Tahap 5, karena belum pernah ada saran untuknya. Usulan Claude Code (belum diputuskan): rute dari jenis perubahan menjadi rute awal dokumen, penyusun tetap boleh menggantinya dengan alasan yang tercatat (D-47), dan dokumen kontrol melihat penggantian itu saat mencatat rilis (RK-60).

### E-36 · Sumber perubahan mencampur mekanisme sistem (R-CHG-01) dan pemicu (R-CHG-06)

- **Tanggal dan milestone:** 28 Oktober 2026, M1.6. Berdampak ke Tahap 5 (alur perubahan, yokoten, propagasi template).
- **Jenis:** Ambigu
- **Rujukan:** R-CHG-01, R-CHG-04, R-CHG-06, R-TPL-06, R-TPL-07, UC-05.
- **Temuan:** R-CHG-01 (diputuskan) menyebut empat sumber: permintaan revisi manual, yokoten, propagasi template, dan perubahan SQAM surut. Masing-masing punya perilaku sistem sendiri, misalnya yokoten memilih target, dan propagasi template dibuat sistem. R-CHG-06 (usulan) menyebut master data sumber berisi PP, customer claim, ECR, yokoten, propagasi template, SQAM, dan temuan audit. Jadi satu daftar berisi dua hal: mekanisme yang dikenali kode program, dan pemicu untuk permintaan manual. Bila Admin menonaktifkan "Yokoten", alur yokoten di Tahap 5 kehilangan sumbernya.
- **Dampak bila dibiarkan:** perubahan yokoten yang dipicu customer claim (klaim di satu model, lalu diterapkan ke model lain) tidak bisa mencatat keduanya bila sumber hanya satu field.
- **Opsi:** (a) Satu daftar. Entri milik mekanisme sistem ditandai, selalu aktif, dan namanya boleh diganti. Pemicu lain bebas ditambah dan dinonaktifkan. (b) Dua field di perubahan: mekanisme (tetap, dikenali kode: manual, yokoten, template, SQAM) dan pemicu (master data: PP, customer claim, ECR, temuan audit, dan lainnya). (c) Daftar seperti R-CHG-06 apa adanya, tanpa penanda.
- **Rekomendasi Claude Code (usulan):** (b) untuk Tahap 5. Untuk M1.6 cukup (a), yang bisa dipindah ke (b) tanpa kehilangan data.
- **Asumsi sementara di kode (ASUMSI E-36):** (a). Kolom `mechanism` menandai yokoten, propagasi template, dan SQAM, dan batasan `change_sources_system_active` membuat database menolak menonaktifkannya. Ditandai di `sql/migrations/00007_create_document_and_change_types.sql` dan `internal/change/`.
- **Menghambat milestone?** Tidak (perlu sebelum Tahap 5)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-113), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) sekarang. Opsi (b), mekanisme dan pemicu sebagai dua field, dibangun di Tahap 5.

### E-37 · Kode khusus: bentuk, penambahan, dan tumpang tindih dengan kode customer

- **Tanggal dan milestone:** 28 Oktober 2026, M1.6. Berdampak ke M2.1 (struktur nomor) dan R-POR-01.
- **Jenis:** Celah
- **Rujukan:** E-11, E-12, R-NUM-01, R-POR-01, WI SW-ZZ-000.
- **Temuan:** E-12 menetapkan kode khusus ZZ dan HSE sebagai master data. Belum ada aturan:
  1. Bentuk kode baru: ZZ dua huruf, HSE tiga huruf.
  2. Apakah Admin boleh menambah kode khusus.
  3. Apakah kode khusus boleh sama dengan kode customer, sekarang atau dulu.
  4. Apakah PFD, PFMEA, dan CP bisa dinomori dengan kode khusus, misalnya PFD umum dengan ZZ. R-NUM-01 hanya menyebut nomor per customer.
- **Dampak bila dibiarkan:** tanpa aturan butir 3, satu kode di posisi customer bisa punya dua arti, dan pencocokan nomor lama (R-POR-01) bisa salah.
- **Opsi:** (a) Dua atau tiga huruf, bisa ditambah Admin, kodenya tidak bisa diubah, dan tidak boleh sama dengan kode customer mana pun (sekarang atau dulu), begitu pula sebaliknya. Dijaga kunci bersama di aplikasi. (b) Seperti (a), tapi daftarnya tetap (hanya ZZ dan HSE). (c) Satu tabel untuk semua kode di posisi customer (kode customer dan kode khusus), sehingga database menjaga keunikan dengan satu UNIQUE.
- **Rekomendasi Claude Code (usulan):** (a) untuk sekarang. (c) lebih kuat karena dijaga database, tapi mengubah tabel riwayat kode M1.4, jadi layak dipertimbangkan di M2.1 saat struktur nomor dibangun. Butir 4 diputuskan di M2.1.
- **Pelaksanaan (M2.1, 30 Oktober 2026):** opsi (c), satu tabel untuk semua kode di posisi customer, dipertimbangkan dan tidak dibangun: kunci advisory 11003 sudah menjaga keunikan, dan memindah riwayat kode customer M1.4 ke tabel baru menambah risiko tanpa manfaat untuk Tahap 2. Butir 4 memakai asumsi paling aman sesuai R-NUM-01 [DECIDED]: pendaftaran nomor selalu untuk satu customer, jadi kode khusus belum bisa dipakai untuk menomori PFD, PFMEA, dan CP (`internal/numbering/register.go`).
- **Asumsi sementara di kode (ASUMSI E-37):** (a). Kunci advisory 11003 diambil saat customer dibuat, saat kode customer diganti, dan saat kode khusus ditambah, lalu pemeriksaan dilakukan di dalam kunci itu. Ditandai di `sql/migrations/00007_create_document_and_change_types.sql`, `sql/queries/numbering.sql`, `internal/numbering/`, dan `internal/customer/service.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M2.1)
- **Status:** Diputuskan sebagian pemilik 30 Oktober 2026 (D-114), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026"). Butir 4 terbuka
- **Keputusan pemilik:** Opsi (a) untuk butir 1 sampai 3: dua atau tiga huruf, bisa ditambah Admin, kode tidak bisa diubah, dan tidak boleh sama dengan kode customer mana pun, sekarang atau dulu. Opsi (c) dipertimbangkan di M2.1 (hasilnya di catatan M2.1). Butir 4 tetap terbuka, karena belum pernah ada saran untuknya; asumsi M2.1 ada di E-37 bagian Pelaksanaan.

### E-38 · Jumlah pemegang setiap jabatan dan riwayatnya

- **Tanggal dan milestone:** 28 Oktober 2026, M1.8. Berdampak ke Tahap 6 (cetak untuk tanda tangan).
- **Jenis:** Celah
- **Rujukan:** D-54, D-56, D-58, D-70, D-76, R-APR-07, R-APR-11, R-APR-15, UC-02.
- **Temuan:** D-54 menetapkan satu Project Leader dan satu Project Engineer per Project Group, dan menyebut APQP/PPAP Controller "satu orang" untuk keadaan sekarang. Untuk jabatan lain belum ada aturan:
  1. Apakah NPI Manager, QA QC Manager, Advisor, dan Factory Manager hanya boleh satu orang. Cover mencetak nama pemegang di setiap slot (D-76), jadi bila pemegangnya lebih dari satu, nama yang dicetak harus dipilih.
  2. Apakah PPAP Engineer dan PPAP Staff boleh beberapa orang.
  3. Apakah riwayat pemegang (siapa memegang jabatan apa, sejak kapan, sampai kapan) perlu disimpan, atau cukup audit log dan salinan rute di revisi yang dicetak (R-APR-11).
- **Dampak bila dibiarkan:** cetak untuk tanda tangan di Tahap 6 tidak tahu nama siapa yang dicetak di slot Approved bila pemegangnya lebih dari satu.
- **Opsi:** (a) Hanya Project Leader dan Project Engineer yang dibatasi satu orang per group; jabatan lain boleh beberapa pemegang, dan nama yang dicetak dipilih saat cetak. Riwayat cukup di audit log. (b) Jabatan approver (NPI Manager, QA QC Manager, Advisor, Factory Manager) dan APQP/PPAP Controller dibatasi satu orang. (c) Pemegang jabatan punya tanggal mulai dan selesai, supaya riwayatnya bisa dibaca langsung.
- **Rekomendasi Claude Code (usulan):** (b) untuk jabatan approver, karena satu slot mencetak satu nama, dan (a) untuk jabatan lain. Riwayat cukup di audit log, karena revisi yang dicetak menyimpan salinan rutenya sendiri.
- **Asumsi sementara di kode (ASUMSI E-38):** (a). Ditandai di `sql/migrations/00008_create_positions_and_routes.sql`.
- **Menghambat milestone?** Tidak (perlu sebelum Tahap 6)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-115), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (b) untuk jabatan approver (NPI Manager, QA QC Manager, Advisor, dan Factory Manager) dan APQP/PPAP Controller: satu pemegang per jabatan. Jabatan lain tetap opsi (a). Riwayat pemegang cukup di audit log. Dibangun 30 Oktober 2026 (migrasi 00009).

### E-39 · Customer tanpa rute default

- **Tanggal dan milestone:** 28 Oktober 2026, M1.8. Berdampak ke Tahap 6.
- **Jenis:** Celah
- **Rujukan:** D-47, R-APR-09, E-09.
- **Temuan:** D-47 berbunyi: setiap customer punya rute default. Customer yang dibuat sebelum M1.8 belum punya rute, dan formulir customer (M1.4) tidak meminta rute. Belum jelas apa yang terjadi bila dokumen customer tanpa rute default dicetak untuk tanda tangan.
- **Dampak bila dibiarkan:** cetak untuk tanda tangan di Tahap 6 tidak punya rute untuk dokumen customer itu.
- **Opsi:** (a) Rute default boleh kosong; Admin mengaturnya di layar Approval routes, dan cetak ditolak dengan pesan yang jelas sampai rute diatur, kecuali penyusun memilih rute lain dengan alasan (R-APR-09). (b) Rute default wajib dipilih saat customer dibuat, dan customer lama diisi lewat migrasi dengan satu rute yang dipilih pemilik. (c) Satu rute default untuk seluruh sistem, dipakai bila customer belum diatur.
- **Rekomendasi Claude Code (usulan):** (a), ditambah tanda "Not set" di daftar customer. Tidak ada rute yang dikarang untuk customer lama.
- **Asumsi sementara di kode (ASUMSI E-39):** (a). Kolom `customers.default_route_id` boleh kosong. Ditandai di `sql/migrations/00008_create_positions_and_routes.sql` dan `internal/customer/customer.go`.
- **Menghambat milestone?** Tidak (perlu sebelum Tahap 6)
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-116), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a): rute default boleh kosong dan tampil "Not set" di layar Approval routes. Cetak untuk tanda tangan ditolak sampai rute diatur, kecuali penyusun memilih rute lain dengan alasan (Tahap 6).

### E-40 · Batas bentuk rute dan isian slot

- **Tanggal dan milestone:** 28 Oktober 2026, M1.8.
- **Jenis:** Celah
- **Rujukan:** D-45, D-46, D-48, D-49, R-APR-07, R-APR-08, R-APR-17, F-01, E-35.
- **Temuan:** Rute adalah data yang bisa diubah Admin (R-APR-08), jadi sistem perlu tahu bentuk rute yang sah. Keputusan yang ada memberi: tahap Prepared, Checked, dan Approved (D-45), Checked satu tanda tangan (D-48), urutan berurutan (D-49), dan rute berakhir di QA QC Manager atau GM (D-46). Belum ada batas jumlah slot dan panjang judul cetak. Bentuk ini juga bertentangan dengan opsi (a) E-35 bila nanti dibutuhkan rute tanpa slot Approved.
- **Dampak bila dibiarkan:** tanpa pemeriksaan, Admin bisa menyimpan rute yang tidak bisa dicetak (misalnya Checked sebelum Prepared).
- **Opsi:** (a) Tepat satu Prepared di depan, tepat satu Checked sesudahnya, lalu satu atau lebih Approved; paling banyak 10 slot dan judul cetak paling banyak 100 karakter. (b) Seperti (a), tapi Approved boleh tidak ada, untuk rute pendek E-35. (c) Bebas, dan pemeriksaan bentuk baru dilakukan saat cetak.
- **Rekomendasi Claude Code (usulan):** (a) sampai Q-15 dan E-35 dijawab. Bila pemilik memilih alur tanpa Approved untuk perubahan tertentu, aturan ini diubah menjadi (b).
- **Asumsi sementara di kode (ASUMSI E-40):** (a). Ditandai di `internal/approval/approval.go`.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-117), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) sampai Q-15 dan E-35 dijawab: tepat satu Prepared di depan, tepat satu Checked sesudahnya, lalu satu atau lebih Approved; paling banyak 10 slot; judul cetak paling banyak 100 karakter.

### E-41 · Hak bercakupan Project Group untuk operasi tanpa customer

- **Tanggal dan milestone:** 28 Oktober 2026, M1.8. Berdampak ke operasi daftar di Tahap 2 sampai 5 dan ke M8.1.
- **Jenis:** Celah
- **Rujukan:** D-71, R-API-03, R-SEC-09, E-04, E-05.
- **Temuan:** D-71 membatasi Project Leader dan Project Engineer ke customer di Project Group sendiri. Belum ada aturan untuk operasi yang tidak menyangkut satu customer, misalnya daftar draft dari semua customer. Sebelum M1.8, fungsi izin meloloskan hak bercakupan untuk operasi seperti itu.
- **Dampak bila dibiarkan:** daftar lintas customer bisa memperlihatkan data customer di luar Project Group, bila operasinya lupa menyaring.
- **Opsi:** (a) Hak bercakupan tidak berlaku untuk operasi tanpa customer (ditolak). Operasi daftar lintas customer harus menyaring sendiri per customer, atau meminta customer di alamatnya. (b) Hak bercakupan berlaku, dan setiap operasi daftar wajib menyaring hasilnya. (c) Fungsi izin mengembalikan daftar customer yang boleh, dan operasi daftar memakainya untuk menyaring.
- **Rekomendasi Claude Code (usulan):** (a) sekarang, karena tolak bila ragu lebih aman. (c) dipertimbangkan saat daftar lintas customer pertama dibangun (Tahap 2).
- **Asumsi sementara di kode (ASUMSI E-41):** (a). Ditandai di `internal/access/checker.go`. Belum ada operasi yang terdampak: semua hak yang bisa bercakupan Project Group (Penyusun dan Verifikator) belum dipakai operasi mana pun.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-118), lewat persetujuan umum atas semua saran Claude Code (lihat bagian "Keputusan 30 Oktober 2026")
- **Keputusan pemilik:** Opsi (a) sekarang: hak bercakupan Project Group tidak berlaku untuk operasi tanpa customer. Opsi (c) dibangun saat daftar lintas customer pertama untuk peran bercakupan dibuat. Catatan yokoten: pemilihan target yokoten (M5.9) memakai opsi (c), supaya Project Leader dan Project Engineer tetap bisa mencari target di customer Project Group-nya. Tahap 2 belum punya daftar seperti itu, karena register nomor dibaca lewat L-2 yang tidak bercakupan.

### E-42 · Hak akun admin: R-SEC-13 menyebut hanya A-1 sampai A-4, U3 dan kode memberi juga L-1 sampai L-3

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen. Berdampak ke M1.2 dan M8.1.
- **Jenis:** Kontradiksi
- **Rujukan:** R-SEC-13, D-72, U3-peta-pengguna bagian 1 dan 3, E-24, `internal/access/access_test.go`.
- **Temuan:** R-SEC-13 berbunyi "the admin uses a separate admin account that holds only admin rights (A-1 to A-4)". Matriks U3 memberi peran Admin juga L-1, L-2, dan L-3, dan U3 bagian 1 memberi semua pengguna yang login hak dasar Lihat. Kode mengikuti U3: semua akun, termasuk akun admin, punya hak peran Viewer. Keputusan E-24 juga bergantung pada akun admin memegang L-2, karena admin mengelola master data yang dibaca lewat L-2.
- **Dampak bila dibiarkan:** tim IT yang membaca R-SEC-13 membangun akun admin tanpa hak lihat, sehingga admin tidak bisa melihat daftar master yang ia kelola.
- **Opsi:** (a) R-SEC-13 diperjelas: akun admin memegang hak admin ditambah hak dasar Lihat (L-1 sampai L-3) yang dimiliki semua akun, dan tidak pernah memegang peran kerja (Buat/Edit atau Verifikasi). (b) Akun admin hanya A-1 sampai A-4, dan hak lihat yang dibutuhkan admin ditambahkan ke A-2.
- **Rekomendasi Claude Code (usulan):** (a). Maksud R-SEC-13 adalah memisahkan pekerjaan admin dari pekerjaan dokumen, dan hak lihat bukan pekerjaan dokumen.
- **Asumsi sementara di kode:** tidak ada. Kode sudah seperti (a).
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-120), lewat persetujuan umum atas semua saran Claude Code.
- **Keputusan pemilik:** Opsi (a). R-SEC-13 di brief v1.16 diperjelas.

### E-43 · Q-21 menyebut lima peran, bagian 1 enam, U3 delapan

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** brief Q-21, brief bagian 1 (System roles), U3-peta-pengguna, E-01.
- **Temuan:** asumsi kerja Q-21 berbunyi "five roles from section 1", padahal bagian 1 mendaftar enam peran, dan U3 (baseline, E-01) memakai delapan.
- **Dampak bila dibiarkan:** pembaca brief mendapat tiga jumlah peran yang berbeda.
- **Opsi:** (a) Asumsi kerja Q-21 menunjuk ke delapan peran U3.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di brief v1.16 (Q-21 dan bagian 1).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-44 · Nomor berjalan pertama: 000 atau 001

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen. Berdampak ke M2.1 dan M2.5.
- **Jenis:** Ambigu
- **Rujukan:** R-NUM-01, brief bagian 17 (contoh SC-KT-000), M2.1 ("SC-KT-001"), U1 (Nomor dokumen, SC-KT-001), E-12 (A001 sampai A999, lalu B001), WI SW-ZZ-000 (DDD 000 sampai 999).
- **Temuan:** R-NUM-01 dan bagian 17 memberi contoh nomor `SC-KT-000`. M2.1, U1, dan asumsi E-12 memakai `SC-KT-001`. WI SW-ZZ-000 menulis DDD sebagai 000 sampai 999, dan WI itu sendiri bernomor SW-ZZ-000. Aturan pemilik untuk nomor di atas 999 dimulai dari A001, bukan A000.
- **Dampak bila dibiarkan:** penghitung M2.1 bisa memberi nomor pertama yang berbeda dari kebiasaan register yang ada.
- **Opsi:** (a) Nomor berjalan pertama 001; nomor 000 di register lama tetap bisa diimpor apa adanya (R-NUM-06). (b) Nomor berjalan pertama 000. (c) Nomor pertama menjadi master data.
- **Rekomendasi Claude Code (usulan):** (a). Sama dengan contoh di milestone dan U1, dan sejalan dengan A001 di atas 999. Nomor lama 000 tidak hilang, karena impor menyimpan teks aslinya.
- **Asumsi sementara di kode (ASUMSI E-44):** (a), di `internal/numbering/sequence.go`.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-45 · Q-50 mengasumsikan satu QA QC Manager, kode M1.8 membolehkan beberapa

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** brief Q-50 ("The QA QC Manager sits outside the NPI chart. One holder is assumed."), E-38, `sql/migrations/00008_create_positions_and_routes.sql`.
- **Temuan:** asumsi kerja Q-50 sudah menyebut satu pemegang untuk QA QC Manager, sedangkan asumsi M1.8 (E-38 opsi a) membolehkan beberapa pemegang untuk semua jabatan selain Project Leader dan Project Engineer.
- **Dampak bila dibiarkan:** kode dan brief berbeda tentang jumlah pemegang jabatan approver.
- **Opsi:** diselesaikan bersama E-38.
- **Rekomendasi Claude Code (usulan):** E-38 opsi (b), yang sejalan dengan Q-50.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan bersama E-38 (D-115).
- **Keputusan pemilik:** E-38 opsi (b).

### E-46 · U1 menyebut permintaan perubahan diajukan PPAP Engineer saja

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** U1 bagian 3 (Permintaan perubahan), UC-40, D-63, D-67, hak E-3.
- **Temuan:** U1 menulis permintaan perubahan "diajukan manual oleh PPAP Engineer". UC-40 dan hak E-3 memberikannya ke semua Penyusun: PPAP Engineer, PPAP Staff, Project Leader, dan Project Engineer.
- **Dampak bila dibiarkan:** tim IT bisa membatasi pengajuan perubahan ke satu jabatan.
- **Opsi:** (a) U1 menulis "oleh Penyusun".
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di U1 (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-47 · Nama peran di U1 bagian 7 berbeda dengan U3

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** U1 bagian 7, U3-peta-pengguna bagian 1 dan 2, E-01.
- **Temuan:** U1 bagian 7 mencampur nama peran Inggris (verifier, document control, admin, viewer) dengan nama Indonesia, masih memuat peran approver yang menurut E-01 belum dibuat, dan tidak memuat peran Petugas migrasi.
- **Dampak bila dibiarkan:** glosarium dan peta pengguna menyebut peran yang berbeda.
- **Opsi:** (a) U1 bagian 7 memakai delapan peran U3, dengan nama Inggris sebagai istilah layar (E-19), dan approver dicatat sebagai peran yang belum dibuat.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di U1 (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-48 · Glosarium brief tidak memuat jabatan PPAP Staff

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** brief bagian 3 (Position), brief bagian 1, D-63, D-72, M1.8.
- **Temuan:** daftar jabatan di glosarium brief tidak memuat PPAP Staff, padahal jabatan itu ada di bagian 1, D-63, D-72, M1.8, U3, dan master jabatan di kode.
- **Dampak bila dibiarkan:** glosarium tidak lengkap untuk jabatan yang menyusun dokumen dan memegang akun admin.
- **Opsi:** (a) Tambahkan PPAP Staff ke glosarium brief.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di brief v1.16.
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-49 · Model domain brief tertinggal dari keputusan dan kode

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** brief bagian 4, E-02, E-05, E-11, E-12, D-54, U5, U8.
- **Temuan:** tabel entitas brief tidak punya entitas Project Group sendiri, tidak membedakan orang dan akun (E-02), dan tidak memuat riwayat kode customer, kode pabrik, kode khusus, atau pemegang jabatan. Diagram ER-nya tidak memuat group model, jabatan, rute approval, peran, atau audit log.
- **Dampak bila dibiarkan:** brief tidak bisa dipakai sebagai gambaran model data yang sudah dibangun.
- **Opsi:** (a) Bagian 4 diperbarui mengikuti kode, dengan rincian tabel di U5 dan U8.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di brief v1.16 bagian 4, U5, dan U8.
- **Keputusan pemilik:** tidak perlu (perbaikan teks, mengikuti keputusan yang sudah ada).

### E-50 · UC-51 berstatus MVP, padahal M5.5a masih usulan

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Ambigu
- **Rujukan:** U3 UC-51, D-30, R-REV-06, milestone M5.5a (Tambah usulan).
- **Temuan:** UC-51 (Catat dokumen dikembalikan) berstatus MVP, karena sumbernya D-30 sudah diputuskan. Milestone-nya M5.5a, yang masih "Tambah (usulan)". Bila M5.5a ditolak, UC-51 tidak punya tempat di milestone.
- **Dampak bila dibiarkan:** use case MVP bisa tidak pernah dibangun.
- **Opsi:** (a) Kolom milestone UC-51 ditulis "M5.5a (usulan)" dengan catatan bahwa D-30 tetap berlaku bila M5.5a ditolak, sehingga UC-51 pindah ke M5.4.
- **Rekomendasi Claude Code (usulan):** (a). Keputusan atas M5.5a tetap di sesi desain.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di U3 (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-51 · UC-57 dijadwalkan di M1.2, padahal dokumen rilis baru ada di Tahap 5

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** U3 UC-57, hak L-1, M5.5, M6.7.
- **Temuan:** UC-57 (Baca dokumen rilis) dijadwalkan di M1.2. Di M1.2 hanya haknya (L-1) yang dibuat; dokumen rilis baru ada setelah snapshot rilis (M5.5) dan pencatatan rilis (M6.7).
- **Dampak bila dibiarkan:** matriks keterlacakan (U13) menunjuk use case ke milestone yang tidak bisa membuktikannya.
- **Opsi:** (a) Kolom milestone UC-57 ditulis "M1.2 (hak), M5.5".
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di U3 (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-52 · Nilai aturan login E-08 belum dikonfirmasi, dan 10 penanda ASUMSI tersisa untuk temuan yang sudah diputuskan

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** E-08 (bagian Pelaksanaan), E-02, E-09, E-12, catatan M1.1 bagian 7, `docs/catatan/Tahap-1-evaluasi.md` bagian 4.
- **Temuan:** catatan M1.1 menyebut nilai aturan login E-08 "menunggu konfirmasi pemilik", tapi lembar evaluasi Tahap 1 tidak menanyakannya. Di kode tersisa 10 penanda `ASUMSI` untuk temuan yang sudah diputuskan: 7 untuk E-08, dan masing-masing satu untuk E-02 (nomor karyawan paling banyak 50 karakter), E-09 (Project Group boleh kosong untuk customer lama), dan E-12 (ZZ bukan kode customer).
- **Dampak bila dibiarkan:** nilai keamanan login dipakai tanpa pernah dikonfirmasi, dan penanda ASUMSI tidak lagi menunjukkan apa yang benar-benar terbuka.
- **Opsi:** (a) Nilai E-08 dan rincian keempat penanda dikonfirmasi apa adanya, lalu penanda diganti keterangan keputusan. (b) Pemilik meninjau nilai satu per satu.
- **Rekomendasi Claude Code (usulan):** (a). Nilainya diambil dari OWASP, ASVS 5.0, dan NIST SP 800-63B-4, dengan sumber di tabel E-08. Perubahan E-22 opsi (c) (akun admin tidak dinonaktifkan pada kegagalan ke-100) berlaku di atasnya.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan pemilik 30 Oktober 2026 (D-85), lewat persetujuan umum atas semua saran Claude Code. Penanda diganti 30 Oktober 2026.
- **Keputusan pemilik:** Opsi (a).

### E-53 · Tanggal rilis PostgreSQL 19 RC1 berbeda antara brief dan versions.md

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** brief bagian 7, `docs/versions.md`, E-13.
- **Temuan:** brief menulis PostgreSQL 19 RC 1 dirilis 2026-10-15, sedangkan `docs/versions.md` menulis 2026-10-12 (tanggal tag `REL_19_RC1`). Halaman rilis postgresql.org tidak bisa dibuka dari cloud, jadi belum diverifikasi mana yang tanggal pengumuman resmi; tag git dan pengumuman memang bisa berbeda hari.
- **Dampak bila dibiarkan:** kecil; PostgreSQL 19 tidak dipakai.
- **Opsi:** (a) Brief bagian 7 hanya menyebut line, sesuai E-13 opsi (b), dan `docs/versions.md` mencatat kedua tanggal beserta sumbernya.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di brief v1.16 dan `docs/versions.md`.
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-54 · Tanggal di kepala milestone tidak sesuai dengan tanggal brief v1.15

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** `docs/milestone/milestone.md` baris 1 dan 5.
- **Temuan:** baris 1 menulis "sinkron dengan brief v1.15, 27 Oktober 2026", sedangkan baris 5 menulis "diperbarui 23 Oktober 2026 (sinkron dengan brief v1.15)". Brief v1.15 baru terbit 27 Oktober 2026.
- **Dampak bila dibiarkan:** kecil; pembaca bisa salah menilai versi milestone.
- **Opsi:** (a) Diperbaiki di docx sumber pada sesi desain berikutnya, lalu diekspor ulang.
- **Rekomendasi Claude Code (usulan):** (a). File md adalah ekspor dan tidak diedit di repo (CLAUDE.md bagian 2).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Ditunda ke sesi desain (docx).
- **Keputusan pemilik:** belum.

### E-55 · Perintah memasang browser Playwright berbeda antara CLAUDE.md dan versions.md

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Kontradiksi
- **Rujukan:** CLAUDE.md bagian 8, `docs/versions.md` (baris Playwright), catatan M0.5.
- **Temuan:** CLAUDE.md dan catatan M0.5 memakai `npm --prefix web exec -- playwright install chromium`, sedangkan `docs/versions.md` memakai `npx --prefix web playwright install chromium`.
- **Dampak bila dibiarkan:** pemilik bisa memakai perintah yang belum pernah dicoba.
- **Opsi:** (a) `docs/versions.md` memakai perintah di CLAUDE.md.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di `docs/versions.md`.
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-56 · E-41 menyebut M1.7 yang belum dibangun

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Ambigu
- **Rujukan:** E-41.
- **Temuan:** temuan E-41 menulis "Sampai M1.7, fungsi izin meloloskan...". Yang dimaksud adalah perilaku sebelum M1.8; M1.7 belum dibangun saat itu.
- **Dampak bila dibiarkan:** kecil.
- **Opsi:** (a) Teks diganti "Sebelum M1.8".
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki.
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-57 · Kolom baris UC-02 di daftar use case bergeser

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** `docs/uml/U3-daftar-use-case.md` (UC-02).
- **Temuan:** baris UC-02 menulis ringkasan dua kali, sehingga ID rujukan pindah ke kolom Milestone dan kolom Rujukan berisi teks.
- **Dampak bila dibiarkan:** matriks keterlacakan (U13) yang dibuat dari tabel ini salah membaca rujukan UC-02.
- **Opsi:** (a) Baris disusun ulang: ringkasan, rujukan (R-SEC-10, D-66), lalu milestone (M1.2, M1.8).
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di U3 (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-58 · README UML usang (U10 dan pertanyaan terbuka)

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** `docs/uml/README.md`, `api/openapi.yaml`, E-19, E-24, E-25, E-41.
- **Temuan:** README UML masih menulis U10 (file OpenAPI) "Belum", padahal `api/openapi.yaml` sudah berisi 60 operasi, dan menulis "Tidak ada pertanyaan terbuka", padahal E-19, E-24, E-25, dan E-41 menyentuh U1 dan U3.
- **Dampak bila dibiarkan:** tim IT tidak tahu bahwa kontrak API sudah ada.
- **Opsi:** (a) README menunjuk `api/openapi.yaml` untuk U10, dan bagian pertanyaan menunjuk temuan terbuka di `docs/evaluasi-desain.md`.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki di README UML (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-59 · README di root repo hampir kosong

- **Tanggal dan milestone:** 30 Oktober 2026, pemeriksaan dokumen.
- **Jenis:** Celah
- **Rujukan:** `README.md`, CLAUDE.md, brief bagian 13 (README spec lama diperbarui di S1).
- **Temuan:** README di root hanya berisi dua baris, tanpa tujuan proyek, cara menjalankan, atau rujukan ke CLAUDE.md dan `docs/`.
- **Dampak bila dibiarkan:** orang yang membuka repo tidak tahu harus mulai dari mana.
- **Opsi:** (a) README singkat: tujuan, status, dan rujukan ke CLAUDE.md (perintah) serta `docs/` (brief, milestone, UML, catatan).
- **Rekomendasi Claude Code (usulan):** (a). Perintah tidak disalin ke README, supaya hanya ada satu sumber (CLAUDE.md bagian 8).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Diperbaiki (30 Oktober 2026).
- **Keputusan pemilik:** tidak perlu (perbaikan teks).

### E-60 · Jadwal, retensi, dan penyimpanan passphrase backup belum ditetapkan

- **Tanggal dan milestone:** 30 Oktober 2026, M1.7. Berdampak ke M8.2 dan dokumen IT (M8.4).
- **Jenis:** Celah
- **Rujukan:** D-119, M1.7, M8.2, R-SEC-04, RK-23, RK-49, Q-18.
- **Temuan:** M1.7 meminta backup harian terenkripsi ke disk lain. Belum ada aturan untuk tiga hal: (1) apa yang menjalankan backup setiap hari, aplikasi sendiri atau penjadwal sistem operasi; (2) berapa lama dan berapa banyak backup disimpan (Q-18 hanya membahas revisi dan lampiran); (3) di mana passphrase disimpan: di `.env` bersama password database, sehingga backup bisa berjalan tanpa orang, atau hanya diketik saat backup.
- **Dampak bila dibiarkan:** backup menumpuk tanpa batas di disk tujuan, atau backup harian berhenti tanpa ada yang tahu.
- **Opsi:** (a) Penjadwal sistem operasi (Task Scheduler di Windows, cron di Linux) menjalankan `revioten backup`; backup lama dihapus tangan oleh pemilik; passphrase di `.env`, dengan salinan di tempat lain. (b) Server menjalankan backup sendiri lewat job queue (River) dan membuang backup yang lebih tua dari N hari. (c) Seperti (a), ditambah perintah untuk membuang backup lama.
- **Rekomendasi Claude Code (usulan):** (a) untuk demo localhost. Opsi (b) dibahas di dokumen IT (M8.4), karena versi IT memakai sistem backup perusahaan.
- **Asumsi sementara di kode (ASUMSI E-60):** (a), di `internal/backup/backup.go`. Langkah Task Scheduler ditulis di catatan M1.7 dan belum dicoba di komputer pemilik.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-61 · Satu pola nomor untuk semua jenis dokumen dan customer

- **Tanggal dan milestone:** 30 Oktober 2026, M2.1. Berdampak ke M2.5 (impor nomor lama).
- **Jenis:** Ambigu
- **Rujukan:** brief bagian 4 (numbering_pattern: "Pattern per document type, optionally overridden per customer"), R-NUM-02, Q-05, E-12, D-89.
- **Temuan:** model domain brief menyebut pola per jenis dokumen yang bisa ditimpa per customer. WI SW-ZZ-000 (D-89) memberi satu struktur untuk semua jenis, dan Q-05 masih menanyakan apakah ada customer dengan pola khusus. Selain itu, impor nomor lama (M2.5) harus bisa membaca teks nomor kembali menjadi bagiannya, padahal pola bebas seperti `{prefix}{customer}{seq}` bisa dibaca dua cara (KT + A001 atau KTA + 001).
- **Dampak bila dibiarkan:** pola per customer butuh layar dan aturan tambahan, dan pola yang tidak bisa dibaca kembali membuat impor salah membaca nomor.
- **Opsi:** (a) Satu pola untuk semua jenis dan customer; token `{prefix}` sudah membedakan jenis. Pola wajib memisahkan token dengan tanda baca. (b) Satu pola per jenis dokumen. (c) Pola per jenis, ditambah pola khusus per customer.
- **Rekomendasi Claude Code (usulan):** (a) sampai Q-05 menunjukkan customer dengan pola khusus. Satu pola juga menjaga nomor satu set tetap sejajar bentuknya.
- **Asumsi sementara di kode (ASUMSI E-61):** (a), di `sql/migrations/00010_create_document_numbers.sql` dan `internal/numbering/format.go`.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-62 · Isi register dan pengingat nomor Reserved yang terlalu lama

- **Tanggal dan milestone:** 30 Oktober 2026, M2.3. Berdampak ke M4.1 (dokumen menunjuk nomor).
- **Jenis:** Celah
- **Rujukan:** R-NUM-04 ("Stale Reserved numbers raise reminders. Search, filter, and export the register as a master document list"), brief bagian 4 (document_number: number text, status, owner, linked document), UC-12, UC-15.
- **Temuan:** brief belum menyebut kolom register selain teks nomor, status, pemilik, dan dokumen tertaut. Nomor Reserved belum punya dokumen, jadi register tidak bisa menunjukkan model atau judul yang dimaksud. R-NUM-04 juga meminta pengingat untuk nomor Reserved yang terlalu lama, tanpa ambang (berapa hari) dan tanpa cara (layar, email, atau job terjadwal). Ekspor master document list belum punya format.
- **Dampak bila dibiarkan:** register nomor Reserved hanya bisa dikenali dari pendaftarnya, dan nomor yang terlupa menumpuk.
- **Opsi:** (a) Seperti sekarang: register memuat nomor, customer, jenis, status, pendaftar, waktu daftar, dan alasan; umur nomor Reserved ditampilkan dalam hari; model dan judul datang dari dokumen di M4.1; pengingat dan ekspor menunggu ambang dan format dari pemilik. (b) Tambah kolom catatan bebas (misalnya nama model) saat mendaftar. (c) Ambang tetap (misalnya 30 hari) dan pengingat lewat job terjadwal (River).
- **Rekomendasi Claude Code (usulan):** (a) sekarang. Ambang pengingat dan format ekspor ditanyakan ke pemilik, karena angkanya tidak boleh dikarang.
- **Asumsi sementara di kode (ASUMSI E-62):** (a), di `web/src/lib/numbers.ts` (umur nomor Reserved). UC-15 (pengingat) dan ekspor UC-12 belum dibangun.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-63 · Kunci idempoten di UI API: cakupan, umur, dan pemakaian ulang

- **Tanggal dan milestone:** 30 Oktober 2026, M2.2. Berdampak ke M9.4 (API integrasi).
- **Jenis:** Celah
- **Rujukan:** R-NUM-03, R-API-10 ("kept for a defined period"), R-API-18.
- **Temuan:** R-NUM-03 meminta kunci idempoten untuk pendaftaran nomor, tapi tidak menyebut siapa yang membuat kunci, cakupannya (per akun atau untuk seluruh sistem), berapa lama disimpan, dan jawaban bila kunci yang sama dikirim dengan isi berbeda. R-API-10 untuk API integrasi juga belum menyebut lamanya.
- **Dampak bila dibiarkan:** setiap endpoint menulis aturan idempotensinya sendiri.
- **Opsi:** (a) Kunci dibuat klien untuk setiap pendaftaran baru (layar memakai UUID), berlaku per akun, dan disimpan selamanya bersama catatan permintaan. Kunci yang sama dengan isi yang sama mendapat hasil lama (200); dengan isi berbeda ditolak (422, `idempotency_key:taken`). (b) Seperti (a), tapi kunci dibuang setelah N hari. (c) Kunci berlaku untuk seluruh sistem.
- **Rekomendasi Claude Code (usulan):** (a) untuk UI API, karena jumlah pendaftaran kecil dan catatannya berguna untuk audit. Lama simpan untuk API integrasi ditetapkan di M9.4.
- **Asumsi sementara di kode (ASUMSI E-63):** (a), di `internal/numbering/register.go` dan tabel `number_batches`.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-64 · Nomor yang dilewati saat daftar sebagai set

- **Tanggal dan milestone:** 30 Oktober 2026, M2.4.
- **Jenis:** Kontradiksi
- **Rujukan:** R-NUM-03 (tanpa lubang), R-NUM-05, Q-05, milestone M2.4 (usulan: status Dilewati, atau satu penghitung per customer untuk set).
- **Temuan:** R-NUM-05 mengambil nomor bebas berikutnya di ketiga jenis, sehingga penghitung yang tertinggal melompati nomor, padahal R-NUM-03 menuntut tanpa lubang. Brief dan milestone sudah menyebut masalah ini tanpa memutuskannya.
- **Dampak bila dibiarkan:** register berlubang, dan laporan lubang di impor (M2.5) tidak bisa membedakan nomor yang hilang dari nomor yang sengaja dilewati.
- **Opsi:** (a) Nomor yang dilewati dicatat dengan status Skipped (usulan pertama milestone, istilah "Dilewati"); tidak pernah diberikan lagi. (b) Satu penghitung per customer untuk set, sehingga ketiga jenis selalu sejajar. (c) Set hanya boleh bila ketiga penghitung sudah sejajar.
- **Rekomendasi Claude Code (usulan):** (a). Register tetap tanpa lubang, penghitung per jenis tetap sesuai R-NUM-03 dan D-93, dan nomor tunggal (yang masih dipakai untuk register lama yang tidak sejajar) tetap bisa didaftarkan. Opsi (b) mengubah penghitung yang sudah diputuskan, dan opsi (c) membuat set tidak pernah bisa dipakai untuk customer dengan register lama yang tidak sejajar.
- **Asumsi sementara di kode (ASUMSI E-64):** (a), di `internal/numbering/set.go` dan status `skipped` di `document_numbers`. Di layar, istilahnya "Skipped" (bahasa layar Inggris, D-96).
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-65 · Bentuk impor nomor lama dan isi laporannya

- **Tanggal dan milestone:** 30 Oktober 2026, M2.5.
- **Jenis:** Ambigu
- **Rujukan:** R-NUM-06, UC-14, milestone M2.5, R-MIG-02 (tempel dari Excel untuk dokumen), RK-24.
- **Temuan:** brief dan milestone menyebut apa yang dilaporkan (lubang, nomor ganda, set tidak sejajar), tapi tidak menyebut bentuk masukan impor: file atau tempel, kolom apa saja, status nomor lama, revisi awalnya, dan batas jumlahnya. Arti "set tidak sejajar" juga belum ditulis, termasuk apakah nomor Skipped (E-64) dihitung.
- **Dampak bila dibiarkan:** kolom lain di register Excel (status, model, revisi, tanggal) tidak ikut masuk, dan laporan bisa dibaca berbeda oleh orang yang berbeda.
- **Opsi:** (a) Tempel kolom nomor saja; satu status untuk satu impor (Reserved, In use, Released, Obsolete, atau Void), tanpa alasan status; revisi awal 0; paling banyak 5000 baris per impor. "Tidak sejajar" berarti nomor berjalan yang hanya dimiliki sebagian dari PFD, PFMEA, dan CP, dengan Skipped dianggap tidak ada. (b) Tempel beberapa kolom dengan langkah pemetaan kolom (nomor, status, model, revisi), seperti R-MIG-02. (c) Unggah file register Excel (excelize sudah ada di stack).
- **Rekomendasi Claude Code (usulan):** (a) untuk sekarang. Pilihan (b) dikerjakan setelah pemilik memeriksa bentuk register Excel yang asli, karena kolomnya belum diverifikasi dan tidak boleh dikarang.
- **Asumsi sementara di kode (ASUMSI E-65):** (a), di `internal/numbering/import.go`. Kolom pertama setiap baris yang dibaca (Excel memisahkan kolom dengan tab), baris kosong dilewati, dan pratinjau (dry run) memberi laporan yang sama tanpa menyimpan apa pun. Laporan mencakup seluruh register customer itu setelah impor, bukan hanya baris yang ditempel, supaya impor bertahap (misalnya per status) tetap memberi lubang yang benar.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-66 · Nomor lama yang tidak bisa masuk apa adanya: bentuk lain dan nomor ganda

- **Tanggal dan milestone:** 30 Oktober 2026, M2.5.
- **Jenis:** Celah
- **Rujukan:** R-NUM-06 (nomor lama masuk dengan teks aslinya; nomor ganda dilaporkan dan tidak pernah diperbaiki), R-NUM-02, R-POR-01, D-89, E-61, milestone M4.1 (header dokumen terhubung ke nomor register), M7.1 (form dokumen lama dengan nomor asli).
- **Temuan:** dua jenis nomor lama tidak punya tempat di register. (1) Nomor dengan bentuk lain dari pola sekarang, misalnya SC-KT-01 (dua digit) atau pola lama tanpa pemisah, tidak bisa dibaca menjadi kode customer, jenis dokumen, dan nomor berjalan, jadi penghitung tidak tahu nomor itu ada. (2) Nomor ganda di register lama (dua dokumen dengan nomor yang sama): register menyimpan setiap nomor sekali (UNIQUE), dan di M4.1 header dokumen menunjuk satu nomor, jadi dokumen kedua tidak punya nomor sendiri.
- **Dampak bila dibiarkan:** nomor baru bisa sama maknanya dengan nomor lama di luar pola (SC-KT-001 dan SC-KT-01), dan dokumen kedua dari nomor ganda tidak bisa dimigrasi di Tahap 7 tanpa keputusan.
- **Opsi:** (a) Hanya dilaporkan (Invalid dan Duplicate) dan tidak masuk; dokumen kontrol memutuskan di luar sistem. (b) Untuk bentuk lain: impor bisa memakai pola lama dan bentuk nomor berjalan lama yang dipilih per impor, sehingga kode, jenis, dan nomor berjalan tetap terbaca dan teks asli tetap disimpan. (c) Untuk nomor ganda: register tetap unik, tetapi satu nomor boleh ditunjuk lebih dari satu dokumen di M4.1, dengan tanda di register.
- **Rekomendasi Claude Code (usulan):** (a) untuk sekarang, karena register asli belum diperiksa. Sebelum M4.1 dan Tahap 7, pemilik memeriksa register Excel: bila ada nomor lama di luar pola, pilih (b); bila ada nomor ganda yang keduanya masih berlaku, pilih (c).
- **Asumsi sementara di kode (ASUMSI E-66):** (a), di `internal/numbering/import.go`. Baris di luar pola sekarang menjadi Invalid; nomor ganda masuk sekali (baris pertama), dan baris berikutnya dilaporkan Duplicate dengan nomor baris pertamanya.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-67 · Pendaftaran nomor untuk customer nonaktif

- **Tanggal dan milestone:** 30 Oktober 2026, M2.1.
- **Jenis:** Celah
- **Rujukan:** E-09 (D-86), R-NUM-01, UC-10.
- **Temuan:** E-09 menetapkan customer nonaktif tidak mendapat group model, model, atau part number baru. Belum ada aturan untuk nomor dokumen baru.
- **Dampak bila dibiarkan:** nomor bisa didaftarkan untuk customer yang sudah tidak dilayani.
- **Opsi:** (a) Ditolak, sama seperti model baru; customer bisa diaktifkan lagi bila perlu. (b) Diizinkan.
- **Rekomendasi Claude Code (usulan):** (a). Revisi dokumen lama tidak butuh nomor baru, jadi pekerjaan yang sedang berjalan tidak terhenti.
- **Asumsi sementara di kode (ASUMSI E-67):** (a), di `internal/numbering/register.go`. Layar Numbers hanya menawarkan customer aktif. Impor nomor lama (M2.5) juga menolak customer nonaktif, dengan aturan dan pemeriksaan yang sama.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-68 · Perubahan status nomor yang dikerjakan tangan di Tahap 2, dan alasannya

- **Tanggal dan milestone:** 30 Oktober 2026, M2.3. Berdampak ke M4.1 (In use) dan M6.7 (Released).
- **Jenis:** Ambigu
- **Rujukan:** R-NUM-04, brief bagian 4 (diagram status nomor), UC-13, E-8, D-93.
- **Temuan:** diagram status nomor menyebut Reserved ke In use saat dokumen dibuat, In use ke Released saat rilis dicatat, Released ke Obsolete saat ditarik, dan Reserved atau In use ke Void "cancelled with reason". Di Tahap 2 belum ada dokumen dan rilis, jadi yang bisa dikerjakan tangan hanya Void dan Obsolete. Belum jelas apakah Obsolete juga wajib beralasan, dan apakah status lain (misalnya Released untuk nomor lama) boleh dipasang tangan.
- **Dampak bila dibiarkan:** nomor lama yang sudah rilis di Excel tidak bisa ditandai Released, kecuali lewat impor (M2.5).
- **Opsi:** (a) Tahap 2 hanya membuka Void (dari Reserved atau In use, alasan wajib) dan Obsolete (dari Released, alasan opsional). Status lain dipasang sistem (M4.1, M6.7) atau lewat impor (M2.5). (b) Seperti (a), dengan alasan wajib untuk Obsolete juga. (c) Dokumen kontrol boleh memasang status apa pun dengan alasan.
- **Rekomendasi Claude Code (usulan):** (a). Diagram status hanya mewajibkan alasan untuk Void, dan status lain mengikuti peristiwa nyata (dokumen dibuat, rilis dicatat).
- **Asumsi sementara di kode (ASUMSI E-68):** (a), di `internal/numbering/registry.go`.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-69 · Urutan kunci transaksi penomoran: deadlock dengan ganti kode customer

- **Tanggal dan milestone:** 30 Oktober 2026, saat menyiapkan Gerbang 2 (kode M2.1, M2.2, dan M2.4).
- **Jenis:** Sulit dibangun
- **Rujukan:** R-POR-01 (kode di nomor adalah kode customer saat nomor dibuat), R-NUM-03, D-88, D-93, U8.
- **Temuan:** test baru `TestRegisterDuringCodeChange` menemukan deadlock antara pendaftaran nomor (satu nomor dan set) dan ganti kode customer. Sebelum diperbaiki, test ini gagal 10 kali dari 10. Urutan kejadiannya:
  1. Pendaftaran menyimpan batch lebih dulu. Foreign key `number_batches.customer_id` memasang kunci KEY SHARE pada baris customer.
  2. Ganti kode masuk dan memegang FOR NO KEY UPDATE, yang tidak bentrok dengan KEY SHARE.
  3. Pendaftaran meminta FOR SHARE, lalu menunggu ganti kode.
  4. Ganti kode mengubah kolom `code` yang unik. Untuk itu ia butuh kunci penuh, jadi ia menunggu KEY SHARE milik pendaftaran.

  Brief dan UML belum menyebut urutan kunci. Tim IT akan bertemu masalah yang sama dengan stack apa pun yang memakai foreign key dan penguncian baris.
- **Dampak bila dibiarkan:** sesekali pendaftaran atau ganti kode gagal dengan error 500 bila keduanya berjalan bersamaan. Data tidak rusak, karena PostgreSQL membatalkan salah satu transaksi, tetapi pengguna harus mengulang.
- **Opsi:**
  - (a) Satu urutan kunci untuk semua transaksi penomoran, ditulis di brief (catatan R-NUM-03) dan U8. Urutannya: customer dikunci FOR SHARE paling awal, sebelum INSERT apa pun yang merujuk customer itu; lalu penghitung menurut kode customer dan ID jenis dokumen; lalu kunci rantai audit paling akhir.
  - (b) Seperti (a), ditambah ganti kode mengunci customer dengan FOR UPDATE sejak awal, supaya pola serupa di operasi lain juga aman.
  - (c) Pendaftaran diulang otomatis saat deadlock.
- **Rekomendasi Claude Code (usulan):** (a), dan sudah dipasang di kode. Pilihan (b) layak dipertimbangkan bila operasi lain mulai mengunci customer. Pilihan (c) hanya menyembunyikan masalah, jadi tidak dipakai.
- **Di kode sekarang:** (a), di `customerCodeForNumber` (`internal/numbering/register.go`). Fungsi itu kini langkah pertama `Register` dan `RegisterSet`; `Import` sudah begitu sejak M2.5. Bukti: `TestRegisterDuringCodeChange` lulus 30 kali berturut-turut (300 putaran).
- **Menghambat milestone?** Tidak (sudah diperbaiki di kode)
- **Status:** Terbuka (menunggu konfirmasi aturan urutan kunci di brief dan U8)
- **Keputusan pemilik:**

### E-70 · Isi awal format template dan aturan field yang belum ditetapkan

- **Tanggal dan milestone:** 31 Oktober 2026, M3.1. Berdampak ke M4.1 (baris dokumen) dan Tahap 6 (cetakan).
- **Jenis:** Celah
- **Rujukan:** R-TPL-01, R-TPL-02, R-TPL-11, R-TPL-12, R-DOC-01, R-DOC-04, R-DOC-09, R-CMP-01, F-09, F-10, F-11, usulan M3.1, UC-20.
- **Temuan:** brief menetapkan isi format template: kolom ber-ID tetap, metode risiko, tabel skala, field header, dan simbol. Yang belum ditetapkan:
  1. Kolom dan field header mana yang wajib diisi.
  2. Teks kriteria tabel skala severity, occurrence, dan detection. Form yang dipakai punya teksnya, tetapi teks manual AIAG tidak boleh disalin begitu saja.
  3. Apakah proses dan nomor langkah termasuk field template, atau atribut baris (R-DOC-04, R-CMP-01).
  4. Metodologi untuk PFD dan CP, karena AIAG 4th dan AIAG-VDA adalah buku FMEA.
  5. Batas panjang label, teks sel, dan kriteria.
  6. Apakah satu sel boleh berisi lebih dari satu simbol.
  7. Apakah metode risiko boleh diubah setelah versi dibuat.
  8. Apakah pilihan field select dan simbol boleh dihapus.
- **Dampak bila dibiarkan:**
  - isi awal bisa berbeda dari form asli;
  - kolom wajib yang ditambah belakangan menandai banyak dokumen "belum lengkap" di Tahap 4;
  - teks kriteria tercetak kosong.
- **Opsi:**
  - (a) Asumsi di bawah dipakai, lalu diganti setelah pemilik membandingkannya dengan form asli.
  - (b) Pemilik menetapkan daftar kolom, field wajib, dan teks kriteria dari form PFD, PFMEA, dan CP asli sebelum M4.1. Teks cukup diketik lewat layar Format templates, tanpa menyimpan form-nya di repo.
- **Rekomendasi Claude Code (usulan):** (b) sebelum M4.1, karena bahannya ada di form asli. Sampai saat itu (a).
- **Asumsi sementara di kode (ASUMSI E-70):**
  - Kolom dan field header diambil dari F-09 (PFD), F-10 dengan kolom standar AIAG 4th (PFMEA), dan F-11 (CP). Labelnya bahasa Inggris sesuai judul kolom, dan semuanya tidak wajib.
  - Teks kriteria kosong.
  - Proses dan nomor langkah adalah atribut baris, bukan field.
  - Metodologi kosong untuk PFD dan CP.
  - Batas: label 100 karakter, teks sel 2000, kriteria 500, paling banyak 50 pilihan per field dan 20 simbol per versi.
  - Satu simbol per sel.
  - Metode risiko tetap sejak versi dibuat; area dan jenis field juga tetap.
  - Pilihan dan simbol tidak bisa dihapus, hanya diganti nama.
  - Template SQAM hanya untuk customer aktif.

  Semua ini ada di `sql/migrations/00011_create_format_templates.sql` dan `internal/template/template.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M4.1)
- **Status:** Diputuskan sebagian pemilik 4 November 2026 (D-124)
- **Bukti baru (tinjauan desain UI, 3 November 2026):** artboard "PFD sheet", "PFMEA sheet", "Control Plan sheet", dan "Cover" di kanvas desain UI menggambar form yang dipakai sekarang ("current form layout"). Perbedaannya dengan isi awal M3.1:
  1. CP, kolom Process Flow: dibentuk dari langkah PFD (R-DOC-09), bukan isian. **Sudah dibetulkan** (migrasi 00013: disembunyikan di isi awal).
  2. PFMEA, kolom Process Step / Function: berisi nama proses baris itu, yang sudah menjadi atribut baris. **Sudah dibetulkan** (migrasi 00013: disembunyikan di isi awal).
  3. CP, karakteristik: form punya dua kolom Specification / Tolerance, satu untuk Product Characteristics dan satu untuk Process Characteristics, masing-masing dengan kolom No. Isi awal M3.1 hanya punya satu kolom Specification. Belum diubah.
  4. CP, kolom Machine, Equipment, Jigs, Fixture, Tools for Mfg: di form ditulis sekali per proses (bersama No dan Process Name), bukan per baris karakteristik. Isi awal M3.1 menaruhnya per baris dengan label "Machine and Tools". Belum diubah.
  5. Label di form memakai bentuk jamak: Product Characteristics, Process Characteristics, Special Characteristics. Belum diubah.
  6. PFMEA, Key Date tertulis "(at print)", jadi mungkin terisi saat dicetak, bukan diketik. Isi awal M3.1 menjadikannya field header tanggal. Belum diubah.
  7. Header CP tidak memuat Process or Line, tetapi cover CP memuatnya (R-PRT-07), jadi field header Process or Line tetap benar.
- **Rekomendasi Claude Code (usulan, diperbarui 3 November 2026):** butir 3 sampai 6 diputuskan pemilik dengan membandingkan artboard di atas dengan file Excel asli, lalu dibangun dalam satu migrasi sebelum M4.1. Saran untuk butir 4: kolom tingkat proses (Machine, Equipment, Jigs, Fixture, Tools for Mfg) disimpan sebagai field per baris yang dicetak sekali per proses, supaya baris tetap datar (R-DOC-01).
- **Keputusan pemilik:** susunan kolom di artboard "current form layout" sama dengan file Excel asli (D-124). Isi awal PFD, PFMEA, dan CP mengikutinya lewat migrasi 00014:
  - PFD: No, Process, Flow Symbol, NG Disposition;
  - PFMEA: Process Item No. dan Process Step / Function di depan, lalu label mengikuti form (Requirement, Class, Potential Cause(s) / Mechanism(s) of Failure, Recommended Action);
  - CP: Process Flow, No, Process Name / Operation Description, Machine, Equipment, Jigs, Fixture, Tools for Mfg, karakteristik produk dan proses masing-masing dengan Specification / Tolerance, lalu Special Characteristics sampai Reaction Plan.

  Nomor operasi, proses, dan Process Flow menjadi kolom terisi otomatis (D-122, D-123), menggantikan perbaikan migrasi 00013 yang menyembunyikannya. Masih asumsi (ASUMSI E-70): kolom wajib, teks kriteria, batas panjang, satu simbol per sel, Key Date tetap diketik sampai layout cetak (M6.1), dan kolom No karakteristik CP yang dinomori layout cetak per proses.

### E-71 · Aturan sorot PFMEA: arti "severity 8" dan nilai mana yang disorot

- **Tanggal dan milestone:** 31 Oktober 2026, M3.1.
- **Jenis:** Ambigu
- **Rujukan:** R-TPL-11, R-DOC-06, F-10.
- **Temuan:** form PFMEA sekarang menulis aturan aksi "RPN di atas 100 atau severity 8". Ada dua hal yang belum jelas:
  - apakah severity 8 berarti tepat 8 atau 8 ke atas;
  - apakah aturan ini berlaku juga untuk nilai hasil tindakan (Result Severity dan Result RPN).
- **Dampak bila dibiarkan:** sorotan di layar bisa lebih banyak atau lebih sedikit daripada cara tim membaca form.
- **Opsi:**
  - (a) Severity 8 ke atas, untuk nilai sekarang dan hasil tindakan.
  - (b) Severity 8 ke atas, hanya untuk nilai sekarang.
  - (c) Tepat 8. Kurang masuk akal, karena severity 9 dan 10 lebih berat.
- **Rekomendasi Claude Code (usulan):** (a). Sorotan pada hasil tindakan menunjukkan risiko yang masih tinggi sesudah tindakan. Ambangnya bisa diubah di layar, dan sorotan tidak pernah memblokir.
- **Asumsi sementara di kode (ASUMSI E-71):** (a), di `Highlights` (`internal/template/template.go`).
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Bukti baru (tinjauan desain UI, 3 November 2026):** artboard PFMEA menulis "Highlight RPN over 100, SEV 8 or more", sesuai ASUMSI E-71 untuk nilai sekarang, dan menambah satu aturan: warna sorot hanya di layar, tidak ikut tercetak. Penerapan untuk hasil tindakan belum terlihat di desain.
- **Keputusan pemilik:**

### E-72 · Satuan process template: satu per proses, atau satu pustaka untuk semua proses

- **Tanggal dan milestone:** 31 Oktober 2026, M3.2. Berdampak ke M3.3 (cakupan), M3.4 (versi), M5.8 (propagasi template), dan template_link (brief bagian 4).
- **Jenis:** Ambigu
- **Rujukan:** R-TPL-01, R-TPL-03, R-TPL-05, R-TPL-08, brief bagian 4 (process_template, template_link), U1 (Process template), M3.2.
- **Temuan:** brief menyebut process template sebagai pustaka proses, sub-proses, dan poin yang berversi, dengan cakupan umum, customer, group model, dan model. Satuannya belum jelas:
  1. satu template untuk satu proses (template IQC, template SMT, dan seterusnya); atau
  2. satu pustaka berisi semua proses untuk setiap cakupan.

  Satuan ini menentukan apa yang berversi dan diterbitkan (M3.4), apa yang ditautkan dokumen (template_link), dan seberapa luas satu propagasi (M5.8).
- **Dampak bila dibiarkan:** dengan satu pustaka untuk semua proses, perubahan kecil di IQC menerbitkan versi baru seluruh pustaka, dan semua dokumen yang mengikuti pustaka itu ikut diperiksa saat propagasi.
- **Opsi:**
  - (a) Satu template per proses per cakupan. Versi dan penerbitan per proses, dan dokumen menautkan template setiap proses yang dipakainya.
  - (b) Satu pustaka per cakupan berisi semua proses. Versi dan penerbitan untuk seluruh pustaka.
- **Rekomendasi Claude Code (usulan):** (a). Perubahan dan propagasi hanya menyentuh dokumen yang memakai proses itu, sehingga revisi yokoten lebih sempit. Opsi ini juga cocok dengan kriteria M3.2 (contoh IQC dengan lima sub-proses).
- **Asumsi sementara di kode (ASUMSI E-72):** (a), dengan cakupan umum saja di M3.2: `process_templates` punya kunci unik `process_id`. M3.3 mengganti kunci itu dengan kunci per proses dan cakupan. Ditandai di `sql/migrations/00012_create_process_templates.sql` dan `internal/template/process.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M3.3)
- **Status:** Terbuka
- **Bukti baru (tinjauan desain UI, 3 November 2026):** masukan pemilik di UX-01 (2 November 2026): "semua proses umum, poinnya bisa umum atau per customer, dan bisa diset saat membuat file". Artboard "Template library" (v2) juga menulis "Every process is general. The points inside a process can be general or belong to one customer, model group, or model." Artinya:
  - satuan template adalah proses, dan setiap proses hanya punya satu template (opsi a);
  - cakupan (umum, customer, group model, model) melekat pada poin, bukan pada template. Catatan di migrasi 00012 bahwa M3.3 mengganti kunci unik `process_id` dengan kunci per proses dan cakupan tidak berlaku lagi bila ini diputuskan: M3.3 menambah cakupan pada poin.
- **Rekomendasi Claude Code (usulan, diperbarui 3 November 2026):** (a), dengan cakupan per poin sesuai UX-01. Tidak perlu perubahan kode M3.2.
- **Keputusan pemilik (8 November 2026, D-129):** (a). Kutipan: "template dibuatkan per process, kemudian setiap point dalam process bisa diputuskan sebagai general, customer, group model atau model. kemudian ada juga informasi bahwa point tersebut biasanya digunakan untuk general atau template tertentu. dan user bisa memilih, walaupun biasanya digunakan untuk general, namun dirubah ke model karena ada sedikit penyesuaian." Contoh pemilik: proses laser marking punya poin laser type, content, dan location. Normalnya poin ini umum, tetapi laser type berbeda per model, jadi pemakaian sebenarnya bisa per model.

### E-73 · Isi poin process template dan format template yang diikutinya

- **Tanggal dan milestone:** 31 Oktober 2026, M3.2. Berdampak ke M4.5 (perakitan), M5.8 (propagasi), dan M7.4 (penautan baris).
- **Jenis:** Celah
- **Rujukan:** R-TPL-01, R-TPL-02, R-TPL-04, R-DOC-01, R-DOC-05, R-CMP-02, R-MIG-05, M4.5, M7.4, keputusan 4 di catatan M3.1 (salinan format template mendapat ID field baru).
- **Temuan:** brief menyebut poin sebagai satu butir setingkat baris dengan ID tetap (R-TPL-04), dan perakitan membuat baris dokumen dari poin (M4.5). Yang belum ditetapkan:
  1. Isi poin: hanya teks, atau isi baris lengkap (kolom PFD, PFMEA, atau CP) yang disalin ke dokumen saat dirakit.
  2. Bila isi baris lengkap: format template mana yang diikutinya. Isi baris disimpan per ID field (R-DOC-01), tetapi salinan format template mendapat ID field baru. Akibatnya isi yang ditulis untuk PFMEA AIAG 4th tidak otomatis cocok dengan format SQAM salinannya.
  3. Batas: panjang teks, banyak sub-proses per versi, dan banyak poin per sub-proses.
  4. Kapan tautan pasangan poin (misalnya PFMEA ke CP, R-TPL-04) dibuat di template.
- **Dampak bila dibiarkan:** M4.5 tidak bisa membuat baris yang terisi, dan propagasi (M5.8) tidak tahu kolom mana yang berubah.
- **Opsi:**
  - (a) Setiap versi process template menunjuk satu versi format template per jenis dokumen. Isi poin disimpan per ID field-nya dan diperiksa dengan `CheckContent` (M3.1).
  - (b) Seperti (a), ditambah asal field (ID field sumber) di salinan format template, supaya isi untuk AIAG 4th bisa dipetakan ke format SQAM salinannya.
  - (c) Poin hanya berisi teks; isi baris diketik di dokumen.
- **Rekomendasi Claude Code (usulan):** (b) sebelum M4.5. Opsi (a) memakai pemeriksa M3.1 yang sudah ada, dan asal field mencegah isi ditulis ulang untuk setiap format customer. Opsi (c) menghilangkan manfaat propagasi template untuk revisi yokoten.
- **Asumsi sementara di kode (ASUMSI E-73):**
  - Poin berisi jenis dokumen (PFD, PFMEA, atau CP) dan teks satu baris paling banyak 500 karakter.
  - Paling banyak 50 sub-proses per versi, dan 100 poin per sub-proses untuk satu jenis dokumen.
  - Tautan pasangan poin belum dibuat.

  Ditandai di `sql/migrations/00012_create_process_templates.sql` dan `internal/template/process.go`.
- **Menghambat milestone?** Tidak (perlu sebelum M4.5)
- **Status:** Terbuka
- **Bukti baru (tinjauan desain UI, 3 November 2026):**
  - UX-02 (masukan pemilik): sel CP boleh berisi gambar atau tabel spesifikasi kecil, di kolom dan proses mana pun, dan ikut tercetak di dalam sel. Satu baris teks tidak cukup untuk isi poin.
  - Artboard "Template items editor": poin PFMEA membawa kolom form sampai RPN, sedangkan "Recommended action and action results are filled per document". Poin juga punya tautan pasangan (kolom Linked CP dan Linked PFMEA).
  - Artboard "Template library" (v2): poin CP menampilkan karakteristik, spesifikasi, cakupan, dan tambahan (gambar atau simbol).
- **Rekomendasi Claude Code (usulan, diperbarui 3 November 2026):** (b), dengan isi poin memakai bentuk isi sel yang sama dengan baris dokumen (teks, gambar, atau tabel spesifikasi, UX-02). Kolom hasil tindakan tidak ikut di poin.
- **Keputusan pemilik:**

### E-74 · Kode customer tiga huruf di register (AJI, AVI) bertentangan dengan D-100

- **Tanggal dan milestone:** 3 November 2026, tinjauan desain UI dan UML draft 0.12 (UX-06). Berdampak ke M1.4 (kode customer), M2.1 (pola nomor), dan M2.5 (impor nomor lama).
- **Jenis:** Kontradiksi
- **Rujukan:** D-100, D-88, D-109, D-114, R-NUM-06, R-POR-01, E-11, E-66, UX-06.
- **Temuan:** UX-06 menemukan satu customer yang tercatat dengan dua kode di register Excel: AJ dan AJI, serta AV dan AVI (HL diganti MS adalah ganti kode biasa). D-100 membatasi kode customer dua huruf A sampai Z. Pola nomor membaca kode dua atau tiga huruf, karena kode khusus boleh tiga huruf (D-114). Akibatnya nomor SC-AJI-xxx terbaca dengan kode AJI. Saat impor untuk customer AJ (M2.5), nomor itu dilaporkan sebagai kode customer lain dan tidak diimpor, karena AJI tidak bisa dicatat sebagai kode lama AJ.
- **Dampak bila dibiarkan:** nomor yang mungkin sudah dipegang customer tidak bisa masuk register, padahal R-NUM-06 meminta nomor lama masuk apa adanya.
- **Opsi:**
  - (a) Kode customer boleh dua atau tiga huruf (D-100 diubah). AJI bisa dicatat sebagai kode lama atau kode berlaku customer yang sama. D-114 tetap menjaga kode khusus tidak pernah sama dengan kode customer.
  - (b) Kode customer tetap dua huruf. Nomor dengan kode tiga huruf masuk sebagai nomor lama (UX-06) yang ditautkan ke customer, tanpa penghitung.
  - (c) AJI dicatat sebagai customer terpisah.
- **Rekomendasi Claude Code (usulan):** (a), karena paling sedikit aturan baru: riwayat kode (D-88) dan foreign key kode di nomor (E-11) sudah menangani kode lama, dan nomor tetap masuk apa adanya. Sebelumnya perlu dipastikan bahwa AJ dan AJI memang customer yang sama, dan kode mana yang berlaku sekarang.
- **Asumsi sementara di kode:** tidak ada perubahan; kode customer tetap dua huruf (D-100).
- **Menghambat milestone?** Tidak (perlu sebelum register asli diimpor)
- **Status:** Diputuskan pemilik 4 November 2026 (D-125)
- **Keputusan pemilik:** "untuk saat ini ikuti WI hanya mengizinkan dua huruf". Kode customer tetap dua huruf (D-100 tidak berubah). Nomor dengan kode tiga huruf di register lama tetap dilaporkan impor dan tidak diimpor, untuk saat ini. Cara menyimpannya sebagai rujukan ikut keputusan UX-06.

### E-75 · Register terpisah PL1 dan PL3: plant, line, atau Project Leader (Q-19)

- **Tanggal dan milestone:** 3 November 2026, tinjauan desain UI dan UML draft 0.12 (UX-06). Berdampak ke M2.1 (penghitung) dan M2.5 (impor).
- **Jenis:** Ambigu
- **Rujukan:** Q-19, D-54, D-89, D-93, UX-06.
- **Temuan:** UX-06 membaca tiga register nomor: PL1, PL3, dan register IC/AVI/AJI/GW. Desain UI menulis "NPI · Plant 1" di bawah nama aplikasi. Q-19 masih terbuka (asumsi kerja: satu plant) dan bertanya apakah ada lebih dari satu plant atau line, serta apakah penomoran per plant. Belum jelas apakah PL1 dan PL3 berarti plant, line, atau Project Leader (Project Group GR1 dan GR3).
- **Dampak bila dibiarkan:** bila PL berarti plant dengan penomoran sendiri, penghitung per kode customer dan jenis dokumen (D-93) tidak cukup, karena dua plant bisa memakai nomor yang sama.
- **Opsi:**
  - (a) PL adalah Project Leader atau Project Group. Tidak ada perubahan, dan register digabung saat impor.
  - (b) PL adalah plant dengan penomoran sendiri. Penghitung ditambah kunci plant, dan kode pabrik (D-89) mungkin berbeda per plant.
- **Rekomendasi Claude Code (usulan):** pemilik memastikan arti PL1 dan PL3 sebelum register asli diimpor. Bila (a), Q-19 bisa ditutup dengan asumsi kerjanya (satu plant).
- **Asumsi sementara di kode:** satu plant (asumsi kerja Q-19).
- **Menghambat milestone?** Tidak (perlu sebelum register asli diimpor)
- **Status:** Diputuskan pemilik 4 November 2026 (D-126)
- **Keputusan pemilik:** "PL = project leader". Opsi (a): register PL1 dan PL3 adalah register Project Leader, jadi penomoran tetap per kode customer dan jenis dokumen (D-93), dan register digabung saat impor. Q-19 tetap memakai asumsi kerjanya (satu plant).

### E-76 · Proses baru diminta saat menyusun, dan urutan standar proses di pustaka

- **Tanggal dan milestone:** 3 November 2026, tinjauan desain UI. Berdampak ke M1.5 (master proses), M3.3 (pustaka), M4.5 (perakitan), dan M4.6 (nomor operasi).
- **Jenis:** Celah
- **Rujukan:** R-DOC-04, R-CMP-01, R-CMP-05, D-64, D-110, artboard "New PFD" dan "Template library" di kanvas desain UI.
- **Temuan:** desain UI menambah dua hal yang belum ada di brief:
  1. Saat menyusun PFD, penyusun yang tidak menemukan proses bisa memilih "Request a new process". Proses baru lalu ditambahkan ke master oleh Pengelola template (D-64). Brief belum punya alur permintaan ini.
  2. Pustaka menampilkan proses dengan nomor standar (1 Receiving, 2 IQC, 5.2 Solder Paste Printing), dan PFD baru bisa dimulai dari "Template General". Process template M3.2 berdiri per proses, tanpa urutan antar proses.
- **Dampak bila dibiarkan:** tanpa alur permintaan, penyusun cenderung mengetik nama sendiri, dan nama proses kembali beragam (F-05). Tanpa urutan standar, setiap PFD baru disusun dari nol atau disalin dari model lain (R-CMP-05).
- **Opsi:**
  - Permintaan proses baru: (a) dicatat di sistem sebagai permintaan yang diputuskan Pengelola template, mirip UC-25; (b) penyusun memberi tahu Pengelola template di luar sistem.
  - Urutan standar: (a) pustaka umum menyimpan urutan dan nomor operasi standar sebagai titik awal PFD baru; (b) tidak ada urutan standar, dan PFD baru dimulai dari model serupa (R-CMP-05).
- **Rekomendasi Claude Code (usulan):** (a) untuk keduanya, dibangun bersama M4.5 dan M4.6. Master proses (M1.5) dan process template (M3.2) tidak perlu diubah sekarang.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-77 · Doc ref (Relevant Document) berbeda per model, padahal poinnya umum

- **Tanggal dan milestone:** 8 November 2026, saat keputusan E-72. Berdampak ke M3.3 (cakupan poin), M4.1 (baris dokumen), M4.5 (perakitan), dan R-DOC-11 (master dokumen terkait).
- **Jenis:** Celah
- **Rujukan:** R-DOC-11, R-TPL-03, R-CMP-04, D-129, E-73, kolom Relevant Document CP (D-124).
- **Temuan:** masukan pemilik, 8 November 2026: "ada juga kolom Doc ref: disini ada beberapa process yang mencantumkan contoh SW-KT-001, ini adalah WI process model, dan tiap model tentu punya nomor WI yang berbeda. namun karna di doc reff tersebut mencantumkan semua doc untuk process tersebut di awal, ini akan menjadi tantangan karna aktual nya doc no nya akan berbeda, padahal point awal tiap process biasanya general seperti pengecekan Programe name, atau verifikasi part". Artinya:
  1. Kolom Doc ref di baris pertama sebuah proses memuat semua dokumen untuk proses itu, termasuk WI proses milik model (misalnya SW-KT-001).
  2. Poin di baris itu biasanya umum (cek nama program, verifikasi part), tetapi nomor dokumennya berbeda untuk setiap model.
  3. Bila nomor itu disimpan di isi poin, setiap model butuh varian poin hanya karena nomor dokumen, dan varian akan menumpuk.
- **Dampak bila dibiarkan:** perakitan (M4.5) menyalin nomor WI model lain ke dokumen baru, atau penyusun harus mengetik ulang Doc ref di setiap dokumen dan bisa salah ketik.
- **Opsi:**
  - (a) Doc ref tidak disimpan di poin. Kolom itu diisi dari master dokumen terkait (R-DOC-11) menurut model dan proses: setiap WI, SOP, atau QEP dicatat sekali beserta proses dan cakupannya (umum, customer, group model, atau model), lalu baris pertama setiap proses menampilkan semuanya saat dirakit. Poin umum tetap umum.
  - (b) Doc ref menjadi kolom lokal: tidak pernah diambil dari template, selalu diketik di dokumen.
  - (c) Doc ref tetap di isi poin, dan nomor yang berbeda dibuat sebagai varian poin per model (D-129).
- **Rekomendasi Claude Code (usulan):** (a). Nomor dan judul tetap sama dengan sumbernya (tujuan R-DOC-11), poin umum tidak perlu varian, dan revisi WI cukup dicatat sekali di master. Pilihan (c) memperbanyak varian hanya karena nomor dokumen, dan (b) menghilangkan pemeriksaan bahwa nomornya benar-benar ada.
- **Asumsi sementara di kode:** tidak ada. M3.3 tidak menyentuh kolom Doc ref, karena poin di M3.2 masih berisi satu baris teks (ASUMSI E-73).
- **Menghambat milestone?** Tidak (perlu sebelum M4.5)
- **Status:** Diputuskan
- **Keputusan pemilik (8 November 2026, D-130):** (a). Kutipan: "E-77 a". Master dokumen terkait (R-DOC-11) dibangun sebelum M4.5.

### E-78 · Cakupan poin yang salah pilih

- **Tanggal dan milestone:** 8 November 2026, M3.3. Berdampak ke M3.4 (versi template) dan M4.5 (perakitan).
- **Jenis:** Celah
- **Rujukan:** D-129, R-TPL-03, R-TPL-04, R-CMP-04.
- **Temuan:** D-129 menetapkan setiap poin punya cakupan, tetapi belum menyebut apakah cakupan itu boleh diubah setelah poin dibuat. Misalnya poin dibuat "General" padahal seharusnya khusus customer AB.
- **Dampak bila dibiarkan:** bila cakupan bisa diubah, baris dokumen yang sudah tertaut ke poin itu (M4.5) bisa tiba-tiba tidak berlaku lagi untuk modelnya. Bila tidak bisa diubah, Pengelola template harus menyembunyikan poin lalu membuat poin baru, dan tautan baris lama tetap ke poin lama.
- **Opsi:**
  - (a) Cakupan tetap. Salah pilih dibetulkan dengan menyembunyikan poin dan membuat poin baru dengan cakupan yang benar.
  - (b) Cakupan boleh diubah di versi draft saja, dan berlaku sejak versi itu diterbitkan (M3.4).
- **Rekomendasi Claude Code (usulan):** (a) untuk sekarang, karena paling aman bagi tautan baris dokumen. Ditinjau lagi di M3.4 saat versi template punya status terbit.
- **Asumsi sementara di kode (ASUMSI E-78):** (a). Role aplikasi tidak punya UPDATE untuk kolom cakupan di `template_points` (migrasi 00015). Sejak keputusan pemilik, asumsi ini menjadi D-131 dan tidak perlu diubah di kode.
- **Menghambat milestone?** Tidak
- **Status:** Diputuskan
- **Keputusan pemilik (8 November 2026, D-131):** (a). Kutipan: "E-78 a".

### E-79 · Daftar password umum untuk 8 sampai 14 karakter

- **Tanggal dan milestone:** 10 November 2026, perubahan panjang password (D-132). Menyentuh M1.1.
- **Jenis:** Celah
- **Rujukan:** D-85, D-132, E-08, `internal/auth/common_passwords.txt`.
- **Temuan:** daftar password umum yang ada disaring dari daftar NCSC hanya untuk panjang 15 sampai 128 karakter, karena waktu itu batas minimalnya 15. Setelah batas turun ke 8 (D-132), password seperti `password1` atau `12345678` tidak ada di daftar itu.
- **Dampak bila dibiarkan:** password pendek yang paling sering dipakai diterima sistem, padahal D-132 mengandalkan daftar ini sebagai pengimbang.
- **Opsi:**
  - (a) Saring ulang daftar NCSC yang sama (SecLists, lisensi MIT) untuk panjang 8 sampai 128, lalu ganti kedua file dengan satu file. Perlu izin pemilik untuk mengunduh, dan file menjadi jauh lebih besar (belum diukur).
  - (b) Tetap memakai daftar tambahan tulis tangan.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode (ASUMSI E-79):** (b). File `internal/auth/common_passwords_short.txt` berisi sekitar 170 password umum yang ditulis tangan dan belum diverifikasi terhadap sumber. Test memastikan `password1` dan `12345678` ditolak.
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

### E-80 · Prompt form set mengandaikan editor, wizard, yokoten, dan ekspor yang belum dibangun

- **Tanggal dan milestone:** 10 November 2026, saat membaca prompt form set (D-133). Menyentuh Tahap 3 sampai Tahap 6.
- **Jenis:** Kontradiksi
- **Rujukan:** D-133, D-137, `docs/rencana/form-set-aiag-vda.md`, milestone Tahap 4 sampai 6.
- **Temuan:** Prompt menulis "desain sebelumnya sudah kamu implementasikan" dan menyusun 11 milestone yang berurutan. Kode baru sampai M3.3: dokumen, baris, editor workbook, wizard, permintaan perubahan, yokoten, cetak, ekspor, dan rilis belum ada.
- **Dampak bila dibiarkan:** milestone prompt nomor 3 sampai 11 tidak bisa diuji, karena layar tempat menumpangnya belum ada.
- **Opsi:**
  - (a) Bagian yang hanya menyentuh format template dikerjakan sekarang (mesin AP, isi awal dua format baru, form set dan default customer). Sisanya menumpang milestone yang sudah ada, seperti tabel di rencana.
  - (b) Seluruh pekerjaan form set ditunda sampai Tahap 4 selesai.
  - (c) Tahap 4 dikerjakan dulu dengan urutan yang diubah supaya milestone prompt bisa berurutan.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada; belum ada kode form set.
- **Menghambat milestone?** Ya, untuk mulai pekerjaan form set. Tidak untuk M3.4.
- **Status:** Diputuskan
- **Keputusan pemilik (11 November 2026, D-140):** (a). Kutipan: "E-80 A".
### E-81 · Isi poin PFMEA di process template untuk form AIAG-VDA

- **Tanggal dan milestone:** 10 November 2026, pertanyaan 3 di prompt form set. Menyentuh M3.2, M4.5.
- **Jenis:** Celah
- **Rujukan:** E-73, D-133, R-TPL-01, R-CMP-04.
- **Temuan:** Poin process template sekarang berisi satu baris teks (ASUMSI E-73). Form AIAG-VDA punya kolom yang tidak ada di AIAG 4th (work element dan tiga kolom fungsi). Belum diputuskan apakah poin menyimpan isi per kolom, dan bagaimana satu poin dipakai di set 1 dan set 2.
- **Dampak bila dibiarkan:** perakitan (M4.5) tidak tahu kolom mana yang diisi dari poin untuk dokumen set 2.
- **Opsi:**
  - (a) Poin menyimpan isi per key kolom. Kolom yang sama di kedua form (failure mode, effect, cause, kontrol) dipakai bersama; kolom khusus VDA hanya diisi untuk set 2.
  - (b) Poin terpisah untuk set 1 dan set 2.
- **Rekomendasi Claude Code (usulan):** (a), diputuskan bersama E-73.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak (perlu sebelum M4.5)
- **Status:** Terbuka
- **Keputusan pemilik:**
### E-82 · Siapa yang menyetujui form set baru dan perubahan tabel AP

- **Tanggal dan milestone:** 10 November 2026, pertanyaan 4 di prompt form set.
- **Jenis:** Celah
- **Rujukan:** D-133, D-134, D-64, U3 (hak A-2 dan E-6).
- **Temuan:** Format template dikelola admin (A-2), sedangkan process template dikelola Pengelola template (E-6). Prompt meminta form set 2 "masuk lewat approval" dan tabel AP "dikoreksi lewat versi baru dengan approval", tanpa menyebut siapa yang menyetujui.
- **Dampak bila dibiarkan:** tabel AP yang menentukan prioritas tindakan bisa diubah satu orang tanpa orang kedua.
- **Opsi:**
  - (a) Admin menyusun draft, PPAP Engineer (E-6) menerbitkan. Dua orang berbeda.
  - (b) Admin saja, seperti format template sekarang.
  - (c) PPAP Engineer saja.
- **Rekomendasi Claude Code (usulan):** (a), karena tabel AP adalah isi teknis dan admin bukan pemilik isinya.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Ya, untuk langkah A dan D di rencana form set
- **Status:** Diputuskan
- **Keputusan pemilik (11 November 2026, D-139):** PPAP Supervisor (jabatan APQP/PPAP Controller) untuk saat ini, dan harus bisa diubah nanti tanpa mengubah kode. Kutipan: "E-82 untuk saat ini PPAP supervisor, namun buat ini juga flexible, jadi di masa depan bisa saja project leader ikut menyetujui". Ini berbeda dari ketiga opsi di atas. Rancangannya (usulan): hak baru di katalog yang dipegang peran baru, sehingga admin cukup memberi peran itu ke akun lain (D-66). Masih terbuka: bila pemegangnya lebih dari satu, cukup satu persetujuan (asumsi kerja) atau semua harus setuju.
### E-83 · Perpindahan dokumen set 1 ke set 2: satu per satu atau massal

- **Tanggal dan milestone:** 10 November 2026, pertanyaan 5 di prompt form set. Menyentuh Tahap 5.
- **Jenis:** Celah
- **Rujukan:** D-133, R-CHG, R-REV-02.
- **Temuan:** Perpindahan dibuat sebagai permintaan perubahan per dokumen. Belum diputuskan apakah boleh massal per customer.
- **Dampak bila dibiarkan:** customer yang mewajibkan AIAG-VDA untuk semua model butuh satu permintaan perubahan per set dokumen.
- **Opsi:**
  - (a) Satu per satu dulu; massal ditambahkan setelah alur satu dokumen terbukti.
  - (b) Massal per customer sejak awal, tetap menghasilkan satu draft revisi per dokumen.
- **Rekomendasi Claude Code (usulan):** (a).
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak (perlu sebelum perpindahan format dibangun)
- **Status:** Terbuka
- **Keputusan pemilik:**
### E-84 · Genba: penulis catatan selain tim PPAP, dan lama penyimpanan foto

- **Tanggal dan milestone:** 10 November 2026, pertanyaan 8 di prompt form set. Menyentuh Tahap 6.
- **Jenis:** Celah
- **Rujukan:** D-137, U3, R-SEC-10.
- **Temuan:** Prompt memberi izin `genba.note` ke Penyusun dan menyebut admin bisa memberi ke peran lain. Belum diputuskan peran mana (QA, leader produksi), dan berapa lama foto catatan disimpan. Katalog hak saat ini memakai kode L-, E-, V-, A-, jadi dua izin itu perlu kode baru, misalnya E-11 dan E-12.
- **Dampak bila dibiarkan:** foto menumpuk tanpa batas, dan akun QA atau produksi tidak bisa menulis catatan.
- **Opsi:**
  - (a) Hak baru bisa diberikan admin ke peran mana pun; foto disimpan selama revisi itu masih punya catatan terbuka, lalu mengikuti retensi rekaman.
  - (b) Hanya tim PPAP; foto disimpan selamanya.
- **Rekomendasi Claude Code (usulan):** (a), setelah pemilik menyebut masa retensi.
- **Asumsi sementara di kode:** tidak ada.
- **Menghambat milestone?** Tidak (perlu sebelum tampilan genba dibangun)
- **Status:** Terbuka
- **Keputusan pemilik:**
### E-85 · Alias proses: siapa yang mengelola

- **Tanggal dan milestone:** 10 November 2026, pertanyaan 9 di prompt form set. Menyentuh M1.5 dan pencarian per proses.
- **Jenis:** Ambigu
- **Rujukan:** D-06, D-64, D-137, E-76.
- **Temuan:** Prompt bertanya apakah alias dikelola "Admin saja". Di kode sejak M1.5 alias dikelola Pengelola template (E-6), bukan admin (D-64). Belum diputuskan apakah penyusun boleh mengusulkan alias.
- **Dampak bila dibiarkan:** penyusun yang menemukan nama lain sebuah proses harus meminta lewat jalur di luar sistem.
- **Opsi:**
  - (a) Tetap Pengelola template saja.
  - (b) Penyusun mengusulkan, Pengelola template menyetujui. Bisa digabung dengan usulan proses baru (E-76).
- **Rekomendasi Claude Code (usulan):** (b), bersama E-76.
- **Asumsi sementara di kode:** alias tetap dikelola pemegang E-6 (M1.5).
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**
### E-86 · Review berkala 3 tahun: rincian yang belum diputuskan

- **Tanggal dan milestone:** 10 November 2026, permintaan pemilik (D-138). Menyentuh Tahap 5, 6, dan 7.
- **Jenis:** Celah
- **Rujukan:** D-138, R-REV-02, R-CHG-02, D-07, `docs/rencana/review-berkala-3-tahun.md`.
- **Temuan:** D-138 menetapkan peringatan setelah 3 tahun tanpa revisi dan dua hasil review. Belum diputuskan: (1) bukti review tanpa perubahan; (2) jam dihitung per dokumen atau per set; (3) apakah review tanpa perubahan memulai lagi jam 3 tahun; (4) siapa yang mereview dan apakah perlu orang kedua; (5) berapa hari sebelum jatuh tempo peringatan muncul. Tanggal rilis baru ada di Tahap 6, jadi fitur ini belum bisa dibangun sekarang.
- **Dampak bila dibiarkan:** review tanpa perubahan tidak punya bukti yang bisa ditunjukkan ke auditor, atau setiap review memaksa enam tanda tangan untuk dokumen yang tidak berubah.
- **Opsi:**
  - (a) Catatan review di sistem dengan daftar periksa, peserta, dan konfirmasi orang kedua; revisi tidak naik. Jam per set dokumen, dimulai lagi oleh catatan review. Peringatan 90 hari sebelumnya.
  - (b) Revisi administratif: dokumen naik revisi dengan riwayat "Periodic review, no change" dan tanda tangan basah lengkap.
  - (c) Seperti (a), ditambah lembar review yang dicetak, ditandatangani, dan dipindai sebagai lampiran.
- **Rekomendasi Claude Code (usulan):** (a), atau (c) bila auditor atau customer meminta tanda tangan.
- **Asumsi sementara di kode:** tidak ada; belum dibangun.
- **Menghambat milestone?** Tidak (perlu sebelum Tahap 6)
- **Status:** Terbuka
- **Keputusan pemilik:**
### E-87 · Menerbitkan versi template: perlu orang kedua atau tidak

- **Tanggal dan milestone:** 11 November 2026, M3.4.
- **Jenis:** Celah
- **Rujukan:** R-TPL-05, D-64, D-139, R-SEC-11.
- **Temuan:** R-TPL-05 tidak menyebut siapa yang menerbitkan versi template. Di M3.4 orang yang menyusun draft juga boleh menerbitkannya: admin untuk format template, Pengelola template untuk process template. Untuk form set dan tabel AP, pemilik sudah memutuskan penyetuju terpisah (D-139).
- **Dampak bila dibiarkan:** satu orang bisa mengubah dan menerbitkan template yang nanti dipakai banyak dokumen, tanpa pemeriksaan orang kedua.
- **Opsi:**
  - (a) Tetap seperti M3.4: penyusun boleh menerbitkan. Pemeriksaan terjadi saat dokumen direvisi dan ditandatangani.
  - (b) Penerbit harus orang lain, dengan hak yang sama seperti D-139.
- **Rekomendasi Claude Code (usulan):** (a) untuk process template, karena perubahan template tetap lewat revisi dokumen yang disetujui (R-TPL-06); (b) hanya untuk form set dan tabel AP, sesuai D-139.
- **Asumsi sementara di kode (ASUMSI E-87):** (a).
- **Menghambat milestone?** Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**

## Usulan desain UI (UX-01 sampai UX-07): dampak ke M0 sampai M3 dan saran

- **Sumber:** `docs/uml/usulan-desain-ui.md` (UML draft 0.12, 3 November 2026) dan kanvas desain UI "Desain UI Sistem PFD PFMEA CP" (18 artboard, baris v2 disarankan).
- **Status usulan:** tetap Terbuka di file sumbernya sampai pemilik memutuskan. Tabel ini menambahkan dampaknya ke kode yang sudah dibangun, serta saran Claude Code.
- **Catatan tinjauan:** `docs/catatan/Tinjauan-desain-UI-UML-0.12.md`.
- **Pembaruan 8 November 2026:** UX-05 diterima sebagian (D-127): design system dan menu berkelompok untuk halaman M0 sampai M3. Workbook dan bahasa EN/ID tetap terbuka. Catatan: `docs/catatan/UX-05-menu-dan-design-system.md`. Bahasa EN/ID dan tampilan responsif diputuskan di hari yang sama (D-128, menggantikan D-96); catatan: `docs/catatan/D-128-dua-bahasa-dan-responsif.md`.

| Usulan | Dampak ke M0 sampai M3 | Perubahan kode sekarang | Saran Claude Code (usulan) |
|---|---|---|---|
| UX-01 Cakupan poin dipilih saat menyusun | M3.2 tetap. Menjawab E-72: satu template per proses, dan cakupan melekat pada poin. Rencana M3.3 berubah menjadi cakupan per poin. | Tidak | Terima. M3.3 menambah cakupan pada poin. Alur usulan ke Pengelola template (UC-25) dibangun di M4.5 dan M4.8. |
| UX-02 Gambar dan tabel spesifikasi di sel | Pemeriksa isi M3.1 (`CheckContent`) dan isi poin M3.2 (E-73) baru menerima teks. Gambar butuh storage file, yang menurut milestone baru datang di M5.2. | Tidak | Terima untuk M4.2: kolom teks mana pun boleh berisi gambar atau tabel spesifikasi. Storage privat dengan SHA-256 dimajukan ke Tahap 4. |
| UX-03 Master simbol karakteristik khusus | M3.1 menyimpan simbol per versi format template, tanpa gambar, dan semua simbol bisa dipilih di kolom simbol. | Sebagian: Revision ditandai otomatis dan ditolak di kolom simbol (migrasi 00013, R-REV-09). | Terima. Simbol customer pindah ke master per customer, dengan gambar dan versi (UC-24, hak A-2), lalu dirujuk format template. Critical Process menjadi tanda proses di dokumen, bukan simbol kolom karakteristik. Dibangun sebelum M4.1. |
| UX-04 Cek sinkronisasi tiga dokumen | Tidak ada di M0 sampai M3. Mendukung E-33 opsi (a). | Tidak | Terima. UC-36 diperluas ke PFD di M4.9. |
| UX-05 Layar workbook, menu berlabel, teks besar, EN/ID | Semua halaman M0 sampai M3: menu (sekarang 15 tautan datar), gaya, dan ukuran teks. Bahasa Indonesia mengubah D-96. | Tidak | Terima workbook untuk layar dokumen (Tahap 4). Menu berlabel dan design system (warna, huruf yang dibundel dengan aplikasi) diterapkan ke semua halaman M0 sampai M3 dalam satu langkah sebelum M4. Bahasa Indonesia ditambah sebagai katalog kedua dengan kunci yang sama, karena D-96 sudah menyiapkannya; perlu keputusan pemilik karena mengubah D-96. |
| UX-06 Temuan register nomor | M2.5: kolom No. (Before) belum disimpan, dan satu nomor untuk beberapa part belum dilaporkan. Kode tiga huruf tetap ditolak (E-74, D-125); PL1 dan PL3 adalah register Project Leader (E-75, D-126). | Tidak | Terima: impor menerima kolom No. (Before) sebagai nomor lama yang bisa dicari, dan laporan menandai satu nomor yang dipakai beberapa part. |
| UX-07 Cakupan yokoten | Tidak ada di M0 sampai M3. | Tidak | Terima untuk M5.9 dan M7.7. |

## Keputusan pemilik atas pertanyaan terbuka brief (bagian 12)

Bagian ini bukan temuan E. Isinya jawaban pemilik atas item Q di brief bagian 12, untuk dibawa ke sesi desain.

| Q | Keputusan | Tanggal | Catatan |
|---|---|---|---|
| Q-14 | Bahasa layar: Inggris untuk saat ini (D-96). | 28 Oktober 2026 | Disiapkan untuk pindah ke bahasa Indonesia: semua teks layar ada di katalog `web/src/lib/i18n/en.ts` dan dipanggil lewat `t(kunci)`. Bahasa Indonesia cukup ditambah sebagai `id.ts` dengan kunci yang sama (lihat catatan M0.4). Istilah Inggrisnya: E-19. |

### Hasil uji yang menunggu keputusan pemilik

| Q | Hasil uji | Tanggal | Dipakai sementara |
|---|---|---|---|
| Q-47 | openapi-typescript 7.13.0 (rilis 11 Februari 2026) gagal dipasang: `npm install` berhenti dengan ERESOLVE karena peer `typescript ^5.x` tidak cocok dengan TypeScript 6.0.3, jadi melanggar R-VER-03. @hey-api/openapi-ts 0.99.0 (rilis 22 Juni 2026, 0.x tapi sudah lebih dari 14 hari, E-14) terpasang tanpa konflik, karena peer-nya `typescript >=5.5.3 \|\| >=6.0.0`. Dengan plugin `@hey-api/typescript` saja, hasilnya hanya dua file tipe (`web/src/lib/api/gen/`). Pemanggilan fetch ditulis tangan di `web/src/lib/api/client.ts`. Catatan npm audit: js-yaml 4.2.0 (tingkat high, CPU kuadratik untuk YAML jahat) terbawa @hey-api/json-schema-ref-parser 1.4.4 yang mengunci versi itu. Hanya dipakai saat generate untuk membaca kontrak kita sendiri, jadi risikonya rendah. `npm audit fix --force` akan menurunkan ke 0.97.0, jadi tidak dipakai. | 28 Oktober 2026 | @hey-api/openapi-ts 0.99.0, plugin tipe saja. Diputuskan pemilik 30 Oktober 2026 lewat persetujuan umum (D-97) |

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
