// Bantuan layar process template atau pustaka proses (M3.2, R-TPL-01, R-TPL-04).
import type { DocumentType, ProcessTemplate, ProcessTemplateVersion, Subprocess, TemplatePoint } from './api/gen/index.ts';
import { t } from './i18n/index.ts';

// templateOf mencari process template milik satu proses. ASUMSI E-72: satu
// proses paling banyak punya satu template.
export function templateOf(templates: ProcessTemplate[], processId: string): ProcessTemplate | null {
	return templates.find((pt) => pt.process.id === processId) ?? null;
}

// pointTypes mengambil jenis dokumen yang punya poin (PFD, PFMEA, dan CP),
// menurut urutan dari server.
export function pointTypes(types: DocumentType[]): DocumentType[] {
	return types.filter((d) => d.key !== undefined);
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

// versionText menulis nomor dan status versi, misalnya "Version 1 · Draft".
export function versionText(v: Pick<ProcessTemplateVersion, 'version' | 'status'>): string {
	return t('processTemplateVersion', {
		version: String(v.version),
		status: v.status === 'draft' ? t('processTemplateDraft') : t('processTemplatePublished')
	});
}
