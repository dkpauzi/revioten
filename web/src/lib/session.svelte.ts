import type { Session } from './api/gen/index.ts';

// Sesi pengguna yang sedang login, dipakai bersama oleh layout dan halaman.
// Rune $state boleh dipakai di file berakhiran .svelte.ts: komponen yang
// membaca session.current ikut diperbarui saat nilainya berubah.
// Nilai yang diekspor tidak bisa diganti dari luar, jadi yang diubah adalah
// isinya: session.current = ..., bukan session = ...
export const session = $state<{ current: Session | null }>({ current: null });
