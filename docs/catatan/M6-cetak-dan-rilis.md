# Tahap 6 · Cetak dan rilis manual (M6.1 sampai M6.9)

- **Tanggal:** 14 November 2026
- **Aturan yang dipakai:** R-PRT-01 sampai R-PRT-04, R-PRT-07, R-APR-02, R-APR-07, R-APR-08, R-APR-09, R-APR-16, R-APR-17, R-REV-04, R-REV-08, R-REV-09, D-46, D-49, D-52, D-53, D-59, D-60, D-62, D-68, D-76, D-121, D-163, D-166.
- **Persetujuan pemilik:** "jika tidak ada masalah di M5, silahkan lanjutkan ke M6 hingga selesai" (14 November 2026).
- **Tidak dibangun (masih usulan):** konfirmasi slot saat rilis (R-APR-10), pembekuan rute (R-APR-11), daftar salinan terkendali yang ditarik (R-PRT-06), label PENDING (R-PUB-07), pembacaan QR dari scan. Lihat E-110.
- **Belum selesai menurut kriteria milestone:** M6.2 "hasil cetak di printer asli rapi" dan Gerbang 6 butuh pemilik mencetak di printer pabrik.

## 1. Apa yang dibuat

| Milestone | Hasil |
|---|---|
| M6.1 Layout sebagai data | `print_layouts` (layout bawaan per jenis dokumen) dan `customer_print_layouts`. Admin (A-2) menambah dan mengubah layout di Template library → **Print layouts**, dan memilih layout lain per customer. Slot tanda tangan dibentuk dari rute dokumen (D-46): layout yang sama mencetak rute sampai QA QC Manager maupun sampai GM. |
| M6.2 Excel A3 dan A4 | excelize: sheet Cover dan sheet isi, judul kolom berulang (`Print_Titles`), area cetak, muat satu halaman lebar, header berisi watermark, footer berisi nomor, revisi, status, halaman, dan hash. |
| M6.3 PDF | HTML dari model cetak yang sama, dicetak Edge/Chrome headless (hasil uji S5, ASUMSI E-108). |
| M6.4 Watermark | Draft selalu UNCONTROLLED COPY; salinan terkendali CONTROLLED COPY No. n. |
| M6.5 QR dan hash | Halaman verifikasi `/verify?id=` (L-1): berlaku, digantikan, draft, dibatalkan, atau usang, dengan SHA-256. QR menuju halaman itu bila saklar di Settings menyala (bawaannya mati, D-62); alamat dari `PUBLIC_BASE_URL`. |
| M6.6 Catatan cetak | `print_logs`: siapa, kapan, revisi, tujuan, format, nomor salinan, dan hash isi. Tampil di halaman dokumen. |
| M6.7 Rilis manual | Cetak untuk tanda tangan (E-5) mengisi tanggal rilis (D-59). Dokumen kontrol (E-7) mengunggah scan bertanda tangan; **rilis tanpa scan ditolak**. Pencatat rilis tetap bukan penyusun; cetak dan unggah scan tidak dihitung menyusun (D-166). |
| M6.8 Tabel riwayat revisi | Cover: No, Date, Revision, Description of Change dari revisi dan alasan perubahannya. |
| M6.9 Status dan footer | Footer PDF: status (DRAFT, EFFECTIVE, SUPERSEDED), hash isi, dan untuk revisi terbit catatan rilis (tanggal, pencatat, nomor catatan, scan tersimpan). Baris yang berubah di revisi yang dicetak diberi ▲n (R-REV-09). |

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00040_printing.sql` |
| Query | `sql/queries/printing.sql` |
| Paket cetak | `internal/printout` (`layout.go`, `model.go`, `xlsx.go`, `html.go`, `pdf.go`, `service.go`, `layouts.go`) |
| Rilis | `internal/document/release.go` (scan wajib) |
| API | `POST /documents/{id}/print`, `GET /documents/{id}/prints`, `POST /documents/{id}/scan`, `GET /documents/{id}/revisions/{revision}/scan`, `GET /verify/{revisionId}`, `/print-layouts`, `/customers/{id}/print-layouts` |
| Layar | `PrintPanel.svelte`, `RevisionsPanel.svelte` (scan), `/verify`, `/print-layouts`, Settings (QR), bab Help "Print and release" |
| Test | `internal/printout` (model, Excel, HTML, PDF sungguhan, layanan), tabel izin, `m6-printing.test.ts`, M5.5 E2E diperbarui |

## 2. Cara mengulang dari nol

1. Uji S5 (`docs/catatan/S5-uji-pdf.md`).
2. Tulis `TestBuild`, `TestXLSX`, `TestHTMLAndPDF` dengan bahan cetak kecil, lalu `model.go`, `xlsx.go`, `html.go`, `pdf.go`.
3. Migrasi 00040, query, `TestPrintService` (draft, tanda tangan, scan wajib, salinan bernomor, catatan, verifikasi, layout customer), lalu `service.go` dan `layouts.go`.
4. Kontrak API, handler, tabel izin, layar, teks dua bahasa, Help, E2E.

## 3. Konsep yang dipelajari

- **Satu model, dua keluaran.** Model cetak netral (`Sheet`) dibentuk sekali; Excel dan HTML/PDF hanya menulisnya. Aturan isi tidak tergandakan.
- **Layout sebagai data.** Hal yang sering berubah per customer (kertas, huruf, lebar kolom) disimpan di database, bukan di kode.
- **Browser sebagai mesin cetak.** CSS `@page`, `thead` berulang, dan `position: fixed` untuk watermark.
- **Bukti di setiap langkah.** Catatan cetak menyimpan hash isi yang dicetak; scan menyimpan SHA-256; halaman verifikasi menunjukkan apakah cetakan masih berlaku.

## 4. Asumsi

- ASUMSI E-108 (mesin PDF), E-109 (scan), E-110 (rincian cetak), E-111 (cakupan layout).

## 5. Status test

- `go test -p 1 ./...` lulus (termasuk PDF sungguhan lewat Edge), Vitest lulus, E2E 57 lulus (14 November 2026).

## 6. Latihan tulis tangan

Tanpa AI, buat satu halaman HTML dengan `@page { size: A4 landscape; margin: 10mm }`, satu tabel 80 baris ber-`thead`, lalu cetak ke PDF dengan perintah Edge di catatan S5. Periksa bahwa judul tabel muncul di setiap halaman.
