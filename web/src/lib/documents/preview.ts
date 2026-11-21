// Pratinjau lembar dokumen (C2, D-122, G11): dokumen tampil per halaman kertas
// seperti cetakannya, dan kolom Process Flow digambar dari langkah PFD dengan
// konektor di setiap batas halaman. Fungsi di sini murni (tanpa DOM), supaya
// bisa dites dengan Vitest; tinggi baris diukur oleh komponen PreviewSheet.

// Ukuran kertas dalam milimeter, sebelum diputar menurut arah halaman.
const papers = { A3: { w: 297, h: 420 }, A4: { w: 210, h: 297 } } as const;

export type Paper = { widthMm: number; heightMm: number };

// paperSize memberi lebar dan tinggi halaman menurut kertas dan arahnya.
export function paperSize(layout: { paper: 'A3' | 'A4'; orientation: 'landscape' | 'portrait' }): Paper {
	const p = papers[layout.paper] ?? papers.A3;
	return layout.orientation === 'landscape' ? { widthMm: p.h, heightMm: p.w } : { widthMm: p.w, heightMm: p.h };
}

// pxPerMm: 96 titik per inci, seperti browser dan Chrome saat mencetak PDF.
export const pxPerMm = 96 / 25.4;

// paginate membagi baris ke halaman menurut tingginya (piksel). Halaman
// pertama boleh punya ruang berbeda (misalnya kepala form lebih tinggi).
// Setiap halaman berisi paling sedikit satu baris, juga bila baris itu lebih
// tinggi dari halaman. Hasilnya indeks baris per halaman.
export function paginate(heights: number[], firstAvailable: number, restAvailable: number): number[][] {
	const pages: number[][] = [[]];
	let used = 0;
	heights.forEach((h, i) => {
		const page = pages[pages.length - 1];
		const available = pages.length === 1 ? firstAvailable : restAvailable;
		if (page.length > 0 && used + h > available) {
			pages.push([i]);
			used = h;
			return;
		}
		page.push(i);
		used += h;
	});
	return pages;
}

// Group adalah baris berurutan satu langkah proses: nomor operasi dan proses
// sama. op kosong berarti baris tanpa nomor operasi (tidak diberi simbol).
export type Group = { start: number; end: number; op: string };

// groupsOf mengelompokkan baris berurutan dengan nomor operasi dan proses yang sama.
export function groupsOf(rows: { operation_number?: string; process_id?: string }[]): Group[] {
	const groups: Group[] = [];
	rows.forEach((r, i) => {
		const op = r.operation_number ?? '';
		const last = groups[groups.length - 1];
		const lastRow = i > 0 ? rows[i - 1] : undefined;
		if (last && op !== '' && last.op === op && lastRow?.process_id === r.process_id) {
			last.end = i;
			return;
		}
		groups.push({ start: i, end: i, op });
	});
	return groups;
}

// Step adalah satu langkah PFD: simbol alur dan, untuk inspeksi, arah NG.
export type Step = { symbol: string; ng?: string };

// stepsOf membaca langkah PFD menurut nomor langkahnya (R-DOC-09). symbolField
// dan ngField adalah ID field Flow Symbol dan NG Disposition di format PFD.
export function stepsOf(
	rows: { operation_number?: string; content: Record<string, unknown> }[],
	symbolField: string,
	ngField: string
): Map<string, Step> {
	const steps = new Map<string, Step>();
	for (const r of rows) {
		if (!r.operation_number) continue;
		const symbol = typeof r.content[symbolField] === 'string' ? (r.content[symbolField] as string) : 'process';
		const ng = typeof r.content[ngField] === 'string' && r.content[ngField] !== '' ? (r.content[ngField] as string) : undefined;
		steps.set(r.operation_number, { symbol, ng });
	}
	return steps;
}

// connectorLetter memberi huruf konektor ke-n: A sampai Z, lalu AA, AB, ...
export function connectorLetter(n: number): string {
	let s = '';
	let i = n;
	do {
		s = String.fromCharCode(65 + (i % 26)) + s;
		i = Math.floor(i / 26) - 1;
	} while (i >= 0);
	return s;
}

