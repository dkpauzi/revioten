<script lang="ts">
	// Laporan kinerja PPAP Engineer (D-197), untuk PPAP Engineer dan
	// supervisornya (L-5). Periode: minggu Jumat sampai Kamis (laporan mingguan
	// disampaikan setiap Jumat), bulan, atau tahun. Semua angka dihitung server
	// dari kejadian yang tercatat; tidak ada angka yang diketik.
	import { onMount } from 'svelte';
	import { getPerformance, listCustomers } from '#lib/api/client.ts';
	import type { Customer, DurationSummary, Performance } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';

	type Kind = 'week' | 'month' | 'year';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let kind = $state<Kind>('week');
	let date = $state(today());
	let report = $state<Performance | null>(null);
	let loadError = $state('');

	function today(): string {
		return iso(new Date());
	}

	function iso(d: Date): string {
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
	}

	// shift memindah tanggal satu periode ke depan atau ke belakang.
	function shift(step: number) {
		const d = new Date(date + 'T12:00:00');
		if (kind === 'week') d.setDate(d.getDate() + 7 * step);
		if (kind === 'month') d.setMonth(d.getMonth() + step, 1);
		if (kind === 'year') d.setFullYear(d.getFullYear() + step, 0, 1);
		date = iso(d);
		load();
	}

	// hoursText menulis lama dalam jam, atau dalam hari bila lebih dari dua hari.
	function hoursText(hours: number): string {
		if (hours >= 48) return t('performanceDays', { days: (Math.round((hours / 24) * 10) / 10).toString() });
		return t('dashboardHours', { hours: (Math.round(hours * 10) / 10).toString() });
	}

	function durationText(d: DurationSummary): string {
		if (d.count === 0) return '—';
		return t('performanceDuration', { average: hoursText(d.average_hours), median: hoursText(d.median_hours), count: String(d.count) });
	}

	function triggerName(name: string, mechanism: string): string {
		if (name !== '') return name;
		return t(`performanceMechanism_${mechanism}` as MessageKey);
	}

	async function load() {
		const result = await getPerformance(kind, date, customerId);
		if (!result.ok) {
			loadError = failureText(result.error);
			report = null;
			return;
		}
		loadError = '';
		report = result.data;
	}

	onMount(async () => {
		const result = await listCustomers();
		customers = result.ok ? result.data.items : [];
		await load();
	});
</script>

<h1>{t('performanceTitle')}</h1>
<p class="hint">{t('performanceHint')}</p>

