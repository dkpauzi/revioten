<script lang="ts">
	// Import akun dari berkas Excel (I1, D-224). Admin mengunduh template,
	// mengisinya di Excel, lalu mengunggahnya: Preview melaporkan setiap baris
	// tanpa menyimpan, Import menyimpan baris yang benar. Login yang sudah ada
	// dilewati, jadi berkas yang sama bisa dipakai lagi sesudah database
	// direset. Preview bisa kapan saja; Import hanya saat Migration mode
	// menyala, dan server tetap memeriksanya.
	import { onMount } from 'svelte';
	import { getSystemSettings, importAccounts, listRoles } from '#lib/api/client.ts';
	import type { AccountImport, AccountImportRow, Role } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { grantLabel } from '#lib/roles.ts';
	import { can } from '#lib/session.svelte.ts';

	let canImport = $derived(can('A-1'));
	let migrationMode = $state<boolean | null>(null);
	let catalog = $state<Role[]>([]);
	let file = $state<File | null>(null);
	let report = $state<AccountImport | null>(null);
	let busy = $state(false);
	let failure = $state('');

	onMount(async () => {
		const [settings, roles] = await Promise.all([getSystemSettings(), listRoles()]);
		migrationMode = settings.ok ? settings.data.migration_mode : false;
		catalog = roles.ok ? roles.data.items : [];
	});

	async function send(commit: boolean) {
		failure = '';
		if (!file) {
			failure = t('accountImportChooseFile');
			return;
		}
		busy = true;
		const result = await importAccounts(file, commit);
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

	function summary(r: AccountImport): string {
		const counts = { new: String(r.new), existing: String(r.existing), invalid: String(r.invalid) };
		if (!r.committed) {
			return t('accountImportPreviewSummary', counts);
		}
		return r.new > 0
			? t('accountImportDoneSummary', { ...counts, time: formatDateTime(r.expires_at ?? '') })
			: t('accountImportNothingSummary', counts);
	}

	function statusText(row: AccountImportRow, committed: boolean): string {
		if (row.status === 'new') {
			return committed ? t('accountImportStatusCreated') : t('accountImportStatusNew');
		}
		if (row.status === 'existing') {
			return t('accountImportStatusExisting');
		}
		return row.problems.map((p) => problemText(p)).join(' ');
	}
</script>

<h1>{t('accountImportTitle')}</h1>
<p>{t('accountImportIntro')}</p>
<p><a href="/admin/accounts">{t('accountImportBack')}</a></p>

{#if !canImport}
	<p>{t('accountImportNoPermission')}</p>
{:else if migrationMode === null}
	<p>{t('numbersLoading')}</p>
{:else}
	{#if !migrationMode}
		<p class="hint" data-migration="off">{t('accountImportMigrationOff')}</p>
	{/if}
	<ol class="steps">
		<li><a href="/api/v1/accounts/import-template" download>{t('accountImportTemplate')}</a> · {t('accountImportTemplateHint')}</li>
		<li>{t('accountImportFillHint')}</li>
		<li>{t('accountImportUploadHint')}</li>
	</ol>
	<p class="hint">{t('accountImportPasswordHint')}</p>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={preview} novalidate>
		<div class="field">
			<label for="account-import-file">{t('accountImportFile')}</label>
			<input
				id="account-import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				onchange={(e) => {
					file = e.currentTarget.files?.[0] ?? null;
					report = null;
				}}
			/>
		</div>
		<div class="row">
			<button type="submit" disabled={busy}>{t('accountImportPreview')}</button>
			{#if report && !report.committed && report.new > 0}
				<button type="button" class="primary" disabled={busy || !migrationMode} onclick={() => send(true)}>{t('accountImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if report}
		<section>
			<p role="status">{summary(report)}</p>
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('accountImportColumnRow')}</th>
							<th>{t('accountPersonName')}</th>
							<th>{t('accountLogin')}</th>
							<th>{t('accountRoles')}</th>
							<th>{t('accountImportColumnResult')}</th>
						</tr>
					</thead>
					<tbody>
						{#each report.rows as row (row.row)}
							<tr data-import={row.status}>
								<td>{row.row}</td>
								<td>{row.name}</td>
								<td class="mono">{row.login}</td>
								<td>{row.status === 'invalid' ? row.roles_text : row.roles.map((g) => grantLabel(g, catalog)).join(', ')}</td>
								<td class:error={row.status === 'invalid'}>{statusText(row, report.committed)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
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
