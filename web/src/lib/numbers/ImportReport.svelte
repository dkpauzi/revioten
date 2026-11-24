<script lang="ts">
	// Laporan impor nomor lama (M2.5, I4), untuk impor tempel dan impor berkas
	// Excel: setiap nomor beserta revisi dan hasilnya, register sesudah impor
	// per kode customer, dan set yang tidak sejajar. Kolom customer dan status
	// hanya tampil untuk impor berkas, yang memuat beberapa customer.
	import type { NumberImportReport } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';
	import { lineResultText, rangesText, revisionLabel, statusLabel, typeNames } from '#lib/numbers.ts';

	let { report, revisionDigits }: { report: NumberImportReport; revisionDigits: number } = $props();

	let fromFile = $derived(report.lines.some((l) => l.customer_code !== undefined));

	function summary(r: NumberImportReport): string {
		if (r.imported === 0) {
			return t('numberImportNothing');
		}
		return t(r.dry_run ? 'numberImportPreviewed' : 'numberImportDone', { count: String(r.imported) });
	}
</script>

<section>
	<p role="status">{summary(report)}</p>

	{#if report.lines.length > 0}
		<h2>{t('numberImportLinesTitle')}</h2>
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t(fromFile ? 'numberImportColumnRow' : 'numberImportColumnLine')}</th>
						{#if fromFile}<th>{t('numberImportColumnCode')}</th>{/if}
						<th>{t('numberImportColumnText')}</th>
						<th>{t('numberImportColumnRevision')}</th>
						{#if fromFile}<th>{t('numberImportStatus')}</th>{/if}
						<th>{t('numberImportColumnResult')}</th>
					</tr>
				</thead>
				<tbody>
					{#each report.lines as l, i (i)}
						<tr data-import={l.result}>
							<td>{l.line}</td>
							{#if fromFile}<td>{l.customer_code ?? ''}</td>{/if}
							<td class="mono">{l.text}{#if l.derived} {t('numberImportDerived')}{/if}</td>
							<td class="mono">{revisionLabel(l.revision, revisionDigits)}</td>
							{#if fromFile}<td>{l.status ? statusLabel(l.status) : ''}</td>{/if}
							<td>{lineResultText(l)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	{#if report.groups.length > 0}
		<h2>{t('numberImportGroupsTitle')}</h2>
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('numberImportColumnCode')}</th>
						<th>{t('numberType')}</th>
						<th>{t('numberImportColumnCount')}</th>
						<th>{t('numberImportColumnHighest')}</th>
						<th>{t('numberImportColumnGaps')}</th>
					</tr>
				</thead>
				<tbody>
					{#each report.groups as g (g.customer_code + g.document_type.id)}
						<tr>
							<td>{g.customer_code}</td>
							<td>{g.document_type.name}</td>
							<td>{g.count}</td>
							<td>{g.highest ?? ''}</td>
							<td>{g.count === 0 ? '' : rangesText(g.gaps) || t('numberImportNoGaps')}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<h2>{t('numberImportMisalignedTitle')}</h2>
		<p class="hint">{t('numberImportMisalignedHint')}</p>
		{#if report.misaligned.length === 0}
			<p>{t('numberImportAligned')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('numberImportColumnCode')}</th>
							<th>{t('numberImportColumnRange')}</th>
							<th>{t('numberImportColumnPresent')}</th>
						</tr>
					</thead>
					<tbody>
						{#each report.misaligned as m (m.customer_code + m.range.first)}
							<tr>
								<td>{m.customer_code}</td>
								<td>{rangesText([m.range])}</td>
								<td>{typeNames(m.present)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</section>

<style>
	.scroll {
		overflow-x: auto;
	}
</style>
