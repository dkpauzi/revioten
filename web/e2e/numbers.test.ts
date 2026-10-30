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

	// Satu nomor, bukan set. .first(): formulir pendaftaran ada di atas
	// saringan register yang berlabel sama.
	await page.getByLabel(en.numberAsSet).uncheck();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'NB · Customer NB' });
	await page.getByLabel(en.numberType, { exact: true }).first().selectOption({ label: 'CP (SC)' });
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

test('dokumen kontrol membatalkan nomor dengan alasan, dan nomornya tetap di register', async ({ page }) => {
	// Seperti PPAP Engineer: Penyusun sekaligus Dokumen kontrol (U3 bagian 2).
	await logInAuthor(page, 'ppap.nomor', ['author', 'document_control']);
	await page.getByRole('link', { name: en.navNumbers }).click();
	await page.getByLabel(en.numberAsSet).uncheck();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'NB · Customer NB' });
	await page.getByLabel(en.numberType, { exact: true }).first().selectOption({ label: 'PFD (SH)' });
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	const status = page.getByRole('status');
	await expect(status).toContainText(en.numberRegistered.replace('{numbers}', ''));
	const number = ((await status.textContent()) ?? '').replace(en.numberRegistered.replace('{numbers}', ''), '').trim();

	const row = page.getByRole('row', { name: new RegExp(`^${number} `) });
	await expect(row).toContainText(en.numberStatusReserved);
	await row.getByRole('button', { name: en.numberVoid }).click();

	// Alasan wajib untuk Void.
	await page.getByRole('button', { name: en.numberVoid, exact: true }).last().click();
	await expect(page.getByText(en.numberReasonRequired)).toBeVisible();
	await page.getByLabel(en.numberReason.replace('{number}', number)).fill('Salah pilih customer');
	await page.getByRole('button', { name: en.numberVoid, exact: true }).last().click();
	await expect(page.getByText(en.numberStatusChanged.replace('{number}', number).replace('{status}', en.numberStatusVoid))).toBeVisible();

	// Saring status Void: nomornya masih ada, dengan alasannya.
	await page.getByLabel(en.numberFilterStatus).selectOption({ label: en.numberStatusVoid });
	await page.getByRole('button', { name: en.numberFilterApply }).click();
	await expect(page.getByRole('row', { name: new RegExp(`^${number} `) })).toContainText('Salah pilih customer');
});

test('Penyusun mendaftarkan set dengan revisi awal -04, dan nomor yang dilewati tercatat', async ({ page }) => {
	await logInAuthor(page, 'penyusun.set', ['author']);
	await page.getByRole('link', { name: en.navNumbers }).click();
	await expect(page.getByLabel(en.numberAsSet)).toBeChecked();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'NB · Customer NB' });
	await page.getByLabel(en.numberInitialRevision).fill('4');
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();

	// Test lain di file ini sudah memakai SC-NB-001 dan SC-NB-002, jadi set
	// mendapat nomor bebas berikutnya di ketiga jenis.
	const status = page.getByRole('status');
	await expect(status).toContainText('SH-NB-');
	const text = (await status.textContent()) ?? '';
	const match = /SH-NB-(\d{3}) \(-04\), SF-NB-(\d{3}) \(-04\), SC-NB-(\d{3}) \(-04\)/.exec(text);
	expect(match, text).not.toBeNull();
	expect(new Set(match!.slice(1)).size).toBe(1);
	await expect(page.getByText(en.numberSkipped.replace('{numbers}', ''))).toBeVisible();
});
