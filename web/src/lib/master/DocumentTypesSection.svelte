<script lang="ts">
	// Jenis dokumen dan hurufnya (M1.6, E-12): daftar dengan awalan nomor,
	// tambah jenis lain dari WI, dan ganti nama atau huruf. PFD, PFMEA, dan CP
	// ditandai sebagai dokumen yang dibuat sistem ini.
	import { createDocumentType, updateDocumentType } from '../api/client.ts';
	import type { DocumentType } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		types,
		canEdit,
		onsaved
	}: { types: DocumentType[]; canEdit: boolean; onsaved: () => void } = $props();

	let name = $state('');
	let letter = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editLetter = $state('');
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
				? await createDocumentType({ name, letter })
				: await updateDocumentType(editingId, { name: editName, letter: editLetter });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'documentType')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('documentTypeSaved', { name: result.data.name });
		name = '';
		letter = '';
		editingId = null;
		onsaved();
	}

	function startEdit(type: DocumentType) {
		editingId = type.id;
		editName = type.name;
		editLetter = type.letter;
		fieldErrors = {};
		message = '';
	}
</script>

<section>
	<h2>{t('documentTypesListTitle')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<form onsubmit={save} novalidate>
		<table>
			<thead>
				<tr>
					<th>{t('documentTypeName')}</th>
					<th>{t('documentTypeLetter')}</th>
					<th>{t('documentTypePrefix')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each types as type (type.id)}
					<tr>
						{#if editingId === type.id}
							<td>
								<input aria-label={t('documentTypeName')} bind:value={editName} autocomplete="off" />
								{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
							</td>
							<td>
								<input aria-label={t('documentTypeLetter')} bind:value={editLetter} autocomplete="off" size="2" />
								{#if fieldErrors.letter}<p class="error">{fieldErrors.letter}</p>{/if}
							</td>
							<td>{type.prefix}</td>
							<td>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</td>
						{:else}
							<td>
								{type.name}
								{#if type.key}<small>· {t('documentTypeBuilt')}</small>{/if}
							</td>
							<td>{type.letter}</td>
							<td>{type.prefix}</td>
							{#if canEdit}
								<td>
									<button type="button" onclick={() => startEdit(type)}>{t('customerEdit')}</button>
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
				<label for="document-type-name">{t('documentTypeName')}</label>
				<input id="document-type-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<div class="field">
				<label for="document-type-letter">{t('documentTypeLetter')}</label>
				<input id="document-type-letter" bind:value={letter} autocomplete="off" size="2" />
				{#if fieldErrors.letter}<p class="error">{fieldErrors.letter}</p>{/if}
			</div>
			<button type="submit">{t('documentTypeAdd')}</button>
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
