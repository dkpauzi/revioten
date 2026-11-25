# D-226 · Tepi halaman Control Plan, dan bab Help mengisi ulang data

- **Tanggal:** 25 November 2026
- **Persetujuan pemilik:** "oke untuk menjelaskan urutannya sesudah database direset", "oke untuk opsi 2, namun tetap rancang untuk opsi 1 namun jangan di bangun dulu code nya. kita akan evaluasi setelah saya melakukan tes manual". Tur otomatis tidak ditambah import: "fokus tur otomatis di proses manual".
- **Aturan yang dipakai:** D-122 dan D-211 (Process Flow otomatis, satu rencana halaman untuk pratinjau, PDF, dan Excel), D-219 (simbol alur dan konektor), D-141 (Help).

## 1. Apa yang dibuat

| Bagian | Isi |
|---|---|
| Pemecah halaman | `Paginate` (`internal/printout/plan.go`) menerima `topMin` dan mengembalikan `kept`. Baris awal langkah yang menjadi baris paling atas halaman 2 dan seterusnya mendapat tinggi paling kecil `topRowMM` (sekitar 15 mm untuk kotak proses, 16,7 mm untuk belah ketupat), dan tinggi itu ikut dihitung saat memecah halaman. |
| Halaman penuh | `BuildPlan` mempertinggi baris terakhir halaman yang berakhir karena aturan keep (baris berikutnya sebenarnya masih muat), sampai ruang halaman habis. Halaman itu ditandai `PlanPage.Filled`. Halaman yang berakhir biasa tidak diubah. |
| Simbol di bawah konektor | `FlowNode.TopMM`: pusat simbol dari atas barisnya. Di baris paling atas halaman yang punya konektor masuk, nilainya `shiftedTopMM` (di bawah konektor, dengan panah 1,5 mm). `FlowSVG` (pratinjau, PDF) dan `flowShapes` (Excel) memakainya. |
| Help | Bab baru **Filling the data again after a database reset** / **Mengisi ulang data sesudah database direset** (`refill-after-reset`), dan tip aturan halaman di bab Printing. |
| Rancangan opsi 1 | `docs/rencana/pemisah-halaman-manual.md`: tanda "mulai halaman baru" per baris. Belum dibangun. |
| Test | `TestPaginate` (kept dan topMin) dan `TestPageEdges` (240 baris: tidak ada halaman yang berakhir dengan baris awal langkah, halaman Filled terisi penuh, simbol tidak menumpuk dengan konektor dan tidak keluar dari barisnya). |

## 2. Hasil diperiksa mata

`REVIOTEN_SAMPLE_DIR=<folder> go test -short ./internal/printout -run TestSampleTokairika`, lalu `sample-cp.html` dipotret dengan Edge headless. Di atas halaman 2, konektor A berada di atas belah ketupat 5.4 dengan panah pendek di antaranya; sebelumnya keduanya menumpuk. Konektor di bawah halaman 1 terpisah dari simbol 5.3.

## 3. Cara mengulang dari nol

1. Tulis kasus baru di `TestPaginate` dan `TestPageEdges`.
2. Ubah `Paginate`, tambah `shiftedTopMM` dan `topRowMM`, `FlowNode.TopMM`, dan `PlanPage.Filled`, lalu pakai di `BuildPlan` dan `flowPages`.
3. Ganti `nodeTopMM` di `FlowSVG` dan `flowShapes` dengan `n.centerMM()`.
4. Jalankan contoh Tokairika dan periksa gambarnya.

## 4. Konsep yang dipelajari

- **Satu rencana, tiga keluaran.** Karena pratinjau, PDF, dan Excel membaca tinggi baris dan letak simbol dari `Plan`, aturan halaman cukup ditulis sekali.
- **Ukuran yang saling bergantung.** Tinggi baris paling atas baru diketahui sesudah halaman dipecah, padahal pemecahan halaman butuh tinggi itu. Jalan keluarnya: `Paginate` memakai tinggi "bila di atas halaman" saat membuka halaman baru, lalu `BuildPlan` menuliskannya ke `RowsMM`.

## 5. Asumsi

- **Halaman hanya diisi penuh bila berakhir karena aturan keep,** sesuai uraian pemilik ("baris di atasnya akan menyesuaikan"). Halaman yang berakhir biasa tetap menyisakan sedikit ruang di bawah.
- **Langkah pertama dokumen tidak mendapat tinggi tambahan,** karena tidak pernah punya konektor masuk.
- **Kotak NG di kiri simbol yang digeser** tetap di kiri dan sedikit di atas simbol; kotaknya tidak menumpuk dengan konektor karena letaknya di kiri garis alur.

## 6. Temuan desain baru

Tidak ada. Pertanyaan opsi 1 menunggu tes manual pemilik (`docs/rencana/pemisah-halaman-manual.md`, bagian 4).

## 7. Status test

| Test | Hasil |
|---|---|
| `go test ./...`, `go vet`, `gofmt` | Semua lulus |
| `npm --prefix web run check` | 0 error |
| Vitest | 195 lulus |
| `npm --prefix web run test:e2e` | 79 lulus |
| help-shots | Lulus, gambar diperbarui |

## 8. Latihan tulis tangan

Tanpa AI, tulis fungsi Go `paginate(heights []float64, available float64) [][]int` yang membagi baris ke halaman tanpa aturan keep, lalu tambahkan satu aturan: baris bertanda `keep` tidak boleh menjadi baris terakhir halaman. Uji dengan tinggi `40, 40, 40`, ruang 100, dan keep `false, true, false` (hasil `{0}`, `{1, 2}`). Bandingkan dengan `Paginate` di `internal/printout/plan.go`.
