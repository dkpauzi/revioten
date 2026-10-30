// Teks layar untuk register nomor dokumen (Tahap 2).
import type { DocumentNumber } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

const statusKeys: Record<DocumentNumber['status'], MessageKey> = {
	reserved: 'numberStatusReserved',
	in_use: 'numberStatusInUse',
	released: 'numberStatusReleased',
	obsolete: 'numberStatusObsolete',
	void: 'numberStatusVoid',
	skipped: 'numberStatusSkipped'
};

// skippedText menulis nomor yang dilewati saat daftar sebagai set (M2.4,
// ASUMSI E-64), dipisah koma, atau "" bila tidak ada.
export function skippedText(numbers: DocumentNumber[]): string {
	return numbers
		.filter((n) => n.status === 'skipped')
		.map((n) => n.number)
		.join(', ');
}

// statusLabel memberi nama tampilan status nomor (R-NUM-04).
export function statusLabel(status: DocumentNumber['status']): string {
	return t(statusKeys[status]);
}

// reservedDays menghitung berapa hari penuh nomor sudah Reserved, untuk
// melihat nomor yang terlalu lama dipesan tapi tidak dipakai (R-NUM-04,
// ASUMSI E-62: belum ada ambang dan pengingat otomatis).
export function reservedDays(createdAt: string, now: Date): number {
	const day = 24 * 60 * 60 * 1000;
	return Math.max(0, Math.floor((now.getTime() - new Date(createdAt).getTime()) / day));
}

// canVoid dan canObsolete mengikuti perubahan status yang boleh di Tahap 2
// (M2.3). Server tetap memeriksanya.
export function canVoid(n: DocumentNumber): boolean {
	return n.status === 'reserved' || n.status === 'in_use';
}

export function canObsolete(n: DocumentNumber): boolean {
	return n.status === 'released';
}

// registeredText menulis nomor hasil pendaftaran, dipisah koma. Revisi awal
// ditulis di belakang nomor bila bukan 0, misalnya "SC-AB-001 (-04)" (M2.4).
export function registeredText(numbers: DocumentNumber[]): string {
	return numbers
		.filter((n) => n.status !== 'skipped')
		.map((n) => (n.initial_revision === 0 ? n.number : `${n.number} (${n.revision_label})`))
		.join(', ');
}
