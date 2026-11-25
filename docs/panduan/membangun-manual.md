# Panduan membangun Revioten secara manual

- **Untuk siapa:** pemula, termasuk pemilik proyek yang ingin bisa membangun ulang sistem ini sendiri tanpa AI (D-12). Anda dianggap sudah bisa sedikit memprogram (variabel, fungsi, if, loop), tetapi belum pernah membuat aplikasi web lengkap.
- **Dibuat:** 21 November 2026. **Ditulis ulang** 25 November 2026 untuk pemula: teknologi dijelaskan satu per satu, dan urutan membangun dimulai dari server kosong dan halaman login, lalu fitur ditambah sedikit demi sedikit. Urutannya tidak mengikuti milestone.
- **Cara memakai:**
  1. Baca bagian 1 sampai 3 sekali, tanpa mengetik apa pun. Tujuannya mengenal nama-nama alat dan konsep.
  2. Siapkan komputer dengan bagian 5.
  3. Kerjakan bagian 6 tahap demi tahap. Setiap tahap berakhir dengan sesuatu yang bisa dicoba di browser atau di test. Jangan lompat tahap.
  4. Mulai tahap fitur pertama (tahap 7), pakai resep di bagian 7 setiap kali membuat fitur baru, dan daftar periksa di bagian 9.
  5. Bagian 8 menjelaskan mekanisme yang dipakai di banyak tempat. Bacalah saat bertemu mekanismenya.

Dokumen lain yang terkait:
- `CLAUDE.md`: aturan kerja dan daftar perintah yang sudah terbukti jalan.
- `docs/catatan/`: satu catatan per milestone, berisi langkah rinci, konsep, dan latihan. Setiap tahap di bagian 6 menyebut catatan mana yang dibaca.
- `docs/panduan/skenario-alur-web.md`: alur memakai web dari sisi pengguna.
- `docs/panduan/panduan-operasional.md`: deploy, backup, dan restore.
- `docs/versions.md`: versi setiap alat beserta sumbernya.
- `docs/uml/U4-activity.puml`: alur kerja pengguna dalam activity diagram.

**Tentang potongan kode di panduan ini:** potongan diambil dari repo, tetapi **diringkas** supaya mudah dibaca. Versi lengkapnya ada di berkas yang disebut. Bila panduan dan kode berbeda, kode yang benar.

---

## 1. Gambaran besar

Revioten adalah **satu program Go** yang melayani dua hal:
- **API JSON** di alamat `/api/v1/...`: tempat data dibaca dan diubah;
- **tampilan web** (halaman yang dilihat di browser): dibuat dengan SvelteKit, di-build menjadi berkas HTML, CSS, dan JavaScript, lalu dimasukkan ke dalam program Go.

Datanya disimpan di **PostgreSQL**, yang berjalan di Docker.

```
 Browser
   │  (halaman Svelte memanggil fetch ke /api/v1/...)
   ▼
 internal/server      ← adapter HTTP: baca request, panggil layanan, tulis JSON
   │                    (kerangkanya dibuat oapi-codegen dari api/openapi.yaml)
   ▼
 internal/<domain>    ← aturan bisnis: izin, validasi, transaksi, audit log
   │                    (contoh: internal/customer, internal/numbering)
   ▼
 internal/db          ← query SQL yang sudah diketik (dibuat sqlc dari sql/queries)
   │
   ▼
 PostgreSQL           ← tabel dari sql/migrations, ditambah trigger, GRANT, dan RLS
```

Bayangkan sebuah kantor:
- **browser** adalah tamu yang mengisi formulir;
- **`internal/server`** adalah resepsionis: menerima formulir, memeriksa bentuknya, lalu meneruskannya. Resepsionis tidak memutuskan apa-apa;
- **`internal/<domain>`** adalah petugas yang tahu aturannya: siapa boleh apa, isian mana yang wajib;
- **`internal/db`** adalah lemari arsip dengan laci berlabel;
- **PostgreSQL** adalah gudang arsipnya. Gudang ini juga punya kunci sendiri, jadi walaupun petugas lalai, arsip tertentu tetap tidak bisa diubah.

Tiga prinsip yang membuat susunan ini mudah diikuti:

1. **Kontrak lebih dulu (API-first, D-61).** Bentuk setiap alamat API ditulis dulu di `api/openapi.yaml`. Kode Go dan tipe TypeScript dibuat otomatis dari kontrak itu, jadi server dan layar selalu sepakat soal bentuk data.
2. **Handler tipis (R-API-01).** Handler di `internal/server` tidak memuat aturan. Semua aturan ada di paket domain, supaya bisa dites tanpa HTTP.
3. **Database ikut menjaga.** Aturan yang tidak boleh dilanggar (audit log tidak bisa diubah, revisi terbit beku, akun tidak dihapus) tidak hanya dicek di Go, tetapi juga dikunci di database dengan GRANT, trigger, dan RLS. Bila ada bug di Go, database tetap menolak.

---

## 2. Mengenal teknologi yang dipakai

Bagian ini menjelaskan setiap bahasa, framework, pustaka (library), dan alat: **apa itu**, **mengapa dipakai di sini**, dan **di mana Anda melihatnya di repo**. Versi tepatnya ada di `docs/versions.md`. Jangan menulis versi dari ingatan; cek dengan perintah di `CLAUDE.md` bagian 4.

Istilah singkat:
- **Framework** adalah kerangka yang menentukan cara Anda menyusun program, misalnya SvelteKit menentukan bahwa setiap halaman adalah berkas `+page.svelte` di folder `routes`.
- **Pustaka (library)** adalah kumpulan fungsi siap pakai yang Anda panggil sendiri, misalnya pgx untuk berbicara dengan PostgreSQL.
- **Dependency** adalah semua pustaka yang dibutuhkan program. Di Go dicatat di `go.mod` dan `go.sum`; di frontend di `web/package.json` dan `web/package-lock.json`. Berkas "lock" dan "sum" mengunci versi persisnya, supaya komputer lain memasang hal yang sama.
- **Generator kode** adalah alat yang menulis kode untuk Anda dari sumber lain (SQL atau kontrak API). Kode hasilnya tidak pernah diedit tangan.

### 2.1 Alat di komputer

| Alat | Apa itu | Untuk apa di proyek ini |
|---|---|---|
| **Go** | Bahasa pemrograman dari Google. Hasilnya satu berkas program (binary) yang langsung jalan, tanpa perlu memasang apa pun lagi di komputer tujuan | Bahasa server. Perintahnya: `go run` (jalankan), `go build` (buat binary), `go test` (jalankan test), `go generate` (jalankan generator kode) |
| **Docker** dan **Docker Compose** | Docker menjalankan program di dalam "container": kotak terisolasi yang membawa sistem operasinya sendiri. Compose menyalakan beberapa container sekaligus dari satu berkas | Menjalankan PostgreSQL tanpa memasangnya langsung di Windows (`compose.yaml`). Test juga menyalakan container PostgreSQL sementara |
| **PostgreSQL** | Database relasional: data disimpan di tabel, dibaca dan diubah dengan bahasa SQL | Semua data aplikasi. Versi 18, karena butuh `uuidv7()` untuk ID |
| **Node.js** dan **npm** | Node menjalankan JavaScript di luar browser. npm memasang paket JavaScript | **Hanya untuk build** tampilan web dan menjalankan test frontend. Server yang berjalan tidak butuh Node |
| **Git** | Pencatat riwayat perubahan kode | Commit kecil dengan ID keputusan di pesannya, lalu push ke GitHub |
| **Chrome atau Edge** | Browser | Membuat PDF cetakan (mode headless, tanpa jendela), dan browser untuk test Playwright |
| **VS Code** | Editor | Pasang ekstensi Go dan Svelte supaya ada warna, saran, dan pemeriksaan tipe |

### 2.2 Backend (server Go)

| Pustaka atau alat | Apa itu | Mengapa dipakai | Di mana |
|---|---|---|---|
| **`net/http`** (bawaan Go) | Pustaka standar Go untuk server HTTP. Sejak Go 1.22, `http.ServeMux` bisa mencocokkan method dan pola alamat, misalnya `GET /api/v1/customers/{id}` | Tidak perlu framework web tambahan (Gin, Echo). Lebih sedikit hal yang harus dipelajari | `internal/server/server.go` |
| **`log/slog`** (bawaan Go) | Pencatat log terstruktur: pesan beserta pasangan kunci dan nilai | Log server dengan request ID, tanpa rahasia (R-SEC-14) | Di semua paket |
| **`embed`** (bawaan Go) | Memasukkan berkas ke dalam binary saat `go build` | Migrasi SQL dan hasil build tampilan web ikut di dalam satu binary, jadi cukup satu berkas untuk dijalankan | `sql/migrations/migrations.go`, `web/web.go` |
| **pgx** (`github.com/jackc/pgx/v5`) | Driver PostgreSQL untuk Go: membuka koneksi, menjalankan query, dan **connection pool** (kumpulan koneksi yang dipakai bergantian) | Driver PostgreSQL paling lengkap dan cepat untuk Go, mendukung tipe khusus PostgreSQL | `internal/db/db.go`, `internal/db/tx.go` |
| **goose** (`github.com/pressly/goose/v3`) | Alat **migrasi database**: menjalankan berkas SQL bernomor satu per satu dan mencatat mana yang sudah dijalankan | Skema database tumbuh sedikit demi sedikit, sama di semua komputer. Dipakai sebagai pustaka, jadi migrasi jalan otomatis saat server mulai | `sql/migrations/*.sql`, `internal/db/migrate.go` |
| **sqlc** (generator, tidak masuk `go.mod`) | Membaca query SQL yang Anda tulis, lalu **menulis fungsi Go bertipe** untuk setiap query | Anda tetap menulis SQL biasa (tanpa ORM), tetapi salah ketik nama kolom sudah ketahuan saat generate, bukan saat server berjalan | `sqlc.yaml`, `sql/queries/*.sql`, hasilnya `internal/db/*.sql.go` |
| **OpenAPI 3.0.4** | Format standar (YAML) untuk menulis bentuk API: alamat, method, isi request, isi jawaban | Kontrak tertulis yang bisa dibaca tim IT dan alat lain (D-38) | `api/openapi.yaml`, `api/integration.yaml` |
| **oapi-codegen** (generator) dan **`oapi-codegen/runtime`** | Membaca kontrak OpenAPI, lalu menulis tipe Go dan **interface server**: daftar method yang wajib Anda tulis, satu per alamat API. Runtime-nya pustaka kecil yang dipakai kode hasil generate | Bila kontrak bertambah satu alamat dan handler-nya belum ditulis, `go build` langsung gagal. Server tidak bisa menyimpang dari kontrak | `internal/server/oapi-codegen.yaml`, hasilnya `internal/server/api.gen.go` |
| **`golang.org/x/crypto/argon2`** | Fungsi hash password **argon2id** | Password disimpan sebagai hash yang sengaja lambat dan butuh banyak memori, sehingga sulit ditebak walau database bocor (rekomendasi OWASP) | `internal/auth/password.go` |
| **`golang.org/x/term`** | Membaca ketikan di terminal tanpa menampilkannya | Perintah `create-admin` meminta password tanpa menampilkannya di layar | `cmd/revioten/createadmin.go` |
| **excelize** (`github.com/xuri/excelize/v2`) | Membaca dan menulis berkas Excel `.xlsx` | Cetak Excel, laporan, dan import akun, customer, model, serta nomor lama dari Excel | `internal/printout`, `internal/masterimport`, `internal/auth/accountimport.go` |
| **`rsc.io/qr`** | Membuat gambar kode QR | QR verifikasi di cetakan (saklarnya bawaan mati, D-62) | `internal/printout` |
| **age** (`filippo.io/age`) | Enkripsi berkas dengan passphrase | Backup database terenkripsi, bisa dibuka juga dengan alat `age` resmi (M1.7, D-119) | `internal/backup` |
| **testcontainers-go** | Menyalakan container Docker dari dalam test Go | Setiap test database memakai PostgreSQL sungguhan yang baru, jadi CHECK, trigger, dan GRANT ikut teruji | `internal/db/dbtest/dbtest.go` |
| **`go.yaml.in/yaml/v3`** | Membaca YAML | Hanya di test: memeriksa bahwa daftar alamat publik sama dengan kontrak | `internal/server/*_test.go` |

