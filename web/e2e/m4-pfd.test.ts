// Test E2E C4 (D-211, E-136): PFD diisi lebih dulu, lalu PFMEA dan CP dirakit
// per langkah PFD dari process template. Langkah tanpa template mendapat satu
// baris "--", seperti CP Tokairika. Berjalan sesudah m4-documents.test.ts,
// yang membuat akun penyusun.dokumen.e2e (PPAP Engineer).
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { asMaster, ensureProjectGroup, openTool, tool } from './helpers.ts';

async function logInAuthor(page: Page) {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill('penyusun.dokumen.e2e');
	await page.getByLabel(en.loginPassword).fill('dokumen draft disimpan per baris');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
}

test('C4: PFD dulu, lalu PFMEA dan CP dirakit per langkah PFD', async ({ page }) => {
	test.slow();
	await logInAuthor(page);
	await asMaster(page, async (m) => {
		await ensureProjectGroup(m, 'GR1');
		await m.evaluate(async () => {
			const post = (path: string, body: object) =>
				fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
			const groups = await (await fetch('/api/v1/project-groups')).json();
			const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
			const customer = await post('/api/v1/customers', { code: 'PF', name: 'Customer PF', project_group_id: group.id });
			await post('/api/v1/models', { customer_id: customer.id, name: 'P-100' });
		});
	});

	// Process template umum "Part Receiving PF" (terbit) dan proses "Packing PF"
	// tanpa template, lalu satu set nomor dengan dokumen PFD, PFMEA, dan CP.
	const ids = await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const receiving = await post('/api/v1/processes', { name: 'Part Receiving PF', aliases: [] });
		const packing = await post('/api/v1/processes', { name: 'Packing PF', aliases: [] });
		const template = await post('/api/v1/process-templates', { process_id: receiving.id });
		const versionId = template.versions[0].id as string;
		const types = (await (await fetch('/api/v1/document-types')).json()).items as { id: string; key?: string }[];
		const typeId = (key: string) => types.find((d) => d.key === key)!.id;
		const version = await post(`/api/v1/process-template-versions/${versionId}/subprocesses`, { name: 'Receiving' });
		const sub = version.subprocesses.at(-1).id;
		for (const [key, text, content] of [
			['pfmea', 'Wrong quantity received', { failure_mode: 'Wrong quantity received' }],
			['cp', 'Check quantity', { product_characteristic: 'Quantity', control_method: 'Count against the delivery note' }],
			['cp', 'Check packing condition', { product_characteristic: 'Packing condition' }]
		] as [string, string, Record<string, unknown>][]) {
			await post(`/api/v1/process-template-versions/${versionId}/points`, { subprocess_id: sub, document_type_id: typeId(key), text, content });
		}
		await post(`/api/v1/process-template-versions/${versionId}/publish`, {});

		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'PF');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		const docs: Record<string, { id: string; format_version_id: string }> = {};
		for (const n of set.items as { id: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			docs[doc.document_type.key] = doc;
		}
		return { pfd: docs.pfd, cp: docs.cp.id, pfmea: docs.pfmea.id, receiving: receiving.id, packing: packing.id };
	});

	// PFD tanpa tombol rakit dari template; langkah berikutnya: isi PFD.
	await page.goto(`/documents/view?id=${ids.pfd.id}`);
	await expect(page.getByRole('region', { name: en.nextStepTitle })).toContainText(en.nextFillPfd);
	await expect(tool(page, en.composeOpen)).toHaveCount(0);
	await expect(tool(page, en.pfdComposeOpen)).toHaveCount(0);

	// CP dengan PFD kosong: langkah berikutnya "isi PFD dulu", dengan tautan ke PFD.
	await page.goto(`/documents/view?id=${ids.cp}`);
	const next = page.getByRole('region', { name: en.nextStepTitle });
	await expect(next).toContainText(en.nextPfdFirst);
	await expect(next.getByRole('link', { name: en.nextPfdOpen })).toHaveAttribute('href', `/documents/view?id=${ids.pfd.id}`);

	// Langkah PFD diketik: 1 Part Receiving (ada template), 2 Packing (tanpa template).
	await page.evaluate(async ({ pfd, receiving, packing }) => {
		const fields = (await (await fetch(`/api/v1/format-template-versions/${pfd.format_version_id}`)).json()).fields as { id: string; key?: string }[];
		const symbol = fields.find((f) => f.key === 'flow_symbol')!.id;
		await fetch(`/api/v1/documents/${pfd.id}/rows/batch`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				rows: [
					{ operation_number: '1', process_id: receiving, content: { [symbol]: 'process' } },
					{ operation_number: '2', process_id: packing, content: { [symbol]: 'process' } }
				]
			})
		});
	}, ids);

	// CP: langkah berikutnya sekarang rakit dari PFD.
	await page.reload();
	await expect(next).toContainText(en.nextComposePfd);
	await next.getByRole('button', { name: en.pfdComposeOpen }).click();
	const steps = page.getByTestId('pfd-steps');
	await expect(steps.getByRole('row')).toHaveCount(3);
	const receivingRow = steps.getByRole('row', { name: /Part Receiving PF/ });
	await expect(receivingRow).toContainText(en.pfdTemplateVersion.replace('{version}', '1'));
	await expect(receivingRow).toContainText(en.pfdStateToAdd.replace('{points}', '2'));
	await expect(receivingRow).toContainText(en.pfdStateToAdd.replace('{points}', '1'));
	const packingRow = steps.getByRole('row', { name: /Packing PF/ });
	await expect(packingRow).toContainText(en.pfdNoTemplate);
	await expect(packingRow).toContainText(en.pfdStateNoTemplate);
	// Pemeriksaan: kedua langkah belum ada di PFMEA dan CP.
	await expect(page.getByTestId('pfd-warnings').getByRole('listitem')).toHaveCount(4);
	await page.getByRole('button', { name: en.pfdApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.pfdDone.replace('{count}', '2'));

	// CP: dua baris Part Receiving dengan nomor langkah 1, lalu satu baris "--" langkah 2.
	const grid = page.locator('.workbook');
	const rows = grid.locator('.tabulator-row');
	const fieldOf = async (title: string) =>
		(await grid.locator('.tabulator-col', { hasText: new RegExp(`^${title}$`) }).first().getAttribute('tabulator-field')) ?? '';
	const cell = (row: number, field: string) => rows.nth(row).locator(`.tabulator-cell[tabulator-field="${field}"]`);
	await expect(rows).toHaveCount(3);
	const product = await fieldOf('Product Characteristics');
	await expect(cell(0, product)).toHaveText('Quantity');
	await expect(cell(1, product)).toHaveText('Packing condition');
	await expect(cell(2, product)).toHaveText('--');
	await expect(cell(0, '_process')).toHaveText('Part Receiving PF');
	await expect(cell(2, '_process')).toHaveText('Packing PF');

	// Panel lagi: semua langkah sudah ada, pemeriksaan bersih.
	await openTool(page, en.pfdComposeOpen);
	await expect(page.getByText(en.pfdNothing)).toBeVisible();
	await expect(page.getByText(en.pfdCheckOk)).toBeVisible();
	await expect(page.getByRole('button', { name: en.pfdApply })).toHaveCount(0);

	// PFMEA set yang sama ikut terisi: satu baris template dan satu baris "--".
	await page.getByRole('navigation', { name: en.documentSheets }).getByRole('link', { name: 'PFMEA', exact: true }).click();
	await expect(rows).toHaveCount(2);

	// Baris CP yang langkahnya tidak ada di PFD muncul di pemeriksaan pratinjau.
	await page.evaluate(async ({ cp, packing }) => {
		await fetch(`/api/v1/documents/${cp}/rows`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ operation_number: '9', process_id: packing, content: {} })
		});
	}, ids);
	await page.goto(`/documents/view?id=${ids.cp}`);
	await tool(page, en.previewOpen).click();
	await expect(page.getByTestId('preview-pfd-check')).toContainText(/Step 9 Packing PF of the .+ is not in the PFD\./);
});
