# revioten

Revisi + Yokoten. Sistem web self-hosted untuk menyusun, merevisi, menomori, mencetak, dan mencatat rilis PFD, PFMEA, dan Control Plan di pabrik EMS. Persetujuan tetap tanda tangan basah di kertas.

Ini proyek belajar pribadi pemilik. Aplikasi berjalan di localhost sebagai demo untuk mematangkan konsep dan alur kerja, dan dibangun sejajar dengan paket UML yang menjadi hasil untuk perusahaan.

## Mulai dari mana

- `CLAUDE.md`: aturan kerja, struktur folder, dan semua perintah yang sudah terbukti jalan (menyalakan database, test, build, menjalankan server). Perintah sengaja hanya ditulis di sana.
- `docs/brief/pfmea-system-design-brief.md`: brief desain (bahasa Inggris), sumber keputusan (D), aturan (R), dan pertanyaan terbuka (Q).
- `docs/milestone/milestone.md`: urutan milestone (ekspor dari docx, jangan diedit).
- `docs/uml/`: paket UML untuk tim IT.
- `docs/catatan/`: catatan setiap milestone, termasuk cara mengulang dari nol dan latihan tulis tangan.
- `docs/evaluasi-desain.md`: temuan desain dari kode (E-nn) dan keputusan pemilik.
- `docs/versions.md`: versi setiap komponen beserta tanggal dan sumbernya.
- `docs/ringkasan-untuk-desain-ui.md`: ringkasan konsep, halaman yang sudah dibangun, dan semua keputusan, untuk sesi desain UI/UX di Claude chat (tanpa data asli).
- `api/openapi.yaml`: kontrak API untuk layar web.

Data asli perusahaan dan customer tidak pernah di-commit (CLAUDE.md bagian 7).
