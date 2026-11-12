// Bantuan layar Form sets (langkah D, D-133).
import type { FormSet, FormSetSheet, FormatTemplate } from './api/gen/index.ts';
import { t } from './i18n/index.ts';

// SheetKey adalah satu dari tiga sheet form set.
export type SheetKey = 'pfd' | 'pfmea' | 'cp';
export const sheetKeys: SheetKey[] = ['pfd', 'pfmea', 'cp'];

const statusText = (status: 'draft' | 'published') => (status === 'draft' ? t('versionStatusDraft') : t('versionStatusPublished'));

// sheetText menulis isi satu sheet, misalnya "PFMEA AIAG 4th · version 1 · Published".
export function sheetText(sheet: FormSetSheet): string {
	return t('formSetSheetOption', { template: sheet.template_name, version: String(sheet.version), status: statusText(sheet.status) });
}

// sheetOptions memberi pilihan versi format untuk satu sheet: semua versi
// dari template dengan jenis dokumen itu. Server tetap memeriksa jenisnya.
export function sheetOptions(templates: FormatTemplate[], sheet: SheetKey): { id: string; label: string }[] {
	return templates
		.filter((ft) => ft.document_type.key === sheet)
		.flatMap((ft) =>
			ft.versions.map((v) => ({
				id: v.id,
				label: t('formSetSheetOption', { template: ft.name, version: String(v.version), status: statusText(v.status) })
			}))
		);
}

// formSetNameOf memberi nama form set yang dipakai satu customer: pilihannya
// sendiri, atau set bawaan sistem bila belum memilih.
export function formSetNameOf(sets: FormSet[], formSetId: string | undefined): string {
	const chosen = sets.find((fs) => fs.id === formSetId) ?? sets.find((fs) => fs.default);
	return chosen?.name ?? '';
}