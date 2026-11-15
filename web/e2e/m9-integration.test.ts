// Test E2E Tahap 9: admin membuat service account dan menerbitkan token, lalu
// "aplikasi lain" (di sini: permintaan HTTP tanpa cookie) membaca dokumen
// terbit dan mengusulkan perubahan lewat /integration/v1.
// Berjalan sesudah m5-changes.test.ts: customer CH punya CP yang sudah terbit.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu } from './helpers.ts';

test('M9: token membaca dokumen terbit customer CH dan mengusulkan perubahan yang menunggu verifikasi', async ({ page, playwright }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navServiceAccounts);
	await expect(page.getByRole('heading', { name: en.serviceAccountsTitle, level: 1 })).toBeVisible();

	await page.getByLabel(en.serviceAccountName, { exact: true }).fill('MES E2E');
	await page.getByLabel(en.serviceAccountTeam, { exact: true }).fill('Tim MES');
	await page.getByLabel(en.serviceAccountContact, { exact: true }).fill('mes@contoh.test');
	await page.getByLabel(en.serviceAccountPurpose, { exact: true }).fill('Uji E2E');
	for (const scope of ['documents_read', 'revisions_read', 'changes_read', 'changes_propose'] as const) {
		await page.getByRole('checkbox', { name: en[`scope_${scope}`] }).check();
	}
	await page.getByRole('checkbox', { name: 'CH · Customer CH' }).check();
	await page.getByRole('button', { name: en.serviceAccountCreate }).click();
	await expect(page.getByRole('status').last()).toHaveText(en.serviceAccountCreated.replace('{name}', 'MES E2E'));

	const account = page.locator('section', { has: page.getByRole('heading', { name: 'MES E2E', level: 2 }) });
	await account.getByRole('button', { name: en.serviceTokenIssue }).click();
	const token = (await page.locator('.issued code').textContent()) ?? '';
	expect(token).toMatch(/^rvt_[a-z0-9]{8}_/);
	await expect(account.getByRole('cell', { name: en.serviceTokenActive })).toBeVisible();

	// Aplikasi lain: tanpa cookie, hanya token.
	const app = await playwright.request.newContext({ baseURL: page.url().replace(/\/admin.*$/, ''), extraHTTPHeaders: { Authorization: `Bearer ${token}` } });
	const anonymous = await playwright.request.newContext({ baseURL: page.url().replace(/\/admin.*$/, '') });
	expect((await anonymous.get('/integration/v1/documents')).status()).toBe(401);

	const list = await app.get('/integration/v1/documents');
	expect(list.status(), await list.text()).toBe(200);
	const items = (await list.json()).items as { id: string; number: string; customer_code: string; document_type: string; revision: number }[];
	expect(items.length).toBeGreaterThan(0);
	expect(items.every((d) => d.customer_code === 'CH')).toBe(true);
	const cp = items.find((d) => d.document_type === 'cp')!;
	const revision = await app.get(`/integration/v1/documents/${cp.id}/revisions/current`);
	expect(revision.status()).toBe(200);
	const body = (await revision.json()) as { fields: { id: string; area: string; automatic: boolean }[]; sections: unknown[] };
	// Kolom yang diisi sistem (misalnya Process Flow) tidak boleh dikirim.
	const field = body.fields.find((f) => f.area === 'row' && !f.automatic)!;

	const proposal = {
		document_id: cp.id,
		base_revision: cp.revision,
		source_application: 'MES',
		on_behalf_of: 'Operator lini 3',
		reason: 'Batas SPI berubah di lini',
		operations: [{ op: 'add', content: { [field.id]: 'Diusulkan dari MES' } }]
	};
	const created = await app.post('/integration/v1/changes', { data: proposal, headers: { 'Idempotency-Key': 'e2e-1' } });
	expect(created.status(), await created.text()).toBe(201);
	const change = await created.json();
	expect(change.status).toBe('submitted');
	// Diulang dengan kunci yang sama: hasil yang sama, tanpa perubahan kedua.
	const again = await app.post('/integration/v1/changes', { data: proposal, headers: { 'Idempotency-Key': 'e2e-1' } });
	expect(again.status()).toBe(200);
	expect((await again.json()).id).toBe(change.id);
	// Tanpa scope ekspor ditolak.
	expect((await app.post('/integration/v1/exports', { data: { document_id: cp.id } })).status()).toBe(403);

	// Usulan tampil di daftar perubahan orang, dengan aplikasi sebagai pemohon.
	await page.evaluate(() => fetch('/api/v1/session', { method: 'DELETE' }));
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	const proposed = page.getByRole('row', { name: new RegExp(change.ref) });
	await expect(proposed).toContainText('Batas SPI berubah di lini');

	await app.dispose();
	await anonymous.dispose();
});
