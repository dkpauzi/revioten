<script lang="ts">
	// Import master data dari berkas Excel (I5). PPAP Engineer (E-11) mengunduh
	// template, mengisi sheet Customers (Project Group dan customer) dan Models
	// (group model, model, part number), lalu mengunggahnya: Preview
	// melaporkan setiap baris tanpa menyimpan, Import menyimpan baris yang
	// benar. Data yang sudah ada dilewati, jadi berkas yang sama bisa dipakai
	// lagi sesudah database direset. Server tetap memeriksa haknya. Judul
	// kolom tabel adalah judul kolom berkas Excel, jadi tidak diterjemahkan.
	import { importMasterData } from '#lib/api/client.ts';
	import type { MasterDataImport, MasterDataImportRow } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let canImport = $derived(can('E-11'));
	let file = $state<File | null>(null);
	let report = $state<MasterDataImport | null>(null);
	let busy = $state(false);
	let failure = $state('');

	async function send(commit: boolean) {
		failure = '';
		if (!file) {
			failure = t('masterImportChooseFile');
			return;
		}
		busy = true;
		const result = await importMasterData(file, commit);
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

	function summary(r: MasterDataImport): string {
		const counts = { new: String(r.new), existing: String(r.existing), invalid: String(r.invalid) };
		return t(r.committed ? 'masterImportDoneSummary' : 'masterImportPreviewSummary', counts);
	}

	function resultText(row: MasterDataImportRow, committed: boolean): string {
		if (row.status === 'existing') {
			return t('masterImportStatusExisting');
		}
		if (row.status === 'invalid') {
			return row.problems.map((p) => problemText(p)).join(' ');
		}
		const what = row.creates.map((c) => t(`masterImportCreates_${c}` as MessageKey)).join(', ');
		return t(committed ? 'masterImportStatusCreated' : 'masterImportStatusNew', { what });
	}

	const sections = [
		{ key: 'customers', title: 'masterImportCustomersTitle', columns: ['Project group code', 'Customer code', 'Customer name'] },
		{ key: 'models', title: 'masterImportModelsTitle', columns: ['Customer code', 'Model group', 'Model', 'Part number'] }
	] as const;
</script>

<h1>{t('masterImportTitle')}</h1>
<p>{t('masterImportIntro')}</p>
<p><a href="/customers">{t('masterImportBack')}</a></p>

{#if !canImport}
	<p>{t('masterImportNoPermission')}</p>
{:else}
	<ol class="steps">
		<li><a href="/api/v1/master-data/import-template" download>{t('masterImportTemplate')}</a> · {t('masterImportTemplateHint')}</li>
		<li>{t('masterImportFillHint')}</li>
		<li>{t('masterImportUploadHint')}</li>
	</ol>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={preview} novalidate>
		<div class="field">
			<label for="master-import-file">{t('masterImportFile')}</label>
			<input
				id="master-import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				onchange={(e) => {
					file = e.currentTarget.files?.[0] ?? null;
					report = null;
				}}
			/>
		</div>
		<div class="row">
			<button type="submit" disabled={busy}>{t('masterImportPreview')}</button>
			{#if report && !report.committed && report.new > 0}
				<button type="button" class="primary" disabled={busy} onclick={() => send(true)}>{t('masterImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if report}
		<p role="status">{summary(report)}</p>
		{#each sections as section (section.key)}
			{#if report[section.key].length > 0}
				<section>
					<h2>{t(section.title)}</h2>
					<div class="scroll">
						<table>
							<thead>
								<tr>
									<th>{t('masterImportColumnRow')}</th>
									{#each section.columns as c (c)}<th>{c}</th>{/each}
									<th>{t('masterImportColumnResult')}</th>
								</tr>
							</thead>
							<tbody>
								{#each report[section.key] as row (row.row)}
									<tr data-import={row.status}>
										<td>{row.row}</td>
										{#each section.columns as c (c)}<td>{row.cells[c.toLowerCase()] ?? ''}</td>{/each}
										<td class:error={row.status === 'invalid'}>{resultText(row, report.committed)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</section>
			{/if}
		{/each}
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
