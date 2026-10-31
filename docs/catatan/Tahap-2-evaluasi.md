# Evaluasi Tahap 2 · Penomoran dokumen

- **Tanggal:** 30 dan 31 Oktober 2026
- **Dasar:** instruksi pemilik "jika tidak ada kendala untuk lanjut ke M2, silahkan selesaikan hingga M2" (30 Oktober 2026), dan D-77 (kode dipakai untuk menguji desain).
- **Isi:** bahan keputusan untuk pemilik. Claude Code tidak memutuskan apa pun di sini. Rincian setiap temuan (opsi, dampak, dan asumsi) ada di `docs/evaluasi-desain.md`.
- **Status:** menunggu keputusan pemilik dan Gerbang 2. Tahap 3 dimulai 31 Oktober 2026 atas permintaan pemilik sebelum Gerbang 2, sebagai pengecualian D-37; Gerbang 2 tetap terbuka.

## 1. Yang sudah dibangun

| Milestone | Selesai bila (milestone) | Hasil |
|---|---|---|
| M1.7 Backup data lokal (dari Tahap 1) | Restore berhasil dan langkahnya tertulis | Selesai (D-119): backup terenkripsi dengan age, manifest SHA-256, dan restore ke database baru yang tidak pernah menimpa database lama |
| M2.1 Pola dan penghitung | Pola menghasilkan nomor seperti SC-KT-001, dan kode customer di nomor dicek (R-POR-01) | Selesai |
| M2.2 Alokasi atomik | 50 pendaftaran paralel menghasilkan 50 nomor berbeda tanpa lubang | Selesai, dengan kunci idempoten per akun |
| M2.3 Register dan status | Nomor tidak pernah dipakai ulang atau dihapus | Selesai: role aplikasi tanpa DELETE, dan UPDATE hanya kolom status |
| M2.4 Set dan format revisi | Revisi awal bisa -04 | Selesai, dengan nomor dilewati berstatus Skipped (ASUMSI E-64) |
| M2.5 Impor nomor lama | Impor satu customer menghasilkan laporan yang benar | Selesai, dengan pratinjau sebelum impor |

Hasilnya 10 migrasi database dan 67 operasi di kontrak `api/openapi.yaml`. Setiap milestone punya catatan di folder ini. UML yang diperbarui: U1, U5 (baru), dan U8 (baru); brief menjadi v1.17.

**Status test** (dijalankan ulang 31 Oktober 2026 di container cloud, setelah perbaikan E-69)

| Perintah | Hasil |
|---|---|
| `go vet ./...`, `gofmt -l .` | Bersih |
| `go test -v ./...` | Lulus di 15 paket: 201 fungsi test dan 290 sub-test, tanpa yang gagal |
| `npm --prefix web run check` | 0 error, 0 peringatan |
| `npm --prefix web test` | 74 test Vitest lulus |
| `npm --prefix web run test:e2e` | 25 test E2E lulus |

Test konkurensi penomoran (R-NUM-03) ada lima:
- `TestRegisterConcurrent`: 50 pendaftaran paralel;
- `TestRegisterSetConcurrent`: set dan nomor tunggal bersamaan;
- `TestImportConcurrent`: impor bertumpuk bersama pendaftaran dan set;
- `TestRegisterDuringCodeChange`: pendaftaran dan set bersamaan dengan ganti kode customer (E-69);
- `TestCodePositionConcurrent`: kode customer dan kode khusus didaftarkan bersamaan.

Semua perintah Windows di bagian 2 setiap catatan belum dicoba di komputer pemilik.

## 2. Apa yang ditunjukkan Tahap 2 tentang desain (D-77)

