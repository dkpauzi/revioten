// Test E2E Tahap 5: perubahan dokumen. M5.1: satu permintaan perubahan
// menunjuk beberapa dokumen. M5.2: lampiran diunggah, diunduh, dan dilepas.
//
// Nama file diawali m5- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { asMaster, createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

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
	// Master data dibuat PPAP Engineer (E-11, D-209).
	await asMaster(page, async (m) => {
		await ensureProjectGroup(m, 'GR1');
		await m.evaluate(async () => {
			const post = (path: string, body: object) =>
				fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
			const groups = await (await fetch('/api/v1/project-groups')).json();
			const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
			const customer = await post('/api/v1/customers', { code: 'CH', name: 'Customer CH', project_group_id: group.id });
			await post('/api/v1/models', { customer_id: customer.id, name: 'C-1' });
		});
	});
	const temporary = await createWorkAccount(page, 'Pemohon perubahan', 'pemohon.e2e', ['ppap_engineer']);
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
	const affected = page.locator('section', { has: page.getByRole('heading', { name: en.changeDocuments, exact: true }) });
	await expect(affected.locator('tbody tr')).toHaveCount(2);

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
	const temporary = await createWorkAccount(page, 'APQP PPAP Controller E2E', 'verifikator.e2e', ['ppap_engineer_supervisor']);
	// Catat orangnya sebagai pemegang jabatan APQP/PPAP Controller (D-51, D-54).
	// Daftar akun dibaca admin; pemegang jabatan dicatat PPAP Engineer (D-209).
	const personId = await page.evaluate(async () => {
		const accounts = await (await fetch('/api/v1/accounts')).json();
		return accounts.items.find((a: { login: string }) => a.login === 'verifikator.e2e').person.id as string;
	});
	await asMaster(page, (m) =>
		m.evaluate(async (personId) => {
			const positions = await (await fetch('/api/v1/positions')).json();
			const controller = positions.items.find((p: { key: string }) => p.key === 'apqp_ppap_controller');
			await fetch(`/api/v1/positions/${controller.id}/holders`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ person_id: personId })
			});
		}, personId)
	);
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

test('M5.5: dokumen kontrol mencatat rilis, dan revisi terbit tidak bisa diubah', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Dokumen kontrol E2E', 'dcc.e2e', ['ppap_engineer']);
	await logOut(page);
	await logInFirstTime(page, 'dcc.e2e', temporary, 'rilis selalu dibekukan');

	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await page.getByRole('row', { name: /CHG-0001/ }).getByRole('link', { name: 'CHG-0001' }).click();
	await page.getByRole('row', { name: / CP / }).getByRole('link').click();
	await expect(page.getByRole('heading', { name: en.revisions, exact: true })).toBeVisible();

	// Sejak M6.7 rilis butuh scan bertanda tangan.
	await expect(page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') })).toBeDisabled();
	await page.getByLabel(en.scanUpload).setInputFiles({ name: 'signed.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4 signed') });
	await page.getByRole('button', { name: en.scanUploadButton }).click();
	await expect(page.getByRole('status')).toHaveText(en.scanUploaded.replace('{file}', 'signed.pdf'));
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') }).click();
	await expect(page.getByRole('status')).toContainText('0');
	await expect(page.getByText(en.documentNoDraft)).toBeVisible();
	const row = page.getByRole('row', { name: new RegExp(`Dokumen kontrol E2E`) });
	await expect(row.locator('[data-status="released"]')).toBeVisible();
	await expect(page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') })).toHaveCount(0);
});

// asUser masuk sebagai akun lain lewat API, menjalankan isi fungsi di
// halaman, lalu keluar lagi.
async function asUser<T>(page: Page, login: string, password: string, run: () => Promise<T>): Promise<T> {
	await page.evaluate(
		({ login, password }) =>
			fetch('/api/v1/session', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ login, password }) }),
		{ login, password }
	);
	const out = await page.evaluate(run);
	await page.evaluate(() => fetch('/api/v1/session', { method: 'DELETE' }));
	return out;
}

