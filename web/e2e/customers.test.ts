// Test E2E M0.5: kriteria selesai "customer yang ditambah di layar tersimpan".
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn } from './helpers.ts';

// Sejak M1.1 semua halaman butuh login.
test.beforeEach(async ({ page }) => {
	await logIn(page);
});

test('customer yang ditambah di layar tersimpan di database', async ({ page }) => {
	await page.goto('/customers');

	await page.getByLabel(en.customerCode).fill(' ab ');
	await page.getByLabel(en.customerName).fill('Example Customer');
	await page.getByRole('button', { name: en.customerSave }).click();

	// Server merapikan kode " ab " menjadi AB.
	await expect(page.getByRole('status')).toHaveText(en.customerSaved.replace('{code}', 'AB'));
	await expect(page.getByRole('row', { name: /AB Example Customer/ })).toBeVisible();

	// Muat ulang halaman: daftar datang lagi dari database, bukan dari memori browser.
	await page.reload();
	await expect(page.getByRole('row', { name: /AB Example Customer/ })).toBeVisible();
});

test('kode yang salah atau sudah dipakai ditolak dengan pesan per field', async ({ page }) => {
	await page.goto('/customers');
	const code = page.getByLabel(en.customerCode);
	const name = page.getByLabel(en.customerName);
	const save = page.getByRole('button', { name: en.customerSave });

	// ZZ dicadangkan untuk dokumen umum, dan nama kosong ditolak.
	await code.fill('zz');
	await save.click();
	await expect(page.getByText(en.customerCodeReserved)).toBeVisible();
	await expect(page.getByText(en.customerNameRequired)).toBeVisible();

	// Kode yang sudah dipakai ditolak, juga bila ditulis dengan huruf kecil.
	await code.fill('CD');
	await name.fill('Customer D');
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.customerSaved.replace('{code}', 'CD'));

	await code.fill('cd');
	await name.fill('Customer D lagi');
	await save.click();
	await expect(page.getByText(en.customerCodeTaken)).toBeVisible();
	await expect(page.getByRole('row', { name: /CD Customer D lagi/ })).toHaveCount(0);
});
