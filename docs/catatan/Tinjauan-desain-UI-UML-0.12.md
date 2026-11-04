# Tinjauan desain UI dan UML draft 0.12 terhadap M0 sampai M3

- **Tanggal:** 3 November 2026
- **Permintaan pemilik:** "update juga UML nya. jangan lanjut dulu ke M4. tinjau ulang semuanya apakah desain UI/UX dan UML ini ada yang mempengaruhi M0-M3. jika ada tolong di update".
- **Bahan:**
  - kanvas desain UI "Desain UI Sistem PFD PFMEA CP", 18 artboard, termasuk baris v2 yang disarankan (Home, Workbook, Template library) dan artboard "current form layout" untuk PFD, PFMEA, Control Plan, dan Cover;
  - paket UML draft 0.12 dari pemilik, termasuk `docs/uml/usulan-desain-ui.md` (UX-01 sampai UX-07, semuanya Terbuka).
- **Dasar:** D-77 (kode dipakai mengevaluasi desain), D-80 (UML mengikuti kode), dan CLAUDE.md bagian 6 (temuan tidak diputuskan sendiri; usulan Terbuka menunggu pemilik).
- **Pembaruan 4 November 2026:** pemilik menjawab tinjauan ini (bagian 9). Kolom Process Flow dan Process Step / Function tidak lagi disembunyikan; keduanya menjadi kolom terisi otomatis (migrasi 00014, D-122, D-123). Bagian 1 sampai 8 di bawah tetap ditulis seperti keadaan 3 November.

## 1. Ringkasan

1. **UML 0.12 dipasang apa adanya** dalam commit tersendiri. Draft 0.13 lalu menambah hasil kode tinjauan ini: tanda `automatic` pada simbol, di U1, U5-C, U8-E, dan README.
2. **Tidak ada pertentangan dengan keputusan pemilik.** Kode M0 sampai M3 tidak bertentangan dengan desain pada hal yang sudah diputuskan (DECIDED).
3. **Tiga perbaikan kecil di M3.1.** Desain menunjukkan isi awal format template tidak sesuai aturan brief yang sudah ada, jadi isi awal itu dibetulkan (migrasi 00013):
   - simbol Revision dipasang sistem (R-REV-09);
   - kolom Process Flow di CP dibentuk dari PFD (R-DOC-09);
   - kolom Process Step / Function di PFMEA berisi proses baris (ASUMSI E-70).
4. **Sisanya menunggu keputusan pemilik:**
   - usulan UX-01 sampai UX-07, yang statusnya Terbuka di paket UML pemilik;
   - temuan E-33 dan E-70 sampai E-76.

   Kode tidak diubah untuk hal-hal ini. Tabel saran ada di `docs/evaluasi-desain.md`, bagian "Usulan desain UI (UX-01 sampai UX-07)".

## 2. Hasil per milestone

