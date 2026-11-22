import { describe, expect, it } from 'vitest';
import type { FormatField, FormatTemplate, FormatVersion } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import {
	automaticSymbolNames,
	fieldsOf,
	kindText,
	mergeOptions,
	mergeSymbols,
	methodologyText,
	optionLines,
	ratingsOf,
	templateLabel
} from './formats.ts';

// field membuat satu field contoh.
function field(over: Partial<FormatField>): FormatField {
	return { id: 'f', area: 'row', kind: 'text', label: 'Label', position: 1, required: false, hidden: false, ...over };
}

describe('teks format template', () => {
	it('menulis jenis field beserta skala dan set rating', () => {
		expect(kindText(field({ kind: 'text' }))).toBe(en.formatKindText);
		expect(kindText(field({ kind: 'rating', rating_scale: 'severity', rating_set: 2 }))).toBe(
			en.formatKindRatingDetail.replace('{scale}', en.formatScaleSeverity).replace('{set}', en.formatSetResult)
		);
		expect(kindText(field({ kind: 'rpn', rating_set: 1 }))).toBe(en.formatKindRpnDetail.replace('{set}', en.formatSetCurrent));
		// Kolom AP dan nomor dokumen (form set langkah B dan C).
		expect(kindText(field({ kind: 'ap', rating_set: 2 }))).toBe(en.formatKindApDetail.replace('{set}', en.formatSetResult));
		expect(kindText(field({ kind: 'document_number' }))).toBe(en.formatKindDocumentNumber);
		expect(kindText(field({ kind: 'process_flow' }))).toBe(en.formatKindProcessFlow);
	});

	it('menulis template dengan jenis dokumen dan kode customer bila ada', () => {
		const ft: FormatTemplate = {
			id: 't',
			name: 'PFMEA SQAM',
			document_type: { id: 'd', key: 'pfmea', name: 'PFMEA' },
			methodology: 'sqam',
			customer_id: 'c',
			customer_code: 'KT',
			hidden: false,
			versions: [],
			created_at: '2026-10-31T00:00:00Z'
		};
		expect(templateLabel(ft)).toBe('PFMEA SQAM · PFMEA · KT');
		// D-218: template tersembunyi diberi tanda.
		expect(templateLabel({ ...ft, hidden: true })).toBe('PFMEA SQAM · PFMEA · KT · ' + en.hiddenMark);
		expect(methodologyText(ft.methodology)).toBe(en.formatMethodSqam);
		expect(methodologyText(undefined)).toBe('');
	});

	it('memilah field per area dan tabel skala per skala', () => {
		const v: FormatVersion = {
			id: 'v',
			template_id: 't',
			version: 1,
			status: 'draft',
			risk_method: 'rpn',
			highlight_rpn_above: 100,
			highlight_severity_at_least: 8,
			fields: [field({ id: 'h', area: 'header' }), field({ id: 'r' })],
			ratings: [
				{ scale: 'severity', value: 1, criteria: '' },
				{ scale: 'detection', value: 1, criteria: 'Almost certain' }
			],
			symbols: [],
			ap_rules: [],
			created_at: '2026-10-31T00:00:00Z'
		};
		expect(fieldsOf(v, 'header').map((f) => f.id)).toEqual(['h']);
		expect(fieldsOf(v, 'row').map((f) => f.id)).toEqual(['r']);
		expect(ratingsOf(v, 'detection')).toEqual([{ scale: 'detection', value: 1, criteria: 'Almost certain' }]);
	});
});

describe('pilihan dan simbol dari kotak isian', () => {
	it('mempertahankan key pilihan lama menurut urutan, dan pilihan baru tanpa key', () => {
		const existing = [
			{ key: 'scrap', label: 'Scrap' },
			{ key: 'ncp', label: 'NCP' }
		];
		expect(optionLines(existing)).toBe('Scrap\nNCP');
		expect(mergeOptions(existing, 'Scrap\n  Nonconforming product \n\nRework\n')).toEqual([
			{ key: 'scrap', label: 'Scrap' },
			{ key: 'ncp', label: 'Nonconforming product' },
			{ label: 'Rework' }
		]);
	});

	it('menyebut simbol yang dipasang sistem', () => {
		const symbols = [
			{ key: 'critical_process', name: 'Critical Process', automatic: false },
			{ key: 'revision', name: 'Revision', automatic: true }
		];
		expect(automaticSymbolNames(symbols)).toBe('Revision');
		expect(automaticSymbolNames(symbols.slice(0, 1))).toBe('');
	});

	it('menyusun simbol dengan cara yang sama', () => {
		expect(mergeSymbols([{ key: 'revision', name: 'Revision' }], 'Revision Mark\nSafety')).toEqual([
			{ key: 'revision', name: 'Revision Mark' },
			{ name: 'Safety' }
		]);
	});
});
