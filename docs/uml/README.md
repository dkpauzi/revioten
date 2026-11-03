# Paket UML · Sistem PFD, PFMEA, dan CP

- **Versi:** draft 0.12, 3 November 2026. Sumbernya brief desain v1.19, kode Tahap 0 sampai M3.2 (D-80: UML diperbarui mengikuti kode), dan usulan dari desain UI serta register nomor yang dicatat di `usulan-desain-ui.md` (UX-01 sampai UX-07, status Terbuka). Nama jabatan mengikuti bagan organisasi NPI (D-54), tanpa nama orang.
- **Tujuan:** bahan bagi tim IT untuk membangun versi mereka sendiri tanpa melihat kode (D-33, D-38). Paket ini dibekukan sebagai v1.0 di milestone M8.4 (R-UML-03).
- **Kode berjalan sejajar:** mulai 27 Oktober 2026 kode dibangun sejajar dengan UML yang tersisa, dan dipakai untuk mengevaluasi desain (D-77). Temuan dari kode dicatat di `docs/evaluasi-desain.md` lalu diputuskan pemilik (usulan, brief bagian 15).

## Isi dan status

| ID | Artefak | File | Status |
|---|---|---|---|
| U1 | Glosarium | `U1-glosarium.md` | Draft 0.12: menambah istilah usulan UX (simbol internal dan customer, cakupan poin, gambar di sel, tabel spesifikasi, cek sinkronisasi, workbook, cakupan yokoten, nomor lama). Draft 0.11: dengan kolom istilah layar Inggris (D-98), istilah penomoran Tahap 2, istilah format template M3.1, dan istilah process template M3.2. Kolom Cek WI diisi pemilik. |
| U2 | Diagram konteks C4 | `U2-c4-konteks.puml`, `export/U2-c4-konteks.svg`, `export/U2-c4-konteks.png` | Draft 0.7 |
| U3 | Diagram use case dan spesifikasi use case | `U3-daftar-use-case.md`, `U3-use-case.puml` (8 diagram), `U3-gaya.iuml`, `export/U3-*.svg` dan `.png` | Draft 0.9: menambah UC-24, UC-25, UC-39 (usulan), memperluas UC-36 ke PFD, dan merinci UC-14 serta UC-64. Spesifikasi lengkap (prasyarat, alur, hasil) masih langkah 5 urutan S2 di milestone. |
| U3 (lampiran) | Peta pengguna dan hak akses | `U3-peta-pengguna.md`, `U3-peta-pengguna.puml`, `export/U3-H-peta-pengguna.svg` dan `.png` | Baseline, disetujui pemilik 27 Oktober 2026 (D-66); draft 0.8 menambah aturan tetap D-104 dan D-118; draft 0.9 memetakan UC usulan ke hak yang sudah ada tanpa hak baru |
| U4 | Activity diagram dengan swimlane | | Belum |
| U5 | Diagram kelas domain | `U5-kelas-domain.puml` (5 diagram) | Draft 0.12: U5-A sampai U5-D tetap (entitas yang dibangun sampai M3.2). U5-E baru berisi usulan UX-01 sampai UX-04 yang belum diputuskan dan belum dibangun, berwarna kuning `<<usulan>>`. |
| U6 | State machine | | Belum |
| U7 | Sequence diagram | | Belum |
| U8 | ERD fisik | `U8-erd.puml` (6 diagram) | Tidak berubah di draft 0.12, karena ERD mengikuti migrasi kode. Draft 0.11: tabel migrasi 00001 sampai 00012, beserta kunci dan constraint penting. Hak role aplikasi ditulis untuk tabel yang dijaga ketat (audit log, register nomor, format template, process template). Diperbarui per milestone. Belum dirender |
| Lampiran | Usulan desain UI dan temuan register | `usulan-desain-ui.md` | Draft 0.12, status semua butir Terbuka |
| U9 | Diagram komponen C4 dan deployment | | Belum (M8.4) |
| U10 | File OpenAPI | `api/openapi.yaml` di root repo (UI API `/api/v1`) | Berjalan per milestone. Kontrak integrasi `/integration/v1` dibuat di Tahap 9 |
| U11 | Catatan keputusan | Log keputusan di brief bagian 6 | Berjalan |
| U12 | Skenario penerimaan (Given, When, Then) | | Belum (S4) |
| U13 | Matriks keterlacakan | | Belum |
| U14 | Kebutuhan kualitas | | Belum (M8.4) |

## Cara merender

Alat dan versinya mengikuti brief bagian 7:
- PlantUML v1.2026.9
- C4-PlantUML v2.14.0, sudah disalin ke `lib/` beserta lisensi MIT-nya
- Java dan Graphviz terpasang. Draft ini dirender dengan OpenJDK 21.0.12 dan Graphviz 2.43.0.

Jalankan dari folder ini:

```
java -jar plantuml-1.2026.9.jar -tsvg -o export U2-c4-konteks.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U2-c4-konteks.puml
java -jar plantuml-1.2026.9.jar -tsvg -o export U3-use-case.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U3-use-case.puml
java -jar plantuml-1.2026.9.jar -tsvg -o export U3-peta-pengguna.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U3-peta-pengguna.puml
java -jar plantuml-1.2026.9.jar -tsvg -o export U5-kelas-domain.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U5-kelas-domain.puml
java -jar plantuml-1.2026.9.jar -tsvg -o export U8-erd.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U8-erd.puml
```

U5 dan U8 ditulis di container cloud. Pada 3 November 2026 keduanya, beserta U3, berhasil dirender di container dengan PlantUML 1.2022.12 dan Graphviz, versi lama yang ada di sana, jadi sintaksnya sudah lolos pemeriksaan dasar. U2 tidak bisa dirender dengan versi lama itu karena C4-PlantUML 2.14.0 butuh PlantUML yang lebih baru. Jalankan perintah di atas di komputer pemilik, lalu commit hasilnya. Bila PlantUML menolak salah satu baris, catat pesannya agar file sumbernya dibetulkan.

File diagram memasang `RELATIVE_INCLUDE` sebelum include. Dengan begitu C4-PlantUML diambil dari `lib/`, bukan dari internet, dan versinya tetap terkunci (R-UML-04).

## Konvensi

- Setiap elemen membawa ID dari brief (R-, D-, SC-), supaya bisa ditelusuri ke aturannya (R-UML-01).
- Paket ini menggambarkan perilaku, data, dan aturan, bukan struktur kode. Tim IT boleh memakai stack lain (R-UML-02).
- Penjelasan ditulis dalam bahasa Indonesia. Nama jabatan, standar, dan tahap tanda tangan memakai istilah brief dan WI (R-UML-05, usulan).
- Elemen masa depan (latihan tahap 9 dan 10) digambar putus-putus dan berwarna abu-abu (R-UML-06, usulan).
- Hubungan putus-putus berarti tindakan di atas kertas, di luar sistem. Contohnya tanda tangan basah.

## Yang masih menunggu jawaban

Pertanyaan yang lebih luas ada di brief bagian 12. Temuan dari kode yang belum diputuskan pemilik, termasuk yang menyentuh U1, U3, U5, dan U8, ada di `docs/evaluasi-desain.md` (status Terbuka).
