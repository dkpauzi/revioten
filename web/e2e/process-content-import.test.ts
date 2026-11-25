// Test E2E I3: PPAP Engineer mengimpor isi process template dari berkas
// Excel. Prosesnya diimpor dulu dari fixture sendiri (nama fiktif berawalan IMP3,
// aman diunggah lagi), supaya tidak bergantung pada test I2.
import { expect, test } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { en } from '../src/lib/i18n/en.ts';
import { logInAsMaster, openFromMenu } from './helpers.ts';

const processFixture = fileURLToPath(new URL('./fixtures/process-content-processes.xlsx', import.meta.url));
const fixture = fileURLToPath(new URL('./fixtures/process-content-import.xlsx', import.meta.url));

test('I3: isi process template dari Excel masuk ke versi draft, lalu unggah ulang melewati semuanya', async ({ page }) => {
	await logInAsMaster(page);
	// Pastikan proses IMP3 Final inspection ada.
	await openFromMenu(page, en.navGroupTemplates, en.navProcessImport);
	await page.getByLabel(en.processImportFile).setInputFiles(processFixture);
	await page.getByRole('button', { name: en.processImportPreview, exact: true }).click();
	await expect(page.getByRole('status')).toBeVisible();
	if (await page.getByRole('button', { name: en.processImportRun, exact: true }).isVisible()) {
		await page.getByRole('button', { name: en.processImportRun, exact: true }).click();
		await expect(page.getByRole('status')).toContainText(en.processImportDoneSummary.split(' ')[1]);
	}

	await openFromMenu(page, en.navGroupTemplates, en.navContentImport);
	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: en.contentImportTemplate }).click();
	expect((await download).suggestedFilename()).toBe('process-template-content-import-template.xlsx');

	await page.getByLabel(en.contentImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.contentImportPreview, exact: true }).click();
	await expect(page.getByRole('status')).toContainText(en.contentImportPreviewSummary.split(':')[0]);
	await expect(page.locator('tr[data-import="new"]')).toHaveCount(3);
	await expect(page.locator('tr[data-import="invalid"]')).toContainText('IMP3 Nothing');
	await expect(page.locator('li[data-version="new_template"]')).toContainText('IMP3 Final inspection');

	await page.getByRole('button', { name: en.contentImportRun, exact: true }).click();
	await expect(page.locator('tr[data-import="new"]').first()).toContainText(en.contentImportStatusCreated);

	// Berkas yang sama diunggah lagi: poin yang benar sudah ada, tombol Import tidak muncul.
	await page.getByLabel(en.contentImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.contentImportPreview, exact: true }).click();
	await expect(page.locator('tr[data-import="existing"]')).toHaveCount(3);
	await expect(page.getByRole('button', { name: en.contentImportRun, exact: true })).toHaveCount(0);

	// Tautan kembali membuka Process templates, dan tautan import ada di sana.
	await page.getByRole('link', { name: en.contentImportBack }).click();
	await expect(page.getByRole('link', { name: en.processTemplateImportLink })).toBeVisible();
});
