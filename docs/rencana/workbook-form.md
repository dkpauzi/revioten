# Rencana C · Halaman dokumen sebagai workbook berbentuk form (D-145)

- **Status:** usulan, menunggu persetujuan pemilik (21 November 2026).
- **Dasar:** D-145 (workbook seperti Excel dengan tab Cover, PFD, PFMEA, Control Plan, Check, menu berlabel, satu langkah berikutnya, mengikuti papan PfmeaVda dan ControlPlan2024). D-145 tercatat "dibangun mulai M4.2", tetapi yang terbangun baru tabel polos (Tabulator) dengan bagian-bagian bertumpuk. Rencana ini menutup selisihnya.
- **Acuan tampilan:** `docs/design/form-set-aiag-vda/` (MultiPartWorkbook, PfmeaVda, ControlPlan2024) dan dokumen nyata di `docs/referensi/` (set TR-020, CP KT).

## 1. Selisih sekarang dan desain

| Desain | Sekarang |
|---|---|
| Tab sheet di bawah: Cover, PFD, PFMEA, Control Plan, Check | Tautan sheet di tengah halaman; tidak ada sheet Cover |
| Toolbar berkelompok: Edit, Rows, Insert in cell, Template, Review, Output | Tombol tersebar di bawah tabel; panel Print, Revisions, Signature, Submission, dan lainnya bertumpuk di atas tabel |
| Bilah rumus: alamat sel (`050 · No. 2 · Specification`) dan isinya | Tidak ada |
| Lembar seperti kertas: kepala perusahaan, judul form, blok header sesuai form, judul grup kolom, sel proses digabung per proses, kolom Process Flow bergambar, AP/RPN berwarna | Tabel polos satu baris judul; header di formulir terpisah |
| Panel kanan "Selected row": rincian baris, penjelasan AP, baris PFMEA/CP yang tertaut | Tidak ada |
| Kotak "Next step" di kanan atas | Tidak ada |
| Kartu bagian per PCB (All parts, A, B, C) | Tombol tab bagian (SectionBar) |

## 2. Pembagian milestone (satu per sesi)

### C1 · Kerangka workbook
- Kepala halaman: model dan nomor, badge form set, status revisi, kotak **Next step** (satu aksi utama menurut status: rakit, cek, kirim tanda tangan, catat rilis).
- Tab sheet di bawah layar: Cover, PFD, PFMEA, Control Plan, Check (dokumen satu set).
- Toolbar berkelompok. Panel yang sekarang bertumpuk (Print, Revisions, Signature, Submission, Template links, Compose, Compare, Copy, Order, Migration, Excel compare) dibuka dari toolbar sebagai laci samping, bukan ditumpuk di halaman.
- Fungsi tidak berubah; semua test E2E lama disesuaikan ke tombol toolbar.

### C2 · Lembar form (inti)
- Komponen baru `FormSheet.svelte` menggantikan Tabulator untuk PFD, PFMEA, dan CP:
  - kepala lembar: nama perusahaan, judul form, blok header sesuai form yang dipakai (field header format template, bisa diubah langsung di tempatnya);
  - dua tingkat judul kolom dari `group` field format template (misalnya STRUCTURE ANALYSIS (STEP 2), Characteristics, Methods);
  - sel nomor operasi dan nama proses digabung untuk baris berproses sama, seperti di Excel;
  - kolom Process Flow bergambar simbol dari PFD (D-122);
  - warna AP/RPN dan simbol karakteristik khusus;
  - pilih sel dengan klik atau panah; ketik di sel atau di bilah rumus; Enter menyimpan baris (penguncian optimistik M4.3 tetap).
- Lebar kolom diambil dari layout cetak customer (M6.1), supaya layar dan cetakan sama.

### C3 · Panel baris terpilih
- PFMEA: S, O, D, AP atau RPN beserta penjelasannya, rantai struktur (item, step, 4M, FE, FM, FC), baris CP yang tertaut.
- CP: baris PFMEA yang tertaut, sumber template (poin, varian).
- PFD: simbol, arah NG, langkah sebelum dan sesudahnya.

### C4 · Sheet Cover dan PFD
- **Cover:** nomor, customer, model, part name dan part number, riwayat revisi, slot persetujuan in-house dan customer beserta status tanda tangan (seperti sampul SH-TR-020-02).
- **PFD:** sesuai PFD nyata, isinya hanya proses: nomor langkah, nama proses, simbol, dan arah NG (Scrap, NCP). Diagram alur tampil berdampingan dengan tabelnya, bukan disembunyikan di tombol.

### C5 · Bagian per PCB, layar kecil, dan penutup
- Kartu bagian seperti MultiPartWorkbook (All parts, A, B, C, + Add part).
- Layar di bawah 1024px: lembar form bergulir di dalam kartunya; di HP tampil daftar baris (form lebar tidak terbaca di HP).
- Help (bab Documents ditulis ulang, gambar baru), test E2E, catatan milestone, UML.
- Sesudah C5 baru tur otomatis ditulis ulang (pilihan B) dengan data TR-020.

## 3. Keputusan yang perlu pemilik

1. **Tabulator diganti komponen sendiri.** Tabulator (stack Tahap 4) tidak bisa menggabungkan sel proses dan sulit dibuat seperti kertas. Usulan: lembar form dibuat sendiri dengan Svelte; Tabulator dilepas dari halaman dokumen. Ini mengubah daftar stack di brief bagian 7.
2. **Nama dan logo perusahaan di kepala lembar** (usulan): field baru di Master data → Document settings, dipakai layar dan cetakan.
3. **Cetakan** (usulan, sesudah C5): model cetak Excel/PDF sekarang juga masih sederhana (header berupa daftar pasangan). Menyamakannya dengan lembar form dibuat milestone terpisah.

