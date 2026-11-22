# D-221 · Efek tombol dan kamera tur (rencana E dan T)

- **Tanggal:** 23 November 2026
- **Keputusan:** D-221 (brief v1.50).
- **Persetujuan pemilik:**
  - "Lanjut ke efek tombol dan kamera tur";
  - logo: "cetakan, termasuk cetak ulang revisi lama, memakai logo yang berlaku saat dicetak".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian E dan T.
- **Aturan yang dipakai:** D-127 (gaya diatur sekali di `app.css`), D-128 (tampilan per perangkat).

## 1. Tujuan

Bagian yang sedang dikerjakan (tombol, kotak isian, centang, baris) mudah dilihat, baik oleh pengguna maupun saat tur otomatis berjalan.

## 2. Apa yang dibuat

| Bagian | Isi |
|---|---|
| `web/src/app.css` | **Focus:** tombol juga diberi garis aksen saat diklik mouse. **Klik:** tombol tertekan (turun 1 piksel dan lebih gelap). **Hover:** kotak isian dan pilihan bergaris aksen, label centang dan radio berlatar aksen tipis, baris tabel data berlatar aksen tipis. **Terpilih:** tautan halaman aktif bercetak tebal. Perubahan warna 120 ms, dan dimatikan untuk `prefers-reduced-motion`. |
| `web/tour/act.ts` | `installSpotlight(page, waitMs)` membungkus `click`, `fill`, `check`, `uncheck`, `selectOption`, `setInputFiles`, `press`, dan `pressSequentially` pada Locator. Sebelum aksi, elemen digulir ke tengah layar dan diberi cincin jingga. Elemen yang belum siap dilewati tanpa menghentikan tur. |
| Tur (lokal) | Lama sorotan: `slow` 1,2 detik, `normal` 0,7 detik, `fast` tanpa sorotan. |
| Test | `web/e2e/interaction.test.ts`: focus tombol, hover kotak isian, hover label centang, hover baris tabel. |

## 3. Cara mengulang dari nol

1. Tulis `interaction.test.ts` dengan `toHaveCSS` untuk setiap keadaan.
2. Tambah aturan di akhir `app.css` sampai test lulus, lalu jalankan `responsive.test.ts`.
3. Tulis `act.ts`, panggil `installSpotlight` di awal tur, lalu jalankan tur dengan `TOUR_SPEED=normal`.

## 4. Konsep yang dipelajari

- **`:focus` dan `:focus-visible`.** `:focus-visible` hanya muncul lewat keyboard. Supaya klik mouse juga terlihat, tombol memakai `:focus`.
- **`:has()` di CSS.** `label:has(> input[type='checkbox'])` memberi gaya ke label yang memuat centang, tanpa kelas tambahan.
- **Membungkus method satu kali.** Semua Locator berbagi prototype yang sama. Membungkus method di prototype itu membuat seluruh langkah tur mendapat sorotan tanpa mengubah langkahnya satu per satu.

## 5. Asumsi

- Hover baris hanya untuk tabel tanpa kelas. Tabel cetak, pratinjau, dan workbook (Tabulator) punya kelas atau komponen sendiri, jadi tidak berubah.
- `act.ts` di-commit karena tidak memuat data. `alur.tour.ts` tetap hanya ada di komputer pemilik.

## 6. Temuan desain baru

Tidak ada.

## 7. Status test

| Test | Hasil |
|---|---|
| `npm --prefix web run check` | 0 error |
| `npm --prefix web run test:e2e` | 73 lulus (baru: `interaction.test.ts`) |
| Tur `normal` tanpa jendela | Lulus (20 menit) dengan kamera aktif |
| Pemeriksaan mata kamera | Pemeriksaan sementara memotret di tengah aksi. Cincin awalnya tergeser karena posisi diukur saat gulir halus belum selesai; sekarang cincin mengikuti elemen setiap frame dan tepat di sekeliling tombol yang digulir ke tengah. |
| help-shots | Lulus |

## 8. Latihan tulis tangan

Tanpa AI, tulis aturan CSS yang membuat tombol tertekan 1 piksel saat diklik, tetapi tidak saat `disabled`, dan matikan geraknya bila pengguna memilih `prefers-reduced-motion`.
