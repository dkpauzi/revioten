// Tur otomatis alur web, dari database kosong sampai dokumen terbit, direvisi,
// dicetak, dokumen lama dimigrasi, dasbor, dan aplikasi lain lewat API
// integrasi (Tahap 0 sampai Tahap 9).
// Mengikuti docs/panduan/skenario-alur-web.md langkah demi langkah. Semua
// dikerjakan lewat layar (klik dan ketik), tidak lewat API, supaya yang
// terlihat sama dengan yang dikerjakan orang.
//
// Semua nama di sini contoh (CLAUDE.md bagian 7).
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { admin } from '../e2e/account.mjs';
import { logOut, openFromMenu, openTool, tool } from '../e2e/helpers.ts';

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
					'position:fixed;left:50%;top:12px;transform:translateX(-50%);z-index:2147483647;' +
					'width:min(1100px,calc(100vw - 32px));padding:16px 24px;border-radius:12px;' +
					'background:rgba(17,24,39,.96);color:#ffffff;font:500 19px/1.45 system-ui,sans-serif;' +
					'white-space:normal;overflow-wrap:anywhere;pointer-events:none;box-shadow:0 8px 28px rgba(0,0,0,.35)';
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
	await openTool(page, en.revisions);
	const revisions = page.locator('section', { has: page.getByRole('heading', { name: en.revisions, exact: true }) });
	await revisions.getByLabel(en.scanUpload).setInputFiles({ name: file, mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4 contoh scan') });
	await revisions.getByRole('button', { name: en.scanUploadButton }).click();
	await expect(revisions.getByRole('status')).toHaveText(en.scanUploaded.replace('{file}', file));
}

test('alur dari admin sampai dokumen terbit, revisi, tanda tangan digital, migrasi, dasbor, dan API integrasi', async ({ page, playwright }) => {
	// Tur ini panjang; batas waktunya mengikuti kecepatan tur (normal lebih dari 15 menit).
	test.setTimeout(({ slow: 90, normal: 60, fast: 15 }[process.env.TOUR_SPEED ?? 'normal'] ?? 60) * 60 * 1000);
	const engineer = { login: 'ppap.engineer', name: 'Contoh PPAP Engineer', password: 'kopi pagi di pabrik' };
	const control = { login: 'dok.kontrol', name: 'Contoh Dokumen Kontrol', password: 'rilis selalu dengan scan' };
	const verifier = { login: 'apqp.controller', name: 'Contoh APQP Controller', password: 'verifikasi dengan catatan' };
	// Penanda tangan digital (D-198): Project Leader GR1 dan dua Manager.
	const leader = { login: 'pl.gr1', name: 'Contoh Project Leader GR1', password: 'cek dulu baru tanda tangan' };
	const npi = { login: 'npi.manager', name: 'Contoh NPI Manager', password: 'setuju sesudah membaca' };
	const qa = { login: 'qa.manager', name: 'Contoh QA QC Manager', password: 'kualitas nomor satu' };

	// ---------- Bagian A: admin (akun dan pengaman, D-209) ----------
	await page.goto('/login');
	await say(page, 'Langkah 1', 'Admin login. Akun admin dibuat sekali lewat perintah create-admin.');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('heading', { name: en.adminHomeTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 1', 'Home admin adalah Web health. Tugas admin: akun, peran, kesehatan web, dan pengaman. Master data dikerjakan PPAP Engineer (D-209).');

	await openFromMenu(page, en.navGroupAdmin, en.navAccounts);
	await say(page, 'Langkah 2', 'Admin > Accounts. Akun kerja untuk PPAP Engineer: orang baru, jenis Work account.');
	const accountForm = page.locator('main form').first();
	await accountForm.getByLabel(en.accountPersonNew).check();
	await accountForm.getByLabel(en.accountPersonName).fill(engineer.name);
	await accountForm.getByLabel(en.accountKindWork).check();
	await accountForm.getByLabel(en.accountLogin).fill(engineer.login);
	await say(page, 'Langkah 2', 'Peran menentukan hak (D-199): PPAP Engineer menyusun, mengurus master data dan template, mencatat rilis, dan migrasi. Isi setiap peran diatur di Admin > Roles.');
	await accountForm.getByLabel(en.rolePpapEngineer, { exact: true }).check();
	await accountForm.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', engineer.login));
	const temporary = (await notice.locator('code').textContent()) ?? '';
	await say(page, 'Langkah 2', 'Password sementara tampil SATU KALI. Berikan ke orangnya; ia wajib menggantinya.');
	await say(page, 'Langkah 2', 'Dua akun lagi: PPAP Engineer kedua (mencatat rilis, juga petugas migrasi) dan PPAP Engineer Supervisor (verifikasi dan penyetujuan form set). Penyusun tidak boleh merilis atau memverifikasi pekerjaannya sendiri.');
	const controlTemporary = await createAccount(page, control.name, control.login, [en.rolePpapEngineer]);
	const verifierTemporary = await createAccount(page, verifier.name, verifier.login, [en.rolePpapEngineerSupervisor]);
	await say(page, 'Langkah 2', 'Untuk tanda tangan digital: Project Leader GR1 (cakupannya Project Group sendiri) dan dua Manager penanda tangan.');
	const leaderTemporary = await createAccount(page, leader.name, leader.login, [en.roleProjectLeader]);
	const npiTemporary = await createAccount(page, npi.name, npi.login, [en.roleManager]);
	const qaTemporary = await createAccount(page, qa.name, qa.login, [en.roleManager]);

	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	await say(page, 'Langkah 3', 'Admin > Settings hanya berisi pengaman: mode migrasi dan aturan penyetujuan form. Mode migrasi dinyalakan selama dokumen lama dimasukkan (bawaannya mati).');
	await page.getByLabel(en.settingsMigration, { exact: true }).check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
	await say(page, 'Langkah 3', 'Admin logout. Master data, template, dan nomor bukan tugas akun admin.');
	await logOut(page);

	// ---------- Bagian B: PPAP Engineer, master data dan template ----------
	await say(page, 'Langkah 4', 'PPAP Engineer login pertama kali dengan password sementara.');
	await logInAs(page, engineer.login, temporary);
	await expect(page).toHaveURL('/account/password');
	await say(page, 'Langkah 4', 'Sistem langsung meminta ganti password (minimal 8 karakter).');
	await page.getByLabel(en.passwordCurrent, { exact: true }).fill(temporary);
	await page.getByLabel(en.passwordNew, { exact: true }).fill(engineer.password);
	await page.getByLabel(en.passwordConfirm, { exact: true }).fill(engineer.password);
	await page.getByRole('button', { name: en.passwordSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.passwordChanged);
	await say(page, 'Langkah 4', 'Menu atas: Home, Documents, Template library, dan Master data.');

	await openFromMenu(page, en.navGroupMaster, en.navCustomers);
	await say(page, 'Langkah 5', 'Master data > Customers. Project group dibuat dulu, baru customer.');
	await page.getByLabel(en.projectGroupCode, { exact: true }).fill('GR1');
	await page.getByLabel(en.projectGroupName, { exact: true }).fill('Group 1');
	await page.getByRole('button', { name: en.projectGroupAdd }).click();
	await expect(page.getByText(en.projectGroupSaved.replace('{code}', 'GR1'))).toBeVisible();

	await say(page, 'Langkah 5', 'Customer: kode dua huruf (masuk ke nomor dokumen), nama, dan project group.');
	await page.getByLabel(en.customerCode, { exact: true }).fill('LM');
	await page.getByLabel(en.customerName, { exact: true }).fill('Contoh Customer LM');
	await page.getByLabel(en.customerProjectGroup, { exact: true }).selectOption({ label: 'GR1 · Group 1' });
	await page.getByRole('button', { name: en.customerSave }).click();
	await expect(page.getByRole('row', { name: /Contoh Customer LM/ })).toBeVisible();

	await openFromMenu(page, en.navGroupMaster, en.navModels);
	await say(page, 'Langkah 6', 'Master data > Models. Pilih customer, lalu buat group model dan model.');
	await page.getByLabel(en.modelsCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await page.getByLabel(en.modelGroupName, { exact: true }).fill('Seri K');
	await page.getByRole('button', { name: en.modelGroupAdd }).click();
	await expect(page.getByText(en.modelGroupSaved.replace('{name}', 'Seri K'))).toBeVisible();
	await page.getByLabel(en.modelName, { exact: true }).fill('K-125');
	await page.getByLabel(en.modelGroup, { exact: true }).selectOption({ label: 'Seri K' });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: 'K-125 Seri K' })).toBeVisible();
	await say(page, 'Langkah 6', 'Model kedua, K-200, dibuat tanpa group untuk mencoba prioritas varian nanti.');
	await page.getByLabel(en.modelName, { exact: true }).fill('K-200');
	await page.getByLabel(en.modelGroup, { exact: true }).selectOption({ index: 0 });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: 'K-200 ' + en.modelGroupNone })).toBeVisible();

	await openFromMenu(page, en.navGroupMaster, en.navPositions);
	await say(page, 'Langkah 7', 'Master data > Positions. Catat siapa memegang jabatan PPAP Engineer.');
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'PPAP Engineer' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: engineer.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', engineer.name).replace('{position}', 'PPAP Engineer'))
	).toBeVisible();

	await say(page, 'Langkah 7', 'Verifikator perubahan harus memegang jabatan: di sini APQP/PPAP Controller.');
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'APQP/PPAP Controller' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: verifier.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', verifier.name).replace('{position}', 'APQP/PPAP Controller'))
	).toBeVisible();

	await say(page, 'Langkah 7', 'Slot tanda tangan rute mengikuti jabatan: Project Leader GR1, NPI Manager, dan QA QC Manager.');
	for (const [position, person, group] of [
		['Project Leader', leader.name, 'GR1 · Group 1'],
		['NPI Manager', npi.name, ''],
		['QA QC Manager', qa.name, '']
	]) {
		await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: position });
		await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: person });
		if (group) await page.getByLabel(en.holderGroup, { exact: true }).selectOption({ label: group });
		await page.getByRole('button', { name: en.holderAdd }).click();
		await expect(page.getByText(en.holderAdded.replace('{name}', person).replace('{position}', position))).toBeVisible();
	}

	await openFromMenu(page, en.navGroupMaster, en.navApprovalRoutes);
	await say(page, 'Langkah 7', 'Master data > Approval routes. Rute default customer LM; dokumen yang dibuat sesudahnya memakai rute ini.');
	await page.getByLabel(en.defaultRouteChoose.replace('{code}', 'LM'), { exact: true }).selectOption({ label: 'Up to QA QC Manager' });
	await expect(page.getByRole('status')).toHaveText(en.defaultRouteSaved.replace('{code}', 'LM').replace('{route}', 'Up to QA QC Manager'));

	await openFromMenu(page, en.navGroupMaster, en.navDocumentSettings);
	await say(page, 'Langkah 7', 'Master data > Document settings: tanggal di slot tanda tangan, QR di cetakan, dan nomor form Form Control Submission.');
	await page.getByLabel(en.settingsSubmissionFormNumber, { exact: true }).fill('QR-ENG-24-K015 (Rev.01)');
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await say(page, 'Langkah 8', 'Format template sudah terisi sejak awal. PPAP Engineer memeriksa labelnya, lalu menerbitkan versi 1.');
	for (const name of ['PFD · PFD', 'PFMEA AIAG 4th · PFMEA', 'CP · CP']) {
		await page.getByLabel(en.formatChoose, { exact: true }).selectOption({ label: name });
		page.once('dialog', (dialog) => dialog.accept());
		await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
		await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	}
	await say(page, 'Langkah 8', 'Format PFD, PFMEA AIAG 4th, dan CP terbit. Form set baru bisa terbit sesudah formatnya terbit.');

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
	await say(page, 'Langkah 10', 'Poin PFMEA dan CP untuk sub-proses yang sama. PFD tidak punya poin template: PFD diisi per dokumen.');
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
	await say(page, 'Langkah 15', 'PPAP Engineer Supervisor login pertama kali untuk menyetujui form set (V-3, D-139).');
	await logOut(page);
	await firstLogin(page, verifier.login, verifierTemporary, verifier.password);
	await openFromMenu(page, en.navGroupTemplates, en.navFormSets);
	await say(page, 'Langkah 15', 'Template library > Form sets. Supervisor menerbitkan form set AIAG 4th.');
	const formSet = page.locator('section').filter({ has: page.getByRole('heading', { name: /^AIAG 4th set/ }) });
	page.once('dialog', (dialog) => dialog.accept());
	await formSet.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toHaveText(en.formSetPublished.replace('{version}', '1').replace('{name}', 'AIAG 4th set'));
	await say(page, 'Langkah 15', 'Dokumen baru memakai form set customer yang sudah terbit.');
	await switchTo(page, engineer.login, engineer.password);

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
	await say(page, 'Langkah 18', 'Workbook dokumen: tab PFD, PFMEA, CP, dan Check. PFD diisi lebih dulu; PFMEA dan CP dirakit dari langkah PFD.');
	await expect(page.getByRole('region', { name: en.nextStepTitle })).toContainText(en.nextPfdFirst);
	// Langkah PFD diketik per dokumen (D-211): 10 Laser marking, simbol proses.
	await page.evaluate(async () => {
		const docs = (await (await fetch('/api/v1/documents?customer_id=' + (await (await fetch('/api/v1/customers')).json()).items.find((c: { code: string }) => c.code === 'LM').id)).json()).items as { id: string; number: string }[];
		const pfd = await (await fetch(`/api/v1/documents/${docs.find((d) => d.number === 'SH-LM-002')!.id}`)).json();
		const fields = (await (await fetch(`/api/v1/format-template-versions/${pfd.format_version_id}`)).json()).fields as { id: string; key?: string }[];
		const processes = (await (await fetch('/api/v1/processes')).json()).items as { id: string; name: string }[];
		await fetch(`/api/v1/documents/${pfd.id}/rows`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ operation_number: '10', process_id: processes.find((p) => p.name === 'Laser marking')!.id, content: { [fields.find((f) => f.key === 'flow_symbol')!.id]: 'process' } })
		});
	});
	await say(page, 'Langkah 18', 'PFD SH-LM-002 berisi satu langkah: 10 Laser marking. Tidak ada template untuk PFD.');
	await page.reload();
	await expect(page.getByRole('region', { name: en.nextStepTitle })).toContainText(en.nextComposePfd);
	await openTool(page, en.pfdComposeOpen);
	await say(page, 'Langkah 18', 'Rakit dari PFD: setiap langkah PFD dengan process template prosesnya, dan baris yang akan didapat PFMEA dan CP.');
	await page.getByRole('button', { name: en.pfdApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.pfdDone.replace('{count}', '1'));
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
	await tool(page, en.compareOpen).click();
	await expect(page.locator('.compare tbody tr').first()).toContainText(en.compareDifferent);
	await say(page, 'Langkah 19', 'Compare with template: baris ini berbeda dari template, kolom per kolom.');

	await page.getByRole('link', { name: en.checkTab, exact: true }).click();
	await expect(page.getByRole('heading', { name: en.checkTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 20', 'Tab Check: PFMEA dan CP berdampingan per proses. PPAP Engineer memeriksa, lalu memberi tanda.');
	await page.getByRole('button', { name: en.checkMarkFor.replace('{process}', 'Laser marking') }).click();
	await expect(page.getByRole('status')).toHaveText(en.checkMarked.replace('{process}', 'Laser marking'));
	await say(page, 'Langkah 20', 'Tanda menyimpan siapa dan kapan. Bila barisnya berubah lagi, tanda meminta dicek ulang.');

	await page.getByRole('link', { name: 'PFD', exact: true }).click();
	await tool(page, en.diagramOpen).click();
	await expect(page.getByTestId('pfd-diagram').locator('.svelte-flow__node')).toHaveCount(1);
	await say(page, 'Langkah 21', 'Diagram PFD digambar otomatis dari baris PFD, menurut urutannya.');
	// ---------- Bagian D: cetak dan rilis (Tahap 6) ----------
	await page.getByRole('link', { name: 'CP', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'SC-LM-002', level: 1 })).toBeVisible();
	// Pratinjau CP (C2, D-122): per halaman seperti cetakan, Process Flow dari PFD.
	await tool(page, en.previewOpen).click();
	await expect(page.getByRole('group', { name: /^Page 1 of \d+$/ })).toBeVisible();
	await say(page, 'Langkah 21', 'Preview: CP tampil per halaman seperti cetakannya. Kolom Process Flow digambar dari PFD, dengan konektor A, B, C di batas halaman yang ikut bergeser bila baris berubah.');
	await tool(page, en.previewOpen).click();
	await openTool(page, en.printTitle);
	const print = page.locator('section', { has: page.getByRole('heading', { name: en.printTitle, exact: true }) });
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
	await openTool(page, en.revisions);
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
	await say(page, 'Langkah 25', 'Verifikator login. Pemohon tidak bisa memverifikasi permintaannya sendiri.');
	await logInAs(page, verifier.login, verifier.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
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
	await openTool(page, en.revisions);
	await page.getByRole('heading', { name: en.diffTitle }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 26', 'Compare revisions menunjukkan beda revisi 0 dan draft, baris demi baris dan sel demi sel.');

	await switchTo(page, control.login, control.password);
	await openDocument(page, 'SC-LM-002');
	await openTool(page, en.revisions);
	await say(page, 'Langkah 27', 'Dokumen kontrol mengunggah scan revisi 1 dan mencatat rilisnya.');
	await uploadScan(page, 'SC-LM-002-01-signed.pdf');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '1') }).click();
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();
	await openTool(page, en.printTitle);
	await print.getByLabel(en.printPurpose, { exact: true }).selectOption('reference');
	download = page.waitForEvent('download');
	await print.getByRole('button', { name: en.printExcel, exact: true }).click();
	await download;
	await say(page, 'Langkah 27', 'Revisi 1 berlaku. Cetakan revisi terbit memuat riwayat revisi dan tanda segitiga pada baris yang berubah.');

	// ---------- Bagian F: dokumen lama (Tahap 7) ----------
	await switchTo(page, engineer.login, engineer.password);
	await openFromMenu(page, en.navGroupDocuments, en.navLegacy);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await expect(page.getByTestId('migration-ready')).toBeVisible();
	await say(page, 'Langkah 28', 'Documents > Legacy documents: satu menu untuk migrasi. Di atas, pemeriksaan kesiapan: mode migrasi, form set terbit, model, dan proses.');
	await page.getByRole('checkbox', { name: en.legacyOneSet }).uncheck();
	const cpRow = page.getByTestId('migration-set').locator('tr').filter({ has: page.getByRole('rowheader', { name: 'CP', exact: true }) });
	const cpNumber = cpRow.getByLabel(en.legacyNumberOf.replace('{type}', 'CP'));
	await cpNumber.fill('SC-LM-003-04');
	await cpNumber.blur();
	await cpRow.getByLabel(en.legacyDateOf.replace('{type}', 'CP')).fill('2025-03-10');
	await cpRow.getByLabel(en.legacyPreparedOf.replace('{type}', 'CP')).fill('Contoh Penyusun Lama');
	await cpRow.getByLabel(en.legacyApprovedOf.replace('{type}', 'CP')).fill('Contoh NPI Manager, Contoh QA Manager');
	await page.getByRole('checkbox', { name: 'K-200' }).check();
	await say(page, 'Langkah 28', 'Nomor diketik seperti di hardcopy: SC-LM-003-04 menjadi nomor SC-LM-003 dan revisi 4. Bila PFD, PFMEA, dan CP satu nomor berjalan, cukup satu nomor dengan Satu set tercentang.');
	await page.getByRole('button', { name: en.legacySetSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.legacySetDone.replace('{numbers}', 'SC-LM-003'));
	await say(page, 'Langkah 28', 'Nomor yang belum ada di register langsung didaftarkan berstatus Released. Papan di bawah menunjukkan status setiap dokumen lama.');
	await page.getByRole('link', { name: 'SC-LM-003' }).click();
	await openTool(page, en.migrationTitle);
	const baseline = page.locator('section', { has: page.getByRole('heading', { name: en.migrationTitle, exact: true }) });
	await baseline.getByRole('button', { name: en.historyAdd }).click();
	await baseline.getByLabel(`${en.historyRevision} 1`).fill('-00');
	await baseline.getByLabel(`${en.historyDate} 1`).fill('2023-01-05');
	await baseline.getByLabel(`${en.historyDescription} 1`).fill('Initial release');
	await baseline.getByRole('button', { name: en.historySave }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.historySaved);
	await say(page, 'Langkah 28', 'Riwayat revisi lama disalin sebagai teks. Isi dokumen diketik atau dirakit sampai sama dengan hardcopy.');
	await say(page, 'Langkah 28', 'Scan hardcopy boleh diunggah sebagai bukti tambahan, tetapi tidak wajib. Yang wajib: verifikasi orang kedua.');

	await switchTo(page, control.login, control.password);
	await openDocument(page, 'SC-LM-003');
	await openTool(page, en.migrationTitle);
	await baseline.getByLabel(en.migrationEffort, { exact: true }).fill('25');
	await say(page, 'Langkah 29', 'Petugas migrasi lain mencocokkan layar dengan hardcopy dan mengisi menit kerjanya.');
	page.once('dialog', (dialog) => dialog.accept());
	await baseline.getByRole('button', { name: en.migrationVerify }).click();
	await expect(baseline.getByRole('status')).toHaveText(en.migrationVerified.replace('{number}', 'SC-LM-003'));
	await say(page, 'Langkah 29', 'Status menjadi Released (migrated). Perubahan berikutnya lewat permintaan perubahan biasa.');

	await openFromMenu(page, en.navGroupDocuments, en.navDashboard);
	await expect(page.getByRole('heading', { name: en.dashboardTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 30', 'Documents > Dashboard. Angka dihitung dari data: baris yang mengikuti template, status dokumen, waktu revisi, dan usaha migrasi.');

	// ---------- Bagian F2: dokumen terbit, review berkala, dan kinerja (D-195 sampai D-197) ----------
	await openFromMenu(page, en.navGroupDocuments, en.navReleasedDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'LM · Contoh Customer LM' });
	await expect(page.getByRole('link', { name: /^SC-LM-003/ })).toBeVisible();
	await say(page, 'Langkah 31', 'Documents > Released documents. Semua akun, juga Viewer (internal audit, QA, produksi), membaca revisi yang berlaku. Draft tidak pernah tampil.');

	await openFromMenu(page, en.navGroupDocuments, en.navPeriodicReviews);
	await page.getByRole('button', { name: en.reviewTabAll, exact: true }).click();
	await say(page, 'Langkah 32', 'Documents > 3-year review. Jam dihitung per dokumen: kuning 90 hari sebelum jatuh tempo, merah sesudahnya. Peringatan juga di Home, dasbor, halaman dokumen, dan email. Model EOL tidak ikut.');

	await logOut(page);
	await logInAs(page, engineer.login, engineer.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupDocuments, en.navPerformance);
	await expect(page.getByRole('heading', { name: en.performanceReceived, exact: true })).toBeVisible();
	await say(page, 'Langkah 33', 'Documents > PPAP performance. PPAP Engineer, supervisor, dan manager melihat angka per minggu (Jumat sampai Kamis), bulan, dan tahun, lalu mengunduhnya sebagai Excel.');

	// ---------- Bagian F3: tanda tangan digital, Form Control Submission, laporan masalah ----------
	await openDocument(page, 'SF-LM-002');
	await openTool(page, en.sigTitle);
	const signing = page.locator('section', { has: page.getByRole('heading', { name: en.sigTitle, level: 2 }) });
	await signing.getByRole('heading', { name: en.sigSendTitle }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 34', 'PFMEA SF-LM-002 dikirim untuk tanda tangan digital. Process. Eng: PPAP Engineer sendiri; Engineer: Project Leader GR1. NPI dan QA QC Manager terisi otomatis.');
	await page.getByLabel(en.sigSlotChoose.replace('{slot}', 'Process. Eng'), { exact: true }).selectOption({ label: `${engineer.name} · PPAP Engineer` });
	await page.getByLabel(en.sigSlotChoose.replace('{slot}', 'Engineer'), { exact: true }).selectOption({ label: `${leader.name} · Project Leader GR1` });
	await page.locator('#sig-password').fill(engineer.password);
	await say(page, 'Langkah 34', 'Password diketik ulang. Isi draft dibekukan dengan SHA-256, dan tanggal revisi menjadi tanggal hari ini.');
	await signing.getByRole('button', { name: en.sigSend }).click();
	await expect(page.getByText(en.sigLockedWaiting)).toBeVisible();
	await say(page, 'Langkah 34', 'Draft terkunci. Penanda tangan berikutnya mendapat email dan tugas di Home.');

	for (const [who, temporary, slot] of [
		[leader, leaderTemporary, 'Engineer'],
		[npi, npiTemporary, 'NPI Manager'],
		[qa, qaTemporary, 'QA / QC Mgr.']
	] as const) {
		await logOut(page);
		await firstLogin(page, who.login, temporary, who.password);
		await openFromMenu(page, en.navGroupDocuments, en.navSignatures);
		await page.getByRole('link', { name: en.sigOpenSign }).first().click();
		await expect(page.getByRole('heading', { name: en.signAs.replace('{slot}', slot) })).toBeVisible();
		await say(page, 'Langkah 35', `${who.name} membuka Documents > Signatures, membaca, mengetik password, lalu menandatangani slot ${slot}. Urutannya tidak bisa dilompati.`);
		await page.locator('#sign-password').fill(who.password);
		await page.getByRole('button', { name: en.signButton, exact: true }).click();
		await expect(page.getByRole('status').first()).toHaveText(en.signDone.replace('{slot}', slot));
	}

	await switchTo(page, control.login, control.password);
	await openDocument(page, 'SF-LM-002');
	await openTool(page, en.revisions);
	await expect(page.getByText(en.revisionSignedDigitally)).toBeVisible();
	await say(page, 'Langkah 35', 'Semua slot sudah ditandatangani. PPAP Engineer lain mencatat rilis tanpa scan.');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') }).click();
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();

	await openTool(page, en.subTitle);
	const submissionPanel = page.locator('section', { has: page.getByRole('heading', { name: en.subTitle, level: 2 }) });
	await submissionPanel.getByRole('heading', { name: en.subTitle, level: 2 }).scrollIntoViewIfNeeded();
	await say(page, 'Langkah 36', 'Bagian Submission control: satu daftar per dokumen. Tanggal mulai, selesai, dan tanda tangan terisi sendiri.');
	await submissionPanel.getByRole('button', { name: en.subEditFor.replace('{revision}', '0') }).click();
	await submissionPanel.getByLabel(en.subSubmittedOn, { exact: true }).fill('2026-11-20');
	await submissionPanel.getByLabel(en.subPackage, { exact: true }).fill('Customer cover + PFMEA');
	await submissionPanel.getByLabel(en.subResultLabel, { exact: true }).selectOption('approved');
	await submissionPanel.getByLabel(en.subResultOn, { exact: true }).fill('2026-11-21');
	await submissionPanel.getByLabel(en.subRemarkLabel, { exact: true }).fill('Contoh: disetujui customer untuk MP');
	await say(page, 'Langkah 36', 'Yang diketik hanya pengiriman ke customer, hasilnya, dan catatan. Bukti persetujuan customer boleh diunggah.');
	await submissionPanel.getByRole('button', { name: en.subSave }).click();
	await expect(page.getByText(en.subSaved.replace('{revision}', '0'))).toBeVisible();
	await say(page, 'Langkah 36', 'Download PDF atau Excel mencetak Form Control Submission dokumen ini saja, dengan nomor form dari Settings.');

	await page.getByRole('button', { name: en.reportProblem }).click();
	const report = page.getByRole('dialog', { name: en.reportProblemTitle });
	await report.getByLabel(en.reportProblemText).fill('Contoh laporan: kolom Package terlalu sempit di layar HP');
	await say(page, 'Langkah 36', 'Report a problem di bilah atas: setiap pengguna bisa melapor. Alamat halaman ikut terkirim ke admin.');
	await report.getByRole('button', { name: en.reportProblemSend }).click();
	await expect(report).toBeHidden();

	// ---------- Bagian G: admin memeriksa peran dan jejak (Tahap 8) ----------
	await logOut(page);
	await say(page, 'Langkah 37', 'Admin login lagi untuk memeriksa isi peran dan audit log.');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('heading', { name: en.adminHomeTitle, level: 1 })).toBeVisible();
	await expect(page.getByText('Contoh laporan: kolom Package terlalu sempit di layar HP')).toBeVisible();
	await say(page, 'Langkah 37', 'Home akun admin adalah Web health: tanda tangan tertahan, error server, laporan masalah, dokumen yang diproses, email, akun, backup, dan audit log. Isi dokumen tidak tampil.');
	await openFromMenu(page, en.navGroupAdmin, en.navRoles);
	await expect(page.getByRole('heading', { name: en.rolesTitle, level: 1 })).toBeVisible();
	await say(page, 'Langkah 37', 'Admin > Roles. Admin mengatur hak setiap peran kerja; hak admin dan hak dasar terkunci.');
	await openFromMenu(page, en.navGroupAdmin, en.navServiceAccounts);
	await say(page, 'Langkah 38', 'Admin > Service accounts. Aplikasi lain (misalnya MES) mendapat token dengan scope dan customer terbatas.');
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
	const token = (await page.locator('.issued code').textContent()) ?? '';
	await say(page, 'Langkah 38', 'Token tampil SATU KALI dan disalin ke clipboard. Server hanya menyimpan hash-nya.');
	const app = await playwright.request.newContext({ baseURL: new URL(page.url()).origin, extraHTTPHeaders: { Authorization: `Bearer ${token}` } });
	const docs = await (await app.get('/integration/v1/documents')).json();
	await app.dispose();
	expect(docs.items.length).toBeGreaterThan(0);
	await say(page, 'Langkah 38', `Aplikasi itu membaca ${docs.items.length} dokumen terbit customer LM lewat /integration/v1. Draft tidak pernah ikut.`);

	await page.goto('/account/password');
	await say(page, 'Langkah 39', 'Setiap orang mengisi alamat email sendiri untuk notifikasi (atau diisi admin di Accounts).');
	await page.getByLabel(en.accountEmail, { exact: true }).fill('admin@contoh.test');
	await page.getByRole('button', { name: en.accountEmailSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.accountEmailSaved);
	await openFromMenu(page, en.navGroupAdmin, en.navNotifications);
	await say(page, 'Langkah 39', 'Admin > Notifications. Email dikirim saat ada yang menunggu seseorang. Server email (Microsoft 365) diatur IT; sebelum itu email hanya tercatat.');
	await say(page, 'Langkah 39', 'Kotak keluar menunjukkan setiap email: perubahan untuk diverifikasi, keputusan untuk pemohon, scan siap rilis, dan lainnya.');

	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await say(page, 'Langkah 40', 'Semua langkah tadi tercatat: siapa, kapan, dan apa. Catatan ini tidak bisa diubah.');
	await page.getByRole('button', { name: en.auditVerify }).click();
	await expect(page.getByRole('status')).toContainText(en.auditChainValid.split('{entries}')[0]);
	await say(page, 'Selesai', 'Rantai audit utuh. Alur dari master data sampai rilis, revisi, migrasi, dasbor, dan API integrasi selesai.');
});
