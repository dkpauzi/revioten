# D-227 · Lompatan di Process Flow, NG lurus ke kiri, dan kolom Process Flow lebih lebar

- **Tanggal:** 25 November 2026
- **Persetujuan pemilik:**
  - "Arah cabang ke NCP / Rework / Scrap, OK" (NG lurus ke samping);
  - "AOI: NG ke VMI, OK langsung ke ICT / Router / Finish 2nd side, OK";
  - "pastikan untuk konektor hanya menggunakan Huruf, karna angka sudah digunakan untuk nomor process";
  - "Lebarkan ke bobot 2,1", "Setuju, mulai A dan B".
- **Aturan yang dipakai:**
  - D-219: simbol alur dan konektor; arah kotak NG diubah oleh keputusan ini.
  - D-122 dan D-211: Process Flow otomatis, satu rencana halaman.
  - D-212: langkah PFD dibekukan di snapshot CP.
  - D-226: tepi halaman CP.
  - D-155: bagian per PCB.
  - D-141: Help.
- **Temuan:** E-147 (kolom Process Flow CP v1 hanya sekitar 25 mm).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| A. Lebar kolom | `defaultWidth` (`internal/printout/model.go`) memberi kolom Process Flow bobot 2,1. Di A3 landscape dengan Control Plan v1 lebarnya sekitar 37,6 mm (sebelumnya 24,1 mm). `flowX` menggeser pusat alur supaya kotak NG 12 mm muat di kiri, selama konektor lompatan di kanan masih muat. |
| A. NG lurus ke kiri | `ngBranch` (SVG) dan `flowShapes` (Excel): kotak NG sejajar dengan pusat simbol, panah mendatar 3 mm dengan label NG di atasnya (`ngBoxSpan`). |
| A. Contoh | Contoh PFMEA di test sekarang A3 landscape, sama dengan layout Standard PFMEA. |
| B. Field PFD | Migrasi 00060: **Jump to** (`jump_to`, teks) dan **Jump label** (`jump_label`: OK, NG, Yes, No) di versi draft PFD v1. Workbook memberi Jump to pilihan nomor langkah dokumen itu. |
| B. Langkah | `document.FlowStep` mendapat `jump` dan `jump_label` (omitempty, jadi snapshot lama tidak berubah). `StepsOfPFD`, `StepsOf` (`PFDFieldsOf`), dan `stepsOfFlow` membacanya. |
| B. Control Plan | `flowPages`: lompatan maju di halaman yang sama menjadi `FlowLine` (garis di kanan yang bergabung ke alur 1,5 mm di atas simbol tujuan). Lompatan mundur, yang menyeberang halaman, atau yang garisnya menumpuk menjadi konektor (`JumpOut`, `JumpIn`). Konektor halaman dan konektor lompatan memakai satu urutan huruf menurut urutan muncul. Label garis ke bawah (`Down`) adalah lawan Jump label. |
| B. Diagram PFD | `DiagramLayout`: lompatan di kolom yang sama menjadi garis di kanan, maju atau mundur, dengan jalur sendiri bila ada beberapa (`freeLane`). Selain itu menjadi konektor huruf. Diagram di layar (`pfdDiagram.ts`) menambah panah berlabel. |
| B. Gambar | `flowjump.go`: bentuk lompatan dihitung sekali sebagai `jumpPart` (`cpJumpParts`, `diagramJumpParts`), lalu digambar ke SVG (`partsSVG`) dan shape Excel (`partsXLSX`). |
| B. Peringatan | `Plan.Warnings` dan field `warnings` di `GET /api/v1/documents/{id}/page-preview`: `jump_unknown`, `jump_self`, `jump_not_decision`, `flow_narrow` (kolom di bawah 37 mm). Tampil sebagai **Check of the process flow** di atas pratinjau. |
| Help | Bab dokumen (Jump to dan Jump label, gambar lompatan), bab Printing (lebar kolom 37 mm dan layout yang tidak dibekukan). |

## 2. Hasil diperiksa mata

Contoh dibuat dengan `REVIOTEN_SAMPLE_DIR=<folder> go test -short ./internal/printout -run TestSampleTokairika`. Hasilnya `sample-cp-jump` dan `sample-pfd-jump`, dengan kolom Control Plan v1 yang sebenarnya.

