<script lang="ts">
	// Halaman satu dokumen (M4.1, M4.2): header, dan baris di workbook mirip
	// Excel (D-145) dengan tab sheet untuk dokumen satu set. Kolom terisi
	// otomatis (nomor operasi, proses) diambil dari atribut baris (D-122,
	// D-123); kolom yang dihitung tidak diketik.
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { addDocumentRow, addDocumentRows, deleteDocumentRow, getDocument, getFormatVersion, listProcesses, setDocumentHeader, updateDocumentRow, type ApiFailure } from '#lib/api/client.ts';
	import type { Document, DocumentRow, DocumentRowInput, FormatVersion, Process } from '#lib/api/gen/index.ts';
	import PastePanel from '#lib/documents/PastePanel.svelte';
	import Workbook from '#lib/documents/Workbook.svelte';
	import { cleanContent, typedFields } from '#lib/documents.ts';
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

	let typedHeader = $derived(format ? typedFields(format.fields, 'header') : []);

	// pasting: panel Tempel dari Excel sedang dibuka (M4.4).
	let pasting = $state(false);

	// Isian header.
	let header = $state<Record<string, string>>({});

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

	// use memakai dokumen terbaru dari server. Isian header hanya diganti bila
	// header di server memang berubah, supaya ketikan yang belum disimpan tidak
	// hilang saat baris ditambah atau dimuat ulang.
	function use(d: Document) {
		const sameHeader = doc !== null && doc.id === d.id && doc.header_version === d.header_version;
		doc = d;
		if (!sameHeader) {
			header = Object.fromEntries(Object.entries(d.header).map(([k, v]) => [k, String(v ?? '')]));
		}
	}

	// changedByOther: server menolak karena isinya sudah diubah orang lain
	// sejak dibaca (penguncian optimistik, M4.3, R-DOC-07).
	function changedByOther(error: ApiFailure): boolean {
		return error.status === 409 && (error.body?.details ?? []).some((d) => d.code === 'changed');
	}

	// reloadAfterConflict memuat isi terbaru dan menampilkan pesan konflik.
	async function reloadAfterConflict(text: string) {
		if (!doc) {
			return;
		}
		const latest = await getDocument(doc.id);
		if (latest.ok) {
			use(latest.data);
		}
		failure = text;
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
		const result = await setDocumentHeader(doc.id, cleanContent(typedHeader, header), doc.header_version);
		if (!result.ok) {
			if (changedByOther(result.error)) {
				await reloadAfterConflict(t('documentHeaderChanged'));
				return;
			}
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentHeaderSaved');
	}

	// saveCell menyimpan satu baris dari tabel workbook (M4.2). Dokumen tidak
	// dimuat ulang, supaya posisi kursor di tabel tidak hilang. Jawabannya:
	// baris yang tersimpan (dengan versi barunya), 'reloaded' bila baris sudah
	// diubah orang lain dan isi terbaru dimuat (M4.3), atau null bila ditolak.
	async function saveCell(rowId: string, input: DocumentRowInput): Promise<DocumentRow | 'reloaded' | null> {
		reset();
		if (!doc) {
			return null;
		}
		const result = await updateDocumentRow(doc.id, rowId, input);
		if (!result.ok) {
			if (changedByOther(result.error)) {
				await reloadAfterConflict(t('documentRowChanged'));
				return 'reloaded';
			}
			showErrors(result.error);
			return null;
		}
		message = t('documentRowSaved');
		return result.data.rows.find((r) => r.id === rowId) ?? null;
	}

	// addRow menambah baris kosong di akhir; isinya diketik langsung di tabel.
	async function addRow() {
		reset();
		if (!doc) {
			return;
		}
		const result = await addDocumentRow(doc.id, { content: {} });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentRowAdded');
	}

	// pasteRows menambah blok baris dari Excel dalam satu permintaan (M4.4).
	async function pasteRows(rows: DocumentRowInput[]): Promise<boolean> {
		reset();
		if (!doc) {
			return false;
		}
		const result = await addDocumentRows(doc.id, rows);
		if (!result.ok) {
			showErrors(result.error);
			return false;
		}
		use(result.data);
		pasting = false;
		message = t('pasteDone', { rows: String(rows.length) });
		return true;
	}

	async function removeRow(rowId: string) {
		reset();
		if (!doc) {
			return;
		}
		const position = doc.rows.find((r) => r.id === rowId)?.position ?? 0;
		if (!confirm(t('documentRowDeleteConfirm', { position: String(position) }))) {
			return;
		}
		const result = await deleteDocumentRow(doc.id, rowId);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentRowRemoved', { position: String(position) });
	}

	// openSheet berpindah ke dokumen lain di set yang sama tanpa memuat ulang halaman.
	async function openSheet(event: MouseEvent, id: string) {
		event.preventDefault();
		if (doc && id === doc.id) {
			return;
		}
		history.pushState({}, '', `/documents/view?id=${id}`);
		reset();
		doc = null;
		await load();
	}</script>

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
		<!-- Tab sheet workbook (D-145): dokumen satu set PFD, PFMEA, dan CP. Tab Check menyusul di M4.9. -->
		<nav class="sheets" aria-label={t('documentSheets')}>
			{#each doc.set as m (m.id)}
				<a href="/documents/view?id={m.id}" aria-current={m.id === doc.id ? 'page' : undefined} onclick={(e) => openSheet(e, m.id)}>{m.document_type.name}</a>
			{/each}
		</nav>
		<h2>{t('documentRowsTitle')}</h2>
		{#if canEdit}<p class="hint">{t('documentGridHint')}</p>{/if}
		{#key doc.id}
			<Workbook {doc} {format} {processes} {canEdit} onsave={saveCell} ondelete={removeRow} />
		{/key}
		{#if canEdit}
			<p class="row-actions">
				<button type="button" onclick={addRow}>{t('documentRowAdd')}</button>
				{#if !pasting}<button type="button" onclick={() => (pasting = true)}>{t('pasteOpen')}</button>{/if}
			</p>
		{/if}
	</section>

	{#if canEdit && pasting}
		<section>
			<PastePanel {format} {processes} onimport={pasteRows} oncancel={() => (pasting = false)} />
		</section>
	{/if}{/if}

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

	.grid > button {
		margin-top: var(--label-space);
	}

	.row-actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.sheets {
		display: flex;
		gap: 4px;
		margin-bottom: 1rem;
		border-bottom: 1px solid var(--border);
	}

	.sheets a {
		padding: 8px 16px;
		border: 1px solid transparent;
		border-bottom: none;
		border-radius: var(--radius) var(--radius) 0 0;
		color: var(--text-2);
	}

	.sheets a[aria-current='page'] {
		background: var(--surface);
		border-color: var(--border);
		color: var(--accent-ink);
		font-weight: 600;
		margin-bottom: -1px;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
