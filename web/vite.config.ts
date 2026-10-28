import { defineConfig } from 'vitest/config';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

// Alamat server Go saat development (go run ./cmd/revioten).
const goServer = 'http://127.0.0.1:8080';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Paksa mode runes (Svelte 5) untuk kode proyek ini, kecuali pustaka di node_modules.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				// Hasil build ditaruh di build/app. Folder build/ sendiri berisi .gitkeep
				// yang di-commit, supaya go build tetap jalan sebelum frontend di-build
				// (lihat web.go). adapter-static menghapus folder output setiap build.
				pages: 'build/app',
				assets: 'build/app',
				// Mode SPA: semua alamat halaman dilayani lewat 200.html (lihat src/routes/+layout.ts).
				fallback: '200.html'
			})
		})
	],
	server: {
		// Saat npm run dev, permintaan ke server diteruskan ke server Go.
		proxy: {
			'/health': goServer,
			'/api': goServer
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.ts'],
		environment: 'node'
	}
});
