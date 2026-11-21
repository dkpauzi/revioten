// Test E2E D-207: Form Control Submission per dokumen. Memakai Control Plan
// customer SG yang diterbitkan dengan tanda tangan digital di
// m6-signatures.test.ts (berjalan lebih dulu karena urutan nama file).
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { logOut } from './helpers.ts';

test('D-207: PPAP Engineer mengisi pengiriman ke customer di form kontrol Control Plan', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill('rec.ttd.e2e');
	await page.getByLabel(en.loginPassword).fill('tanda tangan digital lima');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();

	const cpID = await page.evaluate(async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'SG');
		const docs = await (await fetch(`/api/v1/documents?customer_id=${customer.id}`)).json();
		return docs.items.find((d: { document_type: { key: string } }) => d.document_type.key === 'cp').id as string;
	});
	await page.goto(`/documents/view?id=${cpID}`);
	const panel = page.locator('section').filter({ has: page.getByRole('heading', { name: en.subTitle, level: 2 }) });
	await expect(panel).toBeVisible();

	// Kolom otomatis: revisi awal, tanda tangan digital.
	const row = panel.getByRole('row').filter({ hasText: 'First release' });
	await expect(row).toContainText(en.subSignedDigitally);
	await expect(row).toContainText(en.subRequired);

	// Kolom yang diketik.
	await row.getByRole('button', { name: en.subEditFor.replace('{revision}', '0') }).click();
	await panel.getByLabel(en.subSubmittedOn).fill('2026-11-20');
	await panel.getByLabel(en.subPackage).fill('Customer cover + CP');
	await panel.getByLabel(en.subResultLabel).selectOption('approved');
	await panel.getByLabel(en.subResultOn).fill('2026-11-21');
	await panel.getByLabel(en.subRemarkLabel).fill('MP approval');
	await panel.getByRole('button', { name: en.subSave }).click();
	await expect(page.getByRole('status').first()).toHaveText(en.subSaved.replace('{revision}', '0'));
	await expect(row).toContainText('Customer cover + CP');
	await expect(row).toContainText(en.subResult_approved);
	await expect(row).toContainText('MP approval');

	// Cetak form: tautan Excel dan PDF untuk dokumen ini saja.
	await expect(panel.getByRole('link', { name: en.subDownloadExcel })).toHaveAttribute('href', new RegExp(`${cpID}/submission-control/export\\?format=xlsx`));
	const response = await page.request.get(`/api/v1/documents/${cpID}/submission-control/export?format=xlsx`);
	expect(response.ok()).toBe(true);
	await logOut(page);
});