Beberapa modul di `go.mod` bertanda `// indirect`. Itu dependensi yang ditarik oleh pustaka lain, misalnya pustaka Docker yang dibawa testcontainers-go. Anda tidak memanggilnya langsung.

**Yang sengaja tidak dipakai:** ORM (GORM), framework web (Gin, Echo), Redis, dan sequence PostgreSQL untuk nomor dokumen. Alasannya: lebih sedikit lapisan tersembunyi, sehingga kodenya bisa dibaca dari atas ke bawah (D-12).

### 2.3 Frontend (tampilan web)

| Pustaka atau alat | Apa itu | Mengapa dipakai | Di mana |
|---|---|---|---|
| **TypeScript** | JavaScript dengan tipe. Kesalahan bentuk data ketahuan saat diperiksa, sebelum dijalankan di browser | Tipe data API dibuat dari kontrak, jadi layar ikut tahu bila kontrak berubah | Semua berkas `.ts` dan `<script lang="ts">` |
| **Svelte 5** | Framework komponen. Satu berkas `.svelte` berisi `<script>`, HTML, dan `<style>`. Svelte mengompilasinya menjadi JavaScript biasa | Sintaksnya dekat dengan HTML biasa, jadi mudah dibaca pemula. Versi 5 memakai **runes**: `$state` (nilai yang berubah dan memperbarui layar), `$derived` (nilai turunan), `$effect` (jalankan sesuatu saat nilai berubah), `$props` (masukan komponen) | `web/src/**/*.svelte` |
| **SvelteKit 3** | Framework aplikasi di atas Svelte: routing (alamat → halaman), layout bersama, dan build. **Satu folder = satu alamat**: `src/routes/customers/+page.svelte` adalah halaman `/customers` | Routing dan build tidak perlu disusun sendiri. Di SvelteKit 3, alias impor adalah `#lib`, bukan `$lib` seperti di SvelteKit 2, dan konfigurasinya ada di `vite.config.ts` | `web/src/routes/`, `web/vite.config.ts` |
| **adapter-static** | Pengaya SvelteKit yang menghasilkan berkas statis (SPA) | Tidak perlu server Node. Hasilnya di `web/build/app`, lalu di-embed ke binary Go. Semua alamat halaman dilayani `200.html`, lalu routing dikerjakan di browser | `web/vite.config.ts`, `web/src/routes/+layout.ts` (`ssr = false`) |
| **Vite** | Alat build dan server development yang cepat | `npm run dev` untuk mencoba layar sambil mengubah kode, dan `npm run build` untuk hasil akhir. Saat dev, Vite meneruskan `/api` ke server Go | `web/vite.config.ts` |
| **svelte-check** | Pemeriksa tipe untuk berkas Svelte | `npm --prefix web run check` | |
| **@hey-api/openapi-ts** | Generator tipe TypeScript dari kontrak OpenAPI | Hanya tipe yang dibuat; pemanggilan `fetch` ditulis tangan supaya mudah dibaca (D-97) | `web/openapi-ts.config.ts`, hasilnya `web/src/lib/api/gen/` |
| **Vitest** | Penjalan test untuk TypeScript | Test fungsi kecil di `web/src/lib/*.spec.ts`, tanpa browser | `npm --prefix web test` |
| **Playwright** | Mengendalikan browser sungguhan dari kode | Test E2E (end-to-end: login, klik, isi formulir), tur otomatis, dan pemotret gambar Help | `web/e2e/`, `web/tour/`, `web/help-shots/` |
| **tabulator-tables** | Komponen tabel seperti spreadsheet | Mengubah sel dokumen langsung di tabel (M4.2) | `web/src/lib/documents/Workbook.svelte` |
| **@xyflow/svelte** dan **elkjs** | xyflow menggambar diagram simpul dan panah; elkjs menghitung letaknya otomatis | Diagram PFD di layar, dimuat hanya saat dibuka (M4.10) | `web/src/lib/documents/PfdDiagram.svelte` |
| **@fontsource-variable/geist** dan **geist-mono** | Berkas huruf yang dibundel | Huruf sama di semua komputer tanpa internet (D-127) | `web/src/routes/+layout.svelte` |

### 2.4 Yang dipakai saat deploy (belum perlu di awal)

| Alat | Untuk apa |
|---|---|
| **Caddy** | Server HTTPS di depan aplikasi saat deploy lokal (`deploy/Caddyfile`, M8.3) |
| **ClamAV** | Pemindai antivirus untuk lampiran (`CLAMAV_ADDR`, M8.2) |
| **PlantUML** | Merender diagram UML di `docs/uml` (Java) |

---

## 3. Konsep dasar untuk pemula

Baca sekilas. Setiap konsep muncul lagi di tahap yang memakainya.

| Konsep | Penjelasan singkat | Muncul di tahap |
|---|---|---|
| **HTTP request dan response** | Browser mengirim permintaan (method `GET`, `POST`, `PUT`, `DELETE` ke satu alamat), server membalas dengan kode status (200 berhasil, 400 isian salah, 401 belum login, 403 tidak berhak, 404 tidak ada, 409 bentrok, 500 error server) dan isi | 1 |
| **JSON** | Format teks untuk data, misalnya `{"code":"AB","name":"Example"}` | 1 |
| **Tabel, baris, kolom, primary key** | Data di database disusun seperti sheet Excel. Primary key adalah kolom yang nilainya unik per baris (di sini `id` berupa UUID) | 2 |
| **Migrasi** | Berkas SQL bernomor yang mengubah skema selangkah demi selangkah. Berkas lama tidak pernah diubah; perubahan baru selalu berkas baru | 2 |
| **Constraint** | Aturan di database: `NOT NULL`, `UNIQUE`, `CHECK`, `REFERENCES`. Database menolak data yang melanggar | 2 |
| **Transaksi** | Beberapa perubahan yang "semua atau tidak sama sekali". Bila satu gagal, semuanya dibatalkan (rollback) | 4 |
| **Hash** | Sidik jari satu arah dari data. Dari hash tidak bisa kembali ke data asli. Password disimpan sebagai hash; isi revisi terbit dijaga SHA-256 | 4 |
| **Cookie dan sesi** | Sesudah login, server memberi browser cookie berisi token. Browser mengirim cookie itu di setiap permintaan, sehingga server tahu siapa yang meminta | 4 |
| **Middleware** | Fungsi yang membungkus handler: berjalan sebelum dan sesudah setiap permintaan, misalnya memeriksa sesi atau memberi request ID | 4 |
| **SPA (single-page application)** | Browser memuat satu halaman HTML sekali, lalu JavaScript yang mengganti isi layar saat pindah alamat | 3 |
| **Komponen dan state** | Bagian layar yang bisa dipakai ulang. State adalah nilai yang bila berubah membuat layar digambar ulang (`$state`) | 3 |
| **Test** | Kode yang memeriksa kode lain. Test unit memeriksa satu fungsi, test integrasi memakai database sungguhan, test E2E memakai browser sungguhan | Semua |
| **Generate kode** | Menjalankan alat yang menulis kode dari sumber lain. Bila hasilnya salah, perbaiki sumbernya (SQL atau kontrak), bukan hasilnya | 2, 4 |

---

## 4. Peta folder

