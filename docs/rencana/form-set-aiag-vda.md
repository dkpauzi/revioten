# Rencana · Form set AIAG-VDA, multi-part, pencarian per proses, dan tampilan genba

- **Dibuat:** 10 November 2026, dari prompt pemilik `docs/design/form-set-aiag-vda/PROMPT-claude-code-form-set-aiag-vda.md` dan board desain di folder yang sama.
- **Status:** urutan disetujui pemilik pada 11 November 2026 ("E-80 A", D-140). Langkah A sampai D dikerjakan satu per satu, masing-masing dengan rencana singkat lebih dulu. Belum ada kode yang ditulis.
- **Keputusan yang sudah dicatat di brief:** D-133 (form set), D-134 (tabel AP mengikuti handbook), D-135 (fase CP 2024), D-136 (multi-part), D-137 (pencarian per proses dan tampilan genba).

## 1. Hal yang perlu Anda ketahui dulu

### 1.1 Prompt mengandaikan layar yang belum ada (temuan E-80)

Prompt menulis "desain sebelumnya sudah kamu implementasikan" dan menyebut editor workbook, wizard "Make a new document set", yokoten, cetak, dan rilis. Kode saat ini baru sampai **M3.3**. Yang sudah ada: akun, master data, register nomor, format template, dan process template. Yang belum ada:

| Disebut prompt | Milestone tempat ia dibangun |
|---|---|
| Dokumen, baris, dan editor workbook | M4.1 sampai M4.9 |
| Wizard set dokumen baru | M4.x |
| Permintaan perubahan dan yokoten | Tahap 5 |
| Cetak, ekspor, dan rilis | Tahap 6 |

Akibatnya 11 milestone di prompt tidak bisa dikerjakan berurutan sekarang. Bagian yang hanya menyentuh format template bisa dikerjakan lebih dulu; sisanya menumpang pada milestone yang sudah ada.

### 1.2 Bertentangan dengan satu item [DEFERRED] di brief

Brief bagian 9 mencatat "Full AIAG-VDA (structure, function, failure network, Action Priority)" sebagai [DEFERRED], dan NG4 membatasi AIAG-VDA pada "form compatibility". Prompt meminta form PFMEA AIAG-VDA lengkap dengan **AP yang dihitung sistem**.

Ini saya catat di D-133 sebagai penyempitan item DEFERRED itu: form dan hitungan AP masuk lingkup; **jaringan kegagalan (failure network) tetap ditunda**. Baris PFMEA tetap datar, yaitu satu baris berisi satu FE, FM, dan FC. Panel "rantai kegagalan" di board `PfmeaVda` hanya menampilkan tiga sel itu dari baris yang sama. Bila Anda menginginkan jaringan kegagalan yang sebenarnya, katakan, karena itu mengubah model data baris.

### 1.3 Yang sudah ada dan bisa dipakai

- `format_templates.methodology` sudah punya nilai `aiag_vda`, dan metode risiko sudah punya pilihan AP (M3.1).
- Kolom rating sudah punya dua set: sebelum dan sesudah tindakan.
- Alias proses sudah ada sejak M1.5 (master proses), jadi tabel `process_aliases` di prompt tidak perlu dibuat lagi.
- Aturan sorot (R-TPL-11) sudah ada untuk RPN; untuk AP tinggal ditambah.

## 2. Urutan yang saya usulkan

Kolom "Kapan" memakai urutan milestone yang sudah ada. FS = milestone prompt.

| Langkah | Isi | Dari prompt | Kapan | Bergantung pada |
|---|---|---|---|---|
| A ✅ (11 Nov 2026) | **Mesin AP.** Tabel AP sebagai data di versi format template, fungsi hitung murni, endpoint hitung. Test untuk 1.000 kombinasi dan 6 contoh di prompt | FS-2 | Sekarang (lanjutan Tahap 3) | — |
| B ✅ (12 Nov 2026) | **Format PFMEA AIAG-VDA 2019.** Isi awal: 12 field header, 29 kolom dengan grup langkah 2 sampai 6, jenis kolom baru `ap` | FS-3 (bagian data) | Sekarang | A |
| C ✅ (12 Nov 2026) | **Format CP AIAG 2024.** Isi awal: 14 field header termasuk fase (4 pilihan), 14 kolom, tanpa Process Flow | FS-4 (bagian data) | Sekarang | — |
| D | **Form set.** Tabel `form_sets` dan `form_set_versions`, set 1 dari data yang ada, set 2, default per customer, layar Settings › Form sets | FS-1, FS-5 (tanpa wizard) | Sekarang | B, C |
| E | Dokumen menyimpan `form_set_version_id`; part default untuk setiap dokumen (pintu multi-part) | FS-1, FS-9 (migrasi) | M4.1 | D |
| F | Render sheet PFMEA (AIAG-VDA) dan Control Plan (2024) di editor: judul grup, judul vertikal, sel gabung, kotak fase, panel AP | FS-3, FS-4 (layar) | M4.2 sampai M4.4 | E |
| G | Langkah "Choose the form" di wizard | FS-5 | Bersama wizard set dokumen (Tahap 4) | E |
| H | Multi-part: wizard, pemilih part, salin part, cek per part | FS-9 | Setelah editor stabil (akhir Tahap 4) | F |
| I | Pencarian per proses dan teks sel, ekspor daftar | FS-10 | Setelah M4.1 (butuh baris dokumen) | E |
| J | Perpindahan AIAG 4th ke AIAG-VDA lewat permintaan perubahan, dengan pratinjau pemetaan | FS-6 | Tahap 5 | F |
| K | Ekspor Excel A3 dan PDF untuk dua form baru | FS-7 | Tahap 6 | F, uji S5 mesin PDF |
| L | Tampilan genba, ronde, catatan, foto, antrean offline | FS-11 | Setelah rilis ada (Tahap 6) | E, rilis |
| M | UML dan OpenAPI (U1, U3, U5, U8, README) | FS-8 | Bersama setiap langkah | — |

