<script lang="ts">
	// Tabel Action Priority satu versi format template (D-134), dengan
	// kalkulator coba di bawahnya. Untuk versi draft, admin bisa mengubah
	// setiap kotak (form set langkah D); hasilnya dikirim ke onsave sebagai
	// daftar aturan yang selalu menutup 1.000 kombinasi. Penerbitannya oleh
	// penyetuju form (D-139), di baris versi halaman ini.
	import type { ApRule } from '#lib/api/gen/index.ts';
	import { actionPriority, apGrid, apMatrix, bandLabel, rulesFromGrid, type ApGrid } from '#lib/ap.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		rules,
		editable = false,
		onsave
	}: {
		rules: ApRule[];
		editable?: boolean; // versi draft dan akun pemegang A-2
		onsave?: (rules: ApRule[]) => Promise<boolean>; // true bila tersimpan
	} = $props();

	let matrix = $derived(apMatrix(rules));
	let severity = $state(8);
	let occurrence = $state(6);
	let detection = $state(4);
	let result = $derived(actionPriority(rules, severity, occurrence, detection));

	// Salinan yang sedang diubah; null berarti tidak sedang mengubah.
	let grid = $state<ApGrid | null>(null);
	let saving = $state(false);

	const next: Record<ApRule['priority'], ApRule['priority']> = { H: 'M', M: 'L', L: 'H' };

	function cycle(i: number, j: number, k: number) {
		if (grid) {
			grid.cells[i][j][k] = next[grid.cells[i][j][k]];
		}
	}

	async function save() {
		if (!grid || !onsave) {
			return;
		}
		saving = true;
		const ok = await onsave(rulesFromGrid(grid));
		saving = false;
		if (ok) {
			grid = null;
		}
	}
</script>

{#if grid}
	<p class="hint">{t('apEditHint')}</p>
	<table>
		<caption>{t('apTableCaption')}</caption>
		<thead>
			<tr>
				<th>{t('formatScaleSeverity')}</th>
				<th>{t('formatScaleOccurrence')}</th>
				{#each grid.detection as d (d[0])}
					<th>{t('apDetectionBand', { band: bandLabel(d) })}</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each grid.severity as s, i (s[0])}
				{#each grid.occurrence as o, j (o[0])}
					<tr>
						<td>{bandLabel(s)}</td>
						<td>{bandLabel(o)}</td>
						{#each grid.detection as d, k (d[0])}
							<td>
								<button
									type="button"
									class="ap cell"
									data-ap={grid.cells[i][j][k]}
									aria-label={t('apCellLabel', { s: bandLabel(s), o: bandLabel(o), d: bandLabel(d), ap: grid.cells[i][j][k] })}
									onclick={() => cycle(i, j, k)}>{grid.cells[i][j][k]}</button
								>
							</td>
						{/each}
					</tr>
				{/each}
			{/each}
		</tbody>
	</table>
	<div class="actions">
		<button type="button" class="primary" onclick={save} disabled={saving}>{t('apSave')}</button>
		<button type="button" onclick={() => (grid = null)} disabled={saving}>{t('apCancel')}</button>
	</div>
{:else}
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
	{#if editable && onsave}
		<p><button type="button" onclick={() => (grid = apGrid(rules))}>{t('apEdit')}</button></p>
	{/if}
{/if}

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

	/* Kotak yang bisa diklik: ukuran tombol kecil, warna mengikuti AP-nya. */
	.cell {
		min-height: 32px;
		border: 1px solid transparent;
		cursor: pointer;
	}

	.cell:hover {
		border-color: currentColor;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
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