1. **Desain penomoran bisa dibangun.** Semua kriteria "Selesai bila" tercapai, dan tidak ada temuan yang menghentikan pekerjaan.
2. **Aturan inti terbukti di bawah beban bersamaan.** Penghitung di dalam transaksi, UNIQUE pada nomor, dan kunci idempoten (R-NUM-03) bertahan saat pendaftaran, set, dan impor berjalan bersamaan. Syaratnya satu: setiap transaksi yang mengunci beberapa penghitung harus memakai urutan yang sama (kode customer, lalu ID jenis dokumen). Aturan ini perlu ditulis untuk tim IT, dan sekarang tercatat di U8.
3. **Membangun sambil menguji menemukan bug yang tidak terlihat dari desain (E-69).** Saat menyiapkan Gerbang 2, test baru menemukan deadlock antara pendaftaran nomor dan ganti kode customer. Penyebabnya urutan kunci: foreign key mengunci baris customer lebih dulu dengan kunci lemah, lalu kuncinya dinaikkan di tengah transaksi. Bug ini sudah diperbaiki dan dibuktikan dengan test. Urutan kunci kini ditulis di brief (R-NUM-03) dan U8, supaya tim IT tidak mengulanginya.
4. **"Tidak pernah dihapus" bisa ditegakkan database, bukan hanya kode.** Role aplikasi tidak punya DELETE di register, dan UPDATE hanya untuk kolom status. Teks nomor tidak bisa diganti, bahkan oleh kode yang salah.
5. **Celah terbesar ada di batas dengan data lama:** nomor dilewati saat set (E-64), bentuk impor (E-65), serta nomor di luar pola dan nomor ganda (E-66). Ketiganya baru bisa diputuskan dengan benar setelah register Excel asli diperiksa.
6. **Model domain brief untuk pola nomor terlalu umum.** Brief menyebut pola per jenis dokumen yang bisa ditimpa per customer. Yang dibangun adalah satu pola dengan pemisah wajib (E-61), karena impor harus bisa membaca nomor kembali tanpa ragu. Brief v1.17 sudah mengikuti kode, dengan status [PROPOSED].
7. **Terhadap tujuan utama (revisi yokoten lebih mudah):** register dan set menjadi dasar dokumen (M4.1) dan yokoten (Tahap 5). Tidak ada asumsi Tahap 2 yang membuat yokoten lebih sulit. Satu hal perlu dijaga: nomor set yang sejajar memudahkan mencari PFD, PFMEA, dan CP satu model, jadi E-64 sebaiknya diputuskan sebelum Tahap 4.

## 3. Keputusan yang cukup dikonfirmasi

Untuk temuan di bawah, kode sudah memakai rekomendasi Claude Code. Jawab "setuju bagian 3" bila semuanya diterima, atau sebut ID yang ingin diubah.

| ID | Pertanyaan | Rekomendasi (sudah di kode) | Paling lambat |
|---|---|---|---|
| E-44 | Nomor berjalan pertama: 000 atau 001 | (a) 001; nomor lama 000 tetap bisa diimpor apa adanya | Sebelum data asli didaftarkan |
| E-60 | Jadwal, retensi, dan passphrase backup | (a) Task Scheduler menjalankan `revioten backup`; backup lama dihapus tangan; passphrase di `.env` dengan salinan di tempat lain | Sebelum data asli dipakai sehari-hari |
| E-61 | Pola nomor | (a) Satu pola untuk semua jenis dan customer, dengan pemisah wajib, sampai Q-05 menunjukkan customer berpola khusus | Q-05 |
| E-63 | Kunci idempoten | (a) Dibuat klien, berlaku per akun, disimpan selamanya; kunci sama dengan isi berbeda ditolak | M9.4 untuk API integrasi |
| E-64 | Nomor yang dilewati saat set | (a) Status Skipped, tidak pernah diberikan lagi | Sebelum Tahap 4 |
| E-67 | Customer nonaktif | (a) Tidak mendapat nomor baru dan tidak bisa diimpor | — |
| E-68 | Perubahan status nomor dengan tangan | (a) Hanya Void (alasan wajib) dan Obsolete (alasan opsional) | M4.1 |
| E-69 | Urutan kunci transaksi penomoran | (a) Customer dikunci paling awal, lalu penghitung, lalu rantai audit; aturan ini ditulis di brief dan U8 untuk tim IT | Sebelum UML diserahkan (M8.4) |

## 4. Keputusan yang butuh bahan atau pilihan dari pemilik

1. **E-65 dan E-66 · Register Excel asli.** Periksa satu register Excel nomor lama, misalnya untuk satu customer. Yang perlu diketahui:
   - kolom apa saja yang ada;
   - apakah ada nomor dengan bentuk lain, misalnya dua digit atau tanpa pemisah;
   - apakah ada nomor ganda yang keduanya masih berlaku.

   Isinya tidak perlu dibagikan ke saya. Cukup jawab tiga hal itu, supaya rekomendasi (a) bisa dikonfirmasi atau diganti (b) atau (c). Paling lambat sebelum impor data asli dan sebelum M4.1.
