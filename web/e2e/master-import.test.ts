// Test E2E I5: PPAP Engineer mengimpor Project Group, customer, group model,
// model, dan part number dari berkas Excel. Preview tidak menyimpan; baris
// yang salah dilaporkan; berkas yang sama bisa diunggah lagi. Nama di berkas
// fixture fiktif.
import { expect, test } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { en } from '../src/lib/i18n/en.ts';
import { logInAsMaster, openFromMenu } from './helpers.ts';

const fixture = fileURLToPath(new URL('./fixtures/master-import.xlsx', import.meta.url));

function counts(text: string, n: string, existing: string, invalid: string): string {
	return text.replace('{new}', n).replace('{existing}', existing).replace('{invalid}', invalid);
}

test('I5: PPAP Engineer mengimpor customer dan model dari Excel', async ({ page }) => {
	await logInAsMaster(page);
	await openFromMenu(page, en.navGroupMaster, en.navMasterImport);
	await expect(page.getByRole('heading', { name: en.masterImportTitle, level: 1 })).toBeVisible();
	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: en.masterImportTemplate }).click();
	expect((await download).suggestedFilename()).toBe('master-data-import-template.xlsx');

	await page.getByLabel(en.masterImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.masterImportPreview }).click();
	const status = page.getByRole('status');
	await expect(status).toHaveText(counts(en.masterImportPreviewSummary, '3', '0', '1'));
	await expect(page.locator('tr[data-import="invalid"]')).toContainText('X9');

	await page.getByRole('button', { name: en.masterImportRun }).click();
	await expect(status).toHaveText(counts(en.masterImportDoneSummary, '3', '0', '1'));
	await expect(page.locator('tr[data-import="new"]').first()).toContainText(
		en.masterImportStatusCreated.replace('{what}', `${en.masterImportCreates_project_group}, ${en.masterImportCreates_customer}`)
	);

	// Berkas yang sama diunggah lagi: semua sudah ada.
	await page.getByRole('button', { name: en.masterImportPreview }).click();
	await expect(status).toHaveText(counts(en.masterImportPreviewSummary, '0', '3', '1'));

	// Customer baru tampil di Customers.
	await page.getByRole('link', { name: en.masterImportBack }).click();
	await expect(page.getByRole('cell', { name: 'Customer Import' })).toBeVisible();
});
