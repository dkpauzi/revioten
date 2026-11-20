# Ulasan · Pindah ke Laravel dan React?

- **Dibuat:** 21 November 2026, dari pertanyaan pemilik: IT memakai Laravel (dengan Spatie untuk peran dan hak) untuk backend dan React untuk frontend.
- **Status:** ulasan untuk keputusan pemilik. Tidak ada kode yang diubah.
- **Menyentuh keputusan:** brief bagian 7 (stack, [DECIDED]), D-33 (IT membangun versinya sendiri dari UML), D-36 dan D-37 (kode ini demo dan latihan), D-12 (pemilik ingin bisa membangun ulang sendiri).

## 1. Ringkasan

**Saran: jangan menulis ulang demo ini sekarang.** Spesifikasi dan UML yang diserahkan ke IT tidak bergantung pada bahasa, jadi IT tetap bisa membangunnya dengan Laravel dan React (D-33). Yang lebih berguna: satu lampiran **peta Go/Svelte → Laravel/React** untuk IT, dan bila Anda ingin belajar stack IT, satu **percobaan kecil** di Laravel untuk bagian yang paling berisiko.

## 2. Kenapa tidak menulis ulang sekarang

| Pertimbangan | Penjelasan |
|---|---|
| Kode tidak diserahkan ke IT (R-SEC-05, D-33) | IT membangun dari UML dan spesifikasi, bukan dari kode ini. Menulis ulang tidak mengurangi pekerjaan IT |
| Biaya | 51 migrasi, 27 paket Go dengan test, sekitar 60 layar, 67 test E2E. Menulis ulang berarti mengulang Tahap 1 sampai fase 4 |
| Tujuan demo (D-36) | Mematangkan konsep dan alur kerja. Itu sudah tercapai dengan stack sekarang |
| Yang hilang sementara | Selama menulis ulang, tidak ada fitur baru dan demo ke pengguna berhenti |

**Alasan yang mendukung pindah** tetap nyata: bila Anda ingin bisa membangun ulang sistem sendiri (D-12) dan bekerja bersama IT, menguasai Laravel dan React lebih berguna daripada Go dan Svelte. Itu keputusan Anda; lihat opsi di bagian 5.

## 3. Peta komponen untuk IT

