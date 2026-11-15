// Test E2E D-128: tampilan untuk HP, tablet, dan laptop. Di HP dan tablet menu
// atas dilipat ke tombol Menu; di laptop kelompok menu langsung terlihat. Di
// semua ukuran halaman tidak boleh bergeser ke samping.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, mainMenu } from './helpers.ts';

const devices = [
	{ name: 'HP', width: 375, height: 812, folded: true },
	{ name: 'tablet', width: 768, height: 1024, folded: true },
	{ name: 'laptop', width: 1280, height: 800, folded: false }
];

// Halaman dengan tabel dan formulir terlebar, ditambah beranda.
const pages = ['/', '/numbers', '/format-templates', '/process-templates', '/customers', '/admin/accounts', '/admin/audit', '/help', '/form-sets', '/documents', '/referenced-documents', '/changes', '/yokoten', '/print-layouts', '/dashboard', '/admin/roles', '/admin/service-accounts', '/admin/notifications', '/documents/released', '/periodic-reviews', '/performance'];

// noSideScroll memastikan lebar isi halaman tidak melebihi lebar layar.
async function noSideScroll(page: Page, path: string) {
	const { scroll, client } = await page.evaluate(() => ({
		scroll: document.documentElement.scrollWidth,
		client: document.documentElement.clientWidth
	}));
	expect(scroll, `${path} bergeser ke samping`).toBeLessThanOrEqual(client);
}

for (const device of devices) {
	test(`tampilan ${device.name} (${device.width}px)`, async ({ page }) => {
		await page.setViewportSize({ width: device.width, height: device.height });
		await logIn(page);
		const menuButton = page.getByRole('button', { name: en.navMenu, exact: true });
		const master = mainMenu(page).getByRole('button', { name: en.navGroupMaster });

		if (device.folded) {
			// Menu terlipat: kelompok baru terlihat setelah tombol Menu ditekan.
			await expect(menuButton).toBeVisible();
			await expect(menuButton).toHaveAttribute('aria-expanded', 'false');
			await expect(master).toBeHidden();
			await menuButton.click();
			await expect(menuButton).toHaveAttribute('aria-expanded', 'true');
			await master.click();
			await mainMenu(page).getByRole('link', { name: en.navCustomers }).click();
			await expect(page).toHaveURL('/customers');
			// Pindah halaman menutup panel menu lagi.
			await expect(master).toBeHidden();
		} else {
			await expect(menuButton).toBeHidden();
			await master.click();
			await mainMenu(page).getByRole('link', { name: en.navCustomers }).click();
			await expect(page).toHaveURL('/customers');
		}

		for (const path of pages) {
			await page.goto(path);
			await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
			await noSideScroll(page, path);
		}
	});
}
