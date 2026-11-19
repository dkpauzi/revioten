// Test E2E M8.1 dan D-199: admin mengubah isi peran di halaman Roles (D-66),
// menambah, mengganti nama, dan menonaktifkan peran, dan aturan yang tidak
// bisa dimatikan tampak sebagai kotak terkunci (R-SEC-11).
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import type { Page } from '@playwright/test';
import { logIn, openFromMenu } from './helpers.ts';

// acceptNextDialog menyetujui kotak konfirmasi berikutnya dan mengembalikan teksnya.
function acceptNextDialog(page: Page): Promise<string> {
	return new Promise((resolve) => {
		page.once('dialog', async (dialog) => {
			const message = dialog.message();
			await dialog.accept();
			resolve(message);
		});
	});
}

test('M8.1: admin memberi hak E-2 ke peran Manager, lalu mencabutnya lagi', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);
	await expect(page.getByRole('heading', { name: en.rolesTitle, level: 1 })).toBeVisible();

	// Terkunci: peran Admin dan Viewer, hak admin di peran kerja, dan hak dasar.
	await expect(page.getByLabel(`${en.roleAdmin}: A-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.roleViewer}: L-4`)).toBeDisabled();
	await expect(page.getByLabel(`${en.rolePpapEngineer}: A-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.rolePpapEngineer}: L-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.rolePpapEngineer}: L-1`)).toBeChecked();
	// Isi awal D-199: Manager menandatangani dan melihat laporan kinerja.
	await expect(page.getByLabel(`${en.roleManager}: S-1`)).toBeChecked();
	await expect(page.getByLabel(`${en.roleManager}: L-5`)).toBeChecked();
	// Peran lama nonaktif tersembunyi sampai dicentang.
	await expect(page.getByLabel(`${en.roleAuthor}: L-1`)).toHaveCount(0);

	const box = page.getByLabel(`${en.roleManager}: E-2`);
	const save = page.getByRole('button', { name: en.rolesSaveFor.replace('{role}', en.roleManager) });
	await expect(box).not.toBeChecked();
	await expect(save).toBeDisabled();
	await box.check();
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.rolesSaved.replace('{role}', en.roleManager));
	await page.reload();
	await expect(box).toBeChecked();

	// Dikembalikan, supaya test lain tidak terpengaruh.
	await box.uncheck();
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.rolesSaved.replace('{role}', en.roleManager));

	// Perubahan tercatat di audit log.
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await expect(page.getByText(en.auditActionRolePermissionsChanged).first()).toBeVisible();
});

test('D-199: admin menambah peran, mengganti namanya, lalu menonaktifkannya', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);

	// Tambah peran dengan hak salinan dari Viewer.
	const add = page.locator('form').filter({ has: page.getByRole('heading', { name: en.rolesAddTitle }) });
	await add.getByLabel(en.rolesNameEn).fill('QA Auditor E2E');
	await add.getByLabel(en.rolesNameId).fill('Auditor QA E2E');
	await add.getByLabel(en.rolesCopyFrom).selectOption({ label: en.roleViewer });
	await add.getByRole('button', { name: en.rolesAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.rolesAdded.replace('{role}', 'QA Auditor E2E'));
	await expect(page.getByLabel('QA Auditor E2E: G-1')).toBeChecked();
	await expect(page.getByLabel('QA Auditor E2E: L-4')).not.toBeChecked();

	// Ganti nama; isi hak tetap.
	await page.getByRole('button', { name: en.rolesRenameFor.replace('{role}', 'QA Auditor E2E') }).click();
	await page.locator('#rename-en').fill('Internal auditor E2E');
	await page.locator('#rename-id').fill('Auditor internal E2E');
	await page.getByRole('button', { name: en.rolesRenameSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.rolesRenamed.replace('{role}', 'Internal auditor E2E'));
	await expect(page.getByLabel('Internal auditor E2E: G-1')).toBeChecked();

	// Peran Viewer dan Admin tetap: tanpa tombol nonaktif.
	await expect(page.getByRole('button', { name: en.rolesDeactivateFor.replace('{role}', en.roleViewer) })).toHaveCount(0);

	// Nonaktifkan: hilang dari matriks sampai peran nonaktif ditampilkan.
	const dialog = acceptNextDialog(page);
	await page.getByRole('button', { name: en.rolesDeactivateFor.replace('{role}', 'Internal auditor E2E') }).click();
	expect(await dialog).toBe(en.rolesDeactivateConfirm.replace('{role}', 'Internal auditor E2E'));
	await expect(page.getByRole('status')).toHaveText(en.rolesDeactivated.replace('{role}', 'Internal auditor E2E'));
	await expect(page.getByLabel('Internal auditor E2E: G-1')).toHaveCount(0);
	await page.getByLabel(en.rolesShowInactive, { exact: false }).check();
	await expect(page.getByLabel('Internal auditor E2E: G-1')).toBeVisible();

	// Peran nonaktif tidak ditawarkan di halaman Accounts.
	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	await expect(page.locator('form').first().getByLabel('Internal auditor E2E', { exact: true })).toHaveCount(0);
	await expect(page.locator('form').first().getByLabel(en.rolePpapEngineer, { exact: true })).toBeVisible();

	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await expect(page.getByText(en.auditActionRoleCreated).first()).toBeVisible();
	await expect(page.getByText(en.auditActionRoleDeactivated).first()).toBeVisible();
});
