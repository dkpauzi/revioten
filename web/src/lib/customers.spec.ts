import { describe, expect, it } from 'vitest';
import { failureText, formatDateTime, problemText } from './customers.ts';
import { en } from './i18n/en.ts';

describe('problemText', () => {
	it('memakai teks layar untuk masalah yang dikenal', () => {
		expect(problemText({ field: 'code', code: 'taken', message: 'pesan server' })).toBe(
			en.customerCodeTaken
		);
		expect(problemText({ field: 'name', code: 'too_long', message: 'pesan server' })).toBe(
			en.customerNameTooLong
		);
	});

	it('memakai pesan dari server untuk masalah yang belum dikenal', () => {
		expect(problemText({ field: 'lain', code: 'required', message: 'pesan server' })).toBe(
			'pesan server'
		);
	});
});

describe('failureText', () => {
	it('memberi tahu bila server tidak bisa dihubungi', () => {
		expect(failureText({ status: 0, body: null })).toBe(en.errorNetwork);
	});

	it('menyebut request ID bila server mengirim error', () => {
		const text = failureText({
			status: 500,
			body: { code: 'internal_error', message: 'x', request_id: 'UJI500' }
		});
		expect(text).toContain('UJI500');
	});

	it('memakai pesan umum bila isi error tidak ada', () => {
		expect(failureText({ status: 502, body: null })).toBe(en.errorUnknown);
	});
});

describe('formatDateTime', () => {
	it('menulis tanggal dan jam lokal dalam bentuk YYYY-MM-DD HH:mm', () => {
		const iso = '2026-10-28T03:04:00Z';
		const d = new Date(iso);
		const pad = (n: number) => String(n).padStart(2, '0');
		const want = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
		expect(formatDateTime(iso)).toBe(want);
	});
});
