import { describe, expect, it } from 'vitest';
import { safeNext } from './navigation.ts';

describe('safeNext', () => {
	it('memakai alamat di dalam aplikasi apa adanya', () => {
		expect(safeNext('/customers?x=1')).toBe('/customers?x=1');
	});

	it('kembali ke beranda bila parameter kosong', () => {
		expect(safeNext(null)).toBe('/');
		expect(safeNext('')).toBe('/');
	});

	// Tanpa pemeriksaan ini, tautan /login?next=https://situs-palsu bisa membawa
	// pengguna ke situs lain setelah login (open redirect).
	it('menolak alamat yang keluar dari aplikasi', () => {
		for (const next of ['https://situs-lain.example', '//situs-lain.example', '/\\situs-lain.example', 'customers']) {
			expect(safeNext(next), next).toBe('/');
		}
	});

	it('tidak kembali ke halaman login', () => {
		expect(safeNext('/login')).toBe('/');
		expect(safeNext('/login?next=%2F')).toBe('/');
	});
});
