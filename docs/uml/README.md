# Paket UML · Sistem PFD, PFMEA, dan CP

- **Versi:** draft 0.17, 15 November 2026. Sumbernya brief desain v1.35 dan kode sampai Tahap 9 serta D-195 sampai D-197 (migrasi 00047), sesuai D-80 (UML mengikuti kode). Draft 0.17 menambah U5-J review berkala, UC-58 dan UC-65, hak L-5, dan U8 dari migrasi 00047. Draft 0.16 menyusul kode Tahap 4 sampai 9: U5-F sampai U5-I baru, U8 dibangkitkan ulang dari database (79 tabel), U2 menambah aplikasi lain, web ECR, Microsoft 365, dan antivirus, U3 menambah status pembangunan, notifikasi, dan ECR, U6 baru. Riwayat: draft 0.12 dari pemilik; 0.13 dan 0.14 menambah hasil tinjauan desain UI dan keputusan D-121 sampai D-126. Nama jabatan mengikuti bagan organisasi NPI (D-54).
- **Tujuan:** bahan bagi tim IT untuk membangun versi mereka sendiri tanpa melihat kode (D-33, D-38). Paket ini dibekukan sebagai v1.0 di milestone M8.4 (R-UML-03).
- **Kode berjalan sejajar:** mulai 27 Oktober 2026 kode dibangun sejajar dengan UML yang tersisa, dan dipakai untuk mengevaluasi desain (D-77). Temuan dari kode dicatat di `docs/evaluasi-desain.md` lalu diputuskan pemilik (usulan, brief bagian 15).

## Isi dan status

| ID | Artefak | File | Status |
|---|---|---|---|
| U1 | Glosarium | `U1-glosarium.md` | Draft 0.16: istilah migrasi, dasbor, API integrasi, web ECR, email notifikasi, dan row-level security. Draft 0.11 sampai 0.15: istilah layar Inggris (D-98), penomoran, format template, process template, kolom terisi otomatis, menu, dan varian poin. Kolom Cek WI diisi pemilik. |
| U2 | Diagram konteks C4 | `U2-c4-konteks.puml`, `export/U2-c4-konteks.svg`, `export/U2-c4-konteks.png` | Draft 0.16 (sumber diperbarui, gambar di `export/` masih draft 0.7 sampai dirender ulang) |
| U3 | Diagram use case dan spesifikasi use case | `U3-daftar-use-case.md`, `U3-use-case.puml` (10 diagram), `U3-gaya.iuml`, `export/U3-*.svg` dan `.png` | Draft 0.16: bagian Status pembangunan, U3-G API integrasi dan ECR, U3-H email notifikasi, U3-I portal (masa depan). Gambar di `export/` masih draft 0.9 sampai dirender ulang. Spesifikasi lengkap (prasyarat, alur, hasil) masih langkah 5 urutan S2. |
| U3 (lampiran) | Peta pengguna dan hak akses | `U3-peta-pengguna.md`, `U3-peta-pengguna.puml`, `export/U3-H-peta-pengguna.svg` dan `.png` | Baseline, disetujui pemilik 27 Oktober 2026 (D-66). Draft 0.16 menambah layar Roles, row-level security, dan bagian 7 "Siapa yang butuh akun"; isi matriks hak tidak berubah. |
| U4 | Activity diagram dengan swimlane | | Belum |
| U5 | Diagram kelas domain | `U5-kelas-domain.puml` (10 diagram) | Draft 0.16: U5-F dokumen dan baris, U5-G perubahan dan revisi, U5-H cetak dan migrasi, U5-I integrasi dan notifikasi. Draft 0.17: U5-J review berkala. U5-E berisi usulan UX yang belum diputuskan. |
| U6 | State machine | `U6-state-machine.puml` (5 diagram) | Draft 0.16, baru: nomor dokumen, perubahan, revisi, versi template dan form set, email dan job ekspor |
| U7 | Sequence diagram | | Belum |
| U8 | ERD fisik | `U8-erd.puml` (10 diagram), dibangkitkan oleh `tools/u8_dari_database.py` | Draft 0.16: 79 tabel dan 147 hubungan sampai migrasi 00046, ditandai `<<RLS>>` untuk tabel yang disaring database. Jangan diedit tangan: jalankan skripnya lagi sesudah migrasi baru. Belum dirender |
| Lampiran | Usulan desain UI dan temuan register | `usulan-desain-ui.md` | Draft 0.12, status semua butir Terbuka |
| U9 | Diagram komponen C4 dan deployment | | Belum. Gambaran tertulisnya ada di `docs/presentasi/dokumen-untuk-it.md` bagian 2 dan 6 |
| U10 | File OpenAPI | `api/openapi.yaml` (UI API `/api/v1`) dan `api/integration.yaml` (API integrasi `/integration/v1`, Tahap 9) | Berjalan per milestone. Bentuk API integrasi dijaga contract test (`api/integration.contract.json`) |
| U11 | Catatan keputusan | Log keputusan di brief bagian 6 | Berjalan |
| U12 | Skenario penerimaan (Given, When, Then) | | Belum (S4) |
| U13 | Matriks keterlacakan | | Belum |
| U14 | Kebutuhan kualitas | | Belum. Sebagian ada di `docs/presentasi/dokumen-untuk-it.md` bagian 4, 5, dan 7 |

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
java -jar plantuml-1.2026.9.jar -tsvg -o export U6-state-machine.puml
java -jar plantuml-1.2026.9.jar -tpng -o export U6-state-machine.puml
```

U8 dibangkitkan dari database, bukan ditulis tangan. Sesudah migrasi baru, dari root repo dengan PostgreSQL menyala:

```
python docs/uml/tools/u8_dari_database.py revioten
```

**15 November 2026:** sumber draft 0.16 belum dirender, karena Java dan PlantUML belum terpasang di komputer pemilik. Gambar di `export/` masih versi lama sampai perintah di atas dijalankan.

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
