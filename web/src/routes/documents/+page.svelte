<script lang="ts">
	// Halaman Documents (M4.1): dokumen draft satu customer, dan membuat
	// dokumen baru dari nomor Reserved. Melihat butuh L-4, membuat butuh E-2;
	// server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import {
		createDocument,
		getSystemSettings,
		listCustomers,
		listDocumentNumbers,
		listDocuments,
		listModels,
		listPartNumbers
	} from '#lib/api/client.ts';
	import type { Customer, DocumentNumber, DocumentSummary, Model, PartNumber } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let documents = $state<DocumentSummary[] | null>(null);
	let numbers = $state<DocumentNumber[]>([]);
	let models = $state<Model[]>([]);
	let parts = $state<PartNumber[]>([]);
	let loadError = $state('');
	let canCreate = $derived(can('E-2'));

	let draft = $state<{ numberId: string; modelIds: string[]; partIds: string[] }>({ numberId: '', modelIds: [], partIds: [] });
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// Dokumen lama pindah ke menu migrasi (D-215); di sini hanya tautan.
	let migrationMode = $state(false);
	let canMigrate = $derived(can('E-10') && migrationMode);

	// Part number yang bisa dipilih: milik model yang sudah dicentang.
	let partChoices = $derived(parts.filter((p) => draft.modelIds.includes(p.model.id)));

	onMount(async () => {
		const result = await listCustomers();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		customers = result.data.items.filter((c) => c.active);
		const settings = await getSystemSettings();
		migrationMode = settings.ok && settings.data.migration_mode;
	});

	async function chooseCustomer() {
		documents = null;
		numbers = [];
		models = [];
		parts = [];
		draft = { numberId: '', modelIds: [], partIds: [] };
		message = '';
		failure = '';
		if (customerId === '') {
			return;
		}
		const [docResult, numberResult, modelResult, partResult] = await Promise.all([
			listDocuments(customerId),
			listDocumentNumbers({ customer_id: customerId, status: 'reserved' }),
			listModels(customerId),
			listPartNumbers(customerId)
		]);
		if (!docResult.ok) {
			loadError = failureText(docResult.error);
			return;
		}
		loadError = '';
		documents = docResult.data.items;
		// Hanya nomor PFD, PFMEA, dan CP yang menjadi dokumen.
		numbers = numberResult.ok ? numberResult.data.items.filter((n) => n.document_type.key !== undefined) : [];
		models = modelResult.ok ? modelResult.data.items : [];
		parts = partResult.ok ? partResult.data.items : [];
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result = await createDocument({
			document_number_id: draft.numberId,
			model_ids: draft.modelIds,
			part_number_ids: draft.partIds.filter((id) => partChoices.some((p) => p.id === id))
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		const created = result.data;
		await chooseCustomer();
		message = t('documentCreated', { number: created.number });
	}
</script>

<h1>{t('documentsTitle')}</h1>
<p>{t('documentsIntro')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

<div class="field chooser">
	<label for="documents-customer">{t('documentCustomer')}</label>
	<select id="documents-customer" bind:value={customerId} onchange={chooseCustomer}>
		<option value="">{t('numberCustomerChoose')}</option>
		{#each customers as c (c.id)}
			<option value={c.id}>{c.code} · {c.name}</option>
		{/each}
	</select>
</div>

{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if documents !== null}
	<section>
		<h2>{t('documentsListTitle')}</h2>
		{#if documents.length === 0}
			<p>{t('documentsEmpty')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>{t('numberColumnNumber')}</th>
						<th>{t('numberColumnType')}</th>
						<th>{t('documentFormSet')}</th>
						<th>{t('documentRows')}</th>
						<th>{t('numberColumnStatus')}</th>
						<th>{t('documentUpdated')}</th>
					</tr>
				</thead>
				<tbody>
					{#each documents as d (d.id)}
						<tr>
							<td><a href="/documents/view?id={d.id}" class="mono">{d.number}</a></td>
							<td>{d.document_type.name}</td>
							<td>{d.form_set_name}</td>
							<td>{d.row_count}</td>
							<td>
								<span class="badge" data-status={d.status}>
									{t(d.status !== 'released' ? 'versionStatusDraft' : d.released_migrated ? 'documentStatusReleasedMigrated' : 'documentStatusReleased')}
								</span>
							</td>
							<td>{formatDateTime(d.updated_at)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	{#if canCreate}
		<section>
			<h2>{t('documentCreateTitle')}</h2>
			<p class="hint">{t('documentCreateHint')}</p>
			<form onsubmit={create} novalidate>
				<div class="field">
					<label for="document-number">{t('documentNumber')}</label>
					<select id="document-number" bind:value={draft.numberId}>
						<option value="">{t('documentNumberChoose')}</option>
						{#each numbers as n (n.id)}
							<option value={n.id}>{n.number} · {n.document_type.name}</option>
						{/each}
					</select>
					{#if fieldErrors.document_number_id}<p class="error">{fieldErrors.document_number_id}</p>{/if}
					{#if fieldErrors.form_set}<p class="error">{fieldErrors.form_set}</p>{/if}
				</div>
				<fieldset>
					<legend>{t('documentModels')}</legend>
					{#each models as m (m.id)}
						<label><input type="checkbox" bind:group={draft.modelIds} value={m.id} /> {m.name}</label>
					{/each}
					{#if models.length === 0}<p class="hint">{t('documentNoModels')}</p>{/if}
					{#if fieldErrors.model_ids}<p class="error">{fieldErrors.model_ids}</p>{/if}
				</fieldset>
				{#if partChoices.length > 0}
					<fieldset>
						<legend>{t('documentPartNumbers')}</legend>
						{#each partChoices as p (p.id)}
							<label><input type="checkbox" bind:group={draft.partIds} value={p.id} /> {p.number}{p.name ? ` · ${p.name}` : ''}</label>
						{/each}
						{#if fieldErrors.part_number_ids}<p class="error">{fieldErrors.part_number_ids}</p>{/if}
					</fieldset>
				{/if}
				<p><button type="submit">{t('documentCreate')}</button></p>
			</form>
		</section>
	{/if}

	{#if canMigrate}
		<p class="hint"><a href="/migration">{t('documentsLegacyLink')}</a></p>
	{/if}
{/if}

<style>
	.chooser {
		max-width: 28rem;
		margin-bottom: 1.5rem;
	}

	fieldset {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1.25rem;
		margin: 1rem 0;
	}

	fieldset legend {
		padding: 0 4px;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
