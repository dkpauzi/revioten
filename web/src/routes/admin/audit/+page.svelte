<script lang="ts">
	// Halaman audit log (M1.3, hak A-4, hanya akun admin, D-75). Entri terbaru
	// dulu; entri yang lebih lama dimuat per halaman.
	import { onMount } from 'svelte';
	import { listAuditLog, verifyAuditLog } from '#lib/api/client.ts';
	import type { AuditEntry } from '#lib/api/gen/index.ts';
	import {
		actionLabel,
		actorLabel,
		detailsText,
		targetLabel,
		verificationText
	} from '#lib/audit.ts';
	import { failureText } from '#lib/errors.ts';
	import { formatTimestamp } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';

	// Entri yang sudah dimuat. null berarti halaman pertama belum selesai dimuat.
	let entries = $state<AuditEntry[] | null>(null);
	// nextBefore adalah nomor untuk meminta entri yang lebih lama; undefined bila tidak ada lagi.
	let nextBefore = $state<number | undefined>(undefined);
	let loading = $state(false);
	let loadError = $state('');

	let checking = $state(false);
	let verification = $state('');
	let verificationOk = $state(true);

	// load memuat halaman pertama (before kosong) atau halaman berikutnya.
	async function load(before?: number) {
		loading = true;
		const result = await listAuditLog(before);
		loading = false;
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		entries = before === undefined ? result.data.items : [...(entries ?? []), ...result.data.items];
		nextBefore = result.data.next_before;
		loadError = '';
	}

	// verify meminta server memeriksa seluruh rantai hash, dari entri pertama.
	async function verify() {
		checking = true;
		verification = '';
		const result = await verifyAuditLog();
		checking = false;
		if (!result.ok) {
			verification = failureText(result.error);
			verificationOk = false;
			return;
		}
		verification = verificationText(result.data);
		verificationOk = result.data.valid;
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('auditTitle')}</h1>
<p>{t('auditIntro')}</p>

<section>
	<button type="button" onclick={verify} disabled={checking}>
		{checking ? t('auditVerifying') : t('auditVerify')}
	</button>
	{#if verification}
		{#if verificationOk}
			<p role="status">{verification}</p>
		{:else}
			<p class="error" role="alert">{verification}</p>
		{/if}
	{/if}
</section>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if entries === null}
	<p>{t('auditLoading')}</p>
{:else if entries.length === 0}
	<p>{t('auditEmpty')}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>{t('auditSeq')}</th>
				<th>{t('auditTime')}</th>
				<th>{t('auditActor')}</th>
				<th>{t('auditAction')}</th>
				<th>{t('auditTarget')}</th>
				<th>{t('auditDetails')}</th>
				<th>{t('auditRequestId')}</th>
			</tr>
		</thead>
		<tbody>
			{#each entries as entry (entry.seq)}
				<tr>
					<td>{entry.seq}</td>
					<td>{formatTimestamp(entry.occurred_at)}</td>
					<td>{actorLabel(entry)}</td>
					<td>{actionLabel(entry.action)}</td>
					<td>{targetLabel(entry)}</td>
					<td>{detailsText(entry.details)}</td>
					<td><code>{entry.request_id ?? ''}</code></td>
				</tr>
			{/each}
		</tbody>
	</table>
	{#if nextBefore !== undefined}
		<button type="button" onclick={() => load(nextBefore)} disabled={loading}>
			{t('auditLoadMore')}
		</button>
	{/if}
{/if}

<style>
	section {
		margin-bottom: 1.5rem;
	}

	.error {
		color: #b00020;
	}

	table {
		border-collapse: collapse;
		margin-bottom: 1rem;
	}

	th,
	td {
		border: 1px solid #ccc;
		padding: 0.25rem 0.75rem;
		text-align: left;
		vertical-align: top;
	}
</style>
