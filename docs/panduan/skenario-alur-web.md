# Skenario alur web: dari nol sampai dokumen draft

Cara membangun web ini sendiri (dari sisi kode) ada di `docs/panduan/membangun-manual.md`.

- **Dibuat:** 8 November 2026, setelah M3.3. **Diperbarui:** 21 November 2026 (peran D-199, tanda tangan digital D-198, dasbor admin D-205, Form Control Submission D-207, master data milik PPAP Engineer D-209). Diperbarui lagi setiap milestone menambah layar.
- **Untuk siapa:** pemilik, untuk mencoba seluruh sistem dari database kosong secara berurutan.
- **Isinya:** semua layar yang sudah dibangun. Langkah yang belum ada di sistem ditulis di bagian 13.
- **Nama orang:** semua nama di sini contoh. Jangan memakai nama orang asli di data uji.

## Versi otomatis (tur)

Skenario ini juga bisa ditonton: browser terbuka sendiri, mengisi formulir dan memilih menu langkah demi langkah, dengan keterangan langkah di pita bawah layar.

Aplikasi yang dipakai: **Docker Desktop** (database) dan **PowerShell** (mengetik perintah). Browser untuk tur dibuka sendiri oleh tur, jadi Chrome atau Edge Anda tidak perlu dibuka. Server (`go run ./cmd/revioten`) juga tidak perlu dijalankan, karena tur menyalakan servernya sendiri.

1. **Nyalakan Docker Desktop.** Tekan tombol Windows, ketik `Docker Desktop`, lalu buka. Tunggu sampai di pojok kiri bawah tertulis **Engine running**. Jendelanya boleh diperkecil, tetapi jangan ditutup.
2. **Buka PowerShell.** Tekan tombol Windows, ketik `PowerShell`, lalu buka **Windows PowerShell** (atau **Terminal**). Terminal di VS Code atau di panel Terminal aplikasi Claude juga bisa.
3. **Masuk ke folder proyek.** Ketik perintah ini lalu tekan Enter:
   ```powershell
   cd C:\E_Project\revioten-main
   ```
   Untuk memeriksa, ketik `ls`. Harus tampil antara lain `CLAUDE.md`, `web`, dan `compose.yaml`. Semua perintah berikutnya diketik di jendela ini, tanpa pindah folder lagi.
4. **Periksa Node.js.** Ketik `node -v`. Harus tampil `v24...`. Bila muncul pesan "not recognized", Node belum aktif di jendela ini; tutup PowerShell, buka lagi, lalu ulangi dari langkah 3.
5. **Nyalakan database:**
   ```powershell
   docker compose up -d
   ```
   Periksa dengan `docker compose ps`: kolom STATUS harus memuat `healthy`.
6. **Jalankan tur.** Di PowerShell ketik `npm.cmd`, bukan `npm` (lihat tabel masalah di bawah):
   ```powershell
   npm.cmd --prefix web run tour
   ```
7. **Tunggu kira-kira satu menit.** Aplikasi dibangun dulu, lalu jendela browser (Chromium) muncul sendiri. Keterangan setiap langkah tampil di pita bawah layar. Jangan klik apa pun di jendela itu selama tur berjalan.
8. **Selesai:** jendela browser tertutup sendiri, dan PowerShell menulis `1 passed`. Bila ada langkah yang gagal, PowerShell menulis `1 failed` beserta langkahnya; salin teks itu dan kirimkan ke Claude.
9. **Menghentikan tur di tengah jalan:** klik jendela PowerShell, lalu tekan `Ctrl+C`.

Tur memakai database terpisah bernama `revioten_e2e` di port 8090, yang dibuat ulang dari kosong setiap kali. Database kerja Anda di port 8080 tidak disentuh, jadi tur aman dijalankan kapan saja.

