// Test E2E I1 (D-224): admin mengimpor akun dari berkas Excel. Preview bisa
// kapan saja dan tidak menyimpan; Import hanya saat Migration mode menyala; baris yang salah
// dilaporkan; akun hasil import login dengan password123 lalu wajib ganti.
// Nama di berkas fixture fiktif.
import { expect, test, type Page } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

const fixture = fileURLToPath(new URL('./fixtures/accounts-import.xlsx', import.meta.url));

async function setMigrationMode(page: Page, on: boolean) {
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	const box = page.getByLabel(en.settingsMigration, { exact: true });
	if (on) await box.check();
	else await box.uncheck();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
}

test('I1: admin mengimpor akun dari Excel, dan akun baru login dengan password123', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navAccountImport);
	await expect(page.getByRole('heading', { name: en.accountImportTitle, level: 1 })).toBeVisible();
	// Migration mode mati: Preview tetap bisa, tombol Import nonaktif.
	await expect(page.getByText(en.accountImportMigrationOff)).toBeVisible();
	await page.getByLabel(en.accountImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.accountImportPreview }).click();
	await expect(page.getByRole('status')).toHaveText(
		en.accountImportPreviewSummary.replace('{new}', '2').replace('{existing}', '0').replace('{invalid}', '1')
	);
	await expect(page.getByRole('button', { name: en.accountImportRun })).toBeDisabled();
	// Template bisa diunduh.
	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: en.accountImportTemplate }).click();
	expect((await download).suggestedFilename()).toBe('accounts-import-template.xlsx');

	await setMigrationMode(page, true);
	await openFromMenu(page, en.navGroupAdmin, en.navAccountImport);
	await page.getByLabel(en.accountImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.accountImportPreview }).click();
	const status = page.getByRole('status');
	await expect(status).toHaveText(en.accountImportPreviewSummary.replace('{new}', '2').replace('{existing}', '0').replace('{invalid}', '1'));
	await expect(page.locator('tr[data-import="invalid"]')).toContainText('Penyihir');

	await page.getByRole('button', { name: en.accountImportRun }).click();
	const done = en.accountImportDoneSummary.replace('{new}', '2').replace('{existing}', '0').replace('{invalid}', '1').split('{time}')[0];
	await expect(status).toContainText(done);
	await expect(page.locator('tr[data-import="new"]')).toHaveCount(2);
	await expect(page.locator('tr[data-import="new"]').first()).toContainText(en.accountImportStatusCreated);

	// Daftar akun menunjukkan batas password sementara.
	await page.getByRole('link', { name: en.accountImportBack }).click();
	const row = page.getByRole('row', { name: /anisa\.import/ });
	await expect(row).toContainText(en.accountStatusMustChange);
	await expect(row).toContainText(en.accountStatusTemporaryUntil.split('{time}')[0]);

	// Berkas yang sama diunggah lagi: semua sudah ada.
	await openFromMenu(page, en.navGroupAdmin, en.navAccountImport);
	await page.getByLabel(en.accountImportFile).setInputFiles(fixture);
	await page.getByRole('button', { name: en.accountImportPreview }).click();
	await expect(status).toHaveText(en.accountImportPreviewSummary.replace('{new}', '0').replace('{existing}', '2').replace('{invalid}', '1'));

	// Matikan lagi, supaya test lain mulai dengan Migration mode mati.
	await setMigrationMode(page, false);
	await logOut(page);

	await logInFirstTime(page, 'anisa.import', 'password123', 'akun dari excel lebih cepat');
});
