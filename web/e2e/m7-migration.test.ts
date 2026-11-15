// Test E2E Tahap 7: mode migrasi, dokumen lama di revisi -04, riwayat teks,
// verifikasi hardcopy oleh petugas lain, dan dasbor.
// Berjalan sesudah m5-changes.test.ts: customer CH dan model C-1 sudah ada.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('M7: dokumen lama masuk di revisi 4, diverifikasi orang lain, dan tampil di dasbor', async ({ page }) => {
	// Admin menyalakan mode migrasi dan membuat dua petugas migrasi.
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	await page.getByLabel(en.settingsMigration, { exact: true }).check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
	const first = await createWorkAccount(page, 'Petugas migrasi satu', 'migrasi1.e2e', ['migration', 'author']);
	const second = await createWorkAccount(page, 'Petugas migrasi dua', 'migrasi2.e2e', ['migration']);
	await logOut(page);

	// Petugas pertama mendaftarkan nomor dan memasukkan CP lama di revisi 4.
	await logInFirstTime(page, 'migrasi1.e2e', first, 'dokumen lama masuk apa adanya');
	const number = await page.evaluate(async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const set = await (
			await fetch('/api/v1/document-numbers/sets', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', 'Idempotency-Key': crypto.randomUUID() },
				body: JSON.stringify({ customer_id: customer.id })
			})
		).json();
		const free = await (await fetch(`/api/v1/migration/numbers?customer_id=${customer.id}`)).json();
		const ids = new Set((set.items as { id: string }[]).map((n) => n.id));
		const cp = (free.items as { id: string; number: string }[]).filter((n) => ids.has(n.id)).at(-1)!;
		return cp;
	});
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	const form = page.locator('section', { has: page.getByRole('heading', { name: en.migrateTitle }) });
	await form.getByLabel(en.migrateNumber, { exact: true }).selectOption(number.id);
	await form.getByRole('checkbox', { name: 'C-1' }).check();
	// Tanpa revisi ditolak.
	await form.getByRole('button', { name: en.migrateSubmit }).click();
	await expect(form.locator('.error').first()).toBeVisible();
	await form.getByLabel(en.migrateRevision, { exact: true }).fill('4');
	await form.getByLabel(en.migrateEffectiveDate, { exact: true }).fill('2025-03-10');
	await form.getByLabel(en.migratePreparedBy, { exact: true }).fill('Budi Santoso');
	await form.getByLabel(en.migrateApprovedBy, { exact: true }).fill('Siti Aminah, Agus Wijaya');
	await form.getByRole('button', { name: en.migrateSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.migrateDone.replace('{number}', number.number).replace('{revision}', '4'));

	// Halaman dokumen: riwayat teks, pembanding Excel, dan scan hardcopy.
	await page.getByRole('link', { name: number.number }).click();
	const baseline = page.locator('section', { has: page.getByRole('heading', { name: en.migrationTitle, exact: true }) });
	await expect(baseline).toContainText('Budi Santoso');
	await baseline.getByRole('button', { name: en.historyAdd }).click();
	await baseline.getByLabel(`${en.historyRevision} 1`).fill('-00');
	await baseline.getByLabel(`${en.historyDate} 1`).fill('2023-01-05');
	await baseline.getByLabel(`${en.historyDescription} 1`).fill('Initial release');
	await baseline.getByRole('button', { name: en.historySave }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.historySaved);

	const compare = page.locator('section', { has: page.getByRole('heading', { name: en.excelCompareTitle }) });
	await compare.getByLabel(en.excelCompareFile).setInputFiles({ name: 'old.xlsx', mimeType: 'application/octet-stream', buffer: Buffer.from('bukan excel') });
	await compare.getByRole('button', { name: en.excelCompareButton }).click();
	await expect(compare.getByRole('alert')).toBeVisible();

	// Scan hardcopy tidak diunggah: untuk baseline migrasi scan opsional (D-186).
	const revisions = page.locator('section', { has: page.getByRole('heading', { name: en.revisions, exact: true }) });
	// Baseline tidak punya tombol rilis biasa, dan pemasuk tidak bisa memverifikasi sendiri.
	await expect(revisions.getByRole('button', { name: en.revisionRelease.replace('{revision}', '4') })).toHaveCount(0);
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('alert')).toHaveText(en.migrationVerifierNotAllowed);
	const url = page.url();
	await logOut(page);

	// Petugas kedua mencocokkan dengan hardcopy dan mencatat menit usahanya.
	await logInFirstTime(page, 'migrasi2.e2e', second, 'hardcopy selalu jadi acuan');
	await page.goto(url);
	await baseline.getByLabel(en.migrationEffort, { exact: true }).fill('30');
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.migrationVerified.replace('{number}', number.number));
	await expect(page.locator('dl.facts .badge').first()).toHaveText(en.documentStatusReleasedMigrated);
	await expect(baseline).toContainText('-00');

	// Dasbor menghitung dokumen migrasi dan menitnya.
	await openFromMenu(page, en.navGroupDocuments, en.navDashboard);
	await expect(page.getByRole('heading', { name: en.dashboardTitle, level: 1 })).toBeVisible();
	const migration = page.locator('section', { has: page.getByRole('heading', { name: en.dashboardMigration, exact: true }) });
	await expect(migration).toContainText(en.dashboardMinutes.replace('{minutes}', '30'));
	const documents = page.locator('section', { has: page.getByRole('heading', { name: en.dashboardDocuments }) });
	await expect(documents).toContainText(en.documentStatusReleasedMigrated);
});
