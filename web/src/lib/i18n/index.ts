import { en, type MessageKey, type Messages } from './en.ts';

// Katalog bahasa yang aktif. Saat ini hanya bahasa Inggris (Q-14).
//
// Cara menambah bahasa Indonesia nanti:
//   1. Buat id.ts berisi `export const id: Messages = { ... }` dengan kunci yang sama.
//      TypeScript menolak bila ada kunci yang terlewat.
//   2. Ganti katalog aktif di bawah ini menjadi `id`, atau buat pilihannya bisa
//      diatur pengguna.
const active: Messages = en;

// t mengembalikan teks layar untuk satu kunci.
export function t(key: MessageKey): string {
	return active[key];
}