Rekaman videonya tersimpan di folder `C:\E_Project\revioten-main\web\test-results\` (file `video.webm`). Buka folder itu di File Explorer, lalu seret file video ke jendela Chrome atau Edge untuk memutarnya.

**Bila ada masalah:**

| Pesan | Artinya | Yang dilakukan |
|---|---|---|
| `error during connect` atau `docker: not recognized` | Docker Desktop belum menyala | Ulangi langkah 1, tunggu **Engine running** |
| `npm.ps1 cannot be loaded because running scripts is disabled` | PowerShell memblokir skrip `npm.ps1` | Ketik `npm.cmd` sebagai ganti `npm`. Pengaturan Windows tidak perlu diubah |
| `npm: not recognized` | Node belum aktif di jendela ini | Ulangi langkah 4 |
| `Cannot find module` atau `playwright: not recognized` | Paket web belum terpasang | Jalankan sekali `npm --prefix web ci`, lalu `npm --prefix web exec -- playwright install chromium`, lalu ulangi langkah 6 |
| `port 8090 is already in use` | Tur atau test E2E sebelumnya masih jalan | Tutup jendela PowerShell lama, atau tunggu sampai selesai |

Kecepatan diatur lewat variabel `TOUR_SPEED` di PowerShell:

| Perintah | Hasil |
|---|---|
| `$env:TOUR_SPEED="slow"; npm.cmd --prefix web run tour` | Pelan, tiap keterangan ditahan 6 detik |
| `npm.cmd --prefix web run tour` | Normal, kira-kira 20 menit |
| `$env:TOUR_SPEED="fast"; npm.cmd --prefix web run tour` | Tanpa jeda, untuk memeriksa bahwa tur masih jalan |

Setelah memakai `TOUR_SPEED`, kembalikan dengan `Remove-Item Env:TOUR_SPEED`.

Tur ini memakai Playwright, alat yang sama dengan test E2E di `web/e2e/`. Skripnya ada di `web/tour/alur.tour.ts` dan bisa dibaca seperti daftar langkah. Dua alat Playwright lain yang berguna untuk belajar:

| Perintah (dari folder `web`) | Gunanya |
|---|---|
| `npx playwright test --ui` | Menjalankan test E2E satu per satu sambil melihat layarnya di setiap langkah, bisa dimundurkan |
| `npx playwright codegen http://127.0.0.1:8080` | Merekam klik dan ketikan Anda sendiri menjadi skrip test. Butuh server yang sedang berjalan |

## 0. Gambaran besar

Urutannya mengikuti siapa yang boleh melakukan apa:

```
[Admin]            1. Login admin (Home akun admin = Web health)
                   2. Akun kerja + peran (PPAP Engineer, Supervisor, Project Leader, Manager)
                   3. Settings (pengaman: mode migrasi), lalu logout
[PPAP Engineer]    4. Login pertama dan ganti password
                   5. Project group dan customer
                   6. Model group, model, part number
                   7. Jabatan dan pemegangnya, rute default customer, Document settings
                   8. Jenis dokumen, format template (cek isi awal, terbitkan)
                 9-12.  Master proses, process template, versi
                13-14.  Nomor dokumen: daftarkan, Void/obsolete, impor
                15-21.  Form set, dokumen draft, rakit, cek PFMEA-CP, diagram
                22-23.  Cetak dan rilis dengan scan (tanda tangan basah)
                24-27.  Perubahan, verifikasi, revisi 1, rilis
                28-30.  Dokumen lama dan dasbor
                31-33.  Dokumen terbit, review 3 tahun, PPAP performance
                34-36.  Tanda tangan digital, Form Control Submission, Report a problem
[Admin]           37-40.  Web health, Roles, API integrasi, email, audit log
```

Mengapa admin lebih dulu: akun hanya bisa dibuat akun admin (hak A-1). Sesudah itu admin selesai. Sejak D-209 master data (customer, model, jabatan, rute, format template, form set, pengaturan dokumen) dikerjakan PPAP Engineer lewat hak E-11. Admin hanya melihatnya, dan tetap memegang akun, peran, kesehatan web, audit log, email, service account, serta dua pengaman: mode migrasi dan aturan penyetujuan form.

## 1. Peran dan fungsinya

Setiap akun yang login otomatis bisa **melihat** dokumen rilis, register nomor, dan master data (hak L-1 sampai L-3). Peran menambah hak di atas itu. Satu akun boleh punya beberapa peran.

Sejak D-199 peran adalah matriks yang diatur admin di **Admin → Roles**: admin bisa menambah peran, mengganti nama, menonaktifkan, dan mencentang hak setiap peran. Isi awalnya:

| Peran | Untuk siapa | Yang bisa dilakukan | Hak utama |
|---|---|---|---|
| PPAP Engineer | PPAP Engineer dan PPAP Staff | Master data (customer, model, jabatan, rute, format template, form set, layout, pengaturan dokumen), menyusun dan mencetak dokumen, mengajukan perubahan, master proses dan process template, dokumen kontrol (rilis, status nomor, impor, salinan terkendali), migrasi, tanda dicek PFMEA–CP, tanda tangan digital, laporan kinerja | L-1 sampai L-5, E-1 sampai E-11, V-2, S-1, G-1 |
| PPAP Engineer Supervisor | APQP/PPAP Controller | Semua hak PPAP Engineer, ditambah verifikasi perubahan dan penyetujuan form set serta tabel AP | + V-1, V-3 |
| Project Leader | Project Leader (Supervisor) dan Project Engineer (Leader) | Menyusun dan mencetak, mengajukan dan memverifikasi perubahan, tanda tangan digital. Bawaannya hanya customer Project Group sendiri; bisa dibuat semua customer (D-206) | L-1 sampai L-4, E-1 sampai E-5, V-1, S-1, G-1 |
| Manager | NPI Manager, QA QC Manager, GM | Tanda tangan digital, membaca draft, laporan kinerja | L-1 sampai L-5, S-1, G-1 |
| Viewer | Internal audit, QA, QC, produksi | Melihat dokumen terbit. Tidak perlu dicentang, karena semua akun kerja sudah punya | L-1 sampai L-3, G-1 |
| Admin | Akun admin terpisah | Akun, peran, Web health, audit log, email, service account, pengaman (mode migrasi, aturan penyetujuan form). Master data hanya dilihat (D-209) | A-1, A-3, A-4 |

