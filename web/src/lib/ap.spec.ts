import { describe, expect, it } from 'vitest';
import type { ApRule } from './api/gen/index.ts';
import { actionPriority, apMatrix } from './ap.ts';

// Tabel kecil: S 6-10 dengan O 6-10 tinggi di D 6-10 dan sedang di D 1-5; sisanya rendah.
const rule = (s: [number, number], o: [number, number], d: [number, number], priority: ApRule['priority']): ApRule => ({
	s_min: s[0], s_max: s[1], o_min: o[0], o_max: o[1], d_min: d[0], d_max: d[1], priority
});
const rules: ApRule[] = [
	rule([6, 10], [6, 10], [6, 10], 'H'),
	rule([6, 10], [6, 10], [1, 5], 'M'),
	rule([6, 10], [1, 5], [1, 10], 'L'),
	rule([1, 5], [1, 10], [1, 10], 'L')
];

describe('actionPriority', () => {
	it('mencari AP dari aturan yang berlaku', () => {
		expect(actionPriority(rules, 8, 7, 9)).toBe('H');
		expect(actionPriority(rules, 8, 7, 5)).toBe('M');
		expect(actionPriority(rules, 3, 10, 10)).toBe('L');
	});

	it('mengembalikan null di luar tabel', () => {
		expect(actionPriority(rules, 0, 5, 5)).toBeNull();
		expect(actionPriority([], 5, 5, 5)).toBeNull();
	});
});

describe('apMatrix', () => {
	it('menyusun baris S dan O dari yang tertinggi, dengan kolom rentang D', () => {
		const m = apMatrix(rules);
		expect(m.detection).toEqual(['6–10', '1–5']);
		expect(m.rows).toEqual([
			{ severity: '6–10', occurrence: '6–10', cells: ['H', 'M'] },
			{ severity: '6–10', occurrence: '1–5', cells: ['L', 'L'] },
			{ severity: '1–5', occurrence: '6–10', cells: ['L', 'L'] },
			{ severity: '1–5', occurrence: '1–5', cells: ['L', 'L'] }
		]);
	});

	it('menulis rentang satu nilai tanpa tanda pisah', () => {
		const m = apMatrix([rule([1, 1], [1, 10], [1, 10], 'L'), rule([2, 10], [1, 10], [1, 10], 'M')]);
		expect(m.rows.map((r) => r.severity)).toEqual(['2–10', '1']);
	});
});