| Bagian di demo | Padanan di Laravel / React | Catatan penting |
|---|---|---|
| Go `net/http` + kontrak OpenAPI | Route, Controller, Form Request; kontrak tetap `api/openapi.yaml` | Kontrak OpenAPI bisa dipakai IT apa adanya |
| sqlc + pgx (SQL ditulis tangan) | Eloquent untuk data biasa; Query Builder atau SQL mentah untuk penomoran, laporan, dan RLS | Penomoran (R-NUM-03) wajib `DB::transaction` + `lockForUpdate()` pada baris penghitung, bukan auto-increment |
| goose (migrasi) | Laravel migrations | Trigger, RLS, dan hash chain ditulis dengan `DB::statement` (SQL mentah); Schema Builder tidak punya padanannya |
| Peran dan hak (D-66, D-199) | **spatie/laravel-permission**: `permissions` = katalog L-, E-, V-, S-, G-, A-; `roles` = PPAP Engineer, Supervisor, Project Leader, Manager, Viewer, Admin | Cakupan Project Group (D-71, D-206) tidak ada di Spatie bawaan: pakai fitur **teams** Spatie atau Policy sendiri yang memeriksa jabatan di Project Group. Aturan R-SEC-11 (pemohon bukan verifikator, satu orang satu slot) tetap di Policy/Service, bukan di Spatie |
| Audit log append-only + hash chain (R-SEC-03) | Tabel sendiri + trigger PostgreSQL | **spatie/laravel-activitylog** tidak append-only dan tidak punya hash chain; boleh dipakai untuk tampilan, tetapi penjaganya tetap trigger di database |
| Row-level security (M8.1) | Middleware yang menjalankan `SET LOCAL app.account_id` per request, kebijakan RLS tetap di PostgreSQL | Hanya bisa bila IT memakai **PostgreSQL**. Dengan MySQL: tidak ada RLS dan tidak ada partial unique index; semua penyaringan pindah ke Global Scope Eloquent |
| River (job queue) | Laravel Queue dengan driver `database` | Tanpa Redis, sama dengan aturan demo |
| argon2id | `Hash::make` dengan driver `argon2id` (`config/hashing.php`) | Butuh PHP dengan dukungan Argon2 |
| excelize (Excel) | maatwebsite/laravel-excel (PhpSpreadsheet) | |
| PDF lewat Chrome/Edge | spatie/browsershot (Chrome headless) | Sama-sama butuh Chrome di server |
| Email Microsoft 365 (D-194) | Laravel Mail, transport Microsoft Graph atau SMTP | Pola kotak keluar (outbox) tetap disarankan |
| Sesi dan login | Laravel session + Sanctum (SPA) | Cookie sesi + CSRF, sama dengan demo |
| API integrasi + token (Tahap 9) | Sanctum personal access token, atau Passport bila butuh OAuth | |
| SvelteKit SPA | React + Vite (React Router), atau Inertia.js bila IT memakainya | Pilihan IT; kontrak API sama |
| Tabulator, diagram PFD | Tabulator (ada pembungkus React), **React Flow** (xyflow, aslinya untuk React) | elkjs tetap sama |
| Teks dua bahasa | react-i18next dengan katalog `en` dan `id` yang sama | Katalog `en.ts` dan `id.ts` bisa dipindah hampir apa adanya |
| Tipe dari OpenAPI | @hey-api/openapi-ts, sama | |

## 4. Risiko yang perlu diperhatikan IT

1. **Database.** Banyak penjaga demo hidup di PostgreSQL: trigger penolak, RLS, partial unique index, advisory lock. Bila IT memakai MySQL, semua itu perlu dirancang ulang di aplikasi. Pertanyaan pertama ke IT: database apa yang dipakai?
2. **Penjaga di database vs di kode.** Di Laravel, mudah sekali menulis `Model::update()` yang melewati aturan. Aturan "revisi terbit tidak bisa diubah", "tanda tangan tidak bisa diubah", dan "audit log tidak bisa diubah" sebaiknya tetap dijaga trigger.
3. **Spatie dan cakupan.** Spatie menjawab "punya hak apa", bukan "untuk customer mana". Cakupan Project Group dan Project Leader pembantu (D-206) harus dirancang sendiri.
4. **Kunci aplikasi (APP_KEY).** Bila "masalah key" yang Anda maksud adalah `APP_KEY` Laravel: kunci ini mengenkripsi cookie dan data terenkripsi. Bila hilang atau diganti tanpa prosedur, sesi dan data terenkripsi tidak bisa dibuka. Simpan seperti `BACKUP_PASSPHRASE`, dan gunakan `APP_PREVIOUS_KEYS` saat rotasi. (Belum dipastikan bahwa ini masalah yang dimaksud.)

## 5. Opsi untuk pemilik

| Opsi | Isi | Biaya | Cocok bila |
|---|---|---|---|
| **A (saran)** | Demo tetap Go/Svelte. Tambah peta di bagian 3 ke paket untuk IT | Kecil | Tujuan utama tetap mematangkan desain untuk IT |
| B | Percobaan kecil di Laravel + React: penomoran paralel, peran Spatie dengan cakupan Project Group, dan satu layar | Satu fase | Anda ingin belajar stack IT dan menguji bagian paling berisiko |
| C | Menulis ulang seluruh demo ke Laravel + React | Sangat besar | Demo ini akan dipakai sungguhan oleh IT (bertentangan dengan D-33) |

Opsi C mengubah keputusan [DECIDED] di brief bagian 7, jadi perlu keputusan baru (D-) bila dipilih.
