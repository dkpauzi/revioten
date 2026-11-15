# U3 · Peta pengguna dan hak akses

- Paket UML: draft 0.16, 15 November 2026 (draft 0.9: 3 November 2026). Sumber: brief desain v1.34. Draft 0.16 menambah layar Roles (M8.1), row-level security (D-182), service account (Tahap 9), email notifikasi (D-194), dan bagian 8 tentang siapa yang butuh akun. Isi matriks hak tidak berubah. Draft 0.8 menambah keputusan D-103, D-104, D-115, D-118, dan D-120 tanpa mengubah isi matriks hak. Draft 0.9 hanya memetakan use case usulan UC-24, UC-25, dan UC-39 ke hak yang sudah ada (A-2, E-2, E-6), tanpa hak baru dan tanpa mengubah isi peran.
- Status: **baseline hak akses, disetujui pemilik pada 27 Oktober 2026** (D-66, R-SEC-10). Admin mengubah hak akses mulai dari baseline ini, dan setiap perubahannya tercatat di audit log. Hal yang masih bertanda usulan di dalam peta ini tetap usulan.
- Diagram: `U3-peta-pengguna.puml`, diekspor ke `export/U3-H-peta-pengguna.png` dan `.svg`.
- Nama jabatan mengikuti bagan organisasi NPI (D-54). Nama orang tidak dicantumkan.

## 1. Empat level akses

| Level | Artinya | Peran di level ini |
|---|---|---|
| **Buat / Edit** | Membuat atau mengubah isi dokumen, nomor, template, atau status rilis. | Penyusun, Pengelola template, Dokumen kontrol, Petugas migrasi |
| **Verifikasi** | Menerima atau menolak permintaan perubahan, atau memberi tanda dicek, tanpa mengedit isi dokumen. | Verifikator, Pemeriksa PFMEA–CP, Penyetuju form |
| **Lihat** | Hanya membaca dan mengekspor dokumen rilis. | Viewer |
| **Admin** | Mengelola pengguna, hak akses, master data, dan pengaturan, serta melihat audit log. Dikerjakan lewat akun admin terpisah yang tidak memegang peran kerja (D-72, R-SEC-13). | Admin |

Semua pengguna yang login otomatis punya hak dasar level Lihat (L-1 sampai L-3), termasuk akun admin (D-120). Satu orang boleh memegang beberapa peran, dan hak akhirnya adalah gabungan semua perannya (D-58).

## 2. Peta jabatan ke peran (saat ini)

| Jabatan | Level tertinggi | Peran | Cakupan customer | Dasar |
|---|---|---|---|---|
| PPAP Engineer | Buat / Edit | Penyusun, Pengelola template, Pemeriksa PFMEA–CP, Dokumen kontrol, Petugas migrasi | Semua customer | D-63 sampai D-65, D-68, D-69 |
| PPAP Staff (akun kerja) | Buat / Edit | Penyusun, Dokumen kontrol, Petugas migrasi | Semua customer | D-60, D-63, D-69. Juga memegang akun admin terpisah. |
| Akun admin (terpisah) | Admin | Admin | Semua | D-72 sampai D-75. Saat ini dipegang PPAP Staff, sehingga ia punya dua akun. Hanya untuk tugas admin, dan hanya akun ini yang melihat audit log (D-75). Akun ditautkan ke pemegangnya, sehingga audit log menampilkan akun dan orangnya (D-74). |
| Project Leader (level Supervisor, satu per Project Group) | Buat / Edit | Penyusun, Verifikator | Customer di Project Group sendiri (D-71) | D-51, D-54, D-63, D-67. Juga pengecek di kertas. |
| Project Engineer (level Leader, satu per Project Group) | Buat / Edit | Penyusun, Verifikator | Customer di Project Group sendiri (D-71) | D-51, D-54, D-67. Juga pengecek di kertas. |
| APQP/PPAP Controller (PPAP Supervisor) | Verifikasi | Verifikator, Penyetuju form (D-139) | Semua customer | D-51, D-54. Dipegang satu orang (D-115). |
| NPI Manager, QA QC Manager, Advisor (slot GM), Factory Manager | Lihat | Viewer | Semua customer, dokumen rilis | D-70. Tetap menandatangani di kertas. Setiap jabatan ini dipegang satu orang, karena satu slot mencetak satu nama (D-115). |
| QA, QC, internal audit | Lihat | Viewer | Semua customer, dokumen rilis | D-41, D-70 |
| DCC (Dept Document Control) | Lihat, saat ini | Viewer | Semua customer, dokumen rilis | D-70. Nanti admin bisa menambahkan peran Dokumen kontrol (D-60, D-66). |
| Tim WI (WI Handler, WI Engineer, WI Editor) dan ECN & SAP Coordinator | Lihat | Viewer | Semua customer, dokumen rilis | D-70 |

