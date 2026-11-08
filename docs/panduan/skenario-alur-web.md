# Skenario alur web: dari nol sampai nomor dokumen

- **Dibuat:** 8 November 2026, setelah M3.3. Akan diperbarui setiap milestone menambah layar.
- **Untuk siapa:** pemilik, untuk mencoba seluruh sistem dari database kosong secara berurutan.
- **Isinya:** hanya layar yang sudah dibangun (M0.1 sampai M3.3). Langkah yang belum ada di sistem ditulis di bagian 9.
- **Nama orang:** semua nama di sini contoh. Jangan memakai nama orang asli di data uji.

## 0. Gambaran besar

Urutannya mengikuti siapa yang boleh melakukan apa:

```
[Admin]            1. Login admin
                   2. Project group dan customer
                   3. Model group, model, part number
                   4. Akun kerja + peran
                   5. Jabatan dan pemegangnya
                   6. Rute approval, jenis dokumen, format template (cek isi awal)
                   7. Logout
[PPAP Engineer]    8. Login pertama -> ganti password
                   9. Master proses (Processes)
                  10. Process template: sub-proses, poin, cakupan, varian
                  11. Pratinjau poin untuk satu model
                  12. Daftarkan nomor dokumen (sebagai set)
[PPAP Staff]      13. Void / obsolete nomor, impor nomor lama
[Admin]           14. Audit log: periksa rantai
```

Mengapa admin lebih dulu: data dasar (customer, model, akun, jabatan) hanya bisa dibuat akun admin (hak A-1 dan A-2). Akun kerja tidak bisa membuatnya, dan akun admin tidak bisa menyusun template atau mendaftarkan nomor (D-72: akun admin tidak pernah memegang peran kerja).

## 1. Peran dan fungsinya

Setiap akun yang login otomatis bisa **melihat** dokumen rilis, register nomor, dan master data (hak L-1 sampai L-3). Peran menambah hak di atas itu. Satu akun boleh punya beberapa peran.

| Peran (layar) | Peran (glosarium) | Fungsinya sekarang | Fungsinya nanti (belum dibangun) | Hak |
|---|---|---|---|---|
| Author | Penyusun | Mendaftarkan nomor PFD, PFMEA, dan CP, satu-satu atau sebagai set | Membuat dan mengedit draft dokumen, mengajukan perubahan, mencetak untuk tanda tangan | E-1 sampai E-5 |
| Template maintainer | Pengelola template | Mengelola master proses dan process template: sub-proses, poin, cakupan, varian | Menerbitkan versi template dan memutuskan usulan cakupan poin | E-6 |
| PFMEA–CP checker | Pemeriksa PFMEA–CP | Belum ada layar | Memberi tanda dicek di perbandingan PFMEA dan CP | V-2 |
| Verifier | Verifikator | Belum ada layar | Menerima atau menolak permintaan perubahan | V-1 |
| Document control | Dokumen kontrol | Void dan Make obsolete nomor, impor nomor lama | Mencatat rilis dan salinan terkendali | E-7 sampai E-9 |
| Migration | Petugas migrasi | Belum ada layar | Memindahkan dokumen lama ke sistem | E-10 |
| Viewer | Viewer | Melihat saja. Tidak perlu dicentang, karena semua akun sudah punya | — | L-1 sampai L-3 |
| Admin | Admin | Akun, master data, format template, pengaturan, audit log | Sama | A-1 sampai A-4 |

**Cakupan peran.** Untuk setiap peran yang dicentang ada pilihan cakupan:
- **All customers**: berlaku untuk semua customer. Untuk PPAP Engineer dan PPAP Staff.
- **Own project group**: hanya customer di Project Group tempat orang itu tercatat sebagai Project Leader atau Project Engineer (D-71). Cakupan ini baru berfungsi setelah orangnya dicatat sebagai pemegang jabatan (langkah 5).

**Aturan yang tidak bisa dimatikan admin (R-SEC-11).** Akun admin tidak pernah memegang peran kerja. Orang yang memegang akun admin juga punya akun kerja terpisah untuk pekerjaan sehari-hari. Akun tidak pernah dihapus, hanya dinonaktifkan.

