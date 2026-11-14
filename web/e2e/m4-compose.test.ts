// Test E2E M4.5: perakitan dua panel dari process template (R-CMP-01 sampai
// R-CMP-04, D-146). Kriteria selesai: IQC General, MSD, dan PCB dengan
// sebagian poin dicoret menghasilkan dokumen yang benar.
//
// Nama file diawali m4- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('penyusun merakit CP, PFMEA, dan PFD dari process template IQC', async ({ page }) => {
	// Test ini panjang (menyiapkan template, merakit dua kali), jadi batas waktunya dilonggarkan.
	test.slow();
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'KM', name: 'Customer KM', project_group_id: group.id });
		await post('/api/v1/models', { customer_id: customer.id, name: 'M-45' });
		await post('/api/v1/models', { customer_id: customer.id, name: 'M-46' });
	});
	const temporary = await createWorkAccount(page, 'Perakit dokumen', 'perakit.e2e', ['author', 'template_maintainer', 'pfmea_cp_checker']);
	await logOut(page);
	await logInFirstTime(page, 'perakit.e2e', temporary, 'baris dirakit dari process template');

	// Process template "Incoming QC" dengan tiga sub-proses dan poinnya, lalu
	// satu set nomor PFD, PFMEA, dan CP untuk model M-45, semuanya lewat API.
	const setup = await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const process = await post('/api/v1/processes', { name: 'Incoming QC', aliases: [] });
		const template = await post('/api/v1/process-templates', { process_id: process.id });
		const versionId = template.versions[0].id as string;
		const types = (await (await fetch('/api/v1/document-types')).json()).items as { id: string; key?: string }[];
		const typeId = (key: string) => types.find((d) => d.key === key)!.id;
		const tree: [string, [string, string, Record<string, unknown>][]][] = [
			[
				'IQC General',
				[
					['pfd', 'Incoming inspection', {}],
					['pfmea', 'Wrong part received', { failure_mode: 'Wrong part received', severity: 7 }],
					['cp', 'Check part number', { product_characteristic: 'Part number', control_method: 'Visual' }]
				]
			],
			[
				'IQC MSD',
				[
					['cp', 'Check MSL label', {}],
					['cp', 'Check HIC', { product_characteristic: 'Humidity indicator card' }]
				]
			],
			[
				'IQC PCB',
				[
					['pfmea', 'Warped PCB', { failure_mode: 'Warped PCB' }],
					['cp', 'Check warpage', { product_characteristic: 'Warpage' }]
				]
			]
		];
		for (const [name, points] of tree) {
			const version = await post(`/api/v1/process-template-versions/${versionId}/subprocesses`, { name });
			const sub = version.subprocesses.at(-1);
			for (const [key, text, content] of points) {
				await post(`/api/v1/process-template-versions/${versionId}/points`, { subprocess_id: sub.id, document_type_id: typeId(key), text, content });
			}
		}
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'KM');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		let cpNumber = '';
		for (const n of set.items as { id: string; number: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items.find((m: { name: string }) => m.name === 'M-45').id] });
			if (doc.document_type.key === 'cp') cpNumber = n.number;
		}
		return { versionId, cpNumber };
	});
	expect(setup.cpNumber).not.toBe('');

	// D-146: isi per kolom satu poin diisi di layar Process templates.
	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'Incoming QC' });
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'IQC MSD') }).click();
	await page.getByRole('button', { name: en.pointContentFor.replace('{text}', 'Check MSL label') }).click();
	await page.getByLabel(/^Product (Characteristics|control)$/).fill('MSL label');
	await page.getByRole('button', { name: en.pointContentSave.replace('{text}', 'Check MSL label') }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointContentSaved);
	await expect(page.getByRole('row', { name: /Check MSL label/ }).first()).toContainText(en.pointContentFilled.replace('{n}', '1'));

	// Versi template diterbitkan: hanya versi terbit yang dipakai merakit.
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.versionPublish.replace('{version}', '1') }).click();
	await expect(page.getByRole('status')).toBeVisible();

	// Buka dokumen CP, lalu rakit.
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KM · Customer KM' });
	await page.getByRole('link', { name: setup.cpNumber }).click();
	await expect(page.getByRole('heading', { name: setup.cpNumber, level: 1 })).toBeVisible();

	await page.getByRole('button', { name: en.composeOpen }).click();
	await page.getByLabel(en.composeProcess).selectOption({ label: `Incoming QC · ${en.composeVersion.replace('{version}', '1')}` });
	const all = (name: string) => page.getByRole('checkbox', { name: en.composeAll.replace('{name}', name) });
	await all('IQC General').check();
	await all('IQC MSD').check();
	await all('IQC PCB').check();
	// Satu poin dicoret, dengan alasan.
	await page.getByRole('checkbox', { name: 'CP Check HIC' }).uncheck();
	await page.getByLabel(en.composeReason.replace('{text}', 'Check HIC')).fill('No HIC for this part');

	// Panel kanan: baris yang akan ada di setiap dokumen set.
	const result = (key: string) => page.getByTestId(`compose-result-${key}`).getByRole('listitem');
	await expect(result('cp')).toHaveCount(3);
	await expect(result('pfmea')).toHaveCount(2);
	await expect(result('pfd')).toHaveCount(1);
	await expect(result('cp').first()).toContainText(en.composeWillAdd);

	await page.getByRole('button', { name: en.composeApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.composeDone.replace('{process}', 'Incoming QC'));

	// CP: tiga baris menurut urutan template, proses terisi dari template.
	const grid = page.locator('.workbook');
	const rows = grid.locator('.tabulator-row');
	const fieldOf = async (title: string) =>
		(await grid.locator('.tabulator-col', { hasText: new RegExp(`^${title}$`) }).first().getAttribute('tabulator-field')) ?? '';
	const cell = (row: number, field: string) => rows.nth(row).locator(`.tabulator-cell[tabulator-field="${field}"]`);
	await expect(rows).toHaveCount(3);
	const product = await fieldOf('Product Characteristics');
	await expect(cell(0, product)).toHaveText('Part number');
	await expect(cell(0, await fieldOf('Method'))).toHaveText('Visual');
	await expect(cell(1, product)).toHaveText('MSL label');
	await expect(cell(2, product)).toHaveText('Warpage');
	await expect(cell(0, '_process')).toHaveText('Incoming QC');

	// PFMEA di set yang sama ikut terisi (R-CMP-02). Label kolomnya bisa
	// sudah diganti formats.test.ts menjadi "Failure Mode"; key kolom tetap.
	await page.getByRole('link', { name: 'PFMEA', exact: true }).click();
	await expect(rows).toHaveCount(2);
	await expect(cell(0, await fieldOf('(Potential )?Failure Mode'))).toHaveText('Wrong part received');
	await expect(cell(0, await fieldOf('Severity'))).toHaveText('7');

	// Kembali ke CP: keadaan poin tersimpan. Mencoret satu poin lagi menghapus barisnya.
	await page.getByRole('link', { name: 'CP', exact: true }).click();
	await expect(rows).toHaveCount(3);
	await page.getByRole('button', { name: en.composeOpen }).click();
	await page.getByLabel(en.composeProcess).selectOption({ label: `Incoming QC · ${en.composeVersion.replace('{version}', '1')}` });
	await expect(page.getByRole('checkbox', { name: 'CP Check HIC' })).not.toBeChecked();
	await expect(page.getByLabel(en.composeReason.replace('{text}', 'Check HIC'))).toHaveValue('No HIC for this part');
	await expect(page.getByRole('checkbox', { name: 'CP Check MSL label' })).toBeChecked();
	await page.getByRole('checkbox', { name: 'CP Check MSL label' }).uncheck();
	await expect(result('cp').nth(1)).toContainText(en.composeWillRemove);
	await page.getByRole('button', { name: en.composeApply }).click();
	await expect(page.getByRole('status')).toHaveText(en.composeDone.replace('{process}', 'Incoming QC'));
	await expect(rows).toHaveCount(2);
	await expect(cell(1, product)).toHaveText('Warpage');

	// M4.7 (R-CMP-05): model M-46 dimulai dari komposisi M-45, lalu berdiri sendiri.
	const copyNumber = await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'KM');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const model = models.items.find((m: { name: string }) => m.name === 'M-46');
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		let cp = '';
		for (const n of set.items as { id: string; number: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [model.id] });
			if (doc.document_type.key === 'cp') cp = n.number;
		}
		return cp;
	});
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KM · Customer KM' });
	await page.getByRole('link', { name: copyNumber }).click();
	await expect(page.getByRole('heading', { name: copyNumber, level: 1 })).toBeVisible();
	await page.getByRole('button', { name: en.copyOpen }).click();
	await page
		.getByLabel(en.copySource)
		.selectOption({ label: en.copyOption.replace('{number}', setup.cpNumber).replace('{models}', 'M-45').replace('{rows}', '2') });
	await page.getByRole('button', { name: en.copyRun }).click();
	await expect(page.getByRole('status')).toHaveText(en.copyDone.replace('{number}', setup.cpNumber));
	await expect(rows).toHaveCount(2);
	await expect(cell(0, product)).toHaveText('Part number');
	// Sesudah berisi, tombol salin hilang.
	await expect(page.getByRole('button', { name: en.copyOpen })).toHaveCount(0);

	// Salinan diubah; dokumen M-45 tetap.
	await cell(0, product).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Part number M-46');
	await page.getByRole('heading', { name: en.documentRowsTitle }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	await page.getByRole('link', { name: 'PFMEA', exact: true }).click();
	await expect(rows).toHaveCount(2);
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KM · Customer KM' });
	await page.getByRole('link', { name: setup.cpNumber }).click();
	await expect(cell(0, product)).toHaveText('Part number');

	// M4.8 (R-CMP-04): baris yang diubah ditandai berbeda di tampilan banding,
	// lalu tautannya diputus dengan alasan. Di CP M-46 baris pertama sudah
	// diubah menjadi "Part number M-46".
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KM · Customer KM' });
	await page.getByRole('link', { name: copyNumber }).click();
	await expect(page.getByRole('heading', { name: copyNumber, level: 1 })).toBeVisible();
	await page.getByRole('button', { name: en.compareOpen }).click();
	await expect(page.getByTestId('compare-summary')).toHaveText(en.compareSummary.replace('{rows}', '2').replace('{different}', '1'));
	const firstRow = page.locator('.compare tbody tr').first();
	await expect(firstRow).toContainText(en.compareDifferent);
	await expect(firstRow).toContainText(en.compareTemplate.replace('{value}', 'Part number'));
	await expect(firstRow).toContainText(en.compareRow.replace('{value}', 'Part number M-46'));
	await expect(page.locator('.compare tbody tr').nth(1)).toContainText(en.compareSame);

	// Putus tautan tanpa alasan ditolak; dengan alasan tersimpan.
	await page.getByRole('button', { name: en.compareDetachFor.replace('{position}', '1') }).click();
	await page.getByRole('button', { name: en.compareDetach, exact: true }).click();
	await expect(page.locator('.compare .error')).toBeVisible();
	await page.getByLabel(en.compareDetachReasonLabel).fill('Customer drawing for M-46');
	await page.getByRole('button', { name: en.compareDetach, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.compareDetachDone.replace('{position}', '1'));
	await expect(page.getByTestId('compare-summary')).toHaveText(en.compareSummary.replace('{rows}', '2').replace('{different}', '0'));
	await expect(page.locator('.compare tbody tr').first()).toContainText(en.compareDetachReason.replace('{reason}', 'Customer drawing for M-46'));

	// M4.9 (R-DOC-06, D-65): PFMEA dan CP M-45 berdampingan per proses, lalu
	// proses Incoming QC diberi tanda sudah dicek.
	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'KM · Customer KM' });
	await page.getByRole('link', { name: setup.cpNumber }).click();
	await page.getByRole('link', { name: en.checkTab, exact: true }).click();
	await expect(page.getByRole('heading', { name: en.checkTitle, level: 1 })).toBeVisible();
	const iqc = page.getByRole('region', { name: 'Incoming QC' });
	await expect(iqc.getByTestId('check-pfmea')).toContainText('Wrong part received');
	await expect(iqc.getByTestId('check-cp')).toContainText('Part number');
	await expect(iqc.getByTestId('check-mark')).toHaveText(en.checkNotYet);
	await iqc.getByRole('button', { name: en.checkMarkFor.replace('{process}', 'Incoming QC') }).click();
	await expect(page.getByRole('status')).toHaveText(en.checkMarked.replace('{process}', 'Incoming QC'));
	await expect(iqc.getByTestId('check-mark')).toContainText('Perakit dokumen');

	// Baris CP berubah sesudah dicek: tandanya perlu dicek ulang.
	await page.getByRole('link', { name: 'CP', exact: true }).click();
	await cell(0, product).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Part number and revision');
	await page.getByRole('heading', { name: en.documentRowsTitle }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	await page.getByRole('link', { name: en.checkTab, exact: true }).click();
	await expect(iqc.getByTestId('check-mark')).toContainText(en.checkStale.split('{by}')[0].trim());
	await iqc.getByRole('button', { name: en.checkMarkFor.replace('{process}', 'Incoming QC') }).click();
	await expect(iqc.getByTestId('check-mark')).toContainText(en.checkDone.split('{by}')[0].trim());

	// M4.10: diagram PFD terbentuk otomatis dari daftar proses, dan ikut
	// berubah saat baris ditambah.
	await page.getByRole('link', { name: 'PFD', exact: true }).click();
	await expect(rows).toHaveCount(1);
	await page.getByRole('button', { name: en.diagramOpen }).click();
	const diagram = page.getByTestId('pfd-diagram');
	await expect(diagram.locator('.svelte-flow__node')).toHaveCount(1);
	await expect(diagram.locator('.svelte-flow__node').first()).toContainText('Incoming QC');
	await page.getByRole('button', { name: en.documentRowAdd }).click();
	await expect(rows).toHaveCount(2);
	await expect(diagram.locator('.svelte-flow__node')).toHaveCount(2);
	await expect(diagram.locator('.svelte-flow__edge')).toHaveCount(1);
});
