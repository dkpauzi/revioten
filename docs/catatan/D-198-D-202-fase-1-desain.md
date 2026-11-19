# D-198 sampai D-202 · Fase 1: tampilan Home, review 3 tahun, dan kinerja PPAP mengikuti desain

- **Tanggal:** 19 November 2026
- **Keputusan:** D-198 (tanda tangan digital diizinkan, menggantikan D-07 dan R-APR-04; dibangun di fase 3), D-199 (matriks peran; fase 2), D-200 (EOL model), D-201 (tanpa target), D-202 (fase 1). Brief v1.36.
- **Migrasi baru:** 00048 (`models.eol_at`, `document_revisions.ready_at`, `change_sources.report_key`).
- **Desain acuan:** kanvas "Desain UI Sistem PFD PFMEA CP", baris "3-year review warning and PPAP Engineer performance" (HomeV2, ReviewDue, PpapPerformance, PpapPerformanceYear).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| **Password** | Tombol mata di kotak password (login, ganti password); komponen `web/src/lib/PasswordInput.svelte` |
| **Home** | Sapaan, spanduk merah "N document sets need their 3-year review", empat kartu aksi menurut hak, **Waiting for you** (`GET /api/v1/inbox`), **Opened recently** (disimpan di browser), dan "New here?" |
| **3-year review** | Empat kartu (Overdue, Due within 90 days, Due within 1 year, Reviewed this year), tab, saringan customer dan engineer, **Show EOL models**, kolom PFD, PFMEA, dan Control Plan dengan revisi dan tanggal masing-masing (merah bila lewat), panel **Start review** dengan dua hasil, tabel "How the 3-year clock works", **Download list (Excel)** |
| **PPAP Engineer performance** | Week, Month, Year; minggu laporan Jumat sampai Kamis dengan nomor minggu; saringan engineer dan customer; perbandingan dengan periode sebelumnya; empat tahap waktu; pemicu (ECN/ECR, Customer claim, Internal audit finding, Yokoten, 3-year review, Other); tren 8 minggu atau 12 bulan; tabel yokoten dengan cakupan; tabel per engineer; **Download report (Excel)** |
| **Models** | Kolom End of life dan tombol **Mark EOL** / **Not EOL** untuk pemegang E-2 (PPAP Engineer, Project Leader) |
| **Revisions** | Tombol **Ready to print** pada draft, untuk tahap "menunggu cetak" di laporan |

## 2. Yang dibuat berbeda dari desain, dan alasannya

| Di desain | Di kode | Alasan |
|---|---|---|
| Review tanpa perubahan hanya catatan teks | Ditambah daftar periksa 8 butir dan konfirmasi orang kedua | Bukti untuk auditor (E-86, masih asumsi) |
| "Working days", tanpa akhir pekan dan libur | Waktu kalender | Belum ada kalender libur perusahaan (E-130) |
| Target ≤ 5 hari dan warna merah di atas target | Tidak ada | D-201 |
| "Active working hours" untuk edit sampai siap cetak | Waktu kalender dari edit pertama (audit log) sampai tanda siap cetak | Sistem tidak tahu jam kerja aktif (E-130) |
| Tombol "Open genba view" | Tidak ada | Tampilan genba (D-137) belum dibangun |
| Menu atas datar dengan Review dan Reports | Menu berkelompok tetap; Review dan laporan ada di kelompok Documents | Halaman sudah terlalu banyak untuk menu datar (D-127) |
| Kolom Engineer | Orang yang membuat dokumen pertama set itu | Belum ada data penanggung jawab model (E-132) |

## 3. Cara mengulang dari nol

```bash
docker compose up -d
go generate ./internal/db/... ./internal/server/...
npm --prefix web run gen:api
go test -p 1 ./...
npm --prefix web test
npm --prefix web run check
npm --prefix web run test:e2e
npm --prefix web run help:shots
TOUR_SPEED=fast TOUR_HEADLESS=1 npm --prefix web run tour
```

## 4. Konsep yang dipelajari

- **Jam per dokumen, peringatan per set.** Setiap dokumen punya jatuh temponya sendiri; set mengikuti dokumen yang paling dulu jatuh tempo, dan layar menyebut dokumen mana yang terlambat.
- **Kunci laporan yang tahan ganti nama.** Pemicu dikelompokkan lewat `report_key`, bukan nama yang bisa diubah admin.
- **Grafik tanpa library.** SVG biasa: skala `y = top + tinggi - nilai / maks × tinggi`. Tabel tersembunyi di bawah grafik membuat angkanya terbaca pembaca layar.
- **Unduhan lewat tautan.** Browser mengirim cookie sesi sendiri, jadi tautan `download` ke API cukup.

## 5. Asumsi

- **ASUMSI E-86:** review tanpa perubahan tetap dengan daftar periksa dan konfirmasi orang kedua.
- **ASUMSI E-130 (diperluas):** "mulai edit" = edit pertama draft menurut audit log; "siap cetak" = tombol Ready to print, atau cetakan untuk tanda tangan bila tidak ditekan; waktu kalender, bukan hari kerja.
- **ASUMSI E-132:** engineer set = orang yang membuat dokumen pertama set itu.

## 6. Temuan desain baru

- E-132 · Penanggung jawab model (engineer) di daftar review dan laporan.

## 7. Status test

- Go: semua paket lulus, termasuk test baru `internal/inbox`, EOL dan jatuh tempo per dokumen di `internal/review`, laporan kinerja (tahap, tren, saringan orang, Excel), dan tabel izin (5 operasi baru).
- Vitest: 164 lulus (termasuk `reports.spec.ts`).
- E2E dan tur: lihat ringkasan sesi.

## 8. Latihan tulis tangan

Tanpa melihat kode, tulis fungsi `compactNumbers(['SH-KT-005', 'SF-KT-005', 'SC-KT-005'])` yang menghasilkan `SH/SF/SC-KT-005`, dan menulis nomor apa adanya bila bagian setelah tanda hubung pertama berbeda. Bandingkan dengan `web/src/lib/reports.ts`.
