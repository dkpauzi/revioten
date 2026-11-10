# Sistem PFD, PFMEA, dan Control Plan · Gambaran untuk tim IT

- **Tanggal:** 10 November 2026
- **Untuk:** bahan slide presentasi ke tim IT. Claude chat membuat slide-nya dari file ini.
- **Tingkat detail:** garis besar. Rincian aturan ada di brief desain, paket UML (`docs/uml/`), dan kontrak API (`api/openapi.yaml`).
- **Sumber:** repo `main` sampai commit `290f3ee`: brief v1.21 dan keputusan sampai D-131, catatan milestone M0.1 sampai M3.3, panduan `docs/panduan/skenario-alur-web.md`, serta rencana "Form set 2" (bagian 7, masih rencana).
- **Tanpa data asli.** Kode customer dan nomor dokumen di sini hanya contoh.

Setiap bagian bertanda **Slide** di bawah kira-kira satu slide.

---

## Slide 1 · Judul

**Sistem web untuk menyusun, merevisi, dan menomori PFD, PFMEA, dan Control Plan**

Tujuan utamanya: revisi dan yokoten jadi mudah, terkendali, dan bisa ditelusuri.

---

## Slide 2 · Masalah hari ini

- PFD, PFMEA, dan Control Plan (CP) disimpan sebagai file Excel terpisah, lalu beredar sebagai hardcopy bertanda tangan basah.
- **Yokoten lambat dan sering tidak lengkap.** Satu perubahan harus disalin tangan ke banyak model. Contohnya, banyak CP masih di revisi 00 walaupun yokoten sudah berjalan.
- **Nama proses berbeda di tiap dokumen.** Contohnya "Parts Receiving" dan "Part Receiving", atau "Router" dan "PCB Depanelization - Router". Pencocokan PFD, PFMEA, dan CP jadi manual.
- **Mengedit alur proses di CP sangat lama.** Bentuk flowchart dan simbol konektor antarhalaman di Excel digambar ulang setiap kali ada perubahan.
- **Nomor dokumen dicatat di beberapa register Excel.** Ada risiko nomor ganda, lubang, dan kode customer yang tidak konsisten.
- **Riwayat revisi ditulis tangan.** Sulit melihat apa yang berubah di antara dua revisi.

---

## Slide 3 · Tujuan utama

1. **Revisi dan yokoten mudah.** Satu perubahan menjangkau banyak dokumen, dengan keputusan per dokumen dan papan status (G1).
2. **Riwayat perubahan terlihat.** Ada perbandingan per baris dan per sel antara revisi, lengkap dengan alasan dan lampirannya (G2).
3. **Dokumen disusun dari template.** Pilih proses, sub-proses, dan poin dengan centang. Perubahan template sampai ke dokumen hanya lewat revisi yang disetujui (G3).
4. **Nomor dokumen dari satu register.** Server memberi nomor tanpa bentrok antarpengguna (G4).
5. **Cetak tetap utama.** Excel dan PDF ukuran A3 dan A4, dengan tata letak sama seperti form sekarang (G5).
6. **Alur proses mudah disesuaikan.** Kolom Process Flow di CP dibentuk dari langkah PFD, dan simbol konektor antarhalaman dipasang otomatis (G11).

**Yang tidak berubah:** persetujuan tetap tanda tangan basah di kertas. Sistem mencatat rilis secara terpercaya; tanda tangan digital bisa menyusul (G7).

---

## Slide 4 · Prinsip desain yang penting untuk IT

- **ID tetap, nama hanya tampilan.** Proses, kolom form, poin template, dan baris dokumen punya ID permanen. Mengganti nama tidak memutus rujukan.
- **Satu master proses dengan alias.** PFD, PFMEA, dan CP dicocokkan lewat nomor operasi dan ID proses, bukan lewat teks.
- **Kolom terisi otomatis.** Nomor operasi, nama proses, dan Process Flow selalu tampil dan tercetak, tetapi tidak pernah diketik, sehingga ketiga dokumen tidak bisa berbeda.
- **Revisi rilis tidak bisa diubah.** Isinya dibekukan sebagai snapshot dengan SHA-256. Cetakan hanya menampilkan tanda revisi yang berlaku, dan riwayat lengkapnya tetap tersimpan.
- **Format sebagai data.** Kolom form, metode risiko, tabel skala, dan simbol disimpan sebagai format template berversi, bukan ditulis di kode.
- **Audit log append-only.** Dijaga oleh database: role aplikasi tanpa UPDATE dan DELETE, ada trigger penolak, dan ada hash chain.
- **API-first.** Layar web hanyalah klien dari API `/api/v1`, yang dijelaskan kontrak OpenAPI 3.0.4.

