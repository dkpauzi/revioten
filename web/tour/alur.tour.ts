// Tur otomatis alur web, dari database kosong sampai dokumen terbit, direvisi,
// dicetak, dokumen lama dimigrasi, dan dasbor (Tahap 0 sampai Tahap 8).
// Mengikuti docs/panduan/skenario-alur-web.md langkah demi langkah. Semua
// dikerjakan lewat layar (klik dan ketik), tidak lewat API, supaya yang
// terlihat sama dengan yang dikerjakan orang.
//
// Semua nama di sini contoh (CLAUDE.md bagian 7).
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from '../e2e/account.mjs';
import { logOut, openFromMenu } from '../e2e/helpers.ts';

// Lama keterangan langkah ditahan di layar sebelum tur lanjut.
const pause = { slow: 6000, normal: 3500, fast: 0 }[process.env.TOUR_SPEED ?? 'normal'] ?? 3500;

// say menampilkan keterangan langkah di pita bawah layar, lalu menunggu
// sebentar supaya sempat dibaca. Pita ini hanya ditambahkan oleh tur; aplikasi
// aslinya tidak punya pita ini.
async function say(page: Page, step: string, text: string): Promise<void> {
	await page.evaluate(
		({ step, text }) => {
			let bar = document.getElementById('tour-caption');
			if (!bar) {
				bar = document.createElement('div');
				bar.id = 'tour-caption';
				bar.style.cssText =
					'position:fixed;left:0;right:0;bottom:0;z-index:9999;padding:14px 24px;' +
					'background:#111827;color:#ffffff;font:500 17px/1.4 system-ui,sans-serif;' +
					'pointer-events:none;box-shadow:0 -4px 16px rgba(0,0,0,.25)';
				document.body.appendChild(bar);
			}
			bar.textContent = '';
			const label = document.createElement('strong');
			label.textContent = step + '  ';
			label.style.color = '#a5b4fc';
			bar.append(label, text);
		},
		{ step, text }
	);
	await page.waitForTimeout(pause);
}

// logInAs mengisi formulir login untuk satu akun.
async function logInAs(page: Page, login: string, password: string): Promise<void> {
	await page.getByLabel(en.loginName).fill(login);
	await page.getByLabel(en.loginPassword).fill(password);
	await page.getByRole('button', { name: en.loginSubmit }).click();
}

// createAccount mengisi formulir akun kerja di halaman Accounts dan
// mengembalikan password sementaranya. roles berisi nama peran di layar.
async function createAccount(page: Page, name: string, login: string, roles: string[]): Promise<string> {
	const form = page.locator('form');
	await form.getByLabel(en.accountPersonNew).check();
	await form.getByLabel(en.accountPersonName).fill(name);
	await form.getByLabel(en.accountKindWork).check();
	await form.getByLabel(en.accountLogin).fill(login);
	for (const role of roles) {
		await form.getByLabel(role, { exact: true }).check();
	}
	await form.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', login));
	return (await notice.locator('code').textContent()) ?? '';
}

// firstLogin masuk dengan password sementara dan langsung menggantinya.
async function firstLogin(page: Page, login: string, temporary: string, password: string): Promise<void> {
	await logInAs(page, login, temporary);
	await expect(page).toHaveURL('/account/password');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(password);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(password);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);
}

