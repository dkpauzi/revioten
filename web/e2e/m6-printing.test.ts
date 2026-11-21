// Test E2E Tahap 6: cetak, catatan cetak, salinan terkendali, dan verifikasi.
// Berjalan sesudah m5-changes.test.ts: CP customer CH sudah terbit di revisi 0
// dan punya draft revisi 1.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { openTool } from './helpers.ts';

async function logInAs(page: Page, login: string, password: string) {
	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill(login);
	await page.getByLabel(en.loginPassword, { exact: true }).fill(password);
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
}

// cpOfCH memberi ID CP pertama customer CH yang sudah punya revisi terbit.
async function cpOfCH(page: Page): Promise<{ id: string; number: string; revisionZero: string }> {
	return page.evaluate(async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const docs = await (await fetch(`/api/v1/documents?customer_id=${customer.id}`)).json();
		for (const d of docs.items as { id: string; number: string; document_type: { key: string } }[]) {
			if (d.document_type.key !== 'cp') continue;
			const revs = await (await fetch(`/api/v1/documents/${d.id}/revisions`)).json();
			const zero = revs.items.find((r: { revision: number; status: string }) => r.revision === 0 && r.status === 'released');
			if (zero) return { id: d.id, number: d.number, revisionZero: zero.id as string };
		}
		throw new Error('CP terbit tidak ada');
	});
}

test('M6: penyusun mencetak draft Excel dan PDF, dan setiap cetakan tercatat', async ({ page }) => {
	await logInAs(page, 'pemohon.e2e', 'perubahan selalu punya alasan');
	const cp = await cpOfCH(page);
	await page.goto(`/documents/view?id=${cp.id}`);
	await openTool(page, en.printTitle);

	let download = page.waitForEvent('download');
	await page.getByRole('button', { name: en.printExcel, exact: true }).click();
	expect((await download).suggestedFilename()).toBe(`${cp.number}-01-draft.xlsx`);
	await expect(page.getByRole('status')).toHaveText(en.printDone.replace('{file}', `${cp.number}-01-draft.xlsx`));

	download = page.waitForEvent('download');
	await page.getByRole('button', { name: en.printPdf, exact: true }).click();
	const pdf = await download;
	expect(pdf.suggestedFilename()).toBe(`${cp.number}-01-draft.pdf`);
	const stream = await pdf.createReadStream();
	const chunks: Buffer[] = [];
	for await (const chunk of stream) chunks.push(chunk as Buffer);
	expect(Buffer.concat(chunks).subarray(0, 4).toString()).toBe('%PDF');

	const log = page.locator('section', { has: page.getByRole('heading', { name: en.printLog }) }).locator('tbody tr');
	await expect(log).toHaveCount(2);
	await expect(log.first()).toContainText('PDF');
});

test('M6: dokumen kontrol mencetak salinan terkendali bernomor, dan halaman verifikasi menunjukkan status', async ({ page }) => {
	await logInAs(page, 'dcc.e2e', 'rilis selalu dibekukan');
	const cp = await cpOfCH(page);
	await page.goto(`/documents/view?id=${cp.id}`);
	await openTool(page, en.printTitle);
	await page.locator('#print-revision').selectOption('0');
	await page.getByLabel(en.printPurpose, { exact: true }).selectOption('controlled');
	for (const n of [1, 2]) {
		const download = page.waitForEvent('download');
		await page.getByRole('button', { name: en.printExcel, exact: true }).click();
		expect((await download).suggestedFilename()).toBe(`${cp.number}-00-controlled-${n}.xlsx`);
	}
	await page.goto(`/verify?id=${cp.revisionZero}`);
	await expect(page.locator('[data-verify="effective"]')).toHaveText(en.verifyStatus_effective);
	await expect(page.getByText(cp.number)).toBeVisible();
});
