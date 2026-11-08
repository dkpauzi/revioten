<script lang="ts">
	// Kode khusus di posisi kode customer (M1.6, E-12), misalnya ZZ untuk
	// General/Common. Kode tidak bisa diubah setelah dibuat, hanya artinya.
	import { createSpecialCode, updateSpecialCode } from '../api/client.ts';
	import type { SpecialCode } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	let {
		codes,
		canEdit,
		onsaved
	}: { codes: SpecialCode[]; canEdit: boolean; onsaved: () => void } = $props();

	let code = $state('');
	let name = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	// save menyimpan formulir tambah (editingId kosong) atau nama baru satu kode.
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createSpecialCode({ code, name })
				: await updateSpecialCode(editingId, editName);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'specialCode')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('specialCodeSaved', { code: result.data.code });
		code = '';
		name = '';
		editingId = null;
		onsaved();
	}

	function startEdit(special: SpecialCode) {
		editingId = special.id;
		editName = special.name;
		fieldErrors = {};
		message = '';
	}
</script>

<section>
	<h2>{t('specialCodesTitle')}</h2>
	<p>{t('specialCodesIntro')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<form onsubmit={save} novalidate>
		<table>
			<thead>
				<tr>
					<th>{t('specialCode')}</th>
					<th>{t('specialCodeName')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each codes as special (special.id)}
					<tr>
						<td>{special.code}</td>
						{#if editingId === special.id}
							<td>
								<input aria-label={t('specialCodeName')} bind:value={editName} autocomplete="off" />
								{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
							</td>
							<td>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</td>
						{:else}
							<td>{special.name}</td>
							{#if canEdit}
								<td>
									<button type="button" onclick={() => startEdit(special)}>{t('customerEdit')}</button>
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
				<label for="special-code">{t('specialCode')}</label>
				<input id="special-code" bind:value={code} autocomplete="off" size="3" />
				{#if fieldErrors.code}<p class="error">{fieldErrors.code}</p>{/if}
			</div>
			<div class="field">
				<label for="special-code-name">{t('specialCodeName')}</label>
				<input id="special-code-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<button type="submit">{t('specialCodeAdd')}</button>
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
