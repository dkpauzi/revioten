// Test E2E Tahap 3, M3.1: format template lewat layar. Label kolom bisa
// diganti tanpa merusak data (ID field tetap, dibuktikan test Go
// TestRenameKeepsData); kolom disembunyikan, tidak dihapus.
import { expect, test } from '@playwright/test';
import { en } from '../src/lib/i18n/en.ts';
import { createWorkAccount, logIn, logInFirstTime } from './helpers.ts';

test('admin mengganti label kolom PFMEA, menyembunyikan kolom, dan mengisi kriteria skala', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navFormatTemplates }).click();
	await page.getByLabel(en.formatChoose).selectOption({ label: 'PFMEA AIAG 4th · PFMEA' });

	// Ganti label: kolom ke-3 tetap di tempatnya dengan nama baru.
	await page.getByRole('row', { name: /^3 Potential Failure Mode / }).getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel(en.formatColumnLabel, { exact: true }).first().fill('Failure Mode');
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatFieldSaved.replace('{label}', 'Failure Mode'));
	await expect(page.getByRole('row', { name: /^3 Failure Mode / })).toBeVisible();

	// Sembunyikan kolom: tetap di daftar, ditandai tersembunyi.
	await page.getByRole('row', { name: /^2 Requirements / }).getByRole('button', { name: en.formatEdit }).click();
	await page.getByLabel(en.formatHidden, { exact: true }).check();
	await page.getByRole('button', { name: en.masterSave, exact: true }).click();
	await expect(page.getByRole('row', { name: /^2 Requirements / })).toContainText(en.formatYes);

	// Kriteria skala diisi dari form yang dipakai.
	await page.getByLabel(en.formatRatingCriteriaFor.replace('{scale}', en.formatScaleSeverity).replace('{value}', '10')).fill('Hazardous without warning');
	await page.getByRole('button', { name: en.formatRatingsSave }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatRatingsSaved);

	// Setelah dimuat ulang, perubahan tetap ada.
	await page.reload();
	await page.getByLabel(en.formatChoose).selectOption({ label: 'PFMEA AIAG 4th · PFMEA' });
	await expect(page.getByRole('row', { name: /^3 Failure Mode / })).toBeVisible();
	await expect(page.getByLabel(en.formatRatingCriteriaFor.replace('{scale}', en.formatScaleSeverity).replace('{value}', '10'))).toHaveValue(
		'Hazardous without warning'
	);
});

test('Penyusun hanya melihat format template', async ({ page }) => {
	await logIn(page);
	const temporary = await createWorkAccount(page, 'Penyusun format', 'penyusun.format', ['author']);
	await page.getByRole('button', { name: en.logout }).click();
	await logInFirstTime(page, 'penyusun.format', temporary, 'kolom punya ID yang tetap selamanya');

	await page.getByRole('link', { name: en.navFormatTemplates }).click();
	await expect(page.getByText(en.formatReadOnly)).toBeVisible();
	await page.getByLabel(en.formatChoose).selectOption({ label: 'CP · CP' });
	await expect(page.getByRole('row', { name: /^6 Special Characteristic / })).toBeVisible();
	// Tinjauan desain UI: Process Flow dibentuk dari PFD (R-DOC-09), jadi
	// tersembunyi di isi awal, dan Revision dipasang sistem (R-REV-09).
	await expect(page.getByRole('row', { name: /^1 Process Flow / })).toContainText(en.formatYes);
	await expect(page.getByText(en.formatSymbolsAutomatic.replace('{names}', 'Revision'))).toBeVisible();
	await expect(page.getByRole('button', { name: en.formatEdit })).toHaveCount(0);
	await expect(page.getByRole('button', { name: en.formatCreate })).toHaveCount(0);
});

test('admin menyalin format template CP lalu menambah kolom pilihan', async ({ page }) => {
	await logIn(page);
	await page.getByRole('link', { name: en.navFormatTemplates }).click();
	await page.getByLabel(en.formatName, { exact: true }).fill('CP with phase');
	await page.getByLabel(en.formatDocumentType, { exact: true }).selectOption({ label: 'CP' });
	await page.getByLabel(en.formatCopyFrom).selectOption({ label: 'CP · CP' });
	await page.getByRole('button', { name: en.formatCreate }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatCreated.replace('{name}', 'CP with phase'));
	// Salinan langsung terbuka dengan kolom yang sama.
	await expect(page.getByRole('row', { name: /^13 Reaction Plan / })).toBeVisible();

	await page.getByLabel(en.formatKind).selectOption({ label: en.formatKindSelect });
	await page.getByLabel(en.formatColumnLabel, { exact: true }).fill('Phase');
	await page.getByLabel(en.formatOptions).fill('Prototype\nPre-Launch\nProduction');
	await page.getByRole('button', { name: en.formatAddField }).click();
	await expect(page.getByRole('status')).toHaveText(en.formatFieldAdded.replace('{label}', 'Phase'));
	await expect(page.getByRole('row', { name: /^14 Phase / })).toContainText('Prototype, Pre-Launch, Production');
});
