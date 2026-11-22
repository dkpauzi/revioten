<script lang="ts">
	// Pratinjau lembar (C2, C3, D-122, G11): dokumen yang sedang dirancang tampil
	// per halaman kertas, persis seperti PDF dan Excel-nya. Rencana halamannya
	// dihitung server (internal/printout/plan.go): tinggi baris, pemisah
	// halaman, sel nomor langkah dan proses yang digabung, dan SVG kolom
	// Process Flow (simbol, cabang NG, konektor A, B, C). Layar hanya
	// menggambarnya, jadi pratinjau, PDF, dan Excel tidak pernah berbeda.
	import type { PagePreview } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';

	let { plan }: { plan: PagePreview } = $props();

	// Skala tampilan: halaman dikecilkan supaya muat selebar area pratinjau.
	const pxPerMm = 96 / 25.4;
	let areaWidth = $state(0);
	let scale = $derived(areaWidth > 0 ? Math.min(1, areaWidth / (plan.paper_w_mm * pxPerMm)) : 1);
	let contentH = $derived(plan.paper_h_mm - plan.margins_mm.top - plan.margins_mm.bottom);

	// Judul grup kolom: kolom berurutan dengan judul grup yang sama.
	let groups = $derived.by(() => {
		const out: { label: string; span: number }[] = [];
		for (const c of plan.columns) {
			const last = out[out.length - 1];
			if (last && c.group && last.label === c.group) last.span++;
			else out.push({ label: c.group, span: 1 });
		}
		return out;
	});
	let hasGroups = $derived(plan.columns.some((c) => c.group));
	let marks = $derived(plan.mark_mm > 0);
	// Isian header tiga pasang per baris, seperti cetakan.
	let headerRows = $derived.by(() => {
		const out: { label: string; value: string }[][] = [];
		plan.header.forEach((p, i) => {
			if (i % 3 === 0) out.push([]);
			out[out.length - 1].push(p);
		});
		return out;
	});
</script>

