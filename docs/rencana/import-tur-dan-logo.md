# Rencana · Logo, efek tombol, kamera tur, dan menu import

- **Status:** usulan, menunggu persetujuan pemilik (22 November 2026). Belum ada yang dibangun.
- **Asal:** permintaan pemilik setelah bagian A (D-218): logo SIIX yang bisa diganti admin, efek hover/focus/klik supaya tur mudah diikuti, layar tur mengikuti bagian yang sedang diisi, dan import untuk akun, proses, isi template CP/PFMEA, serta nomor dokumen yang sudah dipakai.
- **Urutan yang diusulkan:** L (logo) bersama bagian C, lalu E (efek) dan T (tur) dalam satu sesi, lalu I1 sampai I4 (import), satu per sesi. Bagian B (simbol alur) dan C (cover dan header) tetap seperti rencana sebelumnya.

## L · Logo perusahaan diatur admin

- **Status:** dibangun bersama bagian C (D-220, 23 November 2026). Pertanyaan 2 dijawab pemilik: logo yang berlaku saat dicetak, juga untuk cetak ulang revisi lama (D-221).

Logo dipakai di cover dan kepala lembar cetakan (bagian C), jadi dibangun bersama bagian C.

- **Logo bawaan:** gambar asli dari Excel referensi (`xl/media/image1.png`, sama di SH, SF, dan SC TR-020), disimpan di `docs/referensi/logo-siix.png` (277 × 132 piksel). Logo tidak digambar ulang, karena gambar ulang tidak akan sama persis dengan logo resmi. Bila butuh resolusi lebih tinggi, admin cukup mengunggah file PNG resmi.
- **Penyimpanan:** satu baris di tabel pengaturan (`system_settings`): isi PNG (bytea), SHA-256, waktu, dan pengunggah. Logo bukan lampiran dokumen, jadi tidak disimpan di `ATTACHMENT_DIR`.
- **Layar:** Admin → Settings → bagian **Company logo**: pratinjau logo sekarang, pilih file, **Upload the logo**, dan **Use the default logo**.
- **Aturan:**
  - hanya PNG (diperiksa dari tanda tangan file, bukan nama), paling besar 1 MB, paling lebar 2000 piksel;
  - hanya akun admin (hak A-); tercatat di audit log tanpa isi gambar.
- **API:** `GET /api/v1/settings/logo` (semua akun yang login, untuk pratinjau), `PUT /api/v1/settings/logo` (multipart, admin), `DELETE` (kembali ke bawaan).
- **Cetakan:** PDF dan pratinjau memakai `<img>` dari data PNG; Excel memakai `AddPictureFromBytes`. Revisi terbit tetap mencetak logo yang berlaku saat dicetak. Logo tidak ikut dibekukan di snapshot, karena logo bukan isi dokumen.
- **Pertanyaan untuk pemilik:** logo yang sudah diganti, perlu ikut ke cetakan ulang revisi lama, atau revisi lama tetap memakai logo saat rilis? Usulan: logo saat dicetak.

## E · Efek hover, focus, dan klik

- **Status:** dibangun (D-221, 23 November 2026).

Sebagian sudah ada di `web/src/app.css`: tombol berganti warna saat hover, dan garis fokus `:focus-visible`. Yang ditambahkan di file yang sama (D-127, tanpa `<style>` per halaman):

| Keadaan | Tombol dan tautan tombol | Kotak isian, pilihan, centang | Baris tabel dan menu |
|---|---|---|---|
| Hover | sudah ada (latar lebih gelap) | garis tepi berwarna aksen | latar baris tipis; item menu bergaris bawah |
| Focus | garis aksen 2 px, juga saat diklik mouse (`:focus`, bukan hanya `:focus-visible`) | garis tepi aksen dan cincin lembut | item menu yang terbuka diberi garis aksen |
| Klik (active) | sedikit tertekan: warna lebih gelap dan turun 1 px | — | — |
| Terpilih | — | centang dan radio berwarna aksen (`accent-color`) | halaman aktif di menu (`aria-current="page"`) diberi latar aksen tipis |

- Animasi singkat (sekitar 120 ms), dan dimatikan bila pengguna memilih `prefers-reduced-motion`.
- Warna mengikuti token yang ada (`--accent`, `--accent-ink`, `--surface-2`), sehingga mode terang dan gelap ikut.
- Test: satu E2E memeriksa gaya fokus dan terpilih lewat `getComputedStyle`; `responsive.test.ts` tetap lulus.

## T · Layar tur mengikuti bagian yang sedang dikerjakan

- **Status:** dibangun (D-221, 23 November 2026). Method Locator dibungkus sekali di `web/tour/act.ts`, jadi langkah tur tidak perlu diubah satu per satu.

Playwright memang menggulir elemen ke layar sebelum mengklik, tetapi hanya sampai elemen terlihat, sering di tepi bawah layar dan tertutup pita keterangan. Usulan:

1. **Fungsi pembantu tur** (`web/tour/act.ts`): `click`, `fill`, `select`, dan `check` yang lebih dulu:
   - menggulir elemen ke tengah layar (`scrollIntoView({ block: 'center', behavior: 'smooth' })`);
   - memberi sorotan sementara (garis tebal berwarna di sekeliling elemen, dibuat oleh tur seperti pita keterangan, jadi aplikasi tidak berubah);
   - menunggu sebentar menurut `TOUR_SPEED`, lalu baru mengerjakan aksinya.
2. Semua langkah di `alur.tour.ts` memakai fungsi itu, bukan `page.click` dan sejenisnya langsung.
3. Pita keterangan diberi ruang (`scroll-margin`), supaya elemen yang disorot tidak tertutup.
4. Dengan `TOUR_SPEED=fast`, sorotan dan gulir halus dilewati, jadi tur tetap cepat untuk pengecekan.

