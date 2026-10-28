// Pemanggil API yang ditulis tangan. Bentuk data dan error diambil dari tipe
// hasil generate di gen/ (dibuat dari api/openapi.yaml), jadi bila kontrak
// berubah, TypeScript langsung menunjukkan kode yang harus ikut diubah.
import type { Error as ApiError, Customer, CustomerList, NewCustomer } from './gen/index.ts';

// ApiFailure menjelaskan panggilan yang gagal: status HTTP (0 bila server
// tidak bisa dihubungi) dan isi error dari kontrak, bila ada.
export type ApiFailure = { status: number; body: ApiError | null };

// ApiResult adalah hasil satu panggilan: berhasil dengan data, atau gagal.
// Dengan bentuk ini, kode layar dipaksa menangani kedua kemungkinan.
export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiFailure };

// listCustomers memanggil GET /api/v1/customers.
// Parameter fetchFn bisa diganti di test; di browser nilainya fetch bawaan.
export function listCustomers(fetchFn: typeof fetch = fetch): Promise<ApiResult<CustomerList>> {
	return call<CustomerList>(fetchFn, '/api/v1/customers', { method: 'GET' });
}

// createCustomer memanggil POST /api/v1/customers.
export function createCustomer(
	input: NewCustomer,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, '/api/v1/customers', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(input)
	});
}

// call menjalankan satu permintaan dan membaca jawabannya sebagai JSON.
async function call<T>(fetchFn: typeof fetch, url: string, init: RequestInit): Promise<ApiResult<T>> {
	let response: Response;
	try {
		response = await fetchFn(url, init);
	} catch {
		// fetch gagal sama sekali, misalnya server Go tidak berjalan.
		return { ok: false, error: { status: 0, body: null } };
	}

	// Isi yang bukan JSON (misalnya halaman HTML) dibaca sebagai null.
	const body: unknown = await response.json().catch(() => null);
	if (response.ok && body !== null) {
		// Server dan klien memakai kontrak yang sama, jadi bentuknya dipercaya.
		return { ok: true, data: body as T };
	}
	return { ok: false, error: { status: response.status, body: isApiError(body) ? body : null } };
}

// isApiError memeriksa bahwa isi jawaban berbentuk Error dari kontrak.
function isApiError(body: unknown): body is ApiError {
	return (
		typeof body === 'object' &&
		body !== null &&
		'code' in body &&
		typeof body.code === 'string' &&
		'request_id' in body &&
		typeof body.request_id === 'string'
	);
}
