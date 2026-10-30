// Test E2E M1.8: kriteria selesai "admin bisa mengatur rute default customer
// tanpa mengubah kode", lewat layar. Juga: slot GM dipindah ke Factory Manager
// lewat master data (R-APR-15), pemegang Project Leader per Project Group
// (D-54), satu pemegang NPI Manager (D-115), dan saklar tanggal tanda tangan (D-52).
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn } from './helpers.ts';

// ensureCustomer memastikan customer dengan kode itu ada di GR1, lewat API
// dari browser yang sudah login sebagai admin.
async function ensureCustomer(page: Page, code: string): Promise<void> {
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async (code) => {
		const customers = await (await fetch('/api/v1/customers')).json();
		if (customers.items.some((c: { code: string }) => c.code === code)) {
			return;
		}
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		await fetch('/api/v1/customers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ code, name: `Customer ${code}`, project_group_id: group.id })
		});
	}, code);
}

test('admin mengatur rute default customer lewat layar', async ({ page }) => {
	await logIn(page);
	await ensureCustomer(page, 'RT');
	await page.getByRole('link', { name: en.navApprovalRoutes }).click();

	const select = page.getByLabel(en.defaultRouteChoose.replace('{code}', 'RT'));
	await expect(select.locator('option:checked')).toHaveText(en.defaultRouteNone);
	await select.selectOption({ label: 'Up to GM' });
	await expect(
		page.getByText(en.defaultRouteSaved.replace('{code}', 'RT').replace('{route}', 'Up to GM'))
	).toBeVisible();

	// Tersimpan di server: setelah dimuat ulang, pilihan tetap, dan API menyebut rutenya.
	await page.reload();
	await expect(page.getByLabel(en.defaultRouteChoose.replace('{code}', 'RT')).locator('option:checked')).toHaveText(
		'Up to GM'
	);
	const route = await page.evaluate(async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		return customers.items.find((c: { code: string }) => c.code === 'RT').default_route.name;
	});
	expect(route).toBe('Up to GM');
});

test('slot GM dipindah dari Advisor ke Factory Manager lewat master data', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navApprovalRoutes }).click();
	const section = page.locator('section', { has: page.getByRole('heading', { name: 'Up to GM', exact: true }) });
	await section.getByRole('button', { name: en.routeEdit }).click();

	const slot5 = page.getByRole('group', { name: `${en.routeSlotPositions} 5` });
	await slot5.getByLabel('Advisor').uncheck();
	await slot5.getByLabel('Factory Manager').check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByText(en.routeSaved.replace('{name}', 'Up to GM'))).toBeVisible();
	await expect(section.getByRole('row', { name: /^5 / })).toContainText('Factory Manager');
	await expect(section.getByRole('row', { name: /^5 / })).toContainText('General Mgr.');

	// Bentuk rute dijaga server: Checked tidak boleh mendahului Prepared.
	await section.getByRole('button', { name: en.routeEdit }).click();
	await page.getByRole('button', { name: en.routeSlotDown.replace('{number}', '1') }).click();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByText(en.routeSlotsOrder)).toBeVisible();
	await page.getByRole('button', { name: en.masterCancel }).click();
});

test('Project Leader dipegang satu orang per Project Group', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await createWorkAccount(page, 'Project Leader E2E', 'pl.e2e', ['author']);
	await page.getByRole('link', { name: en.navPositions }).click();

	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'Project Leader' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: 'Project Leader E2E' });
	await page.getByLabel(en.holderGroup, { exact: true }).selectOption({ index: 1 });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', 'Project Leader E2E').replace('{position}', 'Project Leader'))
	).toBeVisible();
	await expect(page.getByRole('row', { name: /^Project Leader / })).toContainText('Project Leader E2E');

	// Group yang sama tidak bisa mendapat Project Leader kedua (D-54).
	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'Project Leader' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: 'Admin E2E' });
	await page.getByLabel(en.holderGroup, { exact: true }).selectOption({ index: 1 });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(page.getByText(en.holderGroupTaken)).toBeVisible();
});

test('NPI Manager dipegang satu orang', async ({ page }) => {
	// Jabatan approver dan APQP/PPAP Controller hanya punya satu pemegang,
	// karena satu slot mencetak satu nama (E-38, D-115).
	await logIn(page);
	await createWorkAccount(page, 'NPI Manager E2E', 'npi.e2e', []);
	await page.getByRole('link', { name: en.navPositions }).click();
	await expect(page.getByRole('row', { name: /^NPI Manager / })).toContainText(en.positionSingleHolder);

	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'NPI Manager' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: 'NPI Manager E2E' });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(
		page.getByText(en.holderAdded.replace('{name}', 'NPI Manager E2E').replace('{position}', 'NPI Manager'))
	).toBeVisible();

	await page.getByLabel(en.positionName, { exact: true }).selectOption({ label: 'NPI Manager' });
	await page.getByLabel(en.holderPerson, { exact: true }).selectOption({ label: 'Admin E2E' });
	await page.getByRole('button', { name: en.holderAdd }).click();
	await expect(page.getByText(en.holderPositionTaken)).toBeVisible();
});

test('tanggal tanda tangan per slot mati, dan bisa dinyalakan admin', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navSettings }).click();
	const signDates = page.getByLabel(en.settingsSignDates);
	await expect(signDates).not.toBeChecked();

	await signDates.check();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
	await page.reload();
	await expect(page.getByLabel(en.settingsSignDates)).toBeChecked();

	// Dikembalikan ke keadaan sekarang: mati (D-52).
	await page.getByLabel(en.settingsSignDates).uncheck();
	await page.getByRole('button', { name: en.settingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.settingsSaved);
});