| Milestone | Yang dibandingkan dengan desain | Hasil |
|---|---|---|
| M0.1 sampai M0.5 | Menu dan gaya halaman. Desain v2 memakai menu berlabel (Home, Documents, Template library, Yokoten, Settings), design system (Geist, aksen indigo, teks lebih besar), dan pilihan bahasa EN/ID | Perlu keputusan (UX-05). Bahasa Indonesia mengubah D-96 |
| M1.1 Login | Tidak ada di desain | Sesuai |
| M1.2 Hak akses | U3 draft 0.9 memetakan UC-24, UC-25, dan UC-39 ke hak yang sudah ada (A-2, E-2, E-6) | Sesuai, tanpa hak baru |
| M1.3 Audit log | Tidak berubah | Sesuai |
| M1.4 Customer dan model | Register nomor memakai kode customer tiga huruf (AJI, AVI) | Perlu keputusan (E-74, bertentangan dengan D-100) |
| M1.5 Master proses | Alias ditampilkan ("also called"); permintaan proses baru dari layar susun | Sesuai; permintaan proses baru dicatat sebagai E-76 untuk Tahap 4 |
| M1.6 Jenis dokumen dan perubahan | Menu "Master data" dan master simbol | Perlu keputusan (UX-03, UX-05) |
| M1.7 Backup | Tidak tersentuh | Sesuai |
| M1.8 Jabatan dan rute | Cover mengambil slot dan judul cetak dari rute, dan nama dari pemegang jabatan (R-PRT-07) | Sesuai |
| M2.1 sampai M2.4 Penomoran | Register, saringan, status, umur nomor Reserved, set, revisi awal | Sesuai. Beda tampilan saja (jumlah per status, pratinjau tiga nomor set): UX-05. Register PL1 dan PL3: E-75 |
| M2.5 Impor nomor lama | Kolom No. (Before), satu nomor untuk beberapa part, kode tiga huruf | Perlu keputusan (UX-06, E-74) |
| M3.1 Format template | Susunan form sekarang di artboard "current form layout"; simbol; sorotan | Tiga perbaikan dibuat (bagian 3). Sisa perbedaan kolom di E-70; master simbol UX-03; gambar dan tabel di sel UX-02; sorotan sesuai ASUMSI E-71 |
| M3.2 Pustaka proses | Pola TR-020 di artboard Sync check dan Compose; cakupan poin; isi poin; urutan standar proses | Kode tidak diubah. Pola TR-020 cocok dengan ASUMSI E-33. Cakupan poin menjawab E-72 (UX-01). Isi poin di E-73 dan UX-02. Urutan standar di E-76 |

## 3. Yang diubah

- **Migrasi 00013** (`sql/migrations/00013_review_ui_design.sql`):
  1. Kolom baru `format_symbols.automatic`. Simbol dengan key `revision` ditandai otomatis.
  2. Kolom Process Flow di isi awal CP disembunyikan, karena dibentuk dari PFD (R-DOC-09).
  3. Kolom Process Step / Function di isi awal PFMEA disembunyikan, karena berisi proses baris (ASUMSI E-70).

  Nomor 2 dan 3 hanya berlaku untuk versi 1 yang masih draft, dan field-nya disembunyikan, tidak dihapus (R-TPL-02). Role aplikasi tetap tidak bisa mengubah tanda `automatic`, karena hak UPDATE-nya hanya untuk kolom `name`.
- **Kode:**
  - `Symbol.Automatic` ditambahkan;
  - `CheckContent` menolak simbol otomatis di kolom simbol (`not_allowed`);
  - salinan template membawa tandanya, dan simpan simbol mempertahankan tanda lama, jadi simbol baru tidak pernah otomatis;
  - kontrak API menambah `automatic` (hanya baca) pada `FormatSymbol`;
  - layar Format templates menyebut simbol yang dipasang sistem.
- **Test:**
  - Go: isi awal, salinan, simpan simbol, penjaga database 42501, pemeriksa isi, dan endpoint;
  - Vitest: `automaticSymbolNames`;
  - E2E: Process Flow tersembunyi dan teks simbol otomatis.
- **Dokumen:**
  - UML 0.12 dari pemilik, lalu UML 0.13;
  - `docs/evaluasi-desain.md`: bukti baru untuk E-33 dan E-70 sampai E-73, temuan baru E-74 sampai E-76, dan tabel UX-01 sampai UX-07;
  - brief v1.20;
  - ringkasan desain UI;
  - CLAUDE.md.

## 4. Keputusan yang dibutuhkan, dengan saran

