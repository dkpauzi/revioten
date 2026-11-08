<script lang="ts">
	// Sumber perubahan (M1.6, R-CHG-06): daftar, tambah, ganti nama, dan
	// nonaktifkan. Sumber yang dipakai mekanisme sistem (yokoten, propagasi
	// template, SQAM) selalu aktif; server tetap menolak bila dicoba.
	import { createChangeSource, updateChangeSource } from '../api/client.ts';
	import type { ChangeSource } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		sources,
		canEdit,
		onsaved
	}: { sources: ChangeSource[]; canEdit: boolean; onsaved: () => void } = $props();

	let name = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editActive = $state(true);
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// save menyimpan formulir tambah (editingId kosong) atau baris yang diubah.
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createChangeSource(name)
				: await updateChangeSource(editingId, { name: editName, active: editActive });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'changeList')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('changeListSaved', { name: result.data.name });
		name = '';
		editingId = null;
		onsaved();
	}

	function startEdit(source: ChangeSource) {
		editingId = source.id;
		editName = source.name;
		editActive = source.active;
		fieldErrors = {};
		message = '';
	}
</script>

<section>
	<h2>{t('changeSourcesTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<form onsubmit={save} novalidate>
		<table>
			<thead>
				<tr>
					<th>{t('changeSourceName')}</th>
					<th>{t('changeSourceStatus')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each sources as source (source.id)}
					<tr>
						{#if editingId === source.id}
							<td>
								<input aria-label={t('changeSourceName')} bind:value={editName} autocomplete="off" />
								{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
							</td>
							<td>
								<!-- Sumber milik mekanisme sistem selalu aktif. -->
								<label>
									<input type="checkbox" bind:checked={editActive} disabled={source.mechanism !== undefined} />
									{t('changeSourceActive')}
								</label>
								{#if fieldErrors.active}<p class="error">{fieldErrors.active}</p>{/if}
							</td>
							<td>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</td>
						{:else}
							<td>{source.name}</td>
							<td>
								{source.active ? t('changeSourceActive') : t('changeSourceInactive')}
								{#if source.mechanism}· {t('changeSourceSystem')}{/if}
							</td>
							{#if canEdit}
								<td>
									<button type="button" onclick={() => startEdit(source)}>{t('customerEdit')}</button>
								</td>
							{/if}
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</form>

	{#if canEdit && editingId === null}
		<form class="add" onsubmit={save} novalidate>
			<div class="field">
				<label for="change-source-name">{t('changeSourceName')}</label>
				<input id="change-source-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<button type="submit">{t('changeSourceAdd')}</button>
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
</style>