<div class="filters">
	<div class="field">
		<label for="performance-period">{t('performancePeriod')}</label>
		<select id="performance-period" bind:value={kind} onchange={load}>
			<option value="week">{t('performanceWeek')}</option>
			<option value="month">{t('performanceMonth')}</option>
			<option value="year">{t('performanceYear')}</option>
		</select>
	</div>
	<div class="field">
		<label for="performance-date">{t('performanceDate')}</label>
		<input id="performance-date" type="date" bind:value={date} onchange={load} />
	</div>
	<div class="field">
		<label for="performance-customer">{t('documentCustomer')}</label>
		<select id="performance-customer" bind:value={customerId} onchange={load}>
			<option value="">{t('dashboardAllCustomers')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
	</div>
	<div class="nav">
		<button type="button" onclick={() => shift(-1)}>{t('performancePrevious')}</button>
		<button type="button" onclick={() => shift(1)}>{t('performanceNext')}</button>
	</div>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if report}
	<h2>{t('performanceRange', { from: report.from, to: report.to })}</h2>
	<div class="cards">
		<section class="card" aria-labelledby="perf-changes">
			<h3 id="perf-changes">{t('performanceChanges')}</h3>
			<dl class="facts">
				<dt>{t('performanceSubmitted')}</dt>
				<dd data-testid="perf-submitted">{report.submitted}</dd>
				<dt>{t('performanceAccepted')}</dt>
				<dd data-testid="perf-accepted">{report.accepted}</dd>
				<dt>{t('performanceRejected')}</dt>
				<dd>{report.rejected}</dd>
			</dl>
		</section>

		<section class="card" aria-labelledby="perf-triggers">
			<h3 id="perf-triggers">{t('performanceByTrigger')}</h3>
			{#if report.by_trigger.length === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<dl class="facts">
					{#each report.by_trigger as tc (tc.name + tc.mechanism)}
						<dt>{triggerName(tc.name, tc.mechanism)}</dt>
						<dd>{tc.count}</dd>
					{/each}
				</dl>
			{/if}
		</section>

		<section class="card" aria-labelledby="perf-yokoten">
			<h3 id="perf-yokoten">{t('performanceYokoten')}</h3>
			<dl class="facts">
				<dt>{t('performanceYokotens')}</dt>
				<dd>{report.yokotens}</dd>
				<dt>{t('performanceYokotenTargets')}</dt>
				<dd>{report.yokoten_targets}</dd>
				<dt>{t('performanceYokotenChanges')}</dt>
				<dd>{report.yokoten_changes}</dd>
			</dl>
		</section>

		<section class="card" aria-labelledby="perf-times">
			<h3 id="perf-times">{t('performanceTimes')}</h3>
			<p class="hint">{t('performanceTimesHint', { count: String(report.releases) })}</p>
			<dl class="facts">
				<dt>{t('performanceAcceptedToRelease')}</dt>
				<dd>{durationText(report.accepted_to_release)}</dd>
				<dt>{t('performanceEditToPrint')}</dt>
				<dd>{durationText(report.edit_to_print)}</dd>
				<dt>{t('performancePrintToSigned')}</dt>
				<dd>{durationText(report.print_to_signed)}</dd>
			</dl>
		</section>
	</div>

	<section>
		<h3>{t('performancePeople')}</h3>
		{#if report.people.length === 0}
			<p>{t('dashboardNoData')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('performancePerson')}</th>
							<th>{t('performanceSubmitted')}</th>
							<th>{t('performanceAccepted')}</th>
							<th>{t('performanceYokotens')}</th>
							<th>{t('performanceReleases')}</th>
							<th>{t('performanceAcceptedToRelease')}</th>
							<th>{t('performanceEditToPrint')}</th>
							<th>{t('performancePrintToSigned')}</th>
						</tr>
					</thead>
					<tbody>
						{#each report.people as p (p.person_id + p.name)}
							<tr>
								<td>{p.name}</td>
								<td>{p.submitted}</td>
								<td>{p.accepted}</td>
								<td>{p.yokotens}</td>
								<td>{p.releases}</td>
								<td>{durationText(p.accepted_to_release)}</td>
								<td>{durationText(p.edit_to_print)}</td>
								<td>{durationText(p.print_to_signed)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	{#if report.release_items.length > 0}
		<section>
			<h3>{t('performanceReleaseList')}</h3>
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('numberColumnNumber')}</th>
							<th>{t('releasedRevision')}</th>
							<th>{t('performancePrintedBy')}</th>
							<th>{t('performanceAcceptedToRelease')}</th>
							<th>{t('performanceEditToPrint')}</th>
							<th>{t('performancePrintToSigned')}</th>
						</tr>
					</thead>
					<tbody>
						{#each report.release_items as r (r.revision_id)}
							<tr>
								<td>{r.number}</td>
								<td>{r.revision}</td>
								<td>{r.printed_by ?? '—'}</td>
								<td>{r.accepted_to_release_hours === undefined ? '—' : hoursText(r.accepted_to_release_hours)}</td>
								<td>{r.edit_to_print_hours === undefined ? '—' : hoursText(r.edit_to_print_hours)}</td>
								<td>{r.print_to_signed_hours === undefined ? '—' : hoursText(r.print_to_signed_hours)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	{/if}
{/if}

<style>
	.filters {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 16px;
		margin-bottom: 1.5rem;
	}

	.filters .field {
		min-width: 180px;
	}

	.nav {
		display: flex;
		gap: 8px;
	}

	.cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 16px;
		margin-bottom: 1.5rem;
	}

	.card {
		min-width: 0;
		padding: 20px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.scroll {
		overflow-x: auto;
	}
</style>
