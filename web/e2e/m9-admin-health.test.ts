// Test E2E fase 4: dasbor admin (D-205). Pengguna melaporkan masalah dari
// bilah atas, admin melihatnya di Home dan di Admin → Web health, lalu
// menandainya selesai.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('fase 4: pengguna melaporkan masalah, admin menanganinya di dasbor kesehatan web', async ({ page }) => {
	await logIn(page);
	// Home akun admin adalah dasbor kesehatan web.
	await expect(page.getByRole('heading', { name: en.adminHomeTitle, level: 1 })).toBeVisible();
	await expect(page.getByText(en.adminAudit)).toBeVisible();
	const temporary = await createWorkAccount(page, 'Pelapor Masalah', 'pelapor.e2e', []);
	await logOut(page);

	await logInFirstTime(page, 'pelapor.e2e', temporary, 'laporan masalah yang jelas');
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await page.getByRole('button', { name: en.reportProblem }).click();
	const dialog = page.getByRole('dialog', { name: en.reportProblemTitle });
	await dialog.getByLabel(en.reportProblemText).fill('Daftar nomor tidak bisa disaring per customer');
	await dialog.getByRole('button', { name: en.reportProblemSend }).click();
	await expect(dialog).toBeHidden();
	await logOut(page);

	await logIn(page);
	await expect(page.getByRole('heading', { name: en.adminLatestProblems })).toBeVisible();
	await expect(page.getByText('Daftar nomor tidak bisa disaring per customer')).toBeVisible();
	await openFromMenu(page, en.navGroupAdmin, en.navAdminHealth);
	await page.getByRole('button', { name: en.adminHealthTabProblems.replace('{count}', '1') }).click();
	const row = page.getByRole('row', { name: /tidak bisa disaring/ });
	await expect(row).toContainText('/numbers');
	await row.getByRole('combobox').selectOption('done');
	await row.getByRole('textbox').fill('Saringan customer sudah ada di atas daftar');
	await row.getByRole('button', { name: en.adminSave }).click();
	await expect(page.getByRole('status').first()).toHaveText(en.adminSaved);
	await expect(page.getByRole('button', { name: en.adminHealthTabProblems.replace('{count}', '0') })).toBeVisible();
});
