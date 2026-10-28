// formatDateTime menulis waktu ISO 8601 sebagai "YYYY-MM-DD HH:mm" waktu lokal.
// Bentuk ini tidak bisa salah dibaca, berbeda dengan 03/04 yang bisa berarti
// 3 April atau 4 Maret.
export function formatDateTime(iso: string): string {
	const d = new Date(iso);
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
