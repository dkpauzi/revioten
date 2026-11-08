<script lang="ts">
	// Group model satu customer (M1.4, D-39, D-42): daftar, tambah, dan ganti nama.
	import { createModelGroup, renameModelGroup } from '../api/client.ts';
	import type { ModelGroup } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		customerId,
		groups,
		canEdit,
		onsaved
	}: { customerId: string; groups: ModelGroup[]; canEdit: boolean; onsaved: () => void } = $props();

	let name = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// save menyimpan formulir tambah (editingId kosong) atau nama baru satu group.
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createModelGroup({ customer_id: customerId, name })
				: await renameModelGroup(editingId, editName);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'modelGroup')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('modelGroupSaved', { name: result.data.name });
		name = '';
		editingId = null;
		onsaved();
	}

	function startRename(group: ModelGroup) {
		editingId = group.id;
		editName = group.name;
		fieldErrors = {};
	}
</script>

<section>
	<h2>{t('modelGroupsTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if groups.length === 0}
		<p>{t('modelGroupsEmpty')}</p>
	{:else}
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('modelGroupName')}</th>
						{#if canEdit}<th>{t('masterActions')}</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each groups as group (group.id)}
						<tr>
							{#if editingId === group.id}
								<td>
									<input aria-label={t('modelGroupName')} bind:value={editName} autocomplete="off" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</td>
								<td>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</td>
							{:else}
								<td>{group.name}</td>
								{#if canEdit}
									<td>
										<button type="button" onclick={() => startRename(group)}>
											{t('modelGroupRename')}
										</button>
									</td>
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
				<label for="model-group-name">{t('modelGroupName')}</label>
				<input id="model-group-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
				{#if fieldErrors.customer_id}<p class="error">{fieldErrors.customer_id}</p>{/if}
			</div>
			<button type="submit">{t('modelGroupAdd')}</button>
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
		align-items: flex-start;
		margin-top: 1rem;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.add > button {
		margin-top: var(--label-space);
	}
</style>
