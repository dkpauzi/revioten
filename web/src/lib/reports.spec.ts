import { describe, expect, it } from 'vitest';
import { compactNumbers, delta, formatDay, hoursParts, overdueTypes, shares } from './reports.ts';

describe('formatDay', () => {
	it('menulis tanggal dengan nama bulan', () => {
		expect(formatDay('2026-08-02')).toBe('2 Aug 2026');
		expect(formatDay('2026-08-02', 'id')).toBe('2 Agu 2026');
	});
});

describe('compactNumbers', () => {
	it('meringkas nomor satu set', () => {
		expect(compactNumbers(['SH-KT-005', 'SF-KT-005', 'SC-KT-005'])).toBe('SH/SF/SC-KT-005');
	});
	it('menulis nomor yang berbeda apa adanya', () => {
		expect(compactNumbers(['SC-KT-005', 'SC-KT-006'])).toBe('SC-KT-005, SC-KT-006');
		expect(compactNumbers(['SC-KT-005'])).toBe('SC-KT-005');
	});
});

describe('overdueTypes', () => {
	it('memberi jenis dokumen yang lewat jatuh tempo', () => {
		expect(overdueTypes([
			{ document_type: 'pfd', overdue: true },
			{ document_type: 'pfmea', overdue: false },
			{ document_type: 'cp', overdue: true }
		])).toEqual(['pfd', 'cp']);
	});
});

describe('hoursParts', () => {
	it('memakai jam di bawah dua hari dan hari sesudahnya', () => {
		expect(hoursParts(3.24)).toEqual({ value: '3.2', unit: 'hours' });
		expect(hoursParts(108)).toEqual({ value: '4.5', unit: 'days' });
	});
});

describe('delta dan shares', () => {
	it('menghitung selisih dan arah', () => {
		expect(delta(7, 5)).toEqual({ diff: 2, direction: 'up' });
		expect(delta(4.5, 5.3)).toEqual({ diff: 0.8, direction: 'down' });
		expect(delta(2, 2)).toEqual({ diff: 0, direction: 'same' });
	});
	it('membagi lebar batang menjadi 100 persen', () => {
		expect(shares([1, 1, 2])).toEqual([25, 25, 50]);
		expect(shares([0, 0])).toEqual([0, 0]);
	});
});
