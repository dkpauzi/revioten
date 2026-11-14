// Test logika panel urutan (M4.6, R-CMP-01).
import { describe, expect, it } from 'vitest';
import type { DocumentRow } from './api/gen/index.ts';
import { moveStep, stepsOf } from './order.ts';

const row = (id: string, process: string, operation = '', manual = false): DocumentRow => ({
	id,
	position: 0,
	version: 1,
	operation_manual: manual,
	process_id: process || undefined,
	process_name: process ? `Process ${process}` : undefined,
	operation_number: operation || undefined,
	content: {}
});

const rows = [row('a', 'p1', '10'), row('b', 'p1', '10'), row('c', 'p2', '20'), row('d', ''), row('e', 'p1', '45', true)];

describe('stepsOf', () => {
	it('baris berurutan dengan proses yang sama adalah satu langkah; tanpa proses berdiri sendiri', () => {
		expect(stepsOf(rows)).toEqual([
			{ rowIds: ['a', 'b'], processName: 'Process p1', current: '10', next: '10' },
			{ rowIds: ['c'], processName: 'Process p2', current: '20', next: '20' },
			{ rowIds: ['d'], processName: '', current: '', next: '30' },
			{ rowIds: ['e'], processName: 'Process p1', current: '45', next: '45' }
		]);
	});
});

describe('moveStep', () => {
	it('memindahkan satu langkah beserta semua barisnya', () => {
		const moved = moveStep(rows, 1, 0);
		expect(moved.map((r) => r.id)).toEqual(['c', 'a', 'b', 'd', 'e']);
		expect(stepsOf(moved).map((s) => s.next)).toEqual(['10', '20', '30', '45']);
	});

	it('dua langkah proses yang sama yang menjadi bersebelahan bergabung', () => {
		const moved = moveStep(rows, 3, 1);
		expect(moved.map((r) => r.id)).toEqual(['a', 'b', 'e', 'c', 'd']);
		// Nomor yang diketik (45) tetap; baris lain di langkah itu mendapat 10.
		expect(stepsOf(moved)[0]).toEqual({ rowIds: ['a', 'b', 'e'], processName: 'Process p1', current: '10, 45', next: '10, 45' });
	});

	it('pindah ke luar batas tidak mengubah apa pun', () => {
		expect(moveStep(rows, 0, -1)).toBe(rows);
		expect(moveStep(rows, 3, 4)).toBe(rows);
	});
});