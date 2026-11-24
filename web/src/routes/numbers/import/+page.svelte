<script lang="ts">
	// Halaman impor nomor lama (M2.5, UC-14, R-NUM-06): dokumen kontrol (E-8)
	// menempel kolom nomor dari register Excel satu customer. Pratinjau dulu,
	// lalu impor. Lubang, nomor ganda, dan set yang tidak sejajar hanya
	// dilaporkan; tidak ada nomor yang diperbaiki atau dinomori ulang.
	// Sejak I4 juga dari berkas Excel (beberapa customer, status per baris),
	// dan nomor boleh memuat revisi, misalnya SC-KT-001-05.
	import { onMount } from 'svelte';
	import { getNumberSettings, importDocumentNumbers, importDocumentNumbersFile, listCustomers } from '#lib/api/client.ts';
	import type { Customer, NumberImportReport, NumberImportRequest } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import ImportReport from '#lib/numbers/ImportReport.svelte';
	import { statusLabel } from '#lib/numbers.ts';
	import { can } from '#lib/session.svelte.ts';

	// Status yang boleh untuk nomor lama. Skipped hanya dari daftar sebagai set.
	const statuses: NumberImportRequest['status'][] = ['reserved', 'in_use', 'released', 'obsolete', 'void'];

	// Data dari server. null berarti belum selesai dimuat.
	let customers = $state<Customer[] | null>(null);
	let loadError = $state('');
	let canImport = $derived(can('E-8'));
	// Hanya customer aktif yang bisa diimpor (ASUMSI E-67), sama dengan pendaftaran.
	let activeCustomers = $derived((customers ?? []).filter((c) => c.active));

	// Isian.
	let customerId = $state('');
	let status = $state('');
	let text = $state('');
	// Bawaannya satu set per baris (D-216).
	let oneSet = $state(true);

	// Isian berkas Excel (I4).
	let file = $state<File | null>(null);
	let fileOneSet = $state(true);
	let fileFailure = $state('');
	// fromFile true bila laporan yang tampil berasal dari berkas.
	let fromFile = $state(false);
	let revisionDigits = $state(2);

	let report = $state<NumberImportReport | null>(null);
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');
	let busy = $state(false);

	async function load() {
		const result = await listCustomers();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		customers = result.data.items;
		loadError = '';
		const settings = await getNumberSettings();
		if (settings.ok) {
			revisionDigits = settings.data.revision_digits;
		}
	}

	// sendFile mengirim berkas sebagai pratinjau (commit false) atau impor.
	async function sendFile(commit: boolean) {
		fileFailure = '';
		if (!file) {
			fileFailure = t('numberImportFileChoose');
			return;
		}
		busy = true;
		const result = await importDocumentNumbersFile(file, commit, fileOneSet);
		busy = false;
		fromFile = true;
		if (!result.ok) {
			report = null;
			const details = result.error.body?.details ?? [];
			fileFailure = details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(result.error);
			return;
		}
		report = result.data;
	}

	function previewFile(event: SubmitEvent) {
		event.preventDefault();
		sendFile(false);
	}

	// send mengirim isian sebagai pratinjau (dryRun true) atau impor sungguhan.
	async function send(dryRun: boolean) {
		fieldErrors = {};
		failure = '';
		busy = true;
		const input = { customer_id: customerId, status: status as NumberImportRequest['status'], text, one_set: oneSet, dry_run: dryRun };
		const result = await importDocumentNumbers(input);
		busy = false;
		fromFile = false;
		if (!result.ok) {
			report = null;
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		report = result.data;
	}

	function preview(event: SubmitEvent) {
		event.preventDefault();
		send(true);
	}

	// Laporan lama dibuang bila isian berubah, supaya tombol Import tidak
	// mengimpor isian yang belum dipratinjau.
	function changed() {
		report = null;
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('numberImportTitle')}</h1>
<p>{t('numberImportIntro')}</p>
<p><a href="/numbers">{t('numberImportBack')}</a></p>

{#if !canImport}
	<p>{t('numberImportNoPermission')}</p>
{:else if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if customers === null}
	<p>{t('numbersLoading')}</p>
{:else}
	<h2>{t('numberImportFileTitle')}</h2>
	<ol class="steps">
		<li><a href="/api/v1/document-number-imports/template" download>{t('numberImportFileTemplate')}</a> · {t('numberImportFileTemplateHint')}</li>
		<li>{t('numberImportFileFillHint')}</li>
		<li>{t('numberImportFileUploadHint')}</li>
	</ol>
	{#if fileFailure}<p class="error" role="alert">{fileFailure}</p>{/if}
	<form onsubmit={previewFile} novalidate>
		<div class="field">
			<label for="number-import-file">{t('numberImportFile')}</label>
			<input
				id="number-import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				onchange={(e) => {
					file = e.currentTarget.files?.[0] ?? null;
					changed();
				}}
			/>
		</div>
		<label class="check"><input type="checkbox" bind:checked={fileOneSet} onchange={changed} /> {t('numberImportFileOneSet')}</label>
		<div class="row">
			<button type="submit" disabled={busy}>{t('numberImportFilePreview')}</button>
			{#if fromFile && report?.dry_run && report.imported > 0}
				<button type="button" class="primary" disabled={busy} onclick={() => sendFile(true)}>{t('numberImportFileRun')}</button>
			{/if}
		</div>
	</form>
	{#if fromFile && report}<ImportReport {report} {revisionDigits} />{/if}

	<h2>{t('numberImportPasteTitle')}</h2>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={preview} novalidate>
		<div class="row">
			<div class="field">
				<label for="import-customer">{t('numberCustomer')}</label>
				<select id="import-customer" bind:value={customerId} onchange={changed}>
					<option value="">{t('numberCustomerChoose')}</option>
					{#each activeCustomers as c (c.id)}
						<option value={c.id}>{c.code} · {c.name}</option>
					{/each}
				</select>
				{#if fieldErrors.customer_id}<p class="error">{fieldErrors.customer_id}</p>{/if}
			</div>
			<div class="field">
				<label for="import-status">{t('numberImportStatus')}</label>
				<select id="import-status" bind:value={status} onchange={changed}>
					<option value="">{t('numberImportStatusChoose')}</option>
					{#each statuses as s (s)}
						<option value={s}>{statusLabel(s)}</option>
					{/each}
				</select>
				{#if fieldErrors.status}<p class="error">{fieldErrors.status}</p>{/if}
			</div>
		</div>
		<div class="field">
			<label for="import-text">{t('numberImportText')}</label>
			<textarea id="import-text" bind:value={text} oninput={changed} rows="12" cols="40" spellcheck="false"></textarea>
			<p class="hint">{t('numberImportTextHint')}</p>
			{#if fieldErrors.text}<p class="error">{fieldErrors.text}</p>{/if}
		</div>
		<label class="check"><input type="checkbox" bind:checked={oneSet} onchange={changed} /> {t('numberImportOneSet')}</label>
		<p class="hint">{t('numberImportOneSetHint')}</p>
		<div class="row">
			<button type="submit" disabled={busy}>{t('numberImportPreview')}</button>
			{#if !fromFile && report?.dry_run && report.imported > 0}
				<button type="button" disabled={busy} onclick={() => send(false)}>{t('numberImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if !fromFile && report}<ImportReport {report} {revisionDigits} />{/if}
{/if}

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		align-items: flex-start;
		margin-bottom: 1.5rem;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
	}

	textarea {
		font-family: var(--font-mono);
	}

	.steps {
		padding-left: 1.25rem;
	}
</style>
