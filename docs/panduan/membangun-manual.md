# Panduan membangun Revioten secara manual

- **Untuk siapa:** pemilik proyek, yang ingin bisa membangun ulang sistem ini sendiri tanpa AI (D-12).
- **Dibuat:** 21 November 2026, sesudah D-207 (migrasi 00052).
- **Cara memakai:** baca bagian 1 sampai 4 sekali sampai paham. Sesudah itu, setiap kali membuat fitur, ikuti bagian 5 langkah demi langkah dan centang daftar di bagian 8. Bagian 6 menjelaskan mekanisme yang dipakai di banyak tempat; bacalah saat bertemu mekanismenya. Bagian 7 adalah urutan membangun dari nol, dengan catatan milestone sebagai bahan rinci.

Dokumen lain yang terkait:
- `CLAUDE.md`: aturan kerja dan daftar perintah yang sudah terbukti.
- `docs/catatan/`: satu catatan per milestone, berisi langkah, konsep, dan latihan.
- `docs/panduan/skenario-alur-web.md`: alur memakai web dari sisi pengguna (langkah 1 sampai 40).
- `docs/panduan/panduan-operasional.md`: deploy, backup, dan restore.
- `docs/versions.md`: versi setiap alat beserta sumbernya.

---

## 1. Gambaran besar

Revioten adalah satu program Go yang melayani dua hal: API JSON di `/api/v1`, dan tampilan web (SPA SvelteKit) yang sudah di-build lalu dimasukkan ke dalam binary. Datanya di PostgreSQL.

```
 Browser
   │  (halaman Svelte memanggil fetch ke /api/v1/...)
   ▼
 internal/server      ← adapter HTTP: baca request, panggil layanan, tulis JSON
   │                    (kerangkanya dibuat oapi-codegen dari api/openapi.yaml)
   ▼
 internal/<domain>    ← aturan bisnis: izin, validasi, transaksi, audit log
   │                    (contoh: internal/submission, internal/numbering)
   ▼
 internal/db          ← query SQL yang sudah diketik (dibuat sqlc dari sql/queries)
   │
   ▼
 PostgreSQL           ← tabel dari sql/migrations, ditambah trigger, GRANT, dan RLS
```

Tiga prinsip yang membuat susunan ini mudah diikuti:

1. **Kontrak lebih dulu (API-first, D-61).** Bentuk setiap endpoint ditulis di `api/openapi.yaml`. Kode Go dan tipe TypeScript dibuat dari kontrak itu, jadi server dan layar selalu sepakat soal bentuk data.
2. **Handler tipis (R-API-01).** Handler di `internal/server` tidak memuat aturan. Semua aturan ada di paket domain, supaya bisa dites tanpa HTTP.
3. **Database ikut menjaga.** Aturan yang tidak boleh dilanggar (audit log tidak bisa diubah, revisi terbit beku, akun tidak dihapus) tidak hanya dicek di Go, tetapi juga dikunci di database dengan GRANT, trigger, dan RLS. Bila ada bug di Go, database tetap menolak.

## 2. Alat yang perlu dipasang

Versi yang tepat ada di `docs/versions.md`. Jangan menulis versi dari ingatan; cek dengan perintah di `CLAUDE.md` bagian 4.

| Alat | Untuk apa |
|---|---|
| Go | Bahasa server. `go test`, `go build`, `go generate` |
| Docker Desktop (atau Docker Engine di WSL 2) | Menjalankan PostgreSQL, dan container sementara saat test |
| Node.js LTS | Hanya untuk build tampilan web dan menjalankan test frontend |
| Git | Menyimpan riwayat dan push ke GitHub |
| Chrome atau Edge | Membuat PDF, dan browser untuk Playwright |
| Editor (VS Code) | Menulis kode. Ekstensi Go dan Svelte membantu |

Alat yang dijalankan lewat `go run` atau `npm` sehingga tidak perlu dipasang sendiri: sqlc, oapi-codegen, goose (dipakai sebagai pustaka), @hey-api/openapi-ts, Vitest, Playwright.

Pemeriksaan awal:

```powershell
go version
```

```powershell
node -v
```

```powershell
docker compose version
```

## 3. Peta folder