**Rangkap jabatan:** Project Leader GR3 juga APQP/PPAP Controller. Perannya digabung, sehingga ia memegang Penyusun dan Verifikator. Cakupan verifikasinya semua customer, sedangkan cakupan menyusunnya hanya customer GR3 (D-71).

**Pengecek** tidak menjadi peran di sistem. Project Engineer dan Project Leader menandatangani slot Checked di kertas. Di sistem, peran ini baru muncul saat izin tampil di portal (UC-85, masa depan).

**Jabatan lain di bagan NPI** (misalnya PP Conductor atau NPI and PJ Admin) belum diberi akun. Admin bisa menambahkannya kapan saja sebagai Viewer.

## 3. Matriks hak akses per peran

✓ = punya hak, — = tidak.

| Kode | Hak | Use case | Penyusun | Pengelola template | Pemeriksa PFMEA–CP | Verifikator | Dokumen kontrol | Petugas migrasi | Viewer | Admin |
|---|---|---|---|---|---|---|---|---|---|---|
| **L-1** | Lihat dokumen rilis dan halaman verifikasi, termasuk halaman Released documents dan tab Check baca saja (D-195) | UC-56, UC-57 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-2** | Lihat register nomor, riwayat dan diff, diagram PFD, dasbor, dan master data (D-103) | UC-12, UC-37, UC-47, UC-64 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-3** | Ekspor Excel dan PDF dokumen rilis | UC-54 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-4** | Lihat draft dan permintaan perubahan yang sedang berjalan (usulan) | UC-47 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | — | — |
| **L-5** | Lihat laporan kinerja PPAP Engineer (D-197, migrasi 00047). Bawaannya juga untuk Penyetuju form (supervisor) | UC-65 | ✓ | — | — | — | — | — | — | — |
| **E-1** | Daftarkan nomor dokumen, termasuk sebagai set | UC-10, UC-11 | ✓ | — | — | — | — | — | — | — |
| **E-2** | Buat, rakit, dan edit dokumen draft, termasuk gambar dan tabel spesifikasi di sel, serta mengusulkan cakupan poin (usulan) | UC-30 sampai UC-35, UC-38, UC-39, UC-25 (mengusulkan) | ✓ | — | — | — | — | — | — | — |
| **E-3** | Ajukan perubahan dan isi daftar periksa dampak | UC-40, UC-42 | ✓ | — | — | — | — | — | — | — |
| **E-4** | Kerjakan draft revisi, yokoten, dan perubahan surut SQAM | UC-43 sampai UC-46 | ✓ | — | — | — | — | — | — | — |
| **E-5** | Cetak untuk tanda tangan dan catat dokumen dikembalikan | UC-50, UC-51 | ✓ | — | — | — | — | — | — | — |
| **E-6** | Kelola master proses dan process template, terbitkan dan propagasikan versi, serta menerima atau menolak usulan cakupan poin (usulan) | UC-04, UC-21 sampai UC-23, UC-25 (memutuskan) | — | ✓ | — | — | — | — | — | — |
| **E-7** | Catat rilis | UC-52 | — | — | — | — | ✓ | — | — | — |
| **E-8** | Ubah status nomor dan impor nomor lama | UC-13, UC-14 | — | — | — | — | ✓ | — | — | — |
| **E-9** | Cetak salinan terkendali dan lihat daftar salinan yang harus ditarik | UC-53, UC-55 | — | — | — | — | ✓ | — | — | — |
| **E-10** | Migrasikan dokumen lama | UC-60 sampai UC-63 | — | — | — | — | — | ✓ | — | — |
| **V-1** | Verifikasi permintaan perubahan | UC-41 | — | — | — | ✓ | — | — | — | — |
| **V-2** | Beri tanda dicek di perbandingan PFMEA dan CP | UC-36 | — | — | ✓ | — | — | — | — | — |
| **V-3** | Setujui (terbitkan) form set dan versi format bermetode AP; bukan oleh penyusun drafnya | Form set (D-139) | — | — | — | — | — | — | — | — |
| **A-1** | Kelola pengguna dan hak akses, isi peran (layar Roles), service account API integrasi, dan alamat email orang lain | UC-02, UC-70, UC-80 | — | — | — | — | — | — | — | ✓ |
| **A-2** | Kelola master data, jenis dokumen dan perubahan, rute approval, dokumen terkait, format template, dan master simbol karakteristik (usulan) | UC-03, UC-05, UC-06, UC-09, UC-20, UC-24 | — | — | — | — | — | — | — | ✓ |
| **A-3** | Kelola pengaturan sistem, termasuk mode migrasi dan email notifikasi | UC-07, UC-71 | — | — | — | — | — | — | — | ✓ |
| **A-4** | Lihat audit log | UC-08 | — | — | — | — | — | — | — | ✓ |

