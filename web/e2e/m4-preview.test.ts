// Test E2E C2 (D-122, G11): pratinjau lembar CP dengan kolom Process Flow yang
// digambar dari PFD. Datanya mengikuti alur PFD SH-TR-020-02 (customer
// Tokairika, model D660A): langkah bertingkat 5.1 sampai 5.7.2, inspeksi
// dengan cabang NG ke SCRAP atau NCP. Berjalan sesudah m4-documents.test.ts,
// yang membuat akun penyusun.dokumen.e2e.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { asMaster, ensureProjectGroup, openTool, tool } from './helpers.ts';

// Alur PFD SH-TR-020-02: nomor langkah, nama, simbol, dan arah NG.
const flow: [string, string, string, string][] = [
	['1', 'Part Receiving', 'process', ''],
	['2', 'IQC', 'decision', 'ncp'],
	['3', 'Material Storage', 'process', ''],
	['4', 'Kitting', 'process', ''],
	['5.1', 'Loading PCB', 'process', ''],
	['5.2', 'Cleaning', 'process', ''],
	['5.3', 'Solder Paste Printing', 'process', ''],
	['5.4', 'SPI', 'decision', 'scrap'],
	['5.5', 'SMT', 'process', ''],
	['5.6', 'Reflow', 'process', ''],
	['5.7.1', 'AOI', 'decision', 'scrap'],
	['5.7.2', 'VMI', 'decision', 'scrap'],
	['6.1', 'Router', 'process', 'scrap'],
	['6.2', 'Dust removal', 'process', ''],
	['7', 'ICT', 'decision', 'scrap'],
	['8', 'LED Lighting Test', 'decision', 'scrap'],
	['9', 'Final Visual Inspection', 'decision', 'scrap'],
	['10', 'OQC', 'decision', 'ncp'],
	['11', 'Delivery', 'process', '']
];

async function logInAuthor(page: Page) {
	await page.goto('/login');
	await page.getByLabel(en.loginName).fill('penyusun.dokumen.e2e');
	await page.getByLabel(en.loginPassword).fill('dokumen draft disimpan per baris');
	await page.getByRole('button', { name: en.loginSubmit }).click();
	await expect(page.getByRole('button', { name: en.logout })).toBeVisible();
}

test('C2: pratinjau CP per halaman, Process Flow dari PFD, dan konektor A, B di batas halaman', async ({ page }) => {
	test.slow();
	await logInAuthor(page);
	await asMaster(page, async (m) => {
		await ensureProjectGroup(m, 'GR1');
		await m.evaluate(async () => {
			const post = (path: string, body: object) =>
				fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
			const groups = await (await fetch('/api/v1/project-groups')).json();
			const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
			const customer = await post('/api/v1/customers', { code: 'TR', name: 'PT. TOKAIRIKA INDONESIA', project_group_id: group.id });
			await post('/api/v1/models', { customer_id: customer.id, name: 'D660A' });
		});
	});

	// Satu set nomor, dokumen PFD dan CP, lalu baris PFD (alur) dan CP (tiga baris per langkah).
	const cpId = await page.evaluate(async (flow) => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'TR');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		const docs: Record<string, { id: string; format_version_id: string }> = {};
		for (const n of set.items as { id: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			docs[doc.document_type.key] = doc;
		}
		const fieldsOf = async (id: string) => (await (await fetch(`/api/v1/format-template-versions/${id}`)).json()).fields as { id: string; key?: string; kind: string; area: string; hidden: boolean }[];
		const pfdFields = await fieldsOf(docs.pfd.format_version_id);
		const symbol = pfdFields.find((f) => f.key === 'flow_symbol')!.id;
		const ng = pfdFields.find((f) => f.key === 'ng_disposition')!.id;
		await post(`/api/v1/documents/${docs.pfd.id}/rows/batch`, {
			rows: flow.map(([op, , sym, disposition]) => ({ operation_number: op, content: disposition ? { [symbol]: sym, [ng]: disposition } : { [symbol]: sym } }))
		});
		const cpText = (await fieldsOf(docs.cp.format_version_id)).filter((f) => f.area === 'row' && !f.hidden && f.kind === 'text').map((f) => f.id);
		const rows = flow.flatMap(([op, name]) =>
			[1, 2, 3].map((n) => ({ operation_number: op, content: { [cpText[0]]: `${name} check ${n}`, [cpText[1]]: 'Visual, according to the work instruction' } }))
		);
		await post(`/api/v1/documents/${docs.cp.id}/rows/batch`, { rows });
		return docs.cp.id;
	}, flow);

	await page.goto(`/documents/view?id=${cpId}`);
	await tool(page, en.previewOpen).click();
	await expect(page.getByRole('heading', { name: en.sheetPreviewTitle, level: 2 })).toBeVisible();
	const pages = page.getByRole('group', { name: /^Page \d+ of \d+$/ });
	await expect(pages.first()).toBeVisible();
	const total = await pages.count();
	expect(total).toBeGreaterThanOrEqual(2);

	// Halaman 1: alur mulai di langkah 1 dan berlanjut lewat konektor A.
	const first = pages.nth(0);
	await expect(first.locator('[data-step="1"]')).toHaveCount(1);
	await expect(first.locator('[data-step="2"]')).toHaveCount(1);
	await expect(first.locator('circle[data-connector="A"]')).toHaveCount(1);
	await expect(first.getByText('NCP', { exact: true })).toBeVisible();
	// Halaman 2 dimulai dari konektor A yang sama.
	await expect(pages.nth(1).locator('circle[data-connector="A"]')).toHaveCount(1);
	// Setiap langkah PFD muncul tepat sekali di seluruh pratinjau.
	for (const [op] of flow) {
		await expect(page.locator(`[data-testid="process-flow"] [data-step="${op}"]`)).toHaveCount(1);
	}

	// Excel memakai rencana yang sama (C3): berkasnya berisi gambar alur (drawing).
	await openTool(page, en.printTitle);
	const download = page.waitForEvent('download');
	await page.getByRole('button', { name: en.printExcel, exact: true }).click();
	const file = await download;
	const stream = await file.createReadStream();
	const chunks: Buffer[] = [];
	for await (const chunk of stream) chunks.push(chunk as Buffer);
	expect(Buffer.concat(chunks).includes('xl/drawings/drawing1.xml')).toBe(true);

	// PFD set yang sama: isinya hanya alur proses, digambar di kolom Flow Symbol.
	await page.getByRole('navigation', { name: en.documentSheets }).getByRole('link', { name: 'PFD', exact: true }).click();
	await tool(page, en.previewOpen).click();
	await expect(pages.first()).toBeVisible();
	await expect(page.locator('[data-testid="process-flow"] [data-step]')).toHaveCount(flow.length);
	await expect(page.locator('[data-testid="process-flow"]').getByText('SCRAP', { exact: true }).first()).toBeVisible();
	await page.goto(`/documents/view?id=${cpId}`);

	// Baris baru di CP: pratinjau dihitung ulang sendiri, tanpa menggeser gambar.
	await page.evaluate(async (id) => {
		const rows = Array.from({ length: 80 }, () => ({ operation_number: '12', content: {} }));
		await fetch(`/api/v1/documents/${id}/rows/batch`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ rows }) });
	}, cpId);
	await page.reload();
	await tool(page, en.previewOpen).click();
	await expect(pages.first()).toBeVisible();
	expect(await pages.count()).toBeGreaterThan(total);
	await expect(page.locator(`[data-testid="process-flow"] [data-step="12"]`)).toHaveCount(1);
	await expect(page.locator('circle[data-connector="B"]').first()).toBeAttached();
});

