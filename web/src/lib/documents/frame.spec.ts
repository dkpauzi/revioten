// Test langkah berikutnya di kepala workbook (D-145, C1).
import { describe, expect, it } from 'vitest';
import { nextStep, type StepInput } from './frame.ts';

const draft: StepInput = { status: 'draft', rows: 3, hasRoute: true, canEdit: true, canPropose: true, canRecord: true };

describe('nextStep', () => {
	it('dokumen kosong: rakit dari process template', () => {
		expect(nextStep({ ...draft, rows: 0 })).toEqual({ text: 'nextCompose', open: 'compose' });
	});

	it('draft berisi: kirim untuk tanda tangan', () => {
		expect(nextStep(draft)).toEqual({ text: 'nextSign', open: 'signature' });
	});

	it('draft berisi tanpa rute approval: cetak dulu', () => {
		expect(nextStep({ ...draft, hasRoute: false })).toEqual({ text: 'nextPrint', open: 'print' });
	});

	it('menunggu tanda tangan digital: lihat siapa yang ditunggu', () => {
		expect(nextStep({ ...draft, signing: 'waiting' })).toEqual({ text: 'nextWaiting', open: 'signature' });
	});

	it('sudah ditandatangani lengkap: catat rilis', () => {
		expect(nextStep({ ...draft, signing: 'completed' })).toEqual({ text: 'nextRecord', open: 'revisions' });
	});

	it('dokumen lama belum diverifikasi: cocokkan dengan hardcopy', () => {
		expect(nextStep({ ...draft, migrated: true, verified: false })).toEqual({ text: 'nextVerify', open: 'migration' });
	});

	it('dokumen terbit: perubahan lewat permintaan perubahan', () => {
		expect(nextStep({ ...draft, status: 'released' })).toEqual({ text: 'nextChange', href: '/changes' });
	});

	it('akun yang hanya membaca tidak mendapat langkah', () => {
		expect(nextStep({ ...draft, canEdit: false, canPropose: false, canRecord: false })).toBeNull();
		expect(nextStep({ ...draft, status: 'released', canPropose: false })).toBeNull();
	});

	it('akun tanpa hak ubah tetap diberi tahu bila tanda tangan lengkap dan ia boleh mencatat rilis', () => {
		expect(nextStep({ ...draft, canEdit: false, signing: 'completed' })).toEqual({ text: 'nextRecord', open: 'revisions' });
	});
});
