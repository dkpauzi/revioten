<script lang="ts">
	// Halaman master proses (M1.5, UC-04): ID tetap, nama tampilan, dan alias
	// (D-06, R-DOC-04). Semua akun bisa melihat; menambah dan mengubah hanya
	// untuk pemegang E-6, yaitu Pengelola template milik PPAP Engineer (D-64).
	import { onMount } from 'svelte';
	import { createProcess, listProcesses, updateProcess } from '#lib/api/client.ts';
	import type { ApiFailure } from '#lib/api/client.ts';
	import type { Process } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { aliasLines, aliasText } from '#lib/processes.ts';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let processes = $state<Process[] | null>(null);
	let loadError = $state('');
	// Tombol dan formulir hanya untuk pemegang E-6; server tetap memeriksa setiap operasi.
	let canEdit = $derived(can('E-6'));

	// Formulir tambah. Alias ditulis satu per baris di kotak teks.
	let name = $state('');
	let aliases = $state('');
	// Baris yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editAliases = $state('');

	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');

	async function load() {
		const result = await listProcesses();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		processes = result.data.items;
		loadError = '';
	}

	// showErrors menampilkan masalah per field dari server, atau pesan umum bila
	// masalahnya bukan pada satu field.
	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'process')]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	// save menyimpan formulir tambah (editingId kosong) atau baris yang diubah.
	async function save(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		failure = '';
		const result =
			editingId === null
				? await createProcess({ name, aliases: aliasLines(aliases) })
				: await updateProcess(editingId, { name: editName, aliases: aliasLines(editAliases) });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		message = t('processSaved', { name: result.data.name });
		name = '';
		aliases = '';
		editingId = null;
		await load();
	}

	function startEdit(process: Process) {
		editingId = process.id;
		editName = process.name;
		editAliases = aliasText(process.aliases);
		fieldErrors = {};
		message = '';
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('processesTitle')}</h1>
<p>{t('processesIntro')}</p>
{#if canEdit}<p><a href="/processes/import">{t('processesImportLink')}</a></p>{/if}

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if processes === null}
	<p>{t('processesLoading')}</p>
{:else}
	{#if message}
		<p role="status">{message}</p>
	{/if}
	{#if failure}
		<p class="error" role="alert">{failure}</p>
	{/if}

	{#if processes.length === 0}
		<p>{t('processesEmpty')}</p>
	{:else}
		<!-- Satu formulir untuk baris yang sedang diubah. -->
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('processName')}</th>
						<th>{t('processAliases')}</th>
						{#if canEdit}
							<th>{t('masterActions')}</th>
						{/if}
					</tr>
				</thead>
				<tbody>
					{#each processes as process (process.id)}
						<tr>
							{#if editingId === process.id}
								<td>
									<input aria-label={t('processName')} bind:value={editName} autocomplete="off" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</td>
								<td>
									<textarea aria-label={t('processAliases')} bind:value={editAliases} rows="3"></textarea>
									{#if fieldErrors.aliases}<p class="error">{fieldErrors.aliases}</p>{/if}
								</td>
								<td>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</td>
							{:else}
								<td>{process.name}</td>
								<td>
									<!-- Satu alias per baris, karena nama proses bisa memuat koma. -->
									<ul class="aliases">
										{#each process.aliases as alias (alias)}
											<li>{alias}</li>
										{/each}
									</ul>
								</td>
								{#if canEdit}
									<td>
										<button type="button" onclick={() => startEdit(process)}>{t('processEdit')}</button>
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
		<section>
			<h2>{t('processAddTitle')}</h2>
			<!-- novalidate: pemeriksaan input dilakukan server, supaya aturannya hanya satu. -->
			<form class="add" onsubmit={save} novalidate>
				<div class="field">
					<label for="process-name">{t('processName')}</label>
					<input id="process-name" bind:value={name} autocomplete="off" />
					{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
				</div>
				<div class="field">
					<label for="process-aliases">{t('processAliases')}</label>
					<textarea
						id="process-aliases"
						bind:value={aliases}
						rows="3"
						aria-describedby="process-aliases-hint"
					></textarea>
					<small id="process-aliases-hint">{t('processAliasesHint')}</small>
					{#if fieldErrors.aliases}<p class="error">{fieldErrors.aliases}</p>{/if}
				</div>
				<button type="submit">{t('processAdd')}</button>
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
		align-items: flex-start;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.add > button {
		margin-top: var(--label-space);
	}

	.field {
		max-width: 28rem;
	}

	.aliases {
		list-style: none;
		margin: 0;
		padding: 0;
	}
</style>
