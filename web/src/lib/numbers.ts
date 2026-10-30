// Teks layar untuk register nomor dokumen (Tahap 2).
import type { DocumentNumber } from './api/gen/index.ts';

// registeredText menulis nomor hasil pendaftaran, dipisah koma. Revisi awal
// ditulis di belakang nomor bila bukan 0, misalnya "SC-AB-001 (-04)" (M2.4).
export function registeredText(numbers: DocumentNumber[]): string {
	return numbers
		.filter((n) => n.status !== 'skipped')
		.map((n) => (n.initial_revision === 0 ? n.number : `${n.number} (${n.revision_label})`))
		.join(', ');
}
