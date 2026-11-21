// Test langkah berikutnya di kepala workbook (D-145, C1).
import { describe, expect, it } from 'vitest';
import { nextStep, type StepInput } from './frame.ts';

const draft: StepInput = { status: 'draft', rows: 3, hasRoute: true, canEdit: true, canPropose: true, canRecord: true };

describe('nextStep', () => {
	it('dokumen kosong: rakit dari process template', () => {
		expect(nextStep({ ...draft, rows: 0 })).toEqual({ text: 'nextCompose', open: 'compose' });
	});

	// C4 (D-211): PFD diisi lebih dulu, lalu PFMEA dan CP dirakit per langkah PFD.
	it('PFD kosong: isi langkah PFD', () => {
		expect(nextStep({ ...draft, rows: 0, setRole: 'pfd' })).toEqual({ text: 'nextFillPfd' });
	});

	it('PFMEA atau CP kosong dengan PFD kosong: isi PFD dulu', () => {
		expect(nextStep({ ...draft, rows: 0, setRole: 'follows_pfd', pfdSteps: 0, pfdHref: '/documents/view?id=p' })).toEqual({
			text: 'nextPfdFirst',
			href: '/documents/view?id=p',
			label: 'nextPfdOpen'
		});
	});

	it('PFMEA atau CP kosong dengan langkah PFD: rakit dari PFD', () => {
		expect(nextStep({ ...draft, rows: 0, setRole: 'follows_pfd', pfdSteps: 4 })).toEqual({ text: 'nextComposePfd', open: 'pfd' });
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
		expect(nextStep({ ...draft, status: 'released' })).toEqual({ text: 'nextChange', href: '/changes', label: 'nextChangeOpen' });
	});

	it('akun yang hanya membaca tidak mendapat langkah', () => {
		expect(nextStep({ ...draft, canEdit: false, canPropose: false, canRecord: false })).toBeNull();
		expect(nextStep({ ...draft, status: 'released', canPropose: false })).toBeNull();
	});

	it('akun tanpa hak ubah tetap diberi tahu bila tanda tangan lengkap dan ia boleh mencatat rilis', () => {
		expect(nextStep({ ...draft, canEdit: false, signing: 'completed' })).toEqual({ text: 'nextRecord', open: 'revisions' });
	});
});
