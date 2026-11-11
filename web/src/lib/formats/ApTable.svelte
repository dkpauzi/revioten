<script lang="ts">
	// Tabel Action Priority satu versi format template (form set langkah A,
	// D-134), hanya untuk dibaca, dengan kalkulator coba di bawahnya. Tabel
	// diganti lewat API (PUT .../ap-rules) pada versi draft; layar untuk
	// mengubahnya dibangun bersama form set (langkah D).
	import type { ApRule } from '#lib/api/gen/index.ts';
	import { actionPriority, apMatrix } from '#lib/ap.ts';
	import { t } from '#lib/i18n/index.ts';

	let { rules }: { rules: ApRule[] } = $props();

	let matrix = $derived(apMatrix(rules));
	let severity = $state(8);
	let occurrence = $state(6);
	let detection = $state(4);
	let result = $derived(actionPriority(rules, severity, occurrence, detection));
</script>

<table>
	<caption>{t('apTableCaption')}</caption>
	<thead>
		<tr>
			<th>{t('formatScaleSeverity')}</th>
			<th>{t('formatScaleOccurrence')}</th>
			{#each matrix.detection as d (d)}
				<th>{t('apDetectionBand', { band: d })}</th>
			{/each}
		</tr>
	</thead>
	<tbody>
		{#each matrix.rows as row (row.severity + '|' + row.occurrence)}
			<tr>
				<td>{row.severity}</td>
				<td>{row.occurrence}</td>
				{#each row.cells as cell, i (i)}
					<td><span class="ap" data-ap={cell}>{cell}</span></td>
				{/each}
			</tr>
		{/each}
	</tbody>
</table>

<h3>{t('apTryTitle')}</h3>
<div class="try">
	<div class="field">
		<label for="ap-severity">{t('formatScaleSeverity')}</label>
		<input id="ap-severity" type="number" min="1" max="10" bind:value={severity} />
	</div>
	<div class="field">
		<label for="ap-occurrence">{t('formatScaleOccurrence')}</label>
		<input id="ap-occurrence" type="number" min="1" max="10" bind:value={occurrence} />
	</div>
	<div class="field">
		<label for="ap-detection">{t('formatScaleDetection')}</label>
		<input id="ap-detection" type="number" min="1" max="10" bind:value={detection} />
	</div>
	<p class="result" aria-live="polite">
		{#if result}
			{t('apTryResult')} <span class="ap" data-ap={result}>{result}</span>
		{:else}
			{t('apTryOutOfRange')}
		{/if}
	</p>
</div>

<style>
	/* Warna AP hanya di layar; cetakan nanti menampilkan hurufnya saja (D-134). */
	.ap {
		display: inline-block;
		min-width: 28px;
		padding: 2px 8px;
		border-radius: 6px;
		text-align: center;
		font-weight: 600;
	}

	.ap[data-ap='H'] {
		background: var(--danger-weak);
		color: var(--danger);
	}

	.ap[data-ap='M'] {
		background: var(--warning-weak);
		color: var(--warning);
	}

	.ap[data-ap='L'] {
		background: var(--success-weak);
		color: var(--success);
	}

	.try {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.try input {
		width: 6rem;
	}

	.result {
		margin: calc(var(--label-space) + 8px) 0 0;
	}
</style>
