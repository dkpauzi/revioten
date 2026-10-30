<script lang="ts">
	// Halaman jabatan (M1.8, UC-02): jabatan sesuai bagan organisasi NPI
	// beserta pemegangnya. Project Leader dan Project Engineer dipegang satu
	// orang per Project Group (D-54), dan satu orang boleh merangkap (D-58).
	// Semua akun bisa melihat; mengubah hanya untuk pemegang A-1.
	import { onMount } from 'svelte';
	import {
		addPositionHolder,
		listAccounts,
		listPositions,
		listProjectGroups,
		removePositionHolder,
		updatePosition
	} from '#lib/api/client.ts';
	import type { PersonInfo, Position, PositionHolder, ProjectGroup } from '#lib/api/gen/index.ts';
	import { peopleOf } from '#lib/accounts.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let positions = $state<Position[] | null>(null);
	let groups = $state<ProjectGroup[]>([]);
	let people = $state<PersonInfo[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('A-1'));

	// Ganti nama satu jabatan.
	let editingId = $state<string | null>(null);
	let editName = $state('');
	// Formulir tambah pemegang.
	let positionId = $state('');
	let personId = $state('');
	let groupId = $state('');
	let perGroup = $derived(positions?.find((p) => p.id === positionId)?.per_project_group ?? false);

	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	async function load() {
		const [positionResult, groupResult] = await Promise.all([listPositions(), listProjectGroups()]);
		if (!positionResult.ok) {
			loadError = failureText(positionResult.error);
			return;
		}
		if (!groupResult.ok) {
			loadError = failureText(groupResult.error);
			return;
		}
		// Daftar orang diambil dari daftar akun, yang hanya terbuka untuk A-1.
		if (canEdit) {
			const accountResult = await listAccounts();
			if (!accountResult.ok) {
				loadError = failureText(accountResult.error);
				return;
			}
			people = peopleOf(accountResult.data.items);
		}
		positions = positionResult.data.items;
		groups = groupResult.data.items;
		loadError = '';
	}

	function reset() {
		fieldErrors = {};
		message = '';
		failure = '';
	}

	// showErrors menampilkan masalah per field dari server, atau pesan umum.
	function showErrors(error: Parameters<typeof failureText>[0], scope: 'position' | 'holder') {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, scope)]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	async function saveName(event: SubmitEvent) {
		event.preventDefault();
		if (editingId === null) {
			return;
		}
		reset();
		const result = await updatePosition(editingId, editName);
		if (!result.ok) {
			showErrors(result.error, 'position');
			return;
		}
		message = t('positionSaved', { name: result.data.name });
		editingId = null;
		await load();
	}

	async function addHolder(event: SubmitEvent) {
		event.preventDefault();
		reset();
		const position = positions?.find((p) => p.id === positionId);
		// Jabatan adalah bagian alamat API, jadi harus dipilih dulu.
		if (!position) {
			fieldErrors = { position_id: t('holderPositionRequired') };
			return;
		}
		const result = await addPositionHolder(positionId, {
			person_id: personId,
			project_group_id: groupId === '' ? undefined : groupId
		});
		if (!result.ok) {
			showErrors(result.error, 'holder');
			return;
		}
		message = t('holderAdded', { name: result.data.person.name, position: position.name });
		personId = '';
		groupId = '';
		await load();
	}

	async function removeHolder(position: Position, holder: PositionHolder) {
		reset();
		const result = await removePositionHolder(position.id, holder.id);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		message = t('holderRemoved', { name: holder.person.name, position: position.name });
		await load();
	}

	// holderLabel menulis pemegang, dengan kode Project Group bila ada.
	function holderLabel(holder: PositionHolder): string {
		return holder.project_group ? `${holder.project_group.code}: ${holder.person.name}` : holder.person.name;
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('positionsTitle')}</h1>
<p>{t('positionsIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if positions === null}
	<p>{t('positionsLoading')}</p>
{:else}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<form onsubmit={saveName} novalidate>
		<table>
			<thead>
				<tr>
					<th>{t('positionName')}</th>
					<th>{t('positionHolders')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each positions as position (position.id)}
					<tr>
						{#if editingId === position.id}
							<td>
								<input aria-label={t('positionName')} bind:value={editName} autocomplete="off" />
								{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
							</td>
						{:else}
							<td>
								{position.name}
								<!-- Aturan jumlah pemegang (D-54, D-115); server tetap menegakkannya. -->
								{#if position.single_holder}
									<br /><small>{t('positionSingleHolder')}</small>
								{:else if position.per_project_group}
									<br /><small>{t('positionPerGroup')}</small>
								{/if}
							</td>
						{/if}
						<td>
							{#if position.holders.length === 0}
								{t('positionHoldersNone')}
							{:else}
								<ul>
									{#each position.holders as holder (holder.id)}
										<li>
											{holderLabel(holder)}
											{#if canEdit}
												<button
													type="button"
													aria-label={t('holderRemove', { name: holderLabel(holder) })}
													onclick={() => removeHolder(position, holder)}>✕</button
												>
											{/if}
										</li>
									{/each}
								</ul>
							{/if}
						</td>
						{#if canEdit}
							<td>
								{#if editingId === position.id}
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								{:else}
									<button
										type="button"
										onclick={() => {
											editingId = position.id;
											editName = position.name;
											reset();
										}}>{t('positionRename')}</button
									>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</form>

	{#if canEdit}
		<section>
			<h2>{t('holderAdd')}</h2>
			<form class="add" onsubmit={addHolder} novalidate>
				<div class="field">
					<label for="holder-position">{t('positionName')}</label>
					<select id="holder-position" bind:value={positionId}>
						<option value="">{t('holderPositionChoose')}</option>
						{#each positions as position (position.id)}
							<option value={position.id}>{position.name}</option>
						{/each}
					</select>
					{#if fieldErrors.position_id}<p class="error">{fieldErrors.position_id}</p>{/if}
				</div>
				<div class="field">
					<label for="holder-person">{t('holderPerson')}</label>
					<select id="holder-person" bind:value={personId}>
						<option value="">{t('holderPersonChoose')}</option>
						{#each people as person (person.id)}
							<option value={person.id}>{person.name}</option>
						{/each}
					</select>
					{#if fieldErrors.person_id}<p class="error">{fieldErrors.person_id}</p>{/if}
				</div>
				{#if perGroup}
					<div class="field">
						<label for="holder-group">{t('holderGroup')}</label>
						<select id="holder-group" bind:value={groupId}>
							<option value="">{t('holderGroupChoose')}</option>
							{#each groups as group (group.id)}
								<option value={group.id}>{group.code} · {group.name}</option>
							{/each}
						</select>
					</div>
				{/if}
				{#if fieldErrors.project_group_id}<p class="error">{fieldErrors.project_group_id}</p>{/if}
				<button type="submit">{t('holderAdd')}</button>
			</form>
		</section>
	{/if}
{/if}

<style>
	section {
		margin-top: 2rem;
	}

	.add {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
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

	ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
</style>
