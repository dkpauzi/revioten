# S5 · Uji mesin PDF (sekali pakai)

- **Tanggal:** 14 November 2026
- **Tujuan:** memilih jalur PDF sebelum milestone cetak (R-PRT-03, Q-10).
- **Kriteria:** huruf, tinggi baris teks terbungkus, pemisah halaman, judul tabel berulang, A3.

## Yang diuji

| Opsi | Diuji? | Hasil |
|---|---|---|
| A. Excel ke PDF lewat LibreOffice/Gotenberg | Tidak | Butuh unduhan besar (LibreOffice atau image Docker Gotenberg); tidak diunduh tanpa izin. |
| B. HTML ke PDF lewat Chromium/Edge headless | Ya | Edge di komputer pemilik: tabel CP 60 baris, teks panjang terbungkus, A3 landscape. Hasil 3 halaman 1191 × 842 pt, huruf Arial dan Arial Bold ter-embed, judul tabel (`thead`) berulang, watermark tetap di setiap halaman. |
| C. Pustaka PDF Go | Tidak | Tata letak harus ditulis di kode; disimpan sebagai cadangan. |
| D. Excel siap cetak | Ya, lewat excelize | A3/A4, judul berulang, area cetak, muat satu halaman lebar, footer bernomor halaman. |

## Kesimpulan kerja (ASUMSI E-108)

B sebagai jalur PDF utama, D sebagai cadangan yang selalu ada. Catatan: lewat baris perintah, Chromium tidak bisa menulis nomor halaman di footer PDF; nomor halaman ada di Excel. Di Windows, proses browser bisa keluar lebih dulu sementara proses anaknya masih menulis PDF, jadi program menunggu sampai berkasnya stabil.

## Cara mengulang

1. Simpan satu HTML dengan `@page { size: A3 landscape }` dan tabel ber-`thead`.
2. `msedge --headless=new --disable-gpu --no-pdf-header-footer --user-data-dir=<folder> --print-to-pdf=<out.pdf> file:///<page.html>`
3. Periksa `/MediaBox` di PDF dan buka hasilnya.