| Folder | Isi | Contoh berkas |
|---|---|---|
| `cmd/revioten/` | Titik masuk: membaca `.env`, migrasi, menyalakan server; juga perintah `create-admin`, `backup`, `restore` | `main.go` |
| `api/` | Kontrak OpenAPI | `openapi.yaml`, `integration.yaml` |
| `sql/migrations/` | Perubahan skema berurutan (goose), 00001 sampai 00052 | `00052_submission_control.sql` |
| `sql/queries/` | Query SQL yang diberi nama, bahan sqlc | `submissions.sql` |
| `internal/db/` | Koneksi pgx, transaksi, subjek RLS, dan kode hasil sqlc (`*.sql.go`, jangan diedit) | `tx.go`, `subject.go` |
| `internal/access/` | Satu titik pemeriksaan izin | `checker.go` |
| `internal/audit/` | Audit log append-only dengan hash chain | `audit.go`, `verify.go` |
| `internal/<domain>/` | Aturan per area | `submission/`, `signature/`, `numbering/` |
| `internal/server/` | Handler HTTP; `api.gen.go` hasil oapi-codegen | `submissions.go` |
| `web/src/routes/` | Halaman SvelteKit, satu folder per alamat | `documents/[id]/+page.svelte` |
| `web/src/lib/` | Komponen, klien API, katalog teks, Help | `documents/SubmissionPanel.svelte` |
| `web/src/lib/i18n/` | Teks layar Inggris dan Indonesia | `en.ts`, `id.ts` |
| `web/src/lib/help/` | Buku panduan pengguna dua bahasa | `en.ts`, `id.ts` |
| `web/e2e/` | Test E2E Playwright | `m6-submission.test.ts` |
| `web/tour/` | Tur otomatis alur web | `alur.tour.ts` |
| `web/help-shots/` | Pemotret gambar Help | `help.shots.ts` |
| `docs/` | Brief, milestone, UML, catatan, panduan | |

## 4. Menyalakan proyek dari nol

1. Ambil repo: `git clone` lalu masuk ke foldernya.
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

---

## 5. Alur membangun satu fitur (contoh: Form Control Submission, D-207)

Setiap fitur di proyek ini dibangun dengan urutan yang sama. Contoh di bawah memakai Form Control Submission karena fitur itu menyentuh semua lapisan. Potongan kode **diringkas** supaya mudah dibaca; versi lengkapnya ada di berkas yang disebut.

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
- **RLS:** policy di atas memakai tabel `documents`, yang sudah tersaring per akun. Akibatnya, baris submission ikut tersaring tanpa aturan baru (lihat bagian 6.4).

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
	// 2. Satu titik pemeriksaan izin (bagian 6.2).
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

## 6. Mekanisme inti dan penjelasannya

### 6.1 Login dan sesi (M1.1, D-132)

- Password disimpan sebagai hash argon2id (`golang.org/x/crypto/argon2`), tidak pernah sebagai teks. Saat login, password yang diketik di-hash dengan garam yang sama lalu dibandingkan.
- Sesudah login berhasil, server membuat token acak, menyimpan SHA-256 token itu di tabel sesi (`internal/auth/token.go`), dan mengirim tokennya sebagai cookie `HttpOnly`, `Secure`, `SameSite=Lax`. JavaScript di browser tidak bisa membaca cookie itu, dan pencuri isi database tidak mendapat token yang bisa dipakai.
- `requireSession` di `internal/server/session.go` memeriksa cookie setiap request, lalu memasang aktor ke `context`. Handler mengambilnya dengan `actorFrom(ctx)`.
- Password sementara dari admin wajib diganti saat login (`MustChangePassword`, E-08).
- Tanda tangan digital meminta password lagi lewat `auth.Service.ConfirmPassword`, supaya orang yang meninggalkan laptop terbuka tidak bisa ditandatangankan orang lain.

### 6.2 Satu titik pemeriksaan izin (M1.2, D-35, D-66)

- Semua operasi memanggil `Checker.Authorize(ctx, actor, perm, resource)` di `internal/access/checker.go`. Tidak ada pemeriksaan izin yang ditulis terpisah di tempat lain.
- **Hak** adalah kode tetap (L-1, E-7, S-1, A-3, dan seterusnya) di `internal/access/access.go`.
- **Peran** adalah kumpulan hak yang disimpan sebagai data (tabel `roles` dan `role_permissions`), jadi admin bisa mengubah isinya di Admin → Roles tanpa mengubah kode.
- **Cakupan:** pemberian peran punya cakupan `group` (hanya customer di Project Group sendiri, D-71) atau `all` (D-206). `Authorize` memeriksa cakupan ini dengan `resource.CustomerID`.
- **Akun admin dan akun kerja dipisah** (D-72): akun admin hanya memegang hak A-, akun kerja tidak pernah memegang hak admin. Pemisahan ini juga dikunci trigger di database.
- **Aturan yang tidak bisa dimatikan** (R-SEC-11) dicek di layanan, bukan lewat peran: misalnya pemohon tidak memverifikasi perubahannya sendiri, dan penyusun draft tidak mencatat rilisnya.

