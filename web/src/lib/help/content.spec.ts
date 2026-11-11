import { describe, expect, it } from 'vitest';
import { isMessageKey, keysIn, parts, type HelpContent } from './content.ts';
import { helpEn } from './en.ts';
import { helpId } from './id.ts';

// texts mengumpulkan semua teks panduan yang bisa memuat penanda.
function texts(content: HelpContent): string[] {
	return content.chapters.flatMap((c) => [c.intro, ...c.steps, ...(c.tips ?? [])]).concat(content.intro);
}

describe('parts', () => {
	it('mengganti kunci teks layar dan mencetak tebal teks bertanda', () => {
		const label = (key: string) => `<${key}>`;
		expect(parts('Tekan {customerSave}, lalu isi [[GR1]].', label)).toEqual([
			{ text: 'Tekan ', strong: false },
			{ text: '<customerSave>', strong: true },
			{ text: ', lalu isi ', strong: false },
			{ text: 'GR1', strong: true },
			{ text: '.', strong: false }
		]);
	});

	it('membiarkan teks tanpa penanda', () => {
		expect(parts('Teks biasa.', () => '')).toEqual([{ text: 'Teks biasa.', strong: false }]);
	});
});

describe('isi panduan', () => {
	it('punya bab, jumlah langkah, dan gambar yang sama di kedua bahasa', () => {
		const shape = (content: HelpContent) =>
			content.chapters.map((c) => ({ id: c.id, steps: c.steps.length, image: c.image, tips: c.tips?.length ?? 0 }));
		expect(shape(helpId)).toEqual(shape(helpEn));
		expect(helpEn.chapters.length).toBeGreaterThan(8);
	});

	it('punya id bab yang unik', () => {
		const ids = helpEn.chapters.map((c) => c.id);
		expect(new Set(ids).size).toBe(ids.length);
	});

	it('hanya memakai kunci teks layar yang ada di katalog', () => {
		for (const content of [helpEn, helpId]) {
			const unknown = texts(content).flatMap(keysIn).filter((key) => !isMessageKey(key));
			expect(unknown).toEqual([]);
		}
	});

	it('memakai kunci teks layar yang sama di langkah yang sama', () => {
		const keys = (content: HelpContent) => content.chapters.map((c) => c.steps.map((s) => keysIn(s).sort()));
		expect(keys(helpId)).toEqual(keys(helpEn));
	});

	it('memberi teks pengganti untuk setiap gambar', () => {
		for (const content of [helpEn, helpId]) {
			for (const c of content.chapters) {
				expect(Boolean(c.image), c.id).toBe(Boolean(c.imageAlt));
			}
		}
	});
});
