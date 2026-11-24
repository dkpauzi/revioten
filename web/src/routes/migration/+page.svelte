<script lang="ts">
	// Menu migrasi dokumen lama (D-215, D-216). Satu halaman untuk seluruh
	// alur: prasyarat, satu set dokumen lama dari nomor aslinya, dan papan
	// status sampai Released (migrated). Isi dokumen tetap diketik di halaman
	// dokumen, dan dicocokkan dengan hardcopy oleh orang lain (M7.5).
	import { onMount } from 'svelte';
	import { getMigrationOverview, listCustomers, listDocumentTypes, listMigratableNumbers, listModels, listPartNumbers, migrateSet } from '#lib/api/client.ts';
	import type { Customer, MigrationOverview, Model, PartNumber } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { emptyRows, fillNames, followingNumber, revisionText, setDocuments, splitLegacyNumber, type SetRow, type TypeKey } from '#lib/migration.ts';
	import { can } from '#lib/session.svelte.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let prefixes = $state<Record<string, string>>({});
	let typeNames = $state<Record<string, string>>({});
	let overview = $state<MigrationOverview | null>(null);
	let models = $state<Model[]>([]);
	let parts = $state<PartNumber[]>([]);
	// Revisi nomor yang sudah terdaftar tanpa dokumen, misalnya dari nomor lama
	// yang diimpor sebagai SC-TR-020-03 (I4). Dipakai mengisi kolom revisi.
	let knownRevisions = $state<Record<string, number>>({});
	let loadError = $state('');
	let canMigrate = $derived(can('E-10'));

	let oneSet = $state(true);
	let rows = $state<SetRow[]>(emptyRows());
	let modelIds = $state<string[]>([]);
	let partIds = $state<string[]>([]);
	let errors = $state<Record<string, string>>({});
	let message = $state('');
	let failure = $state('');
	let busy = $state(false);

	let partChoices = $derived(parts.filter((p) => modelIds.includes(p.model.id)));
	// Nomor sumber satu set: baris bernomor pertama.
	let source = $derived(rows.find((r) => r.number.trim() !== ''));
	let ready = $derived(
		overview !== null && overview.migration_mode && overview.form_set_published && overview.models > 0 && overview.processes > 0
	);

	onMount(async () => {
		const [customerResult, typeResult] = await Promise.all([listCustomers(), listDocumentTypes()]);
		if (!customerResult.ok) {
			loadError = failureText(customerResult.error);
			return;
		}
		customers = customerResult.data.items.filter((c) => c.active);
		if (typeResult.ok) {
			for (const d of typeResult.data.items) {
				if (d.key) {
					prefixes[d.key] = d.prefix;
					typeNames[d.key] = d.name;
				}
			}
		}
	});

	async function chooseCustomer() {
		overview = null;
		message = '';
		failure = '';
		resetForm();
		if (customerId === '') {
			return;
		}
		await refresh();
	}

	async function refresh() {
		const [viewResult, modelResult, partResult, numberResult] = await Promise.all([
			getMigrationOverview(customerId),
			listModels(customerId),
			listPartNumbers(customerId),
			listMigratableNumbers(customerId)
		]);
		if (!viewResult.ok) {
			loadError = failureText(viewResult.error);
			return;
		}
		loadError = '';
		overview = viewResult.data;
		models = modelResult.ok ? modelResult.data.items.filter((m) => !m.eol_at) : [];
		parts = partResult.ok ? partResult.data.items : [];
		knownRevisions = numberResult.ok ? Object.fromEntries(numberResult.data.items.map((n) => [n.number, n.revision])) : {};
	}

	function resetForm() {
		rows = emptyRows();
		modelIds = [];
		partIds = [];
		errors = {};
		oneSet = true;
	}

	// splitNumber memisahkan akhiran revisi yang ikut diketik, misalnya
	// SH-TR-020-02 menjadi nomor SH-TR-020 dan revisi 2.
	function splitNumber(row: SetRow) {
		const parts = splitLegacyNumber(row.number);
		row.number = parts.number;
		if (parts.revision !== null && revisionText(row.revision) === '') {
			row.revision = String(parts.revision);
		}
		fillKnownRevisions();
	}

	// fillKnownRevisions mengisi kolom revisi yang masih kosong dengan revisi
	// nomor itu di register (I4, opsi b), juga untuk nomor yang mengikuti
	// (satu set). ASUMSI E-146: revisi 0 tidak diisi, karena sama dengan nomor
	// tanpa revisi.
	// Isinya tetap bisa diubah.
	function fillKnownRevisions() {
		for (const row of rows) {
			const number = row.number.trim() || placeholder(row.key);
			const known = knownRevisions[number];
			if (known !== undefined && known > 0 && revisionText(row.revision) === '') {
				row.revision = String(known);
			}
		}
	}

	// placeholder memberi contoh nomor yang mengikuti (satu set).
	function placeholder(key: TypeKey): string {
		if (!oneSet || !source || source.key === key) {
			return '';
		}
		return followingNumber(splitLegacyNumber(source.number).number, prefixes[source.key] ?? '', prefixes[key] ?? '');
	}

	function stateText(state: string): string {
		return t(state === 'released' ? 'documentStatusReleasedMigrated' : state === 'empty' ? 'legacyStateEmpty' : 'legacyStateToVerify');
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errors = {};
		message = '';
		failure = '';
		rows.forEach(splitNumber);
		rows = fillNames(rows);
		// Isian disusun sebelum tombol dikunci, supaya kesalahan di sini tidak menahan tombol.
		const input = {
			customer_id: customerId,
			one_set: oneSet,
			documents: setDocuments(rows, oneSet),
			model_ids: modelIds,
			part_number_ids: partIds.filter((id) => partChoices.some((p) => p.id === id))
		};
		busy = true;
		const result = await migrateSet(input);
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			errors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			const shown = details.some((d) => d.field.startsWith('documents.') || d.field === 'model_ids' || d.field === 'part_number_ids');
			if (!shown) {
				failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			}
			return;
		}
		const numbers = result.data.items.map((d) => d.number).join(', ');
		resetForm();
		await refresh();
		message = t('legacySetDone', { numbers });
	}
