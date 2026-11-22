// Test E2E form set langkah D (D-133, D-139): PPAP Engineer menyusun form set
// dan tabel AP (E-11, D-209), penyetuju form (V-3) menerbitkannya, dan PPAP
// Engineer memilih form set untuk setiap customer. Aturan "penyetuju bukan penyusun" dibuktikan test Go
// TestSeparationOfDuties.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInAsMaster, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

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

test('PPAP Engineer menyusun, penyetuju form menerbitkan, lalu form set dipilih per customer', async ({ page }) => {
	// Admin membuat akun penyetuju; master data dan form set diurus PPAP Engineer (D-209).
	await logIn(page);
	const temporary = await createWorkAccount(page, 'PPAP Supervisor E2E', 'ppap.supervisor.e2e', ['ppap_engineer_supervisor']);
	await logOut(page);
	await logInAsMaster(page);
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

	// Ketiga format set AIAG 4th diterbitkan PPAP Engineer dulu; form set baru bisa
	// terbit setelah isinya tidak bisa berubah lagi.
	for (const name of ['PFD v1', 'PFMEA v1', 'Control Plan v1']) {
		await publishFormat(page, name);
	}

	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await expect(page.getByRole('heading', { name: en.formSetsTitle, level: 1 })).toBeVisible();
	const oldSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	await expect(oldSet.getByRole('heading', { name: /^AIAG 4th set/ })).toContainText(en.formSetDefault);
	// PPAP Engineer menyusun, tetapi tidak menerbitkan form set (V-3).
	await expect(oldSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') })).toHaveCount(0);
	await expect(oldSet.getByRole('button', { name: en.formSetSaveSheets })).toBeVisible();

	// D-218: set AIAG-VDA tersembunyi, jadi tidak bisa dipilih customer. Set
	// bawaan tidak punya tombol sembunyikan.
	const customerChoice = page.getByLabel(en.formSetCustomerChoose.replace('{code}', 'FS'));
	await expect(customerChoice.locator('option', { hasText: 'AIAG-VDA set' })).toHaveCount(0);
	await expect(oldSet.getByRole('button', { name: en.formSetHideFor.replace('{name}', 'AIAG 4th set') })).toHaveCount(0);
	await page.getByRole('checkbox', { name: en.showHidden }).check();
	const hiddenSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG-VDA set/ }) });
	await expect(hiddenSet.getByRole('heading')).toContainText(en.hiddenMark);
	await hiddenSet.getByRole('button', { name: en.formSetShowFor.replace('{name}', 'AIAG-VDA set') }).click();
	await expect(page.getByRole('status')).toHaveText(en.formSetShown.replace('{name}', 'AIAG-VDA set'));

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
	await expect(approverView.getByRole('row', { name: new RegExp(`^${en.formSetSheetPfmea} PFMEA v1`) })).toContainText(
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

test('PPAP Engineer mengubah tabel AP, dan hanya penyetuju form yang menerbitkannya', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Penyetuju AP E2E', 'penyetuju.ap.e2e', ['ppap_engineer_supervisor']);
	await logOut(page);
	await logInAsMaster(page);
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
	// PPAP Engineer tanpa V-3 tidak mendapat tombol terbitkan untuk versi bermetode AP.
	await expect(page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') })).toHaveCount(0);

	await logOut(page);
	await logInFirstTime(page, 'penyetuju.ap.e2e', temporary, 'tabel AP diperiksa orang kedua');
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	const choose = page.getByLabel(en.formatChoose);
	const value = await choose.locator('option', { hasText: 'PFMEA AP ubah' }).getAttribute('value');
	await choose.selectOption(value ?? '');
	// Aturan bawaan (D-139): semua penyetuju form harus setuju. Penyetuju dari
	// test sebelumnya (PPAP Supervisor E2E) masih ditunggu.
	acceptNextDialog(page);
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.approvalRecorded);
	await expect(page.getByText(en.approvalsPending)).toBeVisible();
	await expect(page.locator('.approvals')).toContainText('PPAP Supervisor E2E');
	// Supervisor juga memegang E-11 sejak D-209, jadi boleh mengubah tabel AP.
	// Pemisahan tugas dijaga saat terbit: yang terakhir mengubah tidak menerbitkan (D-139).
	await expect(page.getByRole('button', { name: en.apEdit })).toBeVisible();

	// Penyetuju kedua menyetujui: sekarang versi itu terbit.
	await logOut(page);
	await page.getByLabel(en.loginName).fill('ppap.supervisor.e2e');
	await page.getByLabel(en.loginPassword).fill('tanda tangan basah tetap di kertas');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	const again = page.getByLabel(en.formatChoose);
	await again.selectOption((await again.locator('option', { hasText: 'PFMEA AP ubah' }).getAttribute('value')) ?? '');
	acceptNextDialog(page);
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	await expect(page.getByText(en.approvalsPending)).toHaveCount(0);
});

test('admin melihat dan mengubah aturan persetujuan form di Settings', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	const all = page.getByLabel(en.settingsFormApprovalAll);
	await expect(all).toBeChecked();
	await page.getByLabel(en.settingsFormApprovalAny).check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
	await page.reload();
	await expect(page.getByLabel(en.settingsFormApprovalAny)).toBeChecked();
	// Kembalikan ke bawaan, supaya test lain tidak terpengaruh.
	await all.check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
});
