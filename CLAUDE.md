# CLAUDE.md · Sistem PFD, PFMEA, dan CP

File ini dibaca Claude Code di awal setiap sesi. Isinya aturan kerja untuk menulis kode, diringkas dari brief desain. Bila file ini berbeda dengan brief, brief yang berlaku (brief bagian 0 dan bagian 15, butir 12).

- Dibuat 27 Oktober 2026 dari brief desain v1.15 (D-77).
- Baris bertanda **(usulan)** adalah usulan Claude yang belum diputuskan pemilik. Ikuti sebagai rencana kerja, tapi bisa berubah.

## 1. Proyek ini

- Sistem web self-hosted untuk menyusun, merevisi, menomori, mencetak, dan mencatat rilis PFD, PFMEA, dan Control Plan (CP) di pabrik EMS. Persetujuan tetap tanda tangan basah di kertas (D-07).
- Ini proyek belajar pribadi pemilik, bukan pengembangan perusahaan. Perusahaan menerima spesifikasi dan UML, dan tim IT membangun versinya sendiri dari UML (D-33). Kode tidak diserahkan ke IT (R-SEC-05).
- Aplikasi berjalan di localhost sebagai demo untuk mematangkan konsep dan alur kerja (D-36). Semua kode adalah latihan, termasuk tahap 9 dan 10 (D-37).
- Kode dibangun sejajar dengan UML yang tersisa dan dipakai untuk mengevaluasi desain (D-77). Lihat bagian 6.
- API-first: layar web adalah klien dari API `/api/v1` yang dijelaskan kontrak OpenAPI 3.0.4 (D-61, R-API-02, D-38).
- Pemilik ingin bisa membangun ulang sistem ini sendiri tanpa AI (D-12). Kejelasan lebih penting daripada kecepatan.

## 2. Dokumen sumber

Urutan prioritas bila isinya berbeda (brief bagian 0):

1. Pernyataan eksplisit terbaru dari pemilik.
2. Brief desain: `docs/brief/pfmea-system-design-brief.md` (bahasa Inggris).
3. Milestone: `docs/milestone/milestone.md`. File ini ekspor dari docx di folder yang sama, jadi jangan diedit.
4. Paket UML: `docs/uml/` (README, U1 glosarium, U2 konteks C4, U3 use case dan peta pengguna).

Membaca status di brief:
- [DECIDED] mengikat.
- [PROPOSED] adalah rencana kerja yang bisa berubah.
- [OPEN] belum terjawab (brief bagian 12). Pakai asumsi kerja yang tertulis di sana.
- [DEFERRED] jangan dibangun.
- [SUPERSEDED] abaikan.

Di milestone, "Keputusan:" berarti sudah diputuskan pemilik. "Usulan:" dan "Tambah (usulan):" berarti belum.

Baca bagian yang disentuh milestone yang sedang dikerjakan. Tidak perlu membaca seluruh brief setiap sesi.

## 3. Cara kerja (brief bagian 15)

