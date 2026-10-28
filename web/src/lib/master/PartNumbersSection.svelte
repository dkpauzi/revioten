<script lang="ts">
	// Part number satu customer (M1.4): daftar, tambah, dan ubah nomor serta nama.
	// Setiap part number milik satu model (ASUMSI E-30).
	import { createPartNumber, updatePartNumber } from '../api/client.ts';
	import type { Model, PartNumber } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		parts,
		models,
		canEdit,
		onsaved
	}: { parts: PartNumber[]; models: Model[]; canEdit: boolean; onsaved: () => void } = $props();

	let modelId = $state('');
	let number = $state('');
	let name = $state('');
	let editingId = $state<string | null>(null);
	let editNumber = $state('');
	let editName = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// save menyimpan formulir tambah (editingId kosong) atau part number yang
	// diubah. Nama kosong dikirim sebagai field yang tidak ada.
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createPartNumber({ model_id: modelId, number, name: name || undefined })
				: await updatePartNumber(editingId, { number: editNumber, name: editName || undefined });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'partNumber')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('partNumberSaved', { number: result.data.number });
		number = '';
		name = '';
		editingId = null;
		onsaved();
	}

	function startEdit(part: PartNumber) {
		editingId = part.id;
		editNumber = part.number;
		editName = part.name ?? '';
		fieldErrors = {};
	}
</script>

<section>
	<h2>{t('partNumbersTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if parts.length === 0}
		<p>{t('partNumbersEmpty')}</p>
	{:else}
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('partNumberModel')}</th>
						<th>{t('partNumber')}</th>
						<th>{t('partName')}</th>
						{#if canEdit}<th>{t('masterActions')}</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each parts as part (part.id)}
						<tr>
							<td>{part.model.name}</td>
							{#if editingId === part.id}
								<td>
									<input aria-label={t('partNumber')} bind:value={editNumber} autocomplete="off" />
									{#if fieldErrors.number}<p class="error">{fieldErrors.number}</p>{/if}
								</td>
								<td>
									<input aria-label={t('partName')} bind:value={editName} autocomplete="off" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</td>
								<td>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</td>
							{:else}
								<td>{part.number}</td>
								<td>{part.name ?? ''}</td>
								{#if canEdit}
									<td><button type="button" onclick={() => startEdit(part)}>{t('partNumberEdit')}</button></td>
								{/if}
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</form>
	{/if}

	{#if canEdit && editingId === null}
		<form class="add" onsubmit={save} novalidate>
			<div class="field">
				<label for="part-model">{t('partNumberModel')}</label>
				<select id="part-model" bind:value={modelId}>
					<option value="">{t('partNumberModelChoose')}</option>
					{#each models as model (model.id)}
						<option value={model.id}>{model.name}</option>
					{/each}
				</select>
				{#if fieldErrors.model_id}<p class="error">{fieldErrors.model_id}</p>{/if}
			</div>
			<div class="field">
				<label for="part-number">{t('partNumber')}</label>
				<input id="part-number" bind:value={number} autocomplete="off" />
				{#if fieldErrors.number}<p class="error">{fieldErrors.number}</p>{/if}
			</div>
			<div class="field">
				<label for="part-name">{t('partName')}</label>
				<input id="part-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<button type="submit">{t('partNumberAdd')}</button>
		</form>
	{/if}
</section>

<style>
	section {
		margin-bottom: 2rem;
	}

	.add {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
		margin-top: 1rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.error {
		color: #b00020;
		margin: 0;
	}

	table {
		border-collapse: collapse;
	}

	th,
	td {
		border: 1px solid #ccc;
		padding: 0.25rem 0.75rem;
		text-align: left;
		vertical-align: top;
	}
</style>
