// Test E2E Tahap 2: register nomor dokumen lewat layar. M2.1: pola
// menghasilkan nomor seperti SC-KT-001, dan pengaturan pola ada di master
// data. Pendaftaran bersamaan (M2.2) dibuktikan test Go TestRegisterConcurrent.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime } from './helpers.ts';

// ensureCustomer memastikan customer dengan kode itu ada di GR1, lewat API dari
// browser yang sudah login sebagai admin.
async function ensureCustomer(page: Page, code: string): Promise<void> {
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async (code) => {
		const customers = await (await fetch('/api/v1/customers')).json();
		if (customers.items.some((c: { code: string }) => c.code === code)) {
			return;
		}
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		await fetch('/api/v1/customers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ code, name: `Customer ${code}`, project_group_id: group.id })
		});
	}, code);
}

// logInAuthor membuat akun Penyusun baru (bila login itu belum ada) lewat akun
// admin, lalu masuk dengannya.
async function logInAuthor(page: Page, login: string, roles: string[]): Promise<void> {
	await logIn(page);
	await ensureCustomer(page, 'NB');
	const temporary = await createWorkAccount(page, `Penyusun ${login}`, login, roles);
	await page.getByRole('button', { name: en.logout }).click();
	await logInFirstTime(page, login, temporary, 'nomor dokumen tidak pernah dipakai ulang');
}

test('Penyusun mendaftarkan nomor CP dan mendapat nomor berurutan', async ({ page }) => {
	await logInAuthor(page, 'penyusun.nomor', ['author']);
	await page.getByRole('link', { name: en.navNumbers }).click();

	await page.getByLabel(en.numberCustomer, { exact: true }).selectOption({ label: 'NB · Customer NB' });
	await page.getByLabel(en.numberType, { exact: true }).selectOption({ label: 'CP (SC)' });
	await expect(page.getByText(en.numberNext.replace('{number}', 'SC-NB-001'))).toBeVisible();

	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.numberRegistered.replace('{numbers}', 'SC-NB-001'));
	await expect(page.getByText(en.numberNext.replace('{number}', 'SC-NB-002'))).toBeVisible();

	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.numberRegistered.replace('{numbers}', 'SC-NB-002'));
});

test('Viewer tidak mendapat formulir pendaftaran', async ({ page }) => {
	await logInAuthor(page, 'viewer.nomor', []);
	await page.getByRole('link', { name: en.navNumbers }).click();
	await expect(page.getByRole('heading', { name: en.numbersTitle })).toBeVisible();
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toHaveCount(0);
});

test('admin mengubah pola nomor di master data', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navDocumentTypes }).click();
	const pattern = page.getByLabel(en.numberPattern, { exact: true });
	await expect(pattern).toHaveValue('{prefix}-{customer}-{seq}');

	// Pola tanpa pemisah ditolak, karena nomor lama tidak bisa dibaca kembali.
	await pattern.fill('{prefix}{customer}{seq}');
	await page.getByRole('button', { name: en.factoryCodeSave }).click();
	await expect(page.getByText(en.numberPatternInvalid)).toBeVisible();

	await pattern.fill('{prefix}-{customer}-{seq}');
	await page.getByRole('button', { name: en.factoryCodeSave }).click();
	await expect(page.getByText(en.factoryCodeSaved.replace('{code}', 'S'))).toBeVisible();
});
