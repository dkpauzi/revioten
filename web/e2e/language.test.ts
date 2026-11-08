// Test E2E D-128: layar dua bahasa. Bahasa Inggris bawaan, tombol EN/ID ada di
// halaman login dan di bar atas, dan pilihannya tetap setelah halaman dimuat ulang.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { id } from '../src/lib/i18n/id.ts';
import { logIn, mainMenu } from './helpers.ts';

test('tombol EN/ID mengganti bahasa layar dan pilihannya disimpan', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('heading', { name: en.loginTitle })).toBeVisible();
	await expect(page.locator('html')).toHaveAttribute('lang', 'en');

	const switcher = page.getByRole('group', { name: en.languageLabel });
	await switcher.getByRole('button', { name: 'ID' }).click();
	await expect(page.getByRole('heading', { name: id.loginTitle })).toBeVisible();
	await expect(page.locator('html')).toHaveAttribute('lang', 'id');
	await expect(page.getByRole('group', { name: id.languageLabel }).getByRole('button', { name: 'ID' })).toHaveAttribute(
		'aria-pressed',
		'true'
	);

	// Pilihan disimpan di browser, jadi tetap setelah dimuat ulang.
	await page.reload();
	await expect(page.getByRole('heading', { name: id.loginTitle })).toBeVisible();

	// Kembali ke bahasa Inggris untuk login, karena helper logIn memakai label Inggris.
	await page.getByRole('group', { name: id.languageLabel }).getByRole('button', { name: 'EN' }).click();
	await logIn(page);

	// Setelah login, tombol yang sama ada di bar atas dan menu ikut berganti bahasa.
	await page.getByRole('group', { name: en.languageLabel }).getByRole('button', { name: 'ID' }).click();
	// Nama menu utama juga diterjemahkan, jadi dicari dengan label Indonesia.
	const menuId = page.getByRole('navigation', { name: id.navMain });
	await expect(menuId.getByRole('button', { name: id.navGroupDocuments })).toBeVisible();
	await expect(menuId.getByRole('link', { name: id.navHome })).toBeVisible();
	await expect(page.getByRole('button', { name: id.logout })).toBeVisible();
	await page.getByRole('group', { name: id.languageLabel }).getByRole('button', { name: 'EN' }).click();
	await expect(mainMenu(page).getByRole('button', { name: en.navGroupMaster })).toBeVisible();
});
