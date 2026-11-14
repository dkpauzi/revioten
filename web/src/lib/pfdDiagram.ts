// Diagram PFD (M4.10): disusun otomatis dari baris dokumen PFD. Fungsi di
// sini tidak menyentuh layar, supaya mudah diuji; letak simpul dihitung ELK
// di komponen PfdDiagram.svelte.
//
// Kolom dicari lewat key tetap (D-146, D-150), bukan label: flow_symbol
// untuk bentuk simpul, ng_disposition untuk simpul NG di samping langkah.
import type { DocumentRow, FormatField, FormatVersion } from './api/gen/index.ts';

// DiagramNode adalah satu simpul: satu langkah, atau satu disposisi NG.
export type DiagramNode = { id: string; label: string; symbol: string; symbolLabel: string };
// DiagramEdge adalah satu panah. ng menandai panah ke disposisi NG.
export type DiagramEdge = { id: string; source: string; target: string; ng: boolean; label?: string };

function fieldByKey(format: Pick<FormatVersion, 'fields'>, key: string): FormatField | undefined {
	return format.fields.find((f) => f.area === 'row' && f.key === key);
}

function optionLabel(field: FormatField | undefined, value: unknown): string {
	return field?.options?.find((o) => o.key === value)?.label ?? String(value ?? '');
}

// diagramOf menyusun simpul dan panah dari baris PFD, menurut urutan baris.
// Baris tanpa simbol memakai simbol process (ASUMSI E-95).
export function diagramOf(rows: DocumentRow[], format: Pick<FormatVersion, 'fields'>): { nodes: DiagramNode[]; edges: DiagramEdge[] } {
	const symbolField = fieldByKey(format, 'flow_symbol');
	const ngField = fieldByKey(format, 'ng_disposition');
	const nodes: DiagramNode[] = [];
	const edges: DiagramEdge[] = [];
	rows.forEach((r, i) => {
		const symbol = symbolField ? String(r.content[symbolField.id] ?? '') || 'process' : 'process';
		const label = [r.operation_number ?? '', r.process_name ?? ''].filter((s) => s !== '').join(' ');
		// Tanpa kolom simbol tidak ada label simbol yang bisa ditampilkan.
		nodes.push({ id: r.id, label, symbol, symbolLabel: symbolField ? optionLabel(symbolField, symbol) : '' });
		const ng = ngField ? r.content[ngField.id] : undefined;
		if (ngField && ng) {
			const id = `ng-${r.id}`;
			nodes.push({ id, label: optionLabel(ngField, ng), symbol: 'ng', symbolLabel: ngField.label });
			edges.push({ id: `${r.id}-${id}`, source: r.id, target: id, ng: true, label: 'NG' });
		}
		const next = rows[i + 1];
		if (next) {
			edges.push({ id: `${r.id}-${next.id}`, source: r.id, target: next.id, ng: false });
		}
	});
	return { nodes, edges };
}