| Folder | Isi | Contoh berkas |
|---|---|---|
| `cmd/revioten/` | Titik masuk: membaca `.env`, migrasi, menyalakan server; juga perintah `create-admin`, `reset-admin-password`, `backup`, `restore` | `main.go`, `config.go` |
| `api/` | Kontrak OpenAPI | `openapi.yaml`, `integration.yaml` |
| `sql/migrations/` | Perubahan skema berurutan (goose), mulai `00001_create_customers.sql` | `00002_create_people_accounts_sessions.sql` |
| `sql/queries/` | Query SQL yang diberi nama, bahan sqlc | `auth.sql`, `customers.sql` |
| `internal/db/` | Koneksi pgx, transaksi, subjek RLS, dan kode hasil sqlc (`*.sql.go`, `*.gen.go`, jangan diedit) | `db.go`, `tx.go`, `migrate.go`, `subject.go` |
| `internal/db/dbtest/` | PostgreSQL sementara untuk test | `dbtest.go` |
| `internal/auth/` | Login, sesi, password, akun, dan peran | `service.go`, `password.go`, `token.go` |
| `internal/access/` | Satu titik pemeriksaan izin | `checker.go`, `access.go` |
| `internal/audit/` | Audit log append-only dengan hash chain | `audit.go`, `verify.go` |
| `internal/problem/` | Bentuk error domain (tidak ditemukan, isian salah, bentrok) | |
| `internal/<domain>/` | Aturan per area | `customer/`, `numbering/`, `document/`, `signature/` |
| `internal/server/` | Handler HTTP; `api.gen.go` hasil oapi-codegen | `server.go`, `session.go`, `spa.go`, `apierror.go` |
| `web/src/routes/` | Halaman SvelteKit, satu folder per alamat | `login/+page.svelte`, `+layout.svelte` |
| `web/src/lib/` | Komponen, klien API, menu, sesi, katalog teks, Help | `api/client.ts`, `session.svelte.ts`, `menu.ts` |
| `web/src/lib/i18n/` | Teks layar Inggris dan Indonesia | `en.ts`, `id.ts` |
| `web/src/lib/help/` | Buku panduan pengguna dua bahasa | `en.ts`, `id.ts` |
| `web/src/app.css` | Gaya bersama: warna, huruf, tombol, field, tabel (D-127) | |
| `web/e2e/` | Test E2E Playwright | `helpers.ts`, `responsive.test.ts` |
| `web/tour/`, `web/help-shots/` | Tur otomatis dan pemotret gambar Help | |
| `web/web.go` | Memasukkan hasil build web ke binary Go | |
| `docs/` | Brief, milestone, UML, catatan, panduan | |

---

## 5. Menyiapkan komputer

### 5.1 Pasang alat

Pasang Go, Docker Desktop (atau Docker Engine di WSL 2), Node.js LTS, Git, dan VS Code. Versinya lihat `docs/versions.md`. Lalu periksa:

```powershell
go version
```

```powershell
node -v
```

```powershell
docker compose version
```

Alat yang dijalankan lewat `go run` atau `npm` sehingga tidak perlu dipasang sendiri: sqlc, oapi-codegen, @hey-api/openapi-ts, Vitest, Playwright.

### 5.2 Nyalakan proyek yang sudah jadi

Lakukan ini dulu supaya Anda tahu seperti apa hasil akhirnya, sebelum membangun ulang.

1. Ambil repo dengan `git clone`, lalu masuk ke foldernya.
2. Siapkan rahasia: salin `.env.example` menjadi `.env`, lalu ganti `POSTGRES_PASSWORD` dan `APP_DB_PASSWORD` (huruf dan angka, minimal 16 karakter). File `.env` tidak pernah di-commit.
   ```powershell
   Copy-Item .env.example .env
   ```
3. Nyalakan database dan tunggu sampai `healthy`:
   ```powershell
   docker compose up -d
   ```
   ```powershell
   docker compose ps
   ```
4. Pasang paket frontend dan build tampilan:
   ```powershell
   npm --prefix web ci
   ```
   ```powershell
   npm --prefix web run build
   ```
5. Jalankan server. Migrasi berjalan otomatis.
   ```powershell
   go run ./cmd/revioten
   ```
6. Di jendela lain, buat akun admin pertama (password diketik, tidak tampil):
   ```powershell
   go run ./cmd/revioten create-admin -name "Nama Anda" -login admin.nama
   ```
7. Buka http://127.0.0.1:8080 dan login.

Pemeriksaan bahwa semua sehat:

```powershell
go test -p 1 ./...
```

```powershell
npm --prefix web test
```

`-p 1` membuat paket dites satu per satu, supaya tidak terlalu banyak container PostgreSQL menyala bersamaan di laptop.

### 5.3 Tempat berlatih

Untuk membangun ulang sendiri, buat folder kosong terpisah (misalnya `C:\latihan\revioten-ku`) dan jadikan repo ini sebagai contoh di jendela lain. Jangan berlatih di dalam repo ini. Setiap tahap di bagian 6 bisa dicocokkan dengan berkas aslinya.

---

## 6. Membangun bertahap, dari nol

Setiap tahap punya pola yang sama: **tujuan**, **konsep baru**, **langkah**, **cara mencoba**, dan **bacaan rinci** di `docs/catatan/`. Tahap 1 sampai 6 adalah fondasi yang dipakai semua fitur. Mulai tahap 7, setiap fitur dibangun dengan resep di bagian 7.

| Tahap | Hasil yang bisa dicoba | Catatan rinci |
|---|---|---|
| 1 | Server Go menjawab `/health` | M0.2 |
| 2 | Server terhubung ke PostgreSQL, migrasi jalan sendiri | M0.1, M0.3, M0.5 |
| 3 | Halaman web pertama dilayani server Go | M0.4 |
| 4 | **Halaman login**, sesi, dan logout | M1.1 |
| 5 | Akun, peran, dan izin; menu mengikuti hak | M1.2, M8.1, D-199 |
| 6 | Audit log yang tidak bisa diubah | M1.3 |
| 7 | Fitur pertama: customer dan model (irisan penuh) | M0.5, M1.4 |
| 8 | Master data lain: proses, jenis dokumen, jabatan, rute | M1.5, M1.6, M1.8, D-209 |
| 9 | Nomor dokumen tanpa lubang | M2.1 sampai M2.5 |
| 10 | Template: format, form set, process template | M3.1 sampai M3.4, FS-A sampai FS-D |
| 11 | Dokumen dan barisnya, PFD lalu rakit PFMEA dan CP | M4.1 sampai M4.10, C1 sampai C4 |
| 12 | Perubahan, draft revisi, rilis dengan snapshot | M5.1 sampai M5.10 |
| 13 | Cetak, tanda tangan basah, tanda tangan digital | S5, M6, D-198 fase 3, C2, C3 |
| 14 | Migrasi dokumen lama dan import Excel | M7, D-215, I1, I4, I5 |
| 15 | Penguatan: RLS, dua bahasa, Help, email, backup, deploy | M8, D-128, D-141, D-194, M1.7 |

### Tahap 1 · Server Go pertama

**Tujuan:** program Go yang menjawab `GET /health` dengan JSON `{"status":"ok"}`.

**Konsep baru:** modul Go, `package main`, handler HTTP, `http.ServeMux`, kode status.

**Langkah:**
1. Buat modul: `go mod init github.com/<nama>/revioten`. Berkas `go.mod` mencatat nama modul dan versi Go.
2. Buat `cmd/revioten/main.go` dengan fungsi `main` yang membuat `http.ServeMux`, mendaftarkan satu handler, lalu `ListenAndServe`:
   ```go
   mux := http.NewServeMux()
   mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
       w.Header().Set("Content-Type", "application/json")
       w.Write([]byte(`{"status":"ok"}`))
   })
   srv := &http.Server{Addr: "127.0.0.1:8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
   log.Fatal(srv.ListenAndServe())
   ```
   `ReadHeaderTimeout` mencegah klien yang sangat lambat menahan koneksi selamanya.
3. Tulis test pertama dengan `httptest.NewRecorder`: panggil handler, periksa status 200 dan isinya. Di repo: `internal/server/health_test.go`.

**Cara mencoba:**

```powershell
go run ./cmd/revioten
```

Lalu di jendela lain:

```powershell
curl.exe -i http://127.0.0.1:8080/health
```

**Di repo sekarang:** handler `/health` sudah dibuat dari kontrak (`GetHealth` di `internal/server/health.go`) dan juga memeriksa database. Itu terjadi di tahap 2 dan 4.

### Tahap 2 · Database dan migrasi

**Tujuan:** PostgreSQL berjalan di Docker, server membuka koneksi saat mulai, dan menjalankan migrasi sendiri.

**Konsep baru:** container, `.env`, connection pool, migrasi bernomor, `embed`, constraint.

**Langkah:**
1. Tulis `compose.yaml` dengan satu layanan `db` dari image `postgres:18.6`. Port hanya dibuka untuk `127.0.0.1`, password dibaca dari `.env`, dan data disimpan di volume `pgdata` supaya tidak hilang saat container dimatikan.
2. Tulis `.env.example` (di-commit, tanpa rahasia asli) dan `.env` (tidak di-commit). Tambahkan `.env` ke `.gitignore`.
3. Buat `internal/db/db.go` dengan `Connect`: `pgxpool.ParseConfig`, `pgxpool.NewWithConfig`, lalu `Ping` untuk memastikan database bisa dihubungi. Pesan error tidak boleh memuat alamat yang berisi password.
4. Buat migrasi pertama `sql/migrations/00001_create_customers.sql`:
   ```sql
   -- +goose Up
   CREATE TABLE customers (
       id uuid PRIMARY KEY DEFAULT uuidv7(),
       code text NOT NULL,
       name text NOT NULL,
       created_at timestamptz NOT NULL DEFAULT now(),
       CONSTRAINT customers_code_key UNIQUE (code),
       CONSTRAINT customers_code_format CHECK (code ~ '^[A-Z]{2}$')
   );

   -- +goose Down
   DROP TABLE customers;
   ```
   Bagian `Up` dijalankan saat migrasi naik, `Down` hanya bila dibatalkan. Nomor di depan nama berkas menentukan urutan.
5. Masukkan migrasi ke binary dengan `//go:embed *.sql` di `sql/migrations/migrations.go`, lalu jalankan dengan goose di `internal/db/migrate.go`:
   ```go
   sqlDB := stdlib.OpenDBFromPool(pool) // goose memakai database/sql
   provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
   results, err := provider.Up(ctx) // yang sudah tercatat di goose_db_version dilewati
   ```
6. Panggil `Connect` lalu `Migrate` di `main.go` sebelum server menyala.
7. Buat bantuan test `internal/db/dbtest`: testcontainers-go menyalakan PostgreSQL sekali per paket test, dan setiap test mendapat database baru yang sudah dimigrasi.

**Cara mencoba:**

```powershell
docker compose up -d
```

```powershell
go run ./cmd/revioten
```

Log server menulis "migrasi dijalankan" untuk setiap berkas. Lalu lihat tabelnya:

