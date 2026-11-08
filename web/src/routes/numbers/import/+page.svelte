<script lang="ts">
	// Halaman impor nomor lama (M2.5, UC-14, R-NUM-06): dokumen kontrol (E-8)
	// menempel kolom nomor dari register Excel satu customer. Pratinjau dulu,
	// lalu impor. Lubang, nomor ganda, dan set yang tidak sejajar hanya
	// dilaporkan; tidak ada nomor yang diperbaiki atau dinomori ulang.
	import { onMount } from 'svelte';
	import { importDocumentNumbers, listCustomers } from '#lib/api/client.ts';
	import type { Customer, NumberImportReport, NumberImportRequest } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { lineResultText, notImported, rangesText, statusLabel, typeNames } from '#lib/numbers.ts';
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
	}

	// send mengirim isian sebagai pratinjau (dryRun true) atau impor sungguhan.
	async function send(dryRun: boolean) {
		fieldErrors = {};
		failure = '';
		busy = true;
		const input = { customer_id: customerId, status: status as NumberImportRequest['status'], text, dry_run: dryRun };
		const result = await importDocumentNumbers(input);
		busy = false;
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

	// summary adalah ringkasan laporan di atas tabel.
	function summary(r: NumberImportReport): string {
		if (r.imported === 0) {
			return t('numberImportNothing');
		}
		return t(r.dry_run ? 'numberImportPreviewed' : 'numberImportDone', { count: String(r.imported) });
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
		<div class="row">
			<button type="submit" disabled={busy}>{t('numberImportPreview')}</button>
			{#if report?.dry_run && report.imported > 0}
				<button type="button" disabled={busy} onclick={() => send(false)}>{t('numberImportRun')}</button>
			{/if}
		</div>
	</form>

	{#if report}
		<section>
			<p role="status">{summary(report)}</p>

			{#if notImported(report.lines).length > 0}
				<h2>{t('numberImportLinesTitle')}</h2>
				<table>
					<thead>
						<tr>
							<th>{t('numberImportColumnLine')}</th>
							<th>{t('numberImportColumnText')}</th>
							<th>{t('numberImportColumnResult')}</th>
						</tr>
					</thead>
					<tbody>
						{#each notImported(report.lines) as l (l.line)}
							<tr>
								<td>{l.line}</td>
								<td>{l.text}</td>
								<td>{lineResultText(l)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}

			{#if report.groups.length > 0}
				<h2>{t('numberImportGroupsTitle')}</h2>
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

				<h2>{t('numberImportMisalignedTitle')}</h2>
				<p class="hint">{t('numberImportMisalignedHint')}</p>
				{#if report.misaligned.length === 0}
					<p>{t('numberImportAligned')}</p>
				{:else}
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
				{/if}
			{/if}
		</section>
	{/if}
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
</style>
