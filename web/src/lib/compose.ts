// Logika panel perakitan dokumen dari process template (M4.5, R-CMP-01
// sampai R-CMP-03). Fungsi di sini tidak menyentuh layar, supaya mudah diuji.
import type { ComposeInput, Composition, ProcessTemplate } from './api/gen/index.ts';

// PublishedChoice adalah satu process template yang bisa dipakai merakit:
// versi terbit terbarunya.
export type PublishedChoice = { versionId: string; version: number; process: string };

// publishedChoices memberi versi terbit terbaru setiap process template,
// menurut nama proses. Template yang belum punya versi terbit dilewati.
export function publishedChoices(templates: ProcessTemplate[]): PublishedChoice[] {
	const out: PublishedChoice[] = [];
	for (const template of templates) {
		const published = template.versions.filter((v) => v.status === 'published').sort((a, b) => b.version - a.version)[0];
		if (published) {
			out.push({ versionId: published.id, version: published.version, process: template.process.name });
		}
	}
	return out.sort((a, b) => a.process.localeCompare(b.process));
}

// Checks adalah centang per ID poin.
export type Checks = Record<string, boolean>;

// initialChecks: poin yang sudah punya baris tercentang; yang dikecualikan
// atau belum diputuskan tidak.
export function initialChecks(view: Composition): Checks {
	const out: Checks = {};
	for (const s of view.subprocesses) {
		for (const p of s.points) {
			out[p.point_id] = p.state === 'included';
		}
	}
	return out;
}

// composeInput menyusun keputusan untuk server: poin tercentang dipakai,
// poin tanpa centang dikecualikan dengan alasannya bila diisi (R-CMP-03).
export function composeInput(view: Composition, checks: Checks, reasons: Record<string, string>): ComposeInput {
	const include: string[] = [];
	const exclude: NonNullable<ComposeInput['exclude']> = [];
	for (const s of view.subprocesses) {
		for (const p of s.points) {
			if (checks[p.point_id]) {
				include.push(p.point_id);
				continue;
			}
			const reason = (reasons[p.point_id] ?? '').trim();
			exclude.push(reason === '' ? { point_id: p.point_id } : { point_id: p.point_id, reason });
		}
	}
	return { process_template_version_id: view.process_template_version_id, include, exclude };
}

// ResultRow adalah satu baris di panel kanan: hasil perakitan untuk satu
// dokumen. keep berarti barisnya sudah ada dan tetap, add akan ditambahkan,
// remove akan dihapus.
export type ResultRow = { pointId: string; subprocess: string; text: string; change: 'keep' | 'add' | 'remove' };

// resultRows memberi baris satu dokumen sesudah keputusan diterapkan,
// menurut urutan template. Poin tanpa centang yang memang belum punya baris
// tidak ditampilkan.
export function resultRows(view: Composition, checks: Checks, documentId: string): ResultRow[] {
	const out: ResultRow[] = [];
	for (const s of view.subprocesses) {
		for (const p of s.points) {
			if (p.document_id !== documentId) {
				continue;
			}
			const included = p.state === 'included';
			const checked = checks[p.point_id] ?? false;
			if (!included && !checked) {
				continue;
			}
			out.push({ pointId: p.point_id, subprocess: s.name, text: p.text, change: included && checked ? 'keep' : checked ? 'add' : 'remove' });
		}
	}
	return out;
}
