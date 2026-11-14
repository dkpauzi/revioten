// Logika panel urutan dan nomor operasi (M4.6, R-CMP-01). Aturannya sama
// dengan server (internal/document/order.go): baris berurutan dengan proses
// yang sama adalah satu langkah, dan nomor otomatis berjarak 10.
import type { DocumentRow } from './api/gen/index.ts';

const operationStep = 10;

// Step adalah satu langkah proses di panel urutan.
export type Step = {
	rowIds: string[];
	processName: string; // '' untuk baris tanpa proses
	current: string; // nomor operasi sekarang; beberapa nomor dipisah koma
	next: string; // nomor sesudah penomoran otomatis
};

const join = (numbers: string[]) => [...new Set(numbers.filter((n) => n !== ''))].join(', ');

// stepsOf mengelompokkan baris menjadi langkah, menurut urutannya.
export function stepsOf(rows: DocumentRow[]): Step[] {
	const groups: DocumentRow[][] = [];
	rows.forEach((r, i) => {
		const previous = rows[i - 1];
		if (i === 0 || !r.process_id || r.process_id !== previous.process_id) {
			groups.push([r]);
		} else {
			groups[groups.length - 1].push(r);
		}
	});
	return groups.map((group, i) => {
		const auto = String((i + 1) * operationStep);
		return {
			rowIds: group.map((r) => r.id),
			processName: group[0].process_name ?? '',
			current: join(group.map((r) => r.operation_number ?? '')),
			// Nomor yang diketik penyusun tidak ditimpa penomoran otomatis.
			next: join(group.map((r) => (r.operation_manual ? (r.operation_number ?? '') : auto)))
		};
	});
}

// moveStep memindahkan langkah ke-from ke tempat ke-to (mulai 0) beserta
// semua barisnya, dan mengembalikan urutan baris yang baru. Di luar batas,
// daftar yang sama dikembalikan.
export function moveStep(rows: DocumentRow[], from: number, to: number): DocumentRow[] {
	const steps = stepsOf(rows);
	if (from === to || from < 0 || to < 0 || from >= steps.length || to >= steps.length) {
		return rows;
	}
	const byId = new Map(rows.map((r) => [r.id, r]));
	const [moving] = steps.splice(from, 1);
	steps.splice(to, 0, moving);
	return steps.flatMap((s) => s.rowIds.map((id) => byId.get(id)!));
}