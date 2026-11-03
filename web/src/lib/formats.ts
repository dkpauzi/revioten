// Teks dan bantuan layar format template (M3.1, R-TPL-01, R-TPL-02).
import type {
	FieldOption,
	FormatField,
	FormatSymbol,
	FormatTemplate,
	FormatVersion,
	RatingCriterion
} from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

const kindKeys: Record<FormatField['kind'], MessageKey> = {
	text: 'formatKindText',
	date: 'formatKindDate',
	select: 'formatKindSelect',
	symbol: 'formatKindSymbol',
	rating: 'formatKindRating',
	rpn: 'formatKindRpn'
};

const scaleKeys: Record<RatingCriterion['scale'], MessageKey> = {
	severity: 'formatScaleSeverity',
	occurrence: 'formatScaleOccurrence',
	detection: 'formatScaleDetection'
};

const methodologyKeys: Record<NonNullable<FormatTemplate['methodology']>, MessageKey> = {
	aiag_4th: 'formatMethodAiag4th',
	aiag_vda: 'formatMethodAiagVda',
	sqam: 'formatMethodSqam'
};

const riskKeys: Record<FormatVersion['risk_method'], MessageKey> = {
	rpn: 'formatRiskRpn',
	ap: 'formatRiskAp',
	none: 'formatRiskNone'
};

// kindText menulis jenis field. Rating dan RPN menyebut skala dan setnya,
// misalnya "Rating: Severity, action results".
export function kindText(f: FormatField): string {
	if (f.kind === 'rating' && f.rating_scale) {
		return t('formatKindRatingDetail', { scale: scaleText(f.rating_scale), set: setText(f.rating_set) });
	}
	if (f.kind === 'rpn') {
		return t('formatKindRpnDetail', { set: setText(f.rating_set) });
	}
	return t(kindKeys[f.kind]);
}

// kindName memberi nama jenis field untuk pilihan di formulir.
export function kindName(kind: FormatField['kind']): string {
	return t(kindKeys[kind]);
}

// scaleText memberi nama skala rating.
export function scaleText(scale: RatingCriterion['scale']): string {
	return t(scaleKeys[scale]);
}

function setText(set?: number): string {
	return set === 2 ? t('formatSetResult') : t('formatSetCurrent');
}

// methodologyText memberi nama metodologi; kosong untuk form yang dipakai sekarang.
export function methodologyText(methodology?: FormatTemplate['methodology']): string {
	return methodology ? t(methodologyKeys[methodology]) : '';
}

// riskText memberi nama metode risiko.
export function riskText(risk: FormatVersion['risk_method']): string {
	return t(riskKeys[risk]);
}

// templateLabel menulis satu format template untuk daftar pilihan, misalnya
// "PFMEA AIAG 4th · PFMEA" atau "PFMEA SQAM · PFMEA · KT".
export function templateLabel(ft: FormatTemplate): string {
	return [ft.name, ft.document_type.name, ft.customer_code].filter((part) => part).join(' · ');
}

// fieldsOf mengambil field satu area menurut urutan.
export function fieldsOf(v: FormatVersion, area: FormatField['area']): FormatField[] {
	return v.fields.filter((f) => f.area === area);
}

// ratingsOf mengambil kriteria satu skala, nilai 1 sampai 10.
export function ratingsOf(v: FormatVersion, scale: RatingCriterion['scale']): RatingCriterion[] {
	return v.ratings.filter((r) => r.scale === scale);
}

// lines memecah teks kotak isian menjadi baris yang tidak kosong.
function lines(text: string): string[] {
	return text
		.split('\n')
		.map((line) => line.trim())
		.filter((line) => line !== '');
}

// optionLines menulis label pilihan, satu per baris, untuk kotak isian.
export function optionLines(options: FieldOption[] = []): string {
	return options.map((o) => o.label).join('\n');
}

// mergeOptions menyusun pilihan dari kotak isian: baris ke-i mengganti label
// pilihan lama ke-i (key-nya tetap), dan baris sesudahnya menjadi pilihan
// baru tanpa key. Pilihan tidak bisa dihapus; server menolak daftar yang
// lebih pendek dari pilihan lama.
export function mergeOptions(existing: FieldOption[], text: string): FieldOption[] {
	return lines(text).map((label, i) => (i < existing.length ? { key: existing[i].key, label } : { label }));
}

// automaticSymbolNames menulis nama simbol yang dipasang sistem, misalnya
// Revision pada baris yang berubah (R-REV-09), dipisah koma; kosong bila tidak ada.
export function automaticSymbolNames(symbols: FormatSymbol[]): string {
	return symbols
		.filter((s) => s.automatic)
		.map((s) => s.name)
		.join(', ');
}

// symbolLines dan mergeSymbols bekerja seperti optionLines dan mergeOptions,
// untuk simbol karakteristik (R-TPL-11).
export function symbolLines(symbols: FormatSymbol[]): string {
	return symbols.map((s) => s.name).join('\n');
}

export function mergeSymbols(existing: FormatSymbol[], text: string): FormatSymbol[] {
	return lines(text).map((name, i) => (i < existing.length ? { key: existing[i].key, name } : { name }));
}