**Isi awal yang disarankan untuk uji coba** (jabatan mengikuti U3 bagian 2):

| Akun contoh | Jenis akun | Peran | Cakupan |
|---|---|---|---|
| `admin.contoh` | Admin account | Admin (otomatis) | — |
| `ppap.engineer` | Work account | Author, Template maintainer, PFMEA–CP checker, Document control, Migration | All customers |
| `ppap.staff` | Work account | Author, Document control, Migration | All customers |
| `pl.gr1` (Project Leader GR1) | Work account | Author, Verifier | Own project group |
| `pe.gr1` (Project Engineer GR1) | Work account | Author, Verifier | Own project group |
| `ppap.controller` (APQP/PPAP Controller) | Work account | Verifier | All customers |
| `npi.manager` | Work account | Tanpa peran (otomatis Viewer) | — |

Untuk uji coba pertama, `ppap.engineer` dan `ppap.staff` sudah cukup.

## 2. Persiapan sekali (di komputer)

1. Buka Docker Desktop, tunggu sampai statusnya "Engine running".
2. Buka PowerShell di folder `C:\E_Project\revioten-main`, lalu nyalakan database:
   ```bash
   docker compose up -d
   ```
3. Jalankan server, lalu biarkan jendela ini terbuka:
   ```bash
   go run ./cmd/revioten
   ```
4. **Hanya bila belum pernah:** buat akun admin pertama di jendela PowerShell lain. Ketik password dua kali (tidak tampil di layar):
   ```bash
   go run ./cmd/revioten create-admin -name "Contoh Admin" -login admin.contoh
   ```
5. Buka http://127.0.0.1:8080 di browser.

Password saat ini minimal 15 karakter (D-85, aturan NIST untuk password tanpa faktor kedua). Kalimat pendek yang mudah diingat paling praktis, misalnya `kopi pagi di pabrik`. Password yang memuat nama login, nama orang, atau kata `revioten` ditolak.

## 3. Admin: login dan data dasar

### Langkah 1 · Login sebagai admin
1. Di halaman **Log in**, isi **Login name** `admin.contoh` dan **Password**, lalu tekan **Log in**.
2. Menu atas sekarang berisi: Home, Documents, Template library, Master data, dan Admin.
3. Di HP atau tablet (layar di bawah 1024px), menu terlipat di tombol **Menu**.
4. Tombol **EN | ID** di bilah atas mengganti bahasa layar.

### Langkah 2 · Project group dan customer
Menu **Master data → Customers**.

1. Bagian **Project groups**: isi **Project group code** (1 sampai 10 huruf atau angka, misalnya `GR1`) dan **Project group name** (misalnya `Group 1`), lalu tekan **Add project group**. Project group harus ada sebelum customer.
2. Bagian **Add a customer**:
   - **Customer code**: dua huruf A sampai Z, misalnya `LM`. Kode ini masuk ke nomor dokumen, misalnya SH-LM-001. Kode `ZZ` dan `HSE` sudah dipakai sebagai kode khusus.
   - **Customer name**: misalnya `Contoh Customer LM`.
   - **Project group**: pilih `GR1`.
   - Tekan **Save**.
3. Kolom **Actions** di daftar customer:
   - **Edit** untuk mengganti nama, project group, atau status aktif;
   - **Change code** untuk mengganti kode customer. Kode lama tetap tercatat di **Previous codes**.

### Langkah 3 · Model group, model, dan part number
Menu **Master data → Models**.

1. Pilih **Customer** `LM`.
2. **Model groups**: isi **Model group name** (misalnya `Seri K`), lalu tekan **Add model group**. Group model boleh dilewati, karena model tanpa group tetap bisa dipakai.
3. **Models**: isi **Model name** (misalnya `K-125`) dan pilih **Model group** `Seri K` atau **No group**, lalu tekan **Add model**. Buat juga satu model tanpa group, misalnya `K-200`, untuk mencoba prioritas varian nanti.
4. **Part numbers**: tambahkan part number untuk model yang dipilih (boleh lebih dari satu).

