// Konfigurasi Playwright untuk test E2E: browser sungguhan membuka layar,
// mengisi formulir, dan memeriksa hasilnya lewat server Go dan PostgreSQL.
// Jalankan lewat: npm --prefix web run test:e2e
import { defineConfig, devices } from '@playwright/test';

const baseURL = 'http://127.0.0.1:8090';

export default defineConfig({
	testDir: 'e2e',
	// Semua test memakai satu database, jadi dijalankan satu per satu.
	workers: 1,
	reporter: 'list',
	use: {
		baseURL,
		// Rekaman jejak hanya disimpan bila test gagal (folder test-results/).
		trace: 'retain-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				launchOptions: {
					// Biasanya kosong: Playwright memakai browser dari
					// `npx playwright install chromium`. Isi variabel ini bila
					// browser sudah terpasang di tempat lain.
					executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined
				}
			}
		}
	],
	webServer: {
		command: 'node e2e/server.mjs',
		// Server dianggap siap bila /health menjawab 200 (server dan database hidup).
		url: `${baseURL}/health`,
		reuseExistingServer: false,
		timeout: 180_000
	}
});
