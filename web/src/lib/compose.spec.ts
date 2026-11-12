// Test logika panel perakitan (M4.5, R-CMP-01 sampai R-CMP-03).
import { describe, expect, it } from 'vitest';
import type { Composition } from './api/gen/index.ts';
import { composeInput, initialChecks, publishedChoices, resultRows } from './compose.ts';

const cp = { id: 'dt-cp', key: 'cp' as const, name: 'CP' };
const pfmea = { id: 'dt-pfmea', key: 'pfmea' as const, name: 'PFMEA' };

const view: Composition = {
	process_template_version_id: 'pv-1',
	version: 1,
	process: { id: 'p-1', name: 'IQC' },
	subprocesses: [
		{
			id: 's-1',
			name: 'IQC General',
			points: [
				{ point_id: 'a', document_type: pfmea, text: 'Wrong part', document_id: 'd-pfmea', state: 'included' },
				{ point_id: 'b', document_type: cp, text: 'Check part number', document_id: 'd-cp', state: 'included' }
			]
		},
		{
			id: 's-2',
			name: 'IQC MSD',
			points: [
				{ point_id: 'c', document_type: cp, text: 'Check MSL label', document_id: 'd-cp', state: 'excluded', reason: 'No MSD' },
				{ point_id: 'd', document_type: cp, text: 'Check HIC', document_id: 'd-cp', state: 'undecided' }
			]
		}
	]
};

describe('publishedChoices', () => {
	it('memberi versi terbit terbaru setiap process template, menurut nama proses', () => {
		const templates = [
			{
				id: 't-2',
				process: { id: 'p-2', name: 'Reflow' },
				versions: [{ id: 'r1', version: 1, status: 'draft' as const }],
				created_at: ''
			},
			{
				id: 't-1',
				process: { id: 'p-1', name: 'IQC' },
				versions: [
					{ id: 'v1', version: 1, status: 'published' as const },
					{ id: 'v2', version: 2, status: 'published' as const },
					{ id: 'v3', version: 3, status: 'draft' as const }
				],
				created_at: ''
			}
		];
		expect(publishedChoices(templates)).toEqual([{ versionId: 'v2', version: 2, process: 'IQC' }]);
	});
});

describe('initialChecks, composeInput, resultRows', () => {
	it('poin yang sudah punya baris tercentang; lainnya tidak', () => {
		expect(initialChecks(view)).toEqual({ a: true, b: true, c: false, d: false });
	});

	it('centang menjadi include, tanpa centang menjadi exclude dengan alasannya', () => {
		const checks = { a: true, b: false, c: false, d: true };
		expect(composeInput(view, checks, { b: ' Not for this model ', c: 'No MSD', d: 'ignored' })).toEqual({
			process_template_version_id: 'pv-1',
			include: ['a', 'd'],
			exclude: [{ point_id: 'b', reason: 'Not for this model' }, { point_id: 'c', reason: 'No MSD' }]
		});
		expect(composeInput(view, { a: true, b: true, c: false, d: false }, {}).exclude).toEqual([{ point_id: 'c' }, { point_id: 'd' }]);
	});

	it('panel kanan: baris hasil per dokumen, dengan tanda akan ditambah atau dihapus', () => {
		const checks = { a: true, b: false, c: false, d: true };
		expect(resultRows(view, checks, 'd-cp')).toEqual([
			{ pointId: 'b', subprocess: 'IQC General', text: 'Check part number', change: 'remove' },
			{ pointId: 'd', subprocess: 'IQC MSD', text: 'Check HIC', change: 'add' }
		]);
		expect(resultRows(view, checks, 'd-pfmea')).toEqual([{ pointId: 'a', subprocess: 'IQC General', text: 'Wrong part', change: 'keep' }]);
	});
});
