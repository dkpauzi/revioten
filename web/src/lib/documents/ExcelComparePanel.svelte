<script lang="ts">
	// Pembanding Excel (M7.3, R-MIG-04): Excel lama diunggah hanya untuk
	// dibandingkan dengan isi dokumen. Tidak ada yang diimpor atau disimpan.
	import { compareDocumentWithExcel } from '#lib/api/client.ts';
	import type { Document, ExcelComparison } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let { doc }: { doc: Document } = $props();

	let fileInput = $state<HTMLInputElement | null>(null);
	let sheet = $state('');
	let result = $state<ExcelComparison | null>(null);
	let failure = $state('');

	async function compare(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		const file = fileInput?.files?.[0];
		if (!file) {
			failure = t('excelCompareChoose');
			return;
		}
		const response = await compareDocumentWithExcel(doc.id, file, sheet);
		if (!response.ok) {
			const details = response.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(response.error);
			result = null;
			return;
		}
		result = response.data;
		sheet = response.data.sheet;
	}
</script>

<section>
	<h2>{t('excelCompareTitle')}</h2>
	<p class="hint">{t('excelCompareHint')}</p>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form class="row" onsubmit={compare} novalidate>
		<div class="field">
			<label for="excel-compare-file">{t('excelCompareFile')}</label>
			<input id="excel-compare-file" type="file" accept=".xlsx" bind:this={fileInput} onchange={() => ((sheet = ''), (result = null))} />
		</div>
		{#if result}
			<div class="field">
				<label for="excel-compare-sheet">{t('excelCompareSheet')}</label>
				<select id="excel-compare-sheet" bind:value={sheet}>
					{#each result.sheets as name (name)}
						<option value={name}>{name}</option>
					{/each}
				</select>
			</div>
		{/if}
		<button type="submit">{t('excelCompareButton')}</button>
	</form>

	{#if result}
		<p role="status">{t('excelCompareSummary', { sheet: result.sheet, matched: String(result.matched) })}</p>
		{#if result.truncated}<p class="hint">{t('excelCompareTruncated')}</p>{/if}
		<h3>{t('excelCompareMissingExcel')}</h3>
		{#if result.missing_in_excel.length === 0}
			<p>{t('excelCompareNone')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('excelCompareRow')}</th>
							<th>{t('excelCompareColumn')}</th>
							<th>{t('excelCompareText')}</th>
						</tr>
					</thead>
					<tbody>
						{#each result.missing_in_excel as c, i (i)}
							<tr>
								<td>{c.position === 0 ? t('excelCompareHeader') : c.position}</td>
								<td>{c.label}</td>
								<td>{c.text}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
		<h3>{t('excelCompareMissingDocument')}</h3>
		{#if result.missing_in_document.length === 0}
			<p>{t('excelCompareNone')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('excelCompareCell')}</th>
							<th>{t('excelCompareText')}</th>
						</tr>
					</thead>
					<tbody>
						{#each result.missing_in_document as c, i (i)}
							<tr>
								<td class="mono">{c.cell}</td>
								<td>{c.text}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.scroll {
		overflow-x: auto;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
