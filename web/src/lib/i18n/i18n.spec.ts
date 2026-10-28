import { describe, expect, it } from 'vitest';
import { en } from './en.ts';
import { t } from './index.ts';

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
