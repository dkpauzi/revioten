// Test E2E Tahap 7: mode migrasi, dokumen lama di revisi -04, riwayat teks,
// verifikasi hardcopy oleh petugas lain, dan dasbor.
// Berjalan sesudah m5-changes.test.ts: customer CH dan model C-1 sudah ada.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu, openTool } from './helpers.ts';

test('M7: dokumen lama masuk di revisi 4, diverifikasi orang lain, dan tampil di dasbor', async ({ page }) => {
	// Admin menyalakan mode migrasi dan membuat dua petugas migrasi.
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	await page.getByLabel(en.settingsMigration, { exact: true }).check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
	const first = await createWorkAccount(page, 'Petugas migrasi satu', 'migrasi1.e2e', ['ppap_engineer']);
	const second = await createWorkAccount(page, 'Petugas migrasi dua', 'migrasi2.e2e', ['ppap_engineer']);
	await logOut(page);

	// Petugas pertama membuka menu Legacy documents (D-215) dan memasukkan CP lama di revisi 4.
	await logInFirstTime(page, 'migrasi1.e2e', first, 'dokumen lama masuk apa adanya');
	await openFromMenu(page, en.navGroupDocuments, en.navLegacy);
	await expect(page.getByRole('heading', { name: en.legacyTitle, level: 1 })).toBeVisible();
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await expect(page.getByTestId('migration-ready')).toContainText(en.legacyReadyModeOn);
	const set = page.getByTestId('migration-set');
	const row = (type: string) => set.locator('tr').filter({ has: page.getByRole('rowheader', { name: type, exact: true }) });

	// Bukan satu set: hanya CP. Revisi kosong ditolak per kolom.
	await page.getByRole('checkbox', { name: en.legacyOneSet }).uncheck();
	await row('CP').getByLabel(en.legacyNumberOf.replace('{type}', 'CP')).fill('SC-CH-050');
	await page.getByRole('checkbox', { name: 'C-1' }).check();
	await page.getByRole('button', { name: en.legacySetSubmit }).click();
	await expect(row('CP').locator('.error').first()).toBeVisible();
	await row('CP').getByLabel(en.legacyRevisionOf.replace('{type}', 'CP')).fill('4');
	await row('CP').getByLabel(en.legacyDateOf.replace('{type}', 'CP')).fill('2025-03-10');
	await row('CP').getByLabel(en.legacyPreparedOf.replace('{type}', 'CP')).fill('Budi Santoso');
	await row('CP').getByLabel(en.legacyApprovedOf.replace('{type}', 'CP')).fill('Siti Aminah, Agus Wijaya');
	await page.getByRole('button', { name: en.legacySetSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.legacySetDone.replace('{numbers}', 'SC-CH-050'));
	const board = page.getByTestId('migration-board');
	await expect(board.getByRole('row', { name: /SC-CH-050/ })).toContainText(en.legacyStateEmpty);

	// Satu set (bawaan): nomor PFD dengan revisinya; PFMEA dan CP mengikuti nomor berjalan 060.
	await page.getByRole('checkbox', { name: 'C-1' }).check();
	const pfdNumber = row('PFD').getByLabel(en.legacyNumberOf.replace('{type}', 'PFD'));
	await pfdNumber.fill('SH-CH-060-02');
	await pfdNumber.blur();
	await expect(pfdNumber).toHaveValue('SH-CH-060');
	await expect(row('PFD').getByLabel(en.legacyRevisionOf.replace('{type}', 'PFD'))).toHaveValue('2');
	await expect(row('CP').getByLabel(en.legacyNumberOf.replace('{type}', 'CP'))).toHaveAttribute('placeholder', 'SC-CH-060');
	for (const [type, revision] of [['PFD', '2'], ['PFMEA', '3'], ['CP', '4']]) {
		await row(type).getByLabel(en.legacyRevisionOf.replace('{type}', type)).fill(revision);
		await row(type).getByLabel(en.legacyDateOf.replace('{type}', type)).fill('2025-10-23');
	}
	await row('PFD').getByLabel(en.legacyPreparedOf.replace('{type}', 'PFD')).fill('Budi Santoso');
	await row('PFD').getByLabel(en.legacyApprovedOf.replace('{type}', 'PFD')).fill('Siti Aminah');
	await page.getByRole('button', { name: en.legacySetSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.legacySetDone.replace('{numbers}', 'SH-CH-060, SF-CH-060, SC-CH-060'));
	await expect(board.getByRole('row')).toHaveCount(5);
	const number = { number: 'SC-CH-050' };

	// Halaman dokumen: riwayat teks, pembanding Excel, dan scan hardcopy.
	await page.getByRole('link', { name: number.number }).click();
	await openTool(page, en.migrationTitle);
	const baseline = page.locator('section', { has: page.getByRole('heading', { name: en.migrationTitle, exact: true }) });
	await expect(baseline).toContainText('Budi Santoso');
	await baseline.getByRole('button', { name: en.historyAdd }).click();
	await baseline.getByLabel(`${en.historyRevision} 1`).fill('-00');
	await baseline.getByLabel(`${en.historyDate} 1`).fill('2023-01-05');
	await baseline.getByLabel(`${en.historyDescription} 1`).fill('Initial release');
	await baseline.getByRole('button', { name: en.historySave }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.historySaved);

	await openTool(page, en.excelCompareTitle);
	const compare = page.locator('section', { has: page.getByRole('heading', { name: en.excelCompareTitle }) });
	await compare.getByLabel(en.excelCompareFile).setInputFiles({ name: 'old.xlsx', mimeType: 'application/octet-stream', buffer: Buffer.from('bukan excel') });
	await compare.getByRole('button', { name: en.excelCompareButton }).click();
	await expect(compare.getByRole('alert')).toBeVisible();

	// Scan hardcopy tidak diunggah: untuk baseline migrasi scan opsional (D-186).
	await openTool(page, en.revisions);
	const revisions = page.locator('section', { has: page.getByRole('heading', { name: en.revisions, exact: true }) });
	// Baseline tidak punya tombol rilis biasa, dan pemasuk tidak bisa memverifikasi sendiri.
	await expect(revisions.getByRole('button', { name: en.revisionRelease.replace('{revision}', '4') })).toHaveCount(0);
	await openTool(page, en.migrationTitle);
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('alert')).toHaveText(en.migrationVerifierNotAllowed);
	const url = page.url();
	await logOut(page);

	// Petugas kedua mencocokkan dengan hardcopy dan mencatat menit usahanya.
	await logInFirstTime(page, 'migrasi2.e2e', second, 'hardcopy selalu jadi acuan');
	await page.goto(url);
	await openTool(page, en.migrationTitle);
	await baseline.getByLabel(en.migrationEffort, { exact: true }).fill('30');
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.migrationVerified.replace('{number}', number.number));
	await expect(page.locator('.wb-sub .badge[data-status]')).toHaveText(en.documentStatusReleasedMigrated);
	await expect(baseline).toContainText('-00');

	// Dasbor menghitung dokumen migrasi dan menitnya.
	await openFromMenu(page, en.navGroupDocuments, en.navDashboard);
	await expect(page.getByRole('heading', { name: en.dashboardTitle, level: 1 })).toBeVisible();
	const migration = page.locator('section', { has: page.getByRole('heading', { name: en.dashboardMigration, exact: true }) });
	await expect(migration).toContainText(en.dashboardMinutes.replace('{minutes}', '30'));
	const documents = page.locator('section', { has: page.getByRole('heading', { name: en.dashboardDocuments }) });
	await expect(documents).toContainText(en.documentStatusReleasedMigrated);
});
