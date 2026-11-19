// Test E2E Tahap 3, M3.2: kriteria selesai "contoh IQC dengan lima sub-proses
// tersimpan", lewat layar. Process template dikelola PPAP Engineer, pemegang
// peran Pengelola template (E-6, D-64); akun admin hanya melihat. ID tetap
// sub-proses dan poin dibuktikan test Go TestRenameKeepsIDs.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

const five = ['IQC General', 'IQC MSD', 'IQC PCB', 'IQC Wireharness', 'IQC Connector'];

// ensureProcess memastikan proses master dengan nama itu ada, lewat API dari
// browser yang sudah login sebagai Pengelola template (M1.5).
async function ensureProcess(page: Page, name: string): Promise<void> {
	await page.evaluate(async (name) => {
		const list = await (await fetch('/api/v1/processes')).json();
		if (!list.items.some((p: { name: string }) => p.name === name)) {
			await fetch('/api/v1/processes', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name, aliases: [] })
			});
		}
	}, name);
}

test('PPAP Engineer menyimpan contoh IQC dengan lima sub-proses beserta poinnya', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'PPAP Engineer template', 'ppap.template.e2e', ['ppap_engineer']);
	await logOut(page);
	await logInFirstTime(page, 'ppap.template.e2e', temporary, 'pustaka proses dengan id tetap');
	await ensureProcess(page, 'IQC');

	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'IQC' });
	await expect(page.getByText(en.processTemplateNone)).toBeVisible();
	await page.getByRole('button', { name: en.processTemplateCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.processTemplateCreated.replace('{process}', 'IQC'));
	await expect(page.getByText(en.subprocessNone)).toBeVisible();

	for (const name of five) {
		await page.getByLabel(en.subprocessName, { exact: true }).fill(name);
		await page.getByRole('button', { name: en.subprocessAdd }).click();
		await expect(page.getByRole('status')).toHaveText(en.subprocessAdded.replace('{name}', name));
	}
	// Nama kembar ditolak, tanpa membedakan huruf besar dan kecil.
	await page.getByLabel(en.subprocessName, { exact: true }).fill('iqc msd');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await expect(page.getByText(en.subprocessNameTaken)).toBeVisible();

	// Poin IQC MSD untuk PFD, PFMEA, dan CP.
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'IQC MSD') }).click();
	await expect(page.getByRole('heading', { name: en.pointsTitle.replace('{name}', 'IQC MSD') })).toBeVisible();
	const points = [
		['PFD', 'Incoming inspection of MSD'],
		['PFMEA', 'Floor life exceeded before use'],
		['PFMEA', 'Damaged moisture barrier bag'],
		['CP', 'Check MSL label and humidity indicator card']
	];
	for (const [type, text] of points) {
		await page.getByLabel(en.pointDocumentType).selectOption({ label: type });
		await page.getByLabel(en.pointText, { exact: true }).fill(text);
		await page.getByRole('button', { name: en.pointAdd }).click();
		await expect(page.getByRole('status')).toHaveText(en.pointAdded);
	}

	// Poin kedua PFMEA dipindah ke atas, lalu disembunyikan: tetap di daftar.
	await page.getByRole('button', { name: en.pointMoveUp.replace('{text}', 'Damaged moisture barrier bag') }).click();
	await expect(page.getByRole('row', { name: /^1 Damaged moisture barrier bag / })).toBeVisible();
	await page.getByRole('row', { name: /^1 Damaged moisture barrier bag / }).getByRole('button', { name: en.processTemplateEdit }).click();
	await page.getByLabel(en.pointHidden, { exact: true }).check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointSaved);
	await expect(page.getByRole('row', { name: /^1 Damaged moisture barrier bag / })).toContainText(en.processTemplateYes);

	// Setelah dimuat ulang, lima sub-proses dan poinnya tetap ada menurut urutan.
	await page.reload();
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'IQC' });
	for (const [i, name] of five.entries()) {
		await expect(page.getByRole('row', { name: new RegExp(`^${i + 1} ${name} `) })).toBeVisible();
	}
	await expect(page.getByRole('row', { name: /^2 IQC MSD PFD 1 · PFMEA 2 · CP 1 / })).toBeVisible();
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'IQC MSD') }).click();
	await expect(page.getByRole('row', { name: /^2 Floor life exceeded before use / })).toBeVisible();
});

test('akun admin hanya melihat process template, dan audit log mencatat perubahannya', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await expect(page.getByText(en.processTemplateReadOnly)).toBeVisible();
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'IQC' });
	await expect(page.getByRole('row', { name: /^5 IQC Connector / })).toBeVisible();
	await expect(page.getByRole('button', { name: en.processTemplateEdit })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.subprocessAdd })).toHaveCount(0);

	// Poin tetap bisa dilihat, tanpa formulir.
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'IQC MSD') }).click();
	await expect(page.getByRole('row', { name: /^1 Check MSL label and humidity indicator card/ })).toBeVisible();
	await expect(page.getByRole('button', { name: en.pointAdd })).toHaveCount(0);

	await openFromMenu(page, en.navGroupAdmin, en.navAuditLog);
	for (const action of [en.auditActionProcessTemplateCreated, en.auditActionSubprocessAdded, en.auditActionTemplatePointChanged]) {
		await expect(page.getByRole('row', { name: new RegExp(`${action} IQC`) }).first()).toBeVisible();
	}
});