---

## Slide 5 · Pengguna dan hak akses

- **Peran:** Penyusun (Author), Pengelola template (PPAP Engineer), Pemeriksa PFMEA dan CP, Verifikator, Dokumen kontrol, Petugas migrasi, dan Viewer.
- **Akun admin terpisah.** Akun admin hanya memegang hak admin dan tidak pernah memegang peran kerja.
- **Cakupan per Project Group.** Project Leader dan Project Engineer hanya bekerja untuk customer di Project Group-nya.
- **Satu fungsi pemeriksaan izin.** Semua operasi lewat satu fungsi (aktor, aksi, resource), dan setiap operasi API diuji izinnya.
- **Aturan yang tidak bisa dimatikan.** Pemohon tidak memverifikasi permintaannya sendiri, pencatat rilis berbeda dari penyusun, dan akun tidak pernah dihapus.

---

## Slide 6 · Alur kerja utama (gambaran)

```
Master data  ->  Template  ->  Nomor dokumen  ->  Susun dokumen  ->  Cetak, tanda tangan basah  ->  Catat rilis
                                                          ^                                               |
                                                          +---- Permintaan perubahan / Yokoten <----------+
```

1. **Admin** menyiapkan customer, model, part number, akun, jabatan, dan rute approval.
2. **PPAP Engineer** mengelola master proses dan pustaka proses (sub-proses dan poin, dengan cakupan umum, customer, group model, atau model).
3. **Penyusun** mendaftarkan nomor sebagai set (PFD, PFMEA, dan CP mendapat satu nomor berjalan).
4. **Nanti:** menyusun dokumen dari template, mencetak untuk tanda tangan, mencatat rilis, lalu merevisi lewat permintaan perubahan atau yokoten.

---

## Slide 7 · Status saat ini (sudah berjalan sebagai demo di localhost)

| Tahap | Isi | Status |
|---|---|---|
| 0 | Fondasi teknis: server Go, PostgreSQL, SvelteKit | Selesai |
| 1 | Login, izin, audit log, master data, jabatan dan rute approval, backup terenkripsi | Selesai |
| 2 | Penomoran: pola, penghitung atomik, register, set, dan impor nomor lama | Selesai |
| 3 | Template: format template, pustaka proses, cakupan poin | Berjalan (M3.1 sampai M3.3 selesai; versi template berikutnya) |
| 4 | Dokumen dan penyusunan dari template, editor grid | Belum |
| 5 | Perubahan, revisi, diff, propagasi template, yokoten | Belum |
| 6 | Cetak Excel dan PDF, catat rilis | Belum |
| 7 | Migrasi dokumen lama dan dasbor ukuran | Belum |
| 8 | Penguatan dan serah terima ke IT | Belum |

**Layar sudah memakai design system dari desain UI:**
- menu berkelompok (Home, Documents, Template library, Master data, Admin);
- dua bahasa, EN dan ID, dengan tombol untuk berganti;
- tampilan untuk HP, tablet, dan laptop.

**Kualitas:** test otomatis di setiap milestone, meliputi test Go dengan database sungguhan, test frontend, dan test E2E di browser. Penomoran diuji dengan 50 pendaftaran paralel.

---

## Slide 8 · Teknologi (versi demo)