### 6.3 Audit log append-only dengan hash chain (M1.3, R-SEC-03)

- `audit.Record(ctx, q, actor, event)` menulis satu baris di transaksi yang sama dengan perubahan datanya.
- Setiap baris menyimpan `hash` = SHA-256 dari isi baris ditambah `prev_hash` (hash baris sebelumnya). Baris pertama menyambung ke `genesisHash`. Bila satu baris lama diubah diam-diam, hash-nya tidak cocok lagi dan semua baris sesudahnya ikut putus.
- Tiga lapis penjaga: role aplikasi tidak punya hak UPDATE dan DELETE; trigger menolak UPDATE dan DELETE juga dari pemilik database; dan tombol **Check the chain** (`internal/audit/verify.go`) menghitung ulang rantainya.
- Password dan token tidak pernah masuk `Details` (R-SEC-14). Ada test yang membuktikannya.
- Pendengar di `internal/notify` membaca peristiwa audit untuk menulis email (D-194). Di aplikasi, email yang sasarannya tidak ditemukan dilewati supaya aksi orangnya tidak gagal. Di test dipasang mode ketat (`Strict()`), sehingga sasaran yang salah dipetakan langsung membuat test gagal.

### 6.4 Row-level security (M8.1)

- Sebelum query, `db.WithSubject(ctx, accountID)` dipasang oleh sesi. Saat koneksi diambil dari pool, `prepareConn` di `internal/db/subject.go` menjalankan `set_config('app.account_id', ...)`.
- Policy di tabel `documents`, `document_rows`, dan `document_revisions` membaca nilai itu dan hanya mengembalikan baris yang boleh dilihat akun tersebut. Query baru ke tabel-tabel itu otomatis tersaring.
- Tabel turunan (contoh `document_submissions`) cukup diberi policy `EXISTS (SELECT 1 FROM documents ...)`: karena `documents` sudah tersaring, turunannya ikut tersaring.
- Operasi yang memang lintas customer dan sudah lolos izin (misalnya dasbor admin) memakai `db.AsSystem(ctx)`, dengan komentar alasannya.
- **Mengapa di database:** bila sebuah query lupa menyaring customer, database tetap tidak mengembalikan dokumen orang lain.

### 6.5 Penomoran dokumen tanpa lubang (M2.1, M2.2, R-NUM-03)

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

### 6.6 Baris dokumen JSONB dan format template (D-02, M3.1, M4.1)

- Isi satu baris dokumen disimpan di kolom JSONB, dengan kunci = ID kolom format template yang tetap (R-TPL-02). Nama kolom hanya tampilan, jadi mengganti label tidak merusak data lama.
- Setiap penyimpanan divalidasi terhadap versi format template yang dipakai dokumen.
- Versi template yang sudah terbit tidak bisa diubah: trigger di migrasi 00016 menolak UPDATE (SQLSTATE 55000).
- Penguncian optimistik (M4.3): setiap baris punya nomor versi. Bila dua orang mengubah baris yang sama, penyimpan kedua mendapat 409 dan isi terbaru dimuat.

### 6.7 Revisi terbit, snapshot, dan SHA-256 (M5.4, M5.5, R-REV-02)

- Saat rilis dicatat, seluruh isi dokumen (header, bagian, baris) disalin ke kolom `snapshot` di `document_revisions`, dan SHA-256 dari snapshot itu disimpan.
- Trigger menolak perubahan revisi yang berstatus `released`. Diff antar revisi (M5.6) membandingkan snapshot.
- Halaman verifikasi (`/verify?id=`) menghitung ulang hash untuk membuktikan isi cetakan sama dengan yang dirilis.

### 6.8 Tanda tangan digital dan draft beku (D-198, D-203, D-204)

- Mengirim untuk tanda tangan membuat `signature_requests` beserta `signature_slots` (satu per slot rute) dan menghitung SHA-256 isi draft.
- Selama permintaan terbuka, trigger di `document_rows`, `document_sections`, dan `documents` menolak perubahan isi draft. Ini menjamin yang ditandatangani sama dengan yang dibaca penanda tangan.
- Urutan slot berurutan (D-49): slot berikutnya baru bisa ditandatangani bila slot sebelumnya selesai.
- Tabel `signatures` append-only (trigger menolak UPDATE dan DELETE).
- Sesudah semua slot lengkap, `document_revisions.signature_request_id` diisi, sehingga rilis boleh dicatat tanpa scan. Tanggal rilis adalah tanggal dikirim (D-204).

