<script lang="ts">
	// Halaman satu dokumen (M4.1): header dan baris disimpan lalu dibaca lagi
	// dari server. Kolom terisi otomatis (nomor operasi, proses) diambil dari
	// atribut baris (D-122, D-123); kolom yang dihitung tidak diketik. Layar
	// workbook mirip Excel menggantikan tabel ini di M4.2 (D-145).
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { addDocumentRow, getDocument, getFormatVersion, listProcesses, setDocumentHeader, updateDocumentRow, type ApiFailure } from '#lib/api/client.ts';
	import type { Document, DocumentRow, FormatVersion, Process } from '#lib/api/gen/index.ts';
	import { cellText, cleanContent, typedFields } from '#lib/documents.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let doc = $state<Document | null>(null);
	let format = $state<FormatVersion | null>(null);
	let processes = $state<Process[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let canEdit = $derived(can('E-2'));

	let rowFields = $derived(format ? format.fields.filter((f) => f.area === 'row' && !f.hidden) : []);
	let typedRow = $derived(format ? typedFields(format.fields, 'row') : []);
	let typedHeader = $derived(format ? typedFields(format.fields, 'header') : []);

	// Isian header, baris baru, dan baris yang sedang diubah.
	let header = $state<Record<string, string>>({});
	let row = $state<{ processId: string; operation: string; values: Record<string, string> }>({ processId: '', operation: '', values: {} });
	let editingId = $state<string | null>(null);

	onMount(load);

	async function load() {
		const id = page.url.searchParams.get('id') ?? '';
		const result = await getDocument(id);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		const [formatResult, processResult] = await Promise.all([getFormatVersion(result.data.format_version_id), listProcesses()]);
		if (!formatResult.ok) {
			loadError = failureText(formatResult.error);
			return;
		}
		format = formatResult.data;
		processes = processResult.ok ? processResult.data.items : [];
		use(result.data);
	}

	function use(d: Document) {
		doc = d;
		header = Object.fromEntries(Object.entries(d.header).map(([k, v]) => [k, String(v ?? '')]));
	}

	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
		failure = details.length === 0 ? failureText(error) : t('documentFixFields');
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
	}

	async function saveHeader(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (!doc) {
			return;
		}
		const result = await setDocumentHeader(doc.id, cleanContent(typedHeader, header));
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentHeaderSaved');
	}

	// startEdit membuka satu baris di formulir yang sama dengan baris baru.
	function startEdit(r: DocumentRow) {
		reset();
		editingId = r.id;
		row = {
			processId: r.process_id ?? '',
			operation: r.operation_number ?? '',
			values: Object.fromEntries(Object.entries(r.content).map(([k, v]) => [k, String(v ?? '')]))
		};
	}

	function clearRow() {
		editingId = null;
		row = { processId: '', operation: '', values: {} };
	}

	async function saveRow(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (!doc) {
			return;
		}
		const input = { process_id: row.processId || undefined, operation_number: row.operation || undefined, content: cleanContent(typedRow, row.values) };
		const result = editingId ? await updateDocumentRow(doc.id, editingId, input) : await addDocumentRow(doc.id, input);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		message = editingId ? t('documentRowSaved') : t('documentRowAdded');
		use(result.data);
		clearRow();
	}
</script>

