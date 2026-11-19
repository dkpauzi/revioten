<script lang="ts">
	// Laporan kinerja PPAP Engineer (D-197, D-201, D-202), untuk PPAP Engineer,
	// supervisornya, dan manager (L-5). Minggu laporan berjalan Jumat sampai
	// Kamis, jadi laporan mingguan siap setiap Jumat pagi. Semua angka dihitung
	// server dari kejadian yang tercatat; tidak ada target (D-201).
	import { onMount } from 'svelte';
	import { getPerformance, listCustomers, listReviewSets, performanceExportUrl, type PerformanceQuery } from '#lib/api/client.ts';
	import type { Customer, DurationSummary, Performance } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { language } from '#lib/i18n/language.svelte.ts';
	import TrendChart from '#lib/reports/TrendChart.svelte';
	import { delta, formatDay, hoursParts, isoDate, shares } from '#lib/reports.ts';

	type Kind = 'week' | 'month' | 'year';

	const categories = ['ecr', 'customer_claim', 'audit_finding', 'yokoten', 'periodic_review', 'other'];
	const colors: Record<string, string> = {
		ecr: '#4338ca',
		customer_claim: '#dc2626',
		audit_finding: '#f59e0b',
		yokoten: '#0ea5e9',
		periodic_review: '#16a34a',
		other: '#98a2b3'
	};
	const stageColors = ['#4338ca', '#a5b4fc', '#f59e0b', '#16a34a'];

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let personId = $state('');
	let kind = $state<Kind>('week');
	let date = $state(isoDate(new Date()));
	let report = $state<Performance | null>(null);
	let engineers = $state<{ id: string; name: string }[]>([]);
	let overdue = $state<number | null>(null);
	let loadError = $state('');

	let query = $derived<PerformanceQuery>({ period: kind, date, customerId, personId });
	let closed = $derived(report !== null && report.to < isoDate(new Date()));

	function day(iso: string): string {
		return formatDay(iso, language.current);
	}

	// weekday memberi nama hari pendek tanggal YYYY-MM-DD.
	function weekday(iso: string): string {
		const d = new Date(iso + 'T12:00:00');
		return t(`weekdayShort${d.getDay()}` as MessageKey);
	}

	function nextDay(iso: string): string {
		const d = new Date(iso + 'T12:00:00');
		d.setDate(d.getDate() + 1);
		return isoDate(d);
	}

	function amount(hours: number): { value: string; unit: string } {
		const p = hoursParts(hours);
		return { value: p.value, unit: t(p.unit === 'hours' ? 'performanceUnitHours' : 'performanceUnitDays') };
	}

	function durationText(d: DurationSummary): string {
		if (d.count === 0) return '—';
		const a = amount(d.median_hours);
		return `${a.value} ${a.unit}`;
	}

	function rangeTitle(r: Performance): string {
		if (r.period === 'week') {
			return t('performanceWeekTitle', { week: String(r.week), from: `${weekday(r.from)} ${day(r.from)}`, to: `${weekday(r.to)} ${day(r.to)}` });
		}
		if (r.period === 'month') return t('performanceMonthTitle', { from: day(r.from), to: day(r.to) });
		return t('performanceYearTitle', { year: r.from.slice(0, 4), from: day(r.from), to: day(r.to) });
	}

	// shift memindah tanggal satu periode ke depan atau ke belakang.
	function shift(step: number) {
		const d = new Date(date + 'T12:00:00');
		if (kind === 'week') d.setDate(d.getDate() + 7 * step);
		if (kind === 'month') d.setMonth(d.getMonth() + step, 1);
		if (kind === 'year') d.setFullYear(d.getFullYear() + step, 0, 1);
		date = isoDate(d);
		load();
	}

	function choose(k: Kind) {
		kind = k;
		load();
	}

	async function load() {
		const result = await getPerformance(query);
		if (!result.ok) {
			loadError = failureText(result.error);
			report = null;
			return;
		}
		loadError = '';
		report = result.data;
		// Daftar engineer diambil dari laporan tanpa saringan orang.
		if (personId === '') {
			engineers = result.data.people.filter((p) => p.person_id).map((p) => ({ id: p.person_id!, name: p.name }));
		}
	}

	let stageValues = $derived(
		report
			? [report.stages.edit_to_ready, report.stages.waiting_to_print, report.stages.print_to_signed, report.stages.signed_to_released]
			: []
	);
	let stageShares = $derived(shares(stageValues.map((s) => s.median_hours)));
	let maxCategory = $derived(report ? Math.max(1, ...categories.map((c) => report!.by_category[c] ?? 0)) : 1);
	let totalCategory = $derived(report ? categories.reduce((a, c) => a + (report!.by_category[c] ?? 0), 0) : 0);

	onMount(async () => {
		const [c, sets] = await Promise.all([listCustomers(), listReviewSets({})]);
		customers = c.ok ? c.data.items : [];
		overdue = sets.ok ? sets.data.overdue : null;
		await load();
	});
