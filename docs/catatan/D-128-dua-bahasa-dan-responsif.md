# D-128: layar dua bahasa dan tampilan per perangkat

- **Tanggal:** 8 November 2026
- **Keputusan:** D-128 (brief v1.22), menggantikan D-96. Pernyataan pemilik: "kita akan menggunakan dua bahasa. namun sekarang gunakan bahasa inggris. dan sediakan tombol agar mudah peralihan bahasa dari inggris ke indonesia", dan "buat semua tampilan dengan mempertimbangan semua device". Rencana disetujui di hari yang sama ("Setuju, mulai").
- **Bukan milestone di `milestone.md`.** Lanjutan dari UX-05 (`UX-05-menu-dan-design-system.md`).

## 1. Tujuan dan apa yang dibuat

1. **Dua bahasa layar.**
   - Katalog `web/src/lib/i18n/id.ts`, berisi 715 teks dengan kunci yang sama seperti `en.ts`. Bahasa Inggris tetap bawaan.
   - `web/src/lib/i18n/language.svelte.ts` menyimpan bahasa aktif dalam rune `$state`, jadi layar langsung berganti tanpa memuat ulang. Pilihan disimpan di localStorage browser (kunci `revioten.language`).
   - Tombol EN/ID (`web/src/lib/LanguageSwitch.svelte`) tampil di bar atas semua halaman, termasuk halaman login.
   - Atribut `lang` di `<html>` ikut berganti.
2. **Tampilan per perangkat.**

   | Perangkat | Lebar | Yang berubah |
   |---|---|---|
   | HP | di bawah 640px | Menu dilipat ke tombol Menu; nama aplikasi di bar atas disembunyikan (tetap dibacakan lewat `aria-label`); field formulir selebar layar dan ditumpuk; sel tabel lebih rapat |
   | Tablet | 640 sampai 1023px | Menu dilipat ke tombol Menu; jarak tepi lebih kecil |
   | Laptop | 1024px ke atas | Menu berkelompok langsung terlihat, seperti D-127 |

   Berlaku di semua ukuran:
   - tombol Log out selalu terlihat di bar atas, juga di HP (ASVS 7.4.4);
   - tabel lebar bergulir di dalam kartunya, dan halaman tidak pernah bergeser ke samping;
   - di layar sentuh, tombol dan field paling sedikit 44px.

## 2. Cara mengulang dari nol

1. Tulis test dulu:
   - `web/src/lib/i18n/i18n.spec.ts`: kunci `id.ts` sama dan terisi, tempat isian sama, pengalih bahasa;
   - `web/e2e/language.test.ts`;
   - `web/e2e/responsive.test.ts`.
2. Buat `language.svelte.ts`, lalu ubah `t()` di `index.ts` supaya membaca `language.current`.
3. Salin `en.ts` menjadi `id.ts` dan terjemahkan. Pakai kolom "Istilah" di U1 untuk istilah domain.
4. Buat `LanguageSwitch.svelte` dan pasang di `+layout.svelte`.
5. Tambahkan tombol Menu dan panel di `+layout.svelte`, lalu aturan `@media` di `app.css`.
6. Jalankan `npm --prefix web run check`, `npm --prefix web test`, dan `npm --prefix web run test:e2e`. Periksa juga di browser dengan ukuran 375, 768, dan 1280px.

## 3. Konsep yang dipelajari

- **Reaktivitas lintas modul.** `t()` biasa (bukan komponen) membaca `language.current`, yaitu `$state` dari file `.svelte.ts`. Karena template Svelte memanggil `t()` saat render, Svelte mencatat bacaan itu, dan teks dirender ulang saat bahasa berganti.
- **Teks yang tersimpan tidak ikut berganti.** Pesan seperti "Customer AB sudah ditambahkan" disimpan di variabel saat dibuat. Pesan itu tetap dalam bahasa lama sampai muncul pesan baru. Ini sengaja, supaya kode tetap sederhana.
- **Media query dan `pointer: coarse`.** `max-width` memilih tata letak menurut lebar layar. `pointer: coarse` mengenali layar sentuh, berapa pun lebarnya.
- **Spesifisitas CSS.** Gaya di `<style>` komponen Svelte mendapat kelas unik, jadi menang atas gaya global dengan bobot yang sama. Karena itu aturan HP untuk field memakai `main.page .field`.
- **Disclosure bertingkat.** Tombol Menu (`aria-expanded`, `aria-controls`) membuka panel. Di dalam panel, tombol kelompok membuka daftar tautannya. Escape menutup dari yang paling dalam.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Pilihan bahasa disimpan di browser, bukan di akun | Harus berlaku juga di halaman login, sebelum orang masuk; tanpa perubahan database dan API |
| Nama status nomor (Reserved, In use, Released, Obsolete, Void, Skipped) dan tahap tanda tangan (Prepared, Checked, Approved) tidak diterjemahkan | Begitulah istilahnya di U1 dan di form; tidak membuat istilah baru (CLAUDE.md bagian 3 butir 12) |
| Format tanggal tetap `YYYY-MM-DD HH:mm` di dua bahasa | Bentuk ini tidak bisa salah dibaca dalam bahasa apa pun. Rencana awal menyebut format mengikuti bahasa; diganti karena tidak menambah manfaat |
| Batas tablet sampai 1023px memakai tombol Menu | Lima kelompok menu, nama akun, tombol bahasa, dan Log out tidak muat satu baris di 768px |
| Tanpa bayangan tepi pada tabel yang bergulir | Rencana awal menyebut bayangan tepi. Teknik bayangan latar tertutup latar sel tabel, jadi tidak terlihat. Batang gulir browser tetap menjadi tanda |

## 5. Asumsi

- Terjemahan istilah domain mengikuti kolom "Istilah" di U1. Istilah dengan tanda "Cek WI: Belum", misalnya Jenis perubahan, Sumber perubahan, dan Status nomor, perlu dicocokkan pemilik dengan WI.
- Istilah yang di U1 memang bahasa Inggris tetap bahasa Inggris di layar Indonesia: Customer, Model, Part number, Project group, Format template, Process template, Master data, Audit log, dan Viewer.
- Pesan yang ditulis server Go (misalnya pesan galat yang tidak dipetakan ke katalog) tetap berbahasa Inggris (D-128).

## 6. Temuan desain baru

Tidak ada temuan E baru.

## 7. Status test (komputer pemilik, Windows, 8 November 2026)

- `npm --prefix web run check`: 0 error, 0 peringatan.
- `npm --prefix web test`: 97 test lulus (6 baru untuk bahasa).
- `npm --prefix web run test:e2e`: 35 test lulus (1 bahasa, 3 ukuran layar).
- `go test -short ./...`: lulus.
- Dicek di browser:
  - HP 375px: beranda, panel menu dalam bahasa Indonesia, dan register nomor;
  - tablet 768px: halaman Akun dalam bahasa Indonesia, tanpa geser ke samping.

## 8. Latihan tulis tangan (tanpa AI)

1. Tambahkan satu kunci baru, misalnya `homeHello: 'Welcome'`, hanya ke `en.ts`. Jalankan `npm --prefix web run check`, lalu baca pesan TypeScript-nya. Tambahkan terjemahannya di `id.ts` sampai pemeriksaan bersih, lalu hapus lagi kunci itu.
2. Ubah batas HP dari 640px menjadi 600px di `app.css` dan `+layout.svelte`. Jelaskan dengan kata-kata sendiri kenapa batas itu harus sama di kedua file. Kembalikan setelah selesai.