```powershell
docker compose exec db psql -U revioten -d revioten -c "\d customers"
```

**Lanjutan di repo (M1.3, E-06):** server menjalankan migrasi sebagai **pemilik database**, lalu bekerja sebagai role **`revioten_app`** yang haknya sekecil mungkin. Karena itu setiap migrasi yang membuat tabel juga menulis `GRANT` untuk `revioten_app`. Lihat `openDatabase` di `cmd/revioten/main.go`.

### Tahap 3 · Halaman web pertama

**Tujuan:** SvelteKit membuat halaman, hasil build-nya dimasukkan ke binary Go, dan server melayaninya.

**Konsep baru:** SPA, routing berbasis folder, komponen Svelte, `$state`, build, embed.

**Langkah:**
1. Buat proyek SvelteKit di folder `web/`. Pasang paket persis seperti di `web/package.json` (`npm ci` memakai `package-lock.json`).
2. Atur `web/vite.config.ts`: plugin `sveltekit` dengan `adapter-static` (`pages: 'build/app'`, `fallback: '200.html'`), mode runes, dan proxy `/api` serta `/health` ke `http://127.0.0.1:8080` untuk development.
3. Tulis `web/src/routes/+layout.ts` berisi `export const ssr = false;`. Artinya halaman tidak dirender di server; semuanya di browser.
4. Tulis halaman pertama `web/src/routes/+page.svelte`:
   ```svelte
   <script lang="ts">
     let count = $state(0); // nilai yang bila berubah, layar ikut berubah
   </script>

   <h1>Revioten</h1>
   <button onclick={() => count++}>Klik: {count}</button>
   ```
5. Masukkan hasil build ke Go di `web/web.go` dengan `//go:embed all:build`. Awalan `all:` wajib, karena SvelteKit menaruh JavaScript di folder `_app`, dan tanpa `all:` folder berawalan `_` tidak ikut.
6. Layani berkas itu dari server dengan handler SPA (`internal/server/spa.go`): berkas yang ada dikirim apa adanya, alamat halaman lain dijawab `200.html`.

**Cara mencoba:**

```powershell
npm --prefix web run build
```

```powershell
go run ./cmd/revioten
```

Buka http://127.0.0.1:8080. Saat mengubah tampilan, lebih cepat memakai `npm --prefix web run dev` lalu membuka http://localhost:5173, sementara server Go tetap berjalan untuk API.

**Penting tentang SvelteKit 3:** banyak contoh di internet masih SvelteKit 2. Di proyek ini alias impor adalah `#lib/...` (diatur di `"imports"` pada `web/package.json`), bukan `$lib/...`. Bila ragu, baca dokumentasi resmi untuk versi yang terpasang.

### Tahap 4 · Halaman login

**Tujuan:** orang membuka web, diarahkan ke `/login`, mengetik nama login dan password, lalu masuk. Tanpa login, halaman lain dan API menolak.

Ini tahap terpanjang di fondasi, karena untuk pertama kalinya semua lapisan dipakai: tabel, query sqlc, layanan, kontrak OpenAPI, handler, middleware, dan halaman Svelte. Polanya akan Anda ulangi di setiap fitur.

**Konsep baru:** hash password argon2id, token sesi, cookie `HttpOnly`, transaksi, kontrak API, generator kode, middleware, `fetch`.

**Langkah 4.1 · Tabel orang, akun, dan sesi** (`sql/migrations/00002_create_people_accounts_sessions.sql`, diringkas):

```sql
-- +goose Up
CREATE TABLE people (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL
);

CREATE TABLE accounts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    person_id uuid NOT NULL REFERENCES people (id),
    kind text NOT NULL,                 -- work atau admin (D-72)
    login text NOT NULL,
    password_hash text NOT NULL,        -- hash argon2id, bukan password
    must_change_password boolean NOT NULL DEFAULT false,
    failed_logins integer NOT NULL DEFAULT 0,
    locked_until timestamptz,
    CONSTRAINT accounts_login_key UNIQUE (login),
    CONSTRAINT accounts_login_format CHECK (login ~ '^[a-z0-9._-]{3,64}$')
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,       -- hanya hash token yang disimpan
    account_id uuid NOT NULL REFERENCES accounts (id),
    created_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
```

Mengapa **orang** dan **akun** dipisah: satu orang bisa punya akun kerja dan akun admin (D-72), dan aturan "pemohon tidak memverifikasi sendiri" membandingkan orang, bukan akun.

**Langkah 4.2 · Query untuk sqlc** (`sql/queries/auth.sql`, sebagian):

```sql
-- name: GetAccountByLogin :one
SELECT a.id, a.person_id, a.kind, a.login, a.password_hash, a.must_change_password,
       a.failed_logins, a.locked_until, p.name AS person_name
FROM accounts a
JOIN people p ON p.id = a.person_id
WHERE a.login = $1;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, account_id, created_at, last_seen_at, expires_at)
VALUES ($1, $2, $3, $3, $4);
```

Komentar `-- name: Nama :one` memberi tahu sqlc nama fungsinya dan bentuk hasilnya (`:one` satu baris, `:many` banyak baris, `:exec` tanpa hasil). Lalu jalankan generator:

```powershell
go generate ./internal/db/...
```

Hasilnya fungsi Go seperti `q.GetAccountByLogin(ctx, login)`. Pengaturan sqlc ada di `sqlc.yaml`; versinya dikunci di baris `//go:generate` di `internal/db/generate.go`.

**Langkah 4.3 · Hash password dan token** (`internal/auth/password.go`, `internal/auth/token.go`):

```go
// Hash argon2id dengan salt acak 16 byte. Parameter ikut ditulis di teks hasilnya,
// misalnya $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>.
key := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)

// Token sesi: 32 byte acak, dikirim ke browser. Yang disimpan hanya SHA-256-nya.
raw := make([]byte, 32)
rand.Read(raw)
token := base64.RawURLEncoding.EncodeToString(raw)
sum := sha256.Sum256([]byte(token))
```

Mengapa password memakai argon2id tetapi token cukup SHA-256: password dipilih manusia dan bisa ditebak dari kamus, jadi hash-nya harus lambat. Token berisi 256 bit acak dan tidak bisa ditebak, jadi SHA-256 cukup.

**Langkah 4.4 · Layanan login** (`internal/auth/service.go`, `Login`, diringkas):

```go
func (s *Service) Login(ctx context.Context, login, password string) (access.Actor, string, error) {
    acc, err := s.q.GetAccountByLogin(ctx, normalizeLogin(login))
    if errors.Is(err, pgx.ErrNoRows) {
        verifyPassword(password, dummyHash()) // waktu jawab sama, supaya login yang ada tidak bisa ditebak
        return access.Actor{}, "", ErrInvalidCredentials
    }
    match, _, err := verifyPassword(password, acc.PasswordHash)
    if !match {
        // hitung gagal login; sesudah beberapa kali, akun dikunci sementara
        return access.Actor{}, "", ErrInvalidCredentials
    }
    var token string
    err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
        // nolkan hitungan gagal, buat sesi, catat audit log: semua dalam satu transaksi
        ...
    })
    return actor, token, err
}
```

`db.WithTx` (`internal/db/tx.go`) menjalankan fungsi di dalam transaksi: berhasil berarti COMMIT, error berarti ROLLBACK.

**Langkah 4.5 · Kontrak API** (`api/openapi.yaml`, diringkas):

```yaml
/api/v1/session:
  get:
    operationId: getSession       # siapa yang sedang login
  post:
    operationId: login
    security: []                  # satu-satunya alamat yang boleh tanpa login
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Credentials'
  delete:
    operationId: logout
```

Lalu buat ulang kode server dan tipe layar:

```powershell
go generate ./internal/server/...
```

```powershell
npm --prefix web run gen:api
```

**Langkah 4.6 · Handler dan cookie** (`internal/server/session.go`):

```go
func (a *apiServer) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
    actor, token, err := a.auth.Login(ctx, req.Body.Login, req.Body.Password)
    if errors.Is(err, auth.ErrInvalidCredentials) {
        return Login401JSONResponse(errorBody(ctx, ErrorCodeInvalidCredentials,
            "The login name or password is not correct.", nil)), nil
    }
    ...
    cookie := sessionCookie(token) // HttpOnly, Secure, SameSite=Lax, nama berawalan __Host-
    return Login200JSONResponse{Body: toAPISession(actor, perms), Headers: Login200ResponseHeaders{SetCookie: &cookie}}, nil
}
```

`HttpOnly` berarti JavaScript di browser tidak bisa membaca cookie itu. Browser modern menerima cookie `Secure` di `http://127.0.0.1` dan `localhost` karena dianggap aman; di jaringan, aplikasi dibuka lewat HTTPS (Caddy, tahap 15).

**Langkah 4.7 · Middleware sesi:** `requireSession` di `internal/server/session.go` berjalan untuk setiap alamat API. Alamat yang tidak tercantum di `publicRoutes` wajib punya cookie sesi yang sah; bila tidak, jawabannya 401. Aktor yang login dipasang ke `context`, dan handler mengambilnya dengan `actorFrom(ctx)`. Halaman web juga dijaga: `handleSPA` mengarahkan ke `/login?next=...` bila belum login.

**Langkah 4.8 · Akun admin pertama:** belum ada layar untuk membuat akun pertama, jadi dibuat lewat perintah `create-admin` (`cmd/revioten/createadmin.go`). Password dibaca dengan `term.ReadPassword` supaya tidak tampil dan tidak tersimpan di riwayat perintah.

**Langkah 4.9 · Klien API di browser** (`web/src/lib/api/client.ts`):

```ts
export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiFailure };

export function login(credentials: Credentials, fetchFn: typeof fetch = fetch): Promise<ApiResult<Session>> {
	return call<Session>(fetchFn, '/api/v1/session', jsonRequest('POST', credentials), false);
}
```

Fungsi `call` tidak melempar error. Hasilnya selalu `ok: true` atau `ok: false`, sehingga layar wajib menangani kegagalan. Tipe `Credentials` dan `Session` dibuat dari kontrak (`web/src/lib/api/gen/`).

**Langkah 4.10 · Halaman login** (`web/src/routes/login/+page.svelte`, diringkas):

