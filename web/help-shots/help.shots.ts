// Memotret layar untuk halaman Help (D-141), dalam bahasa Inggris dan
// Indonesia. Hasilnya ditulis ke web/static/help/<bahasa>/<nama>.png dan
// ikut di-commit, supaya build biasa tidak butuh browser.
//
// Jalankan lewat: npm --prefix web run help:shots
// lalu npm --prefix web run build, supaya gambar baru masuk ke binary.
//
// Skrip ini memakai database revioten_e2e yang dibuat ulang dari kosong,
// seperti test E2E. Semua nama di sini contoh (CLAUDE.md bagian 7).
import { expect, test, type Locator, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from '../e2e/account.mjs';
import { logOut, openFromMenu } from '../e2e/helpers.ts';

const languages = ['en', 'id'] as const;

// shot memotret layar sekarang dalam kedua bahasa, lalu kembali ke Inggris
// supaya langkah berikutnya tetap menemukan label Inggris. target, bila ada,
// digulir ke atas layar dulu. Kata sandi sementara ditutup kotak abu-abu.
async function shot(page: Page, name: string, target?: Locator): Promise<void> {
	// Pop-up ditutup, supaya tidak menutupi isi layar.
	await page.evaluate(() => document.querySelectorAll<HTMLElement>('.toast').forEach((toast) => toast.click()));
	// Elemen sasaran dipegang dulu: sesudah bahasa diganti, teks Inggrisnya tidak ada lagi.
	const handle = target ? await target.elementHandle() : null;
	for (const language of languages) {
		await page.getByRole('group', { name: /Language|Bahasa/ }).getByRole('button', { name: language.toUpperCase() }).click();
		await expect(page.locator('html')).toHaveAttribute('lang', language);
		if (handle) {
			await handle.evaluate((el) => el.scrollIntoView({ block: 'start' }));
			await page.mouse.wheel(0, -90); // beri ruang untuk bilah atas
		} else {
			await page.evaluate(() => window.scrollTo(0, 0));
		}
		await page.waitForTimeout(250);
		await page.screenshot({
			path: `static/help/${language}/${name}.png`,
			animations: 'disabled',
			mask: [page.locator('[role="status"] code'), page.locator('.toast')]
		});
	}
	await page.getByRole('group', { name: /Language|Bahasa/ }).getByRole('button', { name: 'EN' }).click();
	await expect(page.locator('html')).toHaveAttribute('lang', 'en');
}

async function logInAs(page: Page, login: string, password: string): Promise<void> {
	await page.getByLabel(en.loginName).fill(login);
	await page.getByLabel(en.loginPassword).fill(password);
	await page.getByRole('button', { name: en.loginSubmit }).click();
}

test('potret layar untuk halaman Help', async ({ page }) => {
	const engineer = { login: 'ppap.engineer', name: 'Contoh PPAP Engineer', password: 'kopi pagi di pabrik' };

	await page.goto('/login');
	await shot(page, 'login');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await shot(page, 'home');

	// Master data.
	await openFromMenu(page, en.navGroupMaster, en.navCustomers);
	await page.getByLabel(en.projectGroupCode, { exact: true }).fill('GR1');
	await page.getByLabel(en.projectGroupName, { exact: true }).fill('Group 1');
	await page.getByRole('button', { name: en.projectGroupAdd }).click();
	await expect(page.getByText(en.projectGroupSaved.replace('{code}', 'GR1'))).toBeVisible();
	for (const [code, name] of [
		['LM', 'Contoh Customer LM'],
		['AB', 'Contoh Customer AB']
	]) {
		await page.getByLabel(en.customerCode, { exact: true }).fill(code);
		await page.getByLabel(en.customerName, { exact: true }).fill(name);
		await page.getByLabel(en.customerProjectGroup, { exact: true }).selectOption({ label: 'GR1 · Group 1' });
		await page.getByRole('button', { name: en.customerSave }).click();
		await expect(page.getByRole('row', { name: new RegExp(name) })).toBeVisible();
	}
	await shot(page, 'customers');

	await openFromMenu(page, en.navGroupMaster, en.navModels);
	await page.getByLabel(en.modelsCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.modelGroupName, { exact: true }).fill('Seri K');
	await page.getByRole('button', { name: en.modelGroupAdd }).click();
	await expect(page.getByText(en.modelGroupSaved.replace('{name}', 'Seri K'))).toBeVisible();
	await page.getByLabel(en.modelName, { exact: true }).fill('K-125');
	await page.getByLabel(en.modelGroup, { exact: true }).selectOption({ label: 'Seri K' });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: 'K-125 Seri K' })).toBeVisible();

	// Akun kerja: kotak kata sandi sementara ikut terpotret, isinya ditutup.
	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	const form = page.locator('form');
	await form.getByLabel(en.accountPersonNew).check();
	await form.getByLabel(en.accountPersonName).fill(engineer.name);
	await form.getByLabel(en.accountKindWork).check();
	await form.getByLabel(en.accountLogin).fill(engineer.login);
	await form.getByLabel(en.roleAuthor, { exact: true }).check();
	await form.getByLabel(en.roleTemplateMaintainer, { exact: true }).check();
	await form.getByLabel(en.roleDocumentControl, { exact: true }).check();
	await form.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', engineer.login));
	const temporary = (await notice.locator('code').textContent()) ?? '';
	await shot(page, 'accounts');

	await openFromMenu(page, en.navGroupMaster, en.navPositions);
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'PPAP Engineer' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: engineer.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(page.getByText(en.holderAdded.replace('{name}', engineer.name).replace('{position}', 'PPAP Engineer'))).toBeVisible();
	await shot(page, 'positions');

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatChoose).selectOption({ index: 2 });
	const columns = page.getByRole('heading', { name: en.formatColumns });
	await expect(columns).toBeVisible();
	await shot(page, 'format-templates', columns);

	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await expect(page.getByRole('heading', { name: en.formSetCustomersTitle })).toBeVisible();
	await shot(page, 'form-sets');

	// Pengelola template.
	await logOut(page);
	await logInAs(page, engineer.login, temporary);
	await expect(page).toHaveURL('/account/password');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(engineer.password);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(engineer.password);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);

	await openFromMenu(page, en.navGroupTemplates, en.navProcesses);
	for (const [name, aliases] of [
		['Laser marking', 'Laser mark\nLM marking'],
		['Manual soldering', 'Hand soldering']
	]) {
		await page.getByLabel(en.processName, { exact: true }).fill(name);
		await page.getByLabel(en.processAliases, { exact: true }).fill(aliases);
		await page.getByRole('button', { name: en.processAdd }).click();
		await expect(page.getByText(en.processSaved.replace('{name}', name))).toBeVisible();
	}
	await shot(page, 'processes');

	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await expect(page.getByRole('heading', { name: en.processTemplateTitle, exact: true })).toBeVisible();
	await page.getByLabel(en.processTemplateChoose, { exact: true }).selectOption({ label: 'Laser marking' });
	await page.getByRole('button', { name: en.processTemplateCreate }).click();
	await page.getByLabel(en.subprocessName, { exact: true }).fill('Marking');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'Marking') }).click();
	await page.getByLabel(en.pointDocumentType).selectOption({ index: 1 });
	await page.getByLabel(en.pointText, { exact: true }).fill('Laser type sesuai spesifikasi');
	await page.getByRole('button', { name: en.pointAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointAdded);
	await page.getByRole('row', { name: /Laser type sesuai spesifikasi/ }).getByRole('button', { name: en.variantAdd }).click();
	const variantForm = page.locator('form').filter({ has: page.getByLabel(en.variantText) });
	await variantForm.getByLabel(en.pointScope).selectOption('model');
	await variantForm.getByLabel(en.scopeLevelCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await variantForm.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-125 · Seri K' });
	await variantForm.getByLabel(en.variantText).fill('Laser type: fiber 20 W');
	await variantForm.getByRole('button', { name: en.variantAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.variantAdded);
	await shot(page, 'process-templates', page.getByRole('heading', { name: en.pointsTitle.replace('{name}', 'Marking') }));

	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	await shot(page, 'versions');

	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	// Tunggu formulir daftar tampil dulu: saringan register juga punya kotak "Customer".
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toContainText('SH-LM-001');
	await shot(page, 'numbers');

	// Admin: audit log.
	await logOut(page);
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await expect(page.getByRole('heading', { name: en.auditTitle, exact: true })).toBeVisible();
	await expect(page.getByRole('row').nth(3)).toBeVisible();
	await shot(page, 'audit');
});