### 6.9 Format error dan request ID (R-API-13)

- Setiap request diberi request ID (`internal/server/requestid.go`).
- `apierror.go` mengubah error domain menjadi satu bentuk JSON: `code`, `message`, `request_id`. Error tak terduga menjadi 500 dengan pesan umum; detail teknisnya hanya ditulis ke log server dan tabel `error_reports` (D-205), tidak pernah ke pengguna.

### 6.10 Tampilan, menu, dan dua bahasa (D-127, D-128)

- Menu atas disusun di `web/src/lib/menu.ts`, menurut kelompok dan hak. Halaman baru ditambahkan di sana.
- Di bawah 1024px menu dilipat ke tombol Menu. `web/e2e/responsive.test.ts` memeriksa halaman tidak bergeser ke samping di lebar HP.
- Fungsi `t('kunci')` mengambil teks dari katalog bahasa yang dipilih pengguna.

---

## 7. Urutan membangun dari nol

Setiap baris punya catatan rinci di `docs/catatan/`. Kerjakan berurutan; setiap tahap memakai hasil tahap sebelumnya.

| Tahap | Isi | Catatan |
|---|---|---|
| 0 · Fondasi | Alat dan versi; server Go dengan `/health`; PostgreSQL dan pgx; SvelteKit pertama; irisan penuh pertama (customer dari tabel sampai layar) | M0.1 sampai M0.5 |
| 1 · Akses dan master | Login; titik izin; audit log; customer dan model; proses; jenis dokumen dan perubahan; backup; jabatan dan rute approval | M1.1 sampai M1.8 |
| 2 · Penomoran | Pola dan penghitung; alokasi atomik; register dan status; set PFD/PFMEA/CP; impor nomor lama | M2.1 sampai M2.5 |
| 3 · Template | Format template; pustaka proses; cakupan dan varian; versi terbit; mesin AP; form AIAG-VDA; form set | M3.1 sampai M3.4, FS-A, FS-BC, FS-D |
| 4 · Dokumen | Dokumen dan baris; editor tabel; penguncian optimistik; perakitan dari template; nomor operasi; salin; perbedaan lokal; PFMEA–CP berdampingan; diagram PFD; dokumen terkait; bagian per PCB | M4.1 sampai M4.10, D-130, D-155 |
| 5 · Perubahan | Perubahan dan lampiran; verifikasi; draft revisi; snapshot rilis; diff; daftar periksa dampak; propagasi template; yokoten; perubahan surut SQAM | M5.1 sampai M5.10 |
| 6 · Cetak | Uji PDF; cetak Excel dan PDF; rilis dengan scan | S5, M6 |
| 7 · Migrasi | Dokumen lama dan dasbor bukti nilai | M7 |
| 8 · Penguatan | Isi peran, RLS, antivirus, deploy Docker | M8 |
| 9 · Integrasi | API integrasi dan service account | M9 |
| Sesudahnya | Dua bahasa dan responsif; Help; ECR dan email; dokumen terbit, review 3 tahun, kinerja; matriks peran; tanda tangan digital; dasbor admin; Form Control Submission | D-128, D-141, D-193 sampai D-207 |

Saran untuk berlatih: jangan menyalin seluruh proyek. Bangun ulang **satu irisan tipis** dulu (M0.5: tabel customer → query → layanan → API → layar → test). Bila irisan itu bisa ditulis tanpa melihat, lanjut ke M1.1 sampai M1.3, karena login, izin, dan audit dipakai semua fitur lain.

---

## 8. Daftar periksa satu fitur

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

## 9. Kesalahan yang sering muncul

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

## 10. Latihan tulis tangan

Kerjakan tanpa AI, di branch latihan atau di folder terpisah.

1. **Irisan kecil:** tambahkan kolom `note` (teks, boleh kosong, maksimal 200 karakter) ke Project Group. Tulis migrasi dengan GRANT UPDATE per kolom, query, test layanan (termasuk batas 200 karakter), kontrak, handler, field di layar, dan teks dua bahasa. Periksa perubahan tercatat di audit log.
2. **Penjaga database:** di psql sebagai `revioten_app`, coba `DELETE FROM document_submissions;`. Jelaskan dengan kata-kata sendiri mengapa ditolak, dan di baris migrasi mana penolakannya berasal.
3. **Hash chain:** baca `internal/audit/verify.go`, lalu tulis di kertas: bila baris audit ke-5 diubah, baris mana saja yang dilaporkan putus, dan mengapa.
