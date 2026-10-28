// formatDateTime menulis waktu ISO 8601 sebagai "YYYY-MM-DD HH:mm" waktu lokal.
// Bentuk ini tidak bisa salah dibaca, berbeda dengan 03/04 yang bisa berarti
// 3 April atau 4 Maret.
export function formatDateTime(iso: string): string {
	return format(new Date(iso), false);
}

// formatTimestamp seperti formatDateTime, ditambah detik: "YYYY-MM-DD HH:mm:ss".
// Dipakai audit log, karena beberapa peristiwa bisa terjadi dalam menit yang sama.
export function formatTimestamp(iso: string): string {
	return format(new Date(iso), true);
}

function format(d: Date, seconds: boolean): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	const time = `${pad(d.getHours())}:${pad(d.getMinutes())}` + (seconds ? `:${pad(d.getSeconds())}` : '');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${time}`;
}