```svelte
<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '#lib/api/client.ts';
	import { t } from '#lib/i18n/index.ts';
	import { session } from '#lib/session.svelte.ts';

	let loginName = $state('');
	let password = $state('');
	let errorText = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		const result = await login({ login: loginName, password });
		password = '';
		if (!result.ok) {
			// Pesan sengaja umum: tidak menyebut apakah nama login ada.
			errorText = t('loginFailed');
			return;
		}
		session.current = result.data;
		goto(result.data.must_change_password ? '/account/password' : '/');
	}
</script>

<h1>{t('loginTitle')}</h1>
<form onsubmit={submit}>
	<label for="login-name">{t('loginName')}</label>
	<input id="login-name" autocomplete="username" bind:value={loginName} />
	<label for="login-password">{t('loginPassword')}</label>
	<input id="login-password" type="password" autocomplete="current-password" bind:value={password} />
	<button type="submit">{t('loginSubmit')}</button>
	{#if errorText}<p role="alert">{errorText}</p>{/if}
</form>
```

`bind:value` menyambungkan isi kotak dengan variabel. `session` (`web/src/lib/session.svelte.ts`) adalah state bersama yang dibaca layout untuk menyusun menu. `+layout.svelte` memanggil `getSession()` saat halaman dibuka; bila 401, browser dipindah ke `/login`.

Di awal, Anda boleh menulis teks langsung dalam satu bahasa. Katalog dua bahasa (`t('kunci')`) bisa ditambahkan di tahap 15.

**Cara mencoba:**
1. Build web, jalankan server, lalu `create-admin`.
2. Buka http://127.0.0.1:8080: Anda diarahkan ke `/login`.
3. Login dengan password salah: pesan gagal tampil. Login benar: masuk ke halaman utama.
4. Lewat curl tanpa cookie, `curl.exe -i http://127.0.0.1:8080/api/v1/customers` dijawab 401.

**Test yang perlu ada:** password salah ditolak, akun terkunci sesudah gagal berkali-kali, password tidak pernah muncul di log dan audit log (`internal/auth/*_test.go`, `internal/server/session_test.go`).

### Tahap 5 · Akun, peran, dan izin

**Tujuan:** admin membuat akun kerja dan memberi peran. Setiap operasi diperiksa di satu tempat, dan menu hanya menampilkan yang boleh dipakai.

**Konsep baru:** hak (permission), peran (role) sebagai kumpulan hak, cakupan (scope), pemisahan akun admin dan akun kerja.

**Langkah:**
1. Tabel `roles`, `permissions`, `role_permissions`, dan `account_roles` (migrasi 00003 dan seterusnya). Hak adalah kode tetap (L-1, E-7, S-1, A-3), peran adalah data yang bisa diubah admin (D-66, D-199).
2. Satu fungsi pemeriksaan izin di `internal/access/checker.go`:
   ```go
   func (c *Checker) Authorize(ctx context.Context, actor Actor, perm Permission, res Resource) error
   ```
   Fungsi ini membaca hak akun dari database setiap kali. Hak bercakupan Project Group hanya berlaku untuk customer di Project Group orang itu (D-71).
3. Setiap layanan memanggil `Authorize` di baris pertamanya. Tidak ada pemeriksaan izin di tempat lain.
4. `GET /api/v1/session` mengembalikan daftar hak. Di layar, `can('E-11')` (`session.svelte.ts`) dan `buildMenu(can)` (`menu.ts`) menyembunyikan menu dan tombol yang tidak boleh dipakai. Ingat: menyembunyikan tombol hanya kenyamanan, server tetap memeriksa.
5. Halaman Admin → Accounts untuk membuat akun dengan password sementara yang wajib diganti saat login pertama.

**Cara mencoba:** buat akun kerja tanpa peran, login dengannya, lalu coba panggil API yang butuh hak: jawabannya 403.

### Tahap 6 · Audit log

**Tujuan:** setiap perubahan penting tercatat, dan catatan itu tidak bisa diubah siapa pun.

**Konsep baru:** append-only, trigger, hash chain.

**Langkah:**
1. Tabel `audit_log` (migrasi 00004). Role aplikasi hanya diberi `SELECT, INSERT`, tanpa `UPDATE` dan `DELETE`.
2. Trigger yang menolak UPDATE dan DELETE, juga dari pemilik database.
3. `audit.Record(ctx, q, actor, event)` menulis satu baris **di transaksi yang sama** dengan perubahan datanya. Setiap baris menyimpan hash dirinya dan hash baris sebelumnya.
4. Panggil `audit.Record` di login (tahap 4) dan di setiap operasi tulis sesudahnya.

**Cara mencoba:** jalankan perintah "Coba ubah audit log (harus gagal)" di `CLAUDE.md`, lalu tekan **Check the chain** di Admin → Audit log.

### Tahap 7 · Fitur pertama: customer dan model

**Tujuan:** satu irisan penuh, dari tabel sampai layar, untuk data yang sederhana. Inilah latihan terpenting: bila Anda bisa menulis irisan ini tanpa melihat, semua fitur lain hanya pengulangan dengan aturan yang berbeda.

**Langkah:** ikuti resep di bagian 7 dengan data customer:
1. Migrasi: tabel `customers` dengan CHECK kode dua huruf dan GRANT untuk `revioten_app`.
2. Query `ListCustomers` dan `CreateCustomer` di `sql/queries/customers.sql`.
3. Test layanan dulu (`internal/customer/*_test.go`): kode salah ditolak, kode kembar ditolak, akun tanpa hak E-11 ditolak, audit log tertulis.
4. Layanan `internal/customer`: izin → validasi → transaksi (simpan + audit) → baca ulang.
5. Kontrak `GET` dan `POST /api/v1/customers`, generate, lalu handler di `internal/server/customers.go`.
6. Klien `listCustomers` dan `createCustomer`, halaman `web/src/routes/customers/+page.svelte`, dan menu di `menu.ts`.
7. Test E2E: login, buka menu, tambah customer, lihat di daftar.

Sesudah itu tambahkan Project Group (D-54), lalu model dan part number (`internal/product`).

### Tahap 8 · Master data lain

Proses (nama hanya tampilan, ID tetap, alias), jenis dokumen dan pola nomor, jenis dan sumber perubahan, jabatan dan pemegangnya, rute approval, serta pengaturan dokumen. Semuanya milik PPAP Engineer lewat hak E-11 (D-209). Setiap master dibangun dengan resep yang sama.

**Konsep baru yang muncul:** ID tetap dan nama sebagai tampilan (D-06, D-58), riwayat kode, dan data yang dinonaktifkan, tidak dihapus.

### Tahap 9 · Nomor dokumen

**Tujuan:** daftarkan nomor PFD, PFMEA, dan CP sebagai satu set, tanpa nomor kembar dan tanpa lubang, walau 50 orang menekan bersamaan.

**Konsep baru:** baris penghitung dalam transaksi, kunci baris, kunci idempoten, test konkurensi. Rinciannya di bagian 8.5. Sesudahnya: register dan status (Reserved, In use, Released, Void, Obsolete), lalu impor nomor lama.

### Tahap 10 · Template

**Tujuan:** format template (kolom setiap jenis dokumen), form set per customer, dan process template (poin PFMEA dan CP per proses), masing-masing dengan versi draft dan terbit.

**Konsep baru:** data yang menentukan bentuk data lain (kolom ber-ID tetap, R-TPL-02), versi terbit yang dikunci trigger (migrasi 00016), dan penyetuju yang bukan penyusun (D-139). Alurnya digambar di U4-E1 dan U4-E2.

### Tahap 11 · Dokumen dan baris

**Tujuan:** dokumen dibuat dari nomor Reserved; barisnya berisi JSONB yang divalidasi terhadap versi format; PFD diisi langsung, lalu PFMEA dan CP dirakit dari langkah PFD.

**Konsep baru:** JSONB, penguncian optimistik (409 bila dua orang mengubah baris yang sama), komponen tabel tabulator, dan diagram xyflow. Rinciannya di bagian 8.6.

### Tahap 12 · Perubahan dan revisi

**Tujuan:** permintaan perubahan, verifikasi oleh orang lain, draft revisi berikutnya, rilis yang membekukan isi dengan SHA-256, dan diff antar revisi.

**Konsep baru:** pemisahan tugas (CHECK di database), indeks unik bersyarat (satu draft per dokumen), snapshot. Rinciannya di bagian 8.7, alurnya di U4-D1 dan U4-D2.

### Tahap 13 · Cetak dan tanda tangan

**Tujuan:** pratinjau, PDF, dan Excel dari satu rencana halaman; rilis dengan scan bertanda tangan basah; tanda tangan digital berurutan.

**Konsep baru:** satu model cetak untuk dua keluaran, browser headless sebagai mesin PDF, dan draft yang dibekukan sebelum ditandatangani. Rinciannya di bagian 8.8, alurnya di U4-A3 dan U4-B.

### Tahap 14 · Migrasi dokumen lama dan import Excel

**Tujuan:** dokumen lama masuk di revisinya yang sekarang, diverifikasi orang kedua terhadap hardcopy; akun, customer, model, dan nomor lama bisa diimpor dari Excel dengan **Preview** lalu **Import**.

**Konsep baru:** baseline migrasi, empat mata, dan import yang memakai layanan biasa (aturan dan audit log sama dengan mengisi layar satu per satu). Alurnya di U4-C1 dan U4-C2.

### Tahap 15 · Penguatan

- **Row-level security:** database menyaring dokumen per akun (bagian 8.4).
- **Dua bahasa dan tampilan per perangkat:** katalog `en.ts` dan `id.ts`, menu dilipat di bawah 1024px (bagian 8.10).
- **Help, tur, dan gambar Help:** buku panduan di `web/src/lib/help/`.
- **Email notifikasi:** pendengar audit log di `internal/notify`.
- **Backup terenkripsi dan restore:** `go run ./cmd/revioten backup`.
- **Deploy lokal:** `compose.deploy.yaml` dengan Caddy (HTTPS).

