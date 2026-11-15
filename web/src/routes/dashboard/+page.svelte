<script lang="ts">
	// Dasbor ukuran (M7.7, UC-64): persentase baris yang mengikuti template,
	// dokumen per status, status yokoten, waktu revisi, dan usaha migrasi.
	// Semua angka dihitung server dari data saat halaman dibuka; tidak ada
	// angka yang diketik (brief bagian 15, butir 5).
	import { onMount } from 'svelte';
	import { getMetrics, listCustomers } from '#lib/api/client.ts';
	import type { Customer, Metrics } from '#lib/api/gen/index.ts';
	import ReviewSummary from '#lib/ReviewSummary.svelte';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let metrics = $state<Metrics | null>(null);
	let loadError = $state('');

	// Satu angka di belakang koma cukup untuk dibaca.
	function one(value: number): string {
		return (Math.round(value * 10) / 10).toString();
	}

	function statusLabel(status: string): string {
		if (status === 'released_migrated') return t('documentStatusReleasedMigrated');
		return t(status === 'released' ? 'documentStatusReleased' : 'versionStatusDraft');
	}

	async function load() {
		const result = await getMetrics(customerId);
		if (!result.ok) {
			loadError = failureText(result.error);
			metrics = null;
			return;
		}
		loadError = '';
		metrics = result.data;
	}

	onMount(async () => {
		const result = await listCustomers();
		customers = result.ok ? result.data.items : [];
		await load();
	});
</script>

<h1>{t('dashboardTitle')}</h1>
<p class="hint">{t('dashboardHint')}</p>

<div class="field chooser">
	<label for="dashboard-customer">{t('documentCustomer')}</label>
	<select id="dashboard-customer" bind:value={customerId} onchange={load}>
		<option value="">{t('dashboardAllCustomers')}</option>
		{#each customers as c (c.id)}
			<option value={c.id}>{c.code} · {c.name}</option>
		{/each}
	</select>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

<ReviewSummary {customerId} />

{#if metrics}
	<div class="cards">
		<section class="card" aria-labelledby="dash-rows">
			<h2 id="dash-rows">{t('dashboardRows')}</h2>
			{#if metrics.rows.total === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<p class="big">{one(metrics.rows.linked_percent)}%</p>
				<dl class="facts">
					<dt>{t('dashboardRowsLinked')}</dt>
					<dd>{metrics.rows.linked}</dd>
					<dt>{t('dashboardRowsDetached')}</dt>
					<dd>{metrics.rows.detached}</dd>
					<dt>{t('dashboardRowsFree')}</dt>
					<dd>{metrics.rows.free}</dd>
					<dt>{t('dashboardRowsTotal')}</dt>
					<dd>{metrics.rows.total}</dd>
				</dl>
			{/if}
		</section>

		<section class="card" aria-labelledby="dash-documents">
			<h2 id="dash-documents">{t('dashboardDocuments')}</h2>
			{#if metrics.documents.length === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<table>
					<thead>
						<tr>
							<th>{t('numberColumnType')}</th>
							<th>{t('numberColumnStatus')}</th>
							<th>{t('dashboardCount')}</th>
						</tr>
					</thead>
					<tbody>
						{#each metrics.documents as d (d.document_type + d.status)}
							<tr>
								<td>{d.document_type_name}</td>
								<td>{statusLabel(d.status)}</td>
								<td>{d.count}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</section>

		<section class="card" aria-labelledby="dash-yokoten">
			<h2 id="dash-yokoten">{t('dashboardYokoten')}</h2>
			{#if metrics.yokoten.length === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<dl class="facts">
					{#each metrics.yokoten as y (y.decision)}
						<dt>{t(`dashboardYokoten_${y.decision}`)}</dt>
						<dd>{y.count}</dd>
					{/each}
				</dl>
			{/if}
		</section>

		<section class="card" aria-labelledby="dash-revision">
			<h2 id="dash-revision">{t('dashboardRevision')}</h2>
			<p class="hint">{t('dashboardRevisionHint')}</p>
			{#if metrics.revision.count === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<dl class="facts">
					<dt>{t('dashboardRevisionCount')}</dt>
					<dd>{metrics.revision.count}</dd>
					<dt>{t('dashboardRevisionAverage')}</dt>
					<dd>{t('dashboardHours', { hours: one(metrics.revision.average_hours) })}</dd>
					<dt>{t('dashboardRevisionMedian')}</dt>
					<dd>{t('dashboardHours', { hours: one(metrics.revision.median_hours) })}</dd>
				</dl>
			{/if}
		</section>

		<section class="card" aria-labelledby="dash-migration">
			<h2 id="dash-migration">{t('dashboardMigration')}</h2>
			{#if metrics.migration.documents === 0}
				<p>{t('dashboardNoData')}</p>
			{:else}
				<dl class="facts">
					<dt>{t('dashboardMigrationDocuments')}</dt>
					<dd>{metrics.migration.documents}</dd>
					<dt>{t('dashboardMigrationVerified')}</dt>
					<dd>{metrics.migration.verified}</dd>
					<dt>{t('dashboardMigrationMeasured')}</dt>
					<dd>{metrics.migration.measured}</dd>
					{#if metrics.migration.measured > 0}
						<dt>{t('dashboardMigrationAverage')}</dt>
						<dd>{t('dashboardMinutes', { minutes: one(metrics.migration.average_minutes) })}</dd>
					{/if}
				</dl>
			{/if}
		</section>
	</div>
{/if}

<style>
	.chooser {
		max-width: 28rem;
		margin-bottom: 1.5rem;
	}

	.cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 16px;
	}

	.card {
		min-width: 0;
		padding: 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		overflow-x: auto;
	}

	.big {
		font-size: 2.5rem;
		font-weight: 600;
		margin: 0.25rem 0 0.75rem;
	}
</style>
