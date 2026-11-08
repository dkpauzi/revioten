# UX-05 (sebagian): design system dan menu berkelompok

- **Tanggal:** 8 November 2026
- **Keputusan:** D-127 (brief v1.22). Pemilik menerima UX-05 untuk halaman yang sudah ada ("Terima, buat rencana"), lalu menyetujui rencana beserta nama kelompok menunya ("Setuju, mulai").
- **Sumber desain:** kanvas desain UI "Desain UI Sistem PFD PFMEA CP", artboard "Design system" dan "v2 · Home (simple)".
- **Bukan milestone di `milestone.md`.** Ini langkah ke-3 dari urutan yang disarankan di `Tinjauan-desain-UI-UML-0.12.md` bagian 4. Didahulukan atas permintaan pemilik; langkah 1 dan 2 (perluasan impor nomor, master simbol) tidak bergantung padanya.

## 1. Tujuan dan apa yang dibuat

Tampilan aplikasi sebelumnya memakai gaya bawaan browser dan 15 tautan menu datar. Sekarang:

1. **Design system di satu file**, `web/src/app.css`:
   - warna sebagai variabel CSS di `:root` (`--ink`, `--accent`, `--success`, dan seterusnya), diambil dari artboard;
   - huruf Geist dan Geist Mono, dibundel lewat paket `@fontsource-variable` (143 KB woff2, tidak diambil dari internet);
   - teks dasar 15px;
   - gaya dasar tombol, field, pesan error, tabel, kartu, dan label status (`.badge`, warna selalu disertai kata).
2. **Menu atas berkelompok** (`web/src/lib/menu.ts`, dipakai `web/src/routes/+layout.svelte`):

   | Kelompok | Isi |
   |---|---|
   | Home | Beranda |
   | Documents | Numbers, Import legacy numbers (hanya E-8) |
   | Template library | Format templates, Process templates, Processes |
   | Master data | Customers, Models, Document types, Change types, Positions, Approval routes |
   | Admin | Accounts (A-1), Audit log (A-4), Settings (A-3); kelompok hilang bila tidak ada hak admin |

   Kelompok dibuka dengan klik atau Enter, ditutup dengan Escape (fokus kembali ke tombolnya), klik di luar, atau pindah halaman.
3. **Beranda** berisi ubin per kelompok menu, hanya ke halaman yang sudah dibangun.
4. **Register nomor** menampilkan status sebagai label berwarna, dan nomor serta kode customer dalam Geist Mono.
5. **Gaya lokal dibersihkan** di 29 file `.svelte`: aturan tabel, `.error`, `.hint`, dan warna dipindah ke `app.css`. Yang tersisa di komponen hanya susunan (lebar field, jarak antarbagian).

Fungsi halaman tidak berubah: tidak ada tombol, field, endpoint, atau aturan baru.

## 2. Cara mengulang dari nol

1. Cek versi paket huruf: `npm view @fontsource-variable/geist dist-tags time --json` dan `npm view @fontsource-variable/geist@5.3.0 peerDependencies license`.
2. Pasang dengan versi tepat: `npm --prefix web install --save-dev --save-exact @fontsource-variable/geist@5.3.0 @fontsource-variable/geist-mono@5.3.0`.
3. Tulis test dulu: `web/src/lib/menu.spec.ts` (Vitest) dan `web/e2e/menu.test.ts` (E2E). Jalankan `npm --prefix web test`; test baru harus gagal karena `menu.ts` belum ada.
4. Tulis `web/src/lib/menu.ts` sampai test lulus.
5. Tulis `web/src/app.css`, lalu impor huruf dan file itu di `+layout.svelte`.
6. Ganti menu di `+layout.svelte`, lalu hapus aturan gaya lokal yang bertabrakan.
7. Ganti klik tautan menu di test E2E lama dengan `openFromMenu(page, kelompok, halaman)`.
8. `npm --prefix web run check`, `npm --prefix web test`, `npm --prefix web run test:e2e`, lalu `npm --prefix web run build` dan `go build -o bin/revioten.exe ./cmd/revioten`.

## 3. Konsep yang dipelajari

