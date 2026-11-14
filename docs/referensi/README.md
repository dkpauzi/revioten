# Dokumen referensi

Dokumen asli yang dipakai sebagai acuan desain dan data contoh. Sejak D-152 (14 November 2026) data asli boleh disimpan di repo private ini dan dipakai di pengembangan. Yang tetap dilarang: menyalinnya ke luar (repo publik, gist, situs paste, atau layanan lain), dan meng-commit rahasia (password, token, `.env`).

## `Process_Flow_PCBA_Generic.md` (utama) dan `.xlsx`

- **Diterima:** 14 November 2026 dari pemilik ("Jadikan dokumen lampiran sebagai referensi"). Pemilik meminta file `.md` yang dibaca ("tolong baca file ini saja"): file itu **data master**, dan Excel serta draft PFMEA dan CP diturunkan darinya (bagian 0 file itu).
- **Bentuk `.md`:** bagian 3 register dokumen (tabel), bagian 4 alur level 1 (Mermaid, termasuk panah NG putus-putus), bagian 5 reaksi dan kontrol umum, bagian 6 satu blok YAML per proses (`stage`, `symbol`, `type`, `group_condition`, `previous`, `next`, `ng_flow`, `steps` S01…, `failure_modes`, `references`), bagian 7 matriks dampak dokumen.
- **Isi:** process flow PCBA umum (EMS) revisi 20, status DRAFT. Sumber kebenarannya WI General dan WI Process. SHA-256 berawalan `40DCD2CAB69968E4` untuk `.md` dan `E24EC47974B34755` untuk `.xlsx`.
- **Format PFMEA:** AIAG 4th (RPN), bukan AIAG-VDA (lembar Legend & Rules).

| Lembar | Isi | Bagian sistem yang terkait |
|---|---|---|
| L1 Flow Diagram | Diagram level 1 dalam 4 tahap (1. Material, 2. SMT, 3. Post-SMT / Assembly, 4. Test & Final), satu kolom per tahap, dengan "continue to" antar kolom | Diagram PFD (M4.10), E-95 |
| L1 Process Table | 33 proses P010 sampai P320: Stage, Symbol, Type (Mandatory, Optional, Conditional, Alternative), Group / Condition, Inspection Point, Previous, Next, NG Flow, ringkasan kolom CP | Master proses, baris PFD, E-95 |
| L2 Pxxx (33 lembar) | Per proses: A. langkah S01, S02, … (simbol, alat, kriteria, frekuensi, metode, PIC, reaksi NG, rujukan); B. failure mode (efek, penyebab, kontrol pencegahan dan deteksi, reaksi); C. ringkasan reaksi | Process template: proses, sub-proses, poin PFD, PFMEA, CP (D-146) |
| Common Reaction (NCP), Common Controls (ESD, Equipment) | Kalimat reaksi dan kontrol umum untuk semua proses | Poin umum (cakupan General) |
| Doc Reference per Process | Dokumen (nomor dan revisi) yang dipakai setiap proses, beserta bagian dan poin kuncinya | Master dokumen terkait (D-130, R-DOC-11) |
| Doc Register | Register dokumen: nomor, judul, level (WI, Procedure), revisi, tanggal revisi, proses pemakai | Master dokumen terkait (D-130) |
| Doc Impact Matrix, Doc x Process Matrix | Cakupan dokumen (General, General + conditional, Model-specific) dan dampaknya ke PFMEA dan CP | Cakupan dokumen terkait, daftar dampak perubahan (R-CHG-05) |
| PFMEA (Draft), Control Plan (Draft) | Draft PFMEA dan CP dari L2 | Contoh isi baris; uji perakitan |
| Cara Isi PFMEA, Cara Isi Control Plan | Panduan mengisi kolom (bahasa Indonesia) | Bahan halaman Help |
| Glossary (Istilah) | Istilah lapangan dan istilah standar (IPC-A-610, AIAG) | Glosarium U1 |
| Legend & Rules | Arti simbol dan tipe, aturan rating AIAG, aturan penulisan kontrol | Brief bagian 21 (F-13 sampai F-18) |

Temuan dari dokumen ini dicatat di brief bagian 21 (F-13 sampai F-18).

Catatan baca: di `.md`, tipe langkah L2 ditulis `op`, `insp`, `dec`, `trans`, `store`, dan simbol proses level 1 `Operation`, `Inspection`, `Storage`, `Transport`.