**CP:**
- Garis OK dari AOI 5.7.1 lewat kanan dan bergabung di atas Router 6.1. Garis NG turun ke VMI.
- 6.2 Finish 2nd side menuju konektor A dengan label No. Konektor A yang sama muncul di kanan 5.1 Loading PCB.
- Konektor halaman memakai huruf B.
- Kotak SCRAP dan NCP sejajar dengan simbolnya.

**PFD:**
- Garis OK AOI ada di kolom kedua.
- No dari 6.2 memakai konektor A ke 5.1 di kolom pertama, karena kedua langkah ada di kolom yang berbeda.

## 3. Cara mengulang dari nol

1. Tulis test dulu:
   - `TestFlowColumnWidth`, `TestNGBoxSideways`;
   - `TestFlowJumps*`, `TestJumpWarnings`, `TestFlowJumpsFit`, `TestDiagramJumps`, `TestJumpXLSX`;
   - perluasan `TestStepsOfPFD` dan `TestFlowFrozenAtRelease`.
2. Ubah `defaultWidth` dan `ngBranch`, lalu tambah migrasi 00060.
3. Tambah field lompatan di `FlowStep` dan `Step`.
4. Ubah `flowPages` dan `DiagramLayout` untuk menempatkan lompatan, lalu tambah `flowjump.go` untuk menggambarnya.
5. Tambah `warnings` di OpenAPI, lalu jalankan `go generate ./internal/server/` dan `npm --prefix web run gen:api`.
6. Ubah layar: pratinjau, Workbook, dan `pfdDiagram.ts`.
7. Tambah i18n dan Help, lalu jalankan `help:shots`.

## 4. Konsep yang dipelajari

- **Satu daftar bagian gambar, dua keluaran.** Garis, panah, konektor, dan label lompatan dihitung sekali (`jumpPart`), lalu SVG dan Excel hanya menerjemahkan satuan. Ukurannya otomatis sama.
- **Urutan huruf dari peristiwa.** Konektor halaman dan konektor lompatan dikumpulkan sebagai peristiwa dengan posisi baris, diurutkan, lalu diberi huruf. Dengan begitu satu urutan A, B, C dipakai bersama tanpa huruf ganda.

## 5. Asumsi

- **Lompatan mundur di CP selalu memakai konektor,** juga bila kedua langkah di halaman yang sama, sesuai uraian pemilik ("jika di Control Plan bisa di hubungkan dengan connector"). Di diagram PFD, lompatan di kolom yang sama memakai garis langsung.
- **Tujuan dicari di bagian per PCB yang sama.** Bila nomor langkah dipakai dua kali di satu bagian, yang pertama dipakai.
- **Lompatan di simbol selain Decision tetap digambar,** tetapi diberi peringatan.
- **Template PFD yang versinya sudah terbit** mendapat kedua field lewat versi baru (Add a field dengan key yang sama). Migrasi hanya mengubah versi draft PFD v1.
- **Layout tidak dibekukan di revisi,** jadi cetak ulang revisi terbit memakai lebar kolom yang baru. Ini sudah begitu sebelum D-227; dicatat di Help.

## 6. Temuan desain baru

E-147 (sudah diputuskan, D-227). Tidak ada temuan terbuka baru.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test ./...`, `go vet`, `gofmt` | semua lulus, bersih |
| `npm --prefix web run check` | 0 error |
| Vitest | 196 lulus |
| `npm --prefix web run test:e2e` | 80 test: 78 lulus; `interaction` dan `m8-roles` gagal di jalan penuh (menu tidak terbuka tepat waktu), lalu lulus saat diulang sendiri. Tidak terkait D-227 |
| help-shots | lulus, gambar dan build diperbarui |

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `letters(pageEnds []int, jumps [][2]int) []string`:
- `pageEnds` berisi baris terakhir setiap halaman yang alurnya berlanjut;
- `jumps` berisi pasangan baris asal dan tujuan lompatan yang memakai konektor.

Fungsi memberi huruf A, B, C menurut urutan muncul. Konektor halaman ada di posisi baris terakhir + 0,5, dan konektor lompatan di baris yang lebih atas dari asal dan tujuannya.

Uji dengan `pageEnds = {2}` dan `jumps = {{4, 0}, {1, 3}}`. Hasilnya: lompatan 4→0 = A, lompatan 1→3 = B, batas halaman = C. Bandingkan dengan `flowPages` di `internal/printout/plan.go`.
