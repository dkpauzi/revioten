// Test E2E Tahap 3, M3.1: format template lewat layar. Label kolom bisa
// diganti tanpa merusak data (ID field tetap, dibuktikan test Go
// TestRenameKeepsData); kolom disembunyikan, tidak dihapus.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime, logOut, openFromMenu } from './helpers.ts';

test('admin mengganti label kolom PFMEA, menyembunyikan kolom, dan mengisi kriteria skala', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatChoose).selectOption({ label: 'PFMEA AIAG 4th · PFMEA' });

	// Ganti label: kolom ke-3 tetap di tempatnya dengan nama baru.
	await page.getByRole('row', { name: /^4 Potential Failure Mode / }).getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel(en.formatColumnLabel, { exact: true }).first().fill('Failure Mode');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatFieldSaved.replace('{label}', 'Failure Mode'));
	await expect(page.getByRole('row', { name: /^4 Failure Mode / })).toBeVisible();

	// Sembunyikan kolom: tetap di daftar, ditandai tersembunyi.
	await page.getByRole('row', { name: /^3 Requirement / }).getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel(en.formatHidden, { exact: true }).check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('row', { name: /^3 Requirement / })).toContainText(en.formatYes);

	// Kriteria skala diisi dari form yang dipakai.
	await page.getByLabel(en.formatRatingCriteriaFor.replace('{scale}', en.formatScaleSeverity).replace('{value}', '10')).fill('Hazardous without warning');
	await page.getByRole('button', { name: en.formatRatingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatRatingsSaved);

	// Setelah dimuat ulang, perubahan tetap ada.
	await page.reload();
	await page.getByLabel(en.formatChoose).selectOption({ label: 'PFMEA AIAG 4th · PFMEA' });
	await expect(page.getByRole('row', { name: /^4 Failure Mode / })).toBeVisible();
	await expect(page.getByLabel(en.formatRatingCriteriaFor.replace('{scale}', en.formatScaleSeverity).replace('{value}', '10'))).toHaveValue(
		'Hazardous without warning'
	);
});

test('Penyusun hanya melihat format template', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Penyusun format', 'penyusun.format', ['author']);
	await logOut(page);
	await logInFirstTime(page, 'penyusun.format', temporary, 'kolom punya ID yang tetap selamanya');

	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await expect(page.getByText(en.formatReadOnly)).toBeVisible();
	await page.getByLabel(en.formatChoose).selectOption({ label: 'CP · CP' });
	await expect(page.getByRole('row', { name: /^9 Special Characteristics / })).toBeVisible();
	// Process Flow tampil dan tercetak, terisi dari PFD (D-122); nama proses
	// terisi dari master proses (D-123); Revision dipasang sistem (R-REV-09).
	await expect(page.getByRole('row', { name: /^1 Process Flow / })).toContainText(en.formatKindProcessFlow);
	await expect(page.getByRole('row', { name: /^3 Process Name \/ Operation Description / })).toContainText(en.formatKindProcess);
	await expect(page.getByText(en.formatSymbolsAutomatic.replace('{names}', 'Revision'))).toBeVisible();
	await expect(page.getByRole('button', { name: en.formatEdit })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.formatCreate })).toHaveCount(0);
});

test('admin menyalin format template CP lalu menambah kolom pilihan', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatName, { exact: true }).fill('CP with phase');
	await page.getByLabel(en.formatDocumentType, { exact: true }).selectOption({ label: 'CP' });
	await page.getByLabel(en.formatCopyFrom).selectOption({ label: 'CP · CP' });
	await page.getByRole('button', { name: en.formatCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatCreated.replace('{name}', 'CP with phase'));
	// Salinan langsung terbuka dengan kolom yang sama.
	await expect(page.getByRole('row', { name: /^16 Reaction Plan / })).toBeVisible();

	await page.getByLabel(en.formatKind).selectOption({ label: en.formatKindSelect });
	await page.getByLabel(en.formatColumnLabel, { exact: true }).fill('Phase');
	await page.getByLabel(en.formatOptions).fill('Prototype\nPre-Launch\nProduction');
	await page.getByRole('button', { name: en.formatAddField }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatFieldAdded.replace('{label}', 'Phase'));
	await expect(page.getByRole('row', { name: /^17 Phase / })).toContainText('Prototype, Pre-Launch, Production');
});