- **Variabel CSS (custom properties).** Warna ditulis sekali di `:root` (`--accent: #4338ca`) lalu dipakai dengan `var(--accent)`. Mengganti warna aksen cukup di satu baris.
- **Gaya global dan gaya ber-scope di Svelte.** `<style>` di file `.svelte` hanya berlaku untuk komponen itu, karena Svelte menambah kelas unik ke selektornya. Akibatnya gaya komponen menang atas gaya global untuk elemen yang sama. Karena itu aturan tabel lama harus dihapus, bukan hanya ditimpa.
- **Pola disclosure untuk menu.** Tombol dengan `aria-expanded` dan `aria-controls`, daftar tautan yang hanya ada di halaman saat terbuka. Pembaca layar tahu kelompok terbuka atau tertutup, dan keyboard bekerja tanpa kode tambahan selain Escape.
- **Warna tidak pernah sendirian.** Label status memakai warna dan kata, sehingga tetap terbaca oleh orang yang sulit membedakan warna dan di cetakan hitam putih.
- **Menyembunyikan menu bukan pengaman.** `buildMenu` hanya tidak menawarkan halaman yang pasti ditolak; server tetap memeriksa setiap operasi (M1.2).

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Huruf dari `@fontsource-variable`, bukan Google Fonts | Aplikasi jalan di jaringan internal tanpa internet; artboard menyebut huruf "bundled with the app" |
| Satu file `app.css`, tanpa pustaka CSS (Tailwind dan sejenisnya) | Lebih sedikit alat yang harus dipelajari; pemilik ingin bisa membangun ulang tanpa AI (D-12) |
| Tombol kirim di dalam formulir otomatis jadi tombol utama | "Satu aksi utama per area" (artboard) tanpa mengubah markup 30 halaman |
| Bagian `<section>` langsung di bawah halaman tampil sebagai kartu, dan tabel lebar bergulir di dalam kartunya | Sesuai artboard, dan halaman tidak bergulir ke samping di layar HP |
| Ubin beranda dibentuk dari menu, tanpa teks baru | Tidak membuat istilah atau janji fitur yang belum ada (CLAUDE.md bagian 3 butir 8 dan 12) |

## 5. Asumsi

- Ukuran teks 15px dan warna latar `#F4F5F7` diambil dari artboard "v2 · Home (simple)", yang disebut kanvas sebagai versi yang disarankan. Artboard "Design system" memakai 13px; yang dipilih adalah versi v2 karena UX-05 meminta teks lebih besar.
- Kelompok Admin memakai nama "Admin", sama dengan istilah peran admin di U1. Disetujui pemilik bersama rencana.
- Tidak ada mode gelap, karena artboard tidak menggambarnya.

## 6. Temuan desain baru

Tidak ada temuan E baru. Yang masih terbuka dari UX-05: layar workbook (Tahap 4) dan pilihan bahasa EN/ID (mengubah D-96).

**Perbaikan test E2E (bukan temuan desain).** Ini kali pertama test E2E dijalankan di komputer pemilik (Windows). Dua test lama gagal, juga pada kode sebelum perubahan ini (dibuktikan dengan menjalankan `accounts.test.ts` pada kode lama):

- `accounts.test.ts`: setelah Log out, test langsung mengisi "Login name". Halaman Accounts juga punya field berlabel "Login name", jadi isian jatuh ke formulir akun yang belum hilang, dan login terkirim dengan nama kosong.
- `login.test.ts`: setelah Log out, test langsung membuka `/login` sementara aplikasi masih pindah ke `/login` sendiri. Dua perpindahan bertabrakan, test berhenti setelah password admin uji diganti, dan semua test sesudahnya gagal login.

Perbaikannya helper `logOut(page)` di `web/e2e/helpers.ts`: tekan Log out, lalu tunggu sampai alamatnya `/login`. Semua test yang logout memakainya. Kode aplikasi tidak diubah untuk ini.

## 7. Status test (komputer pemilik, Windows, 8 November 2026)

- `npm --prefix web run check`: 0 error, 0 peringatan.
- `npm --prefix web test`: 91 test lulus (7 baru di `menu.spec.ts`).
- `npm --prefix web run test:e2e`: 31 test lulus (1 baru di `menu.test.ts`), dua kali berturut-turut. Chromium Playwright dipasang dengan `npm --prefix web exec -- playwright install chromium`.
- `go vet ./...` dan `go test ./...` (dengan Docker): bersih, 16 paket.
- Tampilan dicek di browser: login, beranda, menu terbuka, Document types, dan Format templates di lebar layar HP (375px, tanpa geser ke samping).
- Ukuran hasil build tampilan web: 458 KB, termasuk huruf 143 KB.

## 8. Latihan tulis tangan (tanpa AI)

1. Di `web/src/app.css`, ganti nilai `--accent` menjadi hijau tua (misalnya `#05603a`), build ulang, dan lihat bagian mana saja yang berubah. Jelaskan dengan kata-kata sendiri kenapa cukup satu baris.
2. Tambahkan halaman `/about` ke kelompok Admin di `menu.ts` hanya untuk pemegang A-3. Tulis dulu satu kasus test di `menu.spec.ts` yang gagal, baru ubah `menu.ts` sampai lulus. Kembalikan perubahannya setelah selesai.
