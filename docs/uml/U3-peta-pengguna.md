# U3 · Peta pengguna dan hak akses

- Paket UML: draft 0.19, 20 November 2026 (draft 0.9: 3 November 2026). Sumber: brief desain v1.37. Draft 0.19 mengganti peran dan matriks hak dengan matriks D-199 (fase 2, migrasi 00049): peran PPAP Engineer, PPAP Engineer Supervisor, Project Leader, dan Manager, hak baru S-1 dan G-1, serta peran yang bisa ditambah admin. Draft 0.16 menambah layar Roles (M8.1), row-level security (D-182), service account (Tahap 9), email notifikasi (D-194), dan bagian 8 tentang siapa yang butuh akun. Isi matriks hak tidak berubah. Draft 0.8 menambah keputusan D-103, D-104, D-115, D-118, dan D-120 tanpa mengubah isi matriks hak. Draft 0.9 hanya memetakan use case usulan UC-24, UC-25, dan UC-39 ke hak yang sudah ada (A-2, E-2, E-6), tanpa hak baru dan tanpa mengubah isi peran.
- Status: **baseline hak akses, disetujui pemilik pada 27 Oktober 2026** (D-66, R-SEC-10). Admin mengubah hak akses mulai dari baseline ini, dan setiap perubahannya tercatat di audit log. Hal yang masih bertanda usulan di dalam peta ini tetap usulan.
- Diagram: `U3-peta-pengguna.puml`, diekspor ke `export/U3-H-peta-pengguna.png` dan `.svg`.
- Nama jabatan mengikuti bagan organisasi NPI (D-54). Nama orang tidak dicantumkan.

## 1. Empat level akses

| Level | Artinya | Peran bawaan di level ini (D-199) |
|---|---|---|
| **Buat / Edit** | Membuat atau mengubah isi dokumen, nomor, template, atau status rilis. | PPAP Engineer, PPAP Engineer Supervisor, Project Leader |
| **Verifikasi** | Menerima atau menolak permintaan perubahan, memberi tanda dicek, menyetujui form set, atau menandatangani. | PPAP Engineer Supervisor, Project Leader, Manager |
| **Lihat** | Membaca dan mengekspor dokumen rilis, dan memberi catatan genba. | Viewer |
| **Admin** | Mengelola pengguna, hak akses, master data, dan pengaturan, serta melihat audit log. Dikerjakan lewat akun admin terpisah yang tidak memegang peran kerja (D-72, R-SEC-13). | Admin |

Semua akun kerja otomatis memegang hak peran Viewer (L-1 sampai L-3 dan G-1). Akun admin memegang L-1 sampai L-3 lewat peran Admin, tanpa G-1 (D-120, D-199). Satu orang boleh memegang beberapa peran, dan hak akhirnya adalah gabungan semua perannya (D-58).

## 2. Peta jabatan ke peran (saat ini, D-199)

| Jabatan | Peran | Cakupan customer | Dasar |
|---|---|---|---|
| PPAP Engineer dan PPAP Staff (akun kerja) | PPAP Engineer | Semua customer | D-63 sampai D-65, D-68, D-69, D-199. Dokumen kontrol digabung ke peran ini untuk saat ini. PPAP Staff juga memegang akun admin terpisah. |
| Akun admin (terpisah) | Admin | Semua | D-72 sampai D-75. Hanya untuk tugas admin, dan hanya akun ini yang melihat audit log (D-75). |
| APQP/PPAP Controller (PPAP Supervisor) | PPAP Engineer Supervisor | Semua customer | D-51, D-54, D-139, D-199. Semua hak PPAP Engineer, ditambah verifikasi dan penyetujuan form. |
| Project Leader (Supervisor) dan Project Engineer (Leader), satu per Project Group | Project Leader | Customer di Project Group sendiri (D-71) | D-51, D-54, D-67, D-199. Haknya sama; bedanya di slot tanda tangan, yang mengikuti jabatan di rute approval. |
| NPI Manager, QA QC Manager, Advisor (slot GM), Factory Manager | Manager | Semua customer | D-70, D-198, D-199. Penanda tangan; boleh melihat laporan kinerja. Setiap jabatan dipegang satu orang (D-115). |
| QA, QC, internal audit, DCC, tim WI, ECN & SAP Coordinator | Tanpa peran (Viewer) | Semua customer, dokumen rilis | D-41, D-70, D-199 |

**Project Leader pembantu (D-206):** batas Project Group bisa dilepas per akun. Admin memberi peran Project Leader dengan cakupan semua customer, misalnya untuk Project Leader GR5 yang membantu GR1; orang itu lalu juga boleh menandatangani slot Prepared dan Checked serta memverifikasi perubahan Project Group lain.

**Rangkap jabatan:** Project Leader GR3 juga APQP/PPAP Controller, jadi ia memegang PPAP Engineer Supervisor (semua customer) dan Project Leader (Project Group sendiri) sekaligus.

**Peran baru** bisa ditambah admin di layar **Admin → Roles** (D-199), misalnya untuk staf DCC bila dokumen kontrol dipisah lagi dari PPAP Engineer.

## 3. Matriks hak akses per peran (isi awal D-199, migrasi 00049)

