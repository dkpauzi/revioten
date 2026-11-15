// Test E2E D-194: alamat email sendiri, halaman Email notifications, email uji,
// dan kotak keluar. Server E2E memakai MAIL_PROVIDER=log, jadi email melewati
// jalur kirim tanpa server email sungguhan.
// Berjalan sesudah m5-changes.test.ts, yang sudah mengajukan perubahan.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu } from './helpers.ts';

test('D-194: admin mengisi email, mengirim email uji, dan melihatnya terkirim di kotak keluar', async ({ page }) => {
	test.setTimeout(90_000);
	await logIn(page);
	// Alamat email sendiri di halaman akun; alamat salah ditolak.
	await page.goto('/account/password');
	const email = page.getByLabel(en.accountEmail, { exact: true });
	await email.fill('bukan email');
	await page.getByRole('button', { name: en.accountEmailSave }).click();
	await expect(page.getByRole('alert')).toHaveText(en.accountEmailInvalid);
	await email.fill('admin.e2e@contoh.test');
	await page.getByRole('button', { name: en.accountEmailSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.accountEmailSaved);

	await openFromMenu(page, en.navGroupAdmin, en.navNotifications);
	await expect(page.getByRole('heading', { name: en.notifyTitle, level: 1 })).toBeVisible();
	await expect(page.getByText(en.notifyProvider_log)).toBeVisible();
	// Perubahan dari test sebelumnya sudah menulis email; penerimanya belum punya alamat.
	await expect(page.locator('[data-outbox-status="skipped"]').first()).toContainText(en.notifyReason_no_email);

	await page.getByRole('button', { name: en.notifyTest }).click();
	await expect(page.getByRole('status')).toHaveText(en.notifyTestQueued);
	// Pekerja mengirim kotak keluar setiap 15 detik.
	await expect(async () => {
		await page.reload();
		const row = page.getByRole('row', { name: /Test email from the PFD/ }).first();
		await expect(row.locator('[data-outbox-status]')).toHaveAttribute('data-outbox-status', 'sent');
	}).toPass({ timeout: 40_000 });

	// Saklar per peristiwa.
	const submitted = page.getByRole('checkbox', { name: en.notifyEvent_change_submitted });
	await submitted.uncheck();
	await expect(page.getByRole('status')).toHaveText(en.notifyEventOff.replace('{event}', en.notifyEvent_change_submitted));
	await expect(submitted).not.toBeChecked();
	await submitted.check();
	await expect(submitted).toBeChecked();

	// Admin juga bisa mengisi email akun lain di halaman Accounts.
	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	page.once('dialog', (dialog) => dialog.accept('pemohon.e2e@contoh.test'));
	await page.getByRole('row', { name: /pemohon\.e2e/ }).getByRole('button', { name: en.accountEmail, exact: true }).click();
	await expect(page.getByRole('row', { name: /pemohon\.e2e/ })).toContainText('pemohon.e2e@contoh.test');
});