// switchTo keluar, lalu masuk sebagai akun lain yang passwordnya sudah diganti.
async function switchTo(page: Page, login: string, password: string): Promise<void> {
	await logOut(page);
	await logInAs(page, login, password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
}

// openDocument membuka satu dokumen customer LM dari halaman Documents.
async function openDocument(page: Page, number: string): Promise<void> {
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('link', { name: number }).click();
	await expect(page.getByRole('heading', { name: number, level: 1 })).toBeVisible();
}

// uploadScan mengunggah scan bertanda tangan contoh di bagian Revisions.
async function uploadScan(page: Page, file: string): Promise<void> {
	const revisions = page.locator('section', { has: page.getByRole('heading', { name: en.revisions, exact: true }) });
	await revisions.getByLabel(en.scanUpload).setInputFiles({ name: file, mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4 contoh scan') });
	await revisions.getByRole('button', { name: en.scanUploadButton }).click();
	await expect(revisions.getByRole('status')).toHaveText(en.scanUploaded.replace('{file}', file));
}

test('alur dari admin sampai dokumen terbit, revisi, migrasi, dan dasbor', async ({ page }) => {
	// Tur ini panjang; batas waktunya mengikuti kecepatan tur.
	test.setTimeout(15 * 60 * 1000);
	const engineer = { login: 'ppap.engineer', name: 'Contoh PPAP Engineer', password: 'kopi pagi di pabrik' };
	const control = { login: 'dok.kontrol', name: 'Contoh Dokumen Kontrol', password: 'rilis selalu dengan scan' };
	const verifier = { login: 'apqp.controller', name: 'Contoh APQP Controller', password: 'verifikasi dengan catatan' };

	// ---------- Bagian A: admin ----------
	await page.goto('/login');
	await say(page, 'Langkah 1', 'Admin login. Akun admin dibuat sekali lewat perintah create-admin.');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await say(page, 'Langkah 1', 'Menu atas: Home, Documents, Template library, Master data, dan Admin.');

	await openFromMenu(page, en.navGroupMaster, en.navCustomers);
	await say(page, 'Langkah 2', 'Master data > Customers. Project group dibuat dulu, baru customer.');
	await page.getByLabel(en.projectGroupCode, { exact: true }).fill('GR1');
	await page.getByLabel(en.projectGroupName, { exact: true }).fill('Group 1');
	await page.getByRole('button', { name: en.projectGroupAdd }).click();
	await expect(page.getByText(en.projectGroupSaved.replace('{code}', 'GR1'))).toBeVisible();

	await say(page, 'Langkah 2', 'Customer: kode dua huruf (masuk ke nomor dokumen), nama, dan project group.');
	await page.getByLabel(en.customerCode, { exact: true }).fill('LM');
	await page.getByLabel(en.customerName, { exact: true }).fill('Contoh Customer LM');
	await page.getByLabel(en.customerProjectGroup, { exact: true }).selectOption({ label: 'GR1 · Group 1' });
	await page.getByRole('button', { name: en.customerSave }).click();
	await expect(page.getByRole('row', { name: /Contoh Customer LM/ })).toBeVisible();

	await openFromMenu(page, en.navGroupMaster, en.navModels);
	await say(page, 'Langkah 3', 'Master data > Models. Pilih customer, lalu buat group model dan model.');
	await page.getByLabel(en.modelsCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.modelGroupName, { exact: true }).fill('Seri K');
	await page.getByRole('button', { name: en.modelGroupAdd }).click();
	await expect(page.getByText(en.modelGroupSaved.replace('{name}', 'Seri K'))).toBeVisible();
	await page.getByLabel(en.modelName, { exact: true }).fill('K-125');
	await page.getByLabel(en.modelGroup, { exact: true }).selectOption({ label: 'Seri K' });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: 'K-125 Seri K' })).toBeVisible();
	await say(page, 'Langkah 3', 'Model kedua, K-200, dibuat tanpa group untuk mencoba prioritas varian nanti.');
	await page.getByLabel(en.modelName, { exact: true }).fill('K-200');
	await page.getByLabel(en.modelGroup, { exact: true }).selectOption({ index: 0 });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: `K-200 ${en.modelGroupNone}` })).toBeVisible();

	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	await say(page, 'Langkah 4', 'Admin > Accounts. Akun kerja untuk PPAP Engineer: orang baru, jenis Work account.');
	const accountForm = page.locator('form');
	await accountForm.getByLabel(en.accountPersonNew).check();
	await accountForm.getByLabel(en.accountPersonName).fill(engineer.name);
	await accountForm.getByLabel(en.accountKindWork).check();
	await accountForm.getByLabel(en.accountLogin).fill(engineer.login);
	await say(page, 'Langkah 4', 'Peran menentukan hak: Author (nomor, dokumen), Template maintainer (template), Document control (Void).');
	await accountForm.getByLabel(en.roleAuthor, { exact: true }).check();
	await accountForm.getByLabel(en.roleTemplateMaintainer, { exact: true }).check();
	await accountForm.getByLabel(en.roleDocumentControl, { exact: true }).check();
	await say(page, 'Langkah 4', 'Untuk contoh ini akun yang sama juga Form approver (biasanya PPAP Supervisor) dan PFMEA-CP checker (D-65).');
	await accountForm.getByLabel(en.roleFormApprover, { exact: true }).check();
	await accountForm.getByLabel(en.rolePfmeaCpChecker, { exact: true }).check();
	await accountForm.getByLabel(en.roleMigration, { exact: true }).check();
	await accountForm.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', engineer.login));
	const temporary = (await notice.locator('code').textContent()) ?? '';
	await say(page, 'Langkah 4', 'Password sementara tampil SATU KALI. Berikan ke orangnya; ia wajib menggantinya.');
	await say(page, 'Langkah 4', 'Dua akun lagi: Dokumen kontrol (mencatat rilis, juga petugas migrasi) dan Verifikator. Penyusun tidak boleh merilis atau memverifikasi pekerjaannya sendiri.');
	const controlTemporary = await createAccount(page, control.name, control.login, [en.roleDocumentControl, en.roleMigration]);
	const verifierTemporary = await createAccount(page, verifier.name, verifier.login, [en.roleVerifier]);

	await openFromMenu(page, en.navGroupMaster, en.navPositions);
	await say(page, 'Langkah 5', 'Master data > Positions. Catat siapa memegang jabatan PPAP Engineer.');
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'PPAP Engineer' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: engineer.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', engineer.name).replace('{position}', 'PPAP Engineer'))
	).toBeVisible();

	await say(page, 'Langkah 5', 'Verifikator perubahan harus memegang jabatan: di sini APQP/PPAP Controller.');
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'APQP/PPAP Controller' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: verifier.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', verifier.name).replace('{position}', 'APQP/PPAP Controller'))
	).toBeVisible();

	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	await say(page, 'Langkah 5', 'Admin > Settings. Mode migrasi dinyalakan selama dokumen lama dimasukkan (bawaannya mati).');
	await page.getByLabel(en.settingsMigration, { exact: true }).check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await say(page, 'Langkah 6', 'Format template sudah terisi sejak awal. Admin memeriksa labelnya, lalu menerbitkan versi 1.');
	for (const name of ['PFD · PFD', 'PFMEA AIAG 4th · PFMEA', 'CP · CP']) {
		await page.getByLabel(en.formatChoose, { exact: true }).selectOption({ label: name });
		page.once('dialog', (dialog) => dialog.accept());
		await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
		await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	}
	await say(page, 'Langkah 6', 'Format PFD, PFMEA AIAG 4th, dan CP terbit. Form set baru bisa terbit sesudah formatnya terbit.');

	await say(page, 'Langkah 7', 'Admin logout. Pekerjaan template dan nomor bukan tugas akun admin.');
	await logOut(page);

	// ---------- Bagian B: PPAP Engineer ----------
	await say(page, 'Langkah 8', 'PPAP Engineer login pertama kali dengan password sementara.');
	await logInAs(page, engineer.login, temporary);
	await expect(page).toHaveURL('/account/password');
	await say(page, 'Langkah 8', 'Sistem langsung meminta ganti password (minimal 8 karakter).');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(engineer.password);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(engineer.password);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);

	await openFromMenu(page, en.navGroupTemplates, en.navProcesses);
	await say(page, 'Langkah 9', 'Template library > Processes. Master proses: nama dan alias (nama di dokumen lama).');
	await page.getByLabel(en.processName, { exact: true }).fill('Laser marking');
	await page.getByLabel(en.processAliases, { exact: true }).fill('Laser mark\nLM marking');
	await page.getByRole('button', { name: en.processAdd }).click();
	await expect(page.getByText(en.processSaved.replace('{name}', 'Laser marking'))).toBeVisible();

	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	// Tunggu halamannya berganti dulu: halaman Processes juga punya label "Process name".
	await expect(page.getByRole('heading', { name: en.processTemplateTitle, exact: true })).toBeVisible();
	await say(page, 'Langkah 10', 'Process templates. Satu proses punya satu template; versi pertama berstatus Draft.');
	await page.getByLabel(en.processTemplateChoose, { exact: true }).selectOption({ label: 'Laser marking' });
	await page.getByRole('button', { name: en.processTemplateCreate }).click();
	await page.getByLabel(en.subprocessName, { exact: true }).fill('Marking');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'Marking') }).click();
	await say(page, 'Langkah 10', 'Poin: jenis dokumen, teks, dan cakupan ("Use point for"). Bawaannya General.');
	await page.getByLabel(en.pointDocumentType).selectOption({ index: 1 });
	await page.getByLabel(en.pointText, { exact: true }).fill('Laser type sesuai spesifikasi');
	await page.getByRole('button', { name: en.pointAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointAdded);
	await say(page, 'Langkah 10', 'Poin PFMEA dan CP untuk sub-proses yang sama, supaya satu perakitan mengisi ketiga dokumen.');
	for (const [type, text] of [
		['PFMEA', 'Salah laser type'],
		['CP', 'Cek laser type']
	]) {
		await page.getByLabel(en.pointDocumentType).selectOption({ label: type });
		await page.getByLabel(en.pointText, { exact: true }).fill(text);
		await page.getByRole('button', { name: en.pointAdd }).click();
		await expect(page.getByRole('status')).toHaveText(en.pointAdded);
	}
	await say(page, 'Langkah 10', 'Tombol Columns: isi yang ditulis poin ke setiap kolom form saat dipakai di dokumen (D-146).');
	await page.getByRole('button', { name: en.pointContentFor.replace('{text}', 'Cek laser type') }).click();
	await page.getByLabel(/^Product (Characteristics|control)$/).fill('Laser type');
	await page.getByLabel(/^(Method|Control Method)$/).fill('Visual');
	await page.getByRole('button', { name: en.pointContentSave.replace('{text}', 'Cek laser type') }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointContentSaved);
	await say(page, 'Langkah 11', 'Varian: poinnya umum, tetapi model K-125 memakai laser fiber 20 W.');
	const pointRow = page.getByRole('row', { name: /Laser type sesuai spesifikasi/ });
	await pointRow.getByRole('button', { name: en.variantAdd }).click();
	const variantForm = page.locator('form').filter({ has: page.getByLabel(en.variantText) });
	await variantForm.getByLabel(en.pointScope).selectOption('model');
	await variantForm.getByLabel(en.scopeLevelCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await variantForm.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-125 · Seri K' });
	await variantForm.getByLabel(en.variantText).fill('Laser type: fiber 20 W');
	await variantForm.getByRole('button', { name: en.variantAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.variantAdded);

	const preview = page.locator('section').filter({ has: page.getByRole('heading', { name: en.previewTitle }) });
	await preview.scrollIntoViewIfNeeded();
	await say(page, 'Langkah 11', 'Pratinjau "Points for one model": yang paling spesifik menang.');
	await preview.getByLabel(en.scopeLevelCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await preview.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-125 · Seri K' });
	await expect(preview.getByRole('row', { name: /Laser type: fiber 20 W/ })).toBeVisible();
	await say(page, 'Langkah 11', 'K-125 memakai variannya: "fiber 20 W".');
	await preview.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-200' });
	await expect(preview.getByRole('row', { name: /Laser type sesuai spesifikasi/ })).toBeVisible();
	await say(page, 'Langkah 11', 'K-200 tidak punya varian, jadi memakai poin umum.');

	// Terbitkan versi 1, lalu buat versi 2 (M3.4, R-TPL-05).
	await page.getByLabel(en.versionChoose, { exact: true }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 12', 'Template masih Draft. Sesudah diterbitkan, versi ini tidak bisa diubah lagi.');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	await say(page, 'Langkah 12', 'Versi 1 terbit: tombol tambah dan ubah hilang, isinya hanya bisa dibaca.');
	if (process.env.TOUR_SHOT) {
		await page.screenshot({ path: process.env.TOUR_SHOT });
	}
	await page.getByRole('button', { name: en.versionNew }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionCreated.replace('{version}', '2').replace('{from}', '1'));
	await say(page, 'Langkah 12', 'Perubahan berikutnya masuk ke versi 2 (draft), salinan versi 1 dengan ID poin yang sama.');

	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await say(page, 'Langkah 13', 'Documents > Numbers. Satu set: PFD, PFMEA, dan CP dengan nomor berjalan yang sama.');
	// Tunggu formulir daftar tampil dulu: saringan register juga punya kotak "Customer".
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toContainText('SH-LM-001');
	await say(page, 'Langkah 13', 'Tiga nomor terdaftar dengan status Reserved. Nomor tidak pernah dihapus atau dipakai ulang.');

	await say(page, 'Langkah 14', 'Dokumen kontrol membatalkan (Void) satu nomor. Alasannya wajib diisi.');
	const numberRow = page.getByRole('row', { name: /^SH-LM-001 / });
	await numberRow.getByRole('button', { name: en.numberVoid }).click();
	await page.getByLabel(en.numberReason.replace('{number}', 'SH-LM-001')).fill('Contoh: salah pilih customer');
	await page.getByRole('button', { name: en.numberVoid, exact: true }).last().click();
	await expect(
		page.getByText(en.numberStatusChanged.replace('{number}', 'SH-LM-001').replace('{status}', en.numberStatusVoid))
	).toBeVisible();
	await say(page, 'Langkah 14', 'SH-LM-001 sekarang Void, tetapi tetap ada di register beserta alasannya.');

	// ---------- Bagian C: dokumen draft (Tahap 4) ----------
	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await say(page, 'Langkah 15', 'Template library > Form sets. Form approver menerbitkan form set AIAG 4th (D-139).');
	const formSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	page.once('dialog', (dialog) => dialog.accept());
	await formSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.formSetPublished.replace('{version}', '1').replace('{name}', 'AIAG 4th set'));
	await say(page, 'Langkah 15', 'Dokumen baru memakai form set customer yang sudah terbit.');

	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await say(page, 'Langkah 16', 'Set nomor kedua untuk dokumen: SH-LM-002, SF-LM-002, dan SC-LM-002.');
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toContainText('SH-LM-002');

	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await say(page, 'Langkah 17', 'Documents > Documents. Satu dokumen per nomor Reserved, untuk model K-125.');
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	for (const [number, type] of [
		['SH-LM-002', 'PFD'],
		['SF-LM-002', 'PFMEA'],
		['SC-LM-002', 'CP']
	]) {
		await page.getByLabel(en.documentNumber, { exact: true }).selectOption({ label: `${number} · ${type}` });
		await page.getByLabel('K-125').first().check();
		await page.getByRole('button', { name: en.documentCreate }).click();
		await expect(page.getByRole('status')).toHaveText(en.documentCreated.replace('{number}', number));
	}

	await page.getByRole('link', { name: 'SC-LM-002' }).click();
	await expect(page.getByRole('heading', { name: 'SC-LM-002', level: 1 })).toBeVisible();
	await say(page, 'Langkah 18', 'Workbook dokumen: tab PFD, PFMEA, CP, dan Check. Baris dirakit dari process template.');
	await page.getByRole('button', { name: en.composeOpen }).click();
	await page.getByLabel(en.composeProcess).selectOption({ label: `Laser marking · ${en.composeVersion.replace('{version}', '1')}` });
	await page.getByRole('checkbox', { name: en.composeAll.replace('{name}', 'Marking') }).check();
	await say(page, 'Langkah 18', 'Kiri: poin template yang dicentang. Kanan: baris yang akan didapat PFD, PFMEA, dan CP.');
	await page.getByRole('button', { name: en.composeApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.composeDone.replace('{process}', 'Laser marking'));
	const grid = page.locator('.workbook');
	const rows = grid.locator('.tabulator-row');
	await expect(rows).toHaveCount(1);
	await say(page, 'Langkah 18', 'Baris CP terisi dari kolom poin: Product Characteristics dan Method.');

	await say(page, 'Langkah 19', 'Sel diubah langsung di tabel seperti Excel. Perubahan lokal ini tetap tertaut ke template.');
	const product = (await grid.locator('.tabulator-col', { hasText: /^Product Characteristics$/ }).first().getAttribute('tabulator-field')) ?? '';
	await rows.first().locator(`.tabulator-cell[tabulator-field="${product}"]`).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Laser type dan daya');
	await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	await page.getByRole('button', { name: en.compareOpen }).click();
	await expect(page.locator('.compare tbody tr').first()).toContainText(en.compareDifferent);
	await say(page, 'Langkah 19', 'Compare with template: baris ini berbeda dari template, kolom per kolom.');

	await page.getByRole('link', { name: en.checkTab, exact: true }).click();
	await expect(page.getByRole('heading', { name: en.checkTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 20', 'Tab Check: PFMEA dan CP berdampingan per proses. PPAP Engineer memeriksa, lalu memberi tanda.');
	await page.getByRole('button', { name: en.checkMarkFor.replace('{process}', 'Laser marking') }).click();
	await expect(page.getByRole('status')).toHaveText(en.checkMarked.replace('{process}', 'Laser marking'));
	await say(page, 'Langkah 20', 'Tanda menyimpan siapa dan kapan. Bila barisnya berubah lagi, tanda meminta dicek ulang.');

	await page.getByRole('link', { name: 'PFD', exact: true }).click();
	await page.getByRole('button', { name: en.diagramOpen }).click();
	await expect(page.getByTestId('pfd-diagram').locator('.svelte-flow__node')).toHaveCount(1);
	await say(page, 'Langkah 21', 'Diagram PFD digambar otomatis dari baris PFD, menurut urutannya.');
	// ---------- Bagian D: cetak dan rilis (Tahap 6) ----------
	await page.getByRole('link', { name: 'CP', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'SC-LM-002', level: 1 })).toBeVisible();
	const print = page.locator('section', { has: page.getByRole('heading', { name: en.printTitle, exact: true }) });
	await print.getByRole('heading', { name: en.printTitle, exact: true }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 22', 'Bagian Print. Draft dicetak ke Excel atau PDF dan selalu bertanda UNCONTROLLED COPY.');
	let download = page.waitForEvent('download');
	await print.getByRole('button', { name: en.printExcel, exact: true }).click();
	await download;
	await expect(print.getByRole('status')).toContainText('SC-LM-002-00-draft.xlsx');
	await say(page, 'Langkah 22', 'Setiap cetakan tercatat di Print log: siapa, kapan, revisi, dan tujuannya.');

	await logOut(page);
	await say(page, 'Langkah 23', 'Dokumen kontrol login pertama kali. Ia mencatat rilis; penyusun tidak bisa merilis draftnya sendiri.');
	await firstLogin(page, control.login, controlTemporary, control.password);
	await openDocument(page, 'SC-LM-002');
	await page.getByRole('heading', { name: en.revisions, exact: true }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 23', 'Sesudah kertasnya ditandatangani, scan diunggah. Tanpa scan, rilis ditolak.');
	await uploadScan(page, 'SC-LM-002-00-signed.pdf');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') }).click();
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();
	await say(page, 'Langkah 23', 'Revisi 0 terbit. Isinya dibekukan dengan SHA-256 dan tidak bisa diubah lagi.');

	// ---------- Bagian E: perubahan dan revisi (Tahap 5) ----------
	await switchTo(page, engineer.login, engineer.password);
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await say(page, 'Langkah 24', 'Documents > Changes. Dokumen terbit hanya berubah lewat permintaan perubahan.');
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.changeTrigger, { exact: true }).selectOption({ label: 'Customer claim' });
	await page.getByLabel(en.changeType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.changeFourM, { exact: true }).selectOption({ label: 'Method' });
	await page.getByLabel(en.changeReason, { exact: true }).fill('Contoh: tambah pemeriksaan fokus laser');
	await page.getByRole('checkbox', { name: /SC-LM-002/ }).check();
	await say(page, 'Langkah 24', 'Pemicu, jenis perubahan, alasan, dan dokumen yang terdampak.');
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeCreated.replace('{ref}', 'CHG-0001'));
	await say(page, 'Langkah 24', 'Perubahan CHG-0001 diajukan dan menunggu verifikasi atasan.');

	await logOut(page);
	await say(page, 'Langkah 25', 'Verifikator login pertama kali. Pemohon tidak bisa memverifikasi permintaannya sendiri.');
	await firstLogin(page, verifier.login, verifierTemporary, verifier.password);
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('row', { name: /CHG-0001/ }).getByRole('link', { name: 'CHG-0001' }).click();
	await expect(page.getByRole('heading', { name: en.changeVerify })).toBeVisible();
	await page.getByLabel(en.changeVerifyNote).fill('Contoh: klaim sudah diperiksa, pemeriksaan fokus diperlukan');
	await say(page, 'Langkah 25', 'Catatan verifikasi wajib diisi, lalu Accept atau Reject.');
	await page.getByRole('button', { name: en.changeAccept, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeAccepted.replace('{ref}', 'CHG-0001'));
	await say(page, 'Langkah 25', 'Diterima. Dokumen terdampak mendapat draft revisi 1; revisi 0 tetap berlaku sampai revisi 1 terbit.');

	await switchTo(page, engineer.login, engineer.password);
	await openDocument(page, 'SC-LM-002');
	await say(page, 'Langkah 26', 'Penyusun mengubah isi draft revisi 1.');
	await rows.first().locator(`.tabulator-cell[tabulator-field="${product}"]`).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Laser type, daya, dan fokus');
	await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
	await expect(page.getByText(en.documentRowSaved).first()).toBeVisible();
	await page.getByRole('heading', { name: en.diffTitle }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 26', 'Compare revisions menunjukkan beda revisi 0 dan draft, baris demi baris dan sel demi sel.');

	await switchTo(page, control.login, control.password);
	await openDocument(page, 'SC-LM-002');
	await page.getByRole('heading', { name: en.revisions, exact: true }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 27', 'Dokumen kontrol mengunggah scan revisi 1 dan mencatat rilisnya.');
	await uploadScan(page, 'SC-LM-002-01-signed.pdf');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '1') }).click();
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();
	await print.getByRole('heading', { name: en.printTitle, exact: true }).scrollIntoViewIfNeeded();
	await print.getByLabel(en.printPurpose, { exact: true }).selectOption('reference');
	download = page.waitForEvent('download');
	await print.getByRole('button', { name: en.printExcel, exact: true }).click();
	await download;
	await say(page, 'Langkah 27', 'Revisi 1 berlaku. Cetakan revisi terbit memuat riwayat revisi dan tanda segitiga pada baris yang berubah.');

	// ---------- Bagian F: dokumen lama (Tahap 7) ----------
	await switchTo(page, engineer.login, engineer.password);
	await openFromMenu(page, en.navGroupDocuments, en.navNumbers);
	await expect(page.getByRole('button', { name: en.numberRegister, exact: true })).toBeVisible();
	await page.getByLabel(en.numberCustomer, { exact: true }).first().selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByRole('button', { name: en.numberRegister, exact: true }).click();
	await expect(page.getByRole('status')).toContainText('SH-LM-003');
	await say(page, 'Langkah 28', 'Nomor untuk dokumen lama. Di pabrik, nomor lama masuk lewat Import legacy numbers.');
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	const legacy = page.locator('section', { has: page.getByRole('heading', { name: en.migrateTitle }) });
	await legacy.getByRole('heading', { name: en.migrateTitle }).scrollIntoViewIfNeeded();
	await legacy.getByLabel(en.migrateNumber, { exact: true }).selectOption({ label: 'SC-LM-003 · CP' });
	await legacy.getByRole('checkbox', { name: 'K-200' }).check();
	await legacy.getByLabel(en.migrateRevision, { exact: true }).fill('4');
	await legacy.getByLabel(en.migrateEffectiveDate, { exact: true }).fill('2025-03-10');
	await legacy.getByLabel(en.migratePreparedBy, { exact: true }).fill('Contoh Penyusun Lama');
	await legacy.getByLabel(en.migrateApprovedBy, { exact: true }).fill('Contoh NPI Manager, Contoh QA Manager');
	await say(page, 'Langkah 28', 'Enter a legacy document: dokumen kertas masuk di revisinya yang sekarang, di sini revisi 4.');
	await legacy.getByRole('button', { name: en.migrateSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.migrateDone.replace('{number}', 'SC-LM-003').replace('{revision}', '4'));
	await page.getByRole('link', { name: 'SC-LM-003' }).click();
	const baseline = page.locator('section', { has: page.getByRole('heading', { name: en.migrationTitle, exact: true }) });
	await baseline.getByRole('button', { name: en.historyAdd }).click();
	await baseline.getByLabel(`${en.historyRevision} 1`).fill('-00');
	await baseline.getByLabel(`${en.historyDate} 1`).fill('2023-01-05');
	await baseline.getByLabel(`${en.historyDescription} 1`).fill('Initial release');
	await baseline.getByRole('button', { name: en.historySave }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.historySaved);
	await say(page, 'Langkah 28', 'Riwayat revisi lama disalin sebagai teks. Isi dokumen diketik atau dirakit sampai sama dengan hardcopy.');
	await uploadScan(page, 'SC-LM-003-hardcopy.pdf');
	await say(page, 'Langkah 28', 'Scan hardcopy bertanda tangan diunggah sebagai bukti.');

	await switchTo(page, control.login, control.password);
	await openDocument(page, 'SC-LM-003');
	await baseline.getByLabel(en.migrationEffort, { exact: true }).fill('25');
	await say(page, 'Langkah 29', 'Petugas migrasi lain mencocokkan layar dengan hardcopy dan mengisi menit kerjanya.');
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.migrationVerified.replace('{number}', 'SC-LM-003'));
	await say(page, 'Langkah 29', 'Status menjadi Released (migrated). Perubahan berikutnya lewat permintaan perubahan biasa.');

	await openFromMenu(page, en.navGroupDocuments, en.navDashboard);
	await expect(page.getByRole('heading', { name: en.dashboardTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 30', 'Documents > Dashboard. Angka dihitung dari data: baris yang mengikuti template, status dokumen, waktu revisi, dan usaha migrasi.');

	// ---------- Bagian G: admin memeriksa peran dan jejak (Tahap 8) ----------
	await logOut(page);
	await say(page, 'Langkah 31', 'Admin login lagi untuk memeriksa isi peran dan audit log.');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);
	await expect(page.getByRole('heading', { name: en.rolesTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 31', 'Admin > Roles. Admin mengatur hak setiap peran kerja; hak admin dan hak dasar terkunci.');
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await say(page, 'Langkah 32', 'Semua langkah tadi tercatat: siapa, kapan, dan apa. Catatan ini tidak bisa diubah.');
	await page.getByRole('button', { name: en.auditVerify }).click();
	await expect(page.getByRole('status')).toContainText(en.auditChainValid.split('{entries}')[0]);
	await say(page, 'Selesai', 'Rantai audit utuh. Alur dari master data sampai rilis, revisi, migrasi, dan dasbor selesai.');
});
