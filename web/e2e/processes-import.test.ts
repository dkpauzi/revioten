// Test E2E I2: PPAP Engineer mengimpor nama proses dan aliasnya dari berkas
// Excel. Nama proses di fixture fiktif dan berawalan IMP supaya tidak
// bertabrakan dengan proses dari test lain.
import { expect, test } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { en } from '../src/lib/i18n/en.ts';
import { logInAsMaster, openFromMenu } from './helpers.ts';

const fixture = fileURLToPath(new URL('./fixtures/processes-import.xlsx', import.meta.url));

test('I2: import proses dari Excel, lalu unggah ulang melewati semuanya', async ({ page }) => {
	await logInAsMaster(page);
	await openFromMenu(page, en.navGroupTemplates, en.navProcessImport);
	await expect(page.getByRole('heading', { name: en.processImportTitle })).toBeVisible();

	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: en.processImportTemplate }).click();
	expect((await download).suggestedFilename()).toBe('processes-import-template.xlsx');

	await page.getByLabel(en.processImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.processImportPreview, exact: true }).click();
	const status = page.getByRole('status');
	await expect(status).toContainText(en.processImportPreviewSummary.split(':')[0]);
	await expect(page.locator('tr[data-import="new"]')).toHaveCount(2);
	// Alias IMP Printing sudah dipakai baris sebelumnya.
	await expect(page.locator('tr[data-import="invalid"]')).toContainText('IMP Wave solder');

	await page.getByRole('button', { name: en.processImportRun, exact: true }).click();
	await expect(page.locator('tr[data-import="new"]').first()).toContainText(en.processImportStatusCreated);

	// Proses baru tampil di halaman Processes, dengan aliasnya.
	await page.getByRole('link', { name: en.processImportBack }).click();
	await expect(page.getByRole('row', { name: /IMP Solder paste print/ })).toContainText('IMP Stencil print');

	// Berkas yang sama diunggah lagi: proses yang benar sudah ada, tombol Import tidak muncul.
	await page.getByRole('link', { name: en.processesImportLink }).click();
	await page.getByLabel(en.processImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.processImportPreview, exact: true }).click();
	await expect(page.locator('tr[data-import="existing"]')).toHaveCount(2);
	await expect(page.getByRole('button', { name: en.processImportRun, exact: true })).toHaveCount(0);
});