2. **E-62 · Pengingat nomor Reserved.** Berapa hari nomor boleh Reserved sebelum diingatkan, dan lewat apa (layar saja, atau email)? Apa kolom master document list untuk ekspor? Angka dan format ini tidak boleh saya karang.
3. **Dari Tahap 1, masih terbuka:**
   - E-30 (field master data) dan E-33 (pencocokan proses atau sub-proses): menunggu dokumen asli.
   - E-35: mana yang menang bila penyusun memilih rute lain (sebelum Tahap 5).
   - E-37 butir 4 (D-114): bolehkah PFD, PFMEA, dan CP dinomori dengan kode khusus seperti ZZ? Kode sekarang menolaknya.
   - Q-46: siapa yang mengelola group model.
   - E-54: tanggal di kepala milestone, untuk sesi desain.

## 5. Hal lain yang perlu Anda ketahui

1. **Branch.** Sesi ini sempat bekerja di branch `claude/repo-docs-review-f5utv9`, karena lingkungan sesi cloud mewajibkannya. Atas permintaan pemilik (31 Oktober 2026), semua commit sudah di-push ke `main`, dan CLAUDE.md kini menegaskan: selalu push ke `main`, juga di sesi cloud.
2. **Satu commit dengan test gagal.** Commit `5508b51` (perbaikan E-22) membuat `TestAuditLockAndDisable` gagal, karena test itu masih mengharapkan akun admin dinonaktifkan. Test itu dibetulkan di commit berikutnya, `664df85`. Commit lain lulus semua test.
3. **U5 dan U8 belum dirender.** Container cloud tidak punya PlantUML dan Graphviz, dan permintaan mengunduhnya ditolak sistem izin lingkungan. File sumber `.puml` sudah ada. Perintah render ada di `docs/uml/README.md`.
4. **Bug deadlock sudah diperbaiki (E-69).** Kode M2.1 sampai M2.4 punya bug ini sejak dibuat, tetapi test lama tidak menangkapnya karena tidak menjalankan ganti kode bersamaan dengan pendaftaran. Perbaikan dan test-nya ada di commit penutup Tahap 2. Catatan M2.2 bagian 10 menjelaskannya untuk latihan.
5. **Data asli tidak dipakai.** Semua test dan contoh memakai kode contoh (KT, KU, NB, LG) dan nama jabatan, tanpa nama orang.

## 6. Gerbang 2 · untuk Anda kerjakan tanpa AI

Milestone: "Ulangi uji paralel dengan skenario baru yang Anda tulis sendiri." Pilih satu dari dua skenario berikut; keduanya belum ada di test. Skenario ganti kode customer tidak lagi menjadi pilihan, karena sudah menjadi test tetap (`TestRegisterDuringCodeChange`, E-69).

**Pilihan 1: latihan M2.2.** Dua puluh pendaftaran PFD dan dua puluh pendaftaran PFMEA berjalan bersamaan, dan setengahnya dikirim ulang dengan kunci idempoten yang sama. Kriteria lulusnya ada di catatan M2.2 bagian 9.

**Pilihan 2: dua dokumen kontrol membatalkan nomor yang sama.** Sepuluh akun berperan Document control menekan Void untuk satu nomor Reserved pada waktu yang sama.

Kriteria lulus:
1. Test baru di `internal/numbering/registry_test.go`, misalnya `TestVoidConcurrent`, ditulis dengan `sync.WaitGroup` dan `wg.Go`, seperti `TestRegisterConcurrent`.
2. Tepat satu pembatalan berhasil. Sembilan lainnya ditolak dengan `*problem.ConflictError` (409, `status:not_allowed`), tanpa error lain.
3. Audit log berisi tepat satu entri `document_number.voided` untuk nomor itu, dan alasan yang tersimpan adalah alasan dari pembatalan yang berhasil.
4. `go test ./internal/numbering -run TestVoidConcurrent -count=10` lulus sepuluh kali berturut-turut.
5. Test gagal bila Anda menghapus `LockDocumentNumber` dari `changeStatus` (coba, lalu kembalikan). Bila tidak gagal, jelaskan di catatan Anda kenapa.

Petunjuk tanpa kode: baca `changeStatus` di `internal/numbering/registry.go` dan komentar `LockDocumentNumber` di `sql/queries/document_numbers.sql` sebelum menulis test.
