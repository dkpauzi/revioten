// Test E2E Tahap 5: perubahan dokumen. M5.1: satu permintaan perubahan
// menunjuk beberapa dokumen.
//
// Nama file diawali m5- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// setupSet membuat customer CH dengan model C-1 dan satu set dokumen PFD,
// PFMEA, dan CP. Dipanggil sebagai akun penyusun. Hasilnya nomor ketiganya.
async function setupSet(page: Page): Promise<Record<string, string>> {
	return page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		const out: Record<string, string> = {};
		for (const n of set.items as { id: string; number: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			out[doc.document_type.key] = n.number;
		}
		return out;
	});
}

test('M5.1: penyusun mengajukan perubahan yang menunjuk beberapa dokumen', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'CH', name: 'Customer CH', project_group_id: group.id });
		await post('/api/v1/models', { customer_id: customer.id, name: 'C-1' });
	});
	const temporary = await createWorkAccount(page, 'Pemohon perubahan', 'pemohon.e2e', ['author']);
	await logOut(page);
	await logInFirstTime(page, 'pemohon.e2e', temporary, 'perubahan selalu punya alasan');
	const numbers = await setupSet(page);

	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await expect(page.getByText(en.changeNone)).toBeVisible();

	await page.getByLabel(en.changeTrigger, { exact: true }).selectOption({ label: 'Customer claim' });
	await page.getByLabel(en.changeType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.changeFourM, { exact: true }).selectOption({ label: 'Method' });
	await page.getByLabel(en.changeReference, { exact: true }).fill('Claim 2026-031');
	await page.getByLabel(en.changeReason, { exact: true }).fill('Add AOI check for false NG');
	// Tanpa dokumen ditolak.
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.locator('fieldset .error')).toBeVisible();
	await page.getByRole('checkbox', { name: new RegExp(numbers.pfmea) }).check();
	await page.getByRole('checkbox', { name: new RegExp(numbers.cp) }).check();
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeCreated.replace('{ref}', 'CHG-0001'));
	const row = page.getByRole('row', { name: /CHG-0001/ });
	await expect(row).toContainText('Add AOI check for false NG');
	await expect(row).toContainText(`${numbers.pfmea}, ${numbers.cp}`);

	// Halaman perubahan: tambah PFD, lalu lepas PFMEA.
	await row.getByRole('link', { name: 'CHG-0001' }).click();
	await expect(page.getByRole('heading', { name: 'CHG-0001', level: 1 })).toBeVisible();
	await page.getByLabel(en.changeDocumentAdd).selectOption({ label: `${numbers.pfd} · PFD` });
	await page.getByRole('button', { name: en.changeDocumentAddButton }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeDocumentAdded.replace('{number}', numbers.pfd));
	await page.getByRole('button', { name: en.changeDocumentRemoveFor.replace('{number}', numbers.pfmea) }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeDocumentRemoved.replace('{number}', numbers.pfmea));
	await expect(page.locator('table tbody tr')).toHaveCount(2);
});