// FlowNode adalah satu simbol di kolom Process Flow: baris ke-row di halaman itu.
export type FlowNode = { row: number; op: string; symbol: string; ng?: string };

// FlowPage adalah isi kolom Process Flow satu halaman. inLetter: konektor di
// atas halaman (alur datang dari halaman sebelumnya); outLetter: konektor di
// bawah halaman (alur berlanjut ke halaman berikutnya). lastRow: baris
// terakhir di halaman itu, tempat garis berakhir.
export type FlowPage = { nodes: FlowNode[]; lastRow: number; inLetter?: string; outLetter?: string };

// flowPages menempatkan simbol dan konektor di setiap halaman. Simbol ada di
// baris pertama setiap kelompok bernomor operasi; simbolnya dari PFD, atau
// "process" bila langkah itu tidak ada di PFD. Konektor dipasang di setiap
// batas halaman yang dilewati alur, dengan huruf berurutan. Karena semuanya
// dihitung dari baris, menambah atau menghapus baris menggeser simbol dan
// konektor sendiri (G11).
export function flowPages(groups: Group[], pages: number[][], steps: Map<string, Step>): FlowPage[] {
	const flowing = groups.filter((g) => g.op !== '');
	const firstRow = (p: number) => pages[p][0] ?? Number.POSITIVE_INFINITY;
	const lastRowOf = (p: number) => pages[p][pages[p].length - 1] ?? Number.NEGATIVE_INFINITY;
	// Batas sesudah halaman p dilewati alur bila alur sudah mulai di halaman p
	// atau sebelumnya, dan masih berlanjut di halaman sesudahnya.
	const crosses = (p: number) =>
		flowing.some((g) => g.start <= lastRowOf(p)) && flowing.some((g) => g.end >= firstRow(p + 1));
	let letters = 0;
	const out: FlowPage[] = [];
	pages.forEach((rows, p) => {
		const position = new Map(rows.map((r, i) => [r, i]));
		const nodes: FlowNode[] = [];
		for (const g of flowing) {
			const at = position.get(g.start);
			if (at === undefined) continue;
			const step = steps.get(g.op);
			nodes.push({ row: at, op: g.op, symbol: step?.symbol ?? 'process', ng: step?.ng });
		}
		const inLetter = p > 0 && crosses(p - 1) ? out[p - 1].outLetter : undefined;
		const outLetter = p < pages.length - 1 && crosses(p) ? connectorLetter(letters++) : undefined;
		out.push({ nodes, lastRow: rows.length - 1, inLetter, outLetter });
	});
	return out;
}

// Bahan teks sel pratinjau.
type Field = {
	id: string;
	kind: string;
	rating_set?: number;
	rating_scale?: string;
	options?: { key?: string; label: string }[];
};
type Row = { process_name?: string; operation_number?: string; content: Record<string, unknown> };

// previewText memberi teks satu sel pratinjau, sama dengan cetakan: nomor
// operasi dan proses dari baris (D-122, D-123), pilihan dan simbol menurut
// namanya, dan RPN sebagai hasil kali S, O, dan D satu set nilai. AP dan
// Process Flow tidak ditulis sebagai teks.
export function previewText(field: Field, row: Row, fields: Field[], symbols: { key?: string; name: string }[]): string {
	switch (field.kind) {
		case 'operation_number':
			return row.operation_number ?? '';
		case 'process':
			return row.process_name ?? '';
		case 'process_flow':
		case 'ap':
			return '';
		case 'rpn': {
			const set = fields.filter((f) => f.kind === 'rating' && (f.rating_set ?? 1) === (field.rating_set ?? 1));
			const values = set.map((f) => Number(row.content[f.id]));
			if (set.length !== 3 || values.some((v) => !Number.isInteger(v) || v < 1)) return '';
			return String(values[0] * values[1] * values[2]);
		}
	}
	const value = row.content[field.id];
	if (value === undefined || value === null) return '';
	if (field.kind === 'select') return field.options?.find((o) => o.key === value)?.label ?? String(value);
	if (field.kind === 'symbol') return symbols.find((s) => s.key === value)?.name ?? String(value);
	return String(value);
}
