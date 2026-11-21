// Bantuan layar process template atau pustaka proses (M3.2, R-TPL-01, R-TPL-04).
import type {
	DocumentType,
	FormatVersion,
	PointScope,
	ProcessTemplate,
	ProcessTemplateVersion,
	Subprocess,
	TemplatePoint
} from './api/gen/index.ts';
import { t } from './i18n/index.ts';

// templateOf mencari process template milik satu proses. D-129 (keputusan E-72): satu
// proses paling banyak punya satu template.
export function templateOf(templates: ProcessTemplate[], processId: string): ProcessTemplate | null {
	return templates.find((pt) => pt.process.id === processId) ?? null;
}

// pointTypes mengambil jenis dokumen yang punya poin (PFD, PFMEA, dan CP),
// menurut urutan dari server.
export function pointTypes(types: DocumentType[]): DocumentType[] {
	return types.filter((d) => d.key !== undefined);
}

// newPointTypes adalah jenis dokumen yang boleh mendapat poin baru: PFMEA dan
// CP. PFD diisi per dokumen tanpa template (D-214); poin PFD lama tetap tampil.
export function newPointTypes(types: DocumentType[]): DocumentType[] {
	return pointTypes(types).filter((d) => d.key !== 'pfd');
}

// pointsOf mengambil poin satu sub-proses untuk satu jenis dokumen, menurut urutan.
export function pointsOf(s: Subprocess, documentTypeId: string): TemplatePoint[] {
	return s.points.filter((p) => p.document_type.id === documentTypeId);
}

// pointSummary menulis banyaknya poin per jenis dokumen, misalnya
// "PFD 1 · PFMEA 2". Jenis tanpa poin tidak ditulis.
export function pointSummary(s: Subprocess, types: DocumentType[]): string {
	return types
		.map((d) => [d.name, pointsOf(s, d.id).length] as const)
		.filter(([, count]) => count > 0)
		.map(([name, count]) => `${name} ${count}`)
		.join(' · ');
}

// ScopeLevel adalah tingkat cakupan poin (M3.3, D-129).
export type ScopeLevel = PointScope['level'];

// scopeLevels adalah semua tingkat dari yang terluas ke yang tersempit.
export const scopeLevels: ScopeLevel[] = ['general', 'customer', 'model_group', 'model'];

// scopeText menulis cakupan untuk layar, misalnya "General" atau "Model K-125".
export function scopeText(scope: PointScope): string {
	const label = scope.label ?? '';
	switch (scope.level) {
		case 'customer':
			return t('scopeCustomer', { label });
		case 'model_group':
			return t('scopeModelGroup', { label });
		case 'model':
			return t('scopeModel', { label });
		default:
			return t('scopeGeneral');
	}
}

// variantLevels mengembalikan tingkat yang boleh dipakai varian sebuah poin:
// hanya yang lebih sempit dari cakupan poin asalnya. Server tetap memeriksa
// bahwa sasarannya berada di dalam poin asal.
export function variantLevels(origin: ScopeLevel): Exclude<ScopeLevel, 'general'>[] {
	const from = scopeLevels.indexOf(origin);
	return scopeLevels.slice(from + 1) as Exclude<ScopeLevel, 'general'>[];
}

// versionText menulis nomor dan status versi, misalnya "Version 1 · Draft".
export function versionText(v: Pick<ProcessTemplateVersion, 'version' | 'status'>): string {
	return t('processTemplateVersion', {
		version: String(v.version),
		status: v.status === 'draft' ? t('processTemplateDraft') : t('processTemplatePublished')
	});
}

// PointColumn adalah satu kolom yang bisa diisi di poin (D-146): key tetap
// kolom itu, dan nama form yang memilikinya.
export type PointColumn = { key: string; kind: 'text' | 'rating'; label: string; forms: string[] };

// pointColumns menggabungkan kolom ber-key dari form satu jenis dokumen.
// Kolom dengan key yang sama di dua form muncul sekali, dengan label form
// pertama. Hanya kolom teks dan rating yang diisi di poin; pilihan, simbol,
// dan tanggal diisi di dokumen (ASUMSI E-92).
export function pointColumns(forms: { name: string; version: Pick<FormatVersion, 'fields'> }[]): PointColumn[] {
	const out: PointColumn[] = [];
	for (const form of forms) {
		for (const f of form.version.fields) {
			// Relevant Document diisi dari master dokumen terkait, bukan dari poin (D-130).
			if (f.area !== 'row' || f.hidden || !f.key || f.key === 'relevant_document' || (f.kind !== 'text' && f.kind !== 'rating')) {
				continue;
			}
			const known = out.find((c) => c.key === f.key);
			if (known) {
				known.forms.push(form.name);
			} else {
				out.push({ key: f.key, kind: f.kind, label: f.label, forms: [form.name] });
			}
		}
	}
	return out;
}

// contentInput menyiapkan isi poin untuk server: isian kosong dibuang, rating
// dikirim sebagai angka. Isi lama dengan key di luar daftar kolom (misalnya
// kolom form yang sedang disembunyikan) dibawa apa adanya.
export function contentInput(columns: PointColumn[], values: Record<string, string>, old: Record<string, unknown>): Record<string, unknown> {
	const keys = new Set(columns.map((c) => c.key));
	const out: Record<string, unknown> = Object.fromEntries(Object.entries(old).filter(([key]) => !keys.has(key)));
	for (const c of columns) {
		const value = (values[c.key] ?? '').trim();
		if (value !== '') {
			out[c.key] = c.kind === 'rating' ? Number(value) : value;
		}
	}
	return out;
}
