# Prompt untuk sesi berikutnya

Satu milestone per sesi. Ganti `<ID>` dan `<nama>`, misalnya `M0.2` dan `Server Go pertama`.

## Milestone biasa

```text
Kerjakan milestone <ID> <nama> dari docs/milestone/milestone.md.

1. Baca CLAUDE.md, catatan milestone terakhir di docs/catatan/, dan temuan yang masih Terbuka di docs/evaluasi-desain.md. Lanjutkan dari situ, dan jangan mengulang langkah yang sudah selesai.
2. Baca bagian brief, use case U3, dan peta pengguna yang disentuh <ID>. Sebutkan ID aturan dan keputusan yang akan ditegakkan beserta statusnya (DECIDED atau PROPOSED).
3. Tulis rencana: konsep baru yang akan dijelaskan, test yang ditulis lebih dulu, file yang dibuat atau diubah, dan temuan desain yang mungkin muncul. Tunggu persetujuan saya.
4. Setelah saya setuju: jelaskan konsepnya, tulis test dulu, tulis kode sampai test lulus, lalu rapikan.
5. Selama membangun, catat setiap temuan desain di docs/evaluasi-desain.md. Bila temuan menghambat, berhenti dan tanya saya dengan opsi. Bila tidak menghambat, pakai asumsi paling aman, tandai dengan // ASUMSI E-nn, lalu lanjut.
6. Tutup milestone: tulis docs/catatan/<ID>-<nama>.md sesuai CLAUDE.md bagian 9, commit dengan ID di pesannya, lalu kirim ringkasan singkat beserta temuan baru. Berhenti, dan jangan lanjut ke milestone berikutnya.
```

## Gerbang (akhir setiap tahap)

```text
Ini Gerbang <n>. Jangan menulis kode latihan untuk saya.

1. Ringkas apa yang sudah dibangun di Tahap <n>, dan temuan desain yang masih Terbuka.
2. Berikan tugas latihan Gerbang <n> sesuai docs/milestone/milestone.md, lengkap dengan kriteria lulus dan test yang harus lulus.
3. Setelah saya selesai menulisnya sendiri, tinjau kode saya: apa yang benar, apa yang salah, alasannya, dan saran. Jangan menulis ulang kode saya kecuali saya minta.
4. Bandingkan UML di docs/uml dengan kode yang sudah ada, lalu catat perbedaannya sebagai temuan (RK-64, usulan).
```

## Membawa temuan ke sesi desain

Setelah beberapa temuan terkumpul, unggah `docs/evaluasi-desain.md` ke percakapan desain bersama Claude. Di sana temuan diputuskan, lalu brief, UML, dan milestone diperbarui. Setelah itu salin brief, UML, dan milestone yang baru ke repo, dan tulis ID keputusannya di kolom Status temuan.