Bagian E membuat keadaan tombol terlihat di aplikasi; bagian T membuat tur menunjuk elemen yang dikerjakan. Keduanya bisa dikerjakan dalam satu sesi.

## I · Menu import

Tujuannya: pemilik bisa mulai dari database bersih, lalu mengisi data dasar sekaligus untuk pengecekan manual.

### Pola yang sama untuk semua import

1. **Download the import template** (Excel) dengan kolom yang benar dan satu contoh baris.
2. Pilih file, tekan **Preview**: semua baris diperiksa tanpa menyimpan apa pun. Hasilnya berupa tabel per baris (akan dibuat, sudah ada, ditolak beserta alasannya).
3. **Import** menyimpan semuanya dalam satu transaksi: semua masuk, atau tidak ada yang masuk.
4. Satu entri audit log per import (jumlah baris dan nama file), ditambah entri biasa per data yang dibuat.
5. Hak sama dengan layar manualnya. Teks dua bahasa, Help, dan test (Go, Vitest, E2E) seperti fitur lain.

### I1 · Akun (Admin → Accounts → Import accounts)

- **Kolom:** Name, Employee number, Login, Email, Roles (dipisah koma), Scope (All customers / Own project group).
- Login yang sudah ada dilewati dan dilaporkan, tidak diubah.
- **Password:** permintaan pemilik adalah `password123` untuk semua akun hasil import, dan setiap akun wajib mengganti password saat login pertama.
- **Temuan E-143 (perlu keputusan):** permintaan ini berbenturan dengan aturan password D-132:
  - `password123` ada di daftar password terlarang;
  - satu password yang sama dan mudah ditebak untuk semua akun berarti siapa pun yang tahu login seseorang bisa masuk lebih dulu dan memilih passwordnya sendiri.

  Opsi:
  - **(a)** `password123` persis seperti diminta: daftar terlarang hanya berlaku untuk password yang dipilih pengguna, bukan password sementara dari import.
  - **(b)** (a), tetapi hanya bisa dipakai saat **Migration mode** menyala, dan password sementara kedaluwarsa dalam 7 hari bila tidak dipakai.
  - **(c)** password sementara acak per akun, ditampilkan sekali dan bisa diunduh sebagai Excel (pola yang sekarang dipakai satu per satu).

  Usulan Claude Code: (b), karena sesuai permintaan untuk database uji tetapi tidak terbawa ke pemakaian sungguhan.

  **Diputuskan 24 November 2026: opsi (b), D-224.**

### I2 · Nama proses (Template library → Processes → Import processes)

- **Kolom:** Process name, Aliases (dipisah titik koma).
- Nama yang sudah ada (juga sebagai alias proses lain) ditolak per baris. Alias baru untuk proses yang sudah ada tidak ditambahkan diam-diam; barisnya dilaporkan.

### I3 · Isi template PFMEA dan Control Plan (Template library → Process templates → Import content)

- **Satu file, dua sheet:** PFMEA dan Control Plan. Judul kolomnya mengikuti label kolom format **PFMEA v1** dan **Control Plan v1** (kolom yang punya key, D-146), ditambah kolom Process, Sub-process, Point text, dan Use point for (General / Customer / Model group / Model, dengan kode customer dan nama model).
- **Hasil:**
  - untuk setiap proses, poin masuk ke **versi draft** process template-nya (dibuat bila belum ada);
  - versi terbit tidak pernah diubah (M3.4), dan penerbitan tetap lewat tombol **Publish version N**.
- Proses yang belum terdaftar ditolak per baris (import I2 dulu).
- **Pertanyaan untuk pemilik:** perlu juga membaca langsung Excel bentuk Tokairika (SF-TR-020, SC-TR-020), atau cukup template import sendiri? Membaca bentuk Tokairika lebih nyaman, tetapi letak kolom dan sel gabungannya berbeda per customer, jadi lebih rapuh. Usulan: template import sendiri dulu; bentuk Tokairika menyusul bila diperlukan.

### I4 · Nomor dokumen yang sudah dipakai (Documents → Numbers → Import legacy numbers)

Sebagian besar sudah ada:
- impor nomor tiga kolom, satu set per baris (D-216);
- status nomor bisa dipilih;
- menu **Legacy documents** memakai nomor yang sudah terdaftar dan meminta revisi saat dokumen dimigrasi (D-215, D-217).

Jadi revisi memang belum diisi saat import nomor, dan baru diisi di menu migrasi, seperti yang pemilik minta. Yang ditambahkan:
- **File Excel** sebagai pengganti tempel teks: kolom SH, SF, SC, Model, dan Note. Model hanya dicatat untuk mengisi pilihan model di menu migrasi; dokumennya sendiri tetap dibuat di menu migrasi.
- **Status bawaan** untuk import ini: **Released**, dengan tanda "menunggu migrasi" di papan menu Legacy documents.
- **Pertanyaan untuk pemilik:** "nomor document yang sudah digunakan" artinya nomor yang dokumennya sudah terbit di kertas (Released), atau juga nomor yang sudah dipesan tetapi dokumennya belum jadi (Reserved)? Usulan: pilihan status tetap ada, bawaannya Released.

## Ringkasan pertanyaan

1. E-143: cara password akun hasil import, opsi (a), (b), atau (c). Dijawab: (b), D-224.
2. ~~Logo: cetakan ulang revisi lama memakai logo saat dicetak, atau logo saat rilis?~~ Dijawab: saat dicetak (D-221), lalu diganti D-223: logo dibekukan saat rilis.
3. I3: template import sendiri saja dulu, atau juga membaca Excel bentuk Tokairika?
4. I4: nomor yang sudah dipakai berarti Released saja, atau juga Reserved?
