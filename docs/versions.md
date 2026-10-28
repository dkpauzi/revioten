# Versi komponen

Hasil cek ulang M0.1 terhadap R-VER-01 sampai R-VER-05 (D-44), untuk semua item di brief bagian 7.

- **Tanggal dicek:** 27 Oktober 2026, dari container cloud Claude Code. Dicek ulang 28 Oktober 2026 tanpa perubahan.
- **Versi yang dipakai** adalah versi stabil atau LTS terbaru (keputusan pemilik E-13, 28 Oktober 2026), dengan masa tunggu untuk dua kasus khusus (E-14, lihat bagian "Aturan yang berlaku"). Brief bagian 7 diperbarui lewat sesi desain.
- **Tanggal rilis** memakai UTC dari sumber yang disebut. Tanggal di brief bisa berbeda satu hari karena zona waktu (WIB = UTC+7).
- Versi dikunci saat paketnya pertama kali dipasang: modul Go di `go.mod` dan `go.sum` (mulai M0.2), paket npm di `package.json` dan `package-lock.json` (mulai M0.4), dan image Docker di `compose.yaml` (M0.1).

## Aturan yang berlaku

| Aturan | Isi | Dasar |
|---|---|---|
| Stabil | Hanya rilis stabil: tanpa alpha, beta, rc, next, canary, atau nightly. Untuk npm berarti dist-tag `latest`. | R-VER-01, D-44 |
| Line yang didukung | Node memakai line LTS, Go salah satu dari dua line terbaru, dan PostgreSQL major terbaru yang sudah GA. | R-VER-02 |
| Peer dependency | Semua peer dependency cocok tanpa `--force` atau `--legacy-peer-deps`. | R-VER-03 |
| Terbaru | Pakai versi stabil atau LTS terbaru, termasuk patch terbaru. | R-VER-05, E-13 |
| Masa tunggu 14 hari | Major baru biasanya dipakai setelah ada patch. Dua kasus khusus mendapat masa tunggu: (1) line yang baru punya satu rilis boleh dipakai bila tidak ada line lain yang memenuhi peer dependency, dan rilisnya sudah berumur minimal 14 hari; pengecualiannya dicatat di tabel ini, dan begitu patch keluar, pindah ke patch terbaru. (2) Paket 0.x memakai versi terbaru yang sudah berumur minimal 14 hari saat pertama kali dipasang. | R-VER-04, E-14 |
| Kapan dikunci | Versi dikunci saat paketnya pertama kali dipasang, dan dicek ulang di milestone itu. | E-13 |

## Sumber dan cara mengecek

| Sumber | Perintah atau alamat | Bisa dibuka dari cloud? |
|---|---|---|
| Modul Go | `go list -m -versions <modul>`, lalu `go list -m -json <modul>@<versi>` (field `Time`) lewat proxy.golang.org | Ya |
| Toolchain Go | `go list -m -versions golang.org/toolchain` | Ya |
| Halaman rilis Go | https://go.dev/doc/devel/release | Tidak (ditolak proxy) |
| Paket npm | `npm view <paket> dist-tags time --json`, `npm view <paket>@<versi> peerDependencies engines` | Ya |
| Node.js | https://nodejs.org/dist/index.json (kolom `lts`) | Ya |
| PostgreSQL | Tag git `REL_18_6` di github.com/postgres/postgres (tanggal commit "Stamp 18.6"), tag image di Docker Hub | Ya |
| Halaman rilis PostgreSQL | https://www.postgresql.org/support/versioning/ | Tidak (ditolak proxy) |
| PlantUML, C4-PlantUML, ClamAV, OpenAPI | `git ls-remote --tags` ke repo GitHub resmi, lalu tanggal tag | Ya |

Halaman rilis resmi go.dev dan postgresql.org belum dicek. Silakan buka dari komputer Anda bila ingin memastikan tanggal pengumuman resminya.

## Runtime dan database

