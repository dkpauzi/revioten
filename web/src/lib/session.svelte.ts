import type { Session } from './api/gen/index.ts';

// Sesi pengguna yang sedang login, dipakai bersama oleh layout dan halaman.
// Rune $state boleh dipakai di file berakhiran .svelte.ts: komponen yang
// membaca session.current ikut diperbarui saat nilainya berubah.
// Nilai yang diekspor tidak bisa diganti dari luar, jadi yang diubah adalah
// isinya: session.current = ..., bukan session = ...
export const session = $state<{ current: Session | null }>({ current: null });

// can melaporkan apakah pengguna yang login punya hak dengan kode tertentu,
// misalnya can('E-11'). Dipakai untuk menampilkan atau menyembunyikan menu dan
// tombol. Server tetap memeriksa setiap operasi (M1.2), jadi menyembunyikan
// tombol bukan pengaman, hanya supaya layar tidak menawarkan aksi yang pasti ditolak.
export function can(permission: string): boolean {
	return session.current?.permissions.includes(permission) ?? false;
}
