# Evaluasi desain dari kode

Mulai 27 Oktober 2026 kode dibangun sejajar dengan UML, dan dipakai untuk menguji desain (D-77). File ini mencatat setiap temuan dari kode yang menyentuh brief, UML, atau milestone.

## Aturan

- Claude Code menulis temuan, dan pemilik yang memutuskan.
- Claude Code tidak mengubah brief, UML, atau milestone. Keputusan pemilik dicatat di brief sebagai item D, lalu ID-nya ditulis di kolom Status.
- Satu temuan untuk satu entri. ID berurutan (E-01, E-02, dan seterusnya) dan tidak pernah dipakai ulang.
- Asumsi sementara di kode ditandai dengan komentar `// ASUMSI E-nn`, supaya mudah dicari setelah temuannya diputuskan.

**Jenis temuan**
- **Kontradiksi:** dua aturan atau dokumen saling bertentangan.
- **Celah:** kode butuh aturan yang belum ada.
- **Ambigu:** aturan bisa dibaca lebih dari satu cara.
- **Sulit dibangun:** aturannya jelas, tapi biayanya tinggi atau berisiko. Sertakan alternatifnya.
- **Penyederhanaan:** desain bisa dibuat lebih sederhana tanpa kehilangan tujuannya.

**Status:** Terbuka, Diputuskan (D-xx), Ditolak, atau Ditunda.

## Ringkasan

| ID | Judul | Jenis | Milestone | Menghambat? | Status |
|---|---|---|---|---|---|

## Rincian

<!--
Salin blok ini untuk setiap temuan baru.

### E-nn · Judul singkat

- **Tanggal dan milestone:**
- **Jenis:**
- **Rujukan:** ID di brief, UML, atau milestone (R-, D-, UC-, U, M)
- **Temuan:** apa yang ditemukan saat menulis kode.
- **Dampak bila dibiarkan:**
- **Opsi:** (a) ... (b) ...
- **Rekomendasi Claude Code (usulan):**
- **Asumsi sementara di kode:** isi bila temuan tidak menghambat.
- **Menghambat milestone?** Ya atau Tidak
- **Status:** Terbuka
- **Keputusan pemilik:**
-->