test('M5.4 dan M5.6: perubahan kedua membuka revisi 1, lalu revisi 0 dibandingkan dengan draft', async ({ page }) => {
	await page.goto('/login');
	// Pemohon mengajukan CHG-0002 untuk CP yang sudah terbit (M5.5).
	const cpId = await asUser(page, 'pemohon.e2e', 'perubahan selalu punya alasan', async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const list = await (await fetch(`/api/v1/changes?customer_id=${customer.id}`)).json();
		const first = list.items.find((c: { ref: string }) => c.ref === 'CHG-0001');
		const cp = first.documents.find((d: { document_type: { key: string } }) => d.document_type.key === 'cp');
		const sources = await (await fetch('/api/v1/change-sources')).json();
		const types = await (await fetch('/api/v1/change-types')).json();
		await fetch('/api/v1/changes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				customer_id: customer.id,
				trigger_id: sources.items.find((s: { name: string; mechanism?: string }) => s.name === 'Customer claim').id,
				change_type_id: types.items[0].id,
				reason: 'Add label check',
				document_ids: [cp.id]
			})
		});
		return cp.id as string;
	});
	await asUser(page, 'verifikator.e2e', 'verifikasi selalu dengan catatan', async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const list = await (await fetch(`/api/v1/changes?customer_id=${customer.id}`)).json();
		const second = list.items.find((c: { ref: string }) => c.ref === 'CHG-0002');
		await fetch(`/api/v1/changes/${second.id}/verification`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ decision: 'accept', note: 'Label check is needed' })
		});
	});

	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await page.evaluate(
		(id) =>
			fetch(`/api/v1/documents/${id}/rows`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ content: {} }) }),
		cpId
	);
	await page.goto(`/documents/view?id=${cpId}`);
	// M5.4: draft revisi 1 terbuka, revisi 0 tetap berlaku.
	// Baris revisi di bagian Revisions (tabel Submission control juga menyebut CHG-0002).
	const revisionsSection = page.locator('section').filter({ has: page.getByRole('heading', { name: en.revisions, level: 2 }) });
	await expect(revisionsSection.getByRole('row', { name: /CHG-0002/ })).toBeVisible();
	await page.getByLabel(en.diffFrom, { exact: true }).selectOption('0');
	await page.getByLabel(en.diffTo, { exact: true }).selectOption('draft');
	await page.getByRole('button', { name: en.diffCompare }).click();
	await expect(page.locator('table.diff [data-diff="added"]')).toHaveCount(1);
});

test('M5.7: pemohon menyimpan daftar periksa dampak dengan alasan', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await page.getByRole('row', { name: /CHG-0002/ }).getByRole('link', { name: 'CHG-0002' }).click();
	await expect(page.getByRole('heading', { name: en.impactTitle })).toBeVisible();

	// Saran awal: CP terdampak (dokumen terdampak CHG-0002), lainnya belum diputuskan.
	await expect(page.getByLabel(en.impactDecisionFor.replace('{line}', '3'))).toHaveValue('affected');
	await page.getByLabel(en.impactDecisionFor.replace('{line}', '1')).selectOption('not_affected');
	await page.getByLabel(en.impactDecisionFor.replace('{line}', '2')).selectOption('not_affected');
	await page.getByLabel(en.impactReasonFor.replace('{line}', '2')).fill('Failure mode stays the same');
	await page.getByLabel(en.impactDecisionFor.replace('{line}', '4')).selectOption('affected');
	// WI dipilih dari master dokumen terkait (D-168); SW-ZZ-103 dibuat test M4.
	const masterOption = page.getByLabel(en.impactMasterFor.replace('{line}', '4')).locator('option', { hasText: /^SW-ZZ-103 / });
	await page.getByLabel(en.impactMasterFor.replace('{line}', '4')).selectOption({ label: (await masterOption.textContent()) ?? '' });
	await page.getByLabel(en.impactDecisionFor.replace('{line}', '5')).selectOption('not_affected');
	await page.getByLabel(en.impactReasonFor.replace('{line}', '5')).fill('No new inspection item');
	// Baris 1 tidak terdampak tanpa alasan: ditolak dengan nomor barisnya.
	await page.getByRole('button', { name: en.impactSave }).click();
	await expect(page.getByRole('alert')).toHaveText(`${en.impactLine.replace('{line}', '1')}: ${en.impactReasonRequired}`);
	await page.getByLabel(en.impactReasonFor.replace('{line}', '1')).fill('Flow stays the same');
	await page.getByRole('button', { name: en.impactSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.impactSaved.replace('{ref}', 'CHG-0002'));
	// Tersimpan: dimuat ulang dengan isi yang sama.
	await page.reload();
	await expect(page.getByLabel(en.impactMasterFor.replace('{line}', '4'))).not.toHaveValue('');
	await expect(page.getByLabel(en.impactDocumentFor.replace('{line}', '4'))).toHaveCount(0);
	await expect(page.getByLabel(en.impactReasonFor.replace('{line}', '2'))).toHaveValue('Failure mode stays the same');
});

