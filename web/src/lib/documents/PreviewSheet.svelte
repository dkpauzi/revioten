<script lang="ts">
	// Pratinjau lembar (C2, D-122, G11): dokumen yang sedang dirancang tampil per
	// halaman kertas seperti cetakannya, supaya tampilannya bisa dipastikan.
	// Kolom Process Flow digambar dari langkah PFD: simbol dan nomor langkah,
	// garis alur, cabang NG, dan konektor A, B, C di setiap batas halaman.
	// Semuanya dihitung ulang dari baris, jadi menambah atau menghapus baris
	// tidak pernah membuat orang menggeser gambar.
	//
	// Cara kerja: (1) semua baris digambar sekali di halaman ukur yang tidak
	// terlihat, lalu tinggi setiap baris diukur; (2) baris dibagi ke halaman
	// (paginate); (3) setiap halaman digambar, lalu posisi barisnya diukur
	// untuk menggambar alur di atas kolom Process Flow.
	import { tick } from 'svelte';
	import type { DocumentRow, FormatField, FormatVersion, PrintLayoutContent } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';
	import { flowPages, groupsOf, paginate, paperSize, previewText, pxPerMm, stepsOf, type FlowPage, type Step } from './preview.ts';

	let {
		number,
		revision,
		typeKey,
		header,
		rows,
		format,
		layout,
		steps
	}: {
		number: string;
		revision: string;
		typeKey: string;
		// header: pasangan label dan nilai di kepala form, sama seperti cetakan.
		header: { label: string; value: string }[];
		rows: DocumentRow[];
		format: FormatVersion;
		layout: PrintLayoutContent;
		// steps: langkah PFD menurut nomor langkah, untuk kolom Process Flow.
		steps: Map<string, Step>;
	} = $props();

	const titles: Record<string, string> = { pfd: 'PROCESS FLOW DIAGRAM', pfmea: 'PROCESS FMEA', cp: 'CONTROL PLAN' };

	let fields = $derived(format.fields.filter((f) => f.area === 'row' && !f.hidden).sort((a, b) => a.position - b.position));
	// Kolom tempat alur digambar: Process Flow di CP, Flow Symbol di PFD.
	let flowIndex = $derived(fields.findIndex((f) => f.kind === 'process_flow' || (typeKey === 'pfd' && f.key === 'flow_symbol')));
	// Kolom yang digabung untuk baris satu langkah proses, seperti di Excel.
	let merged = $derived(new Set(fields.map((f, i) => (f.kind === 'operation_number' || f.kind === 'process' ? i : -1)).filter((i) => i >= 0)));
	let hasGroups = $derived(fields.some((f) => f.group));

	// widthOf: lebar relatif kolom dari layout cetak (M6.1). Kolom alur diberi
	// lebar bawaan yang cukup untuk simbol dan cabang NG.
	function widthOf(f: FormatField): number {
		const fromLayout = (f.key && layout.widths[f.key]) || layout.widths[f.id];
		if (fromLayout) return fromLayout;
		if (f.kind === 'process_flow' || f.key === 'flow_symbol') return 1.3;
		if (['operation_number', 'rating', 'rpn', 'ap', 'symbol'].includes(f.kind)) return 0.5;
		if (f.kind === 'process') return 1.6;
		if (f.kind === 'date') return 0.8;
		return 1.4;
	}
	let widths = $derived.by(() => {
		const raw = fields.map(widthOf);
		const total = raw.reduce((a, b) => a + b, 0) || 1;
		return raw.map((w) => (w / total) * 100);
	});

	// Kelompok header kolom: kolom berurutan dengan judul grup yang sama.
	let headGroups = $derived.by(() => {
		const out: { label: string; span: number }[] = [];
		for (const f of fields) {
			const last = out[out.length - 1];
			if (last && f.group && last.label === f.group) last.span++;
			else out.push({ label: f.group ?? '', span: 1 });
		}
		return out;
	});

	let paper = $derived(paperSize(layout));
	let contentHeightPx = $derived((paper.heightMm - layout.margins_mm.top - layout.margins_mm.bottom) * pxPerMm);

	let groups = $derived(groupsOf(rows));
	// groupOf[i]: kelompok baris ke-i, untuk menggabung sel per halaman.
	let groupOf = $derived.by(() => {
		const out: number[] = [];
		groups.forEach((g, gi) => {
			for (let i = g.start; i <= g.end; i++) out[i] = gi;
		});
		return out;
	});

	let pages = $state<number[][] | null>(null);
	let flow = $derived(pages ? flowPages(groups, pages, steps) : []);

	// Pengukuran tinggi baris di halaman ukur.
	let measure = $state<HTMLDivElement>();
	$effect(() => {
		// Ukur ulang bila baris, format, atau layout berubah.
		void rows;
		void fields;
		void layout;
		pages = null;
		tick().then(() => {
			if (!measure) return;
			const head = measure.querySelector<HTMLElement>('.form-head')?.offsetHeight ?? 0;
			const thead = measure.querySelector<HTMLElement>('thead')?.offsetHeight ?? 0;
			const foot = measure.querySelector<HTMLElement>('.page-foot')?.offsetHeight ?? 0;
			const heights = Array.from(measure.querySelectorAll<HTMLElement>('tbody tr')).map((tr) => tr.offsetHeight);
			// Sedikit ruang cadangan, supaya pembulatan tidak mendorong baris keluar halaman.
			const available = contentHeightPx - head - thead - foot - 8;
			pages = paginate(heights, available, available);
		});
	});

	// Posisi baris setiap halaman yang sudah digambar, untuk menggambar alur.
	type Geometry = { x: number; width: number; top: number; bottom: number; rows: { top: number; height: number }[] };
	let tables = $state<HTMLTableElement[]>([]);
	let geometry = $state<Geometry[]>([]);
	$effect(() => {
		void pages;
		if (!pages || flowIndex < 0) {
			geometry = [];
			return;
		}
		tick().then(() => {
			geometry = tables.map((table) => {
				const th = table.querySelectorAll<HTMLElement>('thead tr:last-child th')[flowIndex];
				const body = table.tBodies[0];
				const bodyRows = Array.from(body.rows);
				return {
					x: th?.offsetLeft ?? 0,
					width: th?.offsetWidth ?? 0,
					top: body.offsetTop,
					bottom: body.offsetTop + body.offsetHeight,
					rows: bodyRows.map((r) => ({ top: r.offsetTop, height: r.offsetHeight }))
				};
			});
		});
	});

	// Skala tampilan: halaman dikecilkan supaya muat selebar area pratinjau.
	let areaWidth = $state(0);
	let scale = $derived(areaWidth > 0 ? Math.min(1, areaWidth / (paper.widthMm * pxPerMm)) : 1);

	// rowspan sel gabungan di halaman p, baris ke-i halaman itu: 0 berarti sel
	// itu tertutup sel di atasnya.
	function span(pageRows: number[], i: number): number {
		const g = groupOf[pageRows[i]];
		if (i > 0 && groupOf[pageRows[i - 1]] === g) return 0;
		let n = 1;
		while (i + n < pageRows.length && groupOf[pageRows[i + n]] === g) n++;
		return n;
	}

	// Ukuran simbol dan posisi alur di kolom Process Flow.
	const shapeW = 46;
	const shapeH = 26;
	const ngLabels: Record<string, string> = { scrap: 'SCRAP', ncp: 'NCP', rework: 'REWORK' };

	function nodeY(g: Geometry, row: number): number {
		const r = g.rows[row];
		return r.top - g.top + Math.min(r.height / 2, 6 + shapeH / 2);
	}

	// shapePath menggambar simbol alur (bentuk mengikuti simbol PFD TR-020).
	function shape(symbol: string, cx: number, cy: number): string {
		const l = cx - shapeW / 2;
		const r = cx + shapeW / 2;
		const top = cy - shapeH / 2;
		const bottom = cy + shapeH / 2;
		switch (symbol) {
			case 'decision':
				return `M ${cx} ${top - 3} L ${r + 4} ${cy} L ${cx} ${bottom + 3} L ${l - 4} ${cy} Z`;
			case 'merge':
				return `M ${l} ${top} L ${r} ${top} L ${cx} ${bottom} Z`;
			case 'document':
				return `M ${l} ${top} L ${r} ${top} L ${r} ${bottom - 4} Q ${cx + shapeW / 4} ${bottom - 10} ${cx} ${bottom - 4} T ${l} ${bottom - 4} Z`;
			default:
				return `M ${l} ${top} H ${r} V ${bottom} H ${l} Z`;
		}
	}
