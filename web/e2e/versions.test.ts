// Test E2E M3.4: versi template berjalan dari draft ke terbit (R-TPL-05).
// Kriteria selesai "versi terbit tidak bisa diubah" dibuktikan lewat layar
// (tombol ubah hilang) dan lewat API langsung (server menolak dengan 409).
// Penjaga di database dibuktikan test Go TestPublishedFormatGuards dan
// TestPublishedProcessGuards.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// acceptNextDialog menyetujui kotak konfirmasi berikutnya dan mengembalikan teksnya.
function acceptNextDialog(page: Page): Promise<string> {
	return new Promise((resolve) => {
		page.once('dialog', async (dialog) => {
			const message = dialog.message();
			await dialog.accept();
			resolve(message);
		});
	});
}

const optionText = (version: string, status: string) =>
	en.versionOption.replace('{version}', version).replace('{status}', status);

test('process template: terbitkan versi 1, lalu ubah di versi 2 tanpa mengubah versi 1', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'PPAP Engineer versi', 'ppap.versi.e2e', ['template_maintainer']);
	await logOut(page);
	await logInFirstTime(page, 'ppap.versi.e2e', temporary, 'versi terbit tidak bisa diubah');
	await page.evaluate(async () => {
		await fetch('/api/v1/processes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name: 'Reflow Versi', aliases: [] })
		});
	});

	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'Reflow Versi' });
	await page.getByRole('button', { name: en.processTemplateCreate }).click();
	await page.getByLabel(en.subprocessName, { exact: true }).fill('Reflow Oven');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.subprocessAdded.replace('{name}', 'Reflow Oven'));

	// Terbitkan versi 1: minta konfirmasi dulu.
	const versionSelect = page.getByLabel(en.versionChoose, { exact: true });
	await expect(versionSelect.locator('option:checked')).toHaveText(optionText('1', en.versionStatusDraft));
	const dialog = acceptNextDialog(page);
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	expect(await dialog).toBe(en.versionPublishConfirm.replace('{version}', '1'));
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));
	await expect(versionSelect.locator('option:checked')).toHaveText(optionText('1', en.versionStatusPublished));

	// Versi terbit hanya bisa dibaca: tombol tambah, ubah, dan terbitkan hilang.
	await expect(page.getByRole('row', { name: /Reflow Oven/ })).toBeVisible();
	await expect(page.getByRole('button', { name: en.subprocessAdd })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.processTemplateEdit })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') })).toHaveCount(0);

	// Layar dilewati: server tetap menolak perubahan pada versi terbit.
	const status = await page.evaluate(async () => {
		const templates = await (await fetch('/api/v1/process-templates')).json();
		const template = templates.items.find((t: { process: { name: string } }) => t.process.name === 'Reflow Versi');
		const response = await fetch(`/api/v1/process-template-versions/${template.versions[0].id}/subprocesses`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name: 'Bypass' })
		});
		return response.status;
	});
	expect(status).toBe(409);

	// Versi baru: salinan versi 1 yang bisa diubah.
	await page.getByRole('button', { name: en.versionNew }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionCreated.replace('{version}', '2').replace('{from}', '1'));
	await expect(versionSelect.locator('option:checked')).toHaveText(optionText('2', en.versionStatusDraft));
	await expect(page.getByRole('row', { name: /Reflow Oven/ })).toBeVisible();
	// Selama masih ada draft, versi baru berikutnya tidak ditawarkan.
	await expect(page.getByRole('button', { name: en.versionNew })).toHaveCount(0);

	await page.getByLabel(en.subprocessName, { exact: true }).fill('Cooling Zone');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await expect(page.getByRole('row', { name: /Cooling Zone/ })).toBeVisible();

	// Kembali ke versi 1: perubahan versi 2 tidak ada di sana.
	await versionSelect.selectOption({ label: optionText('1', en.versionStatusPublished) });
	await expect(page.getByRole('row', { name: /Reflow Oven/ })).toBeVisible();
	await expect(page.getByRole('row', { name: /Cooling Zone/ })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.subprocessAdd })).toHaveCount(0);
});

test('format template: versi terbit hanya bisa dibaca, dan versi 2 menyalin isinya', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);

	// Template baru khusus test ini, supaya template bawaan tetap draft untuk test lain.
	await page.getByLabel(en.formatName, { exact: true }).fill('PFD versi uji');
	await page.getByLabel(en.formatDocumentType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.formatCopyFrom).selectOption({ index: 1 });
	await page.getByRole('button', { name: en.formatCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatCreated.replace('{name}', 'PFD versi uji'));
	const firstRow = page.getByRole('row', { name: /^3 Flow Symbol / });
	await expect(firstRow).toBeVisible();

	const dialog = acceptNextDialog(page);
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	expect(await dialog).toBe(en.versionPublishConfirm.replace('{version}', '1'));
	await expect(page.getByRole('status')).toHaveText(en.versionPublished.replace('{version}', '1'));

	// Versi terbit: tidak ada tombol ubah kolom dan formulir tambah kolom.
	await expect(firstRow).toBeVisible();
	await expect(page.getByRole('button', { name: en.formatEdit })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.formatAddField })).toHaveCount(0);
	await expect(page.getByLabel(en.formatSymbols)).toBeDisabled();

	// Versi 2 menyalin kolomnya; label diganti di versi 2 saja.
	await page.getByRole('button', { name: en.versionNew }).click();
	await expect(page.getByRole('status')).toHaveText(en.versionCreated.replace('{version}', '2').replace('{from}', '1'));
	await firstRow.getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel(en.formatColumnLabel, { exact: true }).first().fill('Symbol');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('row', { name: /^3 Symbol / })).toBeVisible();

	await page.getByLabel(en.versionChoose, { exact: true }).selectOption({ label: optionText('1', en.versionStatusPublished) });
	await expect(page.getByRole('row', { name: /^3 Flow Symbol / })).toBeVisible();
	await expect(page.getByRole('row', { name: /^3 Symbol / })).toHaveCount(0);
});