Saran untuk berlatih: jangan menyalin seluruh proyek. Kuasai tahap 1 sampai 7 dulu sampai bisa ditulis tanpa melihat. Sesudah itu, tahap 8 sampai 15 bisa dikerjakan dalam urutan yang Anda butuhkan, asalkan prasyaratnya ada (misalnya dokumen butuh nomor dan template).

---

## 7. Resep membangun satu fitur (contoh: Form Control Submission, D-207)

Resep ini dipakai mulai tahap 7. Setiap fitur di proyek ini dibangun dengan urutan yang sama. Contohnya Form Control Submission (D-207), karena fitur itu menyentuh semua lapisan. Bila contoh ini terasa terlalu besar, kerjakan dulu dengan data customer (tahap 7), lalu kembali ke sini.

### Langkah 0 · Pahami dulu, tulis rencana

Sebelum menulis kode:
- Baca keputusan dan aturan yang disentuh (di sini D-207, D-198 untuk tanda tangan, hak E-7).
- Tulis rencana singkat: tabel apa, siapa boleh apa, kolom mana dihitung dan mana diketik. Untuk D-207 rencananya ada di `docs/rencana/submission-control.md`.
- Tentukan pertanyaan yang harus dijawab pemilik, dan jangan menebak jawabannya.

**Mengapa:** fitur yang salah paham dari awal mahal diperbaiki sesudah ada tabel, API, dan layar.

### Langkah 1 · Migrasi database

Buat berkas baru di `sql/migrations/` dengan nomor berikutnya. Goose menjalankan bagian `Up` saat server mulai, dan `Down` hanya bila dibatalkan.

```sql
-- sql/migrations/00052_submission_control.sql (diringkas)
-- +goose Up
CREATE TABLE document_submissions (
    revision_id uuid PRIMARY KEY REFERENCES document_revisions (id),
    document_id uuid NOT NULL REFERENCES documents (id),
    submitted_on date,
    result text NOT NULL DEFAULT 'waiting'
        CONSTRAINT document_submissions_result_check CHECK (result IN ('waiting', 'approved', 'rejected')),
    result_on date,
    remark text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Tanggal hasil hanya untuk hasil yang sudah diputuskan.
    CONSTRAINT document_submissions_result_date_check CHECK (result_on IS NULL OR result IN ('approved', 'rejected'))
);

-- Hak sekecil mungkin untuk role aplikasi (E-06): boleh baca dan tambah,
-- boleh ubah kolom tertentu saja, tidak boleh DELETE.
GRANT SELECT, INSERT ON document_submissions TO revioten_app;
GRANT UPDATE (submitted_on, result, result_on, remark, updated_at) ON document_submissions TO revioten_app;

-- Disaring seperti dokumennya (RLS, M8.1).
ALTER TABLE document_submissions ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_submissions_customer_scope ON document_submissions TO revioten_app
    USING (EXISTS (SELECT 1 FROM documents d WHERE d.id = document_submissions.document_id));

-- +goose Down
DROP TABLE document_submissions;
```

Hal yang perlu diperhatikan:
- **Satu baris per revisi** dijamin `PRIMARY KEY (revision_id)`. Aturan "harus unik" lebih aman di database daripada di Go.
- **CHECK** menolak data yang tidak masuk akal, misalnya tanggal hasil tanpa hasil.
- **Nama constraint harus unik di seluruh skema.** Di D-207 nama `document_submissions_result_check` sempat dipakai dua kali dan migrasi gagal. Beri nama yang menjelaskan isinya.
- **GRANT per kolom** membuat role aplikasi tidak bisa mengubah kolom kunci, misalnya `revision_id`.
- **RLS:** policy di atas memakai tabel `documents`, yang sudah tersaring per akun. Akibatnya, baris submission ikut tersaring tanpa aturan baru (lihat bagian 8.4).

Coba: jalankan server sekali. Bila migrasi salah, server berhenti dengan pesan SQL-nya.

### Langkah 2 · Query SQL untuk sqlc

Tulis query di `sql/queries/submissions.sql`. Setiap query diberi nama dan jenis hasilnya (`:one`, `:many`, `:exec`).

```sql
-- name: SubmissionRows :many
SELECT r.id AS revision_id, r.revision, r.status, s.submitted_on, s.result, s.remark
FROM document_revisions r
LEFT JOIN document_submissions s ON s.revision_id = r.id
WHERE r.document_id = $1
ORDER BY r.revision;

-- name: LockSubmission :one
-- Membuat baris bila belum ada, lalu mengunci baris itu sampai transaksi selesai.
INSERT INTO document_submissions (revision_id, document_id, updated_by_person_id)
VALUES ($1, $2, $3)
ON CONFLICT (revision_id) DO UPDATE SET updated_at = document_submissions.updated_at
RETURNING revision_id;
```

Lalu buat kode Go-nya:

```powershell
go generate ./internal/db/...
```

sqlc membaca skema dari migrasi dan query di atas, lalu menulis fungsi Go bertipe, misalnya `q.SubmissionRows(ctx, documentID)` yang mengembalikan `[]db.SubmissionRowsRow`. Kode hasil generate tidak pernah diedit tangan; bila salah, perbaiki SQL-nya lalu generate ulang.

Pelajaran dari D-207:
- Kolom dari `LEFT JOIN` bisa kosong (NULL). Bila sqlc memberi tipe tidak-null, pakai `coalesce(...)` atau `sqlc.narg(...)`, atau ubah bentuk query.
- `ON CONFLICT DO UPDATE` butuh hak UPDATE pada kolom yang di-SET. Karena itu `LockSubmission` meng-SET `updated_at`, kolom yang memang boleh diubah role aplikasi.

### Langkah 3 · Test layanan domain (tulis test dulu)

Buat `internal/submission/submission_test.go` sebelum kodenya. Test memakai database sungguhan di container (`internal/db/dbtest`), jadi constraint, trigger, dan RLS ikut teruji.

Kasus yang ditulis untuk D-207, misalnya:
- dokumen terbit pertama menampilkan satu baris berisi alasan "First release" dan tanggal rilis;
- akun tanpa hak E-7 ditolak saat menyimpan (`access.ErrForbidden`);
- tanggal hasil tanpa hasil ditolak (`problem.ErrInvalid`);
- akun Project Leader Project Group lain tidak melihat dokumennya (RLS);
- setiap penyimpanan menulis satu entri audit log.

Jalankan test sampai **gagal** dulu (karena fungsinya belum ada), baru tulis kode.

```powershell
go test ./internal/submission/
```

**Mengapa test dulu:** test menjadi definisi "selesai". Bila nanti mengubah kode, test yang sama memberi tahu bila ada yang rusak.

### Langkah 4 · Layanan domain

Pola setiap operasi tulis di proyek ini sama:

```go
// internal/submission/submission.go (diringkas)
func (s *Service) Save(ctx context.Context, actor access.Actor, documentID string, revision int, in Entry) (Control, error) {
	// 1. Cari resource-nya, supaya izin bisa dicek per customer.
	customerID, err := s.customerOf(ctx, documentID)
	if err != nil {
		return Control{}, err
	}
	// 2. Satu titik pemeriksaan izin (bagian 8.2).
	if err := s.authz.Authorize(ctx, actor, access.RecordRelease, access.Resource{CustomerID: customerID}); err != nil {
		return Control{}, err
	}
	// 3. Validasi isian, sebelum menyentuh database.
	params, err := clean(in)
	if err != nil {
		return Control{}, err // problem.ErrInvalid dengan pesan untuk pengguna
	}
	// 4. Semua perubahan dan audit log dalam satu transaksi.
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		rev, err := q.RevisionOfDocument(ctx, ...)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if _, err := q.LockSubmission(ctx, ...); err != nil {
			return err
		}
		if err := q.SaveSubmission(ctx, params); err != nil {
			return err
		}
		// 5. Audit log di transaksi yang sama: gagal satu, batal semua.
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionSubmissionChanged, TargetType: audit.TargetDocument, TargetID: documentID,
			Details: map[string]string{"revision": strconv.Itoa(revision)},
		})
	})
	if err != nil {
		return Control{}, err
	}
	// 6. Kembalikan keadaan terbaru, supaya layar langsung tampil benar.
	return s.load(ctx, documentID)
}
```

Urutan **izin → validasi → transaksi (ubah + audit) → baca ulang** dipakai di semua paket domain. Bila ragu, buka layanan lain dan bandingkan.

Kesalahan dikembalikan sebagai nilai dari `internal/problem` (`ErrNotFound`, `ErrInvalid`, `ErrConflict`) atau `access.ErrForbidden`. Handler menerjemahkannya ke HTTP 404, 400, 409, dan 403.

### Langkah 5 · Kontrak OpenAPI

Tambahkan path dan skema di `api/openapi.yaml`:

```yaml
/api/v1/documents/{documentId}/submission-control/{revision}:
  put:
    operationId: saveSubmissionEntry
    summary: Save the typed columns of one revision (E-7, D-207)
    parameters:
      - $ref: '#/components/parameters/DocumentId'
      - name: revision
        in: path
        required: true
        schema:
          type: integer
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/SubmissionEntry'
    responses:
      '200':
        description: The form after saving.
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SubmissionControl'
      default:
        $ref: '#/components/responses/Error'
```

`operationId` menjadi nama method Go (`SaveSubmissionEntry`) dan nama tipe TypeScript. `default: Error` berarti semua kegagalan memakai satu format error (R-API-13). Lalu buat ulang kode server dan tipe frontend:

```powershell
go generate ./internal/server/...
```

```powershell
npm --prefix web run gen:api
```

oapi-codegen menulis interface `StrictServerInterface` di `internal/server/api.gen.go`. Bila method baru belum ditulis, `go build` gagal dengan pesan bahwa `apiServer` belum punya `SaveSubmissionEntry`. Itu tanda langkah berikutnya.

### Langkah 6 · Handler

```go
// internal/server/submissions.go (diringkas)
func (a *apiServer) SaveSubmissionEntry(ctx context.Context, req SaveSubmissionEntryRequestObject) (SaveSubmissionEntryResponseObject, error) {
	c, err := a.submissions.Save(ctx, actorFrom(ctx), req.DocumentId.String(), req.Revision, fromAPIEntry(*req.Body))
	if err != nil {
		return nil, err // apierror.go mengubahnya menjadi format error R-API-13
	}
	return SaveSubmissionEntry200JSONResponse(toAPISubmission(c)), nil
}
```

