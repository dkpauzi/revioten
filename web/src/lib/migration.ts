// Logika menu migrasi (D-215, D-216). Nomor lama diketik seperti di
// hardcopy atau nama berkasnya, misalnya SH-TR-020-02; layar memisahkan
// nomor (SH-TR-020) dan revisinya (2). Bawaannya satu set: cukup satu nomor,
// dan dua lainnya mengikuti nomor berjalan yang sama di server.
import type { LegacySetDocument } from '#lib/api/gen/index.ts';

export type TypeKey = LegacySetDocument['document_type_key'];

export const typeKeys: TypeKey[] = ['pfd', 'pfmea', 'cp'];

// SetRow adalah satu baris isian: satu dokumen lama.
export type SetRow = {
	key: TypeKey;
	number: string;
	// Isian angka: input type=number memberi number, atau '' / null bila kosong.
	revision: string | number | null;
	effectiveDate: string;
	preparedBy: string;
	approvedBy: string;
};

export function emptyRows(): SetRow[] {
	return typeKeys.map((key) => ({ key, number: '', revision: '', effectiveDate: '', preparedBy: '', approvedBy: '' }));
}

// splitLegacyNumber memisahkan akhiran revisi. ASUMSI E-142: nomor dengan
// pola {prefix}-{customer}-{seq} punya dua tanda hubung, jadi segmen angka
// keempat adalah revisi. Huruf diubah ke huruf besar.
export function splitLegacyNumber(text: string): { number: string; revision: number | null } {
	const value = text.trim().toUpperCase();
	const parts = value.split('-');
	if (parts.length >= 4 && /^\d{1,3}$/.test(parts[parts.length - 1])) {
		return { number: parts.slice(0, -1).join('-'), revision: Number(parts[parts.length - 1]) };
	}
	return { number: value, revision: null };
}

// followingNumber menulis nomor yang mengikuti nomor sumber, untuk contoh di
// layar: awalan sumber diganti awalan jenis lain. Server yang menentukan.
export function followingNumber(source: string, sourcePrefix: string, targetPrefix: string): string {
	if (source === '' || !source.startsWith(sourcePrefix + '-')) {
		return '';
	}
	return targetPrefix + source.slice(sourcePrefix.length);
}

// setDocuments menyusun baris isian untuk POST /api/v1/migration/sets. Satu
// set: ketiga dokumen; bukan satu set: hanya baris yang bernomor.
export function setDocuments(rows: SetRow[], oneSet: boolean): LegacySetDocument[] {
	return rows
		.filter((r) => oneSet || r.number.trim() !== '')
		.map((r) => ({
			document_type_key: r.key,
			number: r.number.trim(),
			// Revisi kosong dikirim -1, supaya server menolaknya dengan pesan yang jelas.
			revision: revisionText(r.revision) === '' ? -1 : Number(r.revision),
			effective_date: r.effectiveDate,
			prepared_by: r.preparedBy,
			approved_by: r.approvedBy
		}));
}

// revisionText menulis isian revisi sebagai teks tanpa spasi.
export function revisionText(value: SetRow['revision']): string {
	return value === null || value === undefined ? '' : String(value).trim();
}

// fillNames menyalin nama penyusun dan approver baris pertama ke baris yang
// masih kosong; biasanya ketiga hardcopy ditandatangani orang yang sama.
export function fillNames(rows: SetRow[]): SetRow[] {
	const [first] = rows;
	return rows.map((r) => ({
		...r,
		preparedBy: r.preparedBy.trim() === '' ? first.preparedBy : r.preparedBy,
		approvedBy: r.approvedBy.trim() === '' ? first.approvedBy : r.approvedBy
	}));
}