| ID | Pertanyaan | Saran Claude Code (usulan) |
|---|---|---|
| E-33 | Pencocokan PFD, PFMEA, dan CP di tingkat proses atau sub-proses | (a) tingkat proses. Pola TR-020 di desain menunjukkan nomor operasi yang sama, sub-proses sebagai pecahan CP, dan item cek khusus CP. Kode M3.2 sudah begitu |
| E-72 | Satuan process template | (a) satu template per proses, dengan cakupan melekat pada poin (masukan pemilik di UX-01) |
| E-70 | Sisa perbedaan kolom form (butir 3 sampai 6) | Bandingkan artboard "current form layout" dengan file Excel asli, lalu bangun dalam satu migrasi sebelum M4.1. **Diputuskan 4 November (D-124), dibangun di migrasi 00014** |
| E-73 | Isi poin process template | (b) isi poin mengikuti format template, ditambah asal field di salinan format, dengan bentuk isi sel yang sama dengan baris dokumen (UX-02) |
| E-74 | Kode customer tiga huruf | (a) kode customer boleh dua atau tiga huruf (D-100 diubah), setelah dipastikan AJ dan AJI customer yang sama. **Diputuskan 4 November: tetap dua huruf sesuai WI (D-125)** |
| E-75 | Arti register PL1 dan PL3 | Pemilik memastikan: plant, line, atau Project Leader. **Diputuskan 4 November: Project Leader (D-126)** |
| E-76 | Permintaan proses baru dan urutan standar proses | (a) untuk keduanya, dibangun di M4.5 dan M4.6 |
| UX-01 sampai UX-07 | Usulan dari desain UI | Lihat tabel di `docs/evaluasi-desain.md` |

**Urutan kerja yang disarankan setelah keputusan, sebelum Tahap 4:**
1. Impor M2.5 diperluas (UX-06, E-74, E-75).
2. Kolom form dan master simbol (E-70, UX-03).
3. Menu dan design system untuk halaman M0 sampai M3, ditambah katalog bahasa Indonesia bila D-96 diubah (UX-05).
4. M3.3 (cakupan per poin, UX-01), lalu M3.4 (versi template).

## 5. Konsep yang dipelajari

- **Desain sebagai bukti, bukan keputusan.** Artboard "current form layout" adalah bukti cara form ditulis sekarang. Usulan perilaku baru tetap Terbuka sampai pemilik memutuskan.
- **Hak kolom di PostgreSQL.** Hak INSERT tingkat tabel ikut mencakup kolom yang ditambah belakangan. Hak UPDATE per kolom (hanya `name`) membuat tanda `automatic` hanya bisa diatur lewat migrasi.
- **Membetulkan isi awal tanpa menghapus.** Field yang salah disembunyikan, sehingga ID-nya tetap dan admin masih bisa memunculkannya lagi.

## 6. Status test (container cloud, 3 November 2026)

- `go test ./...` (16 paket), `go vet ./...`, dan `gofmt -l .`: bersih.
- `npm --prefix web run check`: 0 error.
- `npm --prefix web test`: 84 test lulus.
- `npm --prefix web run test:e2e`: 30 test lulus.

## 7. Catatan lain

- **Render UML belum diperbarui.** File di `docs/uml/export/` lebih tua dari U3 draft 0.9, U5, dan U8. Render ulang di komputer pemilik dengan perintah di `docs/uml/README.md`.
- **Data di kanvas tidak disalin.** Kanvas memuat nama contoh orang dan nama perusahaan. Tidak ada yang disalin ke kode, test, atau isi awal (CLAUDE.md bagian 7).

## 8. Latihan tulis tangan (tanpa AI)

Di psql, sebagai role aplikasi (`docker compose exec db psql -U revioten_app -d revioten`):
1. Tulis query yang menampilkan semua simbol otomatis beserta nama format template dan nomor versinya. Hubungkan `format_symbols`, `format_template_versions`, dan `format_templates`.
2. Coba `UPDATE format_symbols SET automatic = false WHERE key = 'revision';`. Jelaskan dengan kata-kata sendiri kenapa ditolak dengan kode 42501, dan migrasi mana yang mengatur hak itu.

## 9. Tindak lanjut 4 November 2026

Pemilik menjawab tinjauan ini. Kutipan lengkapnya ada di `docs/evaluasi-desain.md`, bagian "Keputusan 4 November 2026". Keputusannya dicatat di brief v1.21 sebagai D-121 sampai D-126.

### 9.1 Jawaban atas tiga pertanyaan pemilik

