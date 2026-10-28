// Pemanggil API yang ditulis tangan. Bentuk data dan error diambil dari tipe
// hasil generate di gen/ (dibuat dari api/openapi.yaml), jadi bila kontrak
// berubah, TypeScript langsung menunjukkan kode yang harus ikut diubah.
//
// Cookie sesi dikirim browser secara otomatis, karena alamat API sama dengan
// alamat halaman. Kode ini tidak pernah membaca cookie itu (HttpOnly).
import type {
	Error as ApiError,
	Credentials,
	Customer,
	CustomerList,
	NewCustomer,
	PasswordChange,
	Session
} from './gen/index.ts';

// ApiFailure menjelaskan panggilan yang gagal: status HTTP (0 bila server
// tidak bisa dihubungi) dan isi error dari kontrak, bila ada.
export type ApiFailure = { status: number; body: ApiError | null };

// ApiResult adalah hasil satu panggilan: berhasil dengan data, atau gagal.
// Dengan bentuk ini, kode layar dipaksa menangani kedua kemungkinan.
export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiFailure };

// onUnauthenticated dipanggil setiap kali API menjawab 401 (sesi berakhir),
// kecuali saat login. Layout mengisinya dengan "pindah ke halaman login".
let onUnauthenticated: (() => void) | null = null;

// setUnauthenticatedHandler mengganti penanganan sesi berakhir (null untuk melepas).
export function setUnauthenticatedHandler(handler: (() => void) | null): void {
	onUnauthenticated = handler;
}

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
	return call<Customer>(fetchFn, '/api/v1/customers', jsonRequest('POST', input));
}

// getSession memanggil GET /api/v1/session: siapa yang sedang login.
export function getSession(fetchFn: typeof fetch = fetch): Promise<ApiResult<Session>> {
	return call<Session>(fetchFn, '/api/v1/session', { method: 'GET' });
}

// login memanggil POST /api/v1/session. Bila berhasil, server memasang cookie sesi.
export function login(
	credentials: Credentials,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Session>> {
	// 401 di sini berarti nama login atau password salah, bukan sesi berakhir.
	return call<Session>(fetchFn, '/api/v1/session', jsonRequest('POST', credentials), false);
}

// logout memanggil DELETE /api/v1/session. Server menghapus sesi dan cookie-nya.
export function logout(fetchFn: typeof fetch = fetch): Promise<ApiResult<undefined>> {
	return call<undefined>(fetchFn, '/api/v1/session', { method: 'DELETE' });
}

// changePassword memanggil PUT /api/v1/account/password. Server memasang
// cookie sesi baru, dan sesi lain milik akun ini berakhir.
export function changePassword(
	input: PasswordChange,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<undefined>> {
	return call<undefined>(fetchFn, '/api/v1/account/password', jsonRequest('PUT', input));
}

// jsonRequest menyusun permintaan dengan isi JSON.
function jsonRequest(method: string, body: unknown): RequestInit {
	return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) };
}

// call menjalankan satu permintaan dan membaca jawabannya sebagai JSON.
async function call<T>(
	fetchFn: typeof fetch,
	url: string,
	init: RequestInit,
	notifyUnauthenticated = true
): Promise<ApiResult<T>> {
	let response: Response;
	try {
		response = await fetchFn(url, init);
	} catch {
		// fetch gagal sama sekali, misalnya server Go tidak berjalan.
		return { ok: false, error: { status: 0, body: null } };
	}

	// 204 No Content: berhasil tanpa isi.
	if (response.status === 204) {
		return { ok: true, data: undefined as T };
	}
	if (response.status === 401 && notifyUnauthenticated) {
		onUnauthenticated?.();
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
