# D-141 · Halaman Help (buku panduan pengguna)

- **Tanggal:** 11 November 2026
- **Keputusan yang dipakai:** D-141 (buku panduan untuk pengguna, berbentuk halaman Help di dalam aplikasi), D-128 (dua bahasa), D-142 (pop-up).
- **Persetujuan pemilik:** "D-141 gunakan halaman help" dan "langsung lanjutkan dengan buku panduan" (11 November 2026).

## 1. Tujuan dan apa yang dibuat

Halaman **Help** (`/help`), dibuka dari tautan **Help** di bilah atas, berisi 13 bab menurut urutan kerja: login, layar, customer dan model, akun dan peran, jabatan dan rute, format template, proses, process template, versi, nomor dokumen, dokumen kontrol, audit log, dan bila ada masalah.

1. **Dua bahasa.** Isi di `web/src/lib/help/en.ts` dan `id.ts` dengan bentuk yang sama.
2. **Nama tombol selalu cocok dengan layar.** Di teks panduan, `{kunciTeksLayar}` diganti teks dari katalog layar. Bila label layar berubah, panduan ikut berubah.
3. **Gambar otomatis.** `npm --prefix web run help:shots` menjalankan aplikasi di database uji, mengerjakan alurnya, dan memotret 11 layar dalam dua bahasa ke `web/static/help/<bahasa>/`. Kata sandi sementara ditutup.
4. **Bisa dicetak.** Tombol **Print this guide**; bilah atas dan tombol tidak ikut tercetak.

Sekaligus diperbaiki:
- **Bilah atas di laptop** sekarang dua baris yang tetap (logo, akun, alat di atas; menu di bawah). Sebelumnya isinya terlipat tidak teratur di lebar 1280px, terutama dalam bahasa Indonesia.
- **Pop-up tidak muncul lagi** hanya karena bahasa diganti.
- **Test dan tur di halaman Numbers** menunggu formulir daftar tampil. Sebelumnya skrip kadang memilih customer di kotak saringan register yang berlabel sama. Ini penyebab tur gagal sesekali pada 10 November; aplikasinya sendiri tidak salah.

| Bagian | File |
|---|---|
| Isi | `web/src/lib/help/content.ts`, `en.ts`, `id.ts` |
| Halaman | `web/src/routes/help/+page.svelte` |
| Pemotret | `web/help-shots/help.shots.ts`, `web/playwright.help.config.ts` |
| Gambar | `web/static/help/en/*.png`, `web/static/help/id/*.png` |

## 2. Cara mengulang dari nol

1. Tulis `web/src/lib/help/content.spec.ts` dulu: bab, jumlah langkah, gambar, dan kunci teks layar harus sama di kedua bahasa, dan setiap kunci harus ada di katalog.
2. Tulis `content.ts` (fungsi `parts`), lalu `en.ts` dan `id.ts`.
3. Tulis halaman `/help` dan tautannya di `+layout.svelte`.
4. Tulis `help.shots.ts`, jalankan `npm --prefix web run help:shots`, lalu periksa gambarnya.
5. Tulis `web/e2e/help.test.ts` dan tambahkan `/help` ke `responsive.test.ts`.

## 3. Konsep yang dipelajari

- **Satu sumber untuk nama tombol.** Panduan tidak menulis ulang "Save"; ia menunjuk kunci katalog. Dua salinan teks yang sama cepat berbeda; satu sumber tidak.
- **Test sebagai penjaga isi.** Test memastikan dua bahasa tidak saling tertinggal, tanpa perlu membaca keduanya dengan mata.
- **Snippet Svelte 5.** `{#snippet rich(text)}` dipakai ulang untuk setiap paragraf dan langkah.
- **`display: contents`.** Membuat anak sebuah kotak diatur langsung oleh kotak di atasnya; dipakai supaya menu dan info akun bisa ditaruh di baris berbeda tanpa mengubah HTML.
- **Folder `static`.** File di `web/static/` disalin apa adanya ke hasil build dan ikut masuk ke binary Go.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Gambar di-commit, bukan dibuat saat build | Build biasa tidak perlu browser dan database; gambar diperbarui dengan satu perintah saat layar berubah |
| Tautan Help di bilah atas, bukan di kelompok menu | Terlihat oleh semua akun di setiap halaman, seperti Change password |
| Semua bab tampil untuk semua akun, dengan keterangan "Who" | Pengguna bisa melihat apa yang dikerjakan orang lain sebelum dan sesudah bagiannya |

## 5. Asumsi

- Bab hanya mencakup layar yang sudah dibangun (sampai M3.4 dan tabel AP). Setiap milestone yang menambah layar juga menambah bab dan gambarnya.
- Gambar memakai data contoh dari skrip, bukan data asli.

## 6. Temuan desain baru

Tidak ada.

## 7. Status test (komputer pemilik, Windows, 11 November 2026)

- `npm --prefix web run check`: 0 error. Vitest: 119 test lulus. E2E: 40 test lulus, termasuk `help.test.ts` dan `/help` di tiga lebar layar.
- Tur otomatis dan `help:shots`: lulus.

## 8. Latihan tulis tangan (tanpa AI)

Tambahkan satu tips ke bab "numbers" di `en.ts` dan `id.ts`, memakai satu kunci katalog (misalnya `{numberStatusVoid}`). Jalankan `npm --prefix web test` dan pastikan lulus. Lalu hapus tips itu dari `id.ts` saja dan lihat test mana yang gagal, dan mengapa.
