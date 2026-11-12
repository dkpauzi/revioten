// Tempel dari Excel (M4.4, R-MIG-02): teks clipboard dipecah menjadi sel,
// kolomnya dipetakan ke kolom format, lalu diubah menjadi isian baris.
// Sel yang tidak cocok dengan jenis kolomnya ditandai, tetapi barisnya tetap
// masuk dengan sel itu dikosongkan (ASUMSI E-91).
import type { DocumentRowInput, FormatField, FormatVersion, Process } from './api/gen/index.ts';
import { typedFields } from './documents.ts';

// Batas yang sama dengan server (internal/template dan internal/document).
const maxText = 2000;
const maxOperation = 20;

// parseClipboard memecah teks yang disalin dari Excel: kolom dipisah tab,
// baris dipisah baris baru. Sel yang berisi tab, baris baru, atau tanda kutip
// ditulis Excel di dalam tanda kutip, dengan tanda kutip digandakan. Baris
// yang semua selnya kosong dibuang.
export function parseClipboard(text: string): string[][] {
	const rows: string[][] = [];
	let row: string[] = [];
	let cell = '';
	let quoted = false;
	const source = text.replace(/\r\n?/g, '\n');
	for (let i = 0; i < source.length; i++) {
		const ch = source[i];
		if (quoted) {
			if (ch === '"' && source[i + 1] === '"') {
				cell += '"';
				i++;
			} else if (ch === '"') {
				quoted = false;
			} else {
				cell += ch;
			}
		} else if (ch === '"' && cell === '') {
			quoted = true;
		} else if (ch === '\t') {
			row.push(cell);
			cell = '';
		} else if (ch === '\n') {
			row.push(cell);
			rows.push(row);
			row = [];
			cell = '';
		} else {
			cell += ch;
		}
	}
	row.push(cell);
	rows.push(row);
	return rows.filter((r) => r.some((c) => c.trim() !== ''));
}

// PasteTarget adalah satu kolom dokumen yang bisa diisi dari tempelan.
// id-nya ID field format, atau _operation dan _process untuk atribut baris.
export type PasteTarget = { id: string; label: string; kind: FormatField['kind'] };

// pasteTargets memberi kolom yang bisa diisi, menurut urutan di format.
export function pasteTargets(format: Pick<FormatVersion, 'fields'>): PasteTarget[] {
	const typed = new Set(typedFields(format.fields, 'row').map((f) => f.id));
	const out: PasteTarget[] = [];
	for (const f of format.fields.filter((x) => x.area === 'row' && !x.hidden).sort((a, b) => a.position - b.position)) {
		if (f.kind === 'operation_number') {
			out.push({ id: '_operation', label: f.label, kind: f.kind });
		} else if (f.kind === 'process') {
			out.push({ id: '_process', label: f.label, kind: f.kind });
		} else if (typed.has(f.id)) {
			out.push({ id: f.id, label: f.label, kind: f.kind });
		}
	}
	return out;
}

const same = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase();

// guessMapping menebak kolom tujuan setiap kolom tempelan. Dengan baris
// judul, kolom dicocokkan menurut label; tanpa itu, menurut urutan. Kolom
// yang tidak ketemu diberi '' (dilewati); penyusun bisa mengubahnya.
export function guessMapping(header: string[] | null, targets: PasteTarget[], columnCount: number): string[] {
	const mapping: string[] = [];
	for (let i = 0; i < columnCount; i++) {
		if (header) {
			const found = targets.find((x) => same(x.label, header[i] ?? '') && !mapping.includes(x.id));
			mapping.push(found?.id ?? '');
		} else {
			mapping.push(targets[i]?.id ?? '');
		}
	}
	return mapping;
}

// PasteFlag menandai satu sel janggal di pratinjau.
export type PasteReason = 'rating' | 'date' | 'option' | 'symbol' | 'process' | 'too_long';
export type PasteFlag = { row: number; column: number; reason: PasteReason };

const months = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec'];

// isoDate menerima YYYY-MM-DD dan bentuk Excel seperti 13-Nov-26. Bentuk
// angka saja seperti 03/04/2026 tidak diterima, karena hari dan bulannya
// bisa tertukar.
function isoDate(text: string): string | null {
	if (/^\d{4}-\d{2}-\d{2}$/.test(text) && !Number.isNaN(Date.parse(text))) {
		return text;
	}
	const m = /^(\d{1,2})[- ]([A-Za-z]{3})[- ](\d{2}|\d{4})$/.exec(text);
	if (!m) {
		return null;
	}
	const month = months.indexOf(m[2].toLowerCase());
	const day = Number(m[1]);
	if (month < 0 || day < 1 || day > 31) {
		return null;
	}
	const year = m[3].length === 2 ? 2000 + Number(m[3]) : Number(m[3]);
	return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

// buildRows mengubah sel tempelan menjadi isian baris. mapping[i] adalah
// kolom tujuan kolom tempelan ke-i. Setiap baris tempelan menjadi satu baris
// dokumen; sel janggal dicatat di flags dan tidak ikut dikirim.
export function buildRows(
	grid: string[][],
	mapping: string[],
	format: Pick<FormatVersion, 'fields' | 'symbols'>,
	processes: Process[]
): { rows: DocumentRowInput[]; flags: PasteFlag[] } {
	const fields = new Map(format.fields.map((f) => [f.id, f]));
	const rows: DocumentRowInput[] = [];
	const flags: PasteFlag[] = [];
	grid.forEach((cells, r) => {
		const input: DocumentRowInput = { content: {} };
		cells.forEach((raw, c) => {
			const target = mapping[c] ?? '';
			const text = raw.trim();
			if (target === '' || text === '') {
				return;
			}
			const flag = (reason: PasteReason) => flags.push({ row: r, column: c, reason });
			if (target === '_operation') {
				if (text.length > maxOperation) {
					flag('too_long');
				} else {
					input.operation_number = text;
				}
				return;
			}
			if (target === '_process') {
				const found = processes.find((p) => same(p.name, text) || p.aliases.some((a) => same(a, text)));
				if (found) {
					input.process_id = found.id;
				} else {
					flag('process');
				}
				return;
			}
			const field = fields.get(target);
			if (!field) {
				return;
			}
			if (field.kind === 'rating') {
				const n = Number(text);
				if (Number.isInteger(n) && n >= 1 && n <= 10) {
					input.content[field.id] = n;
				} else {
					flag('rating');
				}
			} else if (field.kind === 'date') {
				const date = isoDate(text);
				if (date) {
					input.content[field.id] = date;
				} else {
					flag('date');
				}
			} else if (field.kind === 'select') {
				const option = (field.options ?? []).find((o) => same(o.label, text) || same(o.key ?? '', text));
				if (option?.key) {
					input.content[field.id] = option.key;
				} else {
					flag('option');
				}
			} else if (field.kind === 'symbol') {
				const symbol = format.symbols.find((s) => !s.automatic && (same(s.name, text) || same(s.key ?? '', text)));
				if (symbol?.key) {
					input.content[field.id] = symbol.key;
				} else {
					flag('symbol');
				}
			} else if (text.length > maxText) {
				flag('too_long');
			} else {
				input.content[field.id] = text;
			}
		});
		rows.push(input);
	});
	return { rows, flags };
}
