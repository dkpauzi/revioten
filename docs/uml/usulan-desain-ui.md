# Usulan dari desain UI dan temuan register (UX-01 sampai UX-07)

- Paket UML: draft 0.12, 3 November 2026.
- Sumber: desain UI 2 November 2026 (canvas "Desain UI Sistem PFD PFMEA CP", baris v2) dan tiga register nomor dokumen Excel yang dipakai sekarang (PL1, PL3, dan register IC/AVI/AJI/GW), dibaca 3 November 2026.
- Status semua butir: **Terbuka**. Setiap butir menunggu keputusan pemilik sebelum masuk brief sebagai R- atau D-. Sampai diputuskan, elemen UML yang memakainya berstereotype `<<usulan>>`.
- Isi register hanya diringkas sebagai jumlah. Nomor dokumen yang disebut di sini adalah contoh pola, bukan daftar.
- Pembaruan 4 November 2026: dua temuan register sudah diputuskan pemilik. Kode customer tetap dua huruf sesuai WI (E-74, D-125), dan register PL1 dan PL3 adalah register Project Leader (E-75, D-126). Butir UX-01 sampai UX-07 sendiri tetap Terbuka.
- Pembaruan 8 November 2026: UX-05 **diterima sebagian** (D-127). Design system dan menu berkelompok (Home, Documents, Template library, Master data, Admin) dipasang di halaman M0 sampai M3. Layar workbook (Tahap 4) dan pilihan bahasa EN/ID tetap Terbuka. Di hari yang sama pilihan bahasa EN/ID juga diterima (D-128, menggantikan D-96), bersama tampilan untuk HP, tablet, dan laptop. Yang masih Terbuka dari UX-05 tinggal layar workbook.
- Pembaruan 8 November 2026 (2): bagian cakupan UX-01 **diterima** (D-129, keputusan E-72): satu template per proses, dan setiap poin punya cakupan umum, customer, group model, atau model, dengan yang paling spesifik menang. Cara penyusun memilih cakupan saat menyusun (UC-25) tetap Terbuka sampai M4.5.
- Pembaruan 12 November 2026: layar workbook UX-05 **diterima** (D-145), jadi UX-05 selesai seluruhnya. Isi poin per kolom (bagian UX-02 yang menyangkut E-73) diputuskan lewat D-146.

