# C3 · Cetakan sama dengan pratinjau

- **Tanggal:** 22 November 2026
- **Keputusan:** D-122 dan G11 (Process Flow CP dari PFD, konektor otomatis), D-211 (urutan C2 sampai C5). Temuan: E-136 (terjawab dari contoh), E-137 (baru).
- **Data acuan:** CP SC-TR-020-04 dan PFD SH-TR-020-02 (Tokairika) di `docs/referensi/`.

## Jawaban E-136 dari contoh Tokairika

E-136 bukan soal NCP atau SCRAP; keduanya sudah digambar sebagai cabang NG. E-136 menanyakan langkah PFD yang tidak punya baris di CP. Di CP Tokairika, setiap langkah PFD punya baris. Langkah tanpa kontrol tetap ditulis dengan "--" (misalnya Part Receiving baris 2 dan 4). Jadi aturannya: setiap langkah PFD muncul di CP. Pemeriksaan otomatisnya dibangun bersama alur PFD → CP di C4.

Dari contoh yang sama ditemukan satu perbaikan: langkah **2 (IQC)** berisi beberapa proses (IQC untuk material berbeda), tetapi simbolnya hanya satu. Pengelompokan alur sekarang hanya memakai nomor langkah. Sel No digabung per langkah, sedangkan sel nama proses digabung per langkah dan proses.

## Yang dibuat

| Bagian | Isi |
|---|---|
| Rencana halaman (`internal/printout/plan.go`) | Satu perhitungan untuk pratinjau, PDF, dan Excel. Tinggi setiap baris diperkirakan dari teks: dibungkus per kata seperti browser, menurut lebar kolom dan ukuran huruf. Baris dibagi ke halaman; baris pertama satu langkah tidak boleh menjadi baris terakhir halaman. Simbol dipasang di baris pertama setiap langkah, dan konektor A, B, C di setiap batas halaman. |
| Langkah PFD | Cetakan CP membaca PFD satu set (`pfdSteps` di `service.go`); PFD memakai barisnya sendiri. |
| PDF (`html.go`, `flowsvg.go`) | Halaman ditulis satu per satu: kepala form dan judul kolom diulang, tinggi baris dan judul kolom dikunci, sel nomor dan proses digabung per langkah, dan SVG alur di atas kolom Process Flow. |
| Excel (`xlsxflow.go`) | Tinggi baris dari rencana (disesuaikan dengan skala muat selebar kertas), pemisah halaman manual di awal setiap halaman, kepala form dan judul kolom berulang di setiap halaman (seperti baris 2 sampai 16 di Tokairika), sel gabungan, dan alur sebagai shape: simbol proses dan inspeksi, garis, konektor elips, dan tulisan "NG → SCRAP/NCP" di sel langkah. |
| Pratinjau | Endpoint baru `GET /api/v1/documents/{id}/page-preview` (L-4 untuk draft, L-1 untuk revisi terbit) mengirim rencana halaman beserta SVG alurnya. `PreviewSheet.svelte` hanya menggambarnya, tanpa mengukur di browser. Logika TypeScript C2 (`preview.ts`) dihapus, supaya hanya ada satu sumber perhitungan. |
| Lebar kolom alur | Bawaan 1,3 bila layout tidak mengaturnya, sama di ketiga keluaran. |

## Cara mencoba

1. Buka CP satu set yang PFD-nya berisi langkah (nomor langkah, Flow Symbol, NG Disposition).
2. Tekan **Preview** di toolbar dan catat di mana konektor A dan B berada.
3. Tekan **Print**, lalu **PDF**. Halaman, simbol, dan konektornya sama dengan pratinjau.
4. Tekan **Excel** dan buka di Excel. Alurnya berupa shape. Pemisah halaman (View → Page Break Preview) berada di tempat yang sama.
5. Tambah atau hapus baris, lalu ulangi. Semua gambar bergeser sendiri.

## Konsep yang dipelajari

- **Satu sumber perhitungan.** Bila pratinjau, PDF, dan Excel masing-masing menghitung sendiri, hasilnya pasti berbeda sedikit. Server menghitung sekali, dan ketiganya hanya menggambar.
- **Tinggi baris dikunci.** Browser dan Excel biasanya menyesuaikan tinggi baris dengan isinya. Supaya pemisah halaman tidak bergeser, tinggi dari rencana dipakai tetap. Karena itu perkiraannya dibuat longgar (pembungkusan per kata, lebar huruf 0,6 em).
- **Gaya aplikasi bisa bocor ke pratinjau.** `table { font-size: 14px }` di `app.css` sempat membuat huruf pratinjau dua kali lebih besar dari cetakan. Lembar kertas sekarang menentukan ukuran hurufnya sendiri.
- **Skala Excel.** Excel mengecilkan lembar supaya muat selebar kertas, jadi tinggi baris di lembar dibagi skala itu supaya di kertas tingginya sama dengan rencana.

## Asumsi

- **ASUMSI E-137:** cetakan dan pratinjau CP memakai isi PFD yang sekarang (draft bila ada), juga untuk revisi CP lama. Bisa diubah menjadi "PFD revisi yang berlaku saat CP itu terbit" bila perlu.
- Excel menulis "NG → SCRAP" sebagai teks sel, bukan kotak teks, supaya tetap terbaca bila shape digeser.
- AP belum tampil di pratinjau dan cetakan (sama seperti sebelumnya).
- Isi sel yang lebih panjang dari perkiraan terpotong. Perkiraannya sengaja longgar, dan hasilnya terlihat di pratinjau sebelum dicetak.

## Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus. Test baru: `TestPaginate` (termasuk aturan baris awal langkah), `TestConnectorLetter`, `TestFlowPages` (langkah dengan beberapa proses), `TestBuildPlan`, dan `TestTokairikaOutputs` (19 simbol, konektor, dan pemisah halaman sama di HTML dan Excel). Izin endpoint baru ada di `TestPermissionsPerOperation` |
| `npm --prefix web test` | 174 lulus |
| `npm --prefix web run test:e2e` | 70 lulus; `m4-preview` memeriksa pratinjau dari server, berkas Excel berisi gambar alur, pratinjau PFD, dan halaman bertambah sesudah 80 baris baru |
| Help, tur (`fast`) | Lulus |

Contoh berkas untuk diperiksa mata: `REVIOTEN_SAMPLE_DIR=<folder> go test -short ./internal/printout -run TestSampleTokairika` menulis `sample-cp.html`, `.pdf`, dan `.xlsx`.

## Latihan tulis tangan

Halaman muat 100 mm. Tinggi baris 40, 40, dan 40 mm, dan baris kedua memulai langkah baru. Tulis pembagian halamannya dengan aturan "baris awal langkah tidak boleh menjadi baris terakhir", lalu jelaskan mengapa aturan ini mencegah simbol menumpuk dengan konektor.
