// Rakit PFMEA dan CP dari langkah PFD (C4, D-211). Logikanya di sini, terpisah
// dari panel, supaya bisa dites dengan Vitest.
import type { PfdComposition, PfdStep, PfdStepKey } from '#lib/api/gen/index.ts';

// stepKey menandai satu langkah PFD: bagian, nomor langkah, dan proses.
export function stepKey(s: { section_key: string; operation: string; process_id: string }): string {
	return [s.section_key, s.operation, s.process_id].join('\u0000');
}

// open melaporkan apakah langkah itu masih akan mengubah salah satu dokumen.
export function open(s: PfdStep): boolean {
	return s.targets.some((t) => t.state === 'to_add' || t.state === 'no_template');
}

// initialChoice mencentang semua langkah yang belum ada di PFMEA atau CP.
export function initialChoice(view: PfdComposition): Record<string, boolean> {
	return Object.fromEntries(view.steps.map((s) => [stepKey(s), open(s)]));
}

// chosenSteps memberi isian API untuk langkah yang dicentang dan masih terbuka.
export function chosenSteps(view: PfdComposition, chosen: Record<string, boolean>): PfdStepKey[] {
	return view.steps
		.filter((s) => chosen[stepKey(s)] && open(s))
		.map((s) => ({ section_key: s.section_key, operation: s.operation, process_id: s.process_id }));
}

// Warning adalah satu hasil pemeriksaan PFD dengan PFMEA atau CP.
export type Warning = {
	kind: 'missing' | 'extra';
	operation: string;
	process: string;
	documentType: string; // pfmea atau cp
	sectionKey: string;
};

// warnings memeriksa kecocokan langkah (C4, E-136): langkah PFD yang belum
// ada di PFMEA atau CP, dan nomor langkah PFMEA atau CP yang tidak ada di PFD.
export function warnings(view: PfdComposition): Warning[] {
	const out: Warning[] = [];
	for (const s of view.steps) {
		for (const t of s.targets) {
			if (t.state !== 'present') {
				out.push({ kind: 'missing', operation: s.operation, process: s.process_name, documentType: t.document_type_key, sectionKey: s.section_key });
			}
		}
	}
	for (const e of view.extra) {
		out.push({ kind: 'extra', operation: e.operation, process: e.process_name, documentType: e.document_type_key, sectionKey: e.section_key });
	}
	return out;
}