// D-227: lompatan PFD. AOI (5.7.1) OK langsung ke Router (6.1), Finish 2nd
// side (6.2) No kembali ke Loading PCB (5.1), dan satu lompatan ke langkah
// yang tidak ada. Berjalan sesudah test C2, yang membuat customer TR.
test('D-227: lompatan di Process Flow CP dan PFD, huruf konektor tidak ganda, peringatan', async ({ page }) => {
	test.slow();
	await logInAuthor(page);
	const ids = await page.evaluate(async (flow) => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'TR');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		const docs: Record<string, { id: string; format_version_id: string }> = {};
		for (const n of set.items as { id: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			docs[doc.document_type.key] = doc;
		}
		const fields = (await (await fetch(`/api/v1/format-template-versions/${docs.pfd.format_version_id}`)).json()).fields as { id: string; key?: string }[];
		const key = (k: string) => fields.find((f) => f.key === k)!.id;
		const jumps: Record<string, [string, string]> = { '5.7.1': ['6.1', 'ok'], '6.2': ['5.1', 'no'], '9': ['99', 'ok'] };
		await post(`/api/v1/documents/${docs.pfd.id}/rows/batch`, {
			rows: flow.map(([op, , sym, disposition]) => {
				const content: Record<string, string> = { [key('flow_symbol')]: op === '6.2' ? 'decision' : sym };
				if (disposition && op !== '5.7.1') content[key('ng_disposition')] = disposition;
				if (jumps[op]) {
					content[key('jump_to')] = jumps[op][0];
					content[key('jump_label')] = jumps[op][1];
				}
				return { operation_number: op, content };
			})
		});
		await post(`/api/v1/documents/${docs.cp.id}/rows/batch`, { rows: flow.map(([op]) => ({ operation_number: op, content: {} })) });
		return { cp: docs.cp.id, pfd: docs.pfd.id };
	}, flow);

	await page.goto(`/documents/view?id=${ids.cp}`);
	await tool(page, en.previewOpen).click();
	const pages = page.getByRole('group', { name: /^Page \d+ of \d+$/ });
	await expect(pages.first()).toBeVisible();
	const svg = page.locator('[data-testid="process-flow"]');
	await expect(svg.locator('[data-jump-label="OK"]').first()).toBeAttached();
	await expect(svg.locator('[data-jump-label="No"]').first()).toBeAttached();
	await expect(svg.locator('[data-down="Yes"]').first()).toBeAttached();
	// Setiap huruf konektor dipakai tepat untuk satu pasang.
	const letters = await svg.locator('circle[data-connector]').evaluateAll((els) => els.map((e) => e.getAttribute('data-connector')));
	for (const letter of new Set(letters)) {
		expect(letters.filter((l) => l === letter)).toHaveLength(2);
	}
	const check = page.getByTestId('preview-flow-check');
	await expect(check.getByText(en.flowJumpUnknown.replace('{operation}', '9').replace('{target}', '99'))).toBeVisible();

	// PFD: kolom Jump to dan diagramnya.
	await page.goto(`/documents/view?id=${ids.pfd}`);
	await expect(page.locator('.tabulator-col-title').getByText('Jump to', { exact: true })).toBeAttached();
	await tool(page, en.previewOpen).click();
	await expect(pages.first()).toBeVisible();
	await expect(page.locator('[data-testid="process-flow"] [data-jump-label="No"]').first()).toBeAttached();
});
