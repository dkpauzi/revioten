// Test E2E form set langkah D (D-133, D-139): admin menyusun form set dan
// tabel AP, penyetuju form (V-3) menerbitkannya, dan admin memilih form set
// untuk setiap customer. Aturan "penyetuju bukan penyusun" dibuktikan test Go
// TestSeparationOfDuties.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// acceptNextDialog menyetujui kotak konfirmasi berikutnya.
function acceptNextDialog(page: Page): void {
	page.once('dialog', (dialog) => dialog.accept());
}

// publishFormat menerbitkan versi terakhir satu format template lewat API,
// sebagai akun yang sedang login.
async function publishFormat(page: Page, name: string): Promise<void> {
	const status = await page.evaluate(async (name) => {
		const list = await (await fetch('/api/v1/format-templates')).json();
		const ft = list.items.find((x: { name: string }) => x.name === name);
		const last = ft.versions[ft.versions.length - 1];
		if (last.status === 'published') {
			return 200;
		}
		return (await fetch(`/api/v1/format-template-versions/${last.id}/publish`, { method: 'POST' })).status;
	}, name);
	expect(status, name).toBe(200);
}

test('admin menyusun, penyetuju form menerbitkan, lalu form set dipilih per customer', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		await fetch('/api/v1/customers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ code: 'FS', name: 'Customer FS', project_group_id: group.id })
		});
	});
	const temporary = await createWorkAccount(page, 'PPAP Supervisor E2E', 'ppap.supervisor.e2e', ['form_approver']);

	// Ketiga format set AIAG 4th diterbitkan admin dulu; form set baru bisa
	// terbit setelah isinya tidak bisa berubah lagi.
	for (const name of ['PFD', 'PFMEA AIAG 4th', 'CP']) {
		await publishFormat(page, name);
	}

	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await expect(page.getByRole('heading', { name: en.formSetsTitle, level: 1 })).toBeVisible();
	const oldSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	await expect(oldSet.getByRole('heading', { name: /^AIAG 4th set/ })).toContainText(en.formSetDefault);
	// Admin menyusun, tetapi tidak menerbitkan form set (V-3).
	await expect(oldSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') })).toHaveCount(0);
	await expect(oldSet.getByRole('button', { name: en.formSetSaveSheets })).toBeVisible();

	// Form set customer: FS memakai AIAG-VDA set.
	await page.getByLabel(en.formSetCustomerChoose.replace('{code}', 'FS')).selectOption({ label: 'AIAG-VDA set' });
	await expect(page.getByRole('status')).toHaveText(en.formSetCustomerSaved.replace('{code}', 'FS'));
	await expect(page.getByLabel(en.formSetCustomerChoose.replace('{code}', 'FS'))).toHaveValue(/.+/);

	// Penyetuju form login dan menerbitkan set AIAG 4th.
	await logOut(page);
	await logInFirstTime(page, 'ppap.supervisor.e2e', temporary, 'tanda tangan basah tetap di kertas');
	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	const approverView = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	await expect(approverView.getByRole('button', { name: en.formSetSaveSheets })).toHaveCount(0);
	acceptNextDialog(page);
	await approverView.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.formSetPublished.replace('{version}', '1').replace('{name}', 'AIAG 4th set'));
	await expect(approverView.getByRole('row', { name: new RegExp(`^${en.formSetSheetPfmea} PFMEA AIAG 4th`) })).toContainText(
		en.versionStatusPublished
	);
	// Penyetuju form hanya melihat pilihan customer.
	await expect(page.getByRole('row', { name: /^FS · Customer FS/ })).toContainText('AIAG-VDA set');

	// Set AIAG-VDA belum bisa terbit: format di dalamnya masih draft.
	const vdaSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG-VDA set/ }) });
	acceptNextDialog(page);
	await vdaSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('alert').first()).toHaveText(en.formSetSheetsNotPublished);
});

test('admin mengubah tabel AP, dan hanya penyetuju form yang menerbitkannya', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Penyetuju AP E2E', 'penyetuju.ap.e2e', ['form_approver']);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatName, { exact: true }).fill('PFMEA AP ubah');
	await page.getByLabel(en.formatDocumentType, { exact: true }).selectOption({ index: 2 });
	await page.getByLabel(en.formatRiskMethod, { exact: true }).selectOption({ label: en.formatRiskAp });
	await page.getByRole('button', { name: en.formatCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatCreated.replace('{name}', 'PFMEA AP ubah'));

	// Kotak S 9–10, O 1, D 7–10 dari L menjadi H: klik dua kali (L → H).
	await page.getByRole('button', { name: en.apEdit }).click();
	const cell = page.getByRole('button', { name: en.apCellLabel.replace('{s}', '9–10').replace('{o}', '1').replace('{d}', '7–10').replace('{ap}', 'L') });
	await cell.click();
	await expect(page.getByRole('button', { name: en.apCellLabel.replace('{s}', '9–10').replace('{o}', '1').replace('{d}', '7–10').replace('{ap}', 'H') })).toBeVisible();
	await page.getByRole('button', { name: en.apSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.apSaved);
	await expect(page.getByRole('row', { name: /^9–10 1 H L L L$/ })).toBeVisible();
	// Admin tidak mendapat tombol terbitkan untuk versi bermetode AP.
	await expect(page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') })).toHaveCount(0);

	await logOut(page);
	await logInFirstTime(page, 'penyetuju.ap.e2e', temporary, 'tabel AP diperiksa orang kedua');
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	const choose = page.getByLabel(en.formatChoose);
	const value = await choose.locator('option', { hasText: 'PFMEA AP ubah' }).getAttribute('value');
	await choose.selectOption(value ?? '');
	acceptNextDialog(page);
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	// Penyetuju form tidak mengubah tabel AP.
	await expect(page.getByRole('button', { name: en.apEdit })).toHaveCount(0);
});
