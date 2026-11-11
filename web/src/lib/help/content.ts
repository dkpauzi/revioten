// Isi buku panduan pengguna (D-141): halaman Help di dalam aplikasi.
//
// Isi ditulis per bahasa di en.ts dan id.ts dengan bentuk yang sama, dan
// content.spec.ts memastikan keduanya punya bab, langkah, dan gambar yang
// sama. Di dalam teks ada dua penanda:
//   {kunciTeksLayar}  diganti nama tombol atau label dari katalog layar
//                     (web/src/lib/i18n), jadi panduan selalu memakai nama
//                     yang sama dengan layar, di kedua bahasa;
//   [[teks]]          dicetak tebal apa adanya, untuk nama yang tidak ada di
//                     katalog, misalnya contoh isian.
import { en } from '../i18n/en.ts';
import type { MessageKey } from '../i18n/index.ts';

export type HelpChapter = {
	id: string; // dipakai sebagai jangkar tautan, sama di kedua bahasa
	title: string;
	who: string; // siapa yang mengerjakan bab ini
	intro: string;
	steps: string[];
	image?: string; // nama file gambar tanpa akhiran, di static/help/<bahasa>/
	imageAlt?: string;
	tips?: string[];
};

export type HelpContent = {
	title: string;
	intro: string;
	contentsTitle: string;
	whoLabel: string;
	tipsTitle: string;
	print: string;
	chapters: HelpChapter[];
};

// Part adalah satu potongan teks panduan: teks biasa atau teks tebal.
export type Part = { text: string; strong: boolean };

const marker = /\{(\w+)\}|\[\[(.+?)\]\]/g;

// keysIn mengumpulkan kunci teks layar yang dipakai satu teks panduan.
export function keysIn(text: string): string[] {
	return [...text.matchAll(marker)].map((m) => m[1]).filter((key) => key !== undefined);
}

// parts memecah teks panduan menjadi potongan biasa dan tebal. label
// memberi teks layar untuk satu kunci, biasanya t() dari katalog.
export function parts(text: string, label: (key: MessageKey) => string): Part[] {
	const out: Part[] = [];
	let last = 0;
	for (const m of text.matchAll(marker)) {
		if (m.index > last) {
			out.push({ text: text.slice(last, m.index), strong: false });
		}
		out.push({ text: m[1] !== undefined ? label(m[1] as MessageKey) : m[2], strong: true });
		last = m.index + m[0].length;
	}
	if (last < text.length) {
		out.push({ text: text.slice(last), strong: false });
	}
	return out;
}

// isMessageKey melaporkan apakah kunci itu ada di katalog layar.
export function isMessageKey(key: string): key is MessageKey {
	return key in en;
}