test('format template dengan metode AP menampilkan tabel Action Priority dan kalkulatornya', async ({ page }) => {
	// Form set langkah A (D-134): versi baru bermetode AP mendapat tabel handbook.
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	await page.getByLabel(en.formatName, { exact: true }).fill('PFMEA AP E2E');
	// Jenis dokumen kedua (PFMEA) dipilih menurut urutan, karena test lain boleh mengganti namanya.
	await page.getByLabel(en.formatDocumentType, { exact: true }).selectOption({ index: 2 });
	await page.getByLabel(en.formatRiskMethod, { exact: true }).selectOption({ label: en.formatRiskAp });
	await page.getByRole('button', { name: en.formatCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatCreated.replace('{name}', 'PFMEA AP E2E'));

	const section = page.locator('section').filter({ has: page.getByRole('heading', { name: en.apTableTitle }) });
	// Baris pertama tabel handbook: S 9–10, O 8–10 selalu H.
	await expect(section.getByRole('row', { name: /^9–10 8–10 H H H H$/ })).toBeVisible();
	await expect(section.getByRole('row', { name: /^1 8–10 L L L L$/ })).toBeVisible();

	// Kalkulator: contoh dari prompt pemilik.
	const result = section.locator('.result');
	await expect(result).toContainText('H'); // bawaan 8, 6, 4
	await section.getByLabel(en.formatScaleOccurrence).fill('2');
	await expect(result).toContainText('L'); // 8, 2, 4
	await section.getByLabel(en.formatScaleSeverity).fill('7');
	await section.getByLabel(en.formatScaleOccurrence).fill('3');
	await section.getByLabel(en.formatScaleDetection).fill('7');
	await expect(result).toContainText('M'); // 7, 3, 7

	// Server memberi jawaban yang sama untuk klien API.
	const priority = await page.evaluate(async () => {
		const templates = await (await fetch('/api/v1/format-templates')).json();
		const ft = templates.items.find((x: { name: string }) => x.name === 'PFMEA AP E2E');
		const response = await fetch(`/api/v1/format-template-versions/${ft.versions[0].id}/action-priority?severity=8&occurrence=6&detection=4`);
		return (await response.json()).priority;
	});
	expect(priority).toBe('H');
});

test('form PFMEA AIAG-VDA 2019 dan Control Plan AIAG 2024 tersedia sejak awal', async ({ page }) => {
	// Form set langkah B dan C (D-133, D-135): isi awal kedua form baru.
	await logIn(page);
	await openFromMenu(page, en.navGroupTemplates, en.navFormatTemplates);
	const choose = page.getByLabel(en.formatChoose);
	// Pilihan dicari menurut nama template, karena nama jenis dokumennya boleh diganti test lain.
	const pick = async (name: string) => {
		const value = await choose.locator('option', { hasText: name }).getAttribute('value');
		await choose.selectOption(value ?? '');
	};

	await pick('PFMEA AIAG-VDA 2019');
	// Kolom dengan judul grupnya, dan dua kolom AP yang dihitung.
	const failureMode = page.getByRole('row', { name: /^10 2\. Failure Mode \(FM\) FAILURE ANALYSIS \(STEP 4\) / });
	await expect(failureMode).toContainText(en.formatKindText);
	await expect(page.getByRole('row', { name: /^16 PFMEA AP RISK ANALYSIS \(STEP 5\) / })).toContainText(
		en.formatKindApDetail.replace('{set}', en.formatSetCurrent)
	);
	await expect(page.getByRole('row', { name: /^28 PFMEA AP \(after action\) / })).toContainText(
		en.formatKindApDetail.replace('{set}', en.formatSetResult)
	);
	await expect(page.getByRole('row', { name: /^4 3\. Process Work Element \(4M\) / })).toContainText('Man, Machine, Material, Method, Measurement, Milieu');
	await expect(page.getByRole('row', { name: /^10 PFMEA ID Number / })).toContainText(en.formatKindDocumentNumber);
	await expect(page.getByRole('heading', { name: en.apTableTitle })).toBeVisible();

	await pick('Control Plan AIAG 2024');
	await expect(page.getByRole('row', { name: /^1 Phase / })).toContainText('Prototype, Pre-launch, Safe Launch, Production');
	await expect(page.getByRole('row', { name: /^2 Control plan No\. / })).toContainText(en.formatKindDocumentNumber);
	await expect(page.getByRole('row', { name: /^3 Owner Part\/Process Number / })).toBeVisible();
	// CP 2024 tidak punya kolom Process Flow; alurnya ada di PFD.
	await expect(page.getByRole('row').filter({ hasText: en.formatKindProcessFlow })).toHaveCount(0);
});