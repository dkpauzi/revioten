# D-219 · Simbol alur dan diagram PFD (bagian B)

- **Tanggal:** 22 November 2026
- **Keputusan:** D-219 (brief v1.48), menutup bagian B temuan E-144.
- **Persetujuan pemilik:** "Bagian B, simbol alur". Rework: "Kotak REWORK saja". Cetakan PFD: "Halaman diagram".
- **Aturan yang dipakai:** D-122 (kolom alur diambil dari PFD), D-161 (aturan alur NG), D-211 (rencana halaman bersama), R-DOC-09.
- **Acuan:** shape di Excel referensi SH-TR-020-02 dan SC-TR-020-04 (`xl/drawings/drawing2.xml`), dibaca jenis, posisi, dan teksnya.

## 1. Tujuan

Menyamakan simbol alur dengan dokumen Tokairika:
- di kolom Process Flow CP;
- di cetakan PFD, yang sebelumnya berupa tabel.

## 2. Temuan dari referensi

| Hal | SC-TR-020-04 (CP) | SH-TR-020-02 (PFD) |
|---|---|---|
| Proses | kotak berisi nomor | kotak berisi "nomor. nama" |
| Inspeksi | belah ketupat berisi nomor | belah ketupat yang memuat kotak "nomor. nama" |
| NG | kotak di kiri atas, garis siku naik ke kotak, label NG di bawah | kotak di kiri sejajar, garis mendatar berpanah |
| SCRAP, NCP | predefined process | predefined process |
| REWORK | — | kotak proses biasa (hanya di legenda) |
| OK | di bawah belah ketupat | di bawah belah ketupat |
| Router 6.1 | proses dengan NG ke SCRAP, tanpa OK | sama |
| Panah | ada di setiap garis | ada di setiap garis |
| Tata letak | satu kolom di tabel | halaman A4 tegak, legenda, dua kolom, konektor A |

## 3. Apa yang dibuat

| Bagian | Isi |
|---|---|
| `flowsvg.go` | Kolom Process Flow CP (pratinjau dan PDF): kotak NG di kiri dengan garis siku berpanah, OK hanya untuk belah ketupat, panah antar simbol, alternate process segi enam. |
| `diagram.go` | `DiagramLayout` menempatkan langkah ke kolom dan halaman, dengan konektor huruf yang sama di ujung dan pangkal. `DiagramSVG` dan `DiagramLegendSVG` menggambar halaman dan legenda. `BuildPlan` memakai jalur diagram untuk PFD. |
| `xlsxdraw.go`, `xlsxdiagram.go` | Shape Excel untuk CP dan PFD. Posisi tiap shape ditulis di namanya, lalu jangkarnya diganti menjadi jangkar mutlak. Kepala panah dan teks rata tengah ditambahkan ke XML drawing. |
| HTML, pratinjau, API | Halaman PFD menampilkan legenda dan diagram, tanpa tabel. API pratinjau menambah `diagram_svg`, `legend_svg`, dan `legend_mm`. |
| Migrasi 00055 | Layout bawaan PFD menjadi A4 tegak (hanya bila masih mendatar). |
| Test | `flowsvg_test.go`, `diagram_test.go` (tata letak, SVG, bungkus teks, HTML, Excel). Data TR-020 di test kini memberi Router cabang NG. E2E `m4-preview` lulus dengan diagram. |

## 4. Cara mengulang dari nol

1. Baca shape referensi: buka xlsx sebagai zip, `xl/drawings/drawing2.xml`, lalu catat `prstGeom`, `from`/`to`, dan teksnya.
2. Tulis test `flowsvg_test.go` dan `diagram_test.go`, lalu `diagram.go` dan perubahan `flowsvg.go` sampai lulus.
3. Sambungkan ke `BuildPlan`, `pagesOf`, template HTML, `Preview`, kontrak OpenAPI, dan `PreviewSheet.svelte`.
4. Excel: `xlsxdraw.go` (penggambar dan perbaikan XML), `xlsxdiagram.go`, lalu `flowShapes` CP.
5. Periksa mata:
   - `REVIOTEN_SAMPLE_DIR=<folder> go test -short ./internal/printout -run TestSampleTokairika`;
   - potret HTML dengan Chrome headless;
   - untuk Excel, simpan sebagai HTML lewat Excel lalu potret.

## 5. Konsep yang dipelajari

- **Satu tata letak, tiga gambar.** `DiagramLayout` hanya menghitung posisi dalam milimeter. SVG dan Excel membaca hasil yang sama, jadi pratinjau, PDF, dan Excel tidak bisa berbeda.
- **Excel menghitung lebar kolom berbeda dengan excelize.**
  - excelize memakai 8 piksel per huruf dan tinggi baris 4/3,4 point.
  - Excel memakai 7 piksel per huruf dan 4/3 point.
  - Shape yang dijangkarkan ke sel buatan excelize jadi memanjang. Karena itu posisinya dihitung dengan rumus Excel, lalu jangkarnya ditulis sebagai `absoluteAnchor`.
- **Melengkapi berkas yang dibuat pustaka.** Bila pustaka belum mendukung sesuatu (kepala panah, teks rata tengah), XML hasilnya bisa dilengkapi sesudah ditulis, dengan penanda yang jelas (nama shape).

## 6. Asumsi

- **Penempatan NG:**
  - di CP, kotak NG di kiri atas dengan garis siku;
  - di PFD, kotak NG di kiri sejajar dengan garis lurus (seperti referensi masing-masing).
- **Diagram PFD:**
  - dua kolom per halaman;
  - legenda di atas setiap halaman (referensi hanya satu halaman);
  - langkah dengan beberapa baris memakai nama proses baris pertama.
- **Kepala form PFD** masih bentuk umum. Kepala seperti referensi (judul "Process Flow Chart for ...", Part Name, Process Line) masuk bagian C.

## 7. Temuan desain baru

Tidak ada temuan baru. E-144 bagian C (cover dan header) masih terbuka.

## 8. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run check` | 0 error |
| Vitest Help | Lulus |
| `npm --prefix web run test:e2e` | 71 lulus |
| Tur (`fast`) dan help-shots | Lulus |
| Pemeriksaan mata | HTML PFD dan CP (Chrome), Excel PFD dan CP (Excel → HTML), pratinjau PFD di aplikasi |

## 9. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `wrapText(text string, max int) []string` yang membungkus teks per kata menjadi baris paling banyak `max` huruf. Lalu tulis tiga kasus test-nya: teks pendek, teks yang perlu dua baris, dan satu kata yang lebih panjang dari `max`.