</script>

{#snippet sheetHead()}
	<div class="form-head">
		<p class="form-title">{titles[typeKey] ?? typeKey.toUpperCase()}</p>
		<dl class="form-fields">
			{#each header as pair (pair.label)}
				<div><dt>{pair.label}</dt><dd>{pair.value}</dd></div>
			{/each}
		</dl>
	</div>
{/snippet}

{#snippet tableHead()}
	<colgroup>
		{#each widths as w, i (i)}<col style="width: {w}%" />{/each}
	</colgroup>
	<thead>
		{#if hasGroups}
			<tr>
				{#each headGroups as g, i (i)}<th colspan={g.span} class:empty={!g.label}>{g.label}</th>{/each}
			</tr>
		{/if}
		<tr>
			{#each fields as f (f.id)}<th>{f.label}</th>{/each}
		</tr>
	</thead>
{/snippet}

{#snippet foot(page: number, total: number)}
	<p class="page-foot"><span>{number} · {revision}</span><span>{t('previewPage', { page: String(page), total: String(total) })}</span></p>
{/snippet}

<div class="preview" bind:clientWidth={areaWidth}>
	<p class="hint">{t('previewHint')}</p>

	<!-- Halaman ukur: tidak terlihat, selebar halaman sungguhan. -->
	<div class="measure" aria-hidden="true">
		<div
			class="paper"
			bind:this={measure}
			style="width: {paper.widthMm}mm; padding: {layout.margins_mm.top}mm {layout.margins_mm.right}mm {layout.margins_mm.bottom}mm {layout.margins_mm.left}mm; font-size: {layout.font_size}pt"
		>
			{@render sheetHead()}
			<table class="form">
				{@render tableHead()}
				<tbody>
					{#each rows as row (row.id)}
						<tr>
							{#each fields as f, c (f.id)}<td class:flow-cell={c === flowIndex}>{previewText(f, row, fields, format.symbols)}</td>{/each}
						</tr>
					{/each}
				</tbody>
			</table>
			{@render foot(1, 1)}
		</div>
	</div>

	{#if pages}
		{#each pages as pageRows, p (p)}
			{@const page = flow[p]}
			{@const g = geometry[p]}
			<div class="page-slot" style="height: {paper.heightMm * pxPerMm * scale + 16}px">
				<div
					class="paper"
					role="group"
					aria-label={t('previewPage', { page: String(p + 1), total: String(pages.length) })}
					style="width: {paper.widthMm}mm; height: {paper.heightMm}mm; padding: {layout.margins_mm.top}mm {layout.margins_mm.right}mm {layout.margins_mm.bottom}mm {layout.margins_mm.left}mm; font-size: {layout.font_size}pt; transform: scale({scale})"
				>
					{@render sheetHead()}
					<div class="table-wrap">
						<table class="form" bind:this={tables[p]}>
							{@render tableHead()}
							<tbody>
								{#each pageRows as index, i (index)}
									{@const row = rows[index]}
									<tr>
										{#each fields as f, c (f.id)}
											{#if merged.has(c)}
												{@const n = span(pageRows, i)}
												{#if n > 0}<td rowspan={n} class="merged">{previewText(f, row, fields, format.symbols)}</td>{/if}
											{:else if c === flowIndex}
												<td class="flow-cell"></td>
											{:else}
												<td>{previewText(f, row, fields, format.symbols)}</td>
											{/if}
										{/each}
									</tr>
								{/each}
							</tbody>
						</table>
						{#if page && g && flowIndex >= 0}
							{@const cx = g.x + Math.min(g.width / 2, shapeW / 2 + 10)}
							{@const height = g.bottom - g.top}
							{@const firstY = page.nodes.length > 0 ? nodeY(g, page.nodes[0].row) : height}
							{@const lastY = page.nodes.length > 0 ? nodeY(g, page.nodes[page.nodes.length - 1].row) : 0}
							<svg
								class="flow"
								data-testid="process-flow"
								style="left: {0}px; top: {g.top}px; width: {g.x + g.width}px; height: {height}px"
								viewBox="0 0 {g.x + g.width} {height}"
								aria-label={t('previewFlow')}
								role="img"
							>
								<!-- Garis utama: dari konektor atas (atau simbol pertama) sampai konektor bawah (atau simbol terakhir). -->
								{#if page.inLetter || page.outLetter || page.nodes.length > 1}
									<line x1={cx} x2={cx} y1={page.inLetter ? 12 : firstY} y2={page.outLetter ? height - 12 : lastY} class="flow-line" />
								{/if}
								{#if page.inLetter}
									<circle cx={cx} cy={12} r={10} class="connector" data-connector={page.inLetter} />
									<text x={cx} y={12} class="connector-text">{page.inLetter}</text>
								{/if}
								{#each page.nodes as node (node.op + ':' + node.row)}
									{@const y = nodeY(g, node.row)}
									{#if node.ng}
										<line x1={cx + shapeW / 2 + 4} x2={g.x + g.width - 4} y1={y} y2={y} class="flow-line" />
										<text x={cx + shapeW / 2 + 8} y={y - 4} class="flow-label">NG</text>
										<text x={g.x + g.width - 4} y={y + 10} class="flow-label ng" text-anchor="end">{ngLabels[node.ng] ?? node.ng.toUpperCase()}</text>
										<text x={cx + 4} y={y + shapeH / 2 + 10} class="flow-label">OK</text>
									{/if}
									{#if node.symbol === 'connector'}
										<circle cx={cx} cy={y} r={shapeH / 2} class="shape" />
									{:else if node.symbol === 'alternate_process'}
										<rect x={cx - shapeW / 2} y={y - shapeH / 2} width={shapeW} height={shapeH} rx="8" class="shape" />
									{:else}
										<path d={shape(node.symbol, cx, y)} class="shape" />
									{/if}
									<text x={cx} y={y} class="shape-text" data-step={node.op}>{node.op}</text>
								{/each}
								{#if page.outLetter}
									<circle cx={cx} cy={height - 12} r={10} class="connector" data-connector={page.outLetter} />
									<text x={cx} y={height - 12} class="connector-text">{page.outLetter}</text>
								{/if}
							</svg>
						{/if}
					</div>
					{@render foot(p + 1, pages.length)}
				</div>
			</div>
		{/each}
	{:else}
		<p>{t('formatLoading')}</p>
	{/if}
</div>

<style>
	.preview {
		overflow: hidden;
	}

	.measure {
		position: absolute;
		left: -100000px;
		top: 0;
		visibility: hidden;
	}

	.page-slot {
		position: relative;
	}

	/* Satu halaman kertas: ukurannya milimeter, lalu dikecilkan dengan transform. */
	.paper {
		box-sizing: border-box;
		background: #ffffff;
		color: #000000;
		font-family: Arial, Helvetica, sans-serif;
		line-height: 1.25;
		box-shadow: 0 2px 10px rgba(16, 24, 40, 0.12);
		transform-origin: top left;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.form-head {
		margin-bottom: 3mm;
	}

	.form-title {
		margin: 0 0 2mm;
		text-align: center;
		font-weight: 700;
		font-size: 1.4em;
	}

	.form-fields {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0;
		margin: 0;
		border: 1px solid #000000;
	}

	.form-fields div {
		display: flex;
		border: 0.5px solid #000000;
		padding: 1mm 2mm;
		gap: 2mm;
	}

	.form-fields dt {
		font-weight: 600;
		white-space: nowrap;
	}

	.form-fields dd {
		margin: 0;
	}

	.table-wrap {
		position: relative;
	}

	.form {
		width: 100%;
		table-layout: fixed;
		border-collapse: collapse;
		margin: 0;
	}

	.form th,
	.form td {
		border: 1px solid #000000;
		padding: 1mm;
		vertical-align: top;
		overflow-wrap: anywhere;
		background: #ffffff;
		color: #000000;
		font-size: 1em;
		text-align: left;
	}

	.form th {
		text-align: center;
		vertical-align: middle;
		font-weight: 700;
		background: #f2f2f2;
	}

	.form th.empty {
		border-bottom: none;
	}

	.form td.merged {
		font-weight: 600;
	}

	/* Sel kolom alur tanpa garis horizontal, supaya alur terlihat menyambung. */
	.form td.flow-cell {
		border-top: none;
		border-bottom: none;
		height: 9mm;
	}

	.flow {
		position: absolute;
		pointer-events: none;
		overflow: visible;
	}

	.flow-line {
		stroke: #000000;
		stroke-width: 1.2;
	}

	.shape,
	.connector {
		fill: #ffffff;
		stroke: #000000;
		stroke-width: 1.2;
	}

	.shape-text,
	.connector-text {
		font-size: 10px;
		font-weight: 700;
		text-anchor: middle;
		dominant-baseline: central;
	}

	.flow-label {
		font-size: 9px;
	}

	.flow-label.ng {
		font-weight: 700;
	}

	.page-foot {
		margin: auto 0 0;
		padding-top: 2mm;
		display: flex;
		justify-content: space-between;
		font-size: 0.9em;
	}
</style>
