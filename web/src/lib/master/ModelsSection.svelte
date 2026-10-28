<script lang="ts">
	// Model satu customer (M1.4): daftar, tambah, dan ubah nama serta group.
	// Satu model paling banyak masuk satu group milik customer yang sama (D-42).
	import { createModel, updateModel } from '../api/client.ts';
	import type { Model, ModelGroup } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		customerId,
		models,
		groups,
		canEdit,
		onsaved
	}: {
		customerId: string;
		models: Model[];
		groups: ModelGroup[];
		canEdit: boolean;
		onsaved: () => void;
	} = $props();

	let name = $state('');
	// Pilihan group: '' berarti tanpa group.
	let groupId = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editGroupId = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// save menyimpan formulir tambah (editingId kosong) atau model yang diubah.
	// Group kosong dikirim sebagai field yang tidak ada (model tanpa group).
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createModel({ customer_id: customerId, name, model_group_id: groupId || undefined })
				: await updateModel(editingId, { name: editName, model_group_id: editGroupId || undefined });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'model')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('modelSaved', { name: result.data.name });
		name = '';
		groupId = '';
		editingId = null;
		onsaved();
	}

	function startEdit(model: Model) {
		editingId = model.id;
		editName = model.name;
		editGroupId = model.model_group?.id ?? '';
		fieldErrors = {};
	}
</script>

<section>
	<h2>{t('modelsListTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if models.length === 0}
		<p>{t('modelsEmpty')}</p>
	{:else}
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('modelName')}</th>
						<th>{t('modelGroup')}</th>
						{#if canEdit}<th>{t('masterActions')}</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each models as model (model.id)}
						<tr>
							{#if editingId === model.id}
								<td>
									<input aria-label={t('modelName')} bind:value={editName} autocomplete="off" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</td>
								<td>
									<select aria-label={t('modelGroup')} bind:value={editGroupId}>
										<option value="">{t('modelGroupNone')}</option>
										{#each groups as group (group.id)}
											<option value={group.id}>{group.name}</option>
										{/each}
									</select>
									{#if fieldErrors.model_group_id}<p class="error">{fieldErrors.model_group_id}</p>{/if}
								</td>
								<td>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</td>
							{:else}
								<td>{model.name}</td>
								<td>{model.model_group?.name ?? t('modelGroupNone')}</td>
								{#if canEdit}
									<td><button type="button" onclick={() => startEdit(model)}>{t('modelEdit')}</button></td>
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
				<label for="model-name">{t('modelName')}</label>
				<input id="model-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
				{#if fieldErrors.customer_id}<p class="error">{fieldErrors.customer_id}</p>{/if}
			</div>
			<div class="field">
				<label for="model-group">{t('modelGroup')}</label>
				<select id="model-group" bind:value={groupId}>
					<option value="">{t('modelGroupNone')}</option>
					{#each groups as group (group.id)}
						<option value={group.id}>{group.name}</option>
					{/each}
				</select>
				{#if fieldErrors.model_group_id}<p class="error">{fieldErrors.model_group_id}</p>{/if}
			</div>
			<button type="submit">{t('modelAdd')}</button>
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
