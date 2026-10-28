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

	it('memakai teks khusus formulir bila ada, lalu teks umum field itu', () => {
		const taken = { field: 'code', code: 'taken', message: 'pesan server' } as const;
		expect(problemText(taken, 'projectGroup')).toBe(en.projectGroupCodeTaken);
		expect(problemText(taken)).toBe(en.customerCodeTaken);
		expect(problemText({ field: 'name', code: 'required', message: 'x' }, 'model')).toBe(
			en.modelNameRequired
		);
		expect(problemText({ field: 'project_group_id', code: 'required', message: 'x' })).toBe(
			en.customerProjectGroupRequired
		);
	});

	// Nama proses dan alias berbagi satu ruang nama (M1.5, ASUMSI E-32).
	it('memberi teks master proses', () => {
		expect(problemText({ field: 'name', code: 'taken', message: 'x' }, 'process')).toBe(
			en.processNameTaken
		);
		expect(problemText({ field: 'name', code: 'required', message: 'x' }, 'process')).toBe(
			en.processNameRequired
		);
		expect(problemText({ field: 'aliases', code: 'taken', message: 'x' })).toBe(en.processAliasTaken);
		expect(problemText({ field: 'aliases', code: 'too_long', message: 'x' })).toBe(
			en.processAliasTooLong
		);
		expect(problemText({ field: 'aliases', code: 'invalid_format', message: 'x' })).toBe(
			en.processAliasInvalid
		);
	});

	// Jenis dokumen, kode khusus, dan daftar perubahan (M1.6).
	it('memberi teks jenis dokumen dan perubahan', () => {
		expect(problemText({ field: 'factory_code', code: 'invalid_format', message: 'x' })).toBe(
			en.factoryCodeInvalid
		);
		expect(problemText({ field: 'letter', code: 'taken', message: 'x' })).toBe(en.documentTypeLetterTaken);
		expect(problemText({ field: 'name', code: 'taken', message: 'x' }, 'documentType')).toBe(
			en.documentTypeNameTaken
		);
		expect(problemText({ field: 'code', code: 'taken', message: 'x' }, 'specialCode')).toBe(
			en.specialCodeTaken
		);
		expect(problemText({ field: 'code', code: 'invalid_format', message: 'x' }, 'specialCode')).toBe(
			en.specialCodeInvalid
		);
		expect(problemText({ field: 'name', code: 'taken', message: 'x' }, 'changeList')).toBe(
			en.changeListNameTaken
		);
		expect(problemText({ field: 'active', code: 'not_allowed', message: 'x' })).toBe(
			en.changeSourceSystemActive
		);
	});

	// Jabatan dan rute approval (M1.8).
	it('memberi teks jabatan dan rute approval', () => {
		expect(problemText({ field: 'project_group_id', code: 'taken', message: 'x' }, 'holder')).toBe(
			en.holderGroupTaken
		);
		expect(problemText({ field: 'project_group_id', code: 'required', message: 'x' }, 'holder')).toBe(
			en.holderGroupRequired
		);
		expect(problemText({ field: 'person_id', code: 'taken', message: 'x' })).toBe(en.holderPersonTaken);
		expect(problemText({ field: 'slots', code: 'invalid_format', message: 'x' })).toBe(en.routeSlotsOrder);
		expect(problemText({ field: 'position_ids', code: 'required', message: 'x' })).toBe(
			en.routeSlotPositionsRequired
		);
		expect(problemText({ field: 'name', code: 'taken', message: 'x' }, 'route')).toBe(en.routeNameTaken);
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

	it('memberi tahu bila tidak punya hak', () => {
		expect(
			failureText({ status: 403, body: { code: 'forbidden', message: 'x', request_id: 'uji' } })
		).toBe(en.errorForbidden);
	});

	it('memakai pesan umum bila isi error tidak ada', () => {
		expect(failureText({ status: 502, body: null })).toBe(en.errorUnknown);
	});
});
