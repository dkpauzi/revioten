// Test E2E M1.4: kriteria selesai "relasi model ke customer berjalan", lewat
// layar: Project Group, customer beserta riwayat kodenya (E-11), group model,
// model, dan part number.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logIn } from './helpers.ts';

// exact: true, karena label seperti "Model group" juga bagian dari "Model group name".
function field(page: Page, label: string) {
	return page.getByLabel(label, { exact: true });
}

test('master customer sampai part number tersambung dan tersimpan', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navCustomers }).click();

	// Project Group lalu customer yang dialokasikan ke group itu (D-54).
	await field(page, en.projectGroupCode).fill('gr7');
	await field(page, en.projectGroupName).fill('Project Group 7');
	await page.getByRole('button', { name: en.projectGroupAdd }).click();
	await expect(page.getByText(en.projectGroupSaved.replace('{code}', 'GR7'))).toBeVisible();

	await field(page, en.customerCode).fill('ms');
	await field(page, en.customerName).fill('Master Customer');
	await field(page, en.customerProjectGroup).selectOption({ label: 'GR7 · Project Group 7' });
	await page.getByRole('button', { name: en.customerSave }).click();
	const customerRow = page.getByRole('row', { name: /Master Customer/ });
	await expect(customerRow).toContainText('GR7');

	// Ganti kode: kode lama tetap tercatat di riwayat (E-11).
	await customerRow.getByRole('button', { name: en.customerChangeCode }).click();
	await page.getByLabel(en.customerNewCode.replace('{code}', 'MS')).fill('mt');
	await customerRow.getByRole('button', { name: en.masterSave }).click();
	await expect(
		page.getByText(en.customerCodeChanged.replace('{before}', 'MS').replace('{after}', 'MT'))
	).toBeVisible();
	await expect(customerRow).toContainText('MT');
	await expect(customerRow.getByRole('cell', { name: 'MS', exact: true })).toBeVisible();

	// Group model, model di dalam group, model tanpa group, dan part number.
	await page.getByRole('link', { name: en.navModels }).click();
	await field(page, en.modelsCustomer).selectOption({ label: 'MT · Master Customer' });
	await field(page, en.modelGroupName).fill('Family A');
	await page.getByRole('button', { name: en.modelGroupAdd }).click();
	await expect(page.getByText(en.modelGroupSaved.replace('{name}', 'Family A'))).toBeVisible();

	await field(page, en.modelName).fill('X100');
	await field(page, en.modelGroup).selectOption({ label: 'Family A' });
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: 'X100 Family A' })).toBeVisible();
	await field(page, en.modelName).fill('X200');
	await page.getByRole('button', { name: en.modelAdd, exact: true }).click();
	await expect(page.getByRole('row', { name: `X200 ${en.modelGroupNone}` })).toBeVisible();

	await field(page, en.partNumberModel).selectOption({ label: 'X100' });
	await field(page, en.partNumber).fill('PN-0001');
	await field(page, en.partName).fill('Main Board');
	await page.getByRole('button', { name: en.partNumberAdd }).click();
	await expect(page.getByRole('row', { name: /X100 PN-0001 Main Board/ })).toBeVisible();

	// Nama group yang sama ditolak dalam satu customer.
	await field(page, en.modelGroupName).fill('Family A');
	await page.getByRole('button', { name: en.modelGroupAdd }).click();
	await expect(page.getByText(en.modelGroupNameTaken)).toBeVisible();

	// Ganti nama group: model ikut menunjuk group yang sama (ID tetap).
	await page.getByRole('row', { name: /^Family A/ }).getByRole('button', { name: en.modelGroupRename }).click();
	await page.getByLabel(en.modelGroupName, { exact: true }).first().fill('Family B');
	await page.getByRole('button', { name: en.masterSave }).click();
	await expect(page.getByRole('row', { name: 'X100 Family B' })).toBeVisible();

	// Muat ulang: semua datang lagi dari database.
	await page.reload();
	await field(page, en.modelsCustomer).selectOption({ label: 'MT · Master Customer' });
	await expect(page.getByRole('row', { name: 'X100 Family B' })).toBeVisible();
	await expect(page.getByRole('row', { name: /X100 PN-0001 Main Board/ })).toBeVisible();
});
