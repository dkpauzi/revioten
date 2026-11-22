import { describe, expect, it } from 'vitest';
import type { DocumentType, FormatVersion, ProcessTemplate, Subprocess, TemplatePoint } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import { contentInput, newPointTypes, pointColumns, pointSummary, pointTypes, pointsOf, scopeText, templateOf, variantLevels, versionText } from './processTemplates.ts';

const types: DocumentType[] = [
	{ id: 'd-pfd', key: 'pfd', name: 'PFD', letter: 'H', prefix: 'SH' },
	{ id: 'd-pfmea', key: 'pfmea', name: 'PFMEA', letter: 'F', prefix: 'SF' },
	{ id: 'd-cp', key: 'cp', name: 'CP', letter: 'C', prefix: 'SC' },
	{ id: 'd-wi', name: 'Work Instruction', letter: 'W', prefix: 'SW' }
];

// point membuat satu poin contoh untuk satu jenis dokumen.
function point(id: string, type: DocumentType, position: number): TemplatePoint {
	return {
		id,
		document_type: { id: type.id, key: type.key ?? 'pfd', name: type.name },
		text: `Point ${id}`,
		content: {},
		position,
		hidden: false,
		scope: { level: 'general' },
		variants: []
	};
}

// Cakupan poin dan varian (M3.3, D-129).
describe('cakupan poin di layar', () => {
	it('menulis cakupan dengan nama sasarannya', () => {
		expect(scopeText({ level: 'general' })).toBe(en.scopeGeneral);
		expect(scopeText({ level: 'customer', target_id: 'c', label: 'AB' })).toBe('Customer AB');
		expect(scopeText({ level: 'model_group', target_id: 'g', label: 'Group Laser' })).toBe('Model group Group Laser');
		expect(scopeText({ level: 'model', target_id: 'm', label: 'K-125' })).toBe('Model K-125');
	});

	// Varian selalu lebih sempit dari poin asalnya; poin khusus model tidak punya varian.
	it('menawarkan tingkat varian yang lebih sempit dari poin asalnya', () => {
		expect(variantLevels('general')).toEqual(['customer', 'model_group', 'model']);
		expect(variantLevels('customer')).toEqual(['model_group', 'model']);
		expect(variantLevels('model_group')).toEqual(['model']);
		expect(variantLevels('model')).toEqual([]);
	});
});

describe('process template di layar', () => {
	it('mencari template milik satu proses', () => {
		const templates: ProcessTemplate[] = [
			{ id: 'pt-1', process: { id: 'p-iqc', name: 'IQC' }, versions: [], created_at: '2026-10-31T00:00:00Z' }
		];
		expect(templateOf(templates, 'p-iqc')?.id).toBe('pt-1');
		expect(templateOf(templates, 'p-smt')).toBeNull();
	});

	it('hanya PFD, PFMEA, dan CP yang punya poin', () => {
		expect(pointTypes(types).map((d) => d.name)).toEqual(['PFD', 'PFMEA', 'CP']);
		// Poin baru hanya untuk PFMEA dan CP (D-214).
		expect(newPointTypes(types).map((d) => d.name)).toEqual(['PFMEA', 'CP']);
	});

	it('memilah poin per jenis dokumen dan meringkas jumlahnya', () => {
		const msd: Subprocess = {
			id: 's-1',
			name: 'IQC MSD',
			position: 1,
			hidden: false,
			points: [point('a', types[0], 1), point('b', types[1], 1), point('c', types[1], 2)]
		};
		expect(pointsOf(msd, 'd-pfmea').map((p) => p.id)).toEqual(['b', 'c']);
		expect(pointSummary(msd, pointTypes(types))).toBe('PFD 1 · PFMEA 2');
		expect(pointSummary({ ...msd, points: [] }, pointTypes(types))).toBe('');
	});

	it('menulis nomor dan status versi', () => {
		expect(versionText({ version: 1, status: 'draft' })).toBe(
			en.processTemplateVersion.replace('{version}', '1').replace('{status}', en.processTemplateDraft)
		);
	});
});

describe('pointColumns (D-146)', () => {
	const field = (id: string, kind: string, key: string | undefined, label: string, hidden = false) => ({ id, area: 'row', kind, key, label, position: 1, required: false, hidden });
	const forms = [
		{
			name: 'PFMEA v1',
			version: {
				fields: [
					field('a', 'process', undefined, 'Process Step'),
					field('b', 'text', 'failure_mode', 'Potential Failure Mode'),
					field('c', 'rating', 'severity', 'Severity'),
					field('d', 'text', undefined, 'Added by admin'),
					field('e', 'rpn', 'rpn', 'RPN'),
					{ ...field('h', 'text', 'core_team', 'Core Team'), area: 'header' }
				]
			} as unknown as FormatVersion
		},
		{
			name: 'PFMEA v2',
			version: {
				fields: [
					field('f', 'text', 'failure_mode', '2. Failure Mode (FM)'),
					field('g', 'select', 'work_element', '3. Process Work Element (4M)'),
					field('i', 'text', 'function_process_item', '1. Function of the Process Item'),
					field('j', 'text', 'note', 'Note', true)
				]
			} as unknown as FormatVersion
		}
	];

	it('menggabungkan kolom ber-key dari semua form; kolom yang sama muncul sekali', () => {
		expect(pointColumns(forms)).toEqual([
			{ key: 'failure_mode', kind: 'text', label: 'Potential Failure Mode', forms: ['PFMEA v1', 'PFMEA v2'] },
			{ key: 'severity', kind: 'rating', label: 'Severity', forms: ['PFMEA v1'] },
			{ key: 'function_process_item', kind: 'text', label: '1. Function of the Process Item', forms: ['PFMEA v2'] }
		]);
	});

	it('contentInput membuang isian kosong dan mengirim rating sebagai angka', () => {
		const columns = pointColumns(forms);
		expect(contentInput(columns, { failure_mode: ' Wrong part ', severity: '7', function_process_item: ' ' }, { other_key: 'kept' })).toEqual({
			failure_mode: 'Wrong part',
			severity: 7,
			other_key: 'kept'
		});
	});
});
