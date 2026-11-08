import { en, type MessageKey, type Messages } from './en.ts';
import { id } from './id.ts';
import { language, type Language } from './language.svelte.ts';

export type { MessageKey };

// Dua katalog dengan kunci yang sama (D-128). TypeScript menolak katalog yang
// kekurangan kunci, karena keduanya bertipe Messages.
const catalogs: Record<Language, Messages> = { en, id };

// t mengembalikan teks layar untuk satu kunci dalam bahasa yang aktif. Tempat
// seperti {code} di teks diisi dari vars; tempat yang tidak diberi nilai
// dibiarkan apa adanya.
//
// Karena t membaca language.current (rune $state), teks di template Svelte
// ikut berganti saat bahasa diganti. Teks yang sudah disimpan di variabel
// (misalnya pesan "tersimpan") baru berganti saat dibuat ulang.
export function t(key: MessageKey, vars: Record<string, string> = {}): string {
	return catalogs[language.current][key].replace(/\{(\w+)\}/g, (placeholder, name: string) => vars[name] ?? placeholder);
}
