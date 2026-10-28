// Test E2E M1.3: admin membaca audit log dan memeriksa rantai hash-nya. Setiap
// perubahan data, termasuk dari test lain, tercatat bersama pelakunya.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from './account.mjs';
import { logIn } from './helpers.ts';

test('admin melihat audit log, dan rantainya utuh', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navAuditLog }).click();
	await expect(page.getByRole('heading', { name: en.auditTitle, exact: true })).toBeVisible();

	// Akun admin pertama dibuat perintah create-admin, jadi pelakunya sistem (E-07).
	const created = page.getByRole('row').filter({ hasText: en.auditActorSystem });
	await expect(created).toContainText(en.auditActionAccountCreated);
	await expect(created).toContainText(admin.login);

	// Login barusan tercatat dengan akun dan orang pemegangnya (D-74).
	const actor = en.auditActorHuman.replace('{name}', admin.name).replace('{login}', admin.login);
	const logins = page.getByRole('row').filter({ hasText: en.auditActionLogin }).filter({ hasText: actor });
	await expect(logins.first()).toBeVisible();

	await page.getByRole('button', { name: en.auditVerify }).click();
	await expect(page.getByRole('status')).toContainText(en.auditChainValid.split('{entries}')[0]);
});
