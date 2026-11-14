# D-155 · Bagian per PCB

- **Tanggal:** 14 November 2026
- **Pekerjaan:** D-155 pola 2 (E-93), dengan referensi pola D-159: control plan nyata SC-KT-040-00 (PT. Indonesia Koito, model 737D LED HL). Persetujuan pemilik: "bangun untuk E-93 pola 2 dan lanjut D-155".
- **Keputusan yang ikut dicatat:** D-157 (E-96: master dokumen terkait diubah PPAP Engineer), D-158 (Q-61: nama orang asli boleh dipakai), D-159 (referensi SC-KT-040-00).

## 1. Pola dari dokumen nyata (D-159)

- Satu **sampul**: satu nomor, customer, model, daftar semua part name dan part number, satu riwayat revisi, satu persetujuan.
- Satu **lembar per bagian**: PCB A (10029-8A15D), Heatsink Assy R/L (10067-8A1A8, 10167-8A1A8), Base Assy R/L (10063-8A1A8, 10163-8A1A8). Header tiap lembar mengulang data sampul dengan Product = nama bagian.
- Daftar proses **mulai dari nomor 1** di setiap bagian dan bisa berbeda (Router hanya di PCB A).

## 2. Apa yang dibuat

1. **Bagian** punya nama dan part number pilihan dari part number dokumen (satu bagian boleh dua part number, R dan L). Satu part number hanya milik satu bagian.
2. **Berlaku untuk seluruh set** (ASUMSI E-97): bagian yang sama di PFD, PFMEA, dan CP punya `section_key` yang sama. Menambah, mengganti nama, menggeser, atau menghapus bagian berlaku di ketiga dokumen.
3. **Bagian pertama** menampung baris yang sudah ada, jadi dokumen yang sudah berisi bisa dibagi tanpa kehilangan baris.
4. **Per bagian:** baris baru wajib menyebut bagiannya; perakitan dari process template (poin yang sama boleh dipakai di dua bagian, pengecualian dicatat per bagian); urutan dan nomor operasi (mulai lagi dari 10); dokumen terkait (baris pertama setiap proses di setiap bagian); tab Check (judul per bagian, tanda dicek per bagian); diagram PFD.
5. **Salin dari model lain** ikut membuat bagian yang sama namanya di set tujuan (tanpa part number, karena milik model sumber).
6. **Layar:** tab bagian di atas tabel (nama dan part number), tombol **Split into sections per PCB** / **Add section**, **Edit section**, geser kiri dan kanan, **Remove section** (hanya bagian kosong). Pilihan bagian tetap saat berpindah tab PFD, PFMEA, dan CP.

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00029_document_sections.sql` |
| Aturan | `internal/document/sections.go`; perubahan di `compose.go`, `order.go`, `refs.go`, `comparison.go`, `copy.go`, `document.go` |
| API | `/api/v1/documents/{id}/sections` (tambah, ubah, hapus); `sections` di dokumen, `section_id` di baris, `section_key` di perakitan dan tanda dicek, `section_id` di urutan |
| Layar | `web/src/lib/documents/SectionBar.svelte`, `rowsOfSection` di `documents.ts`, halaman dokumen, halaman Check |
| Test | `TestSections`, `TestComposeBySection`, `TestCopyWithSections`, `TestDocumentEndpoints`, tabel izin, `documents.spec.ts`, `m4-sections.test.ts` |

Yang **belum**: cetakan sampul dan header per bagian (Tahap 6, mengikuti pola D-159).

## 3. Cara mengulang dari nol

1. Tulis `TestSections` dulu: bagian pertama menampung baris lama; bagian kedua dengan dua part number; nama dan part number ganda ditolak; baris tanpa bagian ditolak; urutan per bagian mulai dari 10; ganti nama dan geser berlaku di PFD; hapus hanya bagian kosong.
2. Migrasi 00029. Kunci unik yang dulu per dokumen (poin template per baris, pengecualian, tanda dicek) kini menyertakan bagian, dengan `NULLS NOT DISTINCT` supaya dokumen tanpa bagian tetap terjaga.
3. Ubah query: kolom `section_id` ikut di baris baru, pengecualian, dan salinan; salinan memetakan bagian menurut namanya.
4. Tulis `sections.go`, lalu ubah perakitan, urutan, dokumen terkait, cek, dan salin.
5. Kontrak API, handler, `SectionBar.svelte`, teks dua bahasa, Help.

## 4. Konsep yang dipelajari

- **Satu kunci untuk banyak baris.** Setiap dokumen punya baris bagiannya sendiri (ID berbeda), tetapi `section_key` yang sama menyatukan "PCB A" di PFD, PFMEA, dan CP. Pola ini mirip `batch_id` pada nomor satu set (M2.4).
- **`ON CONFLICT` dengan indeks `NULLS NOT DISTINCT`.** PostgreSQL bisa memakai indeks unik seperti itu sebagai penentu konflik, jadi satu perintah `INSERT ... ON CONFLICT` tetap berlaku untuk dokumen dengan dan tanpa bagian.
- **Memakai lagi urutan yang sama.** Saat satu bagian diurutkan, barisnya memakai lagi nomor urut yang sudah mereka tempati di dokumen, jadi baris bagian lain tidak bergeser.

## 5. Asumsi

- ASUMSI E-97: bagian berlaku untuk seluruh set.
- Nomor urut tampilan (`#`) di tabel mulai dari 1 per bagian; urutan di database tetap satu untuk seluruh dokumen.
- Tanda dicek yang diberikan sebelum dokumen dibagi tidak dipindahkan ke bagian pertama; proses itu perlu dicek lagi per bagian.

## 6. Temuan desain baru

- **E-97** (celah): bagian untuk seluruh set atau CP saja, dan cetakan per bagian.

## 7. Status test

Dicatat di pesan commit: `go test -p 1 ./...`, svelte-check, Vitest, dan E2E dijalankan sebelum commit.

## 8. Latihan tulis tangan

Tanpa AI, di kertas: gambar dua tabel, `document_sections` dan `document_rows`, untuk satu set CP dan PFMEA dengan bagian PCB A dan Heatsink. Isi ID bagian yang berbeda per dokumen, tetapi `section_key` yang sama. Lalu tulis query SQL yang memberi semua baris PFMEA di bagian yang sama dengan satu baris CP tertentu, dengan bergabung lewat `section_key`.