**Penyetuju form** (peran `form_approver`, D-139, migrasi 00019, 12 November 2026) memegang L-1 sampai L-4 dan V-3. Kolomnya tidak ditambahkan ke matriks di atas supaya tabel tetap terbaca; untuk saat ini dipegang APQP/PPAP Controller (PPAP Supervisor), dan admin bisa memberikannya ke akun lain, misalnya Project Leader.

Hak yang tercantum di matriks ini adalah isi awal setiap peran. Admin bisa menambah atau mengurangi hak sebuah peran di layar **Admin → Roles** (M8.1), atau memberi peran tambahan kepada seorang pengguna (R-SEC-10). Yang terkunci di layar, kode, dan database (D-183): isi peran Admin dan Viewer, hak admin (A-1 sampai A-4) di peran kerja, dan hak dasar L-1 sampai L-3. Peran baru tidak bisa dibuat dan peran tidak bisa dihapus.

## 4. Aturan yang tidak bisa diubah admin (R-SEC-11)

Aturan 7 dan 8 sudah diputuskan (D-72, D-73). Aturan 1 sampai 6 masih usulan sebagai satu paket, kecuali aturan 1 dan 4 yang sumbernya sudah diputuskan (R-CHG-02, R-REV-02).

1. Pemohon perubahan tidak boleh memverifikasi permintaannya sendiri (R-CHG-02).
2. Satu orang hanya mengisi satu slot dalam satu rute, jadi penyusun tidak bisa sekaligus menjadi pengecek (R-APR-12).
3. Pencatat rilis harus berbeda dari penyusun (R-APR-02).
4. Revisi yang sudah rilis tidak bisa diubah siapa pun, termasuk admin (R-REV-02).
5. Audit log tidak bisa diubah atau dihapus (R-SEC-03).
6. Hak admin tidak mencakup mengedit isi dokumen. Kalau pemegang akun admin perlu menyusun, ia memakai akun kerjanya sendiri (aturan 8).
7. Akun admin terakhir tidak bisa dihapus atau kehilangan peran admin (D-73). Akun tidak pernah dihapus, hanya dinonaktifkan, jadi "dihapus" berarti dinonaktifkan (D-105).
8. Akun admin tidak pernah memegang peran kerja, dan akun kerja tidak pernah memegang peran admin (R-SEC-13, D-72).
9. Setiap akun boleh mengurus akunnya sendiri: melihat sesinya, logout, dan mengganti password. Ini bukan hak di katalog, jadi tidak bisa dicabut admin, tapi tetap diperiksa fungsi izin (D-104).
10. Hak bercakupan Project Group (Penyusun dan Verifikator milik Project Leader dan Project Engineer) hanya berlaku untuk operasi pada satu customer. Operasi tanpa customer ditolak untuk hak itu; daftar lintas customer menyaring per customer (D-118).

