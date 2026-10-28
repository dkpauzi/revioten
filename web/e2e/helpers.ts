// Langkah bersama untuk test E2E.
import { expect, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from './account.mjs';

// logIn membuka halaman login dan masuk sebagai admin uji. Bila password
// diberikan, password itu yang dipakai (misalnya setelah password diganti).
export async function logIn(page: Page, password: string = admin.password): Promise<void> {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill(admin.login);
	await page.getByLabel(en.loginPassword).fill(password);
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
}

// ensureProjectGroup memastikan Project Group dengan kode itu ada, lewat API
// dari browser yang sudah login sebagai admin (M1.4). Customer baru wajib
// dialokasikan ke satu Project Group (D-54).
export async function ensureProjectGroup(page: Page, code: string): Promise<void> {
	await page.evaluate(async (code) => {
		const list = await (await fetch('/api/v1/project-groups')).json();
		if (!list.items.some((g: { code: string }) => g.code === code)) {
			await fetch('/api/v1/project-groups', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ code, name: `Project Group ${code}` })
			});
		}
	}, code);
}

// createWorkAccount membuat akun kerja untuk orang baru lewat API, dari
// browser yang sudah login sebagai admin, lalu mengembalikan password
// sementaranya (M1.2). roles berisi ID peran, misalnya 'template_maintainer'.
export async function createWorkAccount(
	page: Page,
	personName: string,
	login: string,
	roles: string[]
): Promise<string> {
	return page.evaluate(
		async ({ personName, login, roles }) => {
			const response = await fetch('/api/v1/accounts', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					person_name: personName,
					kind: 'work',
					login,
					roles: roles.map((role) => ({ role, scope: 'all' }))
				})
			});
			return (await response.json()).temporary_password as string;
		},
		{ personName, login, roles }
	);
}

// logInFirstTime masuk dengan password sementara akun baru dan langsung
// menggantinya, karena akun baru wajib ganti password dulu (E-08).
export async function logInFirstTime(
	page: Page,
	login: string,
	temporary: string,
	newPassword: string
): Promise<void> {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill(login);
	await page.getByLabel(en.loginPassword).fill(temporary);
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page).toHaveURL('/account/password');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(newPassword);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(newPassword);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);
}
