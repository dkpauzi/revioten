# Prompt untuk Claude Code · Form set 2 (PFMEA AIAG-VDA 2019 + Control Plan AIAG 2024) dan dokumen multi-part (opsional), pencarian per proses, dan tampilan genba

Salin seluruh isi file ini ke Claude Code di repo proyek. Lampirkan juga file di folder `design/` dan `ap-table-aiag-vda-2019.json` dari paket yang sama (taruh di `docs/design/form-set-aiag-vda/` di repo).

---

## Peran dan cara kerja

Kamu melanjutkan proyek sistem PFD, PFMEA, dan CP (Go + PostgreSQL 18 + SvelteKit). Desain sebelumnya sudah kamu implementasikan. Tugas ini menambah **satu set template baru** tanpa merusak set yang sudah ada.

Sebelum menulis kode:
1. Baca `CLAUDE.md`, brief desain terbaru, paket UML di `docs/uml/` (README, U1, U3, U5, U8, `usulan-desain-ui.md`), dan `docs/evaluasi-desain.md`.
2. Lihat board desain di `docs/design/form-set-aiag-vda/`: file PNG adalah tampilannya, file `.dc.html` berisi struktur, label, dan isi contoh (formatnya untuk kanvas desain, jadi baca sebagai referensi, jangan disalin sebagai kode). Board ini sumber tata letak, label, dan perilaku layar. Data di board adalah data contoh.
3. Cocokkan rencana dengan kode yang ada: tabel `format_templates`, `format_template_versions`, `format_fields`, `format_version_fields`, `format_rating_criteria`, `format_symbols`, process template, register nomor, dan editor workbook.
4. Tulis rencana singkat per milestone di bawah (file, migrasi, endpoint, komponen, test) dan **tunggu persetujuan pemilik sebelum milestone pertama**. Bila ada keputusan yang belum jelas, catat di `docs/evaluasi-desain.md` dengan status Terbuka dan tanyakan; jangan menebak.

Aturan yang tetap berlaku:
- Test case ditulis sebelum setiap fitur atau perubahan.
- Nama file dan teks layar dalam bahasa Inggris (katalog `web/src/lib/i18n/en.ts`); komentar kode dalam bahasa Indonesia.
- Migrasi hanya menambah (additive). Revisi yang sudah rilis tidak pernah berubah (R-REV-02). Audit log tetap append-only.
- Data contoh dan fixture tidak memakai data asli customer.
- Set lama (AIAG 4th) harus tetap bekerja persis seperti sekarang; semua test lama harus tetap lulus.
- UML diperbarui mengikuti kode (D-80): U1, U5, U8, dan README paket UML.

## Tujuan

1. Konsep baru **form set**: satu paket versi format template untuk Cover, PFD, PFMEA, dan Control Plan.
   - Set 1 "AIAG 4th set": Cover v1, General PFD v1, PFMEA AIAG 4th (RPN), CP AIAG lama (dengan kolom Process Flow). Ini perilaku saat ini.
   - Set 2 "AIAG-VDA set": Cover v1 dan General PFD v1 **yang sama** (dipakai bersama), PFMEA AIAG-VDA 2019 (Action Priority), CP AIAG 2024.
2. Setiap customer punya **default form set** untuk dokumen baru. Dokumen yang sudah ada tetap memakai set-nya.
3. Wizard "Make a new document set" mendapat langkah **Choose the form** (board `NewSetFormat`).
4. Layar **Settings › Form sets** (board `TemplateSets`): dua set, isi per sheet, default per customer, dan tabel pemetaan kolom.
5. Editor workbook menampilkan sheet sesuai set: tab `PFMEA (AIAG-VDA)` dan `Control Plan (2024)` (board `PfmeaVda`, `ControlPlan2024`). Tab Cover, PFD, dan Check tidak berubah.
6. Action Priority dihitung sistem dari tabel AP yang disimpan sebagai data di versi format template.
7. Dokumen bisa dipindah dari AIAG 4th ke AIAG-VDA lewat **permintaan perubahan** yang membuat revisi baru (bukan edit langsung).
8. Ekspor Excel dan PDF untuk dua form baru, dengan tata letak yang sama dengan form Excel yang dipakai sekarang.
9. **Pencarian per proses** (board `SearchProcess`): cari model yang memakai proses tertentu (misalnya manual solder), dengan filter customer. Berlaku untuk kedua form set.
10. **Tampilan genba** (board `GenbaView`): tampilan baca untuk tim PPAP di line, dengan dropdown proses dan catatan per poin. Berlaku untuk kedua form set.

