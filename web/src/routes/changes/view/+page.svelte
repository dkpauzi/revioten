<script lang="ts">
	// Halaman satu perubahan (M5.1): sumber, jenis, alasan, dan dokumen
	// terdampak. Selama perubahan masih diajukan, pemegang E-3 bisa menambah
	// atau melepas dokumen terdampak (yang terakhir tidak bisa dilepas).
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { addChangeDocument, getChange, listDocuments, removeChangeDocument, type ApiFailure } from '#lib/api/client.ts';
	import type { Change, DocumentSummary } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let change = $state<Change | null>(null);
	let documents = $state<DocumentSummary[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let documentId = $state('');
	let canPropose = $derived(can('E-3'));
	let editable = $derived(canPropose && change?.status === 'submitted');
	let addable = $derived(documents.filter((d) => !change?.documents.some((x) => x.id === d.id)));

	let currentId = $derived(page.url.searchParams.get('id') ?? '');
	$effect(() => {
		const id = currentId;
		untrack(() => load(id));
	});

	async function load(id: string) {
		const result = await getChange(id);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		change = result.data;
		const docs = await listDocuments(result.data.customer_id);
		documents = docs.ok ? docs.data.items : [];
	}

	function showError(error: ApiFailure) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		if (!change || documentId === '') {
			return;
		}
		const result = await addChangeDocument(change.id, documentId);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		const number = documents.find((d) => d.id === documentId)?.number ?? '';
		change = result.data;
		documentId = '';
		message = t('changeDocumentAdded', { number });
	}

	async function remove(id: string, number: string) {
		message = '';
		failure = '';
		if (!change) {
			return;
		}
		const result = await removeChangeDocument(change.id, id);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		change = result.data;
		message = t('changeDocumentRemoved', { number });
	}
</script>

{#if loadError}
	<h1>{t('changesTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
{:else if !change}
	<p>{t('formatLoading')}</p>
{:else}
	<p><a href="/changes">{t('changeBack')}</a></p>
	<h1 class="mono">{change.ref}</h1>
	<dl class="facts">
		<dt>{t('numberColumnStatus')}</dt>
		<dd><span class="badge" data-change-status={change.status}>{t(`changeStatus_${change.status}` as 'changeStatus_submitted')}</span></dd>
		<dt>{t('documentCustomer')}</dt>
		<dd>{change.customer_code}</dd>
		<dt>{t('changeMechanism')}</dt>
		<dd>{t(`changeMechanism_${change.mechanism}` as 'changeMechanism_manual')}</dd>
		{#if change.trigger}
			<dt>{t('changeTrigger')}</dt>
			<dd>{change.trigger.name}</dd>
		{/if}
		<dt>{t('changeType')}</dt>
		<dd>{change.change_type.name}</dd>
		{#if change.four_m}
			<dt>{t('changeFourM')}</dt>
			<dd>{change.four_m.name}</dd>
		{/if}
		{#if change.source_reference}
			<dt>{t('changeReference')}</dt>
			<dd>{change.source_reference}</dd>
		{/if}
		<dt>{t('changeReason')}</dt>
		<dd class="reason">{change.reason}</dd>
		<dt>{t('changeRequestedBy')}</dt>
		<dd>{change.requested_by} · {formatDateTime(change.created_at)}</dd>
	</dl>

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<section>
		<h2>{t('changeDocuments')}</h2>
		<table>
			<thead>
				<tr>
					<th>{t('documentNumber')}</th>
					<th>{t('numberColumnType')}</th>
					<th>{t('changeRevision')}</th>
					{#if editable}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each change.documents as d (d.id)}
					<tr>
						<td class="mono"><a href="/documents/view?id={d.id}">{d.number}</a></td>
						<td>{d.document_type.name}</td>
						<td>{d.revision}</td>
						{#if editable}
							<td>
								<button type="button" aria-label={t('changeDocumentRemoveFor', { number: d.number })} onclick={() => remove(d.id, d.number)}>{t('changeDocumentRemove')}</button>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
		{#if editable && addable.length > 0}
			<form class="row" onsubmit={add} novalidate>
				<div class="field">
					<label for="change-add-document">{t('changeDocumentAdd')}</label>
					<select id="change-add-document" bind:value={documentId}>
						<option value="">{t('changeChoose')}</option>
						{#each addable as d (d.id)}<option value={d.id}>{d.number} · {d.document_type.name}</option>{/each}
					</select>
				</div>
				<button type="submit">{t('changeDocumentAddButton')}</button>
			</form>
		{/if}
	</section>
{/if}

<style>
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 4px 16px;
		margin: 0 0 1.5rem;
	}

	.facts dt {
		color: var(--text-3);
	}

	.facts dd {
		margin: 0;
	}

	.reason {
		white-space: pre-wrap;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
