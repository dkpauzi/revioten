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
import { createWorkAccount, logInFirstTime, logOut, openFromMenu, openTool, tool } from '../e2e/helpers.ts';

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
	test.setTimeout(8 * 60 * 1000);
	const control = { login: 'dok.kontrol', password: 'rilis selalu dengan scan' };
	const engineer = { login: 'ppap.engineer', name: 'Contoh PPAP Engineer', password: 'kopi pagi di pabrik' };

	await page.goto('/login');
	await shot(page, 'login');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	// Home akun admin adalah dasbor kesehatan web (D-205).
	await expect(page.getByRole('heading', { name: en.adminHomeTitle, level: 1 })).toBeVisible();
	await shot(page, 'admin-health');

	// Akun kerja: kotak kata sandi sementara ikut terpotret, isinya ditutup.
	// Master data dikerjakan PPAP Engineer sesudah ini (D-209).
	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	const form = page.locator('main form').first();
	await form.getByLabel(en.accountPersonNew).check();
	await form.getByLabel(en.accountPersonName).fill(engineer.name);
	await form.getByLabel(en.accountKindWork).check();
	await form.getByLabel(en.accountLogin).fill(engineer.login);
	await form.getByLabel(en.rolePpapEngineerSupervisor, { exact: true }).check();
	await form.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', engineer.login));
	const temporary = (await notice.locator('code').textContent()) ?? '';
	await shot(page, 'accounts');

	// Akun lain untuk gambar berikutnya: dokumen kontrol, Project Leader GR1, dan
	// dua Manager penanda tangan (D-198). ID orangnya dipakai PPAP Engineer.
	const controlTemporary = await createWorkAccount(page, 'Contoh Dokumen Kontrol', control.login, ['ppap_engineer']);
	const signers = await page.evaluate(async () => {
		const out: Record<string, string> = {};
		for (const [name, login, role] of [
			['Contoh Project Leader GR1', 'pl.gr1', 'project_leader'],
			['Contoh NPI Manager', 'npi.manager', 'manager'],
			['Contoh QA QC Manager', 'qa.manager', 'manager']
		]) {
			const created = await fetch('/api/v1/accounts', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ person_name: name, kind: 'work', login, roles: [{ role, scope: 'all' }] })
			}).then((r) => r.json());
			out[login] = created.account.person.id;
		}
		return out;
	});
	// Pengaman tetap di admin: mode migrasi (D-209).
	await page.evaluate(() =>
		fetch('/api/v1/system-settings', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ migration_mode: true }) })
	);
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);
	await expect(page.getByRole('heading', { name: en.rolesTitle, level: 1 })).toBeVisible();
	await expect(page.getByRole('row').nth(3)).toBeVisible();
	await shot(page, 'roles');

	// PPAP Engineer Supervisor login pertama kali, lalu mengurus master data (E-11, D-209).
	await logOut(page);
	await logInAs(page, engineer.login, temporary);
	await expect(page).toHaveURL('/account/password');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(engineer.password);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(engineer.password);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);

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

	await openFromMenu(page, en.navGroupMaster, en.navPositions);
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'PPAP Engineer' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: engineer.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(page.getByText(en.holderAdded.replace('{name}', engineer.name).replace('{position}', 'PPAP Engineer'))).toBeVisible();
	await shot(page, 'positions');

	// Untuk gambar tanda tangan digital (D-198): rute default LM, dan Project
	// Leader GR1 serta dua Manager sebagai pemegang jabatan slot. Lewat API.
	await page.evaluate(async (signers) => {
		const json = (path: string, method: string, body?: object) =>
			fetch(path, { method, headers: { 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined }).then((r) => r.json());
		const customers = await json('/api/v1/customers', 'GET');
		const lm = customers.items.find((c: { code: string }) => c.code === 'LM');
		const routes = await json('/api/v1/approval-routes', 'GET');
		const route = routes.items.find((r: { name: string }) => r.name === 'Up to QA QC Manager');
		await json(`/api/v1/customers/${lm.id}/default-route`, 'PUT', { route_id: route.id });
		const groups = await json('/api/v1/project-groups', 'GET');
		const gr1 = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const positions = await json('/api/v1/positions', 'GET');
		for (const [login, key] of [
			['pl.gr1', 'project_leader'],
			['npi.manager', 'npi_manager'],
			['qa.manager', 'qa_qc_manager']
		]) {
			const position = positions.items.find((x: { key: string }) => x.key === key);
			await json(`/api/v1/positions/${position.id}/holders`, 'POST', {
				person_id: signers[login],
				project_group_id: position.per_project_group ? gr1.id : undefined
			});
		}
	}, signers);

	await openFromMenu(page, en.navGroupMaster, en.navDocumentSettings);
	await expect(page.getByLabel(en.settingsSubmissionFormNumber)).toBeVisible();
	await shot(page, 'document-settings');

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatChoose).selectOption({ index: 2 });
	const columns = page.getByRole('heading', { name: en.formatColumns });
	await expect(columns).toBeVisible();
	await shot(page, 'format-templates', columns);

	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await expect(page.getByRole('heading', { name: en.formSetCustomersTitle })).toBeVisible();
	await shot(page, 'form-sets');

	// Persiapan untuk gambar Tahap 4 sampai 8: format terbit.
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	for (const name of ['PFD · PFD', 'PFMEA AIAG 4th · PFMEA', 'CP · CP']) {
		await page.getByLabel(en.formatChoose, { exact: true }).selectOption({ label: name });
		page.once('dialog', (dialog) => dialog.accept());
		await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
		await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	}

	// Admin kembali untuk service account (butuh customer LM) dan email notifikasi.
	await logOut(page);
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('heading', { name: en.adminHomeTitle, level: 1 })).toBeVisible();
	// Service account API integrasi (Tahap 9), dengan satu token aktif.
	await openFromMenu(page, en.navGroupAdmin, en.navServiceAccounts);
	await page.getByLabel(en.serviceAccountName, { exact: true }).fill('Contoh MES');
	await page.getByLabel(en.serviceAccountTeam, { exact: true }).fill('Tim MES');
	await page.getByLabel(en.serviceAccountContact, { exact: true }).fill('mes@contoh.test');
	await page.getByLabel(en.serviceAccountPurpose, { exact: true }).fill('Membaca CP terbit untuk lini');
	await page.getByRole('checkbox', { name: en.scope_documents_read }).check();
	await page.getByRole('checkbox', { name: en.scope_revisions_read }).check();
	await page.getByRole('checkbox', { name: 'LM · Contoh Customer LM' }).check();
	await page.getByRole('button', { name: en.serviceAccountCreate }).click();
	await expect(page.getByRole('status').last()).toHaveText(en.serviceAccountCreated.replace('{name}', 'Contoh MES'));
	const mes = page.locator('section', { has: page.getByRole('heading', { name: 'Contoh MES', level: 2 }) });
	await mes.getByRole('button', { name: en.serviceTokenIssue }).click();
	await expect(mes.getByRole('cell', { name: en.serviceTokenActive })).toBeVisible();
	// Nilai token ditutup kotak abu-abu, seperti kata sandi sementara.
	await page.evaluate(() => document.querySelectorAll('.issued').forEach((el) => el.remove()));
	await shot(page, 'service-accounts');

	// Email notifikasi (D-194): alamat admin, lalu halaman Notifications.
	await page.goto('/account/password');
	await page.getByLabel(en.accountEmail, { exact: true }).fill('admin@contoh.test');
	await page.getByRole('button', { name: en.accountEmailSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.accountEmailSaved);
	await openFromMenu(page, en.navGroupAdmin, en.navNotifications);
	await page.getByRole('button', { name: en.notifyTest }).click();
	await expect(page.getByRole('status')).toHaveText(en.notifyTestQueued);
	await shot(page, 'notifications');

	// PPAP Engineer Supervisor lagi untuk bagian template dan dokumen.
	await logOut(page);
	await logInAs(page, engineer.login, engineer.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	// Home akun kerja.
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await shot(page, 'home');

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

	// Dokumen: form set terbit, dokumen PFD dibuat dan dirakit (poin contoh tadi poin PFD).
	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	const formSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	page.once('dialog', (dialog) => dialog.accept());
	await formSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.formSetPublished.replace('{version}', '1').replace('{name}', 'AIAG 4th set'));
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.documentNumber, { exact: true }).selectOption({ label: 'SH-LM-001 · PFD' });
	await page.getByLabel('K-125').first().check();
	await page.getByRole('button', { name: en.documentCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentCreated.replace('{number}', 'SH-LM-001'));
	await page.getByRole('link', { name: 'SH-LM-001' }).click();
	await expect(page.getByRole('heading', { name: 'SH-LM-001', level: 1 })).toBeVisible();
	// PFD diisi langsung per dokumen, tanpa template (C4, D-211): satu langkah Laser marking.
	await page.evaluate(async () => {
		const id = new URL(location.href).searchParams.get('id');
		const doc = await (await fetch(`/api/v1/documents/${id}`)).json();
		const fields = (await (await fetch(`/api/v1/format-template-versions/${doc.format_version_id}`)).json()).fields as { id: string; key?: string }[];
		const processes = (await (await fetch('/api/v1/processes')).json()).items as { id: string; name: string }[];
		const symbol = fields.find((f) => f.key === 'flow_symbol')!.id;
		await fetch(`/api/v1/documents/${id}/rows`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ operation_number: '10', process_id: processes.find((p) => p.name === 'Laser marking')!.id, content: { [symbol]: 'process' } })
		});
	});
	await page.reload();
	await expect(page.locator('.workbook .tabulator-row')).toHaveCount(1);
	// Workbook (D-145, C1): kepala, langkah berikutnya, toolbar, lembar, dan tab sheet.
	await shot(page, 'documents');
	await openTool(page, en.printTitle);
	await shot(page, 'printing', page.getByRole('heading', { name: en.printTitle, exact: true }));
	// Bagian Digital signature dengan calon penanda tangan setiap slot (D-198, D-203).
	await openTool(page, en.sigTitle);
	await page.getByLabel(en.sigSlotChoose.replace('{slot}', 'Engineer'), { exact: true }).selectOption({ label: 'Contoh Project Leader GR1 · Project Leader GR1' });
	await shot(page, 'signatures', page.getByRole('heading', { name: en.sigTitle, level: 2 }));
	const cpUrl = page.url();

	// Dokumen kontrol mencatat rilis, supaya perubahan bisa diajukan.
	await logOut(page);
	await logInFirstTime(page, control.login, controlTemporary, control.password);
	await page.goto(cpUrl);
	await openTool(page, en.revisions);
	const revisions = page.locator('section', { has: page.getByRole('heading', { name: en.revisions, exact: true }) });
	await revisions.getByLabel(en.scanUpload).setInputFiles({ name: 'SH-LM-001-00-signed.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4 contoh scan') });
	await revisions.getByRole('button', { name: en.scanUploadButton }).click();
	await expect(revisions.getByRole('status')).toHaveText(en.scanUploaded.replace('{file}', 'SH-LM-001-00-signed.pdf'));
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') }).click();
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();
	await shot(page, 'document-control', page.getByRole('heading', { name: en.revisions, exact: true }));
	// Form Control Submission dokumen yang baru terbit (D-207).
	await openTool(page, en.subTitle);
	await shot(page, 'submission-control', page.getByRole('heading', { name: en.subTitle, level: 2 }));

	// Penyusun mengajukan perubahan, lalu memasukkan dokumen lama.
	await logOut(page);
	await logInAs(page, engineer.login, engineer.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.changeTrigger, { exact: true }).selectOption({ label: 'Customer claim' });
	await page.getByLabel(en.changeType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.changeFourM, { exact: true }).selectOption({ label: 'Method' });
	await page.getByLabel(en.changeReason, { exact: true }).fill('Contoh: tambah pemeriksaan fokus laser');
	await page.getByRole('checkbox', { name: /SH-LM-001/ }).check();
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeCreated.replace('{ref}', 'CHG-0001'));
	await shot(page, 'changes');

	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	const legacy = page.locator('section', { has: page.getByRole('heading', { name: en.migrateTitle }) });
	await legacy.getByLabel(en.migrateNumber, { exact: true }).selectOption({ label: 'SC-LM-001 · CP' });
	await legacy.getByRole('checkbox', { name: 'K-125' }).check();
	await legacy.getByLabel(en.migrateRevision, { exact: true }).fill('4');
	await legacy.getByLabel(en.migrateEffectiveDate, { exact: true }).fill('2025-03-10');
	await legacy.getByLabel(en.migratePreparedBy, { exact: true }).fill('Contoh Penyusun Lama');
	await legacy.getByLabel(en.migrateApprovedBy, { exact: true }).fill('Contoh NPI Manager, Contoh QA Manager');
	await legacy.getByRole('button', { name: en.migrateSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.migrateDone.replace('{number}', 'SC-LM-001').replace('{revision}', '4'));
	await page.getByRole('link', { name: 'SC-LM-001' }).click();
	await openTool(page, en.migrationTitle);
	const baseline = page.locator('section', { has: page.getByRole('heading', { name: en.migrationTitle, exact: true }) });
	await baseline.getByRole('button', { name: en.historyAdd }).click();
	await baseline.getByLabel(`${en.historyRevision} 1`).fill('-00');
	await baseline.getByLabel(`${en.historyDate} 1`).fill('2023-01-05');
	await baseline.getByLabel(`${en.historyDescription} 1`).fill('Initial release');
	await baseline.getByRole('button', { name: en.historySave }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.historySaved);
	await shot(page, 'migration', page.getByRole('heading', { name: en.migrationTitle, exact: true }));

	await openFromMenu(page, en.navGroupDocuments, en.navDashboard);
	await expect(page.getByRole('heading', { name: en.dashboardRows })).toBeVisible();
	await shot(page, 'dashboard');

	// Dokumen terbit (D-195), review berkala (D-196), dan laporan kinerja (D-197).
	await openFromMenu(page, en.navGroupDocuments, en.navReleasedDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await expect(page.getByRole('link', { name: 'SH-LM-001' })).toBeVisible();
	await shot(page, 'released-documents');

	await openFromMenu(page, en.navGroupDocuments, en.navPeriodicReviews);
	await page.getByRole('button', { name: en.reviewTabAll, exact: true }).click();
	await expect(page.getByRole('link', { name: 'SH-LM-001' })).toBeVisible();
	await shot(page, 'periodic-review');

	await openFromMenu(page, en.navGroupDocuments, en.navPerformance);
	await expect(page.getByRole('heading', { name: en.performanceReceived, exact: true })).toBeVisible();
	await shot(page, 'performance');

	// Admin: audit log.
	await logOut(page);
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await expect(page.getByRole('heading', { name: en.auditTitle, exact: true })).toBeVisible();
	await expect(page.getByRole('row').nth(3)).toBeVisible();
	await shot(page, 'audit');
});