</script>

<h1>{t('legacyTitle')}</h1>
<p>{t('legacyIntro')}</p>

{#if !canMigrate}
	<p>{t('legacyNoPermission')}</p>
{:else}
	{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

	<div class="field chooser">
		<label for="migration-customer">{t('documentCustomer')}</label>
		<select id="migration-customer" bind:value={customerId} onchange={chooseCustomer}>
			<option value="">{t('numberCustomerChoose')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
	</div>

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if overview}
		<section aria-labelledby="migration-ready-h">
			<h2 id="migration-ready-h">{t('legacyReadyTitle')}</h2>
			<ul class="checks" data-testid="migration-ready">
				<li class:ok={overview.migration_mode}>
					{overview.migration_mode ? '✓' : '✗'}
					{t(overview.migration_mode ? 'legacyReadyModeOn' : 'legacyReadyModeOff')}
				</li>
				<li class:ok={overview.form_set_published}>
					{overview.form_set_published ? '✓' : '✗'}
					{#if overview.form_set_published}{t('legacyReadyFormSet')}{:else}<a href="/form-sets">{t('legacyReadyNoFormSet')}</a>{/if}
				</li>
				<li class:ok={overview.models > 0}>
					{overview.models > 0 ? '✓' : '✗'}
					{#if overview.models > 0}{t('legacyReadyModels', { count: String(overview.models) })}{:else}<a href="/models">{t('legacyReadyNoModels')}</a>{/if}
				</li>
				<li class:ok={overview.processes > 0}>
					{overview.processes > 0 ? '✓' : '✗'}
					{#if overview.processes > 0}{t('legacyReadyProcesses', { count: String(overview.processes) })}{:else}<a href="/processes">{t('legacyReadyNoProcesses')}</a>{/if}
				</li>
			</ul>
		</section>

		<section aria-labelledby="migration-set-h">
			<h2 id="migration-set-h">{t('legacySetTitle')}</h2>
			<p class="hint">{t('legacySetHint')}</p>
			{#if !ready}<p class="hint">{t('legacySetNotReady')}</p>{/if}
			<form onsubmit={submit} novalidate>
				<label class="check"><input type="checkbox" bind:checked={oneSet} /> {t('legacyOneSet')}</label>
				<p class="hint">{t(oneSet ? 'legacyOneSetHint' : 'legacyNotSetHint')}</p>
				<div class="table-wrap">
					<table class="set-table" data-testid="migration-set">
						<thead>
							<tr>
								<th scope="col">{t('numberColumnType')}</th>
								<th scope="col">{t('migrateNumber')}</th>
								<th scope="col">{t('migrateRevision')}</th>
								<th scope="col">{t('migrateEffectiveDate')}</th>
								<th scope="col">{t('migratePreparedBy')}</th>
								<th scope="col">{t('migrateApprovedBy')}</th>
							</tr>
						</thead>
						<tbody>
							{#each rows as row (row.key)}
								{@const name = typeNames[row.key] ?? row.key}
								{@const field = `documents.${row.key}`}
								<tr>
									<th scope="row">{name}</th>
									<td>
										<input
											type="text"
											class="mono"
											aria-label={t('legacyNumberOf', { type: name })}
											placeholder={placeholder(row.key)}
											bind:value={row.number}
											onblur={() => splitNumber(row)}
										/>
										{#if errors[`${field}.number`]}<p class="error">{errors[`${field}.number`]}</p>{/if}
										{#if errors[`${field}.document_type_key`]}<p class="error">{errors[`${field}.document_type_key`]}</p>{/if}
									</td>
									<td>
										<input type="number" min="0" class="narrow" aria-label={t('legacyRevisionOf', { type: name })} bind:value={row.revision} />
										{#if errors[`${field}.revision`]}<p class="error">{errors[`${field}.revision`]}</p>{/if}
									</td>
									<td>
										<input type="date" aria-label={t('legacyDateOf', { type: name })} bind:value={row.effectiveDate} />
										{#if errors[`${field}.effective_date`]}<p class="error">{errors[`${field}.effective_date`]}</p>{/if}
									</td>
									<td>
										<input type="text" aria-label={t('legacyPreparedOf', { type: name })} bind:value={row.preparedBy} />
										{#if errors[`${field}.prepared_by`]}<p class="error">{errors[`${field}.prepared_by`]}</p>{/if}
									</td>
									<td>
										<input type="text" aria-label={t('legacyApprovedOf', { type: name })} bind:value={row.approvedBy} />
										{#if errors[`${field}.approved_by`]}<p class="error">{errors[`${field}.approved_by`]}</p>{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<p class="hint">{t('legacyNamesHint')}</p>
				{#if errors.documents}<p class="error">{errors.documents}</p>{/if}
				<fieldset>
					<legend>{t('documentModels')}</legend>
					{#each models as m (m.id)}
						<label><input type="checkbox" bind:group={modelIds} value={m.id} /> {m.name}</label>
					{/each}
					{#if models.length === 0}<p class="hint">{t('documentNoModels')}</p>{/if}
					{#if errors.model_ids}<p class="error">{errors.model_ids}</p>{/if}
				</fieldset>
				{#if partChoices.length > 0}
					<fieldset>
						<legend>{t('documentPartNumbers')}</legend>
						{#each partChoices as p (p.id)}
							<label><input type="checkbox" bind:group={partIds} value={p.id} /> {p.number}{p.name ? ` · ${p.name}` : ''}</label>
						{/each}
						{#if errors.part_number_ids}<p class="error">{errors.part_number_ids}</p>{/if}
					</fieldset>
				{/if}
				<p><button type="submit" class="primary" disabled={busy}>{t('legacySetSubmit')}</button></p>
			</form>
		</section>

		<section aria-labelledby="migration-board-h">
			<h2 id="migration-board-h">{t('legacyBoardTitle')}</h2>
			<p class="hint">{t('legacyBoardHint')}</p>
			{#if overview.documents.length === 0}
				<p>{t('legacyBoardEmpty')}</p>
			{:else}
				<div class="table-wrap">
					<table data-testid="migration-board">
						<thead>
							<tr>
								<th scope="col">{t('numberColumnNumber')}</th>
								<th scope="col">{t('numberColumnType')}</th>
								<th scope="col">{t('migrateRevision')}</th>
								<th scope="col">{t('documentRows')}</th>
								<th scope="col">{t('legacyEnteredBy')}</th>
								<th scope="col">{t('numberColumnStatus')}</th>
							</tr>
						</thead>
						<tbody>
							{#each overview.documents as d (d.id)}
								<tr>
									<td><a href="/documents/view?id={d.id}" class="mono">{d.number}</a></td>
									<td>{d.document_type}</td>
									<td>{d.revision}</td>
									<td>{d.rows}</td>
									<td>{d.entered_by} · {formatDateTime(d.entered_at)}</td>
									<td>
										<span class="badge" data-status={d.state === 'released' ? 'released' : 'draft'}>{stateText(d.state)}</span>
										{#if d.verified_by}<span class="hint">{t('legacyVerifiedBy', { name: d.verified_by })}</span>{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	{/if}
{/if}

<style>
	.chooser {
		max-width: 28rem;
		margin-bottom: 1.5rem;
	}

	.checks {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 4px;
	}

	.checks li {
		color: var(--warning);
	}

	.checks li.ok {
		color: var(--text-2);
	}

	.table-wrap {
		overflow-x: auto;
	}

	.set-table input {
		width: 100%;
		min-width: 8rem;
	}

	.set-table input.narrow {
		min-width: 4rem;
	}

	fieldset {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1.25rem;
		margin: 1rem 0;
	}

	fieldset legend {
		padding: 0 4px;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.badge + .hint {
		display: block;
	}
</style>
