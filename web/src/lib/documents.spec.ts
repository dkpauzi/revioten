import { describe, expect, it } from 'vitest';
import type { FormatField } from './api/gen/index.ts';
import { cellText, cleanContent, gridRow, rowInputFrom, typedFields } from './documents.ts';

const field = (over: Partial<FormatField>): FormatField => ({
	id: 'f', area: 'row', kind: 'text', label: 'L', position: 1, required: false, hidden: false, ...over
});
const fields = [
	field({ id: 'op', kind: 'operation_number', position: 1 }),
	field({ id: 'proc', kind: 'process', position: 2 }),
	field({ id: 'txt', kind: 'text', position: 3 }),
	field({ id: 'sel', kind: 'select', position: 4, options: [{ key: 'a', label: 'Alpha' }] }),
	field({ id: 'sev', kind: 'rating', rating_scale: 'severity', rating_set: 1, position: 5 }),
	field({ id: 'rpn', kind: 'rpn', rating_set: 1, position: 6 }),
	field({ id: 'hid', kind: 'text', hidden: true, position: 7 }),
	field({ id: 'head', area: 'header', kind: 'text', position: 1 })
];

describe('typedFields', () => {
	it('memberi kolom yang diketik penyusun: bukan terisi otomatis, bukan dihitung, tidak tersembunyi', () => {
		expect(typedFields(fields, 'row').map((f) => f.id)).toEqual(['txt', 'sel', 'sev']);
		expect(typedFields(fields, 'header').map((f) => f.id)).toEqual(['head']);
	});
});

describe('cleanContent', () => {
	it('membuang isian kosong dan mengubah rating menjadi angka', () => {
		expect(cleanContent(fields, { txt: ' Paste ', sel: '', sev: '8' })).toEqual({ txt: 'Paste', sev: 8 });
	});
});

describe('cellText', () => {
	const row = { id: 'r', position: 1, process_name: 'SMT', operation_number: '10', content: { txt: 'Paste', sel: 'a' } };
	it('membaca kolom terisi otomatis dari baris dan label pilihan dari format', () => {
		expect(cellText(fields[0], row)).toBe('10');
		expect(cellText(fields[1], row)).toBe('SMT');
		expect(cellText(fields[3], row)).toBe('Alpha');
		expect(cellText(fields[2], row)).toBe('Paste');
		expect(cellText(fields[4], row)).toBe('');
	});
});

describe('gridRow dan rowInputFrom', () => {
	const format = { fields } as unknown as Parameters<typeof rowInputFrom>[0];
	it('mengubah baris menjadi data tabel dan kembali menjadi isian baris', () => {
		const data = gridRow({ id: 'r', position: 2, process_id: 'p-1', operation_number: '10', content: { txt: 'Paste', sev: 8 } });
		expect(data).toEqual({ id: 'r', position: 2, _process: 'p-1', _operation: '10', txt: 'Paste', sev: 8 });
		expect(rowInputFrom(format, { ...data, txt: ' Paste height ', sel: '' })).toEqual({
			process_id: 'p-1',
			operation_number: '10',
			content: { txt: 'Paste height', sev: 8 }
		});
	});

	it('tidak mengirim proses dan nomor operasi yang kosong', () => {
		expect(rowInputFrom(format, { id: 'r', position: 1, _process: '', _operation: ' ' })).toEqual({ content: {} });
	});
});