test('M5.8: versi template baru dipropagasikan, dan isinya masuk ke draft saat perubahan diterima', async ({ page }) => {
	// Perakit dari test M4 (m4-compose) memegang Pengelola template. Versi 2
	// Incoming QC mengganti metode poin "Check part number" menjadi SPI.
	await page.goto('/login');
	const version2 = await asUser(page, 'perakit.e2e', 'baris dirakit dari process template', async () => {
		const templates = await (await fetch('/api/v1/process-templates')).json();
		const tpl = templates.items.find((x: { process: { name: string } }) => x.process.name === 'Incoming QC');
		const v2 = await (await fetch(`/api/v1/process-templates/${tpl.id}/versions`, { method: 'POST' })).json();
		const point = v2.subprocesses.flatMap((s: { points: { id: string; text: string }[] }) => s.points).find((p: { text: string }) => p.text === 'Check part number');
		await fetch(`/api/v1/process-template-versions/${v2.id}/points/${point.id}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ text: 'Check part number', content: { product_characteristic: 'Part number', control_method: 'SPI' } })
		});
		await fetch(`/api/v1/process-template-versions/${v2.id}/publish`, { method: 'POST' });
		return v2.version as number;
	});
	expect(version2).toBe(2);

	await page.getByLabel(en.loginName, { exact: true }).fill('perakit.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('baris dirakit dari process template');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'Incoming QC' });
	await expect(page.getByRole('heading', { name: en.propagationTitle.replace('{version}', '2') })).toBeVisible();
	await page.getByRole('button', { name: en.propagationPreview }).click();
	// Paling sedikit satu CP dokumen KM mengikuti dan ikut propagasi.
	const cpRow = page.locator('table.preview tbody tr', { has: page.locator('td', { hasText: /^SC-KM-/ }) }).filter({ hasText: en.propagationYes });
	await expect(cpRow.first()).toBeVisible();
	await page.getByLabel(en.changeType, { exact: true }).selectOption({ index: 1 });
	await page.getByLabel(en.changeReason, { exact: true }).fill('IQC uses SPI for the part number');
	await page.getByRole('button', { name: /^Propagate to / }).click();
	await expect(page.getByRole('status')).toContainText('CHG-');
	await logOut(page);

	// Verifikator menerima; isi baru masuk ke draft CP.
	const cp = await asUser(page, 'verifikator.e2e', 'verifikasi selalu dengan catatan', async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'KM');
		const list = await (await fetch(`/api/v1/changes?customer_id=${customer.id}`)).json();
		const change = list.items.find((c: { mechanism: string }) => c.mechanism === 'template');
		const accepted = await (
			await fetch(`/api/v1/changes/${change.id}/verification`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ decision: 'accept', note: 'SPI is the new IQC method' })
			})
		).json();
		return accepted.documents.filter((d: { document_type: { key: string } }) => d.document_type.key === 'cp') as { rows_updated?: number; rows_conflict?: number }[];
	});
	// Template diterapkan ke setiap CP: baris diperbarui, atau dibiarkan karena diubah lokal.
	expect(cp.length).toBeGreaterThan(0);
	for (const d of cp) {
		expect(d.rows_updated).toBeDefined();
		expect((d.rows_updated ?? 0) + (d.rows_conflict ?? 0)).toBeGreaterThan(0);
	}
});

test('M5.9: perbaikan CHG-0001 disebarkan ke model C-2, dan statusnya terlihat per model', async ({ page }) => {
	// Model kedua customer CH dengan satu set dokumen.
	await asMaster(page, (m) =>
		m.evaluate(async () => {
			const customers = await (await fetch('/api/v1/customers')).json();
			const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
			await fetch('/api/v1/models', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ customer_id: customer.id, name: 'C-2' }) });
		})
	);
	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const c2 = models.items.find((m: { name: string }) => m.name === 'C-2');
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		for (const n of set.items as { id: string; status: string }[]) {
			if (n.status === 'reserved') await post('/api/v1/documents', { document_number_id: n.id, model_ids: [c2.id] });
		}
	});

	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await page.getByRole('row', { name: /CHG-0001/ }).getByRole('link', { name: 'CHG-0001' }).click();
	await page.getByRole('link', { name: en.yokotenSpread }).click();
	await expect(page.getByRole('heading', { name: en.yokotenNew })).toBeVisible();
	// Semua model CH: C-1 sudah di perubahan sumber, jadi hanya C-2.
	await page.getByRole('button', { name: en.yokotenSearch }).click();
	await expect(page.getByRole('row', { name: /C-2/ })).toBeVisible();
	await expect(page.getByRole('row', { name: /C-1/ })).toHaveCount(0);
	await page.getByRole('button', { name: en.yokotenCreate.replace('{count}', '1') }).click();
	await expect(page.getByRole('heading', { name: 'YKT-0001', level: 1 })).toBeVisible();
	const row = page.getByRole('row', { name: /C-2/ });
	await expect(row.locator('[data-yokoten="undecided"]')).toBeVisible();
	await page.getByRole('button', { name: en.yokotenApplyFor.replace('{model}', 'C-2') }).click();
	await expect(page.getByRole('status')).toHaveText(`C-2: ${en.yokotenDecision_apply}.`);
	await expect(row.locator('[data-yokoten="waiting_verification"]')).toBeVisible();
	await expect(row.getByRole('link', { name: /^CHG-/ })).toBeVisible();
});

test('M5.10: perubahan surut SQAM mencakup semua dokumen aktif, dan semuanya mendapat draft', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel(en.loginName, { exact: true }).fill('pemohon.e2e');
	await page.getByLabel(en.loginPassword, { exact: true }).fill('perubahan selalu punya alasan');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
	await openFromMenu(page, en.navGroupDocuments, en.navChanges);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'CH · Customer CH' });
	await page.getByRole('button', { name: en.sqamOpen }).click();
	// Tanpa rujukan dan tanggal ditolak.
	await page.getByRole('button', { name: en.sqamSubmit }).click();
	await expect(page.getByRole('alert')).toContainText(en.sqamReferenceRequired);
	await page.getByLabel(en.propagationSqam).fill('SQAM-CH rev 3');
	await page.getByLabel(en.propagationEffectiveDate).fill('2026-12-01');
	await page.locator('#sqam-type').selectOption({ index: 1 });
	await page.locator('#sqam-reason').fill('SQAM rev 3 asks for the reaction plan owner');
	await page.getByRole('button', { name: en.sqamSubmit }).click();
	// CH punya dua set (C-1 dan C-2): enam dokumen aktif.
	const status = page.getByRole('status');
	await expect(status).toContainText('CHG-');
	const ref = (await status.textContent())?.match(/CHG-\d{4}/)?.[0] ?? '';
	expect(ref).not.toBe('');
	await expect(status).toHaveText(en.sqamSubmitted.replace('{ref}', ref).replace('{count}', '6'));
	await logOut(page);

	const docs = await asUser(page, 'verifikator.e2e', 'verifikasi selalu dengan catatan', async () => {
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'CH');
		const list = await (await fetch(`/api/v1/changes?customer_id=${customer.id}`)).json();
		const change = list.items.find((c: { mechanism: string }) => c.mechanism === 'sqam');
		const accepted = await (
			await fetch(`/api/v1/changes/${change.id}/verification`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ decision: 'accept', note: 'SQAM rev 3 applies' })
			})
		).json();
		return accepted.documents as { number: string; draft_revision?: number }[];
	});
	expect(docs).toHaveLength(6);
	for (const d of docs) expect(d.draft_revision, d.number).toBeDefined();
});
