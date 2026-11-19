// Test E2E fase 3: tanda tangan digital (D-198, D-203, D-204). PPAP
// Engineer mengirim Control Plan untuk tanda tangan, Project Leader, NPI
// Manager, dan QA QC Manager menandatangani berurutan, lalu PPAP Engineer
// lain mencatat rilis tanpa scan.
//
// Nama file diawali m6- supaya berjalan sesudah formsets.test.ts, yang
// menerbitkan form set AIAG 4th. Project Group GR9 dipakai sendiri, supaya
// pemegang jabatan per Project Group tidak bentrok dengan test lain.
import { expect, test, type Page } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, ensureProjectGroup, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

const people = {
	engineer: { name: 'Engineer TTD', login: 'eng.ttd.e2e', password: 'tanda tangan digital satu' },
	leader: { name: 'Leader TTD', login: 'pl.ttd.e2e', password: 'tanda tangan digital dua' },
	npi: { name: 'NPI TTD', login: 'npi.ttd.e2e', password: 'tanda tangan digital tiga' },
	qa: { name: 'QA TTD', login: 'qa.ttd.e2e', password: 'tanda tangan digital empat' },
	recorder: { name: 'Recorder TTD', login: 'rec.ttd.e2e', password: 'tanda tangan digital lima' }
};