| Lapisan | Pilihan |
|---|---|
| Backend | Go 1.27 (`net/http`), pgx, sqlc, goose, tanpa ORM |
| Database | PostgreSQL 18 di Docker |
| Frontend | Svelte 5 dan SvelteKit 3 (SPA statis yang di-embed ke binary Go) |
| Kontrak | OpenAPI 3.0.4; tipe Go dan TypeScript dibuat dari kontrak |
| Test | Go test dengan testcontainers, Vitest, Playwright |
| Nanti | Grid (Tabulator), diagram PFD, ekspor Excel dan PDF, antrean job di PostgreSQL |

**Peran tim IT:** IT menerima spesifikasi dan paket UML, lalu membangun versinya sendiri. Kode demo tidak diserahkan. Stack versi IT boleh berbeda, asal aturan di spesifikasi dipenuhi.

---

## Slide 9 · Yang diserahkan ke IT

- **Brief desain:** tujuan, aturan (R-), keputusan (D-), dan pertanyaan terbuka.
- **Paket UML:** glosarium, diagram konteks C4, use case dan peta hak akses, diagram kelas domain, dan ERD fisik. Nanti ditambah activity diagram, state machine, sequence diagram, dan skenario penerimaan.
- **Kontrak OpenAPI** untuk API layar. API integrasi menyusul.
- **Desain UI:** kanvas berisi layar dan design system.
- **Catatan per milestone dan lembar evaluasi desain:** temuan dari membangun demo beserta keputusan pemilik.

Paket UML dibekukan sebagai v1.0 sebelum serah terima (milestone M8.4).

---

## Slide 10 · Rencana berikutnya (masih rencana, belum dibangun)

**Form set 2: PFMEA AIAG-VDA 2019 dan Control Plan AIAG 2024**
- **Form set** adalah satu paket format untuk Cover, PFD, PFMEA, dan CP. Set 1 adalah form AIAG 4th yang dipakai sekarang. Set 2 memakai PFMEA AIAG-VDA dan CP 2024, dengan Cover dan PFD yang sama.
- Setiap customer punya form set default. Dokumen lama tetap memakai set-nya, dan pindah ke set 2 hanya lewat permintaan perubahan.
- **Action Priority (H, M, L)** dihitung sistem dari tabel AP AIAG & VDA Handbook 2019. Tabel ini disimpan sebagai data, bukan di kode. Baris AP H tanpa tindakan ditandai, tetapi cetak tidak diblokir.
- CP 2024 punya pilihan fase: Prototype, Pre-launch, Safe Launch, atau Production.

**Pencarian per proses**
- Untuk pertanyaan seperti "model apa saja yang punya proses manual solder?", dengan filter customer.
- Mencocokkan nama proses dan semua aliasnya. Hasilnya dipakai juga untuk memilih target yokoten.

**Tampilan genba**
- Tampilan baca di tablet untuk tim PPAP di line. Pengguna memilih proses, lalu hanya baris PFMEA dan CP proses itu yang tampil.
- Catatan dan foto per baris tidak pernah mengubah dokumen. Catatan bisa dijadikan permintaan perubahan.

**Opsional: satu nomor dokumen untuk beberapa part.** PFD, PFMEA, dan CP dibagi per part dalam satu dokumen dan satu revisi. Fitur ini mati secara default.

---

## Slide 11 · Tujuan masa depan (garis besar)

- **API integrasi** untuk sistem lain, dengan token khusus (setelah MVP).
- **Portal customer:** customer melihat dokumen rilis yang boleh mereka lihat.
- **Tanda tangan digital** menggantikan tanda tangan basah, bila disetujui.
- **Migrasi dokumen Excel lama** tanpa perlu dirapikan dulu, lalu ditautkan ke template.
- **Dasbor ukuran untuk QCC:** cakupan yokoten, persentase baris yang mengikuti template, dan waktu revisi.

---

## Slide 12 · Yang kami minta dari IT

- **Masukan arsitektur:** hosting on-premise, backup, dan reverse proxy.
- **Kebijakan keamanan:** login (SSO atau lokal), penyimpanan lampiran, dan pemindaian file.
- **Kesepakatan cara serah terima:** spesifikasi dan UML sebagai dasar, dengan demo sebagai contoh perilaku.
- **Jadwal diskusi lanjutan** sebelum tahap serah terima (Tahap 8).
