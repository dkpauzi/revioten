<script lang="ts">
	// Dokumen terbit (D-195, keputusan E-128): daftar dokumen satu customer yang
	// punya revisi berlaku. Untuk semua akun, termasuk Viewer (internal audit,
	// QA, QC, produksi). Draft tidak pernah tampil di sini.
	import { onMount } from 'svelte';
	import { listCustomers, listReleasedDocuments } from '#lib/api/client.ts';
	import type { Customer, DocumentSummary } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let documents = $state<DocumentSummary[] | null>(null);
	let search = $state('');
	let loadError = $state('');

	let shown = $derived(
		(documents ?? []).filter((d) => {
			const q = search.trim().toLowerCase();
			return q === '' || d.number.toLowerCase().includes(q) || d.models.some((m) => m.toLowerCase().includes(q));
		})
	);

	async function load() {
		documents = null;
		if (customerId === '') {
			return;
		}
		const result = await listReleasedDocuments(customerId);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		loadError = '';
		documents = result.data.items;
	}

	onMount(async () => {
		const result = await listCustomers();
		customers = result.ok ? result.data.items : [];
		if (customers.length === 1) {
			customerId = customers[0].id;
			await load();
		}
	});
</script>

<h1>{t('releasedTitle')}</h1>
<p class="hint">{t('releasedHint')}</p>

<div class="filters">
	<div class="field">
		<label for="released-customer">{t('documentCustomer')}</label>
		<select id="released-customer" bind:value={customerId} onchange={load}>
			<option value="">{t('releasedChooseCustomer')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
	</div>
	<div class="field">
		<label for="released-search">{t('releasedSearch')}</label>
		<input id="released-search" type="search" bind:value={search} />
	</div>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if documents}
	{#if shown.length === 0}
		<p>{t('releasedEmpty')}</p>
	{:else}
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('numberColumnNumber')}</th>
						<th>{t('numberColumnType')}</th>
						<th>{t('releasedRevision')}</th>
						<th>{t('documentModels')}</th>
						<th>{t('releasedAt')}</th>
					</tr>
				</thead>
				<tbody>
					{#each shown as d (d.id)}
						<tr>
							<td><a href="/documents/released/view?id={d.id}">{d.number}</a></td>
							<td>{d.document_type.name}</td>
							<td>{d.revision}{d.released_migrated ? ` · ${t('documentStatusReleasedMigrated')}` : ''}</td>
							<td>{d.models.join(', ')}</td>
							<td>{d.updated_at.slice(0, 10)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	.filters {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
		max-width: 44rem;
		margin-bottom: 1.5rem;
	}

	.scroll {
		overflow-x: auto;
	}
</style>
