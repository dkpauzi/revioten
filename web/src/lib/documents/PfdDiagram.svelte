<script lang="ts">
	// Diagram PFD (M4.10): terbentuk otomatis dari baris dokumen PFD. Simpul
	// dan panah disusun diagramOf, lalu letaknya dihitung ELK (layered, dari
	// atas ke bawah). Diagram hanya dibaca: isinya diubah di tabel workbook,
	// dan diagram digambar ulang setiap baris berubah.
	import { Background, Controls, MarkerType, SvelteFlow, type Edge, type Node } from '@xyflow/svelte';
	import '@xyflow/svelte/dist/style.css';
	import ELK from 'elkjs/lib/elk.bundled.js';
	import type { DocumentRow, FormatVersion } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';
	import { diagramOf } from '#lib/pfdDiagram.ts';
	import PfdNode from './PfdNode.svelte';

	let { rows, format }: { rows: DocumentRow[]; format: FormatVersion } = $props();

	const elk = new ELK();
	const nodeTypes = { pfd: PfdNode };
	// Ukuran simpul untuk ELK; sama dengan ukuran di PfdNode.svelte.
	const size = (symbol: string) =>
		symbol === 'ng' ? { width: 110, height: 36 } : symbol === 'connector' ? { width: 56, height: 56 } : { width: 180, height: 64 };

	let nodes = $state.raw<Node[]>([]);
	let edges = $state.raw<Edge[]>([]);
	let failure = $state('');

	// Diagram disusun ulang setiap baris atau format berubah.
	$effect(() => {
		const d = diagramOf(rows, format);
		let current = true;
		elk
			.layout({
				id: 'pfd',
				layoutOptions: {
					'elk.algorithm': 'layered',
					'elk.direction': 'DOWN',
					'elk.spacing.nodeNode': '40',
					'elk.layered.spacing.nodeNodeBetweenLayers': '40'
				},
				children: d.nodes.map((n) => ({ id: n.id, ...size(n.symbol) })),
				edges: d.edges.map((e) => ({ id: e.id, sources: [e.source], targets: [e.target] }))
			})
			.then((layout) => {
				if (!current) {
					return; // baris sudah berubah lagi; hasil ini usang
				}
				const at = new Map((layout.children ?? []).map((c) => [c.id, { x: c.x ?? 0, y: c.y ?? 0 }]));
				nodes = d.nodes.map((n) => ({ id: n.id, type: 'pfd', position: at.get(n.id) ?? { x: 0, y: 0 }, data: n }));
				edges = d.edges.map((e) => ({
					id: e.id,
					source: e.source,
					target: e.target,
					sourceHandle: e.ng ? 'ng' : undefined,
					label: e.label,
					animated: false,
					style: e.ng ? 'stroke-dasharray: 4 3; stroke: var(--danger);' : undefined,
					markerEnd: { type: MarkerType.ArrowClosed }
				}));
				failure = '';
			})
			.catch(() => {
				if (current) {
					failure = t('diagramFailed');
				}
			});
		return () => {
			current = false;
		};
	});
</script>

{#if failure}<p class="error" role="alert">{failure}</p>{/if}
{#if rows.length === 0}
	<p>{t('diagramEmpty')}</p>
{:else}
	<div class="diagram" data-testid="pfd-diagram">
		<SvelteFlow
			bind:nodes
			bind:edges
			{nodeTypes}
			fitView
			nodesDraggable={false}
			nodesConnectable={false}
			elementsSelectable={false}
			minZoom={0.2}
		>
			<Controls showLock={false} />
			<Background />
		</SvelteFlow>
	</div>
{/if}

<style>
	.diagram {
		height: 560px;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
	}

	@media (max-width: 639px) {
		.diagram {
			height: 420px;
		}
	}
</style>
