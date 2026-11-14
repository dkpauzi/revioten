// Test E2E D-155 pola 2: bagian per PCB, mengikuti pola control plan nyata
// SC-KT-040-00 (D-159): satu nomor untuk model 737D LED HL, dengan bagian
// PCB A dan Heatsink Assy (R/L) yang masing-masing punya alur sendiri.
//
// Nama file diawali m4- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('CP satu nomor dengan bagian per PCB: tambah bagian, rakit per bagian, dan cek per bagian', async ({ page }) => {
	test.slow();
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	// Customer, model, dan tiga part number seperti sampul SC-KT-040-00.
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'KT', name: 'PT. Indonesia Koito', project_group_id: group.id });
		const model = await post('/api/v1/models', { customer_id: customer.id, name: '737D LED HL' });
		for (const [number, name] of [
			['10029-8A15D', 'Print Base Assy A'],
			['10067-8A1A8', 'Heatsink Assy R'],
			['10167-8A1A8', 'Heatsink Assy L']
		]) {
			await post('/api/v1/part-numbers', { model_id: model.id, number, name });
		}
	});
	const temporary = await createWorkAccount(page, 'Penyusun KT', 'penyusun.kt.e2e', ['author', 'template_maintainer', 'pfmea_cp_checker']);
	await logOut(page);
	await logInFirstTime(page, 'penyusun.kt.e2e', temporary, 'satu nomor tiga pcb dalam satu dokumen');

	// Process template kecil yang sudah terbit, dan set nomor dengan dokumen
	// yang mencakup ketiga part number.
	const cpNumber = await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const process = await post('/api/v1/processes', { name: 'ICT KT', aliases: [] });
		const template = await post('/api/v1/process-templates', { process_id: process.id });
		const versionId = template.versions[0].id as string;
		const types = (await (await fetch('/api/v1/document-types')).json()).items as { id: string; key?: string }[];
		const typeId = (key: string) => types.find((d) => d.key === key)!.id;
		const version = await post(`/api/v1/process-template-versions/${versionId}/subprocesses`, { name: 'ICT' });
		const sub = version.subprocesses[0];
		await post(`/api/v1/process-template-versions/${versionId}/points`, {
			subprocess_id: sub.id, document_type_id: typeId('pfmea'), text: 'ICT fail not detected', content: { failure_mode: 'ICT fail not detected' }
		});
		await post(`/api/v1/process-template-versions/${versionId}/points`, {
			subprocess_id: sub.id, document_type_id: typeId('cp'), text: 'ICT pass', content: { product_characteristic: 'ICT result PASS' }
		});
		await fetch(`/api/v1/process-template-versions/${versionId}/publish`, { method: 'POST' });
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'KT');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const parts = await (await fetch(`/api/v1/part-numbers?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		let cp = '';
		for (const n of set.items as { id: string; number: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', {
				document_number_id: n.id, model_ids: [models.items[0].id], part_number_ids: parts.items.map((p: { id: string }) => p.id)
			});
			if (doc.document_type.key === 'cp') cp = n.number;
		}
		return cp;
	});
	expect(cpNumber).not.toBe('');

	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KT · PT. Indonesia Koito' });
	await page.getByRole('link', { name: cpNumber }).click();
	await expect(page.getByRole('heading', { name: cpNumber, level: 1 })).toBeVisible();

	// Bagian pertama: PCB A dengan satu part number.
	await page.getByRole('button', { name: en.sectionSplit }).click();
	await page.getByLabel(en.sectionName, { exact: true }).fill('PCB A');
	await page.getByRole('checkbox', { name: /^10029-8A15D/ }).check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.sectionSaved.replace('{name}', 'PCB A'));
	// Bagian kedua: Heatsink R/L dengan dua part number. Part number PCB A tidak ditawarkan lagi.
	await page.getByRole('button', { name: en.sectionAdd }).click();
	await page.getByLabel(en.sectionName, { exact: true }).fill('Heatsink Assy (R/L)');
	await expect(page.getByRole('checkbox', { name: /^10029-8A15D/ })).toBeDisabled();
	await page.getByRole('checkbox', { name: /^10067-8A1A8/ }).check();
	await page.getByRole('checkbox', { name: /^10167-8A1A8/ }).check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.sectionSaved.replace('{name}', 'Heatsink Assy (R/L)'));

	const tabs = page.getByRole('navigation', { name: en.sectionTabs });
	await expect(tabs.getByRole('button')).toHaveCount(2);
	await expect(tabs.getByRole('button', { name: /Heatsink Assy \(R\/L\)/ })).toHaveAttribute('aria-pressed', 'true');
	await expect(page.locator('.section-title')).toHaveText('Heatsink Assy (R/L) · 10067-8A1A8, 10167-8A1A8');

	// Rakit di bagian Heatsink: baris masuk ke Heatsink saja.
	const rows = page.locator('.workbook .tabulator-row');
	await page.getByRole('button', { name: en.composeOpen }).click();
	await page.getByLabel(en.composeProcess).selectOption({ label: `ICT KT · ${en.composeVersion.replace('{version}', '1')}` });
	await page.getByRole('checkbox', { name: en.composeAll.replace('{name}', 'ICT') }).check();
	await page.getByRole('button', { name: en.composeApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.composeDone.replace('{process}', 'ICT KT'));
	await expect(rows).toHaveCount(1);
	await tabs.getByRole('button', { name: /PCB A/ }).click();
	await expect(rows).toHaveCount(0);
	// Satu baris ketik sendiri di PCB A.
	await page.getByRole('button', { name: en.documentRowAdd }).click();
	await expect(rows).toHaveCount(1);
	await tabs.getByRole('button', { name: /Heatsink/ }).click();
	await expect(rows).toHaveCount(1);

	// PFMEA di set yang sama punya bagian yang sama; pilihan bagian tetap.
	await page.getByRole('link', { name: 'PFMEA', exact: true }).click();
	await expect(page.getByRole('navigation', { name: en.sectionTabs }).getByRole('button')).toHaveCount(2);
	await expect(page.locator('.section-title')).toHaveText(/^Heatsink Assy \(R\/L\)/);
	await expect(rows).toHaveCount(1);

	// Tab Check: judul per bagian, dan tanda dicek untuk proses di bagian Heatsink.
	await page.getByRole('link', { name: en.checkTab, exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Heatsink Assy (R/L)', level: 2 })).toBeVisible();
	const heat = page.getByRole('region', { name: 'Heatsink Assy (R/L) · ICT KT' });
	await heat.getByRole('button', { name: en.checkMarkFor.replace('{process}', 'ICT KT') }).click();
	await expect(page.getByRole('status')).toHaveText(en.checkMarked.replace('{process}', 'ICT KT'));
	await expect(heat.getByTestId('check-mark')).toContainText('Penyusun KT');
});