**Cakupan peran.** Untuk setiap peran yang dicentang ada pilihan cakupan:
- **All customers**: berlaku untuk semua customer. Untuk PPAP Engineer dan PPAP Staff, juga Project Leader yang membantu Project Group lain (D-206).
- **Own project group**: hanya customer di Project Group tempat orang itu tercatat sebagai Project Leader atau Project Engineer (D-71). Bawaan untuk peran Project Leader. Cakupan ini baru berfungsi setelah orangnya dicatat sebagai pemegang jabatan (langkah 7).

**Aturan yang tidak bisa dimatikan admin (R-SEC-11).** Akun admin tidak pernah memegang peran kerja. Orang yang memegang akun admin juga punya akun kerja terpisah untuk pekerjaan sehari-hari. Akun tidak pernah dihapus, hanya dinonaktifkan.

**Isi awal yang disarankan untuk uji coba** (jabatan mengikuti U3 bagian 2):

| Akun contoh | Jenis akun | Peran | Cakupan |
|---|---|---|---|
| `admin.contoh` | Admin account | Admin (otomatis) | — |
| `ppap.engineer` | Work account | PPAP Engineer | All customers |
| `ppap.staff` | Work account | PPAP Engineer | All customers |
| `pl.gr1` (Project Leader GR1) | Work account | Project Leader | Own project group |
| `pe.gr1` (Project Engineer GR1) | Work account | Project Leader | Own project group |
| `apqp.controller` (APQP/PPAP Controller) | Work account | PPAP Engineer Supervisor | All customers |
| `npi.manager` | Work account | Manager | All customers |
| `qa.manager` | Work account | Manager | All customers |

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

Password minimal 8 karakter (D-132). Kalimat pendek yang mudah diingat tetap paling aman, misalnya `kopi pagi di pabrik`. Password yang sangat umum seperti `password1` ditolak. Password yang memuat nama login, nama orang, atau kata `revioten` ditolak.

## 3. Admin: login, akun, dan pengaman

### Langkah 1 · Login sebagai admin
1. Di halaman **Log in**, isi **Login name** `admin.contoh` dan **Password**, lalu tekan **Log in**.
2. Home akun admin adalah **Web health**. Menu atas berisi Home, Documents, Template library, Master data (hanya dilihat), dan Admin.
3. Di HP atau tablet (layar di bawah 1024px), menu terlipat di tombol **Menu**.
4. Tombol **EN | ID** di bilah atas mengganti bahasa layar.

### Langkah 2 · Buat akun kerja
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
6. Layar bergulir ke kotak **Temporary password for ppap.engineer** di bagian atas, dan pop-up di pojok kanan atas menampilkan pesan yang sama. Kata sandi sementara ini **otomatis disalin ke clipboard**, jadi cukup tempel dengan Ctrl+V di tempat Anda mencatatnya. Bila penyalinan ditolak browser, tekan **Copy password**. Kata sandi ini tampil satu kali saja dan wajib diganti orangnya saat login pertama.
7. Ulangi untuk akun lain di tabel bagian 1. Untuk tanda tangan digital (langkah 34) dibutuhkan `pl.gr1` (Project Leader), `npi.manager` dan `qa.manager` (Manager).

Di daftar **All accounts**, admin bisa mengubah peran (**Edit roles**, lalu **Save roles**), mereset password (**Reset password**), atau menonaktifkan akun (**Disable**). Akun tidak pernah dihapus.

### Langkah 3 · Pengaman di Settings, lalu logout
Menu **Admin → Settings**. Halaman ini hanya berisi dua pengaman (D-209):

1. **Migration mode**: nyalakan selama dokumen lama dimasukkan (langkah 28), lalu tekan **Save settings**. Bawaannya mati.
2. **Approval of form sets and AP tables**: bawaannya semua penyetuju form harus setuju. Biarkan.
3. Tekan **Log out** di kanan atas. Layar kembali ke halaman login.

Pengaturan dokumen (tanggal di slot, QR, nomor form) tidak ada di sini lagi, tetapi di **Master data → Document settings** milik PPAP Engineer (langkah 7).

## 4. PPAP Engineer: master data dan template