| Komponen | Versi yang dipakai | Tanggal rilis | Dicek | Sumber | Catatan |
|---|---|---|---|---|---|
| Go | 1.27.3 | 2026-10-19 | 2026-10-27 | proxy.golang.org, `golang.org/toolchain` | Terbaru. Line yang didukung: 1.27 dan 1.26 (terbaru 1.26.10), jadi memenuhi R-VER-02. Belum ada 1.28 (rc pun belum). Build `windows-amd64` tersedia. Dikunci di `go.mod` (`go 1.27.3`) sejak M0.2, setelah dicek ulang 28 Oktober 2026. |
| PostgreSQL | 18.6 | 2026-08-11 | 2026-10-27 | Tag `REL_18_6` (commit "Stamp 18.6"), Docker Hub `postgres:18.6` | Terbaru di line 18. PostgreSQL 19 masih RC 1 (tag `REL_19_RC1`, 2026-10-12; image `19rc1`), jadi tidak dipakai (R-VER-01, R-VER-02). |
| Image Docker PostgreSQL | `postgres:18.6` | Tag diperbarui 2026-10-21 | 2026-10-27 | Docker Hub | Digest saat dicek: `sha256:edb0c08dc33befd0262ab3b5cf1c4f3f8cfd8b09e88c05b45adee29d6761ddb6`, sama dengan `18.6-trixie` (basis Debian 13). Tag bisa dibangun ulang untuk patch OS, digest tidak. |
| Node.js | 24.22.1 LTS (Krypton) | 2026-10-20 | 2026-10-27 | nodejs.org/dist/index.json | Terbaru di line 24 LTS. Node 26 (terbaru 26.12.1) masih Current, belum LTS. Cek lagi di M0.4. Memenuhi syarat SvelteKit 3 (Node 22.17 atau lebih baru). |
| npm | 11.19.1 | Ikut Node 24.22.1 | 2026-10-27 | `npm -v` setelah memasang Node 24.22.1 | Tidak dipasang terpisah. Versinya mengikuti Node. |

## Backend Go

| Komponen | Versi yang dipakai | Tanggal rilis | Dicek | Sumber | Catatan |
|---|---|---|---|---|---|
| pgx | v5.11.0 | 2026-09-07 | 2026-10-27 | proxy.golang.org | Terbaru. |
| sqlc | v1.31.1 | 2026-04-22 | 2026-10-27 | proxy.golang.org | Terbaru. |
| goose | v3.29.0 | 2026-10-11 | 2026-10-27 | proxy.golang.org | Terbaru. |
| River | v0.49.0 | 2026-10-05 | 2026-10-28 | proxy.golang.org | Paket 0.x, ikut masa tunggu 14 hari (E-14). v0.50.0 (rilis 2026-10-27) baru memenuhi syarat mulai 2026-11-10. River merilis minor tanpa patch (v0.45.0 sampai v0.50.0 semuanya .0). Versinya dikunci saat pertama kali dipakai. |
| excelize | v2.11.0 | 2026-07-06 | 2026-10-27 | proxy.golang.org | Terbaru. Brief menulis 2026-07-07, selisih zona waktu. |
| testcontainers-go | v0.44.0 | 2026-08-07 | 2026-10-27 | proxy.golang.org | Terbaru. |
| golang.org/x/crypto (argon2id) | v0.58.0 | 2026-10-09 | 2026-10-27 | proxy.golang.org | Terbaru. |
| oapi-codegen (bila dipilih) | v2.8.1 | 2026-10-24 | 2026-10-28 | proxy.golang.org | Diperbarui dari v2.8.0 (E-13). README v2.8.1 masih menyebut dukungan OpenAPI 3.1 sebagai "initial", jadi kontrak tetap 3.0.4 (D-38). |

## Frontend (dipakai mulai M0.4)

Semua versi di bawah adalah dist-tag `latest`, kecuali TypeScript (lihat catatannya).