1. Balas pemilik dalam bahasa Indonesia. Komentar kode dalam bahasa Indonesia. Nama identifier dalam bahasa Inggris **(usulan)**.
2. Mode mengajar: jelaskan konsep baru sebelum menulis kode, dan buat kode sederhana yang mudah dibaca. Setiap milestone ditutup dengan ringkasan, langkah yang bisa diulang, satu latihan tulis tangan tanpa AI, dan catatan keputusan.
3. Test dulu, lalu kode sampai test lulus, lalu rapikan. Termasuk test konkurensi untuk penomoran dan test akses lintas customer saat milestone-nya tiba.
4. Satu milestone per sesi. Sebelum menulis kode, sampaikan rencana singkat dan tunggu persetujuan pemilik. Setelah milestone selesai, berhenti. Jangan lanjut ke milestone berikutnya tanpa persetujuan.
5. Jangan membangun item [DEFERRED], dan jangan mendahului urutan milestone. Tanya dulu.
6. Jangan mengarang data: baseline, ukuran, versi, atau kesimpulan hukum. Verifikasi, atau tandai belum diverifikasi.
7. Bila permintaan bertentangan dengan item [DECIDED], katakan dulu sebelum bertindak. Ajukan satu pertanyaan terfokus setiap kali.
8. Bila instruksi pemilik kurang jelas atau tidak lengkap, tanyakan bagian mana yang dimaksud. Jangan mengisi celah dengan tambahan karangan. Tambahan di luar permintaan harus ditandai jelas sebagai usulan.
9. Saat melanjutkan pekerjaan, baca catatan milestone terakhir di `docs/catatan/` dan lanjutkan dari sana. Jangan mengulang langkah yang sudah selesai.
10. Sebut ID keputusan dan aturan (D-, R-, UC-, M) di catatan milestone dan pesan commit.
11. Nyatakan asumsi secara eksplisit dan catat di catatan milestone.
12. Istilah di layar dan cetakan mengikuti glosarium U1 dan istilah di WI atau prosedur yang sudah ada. Jangan membuat istilah baru. Bahasa UI belum diputuskan (Q-14), jadi tanyakan bila perlu.

## 4. Stack dan versi (brief bagian 7)

Versi di bawah dicek ulang di M0.1 (27 dan 28 Oktober 2026). Daftar lengkap beserta tanggal rilis dan sumbernya ada di `docs/versions.md`, dan file itu yang dipakai bila berbeda dengan daftar ini. Aturannya: pakai versi stabil atau LTS terbaru (keputusan pemilik E-13).

- **Backend:** Go 1.27.3 dengan `net/http`, pgx v5.11.0, sqlc v1.31.1, goose v3.29.0, River v0.49.0 (job queue di PostgreSQL; paket 0.x, ikut masa tunggu 14 hari E-14), excelize v2.11.0, testcontainers-go v0.44.0, golang.org/x/crypto v0.58.0 (argon2id). Tanpa ORM dan tanpa Redis. Kode hasil generate tidak diedit dengan tangan.
- **Database:** PostgreSQL 18.6 di Docker. PostgreSQL 19 masih RC, jadi tidak dipakai. Docker tetap dipakai. Bila Docker Desktop terkendala lisensi atau izin IT, pakai Docker Engine di dalam WSL 2 (keputusan pemilik, 28 Oktober 2026).
- **Frontend:** Svelte 5.57.3 dan SvelteKit 3.1.0 sebagai SPA statis (`@sveltejs/adapter-static` 4.0.0) yang di-embed ke binary Go. Juga Vite 8.3.5, `@sveltejs/vite-plugin-svelte` 7.3.1, TypeScript 6.0.3 (bukan 7), svelte-check 4.7.7, Vitest 5.0.5, dan Playwright 1.64.1. Node 24.22.1 LTS dipakai hanya untuk build.
- **Nanti:** tabulator-tables (grid), @xyflow/svelte dengan elkjs (diagram PFD), ClamAV, Caddy, Docker Compose untuk deployment. Mesin PDF diputuskan lewat uji S5 (R-PRT-03), sebelum Tahap 6 **(usulan)**.
- **Generator TypeScript untuk OpenAPI:** masih terbuka (Q-47) dan diputuskan di M0.5 lewat uji kecil.

Aturan versi (R-VER-01 sampai R-VER-05, D-44):
- Hanya rilis stabil. Tanpa alpha, beta, rc, next, canary, atau nightly. Untuk npm berarti dist-tag `latest`.
- Semua peer dependency harus cocok tanpa `--force` atau `--legacy-peer-deps`.
- Masa tunggu 14 hari (E-14) untuk dua kasus: line yang baru punya satu rilis (hanya bila tidak ada line lain yang cocok peer dependency-nya), dan paket 0.x. Pengecualiannya dicatat di `docs/versions.md`.
- Jangan menulis versi dari ingatan. Cek dengan `go list -m -versions <modul>`, `npm view <paket> dist-tags`, dan halaman rilis resmi.

