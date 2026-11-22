import { describe, expect, it } from 'vitest';
import type { FormatTemplate } from './api/gen/index.ts';
import { formSetNameOf, sheetOptions, sheetText, visibleItems } from './formSets.ts';
import { en } from './i18n/en.ts';

const template = (id: string, name: string, key: 'pfd' | 'pfmea' | 'cp', versions: FormatTemplate['versions'], hidden = false): FormatTemplate => ({
	id, name, document_type: { id: 'dt-' + key, key, name: key.toUpperCase() }, hidden, versions, created_at: '2026-11-12T00:00:00Z'
});
const templates = [
	template('t-1', 'PFD', 'pfd', [{ id: 'v-1', version: 1, status: 'published', risk_method: 'none' }]),
	template('t-2', 'PFMEA v1', 'pfmea', [
		{ id: 'v-2', version: 1, status: 'published', risk_method: 'rpn' },
		{ id: 'v-3', version: 2, status: 'draft', risk_method: 'rpn' }
	]),
	template('t-3', 'CP', 'cp', [{ id: 'v-4', version: 1, status: 'draft', risk_method: 'none' }]),
	template('t-4', 'PFMEA v2', 'pfmea', [{ id: 'v-5', version: 1, status: 'draft', risk_method: 'ap' }], true)
];

describe('sheetOptions', () => {
	it('memberi semua versi format dengan jenis dokumen sheet itu', () => {
		expect(sheetOptions(templates, 'pfmea').map((o) => o.id)).toEqual(['v-2', 'v-3']);
		expect(sheetOptions(templates, 'pfd').map((o) => o.id)).toEqual(['v-1']);
	});

	it('tidak menawarkan template tersembunyi kecuali yang sedang dipakai (D-218)', () => {
		expect(sheetOptions(templates, 'pfmea', 'v-5').map((o) => o.id)).toEqual(['v-2', 'v-3', 'v-5']);
	});

	it('menulis nama template, nomor versi, dan statusnya', () => {
		expect(sheetOptions(templates, 'cp')[0].label).toBe(
			en.formSetSheetOption.replace('{template}', 'CP').replace('{version}', '1').replace('{status}', en.versionStatusDraft)
		);
	});
});

describe('sheetText', () => {
	it('menulis isi satu sheet', () => {
		expect(sheetText({ format_version_id: 'v-2', template_name: 'PFMEA v1', version: 1, status: 'published' })).toBe(
			en.formSetSheetOption.replace('{template}', 'PFMEA v1').replace('{version}', '1').replace('{status}', en.versionStatusPublished)
		);
	});
});

describe('formSetNameOf', () => {
	const sets = [
		{ id: 'fs-1', key: 'aiag_4th', name: 'AIAG 4th set', default: true, hidden: false, versions: [] },
		{ id: 'fs-2', key: 'aiag_vda', name: 'AIAG-VDA set', default: false, hidden: true, versions: [] }
	];

	it('memberi nama set pilihan customer', () => {
		expect(formSetNameOf(sets, 'fs-2')).toBe('AIAG-VDA set');
	});

	it('memberi nama set bawaan bila customer belum memilih', () => {
		expect(formSetNameOf(sets, undefined)).toBe('AIAG 4th set');
	});
});
describe('visibleItems (D-218)', () => {
	const items = [
		{ id: 'a', hidden: false },
		{ id: 'b', hidden: true },
		{ id: 'c', hidden: true }
	];

	it('menyembunyikan yang tersembunyi, kecuali yang sedang dipilih', () => {
		expect(visibleItems(items, false).map((i) => i.id)).toEqual(['a']);
		expect(visibleItems(items, false, 'c').map((i) => i.id)).toEqual(['a', 'c']);
	});

	it('menampilkan semua bila Show hidden dicentang', () => {
		expect(visibleItems(items, true).map((i) => i.id)).toEqual(['a', 'b', 'c']);
	});
});
