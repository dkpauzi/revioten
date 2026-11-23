// Test E2E bagian C (E-144): admin mengganti nama perusahaan dan logo cetakan
// di Admin → Settings, lalu kembali ke logo bawaan. Sejak D-223 admin wajib
// menjawab apakah dokumen direvisi; jawaban "ya" membuat PPAP Engineer
// diminta mengajukan perubahan logo di halaman Changes.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu, tinyPNG } from './helpers.ts';

test('admin mengganti nama perusahaan dan logo cetakan, lalu kembali ke logo bawaan', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	const company = page.locator('section', { has: page.getByRole('heading', { name: en.companyTitle }) });
	await expect(company.getByText(en.companyLogoDefault)).toBeVisible();
	await expect(company.getByRole('img', { name: en.companyLogoAlt })).toBeVisible();

	await company.getByLabel(en.companyName, { exact: true }).fill('PT. SIIX EMS INDONESIA');
	await company.getByRole('button', { name: en.companyNameSave }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyNameSaved);

	// Tanpa jawaban "revisi dokumen?" ditolak di layar (D-223).
	await company.getByLabel(en.companyLogoFile).setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: Buffer.from('bukan png') });
	await company.getByRole('button', { name: en.companyLogoUpload }).click();
	await expect(company.getByRole('alert')).toHaveText(en.companyLogoReviseChoose);

	// Bukan PNG ditolak server.
	await company.getByLabel(en.companyLogoReviseNo).check();
	await company.getByRole('button', { name: en.companyLogoUpload }).click();
	await expect(company.getByRole('alert')).not.toHaveText(en.companyLogoReviseChoose);
	await expect(company.getByRole('alert')).toBeVisible();

	await company.getByLabel(en.companyLogoFile).setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: tinyPNG() });
	await company.getByRole('button', { name: en.companyLogoUpload }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyLogoUploaded);
	await expect(company.getByText(/^Uploaded logo, SHA-256 /)).toBeVisible();

	await company.getByLabel(en.companyLogoReviseNo).check();
	await company.getByRole('button', { name: en.companyLogoReset }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyLogoResetDone);
	await expect(company.getByText(en.companyLogoDefault)).toBeVisible();
});