</script>

<div class="head">
	<div>
		<h1>{t('performanceTitle')}</h1>
		<p class="hint">{t('performanceHint')}</p>
	</div>
	<div class="segmented" role="group" aria-label={t('performancePeriod')}>
		<button type="button" aria-pressed={kind === 'week'} onclick={() => choose('week')}>{t('performanceWeek')}</button>
		<button type="button" aria-pressed={kind === 'month'} onclick={() => choose('month')}>{t('performanceMonth')}</button>
		<button type="button" aria-pressed={kind === 'year'} onclick={() => choose('year')}>{t('performanceYear')}</button>
	</div>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if report}
	<div class="card bar">
		<button type="button" class="nav" aria-label={t('performancePrevious')} onclick={() => shift(-1)}>‹</button>
		<div class="range">
			<h2>{rangeTitle(report)}</h2>
			{#if report.period === 'week'}<p class="hint">{t('performanceWeekHint')}</p>{/if}
		</div>
		<button type="button" class="nav" aria-label={t('performanceNext')} onclick={() => shift(1)}>›</button>
		<label class="visually-hidden" for="performance-engineer">{t('reviewEngineer')}</label>
		<select id="performance-engineer" bind:value={personId} onchange={load}>
			<option value="">{t('performanceAllEngineers')}</option>
			{#each engineers as e (e.id)}<option value={e.id}>{e.name}</option>{/each}
		</select>
		<label class="visually-hidden" for="performance-customer">{t('documentCustomer')}</label>
		<select id="performance-customer" bind:value={customerId} onchange={load}>
			<option value="">{t('dashboardAllCustomers')}</option>
			{#each customers as c (c.id)}<option value={c.id}>{c.code} · {c.name}</option>{/each}
		</select>
		{#if report.period === 'week'}
			<span class="badge" data-status={closed ? 'released' : 'draft'}>
				{closed ? t('performanceWeekClosed', { date: `${weekday(nextDay(report.to))} ${day(nextDay(report.to))}` }) : t('performanceWeekOpen', { date: `${weekday(report.to)} ${day(report.to)}` })}
			</span>
		{/if}
		<a class="button primary" href={performanceExportUrl(query)} download>{t(`performanceDownload_${report.period}` as MessageKey)}</a>
	</div>

	{@const prevLabel = report.period === 'week' ? t('performanceVsWeek', { week: String(report.week - 1 > 0 ? report.week - 1 : 52) }) : report.period === 'year' ? t('performanceVsYear', { year: String(Number(report.from.slice(0, 4)) - 1) }) : t('performanceVsMonth')}
	{@const reqDelta = delta(report.submitted, report.previous.submitted)}
	{@const relDelta = delta(report.releases, report.previous.releases)}
	{@const timeDelta = delta(report.stages.accepted_to_release.median_hours / 24, report.previous.accepted_to_release_median / 24)}
	<div class="stats">
		<section class="stat" aria-labelledby="perf-received">
			<h3 id="perf-received">{t('performanceReceived')}</h3>
			<p class="big" data-testid="perf-submitted">{report.submitted}</p>
			{#if reqDelta.direction !== 'same'}<p class="delta" data-direction={reqDelta.direction}>{reqDelta.direction === 'up' ? '▲' : '▼'} {reqDelta.diff} {prevLabel}</p>{/if}
			<p class="sub">{t('performanceReceivedHint', { accepted: String(report.accepted), waiting: String(report.waiting) })}</p>
		</section>
		<section class="stat" aria-labelledby="perf-released">
			<h3 id="perf-released">{t('performanceReleased')}</h3>
			<p class="big" data-testid="perf-released">{report.releases}</p>
			{#if relDelta.direction !== 'same'}<p class="delta" data-direction={relDelta.direction}>{relDelta.direction === 'up' ? '▲' : '▼'} {relDelta.diff} {prevLabel}</p>{/if}
			<p class="sub">{t('performanceOpenAtEnd', { count: String(report.open_at_end) })}</p>
		</section>
		<section class="stat" aria-labelledby="perf-approved">
			<h3 id="perf-approved">{t('performanceApprovedToReleased')}</h3>
			{#if report.stages.accepted_to_release.count > 0}
				{@const a = amount(report.stages.accepted_to_release.median_hours)}
				<p class="big">{a.value} <span class="unit">{a.unit}</span></p>
				{#if report.previous.accepted_to_release_median > 0 && timeDelta.direction !== 'same'}
					<p class="delta" data-direction={timeDelta.direction === 'down' ? 'good' : 'bad'}>
						{timeDelta.direction === 'down' ? '▼' : '▲'} {t(timeDelta.direction === 'down' ? 'performanceFaster' : 'performanceSlower', { days: String(timeDelta.diff) })}
					</p>
				{/if}
			{:else}
				<p class="big">—</p>
			{/if}
			<p class="sub">{t('performanceMedian')}</p>
		</section>
		<section class="stat" aria-labelledby="perf-review">
			<h3 id="perf-review">{t('performanceOpenReview')}</h3>
			<p class="big"><span class="danger">{overdue ?? '—'}</span> <span class="unit">{t('reviewOverdueBadge')}</span></p>
			<p class="sub">{t('reviewTitle')}</p>
			<a href="/periodic-reviews">{t('reviewBannerLink')}</a>
		</section>
	</div>

	{#if report.period !== 'year'}
		<section class="card" aria-labelledby="perf-where">
			<div class="row-head">
				<h3 id="perf-where">{t('performanceWhere')}</h3>
				<span class="sub">{t('performanceMedianPerRevision')}</span>
			</div>
			{#if report.releases === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<div class="stack" aria-hidden="true">
					{#each stageShares as share, i (i)}
						{#if share > 0}<span style="width: {share}%; background: {stageColors[i]}"></span>{/if}
					{/each}
				</div>
				<div class="stages">
					{#each ['performanceStageEdit', 'performanceStageWaiting', 'performanceStagePrint', 'performanceStageRelease'] as key, i (key)}
						<div class="stage" style="border-top-color: {stageColors[i]}">
							<h4>{t(key as MessageKey)}</h4>
							{#if stageValues[i].count > 0}
								{@const a = amount(stageValues[i].median_hours)}
								<p class="mid">{a.value} <span class="unit">{a.unit}</span></p>
							{:else}
								<p class="mid">—</p>
							{/if}
							<p class="sub">{t(`${key}Hint` as MessageKey)}</p>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	<div class="pair">
		<section class="card" aria-labelledby="perf-why">
			<h3 id="perf-why">{t(report.period === 'year' ? 'performanceWhyYear' : 'performanceWhy', { year: report.from.slice(0, 4) })}</h3>
			<dl class="bars">
				{#each categories as c (c)}
					{@const n = report.by_category[c] ?? 0}
					<dt>{t(`performanceCategory_${c}` as MessageKey)}</dt>
					<dd>
						<span class="track"><span class="fill" style="width: {(n / maxCategory) * 100}%; background: {colors[c]}"></span></span>
						<span class="count">{n}</span>
					</dd>
				{/each}
			</dl>
			{#if report.period === 'year'}
				<p class="total"><strong>{t('performanceTotalReleased')}</strong> <strong>{totalCategory}</strong></p>
			{/if}
			<p class="sub">{t('performanceWhyHint')}</p>
		</section>
		<section class="card" aria-labelledby="perf-trend">
			<div class="row-head">
				<h3 id="perf-trend">{t(report.period === 'year' ? 'performanceTrendYear' : report.period === 'month' ? 'performanceTrendMonth' : 'performanceTrendWeek')}</h3>
				{#if report.period !== 'year'}
					<span class="legend"><span class="key" style="background: var(--accent)"></span>{t('performanceReleased')} <span class="key line"></span>{t('performanceApprovedDays')}</span>
				{/if}
			</div>
			{#if report.period === 'year'}
				<p class="legend wrap">
					{#each categories as c (c)}<span><span class="key" style="background: {colors[c]}"></span>{t(`performanceCategory_${c}` as MessageKey)}</span>{/each}
				</p>
			{/if}
			<TrendChart points={report.trend} stacked={report.period === 'year'} {categories} {colors} />
		</section>
	</div>

	{#if report.period === 'year' || report.yokoten_items.length > 0}
		<section class="card flush" aria-labelledby="perf-yokoten">
			<div class="row-head pad">
				<h3 id="perf-yokoten">{t('performanceYokotenTitle', { count: String(report.yokotens), revised: String(report.yokoten_items.reduce((a, y) => a + y.applied, 0)) })}</h3>
				<a class="button" href="/yokoten">{t('performanceOpenYokoten')}</a>
			</div>
			{#if report.yokoten_items.length === 0}
				<p class="pad">{t('dashboardNoData')}</p>
			{:else}
				<div class="scroll">
					<table>
						<thead>
							<tr>
								<th>{t('performanceYokotenItem')}</th>
								<th>{t('performanceStarted')}</th>
								<th class="num">{t('performanceYokotenTargets')}</th>
								<th class="num">{t('performanceRevised')}</th>
								<th class="num">{t('performanceNotApplicable')}</th>
								<th class="num">{t('performanceOpen')}</th>
								<th>{t('performanceCoverage')}</th>
								<th class="num">{t('performanceDaysToClose')}</th>
							</tr>
						</thead>
						<tbody>
							{#each report.yokoten_items as y (y.ref)}
								{@const decided = y.targets > 0 ? ((y.targets - y.pending) / y.targets) * 100 : 0}
								<tr>
									<td><span class="mono">{y.ref}</span> {y.reason}</td>
									<td>{day(y.started)}</td>
									<td class="num">{y.targets}</td>
									<td class="num">{y.applied}</td>
									<td class="num">{y.not_applicable}</td>
									<td class="num">{y.pending}</td>
									<td><span class="track wide"><span class="fill" style="width: {decided}%; background: {decided >= 100 ? '#16a34a' : decided >= 60 ? '#f59e0b' : '#dc2626'}"></span></span></td>
									<td class="num">{y.days_to_close ?? t('performanceStillOpen')}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}

	<section class="card flush" aria-labelledby="perf-people">
		<div class="row-head pad">
			<h3 id="perf-people">{t('performancePeopleTitle')}</h3>
			<span class="sub">{t('performanceTimesMedian')}</span>
		</div>
		{#if report.people.length === 0}
			<p class="pad">{t('dashboardNoData')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('performancePerson')}</th>
							<th class="num">{t('performanceReceived')}</th>
							<th class="num">{t('performanceReleasedShort')}</th>
							<th class="num">{t('performanceOpen')}</th>
							<th class="num">{t('performanceApprovedToReleased')}</th>
							<th class="num">{t('performanceStageEdit')}</th>
							<th class="num">{t('performanceStagePrint')}</th>
							<th class="num">{t('performanceClaimAuditYokoten')}</th>
						</tr>
					</thead>
					<tbody>
						{#each report.people as p (p.person_id + p.name)}
							<tr>
								<td><strong>{p.name}</strong></td>
								<td class="num">{p.submitted}</td>
								<td class="num">{p.releases}</td>
								<td class="num">{p.open}</td>
								<td class="num">{durationText(p.stages.accepted_to_release)}</td>
								<td class="num">{durationText(p.stages.edit_to_ready)}</td>
								<td class="num">{durationText(p.stages.print_to_signed)}</td>
								<td class="num">{p.customer_claim} · {p.audit_finding} · {p.yokoten}</td>
							</tr>
						{/each}
						<tr class="team">
							<td>{t('performanceTeam')}</td>
							<td class="num">{report.submitted}</td>
							<td class="num">{report.releases}</td>
							<td class="num">{report.open_at_end}</td>
							<td class="num">{durationText(report.stages.accepted_to_release)}</td>
							<td class="num">{durationText(report.stages.edit_to_ready)}</td>
							<td class="num">{durationText(report.stages.print_to_signed)}</td>
							<td class="num">{report.people.reduce((a, p) => a + p.customer_claim, 0)} · {report.people.reduce((a, p) => a + p.audit_finding, 0)} · {report.people.reduce((a, p) => a + p.yokoten, 0)}</td>
						</tr>
					</tbody>
				</table>
			</div>
		{/if}
		<p class="foot">{t('performanceFoot')}</p>
	</section>
{/if}

<style>
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-start;
		gap: 16px;
	}

	.card {
		margin-bottom: 16px;
		padding: 20px 24px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		min-width: 0;
	}

	.card.flush {
		padding: 0;
		overflow: hidden;
	}

	.pad {
		padding: 16px 24px;
		margin: 0;
	}

	.bar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
		margin-top: 1rem;
	}

	.bar select {
		width: auto;
		min-width: 170px;
	}

	.range {
		flex: 1 1 280px;
	}

	.range h2 {
		margin: 0;
		font-size: 20px;
	}

	.range p {
		margin: 4px 0 0;
	}

	.nav {
		width: 44px;
		padding: 0;
		font-size: 20px;
	}

	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
		gap: 16px;
		margin-bottom: 16px;
	}

	.stat {
		padding: 18px 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.stat h3 {
		margin: 0;
		font-size: 16px;
	}

	.stat p {
		margin: 2px 0;
	}

	.big {
		font-size: 34px;
		font-weight: 600;
		color: var(--ink);
	}

	.mid {
		font-size: 26px;
		font-weight: 600;
		color: var(--ink);
	}

	.unit {
		font-size: 16px;
		font-weight: 400;
		color: var(--text-3);
	}

	.danger {
		color: var(--danger);
	}

	.sub {
		color: var(--text-3);
		font-size: 14px;
	}

	.delta {
		font-size: 14px;
	}

	.delta[data-direction='up'],
	.delta[data-direction='good'] {
		color: var(--success);
	}

	.delta[data-direction='down'],
	.delta[data-direction='bad'] {
		color: var(--danger);
	}

	.row-head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: 8px;
	}

	.row-head h3 {
		margin: 0;
	}

	.stack {
		display: flex;
		height: 22px;
		margin: 18px 0;
		gap: 2px;
	}

	.stack span {
		display: block;
		height: 100%;
	}

	.stages {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 14px;
	}

	.stage {
		padding: 14px 18px;
		background: var(--surface-2);
		border-radius: var(--radius-lg);
		border-top: 4px solid transparent;
	}

	.stage h4 {
		margin: 0 0 6px;
		font-size: 15px;
	}

	.stage p {
		margin: 0;
	}

	.pair {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
		gap: 16px;
	}

	.bars {
		display: grid;
		grid-template-columns: minmax(120px, auto) 1fr;
		gap: 10px 16px;
		margin: 14px 0;
	}

	.bars dt {
		margin: 0;
	}

	.bars dd {
		display: flex;
		align-items: center;
		gap: 12px;
		margin: 0;
	}

	.track {
		flex: 1;
		height: 10px;
		background: var(--border-soft);
		border-radius: 999px;
		overflow: hidden;
		display: block;
	}

	.track.wide {
		min-width: 140px;
	}

	.fill {
		display: block;
		height: 100%;
		border-radius: 999px;
	}

	.count {
		min-width: 2ch;
		text-align: right;
		font-weight: 600;
	}

	.total {
		display: flex;
		justify-content: space-between;
		padding-top: 10px;
		border-top: 1px solid var(--border);
	}

	.legend {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		color: var(--text-3);
		font-size: 14px;
	}

	.legend.wrap {
		flex-wrap: wrap;
		margin: 8px 0;
	}

	.legend.wrap > span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		margin-right: 10px;
	}

	.key {
		display: inline-block;
		width: 12px;
		height: 12px;
		border-radius: 3px;
	}

	.key.line {
		height: 3px;
		background: #d97706;
	}

	.scroll {
		overflow-x: auto;
	}

	.scroll table {
		width: 100%;
	}

	.num {
		text-align: right;
		white-space: nowrap;
	}

	.team td {
		font-weight: 700;
		background: var(--surface-2);
	}

	.foot {
		margin: 0;
		padding: 12px 24px;
		color: var(--text-3);
		font-size: 14px;
		border-top: 1px solid var(--border);
	}
</style>
