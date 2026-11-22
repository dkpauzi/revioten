# D-215, D-216 · Menu khusus migrasi dan nomor lama tiga kolom

- **Tanggal:** 22 November 2026
- **Keputusan:** D-215 (satu menu khusus migrasi, memutuskan E-141), D-216 (nomor lama tiga kolom, bawaannya satu set, memutuskan E-140). Brief v1.45.
- **Persetujuan pemilik:** "Ya, mulai", setelah rencana disampaikan. Arti "tidak satu set" menurut pemilik: nomor berjalannya tidak sejajar, misalnya SC-TR-005, SF-TR-005, SH-TR-004. Ketiganya tetap satu set dokumen.
- **Aturan yang dipakai:** R-MIG-01 sampai R-MIG-07, R-NUM-06, R-POR-01, R-SEC-11 (empat mata), M2.4 (set = satu batch), M7.1, M7.5.

## 1. Tujuan

Pemilik menilai alur migrasi "sangat membingungkan". Sebelumnya, satu set dokumen lama harus melewati enam tempat. Nomor asli seperti `SH-TR-020` juga hanya bisa masuk lewat menu impor, dan impor sekaligus beberapa set membuat semuanya satu batch (E-140). Sekarang satu set dokumen lama masuk dari satu halaman.

## 2. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Nomor lama satu set (`internal/numbering/legacy.go`) | `RegisterLegacySet` berjalan di transaksi pemanggil:<br>1. Customer dan penghitung dikunci seperti pada pendaftaran.<br>2. Setiap nomor dibaca dengan pola sekarang. Awalannya harus sesuai kolomnya, dan kode customer-nya harus kode customer itu.<br>3. Dengan OneSet, nomor yang kosong diisi dengan nomor berjalan yang sama.<br>4. Nomor baru disimpan berstatus Released dalam satu batch, lalu penghitungnya dinaikkan. Nomor yang sudah ada dikembalikan apa adanya. |
| Impor tiga kolom (`internal/numbering/import.go`) | Satu baris berisi sampai tiga kolom, dalam urutan bebas, karena jenisnya dibaca dari awalan. Setiap baris mendapat batch sendiri, sehingga satu baris adalah satu set (menutup E-140). Hasil baru `type_repeated` untuk dua nomor sejenis di satu baris. `OneSet` menambah nomor yang mengikuti dan menandainya `derived`. |
| Set dokumen lama (`internal/document/migrate_set.go`) | `MigrateSet` dikerjakan dalam satu transaksi: nomor didaftarkan, nomor diperiksa satu batch, lalu dokumen dibuat lewat `migrateOne` (isi lama `Migrate` yang dipisah). Penolakan diberi nama field `documents.<jenis>.<field>`. `MigrationOverview` memberi prasyarat dan papan status. |
| Query | `FindRegisteredNumber`, `MigrationReadiness`, `ListLegacyDocuments` |
| API | `GET /api/v1/migration/overview`, `POST /api/v1/migration/sets`. Impor nomor menerima `one_set` dan menjawab dengan `derived`. Endpoint lama `migration/numbers` dan `migration/documents` tetap ada. |
| Layar | Documents → **Legacy documents** (`/migration`, hak E-10) dengan tiga bagian: **Ready to migrate?**, **Enter one set** (tabel PFD, PFMEA, CP, centang **One set**), dan papan status. Halaman Documents hanya memberi tautan ke menu ini. Halaman impor punya centang **One set per line**. |
| Help, tur, gambar | Bab Legacy documents dan bab nomor (en dan id). Langkah 28 tur memakai menu baru. Gambar `migration` sekarang memotret halaman menu migrasi. |

## 3. Cara mengulang dari nol

1. Tulis `TestImportSets`: tiga kolom, batch per baris, `type_repeated`, dan OneSet dengan nomor yang mengikuti.
2. Ubah `splitImportLines` dan `insertImported` sampai test lulus. Sesuaikan `TestImport`: kolom keempat diabaikan, dan jumlah batch sama dengan jumlah baris.
3. Tulis `TestMigrateSet` untuk kasus berikut:
   - mode mati;
   - akun tanpa E-10;
   - penolakan per kolom tanpa meninggalkan nomor;
   - satu set dari satu nomor;
   - nomor berikutnya melanjutkan;
   - nomor yang dipakai ulang;
   - nomor tidak sejajar;
   - CP saja;
   - nomor dari set terdaftar;
   - campuran set ditolak;
   - papan status.
4. Tulis `legacy.go`, `migrate_set.go`, dan query, sampai test lulus.
5. Kontrak API, handler, lalu tabel izin di `permissions_test.go`.
6. `migration.spec.ts`, lalu `migration.ts`, halaman `/migration`, teks dua bahasa, menu, Help, dan E2E.

## 4. Konsep yang dipelajari

- **Satu transaksi untuk dua paket.** Penomoran (`numbering`) dan dokumen (`document`) adalah dua layanan. Supaya keduanya "semua atau tidak sama sekali", `RegisterLegacySet` menerima `*db.Queries` milik transaksi pemanggil dan tidak membuka transaksinya sendiri.
- **Kelompok dari data yang sudah ada.** Set dikenali dari batch. Impor cukup membuat satu batch per baris, tanpa kolom atau tabel baru.
- **Memisahkan isian di layar, memeriksa di server.** Layar membantu: memisahkan revisi dan memberi contoh nomor yang mengikuti. Server tetap memeriksa pola, kode customer, dan nomor berjalan, karena API bisa dipanggil tanpa layar.

## 5. Asumsi

- Rincian menu migrasi (semula ASUMSI E-142, diputuskan D-217 pada hari yang sama):
  - revisi dikenali dari segmen angka keempat;
  - E-10 cukup untuk mendaftarkan nomor lewat menu migrasi;
  - campuran batch ditolak;
  - nama yang kosong mengikuti baris pertama.
- Model EOL tidak ditawarkan di menu migrasi, dan tidak dihitung di prasyarat.

## 6. Temuan desain baru

- **E-142:** rincian menu migrasi. Diputuskan pemilik (D-217, brief v1.46): sesuai yang dibangun.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test ./internal/numbering/ ./internal/document/ ./internal/server/` | Lulus (`TestImportSets`, `TestMigrateSet`, `TestImport` disesuaikan, tabel izin) |
| `npm --prefix web test` | 188 lulus (baru: `migration.spec.ts`) |
| `npm --prefix web run check` | 0 error |
| `go test -p 2 ./...` | Semua paket lulus |
| `npm --prefix web run test:e2e` | 71 lulus (`m7-migration` memakai menu baru; impor tiga kolom di `numbers.test.ts`) |
| Tur (`fast`, tanpa jendela) dan help-shots | Lulus; gambar `migration` diperbarui |

Satu kesalahan ditemukan E2E dan dibetulkan: input revisi `type=number` memberi angka, sehingga `.trim()` melempar error dan tombol tertahan. Sekarang `revisionText` menerima teks, angka, atau kosong, dan test-nya ada di `migration.spec.ts`.

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi TypeScript `splitLegacyNumber(text)` yang mengubah `"sh-tr-020-02"` menjadi `{ number: "SH-TR-020", revision: 2 }` dan membiarkan `"SH-TR-020"` apa adanya (revisi `null`). Lalu tulis tiga kasus test-nya.
