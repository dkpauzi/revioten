// Test E2E D-195, D-196, D-197: akun Viewer membaca dokumen terbit tanpa
// draft, daftar review berkala, dan laporan kinerja PPAP Engineer.
// Berjalan sesudah m5-changes.test.ts: CP customer CH sudah terbit di revisi 0
// dan punya draft revisi 1.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, mainMenu, openFromMenu } from './helpers.ts';

async function chooseCH(page: Page, label: string) {
	const select = page.getByLabel(label, { exact: true });
	const value = await select.locator('option', { hasText: /^CH ·/ }).getAttribute('value');
	await select.selectOption(value ?? '');
}

test('D-195: akun Viewer membaca revisi terbit, bukan draft, tanpa tombol ubah', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Internal Audit E2E', 'audit.e2e', []);
	await logOut(page);
	await logInFirstTime(page, 'audit.e2e', temporary, 'audit internal setiap bulan');

	// Viewer melihat dokumen terbit (menu draft tidak ada, diperiksa di akhir).
	await openFromMenu(page, en.navGroupDocuments, en.navReleasedDocuments);
	await chooseCH(page, en.documentCustomer);
	const row = page.getByRole('row').filter({ has: page.getByRole('link', { name: /^SC-CH-/ }) }).first();
	await expect(row).toBeVisible();
	// CP sedang punya draft revisi 1; yang tampil revisi 0.
	await expect(row.getByRole('cell').nth(2)).toHaveText('0');
	await row.getByRole('link').click();
	await expect(page.getByText(en.releasedViewHint)).toBeVisible();
	await expect(page.locator('.workbook, .tabulator').first()).toBeVisible();
	await expect(page.getByRole('button', { name: en.documentRowAdd })).toHaveCount(0);

	// Daftar review berkala: dokumen baru terbit belum jatuh tempo.
	await openFromMenu(page, en.navGroupDocuments, en.navPeriodicReviews);
	await page.getByRole('button', { name: en.reviewTabAll, exact: true }).click();
	await expect(page.getByRole('link', { name: /SC-CH-/ }).first()).toBeVisible();
	await expect(page.getByRole('button', { name: en.reviewStart })).toHaveCount(0);

	// Menu draft (L-4) dan laporan kinerja (L-5) tidak untuk Viewer.
	await mainMenu(page).getByRole('button', { name: en.navGroupDocuments, exact: true }).click();
	await expect(mainMenu(page).getByRole('link', { name: en.navReleasedDocuments, exact: true })).toBeVisible();
	await expect(mainMenu(page).getByRole('link', { name: en.navDocuments, exact: true })).toHaveCount(0);
	await expect(mainMenu(page).getByRole('link', { name: en.navPerformance, exact: true })).toHaveCount(0);
});

test('D-197: PPAP Engineer membaca laporan kinerja minggu ini dan tahun ini', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();

	await openFromMenu(page, en.navGroupDocuments, en.navPerformance);
	await expect(page.getByRole('heading', { name: en.performanceReceived, exact: true })).toBeVisible();
	await page.getByRole('button', { name: en.performanceYear, exact: true }).click();
	// Perubahan di m5-changes diajukan tahun ini oleh orang ini.
	await expect(page.getByTestId('perf-submitted')).not.toHaveText('0');
	await expect(page.getByRole('cell', { name: 'Pemohon perubahan' })).toBeVisible();
});
