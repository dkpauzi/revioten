// Hasil pemeriksaan server: 'ok' bila GET /health menjawab {"status":"ok"},
// 'error' untuk semua keadaan lain (server mati, jawaban bukan 200, isi salah).
export type HealthResult = 'ok' | 'error';

// checkHealth memanggil GET /health dan menyimpulkan status server.
// Parameter fetchFn bisa diganti di test; di browser nilainya fetch bawaan.
export async function checkHealth(fetchFn: typeof fetch = fetch): Promise<HealthResult> {
	try {
		const response = await fetchFn('/health');
		if (!response.ok) {
			return 'error';
		}
		const body: unknown = await response.json();
		return isStatusOk(body) ? 'ok' : 'error';
	} catch {
		// fetch gagal sama sekali, misalnya server Go tidak berjalan.
		return 'error';
	}
}

// isStatusOk memeriksa bahwa isi JSON berbentuk {"status":"ok"}.
// Isi dari jaringan bertipe unknown, jadi bentuknya diperiksa dulu.
function isStatusOk(body: unknown): boolean {
	return (
		typeof body === 'object' && body !== null && 'status' in body && body.status === 'ok'
	);
}
