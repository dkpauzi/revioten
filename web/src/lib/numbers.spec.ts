import { describe, expect, it } from 'vitest';
import type { DocumentNumber } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import {
	canObsolete,
	canVoid,
	lineResultText,
	notImported,
	rangesText,
	registeredText,
	reservedDays,
	skippedText,
	statusLabel,
	typeNames
} from './numbers.ts';

// number membuat satu nomor contoh; hanya field yang dipakai yang diisi berbeda.
function number(text: string, revision: number, status: DocumentNumber['status'] = 'reserved'): DocumentNumber {
	return {
		id: text,
		number: text,
		customer: { id: 'c-1', code: 'AB', name: 'Customer AB' },
		customer_code: 'AB',
		document_type: { id: 'dt-3', key: 'cp', name: 'CP' },
		sequence: 1,
		status,
		initial_revision: revision,
		revision_label: `-${String(revision).padStart(2, '0')}`,
		origin: 'single',
		batch_id: 'b-1',
		registered_by: { id: 'p-1', name: 'Penyusun Contoh' },
		created_at: '2026-10-30T00:00:00Z'
	};
}

describe('teks register nomor', () => {
	it('menulis nomor hasil pendaftaran, dengan revisi awal bila bukan 0', () => {
		expect(registeredText([number('SC-AB-001', 0)])).toBe('SC-AB-001');
		expect(registeredText([number('SH-AB-004', 4), number('SC-AB-004', 4)])).toBe('SH-AB-004 (-04), SC-AB-004 (-04)');
	});

	it('tidak menyebut nomor yang dilewati, tapi menulisnya terpisah', () => {
		const set = [number('SH-AB-002', 0, 'skipped'), number('SH-AB-003', 0)];
		expect(registeredText(set)).toBe('SH-AB-003');
		expect(skippedText(set)).toBe('SH-AB-002');
		expect(skippedText([number('SC-AB-001', 0)])).toBe('');
	});
});

describe('status nomor (M2.3)', () => {
	it('memberi nama status dan tombol yang boleh', () => {
		expect(statusLabel('in_use')).toBe(en.numberStatusInUse);
		expect(canVoid(number('SC-AB-001', 0, 'reserved'))).toBe(true);
		expect(canVoid(number('SC-AB-001', 0, 'released'))).toBe(false);
		expect(canObsolete(number('SC-AB-001', 0, 'released'))).toBe(true);
		expect(canObsolete(number('SC-AB-001', 0, 'void'))).toBe(false);
	});

	it('menghitung hari penuh sejak nomor didaftarkan', () => {
		expect(reservedDays('2026-10-01T08:00:00Z', new Date('2026-10-30T07:59:59Z'))).toBe(28);
		expect(reservedDays('2026-10-30T08:00:00Z', new Date('2026-10-30T09:00:00Z'))).toBe(0);
	});
});

describe('laporan impor nomor lama (M2.5)', () => {
	it('menulis rentang nomor berjalan, satu nomor tanpa tanda pisah', () => {
		expect(rangesText([])).toBe('');
		expect(
			rangesText([
				{ first: '003', last: '006' },
				{ first: '010', last: '010' }
			])
		).toBe('003–006, 010');
	});

	it('menulis nama jenis dokumen yang punya nomor', () => {
		expect(
			typeNames([
				{ id: 'dt-1', key: 'pfd', name: 'PFD' },
				{ id: 'dt-3', key: 'cp', name: 'CP' }
			])
		).toBe('PFD, CP');
	});

	it('menulis hasil baris, dan nomor ganda menunjuk baris pertamanya', () => {
		expect(lineResultText({ line: 9, text: 'SC-AB-002', result: 'duplicate', first_line: 8 })).toBe(
			en.numberImportLineDuplicate.replace('{line}', '8')
		);
		expect(lineResultText({ line: 2, text: 'hello', result: 'invalid' })).toBe(en.numberImportLineInvalid);
	});

	it('hanya baris yang tidak masuk yang ditampilkan di tabel', () => {
		const lines = [
			{ line: 1, text: 'SC-AB-001', result: 'imported' as const },
			{ line: 2, text: 'SC-AB-001', result: 'duplicate' as const, first_line: 1 }
		];
		expect(notImported(lines).map((l) => l.line)).toEqual([2]);
	});
});
