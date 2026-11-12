// Test tempel dari Excel (M4.4, R-MIG-02).
import { describe, expect, it } from 'vitest';
import type { FormatVersion, Process } from './api/gen/index.ts';
import { buildRows, guessMapping, parseClipboard, pasteTargets } from './paste.ts';

const format = {
	fields: [
		{ id: 'op', area: 'row', kind: 'operation_number', label: 'Process No.', position: 1, required: false, hidden: false },
		{ id: 'proc', area: 'row', kind: 'process', label: 'Process Name', position: 2, required: false, hidden: false },
		{ id: 'txt', area: 'row', kind: 'text', label: 'Product Characteristics', position: 3, required: false, hidden: false },
		{ id: 'sev', area: 'row', kind: 'rating', label: 'Severity', position: 4, required: false, hidden: false },
		{ id: 'rpn', area: 'row', kind: 'rpn', label: 'RPN', position: 5, required: false, hidden: false },
		{
			id: 'sel',
			area: 'row',
			kind: 'select',
			label: 'Class',
			position: 6,
			required: false,
			hidden: false,
			options: [
				{ key: 'cc', label: 'Critical' },
				{ key: 'sc', label: 'Significant' }
			]
		},
		{ id: 'sym', area: 'row', kind: 'symbol', label: 'Symbol', position: 7, required: false, hidden: false },
		{ id: 'dat', area: 'row', kind: 'date', label: 'Target Date', position: 8, required: false, hidden: false },
		{ id: 'hid', area: 'row', kind: 'text', label: 'Hidden', position: 9, required: false, hidden: true },
		{ id: 'head', area: 'header', kind: 'text', label: 'Core Team', position: 1, required: false, hidden: false }
	],
	symbols: [
		{ key: 'safety', name: 'Safety' },
		{ key: 'rev', name: 'Revision', automatic: true }
	]
} as unknown as FormatVersion;

const processes = [{ id: 'p-1', name: 'Reflow', aliases: ['Reflow oven'], created_at: '' }] as Process[];

describe('parseClipboard', () => {
	it('memecah baris dan kolom teks dari Excel', () => {
		expect(parseClipboard('10\tReflow\tSolder joint\r\n20\tAOI\t\r\n')).toEqual([
			['10', 'Reflow', 'Solder joint'],
			['20', 'AOI', '']
		]);
	});

	it('menjaga sel berkutip yang berisi baris baru, tab, dan tanda kutip', () => {
		expect(parseClipboard('10\t"Line 1\nLine 2"\t"Say ""ok"""\n20\t"a\tb"\tc')).toEqual([
			['10', 'Line 1\nLine 2', 'Say "ok"'],
			['20', 'a\tb', 'c']
		]);
	});

	it('membuang baris yang semua selnya kosong', () => {
		expect(parseClipboard('a\tb\n\t\n\nc\td')).toEqual([
			['a', 'b'],
			['c', 'd']
		]);
	});
});

describe('pasteTargets dan guessMapping', () => {
	const targets = pasteTargets(format);

	it('memberi kolom yang bisa diisi: nomor operasi, proses, dan kolom yang diketik', () => {
		expect(targets.map((x) => x.id)).toEqual(['_operation', '_process', 'txt', 'sev', 'sel', 'sym', 'dat']);
	});

	it('tanpa baris judul, kolom dipetakan menurut urutan', () => {
		expect(guessMapping(null, targets, 3)).toEqual(['_operation', '_process', 'txt']);
		expect(guessMapping(null, targets, 9)).toEqual(['_operation', '_process', 'txt', 'sev', 'sel', 'sym', 'dat', '', '']);
	});

	it('dengan baris judul, kolom dipetakan menurut label; yang tidak dikenal dilewati', () => {
		expect(guessMapping(['severity', ' Product characteristics ', 'Remarks', 'Process No.'], targets, 4)).toEqual(['sev', 'txt', '', '_operation']);
	});
});

describe('buildRows', () => {
	it('mengubah sel menjadi isian baris menurut jenis kolom', () => {
		const grid = [['10', 'reflow oven', ' Solder joint ', '8', 'critical', 'Safety', '2026-11-13']];
		const result = buildRows(grid, ['_operation', '_process', 'txt', 'sev', 'sel', 'sym', 'dat'], format, processes);
		expect(result.flags).toEqual([]);
		expect(result.rows).toEqual([
			{ operation_number: '10', process_id: 'p-1', content: { txt: 'Solder joint', sev: 8, sel: 'cc', sym: 'safety', dat: '2026-11-13' } }
		]);
	});

	it('menandai sel janggal tetapi barisnya tetap masuk dengan sel itu kosong', () => {
		const grid = [
			['10', 'Wave', 'Joint', '11', 'Minor', 'Revision', '13/11/2026'],
			['', '', 'Second', '', '', '', '13-Nov-26']
		];
		const result = buildRows(grid, ['_operation', '_process', 'txt', 'sev', 'sel', 'sym', 'dat'], format, processes);
		expect(result.rows).toEqual([
			{ operation_number: '10', content: { txt: 'Joint' } },
			{ content: { txt: 'Second', dat: '2026-11-13' } }
		]);
		expect(result.flags).toEqual([
			{ row: 0, column: 1, reason: 'process' },
			{ row: 0, column: 3, reason: 'rating' },
			{ row: 0, column: 4, reason: 'option' },
			{ row: 0, column: 5, reason: 'symbol' },
			{ row: 0, column: 6, reason: 'date' }
		]);
	});

	it('melewati kolom yang tidak dipetakan dan menandai teks yang terlalu panjang', () => {
		const long = 'x'.repeat(2001);
		const result = buildRows([['abcdefghijklmnopqrstuvwxyz', 'skip me', long]], ['_operation', '', 'txt'], format, processes);
		expect(result.rows).toEqual([{ content: {} }]);
		expect(result.flags).toEqual([
			{ row: 0, column: 0, reason: 'too_long' },
			{ row: 0, column: 2, reason: 'too_long' }
		]);
	});
});
