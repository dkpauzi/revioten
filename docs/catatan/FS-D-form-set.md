# Form set langkah D · Form set, penyetuju form, dan pengubah tabel AP

- **Tanggal:** 12 November 2026
- **Langkah:** D di `docs/rencana/form-set-aiag-vda.md` (milestone 1 dan 5 di prompt form set, tanpa wizard).
- **Keputusan yang dipakai:** D-133 (form set), D-139 (penyetuju form set dan tabel AP), D-140 (urutan kerja), D-143 (penerbit template biasa), D-144 (keputusan E-89), D-66 (hak dan peran sebagai data).
- **Persetujuan pemilik:** "Lanjut langkah D" (12 November 2026).

## 1. Tujuan dan apa yang dibuat

1. **Form set.** Paket versi format template untuk tiga sheet: PFD, PFMEA, dan Control Plan. Dua set bawaan:
   - **AIAG 4th set** (bawaan sistem): PFD, PFMEA AIAG 4th, CP;
   - **AIAG-VDA set**: PFD yang sama, PFMEA AIAG-VDA 2019, Control Plan AIAG 2024.
2. **Versi form set** berjalan dari draft ke terbit seperti versi template. Versi terbit dijaga trigger yang sama (migrasi 00016).
3. **Form set per customer.** Admin memilih form set untuk dokumen baru setiap customer; tanpa pilihan berarti set bawaan.
4. **Penyetuju form (D-139).** Hak baru **V-3** dan peran baru **Form approver**. Pemegangnya menerbitkan:
   - versi form set, setelah ketiga format di dalamnya terbit;
   - versi format bermetode AP, karena tabel AP menentukan prioritas tindakan.

   Penyetuju tidak boleh orang yang menyusun drafnya. Yang dibandingkan orang, bukan akun, jadi pemegang akun admin dengan akun kerja kedua tetap ditolak.
5. **Ubah tabel AP.** Admin menekan **Change the AP table**, mengklik kotak prioritas (H, M, L bergantian), lalu **Save AP table**. Hasilnya selalu menutup 1.000 kombinasi.
6. **D-144** (keputusan E-89): kolom "2. Process Step" di PFMEA AIAG-VDA diubah menjadi kolom yang diisi dari master proses.
7. **Halaman Help** mendapat bab Form sets beserta gambarnya.

Belum dibuat: langkah "Choose the form" di wizard dokumen (menumpang wizard Tahap 4), sheet Cover (form Cover belum ada), dan aturan "semua penyetuju harus setuju" (sisa D-139, sekarang cukup satu).

| Bagian | File |
|---|---|
| Migrasi | `sql/migrations/00019_form_sets.sql` |
| Aturan | `internal/template/formset.go`; `PublishFormatVersion` di `version.go`; pencatat pengubah di `editVersion` (`service.go`) |
| Hak | `internal/access`: `ApproveForms` (V-3), `RoleFormApprover` |
| API | `listFormSets`, `createFormSet`, `setFormSetSheets`, `createFormSetVersion`, `publishFormSetVersion`, `setCustomerFormSet` |
| Layar | `web/src/routes/form-sets/+page.svelte`, `web/src/lib/formSets.ts`, `ApTable.svelte` (mode ubah), `VersionBar.svelte` (`canPublish`) |

### Tambahan 12 November 2026: aturan persetujuan bisa diatur (D-148)

Atas permintaan pemilik, sisa D-139 dibuat bisa diatur admin di **Admin → Settings → Approval of form sets and AP tables**:

- **All form approvers must approve** (bawaan): versi baru terbit setelah semua penyetuju form aktif setuju, kecuali penyusun drafnya;
- **One form approver is enough**.

Setiap persetujuan dicatat per orang di tabel `form_approvals` (migrasi 00020), yang hanya bisa ditambah. Layar menampilkan "Approved by" dan "Waiting for" di form set dan di format bertabel AP. Menyetujui dua kali ditolak. File: `internal/template/approval.go`, `internal/settings/settings.go`, `web/src/lib/templates/Approvals.svelte`. Test: `TestApprovalAll`, `TestApprovalAny`, `TestApprovalPreparerNotCounted`, `TestFormApprovalMode`, dan dua test E2E di `formsets.test.ts`.

## 2. Cara mengulang dari nol

1. Tulis `internal/template/formset_test.go` dulu: set bawaan, pilihan customer, versi dan penerbitan, form set baru, dan pemisahan tugas.
2. Tulis migrasi 00019 dan `sql/queries/form_sets.sql`, lalu `go generate ./...`.
3. Tambah hak V-3 di `internal/access` (dan di matriks test), lalu `formset.go` sampai test lulus.
4. Tambah operasi di `api/openapi.yaml`, handler di `internal/server/formsets.go`, dan baris di `TestPermissionsPerOperation`.
5. Tulis `formSets.spec.ts`, `ap.spec.ts` (grid), lalu halaman dan `web/e2e/formsets.test.ts`.

## 3. Konsep yang dipelajari

- **Pemisahan tugas (separation of duties).** Orang yang menyusun tidak menyetujui. Dicatat dengan menyimpan orang penyusun di baris versi, lalu dibandingkan saat menerbitkan.
- **Hak bergantung pada data.** Menerbitkan versi format butuh A-2 atau V-3, tergantung metode risikonya. Hak diperiksa sebelum transaksi, dari data yang dibaca lebih dulu.
- **Indeks unik sebagian untuk "hanya satu".** `form_sets_default_key` memastikan hanya satu form set bawaan.
- **Binding get/set di Svelte 5.** `bind:value={() => nilai, (x) => simpan(x)}` dipakai saat nilai diambil dari fungsi, bukan variabel.

## 4. Keputusan dan alasannya

| Keputusan | Alasan |
|---|---|
| Form set hanya bisa terbit bila ketiga formatnya sudah terbit | Isi set yang sudah disetujui tidak boleh berubah lewat format yang masih draft |
| Penyetuju format bermetode AP juga V-3, bukan admin | D-139 menyebut perubahan tabel AP butuh penyetuju |
| Satu persetujuan cukup | Asumsi kerja D-139, sampai pemilik memutuskan "semua harus setuju" |
| Konfirmasi terbit hanya di VersionBar | Konfirmasi dua kali membingungkan, dan ketahuan lewat test E2E |

## 5. Asumsi

- Sejak D-148: jumlah penyetuju diatur di Settings, bawaannya semua.
- Pengubah terakhir versi format dicatat untuk semua perubahan isi, tetapi hanya dipakai untuk versi bermetode AP.
- Kolom Cover di versi form set disiapkan dan boleh kosong.

## 6. Temuan desain baru

Tidak ada. E-73, E-81, E-88, dan E-89 diputuskan pemilik hari ini (D-144 sampai D-147).

## 7. Status test (komputer pemilik, Windows, 12 November 2026)

- `go test -p 1 ./...`: semua paket lulus.
- `npm --prefix web run check`: 0 error. Vitest: 126 test lulus. E2E: 43 test lulus. `help:shots` lulus.

## 8. Latihan tulis tangan (tanpa AI)

1. Di kertas, tulis urutan klik untuk menerbitkan set AIAG-VDA dari database kosong: akun mana, halaman mana, dan tombol mana. Lalu coba di layar.
2. Di psql, tulis query yang menampilkan setiap customer, nama form set pilihannya, atau "bawaan" bila kosong. Petunjuk: `LEFT JOIN form_sets`.