### Langkah 4 · Login pertama dan ganti password
1. Login dengan `ppap.engineer` dan password sementara dari langkah 2.
2. Sistem langsung membuka **Change password** dengan pesan bahwa password harus diganti.
3. Isi tiga kotak:
   - **Current password**: password sementara;
   - **New password**: password baru Anda (aturannya di bagian 2);
   - **Repeat the new password**: ketik ulang.
4. Tekan **Change password**. Sesi lain milik akun ini otomatis berakhir.
5. Menu **Change password** juga ada di bilah atas, untuk mengganti password kapan saja.

Menu akun ini berbeda dari menu admin: tidak ada kelompok Admin, dan halaman master data bisa diubah (hak E-11, D-209).

### Langkah 5 · Project group dan customer
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

### Langkah 6 · Model group, model, dan part number
Menu **Master data → Models**.

1. Pilih **Customer** `LM`.
2. **Model groups**: isi **Model group name** (misalnya `Seri K`), lalu tekan **Add model group**. Group model boleh dilewati, karena model tanpa group tetap bisa dipakai.
3. **Models**: isi **Model name** (misalnya `K-125`) dan pilih **Model group** `Seri K` atau **No group**, lalu tekan **Add model**. Buat juga satu model tanpa group, misalnya `K-200`, untuk mencoba prioritas varian nanti.
4. **Part numbers**: tambahkan part number untuk model yang dipilih (boleh lebih dari satu).

### Langkah 7 · Jabatan, rute, dan pengaturan dokumen
Menu **Master data → Positions**.

1. Jabatan sudah terisi dari bagan NPI: PPAP Engineer, PPAP Staff, Project Engineer, Project Leader, APQP/PPAP Controller, NPI Manager, QA QC Manager, Advisor, dan Factory Manager. Namanya bisa diganti lewat **Rename**, dan ID-nya tetap.
2. Catat pemegang:
   - pilih **Position** dan **Person**;
   - untuk Project Leader dan Project Engineer, pilih juga **Project group**, karena keduanya dipegang satu orang per project group;
   - tekan **Add holder**.
3. Ini penting untuk akun bercakupan **Own project group**: tanpa catatan pemegang, akun itu tidak punya customer apa pun.
4. Slot tanda tangan rute juga mengikuti jabatan: catat `ppap.engineer` sebagai PPAP Engineer, `apqp.controller` sebagai APQP/PPAP Controller, `pl.gr1` sebagai Project Leader GR1, `npi.manager` sebagai NPI Manager, dan `qa.manager` sebagai QA QC Manager.
5. **Master data → Approval routes**: pilih rute default customer LM, misalnya **Up to QA QC Manager**. Dokumen yang dibuat sesudahnya memakai rute ini; tanpa rute, dokumen tidak bisa dikirim untuk tanda tangan.
6. **Master data → Document settings**: isi nomor form Form Control Submission (D-207). Saklar tanggal di slot tanda tangan dan QR di cetakan juga ada di sini (D-209).

### Langkah 8 · Periksa isi awal dan terbitkan format template
Halaman berikut sudah terisi saat database dibuat. Cukup diperiksa, dan diubah bila berbeda dengan WI.

| Menu | Isi awal | Kapan diubah |
|---|---|---|
| Master data → Document types | PFD (huruf H), PFMEA (F), CP (C), kode pabrik, pola nomor, digit revisi, kode khusus ZZ dan HSE | Bila pola nomor di WI berbeda. Perubahan hanya berlaku untuk nomor baru |
| Master data → Change types | Jenis perubahan (Content, Layout only, Administrative), sumber perubahan, kategori 4M | Bila ada sumber perubahan baru |
| Master data → Approval routes | Rute 5 slot: Prepared, Checked, lalu tiga Approved (NPI Manager, QA / QC Mgr., General Mgr.) | Atur **Default route** per customer, atau rute khusus per jenis perubahan |
| Template library → Format templates | Format PFD, PFMEA (misalnya PFMEA AIAG 4th), dan CP: kolom, field header, tabel skala, simbol, aturan sorot | Ganti label kolom mengikuti form yang dipakai, isi teks tabel skala S, O, D |
| Master data → Document settings | Saklar tanggal tanda tangan, QR, nomor form Form Control Submission | Sesuai kebiasaan cetak |
| Master data → Customers | Kolom **Signatures** (Digital allowed / Wet only) dan **Customer submission** (Required / Not required) | Sesuai kesepakatan dengan customer |

Kolom nomor operasi, proses, dan Process Flow di format template tidak bisa diketik, karena isinya diambil dari baris dan dari PFD (D-122, D-123).

Lalu terbitkan versi 1 format PFD, PFMEA AIAG 4th, dan CP: di **Template library → Format templates**, pilih formatnya, tekan **Publish version 1**, lalu OK. Form set baru bisa terbit sesudah ketiga formatnya terbit (langkah 15).

### Langkah 9 · Master proses
Menu **Template library → Processes**.

