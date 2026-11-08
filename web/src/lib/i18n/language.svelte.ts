// Bahasa layar yang aktif (D-128): Inggris (bawaan) atau Indonesia.
//
// Nilainya disimpan dengan rune $state, jadi setiap teks di layar yang dibuat
// lewat t() ikut berganti saat bahasa diganti, tanpa memuat ulang halaman.
// Pilihannya disimpan di browser (localStorage), bukan di akun, supaya juga
// berlaku di halaman login sebelum orang masuk.

export type Language = 'en' | 'id';

export const languages: Language[] = ['en', 'id'];

const storageKey = 'revioten.language';

// parseLanguage mengubah nilai tersimpan menjadi bahasa yang dikenal. Nilai
// lain (kosong, rusak, atau dari versi lain) dianggap bahasa Inggris.
export function parseLanguage(value: string | null): Language {
	return value === 'id' ? 'id' : 'en';
}

// readStored membaca pilihan tersimpan. localStorage bisa tidak ada (test di
// Node) atau ditolak browser (mode privat), jadi kegagalan berarti bawaan.
function readStored(): Language {
	try {
		return parseLanguage(localStorage.getItem(storageKey));
	} catch {
		return 'en';
	}
}

export const language = $state<{ current: Language }>({ current: readStored() });

// setLanguage mengganti bahasa layar, menyimpannya, dan memperbarui atribut
// lang di <html> supaya pembaca layar dan browser tahu bahasa halaman.
export function setLanguage(next: Language): void {
	language.current = next;
	try {
		localStorage.setItem(storageKey, next);
	} catch {
		// Tidak bisa disimpan: bahasa tetap berganti untuk kunjungan ini saja.
	}
	if (typeof document !== 'undefined') {
		document.documentElement.lang = next;
	}
}