✓ = punya hak, — = tidak, **(PG)** = cakupan Project Group sendiri. Sumber lengkapnya `docs/rencana/matriks-peran-dan-hak.md`.

| Kode | Hak | Use case | PPAP Engineer | PPAP Eng. Supervisor | Project Leader | Manager | Viewer | Admin |
|---|---|---|---|---|---|---|---|---|
| **L-1** | Lihat dokumen rilis dan halaman verifikasi, termasuk Released documents dan tab Check baca saja (D-195) | UC-56, UC-57 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-2** | Lihat register nomor, riwayat dan diff, diagram PFD, dasbor, review berkala, dan master data (D-103) | UC-12, UC-37, UC-47, UC-64 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-3** | Ekspor Excel dan PDF dokumen rilis | UC-54 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **L-4** | Lihat draft dan permintaan perubahan yang sedang berjalan | UC-47 | ✓ | ✓ | ✓ (PG) | ✓ | — | — |
| **L-5** | Lihat laporan kinerja PPAP Engineer (D-197) | UC-65 | ✓ | ✓ | — | ✓ | — | — |
| **E-1** | Daftarkan nomor dokumen, termasuk sebagai set | UC-10, UC-11 | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-2** | Buat, rakit, dan edit dokumen draft; tandai model EOL (D-200) | UC-30 sampai UC-35, UC-38, UC-39, UC-25 | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-3** | Ajukan perubahan dan isi daftar periksa dampak | UC-40, UC-42 | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-4** | Kerjakan draft revisi, yokoten, dan perubahan surut SQAM | UC-43 sampai UC-46 | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-5** | Cetak untuk tanda tangan, catat dokumen dikembalikan, kirim ke tanda tangan digital (fase 3) | UC-50, UC-51 | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-6** | Kelola master proses dan process template | UC-04, UC-21 sampai UC-23, UC-25 | ✓ | ✓ | — | — | — | — |
| **E-7** | Catat rilis | UC-52 | ✓ | ✓ | — | — | — | — |
| **E-8** | Ubah status nomor dan impor nomor lama | UC-13, UC-14 | ✓ | ✓ | — | — | — | — |
| **E-9** | Cetak salinan terkendali | UC-53, UC-55 | ✓ | ✓ | — | — | — | — |
| **E-10** | Migrasikan dokumen lama | UC-60 sampai UC-63 | ✓ | ✓ | — | — | — | — |
| **V-1** | Verifikasi permintaan perubahan | UC-41 | — | ✓ | ✓ (PG) | — | — | — |
| **V-2** | Beri tanda dicek di perbandingan PFMEA dan CP | UC-36 | ✓ | ✓ | — | — | — | — |
| **V-3** | Setujui (terbitkan) form set dan versi format bermetode AP | Form set (D-139) | — | ✓ | — | — | — | — |
| **S-1** | Tanda tangan digital pada slot rute approval (D-198, dipakai fase 3) | — | ✓ | ✓ | ✓ (PG) | ✓ | — | — |
| **G-1** | Beri catatan di tampilan genba (D-137, dipakai saat tampilan genba dibangun) | — | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| **A-1** | Kelola pengguna, peran dan isinya, service account, dan alamat email orang lain | UC-02, UC-70, UC-80 | — | — | — | — | — | ✓ |
| **A-2** | Kelola master data, jenis dokumen dan perubahan, rute approval, dokumen terkait, dan format template | UC-03, UC-05, UC-06, UC-09, UC-20, UC-24 | — | — | — | — | — | ✓ |
| **A-3** | Kelola pengaturan sistem, termasuk mode migrasi dan email notifikasi | UC-07, UC-71 | — | — | — | — | — | ✓ |
| **A-4** | Lihat audit log | UC-08 | — | — | — | — | — | ✓ |

Hak di matriks ini adalah isi awal. Admin menambah atau mengurangi hak sebuah peran di layar **Admin → Roles**, menambah peran baru (nama Inggris dan Indonesia, boleh menyalin hak peran lain), mengganti nama peran, dan menonaktifkan peran yang tidak dipegang akun mana pun (D-199). Peran tidak pernah dihapus. Yang terkunci di layar, kode, dan database: isi peran Admin dan Viewer, hak admin (A-1 sampai A-4) di peran kerja, dan hak dasar L-1 sampai L-3. Peran lama sebelum D-199 (Penyusun, Pengelola template, Pemeriksa PFMEA–CP, Verifikator, Penyetuju form, Dokumen kontrol, Petugas migrasi) nonaktif; pemegangnya dipindah oleh migrasi 00049.

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
10. Hak bercakupan Project Group (peran Project Leader, dan peran lain yang diizinkan admin dibatasi ke Project Group) hanya berlaku untuk operasi pada satu customer. Operasi tanpa customer ditolak untuk hak itu; daftar lintas customer menyaring per customer (D-118).

Aturan 9 dan 10 diputuskan 30 Oktober 2026. Akun admin yang lupa password atau terkunci dibuka lewat perintah di komputer server, bukan lewat hak di katalog (D-101).

## 5. Contoh: DCC diberi hak edit di masa depan

1. Admin, lewat akun admin, membuka Admin → Roles, menambah peran "Document control" (salin hak dari PPAP Engineer, lalu hilangkan centang hak yang tidak perlu), lalu memberikannya ke akun pengguna DCC (D-199).
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
