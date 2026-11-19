# Rancangan · Matriks peran dan hak (gaya Spatie Laravel Permission)

- **Dibuat:** 19 November 2026, dari permintaan pemilik.
- **Status:** **diputuskan 19 November 2026 (D-199)**, **dibangun 20 November 2026 di fase 2** (migrasi 00049, catatan `docs/catatan/D-199-fase-2-matriks-peran.md`). Jawaban pertanyaan bagian 7: dokumen kontrol digabung ke PPAP Engineer, Manager boleh melihat laporan kinerja, Project Leader boleh menyusun dan mencetak.

## 1. Permintaan pemilik

> "untuk pengaturan perannya saya merasa masih agak ribet. bagaimana jika di sediakan seperti matrix peran yang nanti bisa di ubah-ubah? bisa di bilang seperti package Spatie Laravel Permission, jadi role nya yang kita tetapkan dahulu, dan rolenya juga bisa di tambahkan nanti oleh admin. ... untuk request perubahaan, jadikan itu hak, jadi nanti kita bisa berikan hak itu ke peran tertentu, atau mencabut nya dari peran tertentu. buatkan matrix hak nya"

## 2. Cara kerjanya (sama dengan Spatie)

| Konsep | Spatie | Sistem ini |
|---|---|---|
| Hak (permission) | `permissions`, dicek kode lewat `can()` | Katalog tetap L-, E-, V-, S-, G-, A-. Hak baru = fitur baru, jadi butuh kode |
| Peran (role) | `roles`, kumpulan hak | Kumpulan hak. **Baru:** admin bisa menambah, mengganti nama, dan menonaktifkan peran |
| Akun memegang peran | `model_has_roles` | Satu akun bisa punya beberapa peran. Tambahan dari sistem ini: cakupan **semua customer** atau **Project Group sendiri** (D-71) |
| Matriks | (paket tambahan) | Layar Admin → Roles: baris = hak, kolom = peran, centang untuk memberi atau mencabut |

Yang sudah ada hari ini: katalog hak, layar Roles dengan centang (M8.1), dan "Ajukan perubahan" **sudah berupa hak sendiri (E-3)** yang bisa dicabut atau diberikan ke peran mana pun. Yang belum ada: menambah peran baru, dan peran yang mengikuti jabatan di pabrik.

## 3. Peran awal yang diusulkan

| Peran | Untuk siapa | Pengganti peran lama |
|---|---|---|
| **PPAP Engineer** | PPAP Engineer dan PPAP Staff | Penyusun, Pengelola template, Pemeriksa PFMEA–CP, Petugas migrasi, Dokumen kontrol (D-68) |
| **PPAP Engineer Supervisor** | APQP/PPAP Controller | Semua hak PPAP Engineer, ditambah Verifikator dan Penyetuju form |
| **Project Leader** | Project Engineer (Leader) dan Project Leader (Supervisor), cakupan Project Group sendiri (D-54, D-71) | Penyusun dan Verifikator bercakupan Project Group |
| **Manager** | Penanda tangan: NPI Manager, QA QC Manager, GM/Advisor | Viewer (D-70) |
| **Viewer** | Internal audit, QA, QC, produksi | Viewer |
| **Admin** | Akun admin terpisah (D-72) | Admin, tidak berubah |

Engineer dan Supervisor Project Leader **berhak sama**. Bedanya hanya di kolom tanda tangan, dan itu diatur oleh **jabatan** di rute approval (Master data → Positions dan Approval routes), bukan oleh peran. Jadi satu peran "Project Leader" cukup.

## 4. Matriks hak (isi awal, bisa diubah admin)

✓ = diberikan bawaan · — = tidak · **(PG)** = hanya customer di Project Group sendiri · 🔒 = terkunci (tidak bisa diubah di layar)