Handler hanya melakukan tiga hal: ambil aktor dari sesi, ubah bentuk request menjadi bentuk domain, ubah hasil domain menjadi bentuk API. Layanan baru juga didaftarkan di `Deps` di `internal/server/server.go` dan disambungkan di `cmd/revioten/main.go`.

Test handler (`internal/server/*_test.go`) memastikan sambungan ini: status HTTP, bentuk JSON, dan error 403 untuk akun tanpa hak.

### Langkah 7 · Teks layar dua bahasa

Setiap teks di layar punya kunci yang sama di dua katalog (D-128):

```ts
// web/src/lib/i18n/en.ts
subTitle: 'Submission control',
subSave: 'Save',
subSaved: 'Revision {revision} is saved.',

// web/src/lib/i18n/id.ts
subTitle: 'Submission control',
subSave: 'Simpan',
subSaved: 'Revisi {revision} tersimpan.',
```

Istilah mengikuti glosarium U1. Nilai yang datang dari server dalam bahasa Inggris (misalnya alasan "First release") diterjemahkan di layar dengan fungsi kecil, seperti `reasonText()` di `SubmissionPanel.svelte`. Tanggal diformat menurut bahasa yang dipilih (`id-ID` atau `en-GB`), bukan bahasa browser.

`npm --prefix web test` memeriksa bahwa kedua katalog punya kunci yang sama.

### Langkah 8 · Klien API dan komponen

Pemanggilan fetch ditulis tangan (D-97) di satu berkas, `web/src/lib/api/client.ts`, memakai tipe hasil generate di `web/src/lib/api/gen/`. Fungsinya tidak melempar error; hasilnya `ApiResult`, yaitu `{ ok: true, data }` atau `{ ok: false, error }`, sehingga layar wajib menangani kegagalan.

```ts
// web/src/lib/api/client.ts
// saveSubmissionEntry memanggil PUT /api/v1/documents/{id}/submission-control/{revision} (E-7).
export function saveSubmissionEntry(documentId: string, revision: number, entry: SubmissionEntry, fetchFn: typeof fetch = fetch): Promise<ApiResult<SubmissionControl>> {
	return call<SubmissionControl>(fetchFn, `${itemPath('documents', documentId)}/submission-control/${revision}`, jsonRequest('PUT', entry));
}
```

`fetchFn` bisa diganti fetch tiruan di test Vitest (`client.spec.ts`), jadi klien bisa dites tanpa server.

Komponen Svelte 5 memakai `$state` untuk data yang berubah dan `$derived` untuk nilai turunan. Hasil aksi ditulis sebagai `role="status"` (berhasil) atau `role="alert"` (gagal); `Toaster.svelte` menyalinnya ke pop-up (D-142).

```svelte
<!-- web/src/lib/documents/SubmissionPanel.svelte (diringkas) -->
<script lang="ts">
	import { saveSubmissionEntry } from '#lib/api/client.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let message = $state('');
	let failure = $state('');

	async function save(revision: number) {
		const result = await saveSubmissionEntry(control.document_id, revision, entry);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		control = result.data;
		message = t('subSaved', { revision: String(revision) });
	}
</script>

{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p role="alert">{failure}</p>{/if}
```

Perhatikan alias impor `#lib` (SvelteKit 3), bukan `$lib` seperti di SvelteKit 2.

Gaya umum (tombol, tabel, field) sudah ada di `web/src/app.css` (D-127). `<style>` di komponen hanya untuk susunan khusus. Tombol yang butuh hak tertentu disembunyikan bila sesi tidak punya haknya, tetapi server tetap memeriksa (layar hanya kenyamanan, bukan pengaman).

Periksa tipe:

```powershell
npm --prefix web run check
```

### Langkah 9 · Test E2E

`web/e2e/m6-submission.test.ts` membuka browser sungguhan, login, membuka dokumen lewat menu (`openFromMenu`), mengisi form, dan memeriksa pesan `role="status"`. Bila halaman punya tabel lebar, tambahkan juga ke daftar di `web/e2e/responsive.test.ts`.

```powershell
npm --prefix web run test:e2e
```

Pelajaran dari sesi D-207:
- Bila dua elemen punya label mirip ("To" dan "Customer"), pakai `{ exact: true }`.
- Bila ada form lain di halaman (dialog Report a problem), batasi pencarian ke bagian yang dimaksud, misalnya `page.locator('section', { has: heading })`.

### Langkah 10 · Help, gambar Help, dan tur

- **Help:** tambahkan atau perbarui bab di `web/src/lib/help/en.ts` dan `id.ts` dengan bentuk yang sama. Nama tombol ditulis `{kunciKatalog}` supaya selalu sama dengan layar.
- **Gambar Help:** tambahkan satu `shot(...)` di `web/help-shots/help.shots.ts`, lalu jalankan `npm --prefix web run help:shots` dan `npm --prefix web run build`.
- **Tur:** tambahkan langkah di `web/tour/alur.tour.ts` dan tuliskan di `docs/panduan/skenario-alur-web.md` dengan nomor langkah yang sama.

E2E, gambar Help, dan tur memakai database `revioten_e2e` dan port 8090 yang sama, jadi jalankan satu per satu.

### Langkah 11 · Dokumentasi dan UML

- Catatan milestone di `docs/catatan/` (format di `CLAUDE.md` bagian 9).
- Temuan desain di `docs/evaluasi-desain.md`.
- Keputusan baru sebagai baris D di brief (dengan kenaikan versi dan changelog), hanya bila pemilik memutuskan.
- UML mengikuti kode (D-80): tambah kelas di U5, lalu buat ulang ERD U8 dari database:
  ```powershell
  python docs/uml/tools/u8_dari_database.py revioten
  ```
- Baris perintah baru di `CLAUDE.md` bila ada cara memakai yang baru.

### Langkah 12 · Periksa semua, commit, push

```powershell
gofmt -l .
```

```powershell
go vet ./...
```

```powershell
go test -p 1 ./...
```

```powershell
npm --prefix web run check
```

```powershell
npm --prefix web test
```

```powershell
npm --prefix web run test:e2e
```

Lalu commit kecil dengan ID di pesannya, dan push ke `main` (tanpa force-push).

```powershell
git add -A
```

```powershell
git commit -m "D-207: Form Control Submission per dokumen"
```

```powershell
git push origin main
```

Sebelum `git add -A`, lihat dulu `git status` dan pastikan `.env` tidak ikut.


---

## 8. Mekanisme inti dan penjelasannya


### 8.1 Login dan sesi (M1.1, D-132)

- Password disimpan sebagai hash argon2id (`golang.org/x/crypto/argon2`), tidak pernah sebagai teks. Saat login, password yang diketik di-hash dengan garam yang sama lalu dibandingkan.
- Sesudah login berhasil, server membuat token acak, menyimpan SHA-256 token itu di tabel sesi (`internal/auth/token.go`), dan mengirim tokennya sebagai cookie `HttpOnly`, `Secure`, `SameSite=Lax`. JavaScript di browser tidak bisa membaca cookie itu, dan pencuri isi database tidak mendapat token yang bisa dipakai.
- `requireSession` di `internal/server/session.go` memeriksa cookie setiap request, lalu memasang aktor ke `context`. Handler mengambilnya dengan `actorFrom(ctx)`.
- Password sementara dari admin wajib diganti saat login (`MustChangePassword`, E-08).
- Tanda tangan digital meminta password lagi lewat `auth.Service.ConfirmPassword`, supaya orang yang meninggalkan laptop terbuka tidak bisa ditandatangankan orang lain.

### 8.2 Satu titik pemeriksaan izin (M1.2, D-35, D-66)

- Semua operasi memanggil `Checker.Authorize(ctx, actor, perm, resource)` di `internal/access/checker.go`. Tidak ada pemeriksaan izin yang ditulis terpisah di tempat lain.
- **Hak** adalah kode tetap (L-1, E-7, S-1, A-3, dan seterusnya) di `internal/access/access.go`.
- **Peran** adalah kumpulan hak yang disimpan sebagai data (tabel `roles` dan `role_permissions`), jadi admin bisa mengubah isinya di Admin → Roles tanpa mengubah kode.
- **Cakupan:** pemberian peran punya cakupan `group` (hanya customer di Project Group sendiri, D-71) atau `all` (D-206). `Authorize` memeriksa cakupan ini dengan `resource.CustomerID`.
- **Akun admin dan akun kerja dipisah** (D-72): akun admin hanya memegang hak A-, akun kerja tidak pernah memegang hak admin. Pemisahan ini juga dikunci trigger di database.
- **Contoh memindahkan hak (D-209):** master data semula memakai hak admin A-2. Karena hak A- tidak boleh dipegang akun kerja, hak itu diganti hak kerja baru E-11 di migrasi 00053, lalu konstanta `access.ManageMasterData` di Go diganti nilainya. Semua `Authorize(..., access.ManageMasterData, ...)` langsung mengikuti tanpa diubah satu per satu. Itulah gunanya satu titik pemeriksaan izin. Bila satu halaman perlu dibagi dua hak (seperti Settings: pengaman untuk admin, pengaturan dokumen untuk PPAP Engineer), hak diperiksa per kelompok isian (`authorizeUpdate` di `internal/settings`).
- **Aturan yang tidak bisa dimatikan** (R-SEC-11) dicek di layanan, bukan lewat peran: misalnya pemohon tidak memverifikasi perubahannya sendiri, dan penyusun draft tidak mencatat rilisnya.

### 8.3 Audit log append-only dengan hash chain (M1.3, R-SEC-03)

- `audit.Record(ctx, q, actor, event)` menulis satu baris di transaksi yang sama dengan perubahan datanya.
- Setiap baris menyimpan `hash` = SHA-256 dari isi baris ditambah `prev_hash` (hash baris sebelumnya). Baris pertama menyambung ke `genesisHash`. Bila satu baris lama diubah diam-diam, hash-nya tidak cocok lagi dan semua baris sesudahnya ikut putus.
- Tiga lapis penjaga: role aplikasi tidak punya hak UPDATE dan DELETE; trigger menolak UPDATE dan DELETE juga dari pemilik database; dan tombol **Check the chain** (`internal/audit/verify.go`) menghitung ulang rantainya.
- Password dan token tidak pernah masuk `Details` (R-SEC-14). Ada test yang membuktikannya.
- Pendengar di `internal/notify` membaca peristiwa audit untuk menulis email (D-194). Di aplikasi, email yang sasarannya tidak ditemukan dilewati supaya aksi orangnya tidak gagal. Di test dipasang mode ketat (`Strict()`), sehingga sasaran yang salah dipetakan langsung membuat test gagal.

