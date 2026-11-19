// Test E2E M4.1 dan M4.2: dokumen dan baris, diisi di tabel workbook. Kriteria selesai "baris bisa disimpan dan
// dibaca": penyusun membuat dokumen dari nomor Reserved, menambah dan mengubah
// baris, mengisi header, lalu memuat ulang halaman.
//
// Nama file diawali m4- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th; dokumen hanya memakai form set yang terbit.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('penyusun membuat dokumen CP, lalu baris dan header tersimpan', async ({ page }) => {
	// Test ini mengikuti satu dokumen dari M4.1 sampai M4.6, jadi batas waktunya dilonggarkan.
	test.slow();
	await logIn(page);
	await ensureProjectGroup(page, 'GR1');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const group = groups.items.find((g: { code: string }) => g.code === 'GR1');
		const customer = await post('/api/v1/customers', { code: 'DC', name: 'Customer DC', project_group_id: group.id });
		await post('/api/v1/models', { customer_id: customer.id, name: 'D-100' });
	});
	const temporary = await createWorkAccount(page, 'Penyusun dokumen', 'penyusun.dokumen.e2e', ['ppap_engineer']);
	await logOut(page);
	await logInFirstTime(page, 'penyusun.dokumen.e2e', temporary, 'dokumen draft disimpan per baris');

	// Proses master dibuat lewat hak E-6 (Pengelola template), lalu satu nomor CP Reserved untuk DC.
	const number = await page.evaluate(async () => {
		await fetch('/api/v1/processes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name: 'Dokumen Reflow', aliases: [] })
		});
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'DC');
		const types = await (await fetch('/api/v1/document-types')).json();
		const cp = types.items.find((d: { key?: string }) => d.key === 'cp');
		const response = await fetch('/api/v1/document-numbers', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': crypto.randomUUID() },
			body: JSON.stringify({ customer_id: customer.id, document_type_id: cp.id })
		});
		return (await response.json()).items[0].number as string;
	});

	await openFromMenu(page, en.navGroupDocuments, en.navDocuments);
	await page.getByLabel(en.documentCustomer, { exact: true }).selectOption({ label: 'DC · Customer DC' });
	await expect(page.getByText(en.documentsEmpty)).toBeVisible();
	await page.getByLabel(en.documentNumber, { exact: true }).selectOption({ label: `${number} · CP` });
	await page.getByLabel('D-100').check();
	await page.getByRole('button', { name: en.documentCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentCreated.replace('{number}', number));

	await page.getByRole('link', { name: number }).click();
	await expect(page.getByRole('heading', { name: number, level: 1 })).toBeVisible();
	await expect(page.getByText('AIAG 4th set')).toBeVisible();

	// M4.2: baris diisi langsung di tabel workbook. Nomor operasi dan proses
	// adalah atribut baris (D-123).
	const grid = page.locator('.workbook');
	const rows = grid.locator('.tabulator-row');
	const fieldOf = async (title: string) =>
		(await grid.locator('.tabulator-col', { hasText: new RegExp(`^${title}$`) }).first().getAttribute('tabulator-field')) ?? '';
	const cell = (row: number, field: string) => rows.nth(row).locator(`.tabulator-cell[tabulator-field="${field}"]`);
	// type mengetik di satu sel lalu keluar dari sel itu, sehingga barisnya tersimpan.
	const type = async (row: number, field: string, text: string) => {
		await cell(row, field).click();
		await page.keyboard.type(text);
		await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
		await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	};

	for (let i = 0; i < 3; i++) {
		await page.getByRole('button', { name: en.documentRowAdd }).click();
		await expect(rows).toHaveCount(i + 1);
	}
	const product = await fieldOf('Product Characteristics');
	const method = await fieldOf('Method');
	await type(0, '_operation', '10');
	await cell(0, '_process').click();
	await page.locator('.tabulator-edit-list-item', { hasText: 'Dokumen Reflow' }).click();
	await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
	await expect(cell(0, '_process')).toHaveText('Dokumen Reflow');
	await type(0, product, 'Solder joint');
	await type(1, '_operation', '20');
	await type(1, method, 'Visual 100%');

	// Hapus baris ketiga yang kosong: minta konfirmasi dulu.
	page.once('dialog', (dialog) => dialog.accept());
	await rows.nth(2).getByRole('button', { name: en.documentRowDelete }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowRemoved.replace('{position}', '3'));
	await expect(rows).toHaveCount(2);

	// Ubah lagi sel yang sudah berisi: baris tetap di tempatnya.
	await cell(0, product).click();
	await page.keyboard.press('ControlOrMeta+a');
	await page.keyboard.type('Solder joint fillet');
	await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentRowSaved);
	// Header.
	await page.getByLabel('Core Team', { exact: true }).fill('PE, QA, Production');
	await page.getByRole('button', { name: en.documentHeaderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentHeaderSaved);

	// Muat ulang: semua datang lagi dari database.
	await page.reload();
	await expect(cell(0, product)).toHaveText('Solder joint fillet');
	await expect(cell(0, '_process')).toHaveText('Dokumen Reflow');
	await expect(cell(1, '_operation')).toHaveText('20');
	await expect(cell(1, method)).toHaveText('Visual 100%');
	await expect(page.getByLabel('Core Team', { exact: true })).toHaveValue('PE, QA, Production');

	// M4.3 (R-DOC-07): orang lain mengubah baris kedua sesudah halaman ini
	// dimuat. Di sini "orang lain" adalah panggilan API langsung
	// dengan versi terbaru; halaman masih memegang versi lama.
	await page.evaluate(
		async ({ method }) => {
			const id = new URL(location.href).searchParams.get('id');
			const doc = await (await fetch(`/api/v1/documents/${id}`)).json();
			const put = (path: string, body: object) =>
				fetch(path, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
			const row = doc.rows[1];
			await put(`/api/v1/documents/${id}/rows/${row.id}`, {
				operation_number: row.operation_number,
				content: { ...row.content, [method]: 'AOI 100%' },
				version: row.version
			});
		},
		{ method }
	);

	// Konflik baris terdeteksi dan ditampilkan; isi terbaru dimuat dan ketikan
	// yang kalah tidak tersimpan.
	await cell(1, product).click();
	await page.keyboard.type('Bridging');
	await page.getByRole('heading', { name: en.documentRowsTitle, exact: true }).click();
	await expect(page.getByRole('alert')).toHaveText(en.documentRowChanged);
	await expect(cell(1, method)).toHaveText('AOI 100%');
	await expect(cell(1, product)).toHaveText('');
	// Dicoba lagi di atas isi terbaru: tersimpan.
	await type(1, product, 'Bridging');
	await expect(cell(1, method)).toHaveText('AOI 100%');

	// Konflik header: orang lain menyimpan header lebih dulu. Pesan tampil,
	// lalu simpan kedua berhasil.
	await page.evaluate(async () => {
		const id = new URL(location.href).searchParams.get('id');
		const doc = await (await fetch(`/api/v1/documents/${id}`)).json();
		await fetch(`/api/v1/documents/${id}/header`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ header: doc.header, header_version: doc.header_version })
		});
	});
	await page.getByLabel('Core Team', { exact: true }).fill('PE, QA');
	await page.getByRole('button', { name: en.documentHeaderSave }).click();
	await expect(page.getByRole('alert')).toHaveText(en.documentHeaderChanged);
	await page.getByLabel('Core Team', { exact: true }).fill('PE, QA');
	await page.getByRole('button', { name: en.documentHeaderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.documentHeaderSaved);

	// M4.4 (R-MIG-02): blok sel dari Excel ditempel, dipratinjau, lalu diimpor.
	// Baris judul memetakan kolom; proses yang tidak dikenal ditandai, tetapi
	// barisnya tetap masuk.
	await page.getByRole('button', { name: en.pasteOpen }).click();
	// Judul kolom nomor operasi dan proses dibaca dari tabel, seperti orang
	// menyalin baris judul form.
	const titleOf = async (field: string) => ((await grid.locator(`.tabulator-col[tabulator-field="${field}"] .tabulator-col-title`).first().textContent()) ?? '').trim();
	const pasted = [`Method\t${await titleOf('_process')}\tProduct Characteristics\t${await titleOf('_operation')}`, 'X-ray\tDokumen Reflow\t"Void\nunder BGA"\t30', 'Pull test\tBukan Proses\tWire bond\t40'].join('\r\n');
	await page.getByLabel(en.pasteText).fill(pasted);
	await page.getByLabel(en.pasteHasHeader).check();
	await page.getByRole('button', { name: en.pastePreview }).click();
	const summary = page.getByTestId('paste-summary');
	await expect(summary).toHaveText(en.pasteSummary.replace('{rows}', '2').replace('{flags}', '1'));
	await expect(page.locator('.preview td.flag')).toContainText(en.pasteReasonProcess);
	// Pemetaan bisa diubah: kolom pertama dilewati, lalu dikembalikan.
	const firstColumn = page.getByLabel(en.pasteColumn.replace('{n}', '1'));
	await expect(firstColumn).toHaveValue(method);
	await firstColumn.selectOption('');
	await firstColumn.selectOption(method);
	await page.getByRole('button', { name: en.pasteImport.replace('{rows}', '2') }).click();
	await expect(page.getByRole('status')).toHaveText(en.pasteDone.replace('{rows}', '2'));
	await expect(rows).toHaveCount(4);
	await expect(cell(2, '_operation')).toHaveText('30');
	await expect(cell(2, '_process')).toHaveText('Dokumen Reflow');
	await expect(cell(2, product)).toHaveText('Void under BGA');
	await expect(cell(2, method)).toHaveText('X-ray');
	await expect(cell(3, '_process')).toHaveText('');
	await expect(cell(3, method)).toHaveText('Pull test');

	// M4.6 (R-CMP-01): urutan langkah proses diubah dan tersimpan. Empat baris
	// tadi adalah empat langkah: Dokumen Reflow, tanpa proses, Dokumen Reflow,
	// tanpa proses. Nomor operasinya diketik atau ditempel, jadi tidak diubah.
	await page.getByRole('button', { name: en.orderOpen }).click();
	const steps = page.locator('.steps li');
	await expect(steps).toHaveCount(4);
	// Langkah ketiga naik dan bergabung dengan langkah pertama (proses yang sama).
	await page.getByRole('button', { name: en.orderMoveUp.replace('{name}', '3. Dokumen Reflow') }).click();
	await expect(steps).toHaveCount(3);
	await expect(steps.first()).toContainText(en.orderStepRows.replace('{n}', '2'));
	await expect(steps.first().getByTestId('order-number')).toHaveText(en.orderNumber.replace('{number}', '10, 30'));
	await page.getByRole('button', { name: en.orderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.orderSaved);
	await expect(cell(1, product)).toHaveText('Void under BGA');
	await expect(cell(1, '_operation')).toHaveText('30');
	await expect(cell(2, '_operation')).toHaveText('20');

	// Baris baru tanpa nomor: diseret ke tempat kedua, lalu mendapat nomor otomatis.
	await page.getByRole('button', { name: en.documentRowAdd }).click();
	await expect(rows).toHaveCount(5);
	await page.getByRole('button', { name: en.orderOpen }).click();
	await expect(steps).toHaveCount(4);
	await steps.nth(3).dragTo(steps.nth(1));
	await expect(steps.nth(1).getByTestId('order-number')).toHaveText(en.orderNumber.replace('{number}', '20'));
	await page.getByRole('button', { name: en.orderSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.orderSaved);
	await page.reload();
	await expect(rows).toHaveCount(5);
	await expect(cell(2, '_operation')).toHaveText('20');
	await expect(cell(2, product)).toHaveText('');
	await expect(cell(3, method)).toHaveText('AOI 100%');
});