### Langkah 4 · Buat akun kerja
Menu **Admin → Accounts**, bagian **Create an account**.

1. **Person**:
   - **New person** bila orangnya belum punya akun. Isi **Name of the person** (contoh: `Contoh PPAP Engineer`) dan, bila ada, **Employee number (optional)**.
   - **Existing person** bila orangnya sudah punya akun lain, misalnya pemegang akun admin yang butuh akun kerja. Pilih orangnya dari daftar, supaya audit log tahu kedua akun milik orang yang sama (D-74).
2. **Account type**:
   - **Work account** untuk pekerjaan sehari-hari;
   - **Admin account** hanya untuk tugas admin. Akun admin otomatis hanya punya peran Admin.
3. **Login name**: 3 sampai 64 karakter, berupa huruf kecil a sampai z, angka, titik, garis bawah, atau tanda minus. Contoh: `ppap.engineer`. Nama login tidak bisa dipakai dua kali.
4. **Roles**: centang peran sesuai tabel di bagian 1, lalu pilih cakupan setiap peran, yaitu **All customers** atau **Own project group**.
5. Tekan **Create account**.
6. Layar menampilkan **Temporary password for ppap.engineer** satu kali saja. Salin ke tempat aman sementara: password ini tidak bisa dilihat lagi. Orangnya wajib menggantinya saat login pertama.
7. Ulangi untuk `ppap.staff`.

Di daftar **All accounts**, admin bisa mengubah peran (**Edit roles**, lalu **Save roles**), mereset password (**Reset password**), atau menonaktifkan akun (**Disable**). Akun tidak pernah dihapus.

### Langkah 5 · Jabatan dan pemegangnya
Menu **Master data → Positions**.

1. Jabatan sudah terisi dari bagan NPI: PPAP Engineer, PPAP Staff, Project Engineer, Project Leader, APQP/PPAP Controller, NPI Manager, QA QC Manager, Advisor, dan Factory Manager. Namanya bisa diganti lewat **Rename**, dan ID-nya tetap.
2. Catat pemegang:
   - pilih **Position** dan **Person**;
   - untuk Project Leader dan Project Engineer, pilih juga **Project group**, karena keduanya dipegang satu orang per project group;
   - tekan **Add holder**.
3. Ini penting untuk akun bercakupan **Own project group**: tanpa catatan pemegang, akun itu tidak punya customer apa pun.

### Langkah 6 · Periksa isi awal pengaturan dokumen
Halaman berikut sudah terisi saat database dibuat. Cukup diperiksa, dan diubah bila berbeda dengan WI.

| Menu | Isi awal | Kapan diubah |
|---|---|---|
| Master data → Document types | PFD (huruf H), PFMEA (F), CP (C), kode pabrik, pola nomor, digit revisi, kode khusus ZZ dan HSE | Bila pola nomor di WI berbeda. Perubahan hanya berlaku untuk nomor baru |
| Master data → Change types | Jenis perubahan (Content, Layout only, Administrative), sumber perubahan, kategori 4M | Bila ada sumber perubahan baru |
| Master data → Approval routes | Rute 5 slot: Prepared, Checked, lalu tiga Approved (NPI Manager, QA / QC Mgr., General Mgr.) | Atur **Default route** per customer, atau rute khusus per jenis perubahan |
| Template library → Format templates | Format PFD, PFMEA (misalnya PFMEA AIAG 4th), dan CP: kolom, field header, tabel skala, simbol, aturan sorot | Ganti label kolom mengikuti form yang dipakai, isi teks tabel skala S, O, D |
| Admin → Settings | Saklar tanggal tanda tangan di setiap slot | Sesuai kebiasaan cetak |

Kolom nomor operasi, proses, dan Process Flow di format template tidak bisa diketik, karena isinya diambil dari baris dan dari PFD (D-122, D-123).

### Langkah 7 · Logout admin
Tekan **Log out** di kanan atas. Layar kembali ke halaman login.

