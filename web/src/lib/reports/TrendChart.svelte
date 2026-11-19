<script lang="ts">
	// Grafik tren laporan kinerja (D-202), SVG tanpa library grafik:
	//   - stacked=false: batang revisi terbit per titik dan garis median hari
	//     "diterima sampai rilis" (minggu dan bulan);
	//   - stacked=true: batang revisi terbit per bulan, bertumpuk per pemicu (tahun).
	// Angka tetap terbaca tanpa warna: label nilai di atas batang dan tabel
	// di bawah grafik untuk pembaca layar.
	import type { PerformanceTrendPoint } from '#lib/api/gen/index.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';

	let { points, stacked = false, categories, colors }: { points: PerformanceTrendPoint[]; stacked?: boolean; categories: string[]; colors: Record<string, string> } = $props();

	const width = 640;
	const height = 240;
	const pad = { left: 32, right: 40, top: 20, bottom: 28 };
	let plotW = $derived(width - pad.left - pad.right);
	let plotH = height - pad.top - pad.bottom;
	let maxCount = $derived(Math.max(4, ...points.map((p) => p.releases)));
	let maxDays = $derived(Math.max(1, ...points.map((p) => p.accepted_to_release_median / 24)));
	let step = $derived(points.length > 0 ? plotW / points.length : plotW);
	let barW = $derived(Math.min(56, step * 0.6));

	function y(count: number): number {
		return pad.top + plotH - (count / maxCount) * plotH;
	}

	function yDays(days: number): number {
		return pad.top + plotH - (days / maxDays) * plotH;
	}

	function x(i: number): number {
		return pad.left + step * i + step / 2;
	}

	let line = $derived(
		points
			.map((p, i) => (p.releases > 0 ? `${x(i)},${yDays(p.accepted_to_release_median / 24)}` : ''))
			.filter((s) => s !== '')
			.join(' ')
	);

	function short(label: string): string {
		// 2026-03 menjadi Mar, W47 tetap.
		const m = /^\d{4}-(\d{2})$/.exec(label);
		if (!m) return label;
		return t(`monthShort${Number(m[1])}` as MessageKey);
	}
</script>

<svg viewBox="0 0 {width} {height}" role="img" aria-label={t('performanceTrendLabel')} class="chart">
	{#each [0, 0.5, 1] as f (f)}
		<line x1={pad.left} x2={width - pad.right} y1={pad.top + plotH * (1 - f)} y2={pad.top + plotH * (1 - f)} class="grid" />
		<text x={pad.left - 6} y={pad.top + plotH * (1 - f) + 4} text-anchor="end" class="axis">{Math.round(maxCount * f)}</text>
	{/each}
	{#each points as p, i (p.label)}
		{#if stacked}
			{@const parts = categories.map((c) => ({ c, n: p.by_category[c] ?? 0 }))}
			{#each parts as part, j (part.c)}
				{@const below = parts.slice(0, j).reduce((a, b) => a + b.n, 0)}
				{#if part.n > 0}
					<rect x={x(i) - barW / 2} width={barW} y={y(below + part.n)} height={y(below) - y(below + part.n)} fill={colors[part.c]} />
				{/if}
			{/each}
		{:else}
			<rect x={x(i) - barW / 2} width={barW} y={y(p.releases)} height={pad.top + plotH - y(p.releases)} class:current={i === points.length - 1} class="bar" />
		{/if}
		{#if p.releases > 0}
			<text x={x(i)} y={y(p.releases) - 5} text-anchor="middle" class="value">{p.releases}</text>
		{/if}
		<text x={x(i)} y={height - 8} text-anchor="middle" class="axis" class:strong={i === points.length - 1 && !stacked}>{short(p.label)}</text>
	{/each}
	{#if !stacked && line}
		<polyline points={line} class="line" />
		{#each points as p, i (p.label)}
			{#if p.releases > 0}
				<circle cx={x(i)} cy={yDays(p.accepted_to_release_median / 24)} r="4" class="dot" />
			{/if}
		{/each}
	{/if}
</svg>

<table class="visually-hidden">
	<caption>{t('performanceTrendLabel')}</caption>
	<thead><tr><th>{t('performancePeriod')}</th><th>{t('performanceReleased')}</th><th>{t('performanceApprovedToReleased')}</th></tr></thead>
	<tbody>
		{#each points as p (p.label)}
			<tr><td>{p.label}</td><td>{p.releases}</td><td>{(p.accepted_to_release_median / 24).toFixed(1)}</td></tr>
		{/each}
	</tbody>
</table>

<style>
	.chart {
		width: 100%;
		height: auto;
		display: block;
	}

	.grid {
		stroke: var(--border);
	}

	.axis {
		font-size: 11px;
		fill: var(--text-3);
	}

	.axis.strong {
		font-weight: 700;
		fill: var(--ink);
	}

	.value {
		font-size: 11px;
		font-weight: 600;
		fill: var(--text-2);
	}

	.bar {
		fill: #c7d2fe;
	}

	.bar.current {
		fill: var(--accent);
	}

	.line {
		fill: none;
		stroke: #d97706;
		stroke-width: 2.5;
	}

	.dot {
		fill: #d97706;
	}
</style>