## 4. Yang tidak berubah

- API dan aturan bisnis baris (JSONB, penguncian optimistik, RLS, audit log).
- Format template dan form set; hanya cara menampilkannya yang berubah.
- Tab Check (M4.9) tetap halaman sendiri, dibuka dari tab sheet.

## 5. Revisi rencana sesudah penjelasan pemilik (21 November 2026, sesudah C1)

Akar masalah menurut pemilik:

1. **Kolom Process Flow di Control Plan.** Di CP Tokairika (SC-TR-020-04) alurnya digambar dengan shape Excel: kotak proses, belah ketupat inspeksi, cabang NG ke SCRAP/NCP, nomor langkah (5.1, 5.7.1), dan lingkaran konektor A, B, C di batas halaman. Setiap baris ditambah atau dihapus, semua shape sesudahnya harus digeser manual, sering sampai halaman-halaman berikutnya. Ini target utama (sudah tercatat sebagai D-122 dan G11, tetapi belum dibangun: kolom itu masih kosong di cetakan).
2. **Template untuk proses yang mirip.** Proses seperti Part Receiving tidak bergantung model, jadi PFMEA dan CP dirakit dari template. PFD tidak memakai template: PFD dibuat per dokumen, dan langkah PFD menjadi dasar proses di PFMEA dan CP.
3. **Workbook bukan keharusan.** Yang diutamakan adalah melihat dokumen yang sedang dirancang dalam bentuk lembar seperti cetakannya, untuk memastikan tampilannya sudah benar.

Rencana C2 sampai C5 diganti menjadi:

| Milestone | Isi |
|---|---|
| **C2 · Pratinjau lembar dengan Process Flow otomatis** | Tombol **Preview** di toolbar. PFD, PFMEA, dan CP tampil per halaman kertas (ukuran dan arah dari layout cetak): kepala form diulang di setiap halaman, judul grup kolom, sel proses digabung. Kolom Process Flow CP digambar dari langkah PFD: simbol dan nomor langkah, garis alur, cabang NG ke SCRAP/NCP, dan konektor A, B, C di setiap batas halaman, dihitung ulang setiap kali baris berubah. Baca saja; isi tetap diubah di tabel. |
| **C3 · Cetakan sama dengan pratinjau** | PDF dibuat dari halaman pratinjau yang sama, jadi konektornya sama. Excel: kolom Process Flow sebagai shape (dievaluasi dulu dengan excelize). |
| **C4 · Alur PFD → PFMEA dan CP** | PFD dibuat dulu untuk dokumen baru atau dokumen migrasi. Setiap langkah PFD dicocokkan dengan template prosesnya, lalu baris PFMEA dan CP dirakit per langkah. |
| **C5 · Sheet Cover, lalu tur** | Cover seperti sampul form; tur ditulis ulang dengan data TR-020. |

Perubahan atas D-210: Tabulator tetap dipakai untuk mengubah isi. Lembar form dibuat sebagai pratinjau baca saja, bukan pengganti tabel.

## 6. Rencana C4 · PFD lebih dulu, lalu PFMEA dan CP per langkah PFD (22 November 2026)

Alur menurut pemilik: proses yang tidak bergantung model (misalnya Part Receiving) punya template umum. PFD dibuat per dokumen tanpa template, dan langkah PFD menjadi dasar proses di PFMEA dan CP. Dokumen baru dan dokumen migrasi tinggal memilih template untuk setiap langkah.

| Bagian | Isi |
|---|---|
| PFD | Baris PFD diisi langsung: nomor langkah, proses (dari master proses), Flow Symbol, NG Disposition. Tombol rakit dari template tidak ditawarkan di sheet PFD; Copy from another model tetap ada. |
| Rakit dari PFD | Tombol baru **Compose from PFD** di sheet PFMEA dan CP. Laci menampilkan setiap langkah PFD: nomor, proses, template terbit untuk proses itu (versi dan jumlah poin, sesudah varian customer/model diterapkan), dan keadaannya: sudah ada di dokumen, akan ditambah, atau tanpa template. Pilih langkah, tekan Compose: baris PFMEA dan CP dibuat dengan nomor langkah dan proses dari PFD, berurutan mengikuti PFD. Baris yang sudah ada tidak diubah. |
| Langkah tanpa template | Mendapat satu baris berisi nomor langkah dan proses, kolom lain "--", seperti CP Tokairika (E-136). |
| Pemeriksaan | Di laci dan pratinjau: langkah PFD yang belum ada di CP, dan nomor langkah CP yang tidak ada di PFD, ditampilkan sebagai peringatan. |
| Langkah berikutnya | PFMEA/CP dengan PFD kosong: "isi PFD dulu" (catatan evaluasi C1). |
| Dokumen migrasi | Dokumen lama memakai tombol yang sama: PFD diketik dari hardcopy, lalu PFMEA dan CP dirakit per langkah dan dicocokkan dengan hardcopy. |
| D-212 | Langkah PFD yang dipakai menggambar alur dibekukan di snapshot CP saat rilis, jadi cetakan revisi lama tidak berubah. |

**Status (22 November 2026):** C4 dibangun. Catatan: `docs/catatan/C4-pfd-dulu-rakit-dari-pfd.md`. Pemeriksaan di pratinjau tampil sebagai daftar di atas halaman, bukan tanda di dalam lembar. Tur ditulis ulang di C5.