### SvelteKit 3 (RK-55)

SvelteKit 3.0.0 baru rilis pada 1 Oktober 2026, jadi pola dari ingatan model bisa masih pola SvelteKit 2. Yang sudah tercatat di brief:
- konfigurasi pindah ke `vite.config.ts`, bukan `svelte.config.js`;
- alias impor `$lib` menjadi `#lib`;
- butuh Node 22.17 atau lebih baru, TypeScript 6, Svelte 5.57.1 atau lebih baru, Vite 8.0.12 atau lebih baru, dan vite-plugin-svelte 7.

Sebelum menulis kode SvelteKit, baca dokumentasi resmi dan panduan migrasi untuk versi yang terpasang. Bila dokumentasi tidak bisa diakses, katakan dan tanya pemilik. Jangan menebak.

## 5. Aturan desain yang paling cepat disentuh kode

Ini ringkasan. Teks lengkap dan statusnya ada di brief bagian 5 dan 6.

- **Izin:** semua operasi lewat satu fungsi pemeriksaan izin (aktor, aksi, resource). Fungsi ini menegakkan peran kasar sejak M1.2 (D-35, R-SEC-09). Jenis aktornya manusia, service, atau customer user (R-API-03).
- **Model akses:** katalog hak tetap, peran adalah kumpulan hak yang isinya diatur admin, dan ada empat level: Buat/Edit, Verifikasi, Lihat, Admin (D-66, R-SEC-10). Isi awalnya di `docs/uml/U3-peta-pengguna.md`.
- **Akun admin:** akun terpisah yang hanya memegang hak admin dan tidak pernah memegang peran kerja. Akun ini ditautkan ke pemegangnya, dan akun admin terakhir tidak bisa dihapus (D-72, D-73, D-74). Audit log hanya dilihat lewat akun admin (D-75).
- **Cakupan:** Project Leader dan Project Engineer hanya bekerja untuk customer di Project Group sendiri (D-71).
- **Aturan yang tidak bisa dimatikan** (R-SEC-11, sebagian masih usulan): pemohon tidak memverifikasi permintaannya sendiri, satu orang hanya satu slot dalam satu rute, pencatat rilis berbeda dari penyusun, revisi rilis tidak bisa diubah, dan audit log append-only.
- **Audit log:** append-only dan ditegakkan di database. Role aplikasi tanpa UPDATE dan DELETE, ada trigger penolak, dan ada hash chain (R-SEC-03).
- **Nomor dokumen:** dialokasikan server dengan baris penghitung dalam transaksi, bukan sequence PostgreSQL. Ada unique constraint dan kunci idempoten, dan 50 pendaftaran paralel harus menghasilkan 50 nomor berbeda tanpa lubang (R-NUM-03). Nomor disimpan sebagai teks dan tidak dihitung ulang (R-NUM-02).
- **ID tetap, nama hanya tampilan:** proses (D-06), jabatan (D-58), kolom format template (R-TPL-02), dan baris dokumen (R-DOC-02).
- **Baris dokumen:** baris datar dengan isi JSONB yang divalidasi terhadap versi format template (D-02, R-DOC-01).
- **Revisi rilis:** tidak bisa diubah, dengan snapshot dan SHA-256 (R-REV-02).
- **Error API:** satu format berisi kode, pesan, dan request ID (R-API-13).
- **Integrasi:** tidak ada token integrasi sebelum M8.1 (D-19).

## 6. Mengevaluasi desain sambil membangun (D-77)

Kode sekarang punya dua tujuan: belajar, dan menguji apakah desain (brief, UML, milestone) bisa dibangun dan masuk akal.

