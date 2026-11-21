// Test E2E Tahap 2: register nomor dokumen lewat layar. M2.1: pola
// menghasilkan nomor seperti SC-KT-001, dan pengaturan pola ada di master
// data. Pendaftaran bersamaan (M2.2) dibuktikan test Go TestRegisterConcurrent,
// dan impor bersamaan (M2.5) oleh TestImportConcurrent.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { asMaster, createWorkAccount, ensureProjectGroup, logIn, logInAsMaster, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// ensureCustomer memastikan customer dengan kode itu ada di GR1, lewat API
// sebagai PPAP Engineer (E-11, D-209) di jendela terpisah.
async function ensureCustomer(page: Page, code: string): Promise<void> {
	await asMaster(page, (m) => ensureCustomerAsMaster(m, code));
}

async function ensureCustomerAsMaster(page: Page, code: string): Promise<void> {
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
// admin, lalu masuk dengannya. Customer dengan kode itu (bawaan NB) dipastikan ada.
async function logInAuthor(page: Page, login: string, roles: string[], customer = 'NB'): Promise<void> {
	await logIn(page);
	await ensureCustomer(page, customer);
	const temporary = await createWorkAccount(page, `Penyusun ${login}`, login, roles);
	await logOut(page);
	await logInFirstTime(page, login, temporary, 'nomor dokumen tidak pernah dipakai ulang');
}

test('Penyusun mendaftarkan nomor CP dan mendapat nomor berurutan', async ({ page }) => {
	await logInAuthor(page, 'penyusun.nomor', ['ppap_engineer']);
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);

	// Satu nomor, bukan set. .first(): formulir pendaftaran ada di atas
	// saringan register yang berlabel sama.
	await page.getByLabel(en.numberAsSet).uncheck();
	// Tunggu formulir daftar tampil dulu: saringan register juga punya kotak "Customer".
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
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
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await expect(page.getByRole('heading', { name: en.numbersTitle })).toBeVisible();
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toHaveCount(0);
});

test('PPAP Engineer mengubah pola nomor di master data', async ({ page }) => {
	// Pola nomor adalah master data PPAP Engineer (E-11, D-209).
	await logInAsMaster(page);
	await openFromMenu(page, en.navGroupMaster, en.navDocumentTypes);
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
	await logInAuthor(page, 'ppap.nomor', ['ppap_engineer']);
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await page.getByLabel(en.numberAsSet).uncheck();
	// Tunggu formulir daftar tampil dulu: saringan register juga punya kotak "Customer".
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
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
	await logInAuthor(page, 'penyusun.set', ['ppap_engineer']);
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await expect(page.getByLabel(en.numberAsSet)).toBeChecked();
	// Tunggu formulir daftar tampil dulu: saringan register juga punya kotak "Customer".
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
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

test('dokumen kontrol mengimpor nomor lama: pratinjau, laporan, lalu impor', async ({ page }) => {
	await logInAuthor(page, 'dokumen.impor', ['ppap_engineer'], 'LG');
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await page.getByRole('link', { name: en.numberImportLink }).click();
	await expect(page.getByRole('heading', { name: en.numberImportTitle })).toBeVisible();

	await page.getByLabel(en.numberCustomer, { exact: true }).selectOption({ label: 'LG · Customer LG' });
	await page.getByLabel(en.numberImportStatus).selectOption({ label: en.numberStatusReleased });
	const lines = ['SH-LG-001', 'SH-LG-002\tModel A', '', 'SC-LG-001', 'SC-LG-004', 'SC-LG-004', 'SC-NB-001', 'hello'];
	await page.getByLabel(en.numberImportText).fill(lines.join('\n'));
	await page.getByRole('button', { name: en.numberImportPreview }).click();

	// Pratinjau: empat nomor akan masuk; baris lain dilaporkan dengan alasannya.
	await expect(page.getByRole('status')).toHaveText(en.numberImportPreviewed.replace('{count}', '4'));
	await expect(page.getByRole('row', { name: /^6 SC-LG-004 / })).toContainText(en.numberImportLineDuplicate.replace('{line}', '5'));
	await expect(page.getByRole('row', { name: /^7 SC-NB-001 / })).toContainText(en.numberImportLineMismatch);
	await expect(page.getByRole('row', { name: /^8 hello / })).toContainText(en.numberImportLineInvalid);
	// Lubang CP 002 sampai 003, dan nomor berjalan yang tidak dimiliki ketiga jenis.
	await expect(page.getByRole('row', { name: /^LG CP 2 004 / })).toContainText('002–003');
	await expect(page.getByRole('row', { name: /^LG 001 / })).toContainText('PFD, CP');
	await expect(page.getByRole('row', { name: /^LG 004 / })).toContainText('CP');

	await page.getByRole('button', { name: en.numberImportRun, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.numberImportDone.replace('{count}', '4'));
	await expect(page.getByRole('button', { name: en.numberImportRun, exact: true })).toHaveCount(0);

	// Nomor lama ada di register dengan status pilihan dan teks aslinya.
	await page.getByRole('link', { name: en.numberImportBack }).click();
	await page.getByLabel(en.numberFilterSearch).fill('SC-LG-004');
	await page.getByRole('button', { name: en.numberFilterApply }).click();
	await expect(page.getByRole('row', { name: /^SC-LG-004 / })).toContainText(en.numberStatusReleased);
});