1. Di **Add a process**, isi **Process name** dengan nama yang dipakai di WI, misalnya `Laser marking`.
2. **Aliases** (boleh kosong): nama lain proses ini di dokumen lama, satu per baris, misalnya `Laser mark` dan `LM marking`. Alias membantu impor dokumen lama nanti.
3. Tekan **Add process**.
4. Ulangi untuk proses lain, misalnya `SMT`, `Check Qty`, dan `Packing`.

Dokumen menunjuk proses lewat ID tetapnya, jadi mengganti nama proses lewat **Edit** tidak merusak dokumen (D-06).

### Langkah 10 · Process template
Menu **Template library → Process templates**.

1. **Process**: pilih `Laser marking`. Bila belum punya template, tekan **Add process template**. Satu proses hanya punya satu template (D-129). Template baru dibuat sebagai **Version 1 · Draft** (lihat langkah 12).
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

### Langkah 12 · Terbitkan versi, lalu buat versi baru
Template baru berstatus **Draft**. Selama draft, isinya bebas diubah.

1. Di baris **Version** (di atas daftar sub-proses), tekan **Publish version 1**.
2. Jawab **OK** pada pertanyaan konfirmasi. Sesudah terbit, versi itu tidak bisa diubah lagi oleh siapa pun.
3. Label berubah menjadi **Published**, dan tombol tambah serta ubah hilang.
4. Untuk mengubah isi template sesudah terbit, tekan **New version**. Sistem membuat **Version 2 · Draft** sebagai salinan versi 1, dengan ID sub-proses dan poin yang sama.
5. Kotak pilihan **Version** dipakai untuk membuka kembali versi lama (hanya bisa dibaca).

Satu template paling banyak punya satu draft, jadi **New version** baru muncul lagi setelah versi 2 diterbitkan. Format template punya baris versi yang sama; yang menerbitkannya akun admin.

## 5. Penyusun: daftarkan nomor dokumen

### Langkah 13 · Register nomor
Akun `ppap.engineer` memegang peran PPAP Engineer. Buka menu **Documents → Numbers**.

1. Di **Register a number**, pilih **Customer** `LM`.
2. **Register as a set: PFD, PFMEA, and CP with one number** sudah dicentang. Biarkan tercentang supaya ketiga dokumen mendapat nomor berjalan yang sama, misalnya SH-LM-001, SF-LM-001, dan SC-LM-001.
3. **Initial revision**: biasanya `0`. Isi angka lain hanya untuk dokumen lama yang masuk di revisi tertentu.
4. Layar menampilkan **Next number** sebelum Anda menekan tombol. Tekan **Register**.
5. Nomor masuk ke daftar **Register** dengan status **Reserved**. Nomor tidak pernah dihapus atau dipakai ulang.
6. Hilangkan centang set untuk mendaftarkan satu nomor saja. Pilih **Document type**-nya.

Register bisa disaring dengan **Customer**, **Type**, **Status**, dan **Number contains**.

## 6. Dokumen kontrol: status nomor dan impor

### Langkah 14 · Void, obsolete, dan impor
Logout, lalu login dengan `ppap.staff` dan ganti password seperti langkah 4.

1. **Documents → Numbers**: setiap baris register punya tombol **Void** dan **Make obsolete**. Isi alasan (wajib), lalu tekan **Confirm**.
2. **Documents → Import legacy numbers**:
   - pilih customer dan **Status of these numbers**;
   - tempel kolom nomor dari register Excel lama, satu nomor per baris;
   - tekan **Preview** dan periksa laporannya: nomor yang akan diimpor, baris yang ditolak beserta alasannya, celah nomor, dan set yang tidak sejajar;
   - bila sudah benar, tekan **Import**.

## 7. Penyusun: dokumen draft (Tahap 4)

Akun `ppap.engineer` di contoh ini memegang peran PPAP Engineer (D-199). Form set diterbitkan PPAP Engineer Supervisor (V-3); bawaannya semua pemegang V-3 harus setuju (D-148).

### Langkah 15 · Terbitkan form set
Dokumen hanya bisa dibuat dari form set customer yang sudah terbit.

1. Sebagai admin, buka **Template library → Format templates**. Pilih **PFD · PFD**, tekan **Publish version 1**, lalu jawab **OK**. Ulangi untuk **PFMEA AIAG 4th · PFMEA** dan **CP · CP**.
2. Login sebagai `apqp.controller` (PPAP Engineer Supervisor, pemegang V-3), buka **Template library → Form sets**. Pada **AIAG 4th set**, tekan **Publish version 1**, lalu jawab **OK**. Penyusun draft form set tidak bisa menerbitkannya sendiri.

### Langkah 16 · Set nomor untuk dokumen
Di **Documents → Numbers**, daftarkan satu set untuk customer `LM` seperti langkah 13. Hasilnya, misalnya, SH-LM-002, SF-LM-002, dan SC-LM-002 berstatus Reserved.

