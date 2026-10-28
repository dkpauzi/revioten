import { describe, expect, it } from 'vitest';
import { formatDateTime } from './format.ts';

describe('formatDateTime', () => {
	it('menulis tanggal dan jam lokal dalam bentuk YYYY-MM-DD HH:mm', () => {
		const iso = '2026-10-28T03:04:00Z';
		const d = new Date(iso);
		const pad = (n: number) => String(n).padStart(2, '0');
		const want = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
		expect(formatDateTime(iso)).toBe(want);
	});
});
