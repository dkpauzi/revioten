// Test E2E M8.1: admin mengubah isi peran di halaman Roles (D-66), dan aturan
// yang tidak bisa dimatikan tampak sebagai kotak terkunci (R-SEC-11).
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu } from './helpers.ts';

test('M8.1: admin memberi hak E-2 ke peran PFMEA–CP checker, lalu mencabutnya lagi', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);
	await expect(page.getByRole('heading', { name: en.rolesTitle, level: 1 })).toBeVisible();

	// Terkunci: peran Admin dan Viewer, hak admin di peran kerja, dan hak dasar.
	await expect(page.getByLabel(`${en.roleAdmin}: A-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.roleViewer}: L-4`)).toBeDisabled();
	await expect(page.getByLabel(`${en.roleAuthor}: A-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.roleAuthor}: L-1`)).toBeDisabled();
	await expect(page.getByLabel(`${en.roleAuthor}: L-1`)).toBeChecked();

	const box = page.getByLabel(`${en.rolePfmeaCpChecker}: E-2`);
	const save = page.getByRole('button', { name: en.rolesSaveFor.replace('{role}', en.rolePfmeaCpChecker) });
	await expect(box).not.toBeChecked();
	await expect(save).toBeDisabled();
	await box.check();
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.rolesSaved.replace('{role}', en.rolePfmeaCpChecker));
	await page.reload();
	await expect(box).toBeChecked();

	// Dikembalikan, supaya test lain tidak terpengaruh.
	await box.uncheck();
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.rolesSaved.replace('{role}', en.rolePfmeaCpChecker));

	// Perubahan tercatat di audit log.
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await expect(page.getByText(en.auditActionRolePermissionsChanged).first()).toBeVisible();
});