### Langkah 17 · Buat dokumen
Menu **Documents → Documents**.

1. **Customer**: pilih `LM`.
2. **Document number**: pilih `SH-LM-002 · PFD`. Centang model `K-125`. Tekan **Make document**.
3. Ulangi untuk `SF-LM-002 · PFMEA` dan `SC-LM-002 · CP`. Nomornya berubah dari Reserved menjadi In use.
4. Tekan nomor dokumen di daftar untuk membukanya. Bagian atas berisi header; di bawahnya **workbook** dengan tab **PFD**, **PFMEA**, **CP**, dan **Check**.

### Langkah 18 · Rakit dari process template
1. Di dokumen CP, tekan **Compose from process templates**.
2. **Process template**: pilih `Laser marking · version 1` (hanya versi terbit yang muncul).
3. Kiri: centang sub-proses `Marking` untuk memakai semua poinnya, atau centang poin satu per satu. Poin tanpa centang boleh diberi alasan.
4. Kanan: periksa baris yang akan didapat PFD, PFMEA, dan CP.
5. Tekan **Apply**. Ketiga dokumen terisi sekaligus. Model K-125 mendapat teks varian `fiber 20 W`, dan isi kolom diambil dari **Columns** poin (langkah 10).

Cara lain mengisi baris:
- **Add row** menambah baris kosong yang diketik langsung di tabel.
- **Copy from another model** (hanya selama dokumen masih kosong) menyalin baris dokumen model serupa untuk seluruh set.
- **Paste from Excel** ada, tetapi dikesampingkan dulu (D-151).

### Langkah 19 · Ubah sel, urutkan, dan bandingkan
1. Klik sel di tabel, ketik, lalu klik di luar tabel. Barisnya tersimpan; pop-up "The row was saved." muncul. Bila orang lain lebih dulu menyimpan baris yang sama, perubahan Anda tidak disimpan dan isi terbaru ditampilkan (M4.3).
2. **Order and operation numbers**: seret langkah proses atau pakai tombol panah, centang penomoran otomatis (10, 20, 30), lalu **Save order**. Nomor yang Anda ketik sendiri tidak diubah.
3. **Compare with template**: setiap baris hasil perakitan tampil **Same as template**, **Different from template** (dengan kolom yang berbeda), atau **Detached**. **Detach** memutus tautan satu baris; alasannya wajib.

### Langkah 20 · Cek PFMEA dan CP
1. Tekan tab **Check**. PFMEA dan CP tampil berdampingan, satu proses demi satu proses.
2. Periksa bahwa setiap failure mode dan kontrol di PFMEA tercakup di CP.
3. Pemegang PFMEA–CP checker menekan **Mark as checked**. Tanda menyimpan nama dan waktunya untuk revisi saat ini.
4. Bila baris proses itu ditambah, diubah, atau dihapus sesudahnya, tanda tampil **Changed after the check**. Cek lagi dan tekan **Mark as checked** sekali lagi.

### Langkah 21 · Diagram PFD
Buka tab **PFD**, lalu tekan **Show the diagram**. Satu kotak per baris menurut urutan, bentuknya mengikuti **Flow Symbol**, dan **NG Disposition** digambar di samping langkahnya. Diagram hanya untuk dilihat; ubah barisnya di tabel.

## 8. Cetak dan rilis (Tahap 6)

Sebelum bagian ini, admin membuat dua akun lagi di **Admin → Accounts**: `dok.kontrol` (PPAP Engineer, yang kini juga mengerjakan dokumen kontrol) dan `apqp.controller` (PPAP Engineer Supervisor). Di **Master data → Positions**, catat pemegang `apqp.controller` sebagai **APQP/PPAP Controller**. Di **Admin → Settings**, centang **Migration mode**. Hak migrasi (E-10) sudah termasuk di peran PPAP Engineer.

### Langkah 22 · Cetak draft
Di dokumen SC-LM-002, tekan **Print** di toolbar, lalu **Excel**. Berkas `SC-LM-002-00-draft.xlsx` terunduh dengan tanda UNCONTROLLED COPY, dan cetakan itu tercatat di **Print log**.

### Langkah 23 · Catat rilis
Login sebagai `dok.kontrol`. Buka dokumen SC-LM-002, tekan **Revisions** di toolbar. Pilih scan bertanda tangan, tekan **Upload the scan**, lalu **Record release of revision 0**. Tanpa scan, tombol rilis mati. Penyusun tidak bisa merilis draftnya sendiri.

## 9. Perubahan dan revisi (Tahap 5)

### Langkah 24 · Ajukan perubahan
Login sebagai `ppap.engineer`. Buka **Documents → Changes**, pilih customer, isi **Triggered by**, **Type of change**, **4M category**, dan **Reason**, centang SC-LM-002, lalu **Submit change**. Nomornya CHG-0001.