Langkah A sampai D bisa dikerjakan segera setelah Anda setuju, sebelum Tahap 4, karena hanya menyentuh format template. Jumlahnya empat milestone kecil.

## 3. Rincian langkah A sampai D

### Langkah A · Mesin AP
- **Test dulu:** `internal/template/ap_test.go`. Tabel dimuat dari `ap-table-aiag-vda-2019.json`, lalu diuji untuk 1.000 kombinasi S, O, D (setiap kombinasi menghasilkan tepat satu dari H, M, L), ditambah contoh prompt: (8,6,4)=H, (7,3,7)=M, (7,3,4)=L, (5,3,5)=L, (8,2,4)=L, (9,1,10)=L.
- **Migrasi:** tabel `format_ap_rules` per versi format: rentang S, rentang O, rentang D, dan hasil. Ada CHECK rentang 1 sampai 10. Role aplikasi tanpa DELETE.
- **Kode:** `internal/template/ap.go` berisi fungsi murni `ActionPriority(rules, s, o, d)`. Validasi saat menyimpan: tabel harus menutup 1.000 kombinasi tanpa tumpang tindih.
- **API:** tabel AP ikut di jawaban versi format; `PUT` hanya untuk versi draft.
- **Catatan:** isi JSON belum saya cocokkan dengan handbook aslinya (buku berbayar). Enam contoh di prompt cocok dengan isi JSON.

### Langkah B · Format PFMEA AIAG-VDA 2019
- **Migrasi isi awal:** template "PFMEA AIAG-VDA 2019", metodologi `aiag_vda`, metode risiko AP, dengan key field persis seperti tabel di prompt.
- **Jenis kolom baru:** `ap` (dihitung, tidak bisa diketik), dengan set 1 dan set 2, sama seperti RPN.
- **Grup kolom:** kolom baru `group_label` di field versi untuk judul grup (STRUCTURE ANALYSIS (STEP 2) dan seterusnya).
- **Aturan sorot:** baris AP H tanpa Prevention Action dan Detection Action ditandai; tidak memblokir.

### Langkah C · Format CP AIAG 2024
- **Metodologi:** saya usulkan nilai baru `aiag_cp_2024`, supaya CP lama (`aiag_4th`) dan CP 2024 bisa dibedakan tanpa melihat nama template.
- **Fase:** field header jenis select dengan empat pilihan (D-135).
- Kolom `process_no` dan `process_description` memakai jenis terisi otomatis yang sudah ada (D-122).

### Langkah D · Form set
- **Migrasi:** `form_sets`, `form_set_versions` (menunjuk versi format Cover, PFD, PFMEA, CP), `customers.default_form_set_id`. Set 1 dibuat dari versi format yang ada sekarang.
- **Temuan kecil:** format Cover belum ada di kode (M3.1 hanya membuat PFD, PFMEA, dan CP). Kolom `cover_format_version_id` saya buat boleh kosong sampai Cover dibangun.
- **Layar:** Admin › Form sets (board `TemplateSets`): dua set, isi per sheet, default per customer, tabel pemetaan kolom.
- **Test:** customer tanpa default berarti set 1; versi set yang terbit tidak bisa diubah.

## 4. Pertanyaan yang perlu Anda jawab

Pertanyaan 3, 4, 5, 8, dan 9 dari prompt sudah saya catat sebagai temuan Terbuka. E-82 sudah diputuskan (D-139), jadi yang menghambat langkah A sampai D tinggal E-80.

| Temuan | Pertanyaan | Menghambat |
|---|---|---|
| E-80 ✅ | Diputuskan 11 November 2026 (D-140): opsi (a) | — |
| E-81 | Poin PFMEA di process template: satu baris teks, atau isi per kolom (dan bagaimana dipakai di set 1 dan set 2)? | Sebelum M4.5 |
| E-82 ✅ | Diputuskan 11 November 2026 (D-139): PPAP Supervisor (APQP/PPAP Controller) untuk saat ini, lewat hak baru yang bisa diberikan admin ke akun lain, misalnya Project Leader | — |
| E-83 | Dokumen set 1 dipindah ke set 2 satu per satu, atau boleh massal per customer? | Sebelum langkah J |
| E-84 | Genba: siapa selain tim PPAP yang boleh menulis catatan, dan berapa lama foto disimpan? | Sebelum langkah L |
| E-85 | Alias proses: dikelola Pengelola template saja (seperti sekarang), atau penyusun boleh mengusulkan? | Sebelum langkah I |

## 5. Yang tidak saya kerjakan tanpa keputusan baru

- Jaringan kegagalan AIAG-VDA (tetap [DEFERRED], bagian 1.2).
- Mengubah `docs/milestone/milestone.md`. File itu ekspor dari docx; urutan di atas saya tulis di sini saja sampai Anda memperbarui docx-nya.
