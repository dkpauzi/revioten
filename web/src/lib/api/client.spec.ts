import { afterEach, describe, expect, it } from 'vitest';
import {
	changePassword,
	createAccount,
	createCustomer,
	disableAccount,
	enableAccount,
	getSession,
	listAccounts,
	listCustomers,
	listRoles,
	login,
	logout,
	resetAccountPassword,
	setAccountRoles,
	setUnauthenticatedHandler
} from './client.ts';

// Satu panggilan yang dicatat fakeFetch.
type Call = { url: string; init: RequestInit | undefined };

// fakeFetch membuat pengganti fetch yang menjawab dengan respons tertentu dan
// mencatat setiap panggilan.
function fakeFetch(response: Response, calls: Call[] = []): typeof fetch {
	return async (input, init) => {
		calls.push({ url: String(input), init });
		return response;
	};
}

const customer = {
	id: 'id-ab',
	code: 'AB',
	name: 'Example Customer',
	created_at: '2026-10-28T03:00:00Z'
};

describe('listCustomers', () => {
	it('memanggil GET /api/v1/customers dan mengembalikan daftar', async () => {
		const calls: Call[] = [];
		const result = await listCustomers(fakeFetch(Response.json({ items: [customer] }), calls));

		expect(calls).toHaveLength(1);
		expect(calls[0].url).toBe('/api/v1/customers');
		expect(calls[0].init?.method).toBe('GET');
		expect(result).toEqual({ ok: true, data: { items: [customer] } });
	});

	it('melaporkan status 0 bila server tidak bisa dihubungi', async () => {
		const failing: typeof fetch = async () => {
			throw new TypeError('fetch failed');
		};
		expect(await listCustomers(failing)).toEqual({ ok: false, error: { status: 0, body: null } });
	});

	it('menganggap gagal bila jawaban 200 bukan JSON', async () => {
		const html = new Response('<html></html>', { status: 200 });
		expect(await listCustomers(fakeFetch(html))).toEqual({
			ok: false,
			error: { status: 200, body: null }
		});
	});
});

describe('createCustomer', () => {
	it('mengirim input sebagai JSON lewat POST', async () => {
		const calls: Call[] = [];
		const result = await createCustomer(
			{ code: 'ab', name: 'Example Customer' },
			fakeFetch(Response.json(customer, { status: 201 }), calls)
		);

		expect(calls[0].url).toBe('/api/v1/customers');
		expect(calls[0].init?.method).toBe('POST');
		expect(new Headers(calls[0].init?.headers).get('Content-Type')).toBe('application/json');
		expect(JSON.parse(String(calls[0].init?.body))).toEqual({ code: 'ab', name: 'Example Customer' });
		expect(result).toEqual({ ok: true, data: customer });
	});

	it('mengembalikan isi error dari kontrak bila ditolak', async () => {
		const error = {
			code: 'validation_failed',
			message: 'Some fields are not valid.',
			details: [{ field: 'code', code: 'reserved', message: 'ZZ is reserved for general documents.' }],
			request_id: 'uji-422'
		};
		const result = await createCustomer(
			{ code: 'ZZ', name: 'X' },
			fakeFetch(Response.json(error, { status: 422 }))
		);
		expect(result).toEqual({ ok: false, error: { status: 422, body: error } });
	});

	it('mengabaikan isi error yang tidak sesuai kontrak', async () => {
		const result = await createCustomer(
			{ code: 'AB', name: 'X' },
			fakeFetch(new Response('Bad Gateway', { status: 502 }))
		);
		expect(result).toEqual({ ok: false, error: { status: 502, body: null } });
	});
});

const session = {
	account: { id: 'akun-1', login: 'admin.uji', kind: 'admin' },
	person: { id: 'orang-1', name: 'Admin Uji' },
	must_change_password: false,
	permissions: ['L-1', 'A-1']
};

const unauthenticated = {
	code: 'unauthenticated',
	message: 'Log in to continue.',
	request_id: 'uji-401'
};