## 4. PPAP Engineer: login pertama dan process template

### Langkah 8 · Login pertama dan ganti password
1. Login dengan `ppap.engineer` dan password sementara dari langkah 4.
2. Sistem langsung membuka **Change password** dengan pesan bahwa password harus diganti.
3. Isi tiga kotak:
   - **Current password**: password sementara;
   - **New password**: password baru Anda (aturannya di bagian 2);
   - **Repeat the new password**: ketik ulang.
4. Tekan **Change password**. Sesi lain milik akun ini otomatis berakhir.
5. Menu **Change password** juga ada di bilah atas, untuk mengganti password kapan saja.

Menu akun ini berbeda dari menu admin: tidak ada kelompok Admin, dan halaman master data hanya bisa dilihat.

### Langkah 9 · Master proses
Menu **Template library → Processes**.

1. Di **Add a process**, isi **Process name** dengan nama yang dipakai di WI, misalnya `Laser marking`.
2. **Aliases** (boleh kosong): nama lain proses ini di dokumen lama, satu per baris, misalnya `Laser mark` dan `LM marking`. Alias membantu impor dokumen lama nanti.
3. Tekan **Add process**.
4. Ulangi untuk proses lain, misalnya `SMT`, `Check Qty`, dan `Packing`.

Dokumen menunjuk proses lewat ID tetapnya, jadi mengganti nama proses lewat **Edit** tidak merusak dokumen (D-06).

### Langkah 10 · Process template
Menu **Template library → Process templates**.

1. **Process**: pilih `Laser marking`. Bila belum punya template, tekan **Add process template**. Satu proses hanya punya satu template (D-129). Template baru dibuat sebagai **Version 1 · Draft**.
2. **Sub-processes**: isi **Name** (misalnya `Persiapan`, lalu `Marking`, lalu `Inspeksi`) dan tekan **Add sub-process** untuk setiap sub-proses. Urutan bisa diubah dengan tombol panah naik dan turun.
3. Pada sub-proses `Marking`, tekan **Points**. Bagian **Points of Marking** terbuka, dengan daftar terpisah untuk PFD, PFMEA, dan CP.
4. Di **Add a point to Marking**:
   - **Document type**: PFD, PFMEA, atau CP;
   - **Text**: isi poin, misalnya `Laser type sesuai spesifikasi`;
   - **Use point for**: cakupan poin. Pilih **General** bila poin ini biasanya berlaku untuk semua model. Pilih **Customer**, **Model group**, atau **Model** bila poin ini hanya untuk sasaran itu; setelah itu pilih customernya, lalu group atau modelnya;
   - tekan **Add point**.
5. Tambahkan juga poin `Content marking sesuai drawing` dan `Location marking sesuai drawing` dengan cakupan General.

Cakupan poin tidak bisa diubah setelah dibuat (D-131). Bila salah pilih, tekan **Edit**, centang **Hidden** untuk menyembunyikan poin itu, lalu buat poin baru dengan cakupan yang benar. Poin dan sub-proses tidak pernah dihapus.

### Langkah 11 · Varian dan pratinjau
Contoh: laser type biasanya umum, tetapi model K-125 memakai laser fiber 20 W.

1. Pada poin `Laser type sesuai spesifikasi`, tekan **Add variant**.
2. **Use point for**: pilih **Model**, lalu customer `LM`, lalu model `K-125`.
3. **Text of the variant**: `Laser type: fiber 20 W`. Tekan **Add variant**.
4. Varian tampil tepat di bawah poinnya. Varian hanya boleh untuk cakupan yang lebih sempit, di dalam cakupan poin asalnya.
5. Coba juga varian **Customer** `LM` dengan teks `Laser type: CO2`.
6. Bagian **Points for one model**:
   - pilih customer `LM` dan model `K-125`: poin laser type memakai `fiber 20 W` (varian model menang);
   - pilih model `K-200`: poin itu memakai `CO2` (varian customer), karena K-200 tidak punya varian model.