| ID | Usulan | Asal | Dampak ke UML | Aturan terkait |
|---|---|---|---|---|
| UX-01 | **Cakupan poin dipilih saat menyusun.** Di satu proses, penyusun bisa memilih poin berlaku untuk model ini saja, group model, customer, atau umum. Model ini saja langsung menjadi perbedaan lokal. Pilihan lain menjadi usulan perubahan process template yang diputuskan Pengelola template. | Masukan pemilik, 2 Nov 2026: semua proses umum, poinnya bisa umum atau per customer, dan bisa diset saat membuat file. | UC-25 baru, U5-E `PointScopeProposal`, U1 "Cakupan poin". Hak: E-2 mengusulkan, E-6 memutuskan. | R-TPL-03, R-CMP-04, D-39, M3.3 |
| UX-02 | **Gambar dan tabel spesifikasi di sel.** Sel CP boleh berisi gambar (misalnya drawing PCB di IQC PCB) atau tabel kecil (misalnya ROM writing atau FCT), di proses mana pun. Ikut tercetak di dalam sel pada Excel dan PDF. | Masukan pemilik, 2 Nov 2026. | UC-39 baru, U5-E `CellContent`, `CellPicture`, `SpecTable`. Mengubah ASUMSI E-73 (poin satu baris teks). Ekspor Excel harus menaruh gambar di sel (R-PRT-01). | R-DOC-01, R-PRT-01, E-73 |
| UX-03 | **Master simbol karakteristik khusus.** Simbol internal (Critical Process sebagai default, Revision otomatis) dan simbol customer (Important Character, Regulation). Simbol baru = nama teks + gambar unggahan, berversi, tidak dihapus. | Masukan pemilik, 2 Nov 2026. | UC-24 baru (Admin, hak A-2), U5-E `CharacteristicSymbol` dan `SymbolVersion`. Melengkapi `format_symbols` (U8-E) yang belum punya gambar. | R-TPL-11, R-REV-09 |
| UX-04 | **Cek sinkronisasi tiga dokumen.** PFD, PFMEA, dan CP dijajarkan menurut nomor operasi dan ID proses master. Hasil per proses: sama, nama berbeda (alias), CP dipecah, item cek khusus CP, atau tidak ada di satu dokumen. Tanda dicek tetap manual. | Pola di dokumen TR-020: nomor operasi sama di tiga dokumen, nama sering berbeda, CP punya sub-proses dan item IPQC. | UC-36 diperluas ke PFD, U5-E `SyncCheckMark`, U1 "Cek sinkronisasi" dan "Item cek khusus CP". | R-DOC-06, D-06, D-65 |
| UX-05 | **Layar workbook mirip Excel.** Satu layar per set dokumen dengan tab sheet Cover, PFD, PFMEA, Control Plan, dan Check; menu berlabel; teks lebih besar; satu langkah berikutnya yang jelas; pilihan bahasa EN/ID. | Masukan pemilik: pengguna lintas generasi yang terbiasa Excel. | Hanya U1 "Workbook". Bahasa ID mengubah D-96 (layar Inggris saja) bila disetujui. | D-96, D-98 |
| UX-06 | **Temuan register nomor.** Lihat tabel di bawah. Impor nomor (UC-14) perlu menyimpan nomor lama dari kolom No. (Before) sebagai rujukan yang bisa dicari, dan laporannya juga menandai kode customer ganda serta satu nomor yang dipakai beberapa part. | Register Excel saat ini, 3 Nov 2026. | UC-14 dirinci, U1 "Nomor lama". U8 tidak berubah sampai tahapnya dibangun. | R-NUM-06, D-88, E-11 |
| UX-07 | **Cakupan yokoten sebagai ukuran utama.** Persentase dokumen target per yokoten yang revisinya sudah rilis, atau diputuskan tidak berlaku dengan alasan. Dipakai sebagai indikator QCC. | Masalah QCC: tidak semua CP direvisi saat yokoten. | UC-64 dirinci, U1 "Cakupan yokoten". | R-CHG-04, brief bagian 2 |

## Temuan register (UX-06)

| Temuan | Contoh pola | Yang ditangani desain |
|---|---|---|
| Satu customer punya dua kode di register | AJ dan AJI, AV dan AVI; HL diganti MS | Riwayat kode customer (D-88) dan foreign key kode di nomor (E-11) |
| Set tidak sejajar | CP satu customer sampai 068, PFD dan PFMEA sampai 059 | Daftar sebagai set (UC-11) dan laporan set tidak sejajar (R-NUM-06) |
| Satu nomor dipakai lebih dari satu part | Satu nomor PFD tercatat untuk tiga part berbeda | Laporan nomor ganda saat impor (R-NUM-06) |
| Format nomor lama masih tercatat | CP-xx-14-K001, P-FMEA-xx-15-K004, PFD-xx-18-K001 | Nomor lama sebagai rujukan (UX-06) |
| Status MP atau EOL hanya ada di sebagian register | Hanya register PL1 yang punya kolom status | Status nomor (R-NUM-04) dan status dokumen |
| Riwayat revisi tidak menyebut yokoten | Yokoten 2025 hampir tidak muncul di kolom Revision Item | Perubahan dengan sumber yokoten dan papan status (UC-44, UC-45) |
| Banyak CP masih revisi 00 | 83 dari 130 CP yang revisinya tercatat (64%) masih -00 | Cakupan yokoten terukur (UX-07) |
