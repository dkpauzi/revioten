// Test E2E I4: PPAP Engineer mengimpor nomor lama dari berkas Excel, dengan
// revisi di belakang nomor (opsi b). Revisinya lalu mengisi kolom revisi di
// menu Legacy documents. Customer IM berasal dari fixture I5 (nama fiktif).
import { expect, test } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { en } from '../src/lib/i18n/en.ts';
import { logInAsMaster, openFromMenu } from './helpers.ts';

const masterFixture = fileURLToPath(new URL('./fixtures/master-import.xlsx', import.meta.url));
const fixture = fileURLToPath(new URL('./fixtures/numbers-import.xlsx', import.meta.url));

test('I4: nomor lama dari Excel dengan revisi, lalu revisi terisi di Legacy documents', async ({ page }) => {
	await logInAsMaster(page);
	// Pastikan customer IM ada (berkas I5 aman diunggah lagi).
	await openFromMenu(page, en.navGroupMaster, en.navMasterImport);
	await page.getByLabel(en.masterImportFile).setInputFiles(masterFixture);
	await page.getByRole('button', { name: en.masterImportPreview }).click();
	await expect(page.getByRole('status')).toBeVisible();
	if (await page.getByRole('button', { name: en.masterImportRun }).isVisible()) {
		await page.getByRole('button', { name: en.masterImportRun }).click();
		await expect(page.getByRole('status')).toContainText(en.masterImportDoneSummary.split('{new}')[0]);
	}

	await openFromMenu(page, en.navGroupDocuments, en.numberImportLink);
	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: en.numberImportFileTemplate }).click();
	expect((await download).suggestedFilename()).toBe('legacy-numbers-import-template.xlsx');

	await page.getByLabel(en.numberImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.numberImportFilePreview }).click();
	const status = page.getByRole('status');
	await expect(status).toHaveText(en.numberImportPreviewed.replace('{count}', '6'));
	await expect(page.locator('tr[data-import="unknown_customer"]')).toContainText('SH-ZZ-001');
	const first = page.locator('tr[data-import="imported"]').first();
	await expect(first).toContainText('SH-IM-001');
	await expect(first).toContainText('-02');

	await page.getByRole('button', { name: en.numberImportFileRun }).click();
	await expect(status).toHaveText(en.numberImportDone.replace('{count}', '6'));

	// Berkas yang sama diunggah lagi: tidak ada nomor baru.
	await page.getByRole('button', { name: en.numberImportFilePreview }).click();
	await expect(status).toHaveText(en.numberImportNothing);

	// Menu Legacy documents: revisi dari import mengisi kolom revisi.
	await openFromMenu(page, en.navGroupDocuments, en.navLegacy);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'IM · Customer Import' });
	const set = page.getByTestId('migration-set');
	const first_number = set.getByRole('textbox').first();
	await first_number.fill('SH-IM-001');
	await first_number.blur();
	const revisions = set.getByRole('spinbutton');
	await expect(revisions.nth(0)).toHaveValue('2');
	await expect(revisions.nth(1)).toHaveValue('2');
	await expect(revisions.nth(2)).toHaveValue('3');
});