Aturan 9 dan 10 diputuskan 30 Oktober 2026. Akun admin yang lupa password atau terkunci dibuka lewat perintah di komputer server, bukan lewat hak di katalog (D-101).

## 5. Contoh: DCC diberi hak edit di masa depan

1. Admin, lewat akun admin, membuka akun pengguna DCC lalu menambahkan peran Dokumen kontrol. Bila perlu, peran Penyusun juga ditambahkan.
2. Sistem mencatat siapa yang mengubah, apa yang diubah, dan kapan (audit log).
3. Mulai saat itu pengguna DCC bisa mencatat rilis (E-7). Aturan di bagian 4 tetap berlaku, misalnya DCC tidak bisa mencatat rilis dokumen yang ia susun sendiri.

Tidak ada perubahan kode. Yang berubah hanya data peran pengguna (D-66).

## 6. Konflik dan cara penyelesaiannya

| ID | Konflik | Status |
|---|---|---|
| K-1 | PPAP Staff adalah penyusun dan pemegang Dokumen kontrol, padahal pencatat rilis harus berbeda dari penyusun (aturan 3). | Selesai (D-68). PPAP Engineer juga memegang Dokumen kontrol, jadi PPAP Staff dan PPAP Engineer saling mencatat rilis dokumen yang disusun yang lain. Dokumen dari Project Leader atau Project Engineer bisa dicatat salah satunya. |
| K-2 | Project Leader dan Project Engineer masing-masing memegang tiga fungsi: penyusun, verifikator, dan pengecek di kertas. | Dijaga aturan 1 dan 2. Bila Project Leader menyusun, pengeceknya Project Engineer, begitu pula sebaliknya. Verifikatornya adalah pemegang lain di antara Project Engineer, Project Leader, atau APQP/PPAP Controller. |
| K-3 | Pemegang akun admin (PPAP Staff) juga penyusun, sehingga ia bisa memberi hak tambahan ke akun kerjanya sendiri. | Dikurangi dengan akun admin terpisah (D-72). Aturan di bagian 4 tidak bisa dimatikan, dan setiap perubahan hak tercatat beserta akun dan pemegangnya (D-74). Tinjauan catatan perubahan hak oleh NPI Manager ditolak pemilik (D-75), jadi risiko sisanya tetap tercatat di RK-63. |

## 7. Siapa yang butuh akun

| Kebutuhan | Butuh akun di sistem ini? | Dasar |
|---|---|---|
| Mengajukan perubahan dari layar sistem | Ya, akun kerja dengan hak E-3 (peran Penyusun) untuk customer itu | R-CHG-01, D-63, D-71 |
| Mengajukan perubahan dari web ECR atau aplikasi lain | Tidak untuk orangnya. Aplikasinya memakai token service account; nama orangnya dikirim sebagai `on_behalf_of` dan hanya dicatat | D-193, R-API-09 |
| Memverifikasi perubahan, termasuk yang datang dari web ECR | Ya, akun kerja dengan V-1 dan jabatan verifikator | D-51, D-54 |
| Membaca dokumen terbit (QA, QC, internal audit, produksi) | Ya, akun Viewer (L-1 sampai L-3) | D-70 |
| Menerima email notifikasi | Ya, akun dengan alamat email di orangnya | D-194 |

Database juga menyaring dokumen per akun (row-level security, D-182): draft hanya terlihat oleh pemegang L-4 untuk customer itu, dokumen yang pernah terbit terlihat oleh semua akun yang login.

## 8. Yang masih menunggu jawaban

Tidak ada pertanyaan terbuka khusus untuk peta ini. Temuan dari kode yang menyentuh hak akses dicatat di `docs/evaluasi-desain.md`.
