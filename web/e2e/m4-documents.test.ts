// Test E2E M4.1: dokumen dan baris. Kriteria selesai "baris bisa disimpan dan
// dibaca": penyusun membuat dokumen dari nomor Reserved, menambah dan mengubah
// baris, mengisi header, lalu memuat ulang halaman.
//
// Nama file diawali m4- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('penyusun membuat dokumen CP, lalu baris dan header tersimpan', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'DC', name: 'Customer DC', project_group_id: group.id });
		await post('/api/v1/models', { customer_id: customer.id, name: 'D-100' });
	});
	const temporary = await createWorkAccount(page, 'Penyusun dokumen', 'penyusun.dokumen.e2e', ['author', 'template_maintainer']);
	await logOut(page);
	await logInFirstTime(page, 'penyusun.dokumen.e2e', temporary, 'dokumen draft disimpan per baris');

	// Proses master dibuat lewat hak E-6 (Pengelola template), lalu satu nomor CP Reserved untuk DC.
	const number = await page.evaluate(async () => {
		await fetch('/api/v1/processes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name: 'Dokumen Reflow', aliases: [] })
		});
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'DC');
		const types = await (await fetch('/api/v1/document-types')).json();
		const cp = types.items.find((d: { key?: string }) => d.key === 'cp');
		const response = await fetch('/api/v1/document-numbers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': crypto.randomUUID() },
			body: JSON.stringify({ customer_id: customer.id, document_type_id: cp.id })
		});
		return (await response.json()).items[0].number as string;
	});

	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'DC · Customer DC' });
	await expect(page.getByText(en.documentsEmpty)).toBeVisible();
	await page.getByLabel(en.documentNumber, { exact: true }).selectOption({ label: `${number} · CP` });
	await page.getByLabel('D-100').check();
	await page.getByRole('button', { name: en.documentCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentCreated.replace('{number}', number));

	await page.getByRole('link', { name: number }).click();
	await expect(page.getByRole('heading', { name: number, level: 1 })).toBeVisible();
	await expect(page.getByText('AIAG 4th set')).toBeVisible();

	// Dua baris: nomor operasi dan proses adalah atribut baris (D-123).
	await page.getByLabel(en.documentOperationNumber).fill('10');
	await page.getByLabel(en.documentProcess, { exact: true }).selectOption({ label: 'Dokumen Reflow' });
	await page.getByLabel('Product Characteristics', { exact: true }).fill('Solder joint');
	await page.getByRole('button', { name: en.documentRowAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowAdded);
	await page.getByLabel(en.documentOperationNumber).fill('20');
	await page.getByLabel('Method', { exact: true }).fill('Visual 100%');
	await page.getByRole('button', { name: en.documentRowAdd }).click();
	await expect(page.getByRole('row', { name: /^1 .*10 Dokumen Reflow .*Solder joint/ })).toBeVisible();

	// Ubah baris pertama: ID dan urutannya tetap.
	await page.getByRole('row', { name: /^1 / }).getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel('Product Characteristics', { exact: true }).fill('Solder joint fillet');
	await page.getByRole('button', { name: en.documentRowSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);

	// Header.
	await page.getByLabel('Core Team', { exact: true }).fill('PE, QA, Production');
	await page.getByRole('button', { name: en.documentHeaderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentHeaderSaved);

	// Muat ulang: semua datang lagi dari database.
	await page.reload();
	await expect(page.getByRole('row', { name: /^1 .*Solder joint fillet/ })).toBeVisible();
	await expect(page.getByRole('row', { name: /^2 .*20 .*Visual 100%/ })).toBeVisible();
	await expect(page.getByLabel('Core Team', { exact: true })).toHaveValue('PE, QA, Production');
});
