<script lang="ts">
	// Halaman perubahan (M5.1, R-CHG-01, UC-40): daftar perubahan satu customer
	// dan formulir permintaan perubahan. Satu perubahan menunjuk satu atau
	// lebih dokumen terdampak. Pemicunya sumber perubahan biasa (PP, customer
	// claim, ECR, temuan audit); yokoten, propagasi template, dan SQAM dimulai
	// sistem (D-113). Server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import { createChange, listChangeSources, listChangeTypes, listChanges, listCustomers, listDocuments, listFourMCategories } from '#lib/api/client.ts';
	import type { Change, ChangeSource, ChangeType, Customer, DocumentSummary, FourMCategory } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import SQAMForm from '#lib/changes/SQAMForm.svelte';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let changes = $state<Change[] | null>(null);
	let documents = $state<DocumentSummary[]>([]);
	let triggers = $state<ChangeSource[]>([]);
	let types = $state<ChangeType[]>([]);
	let categories = $state<FourMCategory[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let canPropose = $derived(can('E-3'));

	let draft = $state({ triggerId: '', changeTypeId: '', fourMId: '', reason: '', reference: '', documentIds: [] as string[] });

	onMount(async () => {
		const [c, s, ty, f] = await Promise.all([listCustomers(), listChangeSources(), listChangeTypes(), listFourMCategories()]);
		if (!c.ok) {
			loadError = failureText(c.error);
			return;
		}
		customers = c.data.items.filter((x) => x.active);
		// Hanya pemicu biasa yang aktif; sumber mekanisme dipakai sistem (D-113).
		triggers = s.ok ? s.data.items.filter((x) => x.active && !x.mechanism) : [];
		types = ty.ok ? ty.data.items : [];
		categories = f.ok ? f.data.items : [];
	});

	async function chooseCustomer() {
		changes = null;
		documents = [];
		draft = { ...draft, documentIds: [] };
		message = '';
		failure = '';
		if (customerId === '') {
			return;
		}
		const [list, docs] = await Promise.all([listChanges(customerId), listDocuments(customerId)]);
		if (!list.ok) {
			loadError = failureText(list.error);
			return;
		}
		loadError = '';
		changes = list.data.items;
		documents = docs.ok ? docs.data.items : [];
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		fieldErrors = {};
		const result = await createChange({
			customer_id: customerId,
			trigger_id: draft.triggerId,
			change_type_id: draft.changeTypeId,
			four_m_category_id: draft.fourMId || undefined,
			reason: draft.reason,
			source_reference: draft.reference || undefined,
			document_ids: draft.documentIds
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('changeCreated', { ref: result.data.ref });
		draft = { triggerId: '', changeTypeId: '', fourMId: '', reason: '', reference: '', documentIds: [] };
		await chooseCustomer();
		message = t('changeCreated', { ref: result.data.ref });
	}
</script>

<h1>{t('changesTitle')}</h1>
<p>{t('changesIntro')}</p>
{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

<div class="field chooser">
	<label for="changes-customer">{t('documentCustomer')}</label>
	<select id="changes-customer" bind:value={customerId} onchange={chooseCustomer}>
		<option value="">{t('numberCustomerChoose')}</option>
		{#each customers as c (c.id)}
			<option value={c.id}>{c.code} · {c.name}</option>
		{/each}
	</select>
</div>

{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if customerId && changes}
	{#if canPropose}
		<section>
			<h2>{t('changeNewTitle')}</h2>
			<form class="grid" onsubmit={submit} novalidate>
				<div class="field">
					<label for="change-trigger">{t('changeTrigger')}</label>
					<select id="change-trigger" bind:value={draft.triggerId}>
						<option value="">{t('changeChoose')}</option>
						{#each triggers as s (s.id)}<option value={s.id}>{s.name}</option>{/each}
					</select>
					{#if fieldErrors.trigger_id}<p class="error">{fieldErrors.trigger_id}</p>{/if}
				</div>
				<div class="field">
					<label for="change-type">{t('changeType')}</label>
					<select id="change-type" bind:value={draft.changeTypeId}>
						<option value="">{t('changeChoose')}</option>
						{#each types as ty (ty.id)}<option value={ty.id}>{ty.name}</option>{/each}
					</select>
					{#if fieldErrors.change_type_id}<p class="error">{fieldErrors.change_type_id}</p>{/if}
				</div>
				<div class="field">
					<label for="change-4m">{t('changeFourM')}</label>
					<select id="change-4m" bind:value={draft.fourMId}>
						<option value="">{t('changeFourMNone')}</option>
						{#each categories as f (f.id)}<option value={f.id}>{f.name}</option>{/each}
					</select>
				</div>
				<div class="field">
					<label for="change-reference">{t('changeReference')}</label>
					<input id="change-reference" bind:value={draft.reference} maxlength="200" autocomplete="off" />
					<p class="hint">{t('changeReferenceHint')}</p>
				</div>
				<div class="field wide">
					<label for="change-reason">{t('changeReason')}</label>
					<textarea id="change-reason" rows="3" maxlength="2000" bind:value={draft.reason}></textarea>
					{#if fieldErrors.reason}<p class="error">{fieldErrors.reason}</p>{/if}
				</div>
				<fieldset class="wide">
					<legend>{t('changeDocuments')}</legend>
					{#if documents.length === 0}<p class="hint">{t('changeNoDocuments')}</p>{/if}
					{#each documents as d (d.id)}
						<label class="check"><input type="checkbox" value={d.id} bind:group={draft.documentIds} /> <span class="mono">{d.number}</span> · {d.document_type.name} · {d.models.join(', ')}</label>
					{/each}
					{#if fieldErrors.document_ids}<p class="error">{fieldErrors.document_ids}</p>{/if}
				</fieldset>
				<p class="wide"><button type="submit" class="primary">{t('changeSubmit')}</button></p>
			</form>
		</section>
		{#key customerId}
			<SQAMForm
				{customerId}
				{types}
				{categories}
				onsubmitted={(c) => {
					changes = [c, ...(changes ?? [])];
					message = t('sqamSubmitted', { ref: c.ref, count: String(c.documents.length) });
				}}
			/>
		{/key}
	{/if}

	<section>
		<h2>{t('changeListTitle')}</h2>
		{#if changes.length === 0}
			<p>{t('changeNone')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>{t('changeRef')}</th>
						<th>{t('changeReason')}</th>
						<th>{t('changeTrigger')}</th>
						<th>{t('changeDocuments')}</th>
						<th>{t('numberColumnStatus')}</th>
						<th>{t('changeRequestedBy')}</th>
					</tr>
				</thead>
				<tbody>
					{#each changes as c (c.id)}
						<tr>
							<td class="mono"><a href="/changes/view?id={c.id}">{c.ref}</a></td>
							<td>{c.reason}</td>
							<td>{c.trigger?.name ?? ''}{c.source_reference ? ` · ${c.source_reference}` : ''}</td>
							<td class="mono">{c.documents.map((d) => d.number).join(', ')}</td>
							<td><span class="badge" data-change-status={c.status}>{t(`changeStatus_${c.status}` as 'changeStatus_submitted')}</span></td>
							<td>{c.requested_by}<small>{formatDateTime(c.created_at)}</small></td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>
{/if}

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(14rem, 1fr));
		gap: 0 1rem;
	}

	.wide {
		grid-column: 1 / -1;
	}

	textarea {
		width: 100%;
	}

	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 8px 12px;
		margin-bottom: 1rem;
	}

	.check {
		display: flex;
		gap: 8px;
		align-items: center;
		padding: 2px 0;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	td small {
		display: block;
		color: var(--text-3);
	}
</style>