## Spesifikasi form

### PFMEA AIAG-VDA 2019

Judul sheet: `POTENTIAL FAILURE MODE AND EFFECTS ANALYSIS (PROCESS FMEA)`, logo kiri atas.

Header "PLANNING AND PREPARATION (STEP 1)", tiga kolom field header (key tetap, label boleh diganti):

| key | label |
|---|---|
| part_name | Part Name |
| company_name | Company Name |
| manufacturing_location | Manufacturing Location |
| customer_name | Customer Name |
| model_year_program | Model Year(s)/Program(s) |
| part_no | Part No. |
| pfmea_start_date | PFMEA Start Date |
| pfmea_revision_date | PFMEA Revision Date |
| cross_functional_team | Cross Functional Team |
| pfmea_id_number | PFMEA ID Number (dari register nomor, read-only) |
| process_responsibility | Process Responsibility |
| confidentiality_level | Confidentiality Level |

Kolom baris, urut kiri ke kanan, dengan baris judul grup di atasnya:

| grup | key | label | jenis |
|---|---|---|---|
| (tanpa grup) | change_history | History/change authorization (As applicable) | text |
| STRUCTURE ANALYSIS (STEP 2) | process_item | 1. Process Item | text, sel digabung per proses |
| | process_step | 2. Process Step | text |
| | work_element | 3. Process Work Element (4M) | select: Man, Machine, Material, Method, Measurement, Milieu |
| FUNCTION ANALYSIS (STEP 3) | function_process_item | 1. Function of the Process Item | text |
| | function_process_step | 2. Function of the Process Step and Product Characteristic | text |
| | function_work_element | 3. Function of the Work Element and Process Characteristic | text |
| FAILURE ANALYSIS (STEP 4) | failure_effect | 1. Failure Effects (FE) | text |
| | severity | Severity (S) of FE | rating 1–10, set 1 |
| | failure_mode | 2. Failure Mode (FM) | text |
| | failure_cause | 3. Failure Cause (FC) | text |
| RISK ANALYSIS (STEP 5) | prevention_control | Current Prevention Control (PC) of FC | text |
| | occurrence | Occurrence (O) of FC | rating 1–10, set 1 |
| | detection_control | Current Detection Controls (DC) of FC or FM | text |
| | detection | Detection (D) of FC/FM | rating 1–10, set 1 |
| | ap | PFMEA AP | ap (dihitung, read-only), set 1 |
| OPTIMIZATION (STEP 6) | prevention_action | Prevention Action | text |
| | detection_action | Detection Action | text |
| | responsible_person | Responsible Person's Name | text |
| | target_completion_date | Target Completion Date | date |
| | action_status | Status: Open, Completed, Discarded | select |
| | action_taken | Action Taken with Pointer to Evidence | text |
| | completion_date | Completion Date | date |
| | severity_after | Severity (S) | rating, set 2 |
| | occurrence_after | Occurrence (O) | rating, set 2 |
| | detection_after | Detection (D) | rating, set 2 |
| | special_characteristic | Special Characteristic | symbol |
| | ap_after | PFMEA AP | ap (dihitung), set 2 |
| | note | Note | text |

Kolom rating dan AP dicetak dengan judul vertikal, seperti form Excel.

### Control Plan AIAG 2024

