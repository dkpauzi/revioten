# D-223 · Ganti logo sebagai revisi dokumen

- **Tanggal:** 23 November 2026
- **Keputusan:** D-223 (brief v1.52). Keputusan ini menjawab E-145 dengan opsi a dan menggantikan butir 1 D-221 (logo cetakan diambil saat dicetak).
- **Persetujuan pemilik:** "pergantian logo itu seperti melakukan revisi dokumen, jadi ketika ada pergantian logo, harus ada pertanyaan apakah merevisi dokumen atau tidak, jika iya, maka harus minta tanda tangan dan masuk history revisi", lalu "Ya, opsi a".
- **Aturan yang dipakai:** R-REV-02 (revisi terbit tidak bisa diubah, ada snapshot dan SHA-256), R-CHG-01 (perubahan menunjuk dokumen terdampak), M5.10 (perubahan surut SQAM sebagai pola), E-3 (hak mengajukan perubahan), A-3 (pengaturan admin), dan R-SEC-14 (isi gambar tidak masuk log).

## 1. Masalah

Sejak D-220, logo cetakan diambil saat dicetak. Akibatnya cetak ulang revisi lama ikut memakai logo baru, padahal revisi terbit tidak boleh berubah (R-REV-02). Pemilik juga ingin penggantian logo melewati tanda tangan dan tercatat di riwayat revisi.

## 2. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Migrasi 00058 | Tabel `company_logos` menyimpan setiap logo yang pernah dipakai, dengan SHA-256 sebagai kunci; logo bawaan tetap di binary. Tabel `logo_changes` mencatat setiap penggantian beserta jawaban admin (`revise`). `changes` mendapat mekanisme `logo`, kolom `logo_change_id`, dan unique index: satu perubahan logo per customer per penggantian. Role aplikasi hanya punya SELECT dan INSERT pada kedua tabel baru. |
| Snapshot | `Snapshot.LogoSHA256` (`logo_sha256`) diisi saat rilis dari logo yang berlaku. Cetakan revisi terbit memakai logo itu (`internal/printout/service.go`); draft memakai logo yang berlaku sekarang. Revisi yang terbit sebelum D-223 tidak punya hash dan memakai logo yang berlaku. |
| Pengaturan | `SetLogo` dan `ResetLogo` menerima `revise`. Logo baru masuk ke `company_logos`, penggantian dicatat di `logo_changes`, dan audit log hanya memuat hash serta `revise_documents`. |
| Perubahan logo | `internal/change/logo.go`: `PendingLogoChange` memberi penggantian terakhir yang dijawab "ya" selama customer belum mengajukannya dan masih punya dokumen aktif. `SubmitLogoChange` membuat satu perubahan untuk semua dokumen aktif customer. Sesudah itu verifikasi, tanda tangan, dan rilis berjalan seperti biasa, dan alasannya masuk ke riwayat revisi. |
| API | `PUT` dan `DELETE /api/v1/company/logo?revise=true|false` (wajib), `GET /api/v1/customers/{id}/pending-logo-change`, `POST /api/v1/logo-changes`. |
| Layar | Admin → Settings: pertanyaan wajib **Revise the documents because of this logo?** sebelum **Upload the logo** atau **Use the default logo**. Documents → Changes: bagian **Logo change** tampil untuk customer yang menunggu, dengan alasan bawaan "Change company logo" dan tombol **Submit the logo change**. |
| Help | Bab Changes dan tips Printing, dua bahasa. |
| Test | `internal/change/logo_test.go`, `TestLogoFrozenAtRelease` di `internal/printout/service_test.go`, `company_test.go`, baris izin di `permissions_test.go`, `web/e2e/company-logo.test.ts`, dan test D-223 di `web/e2e/m5-changes.test.ts`. |

## 3. Cara mengulang dari nol

1. Tulis `TestLogoChangeAsRevision` dan `TestLogoFrozenAtRelease`. Keduanya gagal dulu.
2. Migrasi 00058, query di `sql/queries/settings.sql` dan `change_requests.sql`, lalu `go generate ./...`.
3. `settings.SetLogo` dan `ResetLogo` dengan `revise`, `CurrentLogoSHA`, dan `LogoBySHA`.
4. `Snapshot.LogoSHA256` di `buildSnapshot`, lalu pilihan logo di `printout.input`.
5. `internal/change/logo.go`, kontrak OpenAPI, handler di `internal/server/company.go`, dan baris izin.
6. `npm --prefix web run gen:api`, fungsi klien, `CompanyLogo.svelte`, `LogoChangeForm.svelte`, teks dua bahasa, Help, dan test E2E.

## 4. Konsep yang dipelajari

- **Content-addressed storage.** Gambar disimpan menurut SHA-256 isinya. Snapshot cukup menyimpan hash, dan gambar yang sama tidak disimpan dua kali (`ON CONFLICT DO NOTHING`).
- **Membekukan rujukan, bukan salinan.** Snapshot tidak memuat gambar; ia memuat hash. Karena tabel `company_logos` hanya bisa ditambah, rujukan itu selalu bisa dibaca lagi.
- **Pekerjaan yang menunggu dihitung, bukan disimpan.** Tidak ada baris "tugas" per customer. Yang menunggu dihitung dari penggantian terakhir, perubahan yang sudah diajukan, dan dokumen aktif. Penggantian yang sudah diganti lagi tidak perlu direvisi.

## 5. Asumsi

- **Hanya penggantian terakhir yang menunggu.** Bila logo diganti dua kali sebelum customer mengajukan perubahan, cukup satu perubahan untuk logo terakhir.
- **Semua dokumen aktif customer** masuk ke perubahan logo, seperti perubahan surut SQAM. Dokumen yang tidak perlu revisi bisa dilepas dari perubahan selama masih diajukan.
- **Jawaban "tidak"** tetap menyimpan logo di riwayat. Draft dan revisi berikutnya memakai logo baru, sedangkan revisi terbit tetap memakai logonya sendiri.
- **Revisi sebelum D-223** tidak punya hash logo dan dicetak dengan logo yang berlaku. Snapshot lama tidak diubah (R-REV-02).
- **Laporan kinerja** menghitung perubahan logo di kelompok "lain", karena tidak punya sumber perubahan.

## 6. Temuan desain baru

Tidak ada. E-145 diputuskan menjadi D-223.

## 7. Status test

| Test | Hasil |
|---|---|
| `go test -p 2 ./...` | Semua paket lulus |
| `go vet ./...`, `gofmt -l .` | Bersih |
| `npm --prefix web run check` | 0 error |
| Vitest | 192 lulus |
| `npm --prefix web run test:e2e` | 74 lulus |
| help-shots | Lulus, gambar diperbarui |

## 8. Latihan tulis tangan

Tanpa AI, tulis query SQL yang menjawab: customer mana saja yang masih menunggu perubahan logo? Petunjuk: ambil baris terakhir `logo_changes`; bila `revise` benar, pilih customer yang punya dokumen aktif dan belum punya baris `changes` dengan `logo_change_id` itu. Bandingkan hasilnya dengan `pendingLogo` di `internal/change/logo.go`.