<div class="preview" bind:clientWidth={areaWidth}>
	<p class="hint">{t('previewHint')}</p>
	{#each plan.pages as page, p (p)}
		<div class="page-slot" style="height: {plan.paper_h_mm * pxPerMm * scale + 16}px">
			<div
				class="paper"
				role="group"
				aria-label={t('previewPage', { page: String(p + 1), total: String(plan.pages.length) })}
				style="width: {plan.paper_w_mm}mm; height: {plan.paper_h_mm}mm; padding: {plan.margins_mm.top}mm {plan.margins_mm.right}mm {plan.margins_mm.bottom}mm {plan.margins_mm.left}mm; font-size: {plan.font_size}pt; transform: scale({scale})"
			>
				<div class="sheet" style="height: {contentH}mm">
					<div class="headbox" style="height: {plan.head_mm}mm">
						<div class="top"><p class="form-title">{plan.title}</p><b>{plan.number} {plan.revision}</b></div>
						<table class="head">
							<tbody>
								{#each headerRows as pairs, i (i)}
									<tr>
										{#each pairs as pair (pair.label)}<td class="l">{pair.label}</td><td>{pair.value}</td>{/each}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					{#if page.diagram_svg}
						<!-- PFD sebagai diagram (E-144). SVG dari server: tulisan di dalamnya sudah di-escape (diagram.go). -->
						<div class="legendbox" style="height: {plan.legend_mm ?? 0}mm">{@html plan.legend_svg ?? ''}</div>
						<div class="diagrambox" data-testid="process-flow">{@html page.diagram_svg}</div>
					{:else}
					<div class="tw">
						<table class="form">
							<colgroup>
								{#if marks}<col style="width: {plan.mark_mm}mm" />{/if}
								{#each plan.columns_mm as w, i (i)}<col style="width: {w}mm" />{/each}
							</colgroup>
							<thead>
								<!-- Tinggi judul kolom dikunci seperti rencana, supaya alur tepat di barisnya. -->
								{#if hasGroups}
									<tr>
										{#if marks}<th><div class="h" style="height: {plan.group_row_mm}mm"></div></th>{/if}
										{#each groups as g, i (i)}<th colspan={g.span}><div class="h" style="height: {plan.group_row_mm}mm">{g.label}</div></th>{/each}
									</tr>
								{/if}
								<tr>
									{#if marks}<th><div class="h" style="height: {plan.label_row_mm}mm">Rev</div></th>{/if}
									{#each plan.columns as c, i (i)}<th><div class="h" style="height: {plan.label_row_mm}mm">{c.label}</div></th>{/each}
								</tr>
							</thead>
							<tbody>
								{#each page.rows as row, r (r)}
									{#if row.section}
										<tr class="section" style="height: {row.height_mm}mm">
											<td colspan={plan.columns.length + (marks ? 1 : 0)}><div class="c">{row.section.trim()}</div></td>
										</tr>
									{:else}
										<tr style="height: {row.height_mm}mm">
											{#if marks}<td class="mark"><div class="c">{row.mark}</div></td>{/if}
											{#each row.cells as cell, c (c)}
												{#if !cell.skip}
													<td rowspan={cell.rowspan > 1 ? cell.rowspan : undefined} class:flowcell={cell.flow} class:merged={cell.merged}>
														<div class="c" style="max-height: {cell.height_mm}mm">{cell.text}</div>
													</td>
												{/if}
											{/each}
										</tr>
									{/if}
								{/each}
							</tbody>
						</table>
						{#if page.flow_svg}
							<!-- SVG dari server: angka dan tulisan di dalamnya sudah di-escape (flowsvg.go). -->
							<div class="flowbox" data-testid="process-flow" style="left: {plan.flow_left_mm}mm; top: {plan.group_row_mm + plan.label_row_mm}mm">
								{@html page.flow_svg}
							</div>
						{/if}
					</div>
					{/if}
				</div>
				<p class="page-foot">
					<span>{plan.number} {plan.revision}</span>
					<span>{t('previewPage', { page: String(p + 1), total: String(plan.pages.length) })}</span>
				</p>
			</div>
		</div>
	{/each}
</div>

<style>
	.preview {
		overflow: hidden;
	}

	.page-slot {
		position: relative;
	}

	/* Satu halaman kertas: ukurannya milimeter, lalu dikecilkan dengan transform.
	   Gaya tabel sama dengan halaman PDF (internal/printout/html.go). */
	.paper {
		position: relative;
		box-sizing: border-box;
		background: #ffffff;
		color: #000000;
		font-family: Arial, Helvetica, sans-serif;
		line-height: 1.25;
		box-shadow: 0 2px 10px rgba(16, 24, 40, 0.12);
		transform-origin: top left;
	}

	.sheet {
		position: relative;
		overflow: hidden;
	}

	.headbox {
		overflow: hidden;
	}

	.top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.form-title {
		margin: 0 0 2mm;
		font-weight: 700;
		font-size: 1.8em;
	}

	/* Ukuran huruf mengikuti kertas, bukan gaya tabel aplikasi (app.css). */
	table {
		width: 100%;
		table-layout: fixed;
		border-collapse: collapse;
		margin: 0;
		font-size: inherit;
		line-height: 1.25;
	}

	th,
	td {
		border: 0.5pt solid #000000;
		vertical-align: top;
		overflow-wrap: anywhere;
		white-space: pre-wrap;
		background: #ffffff;
		color: #000000;
		font-size: 1em;
		text-align: left;
		padding: 0;
	}

	.head td {
		padding: 1pt 2pt;
	}

	.head td.l {
		font-weight: 700;
		width: 14%;
	}

	th {
		background: #d9d9d9;
		text-align: center;
		vertical-align: middle;
		font-weight: 700;
		padding: 0;
	}

	th .h {
		box-sizing: border-box;
		overflow: hidden;
		padding: 1pt 2pt;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	td .c {
		box-sizing: border-box;
	}

	td .c {
		overflow: hidden;
		padding: 1pt 2pt;
	}

	td.merged {
		font-weight: 700;
	}

	tbody tr + tr td {
		border-top: 0.5pt solid #000000;
	}

	thead th:first-child,
	thead th:last-child {
		border-radius: 0;
	}

	td.flowcell {
		border-top: none;
		border-bottom: none;
	}

	.section td {
		font-weight: 700;
		background: #f2f2f2;
	}

	.tw {
		position: relative;
	}

	.legendbox :global(svg),
	.diagrambox :global(svg) {
		display: block;
	}

	.flowbox {
		position: absolute;
		pointer-events: none;
	}

	.flowbox :global(svg) {
		display: block;
	}

	.page-foot {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 4mm;
		margin: 0;
		padding: 0 10mm;
		display: flex;
		justify-content: space-between;
		font-size: 7pt;
	}
</style>
