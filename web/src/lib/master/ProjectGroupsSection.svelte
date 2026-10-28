<script lang="ts">
	// Bagian Project Group di halaman Customers (M1.4, D-54): daftar untuk semua
	// akun, tambah dan ubah hanya untuk pemegang A-2 (E-10).
	import { createProjectGroup, updateProjectGroup } from '../api/client.ts';
	import type { ProjectGroup } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';

	// groups adalah daftar dari halaman. onsaved dipanggil setelah ada yang
	// tersimpan, supaya halaman memuat ulang daftar (juga untuk pilihan di formulir customer).
	let {
		groups,
		canEdit,
		onsaved
	}: { groups: ProjectGroup[]; canEdit: boolean; onsaved: () => void } = $props();

	// Formulir tambah.
	let code = $state('');
	let name = $state('');
	// Baris yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let editCode = $state('');
	let editName = $state('');

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
				? await createProjectGroup({ code, name })
				: await updateProjectGroup(editingId, { code: editCode, name: editName });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			if (details.length > 0) {
				fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'projectGroup')]));
			} else {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('projectGroupSaved', { code: result.data.code });
		code = '';
		name = '';
		editingId = null;
		onsaved();
	}

	function startEdit(group: ProjectGroup) {
		editingId = group.id;
		editCode = group.code;
		editName = group.name;
		fieldErrors = {};
	}
</script>

<section>
	<h2>{t('projectGroupsTitle')}</h2>
	{#if message}
		<p role="status">{message}</p>
	{/if}
	{#if failure}
		<p class="error" role="alert">{failure}</p>
	{/if}

	{#if groups.length === 0}
		<p>{t('projectGroupsEmpty')}</p>
	{:else}
		<!-- Satu formulir untuk baris yang sedang diubah, supaya tombol Enter menyimpan. -->
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('projectGroupCode')}</th>
						<th>{t('projectGroupName')}</th>
						{#if canEdit}
							<th>{t('masterActions')}</th>
						{/if}
					</tr>
				</thead>
				<tbody>
					{#each groups as group (group.id)}
						<tr>
							{#if editingId === group.id}
								<td>
									<input aria-label={t('projectGroupCode')} bind:value={editCode} autocomplete="off" />
									{#if fieldErrors.code}<p class="error">{fieldErrors.code}</p>{/if}
								</td>
								<td>
									<input aria-label={t('projectGroupName')} bind:value={editName} autocomplete="off" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</td>
								<td>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</td>
							{:else}
								<td>{group.code}</td>
								<td>{group.name}</td>
								{#if canEdit}
									<td>
										<button type="button" onclick={() => startEdit(group)}>{t('customerEdit')}</button>
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
				<label for="project-group-code">{t('projectGroupCode')}</label>
				<input id="project-group-code" bind:value={code} autocomplete="off" aria-describedby="project-group-code-hint" />
				<small id="project-group-code-hint">{t('projectGroupCodeHint')}</small>
				{#if fieldErrors.code}<p class="error">{fieldErrors.code}</p>{/if}
			</div>
			<div class="field">
				<label for="project-group-name">{t('projectGroupName')}</label>
				<input id="project-group-name" bind:value={name} autocomplete="off" />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<button type="submit">{t('projectGroupAdd')}</button>
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
