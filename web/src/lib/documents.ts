// Bantuan layar dokumen (M4.1). Isi header dan baris disimpan per ID field
// format (R-DOC-01); kolom terisi otomatis dibaca dari baris (D-122, D-123),
// dan kolom yang dihitung (RPN, AP) tidak pernah diketik.
import type { DocumentRow, DocumentRowInput, FormatField, FormatVersion } from './api/gen/index.ts';

// notTyped adalah jenis field yang isinya tidak diketik penyusun.
const notTyped: FormatField['kind'][] = ['rpn', 'ap', 'operation_number', 'process', 'process_flow', 'document_number', 'product_char_number', 'process_char_number'];

// typedFields memberi field satu area yang diketik penyusun, menurut urutan.
export function typedFields(fields: FormatField[], area: FormatField['area']): FormatField[] {
	return fields.filter((f) => f.area === area && !f.hidden && !notTyped.includes(f.kind));
}

// cleanContent menyiapkan isian formulir untuk server: isian kosong dibuang,
// dan rating dikirim sebagai angka.
export function cleanContent(fields: FormatField[], values: Record<string, string>): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const f of fields) {
		const value = (values[f.id] ?? '').trim();
		if (value === '') {
			continue;
		}
		out[f.id] = f.kind === 'rating' ? Number(value) : value;
	}
	return out;
}

// cellText menulis isi satu sel untuk layar. Nomor operasi dan proses dibaca
// dari baris; pilihan ditulis dengan labelnya.
export function cellText(field: FormatField, row: Pick<DocumentRow, 'process_name' | 'operation_number' | 'content'>): string {
	if (field.kind === 'operation_number') {
		return row.operation_number ?? '';
	}
	if (field.kind === 'process') {
		return row.process_name ?? '';
	}
	const value = row.content[field.id];
	if (value === undefined || value === null) {
		return '';
	}
	if (field.kind === 'select') {
		return field.options?.find((o) => o.key === value)?.label ?? String(value);
	}
	return String(value);
}

// charNumbers memberi nomor karakteristik setiap baris workbook (D-222),
// sama dengan cetakan (internal/printout/charno.go): No produk untuk baris
// yang Product Characteristics-nya diisi, No proses untuk Process
// Characteristics, berurutan dan mulai lagi dari 1 di setiap proses. Isian
// "--" mendapat "--"; yang kosong dibiarkan kosong.
export function charNumbers(fields: FormatField[], rows: Record<string, unknown>[]): Record<string, [string, string]> {
	const ids = ['product_characteristic', 'process_characteristic'].map((key) => fields.find((f) => f.key === key)?.id ?? '');
	const out: Record<string, [string, string]> = {};
	let group = '';
	let counts = [0, 0];
	for (const r of rows) {
		const key = `${r._section ?? ''}|${r._operation ?? ''}|${r._process ?? ''}`;
		if (key !== group) {
			group = key;
			counts = [0, 0];
		}
		const nums = ids.map((id, i) => {
			const value = id === '' || r[id] === undefined || r[id] === null ? '' : String(r[id]).trim();
			if (value === '' || value === '--') {
				return value;
			}
			counts[i]++;
			return String(counts[i]);
		});
		out[String(r.id)] = [nums[0], nums[1]];
	}
	return out;
}

// GridData adalah satu baris di tabel workbook (M4.2): isi per ID kolom,
// ditambah proses dan nomor operasi milik baris.
// _version adalah versi baris yang dibaca dari server (M4.3).
export type GridData = Record<string, unknown> & { id: string; position: number; _version?: number; _process?: string; _operation?: string };

// gridRow mengubah baris dokumen menjadi data tabel workbook.
export function gridRow(row: Pick<DocumentRow, 'id' | 'position' | 'version' | 'process_id' | 'operation_number' | 'content'>): GridData {
	return { id: row.id, position: row.position, _version: row.version, _process: row.process_id ?? '', _operation: row.operation_number ?? '', ...row.content };
}

// rowInputFrom mengubah data tabel kembali menjadi isian baris untuk server:
// hanya kolom yang diketik ikut sebagai isi; isian kosong dibuang.
export function rowInputFrom(format: Pick<FormatVersion, 'fields'>, data: Record<string, unknown>): DocumentRowInput {
	const fields = typedFields(format.fields, 'row');
	const values = Object.fromEntries(fields.map((f) => [f.id, data[f.id] === undefined || data[f.id] === null ? '' : String(data[f.id])]));
	const input: DocumentRowInput = { content: cleanContent(fields, values) };
	const process = String(data._process ?? '').trim();
	const operation = String(data._operation ?? '').trim();
	if (process !== '') {
		input.process_id = process;
	}
	if (operation !== '') {
		input.operation_number = operation;
	}
	// Versi yang dibaca ikut dikirim; server menolak bila baris sudah berubah (R-DOC-07).
	if (typeof data._version === 'number') {
		input.version = data._version;
	}
	return input;
}

// rowsOfSection memberi baris satu bagian (D-155) menurut urutan, dengan
// nomor urut tampilan mulai 1 per bagian, seperti satu lembar per bagian di
// CP kertas. sectionId "" berarti semua baris (dokumen tanpa bagian).
export function rowsOfSection(rows: DocumentRow[], sectionId: string): DocumentRow[] {
	const list = sectionId === '' ? rows : rows.filter((r) => r.section_id === sectionId);
	return list.map((r, i) => ({ ...r, position: i + 1 }));
}