describe('sesi', () => {
	afterEach(() => setUnauthenticatedHandler(null));

	it('login mengirim nama login dan password lewat POST /api/v1/session', async () => {
		const calls: Call[] = [];
		const result = await login(
			{ login: 'admin.uji', password: 'kuda terbang di atas sawah' },
			fakeFetch(Response.json(session), calls)
		);
		expect(calls[0].url).toBe('/api/v1/session');
		expect(calls[0].init?.method).toBe('POST');
		expect(JSON.parse(String(calls[0].init?.body))).toEqual({
			login: 'admin.uji',
			password: 'kuda terbang di atas sawah'
		});
		expect(result).toEqual({ ok: true, data: session });
	});

	it('getSession memanggil GET /api/v1/session', async () => {
		const calls: Call[] = [];
		expect(await getSession(fakeFetch(Response.json(session), calls))).toEqual({
			ok: true,
			data: session
		});
		expect(calls[0].init?.method).toBe('GET');
	});

	it('logout dan ganti password menerima jawaban 204 tanpa isi', async () => {
		const calls: Call[] = [];
		expect(await logout(fakeFetch(new Response(null, { status: 204 }), calls))).toEqual({
			ok: true,
			data: undefined
		});
		expect(calls[0]).toMatchObject({ url: '/api/v1/session', init: { method: 'DELETE' } });

		const result = await changePassword(
			{ current_password: 'lama', new_password: 'baru' },
			fakeFetch(new Response(null, { status: 204 }), calls)
		);
		expect(result).toEqual({ ok: true, data: undefined });
		expect(calls[1]).toMatchObject({ url: '/api/v1/account/password', init: { method: 'PUT' } });
	});

	it('memanggil penanganan sesi berakhir bila API menjawab 401', async () => {
		let called = 0;
		setUnauthenticatedHandler(() => called++);
		await listCustomers(fakeFetch(Response.json(unauthenticated, { status: 401 })));
		expect(called).toBe(1);
	});

	it('login yang gagal bukan tanda sesi berakhir', async () => {
		let called = 0;
		setUnauthenticatedHandler(() => called++);
		const failed = { code: 'invalid_credentials', message: 'x', request_id: 'uji' };
		const result = await login(
			{ login: 'admin.uji', password: 'salah' },
			fakeFetch(Response.json(failed, { status: 401 }))
		);
		expect(result).toEqual({ ok: false, error: { status: 401, body: failed } });
		expect(called).toBe(0);
	});
});

const account = {
	id: 'akun-2',
	kind: 'work',
	login: 'penyusun.uji',
	person: { id: 'orang-2', name: 'Penyusun Uji' },
	roles: [{ role: 'author', scope: 'all' }],
	must_change_password: true,
	disabled: false,
	created_at: '2026-10-28T03:00:00Z'
};

describe('akun dan peran (A-1)', () => {
	it('memanggil alamat dan method yang benar', async () => {
		const calls: Call[] = [];
		await listRoles(fakeFetch(Response.json({ items: [] }), calls));
		await listAccounts(fakeFetch(Response.json({ items: [account] }), calls));
		await createAccount(
			{ person_name: 'Penyusun Uji', kind: 'work', login: 'penyusun.uji', roles: [] },
			fakeFetch(Response.json({ account, temporary_password: 'SEMENTARA' }, { status: 201 }), calls)
		);
		await setAccountRoles('akun-2', [{ role: 'verifier', scope: 'all' }], fakeFetch(Response.json(account), calls));
		await resetAccountPassword('akun-2', fakeFetch(Response.json({ temporary_password: 'X' }), calls));
		await disableAccount('akun-2', fakeFetch(Response.json(account), calls));
		await enableAccount('akun-2', fakeFetch(Response.json(account), calls));

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/roles',
			'GET /api/v1/accounts',
			'POST /api/v1/accounts',
			'PUT /api/v1/accounts/akun-2/roles',
			'POST /api/v1/accounts/akun-2/password-reset',
			'POST /api/v1/accounts/akun-2/disable',
			'POST /api/v1/accounts/akun-2/enable'
		]);
		expect(JSON.parse(String(calls[3].init?.body))).toEqual({
			roles: [{ role: 'verifier', scope: 'all' }]
		});
	});

	it('mengodekan ID akun di alamat', async () => {
		const calls: Call[] = [];
		await disableAccount('a/b?c', fakeFetch(Response.json(account), calls));
		expect(calls[0].url).toBe('/api/v1/accounts/a%2Fb%3Fc/disable');
	});
});