| Kode | Hak | PPAP Engineer | PPAP Eng. Supervisor | Project Leader | Manager | Viewer | Admin |
|---|---|:-:|:-:|:-:|:-:|:-:|:-:|
| **L-1** | Lihat dokumen terbit dan halaman verifikasi | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 |
| **L-2** | Lihat register nomor, riwayat, diff, dasbor, review berkala | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 |
| **L-3** | Ekspor dokumen terbit ke Excel dan PDF | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 | ✓🔒 |
| **L-4** | Lihat draft dan permintaan perubahan yang berjalan | ✓ | ✓ | ✓ (PG) | ✓ | — | — |
| **L-5** | Lihat laporan kinerja PPAP Engineer | ✓ | ✓ | — | ✓ | — | — |
| **E-1** | Daftarkan nomor dokumen | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-2** | Buat, rakit, dan edit draft | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-3** | **Ajukan permintaan perubahan** | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-4** | Draft revisi, yokoten, perubahan surut SQAM | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-5** | Cetak untuk tanda tangan / kirim ke tanda tangan digital | ✓ | ✓ | ✓ (PG) | — | — | — |
| **E-6** | Master proses dan process template | ✓ | ✓ | — | — | — | — |
| **E-7** | Catat rilis | ✓ | ✓ | — | — | — | — |
| **E-8** | Ubah status nomor, impor nomor lama | ✓ | ✓ | — | — | — | — |
| **E-9** | Salinan terkendali | ✓ | ✓ | — | — | — | — |
| **E-10** | Migrasikan dokumen lama | ✓ | ✓ | — | — | — | — |
| **V-1** | Verifikasi permintaan perubahan | — | ✓ | ✓ (PG) | — | — | — |
| **V-2** | Beri tanda dicek PFMEA–CP | ✓ | ✓ | — | — | — | — |
| **V-3** | Setujui form set dan tabel AP | — | ✓ | — | — | — | — |
| **S-1** | **Baru.** Tanda tangan digital pada slot rute approval | ✓ | ✓ | ✓ (PG) | ✓ | — | — |
| **G-1** | **Baru.** Beri catatan di tampilan genba (tidak mengubah dokumen) | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| **A-1** | Kelola akun, peran, dan isi peran | — | — | — | — | — | ✓🔒 |
| **A-2** | Kelola master data | — | — | — | — | — | ✓🔒 |
| **A-3** | Kelola pengaturan sistem | — | — | — | — | — | ✓🔒 |
| **A-4** | Lihat audit log | — | — | — | — | — | ✓🔒 |

Catatan matriks:
- **S-1** hanya membuka tombol tanda tangan. **Slot mana** yang boleh ditandatangani seseorang tetap ditentukan jabatannya di rute approval (R-APR-07, R-APR-08): Prepared, Checked, NPI Manager, QA QC Manager, GM.
- **E-3** bisa dicentang untuk Manager atau Viewer bila suatu saat mereka boleh mengajukan perubahan; tidak perlu kode baru.
- **G-1** baru berguna setelah tampilan genba (D-137, langkah L) dibangun; sampai 20 November 2026 tampilan genba belum ada.
- **Engineer di kolom tanda tangan** dipilih saat dokumen diajukan untuk tanda tangan, dari akun berperan PPAP Engineer atau Project Leader (D-203, fase 3).

## 5. Aturan yang tetap tidak bisa dimatikan (R-SEC-11)

Matriks boleh diubah, tetapi aturan ini tetap dijaga kode dan database, apa pun isi peran:
1. Pemohon tidak memverifikasi permintaannya sendiri.
2. Satu orang hanya mengisi satu slot dalam satu rute tanda tangan.
3. Pencatat rilis bukan penyusun draft.
4. Pencatat review berkala bukan pengonfirmasinya.
5. Peran kerja tidak pernah memegang hak admin (A-), dan akun admin tidak memegang peran kerja (D-72).
6. Hak dasar L-1 sampai L-3 selalu ada di setiap peran.

## 6. Yang berubah di sistem bila disetujui

1. Migrasi: peran baru di atas, hak S-1 dan G-1, dan pemindahan akun dari peran lama ke peran baru (tabel di bagian 3). Peran lama dinonaktifkan, tidak dihapus, supaya audit log tetap terbaca.
2. Admin → Roles: tombol **Add a role** (nama Inggris dan Indonesia), **Rename**, **Deactivate** (hanya bila tidak ada akun yang memegangnya), dan matriks centang yang sudah ada.
3. Admin → Accounts: pilihan peran mengikuti daftar peran aktif.
4. Help, tur, U3-peta-pengguna, dan brief (D- baru menggantikan isi awal D-66).

## 7. Pertanyaan untuk pemilik

1. Dokumen kontrol (catat rilis, salinan terkendali): digabung ke PPAP Engineer seperti usulan (sesuai D-68), atau tetap peran sendiri untuk staf DCC?
2. Manager melihat laporan kinerja (L-5)?
3. Project Leader boleh menyusun draft dan mencetak (E-1, E-2, E-5), seperti R-APR-07 yang menyebut Project Leader sebagai penyusun?
