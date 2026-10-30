import { afterEach, describe, expect, it } from 'vitest';
import {
	changePassword,
	createAccount,
	createCustomer,
	disableAccount,
	enableAccount,
	getSession,
	listAccounts,
	changeCustomerCode,
	createModel,
	createModelGroup,
	createPartNumber,
	createProcess,
	createProjectGroup,
	listAuditLog,
	listCustomers,
	listModelGroups,
	listModels,
	listPartNumbers,
	listProcesses,
	listProjectGroups,
	renameModelGroup,
	updateCustomer,
	updateModel,
	updatePartNumber,
	updateProcess,
	updateProjectGroup,
	listRoles,
	login,
	logout,
	resetAccountPassword,
	setAccountRoles,
	setUnauthenticatedHandler,
	verifyAuditLog,
	getNumberSettings,
	updateNumberSettings,
	registerDocumentNumber,
	getNextDocumentNumber,
	listDocumentTypes,
	createDocumentType,
	updateDocumentType,
	listSpecialCodes,
	createSpecialCode,
	updateSpecialCode,
	listChangeTypes,
	updateChangeType,
	listChangeSources,
	createChangeSource,
	updateChangeSource,
	listFourMCategories,
	updateFourMCategory,
	listPositions,
	updatePosition,
	addPositionHolder,
	removePositionHolder,
	listApprovalRoutes,
	createApprovalRoute,
	updateApprovalRoute,
	setCustomerDefaultRoute,
	setChangeTypeRoute,
	getSystemSettings,
	updateSystemSettings
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
	project_group: { id: 'gr-1', code: 'GR1' },
	active: true,
	codes: [{ code: 'AB', valid_from: '2026-10-28T03:00:00Z' }],
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
			{ code: 'ab', name: 'Example Customer', project_group_id: 'gr-1' },
			fakeFetch(Response.json(customer, { status: 201 }), calls)
		);

		expect(calls[0].url).toBe('/api/v1/customers');
		expect(calls[0].init?.method).toBe('POST');
		expect(new Headers(calls[0].init?.headers).get('Content-Type')).toBe('application/json');
		expect(JSON.parse(String(calls[0].init?.body))).toEqual({
			code: 'ab',
			name: 'Example Customer',
			project_group_id: 'gr-1'
		});
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
			{ code: 'ZZ', name: 'X', project_group_id: 'gr-1' },
			fakeFetch(Response.json(error, { status: 422 }))
		);
		expect(result).toEqual({ ok: false, error: { status: 422, body: error } });
	});

	it('mengabaikan isi error yang tidak sesuai kontrak', async () => {
		const result = await createCustomer(
			{ code: 'AB', name: 'X', project_group_id: 'gr-1' },
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

describe('audit log (A-4)', () => {
	it('meminta entri terbaru, lalu entri yang lebih lama dengan before', async () => {
		const calls: Call[] = [];
		await listAuditLog(undefined, fakeFetch(Response.json({ items: [] }), calls));
		await listAuditLog(42, fakeFetch(Response.json({ items: [] }), calls));
		await verifyAuditLog(fakeFetch(Response.json({ valid: true, entries: 0 }), calls));

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/audit-log',
			'GET /api/v1/audit-log?before=42',
			'GET /api/v1/audit-log/verification'
		]);
	});
});

describe('master data (M1.4)', () => {
	it('memanggil alamat dan method yang benar, dengan ID yang dikodekan', async () => {
		const calls: Call[] = [];
		const ok = () => fakeFetch(Response.json({ items: [] }), calls);
		await listProjectGroups(ok());
		await createProjectGroup({ code: 'gr1', name: 'Project Group 1' }, ok());
		await updateProjectGroup('gr/1', { code: 'GR1', name: 'Satu' }, ok());
		await updateCustomer('c/1', { name: 'X', project_group_id: 'gr-1', active: false }, ok());
		await changeCustomerCode('c/1', 'cd', ok());
		await listModelGroups('c&1', ok());
		await createModelGroup({ customer_id: 'c-1', name: 'Family' }, ok());
		await renameModelGroup('mg-1', 'Family B', ok());
		await listModels('c-1', ok());
		await createModel({ customer_id: 'c-1', name: 'X100' }, ok());
		await updateModel('m-1', { name: 'X100A' }, ok());
		await listPartNumbers('c-1', ok());
		await createPartNumber({ model_id: 'm-1', number: 'PN-1' }, ok());
		await updatePartNumber('p-1', { number: 'PN-1A' }, ok());

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/project-groups',
			'POST /api/v1/project-groups',
			'PUT /api/v1/project-groups/gr%2F1',
			'PUT /api/v1/customers/c%2F1',
			'POST /api/v1/customers/c%2F1/code',
			'GET /api/v1/model-groups?customer_id=c%261',
			'POST /api/v1/model-groups',
			'PUT /api/v1/model-groups/mg-1',
			'GET /api/v1/models?customer_id=c-1',
			'POST /api/v1/models',
			'PUT /api/v1/models/m-1',
			'GET /api/v1/part-numbers?customer_id=c-1',
			'POST /api/v1/part-numbers',
			'PUT /api/v1/part-numbers/p-1'
		]);
		expect(JSON.parse(String(calls[4].init?.body))).toEqual({ code: 'cd' });
		expect(JSON.parse(String(calls[7].init?.body))).toEqual({ name: 'Family B' });
	});
});