{#if loadError}
	<h1>{t('documentsTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
{:else if !doc || !format}
	<p>{t('formatLoading')}</p>
{:else}
	<p><a href="/documents">{t('documentBack')}</a></p>
	<h1 class="mono">{doc.number}</h1>
	<dl class="facts">
		<dt>{t('numberColumnType')}</dt>
		<dd>{doc.document_type.name}</dd>
		<dt>{t('documentCustomer')}</dt>
		<dd>{doc.customer_code}</dd>
		<dt>{t('documentModels')}</dt>
		<dd>{doc.models.map((m) => m.name).join(', ')}</dd>
		{#if doc.part_numbers.length > 0}
			<dt>{t('documentPartNumbers')}</dt>
			<dd>{doc.part_numbers.map((p) => p.number).join(', ')}</dd>
		{/if}
		<dt>{t('documentFormSet')}</dt>
		<dd>{doc.form_set_name}</dd>
		<dt>{t('documentRoute')}</dt>
		<dd>{doc.route?.name ?? t('documentNoRoute')}{doc.route_reason ? ` · ${doc.route_reason}` : ''}</dd>
		<dt>{t('numberColumnStatus')}</dt>
		<dd><span class="badge" data-status="draft">{t('versionStatusDraft')}</span></dd>
	</dl>

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if typedHeader.length > 0}
		<section>
			<h2>{t('formatHeaderFields')}</h2>
			<form class="grid" onsubmit={saveHeader} novalidate>
				{#each typedHeader as f (f.id)}
					<div class="field">
						<label for="header-{f.id}">{f.label}</label>
						{#if f.kind === 'select'}
							<select id="header-{f.id}" bind:value={header[f.id]} disabled={!canEdit}>
								<option value=""></option>
								{#each f.options ?? [] as o (o.key)}<option value={o.key}>{o.label}</option>{/each}
							</select>
						{:else}
							<input id="header-{f.id}" type={f.kind === 'date' ? 'date' : 'text'} bind:value={header[f.id]} disabled={!canEdit} />
						{/if}
						{#if fieldErrors[f.id]}<p class="error">{fieldErrors[f.id]}</p>{/if}
					</div>
				{/each}
				{#if canEdit}<button type="submit">{t('documentHeaderSave')}</button>{/if}
			</form>
		</section>
	{/if}

	<section>
		<h2>{t('documentRowsTitle')}</h2>
		{#if doc.rows.length === 0}
			<p>{t('documentRowsEmpty')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>#</th>
						{#each rowFields as f (f.id)}<th>{f.label}</th>{/each}
						{#if canEdit}<th>{t('masterActions')}</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each doc.rows as r (r.id)}
						<tr class:editing={editingId === r.id}>
							<td>{r.position}</td>
							{#each rowFields as f (f.id)}<td>{cellText(f, r)}</td>{/each}
							{#if canEdit}<td><button type="button" onclick={() => startEdit(r)}>{t('formatEdit')}</button></td>{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}

		{#if canEdit}
			<h3>{editingId ? t('documentRowEditTitle') : t('documentRowAddTitle')}</h3>
			<form class="grid" onsubmit={saveRow} novalidate>
				<div class="field">
					<label for="row-operation">{t('documentOperationNumber')}</label>
					<input id="row-operation" bind:value={row.operation} maxlength="20" />
					{#if fieldErrors.operation_number}<p class="error">{fieldErrors.operation_number}</p>{/if}
				</div>
				<div class="field">
					<label for="row-process">{t('documentProcess')}</label>
					<select id="row-process" bind:value={row.processId}>
						<option value=""></option>
						{#each processes as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
					</select>
					{#if fieldErrors.process_id}<p class="error">{fieldErrors.process_id}</p>{/if}
				</div>
				{#each typedRow as f (f.id)}
					<div class="field">
						<label for="row-{f.id}">{f.label}</label>
						{#if f.kind === 'select' || f.kind === 'symbol'}
							<select id="row-{f.id}" bind:value={row.values[f.id]}>
								<option value=""></option>
								{#if f.kind === 'select'}
									{#each f.options ?? [] as o (o.key)}<option value={o.key}>{o.label}</option>{/each}
								{:else}
									{#each format.symbols.filter((s) => !s.automatic) as s (s.key)}<option value={s.key}>{s.name}</option>{/each}
								{/if}
							</select>
						{:else if f.kind === 'rating'}
							<input id="row-{f.id}" type="number" min="1" max="10" bind:value={row.values[f.id]} />
						{:else}
							<input id="row-{f.id}" type={f.kind === 'date' ? 'date' : 'text'} bind:value={row.values[f.id]} />
						{/if}
						{#if fieldErrors[f.id]}<p class="error">{fieldErrors[f.id]}</p>{/if}
					</div>
				{/each}
				<div class="actions">
					<button type="submit">{editingId ? t('documentRowSave') : t('documentRowAdd')}</button>
					{#if editingId}<button type="button" onclick={clearRow}>{t('masterCancel')}</button>{/if}
				</div>
			</form>
		{/if}
	</section>
{/if}

<style>
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 4px 16px;
		margin: 0 0 1.5rem;
	}

	.facts dt {
		color: var(--text-3);
	}

	.facts dd {
		margin: 0;
	}

	.grid {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.grid .field {
		min-width: 12rem;
	}

	.grid > button,
	.actions {
		margin-top: var(--label-space);
	}

	.actions {
		display: flex;
		gap: 0.5rem;
	}

	tr.editing td {
		background: var(--accent-weak);
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
