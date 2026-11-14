# D-130 · Master dokumen terkait

- **Tanggal:** 14 November 2026
- **Pekerjaan:** D-130 (diputuskan 8 November 2026, E-77) dan D-156 (ide pemilik 14 November 2026: "ada master data nya, jadi nanti bisa kita buat sebagai default dengan memilih dokumen yang tercatat di data master, misalkan untuk proces receving secara default menggunakan doc SW-ZZ-001, SW-ZZ-002 dan SW-ZZ-003"). Tidak ada di milestone; dikerjakan atas persetujuan pemilik ("ya, mulai D-130").
- **Aturan yang dipakai:** D-130, D-156, R-DOC-11, D-129 (cakupan), D-150 (key kolom `relevant_document`), D-154 (referensi `Process_Flow_PCBA_Generic.md`, bagian 3 register dan bagian 7 matriks dokumen × proses).

## 1. Tujuan dan apa yang dibuat

1. **Halaman Template library → Referenced documents.** Pengelola template mencatat dokumen: nomor (disimpan huruf besar, tetap), judul, level, revisi, tanggal revisi. Dokumen tidak pernah dihapus; yang tidak dipakai lagi dinonaktifkan.
2. **Bawaan per proses.** Tombol **Processes** menautkan dokumen ke proses dengan cakupan umum, customer, group model, atau model, seperti cakupan poin template. Untuk satu model semua tautan yang berlaku **dijumlahkan** (bukan "yang paling spesifik menang" seperti varian poin), karena satu proses memang memakai beberapa dokumen.
3. **Pengisian otomatis.** Saat dokumen dirakit, kolom Relevant Document baris pertama setiap proses diisi nomor-nomor itu, satu per baris, urut nomor. Sel yang sudah berisi tidak pernah ditimpa. Tombol **Fill relevant documents** di dokumen melakukan hal yang sama untuk baris yang diketik sendiri.
4. **Tidak di poin template.** Kolom Relevant Document ditolak di isi poin dan tidak tampil di editor **Columns**, dan tampilan banding (M4.8) tidak menghitungnya sebagai perbedaan dari template.
5. **Keputusan pemilik yang ikut dicatat:** D-155 (E-93: dua pola satu dokumen untuk beberapa model; bagian per PCB dicetak berurutan dalam satu CP dengan judul per PCB, belum dibangun).

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00028_referenced_documents.sql` |
| Aturan | `internal/template/refdoc.go` (`RefDocService`, `ReferencedNumbers`), `internal/document/refs.go` (`FillReferences`, `fillReferences`) |
| API | `/api/v1/referenced-documents` (daftar, tambah, ubah, tautan), `/api/v1/documents/{id}/references` |
| Layar | `web/src/routes/referenced-documents/+page.svelte`, tombol isi di halaman dokumen |
| Test | `TestReferencedDocuments`, `TestPointContentRejectsRelevantDocument`, `TestFillReferences`, `TestRefDocEndpoints`, tabel izin, `m4-compose.test.ts` |

## 2. Cara mengulang dari nol

1. Tulis `TestReferencedDocuments`: tambah dokumen, nomor kembar ditolak, tautan umum, customer, dan model; jumlahkan untuk tiga model; nonaktifkan dan lepas tautan.
2. Migrasi 00028. Indeks unik tautan memakai `NULLS NOT DISTINCT` supaya tautan umum (semua sasaran kosong) juga tidak bisa ganda.
3. Tulis `TestFillReferences`: rakit, periksa baris pertama proses, ubah sel, isi ulang tidak menimpa, baris ketik sendiri terisi, PFMEA ditolak.
4. Kontrak API, handler, halaman, teks dua bahasa, menu, dan daftar halaman di test responsif.
5. Coba sendiri: Template library → Referenced documents → tambah `SW-ZZ-103` → Processes → pilih Receiving → Add process. Lalu rakit CP yang memakai Receiving.

## 3. Konsep yang dipelajari

- **`NULLS NOT DISTINCT`.** Biasanya PostgreSQL menganggap dua NULL berbeda, jadi indeks unik membiarkan dua baris dengan kolom kosong. Sejak PostgreSQL 15, `NULLS NOT DISTINCT` membuat NULL dianggap sama.
- **Menjumlahkan vs memilih.** Varian poin memilih satu teks yang paling spesifik. Dokumen terkait menjumlahkan semua yang berlaku. Aturan cakupannya sama (`l.scope = 'general' OR l.customer_id = … OR …`), cara menggabungkannya berbeda.
- **Isi bawaan yang tidak menimpa.** Sistem hanya mengisi sel kosong, jadi penyusun tetap memegang keputusan akhir.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Nomor tetap sesudah dibuat | Nomor adalah identitas dokumen di WI; revisi yang berubah, nomornya tidak |
| Nonaktif, bukan hapus | Dokumen lama tetap terlihat di riwayat dan audit log |
| Hanya nomor yang ditulis ke sel | Kolom Relevant Document di CP memuat nomor (F-12); judul dan revisi ada di master |
| Satu nomor per baris di sel | Mudah dibaca dan dicetak |

## 5. Asumsi

- ASUMSI E-96: yang mengubah Pengelola template (E-6).
- Model yang dipakai model pertama dokumen (ASUMSI E-93, D-155 pola 1).
- Isi awal dari register referensi (bagian 3 file `.md`) belum diimpor otomatis; dicatat lewat layar.

## 6. Temuan desain baru

- **E-96** (celah): siapa yang mengubah master dokumen terkait.

## 7. Status test

Dicatat di pesan commit: `go test -p 1 ./...`, svelte-check, Vitest, dan E2E dijalankan sebelum commit.

## 8. Latihan tulis tangan

Tanpa AI, di psql: buat tabel `tautan (dokumen text, proses text, customer text)` dengan unique `(dokumen, proses, customer)`. Coba masukkan dua kali `('SW-ZZ-103', 'Receiving', NULL)`, catat hasilnya. Lalu buat ulang dengan `UNIQUE NULLS NOT DISTINCT (dokumen, proses, customer)` dan ulangi. Jelaskan dengan kata-kata sendiri bedanya.