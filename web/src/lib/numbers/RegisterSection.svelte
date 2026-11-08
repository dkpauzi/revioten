<script lang="ts">
	// Register nomor (M2.3, UC-12, UC-13): daftar dengan saringan, umur nomor
	// Reserved, dan perubahan status Void atau Obsolete untuk dokumen kontrol
	// (E-8). Nomor tidak pernah dihapus; yang berubah hanya statusnya.
	import { untrack } from 'svelte';
	import {
		listDocumentNumbers,
		makeDocumentNumberObsolete,
		voidDocumentNumber,
		type NumberFilter
	} from '#lib/api/client.ts';
	import type { Customer, DocumentNumber, DocumentType } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { canObsolete, canVoid, reservedDays, statusLabel } from '#lib/numbers.ts';

	// refresh dinaikkan halaman setiap kali nomor baru didaftarkan.
	let {
		customers,
		types,
		canManage,
		refresh
	}: { customers: Customer[]; types: DocumentType[]; canManage: boolean; refresh: number } = $props();

	const statuses: DocumentNumber['status'][] = ['reserved', 'in_use', 'released', 'obsolete', 'void', 'skipped'];

	// Saringan.
	let customerId = $state('');
	let typeId = $state('');
	let status = $state('');
	let query = $state('');

	let items = $state<DocumentNumber[]>([]);
	let nextBefore = $state<string | undefined>(undefined);
	let listError = $state('');
	let message = $state('');

	// Perubahan status yang sedang diisi alasannya.
	let editing = $state<{ number: DocumentNumber; action: 'void' | 'obsolete' } | null>(null);
	let reason = $state('');
	let editError = $state('');

	// load memuat halaman pertama, atau halaman berikutnya bila more true.
	async function load(more = false) {
		const filter: NumberFilter = { customer_id: customerId, document_type_id: typeId, status, q: query };
		if (more) {
			filter.before = nextBefore;
		}
		const result = await listDocumentNumbers(filter);
		if (!result.ok) {
			listError = failureText(result.error);
			return;
		}
		listError = '';
		items = more ? [...items, ...result.data.items] : result.data.items;
		nextBefore = result.data.next_before;
	}

	function applyFilter(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		load();
	}

	function startEdit(number: DocumentNumber, action: 'void' | 'obsolete') {
		editing = { number, action };
		reason = '';
		editError = '';
		message = '';
	}

	async function confirmEdit(event: SubmitEvent) {
		event.preventDefault();
		if (editing === null) {
			return;
		}
		const { number, action } = editing;
		const result =
			action === 'void'
				? await voidDocumentNumber(number.id, reason)
				: await makeDocumentNumberObsolete(number.id, reason);
		if (!result.ok) {
			const detail = result.error.body?.details?.[0];
			editError = detail ? problemText(detail) : failureText(result.error);
			return;
		}
		editing = null;
		message = t('numberStatusChanged', { number: result.data.number, status: statusLabel(result.data.status) });
		await load();
	}

	// Register dimuat saat halaman dibuka dan setiap ada nomor baru (refresh
	// berubah). load dijalankan lewat untrack, supaya mengetik di saringan
	// tidak langsung memuat ulang; saringan berlaku saat tombol Filter ditekan.
	$effect(() => {
		void refresh;
		untrack(() => load());
	});

	// Waktu sekarang untuk umur nomor Reserved; cukup diambil sekali.
	const now = new Date();
</script>

<section>
	<h2>{t('numberRegisterListTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	<form class="filters" onsubmit={applyFilter}>
		<div class="field">
			<label for="filter-customer">{t('numberCustomer')}</label>
			<select id="filter-customer" bind:value={customerId}>
				<option value="">{t('numberFilterAll')}</option>
				{#each customers as c (c.id)}
					<option value={c.id}>{c.code} · {c.name}</option>
				{/each}
			</select>
		</div>
		<div class="field">
			<label for="filter-type">{t('numberType')}</label>
			<select id="filter-type" bind:value={typeId}>
				<option value="">{t('numberFilterAll')}</option>
				{#each types as d (d.id)}
					<option value={d.id}>{d.name}</option>
				{/each}
			</select>
		</div>
		<div class="field">
			<label for="filter-status">{t('numberFilterStatus')}</label>
			<select id="filter-status" bind:value={status}>
				<option value="">{t('numberFilterAll')}</option>
				{#each statuses as s (s)}
					<option value={s}>{statusLabel(s)}</option>
				{/each}
			</select>
		</div>
		<div class="field">
			<label for="filter-query">{t('numberFilterSearch')}</label>
			<input id="filter-query" bind:value={query} autocomplete="off" maxlength="50" />
		</div>
		<button type="submit">{t('numberFilterApply')}</button>
	</form>

	{#if listError}
		<p class="error" role="alert">{listError}</p>
	{:else if items.length === 0}
		<p>{t('numberRegisterEmpty')}</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>{t('numberColumnNumber')}</th>
					<th>{t('numberColumnCustomer')}</th>
					<th>{t('numberColumnType')}</th>
					<th>{t('numberColumnStatus')}</th>
					<th>{t('numberColumnRegisteredBy')}</th>
					<th>{t('numberColumnRegisteredAt')}</th>
					<th>{t('numberColumnReason')}</th>
					{#if canManage}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each items as n (n.id)}
					<tr>
						<td class="mono">
							{n.number}
							{#if n.initial_revision !== 0}<small>{n.revision_label}</small>{/if}
						</td>
						<td class="mono">{n.customer.code}</td>
						<td>{n.document_type.name}</td>
						<td>
							<!-- Label status: warna selalu disertai kata (D-127). -->
							<span class="badge" data-status={n.status}>{statusLabel(n.status)}</span>
							{#if n.status === 'reserved'}
								<br /><small>{t('numberReservedDays', { days: String(reservedDays(n.created_at, now)) })}</small>
							{/if}
						</td>
						<td>{n.registered_by.name}</td>
						<td>{formatDateTime(n.created_at)}</td>
						<td>{n.status_reason ?? ''}</td>
						{#if canManage}
							<td>
								{#if canVoid(n)}
									<button type="button" onclick={() => startEdit(n, 'void')}>{t('numberVoid')}</button>
								{/if}
								{#if canObsolete(n)}
									<button type="button" onclick={() => startEdit(n, 'obsolete')}>{t('numberObsolete')}</button>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
		{#if nextBefore}
			<button type="button" onclick={() => load(true)}>{t('numberLoadMore')}</button>
		{/if}
	{/if}

	{#if editing}
		<form class="edit" onsubmit={confirmEdit} novalidate>
			<div class="field">
				<label for="status-reason">{t('numberReason', { number: editing.number.number })}</label>
				<input id="status-reason" bind:value={reason} autocomplete="off" maxlength="500" size="40" />
				{#if editError}<p class="error">{editError}</p>{/if}
			</div>
			<button type="submit">{editing.action === 'void' ? t('numberVoid') : t('numberObsolete')}</button>
			<button type="button" onclick={() => (editing = null)}>{t('masterCancel')}</button>
		</form>
	{/if}
</section>

<style>
	.filters,
	.edit {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
		margin-bottom: 1rem;
	}
</style>
