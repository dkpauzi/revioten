// Test penyusun diagram PFD (M4.10).
import { describe, expect, it } from 'vitest';
import type { DocumentRow, FormatVersion } from './api/gen/index.ts';
import { diagramOf } from './pfdDiagram.ts';

const format = {
	fields: [
		{ id: 'op', area: 'row', kind: 'operation_number', label: 'No', position: 1, required: false, hidden: false },
		{ id: 'pr', area: 'row', kind: 'process', label: 'Process', position: 2, required: false, hidden: false },
		{
			id: 'sym',
			key: 'flow_symbol',
			area: 'row',
			kind: 'select',
			label: 'Flow Symbol',
			position: 3,
			required: false,
			hidden: false,
			options: [
				{ key: 'process', label: 'Process' },
				{ key: 'decision', label: 'Decision' }
			]
		},
		{
			id: 'ng',
			key: 'ng_disposition',
			area: 'row',
			kind: 'select',
			label: 'NG Disposition',
			position: 4,
			required: false,
			hidden: false,
			options: [
				{ key: 'scrap', label: 'Scrap' },
				{ key: 'rework', label: 'Rework' }
			]
		}
	]
} as unknown as FormatVersion;

const row = (id: string, operation: string, process: string, content: Record<string, unknown> = {}): DocumentRow => ({
	id,
	position: 0,
	version: 1,
	operation_manual: false,
	detached: false,
	operation_number: operation || undefined,
	process_id: process ? `p-${process}` : undefined,
	process_name: process || undefined,
	content
});

describe('diagramOf', () => {
	it('lompatan menjadi panah berlabel ke langkah tujuan (D-227)', () => {
		const withJump = {
			fields: [
				...format.fields,
				{ id: 'jt', key: 'jump_to', area: 'row', kind: 'text', label: 'Jump to', position: 5, required: false, hidden: false },
				{ id: 'jl', key: 'jump_label', area: 'row', kind: 'select', label: 'Jump label', position: 6, required: false, hidden: false, options: [{ key: 'no', label: 'No' }] }
			]
		} as unknown as FormatVersion;
		const d = diagramOf([row('a', '10', 'Loading PCB'), row('b', '20', 'Finish 2nd side', { sym: 'decision', jt: '10', jl: 'no' }), row('c', '30', 'ICT', { jt: '99' })], withJump);
		expect(d.edges.filter((e) => e.id.includes('-jump-'))).toEqual([{ id: 'b-jump-a', source: 'b', target: 'a', ng: false, label: 'No' }]);
	});

	it('satu simpul per baris menurut urutan, dihubungkan berurutan', () => {
		const d = diagramOf([row('a', '10', 'Receiving'), row('b', '20', 'Inspection', { sym: 'decision', ng: 'scrap' }), row('c', '', '')], format);
		expect(d.nodes).toEqual([
			{ id: 'a', label: '10 Receiving', symbol: 'process', symbolLabel: 'Process' },
			{ id: 'b', label: '20 Inspection', symbol: 'decision', symbolLabel: 'Decision' },
			{ id: 'ng-b', label: 'Scrap', symbol: 'ng', symbolLabel: 'NG Disposition' },
			{ id: 'c', label: '', symbol: 'process', symbolLabel: 'Process' }
		]);
		expect(d.edges).toEqual([
			{ id: 'a-b', source: 'a', target: 'b', ng: false },
			{ id: 'b-ng-b', source: 'b', target: 'ng-b', ng: true, label: 'NG' },
			{ id: 'b-c', source: 'b', target: 'c', ng: false }
		]);
	});

	it('tanpa kolom simbol semua simpul memakai simbol process', () => {
		const plain = { fields: format.fields.slice(0, 2) } as unknown as FormatVersion;
		expect(diagramOf([row('a', '10', 'X')], plain).nodes).toEqual([{ id: 'a', label: '10 X', symbol: 'process', symbolLabel: '' }]);
	});

	it('dokumen kosong memberi diagram kosong', () => {
		expect(diagramOf([], format)).toEqual({ nodes: [], edges: [] });
	});
});