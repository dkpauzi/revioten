// Test E2E M1.6: kriteria selesai "semuanya bisa diedit", lewat layar (sejak D-209 oleh PPAP Engineer):
// kode pabrik, jenis dokumen dan awalannya, kode khusus, jenis perubahan,
// sumber perubahan, dan kategori 4M (UC-05).
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { ensureProjectGroup, logInAsMaster, openFromMenu } from './helpers.ts';

// exact: true, karena label seperti "Code" juga bagian dari "Factory code".
function field(page: Page, label: string) {
	return page.getByLabel(label, { exact: true });
}

test('PPAP Engineer mengubah kode pabrik, jenis dokumen, dan kode khusus', async ({ page }) => {
	// Master data diurus PPAP Engineer (E-11, D-209).
	await logInAsMaster(page);
	await openFromMenu(page, en.navGroupMaster, en.navDocumentTypes);
	await expect(page.getByRole('row', { name: /^PFD/ })).toContainText('SH');

	// Kode pabrik diganti: semua awalan ikut berubah. Lalu dikembalikan ke S.
	await field(page, en.factoryCode).fill('t');
	await page.getByRole('button', { name: en.factoryCodeSave }).click();
	await expect(page.getByText(en.factoryCodeSaved.replace('{code}', 'T'))).toBeVisible();
	await expect(page.getByRole('row', { name: /^PFMEA/ })).toContainText('TF');
	await field(page, en.factoryCode).fill('S');
	await page.getByRole('button', { name: en.factoryCodeSave }).click();
	await expect(page.getByRole('row', { name: /^PFMEA/ })).toContainText('SF');

	// Jenis dokumen lain dari WI. Huruf yang sudah dipakai ditolak.
	await field(page, en.documentTypeName).fill('Work Instruction');
	await field(page, en.documentTypeLetter).fill('w');
	await page.getByRole('button', { name: en.documentTypeAdd }).click();
	await expect(page.getByRole('row', { name: /^Work Instruction/ })).toContainText('SW');
	await field(page, en.documentTypeName).fill('Inspection Standard');
	await field(page, en.documentTypeLetter).fill('C');
	await page.getByRole('button', { name: en.documentTypeAdd }).click();
	await expect(page.getByText(en.documentTypeLetterTaken)).toBeVisible();

	// Nama PFD diganti; key-nya tetap, jadi tetap ditandai dokumen sistem ini.
	await page.getByRole('row', { name: /^PFD/ }).getByRole('button', { name: en.customerEdit }).click();
	await page.getByLabel(en.documentTypeName, { exact: true }).fill('Process Flow Diagram');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('row', { name: /^Process Flow Diagram/ })).toContainText(en.documentTypeBuilt);

	// Kode khusus baru.
	await field(page, en.specialCode).fill('qx');
	await field(page, en.specialCodeName).fill('Quality general');
	await page.getByRole('button', { name: en.specialCodeAdd }).click();
	await expect(page.getByRole('row', { name: /^QX/ })).toContainText('Quality general');

	// Customer tidak bisa memakai kode khusus, juga yang baru ditambah (E-37, D-114).
	await ensureProjectGroup(page, 'GR1');
	await openFromMenu(page, en.navGroupMaster, en.navCustomers);
	await field(page, en.customerCode).fill('qx');
	await field(page, en.customerName).fill('Customer QX');
	await field(page, en.customerProjectGroup).selectOption({ index: 1 });
	await page.getByRole('button', { name: en.customerSave }).click();
	await expect(page.getByText(en.customerCodeReserved)).toBeVisible();
});

test('PPAP Engineer mengubah jenis perubahan, sumber perubahan, dan kategori 4M', async ({ page }) => {
	// Master data diurus PPAP Engineer (E-11, D-209).
	await logInAsMaster(page);
	await openFromMenu(page, en.navGroupMaster, en.navChangeTypes);

	// Jenis perubahan diganti nama.
	await page.getByRole('row', { name: /^Layout only/ }).getByRole('button', { name: en.changeListRename }).click();
	await page.getByLabel(en.changeListName, { exact: true }).fill('Layout');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByText(en.changeListSaved.replace('{name}', 'Layout'))).toBeVisible();

	// Sumber perubahan baru, lalu dinonaktifkan.
	await field(page, en.changeSourceName).fill('Kaizen');
	await page.getByRole('button', { name: en.changeSourceAdd }).click();
	await expect(page.getByRole('row', { name: /^Kaizen/ })).toContainText(en.changeSourceActive);
	await page.getByRole('row', { name: /^Kaizen/ }).getByRole('button', { name: en.customerEdit }).click();
	await field(page, en.changeSourceActive).uncheck();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('row', { name: /^Kaizen/ })).toContainText(en.changeSourceInactive);

	// Sumber milik mekanisme sistem selalu aktif: kotaknya tidak bisa diubah (E-36, D-113).
	await page.getByRole('row', { name: /^Yokoten/ }).getByRole('button', { name: en.customerEdit }).click();
	await expect(field(page, en.changeSourceActive)).toBeDisabled();
	await page.getByRole('button', { name: en.masterCancel }).click();

	// Kategori 4M diganti nama.
	await page.getByRole('row', { name: /^Man / }).getByRole('button', { name: en.changeListRename }).click();
	await page.getByLabel(en.changeListName, { exact: true }).fill('Manpower');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByText(en.changeListSaved.replace('{name}', 'Manpower'))).toBeVisible();

	// Semua perubahan tersimpan di server.
	await page.reload();
	await expect(page.getByRole('cell', { name: 'Layout', exact: true })).toBeVisible();
	await expect(page.getByRole('cell', { name: 'Manpower', exact: true })).toBeVisible();
	await expect(page.getByRole('row', { name: /^Kaizen/ })).toContainText(en.changeSourceInactive);
});
