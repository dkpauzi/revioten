// Test E2E M1.5: kriteria selesai "nama proses bisa diubah tanpa memutus
// rujukan", lewat layar. Master proses dikelola PPAP Engineer, pemegang peran
// Pengelola template (E-6, D-64); akun admin hanya melihat.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime } from './helpers.ts';

// exact: true, karena "Process name" juga muncul di label lain.
function field(page: Page, label: string) {
	return page.getByLabel(label, { exact: true });
}

// processIdOf mencari ID proses menurut namanya lewat API, seperti data lain
// yang menyimpan rujukan ke proses.
async function processIdOf(page: Page, name: string): Promise<string> {
	return page.evaluate(async (name) => {
		const list = await (await fetch('/api/v1/processes')).json();
		return list.items.find((p: { name: string }) => p.name === name).id as string;
	}, name);
}

test('nama proses diganti tanpa memutus rujukan', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'PPAP Engineer E2E', 'ppap.engineer.e2e', [
		'template_maintainer'
	]);

	// Akun admin tidak memegang E-6: daftar terlihat, formulir tidak (D-64).
	await page.getByRole('link', { name: en.navProcesses }).click();
	await expect(page.getByRole('heading', { name: en.processesTitle })).toBeVisible();
	await expect(page.getByRole('heading', { name: en.processAddTitle })).toHaveCount(0);

	await page.getByRole('button', { name: en.logout }).click();
	await logInFirstTime(page, 'ppap.engineer.e2e', temporary, 'jembatan bambu di atas kali');
	await page.getByRole('link', { name: en.navProcesses }).click();

	// Proses baru dengan alias, satu alias per baris.
	await field(page, en.processName).fill('Parts  Receiving');
	await field(page, en.processAliases).fill('Part Receiving\nReceiving Area');
	await page.getByRole('button', { name: en.processAdd }).click();
	await expect(page.getByText(en.processSaved.replace('{name}', 'Parts Receiving'))).toBeVisible();
	const row = page.getByRole('row', { name: /Parts Receiving/ });
	await expect(row).toContainText('Part Receiving');
	await expect(row).toContainText('Receiving Area');

	// Satu nama hanya menunjuk satu proses, juga sebagai alias (ASUMSI E-32).
	await field(page, en.processName).fill('IQC');
	await field(page, en.processAliases).fill('parts receiving');
	await page.getByRole('button', { name: en.processAdd }).click();
	await expect(page.getByText(en.processAliasTaken)).toBeVisible();

	// Rujukan disimpan sebagai ID, lalu nama tampilannya diganti.
	const id = await processIdOf(page, 'Parts Receiving');
	await row.getByRole('button', { name: en.processEdit }).click();
	await page.getByLabel(en.processName, { exact: true }).fill('Receiving');
	await page.getByRole('button', { name: en.masterSave }).click();
	await expect(page.getByText(en.processSaved.replace('{name}', 'Receiving'))).toBeVisible();
	const renamed = page.getByRole('row', { name: /^Receiving/ });
	await expect(renamed).toContainText('Part Receiving');
	await expect(page.getByRole('cell', { name: 'Parts Receiving', exact: true })).toHaveCount(0);

	// ID yang disimpan tadi tetap menunjuk proses yang sama, dengan nama baru.
	const byId = await page.evaluate(async (id) => (await fetch(`/api/v1/processes/${id}`)).json(), id);
	expect(byId).toMatchObject({
		id,
		name: 'Receiving',
		aliases: ['Part Receiving', 'Receiving Area']
	});

	// Audit log menunjuk ID yang sama, jadi entri lama ikut menampilkan nama baru.
	await page.getByRole('button', { name: en.logout }).click();
	await logIn(page);
	await page.getByRole('link', { name: en.navAuditLog }).click();
	for (const action of [en.auditActionProcessCreated, en.auditActionProcessUpdated]) {
		await expect(page.getByRole('row', { name: new RegExp(`${action} Receiving`) })).toBeVisible();
	}
});
