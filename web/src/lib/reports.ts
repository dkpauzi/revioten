// Bantuan tampilan untuk review berkala dan laporan kinerja (D-202). Fungsi di
// sini murni (tanpa API), supaya bisa diuji Vitest.

const months: Record<'en' | 'id', string[]> = {
	en: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
	id: ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']
};

// formatDay menulis tanggal YYYY-MM-DD sebagai "2 Aug 2026". Nama bulan tidak
// bisa salah dibaca, berbeda dengan 03/04.
export function formatDay(iso: string, lang: 'en' | 'id' = 'en'): string {
	const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
	if (!y || !m || !d) return iso;
	return `${d} ${months[lang][m - 1]} ${y}`;
}

// compactNumbers meringkas nomor satu set yang hanya beda awalan jenisnya:
// SH-KT-005, SF-KT-005, SC-KT-005 menjadi "SH/SF/SC-KT-005".
export function compactNumbers(numbers: string[]): string {
	if (numbers.length < 2) return numbers.join('');
	const split = numbers.map((n) => {
		const i = n.indexOf('-');
		return i < 0 ? null : { prefix: n.slice(0, i), rest: n.slice(i) };
	});
	if (split.some((p) => p === null) || new Set(split.map((p) => p!.rest)).size !== 1) {
		return numbers.join(', ');
	}
	return split.map((p) => p!.prefix).join('/') + split[0]!.rest;
}

// overdueTypes memberi jenis dokumen set yang lewat jatuh tempo, menurut urutan
// set: ['pfd'], ['pfd', 'pfmea'], atau semuanya.
export function overdueTypes(documents: { document_type: string; overdue: boolean }[]): string[] {
	return documents.filter((d) => d.overdue).map((d) => d.document_type);
}

// hoursParts memberi lama dalam satuan yang mudah dibaca: jam bila kurang
// dari dua hari, selain itu hari. Satu angka di belakang koma.
export function hoursParts(hours: number): { value: string; unit: 'hours' | 'days' } {
	if (hours < 48) return { value: (Math.round(hours * 10) / 10).toFixed(1), unit: 'hours' };
	return { value: (Math.round((hours / 24) * 10) / 10).toFixed(1), unit: 'days' };
}

// delta memberi selisih dengan periode sebelumnya dan arahnya.
export function delta(current: number, previous: number): { diff: number; direction: 'up' | 'down' | 'same' } {
	const diff = Math.round((current - previous) * 10) / 10;
	return { diff: Math.abs(diff), direction: diff > 0 ? 'up' : diff < 0 ? 'down' : 'same' };
}

// shares membagi lebar batang tahap menurut median setiap tahap, dalam
// persen yang jumlahnya 100. Tahap tanpa data mendapat 0.
export function shares(values: number[]): number[] {
	const total = values.reduce((a, b) => a + b, 0);
	if (total <= 0) return values.map(() => 0);
	return values.map((v) => (v / total) * 100);
}

// isoDate menulis tanggal lokal sebagai YYYY-MM-DD.
export function isoDate(d: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
