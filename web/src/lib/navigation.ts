// safeNext memeriksa parameter next di /login?next=... sebelum dipakai.
// Hanya alamat di dalam aplikasi ini yang diterima: diawali satu garis miring,
// bukan "//" (alamat situs lain), tanpa garis miring terbalik (beberapa browser
// membacanya sebagai "/"), dan bukan halaman login itu sendiri. Selain itu
// pengguna dibawa ke beranda. Ini mencegah open redirect: tautan palsu yang
// membawa pengguna ke situs lain setelah login.
export function safeNext(next: string | null): string {
	if (!next || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) {
		return '/';
	}
	if (next === '/login' || next.startsWith('/login?') || next.startsWith('/login/')) {
		return '/';
	}
	return next;
}
