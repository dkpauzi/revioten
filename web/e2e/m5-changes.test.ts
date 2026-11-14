// Test E2E Tahap 5: perubahan dokumen. M5.1: satu permintaan perubahan
// menunjuk beberapa dokumen. M5.2: lampiran diunggah, diunduh, dan dilepas.
//
// Nama file diawali m5- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// setupSet membuat customer CH dengan model C-1 dan satu set dokumen PFD,
// PFMEA, dan CP. Dipanggil sebagai akun penyusun. Hasilnya nomor ketiganya.
async function setupSet(page: Page): Promise<Record<string, string>> {
	return page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		const out: Record<string, string> = {};
		for (const n of set.items as { id: string; number: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			out[doc.document_type.key] = n.number;
		}
		return out;
	});
}

test('M5.1 dan M5.2: penyusun mengajukan perubahan dengan beberapa dokumen dan lampiran', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'CH', name: 'Customer CH', project_group_id: group.id });
		await post('/api/v1/models', { customer_id: customer.id, name: 'C-1' });
	});
	const temporary = await createWorkAccount(page, 'Pemohon perubahan', 'pemohon.e2e', ['author']);
	await logOut(page);
	await logInFirstTime(page, 'pemohon.e2e', temporary, 'perubahan selalu punya alasan');
	const numbers = await setupSet(page);

	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await expect(page.getByText(en.changeNone)).toBeVisible();

	await page.getByLabel(en.changeTrigger, { exact: true }).selectOption({ label: 'Customer claim' });
	await page.getByLabel(en.changeType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.changeFourM, { exact: true }).selectOption({ label: 'Method' });
	await page.getByLabel(en.changeReference, { exact: true }).fill('Claim 2026-031');
	await page.getByLabel(en.changeReason, { exact: true }).fill('Add AOI check for false NG');
	// Tanpa dokumen ditolak.
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.locator('fieldset .error')).toBeVisible();
	await page.getByRole('checkbox', { name: new RegExp(numbers.pfmea) }).check();
	await page.getByRole('checkbox', { name: new RegExp(numbers.cp) }).check();
	await page.getByRole('button', { name: en.changeSubmit }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeCreated.replace('{ref}', 'CHG-0001'));
	const row = page.getByRole('row', { name: /CHG-0001/ });
	await expect(row).toContainText('Add AOI check for false NG');
	await expect(row).toContainText(`${numbers.pfmea}, ${numbers.cp}`);

	// Halaman perubahan: tambah PFD, lalu lepas PFMEA.
	await row.getByRole('link', { name: 'CHG-0001' }).click();
	await expect(page.getByRole('heading', { name: 'CHG-0001', level: 1 })).toBeVisible();
	await page.getByLabel(en.changeDocumentAdd).selectOption({ label: `${numbers.pfd} · PFD` });
	await page.getByRole('button', { name: en.changeDocumentAddButton }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeDocumentAdded.replace('{number}', numbers.pfd));
	await page.getByRole('button', { name: en.changeDocumentRemoveFor.replace('{number}', numbers.pfmea) }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeDocumentRemoved.replace('{number}', numbers.pfmea));
	await expect(page.locator('table tbody tr')).toHaveCount(2);

	// M5.2: berkas yang tidak diizinkan ditolak, PDF diterima lalu diunduh utuh.
	const fileInput = page.getByLabel(en.changeAttachmentAdd);
	await fileInput.setInputFiles({ name: 'setup.exe', mimeType: 'application/octet-stream', buffer: Buffer.from('MZ') });
	await page.getByRole('button', { name: en.changeAttachmentUpload }).click();
	await expect(page.getByRole('alert')).toHaveText(en.changeAttachmentNotAllowed);
	const content = '%PDF-1.4 8D report claim 2026-031';
	await fileInput.setInputFiles({ name: '8D report.pdf', mimeType: 'application/pdf', buffer: Buffer.from(content) });
	await page.getByRole('button', { name: en.changeAttachmentUpload }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeAttachmentAdded.replace('{file}', '8D report.pdf'));
	const download = page.waitForEvent('download');
	await page.getByRole('link', { name: '8D report.pdf' }).click();
	const file = await download;
	expect(file.suggestedFilename()).toBe('8D report.pdf');
	const stream = await file.createReadStream();
	const chunks: Buffer[] = [];
	for await (const chunk of stream) chunks.push(chunk as Buffer);
	expect(Buffer.concat(chunks).toString()).toBe(content);

	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.changeAttachmentRemoveFor.replace('{file}', '8D report.pdf') }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeAttachmentRemoved.replace('{file}', '8D report.pdf'));
	await expect(page.getByText(en.changeAttachmentsNone)).toBeVisible();
});

test('M5.3: APQP/PPAP Controller menerima perubahan dengan catatan wajib', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'APQP PPAP Controller E2E', 'verifikator.e2e', ['verifier']);
	// Catat orangnya sebagai pemegang jabatan APQP/PPAP Controller (D-51, D-54).
	await page.evaluate(async () => {
		const accounts = await (await fetch('/api/v1/accounts')).json();
		const account = accounts.items.find((a: { login: string }) => a.login === 'verifikator.e2e');
		const positions = await (await fetch('/api/v1/positions')).json();
		const controller = positions.items.find((p: { key: string }) => p.key === 'apqp_ppap_controller');
		await fetch(`/api/v1/positions/${controller.id}/holders`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ person_id: account.person.id })
		});
	});
	await logOut(page);
	await logInFirstTime(page, 'verifikator.e2e', temporary, 'verifikasi selalu dengan catatan');

	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await page.getByRole('row', { name: /CHG-0001/ }).getByRole('link', { name: 'CHG-0001' }).click();
	await expect(page.getByRole('heading', { name: en.changeVerify })).toBeVisible();
	// Tanpa catatan ditolak di layar.
	await page.getByRole('button', { name: en.changeAccept, exact: true }).click();
	await expect(page.getByText(en.changeVerifyNoteRequired)).toBeVisible();
	await page.getByLabel(en.changeVerifyNote).fill('8D report checked, AOI check is needed');
	await page.getByRole('button', { name: en.changeAccept, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.changeAccepted.replace('{ref}', 'CHG-0001'));
	await expect(page.locator('[data-change-status="accepted"]')).toHaveText(en.changeStatus_accepted);
	await expect(page.getByText('APQP PPAP Controller E2E ·')).toBeVisible();
	// M5.4: kedua dokumen terdampak masih draft revisi 0, jadi perubahan masuk ke draft itu.
	await expect(page.getByRole('cell', { name: en.changeDraftMerged.replace('{revision}', '0') })).toHaveCount(2);
	// Sesudah diterima, tidak ada tombol verifikasi lagi.
	await expect(page.getByRole('heading', { name: en.changeVerify })).toHaveCount(0);
});
