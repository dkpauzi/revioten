// Bantuan layar dokumen (M4.1). Isi header dan baris disimpan per ID field
// format (R-DOC-01); kolom terisi otomatis dibaca dari baris (D-122, D-123),
// dan kolom yang dihitung (RPN, AP) tidak pernah diketik.
import type { DocumentRow, FormatField } from './api/gen/index.ts';

// notTyped adalah jenis field yang isinya tidak diketik penyusun.
const notTyped: FormatField['kind'][] = ['rpn', 'ap', 'operation_number', 'process', 'process_flow', 'document_number'];

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