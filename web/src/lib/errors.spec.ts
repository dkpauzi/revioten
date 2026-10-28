import { describe, expect, it } from 'vitest';
import { failureText, problemText } from './errors.ts';
import { en } from './i18n/en.ts';

describe('problemText', () => {
	it('memakai teks layar untuk masalah yang dikenal', () => {
		expect(problemText({ field: 'code', code: 'taken', message: 'pesan server' })).toBe(
			en.customerCodeTaken
		);
		expect(problemText({ field: 'name', code: 'too_long', message: 'pesan server' })).toBe(
			en.customerNameTooLong
		);
		expect(
			problemText({ field: 'current_password', code: 'incorrect', message: 'pesan server' })
		).toBe(en.passwordCurrentIncorrect);
		expect(problemText({ field: 'new_password', code: 'too_common', message: 'pesan server' })).toBe(
			en.passwordTooCommon
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