| Komponen | Versi yang dipakai | Tanggal rilis | Dicek | Sumber | Catatan |
|---|---|---|---|---|---|
| Svelte | 5.57.3 | 2026-10-26 | 2026-10-28 | npm | Diperbarui dari 5.57.2 (E-13). Memenuhi peer SvelteKit 3.1.0 (^5.57.1). |
| SvelteKit (`@sveltejs/kit`) | 3.1.0 | 2026-10-14 | 2026-10-27 | npm | Terbaru. Peer: vite ^8.0.12, svelte ^5.57.1, typescript ^6.0.0, vite-plugin-svelte ^7.0.0. Engines: Node >=22.17. Semua cocok (R-VER-03). |
| `@sveltejs/adapter-static` | 4.0.0 | 2026-10-01 | 2026-10-28 | npm | **Masih satu-satunya rilis di line 4** (watch item R-VER-04). Peer `@sveltejs/kit ^3.0.0-next.0` cocok dengan 3.1.0. **Pengecualian E-14:** line ini satu-satunya yang mendukung SvelteKit 3, dan rilisnya sudah lebih dari 14 hari, jadi boleh dipakai di M0.4. Pindah ke patch 4.0.x begitu tersedia. |
| Vite | 8.3.5 | 2026-10-21 | 2026-10-27 | npm | Terbaru. Engines: Node ^20.19 atau >=22.12. |
| `@sveltejs/vite-plugin-svelte` | 7.3.1 | 2026-09-23 | 2026-10-27 | npm | Terbaru. Peer: vite ^8, svelte ^5.46.4. Cocok. |
| TypeScript | 6.0.3 | 2026-04-16 | 2026-10-27 | npm | Versi terbaru di line 6. dist-tag `latest` sudah 7.0.2, tapi SvelteKit 3.1.0 (^6.0.0) dan svelte-check 4.7.7 (^5 atau ^6) belum menerima 7 (R-VER-03). |
| svelte-check | 4.7.7 | 2026-10-16 | 2026-10-27 | npm | Terbaru. Peer typescript ^5.0.0 atau ^6.0.0. |
| Vitest | 5.0.5 | 2026-10-15 | 2026-10-27 | npm | Terbaru. Peer vite ^8 cocok. Engines: Node ^22.12, ^24, atau >=26. |
| Playwright (`@playwright/test`) | 1.64.1 | 2026-10-14 | 2026-10-27 | npm | Terbaru. |
| tabulator-tables | 6.7.1 | 2026-10-22 | 2026-10-27 | npm | Terbaru. Dipakai mulai M4.2. |
| `@xyflow/svelte` (Svelte Flow) | 1.7.0 | 2026-09-24 | 2026-10-27 | npm | Terbaru. Peer svelte ^5.25.0. Dipakai di M4.10 (opsional). |
| elkjs | 0.12.0 | 2026-07-17 | 2026-10-27 | npm | Terbaru. |
| openapi-typescript | 7.13.0 | 2026-02-11 | 2026-10-27 | npm | Terbaru, tapi peer `typescript ^5.x` masih bentrok dengan TypeScript 6 (R-VER-03). Q-47 tetap terbuka sampai M0.5. |
| openapi-fetch | 0.17.0 | 2026-02-11 | 2026-10-27 | npm | Terbaru. |
| `@hey-api/openapi-ts` (calon Q-47) | belum dipilih | 2026-06-22 (0.99.0) | 2026-10-27 | npm | Terbaru 0.99.0. Peer typescript >=5.5.3 atau >=6.0.0, engines Node >=22.18. Paket 0.x, ikut masa tunggu 14 hari (E-14); 0.99.0 sudah memenuhi. |

## Alat desain dan pendukung

| Komponen | Versi yang dipakai | Tanggal rilis | Dicek | Sumber | Catatan |
|---|---|---|---|---|---|
| PlantUML | v1.2026.9 | 2026-10-17 | 2026-10-27 | Tag GitHub plantuml/plantuml | Terbaru. |
| C4-PlantUML | v2.14.0 | 2026-08-26 | 2026-10-27 | Tag GitHub plantuml-stdlib/C4-PlantUML | Terbaru. Sudah disalin ke `docs/uml/lib/`. |
| Caddy | v2.11.7 | 2026-10-03 | 2026-10-27 | proxy.golang.org | Terbaru. Dipakai di M8.3. |
| Gotenberg (PDF opsi A) | v8.39.0 | 2026-10-26 | 2026-10-28 | proxy.golang.org | Diperbarui dari v8.38.1 (E-13). Hanya dipakai bila uji S5 memilihnya. Dicek ulang saat S5. |
| ClamAV | 1.5.4 | 2026-08-06 | 2026-10-27 | Tag GitHub Cisco-Talos/clamav (`clamav-1.5.4`) | Terbaru. Dipakai di M8.2. |
| Spesifikasi OpenAPI | 3.0.4 | 2024-10-24 | 2026-10-27 | Tag GitHub OAI/OpenAPI-Specification | Versi spesifikasi terbaru 3.2.1 (2026-09-10). Tetap 3.0.4 (D-38). |

## Alat di komputer pengembang (tidak ada di brief bagian 7)

Dicatat sebagai info, bukan keputusan.

| Komponen | Versi | Dicek | Sumber | Catatan |
|---|---|---|---|---|
| Docker Engine (cloud) | 29.9.2 | 2026-10-27 | `docker --version` di container cloud | Versi yang dipakai untuk bukti M0.1 di cloud. |
| Docker Compose (cloud) | v5.6.1 | 2026-10-27 | `docker compose version` di container cloud | Sama. |
| Git (cloud) | 2.43.0 | 2026-10-27 | `git --version` | Bawaan Ubuntu 24.04. |
| Git for Windows | v2.56.0.windows.4 | 2026-10-27 | Tag GitHub git-for-windows/git (2026-10-26) | Rilis terbaru saat dicek, untuk komputer pemilik. |
| Docker Desktop untuk Windows | belum dicek | | docs.docker.com ditolak proxy | Pemilik memasang versi stabil terbaru, lalu menulis versinya di sini. |