1. Sebelum mulai satu milestone, baca aturan, keputusan, dan use case yang disentuhnya.
2. Saat membangun, catat setiap temuan di `docs/evaluasi-desain.md` dengan format yang ada di file itu. Jenisnya: kontradiksi, celah, ambigu, sulit dibangun, atau penyederhanaan.
3. Jangan memutuskan temuan sendiri. Bila temuan menghambat milestone, berhenti dan tanya pemilik dengan opsi. Bila tidak menghambat, pakai asumsi paling aman, tandai di kode dengan komentar `// ASUMSI E-nn`, lalu lanjut.
4. Jangan mengubah brief, UML, atau milestone kecuali pemilik memintanya. Pemilik memutuskan temuan, lalu keputusannya dicatat di brief sebagai item D (brief bagian 15, butir 11).
5. Di akhir setiap milestone, laporkan temuan baru di ringkasan.

## 7. Data asli dan keamanan

- Dokumen asli perusahaan dan customer boleh dipakai dalam pengembangan (D-29), tapi hanya di repo private ini dan di komputer pemilik. Jangan pernah menyalinnya ke tempat lain: repo publik, gist, situs paste, atau layanan lain.
- Sampai pemilik memutuskan R-SEC-08, simpan dokumen asli di folder `data/` yang masuk `.gitignore`, dan jangan di-commit. Opsi lainnya adalah repo data private terpisah (R-SEC-08, usulan).
- Jangan menulis nama orang asli di kode, data awal (seed), test, atau contoh. Pakai nama jabatan atau nama contoh **(usulan)**.
- Password dan rahasia lain disimpan di `.env` yang tidak di-commit. Sediakan `.env.example`.
- Password, token, kunci, dan alamat koneksi yang memuat password tidak pernah ditulis ke log, pesan error, atau audit log. Pesan error ke pengguna tidak memuat detail teknis internal; detailnya dicatat di log server bersama request ID (R-API-13). Aturan ini dibuktikan dengan test (E-18, keputusan pemilik 28 Oktober 2026).

## 8. Repo, perintah, dan git

- **Struktur folder:** disetujui pemilik di M0.2 (28 Oktober 2026). Setiap folder baru dibuat saat milestone-nya tiba.

| Folder atau file | Isi | Mulai |
|---|---|---|
| `cmd/revioten/` | Titik masuk program (`package main`): membaca flag, menjalankan server | M0.2 |
| `internal/server/` | Adapter HTTP: daftar route dan handler. Tipis, tanpa aturan bisnis (R-API-01) | M0.2 |
| `internal/<domain>/` | Aturan bisnis per area, misalnya `internal/customer` | M0.5 |
| `internal/db/` | Koneksi pgx, lalu kode hasil sqlc (tidak diedit tangan) | M0.3 |
| `sql/migrations/` | Migrasi goose. Sejak M1.3 setiap migrasi yang membuat tabel juga menulis `GRANT` untuk role aplikasi `revioten_app`, dengan hak sekecil mungkin (E-06) | M0.5 |
| `sql/queries/` | Query SQL untuk sqlc | M0.5 |
| `internal/db/dbtest/` | **(usulan M0.5)** Bantuan test: PostgreSQL di container dan satu database baru per test. Hanya dipakai file `_test.go` | M0.5 |
| `api/openapi.yaml` | Kontrak OpenAPI 3.0.4 (D-38) | M0.5 |
| `web/` | SvelteKit. Hasil build (`web/build/app`) di-embed ke binary Go lewat `web/web.go`; `web/build/.gitkeep` di-commit supaya `go build` tetap jalan sebelum frontend di-build | M0.4 |
| `web/e2e/` | **(usulan M0.5)** Test E2E Playwright dan skrip server E2E | M0.5 |
| `bin/` | Hasil `go build`, tidak di-commit | M0.2 |
| `docs/` | Brief, milestone, UML, catatan milestone, evaluasi desain, versi | Sesi 1 |
| Root | `go.mod`, `compose.yaml`, `.env.example`, `.gitignore`, `.gitattributes`, `sqlc.yaml` (M0.5) | M0.1, M0.2 |
- **Perintah:** hanya perintah yang sudah terbukti jalan (menjalankan database, test, build). Lihat bagian "Perintah yang sudah terbukti" di bawah.
- **Commit:** kecil, jelas, dengan ID di pesannya, misalnya `M0.2: endpoint /health beserta test`.
- **Branch:** commit langsung di `main` lalu push ke `main` (pernyataan pemilik, 27 Oktober 2026). Ini menggantikan usulan satu branch per milestone. Jangan force-push.

