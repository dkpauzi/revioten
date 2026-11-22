import { describe, expect, it } from 'vitest';
import { emptyRows, fillNames, followingNumber, setDocuments, splitLegacyNumber, type SetRow } from './migration.ts';

describe('splitLegacyNumber (D-216)', () => {
	it('memisahkan akhiran revisi dari nomor seperti di nama berkas lama', () => {
		expect(splitLegacyNumber('SH-TR-020-02')).toEqual({ number: 'SH-TR-020', revision: 2 });
		expect(splitLegacyNumber(' sc-tr-020-04 ')).toEqual({ number: 'SC-TR-020', revision: 4 });
	});
	it('membiarkan nomor tanpa revisi', () => {
		expect(splitLegacyNumber('SH-TR-020')).toEqual({ number: 'SH-TR-020', revision: null });
		expect(splitLegacyNumber('SH-TR-A001')).toEqual({ number: 'SH-TR-A001', revision: null });
		expect(splitLegacyNumber('')).toEqual({ number: '', revision: null });
	});
});

describe('followingNumber', () => {
	it('mengganti awalan nomor sumber dengan awalan jenis lain', () => {
		expect(followingNumber('SH-TR-020', 'SH', 'SC')).toBe('SC-TR-020');
		expect(followingNumber('SH-TR-020', 'SF', 'SC')).toBe('');
		expect(followingNumber('', 'SH', 'SC')).toBe('');
	});
});

describe('setDocuments', () => {
	const filled = (): SetRow[] =>
		emptyRows().map((r, i) => ({ ...r, revision: String(i + 2), effectiveDate: '2025-10-23', preparedBy: 'Budi', approvedBy: 'Siti' }));

	it('satu set: ketiga dokumen dikirim, nomor kosong mengikuti di server', () => {
		const rows = filled();
		rows[0].number = 'SH-TR-020';
		const docs = setDocuments(rows, true);
		expect(docs.map((d) => [d.document_type_key, d.number, d.revision])).toEqual([
			['pfd', 'SH-TR-020', 2],
			['pfmea', '', 3],
			['cp', '', 4]
		]);
	});
	it('bukan satu set: hanya baris bernomor', () => {
		const rows = filled();
		rows[2].number = 'SC-TR-030';
		expect(setDocuments(rows, false).map((d) => d.number)).toEqual(['SC-TR-030']);
	});
	it('revisi dari input angka (number) juga terbaca', () => {
		const rows = filled();
		rows[0].number = 'SH-TR-020';
		rows[0].revision = 4;
		rows[1].revision = null;
		expect(setDocuments(rows, true).map((d) => d.revision)).toEqual([4, -1, 4]);
	});
	it('revisi kosong dikirim -1 supaya server menolaknya dengan pesan jelas', () => {
		const rows = filled();
		rows[0].number = 'SH-TR-020';
		rows[1].revision = ' ';
		expect(setDocuments(rows, true)[1].revision).toBe(-1);
	});
});

describe('fillNames', () => {
	it('menyalin nama baris pertama ke baris yang masih kosong', () => {
		const rows = emptyRows();
		rows[0].preparedBy = 'Budi';
		rows[0].approvedBy = 'Siti';
		rows[2].approvedBy = 'Agus';
		const out = fillNames(rows);
		expect(out.map((r) => [r.preparedBy, r.approvedBy])).toEqual([
			['Budi', 'Siti'],
			['Budi', 'Siti'],
			['Budi', 'Agus']
		]);
	});
});
