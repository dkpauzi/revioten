# C1 · Kerangka workbook dokumen

- **Tanggal:** 21 November 2026
- **Keputusan:** D-145 (workbook seperti Excel), D-210 (rencana C1 sampai C5). Temuan: E-135.
- **Rencana lengkap:** `docs/rencana/workbook-form.md`

## Tujuan

Halaman dokumen belum seperti desain workbook: panel Print, Revisions, Signature, dan lainnya bertumpuk di atas tabel, dan tab sheet ada di tengah halaman. C1 membangun kerangkanya. Lembar seperti form kertas menyusul di C2.

## Yang dibuat

| Bagian | Isi |
|---|---|
| Kepala | Nomor dokumen, model dan jenis, badge form set, status, dan revisi. Kotak **Next step** di kanan berisi satu langkah berikutnya beserta tombolnya. |
| Langkah berikutnya | `nextStep` di `web/src/lib/documents/frame.ts`, diuji `frame.spec.ts`. Urutannya: dokumen terbit → buka Changes; tanda tangan lengkap → catat rilis; menunggu tanda tangan → lihat Signature; dokumen lama belum dicocokkan → Baseline; dokumen kosong → Compose; draft berisi → Signature (bila ada rute) atau Print. Akun yang tidak bisa melakukannya tidak mendapat langkah. |
| Toolbar | Kelompok Rows, Template, Review, Output, dan Legacy document, seperti pita Excel. Label tombol pendek (Compose, Print, Revisions). Nama lengkapnya, yaitu judul panel, menjadi tooltip dan nama untuk pembaca layar. |
| Laci samping | `Drawer.svelte`: satu panel terbuka di kanan lembar, ditutup lewat **Close** atau Escape. Di layar di bawah 1024px laci tampil di atas lembar. |
| Tab sheet | `SheetTabs.svelte` menempel di bawah layar: **Cover**, PFD, PFMEA, CP, lalu Check. Dipakai halaman dokumen dan halaman Check. |
| Sheet Cover | `?sheet=cover`: data dokumen yang sama untuk satu set (customer, model, part number, form set, rute, revisi). Bentuk sampul form menyusul di C4. |
| Lebar | Halaman dokumen memakai lebar layar penuh. |
| Tur | Batas waktu ikut kecepatan (fast 15, normal 60, slow 90 menit). Pita keterangan pindah ke atas, lebih besar, dan teks panjang turun baris. |
| Panduan tur | Perintah memakai `npm.cmd` (PowerShell memblokir `npm.ps1`), ditambah satu baris di tabel masalah. |

## Cara mencoba

1. Jalankan server, lalu login dengan akun PPAP Engineer.
2. Buka **Documents → Documents**, pilih customer, lalu buka satu dokumen.
3. Lihat kotak **Next step** di kanan atas, lalu tekan tombolnya.
4. Tekan **Print** di toolbar. Panel Cetak terbuka di kanan; tekan **Close**.
5. Tekan **Cover** di tab bawah, lalu kembali ke PFD, PFMEA, atau CP.

## Konsep yang dipelajari

- **Logika dipisah dari tampilan.** Aturan "langkah berikutnya" adalah fungsi biasa, jadi bisa dites dengan Vitest tanpa browser.
- **Nama aksesibel dan label.** `aria-label` pada tombol ikut ditemukan `getByLabel` di Playwright dan bertabrakan dengan field yang labelnya mirip ("Process template"). Nama lengkap ditulis sebagai teks tersembunyi (`.visually-hidden`), dan teks pendeknya diberi `aria-hidden`.
- **Panel di dalam tata letak, bukan mengambang.** Laci `position: fixed` menutupi toolbar dan tombol Log out. Laci `position: sticky` di dalam flexbox berada di samping lembar dan tetap terlihat saat halaman digulir.

## Keputusan dan alasan

- Panel tetap komponen yang sama. Hanya tempatnya yang pindah ke laci, jadi aturan dan API tidak berubah.
- Satu laci sekaligus, supaya lembar tetap lebar.
- Tabulator tetap dipakai di C1. Penggantinya dibangun di C2 (D-210).

## Asumsi

- Tombol Signature tampil untuk draft dan dokumen yang sedang atau sudah ditandatangani digital.
- Sheet Cover belum menampilkan riwayat revisi dan slot persetujuan; itu bagian C4.

## Temuan desain baru

- **E-135** · Workbook D-145 baru terbangun sebagian (celah, diputuskan D-210).

## Status test

| Test | Hasil |
|---|---|
| `npm --prefix web test` | 174 lulus, termasuk 9 kasus `nextStep` |
| `npm --prefix web run check` | 0 error, 0 peringatan |
| `npm --prefix web run test:e2e` | 69 lulus, termasuk test baru "C1 (D-145)" di `m4-documents.test.ts`; panel dibuka lewat `openTool` |
| `npm --prefix web run help:shots` | Lulus; gambar Documents dan panel memperlihatkan workbook |
| Tur (`fast`, tanpa jendela) | Lulus |
| Test Go | Tidak ada perubahan Go |

## Latihan tulis tangan

Tulis ulang fungsi `nextStep` tanpa melihat kode. Lalu tulis satu kasus test baru: dokumen lama yang sudah diverifikasi dan masih punya draft kosong. Langkah apa yang diharapkan, dan mengapa?
