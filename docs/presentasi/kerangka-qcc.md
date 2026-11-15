# Kerangka presentasi QCC (M8.5)

- **Versi:** kerangka, 15 November 2026.
- **Status:** belum selesai menurut kriteria M8.5. Kriterianya "semua angka berasal dari pengukuran nyata", dan angka itu belum ada: baseline Excel (S7) belum diukur, pilot migrasi (M7.6) belum dijalankan, dan jadwal QCC belum ditetapkan (Q-16).
- **Aturan mengisi:** setiap kotak **[ukur]** diisi angka hasil pengukuran, beserta tanggal dan cara mengukurnya. Jangan diisi perkiraan. Angka dari sistem ini disebut "hasil demo di localhost", bukan hasil pemakaian harian (D-36, RK-50).

## 1. Alur cerita (delapan langkah QCC)

| Langkah | Isi | Sumber |
|---|---|---|
| 1. Tema | Mempermudah revisi dan yokoten PFD, PFMEA, dan Control Plan | Brief bagian 1 dan 2 |
| 2. Keadaan sekarang | Dokumen di Excel per model; nama proses tidak seragam antara PFD, PFMEA, dan CP; nomor tidak selalu sejajar dalam satu set | Temuan F-05 di tabel temuan dokumen referensi di brief, contoh dokumen di `docs/referensi/` |
| 3. Target | Ditetapkan sesudah baseline diukur | **[ukur]** baseline dulu |
| 4. Analisis penyebab | Salin-tempel antar berkas, tidak ada sumber tunggal untuk isi proses, riwayat perubahan tersebar | Brief bagian 2 |
| 5. Rencana perbaikan | Process template sebagai sumber tunggal, perakitan dokumen, perubahan dengan verifikasi, propagasi dan yokoten, rilis dengan snapshot | Brief bagian 5 |
| 6. Pelaksanaan | Demo yang berjalan di localhost, Tahap 0 sampai 8 | Catatan milestone di `docs/catatan/` |
| 7. Hasil | Bandingkan angka sebelum dan sesudah untuk tugas yang sama | Tabel di bagian 2 |
| 8. Standardisasi dan rencana berikutnya | Spesifikasi dan UML diserahkan ke IT (D-33); migrasi bertahap (R-MIG-08, usulan) | `dokumen-untuk-it.md` |

## 2. Tabel angka

Tugas yang diukur harus sama persis di kolom sebelum dan sesudah: model yang sama, orang yang sama bila bisa, dan batas mulai dan selesai yang sama.

| No | Ukuran | Sebelum (Excel) | Sesudah (hasil demo) | Cara mengukur sesudah |
|---|---|---|---|---|
| 1 | Waktu merakit satu set dokumen (PFD, PFMEA, CP) untuk model baru | **[ukur]** | **[ukur]** | Stopwatch: dari membuat dokumen sampai ketiga dokumen terisi lewat **Compose from process templates** |
| 2 | Waktu satu yokoten ke beberapa model | **[ukur]** | **[ukur]** | Stopwatch: dari **Spread to other models** sampai semua target diputuskan |
| 3 | Waktu satu revisi, dari perubahan diterima sampai rilis dicatat | **[ukur]** | **[ukur]** | Dasbor, kartu **Revision time**, nilai median |
| 4 | Jumlah nomor bentrok atau tidak sejajar dalam satu set | **[ukur]** dari register Excel | **[ukur]** | Laporan **Import legacy numbers** untuk register lama; register sistem untuk nomor baru |
| 5 | Waktu mencari riwayat satu perubahan | **[ukur]** | **[ukur]** | Stopwatch: dari membuka dokumen sampai menemukan perubahan di **Compare revisions** |
| 6 | Bagian baris yang mengikuti template | Tidak ada pembanding | **[ukur]** | Dasbor, kartu **Rows that follow a template** |
| 7 | Usaha migrasi per dokumen | Tidak ada pembanding | **[ukur]** | Dasbor, kartu **Migration**, rata-rata menit dari pilot dua sampai tiga model |

## 3. Yang perlu dikerjakan sebelum presentasi

1. Ukur baseline dengan cara kerja Excel sekarang (S7), dan catat tanggal serta caranya.
2. Jalankan pilot migrasi (tugas 17 di catatan pribadi).
3. Ulangi tugas baseline di demo dengan stopwatch, lalu isi kolom "sesudah".
4. Tetapkan jadwal dan cakupan QCC dengan atasan (Q-16, S8).
5. Siapkan satu contoh nyata untuk slide keadaan sekarang, misalnya nama proses yang berbeda di tiga dokumen (F-05).

## 4. Yang sebaiknya dikatakan terus terang

- Angka "sesudah" berasal dari demo di satu laptop, bukan dari pemakaian harian oleh banyak orang.
- Persetujuan tetap tanda tangan basah; sistem tidak mempercepat langkah itu.
- Dokumen lama masuk apa adanya. Migrasi tidak memperbaiki mutunya.
