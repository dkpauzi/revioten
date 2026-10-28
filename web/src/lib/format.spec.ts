import { describe, expect, it } from 'vitest';
import { formatDateTime, formatTimestamp } from './format.ts';

// local menulis waktu lokal dari tanggal d, sama seperti yang diharapkan di layar.
function local(d: Date, seconds: boolean): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	const time = `${pad(d.getHours())}:${pad(d.getMinutes())}` + (seconds ? `:${pad(d.getSeconds())}` : '');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${time}`;
}

describe('formatDateTime', () => {
	it('menulis tanggal dan jam lokal dalam bentuk YYYY-MM-DD HH:mm', () => {
		const iso = '2026-10-28T03:04:00Z';
		expect(formatDateTime(iso)).toBe(local(new Date(iso), false));
	});
});

describe('formatTimestamp', () => {
	// Audit log butuh detik, karena beberapa peristiwa bisa terjadi dalam menit yang sama.
	it('menulis tanggal dan jam lokal dengan detik: YYYY-MM-DD HH:mm:ss', () => {
		const iso = '2026-10-28T03:04:05.678Z';
		expect(formatTimestamp(iso)).toBe(local(new Date(iso), true));
	});
});
