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
