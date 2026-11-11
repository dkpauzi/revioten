// Tur otomatis alur web, dari database kosong sampai nomor dokumen (M0 sampai M3.4).
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

test('alur dari admin sampai nomor dokumen', async ({ page }) => {
	const engineer = { login: 'ppap.engineer', name: 'Contoh PPAP Engineer', password: 'kopi pagi di pabrik' };

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
	await say(page, 'Langkah 4', 'Peran menentukan hak: Author (nomor), Template maintainer (template), Document control (Void).');
	await accountForm.getByLabel(en.roleAuthor, { exact: true }).check();
	await accountForm.getByLabel(en.roleTemplateMaintainer, { exact: true }).check();
	await accountForm.getByLabel(en.roleDocumentControl, { exact: true }).check();
	await accountForm.getByRole('button', { name: en.accountCreate }).click();
	const notice = page.getByRole('status');
	await expect(notice).toContainText(en.accountTemporaryPassword.replace('{login}', engineer.login));
	const temporary = (await notice.locator('code').textContent()) ?? '';
	await say(page, 'Langkah 4', 'Password sementara tampil SATU KALI. Berikan ke orangnya; ia wajib menggantinya.');

	await openFromMenu(page, en.navGroupMaster, en.navPositions);
	await say(page, 'Langkah 5', 'Master data > Positions. Catat siapa memegang jabatan PPAP Engineer.');
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'PPAP Engineer' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: engineer.name });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', engineer.name).replace('{position}', 'PPAP Engineer'))
	).toBeVisible();

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await say(page, 'Langkah 6', 'Format template sudah terisi sejak awal. Admin hanya memeriksa dan menyesuaikan labelnya.');

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

	// ---------- Bagian C: admin memeriksa jejak ----------
	await logOut(page);
	await say(page, 'Langkah 15', 'Admin login lagi untuk memeriksa audit log.');
	await logInAs(page, admin.login, admin.password);
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	await say(page, 'Langkah 15', 'Semua langkah tadi tercatat: siapa, kapan, dan apa. Catatan ini tidak bisa diubah.');
	await page.getByRole('button', { name: en.auditVerify }).click();
	await expect(page.getByRole('status')).toContainText(en.auditChainValid.split('{entries}')[0]);
	await say(page, 'Selesai', 'Rantai audit utuh. Langkah berikutnya (draft dokumen) dibangun di Tahap 4.');
});
