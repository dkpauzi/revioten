# Form set langkah B dan C · Form PFMEA AIAG-VDA 2019 dan Control Plan AIAG 2024

- **Tanggal:** 11 dan 12 November 2026
- **Langkah:** B dan C di `docs/rencana/form-set-aiag-vda.md` (bagian data dari milestone 3 dan 4 di prompt form set).
- **Keputusan yang dipakai:** D-133 (form set, form kompatibel AIAG-VDA), D-134 (AP dari tabel), D-135 (fase CP 2024, tanpa Process Flow), D-140 (urutan kerja), D-122 dan D-123 (kolom terisi otomatis).
- **Persetujuan pemilik:** "E-80 A" dan "langsung lanjutkan hingga M4 selesai" (11 November 2026).

## 1. Tujuan dan apa yang dibuat

Dua form baru tersedia sejak database dibuat, masing-masing versi 1 berstatus draft:

| Form | Isi |
|---|---|
| **PFMEA AIAG-VDA 2019** | 12 field header dan 29 kolom dalam lima grup (langkah 2 sampai 6), metode risiko AP dengan tabel handbook, 30 baris kriteria skala kosong, 4 simbol |
| **Control Plan AIAG 2024** | 14 field header termasuk Phase (Prototype, Pre-launch, Safe Launch, Production), 14 kolom, tanpa Process Flow, 4 simbol |

Yang baru di balik layar:

1. **Key tetap per field** (`format_fields.key`), misalnya `failure_mode`. Label hanya tampilan; key dipakai untuk memetakan kolom antar form nanti dan untuk aturan yang menunjuk kolom tertentu.
2. **Judul grup kolom** (`format_version_fields.group_label`), misalnya "FAILURE ANALYSIS (STEP 4)". Tampil di bawah label kolom di halaman Format templates.
3. **Jenis kolom `ap`**: Action Priority yang dihitung dari S, O, D dengan set yang sama (set 1 nilai sekarang, set 2 sesudah tindakan). Tidak bisa diketik.
4. **Jenis field `document_number`**: nomor dokumen dari register, hanya di header, tidak diketik.
5. **Aturan sorot AP**: baris dengan AP H tanpa Prevention Action dan Detection Action ditandai. Hanya tanda, tidak memblokir (R-TPL-11).
6. **Metodologi `aiag_cp_2024`**, supaya CP lama dan CP 2024 bisa dibedakan.

Belum dibuat: tampilan sheet di editor (judul vertikal, sel gabung, kotak centang fase), yang menumpang M4.2 sampai M4.4; layar untuk mengubah judul grup; form set itu sendiri (langkah D).

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00018_aiag_vda_forms.sql` |
| Aturan | `internal/template/template.go`: `AP`, `Highlights`, `cleanNewField`, `KindAP`, `KindDocumentNumber` |
| API | `key` dan `group` di `FormatField`; jenis `ap` dan `document_number`; metodologi `aiag_cp_2024` |
| Layar | `FieldTable.svelte` (judul grup), `formats.ts` (nama jenis), bab Format templates di halaman Help |

## 2. Cara mengulang dari nol

1. Tulis `internal/template/forms_vda_test.go` dulu: daftar key header dan kolom kedua form, jenis dan grupnya, hitungan AP di isi baris, dan aturan kolom AP.
2. Tulis migrasi 00018 (kolom baru, CHECK, isi awal), ubah query, lalu `go generate ./...`.
3. Tambah jenis baru dan fungsi `AP` di `template.go` sampai `go test -p 1 ./internal/template/` lulus.
4. Ubah `api/openapi.yaml`, `go generate ./...`, `npm --prefix web run gen:api`, lalu layar dan test E2E.

## 3. Konsep yang dipelajari

- **Key dan label dipisah.** ID untuk menyimpan data, key untuk aturan dan pemetaan, label untuk manusia. Masing-masing boleh berubah tanpa mengganggu yang lain.
- **Mengubah CHECK.** PostgreSQL tidak punya "ubah CHECK"; constraint lama dibuang lalu dibuat lagi dengan daftar baru.
- **Isi awal lewat blok `DO`.** Satu daftar `VALUES` diputar dengan `FOR ... LOOP` untuk membuat identitas field dan isinya per versi.
- **Aturan yang menunjuk kolom lewat key.** Aturan sorot AP mencari kolom `prevention_action` dan `detection_action`, bukan labelnya, jadi tetap bekerja setelah label diganti.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Metodologi baru `aiag_cp_2024` | Prompt meminta pilihannya dijelaskan: nilai sendiri membuat CP lama dan CP 2024 bisa dibedakan tanpa membaca nama template |
| Field lama tidak diberi key | Key form lama dibutuhkan saat perpindahan AIAG 4th ke AIAG-VDA (langkah J); memberinya sekarang berarti menebak pemetaan |
| Kedua form dibuat sebagai draft | Sama dengan form lama: admin menyesuaikan label dan kriteria skala dulu, lalu menerbitkan |

## 5. Asumsi

- **ASUMSI E-88:** dua pasang label yang sama di form asli diberi pembeda, karena label wajib unik per area: "PFMEA AP" dan "PFMEA AP (after action)", serta "Other Approval/Date (if req'd) 1" dan "2".
- **ASUMSI E-89:** `process_item` dan `process_step` di PFMEA AIAG-VDA dibuat sebagai teks, seperti tabel di prompt. Pemilik memutuskan lain pada 12 November 2026 (D-144): `process_step` diisi dari master proses; diubah di migrasi 00019.
- `date_rev` di CP 2024 dibuat sebagai tanggal biasa; pengisian otomatis saat cetak menumpang Tahap 6.
- Kriteria skala S, O, D form AIAG-VDA dibiarkan kosong, seperti form lama; isinya dari form yang dipakai perusahaan.

## 6. Temuan desain baru

- **E-88**: label kembar di form asli. Diputuskan 12 November 2026 (D-147): label cetak terpisah, dibangun bersama ekspor.
- **E-89**: nama proses di PFMEA AIAG-VDA. Diputuskan 12 November 2026 (D-144): diisi dari master proses.

## 7. Status test (komputer pemilik, Windows, 12 November 2026)

- `go test -p 1 ./...`: semua paket lulus.
- `npm --prefix web run check`: 0 error. Vitest: 119 test lulus. E2E: 41 test lulus.

## 8. Latihan tulis tangan (tanpa AI)

1. Buka form PFMEA AIAG-VDA 2019 di halaman Format templates. Di kertas, tulis lima grup kolomnya dan kolom mana saja yang tidak diketik.
2. Di psql, tulis query yang menampilkan `key`, label, dan `group_label` semua kolom form itu menurut urutan. Petunjuk: hubungkan `format_version_fields` dengan `format_fields`.
