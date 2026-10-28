import { describe, expect, it } from 'vitest';
import { aliasLines, aliasText } from './processes.ts';

describe('alias proses di kotak teks', () => {
	// Satu alias per baris, karena nama proses bisa memuat koma.
	it('memecah teks menjadi alias, tanpa baris kosong', () => {
		expect(aliasLines('Part Receiving\n\n  Parts Receive  \r\nReceiving, Store\n')).toEqual([
			'Part Receiving',
			'Parts Receive',
			'Receiving, Store'
		]);
		expect(aliasLines('')).toEqual([]);
		expect(aliasLines(' \n ')).toEqual([]);
	});

	it('menulis alias sebagai satu alias per baris', () => {
		expect(aliasText(['Part Receiving', 'Receiving, Store'])).toBe('Part Receiving\nReceiving, Store');
		expect(aliasText([])).toBe('');
	});
});
