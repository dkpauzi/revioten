# C2 · Pratinjau lembar dan Process Flow otomatis

- **Tanggal:** 21 sampai 22 November 2026
- **Keputusan:** D-122 dan G11 (Process Flow CP dari PFD, konektor otomatis), D-211 (urutan C2 sampai C5 sesudah akar masalah dijelaskan). Temuan: E-136.
- **Data acuan:** CP SC-TR-020-04 dan PFD SH-TR-020-02 (Tokairika, D660A) di `docs/referensi/`.

## Akar masalah

Di CP Tokairika, kolom Process Flow adalah gambar (shape) Excel: kotak proses, belah ketupat inspeksi, nomor langkah (1, 2, 5.1, 5.7.1, …), cabang NG ke SCRAP/NCP, dan lingkaran konektor A, B, C di batas halaman. Shape tidak menempel ke baris. Setiap poin ditambah atau dihapus, semua gambar sesudahnya harus digeser manual, sering sampai halaman-halaman berikutnya.

## Evaluasi C1

- Toolbar, laci samping, tab sheet, dan kotak Langkah berikutnya tetap. Tidak ada yang dibuang.
- Toolbar mendapat tombol **Preview**.
- Untuk C4: langkah berikutnya PFMEA/CP seharusnya "isi PFD dulu" bila PFD set masih kosong.

## Yang dibuat

| Bagian | Isi |
|---|---|
| Logika (`web/src/lib/documents/preview.ts`) | `paperSize` (A3/A4, arah), `paginate` (membagi baris ke halaman menurut tinggi terukur), `groupsOf` (baris satu langkah: nomor operasi dan proses sama), `stepsOf` (simbol dan arah NG langkah PFD), `flowPages` (simbol di baris pertama setiap langkah, konektor A, B, C di setiap batas halaman yang dilewati alur), `previewText` (teks sel sama dengan cetakan, termasuk RPN) |
| Tampilan (`PreviewSheet.svelte`) | Tiga tahap: semua baris digambar di halaman ukur tersembunyi dan tingginya diukur; baris dibagi ke halaman; setiap halaman digambar, lalu posisi barisnya diukur untuk menggambar alur sebagai SVG di atas kolom Process Flow. Kepala form dan judul kolom diulang di setiap halaman, sel nomor dan proses digabung per langkah, nomor halaman di kaki. Halaman dikecilkan supaya muat selebar layar. |
| Halaman dokumen | Tombol **Preview** di kelompok Review. Layout cetak diambil dari pilihan customer atau bawaan jenis dokumen. CP mengambil langkah dari PFD satu set; PFD menggambar alurnya sendiri di kolom Flow Symbol. |
| Teks | Kunci baru dua bahasa: `wbShortPreview`, `previewOpen`, `sheetPreviewTitle`, `previewHint`, `previewPage`, `previewFlow`, `previewNoPfd`, `previewNoLayout`. |
| Help, tur, skenario | Satu langkah baru di bab Documents; tur langkah 21 membuka Preview CP; skenario langkah 21. |

## Cara mencoba

1. Buka dokumen CP satu set yang PFD-nya sudah berisi langkah (nomor langkah dan Flow Symbol; inspeksi dengan NG Disposition).
2. Tekan **Preview** di toolbar.
3. Lihat kolom Process Flow: simbol per langkah, NG ke SCRAP/NCP, dan konektor A di bawah halaman 1 serta di atas halaman 2.
4. Tutup Preview, tambah beberapa baris di langkah awal, lalu buka Preview lagi. Simbol dan konektor sudah bergeser sendiri.

## Konsep yang dipelajari

- **Mengukur dulu, baru membagi halaman.** Tinggi baris bergantung pada isi dan lebar kolom, jadi tidak bisa ditebak. Baris digambar sekali di tempat tersembunyi, diukur, lalu dibagi.
- **Gambar dihitung dari data, bukan disimpan.** Posisi simbol dan konektor tidak pernah disimpan. Itulah yang menghapus pekerjaan menggeser gambar.
- **`transform: scale` tidak mengubah ukuran tata letak**, jadi ukuran yang diukur tetap ukuran kertas sungguhan, walaupun di layar halaman tampak lebih kecil.

## Keputusan dan alasan

- Pratinjau hanya untuk dibaca; isi tetap diubah di tabel (D-211). Tabulator tidak diganti.
- Konektor diberi huruf berurutan per batas halaman (A, B, C, …, Z, AA), sama seperti CP Tokairika.

## Asumsi (ASUMSI E-136)

- Langkah PFD yang tidak punya baris di CP tidak digambar; garisnya tetap menyambung.
- Nomor langkah CP yang tidak ada di PFD digambar sebagai proses.
- Bila layout tidak mengatur lebar kolom Process Flow, pratinjau memberinya lebar 1,3. Cetakan memakai 0,5; keduanya disamakan di C3.
- AP belum tampil di pratinjau.

## Temuan desain baru

- **E-136** · Langkah PFD tanpa baris CP dan lebar kolom Process Flow (ambigu, terbuka).

## Status test

| Test | Hasil |
|---|---|
| `npm --prefix web test` | 187 lulus, termasuk 13 kasus pratinjau |
| `npm --prefix web run check` | 0 error |
| `npm --prefix web run test:e2e` | 70 lulus. Test baru `m4-preview.test.ts`: 19 langkah PFD SH-TR-020-02, CP 57 baris; memeriksa halaman, simbol, NCP, konektor A, setiap langkah tepat sekali, pratinjau PFD, dan hitung ulang sesudah 30 baris baru (halaman bertambah, konektor B) |
| `npm --prefix web run help:shots` | Lulus |
| Tur (`fast`, tanpa jendela) | Lulus |
| Test Go | Tidak ada perubahan Go |

## Latihan tulis tangan

Tanpa melihat kode, tulis fungsi `flowPages` untuk tiga halaman: halaman 1 berisi langkah 1 dan awal langkah 2, halaman 2 berisi sisa langkah 2, dan halaman 3 berisi langkah 3. Konektor mana saja yang muncul, dan di halaman mana?