// setupMasterData: customer SG di GR9 dengan rute default, dan pemegang
// jabatan untuk setiap slot rute "Up to QA QC Manager". Dipanggil sebagai admin.
async function setupMasterData(page: Page): Promise<void> {
	await page.evaluate(async (names) => {
		const json = (path: string, method: string, body?: object) =>
			fetch(path, { method, headers: { 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined }).then((r) =>
				r.status === 204 ? null : r.json()
			);
		const groups = await json('/api/v1/project-groups', 'GET');
		const group = groups.items.find((g: { code: string }) => g.code === 'GR9');
		const customer = await json('/api/v1/customers', 'POST', { code: 'SG', name: 'Customer SG', project_group_id: group.id });
		await json('/api/v1/models', 'POST', { customer_id: customer.id, name: 'S-1' });
		const routes = await json('/api/v1/approval-routes', 'GET');
		const route = routes.items.find((r: { name: string }) => r.name === 'Up to QA QC Manager');
		await json(`/api/v1/customers/${customer.id}/default-route`, 'PUT', { route_id: route.id });

		const accounts = await json('/api/v1/accounts', 'GET');
		const person = (login: string) => accounts.items.find((a: { login: string }) => a.login === login).person.id;
		const positions = await json('/api/v1/positions', 'GET');
		const position = (key: string) => positions.items.find((p: { key: string }) => p.key === key);
		const hold = async (key: string, login: string, perGroup: boolean) => {
			const p = position(key);
			// Jabatan satu pemegang mungkin sudah diisi test lain: lepas dulu.
			if (p.single_holder && !perGroup) {
				for (const h of p.holders) await json(`/api/v1/positions/${p.id}/holders/${h.id}`, 'DELETE');
			}
			await json(`/api/v1/positions/${p.id}/holders`, 'POST', { person_id: person(login), project_group_id: perGroup ? group.id : undefined });
		};
		await hold('ppap_engineer', names.engineer, false);
		await hold('project_leader', names.leader, true);
		await hold('npi_manager', names.npi, false);
		await hold('qa_qc_manager', names.qa, false);
	}, { engineer: people.engineer.login, leader: people.leader.login, npi: people.npi.login, qa: people.qa.login });
}

// signNext membuka Documents → Signatures sebagai akun yang ditunggu, lalu menandatangani.
async function signNext(page: Page, who: { login: string; password: string }, temporary: string, slot: string): Promise<void> {
	await logOut(page);
	await logInFirstTime(page, who.login, temporary, who.password);
	await openFromMenu(page, en.navGroupDocuments, en.navSignatures);
	await page.getByRole('link', { name: en.sigOpenSign }).first().click();
	await expect(page.getByRole('heading', { name: en.signAs.replace('{slot}', slot) })).toBeVisible();
	await page.locator('#sign-password').fill(who.password);
	await page.getByRole('button', { name: en.signButton, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.signDone.replace('{slot}', slot));
}

test('fase 3: kirim untuk tanda tangan digital, tanda tangan berurutan, lalu rilis tanpa scan', async ({ page }) => {
	await logIn(page);
	await ensureProjectGroup(page, 'GR9');
	const temporary: Record<string, string> = {};
	temporary.engineer = await createWorkAccount(page, people.engineer.name, people.engineer.login, ['ppap_engineer']);
	temporary.leader = await createWorkAccount(page, people.leader.name, people.leader.login, ['project_leader']);
	temporary.npi = await createWorkAccount(page, people.npi.name, people.npi.login, ['manager']);
	temporary.qa = await createWorkAccount(page, people.qa.name, people.qa.login, ['manager']);
	temporary.recorder = await createWorkAccount(page, people.recorder.name, people.recorder.login, ['ppap_engineer']);
	await setupMasterData(page);

	// PPAP Engineer membuat set dokumen lewat API, lalu membuka Control Plan.
	await logOut(page);
	await logInFirstTime(page, people.engineer.login, temporary.engineer, people.engineer.password);
	const cpID = await page.evaluate(async () => {
		const post = (path: string, body: object, headers: Record<string, string> = {}) =>
			fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...headers }, body: JSON.stringify(body) }).then((r) => r.json());
		const customers = await (await fetch('/api/v1/customers')).json();
		const customer = customers.items.find((c: { code: string }) => c.code === 'SG');
		const models = await (await fetch(`/api/v1/models?customer_id=${customer.id}`)).json();
		const set = await post('/api/v1/document-numbers/sets', { customer_id: customer.id }, { 'Idempotency-Key': crypto.randomUUID() });
		let cp = '';
		for (const n of set.items as { id: string; status: string }[]) {
			if (n.status !== 'reserved') continue;
			const doc = await post('/api/v1/documents', { document_number_id: n.id, model_ids: [models.items[0].id] });
			if (doc.document_type.key === 'cp') cp = doc.id;
		}
		return cp;
	});
	await page.goto(`/documents/view?id=${cpID}`);
	await expect(page.getByRole('heading', { name: en.sigSendTitle })).toBeVisible();

	// Prepared: diri sendiri. Checked: Project Leader. Slot Approved terisi otomatis.
	await page.getByLabel(en.sigSlotChoose.replace('{slot}', 'Process. Eng')).selectOption({ label: `${people.engineer.name} · PPAP Engineer` });
	await page.getByLabel(en.sigSlotChoose.replace('{slot}', 'Engineer')).selectOption({ label: `${people.leader.name} · Project Leader` });
	await page.getByLabel(en.sigNote).fill('Revisi awal');
	await page.locator('#sig-password').fill(people.engineer.password);
	await page.getByRole('button', { name: en.sigSend }).click();
	await expect(page.getByRole('status').first()).toHaveText(en.sigSent.replace('{revision}', '0'));
	await expect(page.getByText(en.sigLockedWaiting)).toBeVisible();

	await signNext(page, people.leader, temporary.leader, 'Engineer');
	await signNext(page, people.npi, temporary.npi, 'NPI Manager');
	await signNext(page, people.qa, temporary.qa, 'QA / QC Mgr.');

	// PPAP Engineer lain mencatat rilis tanpa scan.
	await logOut(page);
	await logInFirstTime(page, people.recorder.login, temporary.recorder, people.recorder.password);
	await openFromMenu(page, en.navGroupDocuments, en.navSignatures);
	await page.getByRole('button', { name: en.signaturesTabRelease.replace('{count}', '1') }).click();
	await page.getByRole('link', { name: en.signaturesOpenDocument }).first().click();
	await expect(page.getByText(en.sigLockedCompleted)).toBeVisible();
	await expect(page.getByText(en.revisionSignedDigitally)).toBeVisible();
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: en.revisionRelease.replace('{revision}', '0') }).click();
	await expect(page.getByRole('status').first()).toContainText(' revision 0 is released.');
});
