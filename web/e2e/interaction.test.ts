// Test E2E rencana E (docs/rencana/import-tur-dan-logo.md): tombol, kotak
// isian, centang, dan baris tabel menunjukkan hover, focus, dan pilihan,
// supaya bagian yang sedang dikerjakan mudah dilihat.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu } from './helpers.ts';

// Warna aksen (--accent) dan latar aksen tipis (--accent-weak) di app.css.
const accent = 'rgb(67, 56, 202)';
const accentWeak = 'rgb(238, 240, 255)';

test('tombol, kotak isian, centang, dan baris tabel menunjukkan keadaannya', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);

	// Focus tombol juga saat diklik mouse.
	const save = page.getByRole('button', { name: en.settingsSave });
	await save.focus();
	await expect(save).toHaveCSS('outline-style', 'solid');
	await expect(save).toHaveCSS('outline-color', accent);

	// Hover kotak isian: garis tepi berwarna aksen.
	const name = page.getByLabel(en.companyName, { exact: true });
	await name.hover();
	await expect(name).toHaveCSS('border-top-color', accent);

	// Hover label centang: latar aksen tipis.
	const migration = page.locator('label', { has: page.getByLabel(en.settingsMigration, { exact: true }) });
	await migration.hover();
	await expect(migration).toHaveCSS('background-color', accentWeak);

	// Hover baris tabel data.
	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	const cell = page.locator('table:not([class]) tbody tr').first().locator('td').first();
	await cell.hover();
	await expect(cell).toHaveCSS('background-color', accentWeak);
});
