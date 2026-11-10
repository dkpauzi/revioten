// Konfigurasi Playwright untuk TUR OTOMATIS: browser tampil di layar, lalu
// mengisi formulir dan memilih menu sendiri, mengikuti
// docs/panduan/skenario-alur-web.md. Tujuannya membantu memahami alur, bukan
// menguji; test yang menjaga kode tetap di folder e2e/.
//
// Jalankan lewat: npm --prefix web run tour
//
// Kecepatan diatur variabel TOUR_SPEED:
//   slow    untuk membaca pelan-pelan (jeda panjang)
//   normal  bawaan
//   fast    tanpa jeda, dipakai untuk memeriksa bahwa turnya masih jalan
// TOUR_HEADLESS=1 menjalankan tur tanpa jendela browser.
import { defineConfig, devices } from '@playwright/test';

const baseURL = 'http://127.0.0.1:8090';

// Jeda (milidetik) di antara setiap aksi browser, misalnya klik atau ketik.
const slowMo = { slow: 1200, normal: 600, fast: 0 }[process.env.TOUR_SPEED ?? 'normal'] ?? 600;

export default defineConfig({
	testDir: 'tour',
	testMatch: '*.tour.ts',
	workers: 1,
	reporter: 'list',
	// Tur berjalan pelan, jadi batas waktunya panjang (20 menit).
	timeout: 20 * 60_000,
	use: {
		baseURL,
		headless: process.env.TOUR_HEADLESS === '1',
		viewport: { width: 1366, height: 768 },
		// Rekaman video tur disimpan di web/test-results/, bisa diputar ulang.
		video: 'on'
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				viewport: { width: 1366, height: 768 },
				launchOptions: {
					slowMo,
					executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined
				}
			}
		}
	],
	// Server yang sama dengan test E2E: database revioten_e2e dibuat ulang dari
	// kosong, jadi tur tidak pernah menyentuh database kerja Anda.
	webServer: {
		command: 'node e2e/server.mjs',
		url: `${baseURL}/health`,
		reuseExistingServer: false,
		timeout: 180_000
	}
});
