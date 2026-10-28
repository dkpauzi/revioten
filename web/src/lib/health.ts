import type { Health } from './api/gen/index.ts';

// Hasil pemeriksaan server:
//   - 'ok': server dan database hidup;
//   - 'database_unreachable': server hidup, tapi database tidak bisa dihubungi;
//   - 'error': semua keadaan lain (server mati, jawaban tidak sesuai kontrak).
export type HealthResult = 'ok' | 'database_unreachable' | 'error';

// checkHealth memanggil GET /health dan menyimpulkan status server.
// Parameter fetchFn bisa diganti di test; di browser nilainya fetch bawaan.
export async function checkHealth(fetchFn: typeof fetch = fetch): Promise<HealthResult> {
	try {
		const response = await fetchFn('/health');
		const body: unknown = await response.json();
		if (!isHealth(body)) {
			return 'error';
		}
		if (response.ok && body.status === 'ok') {
			return 'ok';
		}
		return body.database === 'unreachable' ? 'database_unreachable' : 'error';
	} catch {
		// fetch gagal sama sekali, atau isi jawaban bukan JSON.
		return 'error';
	}
}

// isHealth memeriksa bahwa isi JSON berbentuk Health dari kontrak.
// Isi dari jaringan bertipe unknown, jadi bentuknya diperiksa dulu.
function isHealth(body: unknown): body is Health {
	return (
		typeof body === 'object' &&
		body !== null &&
		'status' in body &&
		typeof body.status === 'string' &&
		'database' in body &&
		typeof body.database === 'string'
	);
}
