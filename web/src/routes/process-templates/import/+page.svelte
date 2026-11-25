<script lang="ts">
	// Import isi process template dari berkas Excel (I3). PPAP Engineer (E-6)
	// mengunduh template, mengisi satu poin per baris di sheet PFMEA dan Control
	// Plan, lalu mengunggahnya: Preview melaporkan setiap baris dan versi yang
	// akan menerima poin tanpa menyimpan, Import menyimpan poin baru ke versi
	// draft. Versi terbit tidak pernah diubah, dan penerbitan tetap di halaman
	// Process templates. Server tetap memeriksa haknya.
	import { importProcessContent } from '#lib/api/client.ts';
	import type { ProcessContentImport, ProcessContentImportRow, ProcessContentImportVersion } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let canImport = $derived(can('E-6'));
	let file = $state<File | null>(null);
	let report = $state<ProcessContentImport | null>(null);
	let busy = $state(false);
	let failure = $state('');

	async function send(commit: boolean) {
		failure = '';
		if (!file) {
			failure = t('contentImportChooseFile');
			return;
		}
		busy = true;
		const result = await importProcessContent(file, commit);
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

	function summary(r: ProcessContentImport): string {
		const counts = { new: String(r.new), existing: String(r.existing), invalid: String(r.invalid) };
		return t(r.committed ? 'contentImportDoneSummary' : 'contentImportPreviewSummary', counts);
	}

	function scopeLabel(row: ProcessContentImportRow): string {
		switch (row.scope_level) {
			case 'general':
				return t('scopeGeneral');
			case 'customer':
				return t('scopeCustomer', { label: row.scope_target });
			case 'model_group':
				return t('scopeModelGroup', { label: row.scope_target });
			case 'model':
				return t('scopeModel', { label: row.scope_target });
		}
		return '';
	}

	function resultText(row: ProcessContentImportRow, committed: boolean): string {
		if (row.status === 'invalid') {
			return row.problems.map((p) => problemText(p)).join(' ');
		}
		if (row.status === 'existing') {
			return t('contentImportStatusExisting');
		}
		return t(committed ? 'contentImportStatusCreated' : 'contentImportStatusNew');
	}

	function versionText(v: ProcessContentImportVersion): string {
		const values = { process: v.process, version: String(v.version), points: String(v.points) };
		switch (v.change) {
			case 'new_template':
				return t('contentImportVersionNewTemplate', values);
			case 'new_version':
				return t('contentImportVersionNew', values);
		}
		return t('contentImportVersionDraft', values);
	}
</script>

<h1>{t('contentImportTitle')}</h1>
<p>{t('contentImportIntro')}</p>
<p><a href="/process-templates">{t('contentImportBack')}</a></p>

{#if !canImport}
	<p>{t('contentImportNoPermission')}</p>
{:else}
	<ol class="steps">
		<li><a href="/api/v1/process-template-imports/template" download>{t('contentImportTemplate')}</a> · {t('contentImportTemplateHint')}</li>
		<li>{t('contentImportFillHint')}</li>
		<li>{t('contentImportUploadHint')}</li>
	</ol>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={preview} novalidate>
		<div class="field">
			<label for="content-import-file">{t('contentImportFile')}</label>
			<input
				id="content-import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				onchange={(e) => {
					file = e.currentTarget.files?.[0] ?? null;
					report = null;
				}}
			/>
		</div>
		<div class="row">
			<button type="submit" disabled={busy}>{t('contentImportPreview')}</button>
			{#if report && !report.committed && report.new > 0}
				<button type="button" class="primary" disabled={busy} onclick={() => send(true)}>{t('contentImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if report}
		<p role="status">{summary(report)}</p>
		{#if report.versions.length > 0}
			<ul class="versions">
				{#each report.versions as v (v.process)}
					<li data-version={v.change}>{versionText(v)}</li>
				{/each}
			</ul>
			<p class="hint">{t('contentImportPublishHint')}</p>
		{/if}
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('contentImportColumnSheet')}</th>
						<th>{t('contentImportColumnRow')}</th>
						<th>{t('contentImportColumnProcess')}</th>
						<th>{t('contentImportColumnSubprocess')}</th>
						<th>{t('contentImportColumnText')}</th>
						<th>{t('pointScope')}</th>
						<th>{t('contentImportColumnResult')}</th>
					</tr>
				</thead>
				<tbody>
					{#each report.rows as row (row.sheet + row.row)}
						<tr data-import={row.status}>
							<td>{row.sheet}</td>
							<td>{row.row}</td>
							<td>{row.process}</td>
							<td>{row.subprocess}</td>
							<td>{row.text}</td>
							<td>{scopeLabel(row)}</td>
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
