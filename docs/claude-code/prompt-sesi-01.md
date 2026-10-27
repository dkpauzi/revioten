# Prompt sesi 1 · Orientasi, evaluasi awal desain, dan M0.1

## Sebelum menempel prompt

1. Salin isi paket ini ke root repo kode yang private di GitHub. Hasilnya:
   ```
   CLAUDE.md
   docs/brief/pfmea-system-design-brief.md
   docs/milestone/milestone.md
   docs/milestone/Milestone_Pembangunan_Web_PFMEA_Control_Plan_dan_PFD.docx
   docs/uml/            (paket UML draft 0.7)
   docs/evaluasi-desain.md
   docs/claude-code/    (file ini dan prompt-milestone.md)
   ```
2. Kalau repo sudah punya CLAUDE.md lama yang pernah di-commit, versi lamanya tetap tersimpan di riwayat git. Claude Code akan melaporkannya di langkah 2.
3. Jangan masukkan dokumen asli customer dulu. Dokumen itu masuk setelah `.gitignore` untuk folder `data/` dibuat di M0.1.
4. Commit, lalu push.
5. Buka Claude Code di repo itu, lalu tempel prompt di bawah.

## Prompt

```text
Ini sesi pertama coding untuk sistem PFD, PFMEA, dan CP. Mulai sekarang kode dibangun sejajar dengan UML dan dipakai untuk mengevaluasi desain (D-77). Sesi ini hanya mencakup orientasi, evaluasi awal desain, dan milestone M0.1.

Langkah 1 · Baca
- CLAUDE.md.
- Brief docs/brief/pfmea-system-design-brief.md: bagian 0, 1, 2, 5 (terutama SEC dan NUM), 6, 7, 14, dan 15. Bagian 16 cukup sekilas.
- Milestone docs/milestone/milestone.md: "Cara memakai dokumen ini", "Hasil pengecekan versi", Tahap S, Tahap 0, dan Tahap 1.
- UML: docs/uml/README.md, U1-glosarium.md, U3-daftar-use-case.md, dan U3-peta-pengguna.md.

Langkah 2 · Periksa repo dan lingkungan
Hanya membaca. Jangan memasang atau mengubah apa pun.
- Isi repo: git status, riwayat commit singkat, dan daftar folder dua tingkat. Laporkan file di luar paket ini, misalnya paket spesifikasi lama atau CLAUDE.md lama di riwayat git. Jangan menghapus apa pun.
- Lingkungan: OS, apakah sesi ini berjalan di komputer saya atau di cloud, versi git, go, node, npm, docker, dan docker compose, serta apakah PostgreSQL bisa dijalankan di Docker. Bila ada yang belum terpasang, catat saja.

Langkah 3 · Evaluasi awal desain
Dari bacaan di langkah 1, cari hal yang akan menghambat atau membingungkan saat membangun Tahap 0 dan Tahap 1. Yang dicari: kontradiksi antardokumen, aturan yang dibutuhkan kode tapi belum ada, aturan yang ambigu, dan aturan yang sulit dibangun.
- Tulis paling banyak 10 temuan ke docs/evaluasi-desain.md dengan format di file itu (E-01 dan seterusnya), urut dari dampak terbesar.
- Tulis hanya temuan yang benar-benar ada di dokumen, dengan rujukan ID-nya. Bila tidak ada temuan, tulis bahwa tidak ada.
- Jangan mengubah brief, UML, atau milestone.

Langkah 4 · Laporan dan rencana, lalu berhenti
Kirim ke saya dalam bahasa Indonesia:
a. Pemahaman Anda tentang proyek ini dalam 5 sampai 8 butir, supaya saya bisa mengecek apakah ada yang salah tangkap.
b. Hasil langkah 2.
c. Daftar temuan langkah 3: ID, judul, dan apakah menghambat.
d. Rencana M0.1: konsep yang akan dijelaskan, file yang akan dibuat atau diubah, perintah yang akan dijalankan, dan apa yang harus saya pasang sendiri di komputer saya.
Lalu tunggu persetujuan saya. Jangan menulis kode atau memasang apa pun sebelum saya setuju.

Langkah 5 · Kerjakan M0.1 setelah saya setuju
- Jelaskan singkat konsep toolchain dan penguncian versi.
- Cek ulang setiap versi di brief bagian 7 terhadap R-VER-01 sampai R-VER-05, memakai go list -m -versions, npm view, dan halaman rilis resmi. Tulis hasilnya di docs/versions.md: komponen, versi yang dipakai, tanggal rilis, tanggal dicek, sumber, dan catatan. Bila ada versi yang berbeda dari brief, jangan langsung diganti. Laporkan dan catat sebagai temuan.
- Buat .gitignore yang setidaknya mengecualikan data/, .env, dan hasil build. R-SEC-08 masih usulan, jadi sampai diputuskan dokumen asli hanya disimpan di data/ dan tidak di-commit.
- Siapkan PostgreSQL 18.6 dengan Docker Compose untuk localhost (D-36). Password diambil dari .env yang tidak di-commit, dan sediakan .env.example.
- M0.1 selesai bila go version, node -v, dan PostgreSQL 18 di Docker berjalan. Tunjukkan buktinya.
- Isi bagian "Perintah" di CLAUDE.md dengan perintah yang sudah terbukti jalan.

Langkah 6 · Tutup M0.1, lalu berhenti
- Tulis docs/catatan/M0.1-alat-dan-versi.md sesuai CLAUDE.md bagian 9, termasuk satu latihan tulis tangan untuk saya.
- Commit dengan ID di pesannya, misalnya "M0.1: cek ulang versi dan PostgreSQL 18.6 di Docker Compose".
- Kirim ringkasan singkat dan temuan desain baru. Jangan lanjut ke M0.2.
```
