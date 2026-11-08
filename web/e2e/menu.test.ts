// Test E2E D-127 (bagian dari UX-05): menu atas berkelompok. Kelompoknya Home,
// Documents, Template library, Master data, dan Admin; kelompok Admin hanya
// tampil bagi pemegang hak admin (dibuktikan juga di accounts.test.ts).
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, mainMenu } from './helpers.ts';

test('menu atas berkelompok dibuka dengan klik dan keyboard', async ({ page }) => {
	await logIn(page);
	const menu = mainMenu(page);

	for (const group of [en.navGroupDocuments, en.navGroupTemplates, en.navGroupMaster, en.navGroupAdmin]) {
		await expect(menu.getByRole('button', { name: group, exact: true })).toBeVisible();
	}
	await expect(menu.getByRole('link', { name: en.navHome, exact: true })).toBeVisible();

	// Isi kelompok baru tampil setelah kelompoknya dibuka.
	const templates = menu.getByRole('button', { name: en.navGroupTemplates });
	await expect(menu.getByRole('link', { name: en.navFormatTemplates })).toHaveCount(0);
	await templates.click();
	await expect(templates).toHaveAttribute('aria-expanded', 'true');
	await menu.getByRole('link', { name: en.navFormatTemplates }).click();
	await expect(page).toHaveURL('/format-templates');

	// Setelah pindah halaman, menu tertutup dan kelompok halaman aktif ditandai.
	await expect(menu.getByRole('link', { name: en.navFormatTemplates })).toHaveCount(0);
	await expect(templates).toHaveAttribute('data-active', 'true');

	// Escape menutup kelompok yang terbuka dan mengembalikan fokus ke tombolnya.
	const master = menu.getByRole('button', { name: en.navGroupMaster });
	await master.focus();
	await page.keyboard.press('Enter');
	await expect(menu.getByRole('link', { name: en.navCustomers })).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(menu.getByRole('link', { name: en.navCustomers })).toHaveCount(0);
	await expect(master).toBeFocused();

	// Klik di luar menu juga menutupnya.
	await master.click();
	await page.getByRole('heading', { level: 1 }).click();
	await expect(menu.getByRole('link', { name: en.navCustomers })).toHaveCount(0);
});
