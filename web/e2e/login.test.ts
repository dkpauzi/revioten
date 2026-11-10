// Test E2E M1.1: kriteria selesai "halaman tanpa login ditolak".
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from './account.mjs';
import { logIn, logOut } from './helpers.ts';

test('halaman tanpa login diarahkan ke login, lalu kembali setelah login', async ({ page }) => {
	await page.goto('/customers');
	await expect(page).toHaveURL('/login?next=%2Fcustomers');

	await page.getByLabel(en.loginName).fill(admin.login);
	await page.getByLabel(en.loginPassword).fill(admin.password);
	await page.getByRole('button', { name: en.loginSubmit }).click();

	await expect(page).toHaveURL('/customers');
	await expect(page.getByRole('heading', { name: en.customersTitle, exact: true })).toBeVisible();
	await expect(
		page.getByText(en.loggedInAs.replace('{name}', admin.name).replace('{login}', admin.login))
	).toBeVisible();
});

test('API menolak permintaan tanpa sesi', async ({ request }) => {
	const response = await request.get('/api/v1/customers');
	expect(response.status()).toBe(401);
	expect((await response.json()).code).toBe('unauthenticated');
});

test('password salah ditolak dengan pesan umum', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill(admin.login);
	await page.getByLabel(en.loginPassword).fill('bukan password yang benar');
	await page.getByRole('button', { name: en.loginSubmit }).click();

	await expect(page.getByRole('alert')).toHaveText(en.loginFailed);
	await expect(page).toHaveURL('/login');
});

test('logout mengakhiri sesi', async ({ page }) => {
	await logIn(page);
	await logOut(page);

	// Sesi sudah dihapus di server: halaman lain kembali diarahkan ke login.
	await page.goto('/customers');
	await expect(page).toHaveURL('/login?next=%2Fcustomers');
});

test('ganti password, lalu login dengan password baru', async ({ page }) => {
	const newPassword = 'burung hantu di pohon mangga';
	await logIn(page);
	await page.getByRole('link', { name: en.navAccount }).click();

	const current = page.getByLabel(en.passwordCurrent, { exact: true });
	const next = page.getByLabel(en.passwordNew, { exact: true });
	const confirm = page.getByLabel(en.passwordConfirm, { exact: true });
	const save = page.getByRole('button', { name: en.passwordSave });

	// Aturan panjang diperiksa server (8 karakter, D-132).
	await current.fill(admin.password);
	await next.fill('pendek!');
	await confirm.fill('pendek!');
	await save.click();
	await expect(page.getByText(en.passwordTooShort)).toBeVisible();

	await next.fill(newPassword);
	await confirm.fill(newPassword);
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);

	await logOut(page);
	await logIn(page, newPassword);

	// Kembalikan password semula, supaya test lain tetap memakai password yang sama.
	await page.goto('/account/password');
	await current.fill(newPassword);
	await next.fill(admin.password);
	await confirm.fill(admin.password);
	await save.click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);
});
