import { afterEach, describe, expect, it } from 'vitest';
import { en } from './en.ts';
import { id } from './id.ts';
import { t } from './index.ts';
import { language, parseLanguage, setLanguage } from './language.svelte.ts';

describe('katalog teks layar', () => {
	it('setiap kunci punya teks yang tidak kosong', () => {
		for (const [key, text] of Object.entries(en)) {
			expect(text.trim(), key).not.toBe('');
		}
	});

	it('t mengembalikan teks untuk kunci yang diminta', () => {
		expect(t('navHome')).toBe(en.navHome);
	});

	it('t mengisi tempat seperti {code} dengan nilai yang diberikan', () => {
		expect(t('customerSaved', { code: 'AB' })).toBe(en.customerSaved.replace('{code}', 'AB'));
		expect(t('customerSaved', { code: 'AB' })).not.toContain('{code}');
	});

	it('t membiarkan tempat yang tidak diberi nilai', () => {
		expect(t('customerSaved')).toContain('{code}');
	});
});

// Katalog bahasa Indonesia (D-128). TypeScript sudah menolak kunci yang
// terlewat; test ini memeriksa isinya.
describe('katalog bahasa Indonesia', () => {
	it('setiap kunci punya teks yang tidak kosong', () => {
		for (const [key, text] of Object.entries(id)) {
			expect(text.trim(), key).not.toBe('');
		}
	});

	it('punya kunci yang sama persis dengan katalog Inggris', () => {
		expect(Object.keys(id).sort()).toEqual(Object.keys(en).sort());
	});

	// Tempat seperti {code} diisi kode layar, jadi nama tempatnya harus sama di dua bahasa.
	it('memakai tempat isian yang sama dengan katalog Inggris', () => {
		const placeholders = (text: string) => (text.match(/\{\w+\}/g) ?? []).sort();
		for (const key of Object.keys(en) as (keyof typeof en)[]) {
			expect(placeholders(id[key]), key).toEqual(placeholders(en[key]));
		}
	});
});

describe('pilihan bahasa', () => {
	afterEach(() => {
		setLanguage('en');
	});

	it('bahasa Inggris adalah bawaan', () => {
		expect(language.current).toBe('en');
	});

	it('t mengikuti bahasa yang dipilih', () => {
		setLanguage('id');
		expect(t('navHome')).toBe(id.navHome);
		expect(t('customerSaved', { code: 'AB' })).toBe(id.customerSaved.replace('{code}', 'AB'));
		setLanguage('en');
		expect(t('navHome')).toBe(en.navHome);
	});

	// Nilai tersimpan bisa rusak atau berasal dari versi lain; selain en dan id dianggap bawaan.
	it('parseLanguage hanya menerima en dan id', () => {
		expect(parseLanguage('id')).toBe('id');
		expect(parseLanguage('en')).toBe('en');
		expect(parseLanguage('fr')).toBe('en');
		expect(parseLanguage(null)).toBe('en');
	});
});
