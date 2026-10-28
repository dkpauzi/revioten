import { en, type MessageKey, type Messages } from './en.ts';

export type { MessageKey };

// Katalog bahasa yang aktif. Saat ini hanya bahasa Inggris (Q-14).
//
// Cara menambah bahasa Indonesia nanti:
//   1. Buat id.ts berisi `export const id: Messages = { ... }` dengan kunci yang sama.
//      TypeScript menolak bila ada kunci yang terlewat.
//   2. Ganti katalog aktif di bawah ini menjadi `id`, atau buat pilihannya bisa
//      diatur pengguna.
const active: Messages = en;

// t mengembalikan teks layar untuk satu kunci. Tempat seperti {code} di teks
// diisi dari vars; tempat yang tidak diberi nilai dibiarkan apa adanya.
export function t(key: MessageKey, vars: Record<string, string> = {}): string {
	return active[key].replace(/\{(\w+)\}/g, (placeholder, name: string) => vars[name] ?? placeholder);
}