### Perintah yang sudah terbukti

Dijalankan dari root repo. Terbukti di container cloud Linux (bash) pada M0.1 (27 Oktober 2026), M0.2, M0.3, M0.4, M0.5, M1.1, M1.2, dan M1.3 (28 Oktober 2026). Docker Compose membaca `.env` sendiri, dan perintah `go` sama di PowerShell Windows. Catatan dalam kurung untuk Windows belum dicoba di komputer pemilik.

| Tujuan | Perintah |
|---|---|
| Cek versi alat | `go version`, `node -v`, `docker --version`, `docker compose version` |
| Cek versi modul Go | `go list -m -versions <modul>`, lalu `go list -m -json <modul>@<versi>` (tanggal di field `Time`) |
| Cek versi paket npm | `npm view <paket> dist-tags time --json`, lalu `npm view <paket>@<versi> peerDependencies engines` |
| Siapkan rahasia lokal (sekali) | `cp .env.example .env` (bash), lalu ganti `POSTGRES_PASSWORD` dan `APP_DB_PASSWORD` (huruf dan angka saja, minimal 16 karakter, sejak M1.3). Di PowerShell: `Copy-Item .env.example .env` (belum dicoba di komputer pemilik). File `.env` lama cukup ditambah baris `APP_DB_PASSWORD=...` |
| Periksa file Compose | `docker compose config --quiet` |
| Nyalakan PostgreSQL | `docker compose up -d` |
| Lihat status (harus `healthy`) | `docker compose ps` |
| Buka psql di container | `docker compose exec db psql -U revioten -d revioten` |
| Cek versi server | `docker compose exec db psql -U revioten -d revioten -c "select version();"` |
| Matikan PostgreSQL, data tetap | `docker compose down` |
| Jalankan semua test Go | `go test ./...` (rinci per kasus: `go test -v ./...`). Sejak M0.3 butuh Docker yang sedang berjalan, karena test database menyalakan container PostgreSQL 18.6 sendiri lewat testcontainers-go |
| Test tanpa Docker | `go test -short ./...` (test database dilewati, test lain tetap berjalan) |
| Periksa kode Go | `go vet ./...`, lalu `gofmt -l .` (tanpa keluaran berarti sudah rapi) |
| Jalankan server | Sejak M0.5 butuh PostgreSQL yang menyala (`docker compose up -d`) dan file `.env` yang terisi. Lalu `go run ./cmd/revioten`, atau di port lain: `go run ./cmd/revioten -addr 127.0.0.1:8081`. Migrasi database berjalan otomatis saat server mulai, sebagai pemilik database; sesudah itu server memakai role `revioten_app` (M1.3). Hentikan dengan Ctrl+C |
| Coba endpoint /health | `curl -i http://127.0.0.1:8080/health` (Windows: `curl.exe -i ...` atau `Invoke-RestMethod http://127.0.0.1:8080/health`) |
| Buat akun admin pertama (sekali, E-07) | `go run ./cmd/revioten create-admin -name "Nama Pemegang" -login admin.nama`, lalu ketik password dua kali (tidak tampil di layar, minimal 15 karakter). Hanya jalan bila belum ada akun admin. Password tidak pernah ditulis sebagai argumen perintah. Di cloud terbukti lewat pipa (password dikirim ke stdin, seperti di `web/e2e/server.mjs`); mode ketik di terminal belum dicoba |
| Login | Buka http://127.0.0.1:8080, yang mengarahkan ke halaman login. Semua halaman dan API (kecuali `/health` dan login) butuh login sejak M1.1 |
| Coba API dengan login (bash) | Tulis `{"login":"admin.nama","password":"..."}` ke `bin/login.json` (folder `bin/` tidak di-commit), lalu `curl -i -c bin/cookies.txt -H 'Content-Type: application/json' -d @bin/login.json http://127.0.0.1:8080/api/v1/session`. Setelah itu `curl -b bin/cookies.txt http://127.0.0.1:8080/api/v1/customers` dan `curl -b bin/cookies.txt -X POST -H 'Content-Type: application/json' -d '{"code":"ab","name":"Example Customer"}' http://127.0.0.1:8080/api/v1/customers`. Hapus `bin/login.json` setelah dipakai. Di Windows, `curl.exe` dengan argumen yang sama (belum dicoba di komputer pemilik) |
| Kelola akun dan peran (sejak M1.2) | Login dengan akun admin, lalu buka menu Accounts (http://127.0.0.1:8080/admin/accounts). Password sementara akun baru tampil satu kali dan wajib diganti pemiliknya saat login pertama. Menu dan tombol mengikuti hak akun yang login; server tetap memeriksa setiap operasi (akun tanpa A-2 mendapat 403 saat menambah customer lewat curl) |
| Lihat dan periksa audit log (sejak M1.3) | Login dengan akun admin, buka menu Audit log (http://127.0.0.1:8080/admin/audit), lalu tekan "Check the chain" |
| Coba ubah audit log (harus gagal) | Sebagai pemilik: `docker compose exec db psql -U revioten -d revioten -c "UPDATE audit_log SET action = action WHERE false"` (ditolak trigger). Sebagai role aplikasi: perintah yang sama dengan `-U revioten_app` (`permission denied`). `WHERE false` membuat perintah ini tidak mengubah apa pun, juga bila trigger sedang mati |
| Buat ulang kode Go dari kontrak dan SQL | `go generate ./...` (oapi-codegen dan sqlc dijalankan lewat `go run` dengan versi terkunci; pertama kali butuh internet dan waktu kompilasi). Lalu `go build ./...` dan `go test ./...` |
| Buat ulang tipe TypeScript dari kontrak | `npm --prefix web run gen:api` |
| Build binary | `go build -o bin/revioten ./cmd/revioten` (Windows: `-o bin/revioten.exe`) |
| Pasang paket frontend (sekali, dan setiap `web/package-lock.json` berubah) | `npm --prefix web ci` (butuh Node 24.22.1) |
| Periksa tipe frontend | `npm --prefix web run check` |
| Test frontend (Vitest) | `npm --prefix web test` |
| Binary lengkap dengan tampilan web | `npm --prefix web run build`, lalu `go build -o bin/revioten ./cmd/revioten` |
| Development tampilan web | Jalankan server Go (`go run ./cmd/revioten`), lalu di jendela lain `npm --prefix web run dev` dan buka http://localhost:5173. Vite meneruskan `/health` dan `/api` ke server Go |
| Pasang browser untuk test E2E (sekali) | `npm --prefix web exec -- playwright install chromium` (belum dicoba di komputer pemilik; di container cloud tidak dipakai, lihat baris berikut) |
| Test E2E (Playwright) | `docker compose up -d` dulu, lalu `npm --prefix web run test:e2e`. Skrip ini membangun tampilan web, membuat ulang database `revioten_e2e` di container, menjalankan server di port 8090, lalu membuka browser. Di container cloud: `PLAYWRIGHT_CHROMIUM_EXECUTABLE=/opt/pw-browsers/chromium npm --prefix web run test:e2e` |

Jangan menjalankan `docker compose down -v` kecuali memang ingin menghapus seluruh isi database, karena `-v` ikut menghapus volume `revioten_pgdata`.

## 9. Catatan per milestone

Setiap milestone menghasilkan `docs/catatan/<ID>-<nama>.md`. Isinya:
- tujuan dan apa yang dibuat;
- cara mengulang dari nol;
- konsep yang dipelajari;
- keputusan beserta alasannya;
- asumsi;
- temuan desain baru (ID E-);
- status test;
- satu latihan tulis tangan untuk pemilik.
