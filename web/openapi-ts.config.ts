// Konfigurasi @hey-api/openapi-ts 0.99.0 (Q-47, diuji di M0.5): membuat tipe
// TypeScript dari kontrak API. Hanya plugin tipe yang dipakai; pemanggilan
// fetch ditulis tangan di src/lib/api/client.ts supaya mudah dibaca.
// Jalankan lewat: npm --prefix web run gen:api
import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
	input: '../api/openapi.yaml',
	output: 'src/lib/api/gen',
	plugins: ['@hey-api/typescript']
});