describe('master proses (M1.5)', () => {
	it('memanggil alamat dan method yang benar, dan mengirim daftar alias apa adanya', async () => {
		const calls: Call[] = [];
		const ok = () => fakeFetch(Response.json({ items: [] }), calls);
		await listProcesses(ok());
		await createProcess({ name: 'IQC', aliases: ['Incoming QC'] }, ok());
		await updateProcess('pr/1', { name: 'IQC', aliases: [] }, ok());

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/processes',
			'POST /api/v1/processes',
			'PUT /api/v1/processes/pr%2F1'
		]);
		// [] berarti semua alias dilepas, jadi harus ikut terkirim.
		expect(JSON.parse(String(calls[2].init?.body))).toEqual({ name: 'IQC', aliases: [] });
	});
});

describe('jenis dokumen dan perubahan (M1.6)', () => {
	it('memanggil alamat dan method yang benar, dengan ID yang dikodekan', async () => {
		const calls: Call[] = [];
		const ok = () => fakeFetch(Response.json({ items: [] }), calls);
		await getNumberSettings(ok());
		await updateNumberSettings({ factory_code: 's' }, ok());
		await listDocumentTypes(ok());
		await createDocumentType({ name: 'Work Instruction', letter: 'W' }, ok());
		await updateDocumentType('dt/1', { name: 'PFD', letter: 'H' }, ok());
		await listSpecialCodes(ok());
		await createSpecialCode({ code: 'QA', name: 'Quality' }, ok());
		await updateSpecialCode('sc-1', 'General', ok());
		await listChangeTypes(ok());
		await updateChangeType('ct-1', 'Content change', ok());
		await listChangeSources(ok());
		await createChangeSource('Kaizen', ok());
		await updateChangeSource('cs-1', { name: 'Kaizen', active: false }, ok());
		await listFourMCategories(ok());
		await updateFourMCategory('fm-1', 'Manpower', ok());

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/number-settings',
			'PUT /api/v1/number-settings',
			'GET /api/v1/document-types',
			'POST /api/v1/document-types',
			'PUT /api/v1/document-types/dt%2F1',
			'GET /api/v1/special-codes',
			'POST /api/v1/special-codes',
			'PUT /api/v1/special-codes/sc-1',
			'GET /api/v1/change-types',
			'PUT /api/v1/change-types/ct-1',
			'GET /api/v1/change-sources',
			'POST /api/v1/change-sources',
			'PUT /api/v1/change-sources/cs-1',
			'GET /api/v1/four-m-categories',
			'PUT /api/v1/four-m-categories/fm-1'
		]);
		expect(JSON.parse(String(calls[7].init?.body))).toEqual({ name: 'General' });
		expect(JSON.parse(String(calls[11].init?.body))).toEqual({ name: 'Kaizen' });
		expect(JSON.parse(String(calls[12].init?.body))).toEqual({ name: 'Kaizen', active: false });
	});
});

describe('jabatan dan rute approval (M1.8)', () => {
	it('memanggil alamat dan method yang benar, dengan ID yang dikodekan', async () => {
		const calls: Call[] = [];
		const ok = () => fakeFetch(Response.json({ items: [] }), calls);
		const route = { name: 'Up to GM', slots: [] };
		await listPositions(ok());
		await updatePosition('pos/1', 'PE Manager', ok());
		await addPositionHolder('pos-1', { person_id: 'p-1', project_group_id: 'g-1' }, ok());
		await removePositionHolder('pos-1', 'h/1', fakeFetch(new Response(null, { status: 204 }), calls));
		await listApprovalRoutes(ok());
		await createApprovalRoute(route, ok());
		await updateApprovalRoute('r-1', route, ok());
		await setCustomerDefaultRoute('c/1', 'r-1', ok());
		await setChangeTypeRoute('ct-1', 'r-1', ok());
		await setChangeTypeRoute('ct-1', null, ok());
		await getSystemSettings(ok());
		await updateSystemSettings({ sign_dates_enabled: true }, ok());

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'GET /api/v1/positions',
			'PUT /api/v1/positions/pos%2F1',
			'POST /api/v1/positions/pos-1/holders',
			'DELETE /api/v1/positions/pos-1/holders/h%2F1',
			'GET /api/v1/approval-routes',
			'POST /api/v1/approval-routes',
			'PUT /api/v1/approval-routes/r-1',
			'PUT /api/v1/customers/c%2F1/default-route',
			'PUT /api/v1/change-types/ct-1/approval-route',
			'PUT /api/v1/change-types/ct-1/approval-route',
			'GET /api/v1/system-settings',
			'PUT /api/v1/system-settings'
		]);
		expect(JSON.parse(String(calls[7].init?.body))).toEqual({ route_id: 'r-1' });
		// null berarti rute default customer: route_id tidak dikirim.
		expect(JSON.parse(String(calls[9].init?.body))).toEqual({});
	});
});

describe('register nomor (Tahap 2)', () => {
	it('mengirim Idempotency-Key bersama pendaftaran, dan menanyakan nomor berikutnya', async () => {
		const calls: Call[] = [];
		const ok = () => fakeFetch(Response.json({ items: [], replayed: false }), calls);
		await registerDocumentNumber({ customer_id: 'c/1', document_type_id: 'dt-3' }, 'kunci-1', ok());
		await getNextDocumentNumber('c/1', 'dt-3', ok());

		expect(calls.map((c) => `${c.init?.method} ${c.url}`)).toEqual([
			'POST /api/v1/document-numbers',
			'GET /api/v1/document-numbers/next?customer_id=c%2F1&document_type_id=dt-3'
		]);
		const headers = calls[0].init?.headers as Record<string, string>;
		expect(headers['Idempotency-Key']).toBe('kunci-1');
		expect(headers['Content-Type']).toBe('application/json');
	});
});
