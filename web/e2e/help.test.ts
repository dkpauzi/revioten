// Test E2E halaman Help (D-141): buku panduan pengguna di dalam aplikasi,
// dua bahasa, dengan nama tombol dari katalog layar dan gambar yang dimuat.
import { expect, test } from '@playwright/test';
import { helpEn } from '../src/lib/help/en.ts';
import { helpId } from '../src/lib/help/id.ts';
import { en } from '../src/lib/i18n/en.ts';
import { id } from '../src/lib/i18n/id.ts';
import { logIn } from './helpers.ts';

test('halaman Help menampilkan panduan, gambar, dan ikut berganti bahasa', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navHelp, exact: true }).click();
	await expect(page).toHaveURL('/help');
	await expect(page.getByRole('heading', { name: helpEn.title, level: 1 })).toBeVisible();

	// Setiap bab ada, dan daftar isi menuju babnya.
	for (const [n, chapter] of helpEn.chapters.entries()) {
		await expect(page.getByRole('heading', { name: `${n + 1}. ${chapter.title}`, exact: true })).toBeVisible();
	}
	await page.getByRole('link', { name: helpEn.chapters[3].title, exact: true }).click(); // Accounts and roles
	await expect(page).toHaveURL('/help#accounts');
	await expect(page.locator('#accounts')).toBeInViewport();

	// Nama tombol di langkah diambil dari katalog layar.
	await expect(page.locator('#accounts')).toContainText(en.accountCreate);
	await expect(page.locator('#accounts strong', { hasText: en.accountCopyPassword })).toBeVisible();

	// Semua gambar benar-benar dimuat (file ada di static/help/en).
	const images = page.locator('figure img');
	await expect(images).toHaveCount(helpEn.chapters.filter((c) => c.image).length);
	for (const image of await images.all()) {
		await image.scrollIntoViewIfNeeded();
		await expect.poll(() => image.evaluate((el: HTMLImageElement) => el.complete && el.naturalWidth > 0)).toBe(true);
	}

	// Bahasa Indonesia: judul, nama tombol, dan alamat gambar ikut berganti.
	await page.getByRole('group', { name: en.languageLabel }).getByRole('button', { name: 'ID' }).click();
	await expect(page.getByRole('heading', { name: helpId.title, level: 1 })).toBeVisible();
	await expect(page.locator('#accounts')).toContainText(id.accountCreate);
	await expect(images.first()).toHaveAttribute('src', '/help/id/login.png');
	await page.getByRole('group', { name: id.languageLabel }).getByRole('button', { name: 'EN' }).click();
});
