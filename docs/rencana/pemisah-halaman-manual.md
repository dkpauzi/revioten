# Rancangan opsi 1 · Pemisah halaman manual di Control Plan

- **Status:** rancangan, **belum dibangun**. Pemilik (25 November 2026): "tetap rancang untuk opsi 1 namun jangan di bangun dulu code nya. kita akan evaluasi setelah saya melakukan tes manual."
- **Hubungan dengan opsi 2:** opsi 2 (D-226) sudah dibangun dan otomatis. Opsi 1 hanya dibangun bila tes manual pemilik menemukan halaman yang masih perlu diatur dengan tangan. Kalau dibangun, opsi 1 menjadi **pengecualian di atas opsi 2**, bukan penggantinya.
- **Yang disentuh:** `internal/printout/plan.go` (`Paginate`), isi baris dokumen (R-DOC-01), snapshot revisi (R-REV-02), diff (M5.6), workbook (D-145, D-210), pratinjau, PDF, dan Excel (D-211).

## 1. Masalah yang dijawab

Opsi 2 memindah baris pertama satu langkah ke halaman berikutnya bila baris itu jatuh di akhir halaman. Mungkin masih ada kasus yang ingin diatur sendiri. Contohnya, satu langkah yang panjang lebih enak dimulai di halaman baru walaupun masih muat, atau halaman disamakan dengan hardcopy lama saat migrasi.

## 2. Bentuk yang diusulkan

**Satu tanda per baris: "mulai halaman baru di baris ini".**

| Bagian | Usulan |
|---|---|
| Penyimpanan | Kolom baru `page_break boolean NOT NULL DEFAULT false` di `document_rows` (migrasi baru). Bukan di `content`, karena `content` divalidasi terhadap kolom format (R-DOC-01) dan tanda ini bukan isi form. |
| Snapshot | `SnapshotRow` mendapat `page_break` (`omitempty`). Revisi lama tanpa field ini dibaca sebagai false, jadi hash revisi lama tidak berubah. Cetakan revisi terbit memakai tanda yang dibekukan. |
| Diff | Perubahan tanda tampil sebagai perubahan baris dengan label "Page break". Usulan: tanda ini **tidak** memberi segitiga revisi di cetakan, karena isinya tidak berubah. |
| Workbook | Di tab CP, setiap baris punya aksi **Start a new page here** dan **Remove the page break** (menu baris). Garis putus-putus di atas baris menandai pemisah manual. |
| Pratinjau | Pemisah manual tampil sama dengan pemisah otomatis, ditambah label kecil "manual" di tepi halaman (hanya di pratinjau, tidak tercetak). |
| `Paginate` | Parameter baru `breakBefore []bool`. Baris bertanda selalu memulai halaman baru, lalu aturan opsi 2 tetap berjalan untuk sisa halaman. Halaman yang berakhir karena pemisah manual **tidak** diisi penuh, kecuali pemilik memilih lain (pertanyaan 2). |
| Excel | Pemisah manual memakai `InsertPageBreak` yang sama dengan pemisah otomatis. |
| PFMEA dan PFD | Tidak ikut. PFMEA tidak punya kolom alur; PFD dicetak sebagai diagram (D-219). |

**Aturan yang dijaga:**
- Tanda hanya bisa diubah di draft. Revisi terbit tidak bisa diubah (R-REV-02).
- Tanda di baris pertama dokumen atau baris pertama bagian per PCB diabaikan, karena baris itu sudah memulai halaman.
- Mengubah tanda dicatat di audit log seperti perubahan baris lain.

## 3. Risiko

- **Tanda menjadi salah saat baris berubah.** Bila baris ditambah atau dihapus di atasnya, pemisah yang tadinya tepat bisa jatuh di tengah halaman yang masih kosong. Mitigasi: pratinjau menandai pemisah manual yang membuat halaman sebelumnya terisi kurang dari separuh, dengan peringatan "page break leaves the page half empty".
- **Baris yang lebih tinggi dari satu halaman** tetap mendapat halaman sendiri, seperti sekarang.
- **Pekerjaan:** migrasi, query sqlc, `document`, snapshot, diff, API, workbook, pratinjau, test, Help, dan gambar Help. Kira-kira satu milestone kecil.

## 4. Pertanyaan untuk pemilik (dijawab sesudah tes manual)

1. Apakah sesudah opsi 2 masih ada halaman yang ingin diatur sendiri? Bila tidak, rancangan ini ditutup.
2. Halaman yang berakhir karena pemisah manual: dibiarkan berisi ruang kosong di bawah, atau baris terakhirnya dipertinggi seperti opsi 2?
3. Apakah perubahan tanda pemisah perlu tercatat di riwayat revisi dokumen, atau cukup di audit log?
