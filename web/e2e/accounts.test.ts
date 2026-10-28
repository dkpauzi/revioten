// Test E2E M1.2: admin mengelola akun lewat layar, dan setiap akun hanya
// mendapat hak dari perannya. Server tetap menolak aksi yang tidak diizinkan,
// juga bila layar dilewati.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn } from './helpers.ts';

// createWorkAccount mengisi formulir akun kerja untuk orang baru, lalu
// mengembalikan password sementara yang ditampilkan satu kali.
async function createWorkAccount(
	page: Page,
	personName: string,
	login: string,
	roles: string[]
): Promise<string> {
	const form = page.locator('form');
	await form.getByLabel(en.accountPersonNew).check();
	await form.getByLabel(en.accountPersonName).fill(personName);
	await form.getByLabel(en.accountKindWork).check();
	await form.getByLabel(en.accountLogin).fill(login);
	for (const role of roles) {
		await form.getByLabel(role, { exact: true }).check();
	}
	await form.getByRole('button', { name: en.accountCreate }).click();
	return temporaryPasswordFor(page, login);
}

// temporaryPasswordFor membaca password sementara dari pemberitahuan di layar.
async function temporaryPasswordFor(page: Page, login: string): Promise<string> {
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', login));
	const password = (await notice.locator('code').textContent()) ?? '';
	expect(password.length).toBeGreaterThan(0);
	return password;
}

// acceptNextDialog menyetujui kotak konfirmasi berikutnya dan mengembalikan teksnya.
function acceptNextDialog(page: Page): Promise<string> {
	return new Promise((resolve) => {
		page.once('dialog', async (dialog) => {
			const message = dialog.message();
			await dialog.accept();
			resolve(message);
		});
	});
}

test('akun baru wajib ganti password, lalu hanya mendapat hak dari perannya', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navAccounts }).click();
	await expect(page.getByRole('heading', { name: en.accountsTitle, exact: true })).toBeVisible();

	// Penyusun yang dibatasi ke Project Group sendiri (D-71).
	const form = page.locator('form');
	await form.getByLabel(en.accountPersonName).fill('Author E2E');
	await form.getByLabel(en.accountLogin).fill('author.e2e');
	await form.getByLabel(en.roleAuthor, { exact: true }).check();
	await form
		.getByLabel(en.accountRoleScope.replace('{role}', en.roleAuthor))
		.selectOption('own_project_group');
	await form.getByRole('button', { name: en.accountCreate }).click();
	const temporary = await temporaryPasswordFor(page, 'author.e2e');

	const row = page.getByRole('row', { name: /author\.e2e/ });
	await expect(row).toContainText(`${en.roleAuthor} (${en.scopeOwnProjectGroup})`);
	await expect(row).toContainText(en.accountStatusMustChange);

	// Pemilik akun login dengan password sementara dan wajib menggantinya dulu.
	await page.getByRole('button', { name: en.logout }).click();
	await page.getByLabel(en.loginName).fill('author.e2e');
	await page.getByLabel(en.loginPassword).fill(temporary);
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page).toHaveURL('/account/password');
	await expect(page.getByRole('alert')).toHaveText(en.passwordMustChange);

	const newPassword = 'lampu jalan menyala sore hari';
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(newPassword);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(newPassword);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);

	// Penyusun tidak memegang A-1 dan A-2: menu akun dan formulir customer tidak ditawarkan.
	await expect(page.getByRole('link', { name: en.navCustomers })).toBeVisible();
	await expect(page.getByRole('link', { name: en.navAccounts })).toHaveCount(0);
	await page.getByRole('link', { name: en.navCustomers }).click();
	await expect(page.getByRole('heading', { name: en.customersListTitle })).toBeVisible();
	await expect(page.getByRole('heading', { name: en.customerAddTitle })).toHaveCount(0);

	// Layar dilewati: permintaan langsung dari browser tetap ditolak server.
	const statuses = await page.evaluate(async () => {
		const create = await fetch('/api/v1/customers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ code: 'XY', name: 'Bypass Customer' })
		});
		const accounts = await fetch('/api/v1/accounts');
		return { create: create.status, createCode: (await create.json()).code, accounts: accounts.status };
	});
	expect(statuses).toEqual({ create: 403, createCode: 'forbidden', accounts: 403 });
});

test('admin mengubah peran, menonaktifkan, mengaktifkan, dan mereset akun', async ({ page }) => {
	await logIn(page);
	await page.goto('/admin/accounts');

	// PPAP Staff punya akun kerja dan akun admin terpisah (D-72).
	const first = await createWorkAccount(page, 'PPAP Staff E2E', 'ppap.staff.e2e', [en.roleAuthor]);
	const form = page.locator('form');
	await form.getByLabel(en.accountPersonExisting).check();
	await form.getByLabel(en.accountPersonChoose).selectOption({ label: 'PPAP Staff E2E' });
	await form.getByLabel(en.accountKindAdmin).check();
	await form.getByLabel(en.accountLogin).fill('admin.ppap.e2e');
	await form.getByRole('button', { name: en.accountCreate }).click();
	await temporaryPasswordFor(page, 'admin.ppap.e2e');
	const adminRow = page.getByRole('row', { name: /admin\.ppap\.e2e/ });
	await expect(adminRow).toContainText(en.accountKindAdmin);
	await expect(adminRow).toContainText(en.roleAdmin);
	// Akun admin selalu hanya memegang peran Admin: perannya tidak bisa diubah.
	await expect(adminRow.getByRole('button', { name: en.accountEditRoles })).toHaveCount(0);

	// Ubah peran: tambah Verifikator.
	const row = page.getByRole('row', { name: /ppap\.staff\.e2e/ });
	await row.getByRole('button', { name: en.accountEditRoles }).click();
	await row.getByLabel(en.roleVerifier, { exact: true }).check();
	await row.getByRole('button', { name: en.accountSaveRoles }).click();
	await expect(row).toContainText(`${en.roleAuthor}, ${en.roleVerifier}`);

	// Nonaktifkan, lalu aktifkan lagi. Setiap aksi yang mengakhiri sesi minta konfirmasi.
	let dialog = acceptNextDialog(page);
	await row.getByRole('button', { name: en.accountDisable }).click();
	expect(await dialog).toBe(en.accountDisableConfirm.replace('{login}', 'ppap.staff.e2e'));
	await expect(row).toContainText(en.accountStatusDisabled);
	await row.getByRole('button', { name: en.accountEnable }).click();
	await expect(row).not.toContainText(en.accountStatusDisabled);

	// Reset password memberi password sementara yang baru.
	dialog = acceptNextDialog(page);
	await row.getByRole('button', { name: en.accountResetPassword }).click();
	expect(await dialog).toBe(en.accountResetConfirm.replace('{login}', 'ppap.staff.e2e'));
	const second = await temporaryPasswordFor(page, 'ppap.staff.e2e');
	expect(second).not.toBe(first);

	// Akun sendiri tidak bisa di-reset atau dinonaktifkan dari halaman ini.
	const ownRow = page.getByRole('row', { name: /admin\.e2e/ });
	await expect(ownRow.getByRole('button', { name: en.accountResetPassword })).toHaveCount(0);
	await expect(ownRow.getByRole('button', { name: en.accountDisable })).toHaveCount(0);
});
