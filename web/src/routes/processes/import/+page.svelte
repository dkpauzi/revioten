<script lang="ts">
	// Import nama proses dari berkas Excel (I2). PPAP Engineer (E-6) mengunduh
	// template, mengisi satu proses per baris (nama dan alias dipisah titik
	// koma), lalu mengunggahnya: Preview melaporkan setiap baris tanpa
	// menyimpan, Import menyimpan proses baru. Proses yang sudah ada dilewati,
	// dan alias yang belum dimilikinya hanya dilaporkan. Server tetap memeriksa
	// haknya.
	import { importProcesses } from '#lib/api/client.ts';
	import type { ProcessImport, ProcessImportRow } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let canImport = $derived(can('E-6'));
	let file = $state<File | null>(null);
	let report = $state<ProcessImport | null>(null);
	let busy = $state(false);
	let failure = $state('');

	async function send(commit: boolean) {
		failure = '';
		if (!file) {
			failure = t('processImportChooseFile');
			return;
		}
		busy = true;
		const result = await importProcesses(file, commit);
		busy = false;
		if (!result.ok) {
			report = null;
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(result.error);
			return;
		}
		report = result.data;
	}

	function preview(event: SubmitEvent) {
		event.preventDefault();
		send(false);
	}

	function summary(r: ProcessImport): string {
		const counts = { new: String(r.new), existing: String(r.existing), invalid: String(r.invalid) };
		return t(r.committed ? 'processImportDoneSummary' : 'processImportPreviewSummary', counts);
	}

	function resultText(row: ProcessImportRow, committed: boolean): string {
		if (row.status === 'invalid') {
			return row.problems.map((p) => problemText(p)).join(' ');
		}
		if (row.status === 'existing') {
			return row.not_added.length > 0
				? t('processImportStatusNotAdded', { aliases: row.not_added.join(', ') })
				: t('processImportStatusExisting');
		}
		return t(committed ? 'processImportStatusCreated' : 'processImportStatusNew');
	}
</script>

<h1>{t('processImportTitle')}</h1>
<p>{t('processImportIntro')}</p>
<p><a href="/processes">{t('processImportBack')}</a></p>

{#if !canImport}
	<p>{t('processImportNoPermission')}</p>
{:else}
	<ol class="steps">
		<li><a href="/api/v1/process-imports/template" download>{t('processImportTemplate')}</a> · {t('processImportTemplateHint')}</li>
		<li>{t('processImportFillHint')}</li>
		<li>{t('processImportUploadHint')}</li>
	</ol>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={preview} novalidate>
		<div class="field">
			<label for="process-import-file">{t('processImportFile')}</label>
			<input
				id="process-import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				onchange={(e) => {
					file = e.currentTarget.files?.[0] ?? null;
					report = null;
				}}
			/>
		</div>
		<div class="row">
			<button type="submit" disabled={busy}>{t('processImportPreview')}</button>
			{#if report && !report.committed && report.new > 0}
				<button type="button" class="primary" disabled={busy} onclick={() => send(true)}>{t('processImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if report}
		<p role="status">{summary(report)}</p>
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('processImportColumnRow')}</th>
						<th>{t('processName')}</th>
						<th>{t('processAliases')}</th>
						<th>{t('processImportColumnResult')}</th>
					</tr>
				</thead>
				<tbody>
					{#each report.rows as row (row.row)}
						<tr data-import={row.status}>
							<td>{row.row}</td>
							<td>{row.name}</td>
							<td>{row.status === 'invalid' ? row.aliases_text : row.aliases.join(', ')}</td>
							<td class:error={row.status === 'invalid'}>{resultText(row, report.committed)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	.steps {
		padding-left: 1.25rem;
	}

	.scroll {
		overflow-x: auto;
	}
</style>
