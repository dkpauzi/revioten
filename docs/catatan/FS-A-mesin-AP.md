# Form set langkah A · Mesin AP

- **Tanggal:** 11 November 2026
- **Langkah:** A di `docs/rencana/form-set-aiag-vda.md` (milestone 2 di prompt form set).
- **Keputusan yang dipakai:** D-133 (form set), D-134 (tabel AP mengikuti handbook, disimpan sebagai data), D-140 (urutan kerja), D-139 (penyetuju, dibangun di langkah D).
- **Persetujuan pemilik:** "kerjakan mesin AP dulu" (11 November 2026).

## 1. Tujuan dan apa yang dibuat

Action Priority (AP) adalah pengganti RPN di PFMEA AIAG-VDA. RPN dihitung (S x O x D). AP **dibaca dari tabel**: setiap dari 1.000 kombinasi S, O, D punya satu prioritas, yaitu H, M, atau L.

1. **Tabel sebagai data.** Kolom `ap_rules` (JSONB) di versi format template. Satu aturan berupa satu kotak rentang, misalnya S 9-10, O 4-5, D 7-10 = H. Tabel handbook menjadi 84 aturan.
2. **Fungsi hitung murni** `ActionPriority(rules, s, o, d)` di `internal/template/ap.go`, tanpa database.
3. **Pemeriksaan tabel** `checkAPRules`: setiap kombinasi ditutup tepat satu aturan.
4. **Isi awal.** Versi baru dengan metode risiko AP langsung mendapat tabel handbook. Tabel ikut disalin ke versi baru dan ke salinan template.
5. **API.** `PUT .../ap-rules` (A-2, hanya draft) dan `GET .../action-priority?severity=&occurrence=&detection=` (L-2).
6. **Layar.** Halaman Format templates menampilkan tabel AP dan kalkulator "Try a combination" untuk versi bermetode AP.

Belum dibuat di langkah ini: layar untuk mengubah tabel AP, hak penyetuju (D-139), kolom jenis `ap` di form (langkah B), dan aturan sorot AP H tanpa tindakan (langkah B).

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00017_ap_rules.sql` |
| Aturan | `internal/template/ap.go`, `internal/template/ap_rules_aiag_vda_2019.json` |
| API | `setFormatApRules`, `getActionPriority`, `ap_rules` di `FormatVersion` |
| Layar | `web/src/lib/ap.ts`, `web/src/lib/formats/ApTable.svelte` |

## 2. Cara mengulang dari nol

1. Tulis `internal/template/ap_test.go` dulu: 1.000 kombinasi dibandingkan dengan file paket desain, enam contoh prompt, tabel yang berlubang atau tumpang tindih, dan alur versi.
2. Ubah file paket desain menjadi 84 aturan datar (`ap_rules_aiag_vda_2019.json`), tulis migrasi 00017, tambah query, lalu `go generate ./...`.
3. Tulis `ap.go` sampai `go test -p 1 ./internal/template/` lulus.
4. Tambah dua operasi di `api/openapi.yaml`, `go generate ./...`, `npm --prefix web run gen:api`, lalu handler.
5. Tulis `web/src/lib/ap.spec.ts`, `ap.ts`, dan `ApTable.svelte`, lalu test E2E di `formats.test.ts`.

## 3. Konsep yang dipelajari

- **Tabel keputusan sebagai data.** Aturan bisnis yang bisa dikoreksi disimpan di database dan diberi versi, bukan ditulis sebagai `if` di kode.
- **Memeriksa kelengkapan dengan mencoba semua kasus.** 1.000 kombinasi cukup kecil untuk diperiksa satu per satu, jadi lubang dan tumpang tindih pasti ketahuan.
- **`go:embed`.** File JSON ikut masuk ke binary. File yang di-embed harus berada di folder paketnya, jadi isinya disalin dari `docs/design/` dan test memastikan keduanya sama.
- **JSONB dengan CHECK.** Database hanya memastikan bentuk kasarnya (array, dan hanya untuk metode AP); isi rincinya diperiksa layanan.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Tabel disimpan sebagai JSONB di baris versi, bukan tabel `format_ap_rules` terpisah | Prompt mengizinkan keduanya. Tabel selalu dibaca dan diganti utuh, dan baris versi sudah dijaga trigger "versi terbit tidak berubah" |
| Kalkulator di layar menghitung di browser | Jawaban langsung tanpa permintaan ke server; test E2E memastikan server memberi hasil yang sama |
| Audit mencatat jumlah kombinasi yang berubah, bukan seluruh tabel | Entri audit tetap pendek; isi tabel lama tetap ada di versi sebelumnya |

## 5. Asumsi

- Isi tabel berasal dari file paket desain pemilik. Saya **belum mencocokkannya dengan buku handbook** (berbayar). Enam contoh di prompt cocok.
- Klaim "186 dari 1.000 kombinasi berbeda dari sheet Excel lama" (D-134) tidak saya hitung ulang, karena sheet itu tidak ada di repo.
- Sampai hak penyetuju D-139 dibangun (langkah D), admin (A-2) bisa mengubah tabel AP di draft dan menerbitkannya sendiri.

## 6. Temuan desain baru

Tidak ada temuan baru. E-87 diputuskan pemilik pada hari yang sama (D-143).

## 7. Status test (komputer pemilik, Windows, 11 November 2026)

- `go test -p 1 ./...`: semua paket lulus.
- `npm --prefix web run check`: 0 error. Vitest: 112 test lulus. E2E: 39 test lulus.

## 8. Latihan tulis tangan (tanpa AI)

1. Di kertas, cari AP untuk S 7, O 5, D 6 dengan membaca tabel di layar. Lalu cocokkan dengan kalkulator.
2. Tulis fungsi `ActionPriority` sendiri di file kosong tanpa melihat `ap.go`, lalu jalankan `TestActionPriorityExamples` terhadap fungsi Anda.