### 8.4 Row-level security (M8.1)

- Sebelum query, `db.WithSubject(ctx, accountID)` dipasang oleh sesi. Saat koneksi diambil dari pool, `prepareConn` di `internal/db/subject.go` menjalankan `set_config('app.account_id', ...)`.
- Policy di tabel `documents`, `document_rows`, dan `document_revisions` membaca nilai itu dan hanya mengembalikan baris yang boleh dilihat akun tersebut. Query baru ke tabel-tabel itu otomatis tersaring.
- Tabel turunan (contoh `document_submissions`) cukup diberi policy `EXISTS (SELECT 1 FROM documents ...)`: karena `documents` sudah tersaring, turunannya ikut tersaring.
- Operasi yang memang lintas customer dan sudah lolos izin (misalnya dasbor admin) memakai `db.AsSystem(ctx)`, dengan komentar alasannya.
- **Mengapa di database:** bila sebuah query lupa menyaring customer, database tetap tidak mengembalikan dokumen orang lain.

### 8.5 Penomoran dokumen tanpa lubang (M2.1, M2.2, R-NUM-03)

- Nomor diambil dari baris penghitung `number_counters` per customer dan jenis dokumen, di dalam transaksi:
  ```sql
  INSERT INTO number_counters (customer_code, document_type_id, last_value)
  VALUES (sqlc.arg(customer_code), sqlc.arg(document_type_id), sqlc.arg(first_value)::integer)
  ON CONFLICT (customer_code, document_type_id)
  DO UPDATE SET last_value = number_counters.last_value + 1
  RETURNING last_value;
  ```
- `ON CONFLICT DO UPDATE` mengunci baris penghitung. Transaksi kedua menunggu sampai transaksi pertama selesai, sehingga tidak ada dua nomor sama.
- Mengapa bukan `SEQUENCE` PostgreSQL: sequence tidak ikut dibatalkan saat transaksi gagal, jadi meninggalkan lubang. Baris penghitung ikut dibatalkan.
- Nomor disimpan sebagai teks dan tidak dihitung ulang (R-NUM-02). Unique constraint menjadi penjaga terakhir, dan kunci idempoten mencegah satu klik ganda mendapat dua nomor.
- Test konkurensi menjalankan 50 pendaftaran paralel dan memeriksa hasilnya 50 nomor berbeda yang berurutan.

### 8.6 Baris dokumen JSONB dan format template (D-02, M3.1, M4.1)

- Isi satu baris dokumen disimpan di kolom JSONB, dengan kunci = ID kolom format template yang tetap (R-TPL-02). Nama kolom hanya tampilan, jadi mengganti label tidak merusak data lama.
- Setiap penyimpanan divalidasi terhadap versi format template yang dipakai dokumen.
- Versi template yang sudah terbit tidak bisa diubah: trigger di migrasi 00016 menolak UPDATE (SQLSTATE 55000).
- Penguncian optimistik (M4.3): setiap baris punya nomor versi. Bila dua orang mengubah baris yang sama, penyimpan kedua mendapat 409 dan isi terbaru dimuat.

### 8.7 Revisi terbit, snapshot, dan SHA-256 (M5.4, M5.5, R-REV-02)

- Saat rilis dicatat, seluruh isi dokumen (header, bagian, baris) disalin ke kolom `snapshot` di `document_revisions`, dan SHA-256 dari snapshot itu disimpan.
- Trigger menolak perubahan revisi yang berstatus `released`. Diff antar revisi (M5.6) membandingkan snapshot.
- Halaman verifikasi (`/verify?id=`) menghitung ulang hash untuk membuktikan isi cetakan sama dengan yang dirilis.

### 8.8 Tanda tangan digital dan draft beku (D-198, D-203, D-204)

- Mengirim untuk tanda tangan membuat `signature_requests` beserta `signature_slots` (satu per slot rute) dan menghitung SHA-256 isi draft.
- Selama permintaan terbuka, trigger di `document_rows`, `document_sections`, dan `documents` menolak perubahan isi draft. Ini menjamin yang ditandatangani sama dengan yang dibaca penanda tangan.
- Urutan slot berurutan (D-49): slot berikutnya baru bisa ditandatangani bila slot sebelumnya selesai.
- Tabel `signatures` append-only (trigger menolak UPDATE dan DELETE).
- Sesudah semua slot lengkap, `document_revisions.signature_request_id` diisi, sehingga rilis boleh dicatat tanpa scan. Tanggal rilis adalah tanggal dikirim (D-204).

### 8.9 Format error dan request ID (R-API-13)

- Setiap request diberi request ID (`internal/server/requestid.go`).
- `apierror.go` mengubah error domain menjadi satu bentuk JSON: `code`, `message`, `request_id`. Error tak terduga menjadi 500 dengan pesan umum; detail teknisnya hanya ditulis ke log server dan tabel `error_reports` (D-205), tidak pernah ke pengguna.

### 8.10 Tampilan, menu, dan dua bahasa (D-127, D-128)

- Menu atas disusun di `web/src/lib/menu.ts`, menurut kelompok dan hak. Halaman baru ditambahkan di sana.
- Di bawah 1024px menu dilipat ke tombol Menu. `web/e2e/responsive.test.ts` memeriksa halaman tidak bergeser ke samping di lebar HP.
- Fungsi `t('kunci')` mengambil teks dari katalog bahasa yang dipilih pengguna.


---

## 9. Daftar periksa satu fitur


Salin daftar ini ke catatan milestone dan centang satu per satu.

- [ ] Keputusan dan aturan yang disentuh sudah dibaca; rencana disetujui pemilik
- [ ] Migrasi: tabel, CHECK, nama constraint unik, GRANT sekecil mungkin, RLS bila memuat isi dokumen, `Down`
- [ ] Query di `sql/queries/`, lalu `go generate ./internal/db/...`
- [ ] Test layanan ditulis dulu dan gagal
- [ ] Layanan: izin → validasi → transaksi (ubah + `audit.Record`) → baca ulang
- [ ] Peristiwa audit baru: konstanta aksi, dan bila perlu email: baris `notification_events`, `case` di `dispatch`, teks dua bahasa
- [ ] Test layanan lulus, termasuk akses ditolak dan lintas customer
- [ ] Kontrak di `api/openapi.yaml`, lalu `go generate ./internal/server/...` dan `npm --prefix web run gen:api`
- [ ] Handler tipis dan test handler
- [ ] Teks layar di `en.ts` dan `id.ts`
- [ ] Klien fetch, komponen, pesan `role="status"` / `role="alert"`
- [ ] Menu di `menu.ts` bila halaman baru; daftar di `responsive.test.ts` bila tabel lebar
- [ ] Test E2E
- [ ] Bab Help dua bahasa, gambar Help, langkah tur dan skenario
- [ ] Catatan milestone, temuan desain, UML U5 dan U8, baris perintah di `CLAUDE.md`
- [ ] `gofmt`, `go vet`, semua test, lalu commit dengan ID dan push ke `main`


---

## 10. Kesalahan yang sering muncul


| Gejala | Sebab | Perbaikan |
|---|---|---|
| Server berhenti saat mulai dengan pesan "relation already exists" atau "constraint already exists" | Nama tabel atau constraint sudah dipakai | Pakai nama lain yang unik di seluruh skema |
| `permission denied for table ...` di test | Lupa `GRANT` untuk `revioten_app`, atau query mengubah kolom yang tidak diberi UPDATE | Tambahkan GRANT yang sempit di migrasi, atau ubah query |
| Query mengembalikan nol baris padahal data ada | RLS menyaring karena `db.WithSubject` tidak terpasang, atau akun memang tidak berhak | Periksa sesi; untuk operasi sistem pakai `db.AsSystem` dengan alasan |
| `go build` gagal: `apiServer does not implement StrictServerInterface` | Kontrak sudah ditambah, handler belum | Tulis method handler-nya |
| Tipe sqlc tidak-null untuk kolom yang bisa kosong | Kolom dari `LEFT JOIN` atau subquery | `coalesce(...)`, `sqlc.narg`, atau ubah query |
| Test E2E: "strict mode violation, resolved to 2 elements" | Locator cocok dengan dua elemen | `{ exact: true }`, atau batasi ke `section` tertentu |
| Test Help gagal sesudah mengubah teks Help | Tampilan belum di-build ulang | `npm --prefix web run build`, lalu ulangi |
| Test database gagal semua | Docker tidak menyala | Nyalakan Docker Desktop, lalu `docker compose up -d` |
| Teks layar masih bahasa Inggris di mode Indonesia | Teks ditulis langsung di komponen, atau nilai dari server tidak diterjemahkan | Pindahkan ke katalog; terjemahkan nilai server di layar |

Jangan pernah menjalankan `docker compose down -v` kecuali memang ingin menghapus seluruh isi database.


---

## 11. Latihan tulis tangan


Kerjakan tanpa AI, di branch latihan atau di folder terpisah.

1. **Irisan kecil:** tambahkan kolom `note` (teks, boleh kosong, maksimal 200 karakter) ke Project Group. Tulis migrasi dengan GRANT UPDATE per kolom, query, test layanan (termasuk batas 200 karakter), kontrak, handler, field di layar, dan teks dua bahasa. Periksa perubahan tercatat di audit log.
2. **Penjaga database:** di psql sebagai `revioten_app`, coba `DELETE FROM document_submissions;`. Jelaskan dengan kata-kata sendiri mengapa ditolak, dan di baris migrasi mana penolakannya berasal.
3. **Hash chain:** baca `internal/audit/verify.go`, lalu tulis di kertas: bila baris audit ke-5 diubah, baris mana saja yang dilaporkan putus, dan mengapa.
