// Test E2E M3.3: cakupan poin dan prioritas (D-129, R-TPL-03), lewat layar.
// Contoh pemilik: proses laser marking punya poin laser type yang biasanya
// umum, tetapi laser type sebenarnya berbeda per model. Pengelola template
// (E-6) menambah varian untuk model K-125, lalu pratinjau memperlihatkan
// K-125 memakai variannya dan model lain tetap memakai poin umum.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { asMaster, createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

// setupModels membuat customer LM dengan group model "Group Laser", model
// K-125 di group itu, dan model K-200 tanpa group, lewat API sebagai PPAP
// Engineer (E-11, D-209) di jendela terpisah.
async function setupModels(page: Page): Promise<void> {
	await asMaster(page, setupModelsAsMaster);
}

async function setupModelsAsMaster(page: Page): Promise<void> {
	await ensureProjectGroup(page, 'GRL');
	await page.evaluate(async () => {
		const post = (path: string, body: object) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then((r) => r.json());
		const groups = await (await fetch('/api/v1/project-groups')).json();
		const projectGroup = groups.items.find((g: { code: string }) => g.code === 'GRL');
		const customer = await post('/api/v1/customers', { code: 'LM', name: 'Customer LM', project_group_id: projectGroup.id });
		const group = await post('/api/v1/model-groups', { customer_id: customer.id, name: 'Group Laser' });
		await post('/api/v1/models', { customer_id: customer.id, model_group_id: group.id, name: 'K-125' });
		await post('/api/v1/models', { customer_id: customer.id, name: 'K-200' });
	});
}

test('varian model K-125 menang atas poin umum, model lain tetap umum', async ({ page }) => {
	await logIn(page);
	await setupModels(page);
	const temporary = await createWorkAccount(page, 'PPAP Engineer laser', 'ppap.laser.e2e', ['ppap_engineer']);
	await logOut(page);
	await logInFirstTime(page, 'ppap.laser.e2e', temporary, 'laser marking per model berbeda');
	await page.evaluate(async () => {
		await fetch('/api/v1/processes', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name: 'Laser Marking', aliases: [] })
		});
	});

	await openFromMenu(page, en.navGroupTemplates, en.navProcessTemplates);
	await page.getByLabel(en.processTemplateChoose).selectOption({ label: 'Laser Marking' });
	await page.getByRole('button', { name: en.processTemplateCreate }).click();
	await page.getByLabel(en.subprocessName, { exact: true }).fill('Laser');
	await page.getByRole('button', { name: en.subprocessAdd }).click();
	await page.getByRole('button', { name: en.subprocessShowPointsOf.replace('{name}', 'Laser') }).click();

	// Poin baru biasanya umum (cakupan bawaan General).
	// Jenis dokumen pertama yang boleh punya poin (PFMEA, D-214) dipilih menurut urutan, karena test lain boleh
	// mengganti namanya (types.test.ts).
	await page.getByLabel(en.pointDocumentType).selectOption({ index: 1 });
	await page.getByLabel(en.pointText, { exact: true }).fill('Laser type sesuai spesifikasi');
	await page.getByRole('button', { name: en.pointAdd }).click();
	await expect(page.getByRole('status')).toHaveText(en.pointAdded);
	const pointRow = page.getByRole('row', { name: /Laser type sesuai spesifikasi/ });
	await expect(pointRow).toContainText(en.scopeGeneral);

	// Varian untuk model K-125.
	const addVariant = async () => {
		await pointRow.getByRole('button', { name: en.variantAdd }).click();
		const form = page.locator('form').filter({ has: page.getByLabel(en.variantText) });
		await form.getByLabel(en.pointScope).selectOption('model');
		await form.getByLabel(en.scopeLevelCustomer, { exact: true }).selectOption({ label: 'LM · Customer LM' });
		await form.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-125 · Group Laser' });
		await form.getByLabel(en.variantText).fill('Laser type: fiber 20 W');
		await form.getByRole('button', { name: en.variantAdd }).click();
	};
	await addVariant();
	await expect(page.getByRole('status')).toHaveText(en.variantAdded);
	const variantRow = page.getByRole('row', { name: /Laser type: fiber 20 W/ });
	await expect(variantRow).toContainText(en.variantFor.replace('{scope}', 'Model K-125'));

	// Satu cakupan hanya punya satu varian dari poin yang sama.
	await addVariant();
	await expect(page.getByText(en.variantTaken)).toBeVisible();
	await page.getByRole('button', { name: en.masterCancel }).click();

	// Pratinjau: K-125 memakai variannya, K-200 tetap memakai poin umum.
	const preview = page.locator('section').filter({ has: page.getByRole('heading', { name: en.previewTitle }) });
	await preview.getByLabel(en.scopeLevelCustomer, { exact: true }).selectOption({ label: 'LM · Customer LM' });
	await preview.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-125 · Group Laser' });
	await expect(preview.getByRole('row', { name: /Laser type: fiber 20 W/ })).toContainText('Model K-125');
	await expect(preview.getByText('Laser type sesuai spesifikasi')).toHaveCount(0);

	await preview.getByLabel(en.scopeLevelModel, { exact: true }).selectOption({ label: 'K-200' });
	await expect(preview.getByRole('row', { name: /Laser type sesuai spesifikasi/ })).toContainText(en.scopeGeneral);
	await expect(preview.getByText('Laser type: fiber 20 W')).toHaveCount(0);
});