1. **Tanda revisi dan version control (D-121).** Yang tercetak hanya tanda revisi yang berlaku. Saat dokumen naik dari revisi 1 ke revisi 2, tanda revisi 1 hilang dari cetakan. Semua perubahan tetap bisa dicek di sistem:
   - setiap revisi yang rilis dibekukan sebagai snapshot dengan SHA-256 dan tidak bisa diubah (R-REV-02, M5.5);
   - diff per baris dan sel antara dua revisi mana pun, beserta riwayat per dokumen dan per proses (R-REV-03, M5.6);
   - tabel riwayat revisi di cover (R-REV-04);
   - audit log.

   Tanda revisi dihitung dari diff saat dicetak, jadi tidak ada tanda yang disimpan atau dihapus tangan.
2. **Process Flow di CP (D-122).** Kolom ini selalu tercetak. Migrasi 00013 menyembunyikannya karena isinya bukan ketikan, tetapi itu membuatnya tampak hilang. Migrasi 00014 memunculkannya lagi sebagai **kolom terisi otomatis**: simbolnya diambil dari langkah PFD (R-DOC-09). Tujuan utama yang dicatat (G11) adalah penyesuaian alur tanpa menggambar ulang. Mengubah alur cukup mengubah daftar langkah PFD, dan konektor di perpindahan halaman dipasang otomatis saat dicetak (Tahap 6).
3. **Nama proses di PFMEA (D-123).** Nama proses tidak pernah dihapus dari dokumen. Yang disembunyikan migrasi 00013 adalah kolom ketikan yang isinya sama dengan proses baris. Sekarang kolom Process Item No. dan Process Step / Function tampil dan tercetak lagi, tetapi isinya diambil dari baris: nomor operasi, dan nama proses dari master proses lewat ID-nya. Pembaca selalu melihat nama proses, dan namanya tidak bisa berbeda antardokumen (F-05).

### 9.2 Yang diubah

- **Migrasi 00014** (`sql/migrations/00014_filled_columns_and_form_layout.sql`):
  - jenis kolom baru `operation_number`, `process`, dan `process_flow`, hanya untuk kolom baris (constraint `format_fields_filled_check`);
  - isi awal mengikuti form Excel asli (D-124):

    | Template | Kolom |
    |---|---|
    | PFD | No, Process, Flow Symbol, NG Disposition |
    | PFMEA | Process Item No., Process Step / Function, Requirement, Potential Failure Mode, ... Class, Potential Cause(s) / Mechanism(s) of Failure, ... Recommended Action, ... Result RPN (20 kolom) |
    | CP | Process Flow, No, Process Name / Operation Description, Machine, Equipment, Jigs, Fixture, Tools for Mfg, Product Characteristics, Product Specification / Tolerance, Process Characteristics, Process Specification / Tolerance, Special Characteristics, Equipment, Method, Frequency, PIC, Relevant Document, Pokayoke, Reaction Plan (16 kolom) |

  - bagian Down sudah dicoba di database percobaan: turun ke 00012 lalu naik lagi ke 00014.
- **Kode:**
  - jenis kolom baru di `internal/template`;
  - kolom terisi otomatis ditolak bila diketik (`not_allowed`), tidak pernah dianggap kosong, hanya untuk baris, dan paling banyak satu per jenis;
  - kontrak API dan layar Format templates menyebut "filled from the row", "filled from the process master", dan "filled from the PFD".
- **Test:**
  - Go: isi awal, aturan field, pemeriksa isi, dan urutan;
  - Vitest: 84 test;
  - E2E: 30 test, dengan label dan nomor kolom baru;
  - tangkapan layar template CP sudah diperiksa.
- **Dokumen:** brief v1.21 (D-121 sampai D-126, G11, R-REV-09 menjadi DECIDED), evaluasi desain (E-70 sebagian, E-74, dan E-75 diputuskan), UML draft 0.14, ringkasan desain UI, dan CLAUDE.md.

### 9.3 Masih terbuka

- E-33, E-72, E-73, dan E-76;
- usulan UX-01 sampai UX-07;
- sisa E-70: kolom wajib, teks kriteria, Key Date diketik atau terisi saat cetak, dan penomoran kolom No karakteristik CP oleh layout cetak.