Urutan yang menang: model, lalu group model, lalu customer, lalu umum (R-TPL-03).

## 5. Penyusun: daftarkan nomor dokumen

### Langkah 12 · Register nomor
Akun `ppap.engineer` juga memegang peran Author. Buka menu **Documents → Numbers**.

1. Di **Register a number**, pilih **Customer** `LM`.
2. **Register as a set: PFD, PFMEA, and CP with one number** sudah dicentang. Biarkan tercentang supaya ketiga dokumen mendapat nomor berjalan yang sama, misalnya SH-LM-001, SF-LM-001, dan SC-LM-001.
3. **Initial revision**: biasanya `0`. Isi angka lain hanya untuk dokumen lama yang masuk di revisi tertentu.
4. Layar menampilkan **Next number** sebelum Anda menekan tombol. Tekan **Register**.
5. Nomor masuk ke daftar **Register** dengan status **Reserved**. Nomor tidak pernah dihapus atau dipakai ulang.
6. Hilangkan centang set untuk mendaftarkan satu nomor saja. Pilih **Document type**-nya.

Register bisa disaring dengan **Customer**, **Type**, **Status**, dan **Number contains**.

## 6. Dokumen kontrol: status nomor dan impor

### Langkah 13 · Void, obsolete, dan impor
Logout, lalu login dengan `ppap.staff` dan ganti password seperti langkah 8.

1. **Documents → Numbers**: setiap baris register punya tombol **Void** dan **Make obsolete**. Isi alasan (wajib), lalu tekan **Confirm**.
2. **Documents → Import legacy numbers**:
   - pilih customer dan **Status of these numbers**;
   - tempel kolom nomor dari register Excel lama, satu nomor per baris;
   - tekan **Preview** dan periksa laporannya: nomor yang akan diimpor, baris yang ditolak beserta alasannya, celah nomor, dan set yang tidak sejajar;
   - bila sudah benar, tekan **Import**.

## 7. Admin: periksa jejak

### Langkah 14 · Audit log
Logout, lalu login lagi sebagai `admin.contoh`. Buka **Admin → Audit log**.

1. Setiap langkah di atas tercatat: siapa, kapan, apa, dan akun mana (termasuk orang pemegangnya).
2. Tekan **Check the chain** untuk memastikan catatan tidak pernah diubah. Audit log tidak bisa diedit siapa pun, juga lewat database.

## 8. Bila ada masalah

| Masalah | Penyebab yang biasa | Yang dilakukan |
|---|---|---|
| Menu atau tombol tidak muncul | Akun tidak punya peran itu | Login admin, buka Accounts, tambah perannya |
| Akun bercakupan Own project group tidak melihat customer apa pun | Orangnya belum dicatat sebagai pemegang jabatan | Positions → Add holder dengan project group |
| Password sementara hilang | Hanya tampil sekali | Admin mereset password akun itu di Accounts |
| Akun admin lupa password atau terkunci | — | `go run ./cmd/revioten reset-admin-password -login admin.contoh` |
| Salah 10 kali saat login | Akun dikunci 15 menit | Tunggu 15 menit |
| Halaman kosong atau galat setelah update kode | Server lama masih berjalan | Ctrl+C di jendela server, lalu `go run ./cmd/revioten` lagi |

## 9. Yang belum ada di sistem (urutan milestone)

Langkah ini akan ditambahkan ke panduan saat milestone-nya selesai.

| Milestone | Langkah yang akan muncul |
|---|---|
| M3.4 | Menerbitkan versi process template (Draft → Published) dan versi baru |
| M4.x | Membuat draft PFD, PFMEA, dan CP dari nomor yang sudah didaftarkan, lalu merakit baris dari process template untuk satu model (varian otomatis terpilih) |
| M4.5 | Doc ref terisi dari master dokumen terkait (D-130) |
| Tahap 5 | Permintaan perubahan, verifikasi, dan draft revisi |
| Tahap 6 | Cetak untuk tanda tangan basah, catat dokumen kembali, dan catat rilis (pencatat rilis harus berbeda dari penyusun) |
