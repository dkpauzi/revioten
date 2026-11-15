# Tahap 9 · API integrasi (M9.1 sampai M9.5)

- **Tanggal:** 15 November 2026
- **Aturan yang dipakai:** R-API-01 sampai R-API-15, R-API-17, R-SEC-14, D-19, D-37, D-61, D-105, RK-25, RK-26.
- **Persetujuan pemilik:** "lanjutkan M9" (15 November 2026). Syarat masuk tahap ini (M8.1 selesai, D-19) sudah terpenuhi.
- **Sifat:** latihan di localhost (D-37). Konsumen nyata belum ditetapkan (Q-22).
- **Tidak dibangun:** webhook (M9.6, opsional), konsumen tiruan berupa skrip (usulan), CI, OAuth2 atau mTLS. Lihat E-125.

## 1. Apa yang dibuat

| Milestone | Hasil |
|---|---|
| M9.1 Service account dan token | Admin → **Service accounts**: nama, tim pemilik, kontak, tujuan, scope, customer, jaringan. Token `rvt_<awalan>_<256 bit acak>` tampil sekali; yang disimpan hanya awalan dan SHA-256. Masa berlaku bawaan 90 hari, bisa dicabut, dan akun bisa dinonaktifkan. Penolakan token yang dikenal tercatat di audit log dengan alasannya. |
| M9.2 Kontrak terpisah | `api/integration.yaml` dengan kode Go sendiri di `internal/integrationapi`. Contract test membandingkan bentuk setiap jawaban dengan `api/integration.contract.json`, jadi perubahan UI API tidak bisa diam-diam mengubah API integrasi. |
| M9.3 Endpoint baca | `GET /integration/v1/documents` (hanya dokumen terbit, customer token saja, paginasi cursor, `updated_since`) dan `GET /integration/v1/documents/{id}/revisions/{n atau current}` (isi beku revisi beserta kolom format). Customer lain dijawab 404. |
| M9.4 Usulan perubahan | `POST /integration/v1/changes`: operasi add, modify, remove terhadap revisi yang berlaku, wajib `Idempotency-Key`, revisi dasar yang salah 409, `on_behalf_of` hanya dicatat. Hasilnya perubahan berstatus diajukan; isinya baru masuk ke draft saat verifikator menerimanya. Tidak ada jalur API untuk verifikasi, approval, atau rilis. |
| M9.5 Ekspor dan batas laju | `POST /integration/v1/exports` mengantrekan ekspor Excel; dikerjakan di latar belakang, lalu diunduh. Batas laju per token (bawaan 120 per menit) menjawab 429 dengan Retry-After. |

| Bagian | File |
|---|---|
| Migrasi | `00043` (scan migrasi opsional), `00044_integration.sql`, `00045_integration_verifier_check.sql` |
| Aturan | `internal/integration` (token, bacaan, usulan, ekspor), `internal/document/patch.go`, `internal/change/verify.go` (usulan masuk ke draft) |
| Adapter | `internal/integrationapi` (handler, batas laju, contract test), `internal/server/serviceaccounts.go` |
| Kontrak | `api/integration.yaml`, `api/integration.contract.json`, tambahan service account di `api/openapi.yaml` |
| Layar | `web/src/routes/admin/service-accounts/+page.svelte`, bab Help "Other applications" |
| Test | `internal/integration` (token, bacaan lintas customer, usulan, ekspor), `internal/integrationapi` (kontrak, error, batas laju), tabel izin, `web/e2e/m9-integration.test.ts` |

## 2. Cara mengulang dari nol

1. Tulis `TestTokens`, lalu migrasi 00044 dan `integration.go`.
2. Tulis `TestRead` dengan dua customer, lalu query `ListReleasedDocuments` dan `read.go`.
3. Tulis `TestPropose`, lalu `patch.go`, `write.go`, dan langkah usulan di `verify.go`.
4. Tulis `TestExport`, lalu antrean ekspor.
5. Tulis kontrak `integration.yaml`, buat kode dengan `go generate`, tulis `handler.go`, lalu `TestContract` dengan `-update` sekali dan periksa berkas acuannya.
6. Sambungkan ke server di `/integration/`, layar service account, Help, tur, E2E.

## 3. Konsep yang dipelajari

- **Token, bukan password.** Token berisi 256 bit acak, jadi hash SHA-256 biasa cukup; password buatan orang butuh argon2id karena mudah ditebak.
- **Scope dan customer.** Izin aplikasi tidak lewat peran orang, tetapi lewat daftar yang melekat pada token.
- **Idempoten.** Permintaan yang diulang (misalnya karena jaringan putus) tidak boleh membuat dua perubahan. Kunci yang sama dengan isi berbeda ditolak.
- **Optimistic concurrency lewat revisi dasar.** Pemanggil menyebut revisi yang ia baca; bila sudah berganti, jawabannya 409 dan ia harus membaca ulang.
- **Contract test.** Bentuk jawaban dibekukan di berkas acuan. Perubahan yang tidak sengaja langsung ketahuan.
- **Kerja di latar belakang.** Ekspor besar diantrekan, supaya permintaan pengguna lain tidak ikut lambat.
- **Paginasi cursor.** Lebih stabil daripada nomor halaman bila data bertambah di tengah sinkronisasi.

## 4. Keputusan dan asumsi

- D-186: scan hardcopy migrasi opsional. D-187: tombol lihat password di login.
- ASUMSI E-122 (antrean tanpa River), E-123 (batas jaringan di belakang proxy), E-124 (aturan usulan), E-125 (yang tidak dibangun).

## 5. Status test

- 15 November 2026: `go test -p 1 ./...` lulus (termasuk contract test), Vitest 156 lulus, E2E 61 lulus, tur otomatis 33 langkah lulus, skrip gambar Help berhasil.

## 6. Latihan tulis tangan (Gerbang 9)

Tanpa AI, tambahkan satu endpoint baca `GET /integration/v1/customers` yang memberi kode dan nama customer milik token: tulis dulu di `api/integration.yaml`, jalankan `go generate ./internal/integrationapi/`, tambah satu baris di tabel `operations` contract test, jalankan test dengan `-update`, lalu periksa selisih `api/integration.contract.json`.