### Langkah 25 · Verifikasi
Login sebagai `apqp.controller`. Buka CHG-0001, isi **Verification note**, lalu **Accept**. Dokumen terdampak mendapat draft revisi 1. Pemohon tidak bisa memverifikasi permintaannya sendiri.

### Langkah 26 · Ubah draft dan bandingkan
Login sebagai `ppap.engineer`. Ubah satu sel di SC-LM-002, lalu lihat **Compare revisions**: beda revisi 0 dan draft tampil per baris dan per sel.

### Langkah 27 · Rilis revisi 1 dan cetak
Login sebagai `dok.kontrol`. Unggah scan revisi 1, **Record release of revision 1**, lalu di **Print** pilih **Reference copy** dan tekan **Excel**.

## 10. Dokumen lama (Tahap 7)

### Langkah 28 · Masukkan dokumen lama
Login sebagai `ppap.engineer`. Daftarkan satu set nomor lagi. Di **Documents → Documents**, bagian **Enter a legacy document**: pilih SC-LM-003, centang K-200, isi **Current revision** 4, **Effective date**, dan nama di hardcopy, lalu **Enter the legacy document**. Buka dokumennya, tambah baris riwayat di **Revision history from the legacy table**, lalu simpan. Scan hardcopy boleh diunggah di **Revisions** sebagai bukti tambahan, tetapi tidak wajib (D-186).

### Langkah 29 · Verifikasi hardcopy
Login sebagai `dok.kontrol`. Buka SC-LM-003, isi **Minutes spent**, lalu **Verify against the hard copy**. Statusnya menjadi Released (migrated). Orang yang memasukkan dokumen tidak bisa memverifikasinya.

### Langkah 30 · Dasbor
Buka **Documents → Dashboard**: baris yang mengikuti template, dokumen per status, target yokoten, waktu revisi, dan usaha migrasi.

## 10a. Dokumen terbit, review berkala, dan kinerja (D-195 sampai D-197)

### Langkah 31 · Dokumen terbit
Buka **Documents → Released documents**, pilih customer LM. Setiap akun, juga akun Viewer tanpa peran kerja (internal audit, QA, QC, produksi), membaca revisi yang berlaku. Tekan nomor dokumen untuk membaca isinya, dan tab **Check** untuk PFMEA dan CP terbit berdampingan. Tidak ada tombol ubah, dan draft tidak pernah tampil.

### Langkah 32 · Review berkala
Buka **Documents → Periodic review**. Set PFD, PFMEA, dan CP yang 36 bulan tidak direvisi tampil kuning 90 hari sebelum jatuh tempo dan merah sesudahnya. Peringatan yang sama muncul di **Home**, di **Dashboard**, di halaman dokumen, dan sebagai email ke penyusun customer itu. Tekan tab **All active** untuk melihat semua set. Untuk set yang jatuh tempo, penyusun menekan **Start review**, lalu **Changes needed** (permintaan perubahan dengan pemicu Periodic review) atau **Reviewed, no change** (daftar periksa, peserta, kesimpulan), lalu verifikator menekan **Confirm**. Model yang sudah tidak diproduksi ditandai **Mark EOL** di Master data → Models.

### Langkah 33 · Laporan kinerja
Login sebagai `ppap.engineer`. Buka **Documents → PPAP performance**. Pilih **Week**, **Month**, atau **Year** (minggu laporan Jumat sampai Kamis), lalu **Download weekly report** untuk Excel: jumlah permintaan perubahan per pemicu (termasuk Customer claim dan Audit finding), yokoten, dan waktu dari diterima sampai rilis, dari mulai edit sampai siap cetak, dan dari cetak sampai ditandatangani, total dan per orang.

## 10b. Tanda tangan digital, Form Control Submission, dan laporan masalah (D-198, D-207, D-205)

### Langkah 34 · Kirim untuk tanda tangan digital
Login sebagai `ppap.engineer`. Buka draft SF-LM-002, tekan **Signature** di toolbar. Pilih **Process. Eng** (diri sendiri, atau Project Leader tingkat Leader) dan **Engineer** (Project Leader GR1); NPI Manager dan QA QC Manager terisi otomatis. Ketik password, lalu **Send for digital signature**. Isi draft dibekukan dengan SHA-256, tanggal revisi menjadi tanggal hari ini (D-204), dan draft terkunci.

### Langkah 35 · Tanda tangan berurutan, lalu rilis tanpa scan
Login berturut-turut sebagai `pl.gr1`, `npi.manager`, dan `qa.manager`. Masing-masing membuka **Documents → Signatures**, **Open and sign**, membaca dokumen, mengetik password, lalu **Sign**. Urutannya tidak bisa dilompati; **Reject** butuh alasan dan membuka draft lagi. Sesudah lengkap, login sebagai `dok.kontrol`, buka SF-LM-002, bagian **Revisions**, lalu **Record release of revision 0**, tanpa scan.