Kepala: logo, `PT. SIIX EMS INDONESIA`, judul `Control Plan`. Di bawahnya kotak centang fase.

| key | label | jenis |
|---|---|---|
| phase | Prototype / Pre-launch / Safe Launch / Production | select satu pilihan (`prototype`, `pre_launch`, `safe_launch`, `production`), dicetak sebagai empat kotak centang. Safe Launch sudah diputuskan pemilik (9 Nov 2026) |
| control_plan_no | Control plan No. | dari register nomor, read-only |
| key_contact | Key Contact | text |
| date_orig | Date (orig) | date |
| date_rev | Date (rev) | date, diisi otomatis saat cetak |
| customer_name | Customer Name | text |
| core_team | Core Team | text |
| customer_engineering_approval | Customer Engineering Approval/Date (if req'd) | text |
| part_number | Part Number | text |
| organization_plant_approval | Organization/Plant Approval/Date | text |
| customer_quality_approval | Customer Quality Approval/Date (if req'd) | text |
| part_name | Part Name | text |
| other_approval_1 | Other Approval/Date (if req'd) | text |
| other_approval_2 | Other Approval/Date (if req'd) | text |

Baris "Remark" menampilkan legenda simbol dari master simbol (Critical Process, Revision, dan simbol customer).

Kolom baris:

| grup | key | label |
|---|---|---|
| Part/Process Number | process_no | No |
| | process_description | Description |
| | owner | Owner |
| (tanpa grup) | machine_tool | Machine, device, jig or tool for MFG. |
| Characteristics | characteristic_no | No. |
| | product_characteristic | Product control |
| | process_characteristic | Process control |
| (tanpa grup) | special_char_class | Special Char. (Class) | 
| Methods | specification_tolerance | Product/Process Specification/Tolerance (boleh berisi gambar dan tabel spesifikasi) |
| | evaluation_technique | Evaluation / Measurement technique |
| | sample_size | Sample Size |
| | sample_frequency | Sample Freq |
| | control_method | Control Method |
| (tanpa grup) | reaction_plan | Reaction Plan |

Tidak ada kolom Process Flow; alur ada di sheet PFD.

### Tabel AP

- Simpan tabel AP sebagai data di versi format template PFMEA (misalnya tabel `format_ap_rules` per `version_id`, atau JSON yang divalidasi), bukan konstanta di kode, supaya bisa dikoreksi lewat versi baru format template dengan approval.
- Isi awal: `ap-table-aiag-vda-2019.json` (tabel PFMEA dari AIAG & VDA FMEA Handbook 2019; S, O, D dalam rentang, D dalam empat kelompok 7–10, 5–6, 2–4, 1).
- AP dihitung di server saat S, O, atau D berubah, juga untuk set 2 (setelah tindakan). Sel AP read-only di layar.
- Aturan sorot (R-TPL-11): baris dengan AP H tanpa Prevention Action atau Detection Action ditandai dan dihitung di banner "N rows with AP High have no action yet". Hanya menandai, tidak memblokir cetak.
- Warna AP hanya di layar; cetakan menampilkan huruf H, M, L.
- **Keputusan pemilik (9 November 2026): AP mengikuti tabel AIAG & VDA FMEA Handbook 2019.** Sheet "AP Reference Table" di file Excel yang dipakai sekarang berbeda dari handbook di 186 dari 1.000 kombinasi S, O, D (contoh: S 9–10 dengan O 1 tertulis H, di handbook L), jadi sheet itu **tidak** dipakai. Catat keputusan ini di brief (log keputusan) dan di `docs/evaluasi-desain.md`.
- Saat dokumen lama dimigrasi atau dipindah ke set 2, AP selalu dihitung ulang dengan tabel handbook. Bila AP hasil hitung berbeda dari AP yang tertulis di Excel lama, tampilkan daftar barisnya di laporan migrasi (contoh dari satu dokumen: 7 dari 129 baris berubah dari M ke L).

## Model data (usulan, sesuaikan dengan kode yang ada)

- `form_sets` (id, key, name) dan `form_set_versions` (id, form_set_id, version, status draft/published, cover_format_version_id, pfd_format_version_id, pfmea_format_version_id, cp_format_version_id). Versi published tidak bisa diubah. Cover dan PFD boleh menunjuk versi yang sama di beberapa set.
- `customers.default_form_set_id` (boleh kosong = set 1).
- Dokumen atau set dokumen menyimpan `form_set_version_id` yang dipakai. Revisi rilis menyimpan versi ini di snapshot.
- `format_fields.kind` mendapat jenis `ap`. Rating set 2 sudah didukung (`rating_set`).
- `format_templates.methodology` sudah punya `aiag_vda`; CP 2024 bisa memakai methodology baru `aiag_cp_2024` atau nilai lain yang konsisten dengan kode, jelaskan pilihannya.
- Isi baris tetap disimpan per field ID (R-TPL-02), jadi label boleh berubah tanpa merusak data.

## Perpindahan AIAG 4th ke AIAG-VDA

Dibuat sebagai permintaan perubahan (sumber: template, jenis: layout + isi) yang menghasilkan draft revisi dengan form set 2. Pemetaan:

| AIAG 4th | AIAG-VDA | cara |
|---|---|---|
| Process Step / Function | process_item, process_step | disalin |
| Potential Failure Mode | failure_mode | disalin |
| Potential Effect(s), SEV | failure_effect, severity | disalin |
| Potential Cause(s), OCC | failure_cause, occurrence | disalin |
| Current Controls Prevention / Detection, DET | prevention_control, detection_control, detection | disalin |
| RPN | ap | dihitung ulang; RPN lama tetap ada di riwayat revisi |
| Class | special_characteristic | disalin |
| Recommended Action, Responsibility, Actions Taken, nilai sesudah | kolom Optimization yang sesuai | disalin bila ada |
| (tidak ada) | work_element, tiga kolom fungsi | kosong dan ditandai "To fill" |

CP lama ke CP 2024: kolom Process Flow dibuang dari sheet (alur tetap di PFD), owner diisi manual, kolom lain dipetakan menurut label. Tampilkan pratinjau pemetaan sebelum draft dibuat.

## Yang tidak berubah

- PFD: satu form umum untuk semua set.
- Cover: layout yang sama untuk semua set.
- Cek PFMEA dan CP (UC-36): pencocokan menurut nomor operasi dan ID proses master; tanda dicek tetap manual. Untuk set 2, panel detail menampilkan AP dari baris PFMEA yang ditautkan.
- Tautan PFMEA ke CP (R-DOC-05) tetap manual atau dari template. Panel "Linked Control Plan row" dan "Linked PFMEA row" di board memakai tautan ini.
- Yokoten, register nomor, cetak untuk tanda tangan, dan rilis.

## Opsional: satu nomor dokumen untuk beberapa part (multi-part)

Kasus nyata: satu Control Plan No. berisi tiga jenis PCB dengan proses berbeda, seperti tiga Control Plan digabung jadi satu. Cover mencantumkan semua part name dan part number; lalu setiap part punya bagian sendiri dengan judul produk, part number (bisa pasangan R/L), dan alur proses yang nomornya mulai lagi dari 1. Board: `MultiPartSetup` (wizard) dan `MultiPartWorkbook` (editor).

Fitur ini **opsional dan mati secara default**: dokumen biasa tetap satu part, dan tidak ada yang berubah bagi pengguna yang tidak memakainya.

Keputusan pemilik (9 Nov 2026):
- **PFD, PFMEA, dan Control Plan ketiganya dibagi per part** (bukan hanya CP).
- **Satu dokumen satu form set.** Part tidak boleh punya form set berbeda dari part lain; form set dipilih di tingkat dokumen.

Aturan:
- Yang dipakai bersama semua part: nomor dokumen, cover, riwayat revisi, nomor revisi, rute approval dan tanda tangan, core team, customer, form set.
- Yang berbeda per part: nama part (product), part number (satu atau lebih, dari master part number), judul bagian, alur proses PFD, baris PFMEA, baris CP, cek sinkronisasi, target yokoten.
- Satu revisi untuk seluruh dokumen. Diff dan riwayat perubahan dikelompokkan per part.
- Nomor operasi unik di dalam satu part, bukan di seluruh dokumen. Pencocokan PFD–PFMEA–CP (UC-36) berjalan per part.
- Yokoten bisa menarget part tertentu ("berlaku untuk part B dan C, tidak untuk A"); papan yokoten menampilkan dokumen dan part.
- Part baru bisa mulai dari template atau dari salinan part lain (lalu diedit). Salinan bersifat mandiri, bukan tautan.
- Menambah, menghapus, mengganti nama, atau mengubah urutan part setelah rilis dilakukan lewat permintaan perubahan dan menghasilkan revisi baru.
- Cetak dan ekspor: cover mencantumkan semua part berurutan; tiap part dicetak sebagai bagian sendiri dengan kepala sheet berisi Product dan judul besar part. Bisa cetak satu part atau semua part. Excel: satu sheet per part per jenis dokumen (misalnya `CP-A`, `CP-B`), urutan mengikuti part.
- Dokumen satu part tidak menampilkan baris pemilih part sama sekali.

Model data (usulan):
- `document_parts` (id, document_id atau document_set_id, key A/B/C…, name, position, created_at) dan `document_part_numbers` (part_id, part_number_id). Part tidak pernah dihapus dari revisi yang sudah rilis.
- Baris PFD, PFMEA, dan CP mendapat `part_id`. Dokumen satu part tetap punya tepat satu part default, supaya kode tidak bercabang dua (pastikan migrasi membuat part default untuk semua dokumen yang ada, dan test lama tetap lulus).
- Snapshot revisi menyimpan daftar part beserta urutannya.

## Pencarian per proses (board `SearchProcess`)

Tujuan: menjawab pertanyaan seperti "model apa saja yang punya proses manual solder?", lalu menyaring per customer. Juga dipakai untuk menentukan target yokoten.

Perilaku:
- Satu kotak cari di menu utama **Search**. Mode default **Process names**: mencocokkan nama master proses dan **semua alias**-nya (contoh: Manual Soldering ↔ Hand Soldering, Manual Solder (THT), Touch-up Solder). Pencocokan tidak peka huruf besar/kecil dan toleran spasi/tanda hubung. Tampilkan baris "Matches process X · also found as …".
- Mode kedua **All cell text**: cari teks bebas di semua sel PFD, PFMEA, dan CP (misalnya nama alat atau spesifikasi). Gunakan full-text search PostgreSQL (`tsvector` + `pg_trgm` untuk teks sebagian), bukan `LIKE` di seluruh tabel.
- Filter: customer (multi-pilih, dengan jumlah hasil per customer), jenis dokumen (PFD/PFMEA/CP), status (default **Released only**; opsi Include drafts; opsi Show EOL models), model group, form set. Filter aktif tampil sebagai chip yang bisa dihapus. Filter disimpan di URL query supaya bisa dibagikan.
- Hasil dikelompokkan **per model**: nama model, customer, nomor dokumen dan revisi, lokasi proses (nomor operasi, nama atau alias yang cocok disorot), jumlah baris PFMEA dan CP di proses itu. Untuk dokumen multi-part, tampilkan part mana yang memuat proses itu. Tandai bila proses hanya ada di PFD tetapi tidak di PFMEA/CP (sinyal sinkronisasi), dan tampilkan badge AP H bila ada baris PFMEA AP H di proses itu (set 2).
- Aksi: **Open at op X** (membuka tampilan genba langsung di proses itu), **Download list (Excel)**, dan **Start yokoten with these** (membuat draft yokoten dengan model hasil filter sebagai kandidat target, tetap lewat alur yokoten yang ada).
- Hak akses: semua pengguna yang boleh melihat dokumen; hasil hanya memuat dokumen yang boleh dilihat pengguna itu.
- Kinerja: hasil untuk ±500 set dokumen di bawah 1 detik. Pencarian berbasis proses memakai `process_master_id` di baris, bukan teks, sehingga alias tidak perlu dicari ulang.

Data: bila belum ada, tambahkan tabel `process_aliases` (process_master_id, alias, created_by) dan pastikan baris PFD/PFMEA/CP menyimpan `process_master_id`. Baris lama tanpa master dicocokkan lewat nama dan ditandai "unmapped" di laporan agar bisa dirapikan.

## Tampilan genba (board `GenbaView`)

Tujuan: tim PPAP membawa tablet ke line, memilih proses, dan hanya melihat poin yang relevan untuk proses itu, lalu memberi catatan.

Perilaku:
- Masuk dari dokumen (tombol **Genba view**), dari menu **Genba**, atau dari hasil pencarian. Pilih model/set dokumen (dan part, untuk multi-part).
- Selalu membuka **revisi rilis terakhir** dan **read-only**. Draft bisa dibuka dengan pilihan eksplisit dan diberi label Draft.
- **Dropdown proses** besar berisi semua operasi dari PFD (nomor + nama), dengan tombol **sebelumnya/berikutnya**. Proses terakhir yang dipilih diingat per pengguna per dokumen.
- Tiga tab: **PFD** ditampilkan **utuh** (seluruh alur), dengan proses terpilih disorot dan digulir ke tengah; **Control Plan** dan **PFMEA** hanya menampilkan baris dari proses terpilih, dengan jumlah baris di tab. Panel samping kecil juga menampilkan PFD utuh dengan sorotan.
- Baris ditampilkan sebagai kartu (bukan tabel lebar) agar terbaca di tablet; tombol **Show as table** menampilkan kolom asli. Kolom yang tampil di kartu: CP = characteristic, specification, method, frequency/size, PIC, reaction plan; PFMEA = FM, FE, FC, S/O/D, AP (set 2) atau RPN (set 1), prevention dan detection control.
- Target sentuh minimal 44 px; layout bekerja di 1024–1280 px lebar (landscape tablet) dan tetap dapat dipakai di ponsel.
- Per baris: tombol **OK** (tercatat sudah dicek di genba) dan **+ Note**. Catatan berisi teks, jenis (Finding / Suggestion), foto opsional (kamera tablet, dikompres di klien), waktu, dan penulis. Catatan menempel pada baris dan revisi, **tidak pernah mengubah dokumen**.
- **Genba round**: setiap sesi genba adalah satu ronde (dokumen, revisi, line opsional, tanggal, pelaksana) dengan progres "x dari y proses dicek" dan jumlah catatan. Ronde bisa ditutup dan dikirim ke penyusun (**Send notes to author**, notifikasi in-app).
- Penyusun melihat daftar catatan per dokumen dengan status **Open / Closed**, dan bisa mengubah catatan menjadi **permintaan perubahan** (catatan tertaut ke CR; tertutup otomatis saat revisi baru rilis) atau menutupnya dengan alasan.
- Catatan pada revisi lama tetap terlihat di riwayat, dengan label revisi.
- Offline ringan: bila sinyal hilang, catatan disimpan sementara di perangkat (IndexedDB) dan dikirim saat online; tampilkan status "Not sent yet".

Model data (usulan): `genba_rounds` (id, document_set_id, revision_id, part_id null, line, started_by, started_at, closed_at), `genba_checks` (round_id, row_ref, result OK, checked_by, checked_at), `genba_notes` (id, round_id, document_id, revision_id, row_ref, process_no, kind finding/suggestion, text, status open/closed, closed_reason, change_request_id null, created_by, created_at), `genba_note_photos` (note_id, file_id). `row_ref` menunjuk id baris stabil, bukan nomor urut.

Hak akses: izin baru `genba.note` (default untuk peran Penyusun/PPAP; Admin bisa memberi ke peran lain, misalnya QA atau leader produksi) dan `genba.note.close` (penyusun dokumen dan PPAP Engineer). Semua catatan dan perubahan status masuk audit log.

## Milestone (PR kecil, masing-masing dengan test lebih dulu)

1. **Form set**: migrasi, layanan, API, seed set 1 dari data yang ada. Test: set 1 menghasilkan perilaku yang sama; default customer kosong berarti set 1.
2. **Mesin AP**: tabel AP sebagai data, fungsi hitung, endpoint. Test berbasis tabel untuk **semua 1.000 kombinasi** dari JSON, plus kasus contoh: (8,6,4)=H, (7,3,7)=M, (7,3,4)=L, (5,3,5)=L, (8,2,4)=L, (9,1,10)=L.
3. **Format PFMEA AIAG-VDA**: seed versi format, field header dan kolom di atas, render sheet di editor (judul grup, judul vertikal, sel digabung per process item), panel samping AP dan rantai kegagalan.
4. **Format CP 2024**: seed versi format, render sheet, kotak centang fase, legenda simbol, panel tautan PFMEA.
5. **Form set 2 dan wizard**: layar Settings › Form sets, default per customer, langkah Choose the form di wizard. Set 2 masuk lewat approval seperti perubahan template lain.
6. **Perpindahan format**: permintaan perubahan dengan pratinjau pemetaan.
7. **Ekspor**: layout Excel A3 dan PDF untuk dua form baru, sesuai form Excel yang dipakai sekarang.
8. **Dokumen**: U1 (istilah: form set, Action Priority, work element 4M, phase), U5 (kelas FormSet, FormSetVersion, ApRule), U8 (tabel baru), README paket UML, OpenAPI.
9. **Opsional · multi-part**: part default untuk semua dokumen (migrasi), lalu wizard, pemilih part di editor, salin dari part lain, cek sinkronisasi dan yokoten per part, cetak dan ekspor per part. Kerjakan setelah milestone 1 sampai 8 stabil, atau lebih awal bila pemilik meminta. Perbarui U1 (istilah: part, dokumen multi-part), U3 (use case baru), U5, dan U8.
10. **Pencarian per proses**: `process_aliases`, indeks pencarian, endpoint `GET /api/search/process` dan `GET /api/search/text` dengan filter, layar Search, ekspor Excel, tombol Start yokoten. Test: alias ditemukan, filter customer dan status bekerja, dokumen tanpa izin tidak muncul, dokumen multi-part menampilkan part.
11. **Tampilan genba dan catatan**: endpoint baca per proses (`GET /api/documents/{id}/genba?process_no=`), layar Genba, ronde, OK per baris, catatan dengan foto, daftar catatan untuk penyusun, ubah catatan menjadi CR, izin `genba.note` dan `genba.note.close`, antrean offline. Perbarui U1 (istilah: genba round, genba note), U3 (use case baru: cari model per proses, lakukan genba, tindak lanjuti catatan genba), matriks izin, U5, U8, OpenAPI.

Milestone 10 dan 11 tidak bergantung pada form set 2; boleh dikerjakan paralel setelah milestone 1, bila pemilik setuju.

## Kriteria penerimaan (contoh)

- Given customer AB punya default set 2, When penyusun membuat set dokumen baru, Then langkah Choose the form memilih AIAG-VDA set lebih dulu, dan PFD yang dibuat sama dengan set 1.
- Given baris PFMEA set 2 dengan S 8, O 6, D 4, When penyusun menyimpan, Then AP = H dan banner menghitung baris itu bila Prevention Action dan Detection Action kosong.
- Given penyusun mengubah O menjadi 2 di kolom sesudah tindakan, When disimpan, Then AP sesudah = L.
- Given dokumen AIAG 4th yang sudah rilis, When dipindah ke AIAG-VDA, Then revisi rilis tidak berubah, draft baru memakai set 2, dan kolom work element serta fungsi ditandai "To fill".
- Given CP 2024 dengan fase Pre-launch, When dicetak, Then kotak Pre-launch tercentang dan kolom Process Flow tidak ada.
- Given dokumen set 1, When dibuka setelah update, Then tampilan, ekspor, dan test lama tidak berubah.
- Given dokumen dengan satu part, When dibuka, Then tidak ada pemilih part dan tampilannya sama seperti sebelum fitur multi-part.
- Given dokumen SC-KT-040 dengan part A, B, C, When penyusun membuka part B, Then nomor proses mulai dari 1, kepala sheet menampilkan nama dan part number part B, dan nomor dokumen serta revisi sama dengan part A.
- Given yokoten yang menarget part B dan C, When diputuskan, Then draft revisi hanya mengubah baris part B dan C, dan dokumen tetap satu revisi.
- Given dokumen multi-part, When dicetak semua part, Then cover mencantumkan tiga part name dan part number berurutan, lalu bagian A, B, C.
- Given dokumen multi-part, When penyusun membuka PFD atau PFMEA, Then keduanya juga dibagi per part seperti CP, dan form set sama untuk semua part.
- Given CP 2024 dengan fase Safe Launch, When dicetak, Then kotak Safe Launch tercentang dan tiga kotak lain kosong.
- Given proses master Manual Soldering dengan alias Hand Soldering, When pengguna mencari "manual solder" dengan filter customer KT, Then hasil memuat model KT yang memakai nama master maupun alias, dan tidak memuat model customer lain.
- Given filter status default, When mencari, Then hanya revisi rilis yang dicari; dengan Include drafts, draft ikut tampil dengan label Draft.
- Given dokumen rilis dengan proses 5.1 Loading PCB, When anggota PPAP memilih 5.1 di tampilan genba, Then tab CP dan PFMEA hanya menampilkan baris proses 5.1, dan tab PFD menampilkan seluruh alur dengan 5.1 disorot.
- Given catatan genba pada baris CP, When disimpan, Then dokumen dan revisi tidak berubah, catatan berstatus Open, dan penyusun menerima notifikasi saat ronde dikirim.
- Given catatan Open, When penyusun mengubahnya menjadi permintaan perubahan, Then catatan tertaut ke CR dan tertutup otomatis saat revisi baru rilis.
- Given pengguna tanpa izin `genba.note`, When membuka tampilan genba, Then ia bisa membaca tetapi tidak melihat tombol + Note.

## Pertanyaan untuk pemilik (catat di evaluasi-desain.md, jangan ditebak)

1. (Sudah diputuskan: tabel AP mengikuti handbook AIAG & VDA 2019.)
2. (Sudah diputuskan: fase CP 2024 memakai empat pilihan, termasuk Safe Launch.)
3. Poin di process template: isi poin sekarang satu baris teks (ASUMSI E-73). Untuk set 2, apakah poin PFMEA menyimpan isi per kolom VDA (work element, FE, FM, FC, PC, DC), dan bagaimana poin yang sama dipakai di set 1 dan set 2?
4. Siapa yang menyetujui form set baru dan perubahan tabel AP: Admin, PPAP Engineer, atau keduanya?
5. Apakah dokumen set 1 boleh dipindah massal ke set 2 per customer, atau hanya satu per satu?
6. (Sudah diputuskan: PFD, PFMEA, dan CP ketiganya dibagi per part.)
7. (Sudah diputuskan: tidak; satu dokumen satu form set.)
8. Genba: siapa saja selain tim PPAP yang boleh menulis catatan (QA, leader produksi)? Berapa lama foto catatan disimpan?
9. Pencarian: apakah daftar alias proses dikelola Admin saja, atau penyusun boleh mengusulkan alias (lewat approval)?
