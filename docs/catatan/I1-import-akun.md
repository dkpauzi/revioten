# I1 · Import akun dari Excel

- **Tanggal:** 24 November 2026
- **Keputusan:** D-224 (brief v1.53). Keputusan ini menjawab E-143 dengan opsi b.
- **Persetujuan pemilik:**
  - "E-143, oke, saya setuju dengan saran anda".
  - "apakah bisa saya import menggunakan excel? ... ketika reset database, saya tinggal masukkan importnya lagi untuk semua akun yang saya perlukan".
  - "oke saya setuju dengan usulan anda, sekarang mulai tulis code nya".
- **Rencana:** `docs/rencana/import-tur-dan-logo.md`, bagian I1.
- **Aturan yang dipakai:**
  - D-132: daftar password terlarang tetap berlaku untuk password pilihan pengguna.
  - E-08: password sementara wajib diganti.
  - E-18: password tidak masuk audit log.
  - D-206: cakupan All customers.
  - R-MIG-06: Migration mode.
  - R-SEC-14: rahasia tidak ditulis ke log.

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Migrasi 00059 | Kolom `accounts.temporary_password_expires_at`. Kolom ini kosong untuk akun lain. `SetPasswordHash` mengosongkannya saat password diganti, dan `ResetAccountPassword` mengosongkannya saat admin mereset. |
| Layanan | `internal/auth/accountimport.go`. **`AccountImportTemplate`** membuat berkas Excel dengan sheet Accounts (judul kolom saja), Roles (peran yang bisa diberikan), dan Help. **`ImportAccounts`** membaca sheet pertama: Preview tanpa menyimpan, atau Import dalam satu transaksi. Login yang sudah ada dilewati, dan baris yang salah dilaporkan lalu tidak diimpor. Hanya admin (A-1), dan hanya saat Migration mode menyala. |
| Login | Password sementara yang sudah lewat batasnya ditolak dengan pesan gagal login biasa. |
| Email | Aturan email dipindah ke `problem.Email`, supaya halaman akun dan import memakai aturan yang sama. |
| API | `GET /api/v1/accounts/import-template` dan `POST /api/v1/accounts/import?commit=` (multipart, kolom `file`). `AccountInfo` mendapat `temporary_password_expires_at`. |
| Layar | Admin → **Import accounts** (`/admin/accounts/import`): **Download the template**, pilih berkas, **Preview**, lalu **Import**. Laporannya satu baris per baris Excel. Daftar Accounts menampilkan "Temporary password until ..." atau "Temporary password expired". |
| Help | Tip baru di bab Akun dan peran, dua bahasa. |
| Test | `internal/auth/accountimport_test.go`, baris izin di `permissions_test.go`, `accounts.spec.ts`, `menu.spec.ts`, `web/e2e/accounts-import.test.ts` dengan fixture nama fiktif `web/e2e/fixtures/accounts-import.xlsx`, dan halaman baru di `responsive.test.ts`. |

## 2. Format berkas

| Kolom | Isi |
|---|---|
| Name | Wajib. Nama lengkap. |
| Employee number | Opsional, unik per orang. |
| Login | Wajib. Login yang sudah ada dilewati. |
| Email | Opsional. |
| Roles | Nama peran dari sheet Roles, dalam bahasa Inggris atau Indonesia, dipisah koma. Kosong berarti hanya melihat. |
| Scope | `All customers` (atau `Semua customer`), atau kosong. Kosong membatasi Project Leader ke Project Group sendiri. Peran lain selalu berlaku untuk semua customer. |

Urutan kolom bebas, asal judulnya sama dengan template. Batasnya 5 MB dan 500 akun per berkas.

## 3. Cara mengulang dari nol

1. Tulis `TestImportAccounts` dan `TestAccountImportTemplate`, lalu pastikan keduanya gagal.
2. Buat migrasi 00059, ubah query di `accounts.sql` dan `auth.sql`, lalu jalankan `go generate ./internal/db/...`.
3. Tambah pemeriksaan batas password di `Login`, buat `accountimport.go`, dan pindahkan aturan email ke `problem.Email`.
4. Tulis kontrak OpenAPI, jalankan `go generate ./internal/server/...`, buat `internal/server/accountimport.go`, lalu tambah baris izin.
5. Jalankan `npm --prefix web run gen:api`, lalu buat `importAccounts` di klien, halaman import, entri menu, status akun, teks dua bahasa, dan Help.
6. Buat fixture Excel, tulis test E2E, dan tambah halaman baru ke `responsive.test.ts`.

## 4. Konsep yang dipelajari

- **Laporan dulu, simpan kemudian.** Preview dan Import menjalankan pemeriksaan yang sama. Bedanya hanya pada transaksi di akhir, jadi yang Anda lihat di Preview sama dengan yang disimpan.
- **Idempoten lewat kunci alami.** Login adalah kunci akun. Baris dengan login yang sudah ada dilewati, sehingga berkas yang sama aman diunggah berkali-kali.
- **Salt per akun.** Semua akun memakai password yang sama, tetapi setiap hash argon2id punya salt sendiri, sehingga hash di database tidak sama. Hash dihitung sebelum transaksi, karena argon2id sengaja lambat.
- **Pesan gagal yang sama.** Password kedaluwarsa ditolak seperti password salah, supaya orang luar tidak bisa mengetahui login mana yang ada.

## 5. Asumsi

- **Preview juga butuh Migration mode,** supaya tombol dan pesannya sederhana.
- **Baris yang salah tidak menghentikan import.** Baris yang benar tetap disimpan, seperti pada import nomor lama (M2.5). Perbaiki berkasnya, lalu unggah lagi; akun yang sudah dibuat akan dilewati.
- **Hanya akun kerja.** Akun admin tetap dibuat satu per satu atau lewat `create-admin`.
- **Login yang sudah ada tidak diperiksa ulang.** Nama, peran, dan email akun lama tidak dibandingkan dan tidak diubah.
- **Reset oleh admin** memberi password sementara acak tanpa batas 7 hari, seperti sebelumnya.

## 6. Temuan desain baru

Tidak ada.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `go vet ./...`, `gofmt -l .` | Bersih |
| `npm --prefix web run check` | 0 error |
| Vitest | 193 lulus |
| `npm --prefix web run test:e2e` | 75 lulus |
| help-shots | Lulus, gambar diperbarui |

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `splitRoles(text string) []string` yang memecah "Project Leader, PPAP Engineer ,," menjadi `["project leader", "ppap engineer"]`: dipisah koma, spasi di tepi dibuang, huruf kecil, dan isian kosong dilewati. Bandingkan dengan perulangan di `importRoles` (`internal/auth/accountimport.go`).