### Langkah 36 · Form Control Submission dan Report a problem
Masih sebagai `dok.kontrol`, gulir ke **Submission control**. Satu daftar per dokumen; tanggal mulai, selesai, dan tanda tangan terisi sendiri. Tekan **Edit** pada revisi 0, isi **Date sent to the customer**, **What was sent**, **Customer result**, **Result date**, dan **Remark**, lalu **Save**. **Download PDF** atau **Download Excel** mencetak form dokumen itu saja. Tekan **Report a problem** di bilah atas untuk mencoba laporan masalah ke admin.

## 11. Admin: periksa peran dan jejak (Tahap 8)

### Langkah 37 · Web health dan Roles
Login sebagai `admin.contoh`. Home akun admin adalah **Web health**: tanda tangan tertahan, error server, laporan masalah (laporan dari langkah 36 tampil di sini), dokumen yang diproses, email, akun, backup terakhir, dan rantai audit log. Rinciannya di **Admin → Web health**. Lalu buka **Admin → Roles**: setiap kolom satu peran, setiap baris satu hak. Kotak yang abu-abu terkunci: peran Admin dan Viewer, hak admin, dan hak dasar. **Add a role** menambah peran baru.

### Langkah 38 · Aplikasi lain (API integrasi, Tahap 9)
Buka **Admin → Service accounts**. Di **Add a service account**, isi nama aplikasi, tim pemilik, kontak, dan tujuan; centang scope **Read the list of released documents** dan **Read released revisions with their rows**, centang customer LM, lalu **Add the service account**. Tekan **Issue a token**: token tampil SATU KALI dan disalin ke clipboard. Aplikasi lain memanggil `GET /integration/v1/documents` dengan header `Authorization: Bearer <token>` dan hanya melihat dokumen terbit customer LM.

### Langkah 39 · Email notifikasi (D-194)
Tekan **Change password** di bilah atas, isi **Email for notifications**, lalu **Save the email address**. Admin juga bisa mengisi alamat orang lain di **Admin → Accounts** dengan tombol **Email**. Buka **Admin → Notifications**: server email (diatur IT), saklar per peristiwa, tombol **Send me a test email**, dan kotak keluar. Sebelum IT mengatur Microsoft 365, email hanya tercatat dengan status **Not sent**.

### Langkah 40 · Audit log
Buka **Admin → Audit log**.

1. Setiap langkah di atas tercatat: siapa, kapan, apa, dan akun mana (termasuk orang pemegangnya).
2. Tekan **Check the chain** untuk memastikan catatan tidak pernah diubah. Audit log tidak bisa diedit siapa pun, juga lewat database.

## 12. Bila ada masalah

| Masalah | Penyebab yang biasa | Yang dilakukan |
|---|---|---|
| Menu atau tombol tidak muncul | Akun tidak punya peran itu | Login admin, buka Accounts, tambah perannya |
| Akun bercakupan Own project group tidak melihat customer apa pun | Orangnya belum dicatat sebagai pemegang jabatan | Positions → Add holder dengan project group |
| Password sementara hilang | Hanya tampil sekali | Admin mereset password akun itu di Accounts |
| Akun admin lupa password atau terkunci | — | `go run ./cmd/revioten reset-admin-password -login admin.contoh` |
| Salah 10 kali saat login | Akun dikunci 15 menit | Tunggu 15 menit |
| Halaman kosong atau galat setelah update kode | Server lama masih berjalan | Ctrl+C di jendela server, lalu `go run ./cmd/revioten` lagi |
| Make document menolak: form set belum terbit | Form set customer belum diterbitkan PPAP Engineer Supervisor | Langkah 15 |
| Bagian Digital signature tidak menawarkan orang | Pemegang jabatan slot belum dicatat, atau orangnya belum punya peran dengan hak S-1 | Langkah 5 |
| Dokumen tidak bisa diubah | Draft sedang atau sudah ditandatangani digital | Tarik kembali (Withdraw) atau catat rilisnya |
| Tombol Compose tidak menampilkan proses | Process template belum punya versi terbit | Langkah 12 |
| Tab Check tidak ada | Dokumen bukan bagian set yang punya PFMEA dan CP | Daftarkan nomor sebagai set (langkah 16) |

## 13. Yang belum ada di sistem

| Milestone | Langkah yang akan muncul |
|---|---|
| Belum diputuskan | Konfirmasi slot saat rilis basah (R-APR-10) dan pembekuan rute saat cetak basah (R-APR-11); untuk tanda tangan digital keduanya sudah ada |
| Langkah L form set | Tampilan genba per proses dengan catatan (D-137); hak G-1 sudah ada |
| Tahap 10 | Portal customer (latihan) |
| M9.6 | Webhook (opsional, belum dibangun) |
