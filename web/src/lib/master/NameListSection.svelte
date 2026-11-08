<script lang="ts">
	// Daftar tetap yang hanya bisa diganti namanya (M1.6): jenis perubahan dan
	// kategori 4M. Entrinya tidak bisa ditambah atau dihapus, karena kode program
	// mengenali masing-masing lewat key-nya.
	import type { ApiResult } from '../api/client.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		title,
		items,
		canEdit,
		rename,
		onsaved
	}: {
		title: string;
		items: { id: string; name: string }[];
		canEdit: boolean;
		// rename menyimpan nama baru satu entri lewat API.
		rename: (id: string, name: string) => Promise<ApiResult<{ name: string }>>;
		onsaved: () => void;
	} = $props();

	let editingId = $state<string | null>(null);
	let editName = $state('');
	let fieldError = $state('');
	let message = $state('');
	let failure = $state('');

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (editingId === null) {
			return;
		}
		fieldError = '';
		message = '';
		failure = '';
		const result = await rename(editingId, editName);
		if (!result.ok) {
			const detail = result.error.body?.details?.[0];
			if (detail) {
				fieldError = problemText(detail, 'changeList');
			} else {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('changeListSaved', { name: result.data.name });
		editingId = null;
		onsaved();
	}

	function startRename(item: { id: string; name: string }) {
		editingId = item.id;
		editName = item.name;
		fieldError = '';
		message = '';
	}
</script>

<section>
	<h2>{title}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<form onsubmit={save} novalidate>
		<table>
			<thead>
				<tr>
					<th>{t('changeListName')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each items as item (item.id)}
					<tr>
						{#if editingId === item.id}
							<td>
								<input aria-label={t('changeListName')} bind:value={editName} autocomplete="off" />
								{#if fieldError}<p class="error">{fieldError}</p>{/if}
							</td>
							<td>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</td>
						{:else}
							<td>{item.name}</td>
							{#if canEdit}
								<td>
									<button type="button" onclick={() => startRename(item)}>{t('changeListRename')}</button>
								</td>
							{/if}
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</form>
</section>

<style>
	section {
		margin-bottom: 2rem;
	}
</style>
