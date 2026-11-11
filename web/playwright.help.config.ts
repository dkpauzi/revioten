// Konfigurasi Playwright untuk memotret layar halaman Help (D-141).
// Jalankan lewat: npm --prefix web run help:shots
// Memakai server dan database revioten_e2e yang sama dengan test E2E.
import { defineConfig, devices } from '@playwright/test';

const baseURL = 'http://127.0.0.1:8090';

export default defineConfig({
	testDir: 'help-shots',
	testMatch: '*.shots.ts',
	workers: 1,
	reporter: 'list',
	timeout: 5 * 60_000,
	// Izin clipboard, supaya kotak kata sandi sementara menampilkan pesan "sudah disalin".
	use: { baseURL, permissions: ['clipboard-read', 'clipboard-write'] },
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				// Ukuran tetap, supaya semua gambar panduan sama besar.
				viewport: { width: 1280, height: 800 },
				launchOptions: { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || undefined }
			}
		}
	],
	webServer: {
		command: 'node e2e/server.mjs',
		url: `${baseURL}/health`,
		reuseExistingServer: false,
		timeout: 180_000
	}
});
