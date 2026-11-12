// Test E2E M4.1 dan M4.2: dokumen dan baris, diisi di tabel workbook. Kriteria selesai "baris bisa disimpan dan
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

	// M4.2: baris diisi langsung di tabel workbook. Nomor operasi dan proses
	// adalah atribut baris (D-123).
	const grid = page.locator('.workbook');
	const rows = grid.locator('.tabulator-row');
	const fieldOf = async (title: string) =>
		(await grid.locator('.tabulator-col', { hasText: new RegExp(`^${title}$`) }).first().getAttribute('tabulator-field')) ?? '';
	const cell = (row: number, field: string) => rows.nth(row).locator(`.tabulator-cell[tabulator-field="${field}"]`);
	// type mengetik di satu sel lalu keluar dari sel itu, sehingga barisnya tersimpan.
	const type = async (row: number, field: string, text: string) => {
		await cell(row, field).click();
		await page.keyboard.type(text);
		await page.getByRole('heading', { name: en.documentRowsTitle }).click();
		await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	};

	for (let i = 0; i < 3; i++) {
		await page.getByRole('button', { name: en.documentRowAdd }).click();
		await expect(rows).toHaveCount(i + 1);
	}
	const product = await fieldOf('Product Characteristics');
	const method = await fieldOf('Method');
	await type(0, '_operation', '10');
	await cell(0, '_process').click();
	await page.locator('.tabulator-edit-list-item', { hasText: 'Dokumen Reflow' }).click();
	await page.getByRole('heading', { name: en.documentRowsTitle }).click();
	await expect(cell(0, '_process')).toHaveText('Dokumen Reflow');
	await type(0, product, 'Solder joint');
	await type(1, '_operation', '20');
	await type(1, method, 'Visual 100%');

	// Hapus baris ketiga yang kosong: minta konfirmasi dulu.
	page.once('dialog', (dialog) => dialog.accept());
	await rows.nth(2).getByRole('button', { name: en.documentRowDelete }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowRemoved.replace('{position}', '3'));
	await expect(rows).toHaveCount(2);

	// Ubah lagi sel yang sudah berisi: baris tetap di tempatnya.
	await cell(0, product).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Solder joint fillet');
	await page.getByRole('heading', { name: en.documentRowsTitle }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	// Header.
	await page.getByLabel('Core Team', { exact: true }).fill('PE, QA, Production');
	await page.getByRole('button', { name: en.documentHeaderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentHeaderSaved);

	// Muat ulang: semua datang lagi dari database.
	await page.reload();
	await expect(cell(0, product)).toHaveText('Solder joint fillet');
	await expect(cell(0, '_process')).toHaveText('Dokumen Reflow');
	await expect(cell(1, '_operation')).toHaveText('20');
	await expect(cell(1, method)).toHaveText('Visual 100%');
	await expect(page.getByLabel('Core Team', { exact: true })).toHaveValue('PE, QA, Production');
});
