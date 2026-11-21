// Test pratinjau lembar (C2, D-122, G11): ukuran kertas, pembagian halaman,
// kelompok proses, dan alur Process Flow dengan konektor antar halaman.
import { describe, expect, it } from 'vitest';
import { connectorLetter, flowPages, groupsOf, paginate, paperSize, previewText, stepsOf } from './preview.ts';

describe('paperSize', () => {
	it('A3 mendatar dan A4 tegak dalam milimeter', () => {
		expect(paperSize({ paper: 'A3', orientation: 'landscape' })).toEqual({ widthMm: 420, heightMm: 297 });
		expect(paperSize({ paper: 'A4', orientation: 'portrait' })).toEqual({ widthMm: 210, heightMm: 297 });
	});
});

describe('paginate', () => {
	it('mengisi halaman sampai penuh, halaman pertama boleh lebih pendek', () => {
		// Halaman pertama muat 100, halaman berikutnya 120.
		expect(paginate([40, 40, 40, 40, 40, 40], 100, 120)).toEqual([[0, 1], [2, 3, 4], [5]]);
	});

	it('baris yang lebih tinggi dari satu halaman tetap mendapat halaman sendiri', () => {
		expect(paginate([30, 200, 30], 100, 100)).toEqual([[0], [1], [2]]);
	});

	it('dokumen tanpa baris tetap punya satu halaman kosong', () => {
		expect(paginate([], 100, 100)).toEqual([[]]);
	});
});

describe('groupsOf', () => {
	it('baris berurutan dengan nomor operasi dan proses yang sama menjadi satu kelompok', () => {
		const rows = [
			{ operation_number: '10', process_id: 'a' },
			{ operation_number: '10', process_id: 'a' },
			{ operation_number: '20', process_id: 'b' },
			{ process_id: 'b' },
			{ operation_number: '5.1', process_id: 'c' }
		];
		expect(groupsOf(rows)).toEqual([
			{ start: 0, end: 1, op: '10' },
			{ start: 2, end: 2, op: '20' },
			{ start: 3, end: 3, op: '' },
			{ start: 4, end: 4, op: '5.1' }
		]);
	});
});

describe('stepsOf', () => {
	it('membaca simbol dan arah NG langkah PFD menurut nomor langkah', () => {
		const steps = stepsOf(
			[
				{ operation_number: '1', content: { sym: 'process' } },
				{ operation_number: '2', content: { sym: 'decision', ng: 'ncp' } },
				{ content: { sym: 'process' } }
			],
			'sym',
			'ng'
		);
		expect(steps.get('1')).toEqual({ symbol: 'process', ng: undefined });
		expect(steps.get('2')).toEqual({ symbol: 'decision', ng: 'ncp' });
		expect(steps.size).toBe(2);
	});
});

describe('connectorLetter', () => {
	it('A sampai Z, lalu AA', () => {
		expect([0, 1, 25, 26, 27].map(connectorLetter)).toEqual(['A', 'B', 'Z', 'AA', 'AB']);
	});
});

describe('flowPages', () => {
	const steps = stepsOf(
		[
			{ operation_number: '1', content: { s: 'process' } },
			{ operation_number: '2', content: { s: 'decision', n: 'scrap' } },
			{ operation_number: '3', content: { s: 'process' } }
		],
		's',
		'n'
	);

	it('satu simbol per kelompok di baris pertamanya, dengan simbol dari PFD', () => {
		const groups = [
			{ start: 0, end: 1, op: '1' },
			{ start: 2, end: 2, op: '2' }
		];
		const pages = flowPages(groups, [[0, 1, 2]], steps);
		expect(pages).toEqual([
			{
				nodes: [
					{ row: 0, op: '1', symbol: 'process', ng: undefined },
					{ row: 2, op: '2', symbol: 'decision', ng: 'scrap' }
				],
				lastRow: 2,
				inLetter: undefined,
				outLetter: undefined
			}
		]);
	});

	it('konektor A dan B ditempatkan sendiri di setiap batas halaman', () => {
		const groups = [
			{ start: 0, end: 0, op: '1' },
			{ start: 1, end: 2, op: '2' },
			{ start: 3, end: 3, op: '3' }
		];
		// Kelompok 2 terpotong di batas halaman pertama.
		const pages = flowPages(groups, [[0, 1], [2], [3]], steps);
		expect(pages.map((p) => [p.inLetter, p.outLetter])).toEqual([
			[undefined, 'A'],
			['A', 'B'],
			['B', undefined]
		]);
		expect(pages[1].nodes).toEqual([]);
		expect(pages[2].nodes[0]).toEqual({ row: 0, op: '3', symbol: 'process', ng: undefined });
	});

	it('langkah yang tidak ada di PFD memakai simbol proses', () => {
		const pages = flowPages([{ start: 0, end: 0, op: '9' }], [[0]], steps);
		expect(pages[0].nodes[0].symbol).toBe('process');
	});

	it('menambah baris di awal menggeser simbol dan konektor sendiri', () => {
		const groups = [
			{ start: 0, end: 1, op: '1' },
			{ start: 2, end: 2, op: '2' },
			{ start: 3, end: 3, op: '3' }
		];
		const before = flowPages(groups, [[0, 1, 2], [3]], steps);
		expect(before[0].outLetter).toBe('A');
		expect(before[1].nodes[0].op).toBe('3');
		// Satu baris baru di kelompok 1: kelompok 2 ikut pindah ke halaman berikutnya.
		const grown = [
			{ start: 0, end: 2, op: '1' },
			{ start: 3, end: 3, op: '2' },
			{ start: 4, end: 4, op: '3' }
		];
		const after = flowPages(grown, [[0, 1, 2], [3, 4]], steps);
		expect(after[0].nodes.map((n) => n.op)).toEqual(['1']);
		expect(after[1].nodes.map((n) => n.op)).toEqual(['2', '3']);
		expect(after[1].inLetter).toBe('A');
	});
});

describe('previewText', () => {
	const fields = [
		{ id: 's', kind: 'rating', rating_set: 1, rating_scale: 'severity' },
		{ id: 'o', kind: 'rating', rating_set: 1, rating_scale: 'occurrence' },
		{ id: 'd', kind: 'rating', rating_set: 1, rating_scale: 'detection' },
		{ id: 'rpn', kind: 'rpn', rating_set: 1 },
		{ id: 'ng', kind: 'select', options: [{ key: 'scrap', label: 'Scrap' }] },
		{ id: 'sc', kind: 'symbol' }
	];
	const row = { operation_number: '5.1', process_name: 'SMT', content: { s: 8, o: 3, d: 4, ng: 'scrap', sc: 'ic' } };

	it('RPN adalah hasil kali S, O, dan D', () => {
		expect(previewText(fields[3], row, fields, [])).toBe('96');
		expect(previewText(fields[3], { ...row, content: { s: 8 } }, fields, [])).toBe('');
	});

	it('pilihan, simbol, nomor operasi, dan proses', () => {
		expect(previewText(fields[4], row, fields, [])).toBe('Scrap');
		expect(previewText(fields[5], row, fields, [{ key: 'ic', name: '<IC>' }])).toBe('<IC>');
		expect(previewText({ id: 'x', kind: 'operation_number' }, row, fields, [])).toBe('5.1');
		expect(previewText({ id: 'x', kind: 'process' }, row, fields, [])).toBe('SMT');
	});
});
